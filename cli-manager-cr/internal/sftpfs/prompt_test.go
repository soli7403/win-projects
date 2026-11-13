package sftpfs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 붙여넣기 파일 이름 규칙 (protocols.md §3): dt-YYYYMMDD-HHMMSS-<4자리>…
func TestPromptNames(t *testing.T) {
	tNow := time.Date(2026, 11, 10, 9, 8, 7, 0, time.Local)
	if got := PromptImageName(tNow, "ab12", ""); got != "dt-20261110-090807-ab12.png" {
		t.Fatalf("이미지 이름: %q", got)
	}
	if got := PromptImageName(tNow, "ab12", "shot.jpg"); got != "dt-20261110-090807-ab12.jpg" {
		t.Fatalf("버퍼 확장자 유지: %q", got)
	}
	if got := PromptFileName(tNow, "ab12", "my file$1.txt"); got != "dt-20261110-090807-ab12-my_file_1.txt" {
		t.Fatalf("파일 이름 정리: %q", got)
	}
}

// 오래된 dt- 파일만 지운다 (protocols.md §3 정리).
func TestCleanupOldPromptFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "dt-old.png")
	fresh := filepath.Join(dir, "dt-new.png")
	other := filepath.Join(dir, "keep.png")
	for _, p := range []string{old, fresh, other} {
		_ = os.WriteFile(p, []byte("x"), 0o600)
	}
	past := time.Now().Add(-10 * 24 * time.Hour)
	_ = os.Chtimes(old, past, past)
	_ = os.Chtimes(other, past, past)
	n, errClean := CleanupOldPrompt(dir, 7, time.Now())
	if errClean != nil || n != 1 {
		t.Fatalf("정리 개수 %d %v", n, errClean)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("오래된 dt- 파일은 지워져야 함")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal("dt- 가 아닌 파일은 남아야 함")
	}
}

// TestIsPromptName 은 dt- 접두어 파일만 정리 대상으로 보는지 확인한다.
func TestIsPromptName(t *testing.T) {
	for szName, bWant := range map[string]bool{"dt-20261114-1.png": true, "tg-20261110-1.png": false, "shot.png": false, "xdt-1.png": false} {
		if IsPromptName(szName) != bWant {
			t.Fatalf("IsPromptName(%q) != %v", szName, bWant)
		}
	}
}
