# DariTerm: 연달아 보낸 터미널 입력의 순서가 뒤바뀌는 문제 — 원인과 수정 계획

| 항목 | 내용 |
|---|---|
| 작성일 | 2026-11-17 |
| 상태 | **구현 완료(2026-11-17, 결정 D92)** — 자동 테스트 통과. §7.2 측정·§7.3 수동 확인은 남음 |
| 대상 | DariTerm (소스 폴더 `cli-manager-cr`) |
| 문서 성격 | 단독 공유용. 이 문서만 읽고 문제·원인·수정안·결정할 일을 알 수 있게 썼다. 코드 위치는 `파일:줄` 로 적었다(2026-11-17 기준) |

---

## 0. 요약

**문제**: 한 세션에 터미널 입력을 아주 짧은 간격(1ms 미만)으로 연달아 보내면, 서버에 도착하는 순서가 보낸 순서와 달라질 수 있다. 측정에서 연달아 보낸 쌍의 **1~3%** 가 뒤바뀌었다.

**원인**: 앱 틀(Wails v2)이 화면에서 온 호출마다 고루틴을 따로 띄워 처리하고, 백엔드 입력 경로에는 순서를 잡는 장치가 없다.

**영향**: 사람의 타이핑 속도(키 사이 수 ms 이상)에서는 조용할 때 한 건도 뒤바뀌지 않았다. 걸릴 수 있는 것은 터미널 라이브러리(xterm.js)가 스스로 연달아 내보내는 입력이다.
- 한글 등 조합 입력의 확정 글자와 바로 다음 키
- vim·tmux 등이 시작할 때 주고받는 터미널 질의의 응답 여러 개

**수정**: 화면 쪽 어댑터에서 세션별로 입력을 줄 세운다. 앞의 호출이 끝난 뒤 다음 것을 보내고, 기다리는 동안 들어온 입력은 이어 붙여 한 번에 보낸다. 백엔드와 IPC 계약은 바뀌지 않는다. 실험 복사본에서 **뒤바뀜 0 건, 지연 변화 없음**을 확인했다.

---

## 1. 용어

| 용어 | 뜻 |
|---|---|
| 터미널 입력 | 키 입력·붙여넣기·터미널 질의 응답 등 화면에서 서버로 가는 바이트. 화면이 백엔드의 `TermInput(세션 id, 텍스트)` 를 호출해 보낸다 |
| 어댑터 | 화면 코드가 백엔드를 부르는 한 곳(`frontend/src/ipc/backend.ts`). Wails 가 만들어 준 전역 함수를 감싼다 |
| 고루틴 | Go 의 가벼운 실행 단위. 여러 개가 동시에 돌며, 서로의 실행 순서는 보장되지 않는다 |
| xterm.js | 화면의 터미널 라이브러리. 키 입력을 바이트로 바꿔 `onData` 로 내보낸다 |
| 조합 입력 | 한글·일본어·중국어처럼 입력기(IME)가 여러 키를 모아 글자를 만드는 입력 |

---

## 2. 원인 분석

### 2.1 입력이 지나가는 길

```
xterm.js onData(data)                              frontend/src/screens/TerminalView.tsx:282-304
  → getBackend().termInput(세션 id, data)           frontend/src/ipc/backend.ts:71
  → window.go.main.App.TermInput(...)              Wails 가 만든 전역 함수
  → window.chrome.webview.postMessage(메시지)       Wails 런타임. 동기 호출, 보낸 순서 유지
──────────────── 여기까지는 순서가 지켜진다 ────────────────
  → Wails: 메시지마다 go f.dispatchMessage(message)  Wails v2.10.2 frontend.go:724  ★ 순서 보장 없음
  → App.TermInput                                   main.go:56
  → Service.TermInput                               internal/app/app.go:97
  → Session.Input → transport.Write                 internal/session/control.go:16-28
  → (SSH) Shell.Write → stdin.Write                 internal/sshconn/shell.go:78
```

### 2.2 문제 지점

Wails v2 의 메시지 처리 끝부분이다. 메시지를 받은 순서대로 고루틴을 띄우지만, 띄운 고루틴들이 어떤 순서로 실행될지는 Go 스케줄러에 달려 있다.

```go
// github.com/wailsapp/wails/v2@v2.10.2/internal/frontend/desktop/windows/frontend.go:683-725
func (f *Frontend) processMessage(message string) {
	...
	go f.dispatchMessage(message)   // 메시지마다 새 고루틴
}
```

백엔드 쪽은 받은 대로 곧바로 전송에 쓴다. 번호를 매기거나 줄을 세우는 곳이 없다.

