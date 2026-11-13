package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDataDirEnv 는 DARITERM_DATA 가 있으면 그 폴더를 만들어 쓰는지 확인한다 (sync-backup.md §1).
func TestDataDirEnv(t *testing.T) {
	szDir := filepath.Join(t.TempDir(), "custom", "data")
	t.Setenv(G_cszDataEnv, szDir)
	szGot, errDir := DataDir()
	if errDir != nil || szGot != szDir {
		t.Fatalf("DataDir = %q, %v", szGot, errDir)
	}
	if pInfo, errStat := os.Stat(szDir); errStat != nil || !pInfo.IsDir() {
		t.Fatalf("폴더가 만들어지지 않음: %v", errStat)
	}
}
