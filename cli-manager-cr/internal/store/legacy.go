package store

import (
	"encoding/json"
	"regexp"
	"strings"
)

// 참고한 원본 앱(BrightTerm)의 store.json 을 그대로 열 때 그 이름이 든 설정 값을 이 앱 이름(DariTerm)으로 바꾸는 규칙이다
// (data/store.md §4 호환 규칙).
const (
	g_cszThemePrefix       = "DariTerm "
	g_cszOriginThemePrefix = "BrightTerm "
	g_cszUploadName        = ".dariterm"
)

// g_reOriginUploadDir 는 업로드 폴더 경로 안의 원본 앱 폴더 이름 한 칸(.brightterm)이다.
var g_reOriginUploadDir = regexp.MustCompile(`(^|[\\/])\.brightterm([\\/]|$)`)

// MigrateThemeName 은 원본 앱 접두어(BrightTerm )의 터미널 테마 이름을 DariTerm 접두어로 바꾼다. 해당 없으면 그대로.
func MigrateThemeName(szName string) string {
	if strings.HasPrefix(szName, g_cszOriginThemePrefix) {
		return g_cszThemePrefix + szName[len(g_cszOriginThemePrefix):]
	}
	return szName
}

// MigrateUploadDir 는 업로드 폴더 경로의 .brightterm 폴더 이름을 .dariterm 으로 바꾼다. 해당 없으면 그대로.
func MigrateUploadDir(szDir string) string {
	return g_reOriginUploadDir.ReplaceAllString(szDir, "${1}"+g_cszUploadName+"${2}")
}

// migrateLegacySettings 는 저장 데이터의 settings 에서 원본 앱 이름 값을 바꾼다. 바뀌었으면 true.
// settings 를 해석하지 못하면 손대지 않는다.
func migrateLegacySettings(mapData map[string]json.RawMessage) bool {
	rawSettings, bOk := mapData["settings"]
	if !bOk {
		return false
	}
	mapSettings := map[string]json.RawMessage{}
	if errUnmarshal := json.Unmarshal(rawSettings, &mapSettings); errUnmarshal != nil {
		return false
	}
	bChanged := false
	for szKey, fnMigrate := range map[string]func(string) string{"terminalTheme": MigrateThemeName, "uploadDir": MigrateUploadDir} {
		rawValue, bFound := mapSettings[szKey]
		if !bFound {
			continue
		}
		var szValue string
		if errValue := json.Unmarshal(rawValue, &szValue); errValue != nil {
			continue
		}
		if szNew := fnMigrate(szValue); szNew != szValue {
			bNew, _ := json.Marshal(szNew)
			mapSettings[szKey] = bNew
			bChanged = true
		}
	}
	if !bChanged {
		return false
	}
	bSettings, errMarshal := json.Marshal(mapSettings)
	if errMarshal != nil {
		return false
	}
	mapData["settings"] = bSettings
	return true
}
