package importer

import (
	"dariterm/internal/i18n"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SecureCRT 파일 이름 규칙 (import.md §1.5 폴더 구조).
const (
	g_cszCrtFolderData = "__folderdata__.ini"
	g_cszCrtDefault    = "default.ini"
	g_cszCrtSessions   = "Sessions"
)

// CrtSession 은 세션 파일 하나를 읽은 결과다. 이름은 파일 이름(확장자 제외), 폴더는 세션 폴더 기준 하위 폴더 이름들이다.
type CrtSession struct {
	M_szName   string
	M_slFolder []string
	M_values   crtValues
}

// crtEnvPattern 은 %이름% 또는 ${이름} 환경 변수 자리다.
var crtEnvPattern = regexp.MustCompile(`%([A-Za-z0-9_]+)%|\$\{([A-Za-z0-9_]+)\}`)

// expandCrtPath 는 키 파일 경로의 `::` 뒤를 떼고 환경 변수와 ${VDS_CONFIG_PATH} 를 바꾼다.
func expandCrtPath(szValue string, szConfigDir string) string {
	if iSep := strings.Index(szValue, "::"); iSep >= 0 {
		szValue = szValue[:iSep]
	}
	return crtEnvPattern.ReplaceAllStringFunc(szValue, func(szMatch string) string {
		slParts := crtEnvPattern.FindStringSubmatch(szMatch)
		szName := slParts[1] + slParts[2]
		if szName == "VDS_CONFIG_PATH" {
			return szConfigDir
		}
		if szEnv, bOk := os.LookupEnv(szName); bOk {
			return szEnv
		}
		return szMatch
	})
}

// crtEncoding 은 Output Transformer Name 을 서버 인코딩 이름으로 바꾼다.
func crtEncoding(szName string) string {
	szLower := strings.ToLower(szName)
	switch {
	case strings.Contains(szLower, "949"), strings.Contains(szLower, "euc-kr"), strings.Contains(szLower, "korean"):
		return "cp949"
	case strings.Contains(szLower, "euc-jp"), strings.Contains(szLower, "eucjp"), strings.Contains(szLower, "japanese") && strings.Contains(szLower, "euc"):
		return "euc-jp" // "japanese" 만 보는 아래 규칙보다 먼저 본다
	case strings.Contains(szLower, "shift"), strings.Contains(szLower, "japanese"):
		return "shift_jis"
	case strings.Contains(szLower, "gbk"), strings.Contains(szLower, "936"), strings.Contains(szLower, "gb2312"):
		return "gbk"
	case strings.Contains(szLower, "big5"), strings.Contains(szLower, "950"):
		return "big5"
	case strings.Contains(szLower, "8859-1"), strings.Contains(szLower, "latin"):
		return "latin1"
	}
	return "utf-8"
}

// crtStopBits 는 Stop Bits 값을 정지 비트로 바꾼다 (0→1, 1→1.5, 2→2).
func crtStopBits(iValue int) float64 {
	switch iValue {
	case 1:
		return 1.5
	case 2:
		return 2
	}
	return 1
}

// ParseCrtForwards 는 포워딩 표 줄(`이름|로컬|플래그|대상 호스트|대상 포트|…`)을 포워딩 목록으로 바꾼다.
func ParseCrtForwards(slLines []string, szType string) []Forward {
	slOut := []Forward{}
	for _, szLine := range slLines {
		slFields := strings.Split(strings.TrimSpace(szLine), "|")
		if len(slFields) < 5 {
			continue
		}
		szBind, szPort := "127.0.0.1", strings.TrimSpace(slFields[1])
		if iColon := strings.LastIndex(szPort, ":"); iColon >= 0 {
			if szAddr := szPort[:iColon]; szAddr != "" {
				szBind = szAddr
			}
			szPort = szPort[iColon+1:]
		}
		iBindPort, errBind := strconv.Atoi(szPort)
		iDestPort, errDest := strconv.Atoi(strings.TrimSpace(slFields[4]))
		if errBind != nil || errDest != nil {
			continue
		}
		szDest := strings.TrimSpace(slFields[3])
		if szDest == "" {
			szDest = "localhost"
		}
		slOut = append(slOut, Forward{M_szType: szType, M_szBindHost: szBind, M_iBindPort: iBindPort, M_szDestHost: szDest, M_iDestPort: iDestPort})
	}
	return slOut
}

// crtProtocol 은 Protocol Name 을 프로토콜과 포트 키로 바꾼다. 지원하지 않으면 bOk=false.
func crtProtocol(szName string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(szName)) {
	case "", "ssh2":
		return "ssh", "[SSH2] Port", true
	case "ssh1":
		return "ssh", "[SSH1] Port", true
	case "telnet":
		return "telnet", "Port", true
	case "serial":
		return "serial", "", true
	}
	return "", "", false
}

