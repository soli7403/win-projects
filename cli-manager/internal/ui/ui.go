// Package ui 는 렌더러로의 이벤트 전송과 "질문하고 답 기다리기"(ui:request / ui:respond)를 담당한다.
package ui

import (
	"encoding/json"
	"sync"
	"time"

	"termgo/internal/vault"
)

// Bridge 는 Wails 런타임에 의존하지 않도록 emit 함수를 주입받는다.
type Bridge struct {
	emit    func(szCh string, args ...any)
	mu      sync.Mutex
	pending map[string]chan json.RawMessage
}

// NewBridge 는 Bridge 를 만든다.
func NewBridge(emit func(szCh string, args ...any)) *Bridge {
	return &Bridge{emit: emit, pending: map[string]chan json.RawMessage{}}
}

// Send 는 렌더러 이벤트를 보낸다.
func (pstrB *Bridge) Send(szCh string, args ...any) { pstrB.emit(szCh, args...) }

// Ask 는 ui:request 를 보내고 응답(JSON)을 기다린다. 취소(null)·시간 초과(10분)면 ok=false.
func (pstrB *Bridge) Ask(mReq map[string]any) (json.RawMessage, bool) {
	szID := vault.NewUUID()
	ch := make(chan json.RawMessage, 1)
	pstrB.mu.Lock()
	pstrB.pending[szID] = ch
	pstrB.mu.Unlock()
	mReq["reqId"] = szID
	pstrB.emit("ui:request", mReq)
	defer func() {
		pstrB.mu.Lock()
		delete(pstrB.pending, szID)
		pstrB.mu.Unlock()
	}()
	select {
	case raw := <-ch:
		if string(raw) == "null" || len(raw) == 0 {
			return nil, false
		}
		return raw, true
	case <-time.After(10 * time.Minute):
		return nil, false
	}
}

// Respond 는 ui:respond 를 받아 대기 중인 Ask 를 깨운다.
func (pstrB *Bridge) Respond(szReqID, szValueJSON string) {
	pstrB.mu.Lock()
	ch := pstrB.pending[szReqID]
	pstrB.mu.Unlock()
	if ch != nil {
		select {
		case ch <- json.RawMessage(szValueJSON):
		default:
		}
	}
}

// AskBool 은 true/false 응답용.
func (pstrB *Bridge) AskBool(mReq map[string]any) bool {
	raw, bOk := pstrB.Ask(mReq)
	if !bOk {
		return false
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b
}

// AskString 은 문자열 응답용 (취소면 ok=false).
func (pstrB *Bridge) AskString(mReq map[string]any) (string, bool) {
	raw, bOk := pstrB.Ask(mReq)
	if !bOk {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}
