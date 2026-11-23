package sshconn

import (
	"dariterm/internal/i18n"
	"errors"
	"sync"

	"github.com/pkg/sftp"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ErrJumpTooDeep 는 점프 호스트 체인이 너무 길 때의 오류다 (protocols.md §2.4).
var ErrJumpTooDeep = i18n.NewError("be.ssh.jumpTooDeep")

// ErrNoUsername 은 사용자 이름이 입력되지 않았을 때의 오류다 (protocols.md §2.2).
var ErrNoUsername = i18n.NewError("be.ssh.noUser")

// Conn 은 대상 서버까지의 SSH 연결이다. 점프 호스트가 있으면 Hops 에 그 연결들을 가진다.
type Conn struct {
	Client     *ssh.Client
	Hops       []*ssh.Client
	Username   string
	Asked      askedSecret // 대상 서버의 비밀번호 입력 결과 (인증 성공 후 저장 판단에 쓴다)
	mtxSftp    sync.Mutex
	sftpClient *sftp.Client
	chClosed   chan struct{} // Close 때 닫힌다 (keepalive 루프 종료용)
	onceClose  sync.Once
	onceAgent  sync.Once // 에이전트 전달 채널 처리기는 연결마다 한 번만 등록한다 (D111)
	errAgent   error     // onceAgent 의 결과
}

// Close 는 대상과 모든 점프 연결을 닫는다 (protocols.md §2.4). 두 번 불러도 안전하다.
func (pConn *Conn) Close() error {
	var errFirst error
	pConn.onceClose.Do(func() {
		if pConn.chClosed != nil {
			close(pConn.chClosed)
		}
		if pConn.Client != nil {
			errFirst = pConn.Client.Close()
		}
		for iIndex := len(pConn.Hops) - 1; iIndex >= 0; iIndex-- {
			_ = pConn.Hops[iIndex].Close()
		}
	})
	return errFirst
}

// Connect 는 tgt 까지 SSH 로 접속한다. 점프 호스트가 있으면 재귀로 먼저 접속한다.
// notice 는 터미널 안내 줄을 보낸다 (예: "점프 호스트 {별칭} 경유 중...").
func Connect(tgt *Target, store HostKeyStore, prompt Prompter, notice func(string)) (*Conn, error) {
	if tgt.Depth > G_iJumpDepthMax {
		return nil, ErrJumpTooDeep
	}
	szUsername := tgt.Username
	if szUsername == "" {
		szAsk, bOk := prompt.Username(tgt.Title)
		if !bOk || szAsk == "" {
			return nil, ErrNoUsername
		}
		szUsername = szAsk
	}
	addr := net.JoinHostPort(tgt.Host, strconv.Itoa(tgt.Port))
	asked := &askedSecret{}

	var hops []*ssh.Client
	var conn net.Conn
	if tgt.Jump != nil {
		tgt.Jump.Depth = tgt.Depth + 1
		notice(i18n.T("be.ssh.viaJump", "name", tgt.Jump.Title))
		jumpConn, errJump := Connect(tgt.Jump, store, prompt, notice)
		if errJump != nil {
			return nil, errJump
		}
		hops = append(append(hops, jumpConn.Hops...), jumpConn.Client)
		conn, errJump = jumpConn.Client.Dial("tcp", addr)
		if errJump != nil {
			closeHops(hops)
			return nil, errJump
		}
	} else {
		if tgt.Proxy != nil {
			notice(i18n.T("be.proxy.via", "addr", net.JoinHostPort(tgt.Proxy.Host, strconv.Itoa(tgt.Proxy.Port))))
		}
		var errDial error
		conn, errDial = dialDirect(tgt, addr) // 프록시가 있으면 거쳐서 (D110)
		if errDial != nil {
			return nil, proxyError(errDial)
		}
	}

	// 1차 시도: 키·에이전트·저장된 비밀번호·키보드-인터랙티브.
	config := &ssh.ClientConfig{
		User:            szUsername,
		Auth:            authMethodsFor(tgt, szUsername, prompt, agentSignersFor(), asked, notice),
		HostKeyCallback: hostKeyCallbackFor(tgt, store, prompt),
		Timeout:         G_cdurConnectTimeout,
	}
	client, errHandshake := handshake(conn, addr, config)
	if errHandshake == nil {
		rememberWorking(tgt, szUsername, asked)
		return &Conn{Client: client, Hops: hops, Username: szUsername, Asked: *asked, chClosed: make(chan struct{})}, nil
	}
	if !isAuthFailure(errHandshake) {
		_ = conn.Close()
		closeHops(hops)
		return nil, errHandshake
	}
	// 2차: 비밀번호 직접 입력. 실패할 때마다 연결을 새로 맺어 최대 4번 묻는다 (protocols.md §2.2 5번).
	for iAttempt := 0; iAttempt < G_iPasswordMax; iAttempt++ {
		szPassword, bSave, bOk := askPasswordFor(tgt, szUsername, prompt, asked, iAttempt > 0)
		if !bOk {
			closeHops(hops)
			return nil, errCancelled
		}
		asked.Password, asked.Save, asked.Asked = szPassword, bSave, true
		nextConn, errNext := dialNext(tgt, addr, jumpClientOf(hops))
		if errNext != nil {
			closeHops(hops)
			return nil, errNext
		}
		cfgPw := &ssh.ClientConfig{
			User:            szUsername,
			Auth:            []ssh.AuthMethod{ssh.Password(szPassword)},
			HostKeyCallback: hostKeyCallbackFor(tgt, store, prompt),
			Timeout:         G_cdurConnectTimeout,
		}
		client, errPw := handshake(nextConn, addr, cfgPw)
		if errPw == nil {
			rememberWorking(tgt, szUsername, asked)
			return &Conn{Client: client, Hops: hops, Username: szUsername, Asked: *asked, chClosed: make(chan struct{})}, nil
		}
		if !isAuthFailure(errPw) {
			closeHops(hops)
			return nil, errPw
		}
	}
	closeHops(hops)
	return nil, i18n.NewError("be.err.auth")
}

// rememberWorking 은 접속에 성공했을 때 실제로 쓰인 사용자 이름과 사용자가 입력한 비밀번호를 tgt 에 남긴다.
// 호출자는 접속 뒤 tgt(와 Jump 체인)를 읽어 세션 동안의 계정 사본으로 쓴다 (protocols.md §2.2 세션 사본).
// 입력한 키 암호는 keyLoader 가 성공할 때 tgt.Passphrase 에 남긴다.
func rememberWorking(tgt *Target, szUsername string, asked *askedSecret) {
	tgt.Username = szUsername
	if asked.Asked && asked.Password != "" {
		tgt.Password = asked.Password
	}
}

// askPasswordFor 는 비밀번호 요청 창을 띄운다 (bFailed 면 실패 표시).
func askPasswordFor(tgt *Target, username string, prompt Prompter, asked *askedSecret, bFailed bool) (string, bool, bool) {
	szPassword, bSave, bOk := prompt.Password(tgt.Title, i18n.T("be.ssh.passwordPrompt", "user", username, "host", tgt.Host), username, bFailed)
	if !bOk {
		return "", false, false
	}
	return szPassword, bSave && tgt.AllowSave, true
}

// handshake 는 이미 맺은 연결 위에서 SSH 핸드셰이크를 한다. 실패하면 연결을 닫는다.
func handshake(conn net.Conn, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	sshConn, chans, reqs, errHandshake := ssh.NewClientConn(conn, addr, config)
	if errHandshake != nil {
		_ = conn.Close()
		return nil, errHandshake
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

// isAuthFailure 는 인증 단계의 실패인지 본다 (호스트 키 거부 등은 제외).
func isAuthFailure(err error) bool {
	return strings.Contains(err.Error(), "unable to authenticate")
}

// jumpClientOf 는 마지막 점프 클라이언트를 돌려준다 (없으면 nil).
func jumpClientOf(hops []*ssh.Client) *ssh.Client {
	if len(hops) == 0 {
		return nil
	}
	return hops[len(hops)-1]
}

// dialNext 는 대상으로 새 TCP 연결을 연다 (점프가 있으면 마지막 점프를 거쳐).
func dialNext(tgt *Target, addr string, jump *ssh.Client) (net.Conn, error) {
	if jump != nil {
		return jump.Dial("tcp", addr)
	}
	conn, errDial := dialDirect(tgt, addr)
	return conn, proxyError(errDial)
}

// proxyError 는 프록시 인증 실패를 안내 문구가 있는 오류로 바꾼다. 그 밖의 오류는 그대로 둔다.
func proxyError(errDial error) error {
	if errors.Is(errDial, ErrProxyAuth) {
		return i18n.NewError("be.proxy.auth")
	}
	return errDial
}

func closeHops(hops []*ssh.Client) {
	for iIndex := len(hops) - 1; iIndex >= 0; iIndex-- {
		_ = hops[iIndex].Close()
	}
}
