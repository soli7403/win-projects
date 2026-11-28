package sshconn

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"dariterm/internal/i18n"
)

// 프록시 명령 (protocols.md §2.7a, D152): 서버까지의 통로를 TCP 대신 외부 명령의 표준 입출력으로 만든다.
// OpenSSH 의 ProxyCommand 와 같다 (예: `ssh -W %h:%p bastion`, `ncat --proxy proxy:8080 %h %p`).

// G_cszProxyCommand 는 프록시 종류 "명령"이다 (data/store.md §3 proxy.type).
const G_cszProxyCommand = "command"

// g_ciCommandStderrMax 는 오류 안내에 붙일 명령의 표준 오류 끝부분 최대 길이(바이트)다.
const g_ciCommandStderrMax = 512

// G_durCommandFirstByte 는 명령을 띄운 뒤 첫 응답(서버 인사)을 기다리는 최대 시간이다. 넘으면 명령을 끝낸다.
// 비밀번호를 묻는 명령(창이 없어 입력할 수 없다)이 영원히 멈추지 않게 한다. 테스트에서 줄일 수 있게 변수로 둔다.
var G_durCommandFirstByte = G_cdurConnectTimeout

// ExpandProxyCommand 는 명령의 자리표시자를 바꾼다: %h 대상 호스트, %p 포트, %r 사용자 이름, %% 는 % 하나.
// 그 밖의 %x 와 끝의 % 는 그대로 둔다(모르는 것을 지우면 명령이 망가진다).
func ExpandProxyCommand(szCommand string, szHost string, iPort int, szUser string) string {
	var sb strings.Builder
	for iIndex := 0; iIndex < len(szCommand); iIndex++ {
		ch := szCommand[iIndex]
		if ch != '%' || iIndex+1 >= len(szCommand) {
			sb.WriteByte(ch)
			continue
		}
		switch szCommand[iIndex+1] {
		case 'h':
			sb.WriteString(szHost)
		case 'p':
			sb.WriteString(strconv.Itoa(iPort))
		case 'r':
			sb.WriteString(szUser)
		case '%':
			sb.WriteByte('%')
		default:
			sb.WriteByte('%')
			sb.WriteByte(szCommand[iIndex+1])
		}
		iIndex++
	}
	return sb.String()
}

// commandProgram 은 안내에 보일 프로그램 이름이다 (명령의 첫 낱말, 따옴표로 감싼 경로는 따옴표 안).
// 인자에 토큰·비밀번호가 들 수 있어 전체는 보이지 않는다.
func commandProgram(szCommand string) string {
	szLine := strings.TrimSpace(szCommand)
	if strings.HasPrefix(szLine, `"`) {
		if iEnd := strings.Index(szLine[1:], `"`); iEnd >= 0 {
			return szLine[1 : 1+iEnd]
		}
		return strings.Trim(szLine, `"`)
	}
	if iSpace := strings.IndexAny(szLine, " \t"); iSpace >= 0 {
		return szLine[:iSpace]
	}
	return szLine
}

// tailBuffer 는 쓰인 바이트 중 끝부분(g_ciCommandStderrMax)만 남기는 버퍼다 (명령의 표준 오류).
type tailBuffer struct {
	mtxData sync.Mutex
	bData   []byte
}

func (pBuf *tailBuffer) Write(bChunk []byte) (int, error) {
	pBuf.mtxData.Lock()
	defer pBuf.mtxData.Unlock()
	pBuf.bData = append(pBuf.bData, bChunk...)
	if iOver := len(pBuf.bData) - g_ciCommandStderrMax; iOver > 0 {
		pBuf.bData = append([]byte{}, pBuf.bData[iOver:]...)
	}
	return len(bChunk), nil
}

func (pBuf *tailBuffer) String() string {
	pBuf.mtxData.Lock()
	defer pBuf.mtxData.Unlock()
	return strings.TrimSpace(string(pBuf.bData))
}

// commandConn 은 외부 명령의 표준 출력을 읽고 표준 입력에 쓰는 net.Conn 이다.
// 시한(SetDeadline)은 지원하지 않는다 — 대신 첫 바이트를 G_durCommandFirstByte 안에 받지 못하면 명령을 끝낸다.
type commandConn struct {
	pCmd      *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	pStderr   *tailBuffer
	szProgram string
	fnKill    func() // 명령과 그 아래 프로세스를 모두 끝낸다
	onceClose sync.Once
	onceFirst sync.Once
	mtxState  sync.Mutex
	timerWait *time.Timer
	bTimedOut bool
}

