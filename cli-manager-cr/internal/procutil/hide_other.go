//go:build !windows

package procutil

import "os/exec"

func hideConsoleOS(pCmd *exec.Cmd) {}
