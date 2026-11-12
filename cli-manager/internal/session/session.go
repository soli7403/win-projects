// Package session 은 열려 있는 터미널 세션(연결 상태·재접속·인코딩·출력 묶음 전송)을 관리한다.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"termgo/internal/model"
	"termgo/internal/sshconn"
	"termgo/internal/store"
	"termgo/internal/transport"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

// Info 는 SessionInfo(shared/types.ts).
type Info struct {
	ID          string  `json:"id"`
	HostID      *string `json:"hostId"`
	Title       string  `json:"title"`
	Protocol    string  `json:"protocol"`
	Target      string  `json:"target"`
	Color       string  `json:"color,omitempty"`
	Env         string  `json:"env,omitempty"`
	State       string  `json:"state"`
	ConnectedAt int64   `json:"connectedAt,omitempty"`
	Message     string  `json:"message,omitempty"`
	CanSftp     bool    `json:"canSftp"`
	// KeepSession: tmux 세션 유지 중 (패널 머리글 "유지" 배지)
	KeepSession bool `json:"keepSession,omitempty"`
}

// Deps 는 세션이 쓰는 서비스들.
type Deps struct {
	Store *store.Store
	Vault *vault.Vault
	UI    *ui.Bridge
	// DataDir 는 로그 폴더의 부모.
	DataDir string
}

const flushInterval = 12 * time.Millisecond

var reANSI = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b[()][0-9A-Za-z]|\x1b[=>78DEHMc]")

// Session 은 세션 하나.
type Session struct {
	d          Deps
	mu         sync.Mutex
	info       Info
	host       model.Host
	persistent bool
	tr         transport.Transport
	ssh        *sshconn.Shell
	gen        int // 연결 세대: 오래된 트랜스포트의 콜백을 무시하기 위해
	dec        *streamDecoder
	cols, rows int
	userClosed bool
	retry      int
	retryTimer *time.Timer
	buf        strings.Builder
	flushTimer *time.Timer
	logf       *os.File
}

func newSession(d Deps, h model.Host, bPersistent bool) *Session {
	pstrS := &Session{d: d, host: h, persistent: bPersistent, cols: 120, rows: 32}
	pstrS.info = Info{
		ID: vault.NewUUID(), Title: h.Alias, Protocol: h.Protocol, Target: targetLabel(h),
		State: "connecting", CanSftp: h.Protocol == "ssh", KeepSession: KeepSessionApplies(h, runtime.GOOS),
	}
	if bPersistent {
		sz := h.ID
		pstrS.info.HostID = &sz
		pstrS.info.Color = resolveColor(d.Store, h)
		pstrS.info.Env = resolveEnv(d.Store, h)
	} else {
		pstrS.info.Env = "none"
	}
	return pstrS
}

func targetLabel(h model.Host) string {
	switch h.Protocol {
	case "serial":
		path, baud := "?", 9600
		if h.Serial != nil {
			path = h.Serial.Path
			if h.Serial.BaudRate != 0 {
				baud = h.Serial.BaudRate
			}
		}
		return fmt.Sprintf("%s @ %d", path, baud)
	case "local":
		cwd := "~"
		if h.Local != nil && strings.TrimSpace(h.Local.Cwd) != "" {
			cwd = strings.TrimSpace(h.Local.Cwd)
		}
		return "로컬 · " + cwd
	}
	sz := fmt.Sprintf("%s:%d", h.Host, h.Port)
	if h.Username != "" {
		sz = h.Username + "@" + sz
	}
	return sz
}

// Info 는 현재 정보의 복사본.
func (pstrS *Session) Info() Info {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	return pstrS.info
}

func (pstrS *Session) setState(szState, szMsg string) {
	pstrS.mu.Lock()
	pstrS.info.State, pstrS.info.Message = szState, szMsg
	if szState == "connected" {
		pstrS.info.ConnectedAt = time.Now().UnixMilli()
	}
	info := pstrS.info
	pstrS.mu.Unlock()
	pstrS.d.UI.Send("session:state", info)
}

// emitText 는 출력을 짧은 간격으로 묶어 렌더러에 보낸다.
func (pstrS *Session) emitText(sz string) {
	if sz == "" {
		return
	}
	pstrS.mu.Lock()
	pstrS.buf.WriteString(sz)
	if pstrS.flushTimer == nil {
		pstrS.flushTimer = time.AfterFunc(flushInterval, pstrS.flush)
	}
	pstrS.mu.Unlock()
}

