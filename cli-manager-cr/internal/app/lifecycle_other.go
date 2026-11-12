//go:build !windows

package app

// idleMillisOS 는 Windows 가 아니면 입력 시각을 알 수 없다 (자동 잠금은 유휴로 잠그지 않는다).
func idleMillisOS() (int64, bool) {
	return 0, false
}

// desktopLockedOS 는 Windows 가 아니면 항상 잠기지 않은 것으로 본다.
func desktopLockedOS() bool {
	return false
}
