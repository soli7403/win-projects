package vault

import (
	"encoding/base64"
)

// wrapKeyWithPassword 는 새 salt 로 마스터 비밀번호 키를 만들어 볼트 키를 감싼다.
func wrapKeyWithPassword(bKey []byte, szPassword string) (kdfParams, cipherBlob, error) {
	paramsKdf, _, errParams := newKdfParams()
	if errParams != nil {
		return kdfParams{}, cipherBlob{}, errParams
	}
	bDerived, errDerive := deriveKey(szPassword, paramsKdf)
	if errDerive != nil {
		return kdfParams{}, cipherBlob{}, errDerive
	}
	blobWrapped, errSeal := sealBytes(bDerived, bKey, G_cszVaultKeyAad)
	return paramsKdf, blobWrapped, errSeal
}

// Setup 은 새 볼트를 만든다. 볼트는 열린 상태가 되며 복구 코드를 돌려준다 (data/vault.md §6).
// 잠금 변경 알림은 보내지 않는다(복구 코드 화면 유지).
func (pv *Vault) Setup(szPassword string) (string, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile != nil {
		return "", ErrAlreadySetup
	}
	if errValid := validPassword(szPassword); errValid != nil {
		return "", errValid
	}
	bKey, errKey := newRandomKey()
	if errKey != nil {
		return "", errKey
	}
	paramsKdf, blobWrapped, errWrap := wrapKeyWithPassword(bKey, szPassword)
	if errWrap != nil {
		return "", errWrap
	}
	szCode, errCode := newRecoveryCode()
	if errCode != nil {
		return "", errCode
	}
	paramsRecovery, _, errRecParams := newKdfParams()
	if errRecParams != nil {
		return "", errRecParams
	}
	bRecKey, errRecDerive := deriveKey(normalizeRecoveryCode(szCode), paramsRecovery)
	if errRecDerive != nil {
		return "", errRecDerive
	}
	blobRecovery, errRecSeal := sealBytes(bRecKey, bKey, G_cszVaultKeyAad)
	if errRecSeal != nil {
		return "", errRecSeal
	}
	pv.pfile = &vaultFile{
		M_iVersion:        1,
		M_kdf:             paramsKdf,
		M_wrapped:         blobWrapped,
		M_recoveryKdf:     &paramsRecovery,
		M_recoveryWrapped: &blobRecovery,
		M_slItems:         []itemRecord{},
	}
	pv.bKey = bKey
	if errSave := pv.saveLocked(); errSave != nil {
		return "", errSave
	}
	return szCode, nil
}

// Unlock 은 마스터 비밀번호로 볼트를 연다. 틀리면 false (상태 변화 없음).
func (pv *Vault) Unlock(szPassword string) bool {
	pv.mtxData.Lock()
	if pv.pfile == nil {
		pv.mtxData.Unlock()
		return false
	}
	bDerived, errDerive := deriveKey(szPassword, pv.pfile.M_kdf)
	if errDerive != nil {
		pv.mtxData.Unlock()
		return false
	}
	bKey, errOpen := openBytes(bDerived, pv.pfile.M_wrapped, G_cszVaultKeyAad)
	if errOpen != nil {
		pv.mtxData.Unlock()
		return false
	}
	bWasOpen := pv.bKey != nil
	pv.bKey = bKey
	pv.mtxData.Unlock()
	if !bWasOpen {
		pv.notifyLock(true)
	}
	return true
}

// UnlockOs 는 Windows 계정(DPAPI)으로 볼트를 연다 (data/vault.md §7).
// 항목이 있으면 첫 항목을 복호화해 키를 검증한다.
func (pv *Vault) UnlockOs() bool {
	pv.mtxData.Lock()
	if pv.pfile == nil || pv.pfile.M_szOsWrappedWin == "" {
		pv.mtxData.Unlock()
		return false
	}
	bWrapped, errB64 := decodeB64(pv.pfile.M_szOsWrappedWin)
	if errB64 != nil {
		pv.mtxData.Unlock()
		return false
	}
	bKey, errUnprotect := dpapiUnprotect(bWrapped)
	if errUnprotect != nil || len(bKey) != g_ciKeyLen {
		pv.mtxData.Unlock()
		return false
	}
	if len(pv.pfile.M_slItems) > 0 {
		if _, errOpen := openItem(bKey, pv.pfile.M_slItems[0]); errOpen != nil {
			pv.mtxData.Unlock()
			return false
		}
	}
	bWasOpen := pv.bKey != nil
	pv.bKey = bKey
	pv.mtxData.Unlock()
	if !bWasOpen {
		pv.notifyLock(true)
	}
	return true
}

