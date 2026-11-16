package session

import (
	"strings"
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
	pSession.mtxData.Unlock()
	if transport != nil {
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
	if pSession.userClosed || pSession.transport != nil || pSession.info.M_szState == G_cszStateConnecting {
		pSession.mtxData.Unlock()
		return
	}
	if pSession.timerRetry != nil {
		pSession.timerRetry.Stop()
		pSession.timerRetry = nil
	}
	pSession.attempts = 0
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
	pSession.publishState()
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
