package session

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 같은 별칭·같은 초에 세 번 열면 파일 세 개가 따로 만들어지고 각자 자기 내용만 담는다 (KI-001, session.md §6).
func TestLogOpensSeparateFilePerSession(t *testing.T) {
	szDir := t.TempDir()
	tNow := time.Date(2026, 11, 16, 14, 32, 5, 0, time.Local)
	slLoggers := make([]*Logger, 0, 3)
	for iIndex := 0; iIndex < 3; iIndex++ {
		pLogger, errOpen := OpenLog(szDir, "web", tNow)
		if errOpen != nil {
			t.Fatalf("%d번째 열기 실패: %v", iIndex+1, errOpen)
		}
		pLogger.Write("session" + strconv.Itoa(iIndex+1) + "\n")
		slLoggers = append(slLoggers, pLogger)
	}
	for _, pLogger := range slLoggers {
		_ = pLogger.Close()
	}
	slWant := []string{
		"web-20261116-143205.log",
		"web-20261116-143205-2.log",
		"web-20261116-143205-3.log",
	}
	for iIndex, szName := range slWant {
		bData, errRead := os.ReadFile(filepath.Join(szDir, "logs", szName))
		if errRead != nil {
			t.Fatalf("%s 가 없음: %v", szName, errRead)
		}
		szWantBody := "session" + strconv.Itoa(iIndex+1) + "\n"
		if string(bData) != szWantBody {
			t.Fatalf("%s 내용이 섞임: %q (기대 %q)", szName, string(bData), szWantBody)
		}
	}
	slAll, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
	if len(slAll) != 3 {
		t.Fatalf("파일이 3개여야 함: %v", slAll)
	}
}

// 이름 규칙: 초까지 쓰고, 순번 1 은 접미 없음, 2 부터 "-N". 별칭 정리(§9): 제어 문자·앞뒤 공백·점, 80자, 빈 값.
func TestLogFileNameSeq(t *testing.T) {
	tNow := time.Date(2026, 11, 16, 14, 32, 5, 0, time.Local)
	slCases := []struct {
		szAlias string
		iSeq    int
		szWant  string
	}{
		{"srv", 1, "srv-20261116-143205.log"},
		{"srv", 2, "srv-20261116-143205-2.log"},
		{"a/b:c", 10, "a_b_c-20261116-143205-10.log"},
		{"  a\tb. ", 1, "a_b-20261116-143205.log"},
		{"", 1, "session-20261116-143205.log"},
		{" . ", 1, "session-20261116-143205.log"},
		{strings.Repeat("가", 100), 1, strings.Repeat("가", 80) + "-20261116-143205.log"},
	}
	for _, tc := range slCases {
		if szGot := LogFileName(tc.szAlias, tNow, tc.iSeq); szGot != tc.szWant {
			t.Fatalf("LogFileName(%q, %d) = %q, 기대 %q", tc.szAlias, tc.iSeq, szGot, tc.szWant)
		}
	}
}

// 세션을 통해 열어도 세션마다 파일이 하나씩 생긴다 (start → OpenLog 경로).
func TestSessionLogIsPerSession(t *testing.T) {
	szDir := t.TempDir()
	fnDial := func(func(string)) (Transport, error) { return newFakeTransport(), nil }
	cfg := sshConfig(fnDial, false)
	cfg.DataDir = szDir
	cfg.LogEnabled = true
	cfg.LogAlias = "web"
	pFirst := newSession(cfg, func(string, ...any) {})
	pSecond := newSession(cfg, func(string, ...any) {})
	pFirst.start()
	pSecond.start()
	waitUntil(t, func() bool {
		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
		return len(slFiles) == 2
	}, time.Second)
	pFirst.Close()
	pSecond.Close()
}

// 묶음 경계에서 잘린 ANSI 시퀀스도 지워진다 (통합 계획 §3).
func TestLogStripsSequenceSplitAcrossWrites(t *testing.T) {
	szDir := t.TempDir()
	pLogger, errOpen := OpenLog(szDir, "x", time.Date(2026, 11, 16, 1, 2, 3, 0, time.Local))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	pLogger.Write("a\x1b[3")           // CSI 가 중간에서 잘림
	pLogger.Write("1mred\x1b[")        // 앞 조각 완성 + 또 잘림
	pLogger.Write("0m\x1b]0;title")    // 앞 조각 완성 + OSC 미종결
	pLogger.Write("\x07b\x1b]0;t\x1b") // OSC 종결 + ESC \ 종결이 잘림
	pLogger.Write("\\c\r\n")
	_ = pLogger.Close()
	bData, errRead := os.ReadFile(pLogger.Path())
	if errRead != nil {
		t.Fatal(errRead)
	}
	if string(bData) != "aredbc\n" {
		t.Fatalf("잘린 시퀀스가 남음: %q", string(bData))
	}
}

