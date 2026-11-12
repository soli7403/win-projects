// Package transport 는 원격 셸/장치로 가는 바이트 파이프(SSH·텔넷·시리얼·로컬 PTY)를 정의한다.
package transport

import "sync"

// Transport 는 세션이 쓰는 공통 인터페이스.
// 수신 바이트는 Start 시 넘긴 콜백(onData)으로, 종료는 onClose(reason)으로 한 번만 전달된다.
type Transport interface {
	Write(arr []byte) error
	Resize(iCols, iRows int)
	Close()
}

// Callbacks 는 트랜스포트 → 세션 방향 알림.
type Callbacks struct {
	OnData  func(arr []byte)
	OnClose func(szReason string) // 사용자가 직접 닫은 경우 reason 은 빈 문자열
}

// OnceClose 는 OnClose 를 한 번만 부르게 감싼다.
type OnceClose struct {
	once sync.Once
	fn   func(string)
}

// NewOnceClose 는 fn 을 한 번만 실행하는 래퍼를 만든다.
func NewOnceClose(fn func(string)) *OnceClose { return &OnceClose{fn: fn} }

// Fire 는 처음 한 번만 fn(szReason) 을 호출한다.
func (pstrO *OnceClose) Fire(szReason string) {
	pstrO.once.Do(func() {
		if pstrO.fn != nil {
			pstrO.fn(szReason)
		}
	})
}
