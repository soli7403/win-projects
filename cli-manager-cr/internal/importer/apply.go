package importer

import (
	"fmt"
	"os"
	"strings"

	"dariterm/internal/uid"
)

// KeyStore 는 가져오기가 개인 키를 볼트에 넣을 때 쓰는 인터페이스다 (app 이 볼트로 구현한다).
type KeyStore interface {
	// Unlocked 는 볼트가 열려 있는지 본다. 잠겨 있으면 키를 복사하지 않는다 (acceptance 04 I8).
	Unlocked() bool
	// SaveKey 는 개인 키 항목을 저장하고 항목 id 를 돌려준다.
	SaveKey(szName string, szUsername string, szPrivateKey string) (string, error)
}

// ApplyResult 는 import.apply 의 결과다 (spec/import.md §4).
type ApplyResult struct {
	M_iHosts    int      `json:"hosts"`
	M_iKeys     int      `json:"keys"`
	M_slSkipped []string `json:"skipped"`
}

// ApplyOutput 은 적용 뒤 저장할 폴더·서버 목록이다.
type ApplyOutput struct {
	Groups []map[string]any
	Hosts  []map[string]any
	Result ApplyResult
}

// G_cszDefaultGroupEnv 는 가져오기로 만든 폴더의 환경 값이다.
const G_cszDefaultGroupEnv = "none"

// envForCandidate 는 후보 이름 접두어와 프로토콜로 환경을 정한다 (import.md §4 환경).
func envForCandidate(szName string, szProtocol string) string {
	szLower := strings.ToLower(szName)
	for _, szPrefix := range []string{"prod", "prd", "live", "운영"} {
		if strings.HasPrefix(szLower, szPrefix) {
			return "prod"
		}
	}
	for _, szPrefix := range []string{"perf", "성능"} {
		if strings.HasPrefix(szLower, szPrefix) {
			return "perf"
		}
	}
	for _, szPrefix := range []string{"stg", "stage", "staging", "func", "기능"} {
		if strings.HasPrefix(szLower, szPrefix) {
			return "stage"
		}
	}
	for _, szPrefix := range []string{"dev", "test", "개발"} {
		if strings.HasPrefix(szLower, szPrefix) {
			return "dev"
		}
	}
	if szProtocol == "serial" {
		return "device"
	}
	return "none"
}

// findTopGroup 은 이름이 같은 최상위 폴더 id 를 찾는다.
func findTopGroup(slGroups []map[string]any, szName string) string {
	for _, mapGroup := range slGroups {
		if mapGroup["parentId"] != nil {
			continue
		}
		if szValue, _ := mapGroup["name"].(string); szValue == szName {
			szId, _ := mapGroup["id"].(string)
			return szId
		}
	}
	return ""
}

