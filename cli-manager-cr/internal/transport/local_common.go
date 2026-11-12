package transport

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LocalOptions 는 로컬 셸 시작 설정이다 (protocols.md §6).
type LocalOptions struct {
	Shell string   // 실행 파일 절대 경로
	Args  []string // 인자
	Cwd   string   // 시작 폴더 (존재하는 폴더여야 한다)
	Cols  int
	Rows  int
	Env   []string // "이름=값" 목록
}

// ShellItem 은 셸 목록 항목 {path, label} 이다 (spec/ipc.md §2 local.shells).
type ShellItem struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// ListShells 는 셸 목록을 돌려준다. 첫 항목이 기본이다 (protocols.md §6).
func ListShells() []ShellItem {
	if runtime.GOOS == "windows" {
		return listWindowsShells()
	}
	return listUnixShells()
}

func listWindowsShells() []ShellItem {
	slOut := []ShellItem{}
	if szPs, errPs := exec.LookPath("powershell.exe"); errPs == nil {
		slOut = append(slOut, ShellItem{Path: szPs, Label: "Windows PowerShell"})
	}
	if szCmd, errCmd := exec.LookPath("cmd.exe"); errCmd == nil {
		slOut = append(slOut, ShellItem{Path: szCmd, Label: "명령 프롬프트 (cmd)"})
	}
	if szPwsh, errPwsh := exec.LookPath("pwsh.exe"); errPwsh == nil {
		slOut = append(slOut, ShellItem{Path: szPwsh, Label: "PowerShell 7"})
	}
	if szWsl, errWsl := exec.LookPath("wsl.exe"); errWsl == nil {
		slOut = append(slOut, ShellItem{Path: szWsl, Label: "WSL (Linux)"})
	}
	return slOut
}

func listUnixShells() []ShellItem {
	slOut := []ShellItem{}
	szDefault := os.Getenv("SHELL")
	if szDefault == "" {
		szDefault = "/bin/sh"
	}
	slOut = append(slOut, ShellItem{Path: szDefault, Label: filepath.Base(szDefault)})
	file, errOpen := os.Open("/etc/shells")
	if errOpen != nil {
		return slOut
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		szLine := strings.TrimSpace(scanner.Text())
		if szLine == "" || strings.HasPrefix(szLine, "#") || szLine == szDefault {
			continue
		}
		if _, errStat := os.Stat(szLine); errStat == nil {
			slOut = append(slOut, ShellItem{Path: szLine, Label: filepath.Base(szLine)})
		}
	}
	return slOut
}

// ResolveShellPath 는 셸 이름을 PATH 에서 절대 경로로 찾는다. 찾지 못하면 입력 그대로 돌려준다.
func ResolveShellPath(szName string) string {
	if szPath, errLook := exec.LookPath(szName); errLook == nil {
		return szPath
	}
	return szName
}
