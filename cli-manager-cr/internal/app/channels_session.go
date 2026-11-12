package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"termgo/internal/session"
	"termgo/internal/transport"
)

// hostRecord 는 store 의 서버 항목 중 세션 열기에 쓰는 필드다 (data/store.md §3).
type hostRecord struct {
	M_szId             string            `json:"id"`
	M_szAlias          string            `json:"alias"`
	M_szProtocol       string            `json:"protocol"`
	M_szHost           string            `json:"host"`
	M_iPort            int               `json:"port"`
	M_szUsername       string            `json:"username"`
	M_szColor          string            `json:"color"`
	M_szEnv            string            `json:"env"`
	M_bKeepSession     bool              `json:"keepSession"`
	M_szTmuxName       string            `json:"tmuxName"`
	M_szStartupCommand string            `json:"startupCommand"`
	M_szEncoding       string            `json:"encoding"`
	M_szLocalCwd       string            `json:"-"`
	M_pSerial          *serialRecord     `json:"serial"`
	M_pLocal           *localRecord      `json:"local"`
	M_szLocalShell     string            `json:"-"`
	M_szTermType       string            `json:"termType"`
	M_szCredentialId   string            `json:"credentialId"`
	M_szJumpHostId     string            `json:"jumpHostId"`
	M_pJumpDirect      *jumpDirectRecord `json:"jumpDirect"`
	M_szAuthType       string            `json:"authType"`
	M_iKeepaliveSec    int               `json:"keepaliveSec"`
	M_slForwards       []forwardRecord   `json:"forwards"`
}

// jumpDirectRecord 는 직접 입력한 점프 호스트 정보다 (data/store.md §3 jumpDirect).
type jumpDirectRecord struct {
	M_szHost         string `json:"host"`
	M_iPort          int    `json:"port"`
	M_szUsername     string `json:"username"`
	M_szAuthType     string `json:"authType"`
	M_szCredentialId string `json:"credentialId"`
}

// forwardRecord 는 포트 포워딩 설정 하나다 (data/store.md §3 forwards).
type forwardRecord struct {
	M_szType     string `json:"type"`
	M_szBindHost string `json:"bindHost"`
	M_iBindPort  int    `json:"bindPort"`
	M_szDestHost string `json:"destHost"`
	M_iDestPort  int    `json:"destPort"`
}

// localRecord 는 서버의 로컬 셸 설정이다 (data/store.md §3 local).
type localRecord struct {
	M_szShell string `json:"shell"`
	M_szCwd   string `json:"cwd"`
}

// serialRecord 는 서버의 시리얼 설정이다 (data/store.md §3 serial).
type serialRecord struct {
	M_szPath        string  `json:"path"`
	M_iBaudRate     int     `json:"baudRate"`
	M_iDataBits     int     `json:"dataBits"`
	M_szParity      string  `json:"parity"`
	M_fStopBits     float64 `json:"stopBits"`
	M_szFlowControl string  `json:"flowControl"`
	M_szEnterSends  string  `json:"enterSends"`
	M_bLocalEcho    bool    `json:"localEcho"`
}

// sessionOpenArgs 는 session.open 의 인자 객체다 (spec/ipc.md §2).
type sessionOpenArgs struct {
	M_szHostId *string    `json:"hostId"`
	M_pAdhoc   *adhocArgs `json:"adhoc"`
	M_iCols    int        `json:"cols"`
	M_iRows    int        `json:"rows"`
}

// adhocArgs 는 즉석 접속 정보다.
type adhocArgs struct {
	M_szHost     string `json:"host"`
	M_szCwd      string `json:"cwd"`
	M_iPort      int    `json:"port"`
	M_szUsername string `json:"username"`
	M_szProtocol string `json:"protocol"`
}

// errNotImplemented 는 아직 구현하지 않은 프로토콜의 접속 시도 오류다.
func errNotImplemented(szProtocol string) error {
	return fmt.Errorf("이 프로토콜은 아직 구현되지 않았습니다: %s", szProtocol)
}

