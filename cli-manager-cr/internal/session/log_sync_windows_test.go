//go:build windows

package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// listedSize 는 폴더 목록(탐색기가 보는 값)에 적힌 파일 크기다. Windows 는 파일이 열려 있는 동안
// 내려 쓰기(fsync)나 닫기 전에는 이 값을 갱신하지 않는다.
func listedSize(t *testing.T, szPath string) int64 {
	t.Helper()
	slEntries, errRead := os.ReadDir(filepath.Dir(szPath))
	if errRead != nil {
		t.Fatal(errRead)
	}
	for _, entry := range slEntries {
		if entry.Name() == filepath.Base(szPath) {
			pInfo, errInfo := entry.Info()
			if errInfo != nil {
				t.Fatal(errInfo)
			}
			return pInfo.Size()
		}
	}
	t.Fatalf("목록에 없음: %s", szPath)
	return 0
}

// 내려 쓰기 주기가 지나면 파일을 닫지 않아도 폴더 목록의 크기가 실제 크기로 바뀐다 (D100).
func TestLogSyncUpdatesListedSize(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(1)
	pLogger.Write("0123456789\n")
	waitUntil(t, func() bool { return listedSize(t, pLogger.Path()) == 11 }, 4*time.Second)

	pLogger.SetSyncSec(0)
	pLogger.Write("abc\n")
	if iSize := listedSize(t, pLogger.Path()); iSize != 15 {
		t.Fatalf("즉시 모드는 쓰자마자 목록 크기가 맞아야 함: %d", iSize)
	}
}
