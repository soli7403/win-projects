// Package sftpfs 는 SFTP 탐색기(목록·조작·미리보기)와 전송 큐를 구현한다 (spec/protocols.md §3).
// 연결 자체(ssh.Client)는 호출자가 주고, 이 패키지는 sftp.Client 위에서만 동작한다.
package sftpfs

import (
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// G_ciPreviewImageMax 는 이미지 미리보기 한도(20MB)다. G_ciPreviewTextMax 는 텍스트 한도(512KB)다.
const (
	G_ciPreviewImageMax = 20 * 1024 * 1024
	G_ciPreviewTextMax  = 512 * 1024
)

// g_mapImageMime 은 미리보기 이미지 확장자와 MIME 이다 (protocols.md §3).
var g_mapImageMime = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".bmp": "image/bmp", ".svg": "image/svg+xml",
}

// Entry 는 목록의 항목 하나다 (protocols.md §3 목록).
type Entry struct {
	M_szName   string `json:"name"`
	M_szPath   string `json:"path"`
	M_bIsDir   bool   `json:"isDir"`
	M_bIsLink  bool   `json:"isLink"`
	M_i64Size  int64  `json:"size"`
	M_i64Mtime int64  `json:"mtime"`
	M_iMode    int    `json:"mode"`
	M_szPerm   string `json:"perm"`
}

// Listing 은 sftp.list 결과다. 상위 폴더(루트면 자기 자신)·경로 구분자·로컬 여부를 함께 준다 (file-panel.md §5).
type Listing struct {
	M_szPath    string  `json:"path"`
	M_slEntries []Entry `json:"entries"`
	M_szParent  string  `json:"parent"`
	M_szSep     string  `json:"sep"`
	M_bLocal    bool    `json:"local"`
}

// Preview 는 미리보기 결과다.
type Preview struct {
	M_szKind string `json:"kind"`
	M_szData string `json:"data"`
}

// Fs 는 Backend(서버 또는 이 PC) 위의 탐색 기능이다.
type Fs struct {
	be Backend
	// szStart 는 빈 경로로 열 때의 폴더다 (로컬 터미널의 시작 폴더). 비면 홈.
	szStart string
}

// NewFs 는 연결된 sftp 클라이언트를 감싼다.
func NewFs(client *sftp.Client) *Fs {
	return &Fs{be: sftpBackend{client: client}}
}

// NewLocalFs 는 이 PC 파일시스템 탐색기다. szStart 는 처음 여는 폴더(비거나 없으면 홈).
func NewLocalFs(szStart string) *Fs {
	return &Fs{be: LocalBackend{}, szStart: szStart}
}

// Backend 는 이 탐색기의 파일시스템이다.
func (pFs *Fs) Backend() Backend { return pFs.be }

// Home 은 홈 경로(서버는 "." 의 실제 경로, 로컬은 사용자 홈)를 돌려준다 (protocols.md §3).
func (pFs *Fs) Home() (string, error) {
	return pFs.be.HomeDir()
}

// Start 는 빈 경로로 열 때의 폴더다: 시작 폴더가 있고 폴더면 그것, 아니면 홈.
func (pFs *Fs) Start() (string, error) {
	if pFs.szStart != "" {
		if info, errStat := pFs.be.Stat(pFs.szStart); errStat == nil && info.IsDir() {
			return pFs.szStart, nil
		}
	}
	return pFs.Home()
}

// Normalize 는 경로를 정한다: 빈 값은 시작 폴더, "~"·"~/..."·"~\..." 는 홈 기준으로 바꾼다.
func (pFs *Fs) Normalize(szPath string) (string, error) {
	if szPath == "" {
		return pFs.Start()
	}
	if bHome, szRest := isHomeRef(szPath); bHome {
		szHome, errHome := pFs.Home()
		if errHome != nil {
			return "", errHome
		}
		if szRest == "" {
			return szHome, nil
		}
		return pFs.be.Join(szHome, szRest), nil
	}
	if szReal, errReal := pFs.be.Resolve(szPath); errReal == nil {
		return szReal, nil
	}
	return szPath, nil
}

// List 는 폴더 목록을 돌려준다. 폴더 먼저, 그다음 이름 오름차순(대소문자 무시).
func (pFs *Fs) List(szPath string) (Listing, error) {
	szReal, errNorm := pFs.Normalize(szPath)
	if errNorm != nil {
		return Listing{}, errNorm
	}
	infos, errRead := pFs.be.ReadDir(szReal)
	if errRead != nil {
		return Listing{}, errRead
	}
	slEntries := make([]Entry, 0, len(infos))
	for _, info := range infos {
		slEntries = append(slEntries, pFs.entryOf(szReal, info))
	}
	sort.SliceStable(slEntries, func(i, j int) bool {
		if slEntries[i].M_bIsDir != slEntries[j].M_bIsDir {
			return slEntries[i].M_bIsDir
		}
		return strings.ToLower(slEntries[i].M_szName) < strings.ToLower(slEntries[j].M_szName)
	})
	return Listing{M_szPath: szReal, M_slEntries: slEntries, M_szParent: pFs.be.Dir(szReal), M_szSep: pFs.be.Separator(), M_bLocal: pFs.be.IsLocal()}, nil
}

