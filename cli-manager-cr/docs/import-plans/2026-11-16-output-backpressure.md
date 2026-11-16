# 고속 출력 세션이 앱 전체를 느리게 하는 문제 — 원인·수정 계획·코드·테스트

**작성**: 2026-11-16 · **상태**: 완료(2026-11-16, D84~D90) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/ipc.md](../../spec/ipc.md) §1·§2·§4, [spec/behavior/session.md](../../spec/behavior/session.md) §8, 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고치는지(코드 포함) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 착수할 때는 `.claude/wip/<날짜>-output-backpressure.md` 를 `_template.md` 로 만들고 §7 체크리스트를 그대로 옮긴다.

---

## 0. 한 줄 요약

세션 하나가 초당 수 MB 를 출력하면 **다른 탭의 키 입력·출력·UI 전체가 같이 느려지고 메모리가 불어난다.**
원인은 "Go 가 읽는 만큼 무조건 프런트로 밀어 넣는 구조(역압 없음) + 웹뷰 JS 스레드 하나 + DOM 렌더러" 다.
수정은 4가지를 한 번에 넣는다. 효과 순서: **① 역압(ack 기반 흐름 제어) ≫ ② WebGL 렌더러 > ④ 숨은 탭 출력 지연 > ③ 묶음 크기 상한**.

| # | 수정 | 바뀌는 층 | 효과 |
|---|---|---|---|
| ① | 프런트가 처리한 만큼만 Go 가 읽는다 (ack 크레딧) | Go `internal/session`, `internal/app`, 프런트 `state/workspace.ts`, `screens/TerminalView.tsx`, 명세 | 메모리 상한·앱 전체 멈춤 해소. 폭주 프로세스는 TCP/PTY 버퍼에 막혀 **자연스럽게 느려진다** (실제 터미널과 같은 동작) |
| ② | xterm WebGL 애드온 (실패 시 DOM 유지) | 프런트 `TerminalView.tsx`, `package.json` | 렌더 비용 수 배 감소 |
| ③ | 12ms 묶음에 256KB 상한 | Go `batch.go` | 한 이벤트의 JSON 직렬화·JS 실행 시간 상한 |
| ④ | 안 보이는 세션은 100ms 단위로 몰아서 쓴다 | 프런트 `workspace.ts`, `TerminalView.tsx` | 보이는 탭에 프레임 시간을 양보 |

**적용 후 남는 한계**: 폭주 세션의 파싱은 여전히 JS 메인 스레드에서 돌므로 "완전 격리"는 아니다. 다만 xterm.js 가 12ms 마다 양보하고 ① 이 떠 있는 데이터 양을 묶어 주므로 UI 는 반응한다. 폭주 세션의 화면은 실제보다 뒤처지되 데이터는 버리지 않는다.

---

## 1. 원인 분석 (Cause) — 코드 경로

### 1.1 데이터 흐름 (현재)

```
[원격/PTY] → transport.Read(32KB)            internal/session/session.go:197 readLoop (세션별 고루틴)
   → Decoder.Decode → writeOutput               session.go:73
   → Batcher.Add (12ms 타이머)                   internal/session/batch.go:26
   → emit("session.data", id, text)              session.go:45  ← 역압 없음: 프런트 상태와 무관하게 계속 보냄
   → ipc.Router.Emit → main.go Emitter.Emit      main.go:34 runtime.EventsEmit (JSON 직렬화 → WebView2 UI 스레드에서 JS 실행)
   → EventsOn('session.data')                    frontend/src/state/actions.ts:219
   → useWorkspace.writeOutput                    frontend/src/state/workspace.ts:158
   → writers[id](text) = xterm.write(text)       frontend/src/screens/TerminalView.tsx:263  ← 완료 콜백 없음
   → xterm.js 파서·버퍼·DOM 렌더러               (JS 메인 스레드, 앱 전체가 공유)
```

### 1.2 왜 전체 앱에 영향을 주는가

