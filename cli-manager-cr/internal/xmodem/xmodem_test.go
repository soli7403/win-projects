package xmodem

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// pipePort 는 한쪽 방향 바이트 통로 두 개로 만든 시험용 연결이다.
type pipePort struct {
	in      chan byte
	out     chan byte
	mangle  func(bData []byte) []byte // 보내기 전에 바꿀 수 있다 (nil 이면 그대로)
	mtxData sync.Mutex
}

func (p *pipePort) ReadByteWithin(durWait time.Duration) (byte, error) {
	timer := time.NewTimer(durWait)
	defer timer.Stop()
	select {
	case b, bOk := <-p.in:
		if !bOk {
			return 0, io.EOF
		}
		return b, nil
	case <-timer.C:
		return 0, ErrTimeout
	}
}

func (p *pipePort) Write(bData []byte) error {
	p.mtxData.Lock()
	fn := p.mangle
	p.mtxData.Unlock()
	if fn != nil {
		bData = fn(append([]byte(nil), bData...))
	}
	for _, b := range bData {
		p.out <- b
	}
	return nil
}

// newPair 는 서로 이어진 두 연결(보내는 쪽, 받는 쪽)을 만든다.
func newPair() (*pipePort, *pipePort) {
	chA := make(chan byte, 1<<16)
	chB := make(chan byte, 1<<16)
	return &pipePort{in: chA, out: chB}, &pipePort{in: chB, out: chA}
}

// fastTimes 는 시험이 빨리 끝나도록 시한을 줄인다.
func fastTimes(t *testing.T) {
	t.Helper()
	durStart, durBlock, durEach, durByte, durPoll := G_durStart, G_durBlock, G_durStartEach, G_durByte, G_durPoll
	G_durStart, G_durBlock, G_durStartEach, G_durByte, G_durPoll = 2*time.Second, 300*time.Millisecond, 200*time.Millisecond, 100*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() {
		G_durStart, G_durBlock, G_durStartEach, G_durByte, G_durPoll = durStart, durBlock, durEach, durByte, durPoll
	})
}

func randomData(iSize int, iSeed int64) []byte {
	bData := make([]byte, iSize)
	rand.New(rand.NewSource(iSeed)).Read(bData)
	if iSize > 0 && bData[iSize-1] == SUB {
		bData[iSize-1] = 0 // 끝의 SUB 는 채움과 구별할 수 없다 (XMODEM 의 한계)
	}
	return bData
}

type result struct {
	n   int64
	err error
}

// transfer 는 보내기·받기를 함께 돌려 받은 내용과 두 쪽의 결과를 돌려준다.
func transfer(t *testing.T, sender *pipePort, receiver *pipePort, bData []byte, opts Options) ([]byte, result, result) {
	t.Helper()
	var bufGot bytes.Buffer
	chSend := make(chan result, 1)
	go func() {
		n, err := Send(sender, bytes.NewReader(bData), int64(len(bData)), opts)
		chSend <- result{n, err}
	}()
	n, err := Receive(receiver, &bufGot, Options{})
	return bufGot.Bytes(), <-chSend, result{n, err}
}

func TestCRC16KnownValue(t *testing.T) {
	if got := CRC16([]byte("123456789")); got != 0x31C3 {
		t.Fatalf("CRC16 = %04X, 0x31C3 이어야 한다", got)
	}
	if got := Checksum([]byte{0xFF, 0x02}); got != 0x01 {
		t.Fatalf("Checksum = %02X", got)
	}
}

func TestRoundTripSizesCrcAnd1K(t *testing.T) {
	fastTimes(t)
	for _, b1K := range []bool{false, true} {
		for _, iSize := range []int{0, 1, 127, 128, 129, 1000, 1024, 1025, 3000} {
			sender, receiver := newPair()
			bData := randomData(iSize, int64(iSize))
			bGot, rs, rr := transfer(t, sender, receiver, bData, Options{OneK: b1K})
			if rs.err != nil || rr.err != nil {
				t.Fatalf("1K=%v 크기 %d: 보내기 %v / 받기 %v", b1K, iSize, rs.err, rr.err)
			}
			if !bytes.Equal(bGot, bData) || rs.n != int64(iSize) || rr.n != int64(iSize) {
				t.Fatalf("1K=%v 크기 %d: 받은 %d바이트 (보냄 %d, 받음 %d)", b1K, iSize, len(bGot), rs.n, rr.n)
			}
		}
	}
}

func TestRoundTripChecksumMode(t *testing.T) {
	fastTimes(t)
	iSaved := g_iCrcStartTries
	g_iCrcStartTries = 0 // 받는 쪽이 처음부터 NAK(체크섬)를 보낸다
	t.Cleanup(func() { g_iCrcStartTries = iSaved })
	sender, receiver := newPair()
	var bHeads []byte
	var mtx sync.Mutex
	sender.mangle = func(bData []byte) []byte {
		mtx.Lock()
		bHeads = append(bHeads, bData[0])
		mtx.Unlock()
		return bData
	}
	bData := randomData(700, 7)
	bGot, rs, rr := transfer(t, sender, receiver, bData, Options{OneK: true})
	if rs.err != nil || rr.err != nil || !bytes.Equal(bGot, bData) {
		t.Fatalf("체크섬 방식: %v / %v, %d바이트", rs.err, rr.err, len(bGot))
	}
	mtx.Lock()
	defer mtx.Unlock()
	for _, b := range bHeads {
		if b == STX {
			t.Fatal("체크섬 방식에서는 1K 블록을 쓰지 않는다")
		}
	}
}

