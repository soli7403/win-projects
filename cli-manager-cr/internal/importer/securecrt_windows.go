//go:build windows

package importer

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// g_cszCrtRegKey 는 SecureCRT 설정 폴더 경로가 있는 레지스트리 키다 (현재 사용자 아래).
const g_cszCrtRegKey = `Software\VanDyke\SecureCRT`

// SecureCrtConfigDir 는 SecureCRT 설정 폴더를 돌려준다: 레지스트리 Config Path, 없으면 %APPDATA%\VanDyke\Config (import.md §1.5).
func SecureCrtConfigDir() string {
	if pKey, errOpen := registry.OpenKey(registry.CURRENT_USER, g_cszCrtRegKey, registry.QUERY_VALUE); errOpen == nil {
		szPath, _, errValue := pKey.GetStringValue("Config Path")
		pKey.Close()
		if errValue == nil && szPath != "" {
			return szPath
		}
	}
	szAppData := os.Getenv("APPDATA")
	if szAppData == "" {
		return ""
	}
	return filepath.Join(szAppData, "VanDyke", "Config")
}
