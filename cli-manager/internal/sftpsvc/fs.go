package sftpsvc

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"

	"termgo/internal/transport"
)

// fileSys 는 파일 패널이 다루는 파일시스템 (원격 SFTP 또는 이 PC).
type fileSys interface {
	// Key 는 같은 파일시스템인지 비교하는 값 (로컬은 "local", 원격은 접속 대상 user@host:port —
	// 같은 서버의 두 탭 사이 복사에서 원본을 자기 자신에 덮어쓰지 않게).
	Key() string
	Local() bool
	Stat(szPath string) (fs.FileInfo, error)
	Lstat(szPath string) (fs.FileInfo, error)
	ReadDir(szPath string) ([]fs.FileInfo, error)
	Open(szPath string) (io.ReadCloser, error)
	Create(szPath string) (io.WriteCloser, error)
	Mkdir(szPath string) error
	MkdirAll(szPath string) error
	Rename(szFrom, szTo string) error
	// Remove 는 파일 또는 빈 폴더를 지운다.
	Remove(szPath string, bDir bool) error
	Chmod(szPath string, mode fs.FileMode) error
	RealPath(szPath string) (string, error)
	Join(szDir, szName string) string
	Base(szPath string) string
	Dir(szPath string) string
	Sep() string
}

// remoteFS 는 SFTP 클라이언트 위의 fileSys.
type remoteFS struct {
	szKey string
	cl    *sftp.Client
}

func (r remoteFS) Key() string                             { return r.szKey }
func (r remoteFS) Local() bool                             { return false }
func (r remoteFS) Stat(p string) (fs.FileInfo, error)      { return r.cl.Stat(p) }
func (r remoteFS) Lstat(p string) (fs.FileInfo, error)     { return r.cl.Lstat(p) }
func (r remoteFS) ReadDir(p string) ([]fs.FileInfo, error) { return r.cl.ReadDir(p) }
func (r remoteFS) Open(p string) (io.ReadCloser, error)    { return r.cl.Open(p) }
func (r remoteFS) Create(p string) (io.WriteCloser, error) { return r.cl.Create(p) }
func (r remoteFS) Mkdir(p string) error                    { return r.cl.Mkdir(p) }
func (r remoteFS) MkdirAll(p string) error                 { return r.cl.MkdirAll(p) }
func (r remoteFS) Rename(a, b string) error                { return r.cl.Rename(a, b) }
func (r remoteFS) Chmod(p string, m fs.FileMode) error     { return r.cl.Chmod(p, m) }
func (r remoteFS) RealPath(p string) (string, error)       { return r.cl.RealPath(p) }
func (r remoteFS) Join(d, n string) string                 { return path.Join(d, n) }
func (r remoteFS) Base(p string) string                    { return path.Base(p) }
func (r remoteFS) Dir(p string) string                     { return path.Dir(p) }
func (r remoteFS) Sep() string                             { return "/" }
func (r remoteFS) Remove(p string, bDir bool) error {
	if bDir {
		return r.cl.RemoveDirectory(p)
	}
	return r.cl.Remove(p)
}

// localFS 는 이 PC 의 파일시스템.
type localFS struct{}

func (localFS) Key() string                         { return "local" }
func (localFS) Local() bool                         { return true }
func (localFS) Stat(p string) (fs.FileInfo, error)  { return os.Stat(p) }
func (localFS) Lstat(p string) (fs.FileInfo, error) { return os.Lstat(p) }
func (localFS) Open(p string) (io.ReadCloser, error) {
	return os.Open(p)
}
func (localFS) Create(p string) (io.WriteCloser, error) { return os.Create(p) }
func (localFS) Mkdir(p string) error                    { return os.Mkdir(p, 0o755) }
func (localFS) MkdirAll(p string) error                 { return os.MkdirAll(p, 0o755) }
func (localFS) Rename(a, b string) error                { return os.Rename(a, b) }
func (localFS) Chmod(p string, m fs.FileMode) error     { return os.Chmod(p, m) }
func (localFS) Join(d, n string) string                 { return filepath.Join(d, n) }
func (localFS) Base(p string) string                    { return filepath.Base(p) }
func (localFS) Dir(p string) string                     { return filepath.Dir(p) }
func (localFS) Sep() string                             { return string(filepath.Separator) }
func (localFS) Remove(p string, _ bool) error           { return os.Remove(p) }

func (localFS) ReadDir(p string) ([]fs.FileInfo, error) {
	ents, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	out := make([]fs.FileInfo, 0, len(ents))
	for _, e := range ents {
		fi, err := e.Info()
		if err != nil {
			continue // 목록 도중 사라진 항목
		}
		out = append(out, fi)
	}
	return out, nil
}

func (localFS) RealPath(p string) (string, error) {
	sz, err := filepath.Abs(transport.ExpandHome(p))
	if err != nil {
		return "", err
	}
	if len(sz) == 2 && sz[1] == ':' { // "C:" → "C:\"
		sz += string(filepath.Separator)
	}
	return sz, nil
}

// fsFor 는 세션의 파일시스템 (로컬 세션은 이 PC, SSH 세션은 SFTP).
func (pstrV *Service) fsFor(szSession string) (fileSys, error) {
	s := pstrV.Sessions.Get(szSession)
	if s == nil {
		return nil, errors.New("세션이 없습니다")
	}
	info := s.Info()
	if info.Protocol == "local" {
		return localFS{}, nil
	}
	cl, err := pstrV.client(szSession)
	if err != nil {
		return nil, err
	}
	return remoteFS{szKey: "ssh:" + info.Target, cl: cl}, nil
}
