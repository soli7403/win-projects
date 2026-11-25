package importer

import (
	"archive/zip"
	"bytes"
	"dariterm/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/korean"
)

// xshBytes 는 Xshell 이 쓰는 모양(UTF-16LE + BOM, CRLF)으로 세션 파일 내용을 만든다.
func xshBytes(slLines ...string) []byte {
	slUnits := utf16.Encode([]rune(strings.Join(slLines, "\r\n") + "\r\n"))
	bOut := []byte{0xFF, 0xFE}
	for _, uUnit := range slUnits {
		bOut = append(bOut, byte(uUnit), byte(uUnit>>8))
	}
	return bOut
}

// xshSample 은 실제 세션 파일에서 본 구역·키 모양을 흉내 낸 SSH 세션이다(값은 지어낸 것).
func xshSample(szHost string, slExtra ...string) []byte {
	slLines := []string{
		"[CONNECTION:PROXY]", "Proxy=", "StartUp=0",
		"[CONNECTION:SERIAL]", "BaudRate=12", "ComPort=0", "DataBits=3",
		"[SessionInfo]", "Version=8.1", "Description=Xshell session file",
		"[CONNECTION:SSH]", "AgentForwarding=0", "FwdReqCount=0", "RemoteCommand=", "UseAuthAgent=0",
		"[CONNECTION]", "Description=랩 DB 서버", "Host=" + szHost, "Port=22", "Protocol=SSH",
		"[TERMINAL]", "CodePage=65001", "Type=xterm",
		"[CONNECTION:AUTHENTICATION]", "AuthMethodList=01,10,20,30", "Password=AAAAsecretAAAA", "Passphrase=", "UserKey=", "UserName=ops",
		"[CONNECTION:KEEPALIVE]", "KeepAlive=1", "KeepAliveInterval=60",
	}
	return xshBytes(append(slLines, slExtra...)...)
}

func useKoreanImport(t *testing.T) {
	t.Helper()
	szBefore := i18n.Lang()
	i18n.SetLang("ko")
	t.Cleanup(func() { i18n.SetLang(szBefore) })
}

// 세션 파일 해석: UTF-16LE·UTF-16BE·UTF-8(BOM)·CP949, 구역·키는 대소문자를 가리지 않고, 같은 키는 처음 것 (import.md §1.6).
func TestParseXshEncodingsAndSections(t *testing.T) {
	const szText = "[Connection]\r\nHost=10.0.0.1\r\nDescription=운영 서버\r\nhost=ignored\r\n; 주석\r\n[CONNECTION:AUTHENTICATION]\r\nUserName= root \r\n"
	slUnits := utf16.Encode([]rune(szText))
	bLe, bBe := []byte{0xFF, 0xFE}, []byte{0xFE, 0xFF}
	for _, uUnit := range slUnits {
		bLe = append(bLe, byte(uUnit), byte(uUnit>>8))
		bBe = append(bBe, byte(uUnit>>8), byte(uUnit))
	}
	bCp949, errEncode := korean.EUCKR.NewEncoder().Bytes([]byte(szText))
	if errEncode != nil {
		t.Fatal(errEncode)
	}
	for szName, bRaw := range map[string][]byte{
		"utf16le": bLe, "utf16be": bBe, "utf8bom": append([]byte{0xEF, 0xBB, 0xBF}, szText...), "utf8": []byte(szText), "cp949": bCp949,
	} {
		values := ParseXsh(bRaw)
		if values.str("CONNECTION", "Host") != "10.0.0.1" || values.str("connection", "HOST") != "10.0.0.1" {
			t.Fatalf("%s: Host %q", szName, values.str("CONNECTION", "Host"))
		}
		if values.str("CONNECTION", "Description") != "운영 서버" {
			t.Fatalf("%s: Description %q", szName, values.str("CONNECTION", "Description"))
		}
		if values.str("CONNECTION:AUTHENTICATION", "UserName") != "root" {
			t.Fatalf("%s: UserName %q", szName, values.str("CONNECTION:AUTHENTICATION", "UserName"))
		}
	}
}

