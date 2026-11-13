// Package vault 는 vault.json (앱 잠금 비밀번호로 감싼 볼트 키, 복구 코드, 계정 항목) 을 다룬다 (data/vault.md).
package vault

import (
	"encoding/base64"
	"errors"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	"dariterm/internal/jsonfile"
)

// G_cszVaultKeyAad 는 볼트 키를 감쌀 때 쓰는 추가 인증 데이터다.
const G_cszVaultKeyAad = "vault-key"

// G_cszOsUnlockKind 는 OS 잠금 해제 방식 이름이다 (data/vault.md §7).
const G_cszOsUnlockKind = "windows"

// G_ciMinPasswordLen 은 앱 잠금 비밀번호 최소 글자 수다.
const G_ciMinPasswordLen = 8

// ErrLocked 는 볼트가 잠겨 있을 때 돌려준다.
var ErrLocked = errors.New("앱이 잠겨 있습니다")

// ErrNoVault 는 볼트 파일이 없을 때 돌려준다.
var ErrNoVault = errors.New("앱 잠금 비밀번호가 설정되지 않음")

// ErrAlreadySetup 은 이미 볼트가 있을 때 돌려준다.
var ErrAlreadySetup = errors.New("이미 앱 잠금 비밀번호가 설정되어 있습니다")

// ErrPasswordShort 는 앱 잠금 비밀번호가 짧을 때 돌려준다.
var ErrPasswordShort = errors.New("앱 잠금 비밀번호는 8자 이상이어야 합니다")

// Vault 는 볼트 파일의 메모리 사본과, 열려 있을 때만 가지는 볼트 키를 가진다.
type Vault struct {
	mtxData          sync.Mutex
	szPath           string
	pfile            *vaultFile
	bKey             []byte
	fnOnLockChanged  func(bool)
	fnOnItemsChanged func()
}

// Open 은 szPath 의 볼트를 읽는다. 파일이 없으면 미초기화 상태다.
func Open(szPath string) (*Vault, error) {
	pv := &Vault{szPath: szPath}
	if !jsonfile.Exists(szPath) {
		return pv, nil
	}
	var file vaultFile
	if errRead := jsonfile.Read(szPath, &file); errRead != nil {
		// 없음과 깨짐을 구분한다 (sync-backup.md §1, 결정 S1): 깨진 볼트는 미초기화로 취급하되
		// 사용자 데이터를 잃지 않도록 vault.json.corrupt-<시각> 으로 옮겨 보존한다.
		szCorrupt := szPath + ".corrupt-" + time.Now().Format("20060102-150405")
		if errMove := os.Rename(szPath, szCorrupt); errMove != nil {
			return nil, errMove
		}
		return pv, nil
	}
	if file.M_slItems == nil {
		file.M_slItems = []itemRecord{}
	}
	pv.pfile = &file
	return pv, nil
}

// SetOnLockChanged 는 볼트가 열리거나 잠길 때 부를 콜백(true=열림)을 등록한다.
func (pv *Vault) SetOnLockChanged(fnCallback func(bool)) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	pv.fnOnLockChanged = fnCallback
}

// SetOnItemsChanged 는 계정 항목이 바뀔 때 부를 콜백을 등록한다 (동기화 예약용).
func (pv *Vault) SetOnItemsChanged(fnCallback func()) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	pv.fnOnItemsChanged = fnCallback
}

// Status 는 현재 상태를 돌려준다.
func (pv *Vault) Status() Status {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	status := Status{
		M_bInitialized:       pv.pfile != nil,
		M_bUnlocked:          pv.bKey != nil,
		M_bOsUnlockAvailable: G_bOsUnlockSupported,
	}
	if pv.pfile != nil && pv.pfile.M_szOsWrappedWin != "" {
		status.M_bOsUnlockEnabled = true
	}
	if G_bOsUnlockSupported {
		status.M_szOsUnlockKind = G_cszOsUnlockKind
	}
	return status
}

// notifyLock 은 잠금 상태 콜백을 락 밖에서 부른다.
func (pv *Vault) notifyLock(bOpen bool) {
	pv.mtxData.Lock()
	fnCallback := pv.fnOnLockChanged
	pv.mtxData.Unlock()
	if fnCallback != nil {
		fnCallback(bOpen)
	}
}

// notifyItems 는 항목 변경 콜백을 락 밖에서 부른다.
func (pv *Vault) notifyItems() {
	pv.mtxData.Lock()
	fnCallback := pv.fnOnItemsChanged
	pv.mtxData.Unlock()
	if fnCallback != nil {
		fnCallback()
	}
}

// validPassword 는 앱 잠금 비밀번호 길이(문자 수)를 검사한다.
func validPassword(szPassword string) error {
	if utf8.RuneCountInString(szPassword) < G_ciMinPasswordLen {
		return ErrPasswordShort
	}
	return nil
}

// saveLocked 는 mtxData 를 잡은 상태에서 부른다.
func (pv *Vault) saveLocked() error {
	return jsonfile.Write(pv.szPath, pv.pfile)
}

// decodeB64 는 base64 문자열을 해석한다.
func decodeB64(szValue string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(szValue)
}
