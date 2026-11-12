package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// testEmitter 는 이벤트를 모아 두고 ui.request 는 채널로 넘긴다.
type testEmitter struct {
	mtxData sync.Mutex
	slData  []string
	chAsk   chan map[string]any
	chToast chan map[string]any
}

func (e *testEmitter) Emit(szEvent string, anyData ...any) {
	e.mtxData.Lock()
	e.slData = append(e.slData, szEvent)
	e.mtxData.Unlock()
	if szEvent == "ui.request" {
		e.chAsk <- anyData[0].(map[string]any)
	}
	if szEvent == "session.data" && len(anyData) == 2 {
		e.mtxData.Lock()
		e.slData = append(e.slData, "DATA:"+anyData[1].(string))
		e.mtxData.Unlock()
	}
}

func (e *testEmitter) allText() string {
	e.mtxData.Lock()
	defer e.mtxData.Unlock()
	return strings.Join(e.slData, "\n")
}

// startTestSSH 는 비밀번호 pw 를 받고, 셸 입력을 되돌려 보내는 SSH 서버를 띄운다.
func startTestSSH(t *testing.T) int {
	t.Helper()
	_, priv, errKey := ed25519.GenerateKey(rand.Reader)
	if errKey != nil {
		t.Fatal(errKey)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
		if string(pw) == "pw" {
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
							if req.Type == "shell" {
								go func() { _, _ = io.Copy(ch, ch) }()
							}
						}
					}(ch, chReqs)
				}
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// 저장된 SSH 서버로 세션을 열고, 호스트 키·비밀번호 질문에 답해 접속한 뒤 입력이 에코로 돌아오는지 본다.
// 흐름: session.open → ui.request(hostkey) → Reply → ui.request(password) → Reply → TermInput → session.data.
func TestSessionOpenAskAndEchoThroughIPC(t *testing.T) {
	port := startTestSSH(t)
	em := &testEmitter{chAsk: make(chan map[string]any, 8), chToast: make(chan map[string]any, 8)}
	pService, errNew := NewService(t.TempDir(), em)
	if errNew != nil {
		t.Fatal(errNew)
	}
	pRouter := pService.Router()

	slHosts := []map[string]any{{
		"id": "h1", "alias": "test-ssh", "protocol": "ssh", "host": "127.0.0.1", "port": port,
		"username": "root", "authType": "ask", "groupId": nil, "tags": []string{}, "encoding": "utf-8",
	}}
	raw, _ := json.Marshal(map[string]any{"hosts": slHosts})
	var mapPatch map[string]any
	_ = json.Unmarshal(raw, &mapPatch)
	if _, errPatch := pRouter.Invoke("store.patch", `[`+string(raw)+`]`); errPatch != nil {
		t.Fatal(errPatch)
	}

	szInfo, errOpen := pRouter.Invoke("session.open", `[{"hostId":"h1"}]`)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	var info struct {
		M_szId string `json:"id"`
	}
	_ = json.Unmarshal([]byte(szInfo), &info)
	if info.M_szId == "" {
		t.Fatalf("세션 id 없음: %s", szInfo)
	}

	answer := func(reqKind string, value string) {
		select {
		case req := <-em.chAsk:
			if req["kind"] != reqKind {
				t.Fatalf("질문 종류 %v, 기대 %s", req["kind"], reqKind)
			}
			pRouter.Reply(req["requestId"].(string), value)
		case <-time.After(10 * time.Second):
			t.Fatalf("질문 %s 이 오지 않음", reqKind)
		}
	}
	answer("hostkey", "true")
	answer("password", `{"password":"pw","save":false}`)

	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(em.allText(), "접속 중") || !strings.Contains(em.allText(), "\x1b[2K") {
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// 연결 상태가 connected 가 될 때까지 기다린 뒤 입력한다.
	for {
		pSession, errGet := pService.pSessions.Get(info.M_szId)
		if errGet == nil && pSession.Snapshot().M_szState == "connected" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connected 상태가 되지 않음")
		}
		time.Sleep(20 * time.Millisecond)
	}
	pService.TermInput(info.M_szId, "ping\n")
	deadline = time.Now().Add(10 * time.Second)
	for !strings.Contains(em.allText(), "ping") {
		if time.Now().After(deadline) {
			t.Fatalf("입력이 에코로 돌아오지 않음:\n%s", em.allText())
		}
		time.Sleep(20 * time.Millisecond)
	}
	_, _ = pRouter.Invoke("session.close", `["`+info.M_szId+`"]`)
}

// 알 수 없는 프로토콜 세션은 오류 안내 줄을 보여 주고 error 로 끝난다 (구현 전 프로토콜과 같은 경로).
func TestUnknownProtocolReportsError(t *testing.T) {
	em := &testEmitter{chAsk: make(chan map[string]any, 1), chToast: make(chan map[string]any, 1)}
	pService, _ := NewService(t.TempDir(), em)
	_, errOpen := pService.Router().Invoke("session.open", `[{"adhoc":{"host":"x","protocol":"ftpx"}}]`)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(em.allText(), "접속 실패") {
		if time.Now().After(deadline) {
			t.Fatalf("오류 안내가 나오지 않음:\n%s", em.allText())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
