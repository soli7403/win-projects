package session

import (
	"sync"
	"time"
)

// 출력 흐름 제어 (spec/behavior/session.md §8 "출력 흐름 제어").
// 프런트가 아직 처리(ack)하지 않은 출력 바이트가 상한을 넘으면 읽기를 멈추고, 하한 아래로 내려오면 다시 읽는다.
const (
	// G_ciCreditHighWater 는 읽기를 멈추는 미처리 바이트 상한이다.
	G_ciCreditHighWater = 512 * 1024
	// G_ciCreditLowWater 는 읽기를 다시 여는 미처리 바이트 하한이다.
	G_ciCreditLowWater = 128 * 1024
	// G_cdurCreditTimeout 은 멈춘 동안 ack 가 전혀 오지 않을 때 읽기를 다시 여는 안전 시한이다 (프런트 결함 대비).
	G_cdurCreditTimeout = 3 * time.Second
)

// Credit 은 세션 하나의 미처리 출력 양을 센다. 보낸 묶음의 크기를 순서대로 기억하고, 프런트의 ack(개수)로 앞에서부터 뺀다.
type Credit struct {
	mtxData   sync.Mutex
	cond      *sync.Cond
	slPending []int // 보낸 순서대로 각 묶음의 바이트 수
	iInFlight int   // slPending 의 합
	iAckSeq   int   // ack 가 올 때마다 늘어난다 (안전 시한의 "그동안 ack 가 왔는가" 판정)
	bClosed   bool
}

// NewCredit 은 빈 크레딧을 만든다.
func NewCredit() *Credit {
	pCredit := &Credit{}
	pCredit.cond = sync.NewCond(&pCredit.mtxData)
	return pCredit
}

// Sent 는 묶음 하나를 보냈음을 기록한다. emit 보다 먼저 불러야 한다 (ack 가 먼저 오면 빈 큐에서 무시되므로).
func (pCredit *Credit) Sent(iBytes int) {
	pCredit.mtxData.Lock()
	pCredit.slPending = append(pCredit.slPending, iBytes)
	pCredit.iInFlight += iBytes
	pCredit.mtxData.Unlock()
}

// Ack 는 프런트가 앞에서부터 iCount 개 묶음을 처리했음을 반영한다. 큐보다 많은 수는 무시한다.
func (pCredit *Credit) Ack(iCount int) {
	pCredit.mtxData.Lock()
	for iCount > 0 && len(pCredit.slPending) > 0 {
		pCredit.iInFlight -= pCredit.slPending[0]
		pCredit.slPending = pCredit.slPending[1:]
		iCount--
	}
	if len(pCredit.slPending) == 0 {
		pCredit.slPending = nil // 밑바닥 배열을 놓아 준다
		pCredit.iInFlight = 0
	}
	pCredit.iAckSeq++
	pCredit.mtxData.Unlock()
	pCredit.cond.Broadcast()
}

// Wait 은 미처리 양이 상한 이상이면 하한 아래로 내려올 때까지 막는다.
// 닫혔으면 바로 돌아온다. 안전 시한 동안 ack 가 하나도 오지 않으면 미처리 양을 비우고 돌아온다.
func (pCredit *Credit) Wait() {
	pCredit.mtxData.Lock()
	defer pCredit.mtxData.Unlock()
	if pCredit.iInFlight < G_ciCreditHighWater || pCredit.bClosed {
		return
	}
	for pCredit.iInFlight > G_ciCreditLowWater && !pCredit.bClosed {
		iSeqBefore := pCredit.iAckSeq
		bTimedOut := false
		timer := time.AfterFunc(G_cdurCreditTimeout, func() {
			pCredit.mtxData.Lock()
			bTimedOut = true
			pCredit.mtxData.Unlock()
			pCredit.cond.Broadcast()
		})
		// ack·닫힘·시한 중 하나가 올 때까지 기다린다.
		for !bTimedOut && pCredit.iAckSeq == iSeqBefore && !pCredit.bClosed {
			pCredit.cond.Wait()
		}
		timer.Stop()
		if bTimedOut && pCredit.iAckSeq == iSeqBefore {
			// 시한 동안 ack 가 없었다: 프런트가 멈췄거나 결함. 굳은 채로 두지 않고 읽기를 다시 연다.
			pCredit.slPending = nil
			pCredit.iInFlight = 0
			return
		}
	}
}

// Close 는 기다리는 읽기를 깨우고 이후의 Wait 을 즉시 돌려보낸다.
func (pCredit *Credit) Close() {
	pCredit.mtxData.Lock()
	pCredit.bClosed = true
	pCredit.mtxData.Unlock()
	pCredit.cond.Broadcast()
}

// InFlight 는 미처리 바이트 수다 (테스트·진단용).
func (pCredit *Credit) InFlight() int {
	pCredit.mtxData.Lock()
	defer pCredit.mtxData.Unlock()
	return pCredit.iInFlight
}
