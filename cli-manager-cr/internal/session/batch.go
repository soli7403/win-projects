package session

import (
	"strings"
	"sync"
	"time"
)

// G_cdurBatchWindow 은 출력 묶음 전송 간격이다 (spec/ipc.md §4, session.md §8).
const G_cdurBatchWindow = 12 * time.Millisecond

// Batcher 는 출력 조각을 모아 12ms 뒤에 한 번에 보낸다.
type Batcher struct {
	mtxData   sync.Mutex
	sbPending strings.Builder
	timer     *time.Timer
	fnFlush   func(string)
}

// NewBatcher 는 묶은 텍스트를 fnFlush 로 보내는 묶음기를 만든다.
func NewBatcher(fnFlush func(string)) *Batcher {
	return &Batcher{fnFlush: fnFlush}
}

// Add 는 조각을 더한다. 첫 조각이면 12ms 타이머를 건다.
func (pBatcher *Batcher) Add(szChunk string) {
	if szChunk == "" {
		return
	}
	pBatcher.mtxData.Lock()
	defer pBatcher.mtxData.Unlock()
	pBatcher.sbPending.WriteString(szChunk)
	if pBatcher.timer == nil {
		pBatcher.timer = time.AfterFunc(G_cdurBatchWindow, pBatcher.Flush)
	}
}

// Flush 는 쌓인 내용을 즉시 보낸다.
func (pBatcher *Batcher) Flush() {
	pBatcher.mtxData.Lock()
	szOut := pBatcher.sbPending.String()
	pBatcher.sbPending.Reset()
	if pBatcher.timer != nil {
		pBatcher.timer.Stop()
		pBatcher.timer = nil
	}
	pBatcher.mtxData.Unlock()
	if szOut != "" {
		pBatcher.fnFlush(szOut)
	}
}
