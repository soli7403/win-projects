package app

import (
	"dariterm/internal/i18n"
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"

	"dariterm/internal/session"
	"dariterm/internal/sshconn"
	"dariterm/internal/vault"
)

// g_ciSshDefaultPort 는 SSH 기본 포트다.
const g_ciSshDefaultPort = 22

// sshSession 은 SSH 셸과 그 연결·포워딩을 함께 닫는 전송이다.
type sshSession struct {
	*sshconn.Shell
	conn    *sshconn.Conn
	closers []io.Closer
}

// Close 는 셸, 포워딩 소켓, 연결을 모두 닫는다.
func (pSess *sshSession) Close() error {
	errShell := pSess.Shell.Close()
	for _, closer := range pSess.closers {
		_ = closer.Close()
	}
	_ = pSess.conn.Close()
	return errShell
}

// sshDialer 는 SSH 접속 함수다 (protocols.md §2). 접속 뒤 keepalive·포워딩·저장 처리를 한다.
// 세션마다 하나씩 만들며, 그 세션이 접속에 쓴 계정 사본(sessionCreds)을 들고 있다.
// 돌려주는 두 번째 함수는 세션을 닫을 때 불러 사본을 지운다.
//
// 앱 잠금(볼트 잠김) 동안의 규칙 (protocols.md §2.2):
//   - 볼트 대신 사본으로 채워 접속한다(질문 없이 재접속).
//   - 어떤 질문 창(비밀번호·호스트 키·키 암호·사용자 이름·키보드-인터랙티브)도 띄우지 않는다.
//     사본이 없거나 사본으로 안 되면 session.ErrWaitUnlock 을 돌려 잠금 해제를 기다린다.
func (pService *Service) sshDialer(record hostRecord) (session.Dialer, func()) {
	creds := newSessionCreds()
	dial := func(notify func(string)) (session.Transport, error) {
		bNeedUnlock := false
		tgt, errBuild := pService.buildSSHTarget(record, 0, creds, &bNeedUnlock)
		if errBuild != nil {
			return nil, errBuild
		}
		if bNeedUnlock {
			return nil, session.ErrWaitUnlock
		}
		allowSave := record.M_szId != ""
		prompt := &askPrompter{pService: pService, allowSave: allowSave}
		store := &knownHostStore{pService: pService}
		conn, errConnect := sshconn.Connect(tgt, store, prompt, notify)
		if errConnect != nil {
			if prompt.blocked.Load() {
				// 잠긴 동안 질문이 필요했다: 실패로 세지 않고 잠금 해제를 기다린다.
				return nil, errors.Join(session.ErrWaitUnlock, errConnect)
			}
			return nil, errConnect
		}
		creds.remember(tgt)
		shell, errShell := sshconn.OpenShellWith(conn, record.M_szTermType, 120, 32,
			sshconn.ShellOptions{AgentForward: record.M_bAgentForward, Notice: notify})
		if errShell != nil {
			_ = conn.Close()
			return nil, errShell
		}
		sess := &sshSession{Shell: shell, conn: conn}
		sess.closers = conn.StartForwards(forwardsOf(record), notify)
		conn.StartKeepalive(record.M_iKeepaliveSec, func() { _ = sess.Close() })
		if allowSave {
			pService.afterSshLogin(record, conn, notify)
		}
		return sess, nil
	}
	return dial, creds.clear
}

// vaultLocked 는 볼트가 만들어져 있고 잠겨 있는지 본다. 볼트를 만들지 않았으면 false (기다려도 열 볼트가 없다).
func (pService *Service) vaultLocked() bool {
	status := pService.pVault.Status()
	return status.M_bInitialized && !status.M_bUnlocked
}

