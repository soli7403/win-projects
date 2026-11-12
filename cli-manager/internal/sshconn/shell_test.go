package sshconn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"termgo/internal/model"
	"termgo/internal/store"
	"termgo/internal/transport"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

// 간이 SSH 서버: 비밀번호 "pw" 만 허용, pty+shell 요청을 받으면 입력을 대문자로 되돌려 준다.
func startServer(t *testing.T) (szAddr string) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, p []byte) (*ssh.Permissions, error) {
		if string(p) == "pw" {
			return nil, nil
		}
		return nil, io.EOF
	}}
	cfg.AddHostKey(signer)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			nc, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(nc, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nch := range chans {
					if nch.ChannelType() == "direct-tcpip" { // 베스천 역할: 요청한 대상으로 이어 준다
						var p struct {
							Host     string
							Port     uint32
							OrigHost string
							OrigPort uint32
						}
						_ = ssh.Unmarshal(nch.ExtraData(), &p)
						out, err := net.Dial("tcp", net.JoinHostPort(p.Host, fmt.Sprint(p.Port)))
						if err != nil {
							_ = nch.Reject(ssh.ConnectionFailed, err.Error())
							continue
						}
						ch, _, _ := nch.Accept()
						go func() { _, _ = io.Copy(out, ch); _ = out.Close() }()
						go func() { _, _ = io.Copy(ch, out); _ = ch.Close() }()
						continue
					}
					ch, creqs, _ := nch.Accept()
					go func() {
						for r := range creqs {
							_ = r.Reply(r.Type == "pty-req" || r.Type == "shell", nil)
						}
					}()
					go func() {
						buf := make([]byte, 256)
						for {
							n, err := ch.Read(buf)
							if n > 0 {
								_, _ = ch.Write(bytes.ToUpper(buf[:n]))
							}
							if err != nil {
								return
							}
						}
					}()
				}
			}()
		}
	}()
	return l.Addr().String()
}

