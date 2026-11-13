package importer

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
)

// crtValues 는 SecureCRT 세션 파일 하나의 값이다 (import.md §1.5 파일 형식).
type crtValues struct {
	mapStr   map[string]string
	mapInt   map[string]int
	mapLines map[string][]string
}

// str 은 S 형 값을 돌려준다 (없으면 빈 문자열).
func (values crtValues) str(szKey string) string {
	return values.mapStr[szKey]
}

// num 은 D 형 값을 돌려준다. 없으면 bOk=false.
func (values crtValues) num(szKey string) (int, bool) {
	iValue, bOk := values.mapInt[szKey]
	return iValue, bOk
}

// lines 는 Z 형 값(여러 줄)을 돌려준다.
func (values crtValues) lines(szKey string) []string {
	return values.mapLines[szKey]
}

// decodeCrtText 는 BOM 을 버리고, 유효한 UTF-8 이 아니면 EUC-KR 로 해석한다.
func decodeCrtText(bRaw []byte) string {
	bRaw = bytes.TrimPrefix(bRaw, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(bRaw) {
		return string(bRaw)
	}
	bDecoded, errDecode := korean.EUCKR.NewDecoder().Bytes(bRaw)
	if errDecode != nil {
		return string(bRaw)
	}
	return string(bDecoded)
}

// splitCrtLine 은 `{형}:"{키}"={값}` 줄을 나눈다. 형식이 아니면 bOk=false.
func splitCrtLine(szLine string) (byte, string, string, bool) {
	if len(szLine) < 5 || szLine[1] != ':' || szLine[2] != '"' {
		return 0, "", "", false
	}
	iEnd := strings.Index(szLine[3:], "\"=")
	if iEnd < 0 {
		return 0, "", "", false
	}
	return szLine[0], szLine[3 : 3+iEnd], szLine[3+iEnd+2:], true
}

// ParseCrtIni 는 세션 파일 텍스트를 값으로 바꾼다. B 형과 해석 못 하는 줄은 무시한다.
func ParseCrtIni(bRaw []byte) crtValues {
	values := crtValues{mapStr: map[string]string{}, mapInt: map[string]int{}, mapLines: map[string][]string{}}
	slLines := strings.Split(decodeCrtText(bRaw), "\n")
	for iIndex := 0; iIndex < len(slLines); iIndex++ {
		szLine := strings.TrimRight(slLines[iIndex], "\r")
		byKind, szKey, szValue, bOk := splitCrtLine(szLine)
		if !bOk {
			continue
		}
		switch byKind {
		case 'S':
			values.mapStr[szKey] = szValue
		case 'D':
			if uValue, errParse := strconv.ParseUint(strings.TrimSpace(szValue), 16, 32); errParse == nil {
				values.mapInt[szKey] = int(uValue)
			}
		case 'Z':
			uCount, errParse := strconv.ParseUint(strings.TrimSpace(szValue), 16, 32)
			if errParse != nil {
				continue
			}
			slBody := []string{}
			for iLine := 0; iLine < int(uCount) && iIndex+1 < len(slLines); iLine++ {
				iIndex++
				szBody := strings.TrimRight(slLines[iIndex], "\r")
				slBody = append(slBody, strings.TrimPrefix(szBody, " "))
			}
			values.mapLines[szKey] = slBody
		}
	}
	return values
}
