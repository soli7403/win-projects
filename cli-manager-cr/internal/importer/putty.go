package importer

import (
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
)

// G_cszPuttyDefaultSettings 는 건너뛸 기본 세션 이름이다.
const G_cszPuttyDefaultSettings = "Default Settings"

// DecodePuttyName 은 레지스트리 키 이름의 %XX 를 바이트로 복원하고 UTF-8 또는 EUC-KR 로 해석한다 (import.md §1).
func DecodePuttyName(szKey string) string {
	var bOut []byte
	for iIndex := 0; iIndex < len(szKey); iIndex++ {
		if szKey[iIndex] == '%' && iIndex+2 < len(szKey) {
			if bByte, errHex := hex.DecodeString(szKey[iIndex+1 : iIndex+3]); errHex == nil {
				bOut = append(bOut, bByte...)
				iIndex += 2
				continue
			}
		}
		bOut = append(bOut, szKey[iIndex])
	}
	if utf8.Valid(bOut) {
		return string(bOut)
	}
	bDecoded, errDecode := korean.EUCKR.NewDecoder().Bytes(bOut)
	if errDecode != nil {
		return string(bOut)
	}
	return string(bDecoded)
}

// puttyStr 은 값 맵에서 문자열을 읽는다. 문자열이 아니면 빈 문자열.
func puttyStr(mapValues map[string]any, szKey string) string {
	if anyValue, bOk := mapValues[szKey]; bOk {
		if szValue, bIsStr := anyValue.(string); bIsStr {
			return szValue
		}
	}
	return ""
}

// puttyInt 은 값 맵에서 정수를 읽는다 (DWORD/QWORD 또는 숫자 문자열). 없거나 해석 못 하면 bOk=false.
func puttyInt(mapValues map[string]any, szKey string) (int, bool) {
	anyValue, bFound := mapValues[szKey]
	if !bFound {
		return 0, false
	}
	switch vValue := anyValue.(type) {
	case uint32:
		return int(vValue), true
	case uint64:
		return int(vValue), true
	case int:
		return vValue, true
	case int64:
		return int(vValue), true
	case string:
		if iValue, errAtoi := strconv.Atoi(vValue); errAtoi == nil {
			return iValue, true
		}
	}
	return 0, false
}

// puttyEncoding 은 LineCodePage 값을 서버 인코딩 이름으로 바꾼다 (import.md §1 인코딩).
func puttyEncoding(szCodePage string) string {
	switch szCodePage {
	case "", "UTF-8":
		return "utf-8"
	case "CP949":
		return "cp949"
	case "EUC-KR":
		return "euc-kr"
	case "Shift_JIS":
		return "shift_jis"
	case "ISO-8859-1:1998 (Latin-1, West Europe)":
		return "latin1"
	}
	szLower := strings.ToLower(szCodePage)
	if strings.Contains(szLower, "949") || strings.Contains(szLower, "euc-kr") {
		return "cp949"
	}
	return "utf-8"
}

// puttyParity 는 SerialParity 숫자를 이름으로 바꾼다 (범위 밖은 none).
func puttyParity(iValue int) string {
	switch iValue {
	case 1:
		return "odd"
	case 2:
		return "even"
	case 3:
		return "mark"
	case 4:
		return "space"
	}
	return "none"
}

// puttyStopBits 는 SerialStopHalfbits 를 정지 비트로 바꾼다 (3→1.5, 4→2, 그 외 1).
func puttyStopBits(iValue int) float64 {
	switch iValue {
	case 3:
		return 1.5
	case 4:
		return 2
	}
	return 1
}

// puttyFlow 는 SerialFlowControl 을 흐름 제어 이름으로 바꾼다 (2→rtscts, 1→xonxoff, 그 외 none).
func puttyFlow(iValue int) string {
	switch iValue {
	case 2:
		return "rtscts"
	case 1:
		return "xonxoff"
	}
	return "none"
}

// ParsePuttyForwards 는 PortForwardings 값을 포워딩 목록으로 바꾼다. D(동적)와 해석 못 하는 항목은 건너뛴다.
func ParsePuttyForwards(szValue string) []Forward {
	slOut := []Forward{}
	for _, szItem := range strings.Split(szValue, ",") {
		szItem = strings.TrimSpace(szItem)
		if len(szItem) < 2 {
			continue
		}
		if szItem[0] == '4' || szItem[0] == '6' {
			szItem = szItem[1:]
		}
		szType := strings.ToUpper(szItem[:1])
		if szType != "L" && szType != "R" {
			continue
		}
		iEq := strings.Index(szItem, "=")
		if iEq < 0 {
			continue
		}
		szLeft := szItem[1:iEq]
		szRight := szItem[iEq+1:]
		szBind, szPort := "127.0.0.1", szLeft
		if iColon := strings.LastIndex(szLeft, ":"); iColon >= 0 {
			szBind, szPort = szLeft[:iColon], szLeft[iColon+1:]
			if szBind == "" {
				szBind = "127.0.0.1"
			}
		}
		iBindPort, errBind := strconv.Atoi(szPort)
		if errBind != nil {
			continue
		}
		iColon := strings.LastIndex(szRight, ":")
		if iColon < 0 {
			continue
		}
		szDest := szRight[:iColon]
		if szDest == "" {
			szDest = "localhost"
		}
		iDestPort, errDest := strconv.Atoi(szRight[iColon+1:])
		if errDest != nil {
			continue
		}
		slOut = append(slOut, Forward{
			M_szType:     szType,
			M_szBindHost: szBind,
			M_iBindPort:  iBindPort,
			M_szDestHost: szDest,
			M_iDestPort:  iDestPort,
		})
	}
	return slOut
}

