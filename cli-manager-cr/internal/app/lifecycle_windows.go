//go:build windows

package app

import (
	"syscall"
	"unsafe"
)

// Windows API 호출 (user32, kernel32). 마지막 입력 시각과 입력 데스크톱 접근으로 잠금 상태를 본다.
var (
	g_procGetLastInputInfo = syscall.NewLazyDLL("user32.dll").NewProc("GetLastInputInfo")
	g_procOpenInputDesktop = syscall.NewLazyDLL("user32.dll").NewProc("OpenInputDesktop")
	g_procGetTickCount     = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount")
)

// g_ciDesktopSwitch 는 DESKTOP_SWITCHDESKTOP 접근 권한이다.
const g_ciDesktopSwitch = 0x0100

// g_ciErrorAccessDenied 는 ERROR_ACCESS_DENIED 다 (잠긴 데스크톱에서 나온다).
const g_ciErrorAccessDenied = 5

// lastInputInfo 는 GetLastInputInfo 의 구조체다.
type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

// idleMillisOS 는 마지막 키보드·마우스 입력 이후 밀리초를 돌려준다. 알 수 없으면 bOk=false.
func idleMillisOS() (int64, bool) {
	info := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	rRet, _, _ := g_procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if rRet == 0 {
		return 0, false
	}
	rTick, _, _ := g_procGetTickCount.Call()
	iTick := uint32(rTick)
	return int64(iTick - info.dwTime), true
}

// desktopLockedOS 는 Windows 세션이 잠겼는지 추정한다. 입력 데스크톱을 열 수 없고 접근 거부이면 잠긴 것으로 본다.
func desktopLockedOS() bool {
	rHandle, _, errCall := g_procOpenInputDesktop.Call(0, 0, g_ciDesktopSwitch)
	if rHandle != 0 {
		syscall.CloseHandle(syscall.Handle(rHandle))
		return false
	}
	errno, bIsErrno := errCall.(syscall.Errno)
	return bIsErrno && errno == g_ciErrorAccessDenied
}
