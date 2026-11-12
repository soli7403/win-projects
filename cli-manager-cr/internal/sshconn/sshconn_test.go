package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	g_szUser = "root"
	g_szPass = "pw"
)

// startServer 는 테스트용 in-process SSH 서버를 띄운다. 셸은 입력을 되돌려 보내고, direct-tcpip 로 점프를 지원한다.
func startServer(t *testing.T) (int, func()) {
	t.Helper()
	_, priv, errKey := ed25519.GenerateKey(rand.Reader)
	if errKey != nil {
		t.Fatal(errKey)
	}
	signer, errSigner := ssh.NewSignerFromKey(priv)
	if errSigner != nil {
		t.Fatal(errSigner)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if c.User() == g_szUser && string(pw) == g_szPass {
				return nil, nil
			}
			return nil, errors.New("denied")
		},
	}
	cfg.AddHostKey(signer)
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	go func() {
		for {
			c, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go serveConn(c, cfg)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, func() { _ = ln.Close() }
}

func serveConn(c net.Conn, cfg *ssh.ServerConfig) {
	sconn, chans, reqs, errHandshake := ssh.NewServerConn(c, cfg)
	if errHandshake != nil {
		_ = c.Close()
		return
	}
	defer sconn.Close()
	go func() {
		for r := range reqs {
			if r.WantReply {
				_ = r.Reply(true, nil)
			}
		}
	}()
	for newCh := range chans {
		switch newCh.ChannelType() {
		case "direct-tcpip":
			var payload struct {
				DestAddr string
				DestPort uint32
				OrigAddr string
				OrigPort uint32
			}
			_ = ssh.Unmarshal(newCh.ExtraData(), &payload)
			dest, errDial := net.Dial("tcp", net.JoinHostPort(payload.DestAddr, strconv.Itoa(int(payload.DestPort))))
			if errDial != nil {
				_ = newCh.Reject(ssh.ConnectionFailed, errDial.Error())
				continue
			}
			ch, chReqs, errAccept := newCh.Accept()
			if errAccept != nil {
				_ = dest.Close()
				continue
			}
			go ssh.DiscardRequests(chReqs)
			go func() { _, _ = io.Copy(ch, dest); _ = ch.Close() }()
			go func() { _, _ = io.Copy(dest, ch); _ = dest.Close() }()
		case "session":
			ch, chReqs, errAccept := newCh.Accept()
			if errAccept != nil {
				continue
			}
			go handleSession(ch, chReqs)
		default:
			_ = newCh.Reject(ssh.UnknownChannelType, "unknown")
		}
	}
}

func handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		switch req.Type {
		case "pty-req", "env":
			_ = req.Reply(true, nil)
		case "shell":
			_ = req.Reply(true, nil)
			go func() { _, _ = io.Copy(ch, ch) }()
		case "window-change":
			_ = req.Reply(true, nil)
		default:
			_ = req.Reply(false, nil)
		}
	}
}

// fakePrompt 은 미리 정한 답을 돌려주는 질문 창구다.
type fakePrompt struct {
	mtxData  sync.Mutex
	password string
	approve  bool
	asked    int
}

func (p *fakePrompt) HostKey(string, string, string, string, bool, string) bool { return p.approve }
func (p *fakePrompt) Password(string, string, string, bool) (string, bool, bool) {
	p.mtxData.Lock()
	defer p.mtxData.Unlock()
	p.asked++
	return p.password, true, true
}
func (p *fakePrompt) Passphrase(string) (string, bool) { return "", false }
func (p *fakePrompt) Username(string) (string, bool)   { return g_szUser, true }
func (p *fakePrompt) Keyboard(string, string, string, []KeyboardPrompt) ([]string, bool) {
	return nil, false
}

// memStore 는 메모리 호스트 키 저장소다.
type memStore struct {
	mtxData sync.Mutex
	m       map[string]string
}

func (s *memStore) Lookup(hp string) (string, bool) {
	s.mtxData.Lock()
	defer s.mtxData.Unlock()
	v, ok := s.m[hp]
	return v, ok
}
func (s *memStore) Save(hp string, _ string, fp string) {
	s.mtxData.Lock()
	defer s.mtxData.Unlock()
	s.m[hp] = fp
}

func targetFor(port int, password string) *Target {
	return &Target{Title: "test", Host: "127.0.0.1", Port: port, Username: g_szUser, AuthType: "ask", Password: password, AllowSave: true}
}

