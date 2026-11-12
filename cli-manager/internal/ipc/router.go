// Package ipc 는 v1.1.0 의 ipcMain.handle 에 해당하는 채널 라우터다.
// 렌더러의 window.bt.call(ch, ...args) 가 App.Call 을 거쳐 여기로 들어온다.
package ipc

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Handler 는 JSON 인자 목록을 받아 JSON 직렬화 가능한 값을 돌려준다.
type Handler func(args []json.RawMessage) (any, error)

// Router 는 채널 이름 → Handler 표.
type Router struct {
	mu       sync.RWMutex
	mHandler map[string]Handler
}

// NewRouter 는 빈 Router 를 만든다.
func NewRouter() *Router {
	return &Router{mHandler: map[string]Handler{}}
}

// Handle 은 채널 핸들러를 등록한다.
func (pstrRouter *Router) Handle(szCh string, fn Handler) {
	pstrRouter.mu.Lock()
	defer pstrRouter.mu.Unlock()
	pstrRouter.mHandler[szCh] = fn
}

// Call 은 szArgsJSON(JSON 배열)을 풀어 핸들러를 실행하고 결과를 JSON 문자열로 돌려준다.
// 반환값이 nil 이면 빈 문자열(렌더러에서는 undefined).
func (pstrRouter *Router) Call(szCh, szArgsJSON string) (szOut string, err error) {
	defer func() {
		if r := recover(); r != nil { // 핸들러 패닉이 앱 전체를 멈추지 않게 오류로 바꾼다 (brief 3.7)
			szOut, err = "", fmt.Errorf("내부 오류(%s): %v", szCh, r)
		}
	}()
	pstrRouter.mu.RLock()
	fn, bOk := pstrRouter.mHandler[szCh]
	pstrRouter.mu.RUnlock()
	if !bOk {
		return "", fmt.Errorf("구현되지 않은 채널: %s", szCh)
	}
	var slArgs []json.RawMessage
	if szArgsJSON != "" {
		if uerr := json.Unmarshal([]byte(szArgsJSON), &slArgs); uerr != nil {
			return "", fmt.Errorf("인자 해석 실패(%s): %w", szCh, uerr)
		}
	}
	val, ferr := fn(slArgs)
	if ferr != nil {
		return "", ferr
	}
	if val == nil {
		return "", nil
	}
	arrOut, merr := json.Marshal(val)
	if merr != nil {
		return "", fmt.Errorf("결과 직렬화 실패(%s): %w", szCh, merr)
	}
	return string(arrOut), nil
}
