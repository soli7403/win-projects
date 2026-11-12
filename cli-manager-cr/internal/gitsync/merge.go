// Package gitsync 는 서버·폴더·스니펫·볼트 항목을 git 저장소로 맞춘다 (spec/data/sync-backup.md §5).
package gitsync

import (
	"encoding/json"
	"sort"
)

// Collection 은 id → 항목 JSON 맵이다 (컬렉션 하나).
type Collection map[string]json.RawMessage

// Data 는 동기화 대상 전체다 (payload 평문, data/sync-backup.md §5.1).
type Data struct {
	M_mapGroups   Collection `json:"groups"`
	M_mapHosts    Collection `json:"hosts"`
	M_mapSnippets Collection `json:"snippets"`
	M_mapCreds    Collection `json:"creds"`
}

// NewData 는 빈 데이터를 만든다 (모든 컬렉션이 빈 맵).
func NewData() Data {
	return Data{
		M_mapGroups:   Collection{},
		M_mapHosts:    Collection{},
		M_mapSnippets: Collection{},
		M_mapCreds:    Collection{},
	}
}

// Canonical 은 JSON 을 키 순서와 무관한 문자열로 바꾼다 (동일성 비교용, sync-backup.md §5.3).
func Canonical(raw json.RawMessage) string {
	var anyValue any
	if errUnmarshal := json.Unmarshal(raw, &anyValue); errUnmarshal != nil {
		return string(raw)
	}
	bOut, errMarshal := json.Marshal(anyValue)
	if errMarshal != nil {
		return string(raw)
	}
	return string(bOut)
}

// MergeCollection 은 기준·이 PC·저장소 세 컬렉션을 3방향으로 병합한다 (sync-backup.md §5.3).
//  1. 기준에 있던 항목: 어느 한쪽이라도 지웠으면 삭제. 둘 다 있으면 이 PC 가 바꿨으면 이 PC 값,
//     아니고 저장소가 바꿨으면 저장소 값, 둘 다 안 바꿨으면 이 PC 값.
//  2. 기준에 없던 항목: 이 PC 에 있으면 이 PC 값, 없고 저장소에만 있으면 저장소 값.
func MergeCollection(base Collection, local Collection, remote Collection) Collection {
	mapOut := Collection{}
	for szId, rawBase := range base {
		rawLocal, bInLocal := local[szId]
		rawRemote, bInRemote := remote[szId]
		if !bInLocal || !bInRemote {
			continue
		}
		bLocalChanged := Canonical(rawLocal) != Canonical(rawBase)
		bRemoteChanged := Canonical(rawRemote) != Canonical(rawBase)
		switch {
		case bLocalChanged:
			mapOut[szId] = rawLocal
		case bRemoteChanged:
			mapOut[szId] = rawRemote
		default:
			mapOut[szId] = rawLocal
		}
	}
	for szId, rawLocal := range local {
		if _, bInBase := base[szId]; bInBase {
			continue
		}
		mapOut[szId] = rawLocal
	}
	for szId, rawRemote := range remote {
		if _, bInBase := base[szId]; bInBase {
			continue
		}
		if _, bInLocal := local[szId]; bInLocal {
			continue
		}
		mapOut[szId] = rawRemote
	}
	return mapOut
}

// MergeData 는 네 컬렉션을 각각 병합한다.
func MergeData(base Data, local Data, remote Data) Data {
	return Data{
		M_mapGroups:   MergeCollection(base.M_mapGroups, local.M_mapGroups, remote.M_mapGroups),
		M_mapHosts:    MergeCollection(base.M_mapHosts, local.M_mapHosts, remote.M_mapHosts),
		M_mapSnippets: MergeCollection(base.M_mapSnippets, local.M_mapSnippets, remote.M_mapSnippets),
		M_mapCreds:    MergeCollection(base.M_mapCreds, local.M_mapCreds, remote.M_mapCreds),
	}
}

// OrderIds 는 병합 결과의 id 순서를 정한다 (sync-backup.md §5.4 5번).
// 이 PC 의 현재 배열 순서를 최대한 유지하고, 새 항목은 id 오름차순으로 뒤에 붙인다.
func OrderIds(mapMerged Collection, slLocalOrder []string) []string {
	slOut := []string{}
	mapPlaced := map[string]bool{}
	for _, szId := range slLocalOrder {
		if _, bOk := mapMerged[szId]; bOk && !mapPlaced[szId] {
			slOut = append(slOut, szId)
			mapPlaced[szId] = true
		}
	}
	slRest := []string{}
	for szId := range mapMerged {
		if !mapPlaced[szId] {
			slRest = append(slRest, szId)
		}
	}
	sort.Strings(slRest)
	return append(slOut, slRest...)
}

// DataEqual 은 두 데이터가 같은지(컬렉션마다 항목 JSON 동일성) 본다.
func DataEqual(a Data, b Data) bool {
	return collectionEqual(a.M_mapGroups, b.M_mapGroups) &&
		collectionEqual(a.M_mapHosts, b.M_mapHosts) &&
		collectionEqual(a.M_mapSnippets, b.M_mapSnippets) &&
		collectionEqual(a.M_mapCreds, b.M_mapCreds)
}

// collectionEqual 은 두 컬렉션의 id 집합과 항목 내용이 같은지 본다.
func collectionEqual(a Collection, b Collection) bool {
	if len(a) != len(b) {
		return false
	}
	for szId, rawA := range a {
		rawB, bOk := b[szId]
		if !bOk || Canonical(rawA) != Canonical(rawB) {
			return false
		}
	}
	return true
}
