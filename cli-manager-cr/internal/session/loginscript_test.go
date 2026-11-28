package session

import (
	"dariterm/internal/i18n"
	"strings"
	"testing"
	"time"
)

// 단계는 순서대로, 기다릴 글자가 나왔을 때만 넘어간다 (session.md §5.1).
func TestLoginRunnerSteps(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{
		{Expect: "$ ", Send: "sudo su -"},
		{Expect: "password", Send: "secret"},
		{Expect: "# ", Send: "cd /var/log"},
	})
	if slSends, _ := pRunner.feed("Last login: Thu\r\n"); len(slSends) != 0 {
		t.Fatalf("프롬프트 전에 보냈다: %v", slSends)
	}
	slSends, bFinished := pRunner.feed("[user@host ~]$ ")
	if len(slSends) != 1 || slSends[0] != "sudo su -" || bFinished {
		t.Fatalf("1단계 = %v, 끝=%v", slSends, bFinished)
	}
	// 대소문자를 가리지 않고, 조각이 나뉘어 와도 찾는다.
	if slSends, _ = pRunner.feed("[sudo] Pass"); len(slSends) != 0 {
		t.Fatalf("반쪽에서 보냈다: %v", slSends)
	}
	slSends, _ = pRunner.feed("word for user: ")
	if len(slSends) != 1 || slSends[0] != "secret" {
		t.Fatalf("2단계 = %v", slSends)
	}
	slSends, bFinished = pRunner.feed("\r\n[root@host ~]# ")
	if len(slSends) != 1 || slSends[0] != "cd /var/log" || !bFinished {
		t.Fatalf("3단계 = %v, 끝=%v", slSends, bFinished)
	}
	if slSends, _ = pRunner.feed("# # # "); len(slSends) != 0 {
		t.Fatalf("끝난 뒤에 보냈다: %v", slSends)
	}
}

// 한 번 맞은 글자는 다음 단계에 다시 쓰이지 않는다: 같은 프롬프트를 두 번 기다리면 두 번 나와야 한다.
func TestLoginRunnerDoesNotReuseMatch(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Expect: "$ ", Send: "a"}, {Expect: "$ ", Send: "b"}})
	slSends, bFinished := pRunner.feed("host$ ")
	if len(slSends) != 1 || bFinished {
		t.Fatalf("첫 프롬프트에서 = %v, 끝=%v", slSends, bFinished)
	}
	slSends, bFinished = pRunner.feed("host$ ")
	if len(slSends) != 1 || slSends[0] != "b" || !bFinished {
		t.Fatalf("둘째 프롬프트에서 = %v, 끝=%v", slSends, bFinished)
	}
}

// 색 시퀀스가 글자 사이에 끼어 있거나 조각 경계에서 잘려도 찾는다.
func TestLoginRunnerIgnoresAnsi(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Expect: "login:", Send: "admin"}})
	if slSends, _ := pRunner.feed("\x1b[1mlog\x1b[3"); len(slSends) != 0 {
		t.Fatalf("반쪽에서 보냈다: %v", slSends)
	}
	slSends, bFinished := pRunner.feed("2min:\x1b[0m ")
	if len(slSends) != 1 || !bFinished {
		t.Fatalf("= %v, 끝=%v", slSends, bFinished)
	}
}

// 기다릴 글자가 빈 단계는 출력 없이 바로 보낸다.
func TestLoginRunnerEmptyExpect(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Send: ""}, {Expect: ">", Send: "enable"}})
	slSends, bFinished := pRunner.feed("")
	if len(slSends) != 1 || slSends[0] != "" || bFinished {
		t.Fatalf("= %v, 끝=%v", slSends, bFinished)
	}
}

func TestTrimLoginSteps(t *testing.T) {
	slIn := []LoginStep{{}, {Expect: "a"}, {Send: "b"}}
	for iIndex := 0; iIndex < 30; iIndex++ {
		slIn = append(slIn, LoginStep{Expect: "x", Send: "y"})
	}
	slOut := TrimLoginSteps(slIn)
	if len(slOut) != G_ciLoginStepMax || slOut[0].Expect != "a" || slOut[1].Send != "b" {
		t.Fatalf("개수 = %d, 앞 = %+v", len(slOut), slOut[:2])
	}
}

// secretTransport 는 로그인 비밀번호를 알려 주는 테스트용 연결이다.
type secretTransport struct {
	*fakeTransport
	szSecret string
}

func (pFake *secretTransport) LoginSecret() string { return pFake.szSecret }

func (pFake *fakeTransport) writes() []string {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return append([]string(nil), pFake.slWrites...)
}