func (pstrS *Session) flush() {
	pstrS.mu.Lock()
	sz := pstrS.buf.String()
	pstrS.buf.Reset()
	pstrS.flushTimer = nil
	id := pstrS.info.ID
	pstrS.mu.Unlock()
	if sz != "" {
		pstrS.d.UI.Send("session:data", id, sz)
	}
}

func (pstrS *Session) note(szMsg, szColor string) {
	pstrS.emitText(fmt.Sprintf("\r\n\x1b[%sm● %s\x1b[0m\r\n", szColor, szMsg))
}

// SetSize 는 터미널 크기를 기억하고 트랜스포트에 전달한다.
func (pstrS *Session) SetSize(iCols, iRows int) {
	pstrS.mu.Lock()
	pstrS.cols, pstrS.rows = iCols, iRows
	tr := pstrS.tr
	pstrS.mu.Unlock()
	if tr != nil {
		tr.Resize(iCols, iRows)
	}
}

func (pstrS *Session) encoding() string {
	if pstrS.host.Protocol == "local" {
		return "utf-8"
	}
	return pstrS.host.Encoding
}

// Connect 는 (재)접속을 시도한다. 블로킹이므로 goroutine 에서 부른다.
func (pstrS *Session) Connect() {
	defer func() {
		if r := recover(); r != nil { // 접속 중 어떤 오류·패닉도 해당 패널의 "연결 실패" 상태로만 처리 (brief 3.7)
			pstrS.note(fmt.Sprintf("접속 실패: %v", r), "31")
			pstrS.scheduleReconnect(true)
		}
	}()
	pstrS.mu.Lock()
	if pstrS.userClosed {
		pstrS.mu.Unlock()
		return
	}
	pstrS.gen++
	iGen := pstrS.gen
	h := pstrS.host
	iCols, iRows := pstrS.cols, pstrS.rows
	bRetry := pstrS.retry > 0
	pstrS.dec = &streamDecoder{enc: encodingFor(pstrS.encoding())}
	szTitle, szTarget := pstrS.info.Title, pstrS.info.Target
	pstrS.mu.Unlock()

	if bRetry {
		pstrS.setState("reconnecting", "")
	} else {
		pstrS.setState("connecting", "")
	}
	if h.Protocol != "local" {
		pstrS.note(fmt.Sprintf("%s (%s) 접속 중...", szTitle, szTarget), "90")
	}

	cb := transport.Callbacks{
		OnData:  func(arr []byte) { pstrS.onData(iGen, arr) },
		OnClose: func(szReason string) { pstrS.onClose(iGen, szReason) },
	}
	var tr transport.Transport
	var shell *sshconn.Shell
	var err error
	switch h.Protocol {
	case "ssh":
		shell, err = sshconn.StartShell(sshconn.Deps{Store: pstrS.d.Store, Vault: pstrS.d.Vault, UI: pstrS.d.UI}, h, sshconn.Context{
			Title: h.Alias, Cols: iCols, Rows: iRows,
			OnInfo:         func(s string) { pstrS.note(s, "90") },
			OnSavePassword: pstrS.savePassword,
		}, cb)
		if err == nil {
			tr = shell
		}
	case "local":
		var l *transport.Local
		opt := transport.LocalOptions{}
		if h.Local != nil {
			opt = *h.Local
		}
		l, err = transport.StartLocal(opt, h.TermType, iCols, iRows, cb)
		if err == nil {
			tr = l
			if l.CwdFallback {
				pstrS.note(fmt.Sprintf("시작 폴더(%s)가 없어 홈 폴더에서 엽니다", opt.Cwd), "33")
			}
		}
	case "telnet":
		var t *transport.Telnet
		t, err = transport.StartTelnet(h.Host, h.Port, h.TermType, iCols, iRows, cb)
		if err == nil {
			tr = t
		}
	case "serial":
		if h.Serial == nil {
			err = fmt.Errorf("시리얼 설정이 없습니다")
		} else {
			var s *transport.Serial
			s, err = transport.StartSerial(*h.Serial, cb)
			if err == nil {
				tr = s
			}
		}
	default:
		err = fmt.Errorf("지원하지 않는 프로토콜: %s", h.Protocol)
	}
	if err != nil {
		szWhat := "접속 실패"
		if h.Protocol == "local" {
			szWhat = "셸 실행 실패"
		}
		pstrS.note(fmt.Sprintf("%s: %s", szWhat, translateError(err.Error())), "31")
		pstrS.scheduleReconnect(true)
		return
	}
	pstrS.mu.Lock()
	if pstrS.userClosed || pstrS.gen != iGen {
		pstrS.mu.Unlock()
		tr.Close()
		return
	}
	pstrS.tr, pstrS.ssh = tr, shell
	pstrS.retry = 0
	pstrS.mu.Unlock()
	pstrS.setState("connected", "")
	if h.Protocol != "local" {
		pstrS.emitText("\x1b[2K\r")
	}
	if pstrS.persistent {
		pstrS.d.Store.TouchHost(h.ID)
	}
	if pstrS.d.Store.SettingBool("sessionLog") {
		pstrS.openLog()
	}
	if KeepSessionApplies(h, runtime.GOOS) {
		// 3.3: tmux 세션에 붙는다 (자동 실행 명령은 세션을 처음 만들 때만 그 안에서 실행)
		szTmux := BuildTmuxCommand(TmuxSessionName(h), h.StartupCommand)
		time.AfterFunc(500*time.Millisecond, func() { _ = tr.Write([]byte(szTmux)) })
	} else if h.StartupCommand != "" {
		szCmd := strings.ReplaceAll(strings.ReplaceAll(h.StartupCommand, "\r\n", "\r"), "\n", "\r") + "\r"
		time.AfterFunc(400*time.Millisecond, func() { _ = tr.Write([]byte(szCmd)) })
	}
}

