package transport

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var (
	g_tmuxOnce sync.Once
	g_tmuxOK   bool
)

// TmuxAvailable 은 로컬 터미널의 세션 유지를 기본으로 켤 수 있는지 (한 번만 검사).
// PATH 외에 흔한 설치 위치도 본다. Windows 는 WSL 안의 tmux 를 확인한다.
func TmuxAvailable() bool {
	g_tmuxOnce.Do(func() {
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("wsl.exe"); err != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "wsl.exe", "-e", "sh", "-c", "command -v tmux")
			hideConsole(cmd)
			g_tmuxOK = cmd.Run() == nil
			return
		}
		if _, err := exec.LookPath("tmux"); err == nil {
			g_tmuxOK = true
			return
		}
		for _, p := range []string{"/opt/homebrew/bin/tmux", "/usr/local/bin/tmux", "/usr/bin/tmux", "/opt/local/bin/tmux", "/snap/bin/tmux"} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				g_tmuxOK = true
				return
			}
		}
	})
	return g_tmuxOK
}
