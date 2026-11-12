//go:build !windows

package importer

import "os/exec"

func hideWindow(*exec.Cmd) {}