func TestCorruptedBlockIsResent(t *testing.T) {
	fastTimes(t)
	sender, receiver := newPair()
	var iPackets atomic.Int32
	sender.mangle = func(bData []byte) []byte {
		if len(bData) > 100 && iPackets.Add(1) == 2 {
			bData[50] ^= 0xFF // 두 번째 블록을 한 번만 깨뜨린다
		}
		return bData
	}
	bData := randomData(600, 3)
	bGot, rs, rr := transfer(t, sender, receiver, bData, Options{})
	if rs.err != nil || rr.err != nil || !bytes.Equal(bGot, bData) {
		t.Fatalf("깨진 블록 재전송: %v / %v, %d바이트", rs.err, rr.err, len(bGot))
	}
	if iPackets.Load() < 6 { // 5블록 + 다시 보낸 1블록
		t.Fatalf("블록을 다시 보내지 않았다: %d", iPackets.Load())
	}
}

func TestLostAckGivesDuplicateThatIsIgnored(t *testing.T) {
	fastTimes(t)
	sender, receiver := newPair()
	var bDropped atomic.Bool
	receiver.mangle = func(bData []byte) []byte {
		if len(bData) == 1 && bData[0] == ACK && !bDropped.Swap(true) {
			return nil // 첫 ACK 가 사라진다: 보내는 쪽이 같은 블록을 다시 보낸다
		}
		return bData
	}
	bData := randomData(400, 4)
	bGot, rs, rr := transfer(t, sender, receiver, bData, Options{})
	if rs.err != nil || rr.err != nil || !bytes.Equal(bGot, bData) {
		t.Fatalf("중복 블록: %v / %v, %d바이트", rs.err, rr.err, len(bGot))
	}
}

func TestSenderCancelReachesReceiver(t *testing.T) {
	fastTimes(t)
	sender, receiver := newPair()
	var bCancel atomic.Bool
	var iBlocks atomic.Int32
	_, rs, rr := transfer(t, sender, receiver, randomData(5000, 5), Options{
		Cancelled: bCancel.Load,
		Progress: func(int64, int64) {
			if iBlocks.Add(1) == 3 {
				bCancel.Store(true)
			}
		},
	})
	if !errors.Is(rs.err, ErrCancelled) || !errors.Is(rr.err, ErrRemoteCancelled) {
		t.Fatalf("취소: 보내기 %v / 받기 %v", rs.err, rr.err)
	}
}

func TestReceiverCancelReachesSender(t *testing.T) {
	fastTimes(t)
	sender, receiver := newPair()
	var bCancel atomic.Bool
	chSend := make(chan error, 1)
	go func() {
		_, err := Send(sender, bytes.NewReader(randomData(5000, 6)), 5000, Options{})
		chSend <- err
	}()
	var iBlocks atomic.Int32
	_, errRecv := Receive(receiver, io.Discard, Options{Cancelled: bCancel.Load, Progress: func(int64, int64) {
		if iBlocks.Add(1) == 2 {
			bCancel.Store(true)
		}
	}})
	if !errors.Is(errRecv, ErrCancelled) || !errors.Is(<-chSend, ErrRemoteCancelled) {
		t.Fatalf("받는 쪽 취소: %v", errRecv)
	}
}

func TestSenderTimesOutWithoutReceiver(t *testing.T) {
	fastTimes(t)
	G_durStart = 300 * time.Millisecond
	sender, _ := newPair()
	if _, errSend := Send(sender, bytes.NewReader([]byte("x")), 1, Options{}); !errors.Is(errSend, ErrNoStart) {
		t.Fatalf("받는 쪽이 없을 때: %v", errSend)
	}
}

func TestReceiverGivesUpWithoutSender(t *testing.T) {
	fastTimes(t)
	_, receiver := newPair()
	if _, errRecv := Receive(receiver, io.Discard, Options{}); !errors.Is(errRecv, ErrNoStart) {
		t.Fatalf("보내는 쪽이 없을 때: %v", errRecv)
	}
}

func TestReceiverStripsOnlyFinalPadding(t *testing.T) {
	fastTimes(t)
	// 가운데 블록에 SUB 가 있어도 지우지 않고, 마지막 블록 끝의 채움만 뗀다.
	bData := bytes.Repeat([]byte{SUB}, 128)
	bData = append(bData, []byte("end")...)
	sender, receiver := newPair()
	bGot, rs, rr := transfer(t, sender, receiver, bData, Options{})
	if rs.err != nil || rr.err != nil || !bytes.Equal(bGot, bData) {
		t.Fatalf("채움 떼기: %v / %v, %q", rs.err, rr.err, bGot)
	}
}
