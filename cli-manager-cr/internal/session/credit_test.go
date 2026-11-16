package session

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errTestDial = errors.New("dial failed")

// floodTransport 는 Read 때마다 버퍼를 'x' 로 채워 돌려주는 가짜 연결이다. Close 뒤에는 오류를 돌려준다.
type floodTransport struct {
	bClosed atomic.Bool
	iReads  atomic.Int64
}

func (pFlood *floodTransport) Read(bBuf []byte) (int, error) {
	if pFlood.bClosed.Load() {
		return 0, errors.New("closed")
	}
	pFlood.iReads.Add(1)
	for iIndex := range bBuf {
		bBuf[iIndex] = 'x'
	}
	return len(bBuf), nil
}
func (pFlood *floodTransport) Write(bData []byte) error          { return nil }
func (pFlood *floodTransport) Resize(iCols int, iRows int) error { return nil }
func (pFlood *floodTransport) Close() error                      { pFlood.bClosed.Store(true); return nil }

// 폭주 출력: ack 가 없으면 보낸 양이 상한 + 묶음 상한 + 읽기 버퍼를 넘지 않고, ack 를 주면 다시 읽는다.
func TestCreditStopsReadingWithoutAck(t *testing.T) {
	var mtxSent sync.Mutex
	iSent := 0
	iBatches := 0
	flood := &floodTransport{}
	cfg := sshConfig(func(func(string)) (Transport, error) { return flood, nil }, false)
	pSession := newSession(cfg, func(szEvent string, anyData ...any) {
		if szEvent != "session.data" {
			return
		}
		mtxSent.Lock()
		iSent += len(anyData[1].(string))
		iBatches++
		mtxSent.Unlock()
	})
	pSession.start()
	defer pSession.Close()

	waitUntil(t, func() bool { mtxSent.Lock(); defer mtxSent.Unlock(); return iSent >= G_ciCreditHighWater }, 2*time.Second)
	time.Sleep(100 * time.Millisecond) // 더 보내는지 지켜본다
	mtxSent.Lock()
	iAfter := iSent
	iAckCount := iBatches
	mtxSent.Unlock()
	iLimit := G_ciCreditHighWater + G_ciBatchMaxBytes + 32*1024
	if iAfter > iLimit {
		t.Fatalf("ack 없이 보낸 양이 상한을 넘음: %d > %d", iAfter, iLimit)
	}

	// 전부 ack 하면 다시 읽는다.
	pSession.AckOutput(iAckCount)
	waitUntil(t, func() bool { mtxSent.Lock(); defer mtxSent.Unlock(); return iSent > iAfter }, 2*time.Second)
}

// 크레딧에 막혀 있는 세션을 닫으면 readLoop 가 깨어나 끝난다 (교착 없음).
func TestCreditWakesOnClose(t *testing.T) {
	flood := &floodTransport{}
	cfg := sshConfig(func(func(string)) (Transport, error) { return flood, nil }, false)
	pSession := newSession(cfg, func(string, ...any) {})
	pSession.start()
	waitUntil(t, func() bool { return pSession.credit.InFlight() >= G_ciCreditHighWater }, 2*time.Second)
	time.Sleep(30 * time.Millisecond) // 묶음 타이머가 남은 조각까지 보내고 readLoop 가 Wait 에 들어가도록
	iReadsBefore := flood.iReads.Load()
	time.Sleep(50 * time.Millisecond)
	if flood.iReads.Load() != iReadsBefore {
		t.Fatal("막혀 있어야 하는데 계속 읽음")
	}
	chDone := make(chan struct{})
	go func() { pSession.Close(); close(chDone) }()
	select {
	case <-chDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close 가 돌아오지 않음")
	}
	if pSession.Snapshot().M_szState != G_cszStateClosed {
		t.Fatalf("닫힌 상태여야 함: %s", pSession.Snapshot().M_szState)
	}
}

// 큐보다 많은 ack 는 무시하고, 미처리 바이트는 음수가 되지 않는다. 묶음 순서대로 뺀다.
func TestCreditAckOrderAndExcess(t *testing.T) {
	pCredit := NewCredit()
	pCredit.Sent(100)
	pCredit.Sent(30)
	pCredit.Ack(1)
	if pCredit.InFlight() != 30 {
		t.Fatalf("앞 묶음부터 빼야 함: %d", pCredit.InFlight())
	}
	pCredit.Ack(5)
	if pCredit.InFlight() != 0 {
		t.Fatalf("0 이어야 함: %d", pCredit.InFlight())
	}
	pCredit.Ack(1) // 빈 큐
	if pCredit.InFlight() != 0 {
		t.Fatal("음수/변화 없어야 함")
	}
}

// 안전 시한: ack 가 전혀 없어도 시한 뒤 Wait 이 돌아온다.
func TestCreditTimeoutReleases(t *testing.T) {
	pCredit := NewCredit()
	pCredit.Sent(G_ciCreditHighWater)
	tStart := time.Now()
	pCredit.Wait()
	if time.Since(tStart) < G_cdurCreditTimeout-100*time.Millisecond {
		t.Fatal("시한 전에 돌아옴")
	}
	if pCredit.InFlight() != 0 {
		t.Fatal("시한 뒤 미처리 바이트는 0 이어야 함")
	}
}

// ack 로 하한 아래가 되면 Wait 이 바로 풀린다.
func TestCreditWaitReleasesOnAck(t *testing.T) {
	pCredit := NewCredit()
	for iIndex := 0; iIndex < 4; iIndex++ {
		pCredit.Sent(G_ciCreditHighWater / 4)
	}
	chDone := make(chan struct{})
	go func() { pCredit.Wait(); close(chDone) }()
	time.Sleep(20 * time.Millisecond)
	pCredit.Ack(2) // 아직 하한(1/4) 위
	select {
	case <-chDone:
		t.Fatal("하한 위에서는 계속 기다려야 함")
	case <-time.After(30 * time.Millisecond):
	}
	pCredit.Ack(2)
	select {
	case <-chDone:
	case <-time.After(time.Second):
		t.Fatal("하한 아래가 되면 풀려야 함")
	}
}

// 묶음 상한: 256KB 를 넘으면 12ms 를 기다리지 않고 바로 보낸다.
func TestBatcherFlushesAtMaxBytes(t *testing.T) {
	var mtxOut sync.Mutex
	var slOut []int
	pBatcher := NewBatcher(func(szText string) { mtxOut.Lock(); slOut = append(slOut, len(szText)); mtxOut.Unlock() })
	pBatcher.Add(strings.Repeat("y", G_ciBatchMaxBytes)) // 타이머 없이 즉시
	mtxOut.Lock()
	if len(slOut) != 1 || slOut[0] != G_ciBatchMaxBytes {
		mtxOut.Unlock()
		t.Fatalf("즉시 1회 전송이어야 함: %v", slOut)
	}
	mtxOut.Unlock()
	pBatcher.Add("tail") // 이건 12ms 뒤
	waitUntil(t, func() bool { mtxOut.Lock(); defer mtxOut.Unlock(); return len(slOut) == 2 }, time.Second)
	pBatcher.Close()
	pBatcher.Add("after close")
	time.Sleep(30 * time.Millisecond)
	mtxOut.Lock()
	defer mtxOut.Unlock()
	if len(slOut) != 2 {
		t.Fatalf("닫은 뒤에는 보내지 않아야 함: %v", slOut)
	}
}
