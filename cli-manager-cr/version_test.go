package main

import (
	"regexp"
	"testing"
)

// 실제 wails.json 의 productVersion 이 x.y.z 형식이어야 한다(MSIX 는 뒤에 .0 을 붙여 x.y.z.0 을 쓴다).
func TestProductVersionFromWailsJson(t *testing.T) {
	szVer := productVersion(gbyWailsJson)
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(szVer) {
		t.Fatalf("wails.json info.productVersion = %q, want x.y.z", szVer)
	}
}

func TestProductVersionBadJson(t *testing.T) {
	for _, szIn := range []string{"", "{", `{"info":{}}`, `{"info":{"productVersion":1}}`} {
		if szVer := productVersion([]byte(szIn)); szVer != "" {
			t.Errorf("productVersion(%q) = %q, want empty", szIn, szVer)
		}
	}
}
