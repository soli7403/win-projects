package transport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	pty "github.com/aymanbagabas/go-pty"
)

// LocalOptions 는 LocalOptions(shared/types.ts) 대응.
type LocalOptions struct {
	Shell string `json:"shell"`
	Cwd   string `json:"cwd"`
}

// ShellInfo 는 설정 화면의 셸 선택지.
type ShellInfo struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// ListShells 는 이 PC 에서 쓸 수 있는 셸 목록 (첫 항목이 기본값).
func ListShells() []ShellInfo {
	if runtime.GOOS == "windows" {
		out := []ShellInfo{{"powershell.exe", "Windows PowerShell"}, {"cmd.exe", "명령 프롬프트 (cmd)"}}
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			out = []ShellInfo{out[0], {"pwsh.exe", "PowerShell 7"}, out[1]}
		}
		if _, err := exec.LookPath("wsl.exe"); err == nil {
			out = append(out, ShellInfo{"wsl.exe", "WSL (Linux)"})
		}
		return out
	}
	szDef := defaultShell()
	out := []ShellInfo{{szDef, filepath.Base(szDef)}}
	if arr, err := os.ReadFile("/etc/shells"); err == nil {
		for _, l := range strings.Split(string(arr), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "/") && l != szDef {
				if _, err := os.Stat(l); err == nil {
					out = append(out, ShellInfo{l, filepath.Base(l)})
				}
			}
		}
	}
	return out
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "powershell.exe"
	}
	if sz := os.Getenv("SHELL"); sz != "" {
		return sz
	}
	return "/bin/sh"
}

// ExpandHome 은 ~ 를 홈 폴더로 바꾼다.
func ExpandHome(szPath string) string {
	szHome, err := os.UserHomeDir()
	if err != nil {
		return szPath
	}
	if szPath == "~" {
		return szHome
	}
	if strings.HasPrefix(szPath, "~/") || strings.HasPrefix(szPath, `~\`) {
		return filepath.Join(szHome, szPath[2:])
	}
	return szPath
}

func shellArgs(szShell string) []string {
	szName := strings.ToLower(strings.TrimSuffix(filepath.Base(szShell), ".exe"))
	if runtime.GOOS == "windows" {
		if szName == "powershell" || szName == "pwsh" {
			return []string{"-NoLogo"}
		}
		return nil
	}
	switch szName {
	case "bash", "zsh", "fish", "sh", "ksh", "tcsh", "csh":
		return []string{"-l"}
	}
	return nil
}

// Local 은 이 PC 의 셸을 가상 터미널(Windows ConPTY 포함)로 띄운다. 입출력은 UTF-8.
type Local struct {
	p       pty.Pty
	cmd     *pty.Cmd
	closing atomic.Bool
	// CwdFallback 은 시작 폴더가 없어 홈으로 바꿨을 때 true.
	CwdFallback bool
}

// StartLocal 은 셸을 실행하고 출력 읽기를 시작한다.
func StartLocal(o LocalOptions, szTerm string, iCols, iRows int, cb Callbacks) (*Local, error) {
	szShell := strings.TrimSpace(o.Shell)
	if szShell == "" {
		szShell = defaultShell()
	}
	pstrL := &Local{}
	szCwd := ExpandHome(strings.TrimSpace(o.Cwd))
	if szCwd == "" {
		szCwd = ExpandHome("~")
	}
	if st, err := os.Stat(szCwd); err != nil || !st.IsDir() {
		pstrL.CwdFallback = strings.TrimSpace(o.Cwd) != ""
		szCwd = ExpandHome("~")
	}
	p, err := pty.New()
	if err != nil {
		return nil, fmt.Errorf("가상 터미널을 만들 수 없습니다: %w", err)
	}
	if err := p.Resize(max(2, iCols), max(2, iRows)); err != nil {
		_ = p.Close()
		return nil, err
	}
	if szAbs, err := exec.LookPath(szShell); err == nil {
		szShell = szAbs
	}
	cmd := p.Command(szShell, shellArgs(szShell)...)
	cmd.Dir = szCwd
	if szTerm == "" {
		szTerm = "xterm-256color"
	}
	cmd.Env = append(os.Environ(), "TERM="+szTerm, "COLORTERM=truecolor", "TERM_PROGRAM=TermGo")
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		return nil, err
	}
	pstrL.p, pstrL.cmd = p, cmd
	once := NewOnceClose(cb.OnClose)
	go func() {
		arr := make([]byte, 32*1024)
		for {
			n, err := p.Read(arr)
			if n > 0 {
				cb.OnData(append([]byte(nil), arr[:n]...))
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		err := cmd.Wait()
		if pstrL.closing.Load() {
			once.Fire("")
			return
		}
		sz := "셸이 종료되었습니다"
		if err != nil {
			var ee *exec.ExitError
			if code := exitCode(err, &ee); code != 0 {
				sz += fmt.Sprintf(" (코드 %d)", code)
			}
		}
		once.Fire(sz)
		_ = p.Close()
	}()
	return pstrL, nil
}

func exitCode(err error, pe **exec.ExitError) int {
	if ee, ok := err.(*exec.ExitError); ok {
		*pe = ee
		return ee.ExitCode()
	}
	return 0
}

// Write 는 입력을 셸로 보낸다.
func (pstrL *Local) Write(arr []byte) error {
	_, err := pstrL.p.Write(arr)
	return err
}

// Resize 는 터미널 크기를 바꾼다.
func (pstrL *Local) Resize(iCols, iRows int) {
	if iCols > 1 && iRows > 1 {
		_ = pstrL.p.Resize(iCols, iRows)
	}
}

// Close 는 셸을 종료한다.
func (pstrL *Local) Close() {
	pstrL.closing.Store(true)
	if pstrL.cmd != nil && pstrL.cmd.Process != nil {
		_ = pstrL.cmd.Process.Kill()
	}
	_ = pstrL.p.Close()
}
