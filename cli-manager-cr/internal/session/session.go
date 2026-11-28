package session

import (
	"dariterm/internal/i18n"
	"dariterm/internal/transport"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"dariterm/internal/zmodem"
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
	timerLogin   *time.Timer  // 로그인 스크립트의 단계 시한 (session.md §5.1)
	login        *loginRunner // 지금 연결에서 진행 중인 로그인 스크립트 (없으면 nil)
	decoder      *Decoder
	batcher      *Batcher
	credit       *Credit // 출력 흐름 제어 (session.md §8)
	logger       *Logger
	dialing      bool           // Dialer 를 부르는 중 (같은 세션에 접속이 두 번 겹치지 않게)
	waitUnlock   bool           // 앱 잠금 해제를 기다리는 중 (ErrWaitUnlock, session.md §4)
	tConnected   time.Time      // 지금 연결이 맺어진 시각 (서버 종료가 사용자 종료인지 가릴 때 쓴다)
	tLastInput   time.Time      // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
	iLogSyncSec  int            // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
	bLogStamp    bool           // 세션 로그 줄 앞에 시각을 붙일지. 설정을 바꾸면 SetLogTimestamp 로 갱신된다
	bLogAnsi     bool           // 세션 로그에 ANSI 시퀀스를 그대로 쓸지. 설정을 바꾸면 SetLogAnsi 로 갱신된다
	zActive      atomic.Bool    // ZMODEM 전송 중 (터미널 입력을 서버로 보내지 않는다)
	zCancel      atomic.Bool    // 사용자가 ZMODEM 전송을 취소했다
	traffic      trafficCounter // 세션을 연 뒤 연결에서 읽은·쓴 바이트 (traffic.go, D160)
}

// G_cdurUserExitWindow 는 서버가 셸을 끝냈을 때 "사용자가 끝낸 것"으로 보는 시간 창이다 (session.md §4).
// 끝나기 전 이 시간 안에 사용자 입력이 있었거나, 연결된 지 이 시간이 안 됐으면 사용자가 끝낸 것으로 본다.
// 그 밖(입력 없이 한참 있다가 끝남)은 서버의 유휴 로그아웃 같은 서버 쪽 종료로 보고 자동 재접속한다.
const G_cdurUserExitWindow = 10 * time.Second

// newSession 은 세션을 만든다. 접속은 start 에서 시작한다.
func newSession(cfg OpenConfig, emit EmitFunc) *Session {
	pSession := &Session{
		cfg:     cfg,
		info:    cfg.Info,
		emit:    emit,
		decoder: NewDecoder(cfg.Encoding),

		iLogSyncSec: ClampLogSyncSec(cfg.LogSyncSec),
		bLogStamp:   cfg.LogTimestamp,
		bLogAnsi:    cfg.LogAnsi,
	}
	pSession.info.M_szState = G_cszStateConnecting
	szId := cfg.Info.M_szId
	pSession.credit = NewCredit()
	pCredit := pSession.credit
	pSession.batcher = NewBatcher(func(szText string) {
		pCredit.Sent(len(szText)) // emit 보다 먼저 (ack 가 먼저 오면 빈 큐에서 무시되므로)
		emit("session.data", szId, szText)
	})
	return pSession
}

// start 는 OpenDelay 뒤에 첫 접속을 시작한다. 로그가 켜져 있으면 연다.
func (pSession *Session) start() {
	if pSession.cfg.LogEnabled {
		pSession.SetLogging(true)
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
	pSession.dialing = true
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

	conn, errDial := pSession.cfg.Dialer(pSession.writeOutput)

	pSession.mtxData.Lock()
	if iGen == pSession.gen {
		pSession.dialing = false
	}
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		return
	}
	if errors.Is(errDial, ErrWaitUnlock) {
		pSession.onWaitUnlockLocked()
		pSession.mtxData.Unlock()
		pSession.publishState()
		return
	}
	pSession.waitUnlock = false
	if errDial != nil {
		pSession.mtxData.Unlock()
		pSession.onConnectFailed(errDial, iGen, bLocal)
		return
	}
	pSession.transport = conn
	pSession.attempts = 0
	pSession.wasConnected = true
	pSession.info.M_szState = G_cszStateConnected
	pSession.info.M_szMessage = ""
	tNow := time.Now()
	pSession.info.M_i64ConnectedAt = tNow.UnixMilli()
	pSession.tConnected = tNow
	pSession.tLastInput = time.Time{}
	pSession.info.M_slForwards = nil
	if reporter, bOk := conn.(ForwardReporter); bOk {
		pSession.info.M_slForwards = reporter.Forwards()
	}
	iCols, iRows := pSession.iCols, pSession.iRows
	pSession.mtxData.Unlock()

	if iCols > 0 && iRows > 0 {
		_ = conn.Resize(iCols, iRows)
	}
	if !bLocal {
		// 접속 중 안내 줄을 지운다 (session.md §3).
		pSession.writeOutput(G_cszClearLine)
	}
	pSession.publishState()
	// 읽기 루프보다 먼저 건다: 접속 직후 이미 와 있는 출력(첫 프롬프트)을 스크립트가 놓치지 않게.
	pSession.startLogin(conn, iGen)
	go pSession.readLoop(conn, iGen)
}