// ParsePuttySession 은 세션 하나(이름 + 값 맵)를 후보로 바꾼다. 건너뛰면 bOk=false (import.md §1).
// szRawName 은 레지스트리 키 이름(퍼센트 인코딩 상태)이다.
func ParsePuttySession(szRawName string, mapValues map[string]any) (Candidate, bool) {
	szName := DecodePuttyName(szRawName)
	szProtocol := strings.ToLower(puttyStr(mapValues, "Protocol"))
	szHostName := puttyStr(mapValues, "HostName")
	if szName == G_cszPuttyDefaultSettings && szHostName == "" {
		return Candidate{}, false
	}
	bSerial := szProtocol == "serial"
	if !bSerial && szHostName == "" {
		return Candidate{}, false
	}
	host := HostDraft{M_szAlias: szName, M_szAuthType: "password", M_szEncoding: puttyEncoding(puttyStr(mapValues, "LineCodePage"))}
	cand := Candidate{M_szSource: G_cszSourcePutty, M_szName: szName}

	switch szProtocol {
	case "telnet":
		host.M_szProtocol = "telnet"
	case "serial":
		host.M_szProtocol = "serial"
	default:
		host.M_szProtocol = "ssh"
	}

	szUser := puttyStr(mapValues, "UserName")
	if szAt := strings.LastIndex(szHostName, "@"); szAt >= 0 {
		if szUser == "" {
			szUser = szHostName[:szAt]
		}
		szHostName = szHostName[szAt+1:]
	}
	host.M_szUsername = szUser

	if bSerial {
		return parsePuttySerial(cand, host, mapValues), true
	}

	host.M_szHost = szHostName
	iPort, bPort := puttyInt(mapValues, "PortNumber")
	if !bPort || iPort == 0 {
		iPort = 22
		if host.M_szProtocol == "telnet" {
			iPort = 23
		}
	}
	host.M_iPort = iPort

	szTerm := puttyStr(mapValues, "TerminalType")
	if szTerm == "" || szTerm == "xterm" {
		szTerm = "xterm-256color"
	}
	host.M_szTermType = szTerm

	iPing, bPing := puttyInt(mapValues, "PingIntervalSecs")
	if !bPing {
		iPing, _ = puttyInt(mapValues, "PingInterval")
	}
	if iPing == 0 {
		iPing = 30
	}
	host.M_iKeepaliveSec = iPing

	if szKeyFile := puttyStr(mapValues, "PublicKeyFile"); szKeyFile != "" {
		host.M_szAuthType = "key"
		cand.M_szKeyFile = szKeyFile
	}
	host.M_szStartupCommand = puttyStr(mapValues, "RemoteCommand")
	host.M_slForwards = ParsePuttyForwards(puttyStr(mapValues, "PortForwardings"))
	cand.M_host = host
	return cand, true
}

// parsePuttySerial 은 시리얼 세션의 설정을 채운다 (import.md §1 시리얼 세션).
func parsePuttySerial(cand Candidate, host HostDraft, mapValues map[string]any) Candidate {
	szLine := puttyStr(mapValues, "SerialLine")
	if szLine == "" {
		szLine = "COM1"
	}
	host.M_szHost = szLine
	iSpeed, bSpeed := puttyInt(mapValues, "SerialSpeed")
	if !bSpeed || iSpeed == 0 {
		iSpeed = 9600
	}
	iData, bData := puttyInt(mapValues, "SerialDataBits")
	if !bData || iData == 0 {
		iData = 8
	}
	iParity, _ := puttyInt(mapValues, "SerialParity")
	iStop, _ := puttyInt(mapValues, "SerialStopHalfbits")
	iFlow, _ := puttyInt(mapValues, "SerialFlowControl")
	iLocalEcho, bEcho := puttyInt(mapValues, "LocalEcho")
	host.M_pSerial = &Serial{
		M_szPath:        szLine,
		M_iBaudRate:     iSpeed,
		M_iDataBits:     iData,
		M_szParity:      puttyParity(iParity),
		M_fStopBits:     puttyStopBits(iStop),
		M_szFlowControl: puttyFlow(iFlow),
		M_szEnterSends:  "CR",
		M_bLocalEcho:    bEcho && iLocalEcho == 0,
	}
	cand.M_host = host
	return cand
}
