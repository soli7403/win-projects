package procutil

import (
	"os/exec"
	"syscall"
)

// g_ciCreateNoWindow 는 CREATE_NO_WINDOW 플래그다.
const g_ciCreateNoWindow = 0x08000000

func hideConsoleOS(pCmd *exec.Cmd) {
	pCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: g_ciCreateNoWindow}
}