```go
// internal/session/control.go:16-24
func (pSession *Session) Input(szText string) {
	pSession.mtxData.Lock()
	transport := pSession.transport
	szState := pSession.info.M_szState
	pSession.mtxData.Unlock()
	if transport != nil {
		_ = transport.Write(Encode(pSession.cfg.Encoding, szText))
		return
	}
	...
}
```

그래서 두 호출이 거의 동시에 들어오면, 나중 호출의 고루틴이 먼저 `transport.Write` 에 도달할 수 있다. 큰 입력은 해석에 시간이 더 걸려, 바로 뒤의 작은 입력에 추월당하기 쉽다.

### 2.3 우리 코드는 연달아 보내지 않는다

`termInput` 을 부르는 곳은 7곳이고, 모두 한 세션에 한 번만 보낸다(동시 입력은 세션마다 한 번씩).

| 위치 | 용도 |
|---|---|
| `frontend/src/screens/TerminalView.tsx:303` | xterm `onData` (키 입력 대부분) |
| `frontend/src/screens/TerminalView.tsx:296` | 위험 명령 확인 뒤 Enter |
| `frontend/src/screens/TerminalView.tsx:340` | Shift+Enter 줄바꿈 |
| `frontend/src/screens/TerminalView.tsx:125` | 텍스트 붙여넣기(확인 창을 거친 경우) |
| `frontend/src/state/paste.ts:19` | 파일·이미지 경로 붙여넣기 |
| `frontend/src/lib/termSend.ts:32`, `:43` | 스니펫, 경로 입력 |

연달아 내보내는 것은 xterm.js 자신이다.
- **조합 입력**: 조합이 끝난 글자를 `setTimeout(0)` 뒤에 내보낸다(xterm `CompositionHelper`). 조합을 끝낸 키(Enter·공백 등)의 입력과 1ms 안에 붙을 수 있다.
- **터미널 질의 응답**: 서버 프로그램이 질의 여러 개(장치 속성, 커서 위치 등)를 한꺼번에 보내면, xterm 이 한 번의 해석 중에 응답을 연달아 `onData` 로 낸다.

---

## 3. 측정 결과

측정용 가짜 전송이 받은 입력 번호의 도착 순서를 기록했다. 값은 "뒤바뀐 수 / 보낸 수"다. 실제 세션 경로(화면 → Wails → 백엔드 → 전송)를 그대로 탔다.

| 보낸 방식 | 현재 (1회) | 현재 (2회) | 현재 (3회) | 수정 후 (2회) |
|---|---|---|---|---|
| 한 작업 안에서 2,000번 연달아 | 233 / 2000 | 129 / 2000 | 184 / 2000 | 0, 0 |
| 두 번 연달아 × 500쌍 | 14 / 1000 | 14 / 1000 | 5 / 1000 | 0, 0 |
| 두 입력 사이 0.2ms × 500쌍 | 10 / 1000 | 9 / 1000 | 4 / 1000 | 0, 0 |
| 두 입력 사이 1ms × 500쌍 | 0 / 1000 | 0 / 1000 | 0 / 1000 | 0, 0 |
| 두 입력 사이 2ms × 500쌍 | 0 / 1000 | 0 / 1000 | 0 / 1000 | 0, 0 |
| 큰 입력(100KB) 바로 뒤 작은 입력 × 50쌍 | 1 / 100 | 0 / 100 | 1 / 100 | 0, 0 |
| 5ms 간격 400번 | 0 / 400 | 0 / 400 | 0 / 400 | 0, 0 |
| 15ms 간격 300번 | 0 / 300 | 0 / 300 | 0 / 300 | 0, 0 |
| 대량 출력(100MB) 중, 두 번 연달아 | 15 / 422 | 9 / 540 | 11 / 770 | 0 / 436, 0 / 432 |
| 대량 출력(100MB) 중, 5ms 간격 | 0 / 744 | 2 / 848 | 1 / 1041 | 0 / 754, 0 / 764 |

읽는 법:
- 연달아(0~0.2ms) 보낸 쌍은 1~3% 가 뒤바뀐다.
- 1ms 이상 떨어지면 조용할 때는 0 건이다. 사람의 타이핑과 키 반복(약 33ms)이 여기에 든다.
- 대량 출력 중에는 5ms 간격에서도 드물게 뒤바뀌었다(3회 합계 2,633개 중 3개).

수정 후의 속도(1회 측정): 조용할 때 키 입력 에코 지연 13.7ms(수정 전 13.6ms), 대량 출력 20.4MB/s(수정 전 20.1MB/s). 차이는 측정 편차 안이다.

