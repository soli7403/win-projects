package i18n

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	mCases := map[string]string{
		"ko-KR": "ko", "zh-Hans-CN": "zh", "zh-TW": "tw", "zh-Hant-HK": "tw",
		"pt-PT": "pt", "de_DE.UTF-8": "de", "it-IT": "en", "": "en",
	}
	for szTag, szWant := range mCases {
		if szGot := Match([]string{szTag}); szGot != szWant {
			t.Errorf("Match(%q)=%q, want %q", szTag, szGot, szWant)
		}
	}
	if szGot := Match([]string{"it-IT", "fr-FR"}); szGot != "fr" {
		t.Errorf("앞에서부터 처음 맞는 언어: %q", szGot)
	}
}

func TestTAndError(t *testing.T) {
	defer SetLang("ko")
	SetLang("en")
	if szGot := T("be.session.connecting", "title", "web", "target", "1.2.3.4"); szGot != "Connecting to web (1.2.3.4)..." {
		t.Fatalf("T=%q", szGot)
	}
	errSentinel := NewError("be.vault.locked")
	if errSentinel.Error() != "The app is locked" {
		t.Fatalf("Error()=%q", errSentinel.Error())
	}
	SetLang("ko")
	if errSentinel.Error() != "앱이 잠겨 있습니다" {
		t.Fatalf("언어를 바꾸면 고정 오류 문구도 바뀐다: %q", errSentinel.Error())
	}
	if !errors.Is(Errorf("be.app.storeOpen", errSentinel), errSentinel) {
		t.Fatal("Errorf 는 원인 오류를 감싼다")
	}
	if szGot := T("없는.키"); szGot != "없는.키" {
		t.Fatalf("없는 키는 키 그대로: %q", szGot)
	}
}

func TestLocalesSameKeys(t *testing.T) {
	mKo := g_mLocales["ko"]
	if len(mKo) == 0 {
		t.Fatal("ko.json 이 비었다")
	}
	for _, szLang := range G_slLangs {
		mDict, bOk := g_mLocales[szLang]
		if !bOk {
			t.Errorf("%s.json 없음", szLang)
			continue
		}
		for szKey := range mKo {
			if strings.TrimSpace(mDict[szKey]) == "" {
				t.Errorf("%s.json: %s 없음/빈 값", szLang, szKey)
			}
		}
	}
}

var g_reCallKey = regexp.MustCompile(`i18n\.(?:T|NewError|Errorf)\("([^"]+)"`)
var g_reHangul = regexp.MustCompile(`[\x{AC00}-\x{D7A3}]`)

// goSources 는 저장소의 Go 소스(테스트·tools 제외)다.
func goSources(t *testing.T) map[string]string {
	t.Helper()
	mOut := map[string]string{}
	szRoot := filepath.Join("..", "..")
	errWalk := filepath.WalkDir(szRoot, func(szPath string, entry fs.DirEntry, errIn error) error {
		if errIn != nil {
			return errIn
		}
		szName := entry.Name()
		if entry.IsDir() {
			if szName == "frontend" || szName == "tools" || szName == "build" || szName == "node_modules" || (strings.HasPrefix(szName, ".") && szPath != szRoot) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(szName, ".go") || strings.HasSuffix(szName, "_test.go") {
			return nil
		}
		arrData, errRead := os.ReadFile(szPath)
		if errRead != nil {
			return errRead
		}
		mOut[szPath] = string(arrData)
		return nil
	})
	if errWalk != nil {
		t.Fatal(errWalk)
	}
	return mOut
}

func TestGoKeysExist(t *testing.T) {
	for szPath, szSrc := range goSources(t) {
		for _, slMatch := range g_reCallKey.FindAllStringSubmatch(szSrc, -1) {
			if _, bOk := g_mLocales["ko"][slMatch[1]]; !bOk {
				t.Errorf("%s: 언어 파일에 없는 키 %q", szPath, slMatch[1])
			}
		}
	}
}

// codePart 는 한 줄에서 줄 끝 주석을 뗀 코드 부분이다(따옴표 안의 // 는 남긴다).
func codePart(szLine string) string {
	var pB strings.Builder
	var runeQuote rune
	slRunes := []rune(szLine)
	for iIndex := 0; iIndex < len(slRunes); iIndex++ {
		runeChar := slRunes[iIndex]
		if runeQuote != 0 {
			pB.WriteRune(runeChar)
			if runeChar == '\\' && runeQuote == '"' && iIndex+1 < len(slRunes) {
				iIndex++
				pB.WriteRune(slRunes[iIndex])
				continue
			}
			if runeChar == runeQuote {
				runeQuote = 0
			}
			continue
		}
		if runeChar == '/' && iIndex+1 < len(slRunes) && slRunes[iIndex+1] == '/' {
			break
		}
		if runeChar == '"' || runeChar == '`' {
			runeQuote = runeChar
		}
		pB.WriteRune(runeChar)
	}
	return pB.String()
}

func TestNoHardcodedKorean(t *testing.T) {
	for szPath, szSrc := range goSources(t) {
		for iLine, szLine := range strings.Split(strings.ReplaceAll(szSrc, "\r\n", "\n"), "\n") {
			if strings.Contains(szLine, "i18n-ok") {
				continue
			}
			if g_reHangul.MatchString(codePart(szLine)) {
				t.Errorf("%s:%d: 번역을 거치지 않은 한국어 — %s", szPath, iLine+1, strings.TrimSpace(szLine))
			}
		}
	}
}
