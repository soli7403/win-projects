//go:build !windows

package vault

import (
	"dariterm/internal/i18n"
)

// G_bOsUnlockSupported 는 이 플랫폼에서 OS 잠금 해제를 쓸 수 없음을 나타낸다 (1차 범위는 Windows).
const G_bOsUnlockSupported = false

var errOsUnlockUnsupported = i18n.NewError("be.vault.osUnsupported")

// dpapiProtect 는 이 플랫폼에서 항상 실패한다.
func dpapiProtect(bPlain []byte) ([]byte, error) {
	return nil, errOsUnlockUnsupported
}

// dpapiUnprotect 는 이 플랫폼에서 항상 실패한다.
func dpapiUnprotect(bCipher []byte) ([]byte, error) {
	return nil, errOsUnlockUnsupported
}