측정 환경: AMD Ryzen 5 220 / 32GB, Windows 11 Pro, WebView2 157, Go 1.27.1, Wails v2.10.2, xterm.js 6.0.0.

---

## 4. 확인되지 않은 점

| # | 내용 | 확인 방법 |
|---|---|---|
| 1 | 실제 한글 입력기로 칠 때 뒤바뀜이 얼마나 나오는지. 측정은 프로그램으로 보낸 입력이다 | 수정 전 빌드에서 한글 문장 + Enter 를 반복 입력해 서버 쪽 수신 내용 비교 |
| 2 | vim·tmux 시작 때 실제로 깨진 글자가 남은 사례가 있는지 | 사용자 보고·재현 |
| 3 | 다른 PC(코어 수가 적거나 부하가 큰 PC)에서의 빈도. 측정은 PC 한 대다 | 같은 측정 장치로 재실행 |
| 4 | 실제 SSH 연결에서의 빈도. 측정은 가짜 전송이다. 원인이 Wails 구간이라 전송 종류와는 무관할 것으로 본다 | SSH 세션에서 같은 검사 |

---

## 5. 수정 방안

### 안 A (권장) — 화면 어댑터에서 세션별로 줄 세우기

**동작**
1. 세션마다 "보내는 중" 표시와 대기 문자열을 둔다.
2. 보내는 중이 아니면 바로 `TermInput` 을 호출하고 "보내는 중"으로 둔다.
3. 보내는 중에 새 입력이 오면 대기 문자열 뒤에 **이어 붙인다.**
4. 호출이 끝나면(Wails 가 돌려주는 Promise 가 끝남 = 백엔드가 전송에 다 씀) 대기 문자열이 있으면 한 번에 보내고, 없으면 "보내는 중"을 푼다.

입력은 바이트 흐름이므로 이어 붙여 한 번에 보내도 서버가 받는 내용은 같다. 한 세션의 호출이 한 번에 하나만 떠 있으므로 고루틴이 서로 추월할 수 없다.

**변경 파일**

| 파일 | 변경 |
|---|---|
| `frontend/src/ipc/inputQueue.ts` (새 파일) | 줄 세우기 함수. 어댑터와 분리해 단위 테스트한다 |
| `frontend/src/ipc/backend.ts:71` | `termInput` 이 줄 세우기를 거치게 한다 |
| `frontend/src/ipc/inputQueue.test.ts` (새 파일) | §7.1 의 테스트 |

**Before** (`frontend/src/ipc/backend.ts:71`)

```ts
termInput: (sessionId, text) => app.TermInput(sessionId, text),
```

**After**

```ts
// frontend/src/ipc/inputQueue.ts
// 세션별 입력 줄 세우기. Wails v2 는 호출마다 고루틴을 따로 띄워, 연달아 보낸 호출의 처리 순서를 보장하지 않는다.
// 한 세션의 호출을 한 번에 하나로 묶어 순서를 지킨다. 기다리는 동안 온 입력은 이어 붙여 다음에 한 번에 보낸다.

/** 실제 전송 함수. 끝나면 풀리는 Promise(또는 값)를 돌려준다. */
export type InputSender = (sessionId: string, text: string) => unknown;

export function createInputQueue(send: InputSender): (sessionId: string, text: string) => void {
  const queues = new Map<string, { pending: string }>();

  const pump = (sessionId: string, chunk: string): void => {
    let done: Promise<unknown>;
    try {
      done = Promise.resolve(send(sessionId, chunk));
    } catch {
      done = Promise.resolve();
    }
    // 실패해도 다음 입력은 보낸다(지금도 실패한 입력은 조용히 버려진다).
    done.catch(() => undefined).then(() => {
      const q = queues.get(sessionId);
      if (!q) return;
      if (q.pending) {
        const next = q.pending;
        q.pending = '';
        pump(sessionId, next);
      } else {
        queues.delete(sessionId); // 한가해지면 지운다(닫힌 세션의 항목이 남지 않게)
      }
    });
  };

  return (sessionId, text) => {
    if (!text) return;
    const q = queues.get(sessionId);
    if (q) {
      q.pending += text; // 보내는 중: 뒤에 이어 붙인다
      return;
    }
    queues.set(sessionId, { pending: '' });
    pump(sessionId, text);
  };
}
```

