package transport

import (
	"dariterm/internal/i18n"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// ErrPortClosed 는 시리얼 포트가 닫혔을 때의 이유다 (protocols.md §5).
var ErrPortClosed = i18n.NewError("be.serial.closed")

// G_cdurSerialReadTick 은 읽기 시한이다. 이 주기로 Read 가 돌아와 대기 중인 로컬 에코를 내보낸다 (protocols.md §5).
const G_cdurSerialReadTick = 50 * time.Millisecond

// 흐름 제어 (protocols.md §5, D108). 드라이버가 아니라 앱이 한다: 보내기 전에 장치가 받을 수 있는지 확인한다.
const (
	// G_cszFlowNone·G_cszFlowRtsCts·G_cszFlowXonXoff 는 서버 설정 serial.flowControl 의 값이다.
	G_cszFlowNone    = "none"
	G_cszFlowRtsCts  = "rtscts"
	G_cszFlowXonXoff = "xonxoff"
	// g_ciSerialFlowChunk 는 흐름 제어가 켜져 있을 때 한 번에 보내는 바이트 수다. 조각 사이마다 장치 상태를 다시 본다.
	g_ciSerialFlowChunk = 16
	// g_cdurSerialFlowPoll 은 장치가 받을 수 있게 되기를 기다리며 상태를 다시 보는 간격이다.
	g_cdurSerialFlowPoll = 5 * time.Millisecond
	// g_cbXon·g_cbXoff 는 XON/XOFF 흐름 제어 문자(DC1·DC3)다.
	g_cbXon  = 0x11
	g_cbXoff = 0x13
)

// G_durSerialFlowWait 는 장치가 받을 수 있게 되기를 기다리는 최대 시간이다. 넘으면 남은 입력을 버리고 안내한다.
// 시험에서 줄일 수 있게 변수로 둔다.
var G_durSerialFlowWait = 2 * time.Second

// SerialOptions 는 시리얼 열기 설정이다 (protocols.md §5, data/store.md §3 serial).
type SerialOptions struct {
	Path       string
	BaudRate   int
	DataBits   int
	Parity     string
	StopBits   float64
	EnterSends string
	LocalEcho  bool
	// FlowControl 은 흐름 제어 방식이다: none(기본) · rtscts · xonxoff.
	FlowControl string
}

// SerialConn 은 열린 시리얼 포트 하나다.
type SerialConn struct {
	port        serial.Port
	opts        SerialOptions
	mtxWrite    sync.Mutex
	mtxState    sync.Mutex
	bPendingOut []byte
	bPeerXoff   bool // XON/XOFF: 장치가 XOFF 를 보내 멈춰 달라고 한 상태 (mtxState)
	bFlowWarned bool // 흐름 제어로 입력을 버렸다는 안내를 이미 냈다 (다시 보낼 수 있게 되면 지운다) (mtxState)
	bClosed     bool // Close 가 불렸다: 기다리던 쓰기를 끝낸다 (mtxState)
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
	// 읽기 시한: 장치가 조용해도 Read 가 주기적으로 돌아와야 로컬 에코가 바로 보인다.
	// go.bug.st/serial 은 시한이 지나면 Read 가 (0, nil) 을 돌려준다 (Windows·Unix 구현 확인, v1.8.0).
	if errTimeout := port.SetReadTimeout(G_cdurSerialReadTick); errTimeout != nil {
		_ = port.Close()
		return nil, fmt.Errorf("%s: %v", opts.Path, errTimeout)
	}
	return newSerialConn(port, opts), nil
}

// newSerialConn 은 열린 포트를 감싼다. RTS/CTS 면 RTS 를 올려 "이 PC 는 받을 수 있다"고 알린다
// (이 앱은 받은 데이터를 곧바로 읽어 가므로 RTS 를 내리지 않는다).
func newSerialConn(port serial.Port, opts SerialOptions) *SerialConn {
	if opts.FlowControl == G_cszFlowRtsCts {
		_ = port.SetRTS(true)
	}
	return &SerialConn{port: port, opts: opts}
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

// takePending 은 대기 중인 출력(로컬 에코·남은 조각)을 bBuf 로 옮기고 옮긴 길이를 돌려준다.
func (pSerial *SerialConn) takePending(bBuf []byte) int {
	pSerial.mtxState.Lock()
	defer pSerial.mtxState.Unlock()
	if len(pSerial.bPendingOut) == 0 {
		return 0
	}
	nCopy := copy(bBuf, pSerial.bPendingOut)
	pSerial.bPendingOut = pSerial.bPendingOut[nCopy:]
	if len(pSerial.bPendingOut) == 0 {
		pSerial.bPendingOut = nil
	}
	return nCopy
}

// Read 는 포트 출력을 읽는다. 로컬 에코가 켜져 있으면 보낸 내용을 화면에도 보여 준다 (Write 에서 쌓는다).
// 읽기 시한(G_cdurSerialReadTick)마다 돌아와 대기 중인 에코가 있으면 먼저 내보낸다.
func (pSerial *SerialConn) Read(bBuf []byte) (int, error) {
	bRaw := make([]byte, 4096)
	for {
		if nPending := pSerial.takePending(bBuf); nPending > 0 {
			return nPending, nil
		}
		nRaw, errRead := pSerial.port.Read(bRaw)
		if errRead != nil {
			return 0, ErrPortClosed
		}
		if nRaw == 0 {
			continue // 시한: 데이터 없음. 위에서 대기분을 다시 본다.
		}
		bData := bRaw[:nRaw]
		if pSerial.opts.FlowControl == G_cszFlowXonXoff {
			bData = pSerial.takeXonXoff(bData)
			if len(bData) == 0 {
				continue // 흐름 제어 문자뿐이었다
			}
		}
		nCopy := copy(bBuf, bData)
		if nCopy < len(bData) {
			pSerial.mtxState.Lock()
			pSerial.bPendingOut = append(pSerial.bPendingOut, bData[nCopy:]...)
			pSerial.mtxState.Unlock()
		}
		return nCopy, nil
	}
}

// takeXonXoff 는 받은 데이터에서 XON·XOFF 를 빼고(화면에 보이지 않게) 장치의 멈춤 상태를 갱신한다.
func (pSerial *SerialConn) takeXonXoff(bData []byte) []byte {
	bOut := bData[:0]
	pSerial.mtxState.Lock()
	defer pSerial.mtxState.Unlock()
	for _, bOne := range bData {
		switch bOne {
		case g_cbXoff:
			pSerial.bPeerXoff = true
		case g_cbXon:
			pSerial.bPeerXoff = false
		default:
			bOut = append(bOut, bOne)
		}
	}
	return bOut
}

// peerReady 는 장치가 지금 받을 수 있는지 본다. RTS/CTS 는 CTS 선, XON/XOFF 는 마지막으로 받은 문자로 판단한다.
// CTS 를 읽을 수 없는 장치(일부 USB 변환기)는 받을 수 있다고 본다 — 영영 막히는 것보다 낫다.
func (pSerial *SerialConn) peerReady() bool {
	switch pSerial.opts.FlowControl {
	case G_cszFlowRtsCts:
		pBits, errBits := pSerial.port.GetModemStatusBits()
		return errBits != nil || pBits == nil || pBits.CTS
	case G_cszFlowXonXoff:
		pSerial.mtxState.Lock()
		defer pSerial.mtxState.Unlock()
		return !pSerial.bPeerXoff
	default:
		return true
	}
}

// waitPeerReady 는 장치가 받을 수 있게 될 때까지 tDeadline 까지 기다린다. 받을 수 있으면 true,
// 시한이 지났거나 포트를 닫았으면 false 다.
func (pSerial *SerialConn) waitPeerReady(tDeadline time.Time) bool {
	for {
		pSerial.mtxState.Lock()
		bClosed := pSerial.bClosed
		pSerial.mtxState.Unlock()
		if bClosed {
			return false
		}
		if pSerial.peerReady() {
			return true
		}
		if !time.Now().Before(tDeadline) {
			return false
		}
		time.Sleep(g_cdurSerialFlowPoll)
	}
}

// writeFlow 는 흐름 제어를 지키며 보낸다: 작은 조각으로 나눠 조각마다 장치가 받을 수 있는지 확인한다.
// 실제로 보낸 길이를 돌려준다. 장치가 G_durSerialFlowWait 동안 받지 못하면 남은 것은 버린다.
func (pSerial *SerialConn) writeFlow(bSend []byte) (int, error) {
	tDeadline := time.Now().Add(G_durSerialFlowWait)
	nSent := 0
	for nSent < len(bSend) {
		if !pSerial.waitPeerReady(tDeadline) {
			return nSent, nil
		}
		nEnd := nSent + g_ciSerialFlowChunk
		if nEnd > len(bSend) {
			nEnd = len(bSend)
		}
		if _, errWrite := pSerial.port.Write(bSend[nSent:nEnd]); errWrite != nil {
			return nSent, errWrite
		}
		nSent = nEnd
		tDeadline = time.Now().Add(G_durSerialFlowWait) // 조각이 나갔으면 다음 조각은 새로 기다린다
	}
	return nSent, nil
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
	bFlow := pSerial.opts.FlowControl == G_cszFlowRtsCts || pSerial.opts.FlowControl == G_cszFlowXonXoff
	nSent := len(bSend)
	var errWrite error
	pSerial.mtxWrite.Lock()
	if bFlow {
		nSent, errWrite = pSerial.writeFlow(bSend)
	} else {
		_, errWrite = pSerial.port.Write(bSend)
	}
	pSerial.mtxWrite.Unlock()
	if errWrite != nil {
		return ErrPortClosed
	}
	if bFlow {
		pSerial.noteFlowResult(nSent == len(bSend))
		if nSent < len(bSend) {
			return nil // 장치가 받지 못해 버렸다: 에코도 하지 않는다 (안내는 noteFlowResult 가 냈다)
		}
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

// noteFlowResult 는 흐름 제어로 입력을 버렸을 때 화면에 노랑 안내 한 줄을 낸다.
// 막힌 동안에는 한 번만 내고, 다시 보낼 수 있게 되면 다음에 또 낼 수 있게 한다.
func (pSerial *SerialConn) noteFlowResult(bAllSent bool) {
	pSerial.mtxState.Lock()
	defer pSerial.mtxState.Unlock()
	if bAllSent {
		pSerial.bFlowWarned = false
		return
	}
	if pSerial.bFlowWarned || pSerial.bClosed {
		return
	}
	pSerial.bFlowWarned = true
	pSerial.bPendingOut = append(pSerial.bPendingOut, []byte("\r\n\x1b[33m● "+i18n.T("be.serial.flowBlocked")+"\x1b[0m\r\n")...)
}

// Resize 는 시리얼에 아무 영향이 없다 (protocols.md §5).
func (pSerial *SerialConn) Resize(iCols int, iRows int) error {
	return nil
}

// Close 는 포트를 닫는다.
func (pSerial *SerialConn) Close() error {
	pSerial.mtxState.Lock()
	pSerial.bClosed = true // 흐름 제어로 기다리던 쓰기를 끝낸다
	pSerial.mtxState.Unlock()
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