// 세션 → 후보: 호스트·포트·사용자·메모·인코딩·연결 유지·폴더, 비밀번호는 어디에도 들어가지 않는다 (import.md §1.6).
func TestParseXshSessionFields(t *testing.T) {
	useKoreanImport(t)
	session, bOk := xshSessionOf("LAB/DB/db01.xsh", xshSample("10.0.0.21"))
	if !bOk {
		t.Fatal("세션 파일로 봐야 함")
	}
	cand, bOk := ParseXshSession(session)
	if !bOk {
		t.Fatal("후보가 돼야 함")
	}
	host := cand.M_host
	if cand.M_szSource != "xshell" || cand.M_szName != "db01" || host.M_szAlias != "db01" {
		t.Fatalf("이름·출처: %+v", cand)
	}
	if strings.Join(cand.M_slFolderPath, "/") != "LAB/DB" {
		t.Fatalf("폴더: %q", cand.M_slFolderPath)
	}
	if host.M_szProtocol != "ssh" || host.M_szHost != "10.0.0.21" || host.M_iPort != 22 || host.M_szUsername != "ops" {
		t.Fatalf("접속 정보: %+v", host)
	}
	if host.M_szAuthType != "password" || cand.M_szKeyFile != "" || cand.M_szNote != "" {
		t.Fatalf("인증: %+v / %+v", host, cand)
	}
	if host.M_szEncoding != "utf-8" || host.M_szTermType != "" || host.M_iKeepaliveSec != 60 || host.M_szNotes != "랩 DB 서버" {
		t.Fatalf("그 밖: %+v", host)
	}
	if host.M_szStartupCommand != "" || len(host.M_slForwards) != 0 || host.M_pSerial != nil {
		t.Fatalf("없어야 하는 값: %+v", host)
	}
	if bJson := strings.ToLower(strings.Join([]string{cand.M_szNote, host.M_szNotes, host.M_szUsername, host.M_szHost}, "|")); strings.Contains(bJson, "secret") {
		t.Fatal("비밀번호가 후보에 들어가면 안 됨")
	}
}