// onConnectFailed 는 접속 실패를 처리한다: 자동 재접속 규칙이 맞으면 예약, 아니면 error 상태.
func (pSession *Session) onConnectFailed(errDial error, iGen int, bLocal bool) {
	if !pSession.isCurrent(iGen) {
		return // 닫았거나 다른 시도가 시작됐다: 안내도 내지 않는다 (session.md §8)
	}
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

// onWaitUnlockLocked 는 mtxData 를 잡은 상태에서 "앱 잠금 해제 대기"로 둔다 (session.md §4).
// 안내 줄은 대기에 처음 들어올 때 한 번만 낸다. 재시도 횟수는 늘리지 않고 타이머도 걸지 않는다 —
// 볼트가 열리면 Manager.ResumeUnlockWaiters 가 다시 접속시킨다.
func (pSession *Session) onWaitUnlockLocked() {
	if !pSession.waitUnlock {
		pSession.writeOutputUnlockedNotice(MsgWaitUnlock())
	}
	pSession.waitUnlock = true
	pSession.info.M_szState = G_cszStateReconnecting
	pSession.info.M_szMessage = i18n.T("be.session.waitUnlock")
}

// resumeIfWaiting 은 잠금 해제를 기다리던 세션이면 곧바로 다시 접속한다. 기다리던 세션이었는지 돌려준다.
func (pSession *Session) resumeIfWaiting() bool {
	pSession.mtxData.Lock()
	if !pSession.waitUnlock || pSession.userClosed || pSession.dialing || pSession.transport != nil {
		pSession.mtxData.Unlock()
		return false
	}
	pSession.waitUnlock = false
	pSession.dialing = true // Reconnect 와 같은 이유
	pSession.mtxData.Unlock()
	go pSession.connect()
	return true
}

// scheduleRetryLocked 는 mtxData 를 잡은 상태에서 다음 재접속을 예약한다 (session.md §4).
func (pSession *Session) scheduleRetryLocked() {
	iDelay := BackoffSec(pSession.attempts)
	pSession.attempts++
	pSession.info.M_szState = G_cszStateReconnecting
	pSession.info.M_szMessage = i18n.T("be.session.reconnectStatus", "sec", iDelay, "n", pSession.attempts, "max", G_ciMaxReconnect)
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

// userEndedLocked 는 mtxData 를 잡은 상태에서, 방금 끝난 셸을 사용자가 끝낸 것으로 볼지 판단한다 (session.md §4).
// 셸의 유휴 로그아웃과 사용자의 exit 는 종료 코드로는 구분되지 않는다(둘 다 종료 코드와 함께 끝난다).
// 유휴 로그아웃은 입력이 한참 없을 때만 일어나므로, 끝나기 직전에 입력이 있었으면 사용자가 끝낸 것이다.
// 연결하자마자 끝나는 서버(로그인 직후 셸이 끝나는 계정 등)를 계속 다시 접속하지 않도록 그 경우도 여기에 넣는다.
func (pSession *Session) userEndedLocked(tNow time.Time) bool {
	if tNow.Sub(pSession.tConnected) < G_cdurUserExitWindow {
		return true
	}
	return !pSession.tLastInput.IsZero() && tNow.Sub(pSession.tLastInput) < G_cdurUserExitWindow
}

// onRemoteExited 는 서버가 셸을 종료 코드·신호와 함께 끝낸 것을 처리한다 (session.md §4).
//   - 사용자가 끝낸 것(exit·logout, userEndedLocked)이면 자동 재접속하지 않고 닫힘 + 안내 두 줄.
//   - 서버 쪽에서 끝낸 것(유휴 로그아웃, 관리자의 세션 종료, 서버 재시작)이면 끊김처럼 자동 재접속을 예약한다.
//
// Enter·[재연결] 로 다시 연결하는 동작은 그대로다. 남은 연결(SSH 접속·keepalive·포워딩)은 닫는다.
func (pSession *Session) onRemoteExited(iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	conn := pSession.transport
	pSession.transport = nil
	pSession.writeOutputUnlockedNotice(MsgRemoteExited())
	bRetry := !pSession.userEndedLocked(time.Now()) &&
		CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
	if bRetry {
		pSession.scheduleRetryLocked()
	} else {
		pSession.attempts = 0
		pSession.info.M_szState = G_cszStateClosed
		pSession.info.M_szMessage = ""
		pSession.writeOutputUnlockedNotice(MsgGiveUp())
	}
	pSession.mtxData.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	pSession.publishState()
}

// readLoop 는 연결의 출력을 읽어 화면과 로그로 보낸다. 끝나면 끊김 처리한다.
func (pSession *Session) readLoop(conn Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	var zDetector zmodem.Detector       // 이 연결의 출력에서 ZMODEM 시작 표시를 찾는다 (zmodem.md)
	var chAhead chan readResult         // 끝난 ZMODEM 전송이 걸어 둔 읽기 하나 (zmodemLink)
	var zMangled zmodem.MangledDetector // tmux·screen 이 ZDLE 을 버린 시작 표시를 찾는다 (zmodem.md §3)
	var tmMangledNotice time.Time       // 그 안내를 마지막으로 낸 때
	for {
		// 프런트가 아직 처리하지 않은 출력이 상한을 넘으면 여기서 멈춘다 (session.md §8 출력 흐름 제어).
		// 읽기를 멈추면 TCP 윈도우·PTY 파이프가 차서 상대 프로세스가 스스로 느려진다.
		pSession.credit.Wait()
		var nRead int
		var errRead error
		if chAhead != nil {
			ahead := <-chAhead
			chAhead = nil
			nRead, errRead = copy(bBuf, ahead.bData), ahead.err
		} else {
			nRead, errRead = conn.Read(bBuf)
			pSession.traffic.addIn(nRead) // 닫힌 뒤 읽힌 조각도 연결에서 받은 것이다
		}
		if nRead > 0 && pSession.isCurrent(iGen) { // 닫힌 뒤 읽힌 조각은 버린다
			bScreen := bBuf[:nRead]
			bMangled := false
			if pSession.zmodemOn() {
				bBefore, kind, bEngine := zDetector.Feed(bScreen)
				if kind == zmodem.KindNone {
					bMangled = zMangled.Feed(bScreen)
				}
				if kind != zmodem.KindNone {
					// 서버에서 sz·rz 가 시작됐다: 표시 앞의 출력만 화면에 보내고, 전송이 끝날 때까지 연결을 엔진에 넘긴다.
					pSession.writeOutput(pSession.decoder.Decode(bBefore))
					bLeft, chNext, errStream := pSession.runZmodem(conn, kind, bEngine)
					bScreen, chAhead = bLeft, chNext
					if errStream != nil {
						errRead = errStream // 전송 중에 연결이 끊겼다: 아래에서 끊김으로 처리한다
					}
				}
			}
			if len(bScreen) > 0 {
				szScreen := pSession.decoder.Decode(bScreen)
				pSession.writeOutput(szScreen)
				pSession.feedLogin(conn, szScreen, iGen)
			}
			if bMangled && time.Since(tmMangledNotice) >= G_durZmodemMangledNotice {
				// 시작 표시가 깨져서 왔다(tmux·screen 안): 전송은 될 수 없으니 까닭과 끝내는 방법을 알린다.
				// 서버로는 아무것도 보내지 않는다 — 이 글이 실행 중인 sz·rz 의 것이 아닐 수도 있다(파일 내용 등).
				tmMangledNotice = time.Now()
				pSession.writeOutput(noticeLine(i18n.T("be.zm.muxed")))
			}
		}
		if errRead != nil {
			if errors.Is(errRead, transport.ErrRemoteExited) {
				pSession.onRemoteExited(iGen)
				return
			}
			szReason := ""
			if !errors.Is(errRead, io.EOF) {
				szReason = errRead.Error()
			}
			pSession.onDisconnected(szReason, iGen)
			return
		}
	}
}

// isCurrent 는 이 세대가 아직 유효하고 사용자가 닫지 않았는지 본다.
func (pSession *Session) isCurrent(iGen int) bool {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return iGen == pSession.gen && !pSession.userClosed
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
		_ = pSession.send(pSession.transport, Encode(pSession.cfg.Encoding, pSession.cfg.StartupInput))
	})
}