// Recover 는 복구 코드로 볼트 키를 풀고 새 비밀번호로 다시 감싼다 (data/vault.md §6).
// 복구 정보 자체는 바꾸지 않는다. 코드가 틀리면 false.
func (pv *Vault) Recover(szCode string, szNewPassword string) (bool, error) {
	if errValid := validPassword(szNewPassword); errValid != nil {
		return false, errValid
	}
	pv.mtxData.Lock()
	if pv.pfile == nil || pv.pfile.M_recoveryKdf == nil || pv.pfile.M_recoveryWrapped == nil {
		pv.mtxData.Unlock()
		return false, nil
	}
	bRecKey, errDerive := deriveKey(normalizeRecoveryCode(szCode), *pv.pfile.M_recoveryKdf)
	if errDerive != nil {
		pv.mtxData.Unlock()
		return false, errDerive
	}
	bKey, errOpen := openBytes(bRecKey, *pv.pfile.M_recoveryWrapped, G_cszVaultKeyAad)
	if errOpen != nil {
		pv.mtxData.Unlock()
		return false, nil
	}
	paramsKdf, blobWrapped, errWrap := wrapKeyWithPassword(bKey, szNewPassword)
	if errWrap != nil {
		pv.mtxData.Unlock()
		return false, errWrap
	}
	pv.pfile.M_kdf = paramsKdf
	pv.pfile.M_wrapped = blobWrapped
	if errSave := pv.saveLocked(); errSave != nil {
		pv.mtxData.Unlock()
		return false, errSave
	}
	bWasOpen := pv.bKey != nil
	pv.bKey = bKey
	pv.mtxData.Unlock()
	if !bWasOpen {
		pv.notifyLock(true)
	}
	return true, nil
}

// Lock 은 메모리의 볼트 키를 0 으로 덮고 비운다. 이미 잠겨 있어도 항상 "잠김" 알림을 보낸다 (data/vault.md §6).
func (pv *Vault) Lock() {
	pv.mtxData.Lock()
	wipe(pv.bKey)
	pv.bKey = nil
	pv.mtxData.Unlock()
	pv.notifyLock(false)
}

// ChangePassword 는 이전 비밀번호로 볼트 키를 풀고 새 비밀번호의 새 kdf 로 감싼다.
// 항목은 다시 암호화하지 않으며 열림 상태도 바꾸지 않는다. 이전 비밀번호가 틀리면 false.
func (pv *Vault) ChangePassword(szCurrent string, szNew string) (bool, error) {
	if errValid := validPassword(szNew); errValid != nil {
		return false, errValid
	}
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return false, ErrNoVault
	}
	bDerived, errDerive := deriveKey(szCurrent, pv.pfile.M_kdf)
	if errDerive != nil {
		return false, errDerive
	}
	bKey, errOpen := openBytes(bDerived, pv.pfile.M_wrapped, G_cszVaultKeyAad)
	if errOpen != nil {
		return false, nil
	}
	paramsKdf, blobWrapped, errWrap := wrapKeyWithPassword(bKey, szNew)
	if errWrap != nil {
		return false, errWrap
	}
	pv.pfile.M_kdf = paramsKdf
	pv.pfile.M_wrapped = blobWrapped
	if errSave := pv.saveLocked(); errSave != nil {
		return false, errSave
	}
	return true, nil
}

// SetOsUnlock 은 OS 잠금 해제를 켜거나 끈다. 켤 때는 볼트가 열려 있어야 한다.
func (pv *Vault) SetOsUnlock(bOn bool) error {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return ErrNoVault
	}
	if !bOn {
		pv.pfile.M_szOsWrappedWin = ""
		return pv.saveLocked()
	}
	if pv.bKey == nil {
		return ErrLocked
	}
	bWrapped, errProtect := dpapiProtect(pv.bKey)
	if errProtect != nil {
		return errProtect
	}
	pv.pfile.M_szOsWrappedWin = base64.StdEncoding.EncodeToString(bWrapped)
	return pv.saveLocked()
}

// wipe 는 키 바이트를 0 으로 덮는다.
func wipe(bData []byte) {
	for iIndex := range bData {
		bData[iIndex] = 0
	}
}
