// Package license 는 업무용 라이선스 키를 오프라인으로 확인한다 (결정 D78, spec/behavior/license.md).
// 발급(서명)은 이 앱이 하지 않는다 — 비밀키를 가진 발급 쪽(나중에 서버, 지금은 tools/licensekey)이 만든다.
//
// 키 형식: "DT1." + base64url(JSON 내용) + "." + base64url(Ed25519 서명). 서명 대상은 "DT1." + 내용 부분 글자 그대로.
// 내용 JSON: {"v":1,"id":"DT-2026-0001","org":"회사","seats":5,"ver":1,"exp":"2027-12-31","iat":"2026-11-14","email":"..."}
//   - ver: 쓸 수 있는 최대 주 버전(1 이면 1.x 까지). 0 이면 모든 버전.
//   - exp: 만료일(그날까지 유효). 비면 영구.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// G_cszPrefix 는 키 머리말(형식 1판)이다.
const G_cszPrefix = "DT1."

// g_szPublicKey 는 발급 서명을 확인하는 공개키(base64 std, 32바이트)다. 비밀키는 저장소 밖에 둔다(docs/host-settings/02).
var g_szPublicKey = "TiHUPkvmtt6Kz8zd8EBpZiWkuv3ZxV5q73vENYAZ2Ng="

// 상태 값 (license.status 의 state).
const (
	G_cszStateNone    = "none"    // 키 없음
	G_cszStateValid   = "valid"   // 유효
	G_cszStateInvalid = "invalid" // 형식·서명이 맞지 않음
	G_cszStateExpired = "expired" // 만료일 지남
	G_cszStateVersion = "version" // 이 버전은 라이선스 범위 밖
)

// Info 는 키 내용이다.
type Info struct {
	M_iVersion  int    `json:"v"`
	M_szId      string `json:"id"`
	M_szOrg     string `json:"org"`
	M_iSeats    int    `json:"seats"`
	M_iMaxMajor int    `json:"ver"`
	M_szExpire  string `json:"exp"`
	M_szIssued  string `json:"iat"`
	M_szEmail   string `json:"email"`
}

// Status 는 화면에 보일 확인 결과다.
type Status struct {
	M_szState   string `json:"state"`
	M_szId      string `json:"id,omitempty"`
	M_szOrg     string `json:"org,omitempty"`
	M_iSeats    int    `json:"seats,omitempty"`
	M_iMaxMajor int    `json:"maxMajor,omitempty"`
	M_szExpire  string `json:"exp,omitempty"`
}

var errFormat = errors.New("license: bad format")
var errSignature = errors.New("license: bad signature")

// Parse 는 키를 해석하고 서명을 확인한다. 공개키 pub 가 nil 이면 내장 공개키를 쓴다.
func Parse(szKey string, pub ed25519.PublicKey) (Info, error) {
	szKey = strings.Join(strings.Fields(szKey), "") // 붙여 넣을 때 들어간 공백·줄바꿈 제거
	if !strings.HasPrefix(szKey, G_cszPrefix) {
		return Info{}, errFormat
	}
	slParts := strings.Split(strings.TrimPrefix(szKey, G_cszPrefix), ".")
	if len(slParts) != 2 {
		return Info{}, errFormat
	}
	arrBody, errBody := base64.RawURLEncoding.DecodeString(slParts[0])
	arrSig, errSig := base64.RawURLEncoding.DecodeString(slParts[1])
	if errBody != nil || errSig != nil || len(arrSig) != ed25519.SignatureSize {
		return Info{}, errFormat
	}
	if pub == nil {
		pub = embeddedPublicKey()
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, []byte(G_cszPrefix+slParts[0]), arrSig) {
		return Info{}, errSignature
	}
	var info Info
	if errJson := json.Unmarshal(arrBody, &info); errJson != nil || info.M_iVersion != 1 || info.M_szId == "" {
		return Info{}, errFormat
	}
	return info, nil
}

func embeddedPublicKey() ed25519.PublicKey {
	arrKey, errKey := base64.StdEncoding.DecodeString(g_szPublicKey)
	if errKey != nil {
		return nil
	}
	return ed25519.PublicKey(arrKey)
}

// Check 는 키를 확인해 상태를 만든다. szAppVersion 은 "1.2.3" 형식, tmNow 는 지금 시각.
func Check(szKey string, szAppVersion string, tmNow time.Time, pub ed25519.PublicKey) Status {
	if strings.TrimSpace(szKey) == "" {
		return Status{M_szState: G_cszStateNone}
	}
	info, errParse := Parse(szKey, pub)
	if errParse != nil {
		return Status{M_szState: G_cszStateInvalid}
	}
	status := Status{M_szState: G_cszStateValid, M_szId: info.M_szId, M_szOrg: info.M_szOrg, M_iSeats: info.M_iSeats, M_iMaxMajor: info.M_iMaxMajor, M_szExpire: info.M_szExpire}
	if info.M_szExpire != "" {
		tmExp, errExp := time.ParseInLocation("2006-01-02", info.M_szExpire, time.Local)
		if errExp != nil {
			return Status{M_szState: G_cszStateInvalid}
		}
		if tmNow.After(tmExp.AddDate(0, 0, 1)) {
			status.M_szState = G_cszStateExpired
			return status
		}
	}
	if info.M_iMaxMajor > 0 && majorOf(szAppVersion) > info.M_iMaxMajor {
		status.M_szState = G_cszStateVersion
	}
	return status
}

func majorOf(szVersion string) int {
	szMajor, _, _ := strings.Cut(strings.TrimPrefix(szVersion, "v"), ".")
	iMajor, errAtoi := strconv.Atoi(szMajor)
	if errAtoi != nil {
		return 0
	}
	return iMajor
}
