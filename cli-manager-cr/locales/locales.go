// Package locales 는 화면·백엔드 문구의 언어 파일(<언어>.json)을 담는다 (D75, spec/behavior/i18n.md).
// 문구를 고칠 때는 이 폴더의 json 파일만 고친다. 기준은 ko.json(키 목록), 다른 언어는 같은 키를 가진다.
// 프런트엔드는 같은 파일을 빌드 때 읽고(frontend/src/i18n), 백엔드는 여기서 실행 파일에 넣는다(internal/i18n).
package locales

import "embed"

// G_fsLocales 는 <언어>.json 파일들이다.
//
//go:embed *.json
var G_fsLocales embed.FS
