package app

import (
	"errors"
	"sync"

	"dariterm/internal/session"
	"dariterm/internal/vault"
)

// loginSecretResolver 는 세션 하나의 로그인 스크립트가 보관함 비밀값을 읽는 함수를 만든다 (session.md §5.1, D148).
// 볼트가 열려 있으면 항목의 비밀번호를 읽어 세션 메모리에 사본으로 두고, 잠겨 있으면 그 사본을 쓴다
// (SSH 계정의 세션 사본과 같은 규칙, protocols.md §2.2). 사본도 없으면 session.ErrSecretLocked.
// 돌려주는 두 번째 함수는 세션을 닫을 때 불러 사본을 지운다.
func (pService *Service) loginSecretResolver() (session.SecretResolver, func()) {
	var mtxCopy sync.Mutex
	mapCopy := map[string]string{}
	resolve := func(szId string) (string, error) {
		body, errGet := pService.pVault.Get(szId)
		if errGet == nil && body != nil && body.M_szPassword != "" {
			mtxCopy.Lock()
			mapCopy[szId] = body.M_szPassword
			mtxCopy.Unlock()
			return body.M_szPassword, nil
		}
		mtxCopy.Lock()
		szCopy, bHad := mapCopy[szId]
		mtxCopy.Unlock()
		if bHad {
			return szCopy, nil
		}
		if errors.Is(errGet, vault.ErrLocked) && pService.vaultLocked() {
			return "", session.ErrSecretLocked
		}
		return "", errGet // 항목이 없거나 비밀번호가 비어 있다 (호출한 쪽이 "찾을 수 없음"으로 알린다)
	}
	clearCopy := func() {
		mtxCopy.Lock()
		defer mtxCopy.Unlock()
		for szId := range mapCopy {
			delete(mapCopy, szId)
		}
	}
	return resolve, clearCopy
}
