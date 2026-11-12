//go:build !windows

package importer

// ReadPuttySessions 는 Windows 가 아니면 PuTTY 세션이 없으므로 빈 맵을 돌려준다 (import.md §1).
func ReadPuttySessions() (map[string]map[string]any, error) {
	return map[string]map[string]any{}, nil
}
