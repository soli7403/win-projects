package importer

import (
	"strings"
	"testing"

	"dariterm/internal/i18n"
)

// ssh config 의 ProxyJump 여러 단계(D153)와 ProxyCommand(D152).

const g_cszChainConfig = `Host a
  HostName 10.0.0.1
Host b
  HostName 10.0.0.2
Host target
  HostName 10.0.0.3
  ProxyJump a, b
Host pc
  HostName 10.0.0.4
  ProxyCommand ssh -W %h:%p a
Host both
  HostName 10.0.0.5
  ProxyJump a
  ProxyCommand nc %h %p
`

func candByName(t *testing.T, slCands []Candidate, szName string) Candidate {
	t.Helper()
	for _, cand := range slCands {
		if cand.M_szName == szName {
			return cand
		}
	}
	t.Fatalf("후보 %s 없음", szName)
	return Candidate{}
}

func hostByAlias(t *testing.T, slHosts []map[string]any, szAlias string) map[string]any {
	t.Helper()
	for _, mapHost := range slHosts {
		if mapHost["alias"] == szAlias {
			return mapHost
		}
	}
	t.Fatalf("서버 %s 없음", szAlias)
	return nil
}

func TestSSHConfigJumpChainAndProxyCommand(t *testing.T) {
	slCands := ParseSSHConfig(g_cszChainConfig, "C:/Users/tester")
	target := candByName(t, slCands, "target")
	if target.M_szJumpVia != "10.0.0.2" || strings.Join(target.M_slJumpChain, ",") != "10.0.0.1,10.0.0.2" {
		t.Fatalf("target: jumpVia=%q chain=%v", target.M_szJumpVia, target.M_slJumpChain)
	}
	pc := candByName(t, slCands, "pc")
	if pc.M_host.M_pProxy == nil || pc.M_host.M_pProxy.M_szType != "command" || pc.M_host.M_pProxy.M_szCommand != "ssh -W %h:%p a" {
		t.Fatalf("pc: proxy=%+v", pc.M_host.M_pProxy)
	}
	both := candByName(t, slCands, "both")
	if both.M_host.M_pProxy != nil || both.M_szJumpVia != "10.0.0.1" || len(both.M_slJumpChain) != 0 {
		t.Fatalf("both: ProxyJump 만이어야 한다: %+v", both)
	}
}

func TestApplyJumpChainLinksEachHop(t *testing.T) {
	slCands := ParseSSHConfig(g_cszChainConfig, "C:/Users/tester")
	out, errApply := ApplyCandidates(slCands, "ssh", nil, nil, &fakeKeys{bOpen: true})
	if errApply != nil {
		t.Fatal(errApply)
	}
	a, b, target := hostByAlias(t, out.Hosts, "a"), hostByAlias(t, out.Hosts, "b"), hostByAlias(t, out.Hosts, "target")
	if target["jumpHostId"] != b["id"] || b["jumpHostId"] != a["id"] || a["jumpHostId"] != nil {
		t.Fatalf("체인: target→%v b→%v a→%v (a=%v b=%v)", target["jumpHostId"], b["jumpHostId"], a["jumpHostId"], a["id"], b["id"])
	}
	pc := hostByAlias(t, out.Hosts, "pc")
	if mapProxy, _ := pc["proxy"].(map[string]any); mapProxy["type"] != "command" || mapProxy["command"] != "ssh -W %h:%p a" {
		t.Fatalf("pc proxy = %#v", pc["proxy"])
	}
	if len(out.Result.M_slSkipped) != 0 {
		t.Fatalf("건너뛴 것이 없어야 한다: %v", out.Result.M_slSkipped)
	}
}

func TestApplyJumpChainKeepsExistingHosts(t *testing.T) {
	i18n.SetLang("ko")
	slOld := []map[string]any{{"id": "h-b", "host": "10.0.0.2", "alias": "old-b", "jumpHostId": nil}}
	slCands := []Candidate{
		candByName(t, ParseSSHConfig(g_cszChainConfig, "C:/Users/tester"), "a"),
		candByName(t, ParseSSHConfig(g_cszChainConfig, "C:/Users/tester"), "target"),
	}
	out, errApply := ApplyCandidates(slCands, "ssh", nil, slOld, &fakeKeys{bOpen: true})
	if errApply != nil {
		t.Fatal(errApply)
	}
	target := hostByAlias(t, out.Hosts, "target")
	if target["jumpHostId"] != "h-b" {
		t.Fatalf("target 은 기존 b 로: %v", target["jumpHostId"])
	}
	if hostByAlias(t, out.Hosts, "old-b")["jumpHostId"] != nil {
		t.Fatal("기존 서버는 바꾸지 않는다")
	}
	if len(out.Result.M_slSkipped) != 1 || !strings.Contains(out.Result.M_slSkipped[0], "old-b") {
		t.Fatalf("안내 하나: %v", out.Result.M_slSkipped)
	}
}

func TestApplyJumpChainMissingHop(t *testing.T) {
	i18n.SetLang("ko")
	slAll := ParseSSHConfig(g_cszChainConfig, "C:/Users/tester")
	slCands := []Candidate{candByName(t, slAll, "b"), candByName(t, slAll, "target")}
	out, errApply := ApplyCandidates(slCands, "ssh", nil, nil, &fakeKeys{bOpen: true})
	if errApply != nil {
		t.Fatal(errApply)
	}
	if b := hostByAlias(t, out.Hosts, "b"); b["jumpHostId"] != nil {
		t.Fatalf("b 는 바로 연결: %v", b["jumpHostId"])
	}
	if len(out.Result.M_slSkipped) != 1 || !strings.Contains(out.Result.M_slSkipped[0], "10.0.0.1") {
		t.Fatalf("찾지 못함 안내: %v", out.Result.M_slSkipped)
	}
}
