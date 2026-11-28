package importer

import (
	"strings"
	"testing"

	"dariterm/internal/i18n"
)

// 가져오기의 로그인 스크립트 (import.md §1.5·§2, D150).

// SecureCRT 의 로그온 동작은 켜져 있고 내용이 있을 때만 "옮기지 않음" 안내를 붙이고, 단계는 만들지 않는다.
func TestCrtLoginScriptDetected(t *testing.T) {
	i18n.SetLang("ko")
	szBase := "S:\"Hostname\"=10.0.0.1\r\nS:\"Protocol Name\"=SSH2\r\n"
	cases := []struct {
		szName string
		szIni  string
		bNote  bool
	}{
		{"꺼짐", szBase + "D:\"Use Login Script\"=00000000\r\nZ:\"Login Script V4\"=00000000\r\n", false},
		{"켜졌지만 빈 값", szBase + "D:\"Use Login Script\"=00000001\r\nZ:\"Login Script V4\"=00000000\r\n", false},
		{"켜지고 내용 있음", szBase + "D:\"Use Login Script\"=00000001\r\nZ:\"Login Script V4\"=00000002\r\n login:\r\n kai\r\n", true},
	}
	for _, c := range cases {
		cand, bOk := ParseCrtSession(CrtSession{M_szName: "s", M_values: ParseCrtIni([]byte(c.szIni))}, "", "")
		if !bOk {
			t.Fatalf("%s: 후보가 없다", c.szName)
		}
		if bGot := strings.Contains(cand.M_szNote, "로그인 스크립트"); bGot != c.bNote {
			t.Errorf("%s: 안내 = %q", c.szName, cand.M_szNote)
		}
		if len(cand.M_host.M_slLoginScript) != 0 {
			t.Errorf("%s: 단계를 만들면 안 된다: %+v", c.szName, cand.M_host.M_slLoginScript)
		}
	}
}

// 후보에 단계가 있으면 서버 loginScript 로 옮기고, 단계마다 서로 다른 새 id 를 준다.
func TestApplyCarriesLoginScript(t *testing.T) {
	slCands := []Candidate{{M_szSource: "xshell", M_szName: "web", M_host: HostDraft{
		M_szAlias: "web", M_szProtocol: "ssh", M_szHost: "10.0.0.9", M_szAuthType: "password",
		M_slLoginScript: []LoginStep{{M_szExpect: "$ ", M_szSend: "sudo -i"}, {M_szExpect: "# ", M_szSend: "cd /srv"}},
	}}}
	out, errApply := ApplyCandidates(slCands, "가져옴", nil, nil, &fakeKeys{bOpen: true})
	if errApply != nil {
		t.Fatal(errApply)
	}
	slSteps, bOk := out.Hosts[0]["loginScript"].([]map[string]any)
	if !bOk || len(slSteps) != 2 {
		t.Fatalf("loginScript = %#v", out.Hosts[0]["loginScript"])
	}
	if slSteps[0]["expect"] != "$ " || slSteps[0]["send"] != "sudo -i" || slSteps[1]["send"] != "cd /srv" {
		t.Fatalf("단계 = %+v", slSteps)
	}
	if slSteps[0]["id"] == "" || slSteps[0]["id"] == slSteps[1]["id"] {
		t.Fatalf("단계 id 가 서로 달라야 한다: %v %v", slSteps[0]["id"], slSteps[1]["id"])
	}
}
