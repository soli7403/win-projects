package main

import (
	_ "embed"
	"encoding/json"
)

// 앱 버전은 wails.json 의 info.productVersion 한 곳에서만 정한다 (D82).
// 설치 파일(NSIS)·스토어 패키지(MSIX)도 같은 값을 쓰므로, 버전을 올릴 때는 wails.json 만 고친다.
//
//go:embed wails.json
var gbyWailsJson []byte

// productVersion 은 wails.json 의 info.productVersion 을 돌려준다. 읽지 못하면 빈 문자열.
func productVersion(byJson []byte) string {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if errParse := json.Unmarshal(byJson, &cfg); errParse != nil {
		return ""
	}
	return cfg.Info.ProductVersion
}
