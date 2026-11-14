//go:build !windows

package clipboard

import (
	"dariterm/internal/i18n"
)

// errUnsupported 는 1차 범위 밖 운영체제의 자리표시자다 (텍스트만 지원 목표, 여기서는 빈 동작).
var errUnsupported = i18n.NewError("be.clip.unsupported")

// ErrNoImage 는 클립보드에 이미지가 없을 때의 오류다.
var ErrNoImage = i18n.NewError("be.clip.noImage")

// ReadText 는 비 Windows 에서 빈 문자열을 돌려준다.
func ReadText() (string, error) { return "", nil }

// WriteText 는 비 Windows 에서 지원하지 않는다.
func WriteText(szText string) error { return errUnsupported }

// HasImage 는 비 Windows 에서 false 다.
func HasImage() bool { return false }

// Files 는 비 Windows 에서 목록이 없다.
func Files() ([]string, error) { return nil, nil }

// ImagePNG 는 비 Windows 에서 지원하지 않는다.
func ImagePNG() ([]byte, error) { return nil, errUnsupported }
