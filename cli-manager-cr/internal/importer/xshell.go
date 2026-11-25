package importer

import (
	"archive/zip"
	"dariterm/internal/i18n"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
)

// Xshell 파일 이름·한도 (import.md §1.6).
const (
	g_cszXshExt      = ".xsh"     // 세션 파일
	g_cszXtsExt      = ".xts"     // 내보내기 파일(zip)
	g_cszXshSessions = "Sessions" // 설치된 Xshell 의 세션 폴더 이름
	g_cszXtsRoot     = "xshell"   // 내보내기 파일 안에서 세션이 든 최상위 폴더 (소문자 비교)
	g_ciXshMaxBytes  = 1 << 20    // 세션 파일 하나의 한도(넘으면 건너뜀)
	g_ciXtsMaxFiles  = 20000      // 내보내기 파일에서 읽는 세션 수 한도
	g_ciXtsMaxBytes  = 64 << 20   // 내보내기 파일에서 읽는 세션 내용의 합계 한도
	g_cszXshNoteSep  = " · "      // 후보 note 를 잇는 구분자
	g_cszXshSecConn  = "CONNECTION"
	g_cszXshSecAuth  = "CONNECTION:AUTHENTICATION"
	g_cszXshSecSsh   = "CONNECTION:SSH"
	g_cszXshSecKeep  = "CONNECTION:KEEPALIVE"
	g_cszXshSecProxy = "CONNECTION:PROXY"
	g_cszXshSecTerm  = "TERMINAL"
)

// XshSession 은 세션 파일 하나를 읽은 결과다. 이름은 파일 이름(확장자 제외), 폴더는 세션 폴더 기준 하위 폴더 이름들이다.
type XshSession struct {
	M_szName   string
	M_slFolder []string
	M_values   xshValues
}

// g_ciXshMaxFwd 는 세션 하나에서 읽는 포워딩 수 한도다.
const g_ciXshMaxFwd = 1000

// xshValues 는 세션 파일의 값이다. 키는 "구역 이름(대문자)\x00키 이름(소문자)".
type xshValues map[string]string

// str 은 구역·키의 값을 앞뒤 공백을 떼어 돌려준다 (없으면 빈 문자열). 구역·키 이름은 대소문자를 가리지 않는다.
func (values xshValues) str(szSection string, szKey string) string {
	return strings.TrimSpace(values[strings.ToUpper(szSection)+"\x00"+strings.ToLower(szKey)])
}

// num 은 구역·키의 값을 10진 정수로 돌려준다 (없거나 숫자가 아니면 0).
func (values xshValues) num(szSection string, szKey string) int {
	iValue, errConv := strconv.Atoi(values.str(szSection, szKey))
	if errConv != nil {
		return 0
	}
	return iValue
}

// decodeXshText 는 세션 파일 바이트를 글로 바꾼다: UTF-16(LE·BE) BOM 이면 그것으로, UTF-8 BOM 은 버리고,
// 그 밖에는 유효한 UTF-8 이면 그대로 아니면 EUC-KR(CP949)로 해석한다.
func decodeXshText(bRaw []byte) string {
	if len(bRaw) >= 2 && ((bRaw[0] == 0xFF && bRaw[1] == 0xFE) || (bRaw[0] == 0xFE && bRaw[1] == 0xFF)) {
		bLittle := bRaw[0] == 0xFF
		bBody := bRaw[2:]
		slUnits := make([]uint16, 0, len(bBody)/2)
		for iPos := 0; iPos+1 < len(bBody); iPos += 2 {
			if bLittle {
				slUnits = append(slUnits, uint16(bBody[iPos])|uint16(bBody[iPos+1])<<8)
			} else {
				slUnits = append(slUnits, uint16(bBody[iPos])<<8|uint16(bBody[iPos+1]))
			}
		}
		return string(utf16.Decode(slUnits))
	}
	if len(bRaw) >= 3 && bRaw[0] == 0xEF && bRaw[1] == 0xBB && bRaw[2] == 0xBF {
		bRaw = bRaw[3:]
	}
	return decodeLocalText(bRaw)
}

// decodeLocalText 는 유효한 UTF-8 이면 그대로, 아니면 EUC-KR(CP949)로 해석한다 (해석하지 못하면 원래 바이트 그대로).
func decodeLocalText(bRaw []byte) string {
	if utf8.Valid(bRaw) {
		return string(bRaw)
	}
	bDecoded, errDecode := korean.EUCKR.NewDecoder().Bytes(bRaw)
	if errDecode != nil {
		return string(bRaw)
	}
	return string(bDecoded)
}

