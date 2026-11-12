package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"termgo/internal/procutil"
)

// G_cdurTmuxProbe 는 WSL 안의 tmux 검사 제한 시간이다 (session.md §7.1).
const G_cdurTmuxProbe = 5 * time.Second

// g_slUnixTmuxDirs 는 tmux 를 찾을 추가 폴더다 (session.md §7.1).
var g_slUnixTmuxDirs = []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/opt/local/bin", "/snap/bin"}

var (
	g_onceTmux    sync.Once
	g_bTmuxResult bool
)

// TmuxAvailable 은 tmux 가 있는지 앱 실행 중 한 번만 검사해 기억한다 (session.md §7.1).
func TmuxAvailable() bool {
	g_onceTmux.Do(func() {
		g_bTmuxResult = detectTmux()
	})
	return g_bTmuxResult
}

func detectTmux() bool {
	if runtime.GOOS == "windows" {
		pathWsl, errLook := exec.LookPath("wsl.exe")
		if errLook != nil {
			return false
		}
		ctxProbe, cancel := context.WithTimeout(context.Background(), G_cdurTmuxProbe)
		defer cancel()
		pCmd := exec.CommandContext(ctxProbe, pathWsl, "-e", "sh", "-c", "command -v tmux")
		procutil.HideConsole(pCmd) // GUI 앱에서 콘솔 창이 번쩍 뜨는 것을 막는다
		return pCmd.Run() == nil
	}
	if _, errLook := exec.LookPath("tmux"); errLook == nil {
		return true
	}
	for _, szDir := range g_slUnixTmuxDirs {
		if _, errStat := os.Stat(filepath.Join(szDir, "tmux")); errStat == nil {
			return true
		}
	}
	return false
}
