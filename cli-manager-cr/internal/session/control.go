package session

import (
	"os"
	"strings"
	"time"
)

// Snapshot 은 현재 세션 정보의 복사본을 돌려준다.
func (pSession *Session) Snapshot() Info {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return pSession.info
}

// Input 은 터미널 입력(UTF-8)을 서버 인코딩으로 바꿔 보낸다 (session.md §8).
// 연결이 없으면 버린다. 다만 끊긴 세션에서 CR(Enter)이 오면 즉시 재접속한다 (session.md §4 수동 재접속).
func (pSession *Session) Input(szText string) {
	pSession.mtxData.Lock()
	transport := pSession.transport
	szState := pSession.info.M_szState
	if transport != nil && szText != "" {
		pSession.tLastInput = time.Now() // 서버가 셸을 끝냈을 때 사용자가 끝낸 것인지 가리는 데 쓴다 (session.md §4)
	}
	pSession.mtxData.Unlock()
	if transport != nil {
		if pSession.zActive.Load() {
			// ZMODEM 전송 중: 입력이 전송 데이터에 섞이면 안 되므로 보내지 않는다. Ctrl+C 는 전송 취소다 (zmodem.md).
			if strings.Contains(szText, "\x03") {
				pSession.cancelZmodem(transport)
			}
			return
		}
		_ = transport.Write(Encode(pSession.cfg.Encoding, szText))
		return
	}
	if strings.Contains(szText, "\r") && szState != G_cszStateConnecting {
		pSession.Reconnect()
	}
}

// Resize 는 터미널 크기를 기억하고, 연결이 있으면 프로토콜에 알린다.
func (pSession *Session) Resize(iCols int, iRows int) {
	if iCols < 2 || iRows < 2 {
		return
	}
	pSession.mtxData.Lock()
	pSession.iCols, pSession.iRows = iCols, iRows
	transport := pSession.transport
	pSession.mtxData.Unlock()
	if transport != nil {
		_ = transport.Resize(iCols, iRows)
	}
}

// Reconnect 은 사용자가 요청한 즉시 재접속이다. 예약된 자동 재접속은 취소한다.
// 이미 연결되어 있으면 아무 일도 하지 않는다.
func (pSession *Session) Reconnect() {
	pSession.mtxData.Lock()
	if pSession.userClosed || pSession.transport != nil || pSession.dialing || pSession.info.M_szState == G_cszStateConnecting {
		pSession.mtxData.Unlock()
		return
	}
	if pSession.timerRetry != nil {
		pSession.timerRetry.Stop()
		pSession.timerRetry = nil
	}
	pSession.attempts = 0
	pSession.dialing = true // go 로 띄우기 전에 켠다: 그사이 온 Reconnect·잠금 해제 재개가 겹치지 않게
	pSession.mtxData.Unlock()
	go pSession.connect()
}

// Close 는 세션을 닫는다: 예약 취소, 연결 종료, 로그 닫기. 사용자가 닫은 세션은 끊김 안내가 없다.
func (pSession *Session) Close() {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	pSession.userClosed = true
	pSession.gen++
	if pSession.timerRetry != nil {
		pSession.timerRetry.Stop()
		pSession.timerRetry = nil
	}
	if pSession.timerStartup != nil {
		pSession.timerStartup.Stop()
		pSession.timerStartup = nil
	}
	transport := pSession.transport
	pSession.transport = nil
	pSession.info.M_szState = G_cszStateClosed
	pSession.info.M_szMessage = ""
	pLogger := pSession.logger
	pSession.logger = nil
	pSession.mtxData.Unlock()

	pSession.credit.Close() // 크레딧을 기다리던 readLoop 를 깨운다 (transport 가 nil 이어도 안전)
	if transport != nil {
		_ = transport.Close()
	}
	pSession.batcher.Close()
	if pLogger != nil {
		_ = pLogger.Close()
	}
	if pSession.cfg.OnClose != nil {
		pSession.cfg.OnClose()
	}
	pSession.publishState()
}

// SetLogging 은 이 세션의 출력 기록을 켜거나 끈다 (session.md §6).
// 켜면 그 순간부터의 출력을 새 로그 파일에 쓴다(이미 기록 중이면 그대로). 끄면 남은 것을 쓰고 파일을 닫는다.
// 세션을 열 때(start)와 설정 "터미널 출력 기록"을 바꿀 때 부른다. 닫힌 세션은 건드리지 않는다.
func (pSession *Session) SetLogging(bEnabled bool) {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	if !bEnabled {
		pLogger := pSession.logger
		pSession.logger = nil
		pSession.mtxData.Unlock()
		if pLogger != nil {
			_ = pLogger.Close()
		}
		return
	}
	bSkip := pSession.logger != nil || pSession.cfg.DataDir == ""
	pSession.mtxData.Unlock()
	if bSkip {
		return
	}
	// 파일 만들기는 잠금 밖에서 한다(디스크를 기다리는 동안 출력이 막히지 않게).
	pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
	if errLog != nil {
		return // 열기 실패는 조용히 넘어간다: 기록 없이 세션은 계속 쓴다
	}
	pSession.mtxData.Lock()
	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
	if !bLost {
		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
		pLogger.SetTimestamp(pSession.bLogStamp)
		pLogger.SetAnsi(pSession.bLogAnsi)
		pSession.logger = pLogger
	}
	pSession.mtxData.Unlock()
	if bLost {
		_ = pLogger.Close()
		_ = os.Remove(pLogger.Path()) // 방금 만든 빈 파일은 남기지 않는다
	}
}

// SetLogSyncSec 은 이 세션 로그의 내려 쓰기 주기(초, 0~5)를 바꾼다. 기록 중이면 곧바로 적용하고,
// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogSyncSec, session.md §6).
func (pSession *Session) SetLogSyncSec(iSec int) {
	iSec = ClampLogSyncSec(iSec)
	pSession.mtxData.Lock()
	pSession.iLogSyncSec = iSec
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.SetSyncSec(iSec)
	}
}

// SetLogTimestamp 는 이 세션 로그의 줄 앞 시각 표시를 켜거나 끈다. 기록 중이면 다음 줄부터 적용하고,
// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogTimestamp, session.md §6, D112).
func (pSession *Session) SetLogTimestamp(bEnabled bool) {
	pSession.mtxData.Lock()
	pSession.bLogStamp = bEnabled
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.SetTimestamp(bEnabled)
	}
}

// SetLogAnsi 는 이 세션 로그에 ANSI 시퀀스와 CR 을 그대로 쓸지 바꾼다. 기록 중이면 다음 조각부터 적용하고,
// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogAnsi, session.md §6, D120).
func (pSession *Session) SetLogAnsi(bEnabled bool) {
	pSession.mtxData.Lock()
	pSession.bLogAnsi = bEnabled
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.SetAnsi(bEnabled)
	}
}

// AckOutput 은 프런트가 출력 묶음 iCount 개를 처리했음을 반영한다 (spec/ipc.md session.ack).
func (pSession *Session) AckOutput(iCount int) {
	if iCount <= 0 {
		return
	}
	pSession.credit.Ack(iCount)
}

// Transport 는 현재 연결된 전송을 돌려준다 (연결이 없으면 nil). SFTP 처럼 프로토콜 기능을 찾을 때 쓴다.
func (pSession *Session) Transport() Transport {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return pSession.transport
}