// ParseXsh 는 세션 파일 내용을 값으로 바꾼다. 형식은 INI: `[구역]` 줄과 `키=값` 줄. 그 밖의 줄은 무시한다.
func ParseXsh(bRaw []byte) xshValues {
	values := xshValues{}
	szSection := ""
	for _, szLine := range strings.Split(decodeXshText(bRaw), "\n") {
		szLine = strings.TrimRight(szLine, "\r")
		szTrim := strings.TrimSpace(szLine)
		if len(szTrim) >= 2 && szTrim[0] == '[' && szTrim[len(szTrim)-1] == ']' {
			szSection = strings.ToUpper(strings.TrimSpace(szTrim[1 : len(szTrim)-1]))
			continue
		}
		iEq := strings.Index(szLine, "=")
		if iEq <= 0 {
			continue
		}
		szKey := szSection + "\x00" + strings.ToLower(strings.TrimSpace(szLine[:iEq]))
		if _, bSeen := values[szKey]; !bSeen { // 같은 키가 두 번 나오면 처음 것
			values[szKey] = szLine[iEq+1:]
		}
	}
	return values
}

// xshProtocol 은 Protocol 값을 프로토콜로 바꾼다. 가져오지 않는 것(SERIAL·RLOGIN·SFTP·FTP·LOCAL·RDP 등)은 bOk=false.
func xshProtocol(szValue string) (string, bool) {
	switch strings.ToUpper(szValue) {
	case "SSH", "":
		return "ssh", true
	case "TELNET":
		return "telnet", true
	}
	return "", false
}

// xshEncoding 은 CodePage(Windows 코드 페이지 번호)를 서버 인코딩 이름으로 바꾼다. 모르는 번호는 utf-8.
func xshEncoding(iCodePage int) string {
	switch iCodePage {
	case 949:
		return "cp949"
	case 51949:
		return "euc-kr"
	case 932:
		return "shift_jis"
	case 51932, 20932:
		return "euc-jp"
	case 936, 54936:
		return "gbk"
	case 950:
		return "big5"
	case 1252, 28591:
		return "latin1"
	}
	return "utf-8"
}

// ParseXshSession 은 세션 하나를 후보로 바꾼다. 건너뛰면 bOk=false (import.md §1.6).
// 저장된 비밀번호(Password)·암호문(Passphrase)은 읽지 않는다.
func ParseXshSession(session XshSession) (Candidate, bool) {
	values := session.M_values
	szProtocol, bOk := xshProtocol(values.str(g_cszXshSecConn, "Protocol"))
	if !bOk {
		return Candidate{}, false
	}
	szHost := values.str(g_cszXshSecConn, "Host")
	if szHost == "" {
		return Candidate{}, false
	}
	iPort := values.num(g_cszXshSecConn, "Port")
	if iPort <= 0 || iPort > 65535 {
		iPort = 22
		if szProtocol == "telnet" {
			iPort = 23
		}
	}
	host := HostDraft{
		M_szAlias:       session.M_szName,
		M_szProtocol:    szProtocol,
		M_szHost:        szHost,
		M_iPort:         iPort,
		M_szUsername:    values.str(g_cszXshSecAuth, "UserName"),
		M_szAuthType:    "password",
		M_szEncoding:    xshEncoding(values.num(g_cszXshSecTerm, "CodePage")),
		M_iKeepaliveSec: 30,
		M_szNotes:       values.str(g_cszXshSecConn, "Description"),
	}
	if szTerm := values.str(g_cszXshSecTerm, "Type"); szTerm != "" && !strings.EqualFold(szTerm, "xterm") {
		host.M_szTermType = szTerm // xterm 은 적용 기본값(xterm-256color)에 맡긴다
	}
	if values.num(g_cszXshSecKeep, "KeepAlive") != 0 {
		if iInterval := values.num(g_cszXshSecKeep, "KeepAliveInterval"); iInterval > 0 {
			host.M_iKeepaliveSec = iInterval
		}
	}

	slNotes := []string{}
	if szProtocol == "ssh" {
		// 사용자 키는 Xshell 의 키 보관소에 있어 파일로 옮길 수 없다: 인증만 키로 두고(적용 때 에이전트가 된다) 알린다.
		if szKey := values.str(g_cszXshSecAuth, "UserKey"); szKey != "" {
			host.M_szAuthType = "key"
			slNotes = append(slNotes, i18n.T("be.xsh.userKey", "name", szKey))
		} else if values.num(g_cszXshSecSsh, "UseAuthAgent") != 0 {
			host.M_szAuthType = "key"
		}
		host.M_szStartupCommand = values.str(g_cszXshSecSsh, "RemoteCommand")
		slForwards, slNames, iSkipped := ParseXshForwards(values)
		host.M_slForwards = slForwards
		if len(slNames) > 0 {
			// 포워딩 항목에는 이름을 둘 자리가 없어, Xshell 에서 붙인 설명은 서버 메모에 적어 둔다.
			szNames := i18n.T("be.xsh.forwardNames") + "\n" + strings.Join(slNames, "\n")
			if host.M_szNotes != "" {
				szNames = host.M_szNotes + "\n\n" + szNames
			}
			host.M_szNotes = szNames
		}
		if iSkipped > 0 {
			slNotes = append(slNotes, i18n.T("be.xsh.forwards", "count", iSkipped))
		}
	}
	if szProxy := values.str(g_cszXshSecProxy, "Proxy"); szProxy != "" {
		slNotes = append(slNotes, i18n.T("be.xsh.proxy", "name", szProxy))
	}
	return Candidate{
		M_szSource:     G_cszSourceXshell,
		M_szName:       session.M_szName,
		M_host:         host,
		M_szNote:       strings.Join(slNotes, g_cszXshNoteSep),
		M_slFolderPath: append([]string{}, session.M_slFolder...),
	}, true
}

