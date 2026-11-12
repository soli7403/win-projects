package importer

import (
	"testing"

	"golang.org/x/text/encoding/korean"
)

func TestParseSSHConfig(t *testing.T) {
	szCfg := `
Host bastion
  HostName 1.2.3.4
  User ops
  IdentityFile ~/.ssh/id_ed25519
  LocalForward 8080 localhost:80

Host app
  HostName 10.0.0.5
  ProxyJump bastion

Host *
  ServerAliveInterval 20
`
	c := ParseSSHConfig(szCfg, "/home/u")
	if len(c) != 2 {
		t.Fatalf("candidates = %d", len(c))
	}
	if c[0].KeyFile != "/home/u/.ssh/id_ed25519" || c[0].Host["username"] != "ops" {
		t.Fatalf("bastion = %+v", c[0])
	}
	if f := c[0].Host["forwards"].([]map[string]any); len(f) != 1 || f[0]["bindPort"] != 8080 || f[0]["destHost"] != "localhost" {
		t.Fatalf("forwards = %v", f)
	}
	if c[1].JumpVia != "1.2.3.4" {
		t.Fatalf("jumpVia = %q", c[1].JumpVia)
	}
}

func TestDecodeSessionName(t *testing.T) {
	arr, _ := korean.EUCKR.NewEncoder().Bytes([]byte("서버"))
	szEsc := ""
	for _, b := range arr {
		szEsc += "%" + string("0123456789ABCDEF"[b>>4]) + string("0123456789ABCDEF"[b&15])
	}
	if got := DecodeSessionName("prod-" + szEsc); got != "prod-서버" {
		t.Fatalf("got %q", got)
	}
}

func TestPuttyToCandidates(t *testing.T) {
	c := PuttyToCandidates(map[string]RegValues{
		"web":                {"HostName": "me@example.com", "PortNumber": 2222, "Protocol": "ssh", "LineCodePage": "CP949", "PortForwardings": "L8080=localhost:80"},
		"Default%20Settings": {},
	})
	if len(c) != 1 {
		t.Fatalf("n = %d", len(c))
	}
	h := c[0].Host
	if h["host"] != "example.com" || h["username"] != "me" || h["port"] != 2222 || h["encoding"] != "cp949" {
		t.Fatalf("host = %v", h)
	}
}

func TestEc2Bastion(t *testing.T) {
	mk := func(id, name, pub, priv string) Ec2 {
		e := Ec2{InstanceId: id, PublicIpAddress: pub, PrivateIpAddress: priv, VpcId: "v"}
		e.State.Name = "running"
		e.Tags = []struct{ Key, Value string }{{"Name", name}}
		return e
	}
	c := Ec2ToCandidates("ap-northeast-2", []Ec2{mk("i-1", "shop-bastion", "3.3.3.3", "10.0.0.1"), mk("i-2", "shop-app", "", "10.0.0.2")}, nil)
	if len(c) != 2 || c[1].JumpVia != "3.3.3.3" || c[0].JumpVia != "" {
		t.Fatalf("c = %+v", c)
	}
}
