package sshconn

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

// 동적 포워딩(SOCKS) 시험 (D109). SSH 대신 일반 TCP 로 대상에 잇는 fnDial 을 넣어 SOCKS 처리만 확인한다.

// startEcho 는 받은 것을 그대로 돌려주는 서버를 띄우고 포트를 돌려준다.
func startEcho(t *testing.T) int {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() { _, _ = io.Copy(conn, conn); _ = conn.Close() }()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// startSocks 는 SOCKS 서버를 띄우고 주소와, 대상 요청 기록을 돌려준다.
func startSocks(t *testing.T) (string, func() []string) {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var mtxData sync.Mutex
	var slTargets []string
	go serveSocks(ln, func(szNetwork string, szAddr string) (net.Conn, error) {
		mtxData.Lock()
		slTargets = append(slTargets, szAddr)
		mtxData.Unlock()
		if szHost, _, _ := net.SplitHostPort(szAddr); szHost == "unreachable.invalid" {
			return nil, errors.New("no route")
		}
		return net.DialTimeout(szNetwork, szAddr, 2*time.Second)
	})
	return ln.Addr().String(), func() []string {
		mtxData.Lock()
		defer mtxData.Unlock()
		return append([]string(nil), slTargets...)
	}
}

func dialSocks(t *testing.T, szAddr string) net.Conn {
	t.Helper()
	conn, errDial := net.DialTimeout("tcp", szAddr, 2*time.Second)
	if errDial != nil {
		t.Fatal(errDial)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readN(t *testing.T, conn net.Conn, iCount int) []byte {
	t.Helper()
	bBuf := make([]byte, iCount)
	if _, errRead := io.ReadFull(conn, bBuf); errRead != nil {
		t.Fatalf("응답 읽기: %v", errRead)
	}
	return bBuf
}

func portBytes(iPort int) []byte {
	return []byte{byte(iPort >> 8), byte(iPort)}
}

func expectEcho(t *testing.T, conn net.Conn, szText string) {
	t.Helper()
	if _, errWrite := conn.Write([]byte(szText)); errWrite != nil {
		t.Fatal(errWrite)
	}
	if bGot := readN(t, conn, len(szText)); string(bGot) != szText {
		t.Fatalf("대상까지 이어져야 함: %q", string(bGot))
	}
}

// SOCKS5: IPv4 주소, 도메인 이름, IPv6 주소로 CONNECT 하면 대상까지 이어진다.
func TestSocks5Connect(t *testing.T) {
	iEchoPort := startEcho(t)
	szSocks, fnTargets := startSocks(t)

	conn := dialSocks(t, szSocks)
	_, _ = conn.Write([]byte{5, 2, 2, 0}) // 방법 2개: 사용자/비밀번호, 인증 없음
	if bGot := readN(t, conn, 2); !bytes.Equal(bGot, []byte{5, 0}) {
		t.Fatalf("인증 없음을 골라야 함: %v", bGot)
	}
	_, _ = conn.Write(append([]byte{5, 1, 0, 1, 127, 0, 0, 1}, portBytes(iEchoPort)...))
	if bGot := readN(t, conn, 10); bGot[1] != 0 {
		t.Fatalf("성공 응답이어야 함: %v", bGot)
	}
	expectEcho(t, conn, "ipv4")

	conn2 := dialSocks(t, szSocks)
	bReq := append([]byte{5, 1, 0, 5, 1, 0, 3, 9}, []byte("localhost")...)
	_, _ = conn2.Write(append(bReq, portBytes(iEchoPort)...)) // 인사와 요청을 한 번에 보내도 된다
	if bGot := readN(t, conn2, 12); bGot[1] != 0 || bGot[3] != 0 {
		t.Fatalf("도메인 이름 요청: %v", bGot)
	}
	expectEcho(t, conn2, "domain")

	slTargets := fnTargets()
	if len(slTargets) != 2 || slTargets[0] != "127.0.0.1:"+strconv.Itoa(iEchoPort) || slTargets[1] != "localhost:"+strconv.Itoa(iEchoPort) {
		t.Fatalf("대상 주소: %v", slTargets)
	}
}

// SOCKS5: 인증 없음을 내놓지 않으면 0xFF, CONNECT 가 아닌 명령은 07, 닿지 않는 대상은 04 로 답한다.
func TestSocks5Refusals(t *testing.T) {
	szSocks, _ := startSocks(t)

	conn := dialSocks(t, szSocks)
	_, _ = conn.Write([]byte{5, 1, 2})
	if bGot := readN(t, conn, 2); !bytes.Equal(bGot, []byte{5, 0xFF}) {
		t.Fatalf("받아들일 방법 없음: %v", bGot)
	}

	conn2 := dialSocks(t, szSocks)
	_, _ = conn2.Write([]byte{5, 1, 0})
	_ = readN(t, conn2, 2)
	_, _ = conn2.Write([]byte{5, 2, 0, 1, 127, 0, 0, 1, 0, 80}) // BIND
	if bGot := readN(t, conn2, 10); bGot[1] != g_cbSocks5CmdUnsupported {
		t.Fatalf("BIND 는 지원하지 않음: %v", bGot)
	}

	conn3 := dialSocks(t, szSocks)
	_, _ = conn3.Write([]byte{5, 1, 0})
	_ = readN(t, conn3, 2)
	szName := "unreachable.invalid"
	_, _ = conn3.Write(append(append([]byte{5, 1, 0, 3, byte(len(szName))}, szName...), 0, 80))
	if bGot := readN(t, conn3, 10); bGot[1] != g_cbSocks5HostUnreach {
		t.Fatalf("닿지 않는 대상: %v", bGot)
	}
}

// SOCKS4 와 SOCKS4a(도메인 이름)도 받는다.
func TestSocks4And4a(t *testing.T) {
	iEchoPort := startEcho(t)
	szSocks, fnTargets := startSocks(t)

	conn := dialSocks(t, szSocks)
	bReq := append([]byte{4, 1}, portBytes(iEchoPort)...)
	_, _ = conn.Write(append(append(bReq, 127, 0, 0, 1), []byte("user\x00")...))
	if bGot := readN(t, conn, 8); bGot[0] != 0 || bGot[1] != 0x5A {
		t.Fatalf("SOCKS4 허용 응답: %v", bGot)
	}
	expectEcho(t, conn, "v4")

	conn2 := dialSocks(t, szSocks)
	bReq2 := append([]byte{4, 1}, portBytes(iEchoPort)...)
	_, _ = conn2.Write(append(append(bReq2, 0, 0, 0, 1), []byte("\x00localhost\x00")...))
	if bGot := readN(t, conn2, 8); bGot[1] != 0x5A {
		t.Fatalf("SOCKS4a 허용 응답: %v", bGot)
	}
	expectEcho(t, conn2, "v4a")

	if slTargets := fnTargets(); len(slTargets) != 2 || slTargets[1] != "localhost:"+strconv.Itoa(iEchoPort) {
		t.Fatalf("대상 주소: %v", slTargets)
	}

	conn3 := dialSocks(t, szSocks)
	_, _ = conn3.Write([]byte{4, 2, 0, 80, 127, 0, 0, 1, 0}) // BIND
	if bGot := readN(t, conn3, 8); bGot[1] != 0x5B {
		t.Fatalf("SOCKS4 BIND 는 거절: %v", bGot)
	}
}

// SOCKS 가 아닌 첫 바이트면 응답 없이 닫는다.
func TestSocksUnknownVersionCloses(t *testing.T) {
	szSocks, _ := startSocks(t)
	conn := dialSocks(t, szSocks)
	_, _ = conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	if bGot, _ := io.ReadAll(conn); len(bGot) != 0 {
		t.Fatalf("응답 없이 닫아야 함: %q", string(bGot))
	}
}
