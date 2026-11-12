//go:build windows

package main

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

var (
	g_user32          = windows.NewLazySystemDLL("user32.dll")
	g_procLastInput   = g_user32.NewProc("GetLastInputInfo")
	g_kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	g_procTickCount64 = g_kernel32.NewProc("GetTickCount")
)

// systemIdle 은 마지막 키보드/마우스 입력 이후 경과 시간.
func systemIdle() time.Duration {
	li := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	if r, _, _ := g_procLastInput.Call(uintptr(unsafe.Pointer(&li))); r == 0 {
		return 0
	}
	tick, _, _ := g_procTickCount64.Call()
	return time.Duration(uint32(tick)-li.dwTime) * time.Millisecond
}
