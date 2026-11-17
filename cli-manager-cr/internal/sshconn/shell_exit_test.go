package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"dariterm/internal/transport"
)

// startExitServer 는 셸 입력 한 줄에 따라 끝내는 방식을 고르는 시험 서버다.
//   - "exit0" / "exit3": 종료 코드 0 / 3 을 보내고 채널을 닫는다 (정상 종료)
//   - "signal": exit-signal(TERM) 을 보내고 닫는다 (정상 종료)
//   - "nostatus": 종료 코드 없이 채널만 닫는다 (끊김)
//   - "dropconn": TCP 연결 자체를 닫는다 (끊김)
func startExitServer(t *testing.T) int {
	t.Helper()
	_, priv, errKey := ed25519.GenerateKey(rand.Reader)
	if errKey != nil {
		t.Fatal(errKey)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
		if string(pw) == g_szPass {
			return nil, nil
		}
		return nil, errors.New("denied")
	}}
	cfg.AddHostKey(signer)
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func(c net.Conn) {
				sconn, chans, reqs, errHs := ssh.NewServerConn(c, cfg)
				if errHs != nil {
					return
				}
				defer sconn.Close()
				go ssh.DiscardRequests(reqs)
				for nc := range chans {
					ch, chReqs, errAcc := nc.Accept()
					if errAcc != nil {
						continue
					}
					go func(ch ssh.Channel, chReqs <-chan *ssh.Request) {
						for req := range chReqs {
							_ = req.Reply(req.Type == "pty-req" || req.Type == "shell", nil)
							if req.Type != "shell" {
								continue
							}
							go func() {
								bBuf := make([]byte, 64)
								n, _ := ch.Read(bBuf)
								switch strings.TrimSpace(string(bBuf[:n])) {
								case "exit0":
									_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
									_ = ch.Close()
								case "exit3":
									_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{3}))
									_ = ch.Close()
								case "signal":
									_, _ = ch.SendRequest("exit-signal", false, ssh.Marshal(struct {
										Signal     string
										CoreDumped bool
										Error      string
										Lang       string
									}{"TERM", false, "", ""}))
									_ = ch.Close()
								case "nostatus":
									_ = ch.Close()
								case "dropconn":
									_ = c.Close()
								}
							}()
						}
					}(ch, chReqs)
				}
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// readUntilEnd 는 셸이 끝날 때까지 읽고 마지막 오류를 돌려준다.
func readUntilEnd(t *testing.T, shell *Shell) error {
	t.Helper()
	chErr := make(chan error, 1)
	go func() {
		bBuf := make([]byte, 1024)
		for {
			if _, errRead := shell.Read(bBuf); errRead != nil {
				chErr <- errRead
				return
			}
		}
	}()
	select {
	case errEnd := <-chErr:
		return errEnd
	case <-time.After(5 * time.Second):
		t.Fatal("셸 종료 대기 시간 초과")
		return nil
	}
}

// 셸이 끝난 방식을 정상 종료(ErrRemoteExited)와 끊김(ErrRemoteClosed)으로 나눈다 (session.md §4 안 F).
func TestShellEndKind(t *testing.T) {
	port := startExitServer(t)
	cases := []struct {
		szCmd  string
		errEnd error
	}{
		{"exit0", transport.ErrRemoteExited},
		{"exit3", transport.ErrRemoteExited},
		{"signal", transport.ErrRemoteExited},
		{"nostatus", transport.ErrRemoteClosed},
		{"dropconn", transport.ErrRemoteClosed},
	}
	for _, tc := range cases {
		t.Run(tc.szCmd, func(t *testing.T) {
			store := &memStore{m: map[string]string{}}
			prompt := &fakePrompt{password: g_szPass, approve: true}
			conn, errConnect := Connect(targetFor(port, g_szPass), store, prompt, func(string) {})
			if errConnect != nil {
				t.Fatal(errConnect)
			}
			defer conn.Close()
			shell, errShell := OpenShell(conn, "xterm", 80, 24)
			if errShell != nil {
				t.Fatal(errShell)
			}
			defer shell.Close()
			if errWrite := shell.Write([]byte(tc.szCmd + "\n")); errWrite != nil {
				t.Fatal(errWrite)
			}
			if errEnd := readUntilEnd(t, shell); !errors.Is(errEnd, tc.errEnd) {
				t.Fatalf("%s: 끝 오류 %v, 기대 %v", tc.szCmd, errEnd, tc.errEnd)
			}
		})
	}
}

// 사용자가 닫으면(Close) 정상 종료로 보지 않는다.
func TestShellCloseIsNotExit(t *testing.T) {
	port := startExitServer(t)
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: true}
	conn, errConnect := Connect(targetFor(port, g_szPass), store, prompt, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	shell, errShell := OpenShell(conn, "xterm", 80, 24)
	if errShell != nil {
		t.Fatal(errShell)
	}
	_ = shell.Close()
	if errEnd := readUntilEnd(t, shell); errors.Is(errEnd, transport.ErrRemoteExited) {
		t.Fatal("Close 는 정상 종료가 아님")
	}
}

// 접속에 성공하면 실제로 쓰인 사용자 이름과 입력한 비밀번호가 Target 에 남는다 (세션 사본용).
func TestConnectRemembersWorkingSecret(t *testing.T) {
	port := startExitServer(t)
	store := &memStore{m: map[string]string{}}
	prompt := &fakePrompt{password: g_szPass, approve: true}
	tgt := targetFor(port, "wrong")
	tgt.Username = ""
	conn, errConnect := Connect(tgt, store, prompt, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	defer conn.Close()
	if tgt.Username != g_szUser || tgt.Password != g_szPass {
		t.Fatalf("성공한 계정이 남아야 함: user=%q pw=%q", tgt.Username, tgt.Password)
	}
}
