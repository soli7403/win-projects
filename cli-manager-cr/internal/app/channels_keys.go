package app

import (
	"encoding/json"
	"os"
	"os/user"
	"strings"

	"dariterm/internal/i18n"
	"dariterm/internal/sshconn"
	"dariterm/internal/vault"
)

// SSH 키 만들기·공개 키 보기·서버 등록 채널 (spec/ipc.md §2 key.*, spec/behavior/ssh-keys.md, D162).

// keyInstaller 는 SSH 세션 전송이 가진 "이 연결로 공개 키 등록" 기능이다.
type keyInstaller interface {
	InstallPublicKey(szLine string) (bool, error)
}

// InstallPublicKey 는 이 SSH 세션의 연결로 공개 키를 서버에 등록한다.
func (pSess *sshSession) InstallPublicKey(szLine string) (bool, error) {
	return pSess.conn.InstallPublicKey(szLine)
}

// keyGenerateInput 은 key.generate 의 입력이다.
type keyGenerateInput struct {
	M_szName       string `json:"name"`
	M_szType       string `json:"type,omitempty"`
	M_iBits        int    `json:"bits,omitempty"`
	M_szPassphrase string `json:"passphrase,omitempty"`
}

// keyPublicResult 는 공개 키 한 줄과 지문이다.
type keyPublicResult struct {
	M_szPublicKey   string `json:"publicKey"`
	M_szFingerprint string `json:"fingerprint"`
}

// defaultKeyComment 는 주석을 비웠을 때 붙이는 "사용자@컴퓨터" 다 (ssh-keygen 과 같은 모양). 알 수 없으면 빈 값.
func defaultKeyComment() string {
	szUser := ""
	if pUser, errUser := user.Current(); errUser == nil {
		szUser = pUser.Username
		if iSlash := strings.LastIndexAny(szUser, `\/`); iSlash >= 0 {
			szUser = szUser[iSlash+1:] // Windows 는 "컴퓨터\사용자"
		}
	}
	szHost, _ := os.Hostname()
	if szUser == "" || szHost == "" {
		return ""
	}
	return szUser + "@" + szHost
}

func (pService *Service) registerKeys() {
	pRouter := pService.pRouter
	pRouter.Handle("key.generate", func(slArgs []json.RawMessage) (any, error) {
		var input keyGenerateInput
		if errArg := argInto(slArgs, 0, &input); errArg != nil {
			return nil, errArg
		}
		szName := strings.TrimSpace(input.M_szName)
		if szName == "" {
			return nil, i18n.NewError("be.key.nameRequired")
		}
		key, errGen := sshconn.GenerateKey(input.M_szType, input.M_iBits, defaultKeyComment(), input.M_szPassphrase)
		if errGen != nil {
			return nil, i18n.Errorf("be.key.generateFailed", errGen)
		}
		save := vault.SaveInput{M_szName: szName, M_szKind: "key", M_pszPrivateKey: &key.PrivateKey}
		if input.M_szPassphrase != "" {
			save.M_pszPassphrase = &input.M_szPassphrase
		}
		meta, errSave := pService.pVault.Save(save)
		if errSave != nil {
			return nil, errSave
		}
		return map[string]any{"cred": meta, "publicKey": key.PublicKey, "fingerprint": key.Fingerprint}, nil
	})
	pRouter.Handle("key.publicKey", func(slArgs []json.RawMessage) (any, error) {
		szCredId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return pService.publicKeyOf(szCredId)
	})
	pRouter.Handle("key.install", func(slArgs []json.RawMessage) (any, error) {
		szSessionId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		szCredId, errCred := argString(slArgs, 1)
		if errCred != nil {
			return nil, errCred
		}
		pSession, errGet := pService.pSessions.Get(szSessionId)
		if errGet != nil {
			return nil, errGet
		}
		installer, bOk := pSession.Transport().(keyInstaller)
		if !bOk {
			return nil, i18n.NewError("be.key.sshOnly")
		}
		pub, errPub := pService.publicKeyOf(szCredId)
		if errPub != nil {
			return nil, errPub
		}
		bExists, errInstall := installer.InstallPublicKey(pub.M_szPublicKey)
		if errInstall != nil {
			return nil, i18n.Errorf("be.key.installFailed", errInstall)
		}
		szStatus := "added"
		if bExists {
			szStatus = "exists"
		}
		return map[string]any{"status": szStatus, "fingerprint": pub.M_szFingerprint}, nil
	})
}

// publicKeyOf 는 보관함의 키 계정에서 공개 키 한 줄과 지문을 만든다. 주석은 만들 때와 같은 "사용자@컴퓨터" 다.
func (pService *Service) publicKeyOf(szCredId string) (keyPublicResult, error) {
	pBody, errGet := pService.pVault.Get(szCredId)
	if errGet != nil {
		return keyPublicResult{}, errGet
	}
	if pBody == nil || strings.TrimSpace(pBody.M_szPrivateKey) == "" {
		return keyPublicResult{}, i18n.NewError("be.key.notKey")
	}
	szLine, szFp, errPub := sshconn.PublicKeyOf(pBody.M_szPrivateKey, pBody.M_szPassphrase, defaultKeyComment())
	if errPub != nil {
		return keyPublicResult{}, i18n.Errorf("be.key.readFailed", errPub)
	}
	return keyPublicResult{M_szPublicKey: szLine, M_szFingerprint: szFp}, nil
}
