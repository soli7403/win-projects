package sshconn

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// 프록시 경유 접속 시험 (D110).

// startHttpProxy 는 CONNECT 만 받는 가짜 HTTP 프록시다. szUserPass("user:pass")가 있으면 Basic 인증을 요구한다.
// szBanner 가 있으면 200 응답과 같은 쓰기에 붙여 보낸다(터널 데이터가 응답과 함께 도착하는 경우).
func startHttpProxy(t *testing.T, szUserPass string, szBanner string) (string, func() []string) {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var mtxData sync.Mutex
	var slTargets []string
	go func() {
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				defer conn.Close()
				pReader := bufio.NewReader(conn)
				pReq, errReq := http.ReadRequest(pReader)
				if errReq != nil || pReq.Method != http.MethodConnect {
					_, _ = io.WriteString(conn, "HTTP/1.1 405 Method Not Allowed\r\n\r\n")
					return
				}
				if szUserPass != "" && pReq.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(szUserPass)) {
					_, _ = io.WriteString(conn, "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"x\"\r\nContent-Length: 0\r\n\r\n")
					return
				}
				mtxData.Lock()
				slTargets = append(slTargets, pReq.Host)
				mtxData.Unlock()
				if strings.HasPrefix(pReq.Host, "blocked.invalid") {
					_, _ = io.WriteString(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
					return
				}
				outbound, errDial := net.DialTimeout("tcp", pReq.Host, 2*time.Second)
				if errDial != nil {
					_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				defer outbound.Close()
				_, _ = io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n"+szBanner)
				go func() { _, _ = io.Copy(outbound, pReader) }()
				_, _ = io.Copy(conn, outbound)
			}()
		}
	}()
	return ln.Addr().String(), func() []string {
		mtxData.Lock()
		defer mtxData.Unlock()
		return append([]string(nil), slTargets...)
	}
}

func proxyOf(t *testing.T, szType string, szAddr string, szUser string, szPass string) *Proxy {
	t.Helper()
	szHost, szPort, _ := net.SplitHostPort(szAddr)
	var iPort int
	for _, ch := range szPort {
		iPort = iPort*10 + int(ch-'0')
	}
	return &Proxy{Type: szType, Host: szHost, Port: iPort, Username: szUser, Password: szPass}
}

// HTTP 프록시: CONNECT 로 대상까지 이어진다. 응답과 함께 도착한 터널 데이터도 잃지 않는다.
func TestHttpProxyConnect(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy, fnTargets := startHttpProxy(t, "", "SSH-2.0-banner\r\n")
	conn, errDial := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if bGot := readN(t, conn, 16); string(bGot) != "SSH-2.0-banner\r\n" {
		t.Fatalf("응답과 함께 온 데이터를 잃으면 안 됨: %q", string(bGot))
	}
	expectEcho(t, conn, "through-http")
	if slTargets := fnTargets(); len(slTargets) != 1 || !strings.HasPrefix(slTargets[0], "127.0.0.1:") {
		t.Fatalf("CONNECT 대상: %v", slTargets)
	}
}

// HTTP 프록시: 인증이 맞으면 통과, 틀리거나 없으면 ErrProxyAuth, 그 밖의 거절은 상태 줄이 오류에 담긴다.
func TestHttpProxyAuthAndRefusal(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy, _ := startHttpProxy(t, "kai:s3cret", "")
	conn, errOk := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "kai", "s3cret"), "127.0.0.1", iEchoPort, 3*time.Second)
	if errOk != nil {
		t.Fatalf("맞는 인증: %v", errOk)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "auth-ok")
	_ = conn.Close()

	if _, errBad := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "kai", "wrong"), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errBad, ErrProxyAuth) {
		t.Fatalf("틀린 비밀번호는 인증 오류여야 함: %v", errBad)
	}
	if _, errNone := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errNone, ErrProxyAuth) {
		t.Fatalf("인증 없이 가면 인증 오류여야 함: %v", errNone)
	}

	szOpen, _ := startHttpProxy(t, "", "")
	_, errBlocked := dialViaProxy(proxyOf(t, G_cszProxyHttp, szOpen, "", ""), "blocked.invalid", 22, 3*time.Second)
	if errBlocked == nil || !strings.Contains(errBlocked.Error(), "403") {
		t.Fatalf("거절 사유가 오류에 담겨야 함: %v", errBlocked)
	}
}

