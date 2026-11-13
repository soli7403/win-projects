//go:build windows

package clipboard

import "testing"

// 텍스트 쓰기 → 읽기 왕복 (한글 포함, 유니코드).
func TestClipboardTextRoundTrip(t *testing.T) {
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
