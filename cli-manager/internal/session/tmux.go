package session

import (
	"path/filepath"
	"regexp"
	"strings"

	"termgo/internal/model"
)

// 3.3 세션 유지(tmux): 접속(또는 로컬 셸 시작) 직후 지정한 이름의 tmux 세션에 붙는다. 없으면 새로 만든다.
//   - 새로 만들 때만 마우스 모드 켜기 · 상태 표시줄 끄기 · 자동 실행 명령 실행
//   - 이미 있는 세션에 다시 붙을 때는 자동 실행 명령을 실행하지 않는다
//   - tmux 가 없으면 안내만 보여 주고 평소처럼 사용한다
//   - tmux 를 끝내면(exit) 원래 셸로 돌아온다 (자동 재접속이 새 세션을 계속 만들지 않도록 exec 하지 않는다)

var reBadName = regexp.MustCompile(`[.:\s'"$\\` + "`" + `]+`)

// TmuxSessionName 은 세션 이름을 정한다: 지정값 → 서버 별칭 → 로컬 폴더 이름 → "termgo".
func TmuxSessionName(h model.Host) string {
	n := strings.TrimSpace(h.TmuxName)
	if n == "" {
		n = strings.TrimSpace(h.Alias)
	}
	if n == "" && h.Local != nil {
		n = filepath.Base(strings.TrimRight(strings.ReplaceAll(h.Local.Cwd, `\`, "/"), "/"))
	}
	n = strings.Trim(reBadName.ReplaceAllString(n, "-"), "-")
	if n == "" || n == "~" || n == "/" {
		return "termgo"
	}
	return n
}

// KeepSessionApplies 는 이 호스트에 세션 유지를 적용할 수 있는지 (원격 SSH, 로컬은 Unix 셸 또는 Windows 의 WSL 셸).
func KeepSessionApplies(h model.Host, szGOOS string) bool {
	if !h.KeepSession {
		return false
	}
	switch h.Protocol {
	case "ssh":
		return true
	case "local":
		if szGOOS != "windows" {
			return true
		}
		sh := ""
		if h.Local != nil {
			sh = strings.ToLower(filepath.Base(h.Local.Shell))
		}
		return sh == "wsl.exe" || sh == "wsl"
	}
	return false
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// BuildTmuxCommand 는 셸에 입력할 한 줄(끝에 \r 포함)을 만든다.
func BuildTmuxCommand(szName, szStartup string) string {
	const script = `if ! command -v tmux >/dev/null 2>&1; then printf '\r\n\033[33m● 이 대상에 tmux 가 없어 세션 유지를 쓰지 않습니다 (자동 실행 명령은 실행되지 않음)\033[0m\r\n'; exit 0; fi; ` +
		`if tmux has-session -t "$1" 2>/dev/null; then tmux attach -t "$1"; else ` +
		`tmux new-session -d -s "$1" && tmux set -t "$1" mouse on && tmux set -t "$1" status off; ` +
		`if [ -n "$2" ]; then tmux send-keys -t "$1" "$2" Enter; fi; tmux attach -t "$1"; fi`
	szStartup = strings.ReplaceAll(strings.ReplaceAll(szStartup, "\r\n", "\n"), "\n", "; ")
	return " sh -c " + shQuote(script) + " sh " + shQuote(szName) + " " + shQuote(szStartup) + "\r"
}
