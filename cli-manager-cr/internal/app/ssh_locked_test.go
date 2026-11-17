package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"dariterm/internal/session"
	"dariterm/internal/sshconn"
	"dariterm/internal/vault"
)

// startPwSSH 는 비밀번호를 바꿀 수 있는 시험 SSH 서버다 (셸은 에코).
func startPwSSH(t *testing.T, pPassword *atomic.Value) int {
	t.Helper()
	_, priv, errKey := ed25519.GenerateKey(rand.Reader)
	if errKey != nil {
		t.Fatal(errKey)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
		if string(pw) == pPassword.Load().(string) {
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

// asker 는 ui.request 에 자동으로 답하고 종류별 횟수를 센다.
type asker struct {
	mtxData   sync.Mutex
	mapCounts map[string]int
	szPw      string
}

func (pAsk *asker) count(szKind string) int {
	pAsk.mtxData.Lock()
	defer pAsk.mtxData.Unlock()
	return pAsk.mapCounts[szKind]
}

func (pAsk *asker) total() int {
	pAsk.mtxData.Lock()
	defer pAsk.mtxData.Unlock()
	iSum := 0
	for _, iCount := range pAsk.mapCounts {
		iSum += iCount
	}
	return iSum
}

// newLockedTestService 는 볼트를 만든(열린) 서비스와 자동 응답기를 돌려준다.
func newLockedTestService(t *testing.T) (*Service, *asker) {
	t.Helper()
	em := &testEmitter{chAsk: make(chan map[string]any, 8), chToast: make(chan map[string]any, 8)}
	pService, errNew := NewService(t.TempDir(), em)
	if errNew != nil {
		t.Fatal(errNew)
	}
	t.Cleanup(func() { _ = pService.Close() })
	if _, errSetup := pService.pVault.Setup("master-password-1"); errSetup != nil {
		t.Fatal(errSetup)
	}
	pAsk := &asker{mapCounts: map[string]int{}, szPw: "pw"}
	go func() {
		for req := range em.chAsk {
			szKind, _ := req["kind"].(string)
			pAsk.mtxData.Lock()
			pAsk.mapCounts[szKind]++
			szPw := pAsk.szPw
			pAsk.mtxData.Unlock()
			szValue := "true"
			if szKind == "password" {
				raw, _ := json.Marshal(map[string]any{"password": szPw, "save": false})
				szValue = string(raw)
			}
			pService.Router().Reply(req["requestId"].(string), szValue)
		}
	}()
	return pService, pAsk
}

func dialOnce(t *testing.T, dial session.Dialer) (session.Transport, error) {
	t.Helper()
	chDone := make(chan struct{})
	var conn session.Transport
	var errDial error
	go func() {
		defer close(chDone)
		conn, errDial = dial(func(string) {})
	}()
	select {
	case <-chDone:
	case <-time.After(15 * time.Second):
		t.Fatal("접속 대기 시간 초과")
	}
	return conn, errDial
}

// 저장된 비밀번호로 접속 → 볼트 잠금 → 재접속은 사본으로, 질문 창 없이 (안 E).
func TestLockedReconnectUsesSessionCopy(t *testing.T) {
	var pw atomic.Value
	pw.Store("pw")
	port := startPwSSH(t, &pw)
	pService, pAsk := newLockedTestService(t)
	meta, errSave := pService.pVault.Save(vault.SaveInput{M_szName: "c", M_szKind: "password", M_pszPassword: strPtrApp("pw")})
	if errSave != nil {
		t.Fatal(errSave)
	}
	record := hostRecord{M_szId: "", M_szAlias: "t", M_szProtocol: "ssh", M_szHost: "127.0.0.1", M_iPort: port, M_szUsername: "root", M_szAuthType: "password", M_szCredentialId: meta.M_szId}
	dial, clear := pService.sshDialer(record)

	conn, errDial := dialOnce(t, dial)
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.Close()
	iAsked := pAsk.total() // 호스트 키 확인 1번

	pService.pVault.Lock()
	conn, errDial = dialOnce(t, dial)
	if errDial != nil {
		t.Fatalf("잠긴 동안 사본으로 접속해야 함: %v", errDial)
	}
	_ = conn.Close()
	if pAsk.total() != iAsked {
		t.Fatalf("잠긴 동안 질문 창이 뜨면 안 됨 (%d → %d)", iAsked, pAsk.total())
	}

	// 사본으로 인증 실패(서버 비밀번호 변경) → 창 없이 잠금 해제 대기.
	pw.Store("pw2")
	_, errDial = dialOnce(t, dial)
	if !errors.Is(errDial, session.ErrWaitUnlock) {
		t.Fatalf("ErrWaitUnlock 기대: %v", errDial)
	}
	if pAsk.total() != iAsked {
		t.Fatal("사본 실패 때도 잠긴 동안에는 창을 띄우지 않음")
	}

	// 닫으면 사본이 지워져, 잠긴 동안에는 접속을 시도하지 않고 대기.
	pw.Store("pw")
	clear()
	_, errDial = dialOnce(t, dial)
	if !errors.Is(errDial, session.ErrWaitUnlock) {
		t.Fatalf("사본을 지운 뒤에는 대기 기대: %v", errDial)
	}
}

// 저장하지 않고 입력한 비밀번호도 같은 세션의 재접속에 쓴다(볼트 잠김·열림 모두) (안 E).
func TestTypedPasswordReusedInSameSession(t *testing.T) {
	var pw atomic.Value
	pw.Store("pw")
	port := startPwSSH(t, &pw)
	pService, pAsk := newLockedTestService(t)
	record := hostRecord{M_szAlias: "t", M_szProtocol: "ssh", M_szHost: "127.0.0.1", M_iPort: port, M_szUsername: "root", M_szAuthType: "ask"}
	dial, _ := pService.sshDialer(record)

	conn, errDial := dialOnce(t, dial)
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.Close()
	if pAsk.count("password") == 0 {
		t.Fatal("처음에는 비밀번호를 물어야 함")
	}
	iAsked := pAsk.total()

	conn, errDial = dialOnce(t, dial) // 볼트 열림
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.Close()
	pService.pVault.Lock()
	conn, errDial = dialOnce(t, dial) // 볼트 잠김
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.Close()
	if pAsk.total() != iAsked {
		t.Fatalf("같은 세션의 재접속에서는 묻지 않음 (%d → %d)", iAsked, pAsk.total())
	}

	// 다른 세션(새 다이얼러)은 사본이 없다: 잠긴 동안에는 창 없이 대기.
	dialOther, _ := pService.sshDialer(record)
	_, errDial = dialOnce(t, dialOther)
	if !errors.Is(errDial, session.ErrWaitUnlock) {
		t.Fatalf("새 세션은 잠긴 동안 대기 기대: %v", errDial)
	}
	if pAsk.total() != iAsked {
		t.Fatal("잠긴 동안 새 세션도 창을 띄우지 않음")
	}
}

// 볼트가 열려 있으면 볼트 값이 사본보다 우선이다 (그사이 바꾼 계정 반영).
func TestUnlockedVaultValueWins(t *testing.T) {
	var pw atomic.Value
	pw.Store("pw")
	port := startPwSSH(t, &pw)
	pService, pAsk := newLockedTestService(t)
	meta, _ := pService.pVault.Save(vault.SaveInput{M_szName: "c", M_szKind: "password", M_pszPassword: strPtrApp("pw")})
	record := hostRecord{M_szAlias: "t", M_szProtocol: "ssh", M_szHost: "127.0.0.1", M_iPort: port, M_szUsername: "root", M_szAuthType: "password", M_szCredentialId: meta.M_szId}
	dial, _ := pService.sshDialer(record)
	conn, errDial := dialOnce(t, dial)
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.Close()
	iAsked := pAsk.total()

	pw.Store("pw2")
	if _, errSave := pService.pVault.Save(vault.SaveInput{M_szId: meta.M_szId, M_szName: "c", M_szKind: "password", M_pszPassword: strPtrApp("pw2")}); errSave != nil {
		t.Fatal(errSave)
	}
	conn, errDial = dialOnce(t, dial)
	if errDial != nil {
		t.Fatalf("바뀐 볼트 값으로 접속해야 함: %v", errDial)
	}
	_ = conn.Close()
	if pAsk.total() != iAsked {
		t.Fatal("볼트 값으로 접속하면 묻지 않음")
	}
	pService.pVault.Lock()
	conn, errDial = dialOnce(t, dial) // 사본도 새 값으로 갱신됐어야 함
	if errDial != nil {
		t.Fatalf("갱신된 사본으로 접속해야 함: %v", errDial)
	}
	_ = conn.Close()
}

// 잠김 구분: 잠긴 볼트 + 저장 계정 → 잠김, 계정 없음 → 없음, 볼트 미설정 → 없음.
func TestApplyCredentialDistinguishesLocked(t *testing.T) {
	pService, _ := newLockedTestService(t)
	meta, _ := pService.pVault.Save(vault.SaveInput{M_szName: "c", M_szKind: "password", M_pszPassword: strPtrApp("pw")})
	pService.pVault.Lock()
	tgtA := newTestTarget()
	if got := pService.applyCredential(tgtA, meta.M_szId); got != credLocked {
		t.Fatalf("잠김 기대: %v", got)
	}
	if got := pService.applyCredential(newTestTarget(), ""); got != credNone {
		t.Fatalf("없음 기대: %v", got)
	}

	em := &testEmitter{chAsk: make(chan map[string]any, 1), chToast: make(chan map[string]any, 1)}
	pFresh, _ := NewService(t.TempDir(), em)
	defer pFresh.Close()
	if got := pFresh.applyCredential(newTestTarget(), "any"); got != credNone {
		t.Fatalf("볼트 미설정은 없음 기대: %v", got)
	}
	if pFresh.vaultLocked() {
		t.Fatal("볼트 미설정은 잠김이 아님 (지금처럼 묻는다)")
	}
}

func newTestTarget() *sshconn.Target {
	return &sshconn.Target{Host: "127.0.0.1", Port: 22}
}

// 잠긴 볼트에서 저장 계정 서버를 열면 창 없이 대기 → 볼트를 열면 곧바로 접속한다 (안 A, 처음 열기 포함).
func TestOpenWhileLockedResumesOnUnlock(t *testing.T) {
	var pw atomic.Value
	pw.Store("pw")
	port := startPwSSH(t, &pw)
	pService, pAsk := newLockedTestService(t)
	meta, _ := pService.pVault.Save(vault.SaveInput{M_szName: "c", M_szKind: "password", M_pszPassword: strPtrApp("pw")})
	raw, _ := json.Marshal(map[string]any{"hosts": []map[string]any{{
		"id": "h1", "alias": "t", "protocol": "ssh", "host": "127.0.0.1", "port": port,
		"username": "root", "authType": "password", "credentialId": meta.M_szId,
		"groupId": nil, "tags": []string{}, "encoding": "utf-8",
	}}})
	if _, errPatch := pService.Router().Invoke("store.patch", `[`+string(raw)+`]`); errPatch != nil {
		t.Fatal(errPatch)
	}
	pService.pVault.Lock()
	szInfo, errOpen := pService.Router().Invoke("session.open", `[{"hostId":"h1"}]`)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	var info struct {
		M_szId string `json:"id"`
	}
	_ = json.Unmarshal([]byte(szInfo), &info)
	pSession, errGet := pService.pSessions.Get(info.M_szId)
	if errGet != nil {
		t.Fatal(errGet)
	}
	waitState := func(szState string) {
		durEnd := time.Now().Add(10 * time.Second)
		for pSession.Snapshot().M_szState != szState {
			if time.Now().After(durEnd) {
				t.Fatalf("상태 %s 기대, 지금 %s", szState, pSession.Snapshot().M_szState)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	waitState(session.G_cszStateReconnecting)
	time.Sleep(100 * time.Millisecond)
	if pAsk.total() != 0 {
		t.Fatal("잠긴 동안 창을 띄우지 않음")
	}
	if !pService.pVault.Unlock("master-password-1") {
		t.Fatal("잠금 해제 실패")
	}
	waitState(session.G_cszStateConnected)
	_, _ = pService.Router().Invoke("session.close", `["`+info.M_szId+`"]`)
}
