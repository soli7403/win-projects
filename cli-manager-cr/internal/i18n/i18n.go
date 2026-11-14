// Package i18n 은 백엔드 문구(오류·터미널 안내·대화상자 제목)를 화면 언어로 바꾼다 (D75, spec/behavior/i18n.md).
// 문구는 저장소 루트 locales/<언어>.json(프런트엔드와 같은 파일)에 있고, 코드는 키로 부른다:
//
//	i18n.T("be.session.connecting", "title", szTitle, "target", szTarget)
//
// 지금 언어는 시작할 때와 설정(settings.language)이 바뀔 때 SetLang 으로 맞춘다. 비어 있으면 OS 언어.
package i18n

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync/atomic"

	"dariterm/locales"
)

// G_cszFallbackLang 은 맞는 언어가 없을 때 쓰는 언어다.
const G_cszFallbackLang = "en"

// G_slLangs 는 지원 언어 id 다 (locales/<id>.json, frontend/src/i18n LANGS 와 같은 순서).
var G_slLangs = []string{"en", "ko", "ja", "zh", "tw", "de", "fr", "es", "pt", "ru"}

// g_mLocales 는 언어 id → (키 → 문구) 표다. 시작할 때 한 번 읽는다.
var g_mLocales = loadLocales()

// g_atomicLang 은 지금 언어 id(string)다.
var g_atomicLang atomic.Value

var g_rePlaceholder = regexp.MustCompile(`\{(\w+)\}`)

func init() {
	// 시작 값은 원문 언어(한국어). 앱은 NewService 에서 설정·OS 언어로 곧바로 바꾼다(applyLanguage).
	g_atomicLang.Store("ko")
}

func loadLocales() map[string]map[string]string {
	mLocales := map[string]map[string]string{}
	slEntries, errRead := locales.G_fsLocales.ReadDir(".")
	if errRead != nil {
		return mLocales
	}
	for _, entry := range slEntries {
		szName := entry.Name()
		if path.Ext(szName) != ".json" {
			continue
		}
		arrData, errFile := locales.G_fsLocales.ReadFile(szName)
		if errFile != nil {
			continue
		}
		mDict := map[string]string{}
		if errJson := json.Unmarshal(arrData, &mDict); errJson != nil {
			continue
		}
		mLocales[strings.TrimSuffix(szName, ".json")] = mDict
	}
	return mLocales
}

// IsLang 은 지원 언어 id 인지 본다.
func IsLang(szLang string) bool {
	for _, szId := range G_slLangs {
		if szId == szLang {
			return true
		}
	}
	return false
}

// Match 는 OS 언어 태그(ko-KR, zh-Hant-TW, pt_BR.UTF-8 …)를 지원 언어로 맞춘다.
// 앞에서부터 처음 맞는 것을 쓰고, 하나도 없으면 영어 (frontend matchLang 과 같은 규칙).
func Match(slTags []string) string {
	for _, szRaw := range slTags {
		szTag := strings.ToLower(strings.TrimSpace(szRaw))
		szTag = strings.ReplaceAll(szTag, "_", "-")
		szTag, _, _ = strings.Cut(szTag, ".")
		if szTag == "" {
			continue
		}
		slParts := strings.Split(szTag, "-")
		szPrimary := slParts[0]
		switch szPrimary {
		case "zh":
			for _, szPart := range slParts[1:] {
				if szPart == "hant" || szPart == "tw" || szPart == "hk" || szPart == "mo" {
					return "tw"
				}
			}
			return "zh"
		case "pt":
			return "pt"
		case "tw":
			continue
		}
		if IsLang(szPrimary) {
			return szPrimary
		}
	}
	return G_cszFallbackLang
}

// Resolve 는 설정 값이 지원 언어면 그것을, 아니면(비어 있음 = 아직 안 고름) OS 언어를 맞춘 값을 돌려준다.
func Resolve(szSetting string) string {
	if IsLang(szSetting) {
		return szSetting
	}
	return Match(DetectOS())
}

// SetLang 은 지금 언어를 바꾼다. 지원하지 않는 값은 영어.
func SetLang(szLang string) {
	if !IsLang(szLang) {
		szLang = G_cszFallbackLang
	}
	g_atomicLang.Store(szLang)
}

// Lang 은 지금 언어 id 다.
func Lang() string {
	szLang, _ := g_atomicLang.Load().(string)
	return szLang
}

// T 는 키를 지금 언어 문구로 바꾼다. 그 언어에 없으면 영어 → 한국어 → 키 순서.
// slPairs 는 자리표시자 이름과 값의 쌍이다: T("x", "name", szName, "count", 3).
func T(szId string, slPairs ...any) string {
	szText := lookup(Lang(), szId)
	if len(slPairs) < 2 {
		return szText
	}
	mVars := map[string]string{}
	for iIndex := 0; iIndex+1 < len(slPairs); iIndex += 2 {
		mVars[fmt.Sprint(slPairs[iIndex])] = fmt.Sprint(slPairs[iIndex+1])
	}
	return g_rePlaceholder.ReplaceAllStringFunc(szText, func(szMatch string) string {
		if szValue, bOk := mVars[szMatch[1:len(szMatch)-1]]; bOk {
			return szValue
		}
		return szMatch
	})
}

func lookup(szLang string, szId string) string {
	for _, szTry := range []string{szLang, G_cszFallbackLang, "ko"} {
		if szText, bOk := g_mLocales[szTry][szId]; bOk {
			return szText
		}
	}
	return szId
}

// Error 는 문구 키를 들고 있다가 Error() 를 부를 때 지금 언어로 바꾸는 오류다.
// 패키지 변수로 두는 고정 오류(var ErrX = i18n.NewError(키))에 쓴다 — 시작 때 언어로 굳지 않고, errors.Is 도 그대로 쓴다.
type Error struct {
	szId string
}

// Error 는 지금 언어의 문구다.
func (pErr *Error) Error() string {
	return T(pErr.szId)
}

// NewError 는 문구 키로 고정 오류를 만든다.
func NewError(szId string) error {
	return &Error{szId: szId}
}

// Errorf 는 번역한 문구로 오류를 만든다. %w 등 fmt 동사는 문구 뒤에 붙일 때 쓴다:
// i18n.Errorf("be.app.storeOpen", errStore) → "<문구>: <오류>".
func Errorf(szId string, errWrapped error, slPairs ...any) error {
	szText := T(szId, slPairs...)
	if errWrapped == nil {
		return fmt.Errorf("%s", szText)
	}
	return fmt.Errorf("%s: %w", szText, errWrapped)
}
