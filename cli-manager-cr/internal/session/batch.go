package session

import (
	"strings"
	"sync"
	"time"
)

// G_cdurBatchWindow 은 출력 묶음 전송 간격이다 (spec/ipc.md §4, session.md §8).
const G_cdurBatchWindow = 12 * time.Millisecond

// G_ciBatchMaxBytes 는 묶음 하나의 상한이다. 넘으면 12ms 를 기다리지 않고 바로 보낸다 (session.md §8).
const G_ciBatchMaxBytes = 256 * 1024

// Batcher 는 출력 조각을 모아 12ms 뒤에 한 번에 보낸다.
type Batcher struct {
	mtxData   sync.Mutex
	mtxSend   sync.Mutex // fnFlush 호출을 한 줄로 세운다 (보낸 순서 = 쌓인 순서)
	sbPending strings.Builder
	timer     *time.Timer
	bClosed   bool // Close 뒤에는 더 보내지 않는다 (닫은 세션의 늦은 출력, session.md §8)
	fnFlush   func(string)
}

// NewBatcher 는 묶은 텍스트를 fnFlush 로 보내는 묶음기를 만든다.
func NewBatcher(fnFlush func(string)) *Batcher {
	return &Batcher{fnFlush: fnFlush}
}

// Add 는 조각을 더한다. 첫 조각이면 12ms 타이머를 걸고, 상한을 넘으면 즉시 보낸다.
func (pBatcher *Batcher) Add(szChunk string) {
	if szChunk == "" {
		return
	}
	pBatcher.mtxData.Lock()
	if pBatcher.bClosed {
		pBatcher.mtxData.Unlock()
		return
	}
	pBatcher.sbPending.WriteString(szChunk)
	if pBatcher.sbPending.Len() >= G_ciBatchMaxBytes {
		pBatcher.mtxData.Unlock()
		pBatcher.Flush()
		return
	}
	if pBatcher.timer == nil {
		pBatcher.timer = time.AfterFunc(G_cdurBatchWindow, pBatcher.Flush)
	}
	pBatcher.mtxData.Unlock()
}

// Flush 는 쌓인 내용을 즉시 보낸다.
func (pBatcher *Batcher) Flush() {
	// 꺼내기와 보내기를 mtxSend 로 묶어, 타이머 Flush 와 상한 Flush 가 겹쳐도 순서가 뒤바뀌지 않게 한다.
	pBatcher.mtxSend.Lock()
	defer pBatcher.mtxSend.Unlock()
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

// Close 는 쌓인 내용을 보내고 이후의 Add 를 버린다.
func (pBatcher *Batcher) Close() {
	pBatcher.Flush()
	pBatcher.mtxData.Lock()
	pBatcher.bClosed = true
	pBatcher.mtxData.Unlock()
	pBatcher.Flush() // Flush 와 닫힘 표시 사이에 들어온 조각
}