// ParseCrtSession 은 세션 하나를 후보로 바꾼다. 건너뛰면 bOk=false (import.md §1.5).
// szGlobalKey 는 SSH2.ini 의 전역 키 파일 경로, szConfigDir 은 설정 폴더다. 경유 서버는 ResolveCrtJumps 가 채운다.
func ParseCrtSession(session CrtSession, szGlobalKey string, szConfigDir string) (Candidate, bool) {
	values := session.M_values
	szProtocol, szPortKey, bOk := crtProtocol(values.str("Protocol Name"))
	if !bOk {
		return Candidate{}, false
	}
	szHost := strings.TrimSpace(values.str("Hostname"))
	if szProtocol != "serial" && szHost == "" {
		return Candidate{}, false
	}
	host := HostDraft{
		M_szAlias:    session.M_szName,
		M_szProtocol: szProtocol,
		M_szUsername: values.str("Username"),
		M_szAuthType: "password",
		M_szEncoding: crtEncoding(values.str("Output Transformer Name")),
	}
	cand := Candidate{M_szSource: G_cszSourceSecureCrt, M_szName: session.M_szName, M_slFolderPath: append([]string{}, session.M_slFolder...)}
	if szNotes := strings.TrimSpace(strings.Join(values.lines("Description"), "\n")); szNotes != "" {
		host.M_szNotes = szNotes
	}
	// 로그온 동작(Logon Actions)의 Expect/Send 는 저장 형식을 확인하지 못해 옮기지 않는다 — 켜져 있으면 알린다 (D150).
	// 형식(Z:"Login Script V4" 의 줄)을 표본으로 확인하면 host.M_slLoginScript 로 옮긴다.
	if crtHasLoginScript(values) {
		cand.M_szNote = joinNote(cand.M_szNote, i18n.T("be.imp.loginScriptSkipped"))
	}

	if szProtocol == "serial" {
		host.M_pSerial = parseCrtSerial(values)
		host.M_szHost = host.M_pSerial.M_szPath
		cand.M_host = host
		return cand, true
	}

	host.M_szHost = szHost
	iPort, _ := values.num(szPortKey)
	if iPort == 0 {
		iPort = 22
		if szProtocol == "telnet" {
			iPort = 23
		}
	}
	host.M_iPort = iPort

	host.M_iKeepaliveSec = 30
	if iCheck, _ := values.num("Idle NO-OP Check"); iCheck != 0 {
		if iTimeout, _ := values.num("Idle NO-OP Timeout"); iTimeout > 0 {
			host.M_iKeepaliveSec = iTimeout
		}
	}

	szKey := szGlobalKey
	if iGlobal, bFound := values.num("Use Global Public Key"); bFound && iGlobal == 0 {
		szKey = values.str("Identity Filename V2")
	}
	if szKey = strings.TrimSpace(expandCrtPath(szKey, szConfigDir)); szKey != "" && szProtocol == "ssh" {
		host.M_szAuthType = "key"
		cand.M_szKeyFile = szKey
	}

	host.M_slForwards = append(ParseCrtForwards(values.lines("Port Forward Table V2"), "L"), ParseCrtForwards(values.lines("Reverse Forward Table V2"), "R")...)
	cand.M_host = host
	return cand, true
}

