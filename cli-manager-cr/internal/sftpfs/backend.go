package sftpfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// Backend 는 파일 패널이 다루는 파일시스템이다: SSH 서버(SFTP) 또는 이 PC (spec/behavior/file-panel.md §1).
// 읽기·쓰기는 모두 바이트 그대로다 (줄바꿈·인코딩 변환 없음).
type Backend interface {
	IsLocal() bool
	// HomeDir 은 홈 폴더의 실제 경로다.
	HomeDir() (string, error)
	// Resolve 는 경로를 실제 절대 경로로 바꾼다. 실패하면 오류.
	Resolve(szPath string) (string, error)
	Stat(szPath string) (fs.FileInfo, error)
	Lstat(szPath string) (fs.FileInfo, error)
	ReadDir(szPath string) ([]fs.FileInfo, error)
	OpenRead(szPath string) (io.ReadCloser, error)
	// OpenWrite 는 파일을 새로 만들거나 비우고 쓴다.
	OpenWrite(szPath string) (io.WriteCloser, error)
	Mkdir(szPath string) error
	MkdirAll(szPath string) error
	Rename(szOld string, szNew string) error
	Chmod(szPath string, mode fs.FileMode) error
	// Replace 는 이름을 바꾸되 szNew 가 이미 있으면 그것을 대신한다 (임시 이름으로 쓴 파일을 제자리에 놓을 때).
	Replace(szOld string, szNew string) error
	// OwnerKey 는 파일의 소유자·그룹을 비교용 문자열로 돌려준다. 알 수 없으면 빈 값.
	OwnerKey(info fs.FileInfo) string
	// RemoveFile 은 파일·링크를, RemoveDir 은 빈 폴더를 지운다.
	RemoveFile(szPath string) error
	RemoveDir(szPath string) error
	Join(szDir string, szName string) string
	Base(szPath string) string
	Dir(szPath string) string
	Separator() string
}

// sftpBackend 는 SFTP 클라이언트 위의 Backend 다.
type sftpBackend struct {
	client *sftp.Client
}

