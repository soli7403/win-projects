// Package sshconn 은 SSH 접속(호스트 키 확인·인증·점프 호스트·셸·포트 포워딩)을 담당한다.
package sshconn

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"termgo/internal/model"
	"termgo/internal/store"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

// Deps 는 접속에 필요한 외부 의존성.
type Deps struct {
	Store *store.Store
	Vault *vault.Vault
	UI    *ui.Bridge
}

// Context 는 접속 한 번의 표시용 정보와 콜백.
type Context struct {
	Title string
	Cols  int
	Rows  int
	// OnSavePassword: 사용자가 입력한 비밀번호를 "저장" 선택했을 때 (대상 서버 계정에만 적용된다)
	OnSavePassword func(szUser, szPassword string)
	OnInfo         func(sz string)
}

func (c Context) info(sz string) {
	if c.OnInfo != nil {
		c.OnInfo(sz)
	}
}

// Conn 은 인증된 SSH 클라이언트와, 점프 체인(닫을 때 함께 닫는다).
type Conn struct {
	Client *ssh.Client
	Chain  []*ssh.Client
}

// Close 는 클라이언트와 점프 체인을 모두 닫는다.
func (pstrC *Conn) Close() {
	if pstrC.Client != nil {
		_ = pstrC.Client.Close()
	}
	for i := len(pstrC.Chain) - 1; i >= 0; i-- {
		_ = pstrC.Chain[i].Close()
	}
}

type passwordReply struct {
	Password string `json:"password"`
	Save     bool   `json:"save"`
}

// Dial 은 host 로 접속한다 (jumpHostId 가 있으면 재귀적으로 점프 호스트를 거친다).
func Dial(d Deps, h model.Host, c Context, iDepth int) (*Conn, error) {
	if iDepth > 4 {
		return nil, errors.New("점프 호스트 체인이 너무 깁니다")
	}
	conn := &Conn{}
	var netConn net.Conn
	szAddr := net.JoinHostPort(h.Host, fmt.Sprint(h.Port))
	if h.JumpHostID != nil && *h.JumpHostID != "" {
		jh, bSave, bOk := resolveJump(d, h)
		if !bOk {
			return nil, errors.New("점프 호스트를 찾을 수 없습니다")
		}
		c.info(fmt.Sprintf("점프 호스트 %s 경유 중...", jh.Alias))
		jc := c
		jc.Title = jh.Alias
		// 베스천 단계에서 입력한 비밀번호는 대상 서버 계정을 덮어쓰지 않는다:
		// 저장된 서버를 거치는 경우 저장하지 않고, "직접 입력" 베스천은 자기 전용 볼트 항목에만 저장한다.
		jc.OnSavePassword = nil
		if bSave {
			jc.OnSavePassword = func(szUser, szPw string) { saveBastionPassword(d, h, szUser, szPw) }
		}
		j, err := Dial(d, jh, jc, iDepth+1)
		if err != nil {
			return nil, err
		}
		conn.Chain = append(append(conn.Chain, j.Chain...), j.Client)
		nc, err := j.Client.Dial("tcp", szAddr)
		if err != nil {
			conn.Close()
			return nil, err
		}
		netConn = nc
	}

	var sec vault.Secret
	if h.CredentialID != nil && *h.CredentialID != "" {
		if s, err := d.Vault.Get(*h.CredentialID); err == nil && s != nil {
			sec = *s
		}
	}
	szUser := h.Username
	if szUser == "" {
		szUser = sec.Username
	}
	if szUser == "" {
		s, bOk := d.UI.AskString(map[string]any{"kind": "username", "sessionTitle": c.Title})
		if !bOk || s == "" {
			conn.Close()
			return nil, errors.New("사용자 이름이 입력되지 않았습니다")
		}
		szUser = s
	}

	var methods []ssh.AuthMethod
	if sec.PrivateKey != "" {
		if signer := parseKey(d, c, sec); signer != nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}
	if h.AuthType == "agent" || h.AuthType == "key" {
		if m := agentAuth(); m != nil {
			methods = append(methods, m)
		}
	}
	var typed *passwordReply
	askCount := 0
	askPassword := func(szPrompt string) (string, error) {
		raw, bOk := d.UI.Ask(map[string]any{
			"kind": "password", "sessionTitle": c.Title, "prompt": szPrompt, "username": szUser,
			"allowSave": true, "failed": askCount > 0,
		})
		askCount++
		if !bOk {
			return "", errors.New("비밀번호 입력이 취소되었습니다")
		}
		var r passwordReply
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", err
		}
		typed = &r
		return r.Password, nil
	}
	if sec.Password != "" {
		methods = append(methods, ssh.Password(sec.Password))
	}
	methods = append(methods, ssh.KeyboardInteractive(func(_, szInstr string, qs []string, echos []bool) ([]string, error) {
		if len(qs) == 1 && !echos[0] && strings.Contains(strings.ToLower(qs[0]), "pass") {
			s, err := askPassword(qs[0])
			if err != nil {
				return nil, err
			}
			return []string{s}, nil
		}
		prompts := make([]map[string]any, len(qs))
		for i, q := range qs {
			prompts[i] = map[string]any{"prompt": q, "echo": echos[i]}
		}
		raw, bOk := d.UI.Ask(map[string]any{"kind": "keyboard", "sessionTitle": c.Title, "name": "", "instructions": szInstr, "prompts": prompts})
		if !bOk {
			return nil, errors.New("입력이 취소되었습니다")
		}
		var ans []string
		if err := json.Unmarshal(raw, &ans); err != nil {
			return nil, err
		}
		return ans, nil
	}))
	methods = append(methods, ssh.RetryableAuthMethod(ssh.PasswordCallback(func() (string, error) {
		return askPassword(fmt.Sprintf("%s@%s 비밀번호", szUser, h.Host))
	}), 4))

	cfg := &ssh.ClientConfig{
		User:            szUser,
		Auth:            methods,
		HostKeyCallback: hostKeyCallback(d, c, h),
		Timeout:         20 * time.Second,
	}

	var client *ssh.Client
	if netConn == nil {
		cl, err := ssh.Dial("tcp", szAddr, cfg)
		if err != nil {
			conn.Close()
			return nil, err
		}
		client = cl
	} else {
		_ = netConn.SetDeadline(time.Now().Add(20 * time.Second))
		cc, chans, reqs, err := ssh.NewClientConn(netConn, szAddr, cfg)
		if err != nil {
			_ = netConn.Close()
			conn.Close()
			return nil, err
		}
		_ = netConn.SetDeadline(time.Time{})
		client = ssh.NewClient(cc, chans, reqs)
	}
	conn.Client = client
	if typed != nil && typed.Save && c.OnSavePassword != nil {
		c.OnSavePassword(szUser, typed.Password)
	}
	if h.KeepaliveSec > 0 {
		go keepalive(client, time.Duration(h.KeepaliveSec)*time.Second)
	}
	return conn, nil
}