// buildSSHTarget 는 저장 서버 기록으로 접속 대상을 만든다. 계정은 볼트에서 읽고, 비어 있는 필드는 세션 사본으로 채운다.
// 저장된 계정이 있는데 볼트가 잠겨 있고 사본도 없으면 *pbNeedUnlock 을 true 로 둔다 (접속하지 않고 잠금 해제를 기다린다).
func (pService *Service) buildSSHTarget(record hostRecord, depth int, creds *sessionCreds, pbNeedUnlock *bool) (*sshconn.Target, error) {
	if depth > 5 {
		return nil, i18n.NewError("be.ssh.jumpTooDeep")
	}
	port := record.M_iPort
	if port == 0 {
		port = g_ciSshDefaultPort
	}
	tgt := &sshconn.Target{
		Title:        record.M_szAlias,
		Host:         record.M_szHost,
		Port:         port,
		Username:     record.M_szUsername,
		AuthType:     record.M_szAuthType,
		AllowSave:    record.M_szId != "",
		KeepaliveSec: record.M_iKeepaliveSec,
		TermType:     record.M_szTermType,
		Depth:        depth,
	}
	if tgt.AuthType == "" {
		tgt.AuthType = "ask"
	}
	pService.fillCredential(tgt, record.M_szCredentialId, creds, pbNeedUnlock)
	tgt.Proxy = pService.buildProxy(record.M_pProxy, tgt, creds, pbNeedUnlock)
	switch {
	case record.M_szJumpHostId == "direct" && record.M_pJumpDirect != nil:
		jd := record.M_pJumpDirect
		jumpPort := jd.M_iPort
		if jumpPort == 0 {
			jumpPort = g_ciSshDefaultPort
		}
		jump := &sshconn.Target{
			Title:     i18n.T("be.jump.title", "alias", record.M_szAlias),
			Host:      jd.M_szHost,
			Port:      jumpPort,
			Username:  jd.M_szUsername,
			AuthType:  jd.M_szAuthType,
			AllowSave: false,
			Depth:     depth + 1,
		}
		if jump.AuthType == "" {
			jump.AuthType = "ask"
		}
		pService.fillCredential(jump, jd.M_szCredentialId, creds, pbNeedUnlock)
		tgt.Jump = jump
	case record.M_szJumpHostId != "":
		jumpRecord, bFound := pService.findHost(record.M_szJumpHostId)
		if !bFound {
			return nil, i18n.NewError("be.jump.notFound")
		}
		jump, errJump := pService.buildSSHTarget(jumpRecord, depth+1, creds, pbNeedUnlock)
		if errJump != nil {
			return nil, errJump
		}
		jump.AllowSave = false
		tgt.Jump = jump
	}
	return tgt, nil
}

// credResult 는 볼트에서 계정을 읽은 결과다.
type credResult int

const (
	credNone   credResult = iota // 연결된 계정이 없거나, 항목이 없거나, 볼트를 만들지 않았다
	credRead                     // 읽어서 채웠다
	credLocked                   // 계정이 연결되어 있지만 볼트가 잠겨 읽을 수 없다
)

// applyCredential 은 볼트 항목의 비밀을 대상에 채우고 결과를 돌려준다. "잠김"과 "없음"을 구분한다.
func (pService *Service) applyCredential(tgt *sshconn.Target, szCredentialId string) credResult {
	if szCredentialId == "" {
		return credNone
	}
	body, errGet := pService.pVault.Get(szCredentialId)
	if errors.Is(errGet, vault.ErrLocked) {
		if pService.vaultLocked() {
			return credLocked
		}
		return credNone // 볼트를 만들지 않음 (Get 은 키가 없으면 먼저 ErrLocked 를 돌려준다)
	}
	if errGet != nil || body == nil {
		return credNone
	}
	if tgt.Username == "" {
		tgt.Username = body.M_szUsername
	}
	tgt.Password = body.M_szPassword
	tgt.PrivateKey = []byte(body.M_szPrivateKey)
	tgt.Passphrase = body.M_szPassphrase
	return credRead
}

