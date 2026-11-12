// Package vault 는 v1.1.0 vault.json 형식과 호환되는 자격 증명 볼트다.
//
//	마스터 비밀번호 --scrypt(N=2^17,r=8,p=1)--> KEK --AES-256-GCM--> 볼트 키(랜덤 256bit)
//	복구 코드       --scrypt-->                  RK  --AES-256-GCM--> 볼트 키
//	볼트 키 --AES-256-GCM(aad=항목 id)--> 각 자격 증명
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/text/unicode/norm"

	"termgo/internal/store"
)

// Enc 는 AES-GCM 결과 (iv / tag / ct 각각 base64).
type Enc struct {
	IV  string `json:"iv"`
	Tag string `json:"tag"`
	CT  string `json:"ct"`
}

// Kdf 는 scrypt 매개변수.
type Kdf struct {
	Alg  string `json:"alg"`
	N    int    `json:"N"`
	R    int    `json:"r"`
	P    int    `json:"p"`
	Salt string `json:"salt"`
}

// Item 은 볼트 항목 하나 (비밀은 Enc 안에 있다).
type Item struct {
	Enc
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Username  string `json:"username,omitempty"`
	UpdatedAt int64  `json:"updatedAt"`
}

// File 은 vault.json.
type File struct {
	Version         int    `json:"version"`
	Kdf             Kdf    `json:"kdf"`
	Wrapped         Enc    `json:"wrapped"`
	RecoveryKdf     *Kdf   `json:"recoveryKdf,omitempty"`
	RecoveryWrapped *Enc   `json:"recoveryWrapped,omitempty"`
	OsWrapped       string `json:"osWrapped,omitempty"`    // v1.1.0(Electron safeStorage) 값 — 읽기 전용, 사용 안 함
	OsWrappedWin    string `json:"osWrappedWin,omitempty"` // TermGo: DPAPI 로 감싼 볼트 키(base64)
	Items           []Item `json:"items"`
}

