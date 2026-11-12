package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeKeys 는 볼트 대역이다.
type fakeKeys struct {
	bOpen   bool
	slSaved []string
}

func (pFake *fakeKeys) Unlocked() bool { return pFake.bOpen }

func (pFake *fakeKeys) SaveKey(szName string, szUsername string, szPrivateKey string) (string, error) {
	pFake.slSaved = append(pFake.slSaved, szName+"|"+szUsername+"|"+szPrivateKey)
	return "cred-" + szName, nil
}

// TestApplyCandidatesRules 은 폴더 재사용, 키 저장, 점프 연결, 건너뛰기 문구를 확인한다 (acceptance 04 I6~I8).
func TestApplyCandidatesRules(t *testing.T) {
	dir := t.TempDir()
	szKeyPath := filepath.Join(dir, "k.pem")
	if errWrite := os.WriteFile(szKeyPath, []byte("KEYDATA"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	slGroups := []map[string]any{{"id": "g-existing", "parentId": nil, "name": "회사", "env": "none", "sort": 0}}
	slHosts := []map[string]any{{"id": "h-old", "host": "203.0.113.10", "alias": "old"}}
	slCands := []Candidate{
		{M_szSource: "putty", M_szName: "prod-web01", M_host: HostDraft{M_szAlias: "prod-web01", M_szProtocol: "ssh", M_szHost: "10.0.0.1", M_iPort: 2222, M_szAuthType: "key"}, M_szKeyFile: szKeyPath, M_szJumpVia: "203.0.113.10"},
		{M_szSource: "ssh-config", M_szName: "dev-api", M_host: HostDraft{M_szAlias: "dev-api", M_szProtocol: "ssh", M_szHost: "10.0.0.2", M_szAuthType: "key"}, M_szKeyFile: filepath.Join(dir, "missing.pem")},
		{M_szSource: "putty", M_szName: "sw1", M_host: HostDraft{M_szAlias: "sw1", M_szProtocol: "serial", M_szHost: "COM3", M_pSerial: &Serial{M_szPath: "COM3", M_iBaudRate: 9600}}},
	}
	keys := &fakeKeys{bOpen: true}
	out, errApply := ApplyCandidates(slCands, "회사", slGroups, slHosts, keys)
	if errApply != nil {
		t.Fatal(errApply)
	}
	if len(out.Groups) != 1 {
		t.Fatalf("같은 이름 최상위 폴더를 재사용해야 함: %d", len(out.Groups))
	}
	if out.Result.M_iHosts != 3 || out.Result.M_iKeys != 1 {
		t.Fatalf("개수 불일치: %+v", out.Result)
	}
	if len(keys.slSaved) != 1 || !strings.HasPrefix(keys.slSaved[0], "prod-web01 키|") {
		t.Fatalf("키 저장 불일치: %v", keys.slSaved)
	}
	if out.Hosts[1]["env"] != "prod" || out.Hosts[1]["authType"] != "key" || out.Hosts[1]["credentialId"] != "cred-prod-web01 키" {
		t.Fatalf("prod 호스트 필드 불일치: %+v", out.Hosts[1])
	}
	if out.Hosts[1]["jumpHostId"] != "h-old" {
		t.Fatalf("기존 서버로 점프 연결 실패: %v", out.Hosts[1]["jumpHostId"])
	}
	if out.Hosts[2]["authType"] != "agent" || out.Hosts[2]["env"] != "dev" {
		t.Fatalf("키 파일 없음은 agent, dev 환경이어야 함: %+v", out.Hosts[2])
	}
	if out.Hosts[3]["env"] != "device" || out.Hosts[3]["serial"] == nil {
		t.Fatalf("시리얼 환경/설정 불일치: %+v", out.Hosts[3])
	}
	if len(out.Result.M_slSkipped) != 1 || !strings.Contains(out.Result.M_slSkipped[0], "키 파일을 읽지 못함") {
		t.Fatalf("건너뛰기 목록 불일치: %v", out.Result.M_slSkipped)
	}
}

// TestApplyLockedVaultUsesAgent 는 볼트가 잠겨 있으면 키를 복사하지 않고 agent 로 둔다 (acceptance 04 I8).
func TestApplyLockedVaultUsesAgent(t *testing.T) {
	dir := t.TempDir()
	szKeyPath := filepath.Join(dir, "k.pem")
	_ = os.WriteFile(szKeyPath, []byte("KEY"), 0o600)
	keys := &fakeKeys{bOpen: false}
	out, errApply := ApplyCandidates([]Candidate{{M_szName: "a", M_host: HostDraft{M_szProtocol: "ssh", M_szHost: "h", M_szAuthType: "key"}, M_szKeyFile: szKeyPath}}, "", nil, nil, keys)
	if errApply != nil {
		t.Fatal(errApply)
	}
	if len(keys.slSaved) != 0 || out.Hosts[0]["authType"] != "agent" || out.Hosts[0]["credentialId"] != nil {
		t.Fatalf("잠긴 볼트에 키가 복사됨: %+v", out.Hosts[0])
	}
	if len(out.Groups) != 0 || out.Hosts[0]["groupId"] != nil {
		t.Fatal("폴더 이름이 비면 최상위여야 함")
	}
}
