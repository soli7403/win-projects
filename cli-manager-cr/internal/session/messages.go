// Package session 은 세션 수명주기(상태 전이, 재접속, 출력 묶음, 인코딩, 로그, tmux)를 다룬다 (spec/behavior/session.md).
package session

import (
	"dariterm/internal/i18n"
	"regexp"
	"strings"
)

// 터미널 안내 줄의 색 (session.md §3): 정보=회색, 경고=노랑, 오류=빨강.
const (
	G_cszColorInfo  = "\x1b[90m"
	G_cszColorWarn  = "\x1b[33m"
	G_cszColorError = "\x1b[31m"
	G_cszColorReset = "\x1b[0m"
	G_cszClearLine  = "\r\x1b[2K"
)

// G_ciMaxReconnect 는 자동 재연결 최대 횟수다 (session.md §4). 대기 2·4·8·16·30초로 5번 (D62, 이전 8번).
const G_ciMaxReconnect = 5

// G_iMaxBackoffSec 는 재접속 대기 상한(초)이다.
const G_iMaxBackoffSec = 30

// 세션 상태 (session.md §2).
const (
	G_cszStateConnecting   = "connecting"
	G_cszStateConnected    = "connected"
	G_cszStateReconnecting = "reconnecting"
	G_cszStateClosed       = "closed"
	G_cszStateError        = "error"
)

// notice 는 색이 입혀진 안내 한 줄을 만든다 (줄 끝에 CRLF 를 붙인다).
func notice(szColor string, szText string) string {
	return szColor + "● " + szText + G_cszColorReset + "\r\n"
}

// MsgConnecting 은 접속 시작 안내다 (로컬 제외).
func MsgConnecting(szTitle string, szTarget string) string {
	return notice(G_cszColorInfo, i18n.T("be.session.connecting", "title", szTitle, "target", szTarget))
}

// MsgConnectFailed 는 접속 실패 안내다. 로컬은 MsgShellFailed 를 쓴다.
func MsgConnectFailed(szReason string) string {
	return notice(G_cszColorError, i18n.T("be.session.connectFailed", "reason", TranslateError(szReason)))
}

// MsgShellFailed 는 로컬 셸 실행 실패 안내다.
func MsgShellFailed(szReason string) string {
	return notice(G_cszColorError, i18n.T("be.session.shellFailed", "reason", szReason))
}

// MsgDisconnected 는 연결 끊김 안내다. 이유가 없으면 기본 문구를 쓴다.
func MsgDisconnected(szReason string) string {
	if szReason == "" {
		return notice(G_cszColorWarn, i18n.T("be.session.disconnected"))
	}
	return notice(G_cszColorWarn, i18n.T("be.session.disconnectedReason", "reason", szReason))
}

// MsgReconnectScheduled 는 자동 재접속 예약 안내다.
func MsgReconnectScheduled(iSec int) string {
	return notice(G_cszColorInfo, i18n.T("be.session.reconnectIn", "sec", iSec))
}

// MsgGiveUp 는 재접속을 포기했을 때의 안내다.
func MsgGiveUp() string {
	return notice(G_cszColorInfo, i18n.T("be.session.pressEnter"))
}

// MsgTmuxMissing 은 tmux 가 없을 때 스크립트가 내는 안내다 (session.md §7 1번).
func MsgTmuxMissing() string {
	return notice(G_cszColorWarn, i18n.T("be.session.tmuxMissing"))
}

// g_slErrorRules 는 오류 문구 번역표다 (session.md §3). 위에서 아래 순서로 첫 일치를 쓴다.
// szOutput 은 문구 키(locales/*.json, D75)이고 {raw} 자리에 원문을 넣는다.
var g_slErrorRules = []struct {
	slWords  []string
	szOutput string
}{
	{[]string{"refused", "actively refused"}, "be.err.refused"},
	{[]string{"timed out", "i/o timeout"}, "be.err.timeout"},
	{[]string{"no such host"}, "be.err.noHost"},
	{[]string{"unreachable"}, "be.err.unreachable"},
	{[]string{"unable to authenticate", "no supported methods"}, "be.err.auth"},
	{[]string{"host key rejected"}, "be.err.hostKeyRejected"},
	{[]string{"access is denied", "cannot find the file"}, "be.err.portBusy"},
}

