package transport

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
)

// fakeSerialPort 는 장치 없이 시리얼 포트를 흉내 낸다. Read 는 시한마다 (0, nil) 을 돌려주고(조용한 장치),
// chIn 에 넣은 바이트가 있으면 그것을 돌려준다. Close 뒤에는 오류다.
type fakeSerialPort struct {
	chIn     chan []byte
	mtxData  sync.Mutex
	slWrites []string
	bClosed  bool
	bCts     bool  // 장치의 CTS 선 (흐름 제어 시험용)
	bRts     bool  // 이 PC 가 올린 RTS
	errBits  error // 모뎀 상태를 읽지 못하는 장치 흉내
}

func (pFake *fakeSerialPort) SetMode(mode *serial.Mode) error { return nil }
func (pFake *fakeSerialPort) Read(bBuf []byte) (int, error) {
	pFake.mtxData.Lock()
	bClosed := pFake.bClosed
	pFake.mtxData.Unlock()
	if bClosed {
		return 0, errors.New("closed")
	}
	select {
	case bData := <-pFake.chIn:
		return copy(bBuf, bData), nil
	case <-time.After(10 * time.Millisecond):
		return 0, nil // 읽기 시한
	}
}
func (pFake *fakeSerialPort) Write(bData []byte) (int, error) {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	pFake.slWrites = append(pFake.slWrites, string(bData))
	return len(bData), nil
}
func (pFake *fakeSerialPort) Drain() error             { return nil }
func (pFake *fakeSerialPort) ResetInputBuffer() error  { return nil }
func (pFake *fakeSerialPort) ResetOutputBuffer() error { return nil }
func (pFake *fakeSerialPort) SetDTR(bDtr bool) error   { return nil }
func (pFake *fakeSerialPort) SetRTS(bRts bool) error {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	pFake.bRts = bRts
	return nil
}
func (pFake *fakeSerialPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	if pFake.errBits != nil {
		return nil, pFake.errBits
	}
	return &serial.ModemStatusBits{CTS: pFake.bCts}, nil
}
func (pFake *fakeSerialPort) SetReadTimeout(durTimeout time.Duration) error { return nil }
func (pFake *fakeSerialPort) Break(durBreak time.Duration) error            { return nil }
func (pFake *fakeSerialPort) Close() error {
	pFake.mtxData.Lock()
	pFake.bClosed = true
	pFake.mtxData.Unlock()
	return nil
}

// 장치가 아무것도 보내지 않아도 로컬 에코는 바로 Read 로 돌아온다 (protocols.md §5, 통합 계획 §2).
func TestSerialLocalEchoWithoutDeviceData(t *testing.T) {
	pFake := &fakeSerialPort{chIn: make(chan []byte, 4)}
	pSerial := &SerialConn{port: pFake, opts: SerialOptions{LocalEcho: true, EnterSends: "CRLF"}}
	chRead := make(chan string, 4)
	go func() {
		bBuf := make([]byte, 64)
		for {
			nRead, errRead := pSerial.Read(bBuf)
			if errRead != nil {
				close(chRead)
				return
			}
			chRead <- string(bBuf[:nRead])
		}
	}()
	time.Sleep(30 * time.Millisecond) // Read 가 포트 읽기에서 기다리는 중
	if errWrite := pSerial.Write([]byte("ab\r")); errWrite != nil {
		t.Fatal(errWrite)
	}
	select {
	case szGot := <-chRead:
		if szGot != "ab\r\n" {
			t.Fatalf("에코 내용: %q", szGot)
		}
	case <-time.After(time.Second):
		t.Fatal("장치 데이터 없이도 에코가 보여야 함")
	}
	pFake.mtxData.Lock()
	if len(pFake.slWrites) != 1 || pFake.slWrites[0] != "ab\r\n" {
		t.Fatalf("장치로 보낸 값(Enter=CRLF): %q", pFake.slWrites)
	}
	pFake.mtxData.Unlock()
	pFake.chIn <- []byte("dev")
	select {
	case szGot := <-chRead:
		if szGot != "dev" {
			t.Fatalf("장치 데이터: %q", szGot)
		}
	case <-time.After(time.Second):
		t.Fatal("장치 데이터가 와야 함")
	}
	_ = pSerial.Close()
	select {
	case _, bOk := <-chRead:
		if bOk {
			t.Fatal("닫은 뒤에는 Read 가 오류로 끝나야 함")
		}
	case <-time.After(time.Second):
		t.Fatal("닫은 뒤 Read 가 돌아오지 않음")
	}
}