// fillCredential 은 볼트 값을 먼저 채우고, 비어 있는 필드는 세션 사본으로 채운다.
// 볼트가 잠겨 저장된 계정을 읽지 못했는데 사본도 없으면 *pbNeedUnlock 을 true 로 둔다.
func (pService *Service) fillCredential(tgt *sshconn.Target, szCredentialId string, creds *sessionCreds, pbNeedUnlock *bool) {
	result := pService.applyCredential(tgt, szCredentialId)
	bHadCopy := creds != nil && creds.fill(tgt)
	if result == credLocked && !bHadCopy && pbNeedUnlock != nil {
		*pbNeedUnlock = true
	}
}

// g_mProxyDefaultPort 는 프록시 종류별 기본 포트다 (포트를 0 으로 저장했을 때).
var g_mProxyDefaultPort = map[string]int{sshconn.G_cszProxyHttp: 8080, sshconn.G_cszProxySocks5: 1080}

// buildProxy 는 서버의 프록시 설정을 연결 쪽 형식으로 바꾼다. 설정이 없거나 종류·주소가 비면 nil 이다.
// 비밀번호는 볼트의 연결된 계정에서 읽고, 볼트가 잠겨 있으면 세션 사본으로 채운다.
// 사본도 없으면 *pbNeedUnlock 을 true 로 둔다 (protocols.md §2.7, D110).
func (pService *Service) buildProxy(pRecord *proxyRecord, tgt *sshconn.Target, creds *sessionCreds, pbNeedUnlock *bool) *sshconn.Proxy {
	if pRecord == nil || pRecord.M_szHost == "" {
		return nil
	}
	iDefault, bKnown := g_mProxyDefaultPort[pRecord.M_szType]
	if !bKnown {
		return nil
	}
	pProxy := &sshconn.Proxy{Type: pRecord.M_szType, Host: pRecord.M_szHost, Port: pRecord.M_iPort, Username: pRecord.M_szUsername}
	if pProxy.Port == 0 {
		pProxy.Port = iDefault
	}
	if pRecord.M_szCredentialId == "" {
		return pProxy
	}
	body, errGet := pService.pVault.Get(pRecord.M_szCredentialId)
	switch {
	case errGet == nil && body != nil:
		pProxy.Password = body.M_szPassword
		if pProxy.Username == "" {
			pProxy.Username = body.M_szUsername
		}
	case errors.Is(errGet, vault.ErrLocked) && pService.vaultLocked():
		szPassword, bHadCopy := "", false
		if creds != nil {
			szPassword, bHadCopy = creds.proxyPassword(tgt)
		}
		if bHadCopy {
			pProxy.Password = szPassword
		} else if pbNeedUnlock != nil {
			*pbNeedUnlock = true
		}
	}
	return pProxy
}

// forwardsOf 는 서버의 포워딩 설정을 연결 쪽 형식으로 바꾼다.
func forwardsOf(record hostRecord) []sshconn.Forward {
	out := make([]sshconn.Forward, 0, len(record.M_slForwards))
	for _, fw := range record.M_slForwards {
		out = append(out, sshconn.Forward{
			Type:     fw.M_szType,
			BindHost: fw.M_szBindHost,
			BindPort: fw.M_iBindPort,
			DestHost: fw.M_szDestHost,
			DestPort: fw.M_iDestPort,
		})
	}
	return out
}

// afterSshLogin 은 인증 성공 뒤 처리다: 입력한 비밀번호 저장, 서버 접속 시각 갱신 (protocols.md §2.2, session.md §5).
func (pService *Service) afterSshLogin(record hostRecord, conn *sshconn.Conn, notify func(string)) {
	if conn.Asked.Asked && conn.Asked.Save && pService.pVault.Status().M_bUnlocked {
		if pService.saveCredentialForHost(record, conn) {
			notify(i18n.T("be.ssh.pwSaved"))
		}
	}
	pService.patchHost(record.M_szId, map[string]any{"lastUsedAt": time.Now().UnixMilli()})
}