// waitWrites 는 보낸 입력이 iCount 개가 될 때까지 기다린다.
func waitWrites(t *testing.T, pFake *fakeTransport, iCount int) []string {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if slWrites := pFake.writes(); len(slWrites) >= iCount {
			return slWrites
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("입력 %d개를 기다렸지만 %v", iCount, pFake.writes())
	return nil
}

func openLoginSession(t *testing.T, conn Transport, cfg OpenConfig) (*Manager, *eventRecorder, string) {
	t.Helper()
	pRec := &eventRecorder{}
	pManager := NewManager(pRec.emit)
	cfg.Info = Info{M_szTitle: "login", M_szProtocol: "ssh"}
	cfg.Dialer = func(notify func(string)) (Transport, error) { return conn, nil }
	cfg.OpenDelay = time.Millisecond
	info, errOpen := pManager.Open(cfg)
	if errOpen != nil {
		t.Fatalf("Open: %v", errOpen)
	}
	return pManager, pRec, info.M_szId
}

// 세션은 출력에서 기다릴 글자를 찾아 보내고, {{password}} 를 로그인 비밀번호로 바꾸고,
// 스크립트가 끝난 뒤에 접속 후 입력을 보낸다.
func TestSessionRunsLoginScript(t *testing.T) {
	pFake := &secretTransport{fakeTransport: newFakeTransport(), szSecret: "pw!"}
	pManager, _, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "$ ", Send: "sudo su -"}, {Expect: "password", Send: G_cszLoginPassword}},
		StartupInput: "cd /srv\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)

	pFake.chIn <- []byte("Welcome\r\nuser@host:~$ ")
	if slWrites := waitWrites(t, pFake.fakeTransport, 1); slWrites[0] != "sudo su -\r" {
		t.Fatalf("1단계 입력 = %q", slWrites[0])
	}
	pFake.chIn <- []byte("[sudo] password for user: ")
	slWrites := waitWrites(t, pFake.fakeTransport, 3)
	if slWrites[1] != "pw!\r" {
		t.Errorf("2단계 입력 = %q", slWrites[1])
	}
	if slWrites[2] != "cd /srv\r" {
		t.Errorf("스크립트 뒤 접속 후 입력 = %q", slWrites[2])
	}
}

// 넣을 비밀번호가 없으면 빈 비밀번호를 보내지 않고 멈춘다.
func TestSessionLoginScriptStopsWithoutPassword(t *testing.T) {
	i18n.SetLang("ko")
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "password", Send: G_cszLoginPassword}},
		StartupInput: "never\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)
	pFake.chIn <- []byte("Password: ")
	waitOutput(t, pRec, "저장된 비밀번호가 없어")
	if slWrites := pFake.writes(); len(slWrites) != 0 {
		t.Fatalf("보내면 안 된다: %v", slWrites)
	}
}

// 기다릴 글자가 시한 안에 나오지 않으면 멈추고 알린다. 접속 후 입력도 보내지 않는다.
func TestSessionLoginScriptTimeout(t *testing.T) {
	i18n.SetLang("ko")
	durOld := G_durLoginStepTimeout
	G_durLoginStepTimeout = 40 * time.Millisecond
	defer func() { G_durLoginStepTimeout = durOld }()
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "never-shown", Send: "x"}},
		StartupInput: "never\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)
	pFake.chIn <- []byte("something else\r\n")
	waitOutput(t, pRec, "never-shown")
	time.Sleep(20 * time.Millisecond)
	if slWrites := pFake.writes(); len(slWrites) != 0 {
		t.Fatalf("보내면 안 된다: %v", slWrites)
	}
}

// 사용자가 직접 입력하면 스크립트는 멈춘다. 터미널의 자동 응답(ESC 로 시작)은 멈추지 않는다.
func TestSessionLoginScriptCancelledByInput(t *testing.T) {
	i18n.SetLang("ko")
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript: []LoginStep{{Expect: "password", Send: "secret"}},
	})
	defer pManager.Close(szId)
	pSession, _ := pManager.Get(szId)
	waitConnected(t, pSession)

	pSession.Input("\x1b[?1;2c") // 터미널의 자동 응답
	pSession.mtxData.Lock()
	bStill := pSession.login != nil
	pSession.mtxData.Unlock()
	if !bStill {
		t.Fatal("자동 응답에 스크립트가 멈췄다")
	}
	pSession.Input("ls\r")
	waitOutput(t, pRec, "로그인 스크립트를 멈췄습니다")
	pFake.chIn <- []byte("Password: ")
	time.Sleep(30 * time.Millisecond)
	for _, szWrite := range pFake.writes() {
		if strings.Contains(szWrite, "secret") {
			t.Fatalf("멈춘 뒤에 보냈다: %q", szWrite)
		}
	}
}

func waitConnected(t *testing.T, pSession *Session) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if pSession.Snapshot().M_szState == G_cszStateConnected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("연결되지 않았다")
}

// waitOutput 은 화면 출력에 szWant 가 나올 때까지 기다린다.
func waitOutput(t *testing.T, pRec *eventRecorder, szWant string) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		pRec.mtxData.Lock()
		szAll := strings.Join(pRec.slData, "")
		pRec.mtxData.Unlock()
		if strings.Contains(szAll, szWant) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	t.Fatalf("출력에 %q 가 없다: %q", szWant, strings.Join(pRec.slData, ""))
}