// ApplyCandidates 는 선택된 후보를 폴더·서버 목록에 추가한다 (import.md §4). 중복은 검사하지 않는다.
func ApplyCandidates(slCands []Candidate, szFolderName string, slGroups []map[string]any, slHosts []map[string]any, keys KeyStore) (ApplyOutput, error) {
	out := ApplyOutput{Groups: slGroups, Hosts: slHosts, Result: ApplyResult{M_slSkipped: []string{}}}
	szGroupId := ""
	if szFolderName != "" {
		szGroupId = findTopGroup(out.Groups, szFolderName)
		if szGroupId == "" {
			szNewId, errId := uid.New()
			if errId != nil {
				return out, errId
			}
			out.Groups = append(out.Groups, map[string]any{
				"id":       szNewId,
				"parentId": nil,
				"name":     szFolderName,
				"env":      G_cszDefaultGroupEnv,
				"sort":     len(slGroups),
			})
			szGroupId = szNewId
		}
	}

	slNewIds := []string{}
	slNewHosts := []map[string]any{}
	mapJumpPending := map[string]string{}
	for _, cand := range slCands {
		szHostGroup, errPath := ensureFolderPath(&out.Groups, szGroupId, cand.M_slFolderPath)
		if errPath != nil {
			return out, errPath
		}
		mapHost, szHostId, errBuild := buildHost(cand, szHostGroup, len(out.Hosts)+len(slNewHosts), keys, &out.Result)
		if errBuild != nil {
			return out, errBuild
		}
		slNewHosts = append(slNewHosts, mapHost)
		slNewIds = append(slNewIds, szHostId)
		if cand.M_szJumpVia != "" {
			mapJumpPending[szHostId] = cand.M_szJumpVia
		}
		out.Result.M_iHosts++
	}

	// 점프 호스트: 이번에 만든 서버 중 같은 주소를 먼저, 없으면 기존 서버 전체에서 찾는다.
	for iIndex, mapHost := range slNewHosts {
		szSelf, _ := mapHost["id"].(string)
		szJump, bPending := mapJumpPending[szSelf]
		if !bPending {
			continue
		}
		szTarget := findHostByAddress(slNewHosts, szJump, szSelf)
		if szTarget == "" {
			szTarget = findHostByAddress(out.Hosts, szJump, szSelf)
		}
		if szTarget == "" {
			name, _ := mapHost["alias"].(string)
			out.Result.M_slSkipped = append(out.Result.M_slSkipped, fmt.Sprintf("%s: 경유 서버 %s 를 찾지 못해 직접 연결로 등록", name, szJump))
			continue
		}
		slNewHosts[iIndex]["jumpHostId"] = szTarget
	}

	out.Hosts = append(out.Hosts, slNewHosts...)
	return out, nil
}

// findChildGroup 은 부모(빈 문자열이면 최상위) 아래 이름이 같은 폴더 id 를 찾는다.
func findChildGroup(slGroups []map[string]any, szParentId string, szName string) string {
	if szParentId == "" {
		return findTopGroup(slGroups, szName)
	}
	for _, mapGroup := range slGroups {
		szParent, _ := mapGroup["parentId"].(string)
		if szValue, _ := mapGroup["name"].(string); szParent == szParentId && szValue == szName {
			szId, _ := mapGroup["id"].(string)
			return szId
		}
	}
	return ""
}

// ensureFolderPath 는 szBaseId 아래에 하위 폴더 경로를 찾거나 만들고 마지막 폴더 id 를 돌려준다 (import.md §4 하위 폴더).
// 빈 이름은 건너뛴다. 새 폴더의 정렬 키는 그때의 폴더 수다.
func ensureFolderPath(pGroups *[]map[string]any, szBaseId string, slPath []string) (string, error) {
	szCurrent := szBaseId
	for _, szName := range slPath {
		szName = strings.TrimSpace(szName)
		if szName == "" {
			continue
		}
		if szFound := findChildGroup(*pGroups, szCurrent, szName); szFound != "" {
			szCurrent = szFound
			continue
		}
		szNewId, errId := uid.New()
		if errId != nil {
			return "", errId
		}
		*pGroups = append(*pGroups, map[string]any{
			"id":       szNewId,
			"parentId": nilIfEmpty(szCurrent),
			"name":     szName,
			"env":      G_cszDefaultGroupEnv,
			"sort":     len(*pGroups),
		})
		szCurrent = szNewId
	}
	return szCurrent, nil
}

// findHostByAddress 는 주소가 같은 서버 id 를 찾는다 (자기 자신 제외).
func findHostByAddress(slHosts []map[string]any, szAddress string, szSelf string) string {
	for _, mapHost := range slHosts {
		szId, _ := mapHost["id"].(string)
		szHost, _ := mapHost["host"].(string)
		if szHost == szAddress && szId != szSelf {
			return szId
		}
	}
	return ""
}