// parseCrtSerial 은 시리얼 설정을 읽는다 (표본 미확인, import.md §1.5).
func parseCrtSerial(values crtValues) *Serial {
	szPort := strings.TrimSpace(values.str("Com Port"))
	if szPort == "" {
		szPort = "COM1"
	}
	iBaud, _ := values.num("Baud Rate")
	if iBaud == 0 {
		iBaud = 9600
	}
	iData, _ := values.num("Data Bits")
	if iData == 0 {
		iData = 8
	}
	iParity, _ := values.num("Parity")
	iStop, _ := values.num("Stop Bits")
	return &Serial{
		M_szPath:        szPort,
		M_iBaudRate:     iBaud,
		M_iDataBits:     iData,
		M_szParity:      puttyParity(iParity),
		M_fStopBits:     crtStopBits(iStop),
		M_szFlowControl: "none",
		M_szEnterSends:  "CR",
	}
}

// crtSessionKey 는 경유 세션 참조를 비교할 키(소문자, / 구분)다.
func crtSessionKey(szPath string) string {
	szPath = strings.ReplaceAll(strings.TrimSpace(szPath), "\\", "/")
	return strings.ToLower(strings.Trim(szPath, "/"))
}

// ResolveCrtJumps 는 Firewall Name 의 `Session:{경로}` 를 읽은 세션의 호스트로 바꿔 jumpVia 에 둔다.
// slCands 와 slSessions 는 같은 순서·길이다.
func ResolveCrtJumps(slCands []Candidate, slSessions []CrtSession) {
	mapHosts := map[string]string{}
	for _, session := range slSessions {
		szPath := strings.Join(append(append([]string{}, session.M_slFolder...), session.M_szName), "/")
		mapHosts[crtSessionKey(szPath)] = strings.TrimSpace(session.M_values.str("Hostname"))
	}
	for iIndex := range slCands {
		szFirewall := strings.TrimSpace(slSessions[iIndex].M_values.str("Firewall Name"))
		if szFirewall == "" || strings.EqualFold(szFirewall, "None") {
			continue
		}
		if len(szFirewall) > 8 && strings.EqualFold(szFirewall[:8], "Session:") {
			if szJump := mapHosts[crtSessionKey(szFirewall[8:])]; szJump != "" {
				slCands[iIndex].M_szJumpVia = szJump
				continue
			}
		}
		slCands[iIndex].M_szNote = joinNote(slCands[iIndex].M_szNote, i18n.T("be.crt.firewall", "name", szFirewall))
	}
}

// ResolveCrtDirs 는 고른 폴더에서 세션 폴더와 설정 폴더를 정한다. 안에 Sessions 가 있으면 그것을 세션 폴더로 본다.
func ResolveCrtDirs(szPicked string) (string, string) {
	szSessions := filepath.Join(szPicked, g_cszCrtSessions)
	if pInfo, errStat := os.Stat(szSessions); errStat == nil && pInfo.IsDir() {
		return szSessions, szPicked
	}
	return szPicked, filepath.Dir(szPicked)
}

// crtHasLoginScript 는 세션의 로그온 동작(Expect/Send)이 켜져 있고 내용이 있는지 본다 (D150).
// 꺼진 세션은 D:"Use Login Script"=00000000 과 빈 Z:"Login Script V4"=00000000 이다.
func crtHasLoginScript(values crtValues) bool {
	iUse, _ := values.num("Use Login Script")
	if iUse == 0 {
		return false
	}
	for _, szKey := range []string{"Login Script V4", "Login Script V3", "Login Script V2"} {
		for _, szLine := range values.lines(szKey) {
			if strings.TrimSpace(szLine) != "" {
				return true
			}
		}
	}
	return false
}

// joinNote 는 후보 메모에 한 줄을 덧붙인다.
func joinNote(szNote string, szAdd string) string {
	if szNote == "" {
		return szAdd
	}
	return szNote + " · " + szAdd
}
