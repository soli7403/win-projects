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

// 열려 있는 세션에서 기록을 켜면 그때부터 새 파일에 쓰고, 끄면 파일을 닫고 더 쓰지 않는다 (session.md §6, D99).
func TestSetLoggingOnOpenSession(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web" // LogEnabled 는 꺼진 채로 연다
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	fnFiles := func() []string {
		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
		return slFiles
	}

	pFake.chIn <- []byte("before\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "before") }, time.Second)
	if len(fnFiles()) != 0 {
		t.Fatalf("꺼져 있으면 파일이 없어야 함: %v", fnFiles())
	}

	pManager.SetLogging(true)
	pManager.SetLogging(true) // 두 번 켜도 파일은 하나
	if len(fnFiles()) != 1 {
		t.Fatalf("켜면 파일 하나가 생겨야 함: %v", fnFiles())
	}
	szPath := fnFiles()[0]
	if !IsLogOpen(szPath) {
		t.Fatal("기록 중인 파일로 등록돼야 함")
	}
	pFake.chIn <- []byte("during\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "during") }, time.Second)

	pManager.SetLogging(false)
	if IsLogOpen(szPath) {
		t.Fatal("끄면 파일을 닫아야 함")
	}
	pFake.chIn <- []byte("after\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "after") }, time.Second)

	bData, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	szLog := string(bData)
	if !strings.Contains(szLog, "during") || strings.Contains(szLog, "before") || strings.Contains(szLog, "after") {
		t.Fatalf("켜져 있던 동안의 출력만 있어야 함: %q", szLog)
	}

	pManager.SetLogging(true) // 다시 켜면 새 파일
	if len(fnFiles()) != 2 {
		t.Fatalf("다시 켜면 파일이 하나 더 생겨야 함: %v", fnFiles())
	}
}

// 닫힌 세션은 기록을 켜도 파일을 만들지 않는다.
func TestSetLoggingIgnoresClosedSession(t *testing.T) {
	szDir := t.TempDir()
	cfg := sshConfig(func(func(string)) (Transport, error) { return newFakeTransport(), nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	pSession := newSession(cfg, func(string, ...any) {})
	pSession.start()
	pSession.Close()
	pSession.SetLogging(true)
	if slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "*")); len(slFiles) != 0 {
		t.Fatalf("닫힌 세션은 파일을 만들지 않음: %v", slFiles)
	}
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

// 내려 쓰기 주기 0: 받는 즉시 파일에 쓴다(1초 주기를 기다리지 않는다) (session.md §6, D100).
func TestLogSyncZeroWritesImmediately(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(0)
	pLogger.Write("now\n")
	bData, _ := os.ReadFile(pLogger.Path())
	if string(bData) != "now\n" {
		t.Fatalf("즉시 모드는 바로 파일에 있어야 함: %q", string(bData))
	}
}

// 버퍼에 쌓인 채로 주기를 0 으로 바꾸면 곧바로 파일에 쓴다. 0 이 아니면 지금처럼 약 1초 뒤에 쓴다.
func TestLogSyncSecSwitch(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(3)
	pLogger.Write("buffered\n")
	if bData, _ := os.ReadFile(pLogger.Path()); len(bData) != 0 {
		t.Fatalf("0 이 아니면 바로 쓰지 않음: %q", string(bData))
	}
	pLogger.SetSyncSec(0)
	if bData, _ := os.ReadFile(pLogger.Path()); string(bData) != "buffered\n" {
		t.Fatalf("0 으로 바꾸면 곧바로 써야 함: %q", string(bData))
	}
	pLogger.SetSyncSec(5)
	pLogger.Write("later\n")
	waitUntil(t, func() bool {
		bData, _ := os.ReadFile(pLogger.Path())
		return string(bData) == "buffered\nlater\n"
	}, 2*time.Second)
}

// 범위 밖 값은 0~5 로 맞춘다.
func TestClampLogSyncSec(t *testing.T) {
	for iIn, iWant := range map[int]int{-3: 0, 0: 0, 2: 2, 5: 5, 9: 5} {
		if iGot := ClampLogSyncSec(iIn); iGot != iWant {
			t.Fatalf("ClampLogSyncSec(%d) = %d, 기대 %d", iIn, iGot, iWant)
		}
	}
}

// 세션에 준 주기는 기록을 켤 때 로그에 전해지고, 바꾸면 기록 중인 로그에도 곧바로 적용된다.
func TestSessionLogSyncSecApplies(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	cfg.LogEnabled = true
	cfg.LogSyncSec = 4
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	if iGot := pSession.logSyncSecForTest(); iGot != 4 {
		t.Fatalf("열 때 준 주기가 로그에 전해져야 함: %d", iGot)
	}
	pManager.SetLogSyncSec(0)
	if iGot := pSession.logSyncSecForTest(); iGot != 0 {
		t.Fatalf("바꾼 주기가 기록 중인 로그에 적용돼야 함: %d", iGot)
	}
	pManager.SetLogging(false)
	pManager.SetLogSyncSec(1)
	pManager.SetLogging(true)
	if iGot := pSession.logSyncSecForTest(); iGot != 1 {
		t.Fatalf("꺼져 있을 때 바꾼 주기도 다시 켤 때 써야 함: %d", iGot)
	}
}

// logSyncSecForTest 는 지금 기록 중인 로그의 내려 쓰기 주기다 (기록 중이 아니면 -1).
func (pSession *Session) logSyncSecForTest() int {
	pSession.mtxData.Lock()
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger == nil {
		return -1
	}
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	return pLogger.iSyncSec
}

// 줄 앞 시각(D112): 켜면 줄마다 한 번, 그 줄의 첫 글자를 받은 때의 시각이 붙는다. 줄이 여러 조각에 걸쳐 와도 한 번이다.
func TestLogTimestampPerLine(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	tClock := time.Date(2026, 11, 23, 9, 5, 7, 123_000_000, time.Local)
	pLogger.fnNow = func() time.Time { return tClock }
	pLogger.SetSyncSec(0)
	pLogger.SetTimestamp(true)
	pLogger.Write("first li")
	tClock = tClock.Add(2 * time.Second)
	pLogger.Write("ne\r\n\r\n\x1b[31msecond\x1b[0m\r\nthi")
	tClock = tClock.Add(time.Second)
	pLogger.Write("rd")
	bData, _ := os.ReadFile(pLogger.Path())
	szWant := "[2026-11-23 09:05:07.123] first line\n" +
		"[2026-11-23 09:05:09.123] \n" +
		"[2026-11-23 09:05:09.123] second\n" +
		"[2026-11-23 09:05:09.123] third"
	if string(bData) != szWant {
		t.Fatalf("줄 앞 시각:\n%q\n기대:\n%q", string(bData), szWant)
	}
}

// 기록 중에 켜고 끄면 쓰던 줄 중간에는 넣지 않고 다음 줄부터 적용한다. 닫을 때 남은 꼬리에도 같은 규칙이다.
func TestLogTimestampToggleMidLine(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	tClock := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
	pLogger.fnNow = func() time.Time { return tClock }
	pLogger.SetSyncSec(0)
	pLogger.Write("plain\nhalf")
	pLogger.SetTimestamp(true)
	pLogger.Write(" line\nstamped\n")
	pLogger.SetTimestamp(false)
	pLogger.Write("off again\n")
	pLogger.SetTimestamp(true)
	pLogger.Write("tail\x1b[") // 미완성 시퀀스는 닫을 때 기록된다
	szPath := pLogger.Path()
	_ = pLogger.Close()
	bData, _ := os.ReadFile(szPath)
	szStamp := "[2026-01-02 03:04:05.000] "
	szWant := "plain\nhalf line\n" + szStamp + "stamped\noff again\n" + szStamp + "tail\x1b["
	if string(bData) != szWant {
		t.Fatalf("중간에 켜고 끄기:\n%q\n기대:\n%q", string(bData), szWant)
	}
}

// 세션에 준 설정은 기록을 켤 때 로그에 전해지고, 바꾸면 기록 중인 로그에도 적용된다 (D112).
func TestSessionLogTimestampApplies(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	cfg.LogEnabled = true
	cfg.LogTimestamp = true
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	fnStamp := func() bool {
		pSession.mtxData.Lock()
		pLogger := pSession.logger
		pSession.mtxData.Unlock()
		pLogger.mtxData.Lock()
		defer pLogger.mtxData.Unlock()
		return pLogger.bTimestamp
	}
	if !fnStamp() {
		t.Fatal("열 때 준 설정이 로그에 전해져야 함")
	}
	pManager.SetLogTimestamp(false)
	if fnStamp() {
		t.Fatal("바꾼 설정이 기록 중인 로그에 적용돼야 함")
	}
	pManager.SetLogging(false)
	pManager.SetLogTimestamp(true)
	pManager.SetLogging(true)
	if !fnStamp() {
		t.Fatal("꺼져 있을 때 바꾼 설정도 다시 켤 때 써야 함")
	}
}

// "ANSI 그대로 저장"이 꺼져 있으면(기본) 지금처럼 시퀀스와 CR 을 지우고, 켜면 받은 그대로 쓴다 (D120).
func TestLogAnsiKeepsSequences(t *testing.T) {
	const szIn = "\x1b[1;31mred\x1b[0m\r\n\x1b]0;title\x07plain\r\n"
	for _, tc := range []struct {
		szName string
		bAnsi  bool
		szWant string
	}{
		{"꺼짐", false, "red\nplain\n"},
		{"켜짐", true, szIn},
	} {
		pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
		if errOpen != nil {
			t.Fatal(errOpen)
		}
		pLogger.SetSyncSec(0)
		pLogger.SetAnsi(tc.bAnsi)
		pLogger.Write(szIn)
		bData, _ := os.ReadFile(pLogger.Path())
		_ = pLogger.Close()
		if string(bData) != tc.szWant {
			t.Fatalf("%s:\n%q\n기대:\n%q", tc.szName, string(bData), tc.szWant)
		}
	}
}

// 시퀀스가 묶음 경계에서 잘려 와도 켜져 있으면 한 덩어리로 남고, 중간에 끄면 그 시퀀스는 통째로 지워진다.
// (반은 남고 반은 지워진 조각이 로그에 생기지 않는다.) 닫을 때 남은 꼬리도 그때의 설정을 따른다.
func TestLogAnsiToggleKeepsSequencesWhole(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	pLogger.SetSyncSec(0)
	pLogger.SetAnsi(true)
	pLogger.Write("a\x1b[3")   // 미완성: 보관
	pLogger.Write("1mb\x1b[0") // 앞 시퀀스는 완성돼 그대로 기록, 뒤는 다시 보관
	pLogger.SetAnsi(false)
	pLogger.Write("mc\r\n\x1b[32mgreen") // 보관분 + 이번 조각은 지워서 기록
	pLogger.SetAnsi(true)
	pLogger.Write("\x1b[0m\r\nend\x1b[") // 다시 그대로. 끝의 미완성 꼬리는 닫을 때 그대로 쓴다
	_ = pLogger.Close()
	bData, _ := os.ReadFile(pLogger.Path())
	szWant := "a\x1b[31mb" + "c\ngreen" + "\x1b[0m\r\nend\x1b["
	if string(bData) != szWant {
		t.Fatalf("켜고 끄기:\n%q\n기대:\n%q", string(bData), szWant)
	}
}

// 줄 앞 시각과 함께 켜면 시각은 줄바꿈(LF) 기준으로 지금처럼 붙고, 시퀀스와 CR 은 그대로 남는다.
func TestLogAnsiWithTimestamp(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	tClock := time.Date(2026, 11, 23, 9, 5, 7, 123_000_000, time.Local)
	pLogger.fnNow = func() time.Time { return tClock }
	pLogger.SetSyncSec(0)
	pLogger.SetTimestamp(true)
	pLogger.SetAnsi(true)
	pLogger.Write("\x1b[31mone\x1b[0m\r\ntwo\r\n")
	bData, _ := os.ReadFile(pLogger.Path())
	szWant := "[2026-11-23 09:05:07.123] \x1b[31mone\x1b[0m\r\n" +
		"[2026-11-23 09:05:07.123] two\r\n"
	if string(bData) != szWant {
		t.Fatalf("시각 + ANSI:\n%q\n기대:\n%q", string(bData), szWant)
	}
}

// 세션을 열 때 준 설정이 로그에 전해지고, 바꾸면 기록 중인 로그에도, 꺼져 있다가 다시 켠 로그에도 적용된다.
func TestSessionLogAnsiApplies(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	cfg.LogEnabled = true
	cfg.LogAnsi = true
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	fnAnsi := func() bool {
		pSession.mtxData.Lock()
		pLogger := pSession.logger
		pSession.mtxData.Unlock()
		pLogger.mtxData.Lock()
		defer pLogger.mtxData.Unlock()
		return pLogger.bAnsi
	}
	if !fnAnsi() {
		t.Fatal("열 때 준 설정이 로그에 전해져야 함")
	}
	pManager.SetLogAnsi(false)
	if fnAnsi() {
		t.Fatal("바꾼 설정이 기록 중인 로그에 적용돼야 함")
	}
	pManager.SetLogging(false)
	pManager.SetLogAnsi(true)
	pManager.SetLogging(true)
	if !fnAnsi() {
		t.Fatal("꺼져 있을 때 바꾼 설정도 다시 켤 때 써야 함")
	}
}
