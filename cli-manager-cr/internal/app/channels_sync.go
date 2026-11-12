package app

import (
	"encoding/json"

	"termgo/internal/gitsync"
	"termgo/internal/vault"
)

// storeLocal 은 이 PC 의 store 배열과 볼트 항목을 동기화 데이터로 읽고 쓰는 어댑터다 (gitsync.LocalStore).
type storeLocal struct {
	pService *Service
}

// LoadData 는 서버·폴더·스니펫·볼트 항목을 동기화 데이터와 배열 순서로 읽는다.
func (pLocal storeLocal) LoadData() (gitsync.Data, gitsync.Orders, error) {
	data := gitsync.NewData()
	orders := gitsync.Orders{}
	for _, pair := range []struct {
		szKey   string
		mapOut  gitsync.Collection
		pSlOrds *[]string
	}{
		{"groups", data.M_mapGroups, &orders.M_slGroups},
		{"hosts", data.M_mapHosts, &orders.M_slHosts},
		{"snippets", data.M_mapSnippets, &orders.M_slSnippets},
	} {
		slItems, errLoad := loadMapArray(pLocal.pService, pair.szKey)
		if errLoad != nil {
			return data, orders, errLoad
		}
		for _, mapItem := range slItems {
			szId, _ := mapItem["id"].(string)
			if szId == "" {
				continue
			}
			rawItem, errMarshal := json.Marshal(mapItem)
			if errMarshal != nil {
				return data, orders, errMarshal
			}
			pair.mapOut[szId] = rawItem
			*pair.pSlOrds = append(*pair.pSlOrds, szId)
		}
	}
	slItems, errItems := pLocal.pService.pVault.Items()
	if errItems != nil {
		// 볼트가 없으면 항목도 없다.
		return data, orders, nil
	}
	for _, record := range slItems {
		rawItem, errMarshal := json.Marshal(record)
		if errMarshal != nil {
			return data, orders, errMarshal
		}
		data.M_mapCreds[record.M_szId] = rawItem
		orders.M_slCreds = append(orders.M_slCreds, record.M_szId)
	}
	return data, orders, nil
}

// SaveData 는 병합 결과를 store 배열과 볼트 항목에 반영하고 즉시 저장한다.
func (pLocal storeLocal) SaveData(data gitsync.Data, orders gitsync.Orders) error {
	slGroups, errGroups := orderedMaps(data.M_mapGroups, orders.M_slGroups)
	if errGroups != nil {
		return errGroups
	}
	slHosts, errHosts := orderedMaps(data.M_mapHosts, orders.M_slHosts)
	if errHosts != nil {
		return errHosts
	}
	slSnippets, errSnippets := orderedMaps(data.M_mapSnippets, orders.M_slSnippets)
	if errSnippets != nil {
		return errSnippets
	}
	if errPatch := pLocal.pService.pStore.Patch(map[string]any{"groups": slGroups, "hosts": slHosts, "snippets": slSnippets}); errPatch != nil {
		return errPatch
	}
	if errFlush := pLocal.pService.pStore.Flush(); errFlush != nil {
		return errFlush
	}
	slIds := gitsync.OrderIds(data.M_mapCreds, orders.M_slCreds)
	slRecords := []vault.ItemRecord{}
	for _, szId := range slIds {
		var record vault.ItemRecord
		if errJson := json.Unmarshal(data.M_mapCreds[szId], &record); errJson != nil {
			return errJson
		}
		slRecords = append(slRecords, record)
	}
	return pLocal.pService.pVault.ReplaceItems(slRecords)
}

// orderedMaps 는 컬렉션을 배열 순서(없는 것은 id 순)로 맵 배열을 만든다.
func orderedMaps(mapItems gitsync.Collection, slOrder []string) ([]map[string]any, error) {
	slOut := []map[string]any{}
	for _, szId := range gitsync.OrderIds(mapItems, slOrder) {
		var mapItem map[string]any
		if errJson := json.Unmarshal(mapItems[szId], &mapItem); errJson != nil {
			return nil, errJson
		}
		slOut = append(slOut, mapItem)
	}
	return slOut, nil
}

// askPassword 는 "저장소 볼트 받기"의 마스터 비밀번호를 묻는다 (ui.request password, 저장 허용 안 함).
func (pService *Service) askPassword() (string, bool) {
	rawAnswer, bOk := pService.pRouter.Asker().Ask("password", map[string]any{
		"sessionTitle": "git 동기화",
		"prompt":       "저장소 볼트의 마스터 비밀번호",
		"username":     "",
		"allowSave":    false,
	})
	if !bOk {
		return "", false
	}
	var answer struct {
		Password string `json:"password"`
	}
	if errJson := json.Unmarshal(rawAnswer, &answer); errJson != nil {
		return "", false
	}
	return answer.Password, true
}

// syncRepoSetting 은 설정의 git 동기화 저장소 주소와 자동 동기화 여부를 읽는다.
func (pService *Service) syncRepoSetting() (string, bool) {
	mapSettings, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return "", false
	}
	mapSet, _ := mapSettings["settings"].(map[string]any)
	szRepo, _ := mapSet["syncRepo"].(string)
	bAuto, bOk := mapSet["syncAuto"].(bool)
	if !bOk {
		bAuto = true
	}
	return szRepo, bAuto
}

// registerSync 는 sync.* 채널을 등록한다 (spec/ipc.md §2 가져오기·동기화).
func (pService *Service) registerSync() {
	pRouter := pService.pRouter
	pRouter.Handle("sync.run", func(slArgs []json.RawMessage) (any, error) {
		szMode := ""
		if len(slArgs) > 0 {
			szMode, _ = argString(slArgs, 0)
		}
		szRepo, _ := pService.syncRepoSetting()
		res := pService.pSync.syncer.Run(szRepo, szMode)
		pService.finishSync(res)
		return res, nil
	})
	pRouter.Handle("sync.status", func(slArgs []json.RawMessage) (any, error) {
		szRepo, _ := pService.syncRepoSetting()
		// last 는 항상 객체다. 동기화 전에는 {"state":""} (spec/ipc.md sync.status, null 금지).
		var pLast any = map[string]any{"state": ""}
		if last := pService.pSync.syncer.Last(); last != nil {
			pLast = last
		}
		return map[string]any{
			"configured":   szRepo != "",
			"gitAvailable": gitsync.GitAvailable(),
			"last":         pLast,
		}, nil
	})
}
