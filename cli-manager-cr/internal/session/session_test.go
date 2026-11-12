package session

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeTransport 는 테스트용 연결이다. drop 으로 끊김을 흉내 낸다.
type fakeTransport struct {
	chIn     chan []byte
	chClosed chan struct{}
	onceDrop sync.Once
	mtxData  sync.Mutex
	slWrites []string
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{chIn: make(chan []byte, 16), chClosed: make(chan struct{})}
}

func (pFake *fakeTransport) Read(bBuf []byte) (int, error) {
	select {
	case bData := <-pFake.chIn:
		return copy(bBuf, bData), nil
	case <-pFake.chClosed:
		return 0, io.EOF
	}
}

func (pFake *fakeTransport) Write(bData []byte) error {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	pFake.slWrites = append(pFake.slWrites, string(bData))
	return nil
}

func (pFake *fakeTransport) Resize(iCols int, iRows int) error { return nil }

func (pFake *fakeTransport) Close() error {
	pFake.drop()
	return nil
}

func (pFake *fakeTransport) drop() {
	pFake.onceDrop.Do(func() { close(pFake.chClosed) })
}

type eventRecorder struct {
	mtxData  sync.Mutex
	slEvents []string
	slData   []string
	slStates []string
}

func (pRec *eventRecorder) emit(szEvent string, anyData ...any) {
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	pRec.slEvents = append(pRec.slEvents, szEvent)
	if szEvent == "session.data" && len(anyData) == 2 {
		pRec.slData = append(pRec.slData, anyData[1].(string))
	}
	if szEvent == "session.state" {
		pRec.slStates = append(pRec.slStates, anyData[0].(Info).M_szState)
	}
}

func (pRec *eventRecorder) output() string {
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	return strings.Join(pRec.slData, "")
}

func (pRec *eventRecorder) hasState(szState string) bool {
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	for _, szItem := range pRec.slStates {
		if szItem == szState {
			return true
		}
	}
	return false
}

// waitUntil 은 조건이 참이 될 때까지 기다린다 (시간 초과 시 실패).
func waitUntil(t *testing.T, fnCond func() bool, durTimeout time.Duration) {
	t.Helper()
	durEnd := time.Now().Add(durTimeout)
	for time.Now().Before(durEnd) {
		if fnCond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("조건이 시간 안에 참이 되지 않음")
}

func sshConfig(dialer Dialer, autoReconnect bool) OpenConfig {
	return OpenConfig{
		Info:          Info{M_szTitle: "web", M_szProtocol: "ssh", M_szTarget: "root@10.0.0.1:22"},
		Encoding:      "utf-8",
		AutoReconnect: autoReconnect,
		Dialer:        dialer,
		OpenDelay:     time.Millisecond,
	}
}

// 접속 → 연결됨 → 출력 묶음 전달 → 닫기 시 끊김 안내 없이 closed. session.md §2·§3·§8.
func TestConnectOutputAndClose(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	transport := newFakeTransport()
	info, errOpen := pManager.Open(sshConfig(func(func(string)) (Transport, error) { return transport, nil }, true))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	waitUntil(t, func() bool { return rec.hasState(G_cszStateConnected) }, time.Second)
	transport.chIn <- []byte("hello")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "hello") }, time.Second)
	if !strings.Contains(rec.output(), "접속 중...") {
		t.Fatal("접속 시작 안내 줄이 나와야 함")
	}
	pManager.Close(info.M_szId)
	waitUntil(t, func() bool { return rec.hasState(G_cszStateClosed) }, time.Second)
	if strings.Contains(rec.output(), "연결 끊김") || strings.Contains(rec.output(), "연결이 끊어졌습니다") {
		t.Fatal("사용자가 닫은 세션은 끊김 안내가 없어야 함")
	}
}

// 처음 접속이 실패하면 자동 재접속 없이 error 상태 (session.md §4).
func TestFirstConnectFailureIsErrorWithoutRetry(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	var iCalls int
	var mtxCalls sync.Mutex
	_, errOpen := pManager.Open(sshConfig(func(func(string)) (Transport, error) {
		mtxCalls.Lock()
		iCalls++
		mtxCalls.Unlock()
		return nil, errors.New("dial tcp: connection refused")
	}, true))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	waitUntil(t, func() bool { return rec.hasState(G_cszStateError) }, time.Second)
	waitUntil(t, func() bool {
		return strings.Contains(rec.output(), "서버가 접속을 거부했습니다 (포트 확인)")
	}, time.Second)
	if !strings.Contains(rec.output(), "서버가 접속을 거부했습니다 (포트 확인)") {
		t.Fatalf("오류 문구 번역 누락: %q", rec.output())
	}
	time.Sleep(50 * time.Millisecond)
	mtxCalls.Lock()
	defer mtxCalls.Unlock()
	if iCalls != 1 {
		t.Fatalf("자동 재접속하면 안 됨, 호출 %d회", iCalls)
	}
}

// 끊긴 뒤 Enter 를 입력하면 즉시 재접속한다 (session.md §4 수동 재접속).
func TestDropThenEnterReconnectsNow(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newFakeTransport()
	second := newFakeTransport()
	var iCalls int
	var mtxCalls sync.Mutex
	info, _ := pManager.Open(sshConfig(func(func(string)) (Transport, error) {
		mtxCalls.Lock()
		defer mtxCalls.Unlock()
		iCalls++
		if iCalls == 1 {
			return first, nil
		}
		return second, nil
	}, true))
	waitUntil(t, func() bool { return rec.hasState(G_cszStateConnected) }, time.Second)
	first.drop()
	waitUntil(t, func() bool { return rec.hasState(G_cszStateReconnecting) }, time.Second)

	pSession, errGet := pManager.Get(info.M_szId)
	if errGet != nil {
		t.Fatal(errGet)
	}
	pSession.Input("\r")
	waitUntil(t, func() bool {
		mtxCalls.Lock()
		defer mtxCalls.Unlock()
		return iCalls == 2
	}, time.Second)
	pManager.Close(info.M_szId)
}