// ParseXshForwards 는 [CONNECTION:SSH] 의 포워딩 항목을 읽는다 (import.md §1.6 포트 포워딩).
// 항목 N(0 ~ FwdReqCount-1)은 키 `FwdReq_{N}_{이름}` 들이다: Incoming(0 나가는 것 = 로컬 L, 1 들어오는 것 = 원격 R),
// Source·Port(듣는 주소·포트), Host·HostPort(대상 주소·포트), Description(설명), LocalOnly(쓰지 않는다).
// 돌려주는 것: 포워딩 목록, 설명이 있는 항목의 "{L|R} {듣는 포트} → {대상}:{포트}  {설명}" 줄들, 옮기지 못한 항목 수.
func ParseXshForwards(values xshValues) ([]Forward, []string, int) {
	iCount := values.num(g_cszXshSecSsh, "FwdReqCount")
	if iCount > g_ciXshMaxFwd {
		iCount = g_ciXshMaxFwd
	}
	slOut := []Forward{}
	slNames := []string{}
	iSkipped := 0
	for iIndex := 0; iIndex < iCount; iIndex++ {
		szPrefix := "FwdReq_" + strconv.Itoa(iIndex) + "_"
		szType := ""
		switch values.str(g_cszXshSecSsh, szPrefix+"Incoming") {
		case "0", "":
			szType = "L"
		case "1":
			szType = "R"
		}
		iBindPort := values.num(g_cszXshSecSsh, szPrefix+"Port")
		iDestPort := values.num(g_cszXshSecSsh, szPrefix+"HostPort")
		szDestHost := values.str(g_cszXshSecSsh, szPrefix+"Host")
		if szType == "" || iBindPort < 1 || iBindPort > 65535 || iDestPort < 1 || iDestPort > 65535 || szDestHost == "" {
			iSkipped++
			continue
		}
		// 듣는 주소: 비었거나 localhost 면 127.0.0.1. LocalOnly=0("다른 PC 의 접속도 받기")은 옮기지 않는다 —
		// 가져오기만으로 이 PC 의 포트가 네트워크에 열리지 않게 한다.
		szBindHost := values.str(g_cszXshSecSsh, szPrefix+"Source")
		if szBindHost == "" || strings.EqualFold(szBindHost, "localhost") {
			szBindHost = "127.0.0.1"
		}
		slOut = append(slOut, Forward{M_szType: szType, M_szBindHost: szBindHost, M_iBindPort: iBindPort, M_szDestHost: szDestHost, M_iDestPort: iDestPort})
		if szDescription := values.str(g_cszXshSecSsh, szPrefix+"Description"); szDescription != "" {
			slNames = append(slNames, szType+" "+strconv.Itoa(iBindPort)+" → "+szDestHost+":"+strconv.Itoa(iDestPort)+"  "+szDescription)
		}
	}
	return slOut, slNames, iSkipped
}

