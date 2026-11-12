package transport

import (
	"bytes"
	"net"
	"testing"
	"time"
)

// 서버가 DO NAWS 를 보내면 WILL NAWS 와 현재 크기를 보내고, 화면 데이터의 IAC IAC 는 255 한 바이트다 (protocols.md §4).
func TestTelnetNegotiationAndEscaping(t *testing.T) {
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	defer ln.Close()
	chServerGot := make(chan []byte, 8)
	go func() {
		conn, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		defer conn.Close()
		// 서버 → 클라이언트: DO NAWS, 데이터 "hi" + IAC IAC(255)
		_, _ = conn.Write([]byte{255, 253, 31, 'h', 'i', 255, 255})
		// 클라이언트 응답 수집
		bBuf := make([]byte, 256)
		time.Sleep(100 * time.Millisecond)
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _ := conn.Read(bBuf)
		chServerGot <- append([]byte{}, bBuf[:n]...)
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	pConn, errDial := DialTelnet("127.0.0.1", port, "xterm")
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer pConn.Close()
	_ = pConn.Resize(120, 32)

	bBuf := make([]byte, 64)
	nRead, errRead := pConn.Read(bBuf)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if !bytes.Equal(bBuf[:nRead], []byte("hi\xff")) {
		t.Fatalf("화면 데이터: % x", bBuf[:nRead])
	}
	bGot := <-chServerGot
	// WILL NAWS 와 NAWS 크기(120x32)가 포함되어야 한다.
	if !bytes.Contains(bGot, []byte{255, 251, 31}) {
		t.Fatalf("WILL NAWS 응답 없음: % x", bGot)
	}
	if !bytes.Contains(bGot, []byte{255, 250, 31, 0, 120, 0, 32, 255, 240}) {
		t.Fatalf("NAWS 크기 응답 없음: % x", bGot)
	}
}

func TestTelnetWriteEscapesIAC(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	chGot := make(chan []byte, 1)
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		bBuf := make([]byte, 16)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, _ := conn.Read(bBuf)
		chGot <- append([]byte{}, bBuf[:n]...)
	}()
	pConn, errDial := DialTelnet("127.0.0.1", ln.Addr().(*net.TCPAddr).Port, "")
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer pConn.Close()
	if errWrite := pConn.Write([]byte{0x61, 0xff}); errWrite != nil {
		t.Fatal(errWrite)
	}
	if got := <-chGot; !bytes.Equal(got, []byte{0x61, 255, 255}) {
		t.Fatalf("입력 이스케이프: % x", got)
	}
}
