//go:build !windows

package dialog

import "errors"

// errUnsupported 는 1차 범위 밖 운영체제의 대화상자 자리표시자다.
var errUnsupported = errors.New("이 운영체제에서는 대화상자를 아직 지원하지 않습니다")

// OpenFiles 는 비 Windows 에서 지원하지 않는다.
func OpenFiles(szTitle string) ([]string, error) { return nil, errUnsupported }

// ChooseDir 는 비 Windows 에서 지원하지 않는다.
func ChooseDir(szTitle string, szInitial string, bAllowNew bool) (string, error) {
	return "", errUnsupported
}

// OpenKeyFile 는 비 Windows 에서 지원하지 않는다.
func OpenKeyFile(szTitle string, szInitial string) (string, error) { return "", errUnsupported }