func keepalive(cl *ssh.Client, dur time.Duration) {
	t := time.NewTicker(dur)
	defer t.Stop()
	iFail := 0
	for range t.C {
		if _, _, err := cl.SendRequest("keepalive@openssh.com", true, nil); err != nil {
			iFail++
			if iFail >= 4 {
				_ = cl.Close()
				return
			}
		} else {
			iFail = 0
		}
	}
}

func hostKeyCallback(d Deps, c Context, h model.Host) ssh.HostKeyCallback {
	szHostPort := fmt.Sprintf("%s:%d", h.Host, h.Port)
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		szFP := ssh.FingerprintSHA256(key)
		known, bKnown := d.Store.FindKnownHost(szHostPort)
		if bKnown && known.Fingerprint == szFP {
			return nil
		}
		req := map[string]any{
			"kind": "hostkey", "sessionTitle": c.Title, "hostPort": szHostPort,
			"keyType": key.Type(), "fingerprint": szFP, "changed": bKnown,
		}
		if bKnown {
			req["oldFingerprint"] = known.Fingerprint
		}
		if !d.UI.AskBool(req) {
			return errors.New("호스트 키를 승인하지 않았습니다 (host key rejected)")
		}
		d.Store.SetKnownHost(model.KnownHost{HostPort: szHostPort, KeyType: key.Type(), Fingerprint: szFP, AddedAt: time.Now().UnixMilli()})
		return nil
	}
}

func parseKey(d Deps, c Context, sec vault.Secret) ssh.Signer {
	arr := []byte(sec.PrivateKey)
	var signer ssh.Signer
	var err error
	if sec.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(arr, []byte(sec.Passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(arr)
	}
	for i := 0; i < 3; i++ {
		var pm *ssh.PassphraseMissingError
		if err == nil || !(errors.As(err, &pm) || strings.Contains(err.Error(), "decrypt")) {
			break
		}
		s, bOk := d.UI.AskString(map[string]any{"kind": "passphrase", "sessionTitle": c.Title})
		if !bOk {
			break
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(arr, []byte(s))
	}
	if err != nil {
		c.info(fmt.Sprintf("개인 키를 읽을 수 없습니다: %v (PPK 키는 PuTTYgen 에서 OpenSSH 형식으로 내보내 주세요)", err))
		return nil
	}
	return signer
}

// resolveJump 는 점프 호스트 정의를 해석한다. "직접 입력"이면 합성 Host 를 만들고 bSave=true.
func resolveJump(d Deps, h model.Host) (jh model.Host, bSave, bOk bool) {
	if *h.JumpHostID == model.JumpHostDirect {
		jd := h.JumpDirect
		if jd == nil || jd.Host == "" {
			return model.Host{}, false, false
		}
		iPort := jd.Port
		if iPort == 0 {
			iPort = 22
		}
		return model.Host{
			ID: "jump-" + h.ID, Alias: h.Alias + " 베스천", Protocol: "ssh", Host: jd.Host, Port: iPort,
			Username: jd.Username, AuthType: jd.AuthType, CredentialID: jd.CredentialID, Tags: []string{},
			Encoding: "utf-8", TermType: "xterm-256color", KeepaliveSec: h.KeepaliveSec, Forwards: []model.PortForward{},
		}, true, true
	}
	jh, bOk = d.Store.HostByID(*h.JumpHostID)
	return jh, false, bOk
}

// saveBastionPassword 는 "<서버 별칭> 베스천" 볼트 항목에 저장하고 대상 서버의 jumpDirect.credentialId 만 갱신한다.
func saveBastionPassword(d Deps, target model.Host, szUser, szPw string) {
	if !d.Vault.IsUnlocked() || target.JumpDirect == nil || strings.HasPrefix(target.ID, "adhoc-") {
		return
	}
	in := vault.Input{Name: target.Alias + " 베스천", Kind: "password", Username: &szUser, Password: &szPw}
	if target.JumpDirect.CredentialID != nil {
		in.ID = *target.JumpDirect.CredentialID
	}
	meta, err := d.Vault.Save(in)
	if err != nil {
		return
	}
	jd := *target.JumpDirect
	jd.CredentialID = &meta.ID
	d.Store.PatchHost(target.ID, map[string]any{"jumpDirect": jd})
	d.UI.Send("store:changed")
}