1. **역압이 없다.** `readLoop` 는 프런트가 소화했는지 보지 않고 `Read → emit` 을 반복한다 ([session.go:197-213](../../internal/session/session.go#L197-L213)). 12ms 마다 쌓인 전부를 한 이벤트로 보낸다 ([batch.go:26-37](../../internal/session/batch.go#L26-L37)). 초당 3MB 면 12ms 마다 약 36KB 짜리 이벤트가 쉼 없이 간다.
2. **이벤트 통로가 앱에 하나다.** `runtime.EventsEmit` 은 인자를 JSON 으로 만들어 WebView2 UI 스레드에서 JS 를 실행한다 ([main.go:28-36](../../main.go#L28-L36)). `session.state`·토스트·**다른 세션의 `session.data`** 가 전부 같은 줄에 선다.
3. **JS 메인 스레드가 하나다.** 받은 텍스트를 `xterm.write` 로 넘기면 파싱·버퍼 갱신·렌더가 메인 스레드에서 돈다. 한 세션이 프레임 시간을 다 먹으면 React 렌더·다른 xterm·키 입력이 밀린다.
4. **프런트가 못 따라가면 xterm 내부 쓰기 큐가 무한정 쌓인다.** `t.write(text)` 를 완료 콜백 없이 부르므로 ([TerminalView.tsx:263](../../frontend/src/screens/TerminalView.tsx#L263)) 밀린 데이터는 xterm 의 쓰기 버퍼에 쌓인다. xterm 은 약 50MB 를 넘으면 뒷부분을 **버린다**(출력 유실). 그 전까지 메모리가 계속 는다.
5. **DOM 렌더러.** `package.json` 에 WebGL/Canvas 애드온이 없다 → 고속 출력에서 가장 느린 렌더 방식.
6. **숨은 탭도 똑같이 처리.** `writeOutput` 은 탭이 보이든 말든 바로 `write` 한다 ([workspace.ts:158-168](../../frontend/src/state/workspace.ts#L158-L168)). 활동 표시를 켤 때 세션 맵을 새로 만드는 `set` 도 매 묶음마다 비용이 든다(활동 표시가 이미 켜진 뒤에는 건너뜀).
7. (부수) 로그가 켜진 세션은 읽기 고루틴 안에서 정규식 치환 + 동기 파일 쓰기를 한다 ([log.go:50-57](../../internal/session/log.go#L50-L57)). 그 세션만 느려지고 다른 세션에는 영향이 없어 이번 범위 밖.

### 1.3 Known issue

`.claude/known-issues.md` 에 해당 항목 없음. 착수 시 **KI-001** 로 `active` 추가하고 완료 시 `resolved` 로 바꾼다 (공통 규칙 §5, 다음 번호 KI-001).

---

## 2. 설계 결정

### 2.1 역압 방식: "묶음 개수 ack + 바이트 크레딧" (권장안)

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 프런트가 처리한 묶음 수를 ack, Go 는 미처리 바이트가 상한을 넘으면 `Read` 를 멈춤** | Go 가 읽기를 멈추면 TCP 윈도우/PTY 파이프가 차서 원격 프로세스가 스스로 느려진다. 데이터 유실 없음. ack 는 "개수" 라 UTF-8/UTF-16 길이 불일치 문제 없음 | **채택** |
| B. 프런트에서만 큐를 두고 프레임당 일정량만 쓰기 | Go 는 계속 읽으므로 메모리가 프런트 큐로 옮겨갈 뿐 상한이 없다 | 기각 |
| C. 밀린 출력을 버리고 따라잡기 | 화면과 로그가 달라진다. `yes` 류를 Ctrl+C 로 멈추는 체감은 좋지만 데이터 유실 | 기각 (A 로도 Ctrl+C 반응은 충분히 빠름, §6.3) |

**ack 전달 통로**: 새 바인딩 메서드를 추가하면 spec/ipc.md §1 "4개 메서드" 가 바뀐다. 대신 기존 `Invoke` 의 **요청 채널 `session.ack`** 를 쓴다 (§2 표에 한 줄 추가로 끝). 호출 빈도는 microtask 로 합쳐서 세션당 xterm 처리 주기(약 12ms)마다 최대 1회.

**수치** (상수로 두고 조정):

| 상수 | 값 | 근거 |
|---|---|---|
| 상한(High water, 읽기 멈춤) | 512KB | 초당 4MB 기준 약 128ms 분량. 보통 출력(수십 KB/s)은 절대 닿지 않는다 |
| 하한(Low water, 읽기 재개) | 128KB | 멈춤/재개가 너무 잦지 않게 상한의 1/4 |
| 묶음 상한(③) | 256KB | 한 이벤트의 JSON·JS 실행 시간을 묶음 |
| ack 안전 시한 | 3초 | 프런트 결함으로 ack 가 영영 안 오면 읽기를 다시 연다 (멈춘 채 굳는 것보다 낫다) |
| 숨은 탭 몰아쓰기 주기(④) | 100ms | 보이는 탭에 프레임을 양보하되 활동 표시·스크롤백 갱신은 체감 지연 없음 |

### 2.2 ack 는 "묶음 수" 로 센다

Go 는 보낸 묶음의 바이트 수를 **FIFO 큐**에 넣고, 프런트는 처리한 **묶음 개수**만 보낸다. Go 는 큐 앞에서 그 수만큼 빼서 미처리 바이트를 줄인다. 프런트가 텍스트 길이를 셀 필요가 없고(UTF-16 vs UTF-8), 묶음을 합쳐 쓴 경우(④)는 합친 개수를 보낸다.

**순서 주의**: Go 는 `emit` **전에** 큐에 넣는다. 그렇지 않으면 ack 가 먼저 와서 빈 큐에서 무시되고 미처리 바이트가 영원히 남는다.

### 2.3 프런트가 ack 를 반드시 보내야 하는 경우

- xterm `write(text, 콜백)` 의 콜백 → ack 1
- writer 가 없을 때(터미널이 아직 안 만들어졌거나 이미 버려짐) → **즉시 ack** (안 하면 Go 가 멈춘다)
- 숨은 탭 몰아쓰기 → 합친 묶음 수만큼 ack
- 세션 제거 시 몰아쓰기 대기분 → ack (Go 쪽은 이미 닫혀 무시되지만 프런트 상태 정리 목적)

### 2.4 명세 변경 (명세 작성자 반영 필요 — 구현 전에 spec 에 먼저 넣는다)

`spec/ipc.md` §2 세션 표에 추가:

```
| `session.ack` | id, 처리한 출력 묶음 수(정수) | 없음. 프런트가 `session.data` 묶음을 터미널에 다 쓴 뒤 보낸다. 모르는 id·넘치는 수는 무시 |
```

`spec/ipc.md` §4 `session.data` 행 설명 끝에 추가:

```
프런트는 묶음을 터미널에 쓴 뒤 `session.ack` 로 처리 개수를 알린다(출력 흐름 제어, behavior/session.md §8).
```

`spec/behavior/session.md` §8 에 추가:

```
- **출력 흐름 제어**: 프런트가 아직 처리(ack)하지 않은 출력이 512KB 를 넘으면 백엔드는 그 세션의 읽기를 멈추고, 128KB 아래로 내려오면 다시 읽는다(원격·로컬 프로세스는 전송 버퍼가 차서 느려진다. 출력은 버리지 않는다). 묶음 하나는 256KB 를 넘지 않는다(넘으면 12ms 를 기다리지 않고 바로 보낸다). ack 가 3초 동안 전혀 오지 않으면 읽기를 다시 연다.
- **숨은 세션의 출력**: 보이지 않는 탭의 세션 출력은 약 100ms 단위로 몰아서 터미널에 쓴다. 보이게 되는 순간 밀린 것을 먼저 쓴다.
- **렌더러**: WebGL 을 쓸 수 있으면 WebGL, 아니면(컨텍스트 생성 실패·손실) DOM 렌더러.
```

---

## 3. 수정 계획 — Go (① 역압, ③ 묶음 상한)

### 3.1 새 파일 `internal/session/credit.go`

```go
package session

import (
	"sync"
	"time"
)

// 출력 흐름 제어 (spec/behavior/session.md §8 "출력 흐름 제어").
// 프런트가 아직 처리(ack)하지 않은 출력 바이트가 상한을 넘으면 읽기를 멈추고, 하한 아래로 내려오면 다시 읽는다.
const (
	// G_ciCreditHighWater 는 읽기를 멈추는 미처리 바이트 상한이다.
	G_ciCreditHighWater = 512 * 1024
	// G_ciCreditLowWater 는 읽기를 다시 여는 미처리 바이트 하한이다.
	G_ciCreditLowWater = 128 * 1024
	// G_cdurCreditTimeout 은 ack 가 전혀 오지 않을 때 읽기를 다시 여는 안전 시한이다 (프런트 결함 대비).
	G_cdurCreditTimeout = 3 * time.Second
)

// Credit 은 세션 하나의 미처리 출력 양을 센다. 보낸 묶음의 크기를 순서대로 기억하고, 프런트의 ack(개수)로 앞에서부터 뺀다.
type Credit struct {
	mtxData   sync.Mutex
	cond      *sync.Cond
	slPending []int // 보낸 순서대로 각 묶음의 바이트 수
	iInFlight int   // slPending 의 합
	bClosed   bool
}

// NewCredit 은 빈 크레딧을 만든다.
func NewCredit() *Credit {
	pCredit := &Credit{}
	pCredit.cond = sync.NewCond(&pCredit.mtxData)
	return pCredit
}

// Sent 는 묶음 하나를 보냈음을 기록한다. emit 보다 먼저 불러야 한다 (ack 가 먼저 오면 무시되므로).
func (pCredit *Credit) Sent(iBytes int) {
	pCredit.mtxData.Lock()
	pCredit.slPending = append(pCredit.slPending, iBytes)
	pCredit.iInFlight += iBytes
	pCredit.mtxData.Unlock()
}

// Ack 는 프런트가 앞에서부터 iCount 개 묶음을 처리했음을 반영한다. 큐보다 많은 수는 무시한다.
func (pCredit *Credit) Ack(iCount int) {
	pCredit.mtxData.Lock()
	for iCount > 0 && len(pCredit.slPending) > 0 {
		pCredit.iInFlight -= pCredit.slPending[0]
		pCredit.slPending = pCredit.slPending[1:]
		iCount--
	}
	if len(pCredit.slPending) == 0 {
		pCredit.slPending = nil // 밑바닥 배열을 놓아 준다
		pCredit.iInFlight = 0
	}
	pCredit.mtxData.Unlock()
	pCredit.cond.Broadcast()
}

// Wait 은 미처리 양이 상한 이상이면 하한 아래로 내려올 때까지 막는다. 닫혔거나 안전 시한이 지나면 돌아온다.
func (pCredit *Credit) Wait() {
	pCredit.mtxData.Lock()
	defer pCredit.mtxData.Unlock()
	if pCredit.iInFlight < G_ciCreditHighWater || pCredit.bClosed {
		return
	}
	bTimedOut := false
	timer := time.AfterFunc(G_cdurCreditTimeout, func() {
		pCredit.mtxData.Lock()
		bTimedOut = true
		pCredit.slPending = nil
		pCredit.iInFlight = 0
		pCredit.mtxData.Unlock()
		pCredit.cond.Broadcast()
	})
	defer timer.Stop()
	for pCredit.iInFlight > G_ciCreditLowWater && !pCredit.bClosed && !bTimedOut {
		pCredit.cond.Wait()
	}
}

// Close 는 기다리는 읽기를 깨우고 이후의 Wait 을 즉시 돌려보낸다.
func (pCredit *Credit) Close() {
	pCredit.mtxData.Lock()
	pCredit.bClosed = true
	pCredit.mtxData.Unlock()
	pCredit.cond.Broadcast()
}

// InFlight 는 미처리 바이트 수다 (테스트용).
func (pCredit *Credit) InFlight() int {
	pCredit.mtxData.Lock()
	defer pCredit.mtxData.Unlock()
	return pCredit.iInFlight
}
```

> 알려진 사소한 경쟁: 시한 타이머가 `Wait` 이 막 돌아온 직후에 발화하면 카운터를 한 번 0 으로 만든다. 흐름 제어가 한 번 느슨해질 뿐 데이터 유실·교착은 없다.

### 3.2 `internal/session/session.go` 변경

**구조체** — 필드 추가:

```go
type Session struct {
	// ... 기존 필드 ...
	batcher      *Batcher
	credit       *Credit   // 추가
	logger       *Logger
}
```

**newSession** — 묶음 전송 시 크레딧 기록을 emit **앞에** 둔다:

```go
// before
	pSession.batcher = NewBatcher(func(szText string) {
		emit("session.data", szId, szText)
	})

// after
	pSession.credit = NewCredit()
	pCredit := pSession.credit
	pSession.batcher = NewBatcher(func(szText string) {
		pCredit.Sent(len(szText)) // emit 보다 먼저 (§2.2 순서 주의)
		emit("session.data", szId, szText)
	})
```

**readLoop** — 읽기 전에 크레딧을 기다린다:

```go
// before
func (pSession *Session) readLoop(transport Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	for {
		nRead, errRead := transport.Read(bBuf)

// after
func (pSession *Session) readLoop(transport Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	for {
		// 프런트가 아직 처리하지 않은 출력이 상한을 넘으면 여기서 멈춘다 (session.md §8 출력 흐름 제어).
		// 읽기를 멈추면 TCP 윈도우·PTY 파이프가 차서 상대 프로세스가 스스로 느려진다.
		pSession.credit.Wait()
		nRead, errRead := transport.Read(bBuf)
```

**AckOutput** — `control.go` 에 추가:

```go
// AckOutput 은 프런트가 출력 묶음 iCount 개를 처리했음을 반영한다 (spec/ipc.md session.ack).
func (pSession *Session) AckOutput(iCount int) {
	if iCount <= 0 {
		return
	}
	pSession.credit.Ack(iCount)
}
```

**Close** (`control.go`) — 기다리는 readLoop 를 깨운다. `transport.Close()` 직전에 넣는다:

```go
	pSession.credit.Close() // 크레딧을 기다리던 readLoop 를 깨워 Read 오류로 끝나게 한다 (transport 가 nil 이어도 안전)
	if transport != nil {
		_ = transport.Close()
	}
```

> 재접속(`connect` 로 새 세대) 때는 크레딧을 **초기화하지 않는다**: 옛 묶음의 ack 가 늦게 와도 큐 앞에서 빠지므로 정확하다. 자동 재접속은 transport 의 Read 오류로 시작하므로 그때 readLoop 가 Wait 에 막혀 있을 수 없다. 사용자 닫기만 Wait 에 막힌 readLoop 를 깨워야 하고, 그건 위 `credit.Close()` 가 한다.

### 3.3 `internal/session/batch.go` 변경 (③ 묶음 상한)

```go
// G_ciBatchMaxBytes 는 묶음 하나의 상한이다. 넘으면 12ms 를 기다리지 않고 바로 보낸다 (session.md §8).
const G_ciBatchMaxBytes = 256 * 1024

// Add 는 조각을 더한다. 첫 조각이면 12ms 타이머를 걸고, 상한을 넘으면 즉시 보낸다.
func (pBatcher *Batcher) Add(szChunk string) {
	if szChunk == "" {
		return
	}
	pBatcher.mtxData.Lock()
	pBatcher.sbPending.WriteString(szChunk)
	if pBatcher.sbPending.Len() >= G_ciBatchMaxBytes {
		szOut := pBatcher.takeLocked()
		pBatcher.mtxData.Unlock()
		pBatcher.fnFlush(szOut)
		return
	}
	if pBatcher.timer == nil {
		pBatcher.timer = time.AfterFunc(G_cdurBatchWindow, pBatcher.Flush)
	}
	pBatcher.mtxData.Unlock()
}

// Flush 는 쌓인 내용을 즉시 보낸다.
func (pBatcher *Batcher) Flush() {
	pBatcher.mtxData.Lock()
	szOut := pBatcher.takeLocked()
	pBatcher.mtxData.Unlock()
	if szOut != "" {
		pBatcher.fnFlush(szOut)
	}
}

// takeLocked 는 mtxData 를 잡은 상태에서 쌓인 내용을 꺼내고 타이머를 끈다.
func (pBatcher *Batcher) takeLocked() string {
	szOut := pBatcher.sbPending.String()
	pBatcher.sbPending.Reset()
	if pBatcher.timer != nil {
		pBatcher.timer.Stop()
		pBatcher.timer = nil
	}
	return szOut
}
```

> `fnFlush` 는 잠금 밖에서 부른다 (기존과 같음). `Add` 를 부르는 쪽은 세션당 readLoop 하나와 안내 줄뿐이라 동시 호출은 드물고, 순서는 잠금이 보장한다.

### 3.4 `internal/app/channels_session.go` — `session.ack` 채널

`registerSession` 안, `session.reconnect` 다음에 추가:

```go
	pRouter.Handle("session.ack", func(slArgs []json.RawMessage) (any, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		var iCount int
		if errCount := argInto(slArgs, 1, &iCount); errCount != nil {
			return nil, errCount
		}
		pSession, errGet := pService.pSessions.Get(szId)
		if errGet != nil {
			return nil, nil // 이미 닫힌 세션의 늦은 ack 는 조용히 무시
		}
		pSession.AckOutput(iCount)
		return nil, nil
	})
```

`internal/app/contract_test.go` 와 `frontend/src/ipc/contract.test.ts` 는 **결과 모양**(예: `session.open` 의 세션 정보)만 검사한다. `session.ack` 는 결과가 없으므로 계약 테스트에 추가할 항목이 없다 (2026-11-16 확인).

---

## 4. 수정 계획 — 프런트 (①-ack, ② WebGL, ④ 숨은 탭)

### 4.1 새 파일 `frontend/src/lib/outputAck.ts` — ack 합치기

```ts
// 출력 묶음 ack 를 세션별로 모아 microtask 한 번에 보낸다 (spec/ipc.md session.ack, behavior/session.md §8).
// xterm 은 한 처리 주기에 여러 묶음의 완료 콜백을 연달아 부르므로 microtask 로 합치면 세션당 주기마다 1회가 된다.
import { getBackend } from '../ipc/backend';

const pending = new Map<string, number>();
let scheduled = false;

/** 세션의 출력 묶음 count 개를 처리했다고 표시한다. 실제 전송은 microtask 에서 한 번에. */
export function ackOutput(sessionId: string, count = 1): void {
  if (count <= 0) return;
  pending.set(sessionId, (pending.get(sessionId) ?? 0) + count);
  if (scheduled) return;
  scheduled = true;
  queueMicrotask(flushAcks);
}

function flushAcks(): void {
  scheduled = false;
  const entries = [...pending];
  pending.clear();
  let b;
  try { b = getBackend(); } catch { return; } // 백엔드 준비 전(테스트)이면 버린다
  for (const [id, n] of entries) void b.invoke('session.ack', [id, n]).catch(() => undefined);
}

/** 테스트용: 쌓인 ack 를 비운다. */
export function resetAcksForTest(): void {
  pending.clear();
  scheduled = false;
}
```

### 4.2 `frontend/src/state/workspace.ts` 변경

**writer 서명** — 완료 콜백을 받는다:

```ts
// before
  writers: Record<string, (text: string) => void>;
  registerWriter: (sessionId: string, fn: ((text: string) => void) | null) => void;

// after
  /** 열린 세션의 터미널 기록기. done 은 xterm 이 text 를 다 쓴 뒤 불린다(출력 흐름 제어 ack). */
  writers: Record<string, (text: string, done: () => void) => void>;
  registerWriter: (sessionId: string, fn: ((text: string, done: () => void) => void) | null) => void;
  /** 숨은 탭에서 모아 둔 출력을 지금 쓴다(탭이 보이게 될 때). */
  flushDeferred: (sessionId: string) => void;
```

**보임 판정 헬퍼와 몰아쓰기 대기분** — 파일 위쪽(스토어 밖)에:

```ts
import { ackOutput } from '../lib/outputAck';

/** 세션이 지금 화면에 보이는 탭(왼쪽·오른쪽 그룹의 선택 탭 또는 활성 탭) 안에 있는지. */
function isSessionVisible(s: WorkspaceState, sessionId: string): boolean {
  for (const t of s.tabs) {
    if (t.id !== s.activeByGroup[0] && t.id !== s.activeByGroup[1] && t.id !== s.activeTabId) continue;
    if (findPaneBySession(t.root, sessionId)) return true;
  }
  return false;
}

/** 숨은 세션의 출력 대기분(세션 id → 조각들, 타이머). 스토어 밖에 둔다(매 묶음마다 상태 갱신을 피함). */
const HIDDEN_FLUSH_MS = 100;
type Writer = (text: string, done: () => void) => void;
const deferred = new Map<string, { parts: string[]; timer: ReturnType<typeof setTimeout> | null }>();

function writeDeferred(sessionId: string, w: Writer): void {
  const d = deferred.get(sessionId);
  if (!d || d.parts.length === 0) return;
  if (d.timer) { clearTimeout(d.timer); d.timer = null; }
  const count = d.parts.length;
  const text = d.parts.join('');
  d.parts = [];
  w(text, () => ackOutput(sessionId, count));
}

/** 세션이 사라지거나 writer 가 빠질 때: 대기분을 버리고 ack 만 보낸다(Go 쪽이 멈추지 않게). */
function dropDeferred(sessionId: string): void {
  const d = deferred.get(sessionId);
  if (!d) return;
  if (d.timer) clearTimeout(d.timer);
  if (d.parts.length) ackOutput(sessionId, d.parts.length);
  deferred.delete(sessionId);
}
```

**writeOutput / registerWriter / flushDeferred / removeSession**:

```ts
  registerWriter(sessionId, fn) {
    if (!fn) dropDeferred(sessionId);
    set((s) => {
      const writers = { ...s.writers };
      if (fn) writers[sessionId] = fn;
      else delete writers[sessionId];
      return { writers };
    });
  },

  writeOutput(sessionId, text) {
    const s = get();
    const w = s.writers[sessionId];
    if (!w) {
      // 터미널이 아직 없거나 이미 버려짐: 출력은 버리되 ack 는 보낸다(안 보내면 백엔드가 읽기를 멈춘다).
      ackOutput(sessionId);
      return;
    }
    if (isSessionVisible(s, sessionId)) {
      w(text, () => ackOutput(sessionId));
      return;
    }
    // 숨은 탭: 100ms 단위로 몰아서 쓴다(보이는 탭에 프레임 시간을 양보).
    let d = deferred.get(sessionId);
    if (!d) { d = { parts: [], timer: null }; deferred.set(sessionId, d); }
    d.parts.push(text);
    if (!d.timer) {
      const entry = d;
      entry.timer = setTimeout(() => {
        entry.timer = null;
        const w2 = get().writers[sessionId];
        if (w2) writeDeferred(sessionId, w2);
        else dropDeferred(sessionId);
      }, HIDDEN_FLUSH_MS);
    }
    // 활동 표시는 한 번만 켠다(이미 켜져 있으면 상태 갱신 없음).
    const entry = s.sessions[sessionId];
    if (entry && !entry.activity) {
      set((st) => ({ sessions: { ...st.sessions, [sessionId]: { ...st.sessions[sessionId], activity: true } } }));
    }
  },

  flushDeferred(sessionId) {
    const w = get().writers[sessionId];
    if (w) writeDeferred(sessionId, w);
    else dropDeferred(sessionId);
  },

  removeSession(sessionId) {
    dropDeferred(sessionId);
    set((s) => { /* 기존 내용 그대로 */ });
  },
```

> 기존 `writeOutput` 도 `!visible` 일 때만 활동 표시를 켰으므로 동작은 같다. 달라지는 건 보임 판정이 헬퍼로 빠지고, 숨은 세션의 쓰기가 100ms 뒤로 미뤄지는 것뿐이다.

### 4.3 `frontend/src/screens/TerminalView.tsx` 변경

**writer 등록** — 완료 콜백 전달 (`createKeptTerm` 안):

```ts
// before
  useWorkspace.getState().registerWriter(sessionId, (text) => t.write(text));

// after
  useWorkspace.getState().registerWriter(sessionId, (text, done) => t.write(text, done));
```

**② WebGL 애드온** — `t.open(el)` **다음 줄**에 (WebGL 은 open 뒤에 붙여야 한다):

```ts
import { WebglAddon } from '@xterm/addon-webgl';
// ...
  t.open(el);
  attachWebgl(t);
```

파일 위쪽에 헬퍼:

```ts
/** WebGL 렌더러를 붙인다. 만들 수 없으면(소프트웨어 렌더링·RDP 등) 조용히 DOM 렌더러를 유지하고,
 *  쓰다가 컨텍스트를 잃으면 애드온을 버려 DOM 렌더러로 돌아간다 (behavior/session.md §8 렌더러). */
function attachWebgl(t: Terminal): void {
  try {
    const gl = new WebglAddon();
    gl.onContextLoss(() => gl.dispose());
    t.loadAddon(gl);
  } catch {
    // WebGL 불가: DOM 렌더러 그대로
  }
}
```

**④ 보이게 될 때 대기분 먼저 쓰기** — 붙이기 effect 안, `host.appendChild(k.el);` 다음:

```ts
    host.appendChild(k.el);
    useWorkspace.getState().flushDeferred(sessionId); // 숨어 있던 동안 모아 둔 출력을 먼저 쓴다
```

(`t.refresh(0, t.rows - 1)` 은 그대로 둔다. xterm 의 write 는 비동기이므로 refresh 가 먼저 돌아도 write 완료 시 다시 그린다.)

### 4.4 `frontend/package.json`

```json
    "@xterm/addon-webgl": "0.19.0",
```

0.19.0 은 2026-11-16 기준 npm 최신 버전이며 xterm 6.0.0 과 같이 나온 것이다 (`npm view @xterm/addon-webgl version` 으로 확인). 의존성이 바뀌므로 `make notices` 로 THIRD_PARTY_NOTICES.md 를 다시 만든다 (CLAUDE.md clean-room 절, D63·D64).

### 4.5 `frontend/src/ipc/fakeBackend.ts`

`handlers` 표([fakeBackend.ts:245](../../frontend/src/ipc/fakeBackend.ts#L245) `'session.reconnect'` 옆)에 한 줄 추가한다. 없으면 `invoke` 가 "구현되지 않은 채널" 오류를 던져 개발 서버에서 오류 토스트가 뜬다:

```ts
    'session.ack': () => undefined,
```

### 4.6 `frontend/src/ipc/backend.ts`

변경 없음. `EventName` 은 `session.data` 그대로이고, `invoke` 의 채널 인자는 `string` 이라 타입 추가가 필요 없다 (2026-11-16 확인).

---

## 5. 문서·기록 변경

| 파일 | 내용 |
|---|---|
| `spec/ipc.md`, `spec/behavior/session.md` | §2.4 의 문구 (명세 작성자) |
| `docs/decisions.md` | **D84** 행: "출력 흐름 제어(ack 크레딧 512KB/128KB, 묶음 256KB, 안전 시한 3초) + WebGL 렌더러(실패 시 DOM) + 숨은 탭 100ms 몰아쓰기. 사용자 질문(한 세션의 고속 출력이 앱 전체에 영향)에서. 데이터 유실 방식(C)은 기각" |
| `.claude/known-issues.md` | 착수 시 KI-001 `active` → 완료 시 `resolved` |
| `THIRD_PARTY_NOTICES.md` | `make notices` 결과 |
| `.claude/wip/…` | 착수 시 생성, §7 체크리스트 |

---

## 6. 테스트 계획

### 6.1 Go 단위 테스트 (`internal/session/credit_test.go` 신규, 묶음 상한은 `session_test.go` 에 추가)

```go
package session

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// floodTransport 는 Read 때마다 버퍼를 'x' 로 채워 돌려주는 가짜 연결이다. Close 뒤에는 오류를 돌려준다.
type floodTransport struct {
	bClosed atomic.Bool
	iReads  atomic.Int64
}

func (pFlood *floodTransport) Read(bBuf []byte) (int, error) {
	if pFlood.bClosed.Load() {
		return 0, errors.New("closed")
	}
	pFlood.iReads.Add(1)
	for i := range bBuf {
		bBuf[i] = 'x'
	}
	return len(bBuf), nil
}
func (pFlood *floodTransport) Write(bData []byte) error          { return nil }
func (pFlood *floodTransport) Resize(iCols int, iRows int) error { return nil }
func (pFlood *floodTransport) Close() error                      { pFlood.bClosed.Store(true); return nil }

// 폭주 출력: ack 가 없으면 보낸 양이 상한 + 묶음 상한 + 읽기 버퍼를 넘지 않고, ack 를 주면 다시 읽는다.
func TestCreditStopsReadingWithoutAck(t *testing.T) {
	var mtxSent sync.Mutex
	iSent := 0
	iBatches := 0
	flood := &floodTransport{}
	cfg := sshConfig(func(func(string)) (Transport, error) { return flood, nil }, false)
	pSession := newSession(cfg, func(szEvent string, anyData ...any) {
		if szEvent != "session.data" {
			return
		}
		mtxSent.Lock()
		iSent += len(anyData[1].(string))
		iBatches++
		mtxSent.Unlock()
	})
	pSession.start()
	defer pSession.Close()

	// 상한에 닿을 때까지 기다린다.
	waitUntil(t, func() bool { mtxSent.Lock(); defer mtxSent.Unlock(); return iSent >= G_ciCreditHighWater }, 2*time.Second)
	time.Sleep(100 * time.Millisecond) // 더 보내는지 지켜본다
	mtxSent.Lock()
	iAfter := iSent
	mtxSent.Unlock()
	iLimit := G_ciCreditHighWater + G_ciBatchMaxBytes + 32*1024
	if iAfter > iLimit {
		t.Fatalf("ack 없이 보낸 양이 상한을 넘음: %d > %d", iAfter, iLimit)
	}

	// 전부 ack 하면 다시 읽는다.
	mtxSent.Lock()
	iAckCount := iBatches
	mtxSent.Unlock()
	pSession.AckOutput(iAckCount)
	waitUntil(t, func() bool { mtxSent.Lock(); defer mtxSent.Unlock(); return iSent > iAfter }, 2*time.Second)
}

// 크레딧에 막혀 있는 세션을 닫으면 readLoop 가 깨어나 끝난다 (교착 없음).
func TestCreditWakesOnClose(t *testing.T) {
	flood := &floodTransport{}
	cfg := sshConfig(func(func(string)) (Transport, error) { return flood, nil }, false)
	pSession := newSession(cfg, func(string, ...any) {})
	pSession.start()
	waitUntil(t, func() bool { return pSession.credit.InFlight() >= G_ciCreditHighWater }, 2*time.Second)
	iReadsBefore := flood.iReads.Load()
	time.Sleep(50 * time.Millisecond)
	if flood.iReads.Load() != iReadsBefore {
		t.Fatal("막혀 있어야 하는데 계속 읽음")
	}
	chDone := make(chan struct{})
	go func() { pSession.Close(); close(chDone) }()
	select {
	case <-chDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close 가 돌아오지 않음")
	}
	if pSession.Snapshot().M_szState != G_cszStateClosed {
		t.Fatalf("닫힌 상태여야 함: %s", pSession.Snapshot().M_szState)
	}
}

// 큐보다 많은 ack 는 무시하고, 미처리 바이트는 음수가 되지 않는다.
func TestCreditIgnoresExcessAck(t *testing.T) {
	pCredit := NewCredit()
	pCredit.Sent(100)
	pCredit.Ack(5)
	if pCredit.InFlight() != 0 {
		t.Fatalf("0 이어야 함: %d", pCredit.InFlight())
	}
	pCredit.Ack(1) // 빈 큐
	if pCredit.InFlight() != 0 {
		t.Fatal("음수/변화 없어야 함")
	}
}

// 안전 시한: ack 가 전혀 없어도 3초 뒤 Wait 이 돌아온다.
func TestCreditTimeoutReleases(t *testing.T) {
	pCredit := NewCredit()
	pCredit.Sent(G_ciCreditHighWater)
	tStart := time.Now()
	pCredit.Wait()
	if time.Since(tStart) < G_cdurCreditTimeout-100*time.Millisecond {
		t.Fatal("시한 전에 돌아옴")
	}
	if pCredit.InFlight() != 0 {
		t.Fatal("시한 뒤 미처리 바이트는 0 이어야 함")
	}
}

// 묶음 상한: 256KB 를 넘으면 12ms 를 기다리지 않고 바로 보낸다.
func TestBatcherFlushesAtMaxBytes(t *testing.T) {
	var mtx sync.Mutex
	var slOut []int
	pBatcher := NewBatcher(func(szText string) { mtx.Lock(); slOut = append(slOut, len(szText)); mtx.Unlock() })
	szBig := strings.Repeat("y", G_ciBatchMaxBytes)
	pBatcher.Add(szBig) // 타이머 없이 즉시
	mtx.Lock()
	if len(slOut) != 1 || slOut[0] != G_ciBatchMaxBytes {
		t.Fatalf("즉시 1회 전송이어야 함: %v", slOut)
	}
	mtx.Unlock()
	pBatcher.Add("tail") // 이건 12ms 뒤
	waitUntil(t, func() bool { mtx.Lock(); defer mtx.Unlock(); return len(slOut) == 2 }, time.Second)
}
```

> `sshConfig`·`waitUntil` 은 기존 `session_test.go` 의 헬퍼를 그대로 쓴다. `TestCreditStopsReadingWithoutAck` 는 `-race` 에서 시간이 더 걸리므로 대기 시한을 2초로 둔다.

실행:

```bash
go test -race ./internal/session/ -run 'Credit|Batcher' -v
go test -race ./...
```

### 6.2 프런트 단위 테스트 (`frontend/src/state/workspace.test.ts` 신규, vitest + jsdom)

검증 항목:
1. writer 가 없을 때 `writeOutput` → `session.ack` 가 `[id, 1]` 로 호출된다.
2. 보이는 세션: writer 가 즉시 불리고, `done()` 호출 뒤 microtask 에 ack 1 이 간다. 한 틱에 `done` 3번 → `session.ack` 호출 **1번**, 개수 3.
3. 숨은 세션: 100ms 안에 3 묶음 → writer **1번**(합친 텍스트), `done` 뒤 ack 3. `activity` 가 true 로 바뀐다.
4. 숨은 세션에 `flushDeferred` → 타이머 전이라도 즉시 writer 호출.
5. `removeSession` → 대기분 ack, writer 호출 없음.

뼈대:

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { setBackend } from '../ipc/backend';
import { useWorkspace } from './workspace';
import { resetAcksForTest } from '../lib/outputAck';

const invoke = vi.fn(async () => undefined);
beforeEach(() => {
  vi.useFakeTimers();
  invoke.mockClear();
  resetAcksForTest();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as any);
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], sessions: {}, writers: {} });
});

describe('writeOutput 흐름 제어', () => {
  it('writer 가 없으면 ack 만 보낸다', async () => {
    useWorkspace.getState().writeOutput('s1', 'abc');
    await Promise.resolve(); // microtask
    expect(invoke).toHaveBeenCalledWith('session.ack', ['s1', 1]);
  });
  it('보이는 세션은 바로 쓰고 done 뒤 ack 를 합쳐 보낸다', async () => {
    // tabs/activeTabId 를 s1 이 보이게 구성 (lib/layout 의 newPane 사용)
    // writer 등록: (text, done) => { written.push(text); done(); }
    // writeOutput 3번 → await microtask → invoke 가 1번, ['s1', 3]
  });
  it('숨은 세션은 100ms 뒤 한 번에 쓴다', () => {
    // 보이지 않게 구성, writer 등록, writeOutput 3번 → written.length === 0
    // vi.advanceTimersByTime(100) → written === ['abc'] (합침), activity === true
  });
});
```

실행:

```bash
cd frontend && npx vitest run src/state/workspace.test.ts && npx tsc --noEmit
```

### 6.3 수동 검증 (빌드 후 `build/bin/dariterm.exe`)

| # | 절차 | 기대 결과 (수정 전 → 후) |
|---|---|---|
| 1 | SSH 세션 A 에서 `yes` 또는 `cat /dev/urandom \| base64` 실행. 탭 B(다른 세션)로 전환해 타이핑 | 전: 글자가 늦게 찍히고 UI 가 끊김 → **후: B 는 평소처럼 반응** |
| 2 | 1 을 유지한 채 작업 관리자에서 dariterm.exe 메모리 관찰(1분) | 전: 계속 증가 → **후: 수십 MB 안에서 안정** |
| 3 | A 로 돌아가 Ctrl+C | 전: 밀린 출력이 몇 초 더 쏟아짐 → **후: 1초 안에 멈춤** (떠 있는 양이 상한 이내) |
| 4 | A 에서 `seq 1 2000000 \| tail -n 3` 과 `seq 1 2000000` 을 각각 실행 | 후: 마지막 줄 `2000000` 이 화면에 정확히 보인다 (유실 없음). 로그가 켜져 있으면 로그 파일 끝도 같다 |
| 5 | 로컬 PowerShell 세션: `1..3000000 \| % { "line $_ $('x' * 60)" }` | 1~4 와 같은 기대. ConPTY 파이프가 차면 PowerShell 이 느려진다 |
| 6 | A 가 폭주하는 동안 A 탭을 숨기고(B 활성) 30초 뒤 A 로 전환 | 전환 즉시 밀린 출력이 그려지고 스크롤백에 남아 있다. 숨은 동안 B 의 입력 지연 없음 |
| 7 | 시리얼 세션(가능하면)에서 긴 출력 | 변화 없음(초당 11KB 는 상한에 닿지 않음) |
| 8 | `wails dev` 의 DevTools 에서 `document.querySelector('.xterm canvas')` 존재 여부 | WebGL 이 붙으면 canvas 요소가 있다. RDP 세션에서 열면 없어도(DOM 폴백) 정상 동작 |
| 9 | WebGL 폴백 강제: `attachWebgl` 첫 줄에 `throw new Error('test')` 를 넣고 한 번 빌드해 DOM 렌더러 경로가 깨지지 않는지 본다 (확인 후 되돌림) | 터미널 정상 |
| 10 | 자동 재접속: A 폭주 중 서버 쪽에서 sshd 를 재시작 | 재접속 안내 → 재접속 성공 → 출력 재개. 교착 없음 |
| 11 | 앱 종료(X): A 폭주 중 닫기 | 종료 확인 → 즉시 종료 (Close 가 Wait 을 깨움) |

### 6.4 회귀 검증 명령

```bash
go build ./... && go test -race ./...
(cd frontend && npx tsc --noEmit && npx vitest run)
wails build
make notices
```

---

## 7. 착수 체크리스트 (WIP 로 옮길 것)

- [ ] 1. 명세 반영: `spec/ipc.md` §2·§4, `spec/behavior/session.md` §8 (§2.4 문구)
- [ ] 2. known-issues KI-001 `active` 추가
- [ ] 3. Go: `credit.go` 신규, `session.go`(필드·newSession·readLoop), `control.go`(AckOutput·Close), `batch.go`(상한)
- [ ] 4. Go: `channels_session.go` `session.ack` 채널, contract_test 채널 목록
- [ ] 5. Go 테스트 §6.1 작성 → `go test -race ./...` 통과
- [ ] 6. 프런트: `lib/outputAck.ts` 신규, `state/workspace.ts`(writer 서명·보임 판정·몰아쓰기·flushDeferred), `screens/TerminalView.tsx`(done 전달·WebGL·flushDeferred), `ipc/fakeBackend.ts`
- [ ] 7. `package.json` addon-webgl 추가 → `npm install` → `make notices`
- [ ] 8. 프런트 테스트 §6.2 → `vitest run`, `tsc --noEmit` 통과
- [ ] 9. `wails build` → §6.3 수동 검증 1·2·3·4·6·11 필수
- [ ] 10. `docs/decisions.md` D84, known-issues KI-001 `resolved`, WIP `done` → archive

---

## 8. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| ack 누락 → 세션 출력이 멈춤 | 프런트 경로 중 하나라도 ack 를 빠뜨리면 그 세션이 512KB 뒤에 멈춘다 | §2.3 의 4가지 경로를 모두 ack. 안전 시한 3초로 "영원히 멈춤" 은 방지. 테스트 §6.2-1·5 |
| Close 교착 | readLoop 가 Wait 에 막힌 채 Close | `credit.Close()` 가 깨움. 테스트 §6.1-2 |
| WebGL 미지원 환경(RDP·VM·구형 GPU) | 터미널이 안 보임 | try/catch 폴백 + 컨텍스트 손실 시 dispose. 수동 §6.3-8·9 |
| WebGL 과 글꼴·줄 간격 | DOM 과 미세하게 다르게 보일 수 있음 | 외형 명세(`spec/ui/`)와 대조. 문제가 크면 설정으로 끌 수 있게 후속 |
| 숨은 탭 100ms 지연 | 활동 표시·스크롤백은 늦어도 100ms. 사용자 체감 없음 | 보이게 될 때 `flushDeferred` |
| 명세 계약 | `session.ack` 채널 추가는 프런트·백엔드 둘 다 바꾸므로 한 빌드에 같이 넣는다 | contract 테스트에 채널 추가 |
| 코딩 표준 | 신규 Go 코드는 `docs/11_coding-standard.md` 접두어 규칙 준수(`pCredit`, `iInFlight`, `G_ci…`) | 리뷰 agent(code-defect-reviewer) 가 편집 직후 검토 |
