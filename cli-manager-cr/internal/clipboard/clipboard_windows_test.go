//go:build windows

package clipboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

// keepUserClipboard 는 시험 전 클립보드 텍스트를 기억했다가 끝나면 되돌린다 (개발 PC 의 클립보드를 망치지 않게).
func keepUserClipboard(t *testing.T) {
	t.Helper()
	szBefore, errRead := ReadText()
	t.Cleanup(func() {
		if errRead == nil && szBefore != "" {
			_ = WriteText(szBefore)
		}
	})
}

// writeFileDrop 은 탐색기의 "복사" 처럼 파일 목록(CF_HDROP, DROPFILES + UTF-16 경로들)을 클립보드에 넣는다.
func writeFileDrop(t *testing.T, slPaths []string) {
	t.Helper()
	// DROPFILES: pFiles(4) + pt.x(4) + pt.y(4) + fNC(4) + fWide(4) = 20바이트, 이어서 NUL 로 끝나는 경로들 + 마지막 NUL.
	const ciHeader = 20
	var slChars []uint16
	for _, szPath := range slPaths {
		slChars = append(slChars, utf16.Encode([]rune(szPath))...)
		slChars = append(slChars, 0)
	}
	slChars = append(slChars, 0)
	bData := make([]byte, ciHeader+len(slChars)*2)
	bData[0] = ciHeader // pFiles = 20 (리틀 엔디언)
	bData[16] = 1       // fWide = TRUE
	for iIndex, wch := range slChars {
		bData[ciHeader+iIndex*2] = byte(wch)
		bData[ciHeader+iIndex*2+1] = byte(wch >> 8)
	}
	closeFn, errOpen := openClip()
	if errOpen != nil {
		t.Skipf("클립보드를 열 수 없는 환경: %v", errOpen)
	}
	defer closeFn()
	g_procEmptyClipboard.Call()
	hMem, _, _ := g_procGlobalAlloc.Call(gmemMoveable, uintptr(len(bData)))
	if hMem == 0 {
		t.Fatal("GlobalAlloc 실패")
	}
	pDst, _, _ := g_procGlobalLock.Call(hMem)
	for iIndex, bOne := range bData {
		*(*byte)(addrAt(pDst, iIndex)) = bOne
	}
	g_procGlobalUnlock.Call(hMem)
	if r, _, _ := g_procSetClipboardData.Call(cfHDrop, hMem); r == 0 {
		t.Fatal("SetClipboardData(CF_HDROP) 실패")
	}
}

// 텍스트 쓰기 → 읽기 왕복 (한글 포함, 유니코드).
func TestClipboardTextRoundTrip(t *testing.T) {
	keepUserClipboard(t)
	want := "dariterm 클립보드 테스트 123"
	if err := WriteText(want); err != nil {
		t.Skipf("클립보드를 쓸 수 없는 환경: %v", err)
	}
	got, err := ReadText()
	if err != nil || got != want {
		t.Fatalf("왕복 불일치: %q %v", got, err)
	}
	if info := GetInfo(); info.M_szType != "text" || info.M_szText != want {
		t.Fatalf("clip.info 판정: %+v", info)
	}
}

// 텍스트만 있을 때 clip.info 는 빨라야 한다 (오른쪽 클릭·Ctrl+V 붙여넣기 지연 수정 — 예전 PowerShell 방식은 0.3초 이상).
func TestClipboardInfoIsFastForText(t *testing.T) {
	keepUserClipboard(t)
	if err := WriteText("fast"); err != nil {
		t.Skipf("클립보드를 쓸 수 없는 환경: %v", err)
	}
	tStart := time.Now()
	for iIndex := 0; iIndex < 20; iIndex++ {
		if info := GetInfo(); info.M_szType != "text" {
			t.Fatalf("텍스트 판정이어야 함: %+v", info)
		}
	}
	if durEach := time.Since(tStart) / 20; durEach > 50*time.Millisecond {
		t.Fatalf("clip.info 한 번이 너무 느림: %v", durEach)
	}
}

// 탐색기에서 복사한 파일 목록을 Win32 로 읽고, 디스크에 없는 경로는 뺀다 (한글·공백 경로 포함).
func TestClipboardFilesFromHDrop(t *testing.T) {
	keepUserClipboard(t)
	szDir := t.TempDir()
	szFile := filepath.Join(szDir, "가나 다.txt")
	if errWrite := os.WriteFile(szFile, []byte("x"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	szMissing := filepath.Join(szDir, "없음.txt")
	writeFileDrop(t, []string{szFile, szMissing})

	slFiles, errFiles := Files()
	if errFiles != nil {
		t.Fatal(errFiles)
	}
	if len(slFiles) != 1 || slFiles[0] != szFile {
		t.Fatalf("있는 파일만 돌려줘야 함: %q", slFiles)
	}
	if info := GetInfo(); info.M_szType != "files" || len(info.M_slFiles) != 1 || info.M_slFiles[0] != szFile {
		t.Fatalf("clip.info 판정: %+v", info)
	}
}