// entryOf 는 파일 정보를 항목으로 바꾼다. 링크는 대상이 폴더인지 다시 확인한다.
func (pFs *Fs) entryOf(szDir string, info fs.FileInfo) Entry {
	szFull := pFs.be.Join(szDir, info.Name())
	bLink := info.Mode()&fs.ModeSymlink != 0
	bDir := info.IsDir()
	if bLink {
		if target, errStat := pFs.be.Stat(szFull); errStat == nil {
			bDir = target.IsDir()
		}
	}
	iMode := int(info.Mode().Perm())
	return Entry{
		M_szName:   info.Name(),
		M_szPath:   szFull,
		M_bIsDir:   bDir,
		M_bIsLink:  bLink,
		M_i64Size:  info.Size(),
		M_i64Mtime: info.ModTime().UnixMilli(),
		M_iMode:    iMode,
		M_szPerm:   permString(info.Mode(), bDir, bLink),
	}
}

// permString 은 drwxr-xr-x 형식이다 (첫 글자 d/l/-).
func permString(mode fs.FileMode, bDir bool, bLink bool) string {
	var szFirst byte = '-'
	if bDir {
		szFirst = 'd'
	} else if bLink {
		szFirst = 'l'
	}
	const szRwx = "rwxrwxrwx"
	var sbOut strings.Builder
	sbOut.WriteByte(szFirst)
	for iBit := 0; iBit < 9; iBit++ {
		if mode.Perm()&(1<<uint(8-iBit)) != 0 {
			sbOut.WriteByte(szRwx[iBit])
		} else {
			sbOut.WriteByte('-')
		}
	}
	return sbOut.String()
}

// Mkdir 은 폴더를 만든다.
func (pFs *Fs) Mkdir(szPath string) error {
	return pFs.be.Mkdir(szPath)
}

// Rename 은 이름을 바꾼다 (이동도 같은 호출).
func (pFs *Fs) Rename(szOld string, szNew string) error {
	return pFs.be.Rename(szOld, szNew)
}

// Chmod 는 권한 모드를 바꾼다.
func (pFs *Fs) Chmod(szPath string, iMode int) error {
	return pFs.be.Chmod(szPath, fs.FileMode(iMode))
}

// Remove 는 경로들을 지운다. 폴더는 안의 내용까지 재귀로 지운다. 하나라도 실패하면 즉시 오류로 중단한다.
func (pFs *Fs) Remove(slPaths []string) error {
	for _, szPath := range slPaths {
		if errRemove := pFs.removeTree(szPath); errRemove != nil {
			return errRemove
		}
	}
	return nil
}

func (pFs *Fs) removeTree(szPath string) error {
	info, errLstat := pFs.be.Lstat(szPath)
	if errLstat != nil {
		return errLstat
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return pFs.be.RemoveFile(szPath)
	}
	infos, errRead := pFs.be.ReadDir(szPath)
	if errRead != nil {
		return errRead
	}
	for _, child := range infos {
		if errChild := pFs.removeTree(pFs.be.Join(szPath, child.Name())); errChild != nil {
			return errChild
		}
	}
	return pFs.be.RemoveDir(szPath)
}

// Preview 는 미리보기를 만든다. 조건에 맞지 않으면 nil (미리보기 없음).
// 이미지는 20MB 이하일 때 data: URL, 그 외는 512KB 이하이고 NUL 바이트가 없을 때 텍스트다.
func (pFs *Fs) Preview(szPath string) (*Preview, error) {
	info, errStat := pFs.be.Stat(szPath)
	if errStat != nil {
		return nil, errStat
	}
	if info.IsDir() {
		return nil, nil
	}
	szExt := strings.ToLower(path.Ext(strings.ReplaceAll(szPath, `\`, "/")))
	if szMime, bImage := g_mapImageMime[szExt]; bImage {
		if info.Size() > G_ciPreviewImageMax {
			return nil, nil
		}
		bData, errRead := pFs.readAll(szPath, G_ciPreviewImageMax)
		if errRead != nil {
			return nil, errRead
		}
		return &Preview{M_szKind: "image", M_szData: "data:" + szMime + ";base64," + base64.StdEncoding.EncodeToString(bData)}, nil
	}
	if info.Size() > G_ciPreviewTextMax {
		return nil, nil
	}
	bData, errRead := pFs.readAll(szPath, G_ciPreviewTextMax)
	if errRead != nil {
		return nil, errRead
	}
	if strings.IndexByte(string(bData), 0) >= 0 {
		return nil, nil
	}
	return &Preview{M_szKind: "text", M_szData: string(bData)}, nil
}

// readAll 은 최대 iLimit 바이트까지 읽는다.
func (pFs *Fs) readAll(szPath string, iLimit int64) ([]byte, error) {
	file, errOpen := pFs.be.OpenRead(szPath)
	if errOpen != nil {
		return nil, errOpen
	}
	defer file.Close()
	bData, errRead := io.ReadAll(io.LimitReader(file, iLimit+1))
	if errRead != nil && !errors.Is(errRead, io.EOF) {
		return nil, errRead
	}
	return bData, nil
}

// modTimeOf 는 파일 수정 시각을 돌려준다 (편집 감시용).
func (pFs *Fs) modTimeOf(szPath string) (time.Time, error) {
	info, errStat := pFs.be.Stat(szPath)
	if errStat != nil {
		return time.Time{}, errStat
	}
	return info.ModTime(), nil
}
