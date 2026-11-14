//go:build !windows

package license

// DomainJoined 는 Windows 가 아니면 항상 false 다 (회사 PC 안내는 Windows 도메인 가입만 본다, D78).
func DomainJoined() bool {
	return false
}
