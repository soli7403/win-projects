//go:build !windows

package importer

import (
	"os"
	"path/filepath"
	"runtime"
)

// SecureCrtConfigDir 는 SecureCRT 설정 폴더를 돌려준다: macOS 는 ~/Library/Application Support/VanDyke/SecureCRT/Config,
// 그 밖은 ~/.vandyke/SecureCRT/Config (import.md §1.5). 홈을 모르면 빈 문자열.
func SecureCrtConfigDir() string {
	szHome, errHome := os.UserHomeDir()
	if errHome != nil || szHome == "" {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(szHome, "Library", "Application Support", "VanDyke", "SecureCRT", "Config")
	}
	return filepath.Join(szHome, ".vandyke", "SecureCRT", "Config")
}