// saveCredentialForHost 는 서버 별칭 이름의 비밀번호 항목을 만들거나 갱신하고 서버에 연결한다.
func (pService *Service) saveCredentialForHost(record hostRecord, conn *sshconn.Conn) bool {
	szPassword := conn.Asked.Password
	meta, errSave := pService.pVault.Save(vault.SaveInput{
		M_szId:        record.M_szCredentialId,
		M_szName:      record.M_szAlias,
		M_szKind:      "password",
		M_pszUsername: strPtrApp(conn.Username),
		M_pszPassword: strPtrApp(szPassword),
	})
	if errSave != nil {
		return false
	}
	fields := map[string]any{"credentialId": meta.M_szId}
	if record.M_szUsername == "" {
		fields["username"] = conn.Username
	}
	if record.M_szAuthType == "ask" || record.M_szAuthType == "" {
		fields["authType"] = "password"
	}
	pService.patchHost(record.M_szId, fields)
	return true
}

func strPtrApp(szValue string) *string {
	return &szValue
}

// patchHost 는 저장 데이터의 서버 항목 필드를 바꾸고 store.changed 를 보낸다.
func (pService *Service) patchHost(szHostId string, mapFields map[string]any) {
	rawHosts, bOk := pService.pStore.Get("hosts")
	if !bOk {
		return
	}
	var slHosts []map[string]any
	if errUnmarshal := json.Unmarshal(rawHosts, &slHosts); errUnmarshal != nil {
		return
	}
	for _, mapHost := range slHosts {
		if mapHost["id"] == szHostId {
			for szKey, anyValue := range mapFields {
				mapHost[szKey] = anyValue
			}
		}
	}
	if errPatch := pService.pStore.Patch(map[string]any{"hosts": slHosts}); errPatch != nil {
		return
	}
	pService.pRouter.Emit("store.changed")
}

// knownHostStore 는 store 의 knownHosts 를 sshconn.HostKeyStore 로 보인다.
type knownHostStore struct {
	pService *Service
}

type knownHostRecord struct {
	M_szHostPort    string `json:"hostPort"`
	M_szKeyType     string `json:"keyType"`
	M_szFingerprint string `json:"fingerprint"`
	M_i64AddedAt    int64  `json:"addedAt"`
}

// Lookup 은 host:port 의 저장된 지문을 찾는다.
func (pStore *knownHostStore) Lookup(szHostPort string) (string, bool) {
	raw, bOk := pStore.pService.pStore.Get("knownHosts")
	if !bOk {
		return "", false
	}
	var slKnown []knownHostRecord
	if errUnmarshal := json.Unmarshal(raw, &slKnown); errUnmarshal != nil {
		return "", false
	}
	for _, record := range slKnown {
		if record.M_szHostPort == szHostPort {
			return record.M_szFingerprint, true
		}
	}
	return "", false
}

// Save 는 신뢰한 키를 추가하거나 갱신하고 store.changed 를 보낸다.
func (pStore *knownHostStore) Save(szHostPort string, szKeyType string, szFingerprint string) {
	raw, _ := pStore.pService.pStore.Get("knownHosts")
	var slKnown []knownHostRecord
	_ = json.Unmarshal(raw, &slKnown)
	next := make([]knownHostRecord, 0, len(slKnown)+1)
	for _, record := range slKnown {
		if record.M_szHostPort != szHostPort {
			next = append(next, record)
		}
	}
	next = append(next, knownHostRecord{M_szHostPort: szHostPort, M_szKeyType: szKeyType, M_szFingerprint: szFingerprint, M_i64AddedAt: time.Now().UnixMilli()})
	if errPatch := pStore.pService.pStore.Patch(map[string]any{"knownHosts": next}); errPatch == nil {
		pStore.pService.pRouter.Emit("store.changed")
	}
}