// dialCommand 는 명령을 띄워 그 표준 입출력을 연결로 돌려준다.
func dialCommand(szCommand string, szHost string, iPort int, szUser string) (net.Conn, error) {
	szLine := ExpandProxyCommand(strings.TrimSpace(szCommand), szHost, iPort, szUser)
	if szLine == "" {
		return nil, i18n.NewError("be.proxy.cmdEmpty")
	}
	pCmd := shellCommand(szLine)
	stdin, errIn := pCmd.StdinPipe()
	if errIn != nil {
		return nil, errIn
	}
	stdout, errOut := pCmd.StdoutPipe()
	if errOut != nil {
		_ = stdin.Close()
		return nil, errOut
	}
	pStderr := &tailBuffer{}
	pCmd.Stderr = pStderr
	szProgram := commandProgram(szLine)
	if errStart := pCmd.Start(); errStart != nil {
		return nil, i18n.Errorf("be.proxy.cmdStart", errStart, "program", szProgram)
	}
	pConn := &commandConn{pCmd: pCmd, stdin: stdin, stdout: stdout, pStderr: pStderr, szProgram: szProgram, fnKill: treeKiller(pCmd)}
	// 시한 타이머는 잠금 아래에서 건다: 타이머가 아주 빨리 울려도 timerWait 를 다 넣은 뒤에 읽는다.
	pConn.mtxState.Lock()
	pConn.timerWait = time.AfterFunc(G_durCommandFirstByte, func() {
		pConn.mtxState.Lock()
		pConn.bTimedOut = true
		pConn.mtxState.Unlock()
		_ = pConn.Close()
	})
	pConn.mtxState.Unlock()
	return pConn, nil
}

// stopTimer 는 첫 응답 시한 타이머를 멈춘다.
func (pConn *commandConn) stopTimer() {
	pConn.mtxState.Lock()
	defer pConn.mtxState.Unlock()
	if pConn.timerWait != nil {
		pConn.timerWait.Stop()
	}
}

// Read 는 명령의 표준 출력을 읽는다. 명령이 끝나 더 읽을 것이 없으면 표준 오류의 끝부분을 붙인 EOF 를 돌려준다.
func (pConn *commandConn) Read(bBuf []byte) (int, error) {
	nRead, errRead := pConn.stdout.Read(bBuf)
	if nRead > 0 {
		pConn.onceFirst.Do(pConn.stopTimer)
	}
	if errRead == nil {
		return nRead, nil
	}
	pConn.mtxState.Lock()
	bTimedOut := pConn.bTimedOut
	pConn.mtxState.Unlock()
	if bTimedOut {
		return nRead, i18n.Errorf("be.proxy.cmdTimeout", nil, "program", pConn.szProgram, "sec", int(G_durCommandFirstByte/time.Second))
	}
	if errors.Is(errRead, io.EOF) || errors.Is(errRead, io.ErrClosedPipe) {
		if szErr := pConn.pStderr.String(); szErr != "" {
			return nRead, fmt.Errorf("%w (%s: %s)", io.EOF, pConn.szProgram, szErr)
		}
		return nRead, io.EOF
	}
	return nRead, errRead
}

// Write 는 명령의 표준 입력에 쓴다.
func (pConn *commandConn) Write(bBuf []byte) (int, error) {
	return pConn.stdin.Write(bBuf)
}

// Close 는 표준 입력을 닫고 명령을 끝낸다. 여러 번 불러도 한 번만 한다.
func (pConn *commandConn) Close() error {
	pConn.onceClose.Do(func() {
		pConn.stopTimer()
		_ = pConn.stdin.Close()
		pConn.fnKill()
		go func() { _ = pConn.pCmd.Wait() }() // 끝난 명령을 거둔다 (좀비 방지)
	})
	return nil
}

// commandAddr 는 표시용 주소다.
type commandAddr string

func (addr commandAddr) Network() string { return "command" }
func (addr commandAddr) String() string  { return string(addr) }

func (pConn *commandConn) LocalAddr() net.Addr  { return commandAddr("local") }
func (pConn *commandConn) RemoteAddr() net.Addr { return commandAddr(pConn.szProgram) }

// SetDeadline 들은 아무것도 하지 않는다 (명령의 파이프에는 시한이 없다. 첫 응답 시한은 위의 타이머가 맡는다).
func (pConn *commandConn) SetDeadline(time.Time) error      { return nil }
func (pConn *commandConn) SetReadDeadline(time.Time) error  { return nil }
func (pConn *commandConn) SetWriteDeadline(time.Time) error { return nil }
