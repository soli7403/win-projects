package transport

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// SerialOptions 는 SerialOptions(shared/types.ts) 대응.
type SerialOptions struct {
	Path        string  `json:"path"`
	BaudRate    int     `json:"baudRate"`
	DataBits    int     `json:"dataBits"`
	Parity      string  `json:"parity"`
	StopBits    float64 `json:"stopBits"`
	FlowControl string  `json:"flowControl"`
	EnterSends  string  `json:"enterSends"`
	LocalEcho   bool    `json:"localEcho"`
}

// PortInfo 는 포트 목록 항목.
type PortInfo struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// ListSerialPorts 는 사용 가능한 시리얼 포트를 돌려준다 (실패 시 빈 목록).
func ListSerialPorts() []PortInfo {
	out := []PortInfo{}
	if arr, err := enumerator.GetDetailedPortsList(); err == nil {
		for _, p := range arr {
			sz := p.Name
			if p.Product != "" {
				sz += " · " + p.Product
			}
			out = append(out, PortInfo{Path: p.Name, Label: sz})
		}
		if len(out) > 0 {
			return out
		}
	}
	if arr, err := serial.GetPortsList(); err == nil {
		for _, s := range arr {
			out = append(out, PortInfo{Path: s, Label: s})
		}
	}
	return out
}

// Serial 은 시리얼 포트 트랜스포트.
type Serial struct {
	port     serial.Port
	o        SerialOptions
	cb       Callbacks
	once     *OnceClose
	bClosing atomic.Bool
}

// StartSerial 은 포트를 열고 읽기 루프를 시작한다.
func StartSerial(o SerialOptions, cb Callbacks) (*Serial, error) {
	mode := &serial.Mode{BaudRate: o.BaudRate, DataBits: o.DataBits}
	if mode.BaudRate == 0 {
		mode.BaudRate = 9600
	}
	if mode.DataBits == 0 {
		mode.DataBits = 8
	}
	switch o.Parity {
	case "even":
		mode.Parity = serial.EvenParity
	case "odd":
		mode.Parity = serial.OddParity
	case "mark":
		mode.Parity = serial.MarkParity
	case "space":
		mode.Parity = serial.SpaceParity
	default:
		mode.Parity = serial.NoParity
	}
	switch o.StopBits {
	case 1.5:
		mode.StopBits = serial.OnePointFiveStopBits
	case 2:
		mode.StopBits = serial.TwoStopBits
	default:
		mode.StopBits = serial.OneStopBit
	}
	p, err := serial.Open(o.Path, mode)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.Path, err)
	}
	pstrS := &Serial{port: p, o: o, cb: cb, once: NewOnceClose(cb.OnClose)}
	go func() {
		arr := make([]byte, 4096)
		for {
			n, err := p.Read(arr)
			if n > 0 {
				cb.OnData(append([]byte(nil), arr[:n]...))
			}
			if err != nil || n == 0 && pstrS.bClosing.Load() {
				if pstrS.bClosing.Load() {
					pstrS.once.Fire("")
				} else {
					pstrS.once.Fire("포트가 닫혔습니다")
				}
				return
			}
		}
	}()
	return pstrS, nil
}

// Write 는 Enter 변환(CR/LF/CRLF)과 로컬 에코를 적용해 보낸다.
func (pstrS *Serial) Write(arr []byte) error {
	szEol := "\r"
	switch pstrS.o.EnterSends {
	case "CRLF":
		szEol = "\r\n"
	case "LF":
		szEol = "\n"
	}
	out := arr
	if szEol != "\r" && bytes.IndexByte(arr, '\r') >= 0 {
		out = bytes.ReplaceAll(arr, []byte("\r"), []byte(szEol))
	}
	if pstrS.o.LocalEcho {
		echo := bytes.ReplaceAll(out, []byte("\r\n"), []byte("\x00"))
		echo = bytes.ReplaceAll(echo, []byte("\r"), []byte("\r\n"))
		echo = bytes.ReplaceAll(echo, []byte("\x00"), []byte("\r\n"))
		pstrS.cb.OnData(echo)
	}
	_, err := pstrS.port.Write(out)
	return err
}

// Resize 는 시리얼에서는 아무 일도 하지 않는다.
func (pstrS *Serial) Resize(int, int) {}

// Close 는 포트를 닫는다.
func (pstrS *Serial) Close() {
	pstrS.bClosing.Store(true)
	_ = pstrS.port.Close()
}
