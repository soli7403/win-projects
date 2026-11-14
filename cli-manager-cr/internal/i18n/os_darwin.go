//go:build darwin

package i18n

import (
	"os"
	"os/exec"
	"strings"
)

// DetectOS 는 macOS 언어 설정(AppleLanguages)을 우선, 없으면 LANG 을 돌려준다.
// Finder 로 실행한 앱에는 LANG 이 없을 때가 많아 defaults 를 먼저 읽는다.
func DetectOS() []string {
	var slTags []string
	arrOut, errRun := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
	if errRun == nil {
		// 출력 예: ( "ko-KR", "en-US" )
		for _, szLine := range strings.Split(string(arrOut), "\n") {
			szTag := strings.Trim(strings.TrimSpace(szLine), `",()`)
			if szTag != "" {
				slTags = append(slTags, szTag)
			}
		}
	}
	for _, szKey := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if szValue := os.Getenv(szKey); szValue != "" && szValue != "C" {
			slTags = append(slTags, szValue)
		}
	}
	return slTags
}
