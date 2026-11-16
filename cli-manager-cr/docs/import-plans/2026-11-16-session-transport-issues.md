# 세션·로그·전송 계층 잠재 문제 전체 — 원인·수정 계획·코드·테스트 (통합)

**작성**: 2026-11-16 · **상태**: 완료(2026-11-16, D84~D90) · **대상**: cli-manager-cr(DariTerm)
**관련 계획서**: [출력 흐름 제어(역압)](./2026-11-16-output-backpressure.md), [KI-001 로그 파일 이름 충돌](./2026-11-16-log-filename-collision.md)
**규칙**: 공통 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol, 코딩 표준 [../../../docs/11_coding-standard.md](../../../docs/11_coding-standard.md)

> 2026-11-16 세션·로그·전송(SSH·텔넷·시리얼·로컬 ConPTY) 코드를 훑어 찾은 항목 전부를 한 곳에 모았다.
> 각 항목은 **증상 → 원인(코드 경로) → 수정(코드) → 테스트** 순서로 쓴다. 이 문서만 보고 고칠 수 있게 한다.
> 이미 별도 계획서가 있는 두 건(역압, KI-001)은 §0 표에만 두고 본문은 그 문서를 따른다.

---

## 0. 전체 목록과 우선순위

| # | 항목 | 심각도 | 처리 | 바뀌는 파일 | 비고 |
|---|---|---|---|---|---|
| A | 한 세션의 고속 출력이 앱 전체를 느리게 함 | **높음** | 수정 | session, app, 프런트 | [별도 계획서](./2026-11-16-output-backpressure.md) |
| B | 같은 별칭 세션을 같은 분에 열면 로그가 섞임 (KI-001) | 중간 | 수정 | log.go | [별도 계획서](./2026-11-16-log-filename-collision.md). §9 의 별칭 정리도 같이 |
| 1 | SSH keepalive 가 조용히 끊긴 연결을 몇 분 뒤에야 감지 | **높음** | 수정 | sshconn/keepalive_forward.go, client.go | |
| 2 | 시리얼 로컬 에코가 장치가 뭔가 보낼 때까지 안 보임 | **높음** | 수정 | transport/serial.go | 실제 장치로 재현 확인 |
| 3 | 로그에 ANSI 조각이 남음 (묶음 경계에서 잘린 시퀀스) | 중간 | 수정 | session/log.go | §10 과 한 번에 |
| 4 | 로컬 셸 Close 뒤 닫힌 프로세스 핸들 사용 | 중간 | 수정 | transport/local_windows.go | |
| 5 | 시리얼 흐름 제어 설정이 동작에 안 쓰임 | 낮음 | UI 안내만 | 프런트 HostEditorParts.tsx, 명세 | 명세 §5 에 "알려진 한계" 로 이미 기록 |
| 6 | cp949 를 EUC-KR 로 처리해 확장 한글이 깨진다는 의심 | — | **문제 아님** | encoding.go 주석 | x/text 의 EUCKR 이 곧 CP949 |
| 7 | 닫은 뒤에도 session.data 이벤트가 옴 | 낮음 | 수정 | session/session.go | |
| 8 | 텔넷·시리얼 끊김 이유가 고정 문구로 덮임 | — | **의도(명세)** | — | protocols.md §4·§5 가 고정 문구 요구 |
| 9 | 로그 파일 별칭이 길거나 제어 문자를 품으면 열기 실패·이상한 이름 | 낮음 | 수정 | session/log.go | B 와 같이 |
| 10 | 로그 쓰기가 읽기 고루틴에서 동기 디스크 I/O | 낮음 | 수정 | session/log.go | 3 과 같이 |

**권장 작업 묶음** (한 묶음 = 한 커밋 단위):

1. 묶음 ① `log.go`: B + 3 + 9 + 10 (한 파일, 테스트 공유)
2. 묶음 ② `keepalive`: 1
3. 묶음 ③ `serial.go`: 2 (+5 의 UI 한 줄)
4. 묶음 ④ `local_windows.go`: 4
5. 묶음 ⑤ `session.go`: 7
6. 묶음 ⑥ 역압: A (가장 크므로 별도)
7. 6 은 주석 한 줄 + 테스트 한 개, 8 은 known-issues 에 dismissed 로만 기록

---

## 1. SSH keepalive 가 죽은 연결을 늦게 감지

### 증상
와이파이 전환·VPN 끊김·케이블 뽑힘처럼 **상대가 FIN/RST 없이 사라지는** 경우, 세션이 몇 분 동안 멈춘 채로 있고 자동 재접속도 시작되지 않는다. 설정의 keepalive 가 "4번 실패하면 끊김 처리" 인데 실제로는 그보다 훨씬 늦다.

