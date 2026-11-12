package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// registerShell 은 app.openExternal·app.openLogs 를 등록한다 (spec/ipc.md §2 클립보드·대화상자·앱).
func (pService *Service) registerShell() {
	pRouter := pService.pRouter
	pRouter.Handle("app.openExternal", func(slArgs []json.RawMessage) (any, error) {
		szURL, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		// http:// 또는 https:// 로 시작하는 주소만 연다. 그 외는 무시한다 (spec/ipc.md app.openExternal).
		lower := strings.ToLower(szURL)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			_ = openWithShell(szURL)
		}
		return nil, nil
	})
	pRouter.Handle("app.openLogs", func(slArgs []json.RawMessage) (any, error) {
		szDir := filepath.Join(pService.szDataDir, "logs")
		if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
			return nil, errMkdir
		}
		_ = openWithShell(szDir)
		return nil, nil
	})
}

// openWithShell 은 운영체제 기본 프로그램으로 주소나 폴더를 연다.
func openWithShell(szTarget string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", szTarget).Start()
	}
	return exec.Command("xdg-open", szTarget).Start()
}