func (pstrS *Session) savePassword(szUser, szPassword string) {
	if !pstrS.d.Vault.IsUnlocked() || !pstrS.persistent {
		return
	}
	in := vault.Input{Name: pstrS.host.Alias, Kind: "password", Username: &szUser, Password: &szPassword}
	if pstrS.host.CredentialID != nil {
		in.ID = *pstrS.host.CredentialID
	}
	meta, err := pstrS.d.Vault.Save(in)
	if err != nil {
		return
	}
	m := map[string]any{"credentialId": meta.ID}
	if pstrS.host.Username == "" {
		m["username"] = szUser
	}
	if pstrS.host.AuthType == "ask" {
		m["authType"] = "password"
	}
	pstrS.d.Store.PatchHost(pstrS.host.ID, m)
	if h, bOk := pstrS.d.Store.HostByID(pstrS.host.ID); bOk {
		pstrS.mu.Lock()
		pstrS.host = h
		pstrS.mu.Unlock()
	}
	pstrS.d.UI.Send("store:changed")
	pstrS.note("비밀번호를 볼트에 저장했습니다", "90")
}

func (pstrS *Session) openLog() {
	szDir := filepath.Join(pstrS.d.DataDir, "logs")
	if os.MkdirAll(szDir, 0o700) != nil {
		return
	}
	szSafe := strings.NewReplacer(`\`, "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_").Replace(pstrS.host.Alias)
	f, err := os.OpenFile(filepath.Join(szDir, fmt.Sprintf("%s-%s.log", szSafe, time.Now().Format("20060102-1504"))), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		pstrS.mu.Lock()
		pstrS.logf = f
		pstrS.mu.Unlock()
	}
}

func (pstrS *Session) onData(iGen int, arr []byte) {
	pstrS.mu.Lock()
	if pstrS.gen != iGen || pstrS.dec == nil {
		pstrS.mu.Unlock()
		return
	}
	sz := pstrS.dec.Write(arr)
	logf := pstrS.logf
	pstrS.mu.Unlock()
	if sz == "" {
		return
	}
	pstrS.emitText(sz)
	if logf != nil {
		_, _ = logf.WriteString(strings.ReplaceAll(reANSI.ReplaceAllString(sz, ""), "\r", ""))
	}
}

func (pstrS *Session) onClose(iGen int, szReason string) {
	pstrS.mu.Lock()
	if pstrS.gen != iGen {
		pstrS.mu.Unlock()
		return
	}
	pstrS.tr, pstrS.ssh = nil, nil
	if pstrS.logf != nil {
		_ = pstrS.logf.Close()
		pstrS.logf = nil
	}
	bUser := pstrS.userClosed
	bLocal := pstrS.host.Protocol == "local"
	pstrS.mu.Unlock()
	if bUser {
		return
	}
	if bLocal {
		pstrS.setState("closed", "")
		if szReason == "" {
			szReason = "셸이 종료되었습니다"
		}
		pstrS.note(szReason+" — Enter 키를 누르면 새 셸을 엽니다.", "90")
		return
	}
	if szReason != "" {
		pstrS.note("연결 끊김: "+szReason, "33")
	} else {
		pstrS.note("연결이 끊어졌습니다", "33")
	}
	pstrS.scheduleReconnect(false)
}

func (pstrS *Session) scheduleReconnect(bFailed bool) {
	bAuto := pstrS.d.Store.SettingBool("autoReconnect") && pstrS.host.Protocol != "serial"
	pstrS.mu.Lock()
	iRetry := pstrS.retry
	pstrS.mu.Unlock()
	if bAuto && (!bFailed || iRetry > 0) && iRetry < 8 {
		iDelay := min(30, (1<<iRetry)*2)
		pstrS.mu.Lock()
		pstrS.retry++
		n := pstrS.retry
		pstrS.mu.Unlock()
		pstrS.setState("reconnecting", fmt.Sprintf("%d초 후 재접속 (%d/8)", iDelay, n))
		pstrS.note(fmt.Sprintf("%d초 후 자동 재접속합니다... (Enter: 즉시 재접속)", iDelay), "90")
		t := time.AfterFunc(time.Duration(iDelay)*time.Second, func() { pstrS.Connect() })
		pstrS.mu.Lock()
		pstrS.retryTimer = t
		pstrS.mu.Unlock()
		return
	}
	pstrS.mu.Lock()
	pstrS.retry = 0
	pstrS.mu.Unlock()
	if bFailed {
		pstrS.setState("error", "")
	} else {
		pstrS.setState("closed", "")
	}
	pstrS.note("Enter 키를 누르면 다시 접속합니다.", "90")
}

// ReconnectNow 는 끊긴 세션을 즉시 다시 연결한다.
func (pstrS *Session) ReconnectNow() {
	pstrS.mu.Lock()
	if pstrS.tr != nil {
		pstrS.mu.Unlock()
		return
	}
	if pstrS.retryTimer != nil {
		pstrS.retryTimer.Stop()
		pstrS.retryTimer = nil
	}
	pstrS.mu.Unlock()
	go pstrS.Connect()
}

// Write 는 터미널 입력(UTF-8 문자열)을 호스트 인코딩으로 바꿔 보낸다.
func (pstrS *Session) Write(sz string) {
	pstrS.mu.Lock()
	tr, szState := pstrS.tr, pstrS.info.State
	enc := encodingFor(pstrS.encoding())
	pstrS.mu.Unlock()
	if tr == nil {
		if sz == "\r" && szState != "connecting" {
			pstrS.ReconnectNow()
		}
		return
	}
	_ = tr.Write(encodeInput(enc, sz))
}

// LocalCwd 는 로컬 세션의 시작 폴더 (설정이 없거나 폴더가 없으면 "").
func (pstrS *Session) LocalCwd() string {
	pstrS.mu.Lock()
	h := pstrS.host
	pstrS.mu.Unlock()
	if h.Protocol != "local" || h.Local == nil || strings.TrimSpace(h.Local.Cwd) == "" {
		return ""
	}
	sz := transport.ExpandHome(strings.TrimSpace(h.Local.Cwd))
	if st, err := os.Stat(sz); err != nil || !st.IsDir() {
		return ""
	}
	return sz
}

// Shell 은 SSH 세션이면 셸(연결 공유용)을 돌려준다.
func (pstrS *Session) Shell() *sshconn.Shell {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	return pstrS.ssh
}

// Close 는 사용자가 세션을 닫을 때.
func (pstrS *Session) Close() {
	pstrS.mu.Lock()
	pstrS.userClosed = true
	if pstrS.retryTimer != nil {
		pstrS.retryTimer.Stop()
	}
	tr := pstrS.tr
	pstrS.tr, pstrS.ssh = nil, nil
	if pstrS.logf != nil {
		_ = pstrS.logf.Close()
	}
	pstrS.mu.Unlock()
	if tr != nil {
		tr.Close()
	}
}

func translateError(m string) string {
	l := strings.ToLower(m)
	switch {
	case strings.Contains(l, "refused") || strings.Contains(l, "actively refused"):
		return "서버가 접속을 거부했습니다 (포트 확인)"
	case strings.Contains(l, "timed out") || strings.Contains(l, "i/o timeout"):
		return "응답 시간 초과 (주소·방화벽 확인)"
	case strings.Contains(l, "no such host") || strings.Contains(l, "no such host is known"):
		return "호스트 이름을 찾을 수 없습니다"
	case strings.Contains(l, "unreachable"):
		return "네트워크에 도달할 수 없습니다"
	case strings.Contains(l, "unable to authenticate") || strings.Contains(l, "no supported methods"):
		return "인증 실패 (사용자 이름/비밀번호/키 확인)"
	case strings.Contains(l, "host key rejected"):
		return "호스트 키를 승인하지 않아 접속을 중단했습니다"
	case strings.Contains(l, "access is denied") || strings.Contains(l, "cannot find the file"):
		return m + " (포트 사용 중이거나 권한 없음)"
	}
	return m
}