// Meta 는 비밀이 빠진 목록용 정보 (CredentialMeta).
type Meta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Username  string `json:"username,omitempty"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Secret 은 CredentialSecret.
type Secret struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// Input 은 CredentialInput. 포인터 필드는 "지정 안 함 = 기존 값 유지".
type Input struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Username   *string `json:"username"`
	Password   *string `json:"password"`
	PrivateKey *string `json:"privateKey"`
	Passphrase *string `json:"passphrase"`
}

// Status 는 VaultStatus.
type Status struct {
	Initialized       bool   `json:"initialized"`
	Unlocked          bool   `json:"unlocked"`
	OsUnlockAvailable bool   `json:"osUnlockAvailable"`
	OsUnlockEnabled   bool   `json:"osUnlockEnabled"`
	OsUnlockKind      string `json:"osUnlockKind,omitempty"`
}

// Vault 는 볼트 본체.
type Vault struct {
	mu           sync.Mutex
	szFile       string
	f            *File
	key          []byte
	OnLockChange func(bUnlocked bool)
	// OnItemsChange 는 항목이 저장·삭제되었을 때 (git 동기화 예약용).
	OnItemsChange func()
}

const (
	scryptN = 1 << 17
	scryptR = 8
	scryptP = 1
)

// ErrLocked 는 볼트가 잠겨 있을 때.
var ErrLocked = errors.New("볼트가 잠겨 있습니다")

// Open 은 data 폴더의 vault.json 을 읽는다 (없으면 미초기화 상태).
func Open(szDir string) *Vault {
	pstrV := &Vault{szFile: filepath.Join(szDir, "vault.json"), OnLockChange: func(bool) {}}
	var f File
	if err := store.ReadJSON(pstrV.szFile, &f); err == nil {
		pstrV.f = &f
	}
	return pstrV
}

func b64(arr []byte) string { return base64.StdEncoding.EncodeToString(arr) }

func unb64(sz string) ([]byte, error) { return base64.StdEncoding.DecodeString(sz) }

func derive(szSecret string, k Kdf) ([]byte, error) {
	arrSalt, err := unb64(k.Salt)
	if err != nil {
		return nil, err
	}
	return scrypt.Key([]byte(norm.NFKC.String(szSecret)), arrSalt, k.N, k.R, k.P, 32)
}

func newKdf() Kdf {
	arr := make([]byte, 16)
	_, _ = rand.Read(arr)
	return Kdf{Alg: "scrypt", N: scryptN, R: scryptR, P: scryptP, Salt: b64(arr)}
}

func gcmOf(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

// seal 은 Node 의 (iv, tag, ct) 분리 형식으로 AES-256-GCM 암호화한다.
func seal(key, plain []byte, szAAD string) (Enc, error) {
	g, err := gcmOf(key)
	if err != nil {
		return Enc{}, err
	}
	arrIV := make([]byte, g.NonceSize())
	if _, err := rand.Read(arrIV); err != nil {
		return Enc{}, err
	}
	var arrAAD []byte
	if szAAD != "" {
		arrAAD = []byte(szAAD)
	}
	out := g.Seal(nil, arrIV, plain, arrAAD) // ct || tag(16)
	n := len(out) - g.Overhead()
	return Enc{IV: b64(arrIV), Tag: b64(out[n:]), CT: b64(out[:n])}, nil
}

func open(key []byte, e Enc, szAAD string) ([]byte, error) {
	g, err := gcmOf(key)
	if err != nil {
		return nil, err
	}
	arrIV, err1 := unb64(e.IV)
	arrTag, err2 := unb64(e.Tag)
	arrCT, err3 := unb64(e.CT)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, errors.New("잘못된 암호문")
	}
	var arrAAD []byte
	if szAAD != "" {
		arrAAD = []byte(szAAD)
	}
	return g.Open(nil, arrIV, append(arrCT, arrTag...), arrAAD)
}

func newRecoveryCode() string {
	const szAlpha = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	arr := make([]byte, 24)
	_, _ = rand.Read(arr)
	var sb strings.Builder
	for i, b := range arr {
		if i > 0 && i%4 == 0 {
			sb.WriteByte('-')
		}
		sb.WriteByte(szAlpha[int(b)%len(szAlpha)])
	}
	return sb.String()
}

func (pstrV *Vault) saveLocked() error {
	if pstrV.f == nil {
		return nil
	}
	return store.WriteJSONAtomic(pstrV.szFile, pstrV.f)
}

// Status 는 현재 상태.
func (pstrV *Vault) Status() Status {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	return Status{
		Initialized:       pstrV.f != nil,
		Unlocked:          pstrV.key != nil,
		OsUnlockAvailable: osProtectAvailable(),
		OsUnlockEnabled:   pstrV.f != nil && pstrV.f.OsWrappedWin != "",
		OsUnlockKind:      "windows",
	}
}

// Setup 은 새 볼트를 만들고 복구 코드를 돌려준다. (잠금 해제 알림은 보내지 않는다: 복구 코드 화면 유지)
func (pstrV *Vault) Setup(szPassword string) (map[string]string, error) {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f != nil {
		return nil, errors.New("이미 볼트가 설정되어 있습니다")
	}
	if len([]rune(szPassword)) < 8 {
		return nil, errors.New("마스터 비밀번호는 8자 이상이어야 합니다")
	}
	arrVK := make([]byte, 32)
	if _, err := rand.Read(arrVK); err != nil {
		return nil, err
	}
	kdf, rkdf := newKdf(), newKdf()
	szRC := newRecoveryCode()
	kek, err := derive(szPassword, kdf)
	if err != nil {
		return nil, err
	}
	rk, err := derive(strings.ReplaceAll(szRC, "-", ""), rkdf)
	if err != nil {
		return nil, err
	}
	wrapped, err := seal(kek, arrVK, "vault-key")
	if err != nil {
		return nil, err
	}
	rwrapped, err := seal(rk, arrVK, "vault-key")
	if err != nil {
		return nil, err
	}
	pstrV.f = &File{Version: 1, Kdf: kdf, Wrapped: wrapped, RecoveryKdf: &rkdf, RecoveryWrapped: &rwrapped, Items: []Item{}}
	pstrV.key = arrVK
	if err := pstrV.saveLocked(); err != nil {
		return nil, err
	}
	return map[string]string{"recoveryCode": szRC}, nil
}

// Unlock 은 마스터 비밀번호로 잠금을 푼다.
func (pstrV *Vault) Unlock(szPassword string) bool {
	pstrV.mu.Lock()
	if pstrV.f == nil {
		pstrV.mu.Unlock()
		return false
	}
	kek, err := derive(szPassword, pstrV.f.Kdf)
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	vk, err := open(kek, pstrV.f.Wrapped, "vault-key")
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	pstrV.key = vk
	pstrV.mu.Unlock()
	pstrV.OnLockChange(true)
	return true
}

// UnlockOS 는 DPAPI 로 감싸 둔 볼트 키로 잠금을 푼다.
func (pstrV *Vault) UnlockOS() bool {
	pstrV.mu.Lock()
	if pstrV.f == nil || pstrV.f.OsWrappedWin == "" {
		pstrV.mu.Unlock()
		return false
	}
	arrBlob, err := unb64(pstrV.f.OsWrappedWin)
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	vk, err := osUnprotect(arrBlob)
	if err != nil || len(vk) != 32 {
		pstrV.mu.Unlock()
		return false
	}
	if len(pstrV.f.Items) > 0 {
		if _, err := open(vk, pstrV.f.Items[0].Enc, pstrV.f.Items[0].ID); err != nil {
			pstrV.mu.Unlock()
			return false
		}
	}
	pstrV.key = vk
	pstrV.mu.Unlock()
	pstrV.OnLockChange(true)
	return true
}

// SetOsUnlock 은 OS 잠금 해제를 켜고 끈다.
func (pstrV *Vault) SetOsUnlock(bEnable bool) error {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.key == nil {
		return ErrLocked
	}
	if bEnable && osProtectAvailable() {
		arr, err := osProtect(pstrV.key)
		if err != nil {
			return err
		}
		pstrV.f.OsWrappedWin = b64(arr)
	} else {
		pstrV.f.OsWrappedWin = ""
	}
	return pstrV.saveLocked()
}

// Recover 는 복구 코드로 비밀번호를 재설정하고 잠금을 푼다.
func (pstrV *Vault) Recover(szCode, szNewPassword string) bool {
	pstrV.mu.Lock()
	if pstrV.f == nil || pstrV.f.RecoveryKdf == nil || pstrV.f.RecoveryWrapped == nil {
		pstrV.mu.Unlock()
		return false
	}
	szClean := strings.ToUpper(strings.NewReplacer("-", "", " ", "", "\t", "").Replace(szCode))
	rk, err := derive(szClean, *pstrV.f.RecoveryKdf)
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	vk, err := open(rk, *pstrV.f.RecoveryWrapped, "vault-key")
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	kdf := newKdf()
	kek, err := derive(szNewPassword, kdf)
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	wrapped, err := seal(kek, vk, "vault-key")
	if err != nil {
		pstrV.mu.Unlock()
		return false
	}
	pstrV.f.Kdf, pstrV.f.Wrapped, pstrV.key = kdf, wrapped, vk
	_ = pstrV.saveLocked()
	pstrV.mu.Unlock()
	pstrV.OnLockChange(true)
	return true
}

// ChangePassword 는 볼트 키만 다시 감싼다.
func (pstrV *Vault) ChangePassword(szOld, szNew string) bool {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f == nil {
		return false
	}
	kek, err := derive(szOld, pstrV.f.Kdf)
	if err != nil {
		return false
	}
	vk, err := open(kek, pstrV.f.Wrapped, "vault-key")
	if err != nil {
		return false
	}
	kdf := newKdf()
	nk, err := derive(szNew, kdf)
	if err != nil {
		return false
	}
	wrapped, err := seal(nk, vk, "vault-key")
	if err != nil {
		return false
	}
	pstrV.f.Kdf, pstrV.f.Wrapped = kdf, wrapped
	return pstrV.saveLocked() == nil
}

// Lock 은 메모리의 볼트 키를 지운다.
func (pstrV *Vault) Lock() {
	pstrV.mu.Lock()
	for i := range pstrV.key {
		pstrV.key[i] = 0
	}
	pstrV.key = nil
	pstrV.mu.Unlock()
	pstrV.OnLockChange(false)
}

// IsUnlocked 는 잠금 해제 여부.
func (pstrV *Vault) IsUnlocked() bool {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	return pstrV.key != nil
}

// List 는 비밀 없는 항목 목록.
func (pstrV *Vault) List() []Meta {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	out := []Meta{}
	if pstrV.f == nil {
		return out
	}
	for _, it := range pstrV.f.Items {
		out = append(out, Meta{ID: it.ID, Name: it.Name, Kind: it.Kind, Username: it.Username, UpdatedAt: it.UpdatedAt})
	}
	return out
}

func (pstrV *Vault) getLocked(szID string) (*Secret, error) {
	if pstrV.key == nil {
		return nil, ErrLocked
	}
	if pstrV.f == nil {
		return nil, nil
	}
	for _, it := range pstrV.f.Items {
		if it.ID == szID {
			arr, err := open(pstrV.key, it.Enc, it.ID)
			if err != nil {
				return nil, err
			}
			var s Secret
			if err := json.Unmarshal(arr, &s); err != nil {
				return nil, err
			}
			return &s, nil
		}
	}
	return nil, nil
}

// Get 은 항목의 비밀을 복호화한다 (없으면 nil).
func (pstrV *Vault) Get(szID string) (*Secret, error) {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	return pstrV.getLocked(szID)
}

// Save 는 항목을 만들거나 갱신한다 (v1.1.0 save_).
func (pstrV *Vault) Save(in Input) (*Meta, error) {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.key == nil {
		return nil, ErrLocked
	}
	if pstrV.f == nil {
		return nil, errors.New("볼트 없음")
	}
	szID := in.ID
	var prev *Secret
	if szID == "" {
		szID = NewUUID()
	} else {
		prev, _ = pstrV.getLocked(szID)
	}
	if prev == nil {
		prev = &Secret{}
	}
	pick := func(p *string, szOld string) string {
		if p != nil {
			return *p
		}
		return szOld
	}
	sec := Secret{
		Username:   pick(in.Username, prev.Username),
		Password:   pick(in.Password, prev.Password),
		PrivateKey: pick(in.PrivateKey, prev.PrivateKey),
		Passphrase: pick(in.Passphrase, prev.Passphrase),
	}
	arrPlain, _ := json.Marshal(sec)
	enc, err := seal(pstrV.key, arrPlain, szID)
	if err != nil {
		return nil, err
	}
	it := Item{Enc: enc, ID: szID, Name: in.Name, Kind: in.Kind, Username: sec.Username, UpdatedAt: time.Now().UnixMilli()}
	items := make([]Item, 0, len(pstrV.f.Items)+1)
	for _, x := range pstrV.f.Items {
		if x.ID != szID {
			items = append(items, x)
		}
	}
	pstrV.f.Items = append(items, it)
	if err := pstrV.saveLocked(); err != nil {
		return nil, err
	}
	if pstrV.OnItemsChange != nil {
		defer pstrV.OnItemsChange()
	}
	return &Meta{ID: szID, Name: it.Name, Kind: it.Kind, Username: it.Username, UpdatedAt: it.UpdatedAt}, nil
}

// Remove 는 항목을 지운다.
func (pstrV *Vault) Remove(szID string) error {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f == nil {
		return nil
	}
	items := pstrV.f.Items[:0:0]
	for _, x := range pstrV.f.Items {
		if x.ID != szID {
			items = append(items, x)
		}
	}
	pstrV.f.Items = items
	if pstrV.OnItemsChange != nil {
		defer pstrV.OnItemsChange()
	}
	return pstrV.saveLocked()
}

// ExportRaw 는 백업용 원본.
func (pstrV *Vault) ExportRaw() *File {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	return pstrV.f
}

// ImportRaw 는 백업에서 볼트를 교체한다 (다른 PC 의 OS 보호 값은 버린다).
func (pstrV *Vault) ImportRaw(f File) error {
	pstrV.mu.Lock()
	f.OsWrapped, f.OsWrappedWin = "", ""
	pstrV.f, pstrV.key = &f, nil
	err := pstrV.saveLocked()
	pstrV.mu.Unlock()
	pstrV.OnLockChange(false)
	return err
}

// NewUUID 는 v4 UUID 문자열을 만든다.
func NewUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ---- 3.11 git 동기화 지원 ----

// Encrypt 는 잠금 해제된 볼트 키로 임의 데이터를 AES-GCM 암호화한다 (동기화 파일 본문용).
func (pstrV *Vault) Encrypt(plain []byte, szAAD string) (Enc, error) {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.key == nil {
		return Enc{}, ErrLocked
	}
	return seal(pstrV.key, plain, szAAD)
}

// Decrypt 는 Encrypt 의 반대. 다른 볼트(키)로 만든 데이터면 오류.
func (pstrV *Vault) Decrypt(e Enc, szAAD string) ([]byte, error) {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.key == nil {
		return nil, ErrLocked
	}
	return open(pstrV.key, e, szAAD)
}

// Header 는 다른 PC 가 같은 마스터 비밀번호로 열 수 있게 하는 최소 정보 (항목·OS 보호 값 제외 — 계정 이름 등이 평문으로 나가지 않는다).
func (pstrV *Vault) Header() *File {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f == nil {
		return nil
	}
	h := *pstrV.f
	h.Items, h.OsWrapped, h.OsWrappedWin = []Item{}, "", ""
	return &h
}

// Items 는 항목 전체 (암호화된 비밀 포함).
func (pstrV *Vault) Items() []Item {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f == nil {
		return nil
	}
	return append([]Item(nil), pstrV.f.Items...)
}

// ReplaceItems 는 항목 목록을 통째로 바꾼다 (병합 결과 적용).
func (pstrV *Vault) ReplaceItems(items []Item) error {
	pstrV.mu.Lock()
	defer pstrV.mu.Unlock()
	if pstrV.f == nil {
		return errors.New("볼트 없음")
	}
	pstrV.f.Items = append([]Item{}, items...)
	return pstrV.saveLocked()
}

// AdoptRemote 는 저장소의 볼트를 이 PC 볼트로 받는다: 헤더+항목으로 교체하고 마스터 비밀번호로 연다.
// 비밀번호가 맞지 않으면 아무것도 바꾸지 않고 false.
func (pstrV *Vault) AdoptRemote(h File, items []Item, szPassword string) bool {
	kek, err := derive(szPassword, h.Kdf)
	if err != nil {
		return false
	}
	vk, err := open(kek, h.Wrapped, "vault-key")
	if err != nil {
		return false
	}
	h.Items, h.OsWrapped, h.OsWrappedWin = append([]Item{}, items...), "", ""
	pstrV.mu.Lock()
	pstrV.f, pstrV.key = &h, vk
	err = pstrV.saveLocked()
	pstrV.mu.Unlock()
	pstrV.OnLockChange(true)
	return err == nil
}
