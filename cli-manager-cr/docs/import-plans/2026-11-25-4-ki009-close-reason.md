# 텔넷·시리얼 끊김 안내에 끊긴 까닭 넣기 (KI-009) — 원인·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D127). 실제 텔넷 서버·시리얼 장치에서의 문구 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: KI-009 (resolved)
**관련**: 처음 찾은 문서 [2026-11-16-session-transport-issues.md](./2026-11-16-session-transport-issues.md) §8. 명세 [spec/protocols.md](../../spec/protocols.md) §4·§5, [spec/behavior/session.md](../../spec/behavior/session.md)(끊김 안내 `● 연결 끊김: {이유}`)
**적용 순서**: 2026-11-25 의 4번째. 앞의 [3번(단축키 설정)](./2026-11-25-3-shortcut-settings.md)이 `locales/*.json` 을 고쳤으므로 그 뒤에 적용한다.

> 이 문서 하나로 **무엇이 문제였는지 → 왜 그랬는지 → 어떻게 고쳤는지 → 코드 전체 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

텔넷·시리얼 세션이 끊기면 터미널에 까닭 없이 늘 같은 문구만 나왔다. 이제 **끊긴 까닭을 함께** 보인다.

| 상황 | 바꾸기 전 | 바꾼 뒤 |
|---|---|---|
| 텔넷: 서버가 연결을 닫음(exit, 서버 쪽 시간 초과 등) | `● 연결 끊김: 연결이 종료되었습니다` | `● 연결 끊김: 서버가 연결을 닫았습니다` |
| 텔넷: 강제로 끊김(방화벽·서버 재시작·회선 문제) | `● 연결 끊김: 연결이 종료되었습니다` | `● 연결 끊김: 네트워크 오류 (wsarecv: An existing connection was forcibly closed by the remote host.)` |
| 시리얼: 장치가 뽑히거나 드라이버 오류 | `● 연결 끊김: 포트가 닫혔습니다` | `● 연결 끊김: 포트가 닫혔습니다 (A device attached to the system is not functioning.)` |
| 사용자가 직접 닫음 | 안내 없음 | 안내 없음(그대로) |

괄호 안의 글은 Windows·드라이버가 준 오류 글 **그대로**다(위 두 번째 줄은 이 PC 의 테스트에서 실제로 나온 글, 세 번째 줄은 예시).

## 1. 사용자 요청

2026-11-25: "KI-009 도 직접적인 원인을 끊김 안내 문구에 적어줘. 사용자가 확인할 수 있게. 수정해서 이것도 md 파일로 추가해주고."

KI-009 는 2026-11-16 점검에서 찾았지만 "명세(spec/protocols.md §4·§5)가 고정 문구를 요구한다"는 까닭으로 고치지 않고 dismissed 로 두었던 항목이다. 이번 지시로 명세와 코드를 함께 고친다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 끊김 안내 | 연결이 끊겼을 때 터미널에 노랑으로 나오는 줄 `● 연결 끊김: {이유}` |
| 이유 | 전송 계층(텔넷·시리얼·SSH 코드)이 "읽기가 끝났다"며 돌려준 오류의 글. 세션이 끊김 안내의 `{이유}` 에 넣는다 |
| 까닭 | 이 문서에서 새로 붙이는 것. OS·드라이버가 읽기·쓰기 실패 때 준 원래 오류 글 |
| EOF | "더 읽을 것이 없다"는 표시. 상대가 연결을 정상적으로 닫았을 때(TCP FIN) 읽기가 이것으로 끝난다 |
| RST | 상대(또는 중간 장비)가 연결을 강제로 끊을 때 보내는 TCP 신호. 읽기가 오류로 끝난다 |

## 2. 원인

전송 계층이 OS 에서 받은 오류를 **버리고** 미리 정해 둔 오류만 돌려줬다.