// registerSession 은 session.* 과 local.tmuxAvailable 채널을 등록한다 (spec/ipc.md §2 세션).
func (pService *Service) registerSession() {
	pRouter := pService.pRouter
	pRouter.Handle("session.open", func(slArgs []json.RawMessage) (any, error) {
		var args sessionOpenArgs
		if errArg := argInto(slArgs, 0, &args); errArg != nil {
			return nil, errArg
		}
		return pService.openSession(args)
	})
	pRouter.Handle("session.close", func(slArgs []json.RawMessage) (any, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		pService.pSessions.Close(szId)
		return nil, nil
	})
	pRouter.Handle("session.reconnect", func(slArgs []json.RawMessage) (any, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		pSession, errGet := pService.pSessions.Get(szId)
		if errGet != nil {
			return nil, errGet
		}
		pSession.Reconnect()
		return nil, nil
	})
	pRouter.Handle("local.tmuxAvailable", func(slArgs []json.RawMessage) (any, error) {
		return session.TmuxAvailable(), nil
	})
	pRouter.Handle("local.shells", func(slArgs []json.RawMessage) (any, error) {
		return listLocalShells(), nil
	})
	pRouter.Handle("serial.list", func(slArgs []json.RawMessage) (any, error) {
		return transport.ListSerialPorts(), nil
	})
}

// openSession 은 저장된 서버 또는 즉석 정보로 세션을 연다.
func (pService *Service) openSession(args sessionOpenArgs) (session.Info, error) {
	var record hostRecord
	var bHasSaved bool
	if args.M_szHostId != nil {
		record, bHasSaved = pService.findHost(*args.M_szHostId)
	}
	if !bHasSaved {
		if args.M_pAdhoc == nil {
			return session.Info{}, errors.New("서버를 찾을 수 없습니다")
		}
		record = adhocRecord(*args.M_pAdhoc)
	}
	cfg := session.OpenConfig{
		Info: session.Info{
			M_szTitle:      record.M_szAlias,
			M_szProtocol:   record.M_szProtocol,
			M_szState:      session.G_cszStateConnecting,
			M_bCanSftp:     record.M_szProtocol == "ssh",
			M_bKeepSession: record.M_bKeepSession,
		},
		Encoding:      record.M_szEncoding,
		AutoReconnect: pService.settingBool("autoReconnect", true),
		IsSerial:      record.M_szProtocol == "serial",
		Dialer:        pService.dialerFor(record),
		DataDir:       pService.szDataDir,
		LogEnabled:    pService.settingBool("sessionLog", false),
		LogAlias:      record.M_szAlias,
		OpenDelay:     0,
	}
	if bHasSaved {
		szId := record.M_szId
		cfg.Info.M_pszHostId = &szId
		cfg.Info.M_szColor = record.M_szColor
		cfg.Info.M_szEnv = record.M_szEnv
	} else {
		cfg.Info.M_szEnv = "none"
	}
	cfg.Info.M_szTarget = targetOf(record)
	cfg.StartupInput, cfg.StartupDelay = startupInputOf(record)
	info, errOpen := pService.pSessions.Open(cfg)
	if errOpen == nil && args.M_iCols > 0 && args.M_iRows > 0 {
		// 열 때 지정한 크기는 접속 직후 프로토콜에 알린다 (session.md §8).
		if pSession, errGet := pService.pSessions.Get(info.M_szId); errGet == nil {
			pSession.Resize(args.M_iCols, args.M_iRows)
		}
	}
	return info, errOpen
}