// xshSessionOf 는 세션 폴더 기준 상대 경로(/ 구분)와 내용으로 세션을 만든다. 세션 파일이 아니거나 경로가 이상하면 bOk=false.
func xshSessionOf(szRel string, bRaw []byte) (XshSession, bool) {
	slParts := strings.Split(szRel, "/")
	szFile := slParts[len(slParts)-1]
	if len(szFile) <= len(g_cszXshExt) || !strings.EqualFold(szFile[len(szFile)-len(g_cszXshExt):], g_cszXshExt) {
		return XshSession{}, false
	}
	slFolder := []string{}
	for _, szPart := range slParts[:len(slParts)-1] {
		if szPart == "" || szPart == "." || szPart == ".." {
			return XshSession{}, false
		}
		slFolder = append(slFolder, szPart)
	}
	return XshSession{M_szName: szFile[:len(szFile)-len(g_cszXshExt)], M_slFolder: slFolder, M_values: ParseXsh(bRaw)}, true
}

// sortXshSessions 는 폴더 경로·이름의 사전순으로 정렬한다.
func sortXshSessions(slSessions []XshSession) {
	fnKey := func(session XshSession) string {
		return strings.Join(append(append([]string{}, session.M_slFolder...), session.M_szName), "\x00")
	}
	sort.SliceStable(slSessions, func(iLeft, iRight int) bool { return fnKey(slSessions[iLeft]) < fnKey(slSessions[iRight]) })
}

// ReadXshDir 는 세션 폴더 아래 .xsh 파일을 모두 읽는다. 읽지 못한 파일·폴더와 너무 큰 파일은 건너뛴다.
func ReadXshDir(szSessionsDir string) []XshSession {
	slOut := []XshSession{}
	_ = filepath.WalkDir(szSessionsDir, func(szPath string, pEntry fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			if pEntry != nil && pEntry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if pEntry.IsDir() || !strings.EqualFold(filepath.Ext(szPath), g_cszXshExt) {
			return nil
		}
		if pInfo, errInfo := pEntry.Info(); errInfo != nil || pInfo.Size() > g_ciXshMaxBytes {
			return nil
		}
		szRel, errRel := filepath.Rel(szSessionsDir, szPath)
		if errRel != nil {
			return nil
		}
		bRaw, errRead := os.ReadFile(szPath)
		if errRead != nil {
			return nil
		}
		if session, bOk := xshSessionOf(filepath.ToSlash(szRel), bRaw); bOk {
			slOut = append(slOut, session)
		}
		return nil
	})
	sortXshSessions(slOut)
	return slOut
}

// ReadXtsFile 은 Xshell 내보내기 파일(.xts, zip)에서 최상위 "Xshell" 폴더 아래의 .xsh 파일을 읽는다.
// 디스크에 풀지 않고 메모리에서 읽는다. zip 이 아니면 오류다. 한도를 넘는 파일은 건너뛴다.
func ReadXtsFile(szPath string) ([]XshSession, error) {
	pZip, errOpen := zip.OpenReader(szPath)
	if errOpen != nil {
		return nil, i18n.Errorf("be.xsh.badFile", errOpen)
	}
	defer pZip.Close()
	slOut := []XshSession{}
	iTotal := 0
	for _, pFile := range pZip.File {
		if len(slOut) >= g_ciXtsMaxFiles || iTotal >= g_ciXtsMaxBytes {
			break
		}
		if pFile.FileInfo().IsDir() || pFile.UncompressedSize64 > g_ciXshMaxBytes {
			continue
		}
		// 이름은 UTF-8 표시가 없으면 내보낸 PC 의 코드 페이지 바이트다: 유효한 UTF-8 이 아니면 EUC-KR 로 해석한다.
		szName := strings.ReplaceAll(decodeLocalText([]byte(pFile.Name)), "\\", "/")
		iSlash := strings.Index(szName, "/")
		if iSlash < 0 || !strings.EqualFold(szName[:iSlash], g_cszXtsRoot) {
			continue
		}
		szRel := szName[iSlash+1:]
		if !strings.EqualFold(filepath.Ext(szRel), g_cszXshExt) {
			continue
		}
		bRaw, errRead := readZipEntry(pFile, g_ciXshMaxBytes)
		if errRead != nil {
			continue
		}
		iTotal += len(bRaw)
		if session, bOk := xshSessionOf(szRel, bRaw); bOk {
			slOut = append(slOut, session)
		}
	}
	sortXshSessions(slOut)
	return slOut, nil
}

