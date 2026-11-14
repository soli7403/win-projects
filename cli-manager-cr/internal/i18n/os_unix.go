//go:build !windows && !darwin

package i18n

import (
	"os"
	"strings"
)

// DetectOS 는 로캘 환경 변수(LC_ALL → LC_MESSAGES → LANGUAGE → LANG)를 돌려준다. LANGUAGE 는 ko:en 처럼 여러 개.
func DetectOS() []string {
	var slTags []string
	for _, szKey := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		szValue := os.Getenv(szKey)
		if szValue == "" || szValue == "C" || szValue == "POSIX" {
			continue
		}
		for _, szTag := range strings.Split(szValue, ":") {
			if szTag != "" {
				slTags = append(slTags, szTag)
			}
		}
	}
	return slTags
}