// 비밀번호를 묻고 접속 → 셸 입출력 → 호스트 키 기록. protocols.md §2.2·§2.3.
func TestSSHPasswordPromptAndShellEcho(t *testing.T) {
	port, stop := startServer(t)
	defer stop()
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: true}
	conn, errConnect := Connect(targetFor(port, ""), store, prompt, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	if len(store.m) != 1 {
		t.Fatal("승인한 호스트 키가 기록되어야 함")
	}
	shell, errShell := OpenShell(conn, "xterm", 80, 24)
	if errShell != nil {
		t.Fatal(errShell)
	}
	defer shell.Close()
	if errWrite := shell.Write([]byte("ping\n")); errWrite != nil {
		t.Fatal(errWrite)
	}
	var sbAll strings.Builder
	bBuf := make([]byte, 1024)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for !strings.Contains(sbAll.String(), "ping") {
			n, errRead := shell.Read(bBuf)
			if n > 0 {
				sbAll.Write(bBuf[:n])
			}
			if errRead != nil {
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("셸 에코 대기 시간 초과")
	}
	if !strings.Contains(sbAll.String(), "ping") {
		t.Fatalf("셸 에코가 돌아오지 않음: %q", sbAll.String())
	}
}

// 호스트 키를 승인하지 않으면 접속이 중단된다.
func TestSSHHostKeyRejected(t *testing.T) {
	port, stop := startServer(t)
	defer stop()
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: false}
	_, errConnect := Connect(targetFor(port, ""), store, prompt, func(string) {})
	if errConnect == nil || !strings.Contains(errConnect.Error(), "host key rejected") {
		t.Fatalf("호스트 키 거부 오류 기대: %v", errConnect)
	}
}

// 저장된 비밀번호가 틀리면 비밀번호 직접 입력으로 넘어간다 (protocols.md §2.2 5번).
func TestSSHStoredPasswordFallsBackToPrompt(t *testing.T) {
	port, stop := startServer(t)
	defer stop()
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: true}
	conn, errConnect := Connect(targetFor(port, "wrong"), store, prompt, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	if prompt.asked == 0 {
		t.Fatal("틀린 저장 비밀번호 뒤에는 직접 입력을 물어야 함")
	}
	if !conn.Asked.Asked || conn.Asked.Password != g_szPass {
		t.Fatalf("입력한 비밀번호가 기록되어야 함: %+v", conn.Asked)
	}
}

// 점프 호스트를 거쳐 대상에 접속한다 (protocols.md §2.4).
func TestSSHJumpHost(t *testing.T) {
	port, stop := startServer(t)
	defer stop()
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: true}
	tgt := targetFor(port, g_szPass)
	tgt.Jump = targetFor(port, g_szPass)
	tgt.Jump.Title = "bastion"
	var slNotices []string
	conn, errConnect := Connect(tgt, store, prompt, func(s string) { slNotices = append(slNotices, s) })
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	if len(conn.Hops) != 1 {
		t.Fatalf("점프 연결 하나가 있어야 함: %d", len(conn.Hops))
	}
	if len(slNotices) == 0 || !strings.Contains(slNotices[0], "bastion 경유 중") {
		t.Fatalf("점프 안내 줄 누락: %v", slNotices)
	}
}

// 점프 깊이가 제한을 넘으면 오류다 (protocols.md §2.4).
func TestSSHJumpDepthLimit(t *testing.T) {
	tgt := &Target{Title: "x", Host: "127.0.0.1", Port: 1, Username: "u", Depth: G_iJumpDepthMax + 1}
	_, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{}, func(string) {})
	if !errors.Is(errConnect, ErrJumpTooDeep) {
		t.Fatalf("깊이 오류 기대: %v", errConnect)
	}
}

// keepalive 요청이 서버에 닿는다 (protocols.md §2.6).
func TestKeepaliveRequestReachesServer(t *testing.T) {
	port, stop := startServer(t)
	defer stop()
	conn, errConnect := Connect(targetFor(port, g_szPass), &memStore{m: map[string]string{}}, &fakePrompt{approve: true}, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	ok, _, errSend := conn.Client.SendRequest("keepalive@openssh.com", true, nil)
	if errSend != nil || !ok {
		t.Fatalf("keepalive 응답 실패: %v %v", ok, errSend)
	}
	_ = strconv.Itoa
}
