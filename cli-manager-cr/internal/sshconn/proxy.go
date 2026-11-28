package sshconn

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 프록시 경유 접속 (protocols.md §2.7, D110).
// 대상(또는 첫 경유 서버)까지의 TCP 연결을 HTTP 프록시(CONNECT) 또는 SOCKS5 프록시를 거쳐 맺는다.

// 프록시 종류 (data/store.md §3 proxy.type).
const (
	G_cszProxyHttp   = "http"
	G_cszProxySocks5 = "socks5"
)

// Proxy 는 프록시 설정 하나다.
type Proxy struct {
	Type     string // http | socks5 | command
	Host     string
	Port     int
	Username string // 비면 인증 없이 시도한다
	Password string
	Command  string // Type 이 command 일 때 실행할 명령 줄 (%h %p %r, D152). 그때 Host·Port 는 쓰지 않는다
}

// ErrProxyAuth 는 프록시가 사용자 이름·비밀번호를 받아들이지 않았을 때의 오류다.
var ErrProxyAuth = errors.New("proxy: authentication failed")

// bufferedConn 은 프록시 응답을 읽다가 미리 읽힌 바이트(SSH 서버의 첫 인사 등)를 잃지 않도록
// 읽기를 bufio.Reader 로 돌린 연결이다.
type bufferedConn struct {
	net.Conn
	pReader *bufio.Reader
}

func (pConn *bufferedConn) Read(bBuf []byte) (int, error) {
	return pConn.pReader.Read(bBuf)
}

// dialDirect 는 tgt 로 가는 첫 TCP 연결을 연다: 프록시가 있으면 프록시를 거치고, 없으면 바로 연결한다.
func dialDirect(tgt *Target, szAddr string) (net.Conn, error) {
	if tgt.Proxy == nil {
		return net.DialTimeout("tcp", szAddr, G_cdurConnectTimeout)
	}
	if tgt.Proxy.Type == G_cszProxyCommand {
		return dialCommand(tgt.Proxy.Command, tgt.Host, tgt.Port, tgt.Username)
	}
	return dialViaProxy(tgt.Proxy, tgt.Host, tgt.Port, G_cdurConnectTimeout)
}

// dialViaProxy 는 프록시에 연결해 대상(szHost:iPort)까지 터널을 연다. 대상 이름은 프록시 쪽에서 푼다.
// durTimeout 은 프록시 연결과 터널 요청 전체의 시한이다.
func dialViaProxy(pProxy *Proxy, szHost string, iPort int, durTimeout time.Duration) (net.Conn, error) {
	szProxyAddr := net.JoinHostPort(pProxy.Host, strconv.Itoa(pProxy.Port))
	conn, errDial := net.DialTimeout("tcp", szProxyAddr, durTimeout)
	if errDial != nil {
		return nil, fmt.Errorf("proxy %s: %w", szProxyAddr, errDial)
	}
	_ = conn.SetDeadline(time.Now().Add(durTimeout))
	var tunnel net.Conn
	var errTunnel error
	switch pProxy.Type {
	case G_cszProxyHttp:
		tunnel, errTunnel = httpConnect(conn, pProxy, szHost, iPort)
	case G_cszProxySocks5:
		tunnel, errTunnel = socks5Connect(conn, pProxy, szHost, iPort)
	default:
		errTunnel = fmt.Errorf("unknown proxy type %q", pProxy.Type)
	}
	if errTunnel != nil {
		_ = conn.Close()
		if errors.Is(errTunnel, ErrProxyAuth) {
			return nil, errTunnel
		}
		return nil, fmt.Errorf("proxy %s: %w", szProxyAddr, errTunnel)
	}
	_ = conn.SetDeadline(time.Time{})
	return tunnel, nil
}

