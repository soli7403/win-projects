package app

import "encoding/json"

// 세션의 색과 환경을 정한다 (spec/behavior/session.md §1 의 color·env, spec/data/store.md §6).

// G_ciGroupDepthMax 는 폴더를 따라 올라가는 최대 단계다 (순환하는 데이터에서도 멈춘다).
const G_ciGroupDepthMax = 32

// G_cszEnvNone 은 환경이 지정되지 않았음을 뜻하는 값이다.
const G_cszEnvNone = "none"

// g_mapEnvColor 는 환경의 색이다. none 은 색을 주지 않는다.
var g_mapEnvColor = map[string]string{
	"prod":   "#ef4444",
	"stage":  "#f59e0b",
	"perf":   "#a855f7",
	"dev":    "#22c55e",
	"device": "#3b82f6",
}

// groupRecord 는 저장된 폴더에서 색·환경을 정하는 데 쓰는 필드다 (data/store.md §2).
type groupRecord struct {
	M_szId       string `json:"id"`
	M_szParentId string `json:"parentId"`
	M_szColor    string `json:"color"`
	M_szEnv      string `json:"env"`
}

// loadGroups 는 저장된 폴더 목록을 읽는다. 없거나 읽을 수 없으면 빈 목록.
func (pService *Service) loadGroups() []groupRecord {
	rawGroups, bOk := pService.pStore.Get("groups")
	if !bOk {
		return nil
	}
	var slGroups []groupRecord
	if errUnmarshal := json.Unmarshal(rawGroups, &slGroups); errUnmarshal != nil {
		return nil
	}
	return slGroups
}

// resolveLook 은 저장된 서버의 세션 색과 환경을 정한다.
//   - 색: 서버 색 → 서버 환경색 → 소속 폴더부터 위로 올라가며 처음 나오는 (폴더 색 → 폴더 환경색). 없으면 빈 값.
//   - 환경: 서버 환경 → 소속 폴더부터 위로 올라가며 처음 나오는 폴더 환경 → none.
//
// 환경 none(또는 빈 값·모르는 값)은 지정하지 않은 것으로 본다.
func resolveLook(record hostRecord, slGroups []groupRecord) (string, string) {
	szColor, szEnv := record.M_szColor, ""
	if g_mapEnvColor[record.M_szEnv] != "" {
		szEnv = record.M_szEnv
		if szColor == "" {
			szColor = g_mapEnvColor[szEnv]
		}
	}
	szGroupId := record.M_szGroupId
	for iDepth := 0; iDepth < G_ciGroupDepthMax && szGroupId != "" && (szColor == "" || szEnv == ""); iDepth++ {
		pGroup := findGroup(slGroups, szGroupId)
		if pGroup == nil {
			break
		}
		szGroupEnvColor := g_mapEnvColor[pGroup.M_szEnv]
		if szColor == "" {
			szColor = pGroup.M_szColor
			if szColor == "" {
				szColor = szGroupEnvColor
			}
		}
		if szEnv == "" && szGroupEnvColor != "" {
			szEnv = pGroup.M_szEnv
		}
		szGroupId = pGroup.M_szParentId
	}
	if szEnv == "" {
		szEnv = G_cszEnvNone
	}
	return szColor, szEnv
}

func findGroup(slGroups []groupRecord, szId string) *groupRecord {
	for iIdx := range slGroups {
		if slGroups[iIdx].M_szId == szId {
			return &slGroups[iIdx]
		}
	}
	return nil
}