// readZipEntry 는 zip 항목을 iMax 바이트까지만 읽는다 (머리의 크기 표시를 믿지 않는다). 넘으면 오류다.
func readZipEntry(pFile *zip.File, iMax int) ([]byte, error) {
	pReader, errOpen := pFile.Open()
	if errOpen != nil {
		return nil, errOpen
	}
	defer pReader.Close()
	bRaw, errRead := io.ReadAll(io.LimitReader(pReader, int64(iMax)+1))
	if errRead != nil {
		return nil, errRead
	}
	if len(bRaw) > iMax {
		return nil, io.ErrUnexpectedEOF
	}
	return bRaw, nil
}

// ResolveXshDir 는 고른 폴더에서 세션 폴더를 정한다. 안에 Sessions 가 있으면 그것, 없으면 고른 폴더 자체다.
func ResolveXshDir(szPicked string) string {
	szSessions := filepath.Join(szPicked, g_cszXshSessions)
	if pInfo, errStat := os.Stat(szSessions); errStat == nil && pInfo.IsDir() {
		return szSessions
	}
	return szPicked
}

// ScanXshell 은 Xshell 세션을 후보로 바꾼다 (import.md §1.6).
// szPath 가 비면 기본 세션 폴더(없으면 빈 목록), 폴더면 그 폴더, .xsh 파일이면 그 세션 하나, 그 밖의 파일이면 내보내기 파일(.xts)로 읽는다.
// 경로가 없거나 내보내기 파일을 열지 못하면 오류다.
func ScanXshell(szPath string) ([]Candidate, error) {
	var slSessions []XshSession
	if szPath == "" {
		szDir := XshellSessionsDir()
		if szDir == "" {
			return []Candidate{}, nil
		}
		slSessions = ReadXshDir(szDir)
	} else {
		pInfo, errStat := os.Stat(szPath)
		if errStat != nil {
			return nil, i18n.Errorf("be.xsh.badFile", errStat)
		}
		if pInfo.IsDir() {
			slSessions = ReadXshDir(ResolveXshDir(szPath))
		} else if strings.EqualFold(filepath.Ext(szPath), g_cszXshExt) {
			// 세션 파일 하나: 폴더 없이 그 세션만 (import.md §1.6).
			if pInfo.Size() > g_ciXshMaxBytes {
				return []Candidate{}, nil
			}
			bRaw, errRead := os.ReadFile(szPath)
			if errRead != nil {
				return nil, i18n.Errorf("be.xsh.badFile", errRead)
			}
			if session, bOk := xshSessionOf(filepath.Base(szPath), bRaw); bOk {
				slSessions = []XshSession{session}
			}
		} else {
			slRead, errRead := ReadXtsFile(szPath)
			if errRead != nil {
				return nil, errRead
			}
			slSessions = slRead
		}
	}
	slCands := []Candidate{}
	for _, session := range slSessions {
		if cand, bOk := ParseXshSession(session); bOk {
			slCands = append(slCands, cand)
		}
	}
	return slCands, nil
}

// xshellSessionsUnder 는 문서 폴더에서 Xshell 세션 폴더를 찾는다: 새 판부터
// `NetSarang Computer\{8,7,6}\Xshell\Sessions`, 그다음 옛 판의 `NetSarang\Xshell\Sessions`. 없으면 빈 문자열.
func xshellSessionsUnder(szDocuments string) string {
	if szDocuments == "" {
		return ""
	}
	slTries := []string{}
	for _, szVersion := range []string{"8", "7", "6"} {
		slTries = append(slTries, filepath.Join(szDocuments, "NetSarang Computer", szVersion, "Xshell", g_cszXshSessions))
	}
	slTries = append(slTries, filepath.Join(szDocuments, "NetSarang", "Xshell", g_cszXshSessions))
	for _, szTry := range slTries {
		if pInfo, errStat := os.Stat(szTry); errStat == nil && pInfo.IsDir() {
			return szTry
		}
	}
	return ""
}
