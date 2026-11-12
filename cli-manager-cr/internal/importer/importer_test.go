package importer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPuttySshSession 은 SSH 세션의 호스트·사용자·포트·포워딩·인증을 확인한다 (acceptance 04 I1, I2).
func TestPuttySshSession(t *testing.T) {
	mapValues := map[string]any{
		"Protocol":        "ssh",
		"HostName":        "ops@prod-web01",
		"PortNumber":      uint32(2222),
		"PortForwardings": "L8080=localhost:80,D1080",
		"TerminalType":    "xterm",
		"PingInterval":    uint32(0),
		"LineCodePage":    "UTF-8",
	}
	cand, bOk := ParsePuttySession("prod-web01", mapValues)
	if !bOk {
		t.Fatal("SSH 세션이 건너뛰어짐")
	}
	if cand.M_host.M_szHost != "prod-web01" || cand.M_host.M_iPort != 2222 || cand.M_host.M_szUsername != "ops" {
		t.Fatalf("호스트/사용자/포트 불일치: %+v", cand.M_host)
	}
	if len(cand.M_host.M_slForwards) != 1 || cand.M_host.M_slForwards[0].M_szType != "L" || cand.M_host.M_slForwards[0].M_iBindPort != 8080 {
		t.Fatalf("포워딩 불일치(D 는 제외): %+v", cand.M_host.M_slForwards)
	}
	if cand.M_host.M_szTermType != "xterm-256color" || cand.M_host.M_iKeepaliveSec != 30 {
		t.Fatalf("터미널/연결 유지 기본값 불일치: %+v", cand.M_host)
	}
	if cand.M_host.M_szAuthType != "password" || cand.M_szSource != G_cszSourcePutty {
		t.Fatalf("인증/출처 불일치")
	}
}

// TestPuttySkipRules 는 건너뛰는 세션을 확인한다.
func TestPuttySkipRules(t *testing.T) {
	if _, bOk := ParsePuttySession("Default Settings", map[string]any{"Protocol": "ssh"}); bOk {
		t.Fatal("기본 설정 세션은 건너뛰어야 함")
	}
	if _, bOk := ParsePuttySession("empty", map[string]any{"Protocol": "ssh"}); bOk {
		t.Fatal("SSH 호스트가 비면 건너뛰어야 함")
	}
	if _, bOk := ParsePuttySession("telnet-empty", map[string]any{"Protocol": "telnet"}); bOk {
		t.Fatal("텔넷 호스트가 비면 건너뛰어야 함")
	}
}

// TestPuttySerial 은 시리얼 세션 변환을 확인한다 (acceptance 04 I3).
func TestPuttySerial(t *testing.T) {
	cand, bOk := ParsePuttySession("switch-1", map[string]any{
		"Protocol":           "serial",
		"SerialLine":         "COM3",
		"SerialSpeed":        uint32(115200),
		"SerialParity":       uint32(2),
		"SerialStopHalfbits": uint32(3),
		"SerialFlowControl":  uint32(2),
	})
	if !bOk {
		t.Fatal("시리얼 세션이 건너뛰어짐")
	}
	if cand.M_host.M_szProtocol != "serial" || cand.M_host.M_pSerial == nil {
		t.Fatalf("시리얼 프로토콜 불일치: %+v", cand.M_host)
	}
	pSerial := cand.M_host.M_pSerial
	if pSerial.M_szPath != "COM3" || pSerial.M_iBaudRate != 115200 || pSerial.M_szParity != "even" || pSerial.M_fStopBits != 1.5 || pSerial.M_szFlowControl != "rtscts" {
		t.Fatalf("시리얼 값 불일치: %+v", pSerial)
	}
}

// TestPuttyTelnetDefaultPort 은 텔넷 기본 포트 23 과 이름 디코딩을 확인한다.
func TestPuttyTelnetDefaultPort(t *testing.T) {
	cand, bOk := ParsePuttySession("old%20box", map[string]any{"Protocol": "telnet", "HostName": "10.0.0.5"})
	if !bOk || cand.M_host.M_iPort != 23 || cand.M_szName != "old box" {
		t.Fatalf("텔넷 기본 포트/이름 불일치: %+v %q", cand.M_host, cand.M_szName)
	}
	if DecodePuttyName("%EC%84%9C%EB%B2%84") != "서버" {
		t.Fatal("UTF-8 퍼센트 디코딩 실패")
	}
}

