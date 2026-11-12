// Package sftpsvc 는 SSH 세션 위의 SFTP 작업(목록·전송·미리보기·편집)을 제공한다.
package sftpsvc

import (
	"errors"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"termgo/internal/session"
	"termgo/internal/store"
	"termgo/internal/ui"
)

// Entry 는 SftpEntry(shared/types.ts).
type Entry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	// IsLink 는 심볼릭 링크 여부.
	IsLink bool   `json:"isLink"`
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Mode   uint32 `json:"mode"`
	Perm   string `json:"perm"`
}

// Service 는 세션별 SFTP 클라이언트와 전송 큐를 관리한다.
type Service struct {
	Sessions *session.Manager
	UI       *ui.Bridge
	Store    *store.Store

	mu      sync.Mutex
	clients map[string]*cached
	homes   map[string]string
	queues  map[string]*sync.Mutex
	// jobs 는 진행 중인 끌어 놓기 복사 작업 (취소용)
	jobs map[string]*copyJob
}

type cached struct {
	ssh *ssh.Client
	cl  *sftp.Client
}

// New 는 서비스를 만든다.
func New(sm *session.Manager, b *ui.Bridge, st *store.Store) *Service {
	return &Service{Sessions: sm, UI: b, Store: st, clients: map[string]*cached{}, homes: map[string]string{}, queues: map[string]*sync.Mutex{}}
}

// client 는 세션의 SFTP 클라이언트 (SSH 연결이 바뀌면 새로 만든다).
func (pstrV *Service) client(szSession string) (*sftp.Client, error) {
	s := pstrV.Sessions.Get(szSession)
	if s == nil || s.Shell() == nil {
		return nil, errors.New("SSH로 연결된 세션에서만 사용할 수 있습니다")
	}
	sc := s.Shell().Conn.Client
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if c := pstrV.clients[szSession]; c != nil && c.ssh == sc {
		return c.cl, nil
	}
	cl, err := sftp.NewClient(sc)
	if err != nil {
		return nil, err
	}
	pstrV.clients[szSession] = &cached{ssh: sc, cl: cl}
	delete(pstrV.homes, szSession)
	return cl, nil
}

// queue 는 세션 하나의 전송을 직렬화하는 잠금.
func (pstrV *Service) queue(szSession string) *sync.Mutex {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	m := pstrV.queues[szSession]
	if m == nil {
		m = &sync.Mutex{}
		pstrV.queues[szSession] = m
	}
	return m
}

// Forget 은 세션이 닫힐 때 캐시를 정리한다.
func (pstrV *Service) Forget(szSession string) {
	pstrV.mu.Lock()
	c := pstrV.clients[szSession]
	delete(pstrV.clients, szSession)
	delete(pstrV.homes, szSession)
	delete(pstrV.queues, szSession)
	pstrV.mu.Unlock()
	if c != nil {
		_ = c.cl.Close()
	}
}

func permString(mode fs.FileMode) string {
	sz := []byte("----------")
	switch {
	case mode.IsDir():
		sz[0] = 'd'
	case mode&fs.ModeSymlink != 0:
		sz[0] = 'l'
	}
	const rwx = "rwxrwxrwx"
	for i := 0; i < 9; i++ {
		if mode&(1<<uint(8-i)) != 0 {
			sz[i+1] = rwx[i]
		}
	}
	return string(sz)
}

// Home 은 홈 폴더 (원격은 realpath ".", 로컬은 사용자 홈).
func (pstrV *Service) Home(szSession string) (string, error) {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return "", err
	}
	if fsys.Local() {
		return os.UserHomeDir()
	}
	pstrV.mu.Lock()
	sz := pstrV.homes[szSession]
	pstrV.mu.Unlock()
	if sz != "" {
		return sz, nil
	}
	sz, err = fsys.RealPath(".")
	if err != nil {
		return "", err
	}
	pstrV.mu.Lock()
	pstrV.homes[szSession] = sz
	pstrV.mu.Unlock()
	return sz, nil
}