// findHost 는 저장 데이터에서 서버 id 를 찾는다.
func (pService *Service) findHost(szId string) (hostRecord, bool) {
	rawHosts, bOk := pService.pStore.Get("hosts")
	if !bOk {
		return hostRecord{}, false
	}
	var slHosts []hostRecord
	if errUnmarshal := json.Unmarshal(rawHosts, &slHosts); errUnmarshal != nil {
		return hostRecord{}, false
	}
	for _, record := range slHosts {
		if record.M_szId == szId {
			if record.M_pLocal != nil {
				record.M_szLocalCwd = record.M_pLocal.M_szCwd
				record.M_szLocalShell = record.M_pLocal.M_szShell
			}
			return record, true
		}
	}
	return hostRecord{}, false
}

// adhocRecord 는 즉석 접속 기본값을 채운다 (session.md §1).
func adhocRecord(adhoc adhocArgs) hostRecord {
	szProtocol := adhoc.M_szProtocol
	if szProtocol == "" {
		szProtocol = "ssh"
	}
	iPort := adhoc.M_iPort
	if iPort == 0 {
		iPort = 22
		if szProtocol == "telnet" {
			iPort = 23
		}
	}
	szAlias := adhoc.M_szHost
	if adhoc.M_szUsername != "" {
		szAlias = adhoc.M_szUsername + "@" + adhoc.M_szHost
	}
	if szProtocol == "local" {
		szAlias = "로컬 터미널"
	}
	return hostRecord{
		M_szAlias:    szAlias,
		M_szProtocol: szProtocol,
		M_szHost:     adhoc.M_szHost,
		M_iPort:      iPort,
		M_szUsername: adhoc.M_szUsername,
		M_szEncoding: "utf-8",
		M_szLocalCwd: adhoc.M_szCwd,
	}
}

// targetOf 는 대상 문자열을 만든다 (session.md §1). 로컬 셸의 표기 규칙은 Q10 미정이라 임시 값이다.
func targetOf(record hostRecord) string {
	switch record.M_szProtocol {
	case "local":
		szCwd := record.M_szLocalCwd
		if szCwd == "" {
			szCwd = "~"
		}
		return "로컬 · " + szCwd
	case "serial":
		szPath := "?"
		iBaud := 9600
		if record.M_pSerial != nil {
			if record.M_pSerial.M_szPath != "" {
				szPath = record.M_pSerial.M_szPath
			}
			if record.M_pSerial.M_iBaudRate > 0 {
				iBaud = record.M_pSerial.M_iBaudRate
			}
		}
		return szPath + " @ " + strconv.Itoa(iBaud)
	default:
		szUser := ""
		if record.M_szUsername != "" {
			szUser = record.M_szUsername + "@"
		}
		return szUser + record.M_szHost + ":" + strconv.Itoa(record.M_iPort)
	}
}

// startupInputOf 는 접속 직후 셸에 보낼 입력과 대기 시간을 정한다 (session.md §5, §7).
func startupInputOf(record hostRecord) (string, time.Duration) {
	if record.M_bKeepSession && record.M_szProtocol == "ssh" {
		szName := session.TmuxSessionName(record.M_szTmuxName, record.M_szAlias, record.M_szLocalCwd)
		return session.TmuxAttachLine(szName, record.M_szStartupCommand), 500 * time.Millisecond
	}
	if record.M_szStartupCommand != "" {
		szCmd := strings.ReplaceAll(strings.ReplaceAll(record.M_szStartupCommand, "\r\n", "\r"), "\n", "\r")
		return szCmd + "\r", 400 * time.Millisecond
	}
	return "", 0
}

// listLocalShells 는 local.shells 채널 결과다.
func listLocalShells() []transport.ShellItem {
	return transport.ListShells()
}

// settingBool 은 설정의 불리언을 읽는다. 없으면 기본값 (data/store.md §4).
func (pService *Service) settingBool(szKey string, bDefault bool) bool {
	mapSnap, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return bDefault
	}
	mapSettings, bOk := mapSnap["settings"].(map[string]any)
	if !bOk {
		return bDefault
	}
	bValue, bIsBool := mapSettings[szKey].(bool)
	if !bIsBool {
		return bDefault
	}
	return bValue
}
