// Package store 는 store.json (서버·폴더·스니펫·설정·레이아웃) 을 메모리에 두고 지연 저장한다 (data/store.md, sync-backup.md §2).
package store

import (
	"encoding/json"
	"sync"
	"time"

	"dariterm/internal/jsonfile"
)

// G_cszSaveDelay 는 변경 후 디스크에 쓰기까지의 지연이다.
const G_cszSaveDelay = 300 * time.Millisecond

// g_mapSettingsDefault 는 설정 기본값이다 (data/store.md §4). 저장 값에 없는 키만 채운다.
var g_mapSettingsDefault = map[string]any{
	"language":              "", // D75: 비면 아직 고르지 않음(첫 실행 언어 선택, 그동안 OS 언어)
	"theme":                 "dark",
	"terminalTheme":         "DariTerm Dark",
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
		`rm\s+-[a-zA-Z]*r[a-zA-Z]*f`,
		`rm\s+-[a-zA-Z]*f[a-zA-Z]*r`,
		`\breboot\b`,
		`\bshutdown\b`,
		`\bhalt\b`,
		`\bpoweroff\b`,
		`mkfs`,
		`dd\s+if=`,
		`\bDROP\s+(TABLE|DATABASE)\b`,
		`\bTRUNCATE\b`,
		`systemctl\s+(stop|restart)`,
	},
	"autoReconnect":     true,
	"autoLockMinutes":   15,
	"uploadDir":         "~/.dariterm/uploads",
	"uploadCleanupDays": 7,
	"transferTempDir":   "",
	"maxVerticalSplits": 3,
	"insertPathQuote":   true,
	"sessionLog":        false,
	"bell":              false,
	"shiftEnterNewline": true,
	"rememberWindow":    true,
	"restoreLayout":     true,
	"updateCheck":       true,
	"syncRepo":          "",
	"syncAuto":          true,
	"osc52Copy":         true,
	"sidebarWidth":      260,
	"rightPanelWidth":   340,
}

// g_slArrayKeys 는 값이 항상 배열이어야 하는 최상위 키다 (data/store.md §1).
var g_slArrayKeys = []string{"groups", "hosts", "snippets", "knownHosts"}

func isArrayKey(szKey string) bool {
	for _, szArr := range g_slArrayKeys {
		if szKey == szArr {
			return true
		}
	}
	return false
}

// g_slSyncKeys 는 바뀌면 동기화 예약을 거는 최상위 키다 (sync-backup.md §2).
var g_slSyncKeys = []string{"groups", "hosts", "snippets"}

// Store 는 store.json 의 메모리 사본과 지연 저장기를 가진다.
type Store struct {
	mtxData      sync.Mutex
	szPath       string
	mapData      map[string]json.RawMessage
	timerSave    *time.Timer
	fnOnSyncKeys func()
}

// Open 은 szPath 를 읽는다. 파일이 없으면 빈 구조로 시작한다.
// 본 파일과 .bak 이 모두 깨져 있으면 오류를 돌려준다 (Q5 미정, 현재는 종료 오류로 본다).
func Open(szPath string) (*Store, error) {
	mapData := map[string]json.RawMessage{}
	if jsonfile.Exists(szPath) {
		if errRead := jsonfile.Read(szPath, &mapData); errRead != nil {
			return nil, errRead
		}
	}
	for _, szKey := range []string{"groups", "hosts", "snippets", "knownHosts"} {
		if _, bOk := mapData[szKey]; !bOk {
			mapData[szKey] = json.RawMessage("[]")
		}
	}
	mapData["version"] = json.RawMessage("1")
	pstrStore := &Store{szPath: szPath, mapData: mapData}
	// 원본 앱(BrightTerm) 이름으로 저장된 설정 값은 열 때 바꾸고 지연 저장으로 파일에도 남긴다 (store.md §4).
	if migrateLegacySettings(mapData) {
		pstrStore.mtxData.Lock()
		pstrStore.scheduleSaveLocked()
		pstrStore.mtxData.Unlock()
	}
	return pstrStore, nil
}

