package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"termgo/internal/session"
)

// TestAutoLockRules 는 자동 잠금 판정표다 (acceptance A1~A4, A5 의 판정 부분).
func TestAutoLockRules(t *testing.T) {
	if ShouldAutoLock(0, true, false, true, 999999999) {
		t.Fatal("A3: 자동 잠금 0 이면 잠기면 안 됨")
	}
	if ShouldAutoLock(1, false, false, true, 999999999) {
		t.Fatal("이미 잠긴 볼트는 다시 잠그지 않음")
	}
	if !ShouldAutoLock(1, true, false, true, 60*1000) {
		t.Fatal("A1: 1분 유휴면 잠가야 함")
	}
	if ShouldAutoLock(1, true, false, true, 59*1000) {
		t.Fatal("1분 미만 유휴면 잠기면 안 됨")
	}
	if ShouldAutoLock(1, true, false, false, 999999999) {
		t.Fatal("입력 시각을 모르면 유휴로 잠그면 안 됨")
	}
	if !ShouldAutoLock(5, true, true, false, 0) {
		t.Fatal("A5: 잠금·절전 신호가 있으면 잠가야 함")
	}
	if !SuspendDetected(time.Now().Add(-10*time.Minute), time.Now()) || SuspendDetected(time.Now().Add(-20*time.Second), time.Now()) {
		t.Fatal("절전 추정 간격 판정 불일치")
	}
}

// TestCloseConfirmText 는 종료 확인 문구 원문을 확인한다 (acceptance X4).
func TestCloseConfirmText(t *testing.T) {
	if got := CloseConfirmText(2, 0); got != "연결된 세션 2개가 있습니다. 모두 닫고 종료할까요?" {
		t.Fatalf("기본 문구 불일치: %q", got)
	}
	got := CloseConfirmText(3, 1)
	if !strings.HasPrefix(got, "연결된 세션 3개가 있습니다. 모두 닫고 종료할까요?\n\n세션 유지(tmux) 중인 창 1개는") || !strings.HasSuffix(got, "완전히 끝내려면 그 창에서 exit 를 입력하세요.") {
		t.Fatalf("세션 유지 문구 불일치: %q", got)
	}
}

// TestWindowPlacement 는 시작 위치 판정과 최대화 보존을 확인한다 (acceptance X5, X6).
func TestWindowPlacement(t *testing.T) {
	if _, _, bCenter := StartupPlacement(WindowState{}, 1920, 1080); !bCenter {
		t.Fatal("저장된 위치가 없으면 가운데")
	}
	if _, _, bCenter := StartupPlacement(WindowState{M_iX: 5000, M_iY: 100, M_bHasPos: true}, 1920, 1080); !bCenter {
		t.Fatal("X6: 화면 밖 위치는 가운데")
	}
	if x, y, bCenter := StartupPlacement(WindowState{M_iX: 100, M_iY: 50, M_bHasPos: true}, 1920, 1080); bCenter || x != 100 || y != 50 {
		t.Fatal("화면 안 위치는 그대로")
	}
	prev := WindowState{M_iX: 10, M_iY: 20, M_iW: 1000, M_iH: 700, M_bHasPos: true}
	got := CloseState(prev, true, 0, 0, 1920, 1080, true)
	if !got.M_bMaximised || got.M_iW != 1000 || got.M_iX != 10 {
		t.Fatalf("X5: 최대화 종료는 일반 크기·위치를 보존해야 함: %+v", got)
	}
	if !(WindowState{M_iW: 400, M_iH: 300}).Valid() || (WindowState{M_iW: 399, M_iH: 300}).Valid() {
		t.Fatal("유효 크기 경계 불일치")
	}
}

// TestWindowStateFileRoundTrip 은 window.json 저장과 읽기를 확인한다.
func TestWindowStateFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ws := WindowState{M_iX: 1, M_iY: 2, M_iW: 800, M_iH: 600, M_bHasPos: true}
	if errSave := SaveWindowState(dir, ws); errSave != nil {
		t.Fatal(errSave)
	}
	back, bOk := LoadWindowState(dir)
	if !bOk || back != ws {
		t.Fatalf("왕복 불일치: %+v %v", back, bOk)
	}
}

