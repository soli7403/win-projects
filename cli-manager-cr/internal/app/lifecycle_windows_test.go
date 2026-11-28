//go:build windows

package app

import "testing"

// TestSessionLockStateReadable 은 Windows 세션 잠금 상태를 실제로 읽을 수 있고, 테스트를 돌리는 지금(풀린 세션)은
// "잠기지 않음"으로 나오는지 본다 (D154). 대화형 세션이 아닌 곳(서비스·일부 CI)에서는 상태를 알 수 없어 건너뛴다.
func TestSessionLockStateReadable(t *testing.T) {
	bLocked, bKnown := sessionLockStateOS()
	if !bKnown {
		t.Skip("이 환경에서는 Windows 세션 잠금 상태를 알 수 없다 (대화형 세션이 아님)")
	}
	if bLocked {
		t.Fatalf("풀린 세션에서 잠김으로 읽었다 — SessionFlags 자리(%d)가 틀렸을 수 있다", g_ciWtsSessionFlagsOffset)
	}
	if desktopLockedOS() {
		t.Fatal("desktopLockedOS 가 풀린 세션을 잠김으로 본다")
	}
}
