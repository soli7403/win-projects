// Package ipc 는 프런트↔백엔드 계약(spec/ipc.md)의 요청 라우팅, 이벤트 발행, 사용자 질문 응답을 담당한다.
package ipc

import (
	"dariterm/internal/i18n"
	"encoding/json"
	"sync"
)

// Handler 는 채널 하나의 처리 함수다. slArgs 는 JSON 배열의 각 원소이며, 반환값은 JSON 으로 직렬화된다.
type Handler func(slArgs []json.RawMessage) (any, error)

// Emitter 는 백엔드→프런트 이벤트 발행기다 (Wails 이벤트 등).
type Emitter interface {
	Emit(szEvent string, anyData ...any)
}

// Router 는 채널 이름 → 처리 함수 표를 가진다.
type Router struct {
	mtxData     sync.RWMutex
	mapHandlers map[string]Handler
	emitter     Emitter
	asker       *Asker
}

// NewRouter 는 빈 라우터를 만든다. emitter 가 nil 이면 이벤트는 버린다.
func NewRouter(emitter Emitter) *Router {
	pRouter := &Router{mapHandlers: map[string]Handler{}, emitter: emitter}
	pRouter.asker = newAsker(pRouter)
	return pRouter
}

// Handle 은 채널에 처리 함수를 등록한다. 같은 이름이면 덮어쓴다.
func (pRouter *Router) Handle(szChannel string, fnHandler Handler) {
	pRouter.mtxData.Lock()
	defer pRouter.mtxData.Unlock()
	pRouter.mapHandlers[szChannel] = fnHandler
}

// Emit 은 이벤트를 프런트로 보낸다.
func (pRouter *Router) Emit(szEvent string, anyData ...any) {
	if pRouter.emitter != nil {
		pRouter.emitter.Emit(szEvent, anyData...)
	}
}

// Asker 는 ui.request 질문 응답 대기기를 돌려준다.
func (pRouter *Router) Asker() *Asker {
	return pRouter.asker
}

// Invoke 는 요청-응답 채널 호출이다 (spec/ipc.md §1).
// 결과가 없으면 빈 문자열을 돌려준다. 오류는 메시지 문자열이다.
func (pRouter *Router) Invoke(szChannel string, szArgsJson string) (szResult string, errResult error) {
	pRouter.mtxData.RLock()
	fnHandler, bFound := pRouter.mapHandlers[szChannel]
	pRouter.mtxData.RUnlock()
	if !bFound {
		return "", i18n.Errorf("be.ipc.noChannel", nil, "name", szChannel)
	}
	var slArgs []json.RawMessage
	if szArgsJson != "" {
		if errArgs := json.Unmarshal([]byte(szArgsJson), &slArgs); errArgs != nil {
			return "", i18n.Errorf("be.ipc.badArgs", nil, "name", szChannel, "error", errArgs)
		}
	}
	defer func() {
		if anyPanic := recover(); anyPanic != nil {
			szResult = ""
			errResult = i18n.Errorf("be.ipc.internal", nil, "name", szChannel, "error", anyPanic)
		}
	}()
	anyValue, errHandle := fnHandler(slArgs)
	if errHandle != nil {
		return "", errHandle
	}
	if anyValue == nil {
		return "", nil
	}
	bOut, errMarshal := json.Marshal(anyValue)
	if errMarshal != nil {
		return "", i18n.Errorf("be.ipc.marshal", nil, "name", szChannel, "error", errMarshal)
	}
	return string(bOut), nil
}