| 곳 | 전의 코드 | 버려진 것 |
|---|---|---|
| [internal/transport/telnet.go](../../internal/transport/telnet.go) `Read` | `if errRead != nil { return 0, ErrRemoteClosed }` | 소켓 읽기 오류(EOF·RST·시간 초과 등) |
| [internal/transport/serial.go](../../internal/transport/serial.go) `Read` | `if errRead != nil { return 0, ErrPortClosed }` | 포트 읽기 오류(장치 뽑힘 등) |
| [internal/transport/serial.go](../../internal/transport/serial.go) `Write` | `if errWrite != nil { return ErrPortClosed }` | 포트 쓰기 오류 |

흐름:

```
OS 오류(errRead) ─ 전송 계층 Read ─ 세션 읽기 루프(session.go) ─ onDisconnected(이유) ─ 터미널 "● 연결 끊김: {이유}"
                      └ 전: errRead 를 버리고 고정 오류(ErrRemoteClosed / ErrPortClosed)
                      └ 뒤: errRead 의 까닭을 든 오류(종류는 같다)
```

세션 쪽([internal/session/session.go](../../internal/session/session.go) 읽기 루프, [messages.go](../../internal/session/messages.go) `MsgDisconnected`)은 받은 오류의 글을 그대로 안내에 넣으므로 고칠 것이 없다. 까닭이 전송 계층에서 사라진 것이 전부다.

## 3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 오류의 모양 | 새 타입 `closedError`(종류 + 문구 키 + 까닭). `Unwrap` 으로 `ErrRemoteClosed`·`ErrPortClosed` 를 돌려준다 | 다른 코드가 `errors.Is(err, ErrRemoteClosed)` 로 종류를 보는 것이 그대로 동작한다. 자동 재접속 판단도 달라지지 않는다 |
| 문구를 만드는 때 | `Error()` 를 부를 때 지금 언어로 | 고정 오류(`i18n.Error`)와 같은 방식. 만들 때 언어로 굳지 않는다 |
| 텔넷 — 서버가 정상적으로 닫음(EOF) | 따로 문구 "서버가 연결을 닫았습니다" | 가장 흔한 경우이고(서버에서 exit, 서버 쪽 유휴 시간 초과) "EOF" 라는 글은 사용자에게 뜻이 없다 |
| 텔넷 — 그 밖의 오류 | "네트워크 오류 ({까닭})" | 까닭은 OS 글 그대로 |
| 시리얼 | "포트가 닫혔습니다 ({까닭})" | 있던 문구에 까닭만 붙인다. 읽기와 쓰기 모두 |
| {까닭}의 언어 | **번역하지 않는다**(OS 가 준 글 그대로) | 종류가 많고 OS·드라이버마다 다르다. 몇 개만 번역하면 나머지는 뭉뚱그려지거나 가려진다. 원문이면 검색해 볼 수 있다 |
| 주소 머리 | 네트워크 오류의 `read tcp 10.0.0.5:51234->10.0.0.9:23:` 머리는 **뺀다** | 길고, 상대 주소는 탭·상태 표시줄에 이미 있다. 뒤의 까닭이 가려진다 |
| 이쪽에서 닫은 뒤의 오류 | 까닭을 붙이지 않는다(전과 같은 고정 오류) | "use of closed network connection" 같은 글은 앱이 닫아서 난 것이라 끊긴 까닭이 아니다. 사용자가 닫은 세션은 안내 자체가 없다 |
| 범위 | 텔넷·시리얼만 | KI-009 의 범위. SSH 는 §I 참고 |

### 3.1 까닭 글이 어떤 것들인가 (참고)

OS 가 정하는 글이라 여기서 다 적을 수는 없다. 흔한 것(Windows):

