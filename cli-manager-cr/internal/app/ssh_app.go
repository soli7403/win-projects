package app

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/pkg/sftp"

	"termgo/internal/session"
	"termgo/internal/sshconn"
	"termgo/internal/vault"
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
func (pService *Service) sshDialer(record hostRecord) session.Dialer {
	return func(notify func(string)) (session.Transport, error) {
		tgt, errBuild := pService.buildSSHTarget(record, 0)
		if errBuild != nil {
			return nil, errBuild
		}
		allowSave := record.M_szId != ""
		prompt := &askPrompter{pService: pService, allowSave: allowSave}
		store := &knownHostStore{pService: pService}
		conn, errConnect := sshconn.Connect(tgt, store, prompt, notify)
		if errConnect != nil {
			return nil, errConnect
		}
		shell, errShell := sshconn.OpenShell(conn, record.M_szTermType, 120, 32)
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
}

// buildSSHTarget 는 저장 서버 기록으로 접속 대상을 만든다. 계정은 볼트에서 읽는다 (볼트가 잠겨 있으면 비밀 없이 진행).
func (pService *Service) buildSSHTarget(record hostRecord, depth int) (*sshconn.Target, error) {
	if depth > 5 {
		return nil, errors.New("경유 서버 체인이 너무 깁니다")
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
	pService.applyCredential(tgt, record.M_szCredentialId)
	switch {
	case record.M_szJumpHostId == "direct" && record.M_pJumpDirect != nil:
		jd := record.M_pJumpDirect
		jumpPort := jd.M_iPort
		if jumpPort == 0 {
			jumpPort = g_ciSshDefaultPort
		}
		jump := &sshconn.Target{
			Title:     record.M_szAlias + " 경유 서버",
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
		pService.applyCredential(jump, jd.M_szCredentialId)
		tgt.Jump = jump
	case record.M_szJumpHostId != "":
		jumpRecord, bFound := pService.findHost(record.M_szJumpHostId)
		if !bFound {
			return nil, errors.New("경유 서버를 찾을 수 없습니다")
		}
		jump, errJump := pService.buildSSHTarget(jumpRecord, depth+1)
		if errJump != nil {
			return nil, errJump
		}
		jump.AllowSave = false
		tgt.Jump = jump
	}
	return tgt, nil
}

// applyCredential 은 볼트 항목의 비밀을 대상에 채운다. 잠겨 있거나 항목이 없으면 아무것도 하지 않는다.
func (pService *Service) applyCredential(tgt *sshconn.Target, szCredentialId string) {
	if szCredentialId == "" {
		return
	}
	body, errGet := pService.pVault.Get(szCredentialId)
	if errGet != nil || body == nil {
		return
	}
	if tgt.Username == "" {
		tgt.Username = body.M_szUsername
	}
	tgt.Password = body.M_szPassword
	tgt.PrivateKey = []byte(body.M_szPrivateKey)
	tgt.Passphrase = body.M_szPassphrase
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
			notify("비밀번호를 보관함에 저장했습니다")
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
type askPrompter struct {
	pService  *Service
	allowSave bool
}

func (pPrompt *askPrompter) ask(szKind string, mapFields map[string]any) (json.RawMessage, bool) {
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