// SetOnSyncKeys 는 groups/hosts/snippets 가 바뀔 때 부를 콜백을 등록한다.
func (pstrStore *Store) SetOnSyncKeys(fnCallback func()) {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	pstrStore.fnOnSyncKeys = fnCallback
}

// Snapshot 은 설정을 기본값과 병합한 저장 데이터 전체를 돌려준다 (store.get 결과).
func (pstrStore *Store) Snapshot() (map[string]any, error) {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	mapOut := make(map[string]any, len(pstrStore.mapData))
	for szKey, rawValue := range pstrStore.mapData {
		var anyValue any
		if errUnmarshal := json.Unmarshal(rawValue, &anyValue); errUnmarshal != nil {
			return nil, errUnmarshal
		}
		mapOut[szKey] = anyValue
	}
	mapSettings := map[string]any{}
	if rawSettings, bOk := pstrStore.mapData["settings"]; bOk {
		if errUnmarshal := json.Unmarshal(rawSettings, &mapSettings); errUnmarshal != nil {
			mapSettings = map[string]any{}
		}
	}
	for szKey, anyDefault := range g_mapSettingsDefault {
		if _, bOk := mapSettings[szKey]; !bOk {
			mapSettings[szKey] = anyDefault
		}
	}
	mapOut["settings"] = mapSettings
	return mapOut, nil
}

// Patch 는 최상위 키 단위로 덮어쓴다. 값 nil 은 null 로 저장하되 배열 키는 빈 배열로 바꾼다. 지연 저장을 건다 (store.patch).
func (pstrStore *Store) Patch(mapPatch map[string]any) error {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	bSync := false
	for szKey, anyValue := range mapPatch {
		if szKey == "version" {
			continue
		}
		// 배열 키에 null 이 오면 빈 배열로 바꾼다 (프런트는 null 배열을 받으면 화면이 깨진다, spec/data/store.md §1).
		if anyValue == nil && isArrayKey(szKey) {
			anyValue = []any{}
		}
		bData, errMarshal := json.Marshal(anyValue)
		if errMarshal != nil {
			return errMarshal
		}
		pstrStore.mapData[szKey] = bData
		for _, szSync := range g_slSyncKeys {
			if szKey == szSync {
				bSync = true
			}
		}
	}
	pstrStore.scheduleSaveLocked()
	if bSync && pstrStore.fnOnSyncKeys != nil {
		fnCallback := pstrStore.fnOnSyncKeys
		go fnCallback()
	}
	return nil
}

// Get 은 최상위 키의 현재 값을 돌려준다. 없으면 bOk=false.
func (pstrStore *Store) Get(szKey string) (json.RawMessage, bool) {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	rawValue, bOk := pstrStore.mapData[szKey]
	return rawValue, bOk
}

// Flush 는 대기 중인 저장을 즉시 디스크에 쓴다 (종료·백업 내보내기 직전).
func (pstrStore *Store) Flush() error {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	if pstrStore.timerSave != nil {
		pstrStore.timerSave.Stop()
		pstrStore.timerSave = nil
	}
	return jsonfile.Write(pstrStore.szPath, pstrStore.mapData)
}

// scheduleSaveLocked 는 mtxData 를 잡은 상태에서 부른다. 연속 요청은 하나로 합친다.
func (pstrStore *Store) scheduleSaveLocked() {
	if pstrStore.timerSave != nil {
		pstrStore.timerSave.Stop()
	}
	pstrStore.timerSave = time.AfterFunc(G_cszSaveDelay, func() {
		pstrStore.mtxData.Lock()
		defer pstrStore.mtxData.Unlock()
		pstrStore.timerSave = nil
		_ = jsonfile.Write(pstrStore.szPath, pstrStore.mapData)
	})
}
