package sshconn

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

// 동적 포워딩(SOCKS 프록시 서버) (protocols.md §2.5, D109).
// 이 PC 의 포트에서 SOCKS4·SOCKS4a·SOCKS5(인증 없음, CONNECT 만) 요청을 받아, 요청한 대상까지 SSH 연결을 거쳐 잇는다.

const (
	// g_cdurSocksHandshake 는 SOCKS 요청을 다 받을 때까지의 시한이다. 접속만 하고 말이 없는 연결을 정리한다.
	g_cdurSocksHandshake = 10 * time.Second
	// g_ciSocksNameMax 는 SOCKS4 의 사용자 이름·도메인 이름으로 받아들이는 최대 길이다.
	g_ciSocksNameMax = 255
)

// SOCKS5 응답 코드 (RFC 1928 §6).
const (
	g_cbSocks5Ok              = 0x00
	g_cbSocks5Failure         = 0x01
	g_cbSocks5HostUnreach     = 0x04
	g_cbSocks5CmdUnsupported  = 0x07
	g_cbSocks5AddrUnsupported = 0x08
)

// errSocksRequest 는 SOCKS 요청 모양이 틀렸거나 지원하지 않는 요청일 때의 오류다.
var errSocksRequest = errors.New("socks: bad or unsupported request")

// serveSocks 는 ln 으로 들어오는 연결마다 SOCKS 요청을 처리한다. fnDial 은 대상까지의 연결을 만든다
// (동적 포워딩에서는 SSH 연결의 Dial). ln 을 닫으면 끝난다.
func serveSocks(ln net.Listener, fnDial func(szNetwork string, szAddr string) (net.Conn, error)) {
	for {
		inbound, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		go handleSocks(inbound, fnDial)
	}
}

// handleSocks 는 연결 하나의 SOCKS 요청을 읽고 대상에 이어 양방향으로 복사한다.
func handleSocks(inbound net.Conn, fnDial func(string, string) (net.Conn, error)) {
	defer inbound.Close()
	_ = inbound.SetDeadline(time.Now().Add(g_cdurSocksHandshake))
	pReader := bufio.NewReader(inbound)
	bVersion, errVersion := pReader.ReadByte()
	if errVersion != nil {
		return
	}
	var szTarget string
	var fnReply func(bOk bool, bCode byte)
	var errRequest error
	switch bVersion {
	case 5:
		szTarget, errRequest = readSocks5(pReader, inbound)
		fnReply = func(bOk bool, bCode byte) {
			if bOk {
				bCode = g_cbSocks5Ok
			}
			_, _ = inbound.Write([]byte{5, bCode, 0, 1, 0, 0, 0, 0, 0, 0})
		}
	case 4:
		szTarget, errRequest = readSocks4(pReader)
		fnReply = func(bOk bool, bCode byte) {
			bReply := byte(0x5B) // 거절
			if bOk {
				bReply = 0x5A // 허용
			}
			_, _ = inbound.Write([]byte{0, bReply, 0, 0, 0, 0, 0, 0})
		}
	default:
		return
	}
	if errRequest != nil {
		var errCode *socksCodeError
		if errors.As(errRequest, &errCode) {
			fnReply(false, errCode.bCode)
		}
		return
	}
	outbound, errDial := fnDial("tcp", szTarget)
	if errDial != nil {
		fnReply(false, g_cbSocks5HostUnreach)
		return
	}
	defer outbound.Close()
	fnReply(true, 0)
	_ = inbound.SetDeadline(time.Time{})
	// 요청과 함께 미리 들어온 데이터는 pReader 가 들고 있으므로 inbound 가 아니라 pReader 에서 읽는다.
	go func() { _, _ = io.Copy(outbound, pReader); _ = outbound.Close() }()
	_, _ = io.Copy(inbound, outbound)
}

// socksCodeError 는 클라이언트에 SOCKS5 응답 코드로 알릴 수 있는 요청 오류다.
type socksCodeError struct {
	bCode byte
}

func (pErr *socksCodeError) Error() string {
	return "socks: request refused (code " + strconv.Itoa(int(pErr.bCode)) + ")"
}

