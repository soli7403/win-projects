package logclean

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// makeFile 은 szName 파일을 만들고 수정 시각을 tMod 로 둔다.
func makeFile(t *testing.T, szDir string, szName string, tMod time.Time) string {
	t.Helper()
	szPath := filepath.Join(szDir, szName)
	if errWrite := os.WriteFile(szPath, []byte("x"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errTime := os.Chtimes(szPath, tMod, tMod); errTime != nil {
		t.Fatal(errTime)
	}
	return szPath
}

func exists(szPath string) bool {
	_, errStat := os.Stat(szPath)
	return errStat == nil
}

// 보관 일수보다 오래된 .log 만 지우고, 새 파일·다른 확장자·하위 폴더·기록 중인 파일은 남긴다 (session.md §6 보관 기간).
func TestCleanOnceRemovesOnlyOldLogs(t *testing.T) {
	szDir := t.TempDir()
	tNow := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	szOld := makeFile(t, szDir, "web-20261001-100000.log", tNow.Add(-31*24*time.Hour))
	szOldUpper := makeFile(t, szDir, "OLD.LOG", tNow.Add(-40*24*time.Hour))
	szNew := makeFile(t, szDir, "web-20261115-100000.log", tNow.Add(-29*24*time.Hour))
	szOther := makeFile(t, szDir, "notes.txt", tNow.Add(-90*24*time.Hour))
	szInUse := makeFile(t, szDir, "busy-20261001-100000.log", tNow.Add(-60*24*time.Hour))
	szSub := filepath.Join(szDir, "sub")
	if errMkdir := os.Mkdir(szSub, 0o700); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	szSubLog := makeFile(t, szSub, "deep.log", tNow.Add(-90*24*time.Hour))

	result, errClean := CleanOnce(szDir, 30, tNow, func(szPath string) bool { return szPath == szInUse })
	if errClean != nil {
		t.Fatal(errClean)
	}
	if result.M_iRemoved != 2 || result.M_iFailed != 0 {
		t.Fatalf("2개를 지워야 함: %+v", result)
	}
	if exists(szOld) || exists(szOldUpper) {
		t.Fatal("오래된 로그가 남음")
	}
	for _, szKeep := range []string{szNew, szOther, szInUse, szSubLog} {
		if !exists(szKeep) {
			t.Fatalf("남아야 할 파일이 지워짐: %s", szKeep)
		}
	}
}

// 0 이하이면 지우지 않고, 폴더가 없으면 오류 없이 빈 결과다.
func TestCleanOnceDisabledAndMissingDir(t *testing.T) {
	szDir := t.TempDir()
	tNow := time.Now()
	szOld := makeFile(t, szDir, "a.log", tNow.Add(-1000*24*time.Hour))
	if result, _ := CleanOnce(szDir, 0, tNow, nil); result.M_iRemoved != 0 || !exists(szOld) {
		t.Fatal("0 일이면 지우지 않아야 함")
	}
	if result, errClean := CleanOnce(filepath.Join(szDir, "none"), 7, tNow, nil); errClean != nil || result.M_iRemoved != 0 {
		t.Fatalf("없는 폴더: %+v %v", result, errClean)
	}
}

// 정리기는 첫 대기 뒤 정리하고, Kick 이면 주기를 기다리지 않고 다시 정리하며, Stop 으로 끝난다.
func TestCleanerRunsFirstKickAndStops(t *testing.T) {
	szDir := t.TempDir()
	var mtxDays sync.Mutex
	iDays := 30
	chRuns := make(chan Result, 8)
	pCleaner := New(szDir, func() int { mtxDays.Lock(); defer mtxDays.Unlock(); return iDays }, nil)
	pCleaner.SetTiming(10*time.Millisecond, time.Hour)
	pCleaner.SetReport(func(result Result, errClean error) { chRuns <- result })
	szTenDays := makeFile(t, szDir, "ten.log", time.Now().Add(-10*24*time.Hour))
	pCleaner.Start()
	pCleaner.Start() // 두 번 불러도 한 번만

	select {
	case result := <-chRuns:
		if result.M_iRemoved != 0 || !exists(szTenDays) {
			t.Fatalf("30일 보관이면 10일 된 파일은 남아야 함: %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("첫 정리가 돌지 않음")
	}

	mtxDays.Lock()
	iDays = 7
	mtxDays.Unlock()
	pCleaner.Kick()
	select {
	case result := <-chRuns:
		if result.M_iRemoved != 1 || exists(szTenDays) {
			t.Fatalf("7일로 바꾸면 10일 된 파일을 지워야 함: %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Kick 뒤 정리가 돌지 않음")
	}

	chStopped := make(chan struct{})
	go func() { pCleaner.Stop(); pCleaner.Stop(); close(chStopped) }()
	select {
	case <-chStopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop 이 돌아오지 않음")
	}
}

// Start 전에 Stop 해도 막히지 않는다 (앱이 시작 중 실패해 바로 닫히는 경우).
func TestCleanerStopWithoutStart(t *testing.T) {
	pCleaner := New(t.TempDir(), func() int { return 30 }, nil)
	pCleaner.Stop()
	pCleaner.Kick() // 멈춘 뒤 Kick 도 막히지 않는다
}
