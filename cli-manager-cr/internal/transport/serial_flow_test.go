package transport

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// 흐름 제어 시험 (protocols.md §5, D108).

func (pFake *fakeSerialPort) setCts(bCts bool) {
	pFake.mtxData.Lock()
	pFake.bCts = bCts
	pFake.mtxData.Unlock()
}

func (pFake *fakeSerialPort) written() string {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return strings.Join(pFake.slWrites, "")
}

func (pFake *fakeSerialPort) writeCount() int {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return len(pFake.slWrites)
}

// withFlowWait 는 시험 동안 "장치를 기다리는 최대 시간"을 줄인다.
func withFlowWait(t *testing.T, durWait time.Duration) {
	t.Helper()
	durOld := G_durSerialFlowWait
	G_durSerialFlowWait = durWait
	t.Cleanup(func() { G_durSerialFlowWait = durOld })
}

// RTS/CTS: 열 때 RTS 를 올리고, CTS 가 내려가 있으면 보내지 않고 기다리다가 올라오면 16바이트 조각으로 보낸다.
func TestSerialRtsCtsWaitsForCts(t *testing.T) {
	withFlowWait(t, 2*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts, EnterSends: "CR"})
	pFake.mtxData.Lock()
	bRts := pFake.bRts
	pFake.mtxData.Unlock()
	if !bRts {
		t.Fatal("RTS/CTS 면 열 때 RTS 를 올려야 함")
	}
	szData := strings.Repeat("0123456789", 4) // 40바이트 → 16·16·8
	chDone := make(chan error, 1)
	go func() { chDone <- pSerial.Write([]byte(szData)) }()
	time.Sleep(60 * time.Millisecond)
	if pFake.writeCount() != 0 {
		t.Fatalf("CTS 가 내려가 있으면 보내지 않아야 함: %q", pFake.written())
	}
	pFake.setCts(true)
	select {
	case errWrite := <-chDone:
		if errWrite != nil {
			t.Fatal(errWrite)
		}
	case <-time.After(time.Second):
		t.Fatal("CTS 가 올라오면 보내야 함")
	}
	if pFake.written() != szData || pFake.writeCount() != 3 {
		t.Fatalf("16바이트 조각 3개로 전부 보내야 함: %d개 %q", pFake.writeCount(), pFake.written())
	}
}

// RTS/CTS: 장치가 끝내 받지 못하면 입력을 버리고 노랑 안내를 한 번만 낸다. 다시 받을 수 있게 되면 정상으로 돌아온다.
func TestSerialRtsCtsDropsAfterWaitAndWarnsOnce(t *testing.T) {
	withFlowWait(t, 40*time.Millisecond)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts, EnterSends: "CR", LocalEcho: true})
	for iTry := 0; iTry < 2; iTry++ {
		if errWrite := pSerial.Write([]byte("lost")); errWrite != nil {
			t.Fatalf("버려도 오류는 아님: %v", errWrite)
		}
	}
	if pFake.writeCount() != 0 {
		t.Fatalf("받지 못하는 장치에는 보내지 않아야 함: %q", pFake.written())
	}
	bBuf := make([]byte, 512)
	nRead, _ := pSerial.Read(bBuf)
	szNotice := string(bBuf[:nRead])
	if strings.Count(szNotice, "●") != 1 {
		t.Fatalf("안내는 한 번만: %q", szNotice)
	}
	if strings.Contains(szNotice, "lost") {
		t.Fatalf("버린 입력은 에코하지 않아야 함: %q", szNotice)
	}
	pFake.setCts(true)
	if errWrite := pSerial.Write([]byte("ok")); errWrite != nil || pFake.written() != "ok" {
		t.Fatalf("다시 받을 수 있으면 보내야 함: %v %q", errWrite, pFake.written())
	}
	nRead, _ = pSerial.Read(bBuf)
	if string(bBuf[:nRead]) != "ok" {
		t.Fatalf("보낸 입력은 에코해야 함: %q", string(bBuf[:nRead]))
	}
	// 다시 막히면 안내를 또 낼 수 있다.
	pFake.setCts(false)
	_ = pSerial.Write([]byte("x"))
	nRead, _ = pSerial.Read(bBuf)
	if strings.Count(string(bBuf[:nRead]), "●") != 1 {
		t.Fatalf("다시 막히면 안내를 다시 내야 함: %q", string(bBuf[:nRead]))
	}
}

// XON/XOFF: 장치가 XOFF 를 보내면 멈추고 XON 을 보내면 다시 보낸다. 두 문자는 화면 출력에서 빠진다.
func TestSerialXonXoff(t *testing.T) {
	withFlowWait(t, 2*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 4)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowXonXoff, EnterSends: "CR"})
	pFake.chIn <- []byte("a\x13b")
	bBuf := make([]byte, 64)
	nRead, errRead := pSerial.Read(bBuf)
	if errRead != nil || string(bBuf[:nRead]) != "ab" {
		t.Fatalf("XOFF 는 출력에서 빠져야 함: %q %v", string(bBuf[:nRead]), errRead)
	}
	chDone := make(chan error, 1)
	go func() { chDone <- pSerial.Write([]byte("hello")) }()
	time.Sleep(60 * time.Millisecond)
	if pFake.writeCount() != 0 {
		t.Fatalf("XOFF 뒤에는 보내지 않아야 함: %q", pFake.written())
	}
	// XON 만 온 조각: Read 는 돌려줄 글자가 없으므로 다음 데이터까지 계속 읽는다.
	pFake.chIn <- []byte("\x11")
	pFake.chIn <- []byte("c")
	nRead, _ = pSerial.Read(bBuf)
	if string(bBuf[:nRead]) != "c" {
		t.Fatalf("XON 은 출력에서 빠져야 함: %q", string(bBuf[:nRead]))
	}
	select {
	case errWrite := <-chDone:
		if errWrite != nil || pFake.written() != "hello" {
			t.Fatalf("XON 뒤에는 보내야 함: %v %q", errWrite, pFake.written())
		}
	case <-time.After(time.Second):
		t.Fatal("XON 뒤에는 보내야 함")
	}
}

// 기다리는 중에 포트를 닫으면 쓰기가 곧바로 끝난다. CTS 를 읽을 수 없는 장치는 받을 수 있다고 본다.
func TestSerialFlowCloseAndUnreadableCts(t *testing.T) {
	withFlowWait(t, 5*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts})
	chDone := make(chan struct{})
	go func() { _ = pSerial.Write([]byte("x")); close(chDone) }()
	time.Sleep(30 * time.Millisecond)
	_ = pSerial.Close()
	select {
	case <-chDone:
	case <-time.After(time.Second):
		t.Fatal("닫으면 기다리던 쓰기가 끝나야 함")
	}

	pFake2 := &fakeSerialPort{chIn: make(chan []byte, 1), errBits: errors.New("not supported")}
	pSerial2 := newSerialConn(pFake2, SerialOptions{FlowControl: G_cszFlowRtsCts})
	if errWrite := pSerial2.Write([]byte("go")); errWrite != nil || pFake2.written() != "go" {
		t.Fatalf("CTS 를 읽을 수 없으면 그대로 보내야 함: %v %q", errWrite, pFake2.written())
	}
}

// 흐름 제어 없음: 전과 같이 한 번에 보낸다(조각내지 않는다).
func TestSerialNoFlowWritesWhole(t *testing.T) {
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{})
	_ = pSerial.Write([]byte(strings.Repeat("z", 100)))
	if pFake.writeCount() != 1 {
		t.Fatalf("한 번에 보내야 함: %d", pFake.writeCount())
	}
}
