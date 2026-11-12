package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/text/unicode/norm"
)

// G_cszRecoveryAlphabet 은 복구 코드에 쓰는 32글자 알파벳이다 (I, O, 0, 1 제외, data/vault.md §5).
const G_cszRecoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// g_ciRecoveryRawLen 은 하이픈을 뺀 복구 코드 길이다.
const g_ciRecoveryRawLen = 24

// g_ciScryptN, g_ciScryptR, g_ciScryptP 는 키 파생 매개변수다 (data/vault.md §2).
const (
	g_ciScryptN = 131072
	g_ciScryptR = 8
	g_ciScryptP = 1
	g_ciKeyLen  = 32
	g_ciSaltLen = 16
)

// errCipherShort 는 암호문 형식이 잘못되었을 때 돌려준다.
var errCipherShort = errors.New("암호문 형식 오류")

// cipherBlob 은 AES-GCM 결과 {iv, tag, ct} 다. 모든 값은 base64 이다.
type cipherBlob struct {
	M_szIv  string `json:"iv"`
	M_szTag string `json:"tag"`
	M_szCt  string `json:"ct"`
}

// kdfParams 는 scrypt 매개변수 객체다.
type kdfParams struct {
	M_szAlg  string `json:"alg"`
	M_iN     int    `json:"N"`
	M_iR     int    `json:"r"`
	M_iP     int    `json:"p"`
	M_szSalt string `json:"salt"`
}

// newKdfParams 는 새 salt 로 매개변수를 만든다. salt 는 호출마다 새로 뽑는다.
func newKdfParams() (kdfParams, []byte, error) {
	bSalt := make([]byte, g_ciSaltLen)
	if _, errRand := rand.Read(bSalt); errRand != nil {
		return kdfParams{}, nil, errRand
	}
	return kdfParams{
		M_szAlg:  "scrypt",
		M_iN:     g_ciScryptN,
		M_iR:     g_ciScryptR,
		M_iP:     g_ciScryptP,
		M_szSalt: base64.StdEncoding.EncodeToString(bSalt),
	}, bSalt, nil
}

// deriveKey 는 입력 문자열을 NFKC 로 정규화해 scrypt 키를 만든다.
func deriveKey(szInput string, paramsKdf kdfParams) ([]byte, error) {
	bSalt, errDecode := base64.StdEncoding.DecodeString(paramsKdf.M_szSalt)
	if errDecode != nil {
		return nil, errDecode
	}
	bNorm := []byte(norm.NFKC.String(szInput))
	return scrypt.Key(bNorm, bSalt, paramsKdf.M_iN, paramsKdf.M_iR, paramsKdf.M_iP, g_ciKeyLen)
}

// sealBytes 는 key 로 plain 을 암호화한다. szAad 가 비어 있지 않으면 추가 인증 데이터로 쓴다.
func sealBytes(bKey []byte, bPlain []byte, szAad string) (cipherBlob, error) {
	block, errAes := aes.NewCipher(bKey)
	if errAes != nil {
		return cipherBlob{}, errAes
	}
	aead, errGcm := cipher.NewGCM(block)
	if errGcm != nil {
		return cipherBlob{}, errGcm
	}
	bNonce := make([]byte, aead.NonceSize())
	if _, errRand := rand.Read(bNonce); errRand != nil {
		return cipherBlob{}, errRand
	}
	bSealed := aead.Seal(nil, bNonce, bPlain, []byte(szAad))
	nTagStart := len(bSealed) - aead.Overhead()
	return cipherBlob{
		M_szIv:  base64.StdEncoding.EncodeToString(bNonce),
		M_szTag: base64.StdEncoding.EncodeToString(bSealed[nTagStart:]),
		M_szCt:  base64.StdEncoding.EncodeToString(bSealed[:nTagStart]),
	}, nil
}

// openBytes 는 cipherBlob 을 key 로 복호화한다. 인증 실패 시 오류.
func openBytes(bKey []byte, blob cipherBlob, szAad string) ([]byte, error) {
	bNonce, errIv := base64.StdEncoding.DecodeString(blob.M_szIv)
	bTag, errTag := base64.StdEncoding.DecodeString(blob.M_szTag)
	bCt, errCt := base64.StdEncoding.DecodeString(blob.M_szCt)
	if errIv != nil || errTag != nil || errCt != nil {
		return nil, errCipherShort
	}
	block, errAes := aes.NewCipher(bKey)
	if errAes != nil {
		return nil, errAes
	}
	aead, errGcm := cipher.NewGCM(block)
	if errGcm != nil {
		return nil, errGcm
	}
	if len(bNonce) != aead.NonceSize() {
		return nil, errCipherShort
	}
	bJoined := append(append([]byte{}, bCt...), bTag...)
	return aead.Open(nil, bNonce, bJoined, []byte(szAad))
}

// newRandomKey 는 무작위 32바이트 볼트 키를 만든다.
func newRandomKey() ([]byte, error) {
	bKey := make([]byte, g_ciKeyLen)
	if _, errRand := rand.Read(bKey); errRand != nil {
		return nil, errRand
	}
	return bKey, nil
}

// newRecoveryCode 는 복구 코드 24글자를 뽑아 "XXXX-XXXX-XXXX-XXXX-XXXX-XXXX" 형식으로 돌려준다.
func newRecoveryCode() (string, error) {
	bRand := make([]byte, g_ciRecoveryRawLen)
	if _, errRand := rand.Read(bRand); errRand != nil {
		return "", errRand
	}
	bOut := make([]byte, 0, g_ciRecoveryRawLen+5)
	for iIndex, byteValue := range bRand {
		if iIndex > 0 && iIndex%4 == 0 {
			bOut = append(bOut, '-')
		}
		// 알파벳 길이 32 는 2의 거듭제곱이라 모듈로 편향이 없다.
		bOut = append(bOut, G_cszRecoveryAlphabet[int(byteValue)%len(G_cszRecoveryAlphabet)])
	}
	return string(bOut), nil
}

// normalizeRecoveryCode 는 입력을 대문자로 바꾸고 하이픈·공백·탭을 뺀 24글자로 만든다.
func normalizeRecoveryCode(szInput string) string {
	szUpper := strings.ToUpper(szInput)
	return strings.NewReplacer("-", "", " ", "", "\t", "").Replace(szUpper)
}