// TestIsQuitChoice 는 종료 판정 반환값 표를 확인한다 (lifecycle.md §5, acceptance X4a).
func TestIsQuitChoice(t *testing.T) {
	for _, szQuit := range []string{"종료", "Yes", "예", "예(Y)"} {
		if !IsQuitChoice(szQuit, false) {
			t.Fatalf("종료로 인식해야 함: %q", szQuit)
		}
	}
	for _, szCancel := range []string{"취소", "No", "아니요(N)", "아니요", "", "Cancel", "종료 "} {
		if IsQuitChoice(szCancel, false) {
			t.Fatalf("취소로 봐야 함: %q", szCancel)
		}
	}
	if IsQuitChoice("Yes", true) || IsQuitChoice("종료", true) {
		t.Fatal("오류가 있으면 값과 무관하게 취소여야 함")
	}
}

// TestNeedsCloseConfirm 은 세션이 없으면 확인 없이 닫히는지 본다 (acceptance X4b, X4).
func TestNeedsCloseConfirm(t *testing.T) {
	if NeedsCloseConfirm(0) {
		t.Fatal("X4b: 세션이 없으면 확인하면 안 됨")
	}
	if !NeedsCloseConfirm(2) {
		t.Fatal("X4: 세션이 있으면 확인해야 함")
	}
}

// TestCloseReturnsPromptly 는 세션이 없을 때 종료 정리가 제한 시간 안에 끝나고 store 를 저장하는지 본다.
func TestCloseReturnsPromptly(t *testing.T) {
	dir := t.TempDir()
	pService, errNew := NewService(dir, nil)
	if errNew != nil {
		t.Fatal(errNew)
	}
	tmStart := time.Now()
	if errClose := pService.Close(); errClose != nil {
		t.Fatal(errClose)
	}
	if time.Since(tmStart) > G_cdurCloseWait+time.Second {
		t.Fatalf("종료 정리가 제한 시간을 넘김: %v", time.Since(tmStart))
	}
	if _, errStat := os.Stat(filepath.Join(dir, "store.json")); errStat != nil {
		t.Fatalf("종료 시 store 가 저장되지 않음: %v", errStat)
	}
}

// TestCloseCountsConnectedOnly 은 확인 문구 집계가 connected 세션만 세는지 본다 (acceptance X4).
func TestCloseCountsConnectedOnly(t *testing.T) {
	slMixed := []session.Info{
		{M_szState: session.G_cszStateConnected},
		{M_szState: session.G_cszStateConnected, M_bKeepSession: true},
		{M_szState: session.G_cszStateError, M_bKeepSession: true},
		{M_szState: session.G_cszStateError},
		{M_szState: session.G_cszStateError},
	}
	slConnected := ConnectedSessions(slMixed)
	if len(slConnected) != 2 {
		t.Fatalf("connected 2개여야 함: %d", len(slConnected))
	}
	if NeedsCloseConfirm(len(slConnected)) != true {
		t.Fatal("connected 2개면 확인해야 함")
	}
	if CountKeepSessions(slConnected) != 1 {
		t.Fatalf("keep 집계는 connected 중 keepSession 만: %d", CountKeepSessions(slConnected))
	}
	if got := CloseConfirmText(len(slConnected), CountKeepSessions(slConnected)); !strings.HasPrefix(got, "연결된 세션 2개가 있습니다.") {
		t.Fatalf("문구의 세션 수가 connected 기준이어야 함: %q", got)
	}

	slFailed := []session.Info{{M_szState: session.G_cszStateError}, {M_szState: session.G_cszStateError}, {M_szState: session.G_cszStateError}}
	if NeedsCloseConfirm(len(ConnectedSessions(slFailed))) {
		t.Fatal("connected 가 0개면 실패 세션이 있어도 확인 없이 종료해야 함")
	}
	if n := len(ConnectedSessions(nil)); n != 0 {
		t.Fatalf("빈 목록 집계 오류: %d", n)
	}
}
