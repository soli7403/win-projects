package session

import (
	"os/exec"
	"strings"
	"testing"
)

// tmux 가 없는 환경에서 명령이 안내만 출력하고 정상 종료하는지 (POSIX sh 가 있을 때).
func TestTmuxCommandWithoutTmux(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh 없음")
	}
	if _, err := exec.LookPath("tmux"); err == nil {
		t.Skip("tmux 가 설치된 환경")
	}
	cmd := strings.TrimSuffix(strings.TrimPrefix(BuildTmuxCommand("w", "echo hi"), " "), "\r")
	out, err := exec.Command(sh, "-c", cmd).CombinedOutput()
	if err != nil {
		t.Fatalf("exit: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "tmux 가 없어") {
		t.Fatalf("out = %q", out)
	}
}
