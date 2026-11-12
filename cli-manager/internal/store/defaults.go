package store

import (
	"encoding/json"
	"strings"
)

// g_mDefaultSettings 는 v1.1.0 shared/types.ts 의 DEFAULT_SETTINGS (Windows 기준).
var g_mDefaultSettings = map[string]any{
	"theme":                 "dark",
	"terminalTheme":         "TermGo Dark",
	"fontFamily":            "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
	"fontSize":              14,
	"lineHeight":            1.15,
	"cursorStyle":           "block",
	"cursorBlink":           true,
	"scrollback":            10000,
	"copyOnSelect":          true,
	"rightClickPaste":       true,
	"ctrlVPaste":            true,
	"confirmMultilinePaste": true,
	"guardDangerousOnProd":  true,
	"dangerousPatterns": []string{
		`rm\s+-[a-zA-Z]*r[a-zA-Z]*f`, `rm\s+-[a-zA-Z]*f[a-zA-Z]*r`, `\breboot\b`, `\bshutdown\b`,
		`\bhalt\b`, `\bpoweroff\b`, `mkfs`, `dd\s+if=`, `\bDROP\s+(TABLE|DATABASE)\b`, `\bTRUNCATE\b`,
		`systemctl\s+(stop|restart)`,
	},
	"autoReconnect":     true,
	"autoLockMinutes":   15,
	"uploadDir":         "~/.termgo/uploads",
	"uploadCleanupDays": 7,
	"transferTempDir":   "",
	"insertPathQuote":   true,
	"sessionLog":        false,
	"bell":              false,
	"shiftEnterNewline": true,
	"osc52Copy":         true,
	"sidebarWidth":      260,
	"rightPanelWidth":   340,
}

// mergeSettings 는 저장된 settings 위에 없는 키만 기본값으로 채운다.
func mergeSettings(raw json.RawMessage) json.RawMessage {
	mOut := make(map[string]json.RawMessage, len(g_mDefaultSettings))
	for k, v := range g_mDefaultSettings {
		arr, _ := json.Marshal(v)
		mOut[k] = arr
	}
	var mSaved map[string]json.RawMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &mSaved)
	}
	for k, v := range mSaved {
		mOut[k] = v
	}
	// v1.1.0 의 테마 이름을 새 이름으로 옮긴다
	var szTheme string
	if json.Unmarshal(mOut["terminalTheme"], &szTheme) == nil && strings.HasPrefix(szTheme, "BrightTerm ") {
		mOut["terminalTheme"], _ = json.Marshal("TermGo " + strings.TrimPrefix(szTheme, "BrightTerm "))
	}
	arr, _ := json.Marshal(mOut)
	return arr
}
