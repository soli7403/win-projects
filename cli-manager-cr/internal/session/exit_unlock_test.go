package session

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dariterm/internal/transport"
)

// exitingTransport 는 서버가 셸을 정상 종료한 것처럼 Read 에서 transport.ErrRemoteExited 를 돌려준다.
type exitingTransport struct {
	*fakeTransport
	chExit  chan struct{}
	iClosed atomic.Int32
}

func newExitingTransport() *exitingTransport {
	return &exitingTransport{fakeTransport: newFakeTransport(), chExit: make(chan struct{})}
}

func (pFake *exitingTransport) Read(bBuf []byte) (int, error) {
	select {
	case bData := <-pFake.chIn:
		return copy(bBuf, bData), nil
	case <-pFake.chExit:
		return 0, transport.ErrRemoteExited
	case <-pFake.chClosed:
		return 0, transport.ErrRemoteClosed
	}
}

func (pFake *exitingTransport) Close() error {
	pFake.iClosed.Add(1)
	pFake.drop()
	return nil
}

// countingDialer 는 호출 수를 세며 fnNext 의 결과를 돌려준다.
type countingDialer struct {
	mtxData sync.Mutex
	iCalls  int
	fnNext  func(iCall int) (Transport, error)
}

func (pDial *countingDialer) dial(func(string)) (Transport, error) {
	pDial.mtxData.Lock()
	pDial.iCalls++
	iCall := pDial.iCalls
	pDial.mtxData.Unlock()
	return pDial.fnNext(iCall)
}

func (pDial *countingDialer) calls() int {
	pDial.mtxData.Lock()
	defer pDial.mtxData.Unlock()
	return pDial.iCalls
}

func lastState(pSession *Session) string {
	return pSession.Snapshot().M_szState
}

// 서버가 셸을 정상 종료하면 자동 재접속하지 않고 닫힘 + 안내 두 줄. Enter 로는 다시 연결한다 (session.md §4, 안 F).
func TestRemoteExitDoesNotAutoReconnect(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	second := newFakeTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return second, nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "Enter 키를 누르면 다시 연결합니다") }, time.Second)
	szOut := rec.output()
	if !strings.Contains(szOut, "서버가 세션을 끝냈습니다.") {
		t.Fatalf("정상 종료 안내가 없음: %q", szOut)
	}
	if strings.Contains(szOut, "자동 재연결") || strings.Contains(szOut, "연결이 끊어졌습니다") {
		t.Fatalf("정상 종료는 끊김·재접속 안내가 없어야 함: %q", szOut)
	}
	if first.iClosed.Load() == 0 {
		t.Fatal("남은 연결을 닫아야 함")
	}
	time.Sleep(2500 * time.Millisecond) // 끊김이었다면 2초 뒤 재접속했을 시간
	if pDial.calls() != 1 {
		t.Fatalf("자동 재접속하면 안 됨, 호출 %d회", pDial.calls())
	}
	if pSession.Snapshot().M_szMessage != "" || pSession.attemptsForTest() != 0 {
		t.Fatal("재시도 횟수·상태 메시지는 비어 있어야 함")
	}

	pSession.Input("\r")
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	if pDial.calls() != 2 {
		t.Fatalf("Enter 로 한 번 다시 접속해야 함, 호출 %d회", pDial.calls())
	}
}

// 입력 없이 한참 있다가 서버가 셸을 끝내면(유휴 로그아웃 등) 끊김처럼 자동 재접속한다 (session.md §4, D98).
func TestRemoteExitAfterIdleAutoReconnects(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.Input("ls\r")
	pSession.backdateForTest(time.Minute) // 연결도 마지막 입력도 1분 전

	close(first.chExit)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "초 후 자동 재연결합니다") }, time.Second)
	szOut := rec.output()
	if !strings.Contains(szOut, "서버가 세션을 끝냈습니다.") {
		t.Fatalf("서버 종료 안내가 없음: %q", szOut)
	}
	if first.iClosed.Load() == 0 {
		t.Fatal("남은 연결을 닫아야 함")
	}
	waitUntil(t, func() bool { return pDial.calls() == 2 && lastState(pSession) == G_cszStateConnected }, 4*time.Second)
}

// 끝나기 직전에 사용자 입력이 있었으면(exit·logout) 오래 연결된 세션이어도 자동 재접속하지 않는다 (session.md §4, D98).
func TestRemoteExitRightAfterInputStaysClosed(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.backdateForTest(time.Minute)
	pSession.Input("exit\r") // 방금 입력

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	time.Sleep(2500 * time.Millisecond)
	if pDial.calls() != 1 {
		t.Fatalf("사용자가 끝낸 세션은 자동 재접속하면 안 됨, 호출 %d회", pDial.calls())
	}
	if strings.Contains(rec.output(), "자동 재연결") {
		t.Fatalf("재접속 안내가 없어야 함: %q", rec.output())
	}
}

// 자동 재접속 설정이 꺼져 있으면 유휴 뒤 서버 종료도 닫힘으로 둔다.
func TestRemoteExitAfterIdleWithAutoReconnectOff(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(int) (Transport, error) { return first, nil }}
	info, _ := pManager.Open(sshConfig(pDial.dial, false))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.backdateForTest(time.Minute)

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "Enter 키를 누르면 다시 연결합니다") }, time.Second)
}

