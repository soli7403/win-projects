package session

import (
	"strings"
	"testing"

	"termgo/internal/model"
	"termgo/internal/transport"
)

func TestTmuxSessionName(t *testing.T) {
	cases := []struct {
		h    model.Host
		want string
	}{
		{model.Host{TmuxName: "work"}, "work"},
		{model.Host{Alias: "my server: prod.1"}, "my-server-prod-1"},
		{model.Host{Alias: "", Local: &transport.LocalOptions{Cwd: `C:\dev\proj\`}}, "proj"},
		{model.Host{}, "termgo"},
		{model.Host{Alias: "서버"}, "서버"},
	}
	for _, c := range cases {
		if got := TmuxSessionName(c.h); got != c.want {
			t.Errorf("name(%+v) = %q, want %q", c.h, got, c.want)
		}
	}
}

func TestKeepSessionApplies(t *testing.T) {
	ssh := model.Host{Protocol: "ssh", KeepSession: true}
	if !KeepSessionApplies(ssh, "windows") {
		t.Error("ssh must apply")
	}
	if KeepSessionApplies(model.Host{Protocol: "ssh"}, "windows") {
		t.Error("off by default")
	}
	ps := model.Host{Protocol: "local", KeepSession: true, Local: &transport.LocalOptions{Shell: "powershell.exe"}}
	if KeepSessionApplies(ps, "windows") {
		t.Error("windows powershell must not apply")
	}
	wsl := model.Host{Protocol: "local", KeepSession: true, Local: &transport.LocalOptions{Shell: "wsl.exe"}}
	if !KeepSessionApplies(wsl, "windows") {
		t.Error("windows wsl must apply")
	}
	if !KeepSessionApplies(model.Host{Protocol: "local", KeepSession: true}, "darwin") {
		t.Error("mac local must apply")
	}
	if KeepSessionApplies(model.Host{Protocol: "telnet", KeepSession: true}, "linux") {
		t.Error("telnet must not apply")
	}
}

func TestBuildTmuxCommandQuoting(t *testing.T) {
	c := BuildTmuxCommand("my work", "cd /var/www && echo 'hi'\nls")
	if !strings.HasPrefix(c, " sh -c '") || !strings.HasSuffix(c, "\r") {
		t.Fatalf("shape: %q", c)
	}
	if !strings.Contains(c, `'my work'`) {
		t.Errorf("name must be quoted: %q", c)
	}
	if !strings.Contains(c, `'cd /var/www && echo '\''hi'\''; ls'`) {
		t.Errorf("startup must be single-quote escaped and newline-joined: %q", c)
	}
	// 자동 실행 명령은 세션을 새로 만들 때 분기 안에서만 send-keys 된다
	if strings.Count(c, "send-keys") != 1 {
		t.Errorf("send-keys must appear once")
	}
}
