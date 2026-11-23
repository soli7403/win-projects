package app

import (
	"dariterm/internal/i18n"
	"os"
	"path/filepath"
	"strings"

	"dariterm/internal/session"
	"dariterm/internal/transport"
)

// dialerFor 는 프로토콜별 접속 함수를 고른다 (protocols.md §2~§6).
// 두 번째 값은 세션을 닫을 때 부를 정리 함수다 (없으면 nil).
func (pService *Service) dialerFor(record hostRecord) (session.Dialer, func()) {
	if record.M_szProtocol == "ssh" {
		return pService.sshDialer(record)
	}
	return pService.plainDialerFor(record), nil
}

// plainDialerFor 는 세션 상태가 없는 프로토콜의 접속 함수다.
func (pService *Service) plainDialerFor(record hostRecord) session.Dialer {
	switch record.M_szProtocol {
	case "local":
		return func(notify func(string)) (session.Transport, error) {
			return openLocalTransport(record, notify)
		}
	case "telnet":
		return func(notify func(string)) (session.Transport, error) {
			pConn, errDial := transport.DialTelnet(record.M_szHost, record.M_iPort, record.M_szTermType)
			if errDial != nil {
				return nil, errDial
			}
			return pConn, nil
		}
	case "serial":
		return func(notify func(string)) (session.Transport, error) {
			return openSerialTransport(record)
		}
	default:
		return func(notify func(string)) (session.Transport, error) {
			return nil, errNotImplemented(record.M_szProtocol)
		}
	}
}

// openSerialTransport 는 시리얼 설정으로 장치를 연다 (protocols.md §5).
func openSerialTransport(record hostRecord) (session.Transport, error) {
	opts := transport.SerialOptions{Path: "", BaudRate: 9600, Parity: "none", StopBits: 1, EnterSends: "CR"}
	if record.M_pSerial != nil {
		opts.Path = record.M_pSerial.M_szPath
		if record.M_pSerial.M_iBaudRate > 0 {
			opts.BaudRate = record.M_pSerial.M_iBaudRate
		}
		opts.DataBits = record.M_pSerial.M_iDataBits
		if record.M_pSerial.M_szParity != "" {
			opts.Parity = record.M_pSerial.M_szParity
		}
		if record.M_pSerial.M_fStopBits > 0 {
			opts.StopBits = record.M_pSerial.M_fStopBits
		}
		if record.M_pSerial.M_szEnterSends != "" {
			opts.EnterSends = record.M_pSerial.M_szEnterSends
		}
		opts.LocalEcho = record.M_pSerial.M_bLocalEcho
		opts.FlowControl = record.M_pSerial.M_szFlowControl // none · rtscts · xonxoff (protocols.md §5, D108)
	}
	if opts.Path == "" {
		return nil, i18n.NewError("be.serial.noDevice")
	}
	pPort, errOpen := transport.OpenSerial(opts)
	if errOpen != nil {
		return nil, errOpen
	}
	return pPort, nil
}

// openLocalTransport 는 로컬 셸을 가상 터미널 위에 띄운다 (protocols.md §6).
// 시작 폴더가 없거나 폴더가 아니면 홈으로 연다.
func openLocalTransport(record hostRecord, notify func(string)) (session.Transport, error) {
	szShell := record.M_szLocalShell
	if szShell == "" {
		shells := transport.ListShells()
		if len(shells) == 0 {
			return nil, i18n.NewError("be.shell.notFound")
		}
		szShell = shells[0].Path
	}
	szShell = transport.ResolveShellPath(szShell)
	var slArgs []string
	if szBase := strings.ToLower(filepath.Base(szShell)); strings.Contains(szBase, "powershell") || strings.Contains(szBase, "pwsh") {
		slArgs = []string{"-NoLogo"}
	}
	szCwd := expandHome(record.M_szLocalCwd)
	if info, errStat := os.Stat(szCwd); szCwd == "" || errStat != nil || !info.IsDir() {
		if record.M_szLocalCwd != "" {
			// 지정한 시작 폴더가 없으면 홈으로 열고 노랑 안내 줄을 보인다 (session.md §3).
			notify("\x1b[33m● " + i18n.T("be.local.noStartDir", "dir", record.M_szLocalCwd) + "\x1b[0m\r\n")
		}
		szCwd, _ = os.UserHomeDir()
	}
	szTerm := record.M_szTermType
	if szTerm == "" {
		szTerm = "xterm-256color"
	}
	slEnv := append(os.Environ(), "TERM="+szTerm, "COLORTERM=truecolor", "TERM_PROGRAM=DariTerm")
	pLocal, errOpen := transport.OpenLocal(transport.LocalOptions{
		Shell: szShell,
		Args:  slArgs,
		Cwd:   szCwd,
		Cols:  120,
		Rows:  32,
		Env:   slEnv,
	})
	if errOpen != nil {
		return nil, errOpen
	}
	return pLocal, nil
}

// expandHome 은 맨 앞의 ~ 를 홈 폴더로 바꾼다.
func expandHome(szPath string) string {
	if szPath == "~" || strings.HasPrefix(szPath, "~/") || strings.HasPrefix(szPath, `~\`) {
		if szHome, errHome := os.UserHomeDir(); errHome == nil {
			return szHome + szPath[1:]
		}
	}
	return szPath
}