// readSocks5 는 버전 바이트 뒤의 SOCKS5 인사와 요청을 읽고 대상 "호스트:포트" 를 돌려준다 (RFC 1928).
// 인증 없음(0x00)만 받는다. 명령은 CONNECT(1)만 받는다.
func readSocks5(pReader *bufio.Reader, wReply io.Writer) (string, error) {
	nMethods, errCount := pReader.ReadByte()
	if errCount != nil {
		return "", errCount
	}
	bMethods := make([]byte, nMethods)
	if _, errMethods := io.ReadFull(pReader, bMethods); errMethods != nil {
		return "", errMethods
	}
	bNoAuth := false
	for _, bMethod := range bMethods {
		if bMethod == 0 {
			bNoAuth = true
		}
	}
	if !bNoAuth {
		_, _ = wReply.Write([]byte{5, 0xFF}) // 받아들일 인증 방법 없음
		return "", errSocksRequest
	}
	if _, errWrite := wReply.Write([]byte{5, 0}); errWrite != nil {
		return "", errWrite
	}
	var arrHead [4]byte // VER CMD RSV ATYP
	if _, errHead := io.ReadFull(pReader, arrHead[:]); errHead != nil {
		return "", errHead
	}
	if arrHead[0] != 5 {
		return "", errSocksRequest
	}
	var szHost string
	switch arrHead[3] {
	case 1: // IPv4
		var arrIp [4]byte
		if _, errIp := io.ReadFull(pReader, arrIp[:]); errIp != nil {
			return "", errIp
		}
		szHost = net.IP(arrIp[:]).String()
	case 3: // 도메인 이름
		nLength, errLength := pReader.ReadByte()
		if errLength != nil {
			return "", errLength
		}
		bName := make([]byte, nLength)
		if _, errName := io.ReadFull(pReader, bName); errName != nil {
			return "", errName
		}
		szHost = string(bName)
	case 4: // IPv6
		var arrIp [16]byte
		if _, errIp := io.ReadFull(pReader, arrIp[:]); errIp != nil {
			return "", errIp
		}
		szHost = net.IP(arrIp[:]).String()
	default:
		return "", &socksCodeError{bCode: g_cbSocks5AddrUnsupported}
	}
	var arrPort [2]byte
	if _, errPort := io.ReadFull(pReader, arrPort[:]); errPort != nil {
		return "", errPort
	}
	if arrHead[1] != 1 {
		return "", &socksCodeError{bCode: g_cbSocks5CmdUnsupported}
	}
	if szHost == "" {
		return "", &socksCodeError{bCode: g_cbSocks5Failure}
	}
	return net.JoinHostPort(szHost, strconv.Itoa(int(binary.BigEndian.Uint16(arrPort[:])))), nil
}

// readSocks4 는 버전 바이트 뒤의 SOCKS4·SOCKS4a 요청을 읽고 대상 "호스트:포트" 를 돌려준다.
// 주소가 0.0.0.x(x≠0)이면 SOCKS4a: 사용자 이름 뒤에 도메인 이름이 온다.
func readSocks4(pReader *bufio.Reader) (string, error) {
	var arrHead [7]byte // CMD PORT(2) IP(4)
	if _, errHead := io.ReadFull(pReader, arrHead[:]); errHead != nil {
		return "", errHead
	}
	if _, errUser := readSocksCString(pReader); errUser != nil {
		return "", errUser
	}
	szHost := net.IP(arrHead[3:7]).String()
	if arrHead[3] == 0 && arrHead[4] == 0 && arrHead[5] == 0 && arrHead[6] != 0 {
		szName, errName := readSocksCString(pReader)
		if errName != nil {
			return "", errName
		}
		szHost = szName
	}
	if arrHead[0] != 1 || szHost == "" {
		return "", &socksCodeError{bCode: g_cbSocks5CmdUnsupported}
	}
	return net.JoinHostPort(szHost, strconv.Itoa(int(binary.BigEndian.Uint16(arrHead[1:3])))), nil
}

// readSocksCString 은 0 으로 끝나는 문자열을 읽는다. 너무 길면 오류다.
func readSocksCString(pReader *bufio.Reader) (string, error) {
	var bOut []byte
	for {
		bOne, errRead := pReader.ReadByte()
		if errRead != nil {
			return "", errRead
		}
		if bOne == 0 {
			return string(bOut), nil
		}
		if len(bOut) >= g_ciSocksNameMax {
			return "", errSocksRequest
		}
		bOut = append(bOut, bOne)
	}
}