// 끊김(정상 종료가 아님)은 지금처럼 자동 재접속을 예약한다.
func TestDropStillAutoReconnects(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	first.drop() // ErrRemoteClosed
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "초 후 자동 재연결합니다") }, time.Second)
	if lastState(pSession) != G_cszStateReconnecting {
		t.Fatalf("끊김은 재접속 중이어야 함: %s", lastState(pSession))
	}
}

// 다이얼러가 ErrWaitUnlock 이면 실패로 세지 않고 "재접속 중"으로 기다리다가, 잠금 해제 재개로 곧바로 접속한다 (안 A).
func TestWaitUnlockThenResume(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	var bLocked atomic.Bool
	bLocked.Store(true)
	pDial := &countingDialer{fnNext: func(int) (Transport, error) {
		if bLocked.Load() {
			return nil, ErrWaitUnlock
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "앱 잠금을 해제하면 다시 연결합니다.") }, time.Second)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateReconnecting }, time.Second)
	if rec.hasState(G_cszStateError) {
		t.Fatal("처음 열기도 오류 상태로 가면 안 됨")
	}
	if pSession.attemptsForTest() != 0 {
		t.Fatal("대기는 재시도 횟수에 넣지 않음")
	}
	// Enter 로 다시 시도해도(아직 잠김) 안내 줄은 한 번만.
	pSession.Input("\r")
	waitUntil(t, func() bool { return pDial.calls() == 2 }, time.Second)
	waitUntil(t, func() bool { return !pSession.dialingForTest() }, time.Second)
	if iCount := strings.Count(rec.output(), "앱 잠금을 해제하면"); iCount != 1 {
		t.Fatalf("대기 안내는 한 번만, %d번", iCount)
	}
	time.Sleep(2500 * time.Millisecond)
	if pDial.calls() != 2 {
		t.Fatalf("대기 중에는 타이머로 재접속하지 않음, 호출 %d회", pDial.calls())
	}

	bLocked.Store(false)
	if pManager.ResumeUnlockWaiters() != 1 {
		t.Fatal("기다리던 세션 1개를 재개해야 함")
	}
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	if pManager.ResumeUnlockWaiters() != 0 {
		t.Fatal("연결된 세션은 재개 대상이 아님")
	}
}

// 대기 중에 닫은 세션은 재개하지 않고, 사본 지우기(OnClose)는 한 번 불린다.
func TestWaitUnlockClosedIsIgnored(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	pDial := &countingDialer{fnNext: func(int) (Transport, error) { return nil, ErrWaitUnlock }}
	var iOnClose atomic.Int32
	cfg := sshConfig(pDial.dial, true)
	cfg.OnClose = func() { iOnClose.Add(1) }
	info, _ := pManager.Open(cfg)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.waitingForTest() }, time.Second)
	pManager.Close(info.M_szId)
	pSession.Close()
	if pSession.resumeIfWaiting() {
		t.Fatal("닫힌 세션은 재개하지 않음")
	}
	if iOnClose.Load() != 1 {
		t.Fatalf("OnClose 는 한 번, %d번", iOnClose.Load())
	}
}

// 대기 중 Enter 와 잠금 해제 재개가 겹쳐도 접속은 한 번만 한다.
func TestWaitUnlockResumeAndEnterRace(t *testing.T) {
	for iRun := 0; iRun < 20; iRun++ {
		rec := &eventRecorder{}
		pManager := NewManager(rec.emit)
		var bLocked atomic.Bool
		bLocked.Store(true)
		chRelease := make(chan struct{})
		pDial := &countingDialer{fnNext: func(int) (Transport, error) {
			if bLocked.Load() {
				return nil, ErrWaitUnlock
			}
			<-chRelease // 접속이 끝나기 전에 다른 요청이 겹치게 붙잡는다
			return newFakeTransport(), nil
		}}
		info, _ := pManager.Open(sshConfig(pDial.dial, true))
		pSession, _ := pManager.Get(info.M_szId)
		waitUntil(t, func() bool { return pSession.waitingForTest() && !pSession.dialingForTest() }, time.Second)
		bLocked.Store(false)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); pManager.ResumeUnlockWaiters() }()
		go func() { defer wg.Done(); pSession.Input("\r") }()
		wg.Wait()
		waitUntil(t, func() bool { return pDial.calls() >= 2 }, time.Second)
		time.Sleep(20 * time.Millisecond)
		close(chRelease)
		waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
		if pDial.calls() != 2 {
			t.Fatalf("run %d: 접속은 한 번만 겹쳐야 함(대기 1 + 재개 1), 호출 %d회", iRun, pDial.calls())
		}
		pManager.Close(info.M_szId)
	}
}

// backdateForTest 는 연결 시각과 마지막 입력 시각(있으면)을 durAgo 만큼 과거로 옮긴다.
func (pSession *Session) backdateForTest(durAgo time.Duration) {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	pSession.tConnected = pSession.tConnected.Add(-durAgo)
	if !pSession.tLastInput.IsZero() {
		pSession.tLastInput = pSession.tLastInput.Add(-durAgo)
	}
}

func (pSession *Session) attemptsForTest() int {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return pSession.attempts
}

func (pSession *Session) dialingForTest() bool {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return pSession.dialing
}

func (pSession *Session) waitingForTest() bool {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return pSession.waitUnlock
}
