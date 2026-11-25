package transport

import (
	"dariterm/internal/i18n"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// useKorean 은 시험 동안 문구 언어를 한국어로 맞춘다.
func useKorean(t *testing.T) {
	t.Helper()
	szBefore := i18n.Lang()
	i18n.SetLang("ko")
	t.Cleanup(func() { i18n.SetLang(szBefore) })
}

// dialTelnetTo 는 fnServer 가 받은 연결을 다루는 가짜 서버에 텔넷으로 붙는다.
func dialTelnetTo(t *testing.T, fnServer func(conn net.Conn)) *TelnetConn {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		fnServer(conn)
	}()
	pConn, errDial := DialTelnet("127.0.0.1", ln.Addr().(*net.TCPAddr).Port, "xterm")
	if errDial != nil {
		t.Fatal(errDial)
	}
	t.Cleanup(func() { _ = pConn.Close() })
	return pConn
}

// readUntilError 는 오류가 날 때까지 읽고 그 오류를 돌려준다.
func readUntilError(t *testing.T, fnRead func([]byte) (int, error)) error {
	t.Helper()
	chErr := make(chan error, 1)
	go func() {
		bBuf := make([]byte, 256)
		for {
			if _, errRead := fnRead(bBuf); errRead != nil {
				chErr <- errRead
				return
			}
		}
	}()
	select {
	case errRead := <-chErr:
		return errRead
	case <-time.After(3 * time.Second):
		t.Fatal("읽기가 끝나지 않음")
		return nil
	}
}

// 서버가 연결을 정상적으로 닫으면(FIN) 이유는 "서버가 연결을 닫았습니다"다 (KI-009, D127).
func TestTelnetClosedByPeerReason(t *testing.T) {
	useKorean(t)
	pConn := dialTelnetTo(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte("bye"))
		_ = conn.Close()
	})
	errRead := readUntilError(t, pConn.Read)
	if !errors.Is(errRead, ErrRemoteClosed) {
		t.Fatalf("끊김 종류는 ErrRemoteClosed 여야 함: %v", errRead)
	}
	if errRead.Error() != "서버가 연결을 닫았습니다" {
		t.Fatalf("이유: %q", errRead.Error())
	}
}

// 서버가 연결을 강제로 끊으면(RST) 이유에 OS 가 말한 까닭이 들어가고, 주소 머리("read tcp …")는 들어가지 않는다.
func TestTelnetResetReasonHasCause(t *testing.T) {
	useKorean(t)
	chReset := make(chan struct{})
	pConn := dialTelnetTo(t, func(conn net.Conn) {
		<-chReset
		if pTcp, bOk := conn.(*net.TCPConn); bOk {
			_ = pTcp.SetLinger(0) // 닫을 때 FIN 대신 RST 를 보낸다
		}
		_ = conn.Close()
	})
	// 서버가 읽지 않은 데이터를 남긴 채 닫게 해 RST 가 확실히 나가게 한다.
	if errWrite := pConn.Write([]byte("x")); errWrite != nil {
		t.Fatal(errWrite)
	}
	time.Sleep(50 * time.Millisecond)
	close(chReset)
	errRead := readUntilError(t, pConn.Read)
	if !errors.Is(errRead, ErrRemoteClosed) {
		t.Fatalf("끊김 종류는 ErrRemoteClosed 여야 함: %v", errRead)
	}
	szReason := errRead.Error()
	t.Logf("이유: %s", szReason) // OS 마다 글이 다르다
	if !strings.HasPrefix(szReason, "네트워크 오류 (") || !strings.HasSuffix(szReason, ")") || len(szReason) <= len("네트워크 오류 ()") {
		t.Fatalf("이유에 까닭이 들어가야 함: %q", szReason)
	}
	if strings.Contains(szReason, "127.0.0.1") || strings.Contains(szReason, "read tcp") {
		t.Fatalf("주소 머리는 빼야 함: %q", szReason)
	}
}