// TestSshConfigRules 는 와일드카드 제외, ProxyJump, 포워딩, 첫 값 우선을 확인한다 (acceptance 04 I4, I5).
func TestSshConfigRules(t *testing.T) {
	szText := `
# 전역
Host *
  ServerAliveInterval 60
Host bastion
  HostName 203.0.113.9
  User ec2-user
Host web1
  HostName 10.0.0.11
  User root
  User other
  IdentityFile "~/.ssh/web.pem"
  ProxyJump bastion
  LocalForward 8080 localhost:80
  RemoteForward [::1]:9000 10.0.0.1:9001
Match host foo
  Port 9999
Host db
  ProxyJump ops@10.9.9.9:2200
  Port 2200
`
	slCands := ParseSSHConfig(szText, "C:/Users/tester")
	mapByName := map[string]Candidate{}
	for _, cand := range slCands {
		mapByName[cand.M_szName] = cand
	}
	if _, bWild := mapByName["*"]; bWild {
		t.Fatal("와일드카드 Host 가 후보에 있음")
	}
	web1 := mapByName["web1"]
	if web1.M_host.M_szHost != "10.0.0.11" || web1.M_host.M_szUsername != "root" {
		t.Fatalf("web1 호스트/사용자 불일치(첫 값 우선): %+v", web1.M_host)
	}
	if web1.M_szJumpVia != "203.0.113.9" {
		t.Fatalf("같은 파일 ProxyJump 가 HostName 으로 풀리지 않음: %q", web1.M_szJumpVia)
	}
	if web1.M_szKeyFile != filepath.Join("C:/Users/tester", ".ssh/web.pem") || web1.M_host.M_szAuthType != "key" {
		t.Fatalf("키 파일 해석 불일치: %q", web1.M_szKeyFile)
	}
	if len(web1.M_host.M_slForwards) != 2 || web1.M_host.M_slForwards[0].M_iBindPort != 8080 || web1.M_host.M_slForwards[1].M_szBindHost != "::1" {
		t.Fatalf("포워딩 해석 불일치: %+v", web1.M_host.M_slForwards)
	}
	if web1.M_host.M_iKeepaliveSec != 0 {
		t.Fatalf("전역 Host * 옵션이 블록에 섞임: %d", web1.M_host.M_iKeepaliveSec)
	}
	if _, bMatch := mapByName["foo"]; bMatch {
		t.Fatal("Match 뒤 값이 블록에 속함")
	}
	db := mapByName["db"]
	if db.M_szJumpVia != "10.9.9.9" || db.M_host.M_iPort != 2200 {
		t.Fatalf("직접 ProxyJump 해석 불일치: %q %d", db.M_szJumpVia, db.M_host.M_iPort)
	}
}

// TestSshConfigMissingFile 은 config 가 없으면 빈 목록이다.
func TestSshConfigMissingFile(t *testing.T) {
	if slCands := ScanSshConfig(t.TempDir()); len(slCands) != 0 {
		t.Fatalf("없는 config 에 후보가 있음: %d", len(slCands))
	}
}

// TestAwsParsers 는 리전·인스턴스·이미지 파싱과 로그인 사용자 규칙을 확인한다.
func TestAwsParsers(t *testing.T) {
	slRegions, errRegions := ParseAwsRegions([]byte(`{"Regions":[{"RegionName":"ap-northeast-2"},{"RegionName":"us-east-1"}]}`))
	if errRegions != nil || len(slRegions) != 2 {
		t.Fatalf("리전 파싱 실패: %v %v", slRegions, errRegions)
	}
	if _, errBad := ParseAwsRegions([]byte("not json")); errBad == nil {
		t.Fatal("깨진 리전 출력은 오류여야 함")
	}
	bInst := []byte(`{"Reservations":[{"Instances":[
	 {"InstanceId":"i-1","ImageId":"ami-u","KeyName":"k1","Platform":"","State":{"Name":"running"},"PublicIpAddress":"1.1.1.1","VpcId":"vpc-1","Tags":[{"Key":"Name","Value":"web-app1"}]},
	 {"InstanceId":"i-2","ImageId":"ami-w","State":{"Name":"running"},"Platform":"windows","PublicIpAddress":"2.2.2.2","VpcId":"vpc-1"},
	 {"InstanceId":"i-3","ImageId":"ami-u","State":{"Name":"terminated"},"PublicIpAddress":"3.3.3.3"},
	 {"InstanceId":"i-4","ImageId":"ami-u","KeyName":"k1","Platform":"","State":{"Name":"running"},"PrivateIpAddress":"10.0.0.4","VpcId":"vpc-1","Tags":[{"Key":"Name","Value":"web-app2"}]},
	 {"InstanceId":"i-5","ImageId":"ami-u","Platform":"","State":{"Name":"running"},"PublicIpAddress":"4.4.4.4","VpcId":"vpc-1","Tags":[{"Key":"Name","Value":"bastion"}]}
	]}]}`)
	slInst, errInst := ParseAwsInstances(bInst, "ap-northeast-2")
	if errInst != nil || len(slInst) != 4 {
		t.Fatalf("인스턴스 파싱 개수 불일치(종료 제외): %d %v", len(slInst), errInst)
	}
	if slInst[1].M_szName != "i-2" {
		t.Fatalf("이름 없으면 id 사용 규칙 불일치: %q", slInst[1].M_szName)
	}
	mapImages, _ := ParseAwsImages([]byte(`{"Images":[{"ImageId":"ami-u","Name":"ubuntu/images/hvm-ssd"},{"ImageId":"ami-x","Name":"debian-12"}]}`))
	if AwsLoginUser(mapImages["ami-u"]) != "ubuntu" || AwsLoginUser("Debian-12") != "admin" || AwsLoginUser("amzn2") != "ec2-user" || AwsLoginUser("Rocky-9") != "rocky" {
		t.Fatal("로그인 사용자 규칙 불일치")
	}
}