// start 는 패널을 처음 열 때의 폴더 (로컬 세션은 시작 폴더, 원격은 홈).
func (pstrV *Service) start(szSession string, fsys fileSys) (string, error) {
	if fsys.Local() {
		if s := pstrV.Sessions.Get(szSession); s != nil {
			if sz := s.LocalCwd(); sz != "" {
				return sz, nil
			}
		}
	}
	return pstrV.Home(szSession)
}

func (pstrV *Service) expand(szSession, szPath string) (string, error) {
	if szPath == "~" || strings.HasPrefix(szPath, "~/") || strings.HasPrefix(szPath, `~\`) {
		h, err := pstrV.Home(szSession)
		if err != nil {
			return "", err
		}
		if len(szPath) <= 2 {
			return h, nil
		}
		fsys, err := pstrV.fsFor(szSession)
		if err != nil {
			return "", err
		}
		return fsys.Join(h, szPath[2:]), nil
	}
	return szPath, nil
}

// ListResult 는 sftp:list 결과.
type ListResult struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
	// Parent 는 상위 폴더 (루트면 Path 와 같다).
	Parent string `json:"parent"`
	// Sep 는 경로 구분자 ("/" 또는 "\\").
	Sep   string `json:"sep"`
	Local bool   `json:"local"`
}

func toEntry(fsys fileSys, szDir string, fi fs.FileInfo) Entry {
	e := Entry{
		Name: fi.Name(), Path: fsys.Join(szDir, fi.Name()), IsDir: fi.IsDir(),
		IsLink: fi.Mode()&fs.ModeSymlink != 0, Size: fi.Size(), Mtime: fi.ModTime().UnixMilli(),
		Mode: uint32(fi.Mode().Perm()), Perm: permString(fi.Mode()),
	}
	if e.IsLink {
		if st, err := fsys.Stat(e.Path); err == nil {
			e.IsDir = st.IsDir()
		}
	}
	return e
}

// List 는 폴더 내용을 돌려준다 (폴더 우선, 이름순). 빈 경로는 시작 폴더.
func (pstrV *Service) List(szSession, szPath string) (*ListResult, error) {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return nil, err
	}
	var szExp string
	if szPath == "" {
		szExp, err = pstrV.start(szSession, fsys)
	} else {
		szExp, err = pstrV.expand(szSession, szPath)
	}
	if err != nil {
		return nil, err
	}
	szReal, err := fsys.RealPath(szExp)
	if err != nil {
		szReal = szExp
	}
	infos, err := fsys.ReadDir(szReal)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		out = append(out, toEntry(fsys, szReal, fi))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return &ListResult{Path: szReal, Entries: out, Parent: fsys.Dir(szReal), Sep: fsys.Sep(), Local: fsys.Local()}, nil
}

// Mkdir 은 폴더를 만든다.
func (pstrV *Service) Mkdir(szSession, szPath string) error {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return err
	}
	return fsys.Mkdir(szPath)
}

// Rename 은 이름/위치를 바꾼다.
func (pstrV *Service) Rename(szSession, szFrom, szTo string) error {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return err
	}
	return fsys.Rename(szFrom, szTo)
}

// Chmod 는 권한을 바꾼다.
func (pstrV *Service) Chmod(szSession, szPath string, iMode uint32) error {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return err
	}
	return fsys.Chmod(szPath, fs.FileMode(iMode))
}

func removeRec(fsys fileSys, szPath string) error {
	st, err := fsys.Lstat(szPath)
	if err != nil {
		return err
	}
	if st.IsDir() {
		items, err := fsys.ReadDir(szPath)
		if err != nil {
			return err
		}
		for _, it := range items {
			if err := removeRec(fsys, fsys.Join(szPath, it.Name())); err != nil {
				return err
			}
		}
		return fsys.Remove(szPath, true)
	}
	return fsys.Remove(szPath, false)
}

// Remove 는 파일/폴더를 재귀적으로 지운다 (링크는 따라가지 않는다).
func (pstrV *Service) Remove(szSession string, slPaths []string) error {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return err
	}
	for _, p := range slPaths {
		if err := removeRec(fsys, p); err != nil {
			return err
		}
	}
	return nil
}

func mkdirAll(cl *sftp.Client, szPath string) {
	_ = cl.MkdirAll(szPath)
}
