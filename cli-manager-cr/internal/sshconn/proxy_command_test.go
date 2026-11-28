package sshconn

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// 프록시 명령 시험 (protocols.md §2.7a, D152). 이 테스트 실행 파일 자신을 "명령"으로 띄운다:
// 환경 변수 DT_PROXYCMD_MODE 가 있으면 TestHelperProxyCommand 가 도우미로 동작한다(echo·fail·silent·bridge).
// 그래서 Windows 에서는 실제로 cmd.exe /d /s /c "…" 를 거친다.

const g_cszHelperModeEnv = "DT_PROXYCMD_MODE"

// helperCommand 는 이 실행 파일을 도우미로 띄우는 명령 줄이다. 인자는 "--" 뒤에 붙인다.
func helperCommand(t *testing.T, szMode string, slArgs ...string) string {
	t.Helper()
	t.Setenv(g_cszHelperModeEnv, szMode)
	szExe, errExe := os.Executable()
	if errExe != nil {
		t.Fatal(errExe)
	}
	return `"` + szExe + `" -test.run=TestHelperProxyCommand -- ` + strings.Join(slArgs, " ")
}

// TestHelperProxyCommand 는 시험의 도우미 프로세스다. 그냥 테스트로 돌 때는 아무것도 하지 않는다.
func TestHelperProxyCommand(t *testing.T) {
	szMode := os.Getenv(g_cszHelperModeEnv)
	if szMode == "" {
		return
	}
	var slArgs []string
	for iIndex, szArg := range os.Args {
		if szArg == "--" {
			slArgs = os.Args[iIndex+1:]
			break
		}
	}
	switch szMode {
	case "echo":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "fail":
		fmt.Fprintln(os.Stderr, "boom: no route to bastion")
		os.Exit(1)
	case "silent":
		time.Sleep(10 * time.Second)
	case "bridge":
		conn, errDial := net.Dial("tcp", net.JoinHostPort(slArgs[0], slArgs[1]))
		if errDial != nil {
			fmt.Fprintln(os.Stderr, errDial)
			os.Exit(1)
		}
		go func() { _, _ = io.Copy(conn, os.Stdin); _ = conn.Close() }()
		_, _ = io.Copy(os.Stdout, conn)
	}
	os.Exit(0)
}

func TestExpandProxyCommand(t *testing.T) {
	szGot := ExpandProxyCommand("ssh -W %h:%p -l %r bastion 100%% %x end%", "10.0.0.5", 2222, "kai")
	if szWant := "ssh -W 10.0.0.5:2222 -l kai bastion 100% %x end%"; szGot != szWant {
		t.Fatalf("= %q, 기대 %q", szGot, szWant)
	}
}

func TestCommandProgram(t *testing.T) {
	cases := map[string]string{
		"ssh -W %h:%p bastion":                                 "ssh",
		`"C:\Program Files\Nmap\ncat.exe" --proxy p:8080 %h %p`: `C:\Program Files\Nmap\ncat.exe`,
		"  cloudflared  ":                                      "cloudflared",
	}
	for szIn, szWant := range cases {
		if szGot := commandProgram(szIn); szGot != szWant {
			t.Errorf("%q → %q, 기대 %q", szIn, szGot, szWant)
		}
	}
}

// 쓰면 되돌아온다 (명령의 입출력이 연결처럼 동작한다).
func TestDialCommandEcho(t *testing.T) {
	conn, errDial := dialCommand(helperCommand(t, "echo"), "h", 22, "u")
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	if _, errWrite := conn.Write([]byte("SSH-2.0-test\n")); errWrite != nil {
		t.Fatal(errWrite)
	}
	bBuf := make([]byte, len("SSH-2.0-test\n"))
	if _, errRead := io.ReadFull(conn, bBuf); errRead != nil || string(bBuf) != "SSH-2.0-test\n" {
		t.Fatalf("되돌아온 것 = %q, %v", bBuf, errRead)
	}
}

// 명령이 곧바로 끝나면 EOF 이고, 오류에 명령의 표준 오류가 실린다.
func TestDialCommandFailShowsStderr(t *testing.T) {
	conn, errDial := dialCommand(helperCommand(t, "fail"), "h", 22, "u")
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	_, errRead := io.ReadAll(conn)
	if errRead == nil {
		// ReadAll 은 EOF 를 오류로 보지 않는다: 한 번 더 읽어 본다.
		_, errRead = conn.Read(make([]byte, 1))
	}
	if errRead == nil || !strings.Contains(errRead.Error(), "boom: no route to bastion") {
		t.Fatalf("오류 = %v", errRead)
	}
}

// 시한 안에 아무것도 오지 않으면 명령을 끝내고 읽기가 오류로 끝난다.
func TestDialCommandFirstByteTimeout(t *testing.T) {
	durOld := G_durCommandFirstByte
	G_durCommandFirstByte = 300 * time.Millisecond
	defer func() { G_durCommandFirstByte = durOld }()
	conn, errDial := dialCommand(helperCommand(t, "silent"), "h", 22, "u")
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	tmStart := time.Now()
	_, errRead := conn.Read(make([]byte, 1))
	if errRead == nil {
		t.Fatal("오류가 나야 한다")
	}
	if durElapsed := time.Since(tmStart); durElapsed > 5*time.Second {
		t.Fatalf("너무 오래 걸림: %s", durElapsed)
	}
}

// 빈 명령은 띄우지 않는다.
func TestDialCommandEmpty(t *testing.T) {
	if _, errDial := dialCommand("   ", "h", 22, "u"); errDial == nil {
		t.Fatal("오류가 나야 한다")
	}
}

// 통합: 도우미가 %h %p 로 시험용 SSH 서버에 중계하면 Connect 가 핸드셰이크·비밀번호 재시도(명령을 다시 띄움)까지 성공한다.
func TestSSHConnectThroughProxyCommand(t *testing.T) {
	iPort, fnStop := startServer(t)
	defer fnStop()
	tgt := targetFor(iPort, "")
	tgt.Proxy = &Proxy{Type: G_cszProxyCommand, Command: helperCommand(t, "bridge", "%h", "%p")}
	var slNotices []string
	conn, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{password: g_szPass, approve: true}, func(szLine string) { slNotices = append(slNotices, szLine) })
	if errConnect != nil {
		t.Fatalf("프록시 명령 경유 접속: %v", errConnect)
	}
	_ = conn.Close()
	bFound := false
	for _, szLine := range slNotices {
		if strings.Contains(szLine, ".exe") || strings.Contains(szLine, "sshconn") {
			bFound = true
		}
	}
	if !bFound {
		t.Fatalf("프록시 명령 안내 줄이 있어야 함: %v", slNotices)
	}
}
