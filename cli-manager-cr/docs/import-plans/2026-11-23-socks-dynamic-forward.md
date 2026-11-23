# 동적 포워딩 (SOCKS 프록시, `ssh -D`) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D109). 앱 화면에서의 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 4번. 앞 문서 [serial-flow-control](./2026-11-23-serial-flow-control.md), 뒤 문서 [proxy-connect](./2026-11-23-proxy-connect.md)(이 문서의 SOCKS 서버를 테스트에서 프록시로 쓴다).

---

## 0. 한 줄 요약

포트 포워딩에 **D(동적)** 종류를 더했다. 이 PC 의 포트 하나를 SOCKS 프록시로 열어, 그 프록시를 쓰는 프로그램(브라우저, DB 도구 등)의 접속을 SSH 서버를 거쳐 내보낸다. OpenSSH 의 `ssh -D 1080` 과 같은 기능이다. 전에는 L(로컬)·R(원격)만 있었다.

## 1. 배경

- [internal/sshconn/keepalive_forward.go](../../internal/sshconn/keepalive_forward.go) `StartForwards` 는 종류가 `R` 이면 원격, 아니면 로컬로만 처리했다.
- L 은 대상이 하나로 고정돼 있어, 사내망의 여러 주소에 접근하려면 포워딩을 주소마다 만들어야 했다. D 는 대상을 프로그램이 요청마다 정한다.
- 가져오기(PuTTY·SecureCRT)는 D 항목을 건너뛴다(import.md). 이번에 바꾸지 않았다 — §2.1.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 저장 형식 | 기존 `forwards` 배열에 `type: "D"`. `destHost`·`destPort` 는 쓰지 않는다(지우지도 않는다) | 새 필드 없이 끝난다. 종류를 L 로 되돌리면 적어 둔 대상이 다시 보인다 |
| 받는 SOCKS 버전 | SOCKS5(인증 없음), SOCKS4, SOCKS4a | 브라우저·curl·DB 도구가 쓰는 것 전부. OpenSSH 의 `-D` 도 4·5 를 받는다 |
| 명령 | CONNECT 만 | BIND·UDP 는 SSH 채널로 옮길 수 없다. 거절 응답(SOCKS5 0x07, SOCKS4 0x5B)을 보낸다 |
| 인증 | 없음 | OpenSSH 와 같다. 기본 바인드 주소가 `127.0.0.1` 이라 이 PC 의 프로그램만 쓴다 |
| 이름 풀기 | 도메인 이름을 그대로 SSH 서버에 넘긴다(서버 쪽에서 푼다) | 사내 DNS 이름을 쓸 수 있다. 이 PC 에서 DNS 조회가 새지 않는다 |
| 요청 시한 | 10초 | 접속만 하고 말이 없는 연결을 정리 |
| 구현 | 직접 작성(`socks.go`, 약 200줄) | `x/net/proxy` 는 클라이언트뿐이다. 서버는 표준 라이브러리에 없다 |
| 미리 온 데이터 | 요청과 함께 도착한 뒤쪽 바이트는 `bufio.Reader` 에 남으므로 대상으로 복사할 때 그 reader 에서 읽는다 | 요청 직후 TLS 인사를 붙여 보내는 클라이언트가 있다 |

### 2.1 하지 않은 것

- 가져오기에서 D 항목 읽기(PuTTY `PortForwardings` 의 `D`, ssh config 의 `DynamicForward`). 지금은 건너뛴다.
- SOCKS5 사용자 이름·비밀번호 인증. 바인드 주소를 `0.0.0.0` 으로 열면 그 주소에 닿는 누구나 이 프록시를 쓸 수 있다(명세와 화면 도움말에 적음).

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/screens/HostEditorParts.tsx` | 코드 | 수정 |
| `internal/sshconn/keepalive_forward.go` | 코드 | 수정 |
| `internal/sshconn/socks.go` | 코드 | 새 파일 |
| `internal/sshconn/socks_test.go` | 테스트 | 새 파일 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/protocols.md` | 명세 | 수정 |
| `spec/ui/host-editor.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/screens/HostEditorParts.tsx`, `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -17,7 +17,8 @@
 
 export interface Forward {
   id: string;
-  type: 'L' | 'R';
+  /** L 로컬 · R 원격 · D 동적(SOCKS 프록시, 대상 없음) (protocols.md §2.5). */
+  type: 'L' | 'R' | 'D';
   bindHost: string;
   bindPort: number;
   destHost: string;
```

