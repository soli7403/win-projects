//go:build windows

package importer

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// g_cszShellFoldersKey 는 "문서" 폴더의 실제 위치가 있는 레지스트리 키다 (OneDrive 등으로 옮긴 경우를 따른다).
const g_cszShellFoldersKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`

// XshellSessionsDir 는 이 PC 의 Xshell 세션 폴더를 돌려준다. 없으면 빈 문자열 (import.md §1.6).
// "문서" 폴더는 레지스트리의 Personal 값(환경 변수를 펼친 것), 그다음 %USERPROFILE%\Documents 순으로 본다.
func XshellSessionsDir() string {
	slDocs := []string{}
	if pKey, errOpen := registry.OpenKey(registry.CURRENT_USER, g_cszShellFoldersKey, registry.QUERY_VALUE); errOpen == nil {
		szPersonal, _, errValue := pKey.GetStringValue("Personal")
		pKey.Close()
		if errValue == nil && szPersonal != "" {
			if szExpanded, errExpand := registry.ExpandString(szPersonal); errExpand == nil && szExpanded != "" {
				szPersonal = szExpanded
			}
			slDocs = append(slDocs, szPersonal)
		}
	}
	if szHome, errHome := os.UserHomeDir(); errHome == nil && szHome != "" {
		slDocs = append(slDocs, filepath.Join(szHome, "Documents"))
	}
	for _, szDocs := range slDocs {
		if szDir := xshellSessionsUnder(szDocs); szDir != "" {
			return szDir
		}
	}
	return ""
}