// 이쪽에서 닫은 뒤의 읽기 오류에는 까닭을 붙이지 않는다(전과 같은 문구).
func TestTelnetLocalCloseHasNoCause(t *testing.T) {
	useKorean(t)
	chDone := make(chan struct{})
	pConn := dialTelnetTo(t, func(conn net.Conn) {
		<-chDone
		_ = conn.Close()
	})
	defer close(chDone)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = pConn.Close()
	}()
	errRead := readUntilError(t, pConn.Read)
	if errRead != ErrRemoteClosed {
		t.Fatalf("까닭 없는 ErrRemoteClosed 여야 함: %v", errRead)
	}
	if errRead.Error() != "연결이 종료되었습니다" {
		t.Fatalf("이유: %q", errRead.Error())
	}
}

// brokenSerialPort 는 읽기·쓰기에서 정해 둔 오류를 내는 포트다(장치가 뽑힌 상황 흉내).
type brokenSerialPort struct {
	*fakeSerialPort
	errRead  error
	errWrite error
}

func (pBroken *brokenSerialPort) Read(bBuf []byte) (int, error) {
	if pBroken.errRead != nil {
		return 0, pBroken.errRead
	}
	return pBroken.fakeSerialPort.Read(bBuf)
}

func (pBroken *brokenSerialPort) Write(bData []byte) (int, error) {
	if pBroken.errWrite != nil {
		return 0, pBroken.errWrite
	}
	return pBroken.fakeSerialPort.Write(bData)
}

// 장치 오류로 읽기·쓰기가 끝나면 이유에 그 오류가 들어간다 (KI-009, D127).
func TestSerialClosedReasonHasCause(t *testing.T) {
	useKorean(t)
	errDevice := errors.New("A device attached to the system is not functioning.")
	pPort := &brokenSerialPort{fakeSerialPort: &fakeSerialPort{chIn: make(chan []byte, 1)}, errRead: errDevice, errWrite: errDevice}
	pSerial := &SerialConn{port: pPort, opts: SerialOptions{}}
	const szWant = "포트가 닫혔습니다 (A device attached to the system is not functioning.)"

	errRead := readUntilError(t, pSerial.Read)
	if !errors.Is(errRead, ErrPortClosed) {
		t.Fatalf("끊김 종류는 ErrPortClosed 여야 함: %v", errRead)
	}
	if errRead.Error() != szWant {
		t.Fatalf("읽기 이유: %q", errRead.Error())
	}
	errWrite := pSerial.Write([]byte("a"))
	if !errors.Is(errWrite, ErrPortClosed) || errWrite.Error() != szWant {
		t.Fatalf("쓰기 이유: %v", errWrite)
	}
}

// 이쪽에서 포트를 닫은 뒤의 오류에는 까닭을 붙이지 않는다(전과 같은 문구).
func TestSerialLocalCloseHasNoCause(t *testing.T) {
	useKorean(t)
	pSerial := &SerialConn{port: &fakeSerialPort{chIn: make(chan []byte, 1)}, opts: SerialOptions{}}
	if errClose := pSerial.Close(); errClose != nil {
		t.Fatal(errClose)
	}
	errRead := readUntilError(t, pSerial.Read)
	if errRead != ErrPortClosed || errRead.Error() != "포트가 닫혔습니다" {
		t.Fatalf("까닭 없는 ErrPortClosed 여야 함: %v", errRead)
	}
}

// 문구는 Error() 를 부를 때의 언어를 따른다(만들 때 굳지 않는다).
func TestClosedErrorFollowsLanguage(t *testing.T) {
	useKorean(t)
	errClosed := serialClosed(errors.New("gone"), false)
	if errClosed.Error() != "포트가 닫혔습니다 (gone)" {
		t.Fatalf("ko: %q", errClosed.Error())
	}
	i18n.SetLang("en")
	if errClosed.Error() != "The port was closed (gone)" {
		t.Fatalf("en: %q", errClosed.Error())
	}
}
