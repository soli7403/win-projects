//go:build windows

// Package clipboard 는 Windows 클립보드의 텍스트·파일 목록·이미지를 읽고 쓴다 (spec/protocols.md §3 클립보드 세부).
package clipboard

import (
	"dariterm/internal/i18n"
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 클립보드 형식과 메모리 플래그 (WinUser.h / WinBase.h).
const (
	cfUnicodeText = 13
	cfDIB         = 8
	cfHDrop       = 15 // 탐색기에서 복사한 파일 목록 (DROPFILES)
	gmemMoveable  = 0x0002
	// g_ciDragQueryCount 를 DragQueryFileW 의 순번으로 주면 파일 개수를 돌려준다.
	g_ciDragQueryCount = 0xFFFFFFFF
)

var (
	g_modUser32                  = windows.NewLazySystemDLL("user32.dll")
	g_procOpenClipboard          = g_modUser32.NewProc("OpenClipboard")
	g_procCloseClipboard         = g_modUser32.NewProc("CloseClipboard")
	g_procEmptyClipboard         = g_modUser32.NewProc("EmptyClipboard")
	g_procSetClipboardData       = g_modUser32.NewProc("SetClipboardData")
	g_procGetClipboardData       = g_modUser32.NewProc("GetClipboardData")
	g_procIsClipboardFormatAvail = g_modUser32.NewProc("IsClipboardFormatAvailable")
	g_modKernel32Clip            = windows.NewLazySystemDLL("kernel32.dll")
	g_procGlobalAlloc            = g_modKernel32Clip.NewProc("GlobalAlloc")
	g_procGlobalLock             = g_modKernel32Clip.NewProc("GlobalLock")
	g_procGlobalUnlock           = g_modKernel32Clip.NewProc("GlobalUnlock")
	g_modShell32Clip             = windows.NewLazySystemDLL("shell32.dll")
	g_procDragQueryFileW         = g_modShell32Clip.NewProc("DragQueryFileW")
)

// ErrNoImage 는 클립보드에 이미지가 없을 때의 오류다.
var ErrNoImage = i18n.NewError("be.clip.noImage")

// openClip 은 클립보드를 열고 닫는 함수를 돌려준다. 열지 못하면 오류.
func openClip() (func(), error) {
	for iTry := 0; iTry < 10; iTry++ {
		r, _, _ := g_procOpenClipboard.Call(0)
		if r != 0 {
			return func() { g_procCloseClipboard.Call() }, nil
		}
	}
	return nil, i18n.NewError("be.clip.open")
}

// ReadText 는 유니코드 텍스트를 읽는다. 없으면 빈 문자열.
func ReadText() (string, error) {
	closeFn, errOpen := openClip()
	if errOpen != nil {
		return "", errOpen
	}
	defer closeFn()
	hMem, _, _ := g_procGetClipboardData.Call(cfUnicodeText)
	if hMem == 0 {
		return "", nil
	}
	pText, _, _ := g_procGlobalLock.Call(hMem)
	if pText == 0 {
		return "", nil
	}
	defer g_procGlobalUnlock.Call(hMem)
	var slRunes []uint16
	for iIndex := 0; ; iIndex++ {
		wch := *(*uint16)(addrAt(pText, iIndex*2))
		if wch == 0 {
			break
		}
		slRunes = append(slRunes, wch)
	}
	return string(utf16.Decode(slRunes)), nil
}

// WriteText 는 텍스트를 클립보드에 쓴다.
func WriteText(szText string) error {
	closeFn, errOpen := openClip()
	if errOpen != nil {
		return errOpen
	}
	defer closeFn()
	g_procEmptyClipboard.Call()
	slUtf16 := utf16.Encode([]rune(szText))
	slUtf16 = append(slUtf16, 0)
	nBytes := uintptr(len(slUtf16) * 2)
	hMem, _, _ := g_procGlobalAlloc.Call(gmemMoveable, nBytes)
	if hMem == 0 {
		return i18n.NewError("be.clip.alloc")
	}
	pDst, _, _ := g_procGlobalLock.Call(hMem)
	for iIndex, wch := range slUtf16 {
		*(*uint16)(addrAt(pDst, iIndex*2)) = wch
	}
	g_procGlobalUnlock.Call(hMem)
	if r, _, _ := g_procSetClipboardData.Call(cfUnicodeText, hMem); r == 0 {
		return i18n.NewError("be.clip.write")
	}
	return nil
}

// HasImage 는 비트맵(DIB) 형식이 있는지 본다.
func HasImage() bool {
	r, _, _ := g_procIsClipboardFormatAvail.Call(cfDIB)
	return r != 0
}

// Files 는 탐색기에서 복사한 파일 목록 중 디스크에 실제로 있는 것만 돌려준다.
// CF_HDROP 을 Win32 로 바로 읽는다 (붙여넣기마다 불리므로 빨라야 한다 — 예전 PowerShell 방식은 0.3~0.8초 걸렸다).
// 파일 목록 형식이 없으면 클립보드를 열지도 않고 바로 돌아온다.
func Files() ([]string, error) {
	if r, _, _ := g_procIsClipboardFormatAvail.Call(cfHDrop); r == 0 {
		return nil, nil
	}
	closeFn, errOpen := openClip()
	if errOpen != nil {
		return nil, errOpen
	}
	defer closeFn()
	hDrop, _, _ := g_procGetClipboardData.Call(cfHDrop)
	if hDrop == 0 {
		return nil, nil
	}
	nCount, _, _ := g_procDragQueryFileW.Call(hDrop, g_ciDragQueryCount, 0, 0)
	var slOut []string
	for iIndex := uintptr(0); iIndex < nCount; iIndex++ {
		nLen, _, _ := g_procDragQueryFileW.Call(hDrop, iIndex, 0, 0) // 끝 NUL 을 뺀 글자 수
		if nLen == 0 {
			continue
		}
		slBuf := make([]uint16, nLen+1)
		nGot, _, _ := g_procDragQueryFileW.Call(hDrop, iIndex, uintptr(unsafe.Pointer(&slBuf[0])), nLen+1)
		if nGot == 0 {
			continue
		}
		szPath := windows.UTF16ToString(slBuf[:nGot])
		if _, errStat := os.Stat(szPath); errStat == nil {
			slOut = append(slOut, szPath)
		}
	}
	return slOut, nil
}

// ImagePNG 는 클립보드 이미지를 PNG 바이트로 돌려준다 (숨긴 STA PowerShell, 드문 경로라 느려도 된다).
func ImagePNG() ([]byte, error) {
	if !HasImage() {
		return nil, ErrNoImage
	}
	out, errRun := runPS(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $img = [System.Windows.Forms.Clipboard]::GetImage(); if ($img -eq $null) { exit 3 }; $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); Write-Output ([Convert]::ToBase64String($ms.ToArray()))`)
	if errRun != nil || len(out) == 0 {
		return nil, i18n.NewError("be.clip.readImage")
	}
	return base64.StdEncoding.DecodeString(out[0])
}

// runPS 는 숨긴 STA PowerShell 로 스크립트를 돌리고 출력 줄을 돌려준다.
func runPS(szScript string) ([]string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, errRun := cmd.Output()
	if errRun != nil {
		return nil, errRun
	}
	var slOut []string
	for _, szLine := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(szLine) != "" {
			slOut = append(slOut, strings.TrimSpace(szLine))
		}
	}
	return slOut, nil
}

// addrAt 는 GlobalLock 이 돌려준 주소에서 바이트 오프셋만큼 떨어진 포인터다.
// uintptr 를 직접 unsafe.Pointer 로 바꾸면 vet 이 막으므로, 값을 메모리에서 다시 읽어 바꾼다.
func addrAt(pBase uintptr, iOffset int) unsafe.Pointer {
	pRaw := pBase + uintptr(iOffset)
	return *(*unsafe.Pointer)(unsafe.Pointer(&pRaw))
}