### 원인
[keepalive_forward.go:22](../../internal/sshconn/keepalive_forward.go#L22) 가 `SendRequest("keepalive@openssh.com", true, nil)` 로 **응답을 기다리는** 요청을 보낸다. x/crypto/ssh 의 `SendRequest` 는 시한이 없어서 응답이 영영 안 오면 TCP 재전송이 포기할 때까지(Windows 기본 약 1~2분 이상) 막힌다. 그동안 ticker 루프는 다음 주기로 넘어가지 못하므로 "연속 4번 실패" 가 세어지지 않는다. 결국 끊김 감지는 keepalive 가 아니라 OS 의 TCP 타임아웃에 달려 있다.

또 하나: 사용자가 세션을 닫아도 keepalive 고루틴은 다음 4번 실패(4 × 주기)까지 살아 있다. 누수는 아니지만 불필요하다.

### 수정

`internal/sshconn/client.go` — `Conn` 에 닫힘 신호 추가:

```go
type Conn struct {
	Client     *ssh.Client
	Hops       []*ssh.Client
	Username   string
	Asked      askedSecret
	mtxSftp    sync.Mutex
	sftpClient *sftp.Client
	chClosed   chan struct{} // Close 때 닫힌다 (keepalive 루프 종료용)
	onceClose  sync.Once
}

// Close 는 대상과 모든 점프 연결을 닫는다 (protocols.md §2.4). 두 번 불러도 안전하다.
func (pConn *Conn) Close() error {
	var errFirst error
	pConn.onceClose.Do(func() {
		if pConn.chClosed != nil {
			close(pConn.chClosed)
		}
		if pConn.Client != nil {
			errFirst = pConn.Client.Close()
		}
		for iIndex := len(pConn.Hops) - 1; iIndex >= 0; iIndex-- {
			_ = pConn.Hops[iIndex].Close()
		}
	})
	return errFirst
}
```

`Connect` 에서 `Conn` 을 만드는 곳에 `chClosed: make(chan struct{})` 를 넣는다 (파일 안에서 `&Conn{` 를 찾아 추가).

`internal/sshconn/keepalive_forward.go` — 요청마다 시한을 둔다:

```go
// StartKeepalive 는 keepalive 요청을 주기적으로 보낸다. 연속 4번 실패하면 onFail 을 부른다 (protocols.md §2.6).
// 한 요청의 응답이 주기 안에 오지 않으면(네트워크가 조용히 끊김) 그 요청은 실패로 센다.
// iSec 가 0 이하이면 아무 일도 하지 않는다. 연결이 닫히면 루프를 끝낸다.
func (pConn *Conn) StartKeepalive(iSec int, onFail func()) {
	if iSec <= 0 {
		return
	}
	durInterval := time.Duration(iSec) * time.Second
	go func() {
		ticker := time.NewTicker(durInterval)
		defer ticker.Stop()
		iFailed := 0
		for {
			select {
			case <-pConn.chClosed:
				return
			case <-ticker.C:
			}
			if keepaliveOnce(fnSend, durInterval, pConn.chClosed) {
				iFailed = 0
				continue
			}
			iFailed++
			if iFailed >= G_iKeepaliveFailMax {
				onFail()
				return
			}
		}
	}()
}

// keepaliveOnce 는 fnSend 로 요청 하나를 보내고 durTimeout 안에 끝나면 그 결과(오류 없음 = 성공)를 돌려준다.
// 시한이 지나거나 chClosed 가 닫히면 false 다. 늦게 끝나는 fnSend 고루틴은 버퍼 1 채널에 결과를 넣고 스스로 끝난다.
// 보내기 함수를 받는 이유는 테스트에서 SSH 없이 "응답 없음" 을 흉내 내기 위해서다.
func keepaliveOnce(fnSend func() error, durTimeout time.Duration, chClosed <-chan struct{}) bool {
	chResult := make(chan error, 1)
	go func() { chResult <- fnSend() }()
	timer := time.NewTimer(durTimeout)
	defer timer.Stop()
	select {
	case errSend := <-chResult:
		return errSend == nil
	case <-timer.C:
		return false
	case <-chClosed:
		return false
	}
}
```

`StartKeepalive` 의 고루틴 시작 전에 보내기 함수를 만든다:

```go
	fnSend := func() error {
		_, _, errSend := pConn.Client.SendRequest("keepalive@openssh.com", true, nil)
		return errSend
	}
```

> 결과: 주기 15초면 늦어도 약 60초(4 × 15초) 안에 `onFail` → `sshSession.Close()` → 셸 Read 가 끝나 session 이 끊김 처리와 자동 재접속을 시작한다. `onFail` 이 부르는 `sshSession.Close` 는 `Shell.Close`(Once)·`conn.Close`(Once) 라 중복 호출에 안전하다.

### 테스트

`internal/sshconn/sshconn_test.go` 에 추가. `keepaliveOnce` 가 보내기 함수를 받으므로 SSH 서버 없이 "응답 없음"·"즉시 실패"·"닫힘" 세 경로를 검증한다.

```go
// 응답이 없으면 시한 안에 false 로 돌아온다 (§1 핵심).
func TestKeepaliveOnceTimesOut(t *testing.T) {
	chNever := make(chan struct{})
	fnBlock := func() error { <-chNever; return nil } // 영원히 응답하지 않는 요청
	defer close(chNever)
	tStart := time.Now()
	if keepaliveOnce(fnBlock, 100*time.Millisecond, make(chan struct{})) {
		t.Fatal("응답이 없으면 실패여야 함")
	}
	if durTook := time.Since(tStart); durTook > 500*time.Millisecond {
		t.Fatalf("시한 안에 돌아와야 함: %v", durTook)
	}
}

// 즉시 성공·즉시 실패는 그대로 전달한다.
func TestKeepaliveOnceReturnsResult(t *testing.T) {
	if !keepaliveOnce(func() error { return nil }, time.Second, make(chan struct{})) {
		t.Fatal("성공이어야 함")
	}
	if keepaliveOnce(func() error { return errors.New("eof") }, time.Second, make(chan struct{})) {
		t.Fatal("실패여야 함")
	}
}

// 연결을 닫으면 keepalive 루프가 바로 끝나 onFail 이 불리지 않는다.
func TestKeepaliveStopsOnClose(t *testing.T) {
	pConn := &Conn{chClosed: make(chan struct{})}
	chFail := make(chan struct{}, 1)
	pConn.StartKeepalive(60, func() { chFail <- struct{}{} }) // Client 가 nil 이라 첫 tick(60초) 전에 닫아야 한다
	_ = pConn.Close()
	select {
	case <-chFail:
		t.Fatal("닫은 뒤에는 onFail 이 불리면 안 됨")
	case <-time.After(200 * time.Millisecond):
	}
}
```

> **수동 검증**: 서버에 접속 → keepalive 15초 → 서버 쪽에서 `iptables -A INPUT -p tcp --dport 22 -j DROP`(또는 클라이언트 PC 의 방화벽으로 해당 서버 IP 차단) → **60~75초 안에** "연결이 종료되었습니다" 안내와 재접속 카운트다운이 떠야 한다. 수정 전에는 몇 분이 걸린다. 끝나면 규칙을 되돌린다.

---

## 2. 시리얼 로컬 에코가 다음 수신 데이터까지 안 보임

### 증상
"입력 글자 표시 (로컬 에코)" 를 켜고 조용한 장치(아무것도 보내지 않는 장비)에 타이핑하면 글자가 화면에 안 나온다. 장치가 뭔가 보내는 순간 밀렸던 글자가 한꺼번에 나온다.

### 원인
[serial.go:135-143](../../internal/transport/serial.go#L135-L143) `Write` 는 에코 텍스트를 `bPendingOut` 에 쌓기만 한다. 그것을 화면으로 보내는 건 [serial.go:85-108](../../internal/transport/serial.go#L85-L108) `Read` 인데, `Read` 는 들어올 때 한 번 `bPendingOut` 을 보고 없으면 `port.Read` 에 들어가 **데이터가 올 때까지 막힌다**. 포트를 열 때 읽기 시한을 두지 않았으므로(`SetReadTimeout` 미호출, 기본은 무한 대기) 장치가 조용하면 `Read` 가 돌아오지 않고, 그 사이 쌓인 에코는 다음 수신 때까지 못 나간다.

### 수정 (`internal/transport/serial.go`)

열 때 읽기 시한을 둔다. go.bug.st/serial v1.8.0 은 시한이 지나면 `Read` 가 `(0, nil)` 로 돌아온다(라이브러리 README·Windows 구현의 동작. `Port` 인터페이스 주석에는 명시가 없으므로 **구현 때 실제 포트로 한 번 확인**한다. `(0, 오류)` 로 돌아오는 환경이면 아래 `Read` 의 `errRead != nil` 분기 앞에 시한 오류를 구분하는 조건을 둔다).

```go
// G_cdurSerialReadTick 은 읽기 시한이다. 이 주기로 Read 가 돌아와 대기 중인 로컬 에코를 내보낸다.
const G_cdurSerialReadTick = 50 * time.Millisecond

func OpenSerial(opts SerialOptions) (*SerialConn, error) {
	// ... mode 구성 그대로 ...
	port, errOpen := serial.Open(opts.Path, mode)
	if errOpen != nil {
		return nil, fmt.Errorf("%s: %v", opts.Path, errOpen)
	}
	if errTimeout := port.SetReadTimeout(G_cdurSerialReadTick); errTimeout != nil {
		_ = port.Close()
		return nil, fmt.Errorf("%s: %v", opts.Path, errTimeout)
	}
	return &SerialConn{port: port, opts: opts}, nil
}

// takePending 은 대기 중인 출력(로컬 에코·남은 조각)을 bBuf 로 옮기고 옮긴 길이를 돌려준다.
func (pSerial *SerialConn) takePending(bBuf []byte) int {
	pSerial.mtxState.Lock()
	defer pSerial.mtxState.Unlock()
	if len(pSerial.bPendingOut) == 0 {
		return 0
	}
	nCopy := copy(bBuf, pSerial.bPendingOut)
	pSerial.bPendingOut = pSerial.bPendingOut[nCopy:]
	if len(pSerial.bPendingOut) == 0 {
		pSerial.bPendingOut = nil
	}
	return nCopy
}

// Read 는 포트 출력을 읽는다. 로컬 에코가 켜져 있으면 보낸 내용을 화면에도 보여 준다 (Write 에서 쌓는다).
// 읽기 시한(G_cdurSerialReadTick)마다 돌아와 대기 중인 에코가 있으면 먼저 내보낸다.
func (pSerial *SerialConn) Read(bBuf []byte) (int, error) {
	bRaw := make([]byte, 4096)
	for {
		if nPending := pSerial.takePending(bBuf); nPending > 0 {
			return nPending, nil
		}
		nRaw, errRead := pSerial.port.Read(bRaw)
		if errRead != nil {
			return 0, ErrPortClosed
		}
		if nRaw == 0 {
			continue // 시한: 데이터 없음. 위에서 대기분을 다시 본다.
		}
		nCopy := copy(bBuf, bRaw[:nRaw])
		if nCopy < nRaw {
			pSerial.mtxState.Lock()
			pSerial.bPendingOut = append(pSerial.bPendingOut, bRaw[nCopy:nRaw]...)
			pSerial.mtxState.Unlock()
		}
		return nCopy, nil
	}
}
```

`Write` 는 그대로. `import "time"` 추가.

> 50ms 는 타이핑 체감 지연이 없는 값이다. 시한 루프는 데이터가 없을 때 50ms 마다 한 번 깨어나므로 CPU 부담은 무시할 수준이다. 포트를 닫으면 `port.Read` 가 오류를 돌려줘 루프가 끝난다(기존과 같음).

### 테스트
- 단위: `SerialConn` 은 실제 포트가 필요해 `serial.Port` 인터페이스를 가짜로 만든다. `Read` 가 `(0, nil)` 을 돌려주는 가짜 포트에 `Write("ab\r")`(로컬 에코 켬) 뒤 `Read` 를 부르면 **데이터 없이도** `"ab\r\n"` 이 돌아와야 한다. 가짜 포트는 `go.bug.st/serial.Port` 의 메서드 전부(Read/Write/Close/SetMode/SetDTR/SetRTS/GetModemStatusBits/SetReadTimeout/ResetInputBuffer/ResetOutputBuffer/Break/Drain) 를 빈 구현으로 둔다.
- 수동: 아무것도 보내지 않는 장치(또는 루프백 없이 열어 둔 USB-시리얼)에 로컬 에코 켜고 타이핑 → 즉시 보임. 루프백(TX-RX 단락) 장치에서는 에코 + 장치 회신으로 글자가 **두 번** 보이는 게 정상(명세대로).

---

## 3. 로그에 ANSI 조각이 남음 + 10. 로그 동기 디스크 쓰기

### 증상
세션 로그 파일에 `[31m`, `[0m`, `[?2004h` 같은 조각이 가끔 남는다. 고속 출력일수록 자주 보인다. 또 로그가 켜진 세션은 느린 디스크(네트워크 드라이브의 %APPDATA% 등)에서 출력이 더 밀린다.

### 원인
- [log.go:18-21](../../internal/session/log.go#L18-L21) `StripForLog` 가 **묶음 하나**(최대 12ms 분량) 안에서만 정규식으로 시퀀스를 지운다. 시퀀스가 묶음 경계에서 `\x1b[3` / `1m` 처럼 잘리면 앞 조각은 ESC 하나만 남은 꼴이라 패턴에 안 맞고(`\x1b[` 뒤에 종결 문자가 없음), 뒤 조각은 ESC 가 없어 일반 글자로 남는다.
- [log.go:50-57](../../internal/session/log.go#L50-L57) `Write` 가 세션의 읽기 고루틴에서 `os.File.WriteString` 을 직접 부른다. 묶음(12ms)마다 syscall 한 번이고 디스크가 느리면 그만큼 읽기가 늦어진다.

### 수정 (`internal/session/log.go`)

로거가 **미완성 시퀀스 꼬리를 보관**했다가 다음 묶음과 이어 처리하고, `bufio.Writer` 로 모아 쓰되 1초마다·닫을 때 flush 한다.

```go
import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// G_ciLogBufferBytes 는 로그 쓰기 버퍼 크기다. G_cdurLogFlush 마다, 그리고 닫을 때 비운다.
const (
	G_ciLogBufferBytes = 64 * 1024
	G_cdurLogFlush     = time.Second
	// g_ciLogCarryMax 는 미완성 ANSI 시퀀스로 보관하는 최대 길이다. 넘으면 시퀀스가 아니라고 보고 그대로 기록한다.
	g_ciLogCarryMax = 256
)

// Logger 는 세션 로그 파일을 이어 쓴다. ANSI 시퀀스가 묶음 경계에서 잘려도 지울 수 있도록 꼬리를 보관한다.
type Logger struct {
	mtxData    sync.Mutex
	pFile      *os.File
	pWriter    *bufio.Writer
	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
	timerFlush *time.Timer // 버퍼에 내용이 있을 때만 걸린다
}

func OpenLog(szDataDir string, szAlias string, tNow time.Time) (*Logger, error) {
	// ... KI-001 계획서(§3.3)의 O_EXCL 재시도 루프 그대로, 성공 시:
	//     return &Logger{pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes)}, nil
}

// splitAnsiTail 은 텍스트 끝에 있는 미완성 ANSI 시퀀스를 떼어 낸다 (본문, 꼬리).
// 마지막 ESC 뒤가 완성된 시퀀스이면(또는 ESC 가 없으면) 꼬리는 비어 있다.
func splitAnsiTail(szIn string) (string, string) {
	iEsc := strings.LastIndexByte(szIn, 0x1b)
	if iEsc < 0 {
		return szIn, ""
	}
	szRest := szIn[iEsc:]
	if len(szRest) > g_ciLogCarryMax {
		return szIn, "" // 너무 길다: 시퀀스가 아니라 본문에 섞인 ESC 로 본다
	}
	if loc := g_reAnsi.FindStringIndex(szRest); loc != nil && loc[0] == 0 {
		return szIn, "" // ESC 에서 시작하는 완성된 시퀀스가 있다 (뒤는 일반 글자)
	}
	return szIn[:iEsc], szRest
}

// Write 는 출력 조각을 정리해 기록한다. 끝이 미완성 시퀀스면 보관했다가 다음 조각과 이어 처리한다.
func (pLogger *Logger) Write(szText string) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pWriter == nil {
		return
	}
	szBody, szTail := splitAnsiTail(pLogger.szCarry + szText)
	pLogger.szCarry = szTail
	if szBody == "" {
		return
	}
	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
	if pLogger.timerFlush == nil {
		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
	}
}

// flushTimer 는 주기 flush 다. 닫힌 뒤에는 아무 일도 하지 않는다.
func (pLogger *Logger) flushTimer() {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	pLogger.timerFlush = nil
	if pLogger.pWriter != nil {
		_ = pLogger.pWriter.Flush()
	}
}

// Close 는 남은 꼬리와 버퍼를 쓰고 파일을 닫는다. 두 번 불러도 안전하다.
func (pLogger *Logger) Close() error {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pFile == nil {
		return nil
	}
	if pLogger.timerFlush != nil {
		pLogger.timerFlush.Stop()
		pLogger.timerFlush = nil
	}
	if pLogger.szCarry != "" {
		_, _ = pLogger.pWriter.WriteString(StripForLog(pLogger.szCarry))
		pLogger.szCarry = ""
	}
	_ = pLogger.pWriter.Flush()
	errClose := pLogger.pFile.Close()
	pLogger.pFile = nil
	pLogger.pWriter = nil
	return errClose
}
```

> `g_reAnsi` 는 기존 것을 그대로 쓴다 (CSI·OSC·2바이트 ESC). `szCarry + szText` 문자열 결합은 꼬리가 최대 256바이트라 비용이 작다. 앱이 비정상 종료하면 마지막 1초 분량이 유실될 수 있다. 세션 로그 용도에서는 허용 범위다 (기존에도 OS 캐시 유실은 있었다).

### 테스트 (`session_test.go`)

```go
// 묶음 경계에서 잘린 ANSI 시퀀스도 지워진다 (§3).
func TestLogStripsSequenceSplitAcrossWrites(t *testing.T) {
	szDir := t.TempDir()
	pLogger, errOpen := OpenLog(szDir, "x", time.Date(2026, 11, 16, 1, 2, 3, 0, time.Local))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	pLogger.Write("a\x1b[3")   // CSI 가 중간에서 잘림
	pLogger.Write("1mred\x1b[") // 앞 조각 완성 + 또 잘림
	pLogger.Write("0m\x1b]0;title") // 앞 조각 완성 + OSC 미종결
	pLogger.Write("\x07b\r\n")      // OSC 종결
	_ = pLogger.Close()
	slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "x-*.log"))
	bData, _ := os.ReadFile(slFiles[0])
	if string(bData) != "aredb\n" {
		t.Fatalf("잘린 시퀀스가 남음: %q", string(bData))
	}
}

// 시퀀스가 아닌 ESC(256바이트 넘게 종결 없음)는 보관하지 않고 그대로 기록한다.
func TestLogCarryGivesUpOnLongTail(t *testing.T) {
	szBody, szTail := splitAnsiTail("\x1b" + strings.Repeat("x", 300))
	if szTail != "" || len(szBody) != 301 {
		t.Fatalf("긴 꼬리는 본문으로 처리해야 함: body=%d tail=%q", len(szBody), szTail)
	}
}

// 1초 안에 flush 되어 파일에서 읽힌다 (§10).
func TestLogFlushesPeriodically(t *testing.T) {
	szDir := t.TempDir()
	pLogger, _ := OpenLog(szDir, "x", time.Now())
	defer pLogger.Close()
	pLogger.Write("hello\n")
	waitUntil(t, func() bool {
		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "x-*.log"))
		bData, _ := os.ReadFile(slFiles[0])
		return string(bData) == "hello\n"
	}, 2*time.Second)
}
```

기존 `TestLogStripAndFileName` 의 `pLogger.Write("x"); pLogger.Close()` 는 Close 가 flush 하므로 그대로 통과한다.

---

## 4. 로컬 셸 Close 뒤 닫힌 프로세스 핸들 사용

### 증상
보통은 드러나지 않는다. 드물게 로컬 탭을 닫을 때 종료 이유 문구가 엉뚱하거나, 이론상 핸들 값이 재사용되면 다른 객체를 기다릴 수 있다(앱이 잠깐 멈춤).

### 원인
[local_windows.go:165-173](../../internal/transport/local_windows.go#L165-L173) `Close` 가 `CloseHandle(hProcess)` 까지 닫는다. 그런데 읽기 고루틴은 `Read` 가 실패하면 [exitReason](../../internal/transport/local_windows.go#L135-L142) 에서 **같은 `hProcess`** 로 `WaitForSingleObject`·`GetExitCodeProcess` 를 부른다. 사용자가 닫는 순서는 `Close`(핸들 닫음) → 파이프가 끊겨 `Read` 실패 → `exitReason`(닫힌 핸들 사용) 이다. Windows 는 닫힌 핸들 값을 곧바로 재사용할 수 있어 use-after-close 다.

### 수정 (`internal/transport/local_windows.go`)

프로세스 핸들을 뮤텍스로 보호하고, `Close` 는 `TerminateProcess` 뒤에 핸들을 닫되 `exitReason` 은 닫힌 뒤에는 핸들을 만지지 않는다.

```go
type LocalConn struct {
	hPty        windows.Handle
	hProcess    windows.Handle
	hInWrite    windows.Handle
	hOutRead    windows.Handle
	mtxWrite    sync.Mutex
	mtxClose    sync.Once
	mtxProcess  sync.Mutex // hProcess 사용·닫기를 배타한다
	bProcClosed bool
}

// exitReason 은 셸 종료 이유 문구를 만든다 (protocols.md §6). 사용자가 닫아 핸들이 이미 없으면 일반 문구다.
func (pLocal *LocalConn) exitReason() error {
	pLocal.mtxProcess.Lock()
	defer pLocal.mtxProcess.Unlock()
	if pLocal.bProcClosed {
		return i18n.Errorf("be.shell.exited", nil)
	}
	_, _ = windows.WaitForSingleObject(pLocal.hProcess, windows.INFINITE)
	var dwCode uint32
	if errCode := windows.GetExitCodeProcess(pLocal.hProcess, &dwCode); errCode == nil && dwCode != 0 {
		return i18n.Errorf("be.shell.exitedCode", nil, "code", dwCode)
	}
	return i18n.Errorf("be.shell.exited", nil)
}

// Close 는 셸을 강제 종료하고 핸들을 닫는다 (사용자가 닫을 때).
func (pLocal *LocalConn) Close() error {
	pLocal.mtxClose.Do(func() {
		_ = windows.TerminateProcess(pLocal.hProcess, 1) // 먼저 끝내야 exitReason 의 Wait 이 바로 풀린다
		go g_procClosePseudoConsole.Call(uintptr(pLocal.hPty))
		_ = windows.CloseHandle(pLocal.hInWrite)
		_ = windows.CloseHandle(pLocal.hOutRead)
		pLocal.mtxProcess.Lock()
		_ = windows.CloseHandle(pLocal.hProcess)
		pLocal.bProcClosed = true
		pLocal.mtxProcess.Unlock()
	})
	return nil
}
```

> `TerminateProcess` 를 잠금 **밖**에서 먼저 부르는 이유: `exitReason` 이 잠금을 쥔 채 `WaitForSingleObject(INFINITE)` 에 있을 수 있는데, 프로세스가 끝나야 그 대기가 풀리고 잠금이 풀린다. 순서를 바꾸면 교착한다.

### 테스트
기존 [local_windows_test.go:13](../../internal/transport/local_windows_test.go#L13) `TestLocalConPtyEchoAndExit` 에 "Read 가 막혀 있는 동안 Close → Read 가 오류로 돌아오고 exitReason 이 panic 없이 끝난다" 케이스를 추가한다:

```go
func TestLocalCloseWhileReading(t *testing.T) {
	pLocal, errOpen := OpenLocal(LocalOptions{Shell: "cmd.exe", Cols: 80, Rows: 24})
	if errOpen != nil {
		t.Skip(errOpen)
	}
	chErr := make(chan error, 1)
	go func() {
		bBuf := make([]byte, 4096)
		for {
			if _, errRead := pLocal.Read(bBuf); errRead != nil {
				chErr <- errRead
				return
			}
		}
	}()
	time.Sleep(300 * time.Millisecond) // 셸이 프롬프트를 내고 Read 가 막힐 때까지
	_ = pLocal.Close()
	select {
	case <-chErr:
	case <-time.After(5 * time.Second):
		t.Fatal("Close 뒤 Read 가 돌아오지 않음")
	}
}
```

---

## 5. 시리얼 흐름 제어 설정이 동작에 안 쓰임

### 증상
서버 편집의 "흐름 제어" 를 RTS/CTS 나 XON/XOFF 로 골라도 실제 통신은 "없음" 과 같다. 하드웨어 흐름 제어가 필요한 장비에서 데이터가 깨질 수 있다.

### 원인
[serial.go:38-58](../../internal/transport/serial.go#L38-L58) `OpenSerial` 이 `flowControl` 을 읽지 않는다. 사용 중인 go.bug.st/serial v1.8.0 의 `Mode` 에는 흐름 제어 필드가 없다(RTS/DTR 을 수동으로 켜고 끄는 `SetRTS`/`SetDTR` 과 `GetModemStatusBits` 만 있다). [spec/protocols.md §5](../../spec/protocols.md) 는 이를 "저장하지만 현재 동작에 쓰지 않는다(알려진 한계)" 로 이미 적어 두었다.

### 처리 (권장: UI 안내만, 구현은 후속)
명세에 한계로 명시돼 있으므로 버그가 아니다. 다만 사용자가 모른 채 고를 수 있어 **드롭다운 옆에 안내 한 줄**을 둔다.

- `frontend/src/screens/HostEditorParts.tsx` 의 흐름 제어 `<select>` 아래에 도움말: `t('host.serial.flowNotYet')` = "현재 버전은 흐름 제어 설정을 적용하지 않습니다(저장만)". 10개 언어 `locales/*.json` 에 키 추가 (D75).
- 명세 `spec/ui/host-editor.md` §시리얼 필드에 같은 도움말 문구를 적는다.

실제 구현이 필요해지면: XON/XOFF 는 소프트웨어로 가능(수신 측: 역압 계획 A 의 "읽기 멈춤" 시점에 XOFF(0x13), 재개 시 XON(0x11) 전송 / 송신 측: 장치가 보낸 XOFF 를 받으면 `Write` 를 XON 까지 보류). RTS/CTS 는 `SetRTS` 로 수신 가능 상태를 알리고 `GetModemStatusBits().CTS` 를 폴링해 송신을 보류하는 식으로 흉내 낼 수 있으나 OS 드라이버 수준의 하드웨어 흐름 제어만큼 정확하지 않다. 별도 과제로 둔다.

---

## 6. cp949 → EUC-KR 처리로 확장 한글이 깨진다는 의심 — 문제 아님

### 확인 결과
[encoding.go:16](../../internal/session/encoding.go#L16) 주석은 "cp949 는 EUC-KR 디코더로 처리한다 (x/text 에 별도 CP949 가 없음, 알려진 한계)" 이지만, x/text v0.44.0 의 `korean.EUCKR` 문서는 **"EUC-KR encoding, also known as Code Page 949"** 다. WHATWG 의 euc-kr 표가 곧 Windows-949(UHC) 전체라서 확장 한글("똠", "펲" 등)도 디코딩된다. 즉 동작에는 문제가 없고 주석만 틀렸다.

### 처리
주석을 고치고, 확장 한글 한 글자로 회귀 테스트를 둔다.

```go
// cp949 와 euc-kr 은 같은 디코더를 쓴다: x/text 의 EUCKR 은 Code Page 949(UHC) 전체 표다.
```

```go
// cp949 확장 영역의 한글("똠" = 0x8C 0x63)도 디코딩된다 (§6).
func TestDecoderHandlesCp949Extension(t *testing.T) {
	pDecoder := NewDecoder("cp949")
	if got := pDecoder.Decode([]byte{0x8C, 0x63}); got != "똠" {
		t.Fatalf("cp949 확장 한글: %q", got)
	}
}
```

> 바이트 값은 테스트 작성 시 `korean.EUCKR.NewEncoder().Bytes([]byte("똠"))` 로 확인한다.

---

## 7. 닫은 뒤에도 session.data 이벤트가 옴

### 증상
세션을 닫은 직후 그 세션 id 로 `session.data` 가 한두 번 더 온다. 프런트는 writer 가 없으면 버리므로 화면에는 안 보이지만, 역압(계획 A) 도입 후에는 "모르는 세션의 ack" 가 되어 로그에 남을 수 있다.

### 원인
- [session.go:197-213](../../internal/session/session.go#L197-L213) `readLoop` 가 `Close` 의 `batcher.Flush()` 뒤에 마지막 조각을 `writeOutput` 할 수 있다 (`Close` 는 transport 를 닫고 Flush 하지만, 읽기 고루틴이 이미 읽어 둔 조각은 그 뒤에 들어간다).
- [session.go:149-152](../../internal/session/session.go#L149-L152) `onConnectFailed` 가 **세대 검사 전에** 실패 안내를 `writeOutput` 한다. 닫은 뒤 늦게 끝난 접속 시도의 안내가 나간다.

### 수정 (`internal/session/session.go`)

```go
// isCurrent 는 이 세대가 아직 유효하고 사용자가 닫지 않았는지 본다.
func (pSession *Session) isCurrent(iGen int) bool {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	return iGen == pSession.gen && !pSession.userClosed
}

func (pSession *Session) readLoop(transport Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	for {
		// (계획 A 의 pSession.credit.Wait() 가 여기 들어간다)
		nRead, errRead := transport.Read(bBuf)
		if nRead > 0 && pSession.isCurrent(iGen) { // 닫힌 뒤 읽힌 조각은 버린다
			pSession.writeOutput(pSession.decoder.Decode(bBuf[:nRead]))
		}
		if errRead != nil {
			// ... 기존 그대로 ...
		}
	}
}

func (pSession *Session) onConnectFailed(errDial error, iGen int, bLocal bool) {
	if !pSession.isCurrent(iGen) {
		return // 닫았거나 다른 시도가 시작됐다: 안내도 내지 않는다
	}
	szReason := errDial.Error()
	if bLocal {
		pSession.writeOutput(MsgShellFailed(szReason))
	} else {
		pSession.writeOutput(MsgConnectFailed(szReason))
	}
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed { // 안내를 쓰는 사이 닫혔을 수 있다
		pSession.mtxData.Unlock()
		return
	}
	// ... 기존 그대로 ...
}
```

### 테스트
기존 `TestConnectOutputAndClose` 에 "Close 뒤 fakeTransport 에 조각을 넣어도 `session.data` 가 더 오지 않는다" 를 추가한다: Close → `chIn <- []byte("late")` → 100ms 대기 → recorder 출력에 "late" 없음.

---

## 8. 텔넷·시리얼 끊김 이유가 고정 문구 — 의도(명세)

[telnet.go:96](../../internal/transport/telnet.go#L96) 과 [serial.go:99](../../internal/transport/serial.go#L99) 가 원래 오류를 `ErrRemoteClosed`/`ErrPortClosed` 로 바꾸는 것은 [spec/protocols.md](../../spec/protocols.md) §4 "연결이 끊기면 이유 '연결이 종료되었습니다'", §5 "포트가 닫히면 이유 '포트가 닫혔습니다'" 를 그대로 구현한 것이다. **수정하지 않는다.** known-issues 에 적지 않거나, 적으면 `dismissed` 로 둔다 (사유: 명세의 고정 문구).

---

## 9. 로그 파일 별칭의 길이·제어 문자

### 증상
별칭이 아주 길면(예: 200자 넘는 설명을 별칭에 넣은 경우) `%APPDATA%\DariTerm\data\logs\<별칭>-<시각>.log` 가 Windows 경로 한도(260자)를 넘어 로그 열기가 **조용히** 실패하고(로그 없이 세션이 열림), 별칭에 탭·개행 같은 제어 문자가 있으면 그대로 파일 이름에 들어간다.

### 원인
[log.go:14](../../internal/session/log.go#L14) `g_reFileNameBad` 가 `\ / : * ? " < > |` 만 바꾸고, 길이 제한이 없다. [session.go:56](../../internal/session/session.go#L56) 은 실패를 무시한다(명세대로).

### 수정 (`internal/session/log.go`, KI-001 계획서 §3.2 의 `LogFileName` 에 포함)

```go
// g_ciLogAliasMaxRunes 는 로그 파일 이름에 쓰는 별칭의 최대 길이다 (경로 260자 한도 대비).
const g_ciLogAliasMaxRunes = 80

// g_reFileNameBad 는 로그 파일 이름에서 금지 문자(`\ / : * ? " < > |`)와 제어 문자를 `_` 로 바꾸는 규칙이다.
var g_reFileNameBad = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f\x7f]`)

// logAlias 는 별칭을 파일 이름용으로 다듬는다: 금지·제어 문자 치환, 앞뒤 공백·점 제거, 길이 제한, 비면 "session".
func logAlias(szAlias string) string {
	szOut := g_reFileNameBad.ReplaceAllString(szAlias, "_")
	szOut = strings.Trim(szOut, " .") // Windows 는 끝의 공백·점을 허용하지 않는다
	if slRunes := []rune(szOut); len(slRunes) > g_ciLogAliasMaxRunes {
		szOut = string(slRunes[:g_ciLogAliasMaxRunes])
	}
	if szOut == "" {
		return "session"
	}
	return szOut
}

// LogFileName 안에서: szBase := logAlias(szAlias) + "-" + tNow.Format("20060102-150405")
```

명세 §6 의 괄호 문구를 "별칭의 `\ / : * ? " < > |` 와 제어 문자는 `_` 로 바꾸고, 앞뒤 공백·점을 지우며, 80자로 자른다. 비면 `session`" 으로 바꾼다.

### 테스트
`TestLogFileNameSeq` 에 케이스 추가: `LogFileName("  a\tb. ", t, 1)` → `a_b-….log`, 100자 별칭 → 80자로 잘림, `""` → `session-….log`.

---

## 10. 로그 쓰기가 읽기 고루틴에서 동기 I/O
§3 에 포함했다 (`bufio.Writer` + 1초 flush). 별도 작업 없음.

---

## 11. 명세 변경 모음 (명세 작성자 반영)

| 문서 | 변경 |
|---|---|
| `spec/behavior/session.md` §6 | KI-001 계획서 §2.1 문구 + §9 의 별칭 정리 규칙 + "로그는 1초 단위로 디스크에 쓴다" |
| `spec/protocols.md` §2.6 (SSH keepalive) | "요청의 응답이 keepalive 주기 안에 오지 않으면 실패로 센다" 한 문장 추가 |
| `spec/protocols.md` §5 (시리얼) | "읽기는 50ms 시한으로 돌아와 대기 중인 로컬 에코를 내보낸다" 추가 |
| `spec/ui/host-editor.md` 시리얼 | 흐름 제어 도움말 문구 (§5) |

---

## 12. 기록

| 파일 | 내용 |
|---|---|
| `.claude/known-issues.md` | 1·2·3·4·7·9 를 KI-002~KI-007 로 `active` 등록(착수 시), 8 은 `dismissed`(명세), 6 은 등록하지 않음(문제 아님). 완료 시 `resolved` |
| `docs/decisions.md` | D86: keepalive 응답 시한 / D87: 시리얼 읽기 시한 50ms 와 에코 처리 / D88: 로그 꼬리 보관·버퍼 쓰기 / D89: 로컬 셸 핸들 배타 (D84 역압, D85 KI-001 다음) |
| `.claude/wip/` | 묶음별로 WIP 하나씩 (공통 규칙 §4) |

---

## 13. 전체 검증 순서

```bash
go build ./... && go vet ./... && go test -race ./...
(cd frontend && npx tsc --noEmit && npx vitest run)
wails build
```

수동 검증 요약 (각 항목의 "테스트" 절 참고):

| # | 핵심 확인 |
|---|---|
| 1 | 서버 IP 차단 → 60~75초 안에 끊김 안내·재접속 시작 |
| 2 | 조용한 시리얼 장치에 로컬 에코 켜고 타이핑 → 즉시 보임 |
| 3 | `yes` 로 폭주 출력 후 로그 파일에 `[31m` 류 조각 없음 (`grep -c $'\x1b' 파일` 이 0) |
| 4 | 로컬 탭을 셸이 프롬프트를 낸 상태에서 닫기 → 즉시 닫힘, 종료 문구 정상 |
| 7 | 탭 닫기 직후 DevTools 콘솔에 닫힌 id 의 session.data 없음 (역압 적용 후엔 "모르는 세션 ack" 로그 없음) |
| 9 | 별칭 200자·탭 문자 포함 서버로 로그 켜고 접속 → logs 폴더에 80자 이름 파일 생성 |
