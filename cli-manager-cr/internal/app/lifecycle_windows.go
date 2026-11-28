//go:build windows

package app

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

// Windows API 호출 (user32, kernel32, wtsapi32). 마지막 입력 시각과 세션 잠금 상태를 본다.
var (
	g_procGetLastInputInfo = syscall.NewLazyDLL("user32.dll").NewProc("GetLastInputInfo")
	g_procOpenInputDesktop = syscall.NewLazyDLL("user32.dll").NewProc("OpenInputDesktop")
	g_procCloseDesktop     = syscall.NewLazyDLL("user32.dll").NewProc("CloseDesktop")
	g_procGetTickCount     = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount")
	g_procWtsQuerySession  = syscall.NewLazyDLL("wtsapi32.dll").NewProc("WTSQuerySessionInformationW")
	g_procWtsFreeMemory    = syscall.NewLazyDLL("wtsapi32.dll").NewProc("WTSFreeMemory")
)

// WTSQuerySessionInformationW 로 세션 잠금 상태를 읽는 데 쓰는 값 (wtsapi32.h).
const (
	g_cuWtsCurrentSession   = 0xFFFFFFFF // WTS_CURRENT_SESSION: 이 프로세스의 세션
	g_ciWtsSessionInfoEx    = 25         // WTSSessionInfoEx
	g_ciWtsInfoExLevel1     = 1          // WTSINFOEXW.Level
	g_ciWtsSessionStateLock = 0          // WTS_SESSIONSTATE_LOCK
	g_ciWtsSessionStateOpen = 1          // WTS_SESSIONSTATE_UNLOCK
	// WTSINFOEXW 는 Level(DWORD) 뒤에 8바이트 정렬된 공용체가 온다. 그 안의 WTSINFOEX_LEVEL1_W 는
	// SessionId(4) · SessionState(4) · SessionFlags(4) 순이라 SessionFlags 는 맨 앞에서 16바이트 자리다.
	g_ciWtsSessionFlagsOffset = 16
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

// desktopLockedOS 는 Windows 세션이 잠겼는지 본다 (lifecycle.md §2, D154).
// 세션의 잠금 상태를 먼저 읽고, 알 수 없을 때만 예전 추정(입력 데스크톱 접근 거부)을 쓴다.
// 화면 보호기는 별도 데스크톱에서 돌아 예전 추정만으로는 잠긴 것으로 잘못 볼 수 있었다.
func desktopLockedOS() bool {
	if bLocked, bKnown := sessionLockStateOS(); bKnown {
		return bLocked
	}
	return inputDesktopDeniedOS()
}

// sessionLockStateOS 는 이 프로세스가 도는 Windows 세션의 잠금 상태를 읽는다. 알 수 없으면 bKnown=false.
func sessionLockStateOS() (bLocked bool, bKnown bool) {
	if g_procWtsQuerySession.Find() != nil || g_procWtsFreeMemory.Find() != nil {
		return false, false
	}
	var pBuf *byte
	var uBytes uint32
	rRet, _, _ := g_procWtsQuerySession.Call(0, uintptr(g_cuWtsCurrentSession), uintptr(g_ciWtsSessionInfoEx),
		uintptr(unsafe.Pointer(&pBuf)), uintptr(unsafe.Pointer(&uBytes)))
	if rRet == 0 || pBuf == nil {
		return false, false
	}
	defer g_procWtsFreeMemory.Call(uintptr(unsafe.Pointer(pBuf)))
	if uBytes < g_ciWtsSessionFlagsOffset+4 {
		return false, false
	}
	bInfo := unsafe.Slice(pBuf, uBytes)
	if binary.LittleEndian.Uint32(bInfo[0:4]) != g_ciWtsInfoExLevel1 {
		return false, false
	}
	switch int32(binary.LittleEndian.Uint32(bInfo[g_ciWtsSessionFlagsOffset : g_ciWtsSessionFlagsOffset+4])) {
	case g_ciWtsSessionStateLock:
		return true, true
	case g_ciWtsSessionStateOpen:
		return false, true
	default: // WTS_SESSIONSTATE_UNKNOWN
		return false, false
	}
}

// inputDesktopDeniedOS 는 예전 판정이다: 입력 데스크톱을 열 수 없고 접근 거부이면 잠긴 것으로 본다.
// 데스크톱 핸들은 CloseHandle 이 아니라 CloseDesktop 으로 닫는다.
func inputDesktopDeniedOS() bool {
	rHandle, _, errCall := g_procOpenInputDesktop.Call(0, 0, g_ciDesktopSwitch)
	if rHandle != 0 {
		_, _, _ = g_procCloseDesktop.Call(rHandle)
		return false
	}
	errno, bIsErrno := errCall.(syscall.Errno)
	return bIsErrno && errno == g_ciErrorAccessDenied
}