// TranslateError 는 접속 실패 원문을 안내 문구로 바꾼다. 일치하지 않으면 원문을 돌려준다.
func TranslateError(szRaw string) string {
	szLower := strings.ToLower(szRaw)
	for _, ruleItem := range g_slErrorRules {
		for _, szWord := range ruleItem.slWords {
			if strings.Contains(szLower, szWord) {
				return i18n.T(ruleItem.szOutput, "raw", szRaw)
			}
		}
	}
	return szRaw
}

// CanAutoReconnect 는 자동 재접속 조건을 판단한다 (session.md §4).
// bDropped: 연결이 끊긴 경우. iAttempts: 지금까지 재시도 횟수.
func CanAutoReconnect(bAutoOn bool, bSerial bool, bDropped bool, iAttempts int) bool {
	if !bAutoOn || bSerial {
		return false
	}
	if !bDropped && iAttempts == 0 {
		// 처음 접속이 실패했으면 자동 재접속하지 않는다.
		return false
	}
	return iAttempts < G_ciMaxReconnect
}

// BackoffSec 는 시도 횟수에 따른 대기 초다: min(30, 2 × 2^시도) → 2, 4, 8, 16, 30, ...
func BackoffSec(iAttempts int) int {
	iSec := 2
	for iIndex := 0; iIndex < iAttempts && iSec < G_iMaxBackoffSec; iIndex++ {
		iSec *= 2
	}
	if iSec > G_iMaxBackoffSec {
		return G_iMaxBackoffSec
	}
	return iSec
}

var g_reTmuxNameSep = regexp.MustCompile("[.:\\s'\"$\\\\`]+")

// TmuxSessionName 은 세션 이름을 만든다 (session.md §7). 결과가 비었거나 "~", "/" 이면 "dariterm".
func TmuxSessionName(szTmuxName string, szAlias string, szLocalFolder string) string {
	szName := szTmuxName
	if szName == "" {
		szName = szAlias
	}
	if szName == "" {
		szName = lastPathPart(szLocalFolder)
	}
	szName = strings.Trim(g_reTmuxNameSep.ReplaceAllString(szName, "-"), "-")
	if szName == "" || szName == "~" || szName == "/" {
		return "dariterm"
	}
	return szName
}

func lastPathPart(szPath string) string {
	szNorm := strings.ReplaceAll(szPath, "\\", "/")
	szNorm = strings.TrimRight(szNorm, "/")
	if i := strings.LastIndex(szNorm, "/"); i >= 0 {
		return szNorm[i+1:]
	}
	return szNorm
}

// shellQuote 는 셸 인자를 작은따옴표로 감싸고 내부 작은따옴표를 이스케이프한다.
func shellQuote(szValue string) string {
	return "'" + strings.ReplaceAll(szValue, "'", `'\''`) + "'"
}

// TmuxAttachLine 은 접속 직후 셸에 입력하는 한 줄을 만든다 (session.md §7). 앞 공백 하나, 끝에 CR.
// szStartupCommand 는 새 세션을 만들 때만 키 입력으로 보낸다. 줄바꿈은 "; " 로 바꾼다.
func TmuxAttachLine(szSessionName string, szStartupCommand string) string {
	szStartup := strings.ReplaceAll(strings.ReplaceAll(szStartupCommand, "\r\n", "\n"), "\n", "; ")
	// exec 를 쓰지 않는다: tmux 를 끝내면 원래 셸로 돌아온다 (session.md §7 4번, 재접속 루프 방지).
	szQ := shellQuote(szSessionName)
	szScript := "" +
		"command -v tmux >/dev/null 2>&1 || { printf '%s' " + shellQuote(MsgTmuxMissing()) + "; exit 0; }; " +
		"if tmux has-session -t " + szQ + " 2>/dev/null; then " +
		"tmux attach -t " + szQ + "; " +
		"else " +
		"tmux new-session -d -s " + szQ + "; " +
		"tmux set-option -t " + szQ + " mouse on >/dev/null 2>&1; " +
		"tmux set-option -t " + szQ + " status off >/dev/null 2>&1; "
	if szStartup != "" {
		szScript += "tmux send-keys -t " + szQ + " " + shellQuote(szStartup) + " Enter; "
	}
	szScript += "tmux attach -t " + szQ + "; fi"
	return " sh -c " + shellQuote(szScript) + "\r"
}
