//go:build !windows

package dialog

import (
	"dariterm/internal/i18n"
)

// errUnsupported 는 1차 범위 밖 운영체제의 대화상자 자리표시자다.
var errUnsupported = i18n.NewError("be.dialog.unsupported")

// OpenFiles 는 비 Windows 에서 지원하지 않는다.
func OpenFiles(szTitle string) ([]string, error) { return nil, errUnsupported }

// ChooseDir 는 비 Windows 에서 지원하지 않는다.
func ChooseDir(szTitle string, szInitial string, bAllowNew bool) (string, error) {
	return "", errUnsupported
}

// OpenKeyFile 는 비 Windows 에서 지원하지 않는다.
func OpenKeyFile(szTitle string, szInitial string) (string, error) { return "", errUnsupported }