// SOCKS5 프록시(인증 없음): 이 패키지의 SOCKS 서버(동적 포워딩)를 프록시로 써서 맞물려 확인한다. 이름은 프록시가 푼다.
func TestSocks5ProxyConnect(t *testing.T) {
	iEchoPort := startEcho(t)
	szSocks, fnTargets := startSocks(t)
	conn, errDial := dialViaProxy(proxyOf(t, G_cszProxySocks5, szSocks, "", ""), "localhost", iEchoPort, 3*time.Second)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "through-socks5")
	if slTargets := fnTargets(); len(slTargets) != 1 || !strings.HasPrefix(slTargets[0], "localhost:") {
		t.Fatalf("도메인 이름을 그대로 넘겨야 함: %v", slTargets)
	}
	if _, errUnreach := dialViaProxy(proxyOf(t, G_cszProxySocks5, szSocks, "", ""), "unreachable.invalid", 22, 3*time.Second); errUnreach == nil || !strings.Contains(errUnreach.Error(), "code 4") {
		t.Fatalf("프록시의 거절 코드가 오류에 담겨야 함: %v", errUnreach)
	}
}

// startSocks5Auth 는 사용자 이름·비밀번호 인증(RFC 1929)만 받는 가짜 SOCKS5 프록시다. 인증 뒤 대상에 잇는다.
func startSocks5Auth(t *testing.T, szUser string, szPass string) string {
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
			go func() {
				defer conn.Close()
				bHead := make([]byte, 2)
				if _, errRead := io.ReadFull(conn, bHead); errRead != nil {
					return
				}
				bMethods := make([]byte, bHead[1])
				_, _ = io.ReadFull(conn, bMethods)
				if !strings.ContainsRune(string(bMethods), 2) {
					_, _ = conn.Write([]byte{5, 0xFF})
					return
				}
				_, _ = conn.Write([]byte{5, 2})
				_, _ = io.ReadFull(conn, bHead) // VER ULEN
				bUser := make([]byte, bHead[1])
				_, _ = io.ReadFull(conn, bUser)
				bLen := make([]byte, 1)
				_, _ = io.ReadFull(conn, bLen)
				bPass := make([]byte, bLen[0])
				_, _ = io.ReadFull(conn, bPass)
				if string(bUser) != szUser || string(bPass) != szPass {
					_, _ = conn.Write([]byte{1, 1})
					return
				}
				_, _ = conn.Write([]byte{1, 0})
				pReader := bufio.NewReader(conn)
				_, _ = pReader.Discard(3) // VER CMD RSV
				szTarget, errTarget := "", error(nil)
				bAtyp, _ := pReader.ReadByte()
				if bAtyp == 1 {
					bIp := make([]byte, 4)
					_, _ = io.ReadFull(pReader, bIp)
					szTarget = net.IP(bIp).String()
				} else {
					errTarget = errors.New("test proxy: only IPv4")
				}
				bPort := make([]byte, 2)
				_, _ = io.ReadFull(pReader, bPort)
				if errTarget != nil {
					return
				}
				outbound, errDial := net.DialTimeout("tcp", net.JoinHostPort(szTarget, itoaPort(bPort)), 2*time.Second)
				if errDial != nil {
					_, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				defer outbound.Close()
				// 응답의 바인드 주소를 도메인 이름 형식(ATYP 3)으로 보내, 길이가 다른 응답도 바르게 넘기는지 본다.
				_, _ = conn.Write([]byte{5, 0, 0, 3, 4, 't', 'e', 's', 't', 0, 0})
				go func() { _, _ = io.Copy(outbound, pReader) }()
				_, _ = io.Copy(conn, outbound)
			}()
		}
	}()
	return ln.Addr().String()
}

func itoaPort(bPort []byte) string {
	iPort := int(bPort[0])<<8 | int(bPort[1])
	szOut := ""
	for iPort > 0 {
		szOut = string(rune('0'+iPort%10)) + szOut
		iPort /= 10
	}
	return szOut
}