// askPrompter 는 sshconn.Prompter 를 ui.request 질문으로 구현한다 (spec/ipc.md §3).
// 볼트가 잠겨 있으면 창을 띄우지 않고 취소로 답하며 blocked 를 켠다 → 다이얼러가 잠금 해제 대기로 바꾼다.
// 잠금 화면 뒤의 창에 자동으로 답하지 않기 위해서다 (protocols.md §2.2).
type askPrompter struct {
	pService  *Service
	allowSave bool
	blocked   atomic.Bool
}

func (pPrompt *askPrompter) ask(szKind string, mapFields map[string]any) (json.RawMessage, bool) {
	if pPrompt.pService.vaultLocked() {
		pPrompt.blocked.Store(true)
		return nil, false
	}
	return pPrompt.pService.pRouter.Asker().Ask(szKind, mapFields)
}

// HostKey 는 호스트 키 확인 질문이다.
func (pPrompt *askPrompter) HostKey(szTitle string, szHostPort string, szKeyType string, szFingerprint string, bChanged bool, szOld string) bool {
	mapFields := map[string]any{
		"sessionTitle": szTitle,
		"hostPort":     szHostPort,
		"keyType":      szKeyType,
		"fingerprint":  szFingerprint,
		"changed":      bChanged,
	}
	if bChanged {
		mapFields["oldFingerprint"] = szOld
	}
	raw, bOk := pPrompt.ask("hostkey", mapFields)
	if !bOk {
		return false
	}
	var bTrust bool
	return json.Unmarshal(raw, &bTrust) == nil && bTrust
}

// Password 는 비밀번호 요청이다. 응답은 {password, save}.
func (pPrompt *askPrompter) Password(szTitle string, szPrompt string, szUsername string, bFailed bool) (string, bool, bool) {
	mapFields := map[string]any{
		"sessionTitle": szTitle,
		"prompt":       szPrompt,
		"username":     szUsername,
		"allowSave":    pPrompt.allowSave,
	}
	if bFailed {
		mapFields["failed"] = true
	}
	raw, bOk := pPrompt.ask("password", mapFields)
	if !bOk {
		return "", false, false
	}
	var answer struct {
		Password string `json:"password"`
		Save     bool   `json:"save"`
	}
	if json.Unmarshal(raw, &answer) != nil {
		return "", false, false
	}
	return answer.Password, answer.Save, true
}

// Passphrase 는 개인 키 암호 요청이다.
func (pPrompt *askPrompter) Passphrase(szTitle string) (string, bool) {
	raw, bOk := pPrompt.ask("passphrase", map[string]any{"sessionTitle": szTitle})
	if !bOk {
		return "", false
	}
	var szOut string
	return szOut, json.Unmarshal(raw, &szOut) == nil
}

// Username 은 사용자 이름 요청이다.
func (pPrompt *askPrompter) Username(szTitle string) (string, bool) {
	raw, bOk := pPrompt.ask("username", map[string]any{"sessionTitle": szTitle})
	if !bOk {
		return "", false
	}
	var szOut string
	return szOut, json.Unmarshal(raw, &szOut) == nil
}

// Keyboard 는 키보드-인터랙티브 추가 인증 요청이다.
func (pPrompt *askPrompter) Keyboard(szTitle string, szName string, szInstructions string, slPrompts []sshconn.KeyboardPrompt) ([]string, bool) {
	raw, bOk := pPrompt.ask("keyboard", map[string]any{
		"sessionTitle": szTitle,
		"name":         szName,
		"instructions": szInstructions,
		"prompts":      slPrompts,
	})
	if !bOk {
		return nil, false
	}
	var slAnswers []string
	if json.Unmarshal(raw, &slAnswers) != nil {
		return nil, false
	}
	return slAnswers, true
}

// SftpClient 는 이 SSH 세션의 sftp 클라이언트다 (SFTP 기능이 쓴다).
func (pSess *sshSession) SftpClient() (*sftp.Client, error) {
	return pSess.conn.SftpClient()
}
