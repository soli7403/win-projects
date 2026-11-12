//go:build windows

package session

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"termgo/internal/transport"
)

func wslRun(args ...string) (string, error) {
	out, err := exec.Command("wsl.exe", append([]string{"-e"}, args...)...).CombinedOutput()
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\x00", "")), err
}

func startWSL(t *testing.T) *transport.Local {
	t.Helper()
	l, err := transport.StartLocal(transport.LocalOptions{Shell: "wsl.exe"}, "xterm-256color", 120, 30, transport.Callbacks{
		OnData: func([]byte) {}, OnClose: func(string) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func waitFor(t *testing.T, d time.Duration, f func() bool) bool {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if f() {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// 3.3 실제 tmux(WSL)로 확인: 새로 만들 때만 옵션·자동 실행 명령이 적용되고, 다시 붙을 때는 실행되지 않는다.
func TestTmuxKeepSessionInWSL(t *testing.T) {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		t.Skip("wsl 없음")
	}
	if out, err := wslRun("sh", "-c", "command -v tmux"); err != nil || out == "" {
		t.Skip("WSL 에 tmux 없음")
	}
	const name = "termgo-test-keep"
	const marker = "/tmp/termgo_tmux_marker"
	_, _ = wslRun("tmux", "kill-session", "-t", name)
	_, _ = wslRun("rm", "-f", marker)
	t.Cleanup(func() { _, _ = wslRun("tmux", "kill-session", "-t", name); _, _ = wslRun("rm", "-f", marker) })

	cmd := BuildTmuxCommand(name, "touch "+marker)

	// 1) 처음 만들기
	l1 := startWSL(t)
	time.Sleep(2 * time.Second)
	_ = l1.Write([]byte(cmd))
	if !waitFor(t, 15*time.Second, func() bool { _, err := wslRun("tmux", "has-session", "-t", name); return err == nil }) {
		t.Fatal("tmux 세션이 만들어지지 않음")
	}
	if !waitFor(t, 10*time.Second, func() bool { _, err := wslRun("test", "-f", marker); return err == nil }) {
		t.Fatal("자동 실행 명령이 세션 안에서 실행되지 않음")
	}
	if out, _ := wslRun("tmux", "show-options", "-t", name, "-v", "mouse"); out != "on" {
		t.Fatalf("mouse = %q", out)
	}
	if out, _ := wslRun("tmux", "show-options", "-t", name, "-v", "status"); out != "off" {
		t.Fatalf("status = %q", out)
	}
	l1.Close() // 연결(창)을 닫아도 세션은 남는다
	time.Sleep(1500 * time.Millisecond)
	if _, err := wslRun("tmux", "has-session", "-t", name); err != nil {
		t.Fatal("창을 닫았더니 tmux 세션이 사라짐")
	}

	// 2) 다시 붙기 — 자동 실행 명령은 실행되지 않아야 한다
	_, _ = wslRun("rm", "-f", marker)
	l2 := startWSL(t)
	defer l2.Close()
	time.Sleep(2 * time.Second)
	_ = l2.Write([]byte(cmd))
	time.Sleep(4 * time.Second)
	if _, err := wslRun("test", "-f", marker); err == nil {
		t.Fatal("다시 붙을 때 자동 실행 명령이 또 실행됨")
	}
	if out, _ := wslRun("tmux", "list-clients", "-t", name); out == "" {
		t.Fatal("다시 붙지 못함 (attach 된 클라이언트 없음)")
	}
}
