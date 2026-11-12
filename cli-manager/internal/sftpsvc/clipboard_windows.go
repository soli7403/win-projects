//go:build windows

package sftpsvc

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	g_user32     = syscall.NewLazyDLL("user32.dll")
	g_shell32    = syscall.NewLazyDLL("shell32.dll")
	g_pAvail     = g_user32.NewProc("IsClipboardFormatAvailable")
	g_pOpen      = g_user32.NewProc("OpenClipboard")
	g_pClose     = g_user32.NewProc("CloseClipboard")
	g_pGetData   = g_user32.NewProc("GetClipboardData")
	g_pDragQuery = g_shell32.NewProc("DragQueryFileW")
)

const (
	cfDIB   = 8
	cfHDROP = 15
)

// ClipboardHasImage 는 클립보드에 비트맵 이미지가 있는지 (빠른 검사).
func ClipboardHasImage() bool {
	r, _, _ := g_pAvail.Call(cfDIB)
	return r != 0
}

// ClipboardFiles 는 탐색기에서 복사한 파일 경로 목록.
func ClipboardFiles() []string {
	if r, _, _ := g_pAvail.Call(cfHDROP); r == 0 {
		return nil
	}
	if r, _, _ := g_pOpen.Call(0); r == 0 {
		return nil
	}
	defer g_pClose.Call()
	h, _, _ := g_pGetData.Call(cfHDROP)
	if h == 0 {
		return nil
	}
	n, _, _ := g_pDragQuery.Call(h, 0xFFFFFFFF, 0, 0)
	out := []string{}
	for i := uintptr(0); i < n; i++ {
		l, _, _ := g_pDragQuery.Call(h, i, 0, 0)
		buf := make([]uint16, l+1)
		g_pDragQuery.Call(h, i, uintptr(unsafe.Pointer(&buf[0])), l+1)
		out = append(out, syscall.UTF16ToString(buf))
	}
	return out
}

// ClipboardImagePNG 는 클립보드 이미지를 PNG 바이트로 돌려준다 (PowerShell 사용 — 드문 경로).
func ClipboardImagePNG() ([]byte, error) {
	if !ClipboardHasImage() {
		return nil, errors.New("클립보드에 이미지가 없습니다")
	}
	const szScript = `Add-Type -AssemblyName System.Windows.Forms,System.Drawing;` +
		`$i=[System.Windows.Forms.Clipboard]::GetImage(); if($i -eq $null){exit 3};` +
		`$m=New-Object System.IO.MemoryStream; $i.Save($m,[System.Drawing.Imaging.ImageFormat]::Png);` +
		`[Console]::Out.Write([Convert]::ToBase64String($m.ToArray()))`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, errors.New("클립보드 이미지를 읽지 못했습니다")
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(out.String()))
}
