package transport

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	tIAC  = 255
	tDONT = 254
	tDO   = 253
	tWONT = 252
	tWILL = 251
	tSB   = 250
	tSE   = 240

	tEcho  = 1
	tSGA   = 3
	tTType = 24
	tNaws  = 31
)

// Telnet 은 RFC854 최소 구현 (ECHO, SGA, TTYPE, NAWS 협상).
type Telnet struct {
	conn     net.Conn
	szTerm   string
	mu       sync.Mutex
	iCols    int
	iRows    int
	bNaws    bool
	state    int // 0 data, 1 iac, 2 cmd, 3 sb, 4 sbiac
	iCmd     byte
	arrSB    []byte
	closeFn  *OnceClose
	cbData   func([]byte)
	bClosing bool
}

// StartTelnet 은 접속 후 읽기 루프를 시작한다.
func StartTelnet(szHost string, iPort int, szTerm string, iCols, iRows int, cb Callbacks) (*Telnet, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(szHost, fmt.Sprint(iPort)), 15*time.Second)
	if err != nil {
		return nil, err
	}
	pstrT := &Telnet{conn: conn, szTerm: szTerm, iCols: iCols, iRows: iRows, cbData: cb.OnData, closeFn: NewOnceClose(cb.OnClose)}
	go pstrT.readLoop()
	return pstrT, nil
}

func (pstrT *Telnet) readLoop() {
	arr := make([]byte, 16*1024)
	for {
		n, err := pstrT.conn.Read(arr)
		if n > 0 {
			if out := pstrT.parse(arr[:n]); len(out) > 0 {
				pstrT.cbData(out)
			}
		}
		if err != nil {
			if pstrT.bClosing {
				pstrT.closeFn.Fire("")
			} else {
				pstrT.closeFn.Fire("연결이 종료되었습니다")
			}
			_ = pstrT.conn.Close()
			return
		}
	}
}

func (pstrT *Telnet) sendCmd(verb, opt byte) { _, _ = pstrT.conn.Write([]byte{tIAC, verb, opt}) }

func (pstrT *Telnet) sendNaws() {
	pstrT.mu.Lock()
	bOn, c, r := pstrT.bNaws, pstrT.iCols, pstrT.iRows
	pstrT.mu.Unlock()
	if !bOn {
		return
	}
	_, _ = pstrT.conn.Write([]byte{tIAC, tSB, tNaws, byte(c >> 8), byte(c), byte(r >> 8), byte(r), tIAC, tSE})
}

func (pstrT *Telnet) parse(d []byte) []byte {
	out := make([]byte, 0, len(d))
	for _, b := range d {
		switch pstrT.state {
		case 0:
			if b == tIAC {
				pstrT.state = 1
			} else {
				out = append(out, b)
			}
		case 1:
			switch {
			case b == tIAC:
				out = append(out, 255)
				pstrT.state = 0
			case b == tSB:
				pstrT.state, pstrT.arrSB = 3, pstrT.arrSB[:0]
			case b >= tWILL && b <= tDONT:
				pstrT.iCmd, pstrT.state = b, 2
			default:
				pstrT.state = 0
			}
		case 2:
			pstrT.negotiate(pstrT.iCmd, b)
			pstrT.state = 0
		case 3:
			if b == tIAC {
				pstrT.state = 4
			} else {
				pstrT.arrSB = append(pstrT.arrSB, b)
			}
		case 4:
			if b == tSE {
				pstrT.subneg(pstrT.arrSB)
				pstrT.state = 0
			} else {
				pstrT.arrSB = append(pstrT.arrSB, b)
				pstrT.state = 3
			}
		}
	}
	return out
}

func (pstrT *Telnet) negotiate(verb, opt byte) {
	switch verb {
	case tDO:
		switch opt {
		case tNaws:
			pstrT.sendCmd(tWILL, tNaws)
			pstrT.mu.Lock()
			pstrT.bNaws = true
			pstrT.mu.Unlock()
			pstrT.sendNaws()
		case tTType:
			pstrT.sendCmd(tWILL, tTType)
		default:
			pstrT.sendCmd(tWONT, opt)
		}
	case tWILL:
		if opt == tEcho || opt == tSGA {
			pstrT.sendCmd(tDO, opt)
		} else {
			pstrT.sendCmd(tDONT, opt)
		}
	}
}

func (pstrT *Telnet) subneg(b []byte) {
	if len(b) >= 2 && b[0] == tTType && b[1] == 1 {
		sz := pstrT.szTerm
		if sz == "" {
			sz = "xterm-256color"
		}
		msg := append([]byte{tIAC, tSB, tTType, 0}, []byte(sz)...)
		_, _ = pstrT.conn.Write(append(msg, tIAC, tSE))
	}
}

// Write 는 IAC(255) 바이트를 이스케이프해 보낸다.
func (pstrT *Telnet) Write(arr []byte) error {
	out := make([]byte, 0, len(arr))
	for _, b := range arr {
		out = append(out, b)
		if b == tIAC {
			out = append(out, tIAC)
		}
	}
	_, err := pstrT.conn.Write(out)
	return err
}

// Resize 는 NAWS 로 창 크기를 알린다.
func (pstrT *Telnet) Resize(iCols, iRows int) {
	pstrT.mu.Lock()
	pstrT.iCols, pstrT.iRows = iCols, iRows
	pstrT.mu.Unlock()
	pstrT.sendNaws()
}

// Close 는 연결을 닫는다.
func (pstrT *Telnet) Close() {
	pstrT.bClosing = true
	_ = pstrT.conn.Close()
}
