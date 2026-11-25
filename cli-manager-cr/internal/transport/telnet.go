// Package transport 는 프로토콜별 전송(텔넷·시리얼·SSH·로컬 셸)을 구현한다. session.Transport 를 만족한다 (spec/protocols.md).
package transport

import (
	"dariterm/internal/i18n"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// G_cdurTelnetConnect 는 텔넷 접속 제한 시간이다 (protocols.md §4).
const G_cdurTelnetConnect = 15 * time.Second

// ErrRemoteClosed 는 상대가 연결을 끊었을 때의 오류다 (protocols.md §4: "연결이 종료되었습니다").
// 텔넷은 끊긴 까닭을 붙인 오류(closecause.go 의 telnetClosed)를 돌려주며, errors.Is 로는 이것과 같다 (D127).
var ErrRemoteClosed = i18n.NewError("be.remote.closed")

// ErrRemoteExited 는 서버가 셸을 정상적으로 끝냈을 때(종료 코드·신호와 함께 채널을 닫음)의 오류다.
// 세션은 이 오류면 자동 재접속하지 않는다 (session.md §4). 지금은 SSH 만 돌려준다(텔넷은 구분할 수 없음).
var ErrRemoteExited = i18n.NewError("be.remote.exited")

// 텔넷 명령·옵션 바이트 (RFC 854/855, NAWS=31, TERMINAL-TYPE=24).
const (
	g_byteSE      = 240
	g_byteSB      = 250
	g_byteWill    = 251
	g_byteWont    = 252
	g_byteDo      = 253
	g_byteDont    = 254
	g_byteIAC     = 255
	g_optEcho     = 1
	g_optSGA      = 3
	g_optTermType = 24
	g_optNAWS     = 31
)

// telnetParseState 는 IAC 해석 상태다.
type telnetParseState int

const (
	stateData telnetParseState = iota
	stateIAC
	stateCommand
	stateSub
	stateSubIAC
)

// TelnetConn 은 텔넷 연결 하나다.
type TelnetConn struct {
	conn        net.Conn
	szTermType  string
	mtxWrite    sync.Mutex
	mtxState    sync.Mutex
	iCols       int
	iRows       int
	naws        bool
	state       telnetParseState
	byteCmd     byte
	bSub        []byte
	bPendingOut []byte
	bSubDone    bool
}

// DialTelnet 은 텔넷 서버에 접속한다. szTermType 이 비면 xterm-256color 를 쓴다.
func DialTelnet(szHost string, iPort int, szTermType string) (*TelnetConn, error) {
	if szTermType == "" {
		szTermType = "xterm-256color"
	}
	conn, errDial := net.DialTimeout("tcp", net.JoinHostPort(szHost, strconv.Itoa(iPort)), G_cdurTelnetConnect)
	if errDial != nil {
		return nil, errDial
	}
	return &TelnetConn{conn: conn, szTermType: szTermType}, nil
}

// Read 는 IAC 를 해석해 화면 데이터만 돌려준다. 협상 응답은 내부에서 보낸다.
func (pTelnet *TelnetConn) Read(bBuf []byte) (int, error) {
	pTelnet.mtxState.Lock()
	if len(pTelnet.bPendingOut) > 0 {
		nCopy := copy(bBuf, pTelnet.bPendingOut)
		pTelnet.bPendingOut = pTelnet.bPendingOut[nCopy:]
		pTelnet.mtxState.Unlock()
		return nCopy, nil
	}
	pTelnet.mtxState.Unlock()

	bRaw := make([]byte, 32*1024)
	for {
		nRaw, errRead := pTelnet.conn.Read(bRaw)
		if nRaw > 0 {
			bOut := pTelnet.parse(bRaw[:nRaw])
			if len(bOut) > 0 {
				nCopy := copy(bBuf, bOut)
				pTelnet.mtxState.Lock()
				pTelnet.bPendingOut = append(pTelnet.bPendingOut, bOut[nCopy:]...)
				pTelnet.mtxState.Unlock()
				return nCopy, nil
			}
		}
		if errRead != nil {
			return 0, telnetClosed(errRead) // 끊긴 까닭을 문구에 붙인다 (KI-009, D127)
		}
	}
}

// parse 는 원시 바이트에서 IAC 를 걷어내고 화면 데이터를 돌려준다.
func (pTelnet *TelnetConn) parse(bRaw []byte) []byte {
	pTelnet.mtxState.Lock()
	defer pTelnet.mtxState.Unlock()
	var bOut []byte
	for _, byteValue := range bRaw {
		switch pTelnet.state {
		case stateData:
			if byteValue == g_byteIAC {
				pTelnet.state = stateIAC
			} else {
				bOut = append(bOut, byteValue)
			}
		case stateIAC:
			switch byteValue {
			case g_byteIAC:
				bOut = append(bOut, g_byteIAC)
				pTelnet.state = stateData
			case g_byteWill, g_byteWont, g_byteDo, g_byteDont:
				pTelnet.byteCmd = byteValue
				pTelnet.state = stateCommand
			case g_byteSB:
				pTelnet.bSub = pTelnet.bSub[:0]
				pTelnet.state = stateSub
			default:
				pTelnet.state = stateData
			}
		case stateCommand:
			pTelnet.handleOption(pTelnet.byteCmd, byteValue)
			pTelnet.state = stateData
		case stateSub:
			if byteValue == g_byteIAC {
				pTelnet.state = stateSubIAC
			} else {
				pTelnet.bSub = append(pTelnet.bSub, byteValue)
			}
		case stateSubIAC:
			if byteValue == g_byteSE {
				pTelnet.handleSub(pTelnet.bSub)
				pTelnet.state = stateData
			} else {
				pTelnet.bSub = append(pTelnet.bSub, byteValue)
				pTelnet.state = stateSub
			}
		}
	}
	return bOut
}

// handleOption 은 서버의 WILL/WONT/DO/DONT 에 응답한다 (protocols.md §4).
func (pTelnet *TelnetConn) handleOption(byteCmd byte, byteOpt byte) {
	switch byteCmd {
	case g_byteWill:
		// 서버의 WILL ECHO·SGA 에는 DO, 그 외 WILL 에는 DONT.
		if byteOpt == g_optEcho || byteOpt == g_optSGA {
			pTelnet.send([]byte{g_byteIAC, g_byteDo, byteOpt})
		} else {
			pTelnet.send([]byte{g_byteIAC, g_byteDont, byteOpt})
		}
	case g_byteDo:
		switch byteOpt {
		case g_optNAWS:
			pTelnet.send([]byte{g_byteIAC, g_byteWill, g_optNAWS})
			pTelnet.naws = true
			pTelnet.sendSizeLocked()
		case g_optTermType:
			pTelnet.send([]byte{g_byteIAC, g_byteWill, g_optTermType})
		default:
			pTelnet.send([]byte{g_byteIAC, g_byteWont, byteOpt})
		}
	}
}

// handleSub 은 부옵션 요청을 처리한다 (터미널 종류 SEND → IS).
func (pTelnet *TelnetConn) handleSub(bSub []byte) {
	if len(bSub) >= 2 && bSub[0] == g_optTermType && bSub[1] == 1 {
		bReply := []byte{g_byteIAC, g_byteSB, g_optTermType, 0}
		bReply = append(bReply, []byte(pTelnet.szTermType)...)
		bReply = append(bReply, g_byteIAC, g_byteSE)
		pTelnet.send(bReply)
	}
}

// sendSizeLocked 는 현재 크기를 NAWS 로 보낸다. mtxState 를 잡은 상태에서 부른다.
func (pTelnet *TelnetConn) sendSizeLocked() {
	if !pTelnet.naws || pTelnet.iCols <= 0 || pTelnet.iRows <= 0 {
		return
	}
	bSize := make([]byte, 2)
	bRows := make([]byte, 2)
	binary.BigEndian.PutUint16(bSize, uint16(pTelnet.iCols))
	binary.BigEndian.PutUint16(bRows, uint16(pTelnet.iRows))
	bMsg := []byte{g_byteIAC, g_byteSB, g_optNAWS}
	bMsg = append(bMsg, escapeIAC(bSize)...)
	bMsg = append(bMsg, escapeIAC(bRows)...)
	bMsg = append(bMsg, g_byteIAC, g_byteSE)
	pTelnet.send(bMsg)
}

// escapeIAC 는 바이트 255 를 IAC IAC 로 바꾼다.
func escapeIAC(bIn []byte) []byte {
	var bOut []byte
	for _, byteValue := range bIn {
		bOut = append(bOut, byteValue)
		if byteValue == g_byteIAC {
			bOut = append(bOut, g_byteIAC)
		}
	}
	return bOut
}

func (pTelnet *TelnetConn) send(bData []byte) {
	pTelnet.mtxWrite.Lock()
	defer pTelnet.mtxWrite.Unlock()
	_, _ = pTelnet.conn.Write(bData)
}

// Write 는 입력 바이트를 보낸다. 바이트 255 는 IAC IAC 로 이스케이프한다.
func (pTelnet *TelnetConn) Write(bData []byte) error {
	pTelnet.mtxWrite.Lock()
	defer pTelnet.mtxWrite.Unlock()
	_, errWrite := pTelnet.conn.Write(escapeIAC(bData))
	return errWrite
}

// Resize 는 크기를 기억하고 NAWS 를 보낸다 (크기가 바뀔 때마다).
func (pTelnet *TelnetConn) Resize(iCols int, iRows int) error {
	pTelnet.mtxState.Lock()
	pTelnet.iCols, pTelnet.iRows = iCols, iRows
	pTelnet.sendSizeLocked()
	pTelnet.mtxState.Unlock()
	return nil
}

// Close 는 연결을 닫는다.
func (pTelnet *TelnetConn) Close() error {
	return pTelnet.conn.Close()
}

// String 은 디버그용 표시다.
func (pTelnet *TelnetConn) String() string {
	return fmt.Sprintf("telnet(%s)", pTelnet.conn.RemoteAddr())
}
