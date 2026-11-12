//go:build !windows

package gitsync

import "os/exec"

func hideWindow(*exec.Cmd) {}
