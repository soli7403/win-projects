//go:build !windows

package sftpsvc

import "errors"

// Windows 외 OS 의 클립보드 이미지·파일 목록은 아직 구현 전이다.
func ClipboardHasImage() bool { return false }

// ClipboardFiles 는 구현 전.
func ClipboardFiles() []string { return nil }

// ClipboardImagePNG 는 구현 전.
func ClipboardImagePNG() ([]byte, error) {
	return nil, errors.New("이 OS 에서는 클립보드 이미지를 지원하지 않습니다")
}