// 출력 묶음 전송: 짧은 시간의 여러 조각이 한 번의 session.data 로 합쳐진다 (12ms).
func TestOutputIsBatched(t *testing.T) {
	rec := &eventRecorder{}
	var mtxEmit sync.Mutex
	var iDataEvents int
	batcher := NewBatcher(func(szText string) {
		mtxEmit.Lock()
		iDataEvents++
		mtxEmit.Unlock()
		rec.emit("session.data", "id", szText)
	})
	batcher.Add("a")
	batcher.Add("b")
	batcher.Add("c")
	waitUntil(t, func() bool { return rec.output() == "abc" }, time.Second)
	mtxEmit.Lock()
	defer mtxEmit.Unlock()
	if iDataEvents != 1 {
		t.Fatalf("묶음 전송은 한 번이어야 함: %d", iDataEvents)
	}
}

func TestBackoffAndReconnectRules(t *testing.T) {
	want := []int{2, 4, 8, 16, 30, 30, 30, 30}
	for iIndex, iWant := range want {
		if got := BackoffSec(iIndex); got != iWant {
			t.Fatalf("BackoffSec(%d)=%d, want %d", iIndex, got, iWant)
		}
	}
	if CanAutoReconnect(true, false, false, 0) {
		t.Fatal("처음 접속 실패는 자동 재접속 안 함")
	}
	if !CanAutoReconnect(true, false, true, 0) {
		t.Fatal("끊김 뒤에는 자동 재접속")
	}
	if CanAutoReconnect(true, true, true, 0) {
		t.Fatal("시리얼은 자동 재접속 안 함")
	}
	if CanAutoReconnect(true, false, true, 8) {
		t.Fatal("8회 넘으면 중단")
	}
	if CanAutoReconnect(false, false, true, 0) {
		t.Fatal("설정이 꺼져 있으면 중단")
	}
}

func TestTranslateError(t *testing.T) {
	cases := map[string]string{
		"dial tcp: i/o timeout":                  "응답 시간 초과 (주소·방화벽 확인)",
		"ssh: unable to authenticate, attempted": "인증 실패 (사용자 이름/비밀번호/키 확인)",
		"CreateFile: Access is denied.":          "CreateFile: Access is denied. (포트 사용 중이거나 권한 없음)",
		"something else":                         "something else",
	}
	for szRaw, szWant := range cases {
		if got := TranslateError(szRaw); got != szWant {
			t.Fatalf("TranslateError(%q)=%q, want %q", szRaw, got, szWant)
		}
	}
}

func TestTmuxSessionNameAndLine(t *testing.T) {
	if got := TmuxSessionName("", "my.web server", ""); got != "my-web-server" {
		t.Fatalf("이름 치환: %q", got)
	}
	if got := TmuxSessionName("~", "", ""); got != "termgo" {
		t.Fatalf("빈 이름은 termgo: %q", got)
	}
	szLine := TmuxAttachLine("dev", "echo 'hi'\nls")
	if !strings.HasPrefix(szLine, " sh -c ") || !strings.HasSuffix(szLine, "\r") {
		t.Fatalf("한 줄 형식 오류: %q", szLine)
	}
	if strings.Contains(szLine, "exec tmux") {
		t.Fatal("exec 를 쓰면 안 됨")
	}
	if !strings.Contains(szLine, "tmux send-keys") {
		t.Fatal("자동 실행 명령 입력이 있어야 함")
	}
}

func TestLogStripAndFileName(t *testing.T) {
	if got := StripForLog("\x1b[31mred\x1b[0m\r\nok"); got != "red\nok" {
		t.Fatalf("ANSI·CR 제거: %q", got)
	}
	szDir := t.TempDir()
	pLogger, errOpen := OpenLog(szDir, "a/b:c", time.Date(2026, 11, 10, 9, 5, 0, 0, time.Local))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	pLogger.Write("x")
	_ = pLogger.Close()
	if _, errStat := filepath.Glob(filepath.Join(szDir, "logs", "a_b_c-20261110-0905.log")); errStat != nil {
		t.Fatal(errStat)
	}
	matches, _ := filepath.Glob(filepath.Join(szDir, "logs", "a_b_c-20261110-0905.log"))
	if len(matches) != 1 {
		t.Fatalf("로그 파일 이름 규칙 불일치")
	}
}

func TestDecoderKeepsSplitCharacters(t *testing.T) {
	dec := NewDecoder("utf-8")
	bFull := []byte("가나")
	out := dec.Decode(bFull[:4]) + dec.Decode(bFull[4:])
	if out != "가나" {
		t.Fatalf("UTF-8 잘린 글자 이어 붙이기 실패: %q", out)
	}
	// EUC-KR: "한" = B9 AB
	dec2 := NewDecoder("euc-kr")
	out2 := dec2.Decode([]byte{0xB9}) + dec2.Decode([]byte{0xAB}) // 0xB9 0xAB = 무
	if out2 != "무" {
		t.Fatalf("EUC-KR 잘린 글자 이어 붙이기 실패: %q", out2)
	}
	if got := Encode("euc-kr", "무"); len(got) != 2 || got[0] != 0xB9 || got[1] != 0xAB {
		t.Fatalf("EUC-KR 인코딩 실패: % x", got)
	}
}
