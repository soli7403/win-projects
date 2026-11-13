package session

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// EmitFunc 은 이벤트 발행 함수다 (ipc.Router.Emit 과 같은 모양).
type EmitFunc func(szEvent string, anyData ...any)

// G_cdurDefaultOpenDelay 는 열기 후 접속 시작까지의 기본 대기다 (spec/ipc.md session.open).
const G_cdurDefaultOpenDelay = 30 * time.Millisecond

// Session 은 열려 있는 접속 하나다. 상태 전이·재접속·출력 전달을 맡는다.
type Session struct {
	mtxData      sync.Mutex
	cfg          OpenConfig
	info         Info
	emit         EmitFunc
	transport    Transport
	gen          int
	attempts     int
	wasConnected bool
	userClosed   bool
	iCols        int
	iRows        int
	timerRetry   *time.Timer
	timerStartup *time.Timer
	decoder      *Decoder
	batcher      *Batcher
	logger       *Logger
}

// newSession 은 세션을 만든다. 접속은 start 에서 시작한다.
func newSession(cfg OpenConfig, emit EmitFunc) *Session {
	pSession := &Session{
		cfg:     cfg,
		info:    cfg.Info,
		emit:    emit,
		decoder: NewDecoder(cfg.Encoding),
	}
	pSession.info.M_szState = G_cszStateConnecting
	szId := cfg.Info.M_szId
	pSession.batcher = NewBatcher(func(szText string) {
		emit("session.data", szId, szText)
	})
	return pSession
}

// start 는 OpenDelay 뒤에 첫 접속을 시작한다. 로그가 켜져 있으면 연다.
func (pSession *Session) start() {
	if pSession.cfg.LogEnabled && pSession.cfg.DataDir != "" {
		pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
		if errLog == nil {
			pSession.mtxData.Lock()
			pSession.logger = pLogger
			pSession.mtxData.Unlock()
		}
	}
	durDelay := pSession.cfg.OpenDelay
	if durDelay <= 0 {
		durDelay = G_cdurDefaultOpenDelay
	}
	time.AfterFunc(durDelay, pSession.connect)
}

// publishState 는 현재 세션 정보를 session.state 로 보낸다.
func (pSession *Session) publishState() {
	pSession.mtxData.Lock()
	infoCopy := pSession.info
	pSession.mtxData.Unlock()
	pSession.emit("session.state", infoCopy)
}

// writeOutput 은 화면 출력 조각을 묶음기와 로그로 보낸다.
func (pSession *Session) writeOutput(szText string) {
	if szText == "" {
		return
	}
	pSession.batcher.Add(szText)
	pSession.mtxData.Lock()
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.Write(szText)
	}
}

// connect 는 접속 한 번을 시도한다. 성공이면 읽기 루프를 돌린다.
func (pSession *Session) connect() {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	pSession.gen++
	iGen := pSession.gen
	bRetry := pSession.attempts > 0
	pSession.timerRetry = nil
	if bRetry {
		pSession.info.M_szState = G_cszStateReconnecting
	} else {
		pSession.info.M_szState = G_cszStateConnecting
	}
	pSession.info.M_szMessage = ""
	bLocal := pSession.info.M_szProtocol == "local"
	szTitle, szTarget := pSession.info.M_szTitle, pSession.info.M_szTarget
	pSession.mtxData.Unlock()
	pSession.publishState()
	if !bLocal {
		pSession.writeOutput(MsgConnecting(szTitle, szTarget))
	}

	transport, errDial := pSession.cfg.Dialer(pSession.writeOutput)

	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		if transport != nil {
			_ = transport.Close()
		}
		return
	}
	if errDial != nil {
		pSession.mtxData.Unlock()
		pSession.onConnectFailed(errDial, iGen, bLocal)
		return
	}
	pSession.transport = transport
	pSession.attempts = 0
	pSession.wasConnected = true
	pSession.info.M_szState = G_cszStateConnected
	pSession.info.M_szMessage = ""
	pSession.info.M_i64ConnectedAt = time.Now().UnixMilli()
	iCols, iRows := pSession.iCols, pSession.iRows
	pSession.mtxData.Unlock()

	if iCols > 0 && iRows > 0 {
		_ = transport.Resize(iCols, iRows)
	}
	if !bLocal {
		// 접속 중 안내 줄을 지운다 (session.md §3).
		pSession.writeOutput(G_cszClearLine)
	}
	pSession.publishState()
	go pSession.readLoop(transport, iGen)
	pSession.scheduleStartup(iGen)
}