// 값에 따른 갈래: 텔넷 기본 포트, 코드 페이지, 터미널 종류, 연결 유지 꺼짐, 사용자 키·에이전트·포워딩·프록시 안내.
func TestParseXshSessionVariants(t *testing.T) {
	useKoreanImport(t)
	fnCand := func(slLines ...string) (Candidate, bool) {
		session, _ := xshSessionOf("s.xsh", xshBytes(slLines...))
		return ParseXshSession(session)
	}
	cand, bOk := fnCand("[CONNECTION]", "Host=sw1", "Port=0", "Protocol=TELNET", "[TERMINAL]", "CodePage=949", "Type=vt100", "[CONNECTION:KEEPALIVE]", "KeepAlive=0", "KeepAliveInterval=60")
	if !bOk || cand.M_host.M_szProtocol != "telnet" || cand.M_host.M_iPort != 23 || cand.M_host.M_szEncoding != "cp949" || cand.M_host.M_szTermType != "vt100" || cand.M_host.M_iKeepaliveSec != 30 {
		t.Fatalf("텔넷: %+v", cand.M_host)
	}
	for iCodePage, szWant := range map[string]string{"65001": "utf-8", "51949": "euc-kr", "932": "shift_jis", "51932": "euc-jp", "936": "gbk", "950": "big5", "1252": "latin1", "12345": "utf-8", "": "utf-8"} {
		c, _ := fnCand("[CONNECTION]", "Host=h", "Protocol=SSH", "[TERMINAL]", "CodePage="+iCodePage)
		if c.M_host.M_szEncoding != szWant {
			t.Fatalf("코드 페이지 %q → %q (기대 %q)", iCodePage, c.M_host.M_szEncoding, szWant)
		}
	}
	cand, _ = fnCand("[CONNECTION]", "Host=h", "Port=2222", "Protocol=SSH",
		"[CONNECTION:AUTHENTICATION]", "UserKey=id_lab", "Password=zzz",
		"[CONNECTION:SSH]", "FwdReqCount=2", "RemoteCommand=tmux attach",
		"[CONNECTION:PROXY]", "Proxy=corp")
	if cand.M_host.M_iPort != 2222 || cand.M_host.M_szAuthType != "key" || cand.M_szKeyFile != "" || cand.M_host.M_szStartupCommand != "tmux attach" {
		t.Fatalf("키 세션: %+v / %+v", cand.M_host, cand)
	}
	const szWantNote = `Xshell 사용자 키 "id_lab" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정) · 포트 포워딩 2개는 옮기지 않음 · 프록시 설정 "corp" 은 옮기지 않음`
	if cand.M_szNote != szWantNote {
		t.Fatalf("안내:\n%q\n기대:\n%q", cand.M_szNote, szWantNote)
	}
	cand, _ = fnCand("[CONNECTION]", "Host=h", "Protocol=SSH", "[CONNECTION:SSH]", "UseAuthAgent=1")
	if cand.M_host.M_szAuthType != "key" || cand.M_szNote != "" {
		t.Fatalf("에이전트: %+v", cand)
	}
	// 건너뛰는 것: 호스트가 빈 세션, 가져오지 않는 프로토콜.
	if _, bSkip := fnCand("[CONNECTION]", "Host=", "Protocol=SSH"); bSkip {
		t.Fatal("호스트가 비면 건너뛰어야 함")
	}
	for _, szProtocol := range []string{"SERIAL", "RLOGIN", "SFTP", "FTP", "LOCAL", "RDP"} {
		if _, bSkip := fnCand("[CONNECTION]", "Host=h", "Protocol="+szProtocol); bSkip {
			t.Fatalf("%s 는 건너뛰어야 함", szProtocol)
		}
	}
}

