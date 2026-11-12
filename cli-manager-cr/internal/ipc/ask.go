package ipc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// G_cdurAskTimeout 은 사용자 답을 기다리는 시간이다. 넘으면 취소로 본다 (spec/ipc.md §3).
const G_cdurAskTimeout = 10 * time.Minute

// Asker 는 ui.request 를 보내고 Reply 로 온 답을 기다린다.
type Asker struct {
	mtxData    sync.Mutex
	pRouter    *Router
	durTimeout time.Duration
	mapPending map[string]chan json.RawMessage
}

func newAsker(pRouter *Router) *Asker {
	return &Asker{pRouter: pRouter, durTimeout: G_cdurAskTimeout, mapPending: map[string]chan json.RawMessage{}}
}

// SetTimeout 은 대기 시간을 바꾼다 (시험용).
func (pAsker *Asker) SetTimeout(durTimeout time.Duration) {
	pAsker.mtxData.Lock()
	defer pAsker.mtxData.Unlock()
	pAsker.durTimeout = durTimeout
}

// Ask 는 kind 와 필드로 ui.request 이벤트를 보내고 답을 기다린다.
// 답이 null 이거나 시간이 지나면 bOk=false (취소)다.
func (pAsker *Asker) Ask(szKind string, mapFields map[string]any) (json.RawMessage, bool) {
	szId, errId := newRequestId()
	if errId != nil {
		return nil, false
	}
	chAnswer := make(chan json.RawMessage, 1)
	pAsker.mtxData.Lock()
	pAsker.mapPending[szId] = chAnswer
	durTimeout := pAsker.durTimeout
	pAsker.mtxData.Unlock()

	mapEvent := map[string]any{"requestId": szId, "kind": szKind}
	for szKey, anyValue := range mapFields {
		mapEvent[szKey] = anyValue
	}
	pAsker.pRouter.Emit("ui.request", mapEvent)

	timerTimeout := time.NewTimer(durTimeout)
	defer timerTimeout.Stop()
	select {
	case rawAnswer := <-chAnswer:
		pAsker.remove(szId)
		if len(rawAnswer) == 0 || string(rawAnswer) == "null" {
			return nil, false
		}
		return rawAnswer, true
	case <-timerTimeout.C:
		pAsker.remove(szId)
		return nil, false
	}
}

// Reply 는 질문 id 의 답을 전한다 (spec/ipc.md §1 Reply). 없는 id 는 무시한다.
func (pAsker *Asker) Reply(szRequestId string, rawValue json.RawMessage) {
	pAsker.mtxData.Lock()
	chAnswer, bFound := pAsker.mapPending[szRequestId]
	if bFound {
		delete(pAsker.mapPending, szRequestId)
	}
	pAsker.mtxData.Unlock()
	if bFound {
		chAnswer <- rawValue
	}
}

// Reply 메서드를 라우터에서도 부를 수 있게 한다.
func (pRouter *Router) Reply(szRequestId string, szValueJson string) {
	pRouter.asker.Reply(szRequestId, json.RawMessage(szValueJson))
}

func (pAsker *Asker) remove(szId string) {
	pAsker.mtxData.Lock()
	defer pAsker.mtxData.Unlock()
	delete(pAsker.mapPending, szId)
}

func newRequestId() (string, error) {
	bRand := make([]byte, 12)
	if _, errRand := rand.Read(bRand); errRand != nil {
		return "", errRand
	}
	return hex.EncodeToString(bRand), nil
}