// onConnectFailed 는 접속 실패를 처리한다: 자동 재접속 규칙이 맞으면 예약, 아니면 error 상태.
func (pSession *Session) onConnectFailed(errDial error, iGen int, bLocal bool) {
	szReason := errDial.Error()
	if bLocal {
		pSession.writeOutput(MsgShellFailed(szReason))
	} else {
		pSession.writeOutput(MsgConnectFailed(szReason))
	}
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	bAllowed := CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, false, pSession.attempts)
	if bAllowed {
		pSession.scheduleRetryLocked()
		pSession.mtxData.Unlock()
		pSession.publishState()
		return
	}
	pSession.attempts = 0
	pSession.info.M_szState = G_cszStateError
	pSession.mtxData.Unlock()
	pSession.writeOutput(MsgGiveUp())
	pSession.publishState()
}

// scheduleRetryLocked 는 mtxData 를 잡은 상태에서 다음 재접속을 예약한다 (session.md §4).
func (pSession *Session) scheduleRetryLocked() {
	iDelay := BackoffSec(pSession.attempts)
	pSession.attempts++
	pSession.info.M_szState = G_cszStateReconnecting
	pSession.info.M_szMessage = fmt.Sprintf("%d초 후 재연결 (%d/%d)", iDelay, pSession.attempts, G_ciMaxReconnect)
	pSession.writeOutputUnlockedNotice(MsgReconnectScheduled(iDelay))
	if pSession.timerRetry != nil {
		pSession.timerRetry.Stop()
	}
	pSession.timerRetry = time.AfterFunc(time.Duration(iDelay)*time.Second, pSession.connect)
}

// writeOutputUnlockedNotice 는 mtxData 를 잡은 상태에서도 안전하게 안내를 보낸다 (batcher 와 logger 는 자체 잠금).
func (pSession *Session) writeOutputUnlockedNotice(szText string) {
	pSession.batcher.Add(szText)
	if pSession.logger != nil {
		pSession.logger.Write(szText)
	}
}

// onDisconnected 는 연결이 끊긴 것을 처리한다. 오래된 세대의 알림과 사용자 닫기 뒤의 알림은 무시한다.
func (pSession *Session) onDisconnected(szReason string, iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	pSession.transport = nil
	pSession.writeOutputUnlockedNotice(MsgDisconnected(szReason))
	bAllowed := CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
	if bAllowed {
		pSession.scheduleRetryLocked()
		pSession.mtxData.Unlock()
		pSession.publishState()
		return
	}
	pSession.attempts = 0
	pSession.info.M_szState = G_cszStateClosed
	pSession.info.M_szMessage = ""
	pSession.writeOutputUnlockedNotice(MsgGiveUp())
	pSession.mtxData.Unlock()
	pSession.publishState()
}

// readLoop 는 연결의 출력을 읽어 화면과 로그로 보낸다. 끝나면 끊김 처리한다.
func (pSession *Session) readLoop(transport Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	for {
		nRead, errRead := transport.Read(bBuf)
		if nRead > 0 {
			pSession.writeOutput(pSession.decoder.Decode(bBuf[:nRead]))
		}
		if errRead != nil {
			szReason := ""
			if !errors.Is(errRead, io.EOF) {
				szReason = errRead.Error()
			}
			pSession.onDisconnected(szReason, iGen)
			return
		}
	}
}

// scheduleStartup 은 접속 직후 입력(tmux 한 줄 또는 자동 실행 명령)을 예약한다.
func (pSession *Session) scheduleStartup(iGen int) {
	if pSession.cfg.StartupInput == "" {
		return
	}
	durDelay := pSession.cfg.StartupDelay
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	pSession.timerStartup = time.AfterFunc(durDelay, func() {
		pSession.mtxData.Lock()
		defer pSession.mtxData.Unlock()
		if iGen != pSession.gen || pSession.transport == nil {
			return
		}
		_ = pSession.transport.Write(Encode(pSession.cfg.Encoding, pSession.cfg.StartupInput))
	})
}
