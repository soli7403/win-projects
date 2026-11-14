//go:build windows

package i18n

import "golang.org/x/sys/windows"

// DetectOS 는 Windows 사용자 표시 언어 목록(우선순위 순)을 돌려준다. 예: ["ko-KR", "en-US"].
func DetectOS() []string {
	slLangs, errLangs := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if errLangs != nil {
		return nil
	}
	return slLangs
}