```ts
// frontend/src/ipc/backend.ts — createWailsBackend 안
const sendInput = createInputQueue((sessionId, text) => app.TermInput(sessionId, text));
return {
  ...
  termInput: (sessionId, text) => sendInput(sessionId, text),
  ...
};
```

실험 복사본에서 검증한 코드는 같은 동작을 어댑터 안에 직접 넣은 형태였다. 위 코드는 테스트하기 쉽게 파일을 나눈 정리본이며, 제품에 넣을 때 다시 측정한다(§7.2).

**영향 범위**
- 화면 → 백엔드 터미널 입력 한 통로만 바뀐다. 백엔드, IPC 계약, 화면 컴포넌트, 가짜 백엔드(개발·테스트용)는 그대로다.
- 창 크기 알림(`TermResize`)과 요청-응답(`Invoke`)은 건드리지 않는다.
- 동시 입력(여러 세션에 같이 입력)은 세션마다 줄이 따로라 서로 기다리지 않는다.

**회귀 위험**

| 위험 | 평가·대응 |
|---|---|
| 뒤 입력이 앞 호출이 끝날 때까지 기다린다 | 호출 왕복은 0.3ms(중앙값), 0.6ms(95%)로 쟀다. 조용할 때 에코 지연 변화 없음(13.6 → 13.7ms) |
| 백엔드 쓰기가 막히면(서버가 입력을 읽지 않음) 대기 문자열이 계속 쌓인다 | 지금은 같은 상황에서 고루틴이 입력 수만큼 쌓인다. 메모리로 보면 지금보다 가볍다. 상한을 둘지는 §6 Q2 |
| 호출이 끝나지 않으면 그 세션 입력이 멈춘다 | 위와 같은 상황이며, 지금도 그 입력들은 전달되지 않는다. 세션을 닫거나 재접속하면 새 전송으로 풀린다. 시한을 둘지는 §6 Q2 |
| 입력이 합쳐져 한 번에 간다 | 서버가 받는 바이트는 같다. 끊긴 세션에서 Enter 로 재접속하는 규칙은 "텍스트에 CR 이 들어 있는가"로 판단하므로 합쳐져도 동작한다(`internal/session/control.go:25`) |
| 위험 명령 확인·여러 줄 붙여넣기 확인 | 모두 `termInput` 호출 **전에** 일어나므로 영향 없다 |
| 세션을 닫은 뒤 남은 대기 입력 | 한 번 더 보내지만 백엔드가 모르는 세션 id 는 무시한다(`internal/app/app.go:97-103`). 줄은 한가해지면 스스로 지워진다 |

### 안 B — 입력에 일련번호를 붙여 백엔드에서 순서대로 쓰기

화면이 세션별 번호를 붙여 보내고, 백엔드가 번호 순서대로 전송에 쓴다(먼저 온 뒤 번호는 잠깐 붙잡아 둠).
- 장점: 호출을 기다리지 않아 뒤 입력의 대기가 없다.
- 단점: `TermInput` 의 인자가 바뀐다(IPC 계약·명세 변경). 번호가 빠졌을 때 얼마나 기다릴지 같은 규칙이 더 필요하다. 백엔드·화면·명세를 모두 고쳐야 한다.
- 안 A 의 대기가 0.3ms 수준이라 얻는 것이 작다. **채택하지 않는다.**

### 안 C — 앱 틀을 Wails v3 로 옮겨 Streams 사용

v3 의 Streams 는 순서를 보장한다(측정에서 0 건). 그러나 v3 의 일반 호출은 v2 보다 낫지 않았고(큰 입력 뒤 작은 입력 50쌍 모두 뒤바뀜), 이전 비용이 크며 키 입력 에코가 약 3ms 느려진다. 이 문제만을 위해서는 **채택하지 않는다.**

### 권장

**안 A.** 한 파일 수준의 변경으로 뒤바뀜이 0 건이 되고, 측정된 속도 손해가 없다.

---

## 6. 결정이 필요한 질문

DariTerm 은 명세(`spec/`)를 기준으로만 구현하는 clean-room 방식이다. 답은 이 표의 "답" 칸에 적는다.

