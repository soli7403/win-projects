// zmodemcheck 는 internal/zmodem 엔진을 실제 서버의 lrzsz(sz·rz)와 맞대어 보는 검증 도구다 (KI-012).
// 앱의 백엔드와 같은 코드(internal/session 의 세션 — 읽기 루프·저장·안내 줄·취소)를 실제 SSH PTY 셸에 물려 돌린다.
// 화면(프런트)만 빠져 있다: 세션이 화면으로 보내는 글을 모아 끝에 찍는다.
//
// 사용:
//
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U exec "명령"
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U recv -dir 받을폴더 [-cancelAfter N] "sz 파일…"
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U send [-files a,b] "rz"
//
// 비밀번호는 환경 변수 ZCHECK_PASS 로만 받는다. 호스트 키는 확인하지 않는다(사내망 검증용 — 앱의 접속 경로가 아니다).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"dariterm/internal/i18n"
	"dariterm/internal/session"
)

func main() {
	szHost := flag.String("host", "", "서버 주소")
	iPort := flag.Int("port", 22, "포트")
	szUser := flag.String("user", "", "계정")
	iTimeout := flag.Int("timeout", 300, "전체 시한(초)")
	iStall := flag.Int("stall", 0, "서버 응답 없음 시한(초). 0 이면 앱의 기본값")
	szThen := flag.String("then", "", "끝 안내가 나온 뒤 칠 입력 (Go 문자열 표기, 예: \"\\x18\\x18\\x18\\x18\\x18\")")
	flag.Parse()
	slArgs := flag.Args()
	if *iStall > 0 {
		session.G_durZmodemStall = time.Duration(*iStall) * time.Second
	}
	if *szHost == "" || *szUser == "" || len(slArgs) < 2 {
		fmt.Fprintln(os.Stderr, "usage: zmodemcheck -host H -user U exec|recv|send [옵션] \"명령\"")
		os.Exit(2)
	}
	cfg := &ssh.ClientConfig{
		User:            *szUser,
		Auth:            []ssh.AuthMethod{ssh.Password(os.Getenv("ZCHECK_PASS"))},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, errDial := ssh.Dial("tcp", fmt.Sprintf("%s:%d", *szHost, *iPort), cfg)
	if errDial != nil {
		fail("접속 실패: %v", errDial)
	}
	defer client.Close()
	// 끝 안내가 나오지 않는 경우(시작 표시가 오지 않음 등)를 대비한 도구 자체의 시한이다.
	timer := time.AfterFunc(time.Duration(*iTimeout)*time.Second, func() {
		fmt.Fprintln(os.Stderr, "\n[zmodemcheck] 시한 초과 — 연결을 닫고 끝낸다")
		if g_pScreen != nil {
			fmt.Printf("[zmodemcheck] 그때까지의 화면 출력:\n%s\n", visible(g_pScreen.text()))
		}
		_ = client.Close()
		os.Exit(3)
	})
	defer timer.Stop()

	switch slArgs[0] {
	case "exec":
		runExec(client, slArgs[1])
	case "put":
		// put 로컬파일 원격경로 — 검증 준비물(패키지 등)을 올린다. ZMODEM 과 무관한 경로(셸의 cat)다.
		if len(slArgs) < 3 {
			fail("usage: put 로컬파일 원격경로")
		}
		runPut(client, slArgs[1], slArgs[2])
	case "recv":
		fs := flag.NewFlagSet("recv", flag.ExitOnError)
		szDir := fs.String("dir", ".", "받을 폴더")
		i64Cancel := fs.Int64("cancelAfter", 0, "받는 파일이 이만큼 커지면 Ctrl+C 로 취소(0 = 취소 안 함)")
		_ = fs.Parse(slArgs[1:])
		runTransfer(client, fs.Arg(0), *szDir, nil, *i64Cancel, unquote(*szThen))
	case "send":
		fs := flag.NewFlagSet("send", flag.ExitOnError)
		szFiles := fs.String("files", "", "보낼 파일(쉼표로 구분). 비우면 선택 취소")
		_ = fs.Parse(slArgs[1:])
		var slFiles []string
		if *szFiles != "" {
			slFiles = strings.Split(*szFiles, ",")
		}
		runTransfer(client, fs.Arg(0), "", slFiles, 0, unquote(*szThen))
	default:
		fail("모르는 동작: %s", slArgs[0])
	}
}

// unquote 는 Go 문자열 표기(\\x18 등)를 실제 글자로 바꾼다.
func unquote(szText string) string {
	if szText == "" {
		return ""
	}
	szOut, errUnquote := strconv.Unquote(`"` + szText + `"`)
	if errUnquote != nil {
		fail("-then 값이 잘못됨: %v", errUnquote)
	}
	return szOut
}

func fail(szFormat string, args ...any) {
	fmt.Fprintf(os.Stderr, "[zmodemcheck] "+szFormat+"\n", args...)
	os.Exit(1)
}

// runExec 는 PTY 없이 명령 하나를 돌리고 출력을 그대로 낸다.
func runExec(client *ssh.Client, szCommand string) {
	session, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	defer session.Close()
	bOut, errRun := session.CombinedOutput(szCommand)
	_, _ = os.Stdout.Write(bOut)
	if errRun != nil {
		fail("명령 실패: %v", errRun)
	}
}

// runPut 은 로컬 파일을 서버의 경로에 쓴다 (PTY 없는 세션의 표준 입력 → cat).
func runPut(client *ssh.Client, szLocal string, szRemote string) {
	pFile, errOpen := os.Open(szLocal)
	if errOpen != nil {
		fail("열기 실패: %v", errOpen)
	}
	defer pFile.Close()
	session, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	defer session.Close()
	session.Stdin = pFile
	szQuoted := "'" + strings.ReplaceAll(szRemote, "'", `'\''`) + "'"
	if bOut, errRun := session.CombinedOutput("cat > " + szQuoted); errRun != nil {
		fail("올리기 실패: %v %s", errRun, bOut)
	}
}

// shellTransport 는 SSH PTY 셸을 세션의 Transport 로 본다.
type shellTransport struct {
	session *ssh.Session
	stdin   io.Writer
	stdout  io.Reader
}

func (pShell *shellTransport) Read(bBuf []byte) (int, error) { return pShell.stdout.Read(bBuf) }
func (pShell *shellTransport) Write(bData []byte) error {
	_, errWrite := pShell.stdin.Write(bData)
	return errWrite
}
func (pShell *shellTransport) Resize(iCols int, iRows int) error {
	return pShell.session.WindowChange(iRows, iCols)
}
func (pShell *shellTransport) Close() error { return pShell.session.Close() }

// g_pScreen 은 지금 돌고 있는 전송의 화면 기록이다 (시한 초과 때 찍는다).
var g_pScreen *screenLog

// screenLog 는 세션이 화면으로 보낸 글을 모은다.
type screenLog struct {
	mtxText sync.Mutex
	sbText  strings.Builder
}

func (pLog *screenLog) emit(szEvent string, anyData ...any) {
	if szEvent != "session.data" || len(anyData) != 2 {
		return
	}
	pLog.mtxText.Lock()
	pLog.sbText.WriteString(anyData[1].(string))
	pLog.mtxText.Unlock()
}

func (pLog *screenLog) text() string {
	pLog.mtxText.Lock()
	defer pLog.mtxText.Unlock()
	return pLog.sbText.String()
}

// runTransfer 는 앱처럼 PTY 셸을 세션으로 열어 명령을 치고, 전송이 끝난 뒤 셸이 살아 있는지 echo 로 확인한다.
func runTransfer(client *ssh.Client, szCommand string, szDir string, slFiles []string, i64CancelAfter int64, szThen string) {
	i18n.SetLang("ko")
	shell, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	// 앱(internal/sshconn/shell.go)과 같은 PTY 요청.
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if errPty := shell.RequestPty("xterm-256color", 24, 80, modes); errPty != nil {
		fail("PTY 실패: %v", errPty)
	}
	stdin, _ := shell.StdinPipe()
	stdout, _ := shell.StdoutPipe()
	if errShell := shell.Shell(); errShell != nil {
		fail("셸 실패: %v", errShell)
	}
	transport := &shellTransport{session: shell, stdin: stdin, stdout: stdout}

	log := &screenLog{}
	g_pScreen = log
	pManager := session.NewManager(log.emit)
	info, errOpen := pManager.Open(session.OpenConfig{
		Info:     session.Info{M_szTitle: "zmodemcheck", M_szProtocol: "ssh", M_szTarget: client.RemoteAddr().String()},
		Encoding: "utf-8",
		Dialer:   func(func(string)) (session.Transport, error) { return transport, nil },
		Zmodem: &session.ZmodemHooks{
			DownloadDir: func() string { return szDir },
			PickFiles:   func() ([]string, error) { return slFiles, nil },
		},
	})
	if errOpen != nil {
		fail("세션 열기 실패: %v", errOpen)
	}
	pSession, _ := pManager.Get(info.M_szId)
	go func() { // 프런트 대신 출력 확인(ack)을 해 준다 (출력 흐름 제어에 막히지 않게)
		for {
			time.Sleep(5 * time.Millisecond)
			pSession.AckOutput(1000)
		}
	}()
	time.Sleep(1500 * time.Millisecond) // 로그인 배너·프롬프트
	tmStart := time.Now()
	pSession.Input(szCommand + "\r")

	slEnds := []string{"ZMODEM 전송 완료", i18n.T("be.zm.cancelled"), i18n.T("be.zm.remoteCancelled"), "ZMODEM 전송 실패", "ZMODEM 전송을 중단했습니다", "ZMODEM 시작 표시가 깨져서"}
	szEnd := ""
	bCancelled := false
	for szEnd == "" {
		time.Sleep(20 * time.Millisecond)
		if i64CancelAfter > 0 && !bCancelled && partSize(szDir) >= i64CancelAfter {
			fmt.Printf("[zmodemcheck] 받는 파일이 %d바이트를 넘어 Ctrl+C 를 보냄\n", i64CancelAfter)
			pSession.Input("\x03")
			bCancelled = true
		}
		szScreen := log.text()
		for _, szOne := range slEnds {
			if strings.Contains(szScreen, szOne) {
				szEnd = szOne
			}
		}
	}
	fmt.Printf("[zmodemcheck] 끝 안내: %q (명령 뒤 %s)\n", szEnd, time.Since(tmStart).Round(time.Millisecond))
	iEndAt := strings.Index(log.text(), szEnd)

	if szThen != "" {
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("[zmodemcheck] 끝 안내 뒤 입력: %q\n", szThen)
		pSession.Input(szThen)
		time.Sleep(time.Second)
	}
	// 셸이 살아 있는지: 계산 결과가 찍혀야 한다(친 글자의 에코와 구분된다). 전송 상태가 풀릴 때까지 몇 번 다시 친다.
	bAlive := false
	for iTry := 0; iTry < 10 && !bAlive; iTry++ {
		pSession.Input("echo ZC_ALIVE_$((40+2))\r")
		for iWait := 0; iWait < 50 && !bAlive; iWait++ {
			time.Sleep(20 * time.Millisecond)
			bAlive = strings.Contains(log.text(), "ZC_ALIVE_42")
		}
	}
	szScreen := log.text()
	// 끝 안내 뒤로 화면에 나온 것 중 글자가 아닌 바이트 수: 전송 데이터가 화면으로 샜는지 본다.
	iJunk := 0
	for _, bOne := range []byte(szScreen[iEndAt:]) {
		if bOne < 0x20 && bOne != '\r' && bOne != '\n' && bOne != 0x1b && bOne != 0x07 && bOne != 0x08 || bOne == 0x7f {
			iJunk++
		}
	}
	szShow := szScreen
	if len(szShow) > 6000 {
		szShow = szShow[:2500] + "\n…(가운데 생략)…\n" + szShow[len(szShow)-2500:]
	}
	fmt.Printf("[zmodemcheck] 화면 출력(%d바이트, 끝 안내 뒤 %d바이트 중 제어 문자 %d개):\n%s\n", len(szScreen), len(szScreen)-iEndAt, iJunk, visible(szShow))
	if bAlive {
		fmt.Println("[zmodemcheck] 셸 확인: 전송 뒤 프롬프트가 돌아와 명령이 실행됨")
	} else {
		fmt.Println("[zmodemcheck] 셸 확인 실패: 전송 뒤 명령이 실행되지 않음")
	}
	pSession.Input("exit\r")
	time.Sleep(200 * time.Millisecond)
	if !bAlive {
		os.Exit(1)
	}
}

// partSize 는 폴더에서 받는 중인 파일(.dtpart)의 크기다. 없으면 0.
func partSize(szDir string) int64 {
	slParts, _ := filepath.Glob(filepath.Join(szDir, "*.dtpart"))
	var i64Max int64
	for _, szPart := range slParts {
		if pInfo, errStat := os.Stat(szPart); errStat == nil && pInfo.Size() > i64Max {
			i64Max = pInfo.Size()
		}
	}
	return i64Max
}

// visible 은 제어 문자를 보이는 글자로 바꾼다(화면에 이진 쓰레기가 섞였는지 눈으로 볼 수 있게).
func visible(szText string) string {
	var sb strings.Builder
	for _, bOne := range []byte(szText) {
		switch {
		case bOne == '\n' || bOne == '\t' || bOne >= 0x20 && bOne != 0x7f:
			sb.WriteByte(bOne)
		case bOne == '\r':
		default:
			fmt.Fprintf(&sb, "\\x%02x", bOne)
		}
	}
	return sb.String()
}
