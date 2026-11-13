package store

import (
	"encoding/json"

	"dariterm/internal/jsonfile"
)

// ReplaceAll 은 저장 데이터 전체를 바꾸고 즉시 저장한다 (백업 가져오기, data/sync-backup.md §3).
// 배열 키가 없으면 빈 배열로 채우고 version 은 1 로 둔다. 동기화 예약은 걸지 않는다.
func (pstrStore *Store) ReplaceAll(mapNew map[string]any) error {
	pstrStore.mtxData.Lock()
	defer pstrStore.mtxData.Unlock()
	mapData := map[string]json.RawMessage{}
	for szKey, anyValue := range mapNew {
		if szKey == "version" {
			continue
		}
		bData, errMarshal := json.Marshal(anyValue)
		if errMarshal != nil {
			return errMarshal
		}
		mapData[szKey] = bData
	}
	for _, szKey := range []string{"groups", "hosts", "snippets", "knownHosts"} {
		if _, bOk := mapData[szKey]; !bOk {
			mapData[szKey] = json.RawMessage("[]")
		}
	}
	mapData["version"] = json.RawMessage("1")
	pstrStore.mapData = mapData
	if pstrStore.timerSave != nil {
		pstrStore.timerSave.Stop()
		pstrStore.timerSave = nil
	}
	return jsonfile.Write(pstrStore.szPath, pstrStore.mapData)
}
