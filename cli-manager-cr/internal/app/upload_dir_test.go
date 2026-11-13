package app

import (
	"os"
	"path/filepath"
	"testing"

	"dariterm/internal/sftpfs"
)

// TestEnsureUploadDirNested 는 상위 폴더까지 없는 업로드 폴더(기본 ~/.dariterm/uploads 같은)를 만들고,
// 이미 있으면 그대로 두는지 확인한다 (behavior/terminal.md §5, acceptance T9).
func TestEnsureUploadDirNested(t *testing.T) {
	szHome := t.TempDir()
	szDir := filepath.Join(szHome, ".dariterm", "uploads")
	pFs := sftpfs.NewLocalFs(szHome)
	if errDir := ensureUploadDir(pFs, szDir); errDir != nil {
		t.Fatalf("중간 폴더까지 만들어야 함: %v", errDir)
	}
	if pInfo, errStat := os.Stat(szDir); errStat != nil || !pInfo.IsDir() {
		t.Fatalf("업로드 폴더가 없음: %v", errStat)
	}
	if errAgain := ensureUploadDir(pFs, szDir); errAgain != nil {
		t.Fatalf("이미 있으면 오류가 아니어야 함: %v", errAgain)
	}
}

// TestEnsureUploadDirBlocked 는 같은 이름의 파일이 막고 있으면 폴더 경로가 든 오류를 내는지 확인한다.
func TestEnsureUploadDirBlocked(t *testing.T) {
	szHome := t.TempDir()
	if errWrite := os.WriteFile(filepath.Join(szHome, ".dariterm"), []byte("x"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	szDir := filepath.Join(szHome, ".dariterm", "uploads")
	errDir := ensureUploadDir(sftpfs.NewLocalFs(szHome), szDir)
	if errDir == nil {
		t.Fatal("파일이 막고 있으면 오류여야 함")
	}
	if want := "업로드 폴더를 만들지 못했습니다 (" + szDir + ")"; len(errDir.Error()) < len(want) || errDir.Error()[:len(want)] != want {
		t.Fatalf("오류 문구 불일치: %v", errDir)
	}
}
