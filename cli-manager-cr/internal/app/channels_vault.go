package app

import (
	"encoding/json"

	"dariterm/internal/vault"
)

// registerVault 는 vault.* 채널을 등록한다 (spec/ipc.md §2 볼트·계정).
func (pService *Service) registerVault() {
	pRouter := pService.pRouter
	pRouter.Handle("vault.status", func(slArgs []json.RawMessage) (any, error) {
		return pService.pVault.Status(), nil
	})
	pRouter.Handle("vault.setup", func(slArgs []json.RawMessage) (any, error) {
		szPassword, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		szCode, errSetup := pService.pVault.Setup(szPassword)
		if errSetup != nil {
			return nil, errSetup
		}
		return map[string]any{"recoveryCode": szCode}, nil
	})
	pRouter.Handle("vault.unlock", func(slArgs []json.RawMessage) (any, error) {
		szPassword, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return pService.pVault.Unlock(szPassword), nil
	})
	pRouter.Handle("vault.unlockOs", func(slArgs []json.RawMessage) (any, error) {
		return pService.pVault.UnlockOs(), nil
	})
	pRouter.Handle("vault.recover", func(slArgs []json.RawMessage) (any, error) {
		szCode, errCode := argString(slArgs, 0)
		if errCode != nil {
			return nil, errCode
		}
		szNew, errNew := argString(slArgs, 1)
		if errNew != nil {
			return nil, errNew
		}
		bOk, errRecover := pService.pVault.Recover(szCode, szNew)
		return bOk, errRecover
	})
	pRouter.Handle("vault.lock", func(slArgs []json.RawMessage) (any, error) {
		pService.pVault.Lock()
		return nil, nil
	})
	pRouter.Handle("vault.changePassword", func(slArgs []json.RawMessage) (any, error) {
		szCurrent, errCur := argString(slArgs, 0)
		if errCur != nil {
			return nil, errCur
		}
		szNew, errNew := argString(slArgs, 1)
		if errNew != nil {
			return nil, errNew
		}
		bOk, errChange := pService.pVault.ChangePassword(szCurrent, szNew)
		return bOk, errChange
	})
	pRouter.Handle("vault.setOsUnlock", func(slArgs []json.RawMessage) (any, error) {
		bOn, errArg := argBool(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return nil, pService.pVault.SetOsUnlock(bOn)
	})
}

// registerCred 는 cred.* 채널을 등록한다.
func (pService *Service) registerCred() {
	pRouter := pService.pRouter
	pRouter.Handle("cred.list", func(slArgs []json.RawMessage) (any, error) {
		return pService.pVault.List(), nil
	})
	pRouter.Handle("cred.get", func(slArgs []json.RawMessage) (any, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		pBody, errGet := pService.pVault.Get(szId)
		if errGet != nil {
			return nil, errGet
		}
		if pBody == nil {
			// 빈 문자열이 아니라 JSON null 로 돌려준다 (spec/ipc.md cred.get).
			return json.RawMessage("null"), nil
		}
		return pBody, nil
	})
	pRouter.Handle("cred.save", func(slArgs []json.RawMessage) (any, error) {
		var input vault.SaveInput
		if errArg := argInto(slArgs, 0, &input); errArg != nil {
			return nil, errArg
		}
		return pService.pVault.Save(input)
	})
	pRouter.Handle("cred.remove", func(slArgs []json.RawMessage) (any, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return nil, pService.pVault.Remove(szId)
	})
}
