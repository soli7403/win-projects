package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCrt 는 세션 폴더 아래에 BOM·CRLF 를 붙인 세션 파일을 쓴다.
func writeCrt(t *testing.T, szRoot string, szRel string, slLines ...string) {
	t.Helper()
	szPath := filepath.Join(szRoot, filepath.FromSlash(szRel))
	if errMkdir := os.MkdirAll(filepath.Dir(szPath), 0o755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	bText := append([]byte{0xEF, 0xBB, 0xBF}, []byte(strings.Join(slLines, "\r\n")+"\r\n")...)
	if errWrite := os.WriteFile(szPath, bText, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
}

// TestParseCrtIni 는 S/D/Z 형과 B 형 건너뛰기를 확인한다.
func TestParseCrtIni(t *testing.T) {
	values := ParseCrtIni([]byte("\xEF\xBB\xBFS:\"Hostname\"=10.0.0.1\r\nB:\"Font\"=00000004\r\n 00 01 02 03\r\nD:\"[SSH2] Port\"=00000016\r\nZ:\"Description\"=00000002\r\n 첫 줄\r\n 둘째 줄\r\nS:\"Username\"=ops\r\n"))
	if values.str("Hostname") != "10.0.0.1" || values.str("Username") != "ops" {
		t.Fatalf("S 형 불일치: %+v", values.mapStr)
	}
	if iPort, _ := values.num("[SSH2] Port"); iPort != 22 {
		t.Fatalf("D 형(16진) 불일치: %d", iPort)
	}
	if slLines := values.lines("Description"); len(slLines) != 2 || slLines[1] != "둘째 줄" {
		t.Fatalf("Z 형 불일치: %q", slLines)
	}
}

// TestScanSecureCrtFolders 는 폴더 구조·건너뛰는 파일·필드 변환·포워딩·경유 세션을 확인한다 (import.md §1.5).
func TestScanSecureCrtFolders(t *testing.T) {
	szConfig := t.TempDir()
	szSessions := filepath.Join(szConfig, "Sessions")
	writeCrt(t, szSessions, "Default.ini", `S:"Protocol Name"=SSH2`, `S:"Hostname"=default.example`)
	writeCrt(t, szSessions, "MDS/__FolderData__.ini", `S:"Hostname"=folder.example`)
	writeCrt(t, szSessions, "MDS/web01(213).ini",
		`S:"Protocol Name"=SSH2`, `S:"Hostname"=10.0.0.13`, `S:"Username"=ops`, `D:"[SSH2] Port"=00000016`,
		`S:"Output Transformer Name"=UTF-8`, `D:"Idle NO-OP Check"=00000001`, `D:"Idle NO-OP Timeout"=0000003c`,
		`D:"Use Global Public Key"=00000001`, `S:"Firewall Name"=None`, `S:"Password V2"=02:secret`)
	writeCrt(t, szSessions, "mds-테스트/gw_tunneling.ini",
		`S:"Protocol Name"=SSH2`, `S:"Hostname"=203.0.113.5`, `D:"[SSH2] Port"=00005e4c`,
		`D:"Use Global Public Key"=00000000`, `S:"Identity Filename V2"=${VDS_CONFIG_PATH}\gw.pem::rawkey`,
		`Z:"Port Forward Table V2"=00000003`, ` app|25095|1|10.0.5.95|22||`, ` db|0.0.0.0:35101|1||20300||`, ` bad|x|1|h|22||`,
		`Z:"Reverse Forward Table V2"=00000000`, `Z:"Description"=00000001`, ` 터널용 `)
	writeCrt(t, szSessions, "mds-테스트/inner/app.ini",
		`S:"Protocol Name"=SSH2`, `S:"Hostname"=10.0.5.95`, `S:"Firewall Name"=Session:mds-테스트\gw_tunneling`)
	writeCrt(t, szSessions, "rlogin.ini", `S:"Protocol Name"=RLogin`, `S:"Hostname"=10.0.0.9`)
	writeCrt(t, szSessions, "empty.ini", `S:"Protocol Name"=SSH2`)
	writeCrt(t, szSessions, "fw.ini", `S:"Protocol Name"=Telnet`, `S:"Hostname"=10.0.0.7`, `S:"Firewall Name"=CorpSocks`)

	slCands := ScanSecureCrt(szConfig)
	if len(slCands) != 4 {
		t.Fatalf("후보 4개(web01, gw, app, fw) 여야 함: %+v", slCands)
	}
	mapByName := map[string]Candidate{}
	for _, cand := range slCands {
		if cand.M_szSource != G_cszSourceSecureCrt {
			t.Fatalf("출처 불일치: %s", cand.M_szSource)
		}
		mapByName[cand.M_szName] = cand
	}

	web := mapByName["web01(213)"]
	if strings.Join(web.M_slFolderPath, "/") != "MDS" || web.M_host.M_szHost != "10.0.0.13" || web.M_host.M_iPort != 22 || web.M_host.M_szUsername != "ops" {
		t.Fatalf("web01 불일치: %+v", web)
	}
	if web.M_host.M_iKeepaliveSec != 60 || web.M_host.M_szAuthType != "password" || web.M_szKeyFile != "" || web.M_host.M_szTermType != "" {
		t.Fatalf("web01 연결 유지/인증/터미널 불일치: %+v", web.M_host)
	}

	gw := mapByName["gw_tunneling"]
	if strings.Join(gw.M_slFolderPath, "/") != "mds-테스트" || gw.M_host.M_iPort != 24140 || gw.M_host.M_iKeepaliveSec != 30 {
		t.Fatalf("gw 불일치: %+v", gw)
	}
	if gw.M_host.M_szAuthType != "key" || gw.M_szKeyFile != szConfig+`\gw.pem` {
		t.Fatalf("gw 키 파일 불일치: %q", gw.M_szKeyFile)
	}
	if len(gw.M_host.M_slForwards) != 2 {
		t.Fatalf("포워딩 2개여야 함(해석 못 하는 줄 제외): %+v", gw.M_host.M_slForwards)
	}
	fwd0, fwd1 := gw.M_host.M_slForwards[0], gw.M_host.M_slForwards[1]
	if fwd0 != (Forward{M_szType: "L", M_szBindHost: "127.0.0.1", M_iBindPort: 25095, M_szDestHost: "10.0.5.95", M_iDestPort: 22}) {
		t.Fatalf("첫 포워딩 불일치: %+v", fwd0)
	}
	if fwd1 != (Forward{M_szType: "L", M_szBindHost: "0.0.0.0", M_iBindPort: 35101, M_szDestHost: "localhost", M_iDestPort: 20300}) {
		t.Fatalf("둘째 포워딩 불일치: %+v", fwd1)
	}
	if gw.M_host.M_szNotes != "터널용" {
		t.Fatalf("메모 불일치: %q", gw.M_host.M_szNotes)
	}

	app := mapByName["app"]
	if strings.Join(app.M_slFolderPath, "/") != "mds-테스트/inner" || app.M_szJumpVia != "203.0.113.5" {
		t.Fatalf("경유 세션 불일치: %+v", app)
	}

	fw := mapByName["fw"]
	if fw.M_host.M_szProtocol != "telnet" || fw.M_host.M_iPort != 23 || len(fw.M_slFolderPath) != 0 || !strings.Contains(fw.M_szNote, "CorpSocks") {
		t.Fatalf("텔넷/전역 방화벽 메모 불일치: %+v", fw)
	}
}

// TestScanSecureCrtPickSessionsDir 는 Sessions 폴더를 직접 골라도 읽고, 없는 폴더는 빈 목록인지 확인한다.
func TestScanSecureCrtPickSessionsDir(t *testing.T) {
	szSessions := t.TempDir()
	writeCrt(t, szSessions, "a.ini", `S:"Protocol Name"=SSH2`, `S:"Hostname"=10.0.0.1`)
	if slCands := ScanSecureCrt(szSessions); len(slCands) != 1 || len(slCands[0].M_slFolderPath) != 0 {
		t.Fatalf("세션 폴더 직접 선택 실패: %+v", slCands)
	}
	if slCands := ScanSecureCrt(filepath.Join(szSessions, "none")); len(slCands) != 0 {
		t.Fatalf("없는 폴더는 빈 목록이어야 함: %+v", slCands)
	}
}

// TestCrtSerialAndEucKr 은 시리얼 세션과 EUC-KR 파일 해석을 확인한다.
func TestCrtSerialAndEucKr(t *testing.T) {
	cand, bOk := ParseCrtSession(CrtSession{M_szName: "콘솔", M_values: ParseCrtIni([]byte(
		"S:\"Protocol Name\"=Serial\r\nS:\"Com Port\"=COM3\r\nD:\"Baud Rate\"=0001c200\r\nD:\"Parity\"=00000002\r\nD:\"Stop Bits\"=00000001\r\nZ:\"Description\"=00000001\r\n \xc7\xd1\xb1\xdb\r\n"))}, "", "")
	if !bOk || cand.M_host.M_pSerial == nil {
		t.Fatal("시리얼 세션이 건너뛰어짐")
	}
	serial := *cand.M_host.M_pSerial
	if serial.M_szPath != "COM3" || serial.M_iBaudRate != 115200 || serial.M_iDataBits != 8 || serial.M_szParity != "even" || serial.M_fStopBits != 1.5 {
		t.Fatalf("시리얼 불일치: %+v", serial)
	}
	if cand.M_host.M_szNotes != "한글" {
		t.Fatalf("EUC-KR 메모 불일치: %q", cand.M_host.M_szNotes)
	}
}
