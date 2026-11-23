package sshconn

import (
	"errors"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"dariterm/internal/i18n"

	"dariterm/internal/transport"
)

// Shell 은 SSH 셸 세션 하나다. session.Transport 를 만족한다.
type Shell struct {
	conn      *Conn
	session   *ssh.Session
	stdin     io.WriteCloser
	outReader *io.PipeReader
	outWriter *io.PipeWriter
	mtxClose  sync.Once
}

// ShellOptions 는 셸을 열 때의 선택 사항이다.
type ShellOptions struct {
	// AgentForward 가 true 면 이 PC 의 SSH 에이전트를 서버에서 쓸 수 있게 전달한다 (protocols.md §2.8, D111).
	AgentForward bool
	// Notice 는 터미널 안내 줄을 보낸다 (에이전트 전달을 못 했을 때). nil 이면 안내하지 않는다.
	Notice func(string)
}

// errNoAgent 는 전달할 에이전트가 이 PC 에 없을 때의 내부 표시다.
var errNoAgent = errors.New("no ssh agent")

// enableAgentForward 는 이 연결에서 서버가 여는 에이전트 채널을 이 PC 의 첫 번째 에이전트로 잇는다.
// 연결마다 한 번만 등록하고 결과를 기억한다.
func (pConn *Conn) enableAgentForward() error {
	pConn.onceAgent.Do(func() {
		slAgents := g_fnAvailableAgents()
		if len(slAgents) == 0 {
			pConn.errAgent = errNoAgent
			return
		}
		pConn.errAgent = agent.ForwardToAgent(pConn.Client, slAgents[0])
	})
	return pConn.errAgent
}

// requestAgentForward 는 세션에 에이전트 전달을 요청한다. 실패해도 셸은 그대로 연다(안내만 한다).
func requestAgentForward(conn *Conn, session *ssh.Session, notice func(string)) {
	if notice == nil {
		notice = func(string) {}
	}
	if errEnable := conn.enableAgentForward(); errEnable != nil {
		if errors.Is(errEnable, errNoAgent) {
			notice(i18n.T("be.ssh.agentFwdNoAgent"))
		} else {
			notice(i18n.T("be.ssh.agentFwdRefused", "error", errEnable))
		}
		return
	}
	if errRequest := agent.RequestAgentForwarding(session); errRequest != nil {
		notice(i18n.T("be.ssh.agentFwdRefused", "error", errRequest))
	}
}

// OpenShell 은 PTY 를 요청하고 셸을 시작한다 (protocols.md §2.1 3·4번).
// termType 이 비면 xterm-256color, 크기는 각각 최소 2 다.
func OpenShell(conn *Conn, termType string, iCols int, iRows int) (*Shell, error) {
	return OpenShellWith(conn, termType, iCols, iRows, ShellOptions{})
}

// OpenShellWith 는 선택 사항(에이전트 전달)을 주어 셸을 연다.
func OpenShellWith(conn *Conn, termType string, iCols int, iRows int, opts ShellOptions) (*Shell, error) {
	if termType == "" {
		termType = "xterm-256color"
	}
	if iCols < 2 {
		iCols = 2
	}
	if iRows < 2 {
		iRows = 2
	}
	session, errSession := conn.Client.NewSession()
	if errSession != nil {
		return nil, errSession
	}
	if opts.AgentForward {
		requestAgentForward(conn, session, opts.Notice)
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if errPty := session.RequestPty(termType, iRows, iCols, modes); errPty != nil {
		_ = session.Close()
		return nil, errPty
	}
	stdin, errIn := session.StdinPipe()
	if errIn != nil {
		_ = session.Close()
		return nil, errIn
	}
	outReader, outWriter := io.Pipe()
	session.Stdout = outWriter
	session.Stderr = outWriter
	if errShell := session.Shell(); errShell != nil {
		_ = session.Close()
		return nil, errShell
	}
	sh := &Shell{conn: conn, session: session, stdin: stdin, outReader: outReader, outWriter: outWriter}
	// 셸이 끝나면 출력 파이프를 닫아 Read 가 끝나게 한다. 끝난 방식을 파이프 오류로 넘긴다:
	// 서버가 종료 코드·신호를 보내고 끝냈으면 정상 종료(errShellExited), 그 밖에는 끊김.
	// 먼저 닫은 쪽의 오류가 남으므로(io.Pipe), Close·keepalive 실패로 먼저 닫히면 끊김이 된다.
	go func() {
		errWait := session.Wait()
		if isNormalExit(errWait) {
			_ = outWriter.CloseWithError(errShellExited)
			return
		}
		_ = outWriter.Close()
	}()
	return sh, nil
}

// errShellExited 는 셸이 정상 종료됐음을 출력 파이프로 알리는 내부 표시다.
var errShellExited = errors.New("shell exited")

// isNormalExit 는 session.Wait 결과가 서버의 정상 종료인지 본다.
// 오류 없음(종료 코드 0) 또는 *ssh.ExitError(0 이 아닌 종료 코드나 신호)면 정상 종료다.
// *ssh.ExitMissingError(종료 코드 없이 채널이 닫힘)와 전송 오류는 끊김이다.
func isNormalExit(errWait error) bool {
	if errWait == nil {
		return true
	}
	var pExit *ssh.ExitError
	return errors.As(errWait, &pExit)
}

// Read 는 셸 출력을 읽는다. 서버가 셸을 정상 종료했으면 transport.ErrRemoteExited,
// 그 밖에 끝나면 "연결이 종료되었습니다"(transport.ErrRemoteClosed) 를 돌려준다 (protocols.md §2.1 6번).
func (pShell *Shell) Read(bBuf []byte) (int, error) {
	nRead, errRead := pShell.outReader.Read(bBuf)
	if errRead != nil {
		if errors.Is(errRead, errShellExited) {
			return nRead, transport.ErrRemoteExited
		}
		return nRead, transport.ErrRemoteClosed
	}
	return nRead, nil
}

// Write 는 입력 바이트를 셸로 보낸다.
func (pShell *Shell) Write(bData []byte) error {
	_, errWrite := pShell.stdin.Write(bData)
	return errWrite
}

// Resize 는 PTY 크기를 바꾼다 (window-change).
func (pShell *Shell) Resize(iCols int, iRows int) error {
	if iCols < 2 || iRows < 2 {
		return nil
	}
	return pShell.session.WindowChange(iRows, iCols)
}

// Close 는 셸 세션을 닫는다. 사용자가 닫을 때 쓴다 (끊김 안내 없음은 session 쪽에서 처리한다).
func (pShell *Shell) Close() error {
	var errClose error
	pShell.mtxClose.Do(func() {
		errClose = pShell.session.Close()
		_ = pShell.outWriter.Close()
	})
	return errClose
}