func TestShellPasswordAndHostKey(t *testing.T) {
	szAddr := startServer(t)
	host, portStr, _ := net.SplitHostPort(szAddr)
	var port int
	_, _ = fmtSscan(portStr, &port)

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var br *ui.Bridge
	var nAsk int
	br = ui.NewBridge(func(szCh string, args ...any) {
		if szCh != "ui:request" {
			return
		}
		req := args[0].(map[string]any)
		nAsk++
		switch req["kind"] {
		case "hostkey":
			go br.Respond(req["reqId"].(string), "true")
		case "password":
			go br.Respond(req["reqId"].(string), `{"password":"pw","save":false}`)
		default:
			go br.Respond(req["reqId"].(string), "null")
		}
	})
	d := Deps{Store: st, Vault: vault.Open(t.TempDir()), UI: br}
	h := model.Host{ID: "h1", Alias: "t", Protocol: "ssh", Host: host, Port: port, Username: "u", AuthType: "ask", TermType: "xterm", Forwards: []model.PortForward{}}

	var mu sync.Mutex
	var out strings.Builder
	sh, err := StartShell(d, h, Context{Title: "t", Cols: 80, Rows: 24}, transport.Callbacks{
		OnData:  func(a []byte) { mu.Lock(); out.Write(a); mu.Unlock() },
		OnClose: func(string) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sh.Close()
	_ = sh.Write([]byte("hello"))
	for i := 0; i < 30; i++ {
		mu.Lock()
		s := out.String()
		mu.Unlock()
		if s == "HELLO" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if out.String() != "HELLO" {
		t.Fatalf("out = %q", out.String())
	}
	if _, ok := st.FindKnownHost(szAddr); !ok {
		t.Fatal("host key was not remembered")
	}
	// 두 번째 접속은 호스트 키를 다시 묻지 않는다
	nBefore := nAsk
	sh2, err := StartShell(d, h, Context{Title: "t", Cols: 80, Rows: 24}, transport.Callbacks{OnData: func([]byte) {}, OnClose: func(string) {}})
	if err != nil {
		t.Fatal(err)
	}
	sh2.Close()
	if nAsk-nBefore != 1 { // 비밀번호만 다시 묻는다
		t.Fatalf("asks on second connect = %d", nAsk-nBefore)
	}
	_ = json.Marshal
}

// 3.2: "직접 입력" 베스천 — 베스천 비밀번호는 전용 볼트 항목에만 저장되고 대상 서버 계정을 덮어쓰지 않는다.
func TestJumpDirectKeepsCredentialsSeparate(t *testing.T) {
	szBastion := startServer(t)
	szTarget := startServer(t)
	bh, bp, _ := net.SplitHostPort(szBastion)
	th, tp, _ := net.SplitHostPort(szTarget)
	var iBP, iTP int
	_, _ = fmtSscan(bp, &iBP)
	_, _ = fmtSscan(tp, &iTP)

	szDir := t.TempDir()
	st, _ := store.Open(szDir)
	vt := vault.Open(szDir)
	if _, err := vt.Setup("master password"); err != nil {
		t.Fatal(err)
	}
	szDirect := model.JumpHostDirect
	target := model.Host{ID: "t1", Alias: "내서버", Protocol: "ssh", Host: th, Port: iTP, Username: "u", AuthType: "ask",
		JumpHostID: &szDirect, JumpDirect: &model.JumpDirect{Host: bh, Port: iBP, Username: "ops", AuthType: "password"},
		TermType: "xterm", Forwards: []model.PortForward{}}
	raw, _ := json.Marshal([]model.Host{target})
	st.Patch(map[string]json.RawMessage{"hosts": raw})

	var br *ui.Bridge
	br = ui.NewBridge(func(szCh string, args ...any) {
		if szCh != "ui:request" {
			return
		}
		req := args[0].(map[string]any)
		switch req["kind"] {
		case "hostkey":
			go br.Respond(req["reqId"].(string), "true")
		case "password":
			go br.Respond(req["reqId"].(string), `{"password":"pw","save":true}`)
		default:
			go br.Respond(req["reqId"].(string), "null")
		}
	})
	d := Deps{Store: st, Vault: vt, UI: br}
	var mu sync.Mutex
	var out strings.Builder
	sh, err := StartShell(d, target, Context{Title: "t", Cols: 80, Rows: 24,
		OnSavePassword: func(u, p string) {
			if u != "u" { // 대상 서버 콜백에는 대상 계정만 와야 한다 (베스천 "ops" 가 섞이면 안 됨)
				t.Errorf("대상 서버 저장 콜백에 베스천 계정이 전달됨: %s", u)
			}
		}}, transport.Callbacks{OnData: func(a []byte) { mu.Lock(); out.Write(a); mu.Unlock() }, OnClose: func(string) {}})
	if err != nil {
		t.Fatal(err)
	}
	defer sh.Close()
	_ = sh.Write([]byte("via"))
	for i := 0; i < 40; i++ {
		mu.Lock()
		s := out.String()
		mu.Unlock()
		if s == "VIA" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if out.String() != "VIA" {
		t.Fatalf("out = %q", out.String())
	}
	time.Sleep(100 * time.Millisecond)
	// 베스천 전용 항목이 생겼고, 대상 서버의 credentialId 는 그대로(없음)
	var names []string
	for _, m := range vt.List() {
		names = append(names, m.Name)
	}
	if len(names) == 0 || names[0] != "내서버 베스천" {
		t.Fatalf("vault items = %v", names)
	}
	got, _ := st.HostByID("t1")
	if got.CredentialID != nil {
		t.Fatalf("target credentialId must stay empty, got %v", *got.CredentialID)
	}
	if got.JumpDirect == nil || got.JumpDirect.CredentialID == nil {
		t.Fatal("bastion credential must be linked on jumpDirect")
	}
}
