//go:build windows

package importer

import (
	"os/exec"
	"syscall"
)

func hideWindow(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
