//go:build windows

package importer

import (
	"golang.org/x/sys/windows/registry"
)

// g_cszPuttySessionsKey 는 PuTTY 세션이 있는 레지스트리 경로다 (현재 사용자 아래).
const g_cszPuttySessionsKey = `Software\SimonTatham\PuTTY\Sessions`

// 레지스트리 값 형식 번호 (REG_SZ, REG_EXPAND_SZ, REG_DWORD, REG_QWORD).
const (
	g_iRegSz       = 1
	g_iRegExpandSz = 2
	g_iRegDword    = 4
	g_iRegQword    = 11
)

// ReadPuttySessions 는 레지스트리의 PuTTY 세션을 읽는다. 키가 없으면 빈 맵이다.
func ReadPuttySessions() (map[string]map[string]any, error) {
	mapOut := map[string]map[string]any{}
	pRoot, errOpen := registry.OpenKey(registry.CURRENT_USER, g_cszPuttySessionsKey, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if errOpen != nil {
		return mapOut, nil
	}
	defer pRoot.Close()
	slNames, errNames := pRoot.ReadSubKeyNames(-1)
	if errNames != nil {
		return mapOut, errNames
	}
	for _, szName := range slNames {
		pSession, errSession := registry.OpenKey(pRoot, szName, registry.QUERY_VALUE)
		if errSession != nil {
			continue
		}
		mapOut[szName] = readPuttyValues(pSession)
		pSession.Close()
	}
	return mapOut, nil
}

// readPuttyValues 는 세션 키의 문자열·정수 값을 맵으로 읽는다 (다른 형식은 건너뛴다).
func readPuttyValues(pKey registry.Key) map[string]any {
	mapValues := map[string]any{}
	slNames, errNames := pKey.ReadValueNames(-1)
	if errNames != nil {
		return mapValues
	}
	for _, szName := range slNames {
		_, iType, errValue := pKey.GetValue(szName, nil)
		if errValue != nil {
			continue
		}
		switch iType {
		case g_iRegSz, g_iRegExpandSz:
			if szValue, _, errStr := pKey.GetStringValue(szName); errStr == nil {
				mapValues[szName] = szValue
			}
		case g_iRegDword, g_iRegQword:
			if iValue, _, errInt := pKey.GetIntegerValue(szName); errInt == nil {
				mapValues[szName] = iValue
			}
		}
	}
	return mapValues
}
