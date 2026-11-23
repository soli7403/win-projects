package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// 에이전트 전달 시험 (D111).

// startAgentServer 는 에이전트 전달 요청을 받는 시험용 SSH 서버다.
// 세션에 auth-agent-req@openssh.com 이 오면(bAccept 일 때 수락) 서버 쪽에서 에이전트 채널을 열어
// 키 주석 목록을 chKeys 로 보낸다. 요청이 온 횟수는 chRequests 로 알린다.
func startAgentServer(t *testing.T, bAccept bool) (int, chan []string, chan struct{}) {
	t.Helper()
	_, hostKey, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(hostKey)
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	chKeys := make(chan []string, 4)
	chRequests := make(chan struct{}, 4)
	go func() {
		for {
			c, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				sconn, chans, reqs, errHs := ssh.NewServerConn(c, cfg)
				if errHs != nil {
					return
				}
				defer sconn.Close()
				go ssh.DiscardRequests(reqs)
				for newCh := range chans {
					if newCh.ChannelType() != "session" {
						_ = newCh.Reject(ssh.UnknownChannelType, "unknown")
						continue
					}
					ch, chReqs, errCh := newCh.Accept()
					if errCh != nil {
						continue
					}
					go func() {
						defer ch.Close()
						for req := range chReqs {
							switch req.Type {
							case "auth-agent-req@openssh.com":
								chRequests <- struct{}{}
								_ = req.Reply(bAccept, nil)
								if !bAccept {
									continue
								}
								go func() {
									agentCh, agentReqs, errOpen := sconn.OpenChannel("auth-agent@openssh.com", nil)
									if errOpen != nil {
										chKeys <- []string{"open failed: " + errOpen.Error()}
										return
									}
									go ssh.DiscardRequests(agentReqs)
									defer agentCh.Close()
									slKeys, errList := agent.NewClient(agentCh).List()
									if errList != nil {
										chKeys <- []string{"list failed: " + errList.Error()}
										return
									}
									var slComments []string
									for _, pKey := range slKeys {
										slComments = append(slComments, pKey.Comment)
									}
									chKeys <- slComments
								}()
							case "pty-req", "shell":
								_ = req.Reply(true, nil)
							default:
								_ = req.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, chKeys, chRequests
}

// withAgents 는 시험 동안 "이 PC 의 에이전트" 목록을 바꾼다.
func withAgents(t *testing.T, slAgents []agent.ExtendedAgent) {
	t.Helper()
	fnOld := g_fnAvailableAgents
	g_fnAvailableAgents = func() []agent.ExtendedAgent { return slAgents }
	t.Cleanup(func() { g_fnAvailableAgents = fnOld })
}

func keyringWith(t *testing.T, szComment string) agent.ExtendedAgent {
	t.Helper()
	keyring := agent.NewKeyring()
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	if errAdd := keyring.Add(agent.AddedKey{PrivateKey: privKey, Comment: szComment}); errAdd != nil {
		t.Fatal(errAdd)
	}
	return keyring.(agent.ExtendedAgent)
}

func connectNoAuth(t *testing.T, iPort int) *Conn {
	t.Helper()
	tgt := &Target{Title: "t", Host: "127.0.0.1", Port: iPort, Username: "u", AuthType: "ask"}
	conn, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{approve: true}, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// 에이전트 전달을 켜면 서버가 이 PC 에이전트의 키 목록을 볼 수 있다. 같은 연결에서 셸을 두 번 열어도 된다.
func TestAgentForwardReachesLocalAgent(t *testing.T) {
	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "local-agent-key")})
	iPort, chKeys, _ := startAgentServer(t, true)
	conn := connectNoAuth(t, iPort)
	for iRound := 0; iRound < 2; iRound++ {
		var slNotices []string
		shell, errShell := OpenShellWith(conn, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
		if errShell != nil {
			t.Fatal(errShell)
		}
		select {
		case slKeys := <-chKeys:
			if len(slKeys) != 1 || slKeys[0] != "local-agent-key" {
				t.Fatalf("서버가 본 키 목록: %v", slKeys)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("서버가 전달된 에이전트에 닿지 못함")
		}
		if len(slNotices) != 0 {
			t.Fatalf("성공하면 안내가 없어야 함: %v", slNotices)
		}
		_ = shell.Close()
	}
}

// 끄면(기본) 에이전트 전달을 요청하지 않는다.
func TestAgentForwardOffByDefault(t *testing.T) {
	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "k")})
	iPort, _, chRequests := startAgentServer(t, true)
	conn := connectNoAuth(t, iPort)
	shell, errShell := OpenShell(conn, "xterm", 80, 24)
	if errShell != nil {
		t.Fatal(errShell)
	}
	defer shell.Close()
	select {
	case <-chRequests:
		t.Fatal("끄면 요청하지 않아야 함")
	case <-time.After(150 * time.Millisecond):
	}
}

// 에이전트가 없거나 서버가 거절하면 안내 한 줄을 내고 셸은 그대로 연다.
func TestAgentForwardFailuresOnlyNotify(t *testing.T) {
	iPort, _, chRequests := startAgentServer(t, false)

	withAgents(t, nil)
	conn := connectNoAuth(t, iPort)
	var slNotices []string
	shell, errShell := OpenShellWith(conn, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
	if errShell != nil {
		t.Fatalf("에이전트가 없어도 셸은 열려야 함: %v", errShell)
	}
	_ = shell.Close()
	if len(slNotices) != 1 {
		t.Fatalf("에이전트 없음 안내 한 줄: %v", slNotices)
	}
	select {
	case <-chRequests:
		t.Fatal("에이전트가 없으면 서버에 요청하지 않아야 함")
	case <-time.After(100 * time.Millisecond):
	}

	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "k")})
	conn2 := connectNoAuth(t, iPort)
	slNotices = nil
	shell2, errShell2 := OpenShellWith(conn2, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
	if errShell2 != nil {
		t.Fatalf("서버가 거절해도 셸은 열려야 함: %v", errShell2)
	}
	_ = shell2.Close()
	if len(slNotices) != 1 || !strings.Contains(slNotices[0], "forwarding request denied") {
		t.Fatalf("거절 안내 한 줄(원인 포함): %v", slNotices)
	}
}
