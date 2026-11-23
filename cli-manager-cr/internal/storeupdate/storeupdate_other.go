//go:build !windows

package storeupdate

// IsStoreInstall 은 Windows 가 아니면 항상 false 다 (Microsoft Store 는 Windows 에만 있다).
func IsStoreInstall() bool { return false }

// Check 는 Windows 가 아니면 쓸 수 없다.
func Check() (*Update, error) { return nil, ErrNotStore }

// Install 은 Windows 가 아니면 쓸 수 없다.
func Install() (string, error) { return "", ErrNotStore }