// SOCKS5 프록시(사용자 이름·비밀번호): 맞으면 통과, 틀리거나 사용자 이름이 없으면 ErrProxyAuth.
func TestSocks5ProxyUserPass(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy := startSocks5Auth(t, "kai", "s3cret")
	conn, errOk := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "kai", "s3cret"), "127.0.0.1", iEchoPort, 3*time.Second)
	if errOk != nil {
		t.Fatalf("맞는 인증: %v", errOk)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "socks5-auth")
	_ = conn.Close()
	if _, errBad := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "kai", "wrong"), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errBad, ErrProxyAuth) {
		t.Fatalf("틀린 비밀번호: %v", errBad)
	}
	if _, errNone := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errNone, ErrProxyAuth) {
		t.Fatalf("사용자 이름 없이: %v", errNone)
	}
}

// 프록시에 닿지 못하면 프록시 주소가 담긴 오류다. 모르는 종류도 오류다. 프록시가 없으면 바로 연결한다.
func TestProxyDialErrorsAndDirect(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	szDead := ln.Addr().String()
	_ = ln.Close()
	if _, errDead := dialViaProxy(proxyOf(t, G_cszProxyHttp, szDead, "", ""), "example.invalid", 22, time.Second); errDead == nil || !strings.Contains(errDead.Error(), "proxy "+szDead) {
		t.Fatalf("프록시 주소가 오류에 담겨야 함: %v", errDead)
	}
	iEchoPort := startEcho(t)
	szProxy, _ := startHttpProxy(t, "", "")
	if _, errType := dialViaProxy(proxyOf(t, "ftp", szProxy, "", ""), "127.0.0.1", iEchoPort, time.Second); errType == nil {
		t.Fatal("모르는 종류는 오류여야 함")
	}
	conn, errDirect := dialDirect(&Target{Host: "127.0.0.1", Port: iEchoPort}, net.JoinHostPort("127.0.0.1", itoaPort([]byte{byte(iEchoPort >> 8), byte(iEchoPort)})))
	if errDirect != nil {
		t.Fatal(errDirect)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "direct")
	_ = conn.Close()
}

// 통합: SSH 접속 전체가 프록시를 거쳐 이뤄진다. 비밀번호를 물어 다시 연결하는 경로(dialNext)도 프록시를 거친다.
// 프록시 인증이 틀리면 안내 문구가 있는 오류다.
func TestSSHConnectThroughProxy(t *testing.T) {
	iPort, fnStop := startServer(t)
	defer fnStop()
	for _, szType := range []string{G_cszProxyHttp, G_cszProxySocks5} {
		var szProxy string
		var fnTargets func() []string
		if szType == G_cszProxyHttp {
			szProxy, fnTargets = startHttpProxy(t, "kai:s3cret", "")
		} else {
			szProxy, fnTargets = startSocks(t)
		}
		tgt := targetFor(iPort, "")
		tgt.Proxy = proxyOf(t, szType, szProxy, "kai", "s3cret")
		var slNotices []string
		conn, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{password: g_szPass, approve: true}, func(szLine string) { slNotices = append(slNotices, szLine) })
		if errConnect != nil {
			t.Fatalf("%s 프록시 경유 접속: %v", szType, errConnect)
		}
		_ = conn.Close()
		// 저장된 비밀번호가 없어 1차 시도가 인증 실패 → 비밀번호를 묻고 새 연결: 프록시를 두 번 거친다.
		if slTargets := fnTargets(); len(slTargets) != 2 {
			t.Fatalf("%s: 두 번의 연결이 모두 프록시를 거쳐야 함: %v", szType, slTargets)
		}
		if len(slNotices) == 0 {
			t.Fatalf("%s: 프록시 경유 안내 줄이 있어야 함", szType)
		}
	}
	szProxy, _ := startHttpProxy(t, "kai:s3cret", "")
	tgt := targetFor(iPort, g_szPass)
	tgt.Proxy = proxyOf(t, G_cszProxyHttp, szProxy, "kai", "wrong")
	if _, errAuth := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{approve: true}, func(string) {}); errAuth == nil || errors.Is(errAuth, ErrProxyAuth) {
		t.Fatalf("프록시 인증 실패는 안내 문구 오류로 바뀌어야 함: %v", errAuth)
	}
}
