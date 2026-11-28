package session

import (
	"errors"
	"sync"
	"testing"
	"time"

	"dariterm/internal/i18n"
)

// 로그인 스크립트의 보관함 값 (session.md §5.1, D148).

// 보관함 값을 고른 단계는 send 가 아니라 해석기가 돌려준 비밀번호를 보낸다. 로그인 비밀번호와 따로다.
func TestSessionLoginScriptSendsVaultSecret(t *testing.T) {
	pFake := &secretTransport{fakeTransport: newFakeTransport(), szSecret: "login-pw"}
	var mtxAsked sync.Mutex
	var slAsked []string
	pManager, _, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript: []LoginStep{{Expect: "$ ", Send: "su -"}, {Expect: "password", Send: "", SecretId: "cred-root"}},
		ResolveSecret: func(szId string) (string, error) {
			mtxAsked.Lock()
			defer mtxAsked.Unlock()
			slAsked = append(slAsked, szId)
			return "root-pw", nil
		},
		StartupInput: "id\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)

	pFake.chIn <- []byte("user@host:~$ ")
	waitWrites(t, pFake.fakeTransport, 1)
	pFake.chIn <- []byte("Password: ")
	slWrites := waitWrites(t, pFake.fakeTransport, 3)
	if slWrites[0] != "su -\r" || slWrites[1] != "root-pw\r" || slWrites[2] != "id\r" {
		t.Fatalf("입력 = %q", slWrites)
	}
	mtxAsked.Lock()
	defer mtxAsked.Unlock()
	if len(slAsked) != 1 || slAsked[0] != "cred-root" {
		t.Fatalf("해석기 호출 = %v", slAsked)
	}
}

// 보관함이 잠겨 있고 사본도 없으면 아무것도 보내지 않고 잠김 안내를 낸다. 접속 후 입력도 보내지 않는다.
func TestSessionLoginScriptSecretLocked(t *testing.T) {
	i18n.SetLang("ko")
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:   []LoginStep{{Expect: "password", SecretId: "cred-root"}},
		ResolveSecret: func(string) (string, error) { return "", ErrSecretLocked },
		StartupInput:  "never\r",
		StartupDelay:  time.Millisecond,
	})
	defer pManager.Close(szId)
	pFake.chIn <- []byte("Password: ")
	waitOutput(t, pRec, "보관함이 잠겨")
	time.Sleep(30 * time.Millisecond)
	if slWrites := pFake.writes(); len(slWrites) != 0 {
		t.Fatalf("보내면 안 된다: %v", slWrites)
	}
}

// 해석기가 없거나 항목을 찾지 못하면 "찾을 수 없음" 안내를 낸다.
func TestSessionLoginScriptSecretMissing(t *testing.T) {
	i18n.SetLang("ko")
	for _, resolve := range []SecretResolver{nil, func(string) (string, error) { return "", errors.New("gone") }} {
		pFake := newFakeTransport()
		pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
			LoginScript:   []LoginStep{{Expect: "password", SecretId: "cred-x"}},
			ResolveSecret: resolve,
		})
		pFake.chIn <- []byte("Password: ")
		waitOutput(t, pRec, "찾을 수 없거나")
		if slWrites := pFake.writes(); len(slWrites) != 0 {
			t.Fatalf("보내면 안 된다: %v", slWrites)
		}
		pManager.Close(szId)
	}
}

// 보관함 값만 있는 단계(기다릴 글자·보낼 글자 없음)도 버리지 않는다.
func TestTrimLoginStepsKeepsSecretOnly(t *testing.T) {
	slOut := TrimLoginSteps([]LoginStep{{}, {SecretId: "cred-1"}, {Expect: "", Send: ""}})
	if len(slOut) != 1 || slOut[0].SecretId != "cred-1" {
		t.Fatalf("남은 단계 = %+v", slOut)
	}
}
