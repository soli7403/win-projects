package vault

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// newUUID 는 UUID v4 문자열을 만든다.
func newUUID() (string, error) {
	bUuid := make([]byte, 16)
	if _, errRand := rand.Read(bUuid); errRand != nil {
		return "", errRand
	}
	bUuid[6] = (bUuid[6] & 0x0f) | 0x40
	bUuid[8] = (bUuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bUuid[0:4], bUuid[4:6], bUuid[6:8], bUuid[8:10], bUuid[10:16]), nil
}

// sealItem 은 비밀 객체를 볼트 키로 암호화한다. AAD 는 항목 id 다.
func sealItem(bKey []byte, szId string, body SecretBody) (cipherBlob, error) {
	bPlain, errMarshal := json.Marshal(body)
	if errMarshal != nil {
		return cipherBlob{}, errMarshal
	}
	return sealBytes(bKey, bPlain, szId)
}

// openItem 은 항목의 비밀을 복호화한다.
func openItem(bKey []byte, record itemRecord) (SecretBody, error) {
	var body SecretBody
	bPlain, errOpen := openBytes(bKey, record.cipherBlob, record.M_szId)
	if errOpen != nil {
		return body, errOpen
	}
	if errUnmarshal := json.Unmarshal(bPlain, &body); errUnmarshal != nil {
		return body, errUnmarshal
	}
	return body, nil
}

// applyOptional 은 요청에 값이 있을 때만 대상 필드를 바꾼다 (nil 은 유지, "" 는 지우기).
func applyOptional(pszTarget *string, pszInput *string) {
	if pszInput != nil {
		*pszTarget = *pszInput
	}
}

// List 는 계정 메타 목록을 돌려준다. 잠겨 있어도 가능하다.
func (pv *Vault) List() []CredMeta {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	slOut := []CredMeta{}
	if pv.pfile == nil {
		return slOut
	}
	for _, record := range pv.pfile.M_slItems {
		slOut = append(slOut, CredMeta{
			M_szId:         record.M_szId,
			M_szName:       record.M_szName,
			M_szKind:       record.M_szKind,
			M_szUsername:   record.M_szUsername,
			M_i64UpdatedAt: record.M_i64UpdatedAt,
		})
	}
	return slOut
}

// Get 은 항목의 비밀을 돌려준다. 없으면 (nil, nil). 잠겨 있으면 ErrLocked.
func (pv *Vault) Get(szId string) (*SecretBody, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.bKey == nil {
		return nil, ErrLocked
	}
	if pv.pfile == nil {
		return nil, ErrNoVault
	}
	for _, record := range pv.pfile.M_slItems {
		if record.M_szId == szId {
			body, errOpen := openItem(pv.bKey, record)
			if errOpen != nil {
				return nil, errOpen
			}
			return &body, nil
		}
	}
	return nil, nil
}

// Save 는 항목을 만들거나 교체한다 (data/vault.md §6 저장).
// 같은 id 는 제거 후 목록 맨 뒤에 다시 넣는다.
func (pv *Vault) Save(input SaveInput) (CredMeta, error) {
	meta, errSave := pv.saveInternal(input)
	if errSave != nil {
		return CredMeta{}, errSave
	}
	pv.notifyItems()
	return meta, nil
}

func (pv *Vault) saveInternal(input SaveInput) (CredMeta, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.bKey == nil {
		return CredMeta{}, ErrLocked
	}
	if pv.pfile == nil {
		return CredMeta{}, ErrNoVault
	}
	szId := input.M_szId
	if szId == "" {
		newId, errUuid := newUUID()
		if errUuid != nil {
			return CredMeta{}, errUuid
		}
		szId = newId
	}
	body := SecretBody{}
	szName, szKind := input.M_szName, input.M_szKind
	iExisting := -1
	for iIndex, record := range pv.pfile.M_slItems {
		if record.M_szId == szId {
			iExisting = iIndex
			break
		}
	}
	if iExisting >= 0 {
		existing := pv.pfile.M_slItems[iExisting]
		oldBody, errOpen := openItem(pv.bKey, existing)
		if errOpen != nil {
			return CredMeta{}, errOpen
		}
		body = oldBody
		if szName == "" {
			szName = existing.M_szName
		}
		if szKind == "" {
			szKind = existing.M_szKind
		}
	}
	// 비밀 필드: nil 이면 유지, 값이 있으면 교체, "" 이면 지운다 (data/vault.md §6).
	applyOptional(&body.M_szUsername, input.M_pszUsername)
	applyOptional(&body.M_szPassword, input.M_pszPassword)
	applyOptional(&body.M_szPrivateKey, input.M_pszPrivateKey)
	applyOptional(&body.M_szPassphrase, input.M_pszPassphrase)
	// 평문 사용자 이름은 비밀 객체의 값과 항상 같게 쓴다.
	szUsername := body.M_szUsername
	blobSealed, errSeal := sealItem(pv.bKey, szId, body)
	if errSeal != nil {
		return CredMeta{}, errSeal
	}
	now := time.Now().UnixMilli()
	record := itemRecord{
		M_szId:         szId,
		M_szName:       szName,
		M_szKind:       szKind,
		M_szUsername:   szUsername,
		M_i64UpdatedAt: now,
		cipherBlob:     blobSealed,
	}
	slNext := make([]itemRecord, 0, len(pv.pfile.M_slItems)+1)
	for iIndex, existing := range pv.pfile.M_slItems {
		if iIndex != iExisting {
			slNext = append(slNext, existing)
		}
	}
	pv.pfile.M_slItems = append(slNext, record)
	if errSave := pv.saveLocked(); errSave != nil {
		return CredMeta{}, errSave
	}
	return CredMeta{
		M_szId:         szId,
		M_szName:       szName,
		M_szKind:       szKind,
		M_szUsername:   szUsername,
		M_i64UpdatedAt: now,
	}, nil
}

// Remove 는 항목을 지운다. 없는 id 면 아무 일도 하지 않는다.
func (pv *Vault) Remove(szId string) error {
	errRemove := pv.removeInternal(szId)
	if errRemove != nil {
		return errRemove
	}
	pv.notifyItems()
	return nil
}

func (pv *Vault) removeInternal(szId string) error {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return ErrNoVault
	}
	slNext := make([]itemRecord, 0, len(pv.pfile.M_slItems))
	for _, record := range pv.pfile.M_slItems {
		if record.M_szId != szId {
			slNext = append(slNext, record)
		}
	}
	pv.pfile.M_slItems = slNext
	return pv.saveLocked()
}
