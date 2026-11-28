//go:build !windows

package sshconn

import (
	"os/exec"
	"syscall"
)

// shellCommand 는 명령 줄을 /bin/sh 로 실행하는 명령을 만든다 (터미널에 친 것과 같은 해석, D152).
// 새 프로세스 그룹으로 띄워 끝낼 때 그 아래 프로그램까지 함께 끝낸다.
func shellCommand(szLine string) *exec.Cmd {
	pCmd := exec.Command("/bin/sh", "-c", szLine)
	pCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return pCmd
}

// treeKiller 는 명령의 프로세스 그룹 전체를 끝내는 함수를 돌려준다.
func treeKiller(pCmd *exec.Cmd) func() {
	return func() {
		if pCmd.Process == nil {
			return
		}
		_ = syscall.Kill(-pCmd.Process.Pid, syscall.SIGKILL)
		_ = pCmd.Process.Kill()
	}
}
