package transport

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// ErrPortClosed 는 시리얼 포트가 닫혔을 때의 이유다 (protocols.md §5).
var ErrPortClosed = errors.New("포트가 닫혔습니다")

// SerialOptions 는 시리얼 열기 설정이다 (protocols.md §5, data/store.md §3 serial).
type SerialOptions struct {
	Path       string
	BaudRate   int
	DataBits   int
	Parity     string
	StopBits   float64
	EnterSends string
	LocalEcho  bool
}

// SerialConn 은 열린 시리얼 포트 하나다.
type SerialConn struct {
	port        serial.Port
	opts        SerialOptions
	mtxWrite    sync.Mutex
	mtxState    sync.Mutex
	bPendingOut []byte
}

// OpenSerial 은 시리얼 장치를 연다. 속도·데이터 비트는 0 이면 기본값 (9600, 8).
// 열기 실패 오류는 "{장치}: {원인}" 이다.
func OpenSerial(opts SerialOptions) (*SerialConn, error) {
	mode := &serial.Mode{
		BaudRate: opts.BaudRate,
		DataBits: opts.DataBits,
		Parity:   parityOf(opts.Parity),
		StopBits: stopBitsOf(opts.StopBits),
	}
	if mode.BaudRate == 0 {
		mode.BaudRate = 9600
	}
	if mode.DataBits == 0 {
		mode.DataBits = 8
	}
	port, errOpen := serial.Open(opts.Path, mode)
	if errOpen != nil {
		return nil, fmt.Errorf("%s: %v", opts.Path, errOpen)
	}
	return &SerialConn{port: port, opts: opts}, nil
}

func parityOf(szName string) serial.Parity {
	switch szName {
	case "even":
		return serial.EvenParity
	case "odd":
		return serial.OddParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

func stopBitsOf(fStop float64) serial.StopBits {
	switch fStop {
	case 1.5:
		return serial.OnePointFiveStopBits
	case 2:
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

// Read 는 포트 출력을 읽는다. 로컬 에코가 켜져 있으면 보낸 내용을 화면에도 보여 준다 (write 에서 쌓는다).
func (pSerial *SerialConn) Read(bBuf []byte) (int, error) {
	pSerial.mtxState.Lock()
	if len(pSerial.bPendingOut) > 0 {
		nCopy := copy(bBuf, pSerial.bPendingOut)
		pSerial.bPendingOut = pSerial.bPendingOut[nCopy:]
		pSerial.mtxState.Unlock()
		return nCopy, nil
	}
	pSerial.mtxState.Unlock()
	bRaw := make([]byte, 4096)
	for {
		nRaw, errRead := pSerial.port.Read(bRaw)
		if errRead != nil {
			return 0, ErrPortClosed
		}
		if nRaw > 0 {
			nCopy := copy(bBuf, bRaw[:nRaw])
			if nCopy < nRaw {
				pSerial.mtxState.Lock()
				pSerial.bPendingOut = append(pSerial.bPendingOut, bRaw[nCopy:nRaw]...)
				pSerial.mtxState.Unlock()
			}
			return nCopy, nil
		}
	}
}

// enterBytes 는 Enter 전송값(CR/LF/CRLF)을 돌려준다 (protocols.md §5).
func enterBytes(szMode string) string {
	switch szMode {
	case "LF":
		return "\n"
	case "CRLF":
		return "\r\n"
	default:
		return "\r"
	}
}

// Write 는 입력의 CR 을 Enter 전송값으로 바꿔 보낸다. 로컬 에코가 켜져 있으면 화면에도 같은 내용을 보낸다.
func (pSerial *SerialConn) Write(bData []byte) error {
	szEnter := enterBytes(pSerial.opts.EnterSends)
	szText := strings.ReplaceAll(string(bData), "\r", szEnter)
	bSend := []byte(szText)
	pSerial.mtxWrite.Lock()
	_, errWrite := pSerial.port.Write(bSend)
	pSerial.mtxWrite.Unlock()
	if errWrite != nil {
		return ErrPortClosed
	}
	if pSerial.opts.LocalEcho {
		// CR 과 CRLF 는 화면에서 줄바꿈으로 보인다.
		szEcho := strings.ReplaceAll(string(bData), "\r\n", "\n")
		szEcho = strings.ReplaceAll(szEcho, "\r", "\n")
		szEcho = strings.ReplaceAll(szEcho, "\n", "\r\n")
		pSerial.mtxState.Lock()
		pSerial.bPendingOut = append(pSerial.bPendingOut, []byte(szEcho)...)
		pSerial.mtxState.Unlock()
	}
	return nil
}

// Resize 는 시리얼에 아무 영향이 없다 (protocols.md §5).
func (pSerial *SerialConn) Resize(iCols int, iRows int) error {
	return nil
}

// Close 는 포트를 닫는다.
func (pSerial *SerialConn) Close() error {
	return pSerial.port.Close()
}

// ListSerialPorts 는 장치 목록을 돌려준다. 라벨은 "포트 이름 · 제품명"이고 제품명이 없으면 이름만 쓴다.
// 상세 열거가 실패하거나 비면 단순 목록으로 대체하며, 그것도 실패하면 빈 목록이다 (protocols.md §5).
func ListSerialPorts() []PortItem {
	slOut := []PortItem{}
	details, errDetail := enumerator.GetDetailedPortsList()
	if errDetail == nil && len(details) > 0 {
		for _, item := range details {
			szLabel := item.Name
			if item.Product != "" {
				szLabel += " · " + item.Product
			}
			slOut = append(slOut, PortItem{Path: item.Name, Label: szLabel})
		}
		return slOut
	}
	names, errNames := serial.GetPortsList()
	if errNames != nil {
		return slOut
	}
	for _, szName := range names {
		slOut = append(slOut, PortItem{Path: szName, Label: szName})
	}
	return slOut
}

// PortItem 은 장치 목록 항목 {path, label} 이다 (spec/ipc.md §2 serial.list).
type PortItem struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}
