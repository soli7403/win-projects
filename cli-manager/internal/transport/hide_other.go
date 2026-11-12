//go:build !windows

package transport

import "os/exec"

func hideConsole(*exec.Cmd) {}