// 시퀀스가 아닌 ESC(256바이트 넘게 종결 없음)는 보관하지 않고 그대로 기록한다.
func TestLogCarryGivesUpOnLongTail(t *testing.T) {
	szBody, szTail := splitAnsiTail("\x1b" + strings.Repeat("x", 300))
	if szTail != "" || len(szBody) != 301 {
		t.Fatalf("긴 꼬리는 본문으로 처리해야 함: body=%d tail=%q", len(szBody), szTail)
	}
	szBody, szTail = splitAnsiTail("ok\x1b]" + strings.Repeat("y", 300))
	if szTail != "" || !strings.HasPrefix(szBody, "ok") {
		t.Fatalf("긴 미종결 OSC 는 본문으로: tail=%q", szTail)
	}
}

// 1초 안에 flush 되어 닫기 전에도 파일에서 읽힌다 (통합 계획 §10).
func TestLogFlushesPeriodically(t *testing.T) {
	szDir := t.TempDir()
	pLogger, errOpen := OpenLog(szDir, "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.Write("hello\n")
	waitUntil(t, func() bool {
		bData, _ := os.ReadFile(pLogger.Path())
		return string(bData) == "hello\n"
	}, 3*time.Second)
}

// 열린 로그는 IsLogOpen 이 true, 닫으면 false (오래된 로그 정리가 기록 중인 파일을 건너뛰는 근거).
func TestIsLogOpenTracksOpenFiles(t *testing.T) {
	szDir := t.TempDir()
	pLogger, errOpen := OpenLog(szDir, "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	if !IsLogOpen(pLogger.Path()) {
		t.Fatal("열린 로그는 열림으로 보여야 함")
	}
	szRel, errRel := filepath.Rel(".", pLogger.Path())
	if errRel == nil && !IsLogOpen(szRel) {
		t.Fatal("상대 경로로 물어도 같아야 함")
	}
	_ = pLogger.Close()
	if IsLogOpen(pLogger.Path()) {
		t.Fatal("닫은 로그는 열림이 아니어야 함")
	}
}

// Close 뒤에 transport 에서 조각이 와도 session.data 가 더 나가지 않는다 (통합 계획 §7).
func TestNoDataAfterClose(t *testing.T) {
	rec := &eventRecorder{}
	transport := newFakeTransport()
	pSession := newSession(sshConfig(func(func(string)) (Transport, error) { return transport, nil }, false), rec.emit)
	pSession.start()
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	// Close 는 transport 를 닫지만 fakeTransport 의 Read 는 채널에 남은 조각을 먼저 돌려줄 수 있다.
	pSession.Close()
	select {
	case transport.chIn <- []byte("late"):
	default:
	}
	time.Sleep(100 * time.Millisecond)
	if strings.Contains(rec.output(), "late") {
		t.Fatalf("닫은 뒤 출력이 나감: %q", rec.output())
	}
}

// 닫은 뒤 늦게 끝난 접속 실패는 안내를 내지 않는다 (통합 계획 §7, onConnectFailed 세대 검사).
func TestNoFailureNoticeAfterClose(t *testing.T) {
	rec := &eventRecorder{}
	chRelease := make(chan struct{})
	fnDial := func(func(string)) (Transport, error) {
		<-chRelease
		return nil, errTestDial
	}
	pSession := newSession(sshConfig(fnDial, false), rec.emit)
	pSession.start()
	time.Sleep(30 * time.Millisecond) // 접속 시도가 Dialer 에서 기다리도록
	pSession.Close()
	szBefore := rec.output()
	close(chRelease)
	time.Sleep(100 * time.Millisecond)
	if rec.output() != szBefore {
		t.Fatalf("닫은 뒤 접속 실패 안내가 나감: %q", strings.TrimPrefix(rec.output(), szBefore))
	}
}

// cp949 확장 영역의 한글("똠" = 0x8C 0x63)도 디코딩된다 (통합 계획 §6).
func TestDecoderHandlesCp949Extension(t *testing.T) {
	pDecoder := NewDecoder("cp949")
	if szGot := pDecoder.Decode([]byte{0x8C, 0x63}); szGot != "똠" {
		t.Fatalf("cp949 확장 한글: %q", szGot)
	}
}
