package app

import (
	"dariterm/internal/i18n"
	"encoding/json"
)

// argString 은 slArgs[iIndex] 를 문자열로 해석한다. 없거나 형식이 다르면 오류.
func argString(slArgs []json.RawMessage, iIndex int) (string, error) {
	if iIndex >= len(slArgs) {
		return "", i18n.Errorf("be.arg.missing", nil, "n", iIndex+1)
	}
	var szValue string
	if errUnmarshal := json.Unmarshal(slArgs[iIndex], &szValue); errUnmarshal != nil {
		return "", i18n.Errorf("be.arg.notString", nil, "n", iIndex+1)
	}
	return szValue, nil
}

// argBool 은 slArgs[iIndex] 를 불리언으로 해석한다.
func argBool(slArgs []json.RawMessage, iIndex int) (bool, error) {
	if iIndex >= len(slArgs) {
		return false, i18n.Errorf("be.arg.missing", nil, "n", iIndex+1)
	}
	var bValue bool
	if errUnmarshal := json.Unmarshal(slArgs[iIndex], &bValue); errUnmarshal != nil {
		return false, i18n.Errorf("be.arg.notBool", nil, "n", iIndex+1)
	}
	return bValue, nil
}

// argInto 은 slArgs[iIndex] 를 임의 구조체로 해석한다.
func argInto(slArgs []json.RawMessage, iIndex int, anyOut any) error {
	if iIndex >= len(slArgs) {
		return i18n.Errorf("be.arg.missing", nil, "n", iIndex+1)
	}
	if errUnmarshal := json.Unmarshal(slArgs[iIndex], anyOut); errUnmarshal != nil {
		return i18n.Errorf("be.arg.badFormat", nil, "n", iIndex+1)
	}
	return nil
}