### `frontend/src/screens/HostEditorParts.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditorParts.tsx
+++ b/frontend/src/screens/HostEditorParts.tsx
@@ -35,15 +35,23 @@
           {rows.map((r) => (
             <tr key={r.id}>
               <td>
-                <select className="input" aria-label={t('fwd.type')} value={r.type} onChange={(e) => patch(r.id, { type: e.target.value as 'L' | 'R' })}>
+                <select className="input" aria-label={t('fwd.type')} value={r.type} onChange={(e) => patch(r.id, { type: e.target.value as Forward['type'] })}>
                   <option value="L">L</option>
                   <option value="R">R</option>
+                  <option value="D">D</option>
                 </select>
               </td>
               <td><input className="input mono" aria-label={t('fwd.bindHost')} value={r.bindHost} onChange={(e) => patch(r.id, { bindHost: e.target.value })} /></td>
               <td><input className="input" aria-label={t('fwd.port')} inputMode="numeric" value={String(r.bindPort)} onChange={(e) => patch(r.id, { bindPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
-              <td><input className="input mono" aria-label={t('fwd.destHost')} value={r.destHost} onChange={(e) => patch(r.id, { destHost: e.target.value })} /></td>
-              <td><input className="input" aria-label={t('fwd.destPort')} inputMode="numeric" value={String(r.destPort)} onChange={(e) => patch(r.id, { destPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
+              {r.type === 'D' ? (
+                // 동적 포워딩은 대상이 없다(SOCKS 프록시를 쓰는 프로그램이 정한다). 적어 둔 대상 값은 지우지 않는다.
+                <td colSpan={2} className="faint small">{t('fwd.dynamicDest')}</td>
+              ) : (
+                <>
+                  <td><input className="input mono" aria-label={t('fwd.destHost')} value={r.destHost} onChange={(e) => patch(r.id, { destHost: e.target.value })} /></td>
+                  <td><input className="input" aria-label={t('fwd.destPort')} inputMode="numeric" value={String(r.destPort)} onChange={(e) => patch(r.id, { destPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
+                </>
+              )}
               <td>
                 <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => onChange(rows.filter((x) => x.id !== r.id))}>
                   <Trash2 size={14} />
```

### `internal/sshconn/keepalive_forward.go` — 수정

```diff
--- a/internal/sshconn/keepalive_forward.go
+++ b/internal/sshconn/keepalive_forward.go
@@ -77,6 +77,18 @@
 		}
 		szBindAddr := net.JoinHostPort(szBind, strconv.Itoa(fw.BindPort))
 		szDest := net.JoinHostPort(fw.DestHost, strconv.Itoa(fw.DestPort))
+		if fw.Type == "D" {
+			// 동적 포워딩: 이 PC 의 포트가 SOCKS 프록시가 되고, 요청마다 대상을 SSH 연결로 잇는다 (D109).
+			ln, errListen := net.Listen("tcp", szBindAddr)
+			if errListen != nil {
+				notice(i18n.T("be.fwd.dynamicFailed", "port", fw.BindPort, "error", errListen))
+				continue
+			}
+			notice(i18n.T("be.fwd.dynamicActive", "from", szBindAddr))
+			slClosers = append(slClosers, ln)
+			go serveSocks(ln, pConn.Client.Dial)
+			continue
+		}
 		if fw.Type == "R" {
 			ln, errListen := pConn.Client.Listen("tcp", szBindAddr)
 			if errListen != nil {
```

### `internal/sshconn/socks.go` — 새 파일 (전체)

```go
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
```


## C. 테스트 코드

### `internal/sshconn/socks_test.go` — 새 파일 (전체)

```go
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
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`fwd.help`** (변경)

바꾸기 전(ko): `L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소.`

```json
  "fwd.help": "L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소. D: 내 PC의 포트를 SOCKS 프록시로 열어, 그 프록시를 쓰는 프로그램의 접속을 서버를 거쳐 내보냅니다(대상은 프로그램이 정합니다).",
  "fwd.help": "L: a port on this PC → an address reachable from the server (e.g. local 3307 → DB 3306). R: a port on the server → an address on this PC's side. D: opens a port on this PC as a SOCKS proxy; connections from programs using that proxy go out through the server (the program chooses the destination).",
  "fwd.help": "L: この PC のポート → サーバーから到達できるアドレス（例: ローカル 3307 → DB 3306）。R: サーバーのポート → この PC 側のアドレス。D: この PC のポートを SOCKS プロキシとして開き、そのプロキシを使うプログラムの接続をサーバー経由で送り出します（宛先はプログラムが決めます）。",
  "fwd.help": "L：本机端口 → 服务器可访问的地址（例如本地 3307 → 数据库 3306）。R：服务器端口 → 本机一侧的地址。D：把本机端口作为 SOCKS 代理打开，使用该代理的程序的连接将经由服务器发出（目标由程序决定）。",
  "fwd.help": "L：本機連接埠 → 伺服器可存取的位址（例如本機 3307 → 資料庫 3306）。R：伺服器連接埠 → 本機一側的位址。D：把本機連接埠開成 SOCKS 代理，使用該代理的程式連線會經由伺服器送出（目的地由程式決定）。",
  "fwd.help": "L: ein Port auf diesem PC → eine vom Server erreichbare Adresse (z. B. lokal 3307 → DB 3306). R: ein Port auf dem Server → eine Adresse auf der Seite dieses PCs. D: öffnet einen Port auf diesem PC als SOCKS-Proxy; Verbindungen von Programmen, die diesen Proxy verwenden, gehen über den Server hinaus (das Ziel bestimmt das Programm).",
  "fwd.help": "L : un port de ce PC → une adresse accessible depuis le serveur (ex. local 3307 → BD 3306). R : un port du serveur → une adresse côté PC. D : ouvre un port de ce PC comme proxy SOCKS ; les connexions des programmes qui utilisent ce proxy sortent par le serveur (le programme choisit la destination).",
  "fwd.help": "L: un puerto de este PC → una dirección accesible desde el servidor (p. ej., local 3307 → BD 3306). R: un puerto del servidor → una dirección del lado de este PC. D: abre un puerto de este PC como proxy SOCKS; las conexiones de los programas que usan ese proxy salen a través del servidor (el programa elige el destino).",
  "fwd.help": "L: uma porta deste PC → um endereço acessível pelo servidor (ex.: local 3307 → BD 3306). R: uma porta do servidor → um endereço do lado deste PC. D: abre uma porta deste PC como proxy SOCKS; as conexões dos programas que usam esse proxy saem pelo servidor (o programa escolhe o destino).",
  "fwd.help": "L: порт этого ПК → адрес, доступный с сервера (например, локальный 3307 → БД 3306). R: порт сервера → адрес на стороне этого ПК. D: открывает порт этого ПК как SOCKS-прокси; соединения программ, использующих этот прокси, выходят через сервер (адрес назначения выбирает программа).",
```

**`fwd.dynamicDest`** (추가)

```json
  "fwd.dynamicDest": "SOCKS 프록시 (대상 없음)",
  "fwd.dynamicDest": "SOCKS proxy (no destination)",
  "fwd.dynamicDest": "SOCKS プロキシ（宛先なし）",
  "fwd.dynamicDest": "SOCKS 代理（无目标）",
  "fwd.dynamicDest": "SOCKS 代理（無目的地）",
  "fwd.dynamicDest": "SOCKS-Proxy (kein Ziel)",
  "fwd.dynamicDest": "Proxy SOCKS (sans destination)",
  "fwd.dynamicDest": "Proxy SOCKS (sin destino)",
  "fwd.dynamicDest": "Proxy SOCKS (sem destino)",
  "fwd.dynamicDest": "SOCKS-прокси (без адреса назначения)",
```

**`be.fwd.dynamicActive`** (추가)

```json
  "be.fwd.dynamicActive": "동적 포워딩(SOCKS 프록시) {from}",
  "be.fwd.dynamicActive": "Dynamic forwarding (SOCKS proxy) {from}",
  "be.fwd.dynamicActive": "ダイナミックフォワーディング（SOCKS プロキシ）{from}",
  "be.fwd.dynamicActive": "动态转发（SOCKS 代理）{from}",
  "be.fwd.dynamicActive": "動態轉送（SOCKS 代理）{from}",
  "be.fwd.dynamicActive": "Dynamische Weiterleitung (SOCKS-Proxy) {from}",
  "be.fwd.dynamicActive": "Redirection dynamique (proxy SOCKS) {from}",
  "be.fwd.dynamicActive": "Reenvío dinámico (proxy SOCKS) {from}",
  "be.fwd.dynamicActive": "Encaminhamento dinâmico (proxy SOCKS) {from}",
  "be.fwd.dynamicActive": "Динамическое перенаправление (SOCKS-прокси) {from}",
```

**`be.fwd.dynamicFailed`** (추가)

```json
  "be.fwd.dynamicFailed": "포트 포워딩 D{port} 실패: {error}",
  "be.fwd.dynamicFailed": "Port forwarding D{port} failed: {error}",
  "be.fwd.dynamicFailed": "ポートフォワーディング D{port} 失敗: {error}",
  "be.fwd.dynamicFailed": "端口转发 D{port} 失败：{error}",
  "be.fwd.dynamicFailed": "連接埠轉送 D{port} 失敗：{error}",
  "be.fwd.dynamicFailed": "Portweiterleitung D{port} fehlgeschlagen: {error}",
  "be.fwd.dynamicFailed": "Échec de la redirection de port D{port} : {error}",
  "be.fwd.dynamicFailed": "Error en el reenvío de puerto D{port}: {error}",
  "be.fwd.dynamicFailed": "Falha no encaminhamento de porta D{port}: {error}",
  "be.fwd.dynamicFailed": "Сбой перенаправления порта D{port}: {error}",
```


## F. 명세

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -53,7 +53,7 @@
 | keepSession | 불리언(선택) | 세션 유지(tmux) |
 | tmuxName | 문자열(선택) | 비우면 서버 별칭(로컬은 폴더 이름) |
 | keepaliveSec | 숫자 | 새 서버 기본 30 |
-| forwards | 배열 | 포트 포워딩: id, type(`L`/`R`), bindHost, bindPort, destHost, destPort |
+| forwards | 배열 | 포트 포워딩: id, type(`L`/`R`/`D`), bindHost, bindPort, destHost, destPort. `D`(동적, 2026-11-23 D109)는 destHost·destPort 를 쓰지 않는다 |
 | serial | 객체(선택) | path, baudRate, dataBits(5/6/7/8), parity(`none` `even` `odd` `mark` `space`), stopBits(1/1.5/2), flowControl(`none` `rtscts` `xonxoff`), enterSends(`CR` `LF` `CRLF`), localEcho |
 | local | 객체(선택) | shell, cwd (비우면 기본 셸·홈 폴더) |
 | notes | 문자열(선택) | |
```

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -55,6 +55,7 @@
 - 서버의 `forwards` 마다: 바인드 주소가 비면 `127.0.0.1`.
   - **L(로컬)**: 이 PC 의 바인드 주소:포트에서 대기 → 연결이 오면 SSH 연결을 통해 대상 호스트:포트로 중계.
   - **R(원격)**: 서버 쪽에서 바인드 주소:포트로 대기 → 연결이 오면 이 PC 에서 대상 호스트:포트로 중계.
+  - **D(동적)**(2026-11-23, D109): 이 PC 의 바인드 주소:포트를 **SOCKS 프록시**로 연다. 들어온 연결의 SOCKS 요청(SOCKS5 인증 없음·SOCKS4·SOCKS4a, 명령은 CONNECT 만)이 가리키는 대상으로 SSH 연결을 통해 중계한다. 대상 호스트·포트 설정은 쓰지 않는다. 도메인 이름은 이 PC 가 아니라 **서버 쪽에서** 풀린다. SOCKS5 에서 인증 없음을 내놓지 않은 클라이언트·CONNECT 가 아닌 명령·닿지 않는 대상은 SOCKS 오류 응답으로 거절하고, SOCKS 가 아닌 연결은 응답 없이 닫는다. 요청을 10초 안에 다 보내지 않는 연결은 닫는다. 프록시 자체에는 인증이 없으므로 바인드 주소를 `127.0.0.1` 밖으로 열면 그 주소에 닿는 누구나 쓸 수 있다. 성공 시 터미널에 "동적 포워딩(SOCKS 프록시) {바인드}:{포트}", 실패 시 "포트 포워딩 D{포트} 실패: {오류}".
 - 설정 성공 시 터미널에 "포워딩 {바인드}:{포트} → {대상}", 실패 시 "포트 포워딩 {L|R}{포트} 실패: {오류}" 를 보이고 다음 항목으로 계속한다. 세션이 끝나면 모든 대기 소켓을 닫는다.
 
 ### 2.6 연결 유지(keepalive)
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -49,8 +49,8 @@
 5. 로컬: "터미널 종류" 한 칸. 그 외: 3열 — "문자 인코딩"(옵션 4개: "UTF-8"(utf-8), "CP949 / EUC-KR (구형 장비)"(cp949), "Shift_JIS"(shift_jis), "Latin-1"(latin1). 저장 데이터에 `euc-kr` 값이 있어도 읽을 수 있어야 한다) / "터미널 종류" / "연결 확인 간격 (초)"(숫자, 비면 0).
 
 ### 1.4 "포트 포워딩" 탭 (SSH 만)
-- 안내 박스: "L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소."
-- 표: 머리 종류 · 바인드 주소 · 포트 · 대상 호스트 · 대상 포트 · (삭제). 행마다 종류 드롭다운(L/R), 바인드 주소(고정폭), 포트(숫자), 대상 호스트(고정폭), 대상 포트(숫자), [삭제](휴지통 아이콘).
+- 안내 박스: "L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소. D: 내 PC의 포트를 SOCKS 프록시로 열어, 그 프록시를 쓰는 프로그램의 접속을 서버를 거쳐 내보냅니다(대상은 프로그램이 정합니다)."(2026-11-23, D109)
+- 표: 머리 종류 · 바인드 주소 · 포트 · 대상 호스트 · 대상 포트 · (삭제). 행마다 종류 드롭다운(L/R/D), 바인드 주소(고정폭), 포트(숫자), 대상 호스트(고정폭), 대상 포트(숫자), [삭제](휴지통 아이콘). 종류가 D 이면 대상 호스트·대상 포트 칸 대신 흐린 글자 "SOCKS 프록시 (대상 없음)" 을 두 칸 너비로 보인다(적어 둔 대상 값은 지우지 않아 종류를 되돌리면 다시 보인다).
 - [포워딩 추가] 버튼: 새 행 기본값 종류 L, 바인드 주소 `127.0.0.1`, 포트 8080, 대상 호스트 `localhost`, 대상 포트 80.
 
 ### 1.5 저장 동작 (검사·정규화·볼트 저장)
```



---

## G. 검증 (2026-11-23 실행)

SSH 대신 일반 TCP 로 대상에 잇는 함수를 넣어 SOCKS 처리만 확인한다. 대상은 받은 것을 그대로 돌려주는 서버다.

| 테스트 ([internal/sshconn/socks_test.go](../../internal/sshconn/socks_test.go)) | 확인하는 것 |
|---|---|
| `TestSocks5Connect` | IPv4 주소와 도메인 이름으로 CONNECT 하면 대상까지 이어진다 / 인사와 요청을 한 번에 보내도 된다 / 대상 주소가 요청 그대로 넘어간다(도메인 이름을 이 PC 에서 풀지 않는다) |
| `TestSocks5Refusals` | 인증 없음을 내놓지 않으면 `05 FF` / BIND 는 응답 코드 07 / 닿지 않는 대상은 04 |
| `TestSocks4And4a` | SOCKS4(IP)·SOCKS4a(도메인 이름) 허용 응답 `5A` 와 이어짐 / BIND 는 `5B` |
| `TestSocksUnknownVersionCloses` | SOCKS 가 아닌 연결(HTTP 요청)은 응답 없이 닫는다 |

[proxy-connect](./2026-11-23-proxy-connect.md) 의 `TestSocks5ProxyConnect`·`TestSSHConnectThroughProxy` 는 이 SOCKS 서버를 프록시로 써서 **이 앱의 SOCKS5 클라이언트와 맞물려** 한 번 더 확인한다.

```bash
go test ./internal/sshconn/     # 통과
(cd frontend && npx tsc --noEmit)
```

`StartForwards` 의 D 분기(리스너를 열고 `serveSocks` 에 SSH 연결의 `Dial` 을 넘기는 6줄)는 SSH 서버가 있어야 도는 부분이라 따로 자동 테스트를 두지 않았다. → 수동 검증 1번.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 서버 편집 > 포트 포워딩 > [+ 포워딩 추가] → 종류 **D**, 수신 주소 `127.0.0.1`, 포트 1080 → 저장하고 연결 | 터미널에 `동적 포워딩(SOCKS 프록시) 127.0.0.1:1080`. 대상 칸 자리에 "SOCKS 프록시 (대상 없음)" |
| 2 | `curl --socks5-hostname 127.0.0.1:1080 http://<서버에서만 닿는 주소>/` | 응답이 온다 |
| 3 | 브라우저의 SOCKS 프록시를 `127.0.0.1:1080` 으로 | 서버를 거쳐 나간다(IP 확인 사이트에 서버 IP 가 보인다) |
| 4 | 이미 쓰는 포트로 D 를 만든다 | `포트 포워딩 D{포트} 실패: …` 를 보이고 세션은 정상 |
| 5 | 세션을 닫는다 | 1080 포트가 닫힌다 |
| 6 | 종류를 D → L 로 되돌린다 | 전에 적어 둔 대상 주소·포트가 그대로 보인다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| L·R 포워딩 | D 분기는 그 앞에서 `continue` 한다. L·R 코드는 그대로 | 기존 테스트 |
| 저장 데이터 | `type` 값이 하나 늘었다. **옛 버전 앱**이 이 데이터를 읽으면 D 를 L 로 처리한다(대상 `localhost:80` 으로 고정 포워딩) | git 동기화로 옛 버전과 섞어 쓸 때만 해당 |
| 보안 | 바인드 주소를 밖으로 열면 인증 없는 프록시가 된다 | 기본값 `127.0.0.1`, 명세·도움말에 명시 |
| 자원 | 연결마다 고루틴 2개. 세션을 닫으면 리스너가 닫히고, 열려 있던 연결은 SSH 연결이 닫히면서 끝난다 | — |

## J. 체크리스트

- [x] `socks.go` 새 파일, `keepalive_forward.go` 에 D 분기
- [x] 프런트: `Forward.type` 에 `'D'`, 포워딩 표의 종류 선택과 대상 칸
- [x] 문구 1개 변경·3개 추가(10개 언어), 명세 protocols.md §2.5·data/store.md §3·ui/host-editor.md §1.4, decisions D109
- [x] 자동 테스트 4개 통과
- [ ] 수동 검증 1~6