func (be sftpBackend) IsLocal() bool                           { return false }
func (be sftpBackend) HomeDir() (string, error)                { return be.client.RealPath(".") }
func (be sftpBackend) Resolve(szPath string) (string, error)   { return be.client.RealPath(szPath) }
func (be sftpBackend) Stat(szPath string) (fs.FileInfo, error) { return be.client.Stat(szPath) }
func (be sftpBackend) Lstat(szPath string) (fs.FileInfo, error) {
	return be.client.Lstat(szPath)
}
func (be sftpBackend) ReadDir(szPath string) ([]fs.FileInfo, error) { return be.client.ReadDir(szPath) }
func (be sftpBackend) OpenRead(szPath string) (io.ReadCloser, error) {
	return be.client.Open(szPath)
}
func (be sftpBackend) OpenWrite(szPath string) (io.WriteCloser, error) {
	return be.client.OpenFile(szPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}
func (be sftpBackend) Mkdir(szPath string) error               { return be.client.Mkdir(szPath) }
func (be sftpBackend) MkdirAll(szPath string) error            { return be.client.MkdirAll(szPath) }
func (be sftpBackend) Rename(szOld string, szNew string) error { return be.client.Rename(szOld, szNew) }
func (be sftpBackend) Chmod(szPath string, mode fs.FileMode) error {
	return be.client.Chmod(szPath, mode)
}
func (be sftpBackend) RemoveFile(szPath string) error          { return be.client.Remove(szPath) }
func (be sftpBackend) RemoveDir(szPath string) error           { return be.client.RemoveDirectory(szPath) }
func (be sftpBackend) Join(szDir string, szName string) string { return path.Join(szDir, szName) }
func (be sftpBackend) Base(szPath string) string               { return path.Base(szPath) }
func (be sftpBackend) Dir(szPath string) string                { return path.Dir(szPath) }
func (be sftpBackend) Separator() string                       { return "/" }

// G_cszPosixRename 은 대상이 있어도 바꾸는 이름 바꾸기 확장 이름이다 (OpenSSH).
const G_cszPosixRename = "posix-rename@openssh.com"

// Replace 는 OpenSSH 확장(posix-rename)이 있으면 그것으로 한 번에 바꾼다.
// 없는 서버는 표준 이름 바꾸기가 대상이 있을 때 실패할 수 있어, 실패하면 대상을 지우고 다시 한다.
func (be sftpBackend) Replace(szOld string, szNew string) error {
	if _, bPosix := be.client.HasExtension(G_cszPosixRename); bPosix {
		return be.client.PosixRename(szOld, szNew)
	}
	errRename := be.client.Rename(szOld, szNew)
	if errRename == nil {
		return nil
	}
	if _, errStat := be.client.Lstat(szNew); errStat != nil {
		return errRename
	}
	if errRemove := be.client.Remove(szNew); errRemove != nil {
		return errRename
	}
	return be.client.Rename(szOld, szNew)
}

// OwnerKey 는 서버가 알려 준 uid:gid 다.
func (be sftpBackend) OwnerKey(info fs.FileInfo) string {
	if pStat, bOk := info.Sys().(*sftp.FileStat); bOk {
		return fmt.Sprintf("%d:%d", pStat.UID, pStat.GID)
	}
	return ""
}

// LocalBackend 는 이 PC 의 파일시스템이다.
type LocalBackend struct{}

func (LocalBackend) IsLocal() bool            { return true }
func (LocalBackend) HomeDir() (string, error) { return os.UserHomeDir() }

// Resolve 는 절대 경로로 바꾼다. "C:" 처럼 드라이브만 주면 그 드라이브 루트로 본다.
func (LocalBackend) Resolve(szPath string) (string, error) {
	if len(szPath) == 2 && szPath[1] == ':' {
		szPath += string(filepath.Separator)
	}
	return filepath.Abs(szPath)
}
func (LocalBackend) Stat(szPath string) (fs.FileInfo, error)  { return os.Stat(szPath) }
func (LocalBackend) Lstat(szPath string) (fs.FileInfo, error) { return os.Lstat(szPath) }
func (LocalBackend) ReadDir(szPath string) ([]fs.FileInfo, error) {
	slEntries, errRead := os.ReadDir(szPath)
	if errRead != nil {
		return nil, errRead
	}
	slInfos := make([]fs.FileInfo, 0, len(slEntries))
	for _, entry := range slEntries {
		info, errInfo := entry.Info()
		if errInfo != nil {
			continue // 읽는 사이 사라진 항목
		}
		slInfos = append(slInfos, info)
	}
	return slInfos, nil
}
func (LocalBackend) OpenRead(szPath string) (io.ReadCloser, error) { return os.Open(szPath) }
func (LocalBackend) OpenWrite(szPath string) (io.WriteCloser, error) {
	return os.OpenFile(szPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}
func (LocalBackend) Mkdir(szPath string) error                   { return os.Mkdir(szPath, 0o755) }
func (LocalBackend) MkdirAll(szPath string) error                { return os.MkdirAll(szPath, 0o755) }
func (LocalBackend) Rename(szOld string, szNew string) error     { return os.Rename(szOld, szNew) }
func (LocalBackend) Chmod(szPath string, mode fs.FileMode) error { return os.Chmod(szPath, mode) }
func (LocalBackend) RemoveFile(szPath string) error              { return os.Remove(szPath) }
func (LocalBackend) RemoveDir(szPath string) error               { return os.Remove(szPath) }
func (LocalBackend) Join(szDir string, szName string) string     { return filepath.Join(szDir, szName) }
func (LocalBackend) Base(szPath string) string                   { return filepath.Base(szPath) }
func (LocalBackend) Dir(szPath string) string                    { return filepath.Dir(szPath) }
func (LocalBackend) Separator() string                           { return string(filepath.Separator) }

// Replace 는 os.Rename 이다 (Windows·Unix 모두 대상 파일이 있으면 대신한다).
func (LocalBackend) Replace(szOld string, szNew string) error { return os.Rename(szOld, szNew) }
func (LocalBackend) OwnerKey(info fs.FileInfo) string         { return localOwnerKey(info) }

// isHomeRef 는 "~", "~/x", "~\x" 인지와 나머지 부분을 돌려준다.
func isHomeRef(szPath string) (bool, string) {
	if szPath == "~" {
		return true, ""
	}
	if strings.HasPrefix(szPath, "~/") || strings.HasPrefix(szPath, `~\`) {
		return true, szPath[2:]
	}
	return false, ""
}
