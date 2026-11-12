//go:build !windows

package vault

import "errors"

// G_bOsUnlockSupported 는 이 플랫폼에서 OS 잠금 해제를 쓸 수 없음을 나타낸다 (1차 범위는 Windows).
const G_bOsUnlockSupported = false

var errOsUnlockUnsupported = errors.New("이 운영체제에서는 OS 잠금 해제를 쓸 수 없습니다")

// dpapiProtect 는 이 플랫폼에서 항상 실패한다.
func dpapiProtect(bPlain []byte) ([]byte, error) {
	return nil, errOsUnlockUnsupported
}

// dpapiUnprotect 는 이 플랫폼에서 항상 실패한다.
func dpapiUnprotect(bCipher []byte) ([]byte, error) {
	return nil, errOsUnlockUnsupported
}
