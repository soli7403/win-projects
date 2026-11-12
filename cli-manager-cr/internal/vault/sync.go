package vault

import (
	"encoding/json"
)

// Blob 은 암호문 객체 {iv, tag, ct} 다 (data/vault.md §3). 동기화 파일의 payload 가 쓴다.
type Blob = cipherBlob

// ItemRecord 는 볼트 항목 레코드 전체(평문 메타 + 암호문)다. 동기화 파일의 creds 가 쓴다 (data/vault.md §4).
type ItemRecord = itemRecord

// G_cszSyncPayloadAad 는 동기화 payload 암호화의 AAD 다 (data/sync-backup.md §5.1).
const G_cszSyncPayloadAad = "termgo-sync"

// EncryptData 는 열린 볼트 키로 임의 데이터를 암호화한다 (data/vault.md §8). szAad 는 호출자가 정한다.
func (pv *Vault) EncryptData(szAad string, bPlain []byte) (Blob, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.bKey == nil {
		return Blob{}, ErrLocked
	}
	return sealBytes(pv.bKey, bPlain, szAad)
}

// DecryptData 는 열린 볼트 키로 암호문을 복호화한다. 인증이 맞지 않으면 오류.
func (pv *Vault) DecryptData(szAad string, blob Blob) ([]byte, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.bKey == nil {
		return nil, ErrLocked
	}
	return openBytes(pv.bKey, blob, szAad)
}

// HeaderJSON 은 파일 구조에서 items, osWrapped, osWrappedWin 을 비운 복사본(JSON)을 돌려준다 (data/vault.md §8 헤더).
func (pv *Vault) HeaderJSON() ([]byte, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return nil, ErrNoVault
	}
	return json.Marshal(headerOf(*pv.pfile))
}

// headerOf 는 파일 구조에서 항목과 OS 보호 값을 뺀 복사본을 만든다.
func headerOf(file vaultFile) vaultFile {
	file.M_slItems = []itemRecord{}
	file.M_szOsWrapped = ""
	file.M_szOsWrappedWin = ""
	return file
}

// Items 는 항목 레코드 목록의 복사본을 돌려준다 (암호문 그대로, 잠겨 있어도 가능).
func (pv *Vault) Items() ([]ItemRecord, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return nil, ErrNoVault
	}
	slOut := make([]ItemRecord, len(pv.pfile.M_slItems))
	copy(slOut, pv.pfile.M_slItems)
	return slOut, nil
}

// ReplaceItems 는 항목 목록 전체를 바꾸고 저장한다 (data/vault.md §8 항목 목록 교체). 항목 변경 알림은 보내지 않는다.
func (pv *Vault) ReplaceItems(slItems []ItemRecord) error {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return ErrNoVault
	}
	slCopy := make([]itemRecord, len(slItems))
	copy(slCopy, slItems)
	pv.pfile.M_slItems = slCopy
	return pv.saveLocked()
}

// UnwrapHeaderKey 는 헤더(JSON)의 wrapped 를 마스터 비밀번호로 풀어 볼트 키를 돌려준다 (data/vault.md §8).
// 비밀번호가 맞지 않거나 헤더가 깨졌으면 bOk=false 다. 볼트 상태는 바꾸지 않는다.
func UnwrapHeaderKey(headerRaw []byte, szPassword string) ([]byte, bool) {
	var file vaultFile
	if errJson := json.Unmarshal(headerRaw, &file); errJson != nil {
		return nil, false
	}
	bDerived, errDerive := deriveKey(szPassword, file.M_kdf)
	if errDerive != nil {
		return nil, false
	}
	bKey, errOpen := openBytes(bDerived, file.M_wrapped, G_cszVaultKeyAad)
	if errOpen != nil {
		return nil, false
	}
	return bKey, true
}

// DecryptWithKey 는 주어진 키로 암호문을 복호화한다 (볼트가 잠겨 있어도 동기화 해석에 쓴다).
func DecryptWithKey(bKey []byte, blob Blob, szAad string) ([]byte, error) {
	return openBytes(bKey, blob, szAad)
}

// AdoptRemote 는 저장소의 헤더와 항목으로 이 PC 볼트를 교체하고 연다 (data/vault.md §8 원격 볼트 받기).
// 비밀번호가 맞지 않으면 아무것도 바꾸지 않고 false 다. OS 보호 값은 비운다.
func (pv *Vault) AdoptRemote(headerRaw []byte, slItems []ItemRecord, szPassword string) (bool, error) {
	bKey, bOk := UnwrapHeaderKey(headerRaw, szPassword)
	if !bOk {
		return false, nil
	}
	var file vaultFile
	if errJson := json.Unmarshal(headerRaw, &file); errJson != nil {
		return false, errJson
	}
	file = headerOf(file)
	file.M_slItems = make([]itemRecord, len(slItems))
	copy(file.M_slItems, slItems)

	pv.mtxData.Lock()
	bWasOpen := pv.bKey != nil
	pv.pfile = &file
	wipe(pv.bKey)
	pv.bKey = bKey
	errSave := pv.saveLocked()
	pv.mtxData.Unlock()
	if errSave != nil {
		return false, errSave
	}
	if !bWasOpen {
		pv.notifyLock(true)
	}
	return true, nil
}
