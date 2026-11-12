//go:build windows

package transport

import (
	"os/exec"
	"syscall"
)

func hideConsole(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
