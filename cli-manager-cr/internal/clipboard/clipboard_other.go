//go:build !windows

package clipboard

import "errors"

// errUnsupported 는 1차 범위 밖 운영체제의 자리표시자다 (텍스트만 지원 목표, 여기서는 빈 동작).
var errUnsupported = errors.New("이 운영체제에서는 클립보드를 아직 지원하지 않습니다")

// ErrNoImage 는 클립보드에 이미지가 없을 때의 오류다.
var ErrNoImage = errors.New("클립보드에 이미지가 없습니다")

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