| # | 질문 | 구현자 제안 | 관련 명세 | 답 | 명세 반영 |
|---|---|---|---|---|---|
| Q1 | "한 세션의 터미널 입력은 보낸 순서대로 전송에 쓰인다"를 명세에 적을 것인가? 지금 명세는 `TermInput` 을 "응답 없음"으로만 정한다 | 적는다. 구현 방법(화면에서 줄 세우기)은 명세에 넣지 않고 보장만 적는다 | IPC 메서드(ipc §1), 터미널 입력(behavior/terminal) | 제안대로(2026-11-17, D92). spec/ipc.md §1 `TermInput` 행에 보장 문구 | [x] |
| Q2 | 백엔드 쓰기가 막혀 대기 입력이 쌓일 때 상한이나 시한을 둘 것인가? | 두지 않는다(지금 동작보다 나빠지지 않음). 문제 사례가 나오면 그때 정한다 | 세션 입력(behavior/session §8) | 제안대로 — 두지 않음(D92) | — |
| Q3 | 명세 변경 없이 구현만 먼저 넣어도 되는가? 밖에서 보이는 동작은 "순서가 지켜진다"는 것뿐이다 | 가능하다고 본다. 어댑터 내부 구현이며 계약(인자·응답)이 그대로다 | — | 명세 보장 문구와 함께 넣음(D92) | — |

---

## 7. 검증 계획

### 7.1 자동 테스트 (`frontend`, `npx vitest run`)

`frontend/src/ipc/inputQueue.test.ts` — 전송 함수를 "끝나는 시점을 테스트가 정하는 가짜"로 바꿔 검사한다.

| 테스트 | 확인 내용 |
|---|---|
| 순서 | 연달아 넣은 입력이 넣은 순서대로, 한 번에 하나씩 전송 함수에 간다 |
| 이어 붙이기 | 앞 호출이 끝나기 전에 넣은 입력 여러 개가 다음 호출 한 번에 합쳐져 간다(`"a"`, `"b"`, `"c"` → `"a"` 다음 `"bc"`) |
| 세션 독립 | 한 세션이 기다리는 동안 다른 세션의 입력은 바로 간다 |
| 실패 뒤 계속 | 전송 함수가 실패(거부·예외)해도 다음 입력은 간다 |
| 정리 | 대기 입력이 없어지면 그 세션의 줄이 지워지고, 다음 입력은 다시 바로 간다 |
| 빈 입력 | 빈 문자열은 보내지 않는다 |
| 동기 전송 함수 | 전송 함수가 Promise 가 아닌 값을 돌려줘도 동작한다(가짜 백엔드 호환) |

그 밖: `npx tsc --noEmit`, 기존 화면 테스트 전체 통과. 백엔드는 바뀌지 않으므로 `go test -race ./...` 는 회귀 확인용으로만 돌린다.

### 7.2 측정으로 확인 (수정 전후 비교)

§3 과 같은 측정 장치로 제품 빌드를 다시 잰다.

| 항목 | 기준 |
|---|---|
| 순서 검사 10가지 방식 | 모두 0 건 (2회 이상) |
| 조용할 때 키 입력 에코 지연 | 수정 전과 차이 1ms 이내 |
| 대량 출력 처리량 | 수정 전과 차이 5% 이내 |

### 7.3 수동 확인

| # | 시나리오 | 기대 결과 | 결과 |
|---|---|---|---|
| 1 | SSH 세션에서 한글 문장을 빠르게 치고 곧바로 Enter (20회) | 서버가 받은 내용이 입력한 그대로 | |
| 2 | vim, tmux 를 여러 번 시작 | 시작 화면에 깨진 글자가 남지 않음 | |
| 3 | 긴 텍스트(수백 KB) 붙여넣기 직후 Enter | 붙여넣은 내용 뒤에 Enter 가 적용됨 | |
| 4 | 동시 입력을 켜고 여러 칸에 빠르게 입력 | 모든 칸이 같은 내용을 같은 순서로 받음 | |
| 5 | 끊긴 세션에서 Enter | 재접속이 시작됨(기존 동작 유지) | |
| 6 | 대량 출력(큰 파일 `cat`) 중에 타이핑·Ctrl+C | 입력이 지연 없이 전달됨 | |

---

## 8. 진행 체크리스트

- [x] §6 질문 Q1~Q3 답 받기 → 답을 §6 표에 적는다 (구현자 제안대로, D92)
- [x] `frontend/src/ipc/inputQueue.ts` 작성
- [x] `frontend/src/ipc/backend.ts` 의 `termInput` 연결
- [x] `frontend/src/ipc/inputQueue.test.ts` 작성(§7.1 7가지), `npx vitest run`(36 파일 216 테스트 통과), `npx tsc --noEmit`
- [ ] §7.2 측정 → 결과를 §7.2 표 아래에 적는다
- [ ] §7.3 수동 확인 → 결과를 표에 적는다
- [ ] §4 의 미확인 항목 1(한글 입력기 실측) 결과 기록
- [x] 명세 반영(Q1 답에 따라) — spec/ipc.md §1