| 상황 | 까닭 글(대개) |
|---|---|
| 서버·방화벽이 강제로 끊음 | `wsarecv: An existing connection was forcibly closed by the remote host.` |
| 회선이 끊겨 응답이 없음(OS 의 TCP 시한) | `wsarecv: A connection attempt failed because the connected party did not properly respond after a period of time, …` |
| 이 PC 의 네트워크가 내려감 | `wsarecv: An established connection was aborted by the software in your host machine.` |
| 시리얼 장치(USB 변환기)를 뽑음 | 드라이버에 따라 `A device attached to the system is not functioning.` · `Access is denied.` · `The I/O operation has been aborted…` 등 |

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/transport/closecause.go` | 코드(Go) | **새 파일** |
| `internal/transport/telnet.go` | 코드(Go) | 수정(1곳 + 주석) |
| `internal/transport/serial.go` | 코드(Go) | 수정(2곳 + `isClosed` + 주석) |
| `internal/transport/closecause_test.go` | 테스트 | **새 파일**(6개) |
| `locales/*.json` (10개) | 문구 | 키 3개 추가 |
| `spec/protocols.md` | 명세 | 수정(§4·§5) |
| `.claude/known-issues.md` | 기록 | KI-009 dismissed → resolved |
| `CHANGELOG.md`, `docs/decisions.md`(D127), `CLAUDE.md` | 문서 | 수정 |

프런트 코드는 바꾸지 않았다(문구 파일만 함께 쓴다).

## B. 코드

### `internal/transport/closecause.go` — 새 파일

```go
package transport

import (
	"dariterm/internal/i18n"
	"errors"
	"io"
	"net"
)

// closedError 는 끊긴 까닭을 붙인 끊김 오류다 (protocols.md §4·§5, D127).
// errors.Is 로는 errBase(ErrRemoteClosed·ErrPortClosed)와 같고, Error() 는 지금 언어의 "까닭이 든 문구"다.
// 문구는 Error() 를 부를 때 만든다(고정 오류 i18n.Error 와 같이 시작 때 언어로 굳지 않는다).
type closedError struct {
	errBase error  // 종류: ErrRemoteClosed 또는 ErrPortClosed
	szId    string // 문구 키
	szCause string // 문구의 {cause} 에 넣을 원래 오류 글. 비면 넣지 않는다
}

// Error 는 지금 언어의 문구다.
func (pErr *closedError) Error() string {
	if pErr.szCause == "" {
		return i18n.T(pErr.szId)
	}
	return i18n.T(pErr.szId, "cause", pErr.szCause)
}

// Unwrap 은 끊김의 종류를 돌려준다 (errors.Is(err, ErrRemoteClosed) 가 그대로 참이다).
func (pErr *closedError) Unwrap() error {
	return pErr.errBase
}

// causeText 는 OS·라이브러리가 준 오류에서 사용자가 볼 까닭만 뽑는다.
// 네트워크 오류는 "read tcp 내주소->상대주소: " 머리를 떼고 OS 가 말한 까닭만 남긴다.
func causeText(errCause error) string {
	var pOpError *net.OpError
	if errors.As(errCause, &pOpError) && pOpError.Err != nil {
		return pOpError.Err.Error()
	}
	return errCause.Error()
}

// telnetClosed 는 텔넷 읽기가 errRead 로 끝났을 때의 끊김 오류를 만든다.
// 이쪽에서 닫은 것(Close)이면 까닭 없는 ErrRemoteClosed, 서버가 정상적으로 닫았으면(EOF) "서버가 연결을 닫았습니다",
// 그 밖에는 "네트워크 오류 ({까닭})" 이다.
func telnetClosed(errRead error) error {
	if errRead == nil || errors.Is(errRead, net.ErrClosed) {
		return ErrRemoteClosed
	}
	if errors.Is(errRead, io.EOF) {
		return &closedError{errBase: ErrRemoteClosed, szId: "be.remote.closedByPeer"}
	}
	return &closedError{errBase: ErrRemoteClosed, szId: "be.remote.closedCause", szCause: causeText(errRead)}
}

// serialClosed 는 시리얼 읽기·쓰기가 errIo 로 끝났을 때의 끊김 오류를 만든다.
// 이쪽에서 닫은 것(bUserClosed)이면 까닭 없는 ErrPortClosed, 그 밖에는 "포트가 닫혔습니다 ({까닭})" 이다.
func serialClosed(errIo error, bUserClosed bool) error {
	if errIo == nil || bUserClosed {
		return ErrPortClosed
	}
	return &closedError{errBase: ErrPortClosed, szId: "be.serial.closedCause", szCause: causeText(errIo)}
}
```

### `internal/transport/telnet.go` — 수정

```diff
--- a/internal/transport/telnet.go
+++ b/internal/transport/telnet.go
@@ -15,6 +15,7 @@
 const G_cdurTelnetConnect = 15 * time.Second
 
 // ErrRemoteClosed 는 상대가 연결을 끊었을 때의 오류다 (protocols.md §4: "연결이 종료되었습니다").
+// 텔넷은 끊긴 까닭을 붙인 오류(closecause.go 의 telnetClosed)를 돌려주며, errors.Is 로는 이것과 같다 (D127).
 var ErrRemoteClosed = i18n.NewError("be.remote.closed")
 
 // ErrRemoteExited 는 서버가 셸을 정상적으로 끝냈을 때(종료 코드·신호와 함께 채널을 닫음)의 오류다.
@@ -100,7 +101,7 @@
 			}
 		}
 		if errRead != nil {
-			return 0, ErrRemoteClosed
+			return 0, telnetClosed(errRead) // 끊긴 까닭을 문구에 붙인다 (KI-009, D127)
 		}
 	}
 }
```

### `internal/transport/serial.go` — 수정

```diff
--- a/internal/transport/serial.go
+++ b/internal/transport/serial.go
@@ -12,6 +12,7 @@
 )
 
 // ErrPortClosed 는 시리얼 포트가 닫혔을 때의 이유다 (protocols.md §5).
+// 읽기·쓰기 오류로 닫힌 경우는 까닭을 붙인 오류(closecause.go 의 serialClosed)를 돌려주며, errors.Is 로는 이것과 같다 (D127).
 var ErrPortClosed = i18n.NewError("be.serial.closed")
 
 // G_cdurSerialReadTick 은 읽기 시한이다. 이 주기로 Read 가 돌아와 대기 중인 로컬 에코를 내보낸다 (protocols.md §5).
@@ -149,7 +150,7 @@
 		}
 		nRaw, errRead := pSerial.port.Read(bRaw)
 		if errRead != nil {
-			return 0, ErrPortClosed
+			return 0, serialClosed(errRead, pSerial.isClosed()) // 닫힌 까닭을 문구에 붙인다 (KI-009, D127)
 		}
 		if nRaw == 0 {
 			continue // 시한: 데이터 없음. 위에서 대기분을 다시 본다.
@@ -275,7 +276,7 @@
 	}
 	pSerial.mtxWrite.Unlock()
 	if errWrite != nil {
-		return ErrPortClosed
+		return serialClosed(errWrite, pSerial.isClosed())
 	}
 	if bFlow {
 		pSerial.noteFlowResult(nSent == len(bSend))
@@ -316,6 +317,13 @@
 	return nil
 }
 
+// isClosed 는 이쪽에서 Close 를 불렀는지 돌려준다 (그 뒤의 읽기·쓰기 오류에는 까닭을 붙이지 않는다).
+func (pSerial *SerialConn) isClosed() bool {
+	pSerial.mtxState.Lock()
+	defer pSerial.mtxState.Unlock()
+	return pSerial.bClosed
+}
+
 // Close 는 포트를 닫는다.
 func (pSerial *SerialConn) Close() error {
 	pSerial.mtxState.Lock()
```


## C. 테스트 코드

### `internal/transport/closecause_test.go` — 새 파일

```go
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
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다. `{cause}` 자리에는 OS·드라이버가 준 오류 글이 그대로 들어간다.

넣는 자리: `be.remote.closedByPeer`·`be.remote.closedCause` 는 `be.remote.closed` 바로 다음 줄, `be.serial.closedCause` 는 `be.serial.closed` 바로 다음 줄. 있던 문구(`be.remote.closed`·`be.serial.closed`)는 그대로 두고 계속 쓴다(이쪽에서 닫은 경우, SSH).

**`be.serial.closedCause`** (추가)

```json
  "be.serial.closedCause": "포트가 닫혔습니다 ({cause})",
  "be.serial.closedCause": "The port was closed ({cause})",
  "be.serial.closedCause": "ポートが閉じられました（{cause}）",
  "be.serial.closedCause": "端口已关闭（{cause}）",
  "be.serial.closedCause": "連接埠已關閉（{cause}）",
  "be.serial.closedCause": "Der Anschluss wurde geschlossen ({cause})",
  "be.serial.closedCause": "Le port a été fermé ({cause})",
  "be.serial.closedCause": "El puerto se cerró ({cause})",
  "be.serial.closedCause": "A porta foi fechada ({cause})",
  "be.serial.closedCause": "Порт закрыт ({cause})",
```

**`be.remote.closedByPeer`** (추가)

```json
  "be.remote.closedByPeer": "서버가 연결을 닫았습니다",
  "be.remote.closedByPeer": "The server closed the connection",
  "be.remote.closedByPeer": "サーバーが接続を閉じました",
  "be.remote.closedByPeer": "服务器关闭了连接",
  "be.remote.closedByPeer": "伺服器關閉了連線",
  "be.remote.closedByPeer": "Der Server hat die Verbindung geschlossen",
  "be.remote.closedByPeer": "Le serveur a fermé la connexion",
  "be.remote.closedByPeer": "El servidor cerró la conexión",
  "be.remote.closedByPeer": "O servidor fechou a conexão",
  "be.remote.closedByPeer": "Сервер закрыл соединение",
```

**`be.remote.closedCause`** (추가)

```json
  "be.remote.closedCause": "네트워크 오류 ({cause})",
  "be.remote.closedCause": "Network error ({cause})",
  "be.remote.closedCause": "ネットワークエラー（{cause}）",
  "be.remote.closedCause": "网络错误（{cause}）",
  "be.remote.closedCause": "網路錯誤（{cause}）",
  "be.remote.closedCause": "Netzwerkfehler ({cause})",
  "be.remote.closedCause": "Erreur réseau ({cause})",
  "be.remote.closedCause": "Error de red ({cause})",
  "be.remote.closedCause": "Erro de rede ({cause})",
  "be.remote.closedCause": "Ошибка сети ({cause})",
```


## F. 명세·문서

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -120,12 +120,16 @@
   - 서버의 `DO NAWS`(창 크기)에는 `WILL NAWS` 로 응답하고 현재 크기를 하위 협상으로 즉시 전송, 이후 크기가 바뀔 때마다 전송(16비트 열·행). `DO TERMINAL-TYPE` 에는 `WILL`, 서버가 종류를 물으면(`SEND`) 서버 설정의 터미널 종류(비면 `xterm-256color`)를 보낸다. 그 외 `DO` 에는 `WONT`.
   - 데이터의 `IAC IAC` 는 바이트 255 하나로 해석.
 - 보낼 때 입력의 바이트 255 는 `IAC IAC` 로 이스케이프한다.
-- 연결이 끊기면 이유 "연결이 종료되었습니다"(사용자가 닫은 경우는 이유 없음).
+- 연결이 끊기면 이유를 **끊긴 까닭과 함께** 낸다(2026-11-25, D127 — 전에는 늘 "연결이 종료되었습니다"였다. 사용자가 닫은 경우는 이유 없음):
+  - 서버가 연결을 정상적으로 닫음(더 읽을 것이 없음): "서버가 연결을 닫았습니다"
+  - 그 밖의 읽기 오류: "네트워크 오류 ({까닭})" — {까닭}은 OS 가 준 오류 글이고, 앞의 "read tcp {내 주소}->{상대 주소}:" 머리는 뺀다. 예(Windows, 상대가 강제로 끊음): `네트워크 오류 (wsarecv: An existing connection was forcibly closed by the remote host.)`. {까닭}은 OS 의 글 그대로라 화면 언어로 번역되지 않는다.
+  - 이쪽에서 연결을 닫은 뒤에 난 읽기 오류: 전과 같은 "연결이 종료되었습니다"(화면에는 나오지 않는다).
+  - 터미널에는 세션의 끊김 안내에 넣어 보인다: `연결 끊김: {이유}` (behavior/session.md §4).
 
 ## 5. 시리얼
 
 - 장치 열기: 속도(0 이면 9600), 데이터 비트(0 이면 8), 패리티(none/even/odd/mark/space), 정지 비트(1/1.5/2). **흐름 제어**(`flowControl`, 2026-11-23, D108 — 전에는 저장만 하고 쓰지 않았다): `none`(기본)은 입력을 그대로 보낸다. `rtscts` 는 열 때 RTS 선을 올려 두고, 보낼 때 **장치의 CTS 선이 켜져 있을 때만** 보낸다. `xonxoff` 는 장치가 XOFF(0x13)를 보낸 뒤부터 XON(0x11)을 보낼 때까지 보내지 않으며, 장치가 보낸 XON·XOFF 는 화면 출력에서 뺀다. 두 방식 모두 입력을 **16바이트 조각**으로 나눠 조각마다 장치 상태를 다시 본다. 장치가 **2초** 넘게 받을 수 없으면 그 입력의 남은 부분을 버리고 터미널에 노랑 안내 "장치가 받을 준비가 되지 않아 입력을 보내지 못했습니다 (흐름 제어)" 를 낸다(막혀 있는 동안 한 번만, 다시 보낼 수 있게 된 뒤 또 막히면 다시). 버린 입력은 로컬 에코도 하지 않는다. CTS 상태를 읽을 수 없는 장치는 받을 수 있는 것으로 본다. 포트를 닫으면 기다리던 보내기는 곧바로 끝난다. 받는 방향(장치 → 이 PC)은 조절하지 않는다(RTS 를 내리거나 XOFF 를 보내지 않는다 — 앱이 받은 데이터를 곧바로 읽어 가기 때문). 열기 실패 시 오류 메시지는 `{장치}: {원인}`.
-- 읽기 단위 4KB. 포트가 닫히면 이유 "포트가 닫혔습니다"(사용자가 닫은 경우는 이유 없음). **시리얼은 자동 재접속을 하지 않는다**.
+- 읽기 단위 4KB. 포트가 닫히면(읽기·쓰기 오류) 이유 "포트가 닫혔습니다 ({까닭})" — {까닭}은 OS·드라이버가 준 오류 글 그대로다(2026-11-25, D127 — 전에는 까닭 없이 "포트가 닫혔습니다"였다). 예: 장치를 뽑음 → `포트가 닫혔습니다 (A device attached to the system is not functioning.)` 류. 이쪽에서 포트를 닫은 뒤의 오류는 까닭 없이 "포트가 닫혔습니다"(사용자가 닫은 경우는 이유 없음). **시리얼은 자동 재접속을 하지 않는다**.
 - 읽기는 50ms 시한으로 돌아와 그사이 쌓인 로컬 에코를 바로 내보낸다(장치가 아무것도 보내지 않아도 입력한 글자가 즉시 보인다).
 - 쓰기: 입력의 CR(`\r`)을 **Enter 전송값**(CR/LF/CRLF)으로 바꿔 보낸다. **로컬 에코**가 켜져 있으면 보낸 내용을 그대로 화면에도 출력하되 CR 과 CRLF 는 화면에서 줄바꿈(`\r\n`)으로 보이게 한다.
 - 터미널 크기 변경은 시리얼에 아무 영향이 없다.
```

### `.claude/known-issues.md` — 수정

```diff
--- a/.claude/known-issues.md
+++ b/.claude/known-issues.md
@@ -10,8 +10,12 @@
 
 ## ⏸️ DISMISSED
 
+(없음)
+
+## ✔️ RESOLVED
+
 ### KI-009: 텔넷·시리얼 끊김 이유가 원래 오류 대신 고정 문구
-- **Status**: dismissed
+- **Status**: resolved
 - **Severity**: low
 - **Module**: internal/transport (telnet.go, serial.go)
 - **Trigger areas**: 텔넷·시리얼 끊김 안내 문구
@@ -19,8 +23,7 @@
 - **원인**: spec/protocols.md §4·§5 가 고정 문구를 요구한다.
 - **History**:
   - 2026-11-16: 세션·전송 계층 점검(docs/import-plans/2026-11-16-session-transport-issues.md §8)에서 발견. 명세대로의 동작이라 dismissed (사유: 명세의 고정 문구).
-
-## ✔️ RESOLVED
+  - 2026-11-25: 사용자 요청("KI-009 도 직접적인 원인을 끊김 안내 문구에 적어줘. 사용자가 확인할 수 있게")으로 수정 (D127). 텔넷·시리얼이 읽기·쓰기 오류를 버리지 않고 이유에 붙인다(`internal/transport/closecause.go` `telnetClosed`·`serialClosed`): 텔넷은 "서버가 연결을 닫았습니다" 또는 "네트워크 오류 ({까닭})", 시리얼은 "포트가 닫혔습니다 ({까닭})". `errors.Is(…, ErrRemoteClosed/ErrPortClosed)` 는 그대로 참. 명세 spec/protocols.md §4·§5 도 고쳤다. 테스트 `closecause_test.go` 6개. 실제 장비(텔넷 서버·시리얼 장치)에서의 문구 확인은 계획서 docs/import-plans/2026-11-25-4-ki009-close-reason.md §H
 
 ### KI-011: 찾기 막대가 열린 동안 찾은 글자가 클립보드에 계속 복사되고, 출력 중에는 강조가 늦게 갱신됨
 - **Status**: resolved
```

`CHANGELOG.md` 의 미배포 절 "수정"에 한 줄, `docs/decisions.md` 에 D127 을 넣었다.

---

## G. 검증 (2026-11-25 실행)

### G.1 새 파일 [internal/transport/closecause_test.go](../../internal/transport/closecause_test.go) 6개

| 테스트 | 확인하는 것 |
|---|---|
| `TestTelnetClosedByPeerReason` | 가짜 서버가 연결을 정상적으로 닫으면 종류는 `ErrRemoteClosed`, 글은 "서버가 연결을 닫았습니다" |
| `TestTelnetResetReasonHasCause` | 가짜 서버가 강제로 끊으면(RST) 글이 "네트워크 오류 (…)" 이고 괄호 안이 비어 있지 않으며, 주소(`127.0.0.1`)·`read tcp` 머리가 없다. 이 PC 에서 나온 글: `네트워크 오류 (wsarecv: An existing connection was forcibly closed by the remote host.)` |
| `TestTelnetLocalCloseHasNoCause` | 이쪽에서 닫은 뒤의 읽기 오류는 까닭 없는 `ErrRemoteClosed`("연결이 종료되었습니다") 그 자체 |
| `TestSerialClosedReasonHasCause` | 장치 오류를 내는 가짜 포트: 읽기·쓰기 모두 종류는 `ErrPortClosed`, 글은 "포트가 닫혔습니다 (A device attached to the system is not functioning.)" |
| `TestSerialLocalCloseHasNoCause` | 이쪽에서 포트를 닫은 뒤의 오류는 까닭 없는 `ErrPortClosed` 그 자체 |
| `TestClosedErrorFollowsLanguage` | 같은 오류의 글이 언어를 바꾸면 그 언어로 나온다(만들 때 굳지 않는다) |

### G.2 그 밖의 자동 테스트

| 테스트 | 확인하는 것 |
|---|---|
| `internal/transport` 의 기존 테스트 | 텔넷 협상·이스케이프, 시리얼 에코·흐름 제어 — 그대로 통과 |
| `internal/session` | 끊김 → 자동 재접속, 정상 종료 구분 등 — 그대로 통과(오류의 종류가 같다) |
| 문구 검사(`go test ./internal/i18n`, 프런트 `src/i18n`) | 10개 언어에 같은 키·같은 자리표시자(`{cause}`), Go 코드가 쓰는 키가 모두 있다 |

### G.3 실행한 명령

```bash
go build ./... && go vet ./internal/transport/     # 통과
go test -count=1 ./...                             # 전부 통과 (이 PC 는 cgo 가 꺼져 있어 -race 는 돌리지 못했다)
(cd frontend && npx vitest run src/i18n)           # PowerShell 에서. 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

자동 테스트가 확인하지 못하는 것:
- 실제 텔넷 서버·실제 시리얼 장치에서 나오는 까닭 글(드라이버마다 다르다).
- 터미널 화면에 안내가 한 줄로 어떻게 보이는지(긴 까닭 글의 줄 넘김).
- 시리얼 장치를 뽑았을 때 드라이버가 오류를 내지 않고 조용히 있는 경우 — 그때는 끊김 자체를 알 수 없다(이 수정과 무관한 원래 한계).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 텔넷 서버에 접속해 `exit` | `● 연결 끊김: 서버가 연결을 닫았습니다` 뒤 자동 재연결 안내(설정이 켜져 있으면) |
| 2 | 텔넷 접속 중 서버에서 그 세션의 프로세스를 강제 종료하거나 서버를 재시작 | `● 연결 끊김: 서버가 연결을 닫았습니다` 또는 `● 연결 끊김: 네트워크 오류 (…forcibly closed…)` |
| 3 | 텔넷 접속 중 이 PC 의 네트워크를 끄고 아무 키나 입력 | 잠시 뒤 `● 연결 끊김: 네트워크 오류 (…)` — 괄호 안에 Windows 의 오류 글, 주소(`read tcp …`)는 없다 |
| 4 | USB 시리얼 변환기로 접속한 뒤 변환기를 뽑는다 | `● 연결 끊김: 포트가 닫혔습니다 (…)` — 괄호 안에 드라이버의 오류 글. 자동 재연결은 하지 않는다 |
| 5 | 텔넷·시리얼 탭을 직접 닫는다 | 끊김 안내가 나오지 않는다(전과 같다) |
| 6 | 화면 언어를 영어로 바꾸고 1·4 를 다시 | `Disconnected: The server closed the connection` / `Disconnected: The port was closed (…)` |
| 7 | SSH 세션이 끊기는 경우 | 전과 같다(`● 연결 끊김: 연결이 종료되었습니다`) — 이번 범위 밖 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 자동 재접속(텔넷) | 오류가 다른 값이 됐다 | 종류는 `errors.Is` 로 같다. 세션은 `ErrRemoteExited` 인지만 따로 보고 나머지는 끊김으로 처리한다. `internal/session` 테스트 통과 |
| 오류를 `==` 로 비교하는 코드 | 까닭이 붙은 오류는 `ErrRemoteClosed` 와 `==` 가 아니다 | 저장소에서 이 두 오류를 `==` 로 비교하는 곳은 없다(테스트의 가짜 전송이 돌려주는 곳뿐) |
| 까닭 글의 언어 | 화면이 한국어여도 괄호 안은 Windows 의 언어(대개 영어)다 | 일부러 그렇게 뒀다(§3). 앞의 문구("네트워크 오류", "포트가 닫혔습니다")는 화면 언어다 |
| 까닭 글의 길이·내용 | 긴 글이 터미널 한 줄을 넘길 수 있다. 주소 머리는 뺐지만 OS 글 안에 다른 정보가 들어 있을 수는 있다 | 터미널이 줄을 넘겨 보인다. 세션 로그를 켜 두면 로그에도 남는다(안내 줄은 원래 로그에 남는다) |
| 시리얼 쓰기 오류 | 쓰기 오류도 같은 글이 된다 | 쓰기 오류의 글이 쓰이는 길은 전과 같다(종류가 같다) |
| SSH | **바꾸지 않았다.** SSH 도 끊기면 까닭 없이 "연결이 종료되었습니다"다([internal/sshconn/shell.go](../../internal/sshconn/shell.go) `Read`, protocols.md §2.1 6번) | KI-009 의 범위가 텔넷·시리얼이다. SSH 는 keepalive 시한·채널 종료·연결 종료가 얽혀 있어 따로 봐야 한다. 필요하면 새 항목으로 다룬다 |
| 번역 | 한국어·영어 외 8개 언어의 문구 3개는 이 작업을 한 세션이 번역했다 | 검수가 필요할 수 있다 |

되돌리기: `closecause.go`·`closecause_test.go` 를 지우고 §B 의 `-` 줄로 되돌린 뒤, 문구 3개(§E)를 10개 파일에서 지운다.

## J. 체크리스트

- [x] `closecause.go`: `closedError`, `causeText`, `telnetClosed`, `serialClosed`
- [x] `telnet.go` `Read`, `serial.go` `Read`·`Write`·`isClosed`
- [x] 문구 3개(10개 언어)
- [x] Go 테스트 6개 추가. `go test ./...`·빌드
- [x] 명세 protocols.md §4·§5, known-issues KI-009 resolved, CHANGELOG, decisions D127, CLAUDE.md
- [ ] 수동 검증 1~7