// buildHost 는 후보 하나를 서버 항목으로 바꾼다. 키 파일이 있고 볼트가 열려 있으면 볼트에 저장한다.
func buildHost(cand Candidate, szGroupId string, iSort int, keys KeyStore, pResult *ApplyResult) (map[string]any, string, error) {
	szId, errId := uid.New()
	if errId != nil {
		return nil, "", errId
	}
	host := cand.M_host
	szAuth := host.M_szAuthType
	var pszCredId any
	if cand.M_szKeyFile != "" {
		bKey, errRead := os.ReadFile(cand.M_szKeyFile)
		if errRead != nil {
			pResult.M_slSkipped = append(pResult.M_slSkipped, fmt.Sprintf("%s: 키 파일을 읽지 못함 (%s)", cand.M_szName, cand.M_szKeyFile))
		} else if keys != nil && keys.Unlocked() {
			szCred, errSave := keys.SaveKey(cand.M_szName+" 키", host.M_szUsername, string(bKey))
			if errSave != nil {
				return nil, "", errSave
			}
			pszCredId = szCred
			szAuth = "key"
			pResult.M_iKeys++
		}
	}
	if szAuth == "key" && pszCredId == nil {
		// 키를 볼트에 넣지 못했으면 OS 에이전트 인증으로 둔다 (acceptance 04 I8).
		szAuth = "agent"
	}
	if szAuth == "" {
		szAuth = "password"
	}
	szProtocol := host.M_szProtocol
	if szProtocol == "" {
		szProtocol = "ssh"
	}
	iPort := host.M_iPort
	if iPort == 0 && szProtocol != "serial" {
		iPort = 22
	}
	szAlias := host.M_szAlias
	if szAlias == "" {
		szAlias = cand.M_szName
	}
	szEncoding := host.M_szEncoding
	if szEncoding == "" {
		szEncoding = "utf-8"
	}
	szTerm := host.M_szTermType
	if szTerm == "" {
		szTerm = "xterm-256color"
	}
	iKeepalive := host.M_iKeepaliveSec
	if iKeepalive == 0 {
		iKeepalive = 30
	}
	slForwards := []map[string]any{}
	for _, fwd := range host.M_slForwards {
		szFwdId, errFwd := uid.New()
		if errFwd != nil {
			return nil, "", errFwd
		}
		slForwards = append(slForwards, map[string]any{
			"id":       szFwdId,
			"type":     fwd.M_szType,
			"bindHost": fwd.M_szBindHost,
			"bindPort": fwd.M_iBindPort,
			"destHost": fwd.M_szDestHost,
			"destPort": fwd.M_iDestPort,
		})
	}
	mapHost := map[string]any{
		"id":           szId,
		"groupId":      nilIfEmpty(szGroupId),
		"alias":        szAlias,
		"protocol":     szProtocol,
		"host":         host.M_szHost,
		"port":         iPort,
		"username":     host.M_szUsername,
		"authType":     szAuth,
		"credentialId": pszCredId,
		"jumpHostId":   nil,
		"color":        "",
		"env":          envForCandidate(cand.M_szName, szProtocol),
		"tags":         []string{cand.M_szSource},
		"encoding":     szEncoding,
		"termType":     szTerm,
		"keepSession":  false,
		"tmuxName":     "",
		"keepaliveSec": iKeepalive,
		"forwards":     slForwards,
		"local":        nil,
		"favorite":     false,
		"sort":         iSort,
	}
	if host.M_szStartupCommand != "" {
		mapHost["startupCommand"] = host.M_szStartupCommand
	}
	if host.M_szNotes != "" {
		mapHost["notes"] = host.M_szNotes
	}
	if host.M_pSerial != nil {
		mapHost["serial"] = map[string]any{
			"path":        host.M_pSerial.M_szPath,
			"baudRate":    host.M_pSerial.M_iBaudRate,
			"dataBits":    host.M_pSerial.M_iDataBits,
			"parity":      host.M_pSerial.M_szParity,
			"stopBits":    host.M_pSerial.M_fStopBits,
			"flowControl": host.M_pSerial.M_szFlowControl,
			"enterSends":  host.M_pSerial.M_szEnterSends,
			"localEcho":   host.M_pSerial.M_bLocalEcho,
		}
	}
	return mapHost, szId, nil
}

// nilIfEmpty 는 빈 문자열이면 nil(JSON null)을 돌려준다.
func nilIfEmpty(szValue string) any {
	if szValue == "" {
		return nil
	}
	return szValue
}
