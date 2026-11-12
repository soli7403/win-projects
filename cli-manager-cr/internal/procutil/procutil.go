// Package procutil 은 외부 명령 실행 공통 설정을 둔다 (git, aws 호출).
package procutil

import "os/exec"

// G_cslEnvQuiet 는 외부 명령을 비대화형 C 로케일로 돌리는 환경 변수다 (sync-backup.md §5.4).
var G_cslEnvQuiet = []string{
	"GIT_TERMINAL_PROMPT=0",
	"GIT_ASKPASS=",
	"SSH_ASKPASS=",
	"GCM_INTERACTIVE=never",
	"LC_ALL=C",
	"LANG=C",
}

// HideConsole 은 Windows 에서 콘솔 창이 뜨지 않게 한다. 다른 OS 에서는 아무것도 하지 않는다.
func HideConsole(pCmd *exec.Cmd) {
	hideConsoleOS(pCmd)
}