// httpConnect 는 HTTP 프록시에 CONNECT 를 보내 터널을 연다. 사용자 이름이 있으면 Basic 인증을 붙인다.
func httpConnect(conn net.Conn, pProxy *Proxy, szHost string, iPort int) (net.Conn, error) {
	szTarget := net.JoinHostPort(szHost, strconv.Itoa(iPort))
	szRequest := "CONNECT " + szTarget + " HTTP/1.1\r\nHost: " + szTarget + "\r\n"
	if pProxy.Username != "" {
		szToken := base64.StdEncoding.EncodeToString([]byte(pProxy.Username + ":" + pProxy.Password))
		szRequest += "Proxy-Authorization: Basic " + szToken + "\r\n"
	}
	szRequest += "\r\n"
	if _, errWrite := io.WriteString(conn, szRequest); errWrite != nil {
		return nil, errWrite
	}
	pReader := bufio.NewReader(conn)
	pResp, errResp := http.ReadResponse(pReader, &http.Request{Method: http.MethodConnect})
	if errResp != nil {
		return nil, errResp
	}
	// CONNECT 의 2xx 응답에는 본문이 없다. 뒤따르는 바이트는 터널 데이터다.
	if pResp.StatusCode == http.StatusProxyAuthRequired {
		return nil, ErrProxyAuth
	}
	if pResp.StatusCode < 200 || pResp.StatusCode > 299 {
		return nil, fmt.Errorf("CONNECT refused: %s", pResp.Status)
	}
	return &bufferedConn{Conn: conn, pReader: pReader}, nil
}

// socks5Connect 는 SOCKS5 프록시에 CONNECT 를 보내 터널을 연다 (RFC 1928, 사용자 이름·비밀번호 인증은 RFC 1929).
func socks5Connect(conn net.Conn, pProxy *Proxy, szHost string, iPort int) (net.Conn, error) {
	if len(szHost) > 255 || len(pProxy.Username) > 255 || len(pProxy.Password) > 255 {
		return nil, errors.New("socks5: name too long")
	}
	bMethods := []byte{5, 1, 0}
	if pProxy.Username != "" {
		bMethods = []byte{5, 2, 0, 2}
	}
	if _, errWrite := conn.Write(bMethods); errWrite != nil {
		return nil, errWrite
	}
	var arrChoice [2]byte
	if _, errRead := io.ReadFull(conn, arrChoice[:]); errRead != nil {
		return nil, errRead
	}
	if arrChoice[0] != 5 {
		return nil, errors.New("socks5: not a SOCKS5 proxy")
	}
	switch arrChoice[1] {
	case 0:
	case 2:
		if pProxy.Username == "" {
			return nil, ErrProxyAuth
		}
		bAuth := append([]byte{1, byte(len(pProxy.Username))}, pProxy.Username...)
		bAuth = append(append(bAuth, byte(len(pProxy.Password))), pProxy.Password...)
		if _, errWrite := conn.Write(bAuth); errWrite != nil {
			return nil, errWrite
		}
		var arrAuth [2]byte
		if _, errRead := io.ReadFull(conn, arrAuth[:]); errRead != nil {
			return nil, errRead
		}
		if arrAuth[1] != 0 {
			return nil, ErrProxyAuth
		}
	default:
		return nil, ErrProxyAuth // 0xFF: 내놓은 방법을 프록시가 받지 않는다
	}
	bRequest := []byte{5, 1, 0}
	if ip := net.ParseIP(szHost); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			bRequest = append(append(bRequest, 1), ip4...)
		} else {
			bRequest = append(append(bRequest, 4), ip.To16()...)
		}
	} else {
		bRequest = append(append(bRequest, 3, byte(len(szHost))), szHost...)
	}
	bRequest = binary.BigEndian.AppendUint16(bRequest, uint16(iPort))
	if _, errWrite := conn.Write(bRequest); errWrite != nil {
		return nil, errWrite
	}
	var arrReply [4]byte // VER REP RSV ATYP
	if _, errRead := io.ReadFull(conn, arrReply[:]); errRead != nil {
		return nil, errRead
	}
	if arrReply[1] != 0 {
		return nil, fmt.Errorf("socks5: connect refused (code %d)", arrReply[1])
	}
	// 응답의 바인드 주소는 쓰지 않지만 길이만큼 읽어 버려야 뒤의 터널 데이터와 섞이지 않는다.
	iSkip := 0
	switch arrReply[3] {
	case 1:
		iSkip = 4
	case 4:
		iSkip = 16
	case 3:
		var arrLength [1]byte
		if _, errRead := io.ReadFull(conn, arrLength[:]); errRead != nil {
			return nil, errRead
		}
		iSkip = int(arrLength[0])
	default:
		return nil, errors.New("socks5: malformed reply")
	}
	if _, errRead := io.CopyN(io.Discard, conn, int64(iSkip+2)); errRead != nil {
		return nil, errRead
	}
	return conn, nil
}