// 세션 폴더 읽기: 하위 폴더 구조, .xsh 만, 사전순. 고른 폴더 안에 Sessions 가 있으면 그것을 읽는다.
func TestScanXshellFolder(t *testing.T) {
	szRoot := t.TempDir()
	fnWrite := func(szRel string, bData []byte) {
		szPath := filepath.Join(szRoot, filepath.FromSlash(szRel))
		if errMkdir := os.MkdirAll(filepath.Dir(szPath), 0o755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		if errWrite := os.WriteFile(szPath, bData, 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	fnWrite("Sessions/web01.xsh", xshSample("10.0.0.1"))
	fnWrite("Sessions/LAB/DB/db01.XSH", xshSample("10.0.0.21"))
	fnWrite("Sessions/LAB/app01.xsh", xshSample("10.0.0.11"))
	fnWrite("Sessions/LAB/readme.txt", []byte("not a session"))
	fnWrite("Sessions/default.xshf.xfs", xshSample("ignored"))
	fnWrite("outside.xsh", xshSample("10.9.9.9")) // Sessions 밖: 읽지 않는다

	for _, szPick := range []string{szRoot, filepath.Join(szRoot, "Sessions")} {
		slCands, errScan := ScanXshell(szPick)
		if errScan != nil {
			t.Fatal(errScan)
		}
		slGot := []string{}
		for _, cand := range slCands {
			slGot = append(slGot, strings.Join(append(append([]string{}, cand.M_slFolderPath...), cand.M_szName), "/")+"="+cand.M_host.M_szHost)
		}
		if szGot := strings.Join(slGot, " "); szGot != "LAB/DB/db01=10.0.0.21 LAB/app01=10.0.0.11 web01=10.0.0.1" {
			t.Fatalf("%s: %s", szPick, szGot)
		}
	}
	if _, errScan := ScanXshell(filepath.Join(szRoot, "nope")); errScan == nil {
		t.Fatal("없는 경로는 오류여야 함")
	}
}

// writeXts 는 내보내기 파일을 만든다. 항목 이름은 준 바이트 그대로(UTF-8 표시 없이) 넣는다.
func writeXts(t *testing.T, mapFiles map[string][]byte, slOrder []string) string {
	t.Helper()
	var buf bytes.Buffer
	pWriter := zip.NewWriter(&buf)
	for _, szName := range slOrder {
		pEntry, errCreate := pWriter.CreateHeader(&zip.FileHeader{Name: szName, Method: zip.Deflate, NonUTF8: true})
		if errCreate != nil {
			t.Fatal(errCreate)
		}
		if _, errWrite := pEntry.Write(mapFiles[szName]); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	if errClose := pWriter.Close(); errClose != nil {
		t.Fatal(errClose)
	}
	szPath := filepath.Join(t.TempDir(), "export.xts")
	if errWrite := os.WriteFile(szPath, buf.Bytes(), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	return szPath
}

// 내보내기 파일(.xts): 최상위 Xshell 폴더 아래의 .xsh 만 읽고, CP949 로 적힌 폴더 이름을 한글로 읽는다 (import.md §1.6).
func TestScanXshellExportFile(t *testing.T) {
	bKoreanDir, errEncode := korean.EUCKR.NewEncoder().Bytes([]byte("운영 센터"))
	if errEncode != nil {
		t.Fatal(errEncode)
	}
	szKoreanDir := string(bKoreanDir)
	mapFiles := map[string][]byte{
		"xts.zcf":                             []byte("[SessionInfo]\r\nVersion=6.0\r\n"),
		"Xshell/default.xshf.xfs":             xshSample("ignored"),
		"Xshell/web01.xsh":                    xshSample("10.0.0.1"),
		"Xshell/LAB/DB/db01.xsh":              xshSample("10.0.0.21"),
		"Xshell/" + szKoreanDir + "/gw.xsh":   xshSample("10.0.0.254"),
		"Xshell/../evil.xsh":                  xshSample("10.6.6.6"), // 폴더 이름이 이상한 항목: 버린다
		"Xftp/files.xsh":                      xshSample("10.7.7.7"), // 다른 프로그램 폴더: 읽지 않는다
		"xsl/Sessions/note.xsh":               xshSample("10.8.8.8"),
		"com/SECSH/HostKeys/key_10.0.0.1.pub": []byte("ssh-rsa AAAA"),
		"Xshell/big.xsh":                      append(xshSample("10.5.5.5"), make([]byte, g_ciXshMaxBytes)...), // 한도 초과: 건너뛴다
	}
	slOrder := []string{"xts.zcf", "Xshell/default.xshf.xfs", "Xshell/web01.xsh", "Xshell/LAB/DB/db01.xsh", "Xshell/" + szKoreanDir + "/gw.xsh", "Xshell/../evil.xsh", "Xftp/files.xsh", "xsl/Sessions/note.xsh", "com/SECSH/HostKeys/key_10.0.0.1.pub", "Xshell/big.xsh"}
	slCands, errScan := ScanXshell(writeXts(t, mapFiles, slOrder))
	if errScan != nil {
		t.Fatal(errScan)
	}
	slGot := []string{}
	for _, cand := range slCands {
		slGot = append(slGot, strings.Join(append(append([]string{}, cand.M_slFolderPath...), cand.M_szName), "/")+"="+cand.M_host.M_szHost)
	}
	if szGot := strings.Join(slGot, " | "); szGot != "LAB/DB/db01=10.0.0.21 | web01=10.0.0.1 | 운영 센터/gw=10.0.0.254" {
		t.Fatalf("후보: %s", szGot)
	}
}

// zip 이 아닌 파일은 오류다.
func TestScanXshellBadFile(t *testing.T) {
	useKoreanImport(t)
	szPath := filepath.Join(t.TempDir(), "x.xts")
	if errWrite := os.WriteFile(szPath, []byte("not a zip"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	_, errScan := ScanXshell(szPath)
	if errScan == nil || !strings.HasPrefix(errScan.Error(), "Xshell 내보내기 파일을 열지 못했습니다: ") {
		t.Fatalf("오류: %v", errScan)
	}
}

// 문서 폴더에서 세션 폴더 찾기: 새 판(8)이 먼저, 옛 위치는 마지막. 없으면 빈 문자열.
func TestXshellSessionsUnder(t *testing.T) {
	szDocs := t.TempDir()
	if szDir := xshellSessionsUnder(szDocs); szDir != "" {
		t.Fatalf("없으면 빈 문자열: %q", szDir)
	}
	szOld := filepath.Join(szDocs, "NetSarang", "Xshell", "Sessions")
	sz7 := filepath.Join(szDocs, "NetSarang Computer", "7", "Xshell", "Sessions")
	sz8 := filepath.Join(szDocs, "NetSarang Computer", "8", "Xshell", "Sessions")
	for _, tc := range []struct{ szMake, szWant string }{{szOld, szOld}, {sz7, sz7}, {sz8, sz8}} {
		if errMkdir := os.MkdirAll(tc.szMake, 0o755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		if szDir := xshellSessionsUnder(szDocs); szDir != tc.szWant {
			t.Fatalf("%q 를 만든 뒤: %q", tc.szMake, szDir)
		}
	}
	if szDir := xshellSessionsUnder(""); szDir != "" {
		t.Fatalf("문서 폴더를 모르면 빈 문자열: %q", szDir)
	}
}

// 실제 내보내기 파일로 맞대어 보기(선택): 환경 변수 DARITERM_XTS 에 .xts 경로를 주면 읽어 수만 적는다.
// 파일의 내용(호스트·이름)은 출력하지 않는다. 변수가 없으면 건너뛴다.
func TestXshellRealExport(t *testing.T) {
	szPath := os.Getenv("DARITERM_XTS")
	if szPath == "" {
		t.Skip("DARITERM_XTS 가 없음")
	}
	slSessions, errRead := ReadXtsFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	slCands, errScan := ScanXshell(szPath)
	if errScan != nil {
		t.Fatal(errScan)
	}
	mapProtocol, mapEncoding, mapDepth, mapAuth := map[string]int{}, map[string]int{}, map[int]int{}, map[string]int{}
	iUser, iNotes, iNote, iBadName, iKeep60 := 0, 0, 0, 0, 0
	for _, cand := range slCands {
		mapProtocol[cand.M_host.M_szProtocol]++
		mapEncoding[cand.M_host.M_szEncoding]++
		mapDepth[len(cand.M_slFolderPath)]++
		mapAuth[cand.M_host.M_szAuthType]++
		if cand.M_host.M_szUsername != "" {
			iUser++
		}
		if cand.M_host.M_szNotes != "" {
			iNotes++
		}
		if cand.M_szNote != "" {
			iNote++
		}
		if cand.M_host.M_iKeepaliveSec == 60 {
			iKeep60++
		}
		for _, szPart := range append(append([]string{}, cand.M_slFolderPath...), cand.M_szName) {
			if strings.ContainsRune(szPart, '�') || strings.ContainsAny(szPart, "¼¿") {
				iBadName++
			}
		}
		if cand.M_host.M_szHost == "" || cand.M_host.M_iPort <= 0 {
			t.Fatalf("호스트·포트가 빈 후보가 있음 (%d번째)", len(mapProtocol))
		}
	}
	t.Logf("세션 파일 %d개 → 후보 %d개, 프로토콜 %v, 인코딩 %v, 폴더 깊이 %v, 인증 %v", len(slSessions), len(slCands), mapProtocol, mapEncoding, mapDepth, mapAuth)
	t.Logf("사용자 이름 있음 %d, 메모 있음 %d, 옮기지 않음 안내 %d, 연결 유지 60초 %d, 깨진 이름 %d", iUser, iNotes, iNote, iKeep60, iBadName)
	if iBadName != 0 {
		t.Fatal("폴더·세션 이름이 깨짐")
	}
}

// 포트 포워딩(FwdReq_{번호}_{항목}): 나가는 것은 L, 들어오는 것은 R, 듣는 주소 localhost 는 127.0.0.1,
// 설명은 서버 메모에 적고, 옮기지 못한 항목은 수를 알린다 (import.md §1.6, D129).
func TestParseXshForwards(t *testing.T) {
	useKoreanImport(t)
	session, _ := xshSessionOf("gw.xsh", xshBytes(
		"[CONNECTION]", "Host=gw.example", "Port=22", "Protocol=SSH", "Description=게이트웨이",
		"[CONNECTION:SSH]",
		// 항목의 줄은 실제 파일처럼 뒤섞여 있다.
		"FwdReq_1_HostPort=8443", "FwdReq_0_Source=localhost", "FwdReq_2_Incoming=1", "FwdReqCount=6",
		"FwdReq_0_Incoming=0", "FwdReq_0_Port=50093", "FwdReq_0_Host=10.1.0.93", "FwdReq_0_HostPort=22", "FwdReq_0_LocalOnly=0", "FwdReq_0_Description=DC1-APP01",
		"FwdReq_1_Incoming=0", "FwdReq_1_Port=50103", "FwdReq_1_Host=10.1.0.103", "FwdReq_1_Source=localhost", "FwdReq_1_LocalOnly=1",
		"FwdReq_2_Source=0.0.0.0", "FwdReq_2_Port=9000", "FwdReq_2_Host=127.0.0.1", "FwdReq_2_HostPort=3000", "FwdReq_2_Description=웹 미리보기",
		"FwdReq_3_Incoming=2", "FwdReq_3_Source=localhost", "FwdReq_3_Port=1080", // 모르는 종류(동적 등): 옮기지 않는다
		"FwdReq_4_Incoming=0", "FwdReq_4_Port=0", "FwdReq_4_Host=10.1.0.1", "FwdReq_4_HostPort=22", // 듣는 포트 없음
		"FwdReq_5_Incoming=0", "FwdReq_5_Port=50001", "FwdReq_5_Host=", "FwdReq_5_HostPort=22", // 대상 없음
		"FwdReq_6_Incoming=0", "FwdReq_6_Port=50002", "FwdReq_6_Host=10.1.0.2", "FwdReq_6_HostPort=22", // 개수 밖: 읽지 않는다
	))
	cand, bOk := ParseXshSession(session)
	if !bOk {
		t.Fatal("후보가 돼야 함")
	}
	slWant := []Forward{
		{M_szType: "L", M_szBindHost: "127.0.0.1", M_iBindPort: 50093, M_szDestHost: "10.1.0.93", M_iDestPort: 22},
		{M_szType: "L", M_szBindHost: "127.0.0.1", M_iBindPort: 50103, M_szDestHost: "10.1.0.103", M_iDestPort: 8443},
		{M_szType: "R", M_szBindHost: "0.0.0.0", M_iBindPort: 9000, M_szDestHost: "127.0.0.1", M_iDestPort: 3000},
	}
	if len(cand.M_host.M_slForwards) != len(slWant) {
		t.Fatalf("포워딩 수: %+v", cand.M_host.M_slForwards)
	}
	for iIndex, fwdWant := range slWant {
		if cand.M_host.M_slForwards[iIndex] != fwdWant {
			t.Fatalf("%d번째: %+v (기대 %+v)", iIndex, cand.M_host.M_slForwards[iIndex], fwdWant)
		}
	}
	const szWantNotes = "게이트웨이\n\n포트 포워딩 이름 (Xshell):\nL 50093 → 10.1.0.93:22  DC1-APP01\nR 9000 → 127.0.0.1:3000  웹 미리보기"
	if cand.M_host.M_szNotes != szWantNotes {
		t.Fatalf("메모:\n%q\n기대:\n%q", cand.M_host.M_szNotes, szWantNotes)
	}
	if cand.M_szNote != "포트 포워딩 3개는 옮기지 않음" {
		t.Fatalf("안내: %q", cand.M_szNote)
	}

	// 설명도 메모도 없으면 메모는 비고, 모두 옮겼으면 안내도 없다. 텔넷 세션의 포워딩은 읽지 않는다.
	session, _ = xshSessionOf("a.xsh", xshBytes("[CONNECTION]", "Host=h", "Protocol=SSH", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"))
	cand, _ = ParseXshSession(session)
	if len(cand.M_host.M_slForwards) != 1 || cand.M_host.M_szNotes != "" || cand.M_szNote != "" {
		t.Fatalf("이름 없는 포워딩: %+v / %q / %q", cand.M_host.M_slForwards, cand.M_host.M_szNotes, cand.M_szNote)
	}
	session, _ = xshSessionOf("t.xsh", xshBytes("[CONNECTION]", "Host=h", "Protocol=TELNET", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"))
	cand, _ = ParseXshSession(session)
	if len(cand.M_host.M_slForwards) != 0 || cand.M_szNote != "" {
		t.Fatalf("텔넷: %+v", cand)
	}
}

// 실제 세션 파일로 맞대어 보기(선택): 환경 변수 DARITERM_XSH 에 .xsh 경로를 주면 포워딩을 읽어 수만 적는다.
// 파일의 내용(호스트·이름)은 출력하지 않는다. 변수가 없으면 건너뛴다.
func TestXshellRealSessionForwards(t *testing.T) {
	szPath := os.Getenv("DARITERM_XSH")
	if szPath == "" {
		t.Skip("DARITERM_XSH 가 없음")
	}
	bRaw, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	session, bOk := xshSessionOf(filepath.Base(szPath), bRaw)
	if !bOk {
		t.Fatal("세션 파일이 아님")
	}
	cand, bOk := ParseXshSession(session)
	if !bOk {
		t.Fatal("후보가 돼야 함")
	}
	slForwards, slNames, iSkipped := ParseXshForwards(session.M_values)
	mapType, mapBind, mapPorts := map[string]int{}, map[string]int{}, map[int]int{}
	for _, fwd := range slForwards {
		mapType[fwd.M_szType]++
		mapBind[fwd.M_szBindHost]++
		mapPorts[fwd.M_iBindPort]++
		if fwd.M_szDestHost == "" || fwd.M_iDestPort < 1 || fwd.M_iBindPort < 1 {
			t.Fatal("빈 값이 든 포워딩이 있음")
		}
	}
	t.Logf("적힌 개수 %d → 포워딩 %d개 (종류 %v, 듣는 주소 %v, 듣는 포트 %d가지), 이름 있음 %d, 옮기지 못함 %d",
		session.M_values.num(g_cszXshSecSsh, "FwdReqCount"), len(slForwards), mapType, mapBind, len(mapPorts), len(slNames), iSkipped)
	t.Logf("후보: 포워딩 %d개, 메모 %d줄, 안내 있음 %v", len(cand.M_host.M_slForwards), strings.Count(cand.M_host.M_szNotes, "\n")+1, cand.M_szNote != "")
}

// 세션 파일(.xsh) 하나를 고르면 그 세션만, 폴더 없이 읽는다 (import.md §1.6, D129).
func TestScanXshellSingleSessionFile(t *testing.T) {
	szDir := t.TempDir()
	szPath := filepath.Join(szDir, "gw [LAB].XSH")
	if errWrite := os.WriteFile(szPath, xshSample("10.0.0.254", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	// 같은 폴더의 다른 세션은 읽지 않는다.
	if errWrite := os.WriteFile(filepath.Join(szDir, "other.xsh"), xshSample("10.0.0.1"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	slCands, errScan := ScanXshell(szPath)
	if errScan != nil {
		t.Fatal(errScan)
	}
	if len(slCands) != 1 || slCands[0].M_szName != "gw [LAB]" || slCands[0].M_host.M_szHost != "10.0.0.254" || len(slCands[0].M_slFolderPath) != 0 {
		t.Fatalf("후보: %+v", slCands)
	}
}
