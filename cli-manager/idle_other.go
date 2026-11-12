//go:build !windows

package main

import "time"

// systemIdle 은 Windows 외에서는 구현 전 — 0 을 돌려 자동 잠금을 하지 않는다.
func systemIdle() time.Duration { return 0 }