// TestAwsBastionPriority 는 베스천 추정 우선순위를 확인한다 (acceptance 04 I9).
func TestAwsBastionPriority(t *testing.T) {
	slAll := []AwsInstance{
		{M_szId: "i-a", M_szName: "web-app1", M_szVpcId: "vpc-1"},
		{M_szId: "i-b", M_szName: "shared", M_szVpcId: "vpc-1", M_szPublicIp: "5.5.5.5"},
		{M_szId: "i-c", M_szName: "web-bastion", M_szVpcId: "vpc-1", M_szPublicIp: "6.6.6.6"},
		{M_szId: "i-d", M_szName: "web-api", M_szVpcId: "vpc-1", M_szPublicIp: "7.7.7.7"},
	}
	// 줄기가 web 인 대상: 줄기 같고 이름에 bastion 이 든 web-bastion 이 이긴다.
	pPick := PickAwsBastion(slAll[0], slAll)
	if pPick == nil || pPick.M_szId != "i-c" {
		t.Fatalf("1순위 베스천 선택 실패: %+v", pPick)
	}
	if pNone := PickAwsBastion(AwsInstance{M_szId: "i-z", M_szVpcId: "vpc-9"}, slAll); pNone != nil {
		t.Fatal("다른 VPC 베스천을 고르면 안 됨")
	}
	pFirst := PickAwsBastion(AwsInstance{M_szId: "i-q", M_szName: "db", M_szVpcId: "vpc-1"}, slAll)
	if pFirst == nil || pFirst.M_szId != "i-c" {
		t.Fatalf("3순위(이름에 bastion) 선택 실패: %+v", pFirst)
	}
}

// TestBuildAwsCandidatesNotes 는 후보 필드와 키 파일 없음 표기를 확인한다.
func TestBuildAwsCandidatesNotes(t *testing.T) {
	home := t.TempDir()
	if errMk := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); errMk != nil {
		t.Fatal(errMk)
	}
	if errWrite := os.WriteFile(filepath.Join(home, ".ssh", "found.pem"), []byte("k"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	slInst := []AwsInstance{
		{M_szRegion: "us-east-1", M_szId: "i-1", M_szName: "api1", M_szState: "stopped", M_szKeyName: "missing", M_szPublicIp: "8.8.8.8", M_szImageId: "ami-u"},
		{M_szRegion: "us-east-1", M_szId: "i-2", M_szName: "api2", M_szState: "running", M_szKeyName: "found", M_szPublicIp: "9.9.9.9"},
		{M_szRegion: "us-east-1", M_szId: "i-3", M_szName: "noip", M_szState: "running"},
	}
	slCands := BuildAwsCandidates(slInst, map[string]string{"ami-u": "ubuntu"}, home)
	if len(slCands) != 2 {
		t.Fatalf("공인 IP 없는 인스턴스 제외 실패: %d", len(slCands))
	}
	if slCands[0].M_szNote != "us-east-1 · 중지됨 · 키 파일 없음: missing.pem" || slCands[0].M_host.M_szUsername != "ubuntu" {
		t.Fatalf("중지·키 없음 표기 불일치: %q", slCands[0].M_szNote)
	}
	if slCands[1].M_szKeyFile == "" || slCands[1].M_host.M_szNotes != "AWS us-east-1 i-2 (키: found)" {
		t.Fatalf("키 파일·메모 불일치: %q %q", slCands[1].M_szKeyFile, slCands[1].M_host.M_szNotes)
	}
}
