# 창 머리글에 세션별 입력·출력 전송량

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D160). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**첨부**: [2026-11-27-20-session-traffic-title/](./2026-11-27-20-session-traffic-title/) — 도움말 그림 2장(`img/`), 다시 만든 매뉴얼 HTML·PDF 4개(`manual/`). 적용은 §5.0.
**관련**: [2026-11-27-13-forward-traffic.md](./2026-11-27-13-forward-traffic.md)(D151 — 포워딩 전송량, 같은 atomic 집계 방식)

---

## 0. 한 줄 요약

열린 세션마다 **보낸 바이트(↑ 입력)·받은 바이트(↓ 출력)** 를 세어 창 머리글 오른쪽 버튼들 앞에 `↑1.2 K ↓34 M` 처럼 보인다. 백엔드는 읽기·쓰기마다 atomic 더하기 한 번으로 세고, **1초마다 값이 바뀐 세션만** `session.traffic` 이벤트 하나로 보낸다. 프런트는 세션별로 구독하는 작은 요소만 다시 그린다. 사용자 요청 "제목줄이 이상하지 않도록"에 맞춰 **폭이 고정**(값이 바뀌어도 머리글이 흔들리지 않음)이고 좁은 창에서는 숨는다. 함께 좁은 창에서 환경 태그("운영")가 두 줄로 꺾이던 기존 문제도 고쳤다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 무엇을 세나 | 세션 연결(Transport)에서 **읽은** 바이트 = 받음(in), **쓴** 바이트 = 보냄(out). 화면 출력·ZMODEM 받기 / 키 입력·붙여넣기·명령 모음·매크로·로그인 스크립트·자동 실행 명령·ZMODEM 올리기·중단 바이트열 | 사용자가 말한 "입력·출력". 서버 인코딩으로 바뀐 실제 바이트 |
| 세지 않는 것 | SFTP(별도 채널), 포트 포워딩(D151 에서 따로 셈), SSH 암호화·프로토콜 머리 | 세션 터미널의 입출력만. 포워딩은 정보 창에 이미 있다 |
| 범위 | **세션을 연 뒤 누적, 재접속해도 이어서** | 자동 재접속(서버 timeout 등) 때 값이 0 으로 사라지면 헷갈린다. 포워딩(D151)은 연결 객체에 묶여 재접속마다 새로 센다 — 다른 대상이라 다르게 둠 |
| 세는 곳 | 세션의 쓰기 지점을 모두 `pSession.send(conn, b)` 로, 읽기는 `readLoop` 와 `zmodemLink` 의 `conn.Read` 바로 뒤 | Transport 를 감싸면 `Transport()` 로 SFTP·포워딩 기능을 형 단언으로 찾는 곳(`ForwardReporter`, `SecretProvider`, `channels_sftp.go`)이 깨진다 |
| 보내는 방식 | `Manager.StartTraffic(1s)` 고루틴이 **바뀐 세션만** 모아 `session.traffic {id:{in,out}}` 하나로. 바뀐 게 없으면 보내지 않는다 | 출력마다 보내면 세션 수 × 출력 빈도만큼 이벤트·재렌더. 1초 묶음이면 세션이 수십 개여도 이벤트 1개 |
| `session.state` 에 싣기 | 하지 않는다 | 상태 이벤트는 상태가 바뀔 때만 오는 계약이고, 세션 정보가 바뀌면 탭·칸·목록 전체가 다시 그려진다 |
| 프런트 저장소 | 작업 영역 저장소와 **따로** `useTraffic`. 바뀐 세션 항목만 새 객체, 같은 값이면 상태 그대로 | 머리글 하나만 다시 그리게 |
| 표시 | `↑보냄 ↓받음`, 짧은 표기 **5글자 이하**(`999 B`, `9.9 K`, `512 M`, `1.0 M`), 툴팁에는 기존 표기(`12.3 KB`) | 머리글이 좁다 |
| 제목줄 안정 | 화살표+숫자 칸 둘을 **고정 6ch, 왼쪽 맞춤, 고정폭 숫자**, `flex: none`. 머리글 360px 이하면 컨테이너 질의로 숨김. 끊긴 세션은 흐리게(55%) 마지막 값 | 처음 오른쪽 맞춤으로 만들었더니 `↓  5 B` 처럼 화살표와 숫자가 떨어져 캡처에서 이상해 보였다 → 왼쪽 맞춤(남는 폭은 칸 뒤로) |
| 환경 태그 | `.pane-head .env-tag { flex: none; white-space: nowrap }` | 캡처 중 발견: 창 3개(머리글 333px)에서 "운영" 이 두 줄로 꺾였다(이번 변경 전부터 있던 문제) |
| 켜고 끄는 설정 | 두지 않는다 | 요청 없음. 필요하면 후속 |

## 2. 성능

| 경로 | 더해진 일 | 부담 |
|---|---|---|
| 출력 읽기(32KB 버퍼마다) | `atomic.Int64.Add` 1번 | 수 ns. 같은 루프의 디코딩·ZMODEM 감지·로그보다 훨씬 작다 |
| 입력 쓰기 | `atomic.Int64.Add` 1번 | 무시 |
| 1초 주기 | 세션 목록 잠금 → 세션마다 atomic Load 2번 → 바뀐 것만 맵에 | 세션 100개여도 마이크로초 단위. 바뀐 게 없으면 이벤트 없음 |
| 프런트 | 1초에 이벤트 최대 1개, 바뀐 세션의 `PaneTraffic` 만 다시 그림 | 터미널·탭·칸은 다시 그리지 않는다 |

## 3. 바꾼 코드

### 3.1 새 파일 `internal/session/traffic.go` (전체)

```go
package session

import (
	"sync"
	"sync/atomic"
	"time"
)

// G_cdurTrafficInterval 은 세션 전송량 이벤트(session.traffic)를 보내는 간격이다 (session.md §8, D160).
const G_cdurTrafficInterval = time.Second

// TrafficInfo 는 세션 하나의 전송량이다 (spec/ipc.md session.traffic, D160).
// 세션을 연 뒤 연결에서 읽은(In)·연결에 쓴(Out) 바이트 누적이고, 재접속해도 이어서 센다.
// SSH 는 채널 데이터 기준이다(암호화·프로토콜 머리 제외). SFTP·포트 포워딩은 따로 센다(여기에 들지 않는다).
type TrafficInfo struct {
	M_i64In  int64 `json:"in"`
	M_i64Out int64 `json:"out"`
}

// trafficCounter 는 세션 하나의 전송량 집계다. 읽기 루프·입력·ZMODEM 이 동시에 더하므로 atomic 이다.
type trafficCounter struct {
	in  atomic.Int64
	out atomic.Int64
}

func (pCounter *trafficCounter) addIn(nRead int) {
	if nRead > 0 {
		pCounter.in.Add(int64(nRead))
	}
}

func (pCounter *trafficCounter) addOut(nWritten int) {
	if nWritten > 0 {
		pCounter.out.Add(int64(nWritten))
	}
}

func (pCounter *trafficCounter) snapshot() TrafficInfo {
	return TrafficInfo{M_i64In: pCounter.in.Load(), M_i64Out: pCounter.out.Load()}
}

// Traffic 은 세션을 연 뒤 지금까지의 전송량이다.
func (pSession *Session) Traffic() TrafficInfo {
	return pSession.traffic.snapshot()
}

// send 는 연결에 바이트를 쓰고, 성공하면 보낸 양으로 센다. 세션이 연결에 쓰는 곳은 모두 이것을 거친다.
func (pSession *Session) send(conn Transport, bData []byte) error {
	errWrite := conn.Write(bData)
	if errWrite == nil {
		pSession.traffic.addOut(len(bData))
	}
	return errWrite
}

// trafficLoop 는 Manager 의 주기 보고 상태다.
type trafficLoop struct {
	once   sync.Once
	chStop chan struct{}
	// mapSent 는 마지막으로 보낸 값이다. 보고 고루틴(또는 시험의 collectTraffic 직접 호출) 하나만 쓴다.
	mapSent map[string]TrafficInfo
}

// StartTraffic 은 durInterval 마다 값이 바뀐 세션의 전송량을 session.traffic 이벤트 하나로 보낸다 (D160).
// 바뀐 세션이 없으면 보내지 않는다. StopTraffic 으로 멈춘다. 두 번 불러도 한 번만 돈다.
func (pManager *Manager) StartTraffic(durInterval time.Duration) {
	pLoop := &pManager.traffic
	pLoop.once.Do(func() {
		chStop := make(chan struct{})
		pManager.mtxData.Lock()
		pLoop.chStop = chStop
		pManager.mtxData.Unlock()
		go func() {
			ticker := time.NewTicker(durInterval)
			defer ticker.Stop()
			for {
				select {
				case <-chStop:
					return
				case <-ticker.C:
				}
				if mapChanged := pManager.collectTraffic(); len(mapChanged) > 0 {
					pManager.emit("session.traffic", mapChanged)
				}
			}
		}()
	})
}

// StopTraffic 은 주기 보고를 멈춘다. 시작하지 않았거나 이미 멈췄으면 아무 일도 하지 않는다.
func (pManager *Manager) StopTraffic() {
	pManager.mtxData.Lock()
	chStop := pManager.traffic.chStop
	pManager.traffic.chStop = nil
	pManager.mtxData.Unlock()
	if chStop != nil {
		close(chStop)
	}
}

// collectTraffic 은 마지막 보고 뒤 값이 바뀐 세션의 전송량을 돌려준다. 닫힌 세션의 기록은 지운다.
func (pManager *Manager) collectTraffic() map[string]TrafficInfo {
	pManager.mtxData.Lock()
	mapNow := make(map[string]TrafficInfo, len(pManager.mapSessions))
	for szId, pSession := range pManager.mapSessions {
		mapNow[szId] = pSession.Traffic()
	}
	pManager.mtxData.Unlock()

	pLoop := &pManager.traffic
	if pLoop.mapSent == nil {
		pLoop.mapSent = map[string]TrafficInfo{}
	}
	for szId := range pLoop.mapSent {
		if _, bOpen := mapNow[szId]; !bOpen {
			delete(pLoop.mapSent, szId)
		}
	}
	mapChanged := map[string]TrafficInfo{}
	for szId, info := range mapNow {
		if pLoop.mapSent[szId] != info {
			mapChanged[szId] = info
			pLoop.mapSent[szId] = info
		}
	}
	return mapChanged
}
```

### 3.2 새 파일 `frontend/src/state/traffic.ts` (전체)

```ts
// 세션별 전송량(창 머리글 표시). 근거: spec/ipc.md session.traffic, spec/ui/panes.md §2.7 (D160).
// 작업 영역 저장소(sessions)와 따로 둔다: 1초마다 오는 값이 세션 목록·탭·칸 전체를 다시 그리지 않게,
// 바뀐 세션의 항목만 새 객체로 바꾸고 각 머리글은 자기 세션 항목만 구독한다.
import { create } from 'zustand';

export interface SessionTraffic {
  /** 세션을 연 뒤 연결에서 받은 바이트(서버 → 터미널 출력). */
  in: number;
  /** 세션을 연 뒤 연결로 보낸 바이트(키 입력·붙여넣기·명령 모음 등). */
  out: number;
}

interface TrafficState {
  bySession: Record<string, SessionTraffic>;
  /** session.traffic 이벤트 값(바뀐 세션만)을 합친다. 값이 같으면 객체를 바꾸지 않는다. */
  apply: (changed: Record<string, Partial<SessionTraffic>> | null | undefined) => void;
  /** 닫힌 세션의 항목을 지운다. */
  remove: (sessionId: string) => void;
}

export const useTraffic = create<TrafficState>((set) => ({
  bySession: {},
  apply(changed) {
    if (!changed || typeof changed !== 'object') return;
    set((s) => {
      let next: Record<string, SessionTraffic> | null = null;
      for (const [id, v] of Object.entries(changed)) {
        const nIn = Number(v?.in) || 0;
        const nOut = Number(v?.out) || 0;
        const cur = s.bySession[id];
        if (cur && cur.in === nIn && cur.out === nOut) continue;
        next ??= { ...s.bySession };
        next[id] = { in: nIn, out: nOut };
      }
      return next ? { bySession: next } : s;
    });
  },
  remove(sessionId) {
    set((s) => {
      if (!(sessionId in s.bySession)) return s;
      const next = { ...s.bySession };
      delete next[sessionId];
      return { bySession: next };
    });
  },
}));

/** 세션 하나의 전송량. 아직 받은 값이 없으면 undefined. */
export function useSessionTraffic(sessionId: string | undefined): SessionTraffic | undefined {
  return useTraffic((s) => (sessionId ? s.bySession[sessionId] : undefined));
}
```

### 3.3 `internal/session/session.go`

```diff
 type Session struct {
 	…
 	zCancel      atomic.Bool // 사용자가 ZMODEM 전송을 취소했다
+	traffic      trafficCounter // 세션을 연 뒤 연결에서 읽은·쓴 바이트 (traffic.go, D160)
 }
```
(gofmt 가 이 구조체의 필드 정렬을 다시 맞춘다.)

```diff
 // readLoop
 		} else {
 			nRead, errRead = conn.Read(bBuf)
+			pSession.traffic.addIn(nRead) // 닫힌 뒤 읽힌 조각도 연결에서 받은 것이다
 		}
```

```diff
 // scheduleStartup
-		_ = pSession.transport.Write(Encode(pSession.cfg.Encoding, pSession.cfg.StartupInput))
+		_ = pSession.send(pSession.transport, Encode(pSession.cfg.Encoding, pSession.cfg.StartupInput))
```

### 3.4 `internal/session/control.go` — `Input`

```diff
-		_ = transport.Write(Encode(pSession.cfg.Encoding, szText))
+		_ = pSession.send(transport, Encode(pSession.cfg.Encoding, szText))
```

### 3.5 `internal/session/loginscript.go`

```diff
-		_ = conn.Write(Encode(pSession.cfg.Encoding, szSend+"\r"))
+		_ = pSession.send(conn, Encode(pSession.cfg.Encoding, szSend+"\r"))
```

### 3.6 `internal/session/zmodem.go`

```diff
 // cancelZmodem, runZmodem 의 시한 중단 (2곳)
-	_ = conn.Write(zmodem.AbortSequence())
+	_ = pSession.send(conn, zmodem.AbortSequence())

 // runZmodem
-	pLink := &zmodemLink{conn: conn, fnCancel: pSession.zCancel.Load}
+	pLink := &zmodemLink{conn: conn, fnCancel: pSession.zCancel.Load, pTraffic: &pSession.traffic}

 type zmodemLink struct {
 	conn      Transport
 	fnCancel  func() bool
 	chPending chan readResult // 걸어 둔 읽기. 없으면 nil
 	bRest     []byte          // 읽었지만 아직 넘기지 않은 바이트
+	pTraffic  *trafficCounter // 세션 전송량 (D160). nil 이면 세지 않는다
 }

-func (pLink *zmodemLink) Write(bData []byte) error { return pLink.conn.Write(bData) }
+func (pLink *zmodemLink) Write(bData []byte) error {
+	errWrite := pLink.conn.Write(bData)
+	if errWrite == nil && pLink.pTraffic != nil {
+		pLink.pTraffic.addOut(len(bData))
+	}
+	return errWrite
+}

 // wait
-		conn := pLink.conn
+		conn, pTraffic := pLink.conn, pLink.pTraffic
 		go func() {
 			bBuf := make([]byte, 32*1024)
 			nRead, errRead := conn.Read(bBuf)
+			if pTraffic != nil {
+				pTraffic.addIn(nRead)
+			}
 			chRead <- readResult{bData: bBuf[:nRead], err: errRead}
 		}()
```

ZMODEM 이 끝날 때 걸어 둔 읽기(`chAhead`)는 `zmodemLink.wait` 에서 이미 셌으므로 `readLoop` 는 그 결과를 다시 세지 않는다(`conn.Read` 를 직접 부른 경우만 센다).

### 3.7 `internal/session/manager.go`

```diff
 type Manager struct {
 	mtxData     sync.Mutex
 	mapSessions map[string]*Session
 	emit        EmitFunc
+	traffic     trafficLoop // 세션 전송량 주기 보고 (traffic.go, D160)
 }
```

### 3.8 `internal/app/app.go`

```diff
 	pService.pSessions = session.NewManager(func(szEvent string, anyData ...any) {
 		pService.pRouter.Emit(szEvent, anyData...)
 	})
+	// 창 머리글의 세션별 전송량: 1초마다 바뀐 세션만 묶어 session.traffic 으로 보낸다 (D160).
+	pService.pSessions.StartTraffic(session.G_cdurTrafficInterval)
 	pService.applyLanguage()
```

```diff
 // Close
 	pService.pLogClean.Stop()
+	pService.pSessions.StopTraffic()
 	chDone := make(chan struct{})
```

### 3.9 프런트엔드

`frontend/src/ipc/backend.ts`

```diff
 export type EventName =
   | 'session.data'
   | 'session.state'
+  | 'session.traffic'
```

`frontend/src/state/actions.ts`

```diff
 import { useUi } from './ui';
+import { useTraffic } from './traffic';
 …
     b.on('session.state', (info: SessionInfo) => { … }),
+    // 창 머리글의 세션별 전송량: 백엔드가 1초마다 바뀐 세션만 보낸다 (D160).
+    b.on('session.traffic', (changed: Record<string, { in: number; out: number }>) => {
+      useTraffic.getState().apply(changed);
+    }),
     b.on('session.data', (id: string, text: string) => {
```

`frontend/src/state/workspace.ts`

```diff
 import { useApp } from './app';
+import { useTraffic } from './traffic';
 …
   removeSession(sessionId) {
     dropDeferred(sessionId);
+    useTraffic.getState().remove(sessionId);
```

`frontend/src/lib/format.ts` — `formatBytes` 아래에 추가

```ts
const COMPACT_UNITS = ['K', 'M', 'G', 'T', 'P'];

/**
 * 좁은 자리(창 머리글)용 바이트 표기: **늘 5글자 이하**라 값이 바뀌어도 폭이 거의 그대로다 (spec/ui/panes.md §2.7, D160).
 * 1000 미만 `n B`, 그 위로는 1024 단위로 나눠 10 미만이면 소수 한 자리(`9.9 K`), 아니면 정수(`512 M`).
 * 정수로 반올림해 1000 이 되면 다음 단위로 올린다(`1000 K` 대신 `1.0 M`).
 */
export function formatBytesCompact(n: number): string {
  if (!Number.isFinite(n) || n < 1000) return `${Math.max(0, Math.floor(Number.isFinite(n) ? n : 0))} B`;
  let v = n;
  for (const unit of COMPACT_UNITS) {
    v /= 1024;
    if (v < 9.95) return `${v.toFixed(1)} ${unit}`;
    if (Math.round(v) < 1000) return `${Math.round(v)} ${unit}`;
  }
  return `${Math.round(v)} ${COMPACT_UNITS[COMPACT_UNITS.length - 1]}`;
}
```

`frontend/src/screens/Panes.tsx`

```diff
 import { failedCount } from '../lib/forwards';
+import { useSessionTraffic } from '../state/traffic';
+import { formatBytes, formatBytesCompact } from '../lib/format';
```

`PaneView` 바로 위에 추가:

```tsx
/**
 * 창 머리글의 세션 전송량 (ui/panes.md §2.7, D160): `↑ 보낸 것 ↓ 받은 것`, 세션을 연 뒤 누적.
 * 이 세션의 값만 구독하므로 1초마다 값이 바뀌어도 이 작은 요소만 다시 그려진다.
 * 제목줄이 흔들리지 않게: 화살표+숫자 칸은 고정 폭(6글자, 왼쪽 맞춤, 고정폭 숫자)이고 줄어들지 않으며, 칸이 좁으면 CSS 가 통째로 숨긴다.
 */
export function PaneTraffic({ sessionId, live }: { sessionId: string; live: boolean }) {
  const traffic = useSessionTraffic(sessionId);
  const nIn = traffic?.in ?? 0;
  const nOut = traffic?.out ?? 0;
  const title = tr('pane.trafficTitle', { out: formatBytes(nOut), in: formatBytes(nIn) });
  return (
    <span className={`pane-traffic${live ? '' : ' idle'}`} title={title} aria-label={title} data-testid="pane-traffic">
      <span className="v" aria-hidden="true">↑{formatBytesCompact(nOut)}</span>
      <span className="v" aria-hidden="true">↓{formatBytesCompact(nIn)}</span>
    </span>
  );
}
```

머리글:

```diff
         <span className="spacer" />
+        {info && <PaneTraffic sessionId={info.id} live={info.state === 'connected'} />}
         {hasSession && entry?.info.state !== 'connected' && (
```

`frontend/src/styles/app.css` — `.pane-head .spacer { flex: 1; }` 다음

```css
/* 세션 전송량 (ui/panes.md §2.7, D160). 제목줄이 흔들리지 않게: 줄어들지 않는 고정 폭 칸 둘(화살표+숫자 6글자, 왼쪽 맞춤, 고정폭 숫자 —
   남는 폭은 칸 뒤로 가서 화살표와 숫자가 떨어지지 않는다),
   제목·주소가 먼저 말줄임으로 줄고, 칸 폭이 360px 아래로 좁아지면 통째로 숨긴다(컨테이너 질의). 끊긴 세션은 흐리게 마지막 값을 둔다. */
.pane-head { container-type: inline-size; container-name: pane-head; }
.pane-traffic {
  flex: none; display: inline-flex; align-items: center; gap: 6px; margin-right: 2px;
  font-family: var(--mono, ui-monospace, Consolas, monospace); font-size: 10.5px; line-height: 1; color: var(--fg-3);
  font-variant-numeric: tabular-nums; white-space: nowrap; user-select: none;
}
.pane-traffic .v { display: inline-block; min-width: 6ch; text-align: left; }
.pane.focused .pane-traffic, .pane:hover .pane-traffic { color: var(--fg-2); }
.pane-traffic.idle, .pane.focused .pane-traffic.idle { opacity: 0.55; }
@container pane-head (max-width: 360px) { .pane-traffic { display: none; } }
```

`.env-tag { … }` 다음

```css
/* 좁은 칸에서 환경 태그("운영")가 두 줄로 꺾이지 않게 — 제목·주소가 먼저 말줄임으로 준다 (D160 때 함께 고침). */
.pane-head .env-tag { flex: none; white-space: nowrap; }
```

`frontend/src/ipc/fakeBackend.ts` (개발 서버·도움말 캡처용 가짜 백엔드)

```diff
   const emit = (event: EventName, ...args: unknown[]) => {
     listeners.get(event)?.forEach((h) => h(...args));
+    // 세션 전송량 흉내(D160): 실제 백엔드는 1초마다 묶어 보내지만 가짜는 출력·입력 때마다 곧바로 보낸다.
+    if (event === 'session.data') countTraffic(String(args[0]), String(args[1] ?? ''), '');
   };
+  const traffic = new Map<string, { in: number; out: number }>();
+  const byteLength = (s: string) => new TextEncoder().encode(s).length;
+  function countTraffic(id: string, received: string, sent: string) {
+    if (!sessions.has(id)) return;
+    const cur = traffic.get(id) ?? { in: 0, out: 0 };
+    const next = { in: cur.in + byteLength(received), out: cur.out + byteLength(sent) };
+    traffic.set(id, next);
+    listeners.get('session.traffic')?.forEach((h) => h({ [id]: next }));
+  }
 …
     'session.close': ([id]) => {
       sessions.delete(String(id));
+      traffic.delete(String(id));
 …
     termInput(sessionId, text) {
+      countTraffic(sessionId, '', text);
```

### 3.10 문구 (10개 언어) — `locales/<언어>.json`, `"pane.broadcasting"` 바로 다음 줄

| 언어 | `pane.trafficTitle` |
|---|---|
| ko | `전송량 (세션을 연 뒤) — 보냄(입력) {out} · 받음(출력) {in}` |
| en | `Traffic since the session opened — sent (input) {out} · received (output) {in}` |
| ja | `転送量（セッションを開いてから）— 送信（入力）{out}・受信（出力）{in}` |
| zh | `流量（会话打开以来）— 发送（输入）{out} · 接收（输出）{in}` |
| tw | `流量（工作階段開啟以來）— 傳送（輸入）{out} · 接收（輸出）{in}` |
| de | `Datenmenge seit dem Öffnen der Sitzung — gesendet (Eingabe) {out} · empfangen (Ausgabe) {in}` |
| fr | `Trafic depuis l'ouverture de la session — envoyé (saisie) {out} · reçu (sortie) {in}` |
| es | `Tráfico desde que se abrió la sesión — enviado (entrada) {out} · recibido (salida) {in}` |
| pt | `Tráfego desde a abertura da sessão — enviado (entrada) {out} · recebido (saída) {in}` |
| ru | `Трафик с открытия сеанса — отправлено (ввод) {out} · получено (вывод) {in}` |

## 4. 테스트

### 4.1 새 파일 `internal/session/traffic_test.go` (전체)

```go
package session

import (
	"sync"
	"testing"
	"time"
)

// trafficRecorder 는 session.traffic 이벤트만 모은다.
type trafficRecorder struct {
	eventRecorder
	mtxTraffic sync.Mutex
	slTraffic  []map[string]TrafficInfo
}

func (pRec *trafficRecorder) emit(szEvent string, anyData ...any) {
	if szEvent == "session.traffic" {
		pRec.mtxTraffic.Lock()
		pRec.slTraffic = append(pRec.slTraffic, anyData[0].(map[string]TrafficInfo))
		pRec.mtxTraffic.Unlock()
		return
	}
	pRec.eventRecorder.emit(szEvent, anyData...)
}

func (pRec *trafficRecorder) last() (map[string]TrafficInfo, int) {
	pRec.mtxTraffic.Lock()
	defer pRec.mtxTraffic.Unlock()
	if len(pRec.slTraffic) == 0 {
		return nil, 0
	}
	return pRec.slTraffic[len(pRec.slTraffic)-1], len(pRec.slTraffic)
}

// openConnected 는 가짜 연결로 세션을 열고 연결될 때까지 기다린다.
func openConnected(t *testing.T, pManager *Manager, rec *eventRecorder) (string, *fakeTransport) {
	t.Helper()
	transport := newFakeTransport()
	info, errOpen := pManager.Open(sshConfig(func(func(string)) (Transport, error) { return transport, nil }, false))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	waitUntil(t, func() bool { return rec.hasState(G_cszStateConnected) }, time.Second)
	return info.M_szId, transport
}

// 연결에서 읽은 바이트는 in, 입력으로 쓴 바이트는 out 으로 센다 (D160). 한글은 UTF-8 바이트 수다.
func TestTrafficCountsReadAndInput(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	pSession, _ := pManager.Get(szId)

	transport.chIn <- []byte("hello")
	transport.chIn <- []byte("한글") // 6 바이트
	pSession.Input("ls\r")
	waitUntil(t, func() bool { return pSession.Traffic() == TrafficInfo{M_i64In: 11, M_i64Out: 3} }, time.Second)
	pManager.Close(szId)
	if got := pSession.Traffic(); got.M_i64In != 11 || got.M_i64Out != 3 {
		t.Fatalf("닫은 뒤에도 값은 그대로여야 함: %+v", got)
	}
}

// 보고는 값이 바뀐 세션만 싣고, 닫힌 세션의 기록은 지운다.
func TestCollectTrafficOnlyChanged(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	pSession, _ := pManager.Get(szId)

	if mapChanged := pManager.collectTraffic(); len(mapChanged) != 0 {
		t.Fatalf("아직 오간 것이 없으면 보고하지 않는다: %v", mapChanged)
	}
	transport.chIn <- []byte("abc")
	waitUntil(t, func() bool { return pSession.Traffic().M_i64In == 3 }, time.Second)
	if mapChanged := pManager.collectTraffic(); mapChanged[szId] != (TrafficInfo{M_i64In: 3}) {
		t.Fatalf("바뀐 세션을 실어야 함: %v", mapChanged)
	}
	if mapChanged := pManager.collectTraffic(); len(mapChanged) != 0 {
		t.Fatalf("그대로면 다시 싣지 않는다: %v", mapChanged)
	}
	pSession.Input("x")
	if mapChanged := pManager.collectTraffic(); mapChanged[szId] != (TrafficInfo{M_i64In: 3, M_i64Out: 1}) {
		t.Fatalf("입력도 바뀐 것으로 본다: %v", mapChanged)
	}
	pManager.Close(szId)
	pManager.collectTraffic()
	if _, bLeft := pManager.traffic.mapSent[szId]; bLeft {
		t.Fatal("닫힌 세션의 기록은 지워야 함")
	}
}

// 주기 보고가 session.traffic 하나로 보내고, StopTraffic 뒤에는 보내지 않는다. 두 번 멈춰도 괜찮다.
func TestStartTrafficEmitsAndStops(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	pManager.StartTraffic(10 * time.Millisecond)
	pManager.StartTraffic(10 * time.Millisecond) // 두 번째는 무시
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	transport.chIn <- []byte("12345")
	waitUntil(t, func() bool {
		mapLast, _ := rec.last()
		return mapLast[szId].M_i64In == 5
	}, time.Second)
	pManager.StopTraffic()
	pManager.StopTraffic()
	_, iBefore := rec.last()
	transport.chIn <- []byte("more")
	time.Sleep(60 * time.Millisecond)
	if _, iAfter := rec.last(); iAfter != iBefore {
		t.Fatalf("멈춘 뒤에는 보내지 않아야 함: %d → %d", iBefore, iAfter)
	}
	pManager.CloseAll()
}

// ZMODEM 전송 중 엔진이 쓰는 연결도 같은 카운터로 센다.
func TestZmodemLinkCountsTraffic(t *testing.T) {
	transport := newFakeTransport()
	var counter trafficCounter
	pLink := &zmodemLink{conn: transport, pTraffic: &counter}
	if errWrite := pLink.Write([]byte("abcd")); errWrite != nil {
		t.Fatal(errWrite)
	}
	transport.chIn <- []byte("xyz")
	bBuf := make([]byte, 16)
	if nRead, errRead := pLink.Read(bBuf); errRead != nil || nRead != 3 {
		t.Fatalf("읽기: %d %v", nRead, errRead)
	}
	if got := counter.snapshot(); got != (TrafficInfo{M_i64In: 3, M_i64Out: 4}) {
		t.Fatalf("ZMODEM 연결 전송량: %+v", got)
	}
}
```

### 4.2 새 파일 `frontend/src/screens/Panes.traffic.test.tsx` (전체)

```tsx
// 창 머리글의 세션 전송량 (D160, spec/ui/panes.md §2.7): 짧은 표기·고정 폭, 바뀐 세션만 새 객체, 닫으면 지움.
import { afterEach, describe, expect, it } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import { PaneTraffic } from './Panes';
import { useTraffic } from '../state/traffic';
import { formatBytesCompact } from '../lib/format';

afterEach(() => useTraffic.setState({ bySession: {} }));

describe('formatBytesCompact (D160)', () => {
  it('늘 5글자 이하, 단위 경계에서 올림', () => {
    const cases: [number, string][] = [
      [0, '0 B'],
      [999, '999 B'],
      [1000, '1.0 K'],
      [1024 * 9.9, '9.9 K'],
      [1024 * 9.96, '10 K'],
      [1024 * 999, '999 K'],
      [1024 * 999.6, '1.0 M'],
      [1024 ** 2 * 512, '512 M'],
      [1024 ** 3 * 2.5, '2.5 G'],
      [1024 ** 4 * 3, '3.0 T'],
      [-5, '0 B'],
      [Number.NaN, '0 B'],
    ];
    for (const [n, want] of cases) expect(formatBytesCompact(n)).toBe(want);
    for (let n = 1; n < 1024 ** 5; n = Math.ceil(n * 1.37)) expect(formatBytesCompact(n).length).toBeLessThanOrEqual(5);
  });
});

describe('useTraffic (D160)', () => {
  it('바뀐 세션만 새 객체로 바꾸고, 같은 값이면 상태를 그대로 둔다', () => {
    useTraffic.getState().apply({ a: { in: 1, out: 2 }, b: { in: 3, out: 4 } });
    const first = useTraffic.getState().bySession;
    useTraffic.getState().apply({ a: { in: 10, out: 2 } });
    const second = useTraffic.getState().bySession;
    expect(second.a).toEqual({ in: 10, out: 2 });
    expect(second.b).toBe(first.b);
    useTraffic.getState().apply({ a: { in: 10, out: 2 } });
    expect(useTraffic.getState().bySession).toBe(second);
    useTraffic.getState().remove('a');
    expect(useTraffic.getState().bySession.a).toBeUndefined();
    useTraffic.getState().apply(null);
    expect(useTraffic.getState().bySession.b).toEqual({ in: 3, out: 4 });
  });
});

describe('PaneTraffic (D160)', () => {
  it('값이 없으면 0, 이벤트가 오면 ↑보냄 ↓받음 으로 바뀐다. 툴팁은 정확한 단위', () => {
    render(<PaneTraffic sessionId="s1" live />);
    const el = screen.getByTestId('pane-traffic');
    expect(el.textContent).toBe('↑0 B↓0 B');
    expect([...el.querySelectorAll('.v')].map((x) => x.textContent)).toEqual(['↑0 B', '↓0 B']);
    act(() => useTraffic.getState().apply({ s1: { in: 2048, out: 12 } }));
    expect(el.textContent).toBe('↑12 B↓2.0 K');
    expect(el.getAttribute('title')).toBe('전송량 (세션을 연 뒤) — 보냄(입력) 12 B · 받음(출력) 2.0 KB');
    expect(el.className).toBe('pane-traffic');
    // 숫자 칸은 고정 폭 클래스(제목줄이 흔들리지 않게)
    expect(el.querySelectorAll('.v')).toHaveLength(2);
  });

  it('연결이 끊긴 세션은 흐리게(idle) 마지막 값을 둔다', () => {
    useTraffic.getState().apply({ s2: { in: 5, out: 1 } });
    render(<PaneTraffic sessionId="s2" live={false} />);
    const el = screen.getByTestId('pane-traffic');
    expect(el.className).toContain('idle');
    expect(el.textContent).toBe('↑1 B↓5 B');
  });
});
```

## 5. 문서·도움말·매뉴얼 (적용할 내용 전체)

이 계획서와 같은 이름의 폴더 [2026-11-27-20-session-traffic-title/](./2026-11-27-20-session-traffic-title/) 에 첨부가 있다. md 에는 그림·PDF 를 담을 수 없어서다.

### 5.0 첨부 파일 옮기기

| 첨부 | 옮길 곳 | 비고 |
|---|---|---|
| `img/ko/screen.png`, `img/en/screen.png` | `locales/help/img/ko/screen.png`, `locales/help/img/en/screen.png` (덮어쓰기) | 도움말 "화면 둘러보기" 그림. 창 머리글에 전송량이 보이는 새 화면. 도움말 그림 24장 가운데 창 머리글이 나오는 것은 이 2장뿐이라 나머지는 바뀌지 않았다. 직접 찍으려면 frontend 에서 `npm run help:capture -- --only screen` |
| `manual/DariTerm-user-manual.{ko,en}.{html,pdf}` | `docs/manual/` (덮어쓰기) | 아래 §5.7 의 시나리오 표를 고친 뒤 `node tools/manual/build.mjs ko`, `… en` 으로 만든 것. 직접 만들어도 같다(이 PC 의 Edge 로 PDF 를 찍는다) |

-18 계획서 폴더(`2026-11-27-18-help-sort-faq-images/img/`)의 `screen.png` 도 이 그림으로 바뀌어 있다(CLAUDE.md 의 "화면을 바꾸면 다시 찍어 그 폴더에 넣는다" 규칙). -18 을 이미 적용했으면 위 표대로 이 폴더의 2장만 옮기면 된다.

### 5.1 `locales/help/ko.md` — 화면 둘러보기

```diff
-- **창 머리글**: 서버 이름과 접속 대상. 포트 포워딩이 있으면 `⇄ N` 버튼이 보입니다.
+- **창 머리글**: 서버 이름과 접속 대상. 포트 포워딩이 있으면 `⇄ N` 버튼이 보입니다. 오른쪽 버튼들 앞의 `↑1.2 K ↓34 M` 은 이 세션의 **전송량**(↑ 보낸 것 = 입력, ↓ 받은 것 = 출력, 세션을 연 뒤 누적)이고 1초마다 바뀝니다. 창이 좁으면 숨겨집니다.
```

자주 묻는 질문 — `Q: 화면 보호기만 떠도 앱이 잠겨요.` 바로 앞에 추가

```diff
+Q: 이 세션에서 얼마나 주고받았는지 보고 싶어요.
+창 머리글 오른쪽의 `↑ … ↓ …` 가 세션을 연 뒤 보낸 양(입력)과 받은 양(출력)입니다. 마우스를 올리면 정확한 값이 보입니다. 재접속해도 이어서 세고, SFTP·포트 포워딩은 따로 셉니다. 창이 좁으면(머리글 360px 이하) 보이지 않으니 창을 넓히거나 한 창만 크게(Ctrl+Shift+Enter) 보세요.
+
 Q: 화면 보호기만 떠도 앱이 잠겨요.
```

### 5.2 `locales/help/en.md`

```diff
-- **Pane header**: server name and target. If port forwarding is set, a `⇄ N` button appears.
+- **Pane header**: server name and target. If port forwarding is set, a `⇄ N` button appears. The `↑1.2 K ↓34 M` before the buttons on the right is this session's **traffic** (↑ sent = input, ↓ received = output, since the session opened), updated every second. It is hidden when the pane is narrow.
```

FAQ — `Q: The app locks while the screen saver is on.` 바로 앞에 추가

```diff
+Q: How much did this session send and receive?
+The `↑ … ↓ …` on the right of the pane header shows bytes sent (input) and received (output) since the session opened. Hover it for exact values. It keeps counting across reconnects; SFTP and port forwards are counted separately. It is hidden in narrow panes (header 360px or less) — widen the pane or maximize it (Ctrl+Shift+Enter).
+
 Q: The app locks while the screen saver is on.
```

### 5.3 `docs/manual/user-guide.ko.md`

§3 화면 둘러보기 — `- **오른쪽 위 아이콘**` 줄 다음에 추가

```diff
 - **오른쪽 위 아이콘**: SFTP(파일) 창.
+- **창 머리글**: 서버 이름과 접속 대상, 포트 포워딩이 있으면 `⇄ 개수` 버튼. 오른쪽 버튼들 앞의 `↑1.2 K ↓34 M` 은 그 세션의 **전송량**입니다 — ↑ 보낸 것(입력), ↓ 받은 것(출력), 세션을 연 뒤 누적, 1초마다 바뀝니다. 마우스를 올리면 정확한 값이 보이고, 창이 좁으면 숨겨집니다.
 - **상태 표시줄 오른쪽 아래 "잠금 해제됨"** 을 누르면 바로 잠깁니다.
```

§14 자주 묻는 질문 — `**Q. PPK 키가 안 읽혀요.**` 바로 앞에 추가

```diff
+**Q. 이 세션에서 얼마나 주고받았는지 보고 싶어요.**
+창 머리글 오른쪽의 `↑ … ↓ …` 가 세션을 연 뒤 보낸 양(입력)과 받은 양(출력)입니다. 재접속해도 이어서 세고, SFTP·포트 포워딩은 따로 셉니다. 창이 좁아 안 보이면 창을 넓히거나 Ctrl+Shift+Enter 로 그 창만 크게 보세요.
+
 **Q. PPK 키가 안 읽혀요.**
```

### 5.4 `docs/manual/user-guide.en.md`

§3 — `- **Top-right icon**` 줄 다음에 추가

```diff
 - **Top-right icon**: SFTP (file) panel.
+- **Pane header**: server name and target, plus a `⇄ N` button when port forwarding is set. The `↑1.2 K ↓34 M` before the buttons on the right is the session's **traffic** — ↑ sent (input), ↓ received (output), since the session opened, updated every second. Hover it for exact values; it is hidden when the pane is narrow.
 - Click **"Unlocked"** at the bottom right to lock the app immediately.
```

FAQ — `**Q. My PPK key won't load.**` 바로 앞(포트 포워딩 문답 다음)에 추가

```diff
+**Q. How much did this session send and receive?**
+The `↑ … ↓ …` on the right of the pane header shows bytes sent (input) and received (output) since the session opened, updated every second; hover it for exact values. It keeps counting across reconnects; SFTP and port forwards are counted separately. If the pane is too narrow to show it, widen the pane or maximize it with Ctrl+Shift+Enter.
+
 **Q. My PPK key won't load.**
```

### 5.5 `docs/scenarios.md` — 7절 표 끝(7-10 다음)

```diff
 | 7-10 | 열린 서버를 다시 열기 | 사이드바에서 이미 열린 서버 | 그 창으로 이동하거나 새로 연다(옆에 열기/새로 열기 버튼). |
+| 7-11 | 세션 전송량 보기 | 창 머리글 오른쪽 `↑ … ↓ …` (마우스를 올리면 정확한 값) | 세션을 연 뒤 보낸 양(입력)·받은 양(출력), 1초마다 갱신, 재접속해도 이어서 센다. 머리글이 360px 이하로 좁으면 숨는다. |
 
```

### 5.6 `docs/scenarios.en.md` — 7절 표 끝(7-10 다음)

```diff
 | 7-10 | Reopen an open server | Pick an already-open server in the sidebar | Jump to that pane or open another (Open beside / Open new buttons). |
+| 7-11 | Session traffic | `↑ … ↓ …` on the right of the pane header (hover for exact values) | Bytes sent (input) and received (output) since the session opened, updated every second, kept across reconnects. Hidden when the header is 360 px wide or less. |
 
```

### 5.7 매뉴얼 HTML·PDF

§5.5·§5.6 을 고친 뒤 저장소 루트에서 `node tools/manual/build.mjs ko` 와 `node tools/manual/build.mjs en` — 또는 §5.0 의 첨부를 옮긴다.

### 5.8 `spec/ipc.md` §4 이벤트 표 — `ui.request` 행 바로 앞

```diff
 | `session.state` | 세션 정보. **`state` 값은 정확히 `connecting` · `connected` · `reconnecting` · `closed`(끊김) · `error`(실패)** — behavior/session.md §1 | 상태 변경(접속 중·연결됨·재접속·끊김·오류) |
+| `session.traffic` | (2026-11-27, D160) `{세션 id: {in, out}}` — 값이 바뀐 세션만. `in` 은 세션을 연 뒤 연결에서 받은 바이트, `out` 은 보낸 바이트(정수, 누적, 재접속해도 이어서) | **1초마다**, 바뀐 세션이 있을 때만(behavior/session.md §8) |
 | `ui.request` | 요청 객체 | §3 |
```

### 5.9 `spec/behavior/session.md` §8 — `- **숨은 세션의 출력**` 바로 앞

```diff
 - **출력 흐름 제어**: 프런트가 아직 처리(`session.ack`)하지 않은 출력이 512KB 를 넘으면 백엔드는 그 세션의 읽기를 멈추고, 128KB 아래로 내려오면 다시 읽는다(원격·로컬 프로세스는 전송 버퍼가 차서 느려진다. 출력은 버리지 않는다). 멈춘 동안 ack 가 3초 동안 오지 않으면 읽기를 다시 연다. 세션을 닫은 뒤에 읽힌 출력과 늦게 끝난 접속 시도의 안내는 보내지 않는다.
+- **전송량**(2026-11-27, D160): 세션마다 연결에서 읽은 바이트(받음)와 연결에 쓴 바이트(보냄)를 센다 — 화면 출력·ZMODEM 받기, 입력·로그인 스크립트·자동 실행 명령·ZMODEM 올리기·중단 바이트열 모두. 세션을 연 뒤 누적이고 재접속해도 이어서 센다. 백엔드는 **1초마다** 지난 보고 뒤 값이 바뀐 세션만 모아 `session.traffic` 이벤트 하나로 보낸다(바뀐 세션이 없으면 보내지 않는다). 화면은 ui/panes.md §2.7.
 - **숨은 세션의 출력**: 보이지 않는 탭의 세션 출력은 약 100ms 단위로 몰아서 터미널에 쓴다. 보이게 되는 순간 밀린 것을 먼저 쓴다.
```

### 5.10 `spec/ui/panes.md`

§2.1 — `- **포트 포워딩 버튼**` 줄 다음에 추가

```diff
 - **포트 포워딩 버튼**(2026-11-27, D137): 접속 대상 문자열 바로 오른쪽 — §2.6.
+- **전송량**(2026-11-27, D160): 가운데 빈 공간 뒤, 오른쪽 버튼들 바로 왼쪽 — §2.7.
 
```

§2.6 과 §2.5 사이에 새 절

```markdown
### 2.7 세션 전송량 (2026-11-27, D160)

원본 앱에 없던 기능이다(사용자 요청: 열린 세션별로 제목줄에 입력·출력 전송량, "제목줄이 이상하지 않도록").

- **언제 보이나**: 칸에 세션이 있으면 늘(상태와 무관). 연결됨이 아니면 마지막 값을 흐리게(불투명도 55%) 둔다. 빈 칸에는 없다.
- **값**: 세션을 연 뒤 연결로 **보낸**(↑, 키 입력·붙여넣기·명령 모음·로그인 스크립트·ZMODEM 올리기) 바이트와 **받은**(↓, 서버 출력·ZMODEM 받기) 바이트 누적. 재접속해도 이어서 센다. SSH 는 채널 데이터 기준(암호화·프로토콜 머리 제외)이고 SFTP·포트 포워딩은 들지 않는다(포워딩은 §2.6 의 정보 창). 값은 `session.traffic` 이벤트로 **1초마다 바뀐 세션만** 온다(spec/ipc.md §4, behavior/session.md §8).
- **모양**: `↑1.2 K ↓34 M` — 글자 10.5px 고정폭 글꼴, 흐린 글자색(`--fg-3`, 포커스·호버 칸은 `--fg-2`). 바이트 표기는 **늘 5글자 이하**: 1000 미만 `n B`, 그 위는 1024 단위로 나눠 10 미만이면 소수 한 자리(`9.9 K`), 아니면 정수(`512 M`), 정수로 1000 이 되면 다음 단위(`1.0 M`). 단위 K·M·G·T·P.
- **제목줄이 흔들리지 않게**: `↑값`·`↓값` 두 칸은 **고정 폭 6글자(화살표 + 5글자), 왼쪽 맞춤, 고정폭 숫자**, 칸 사이 6px 라 값이 바뀌어도 폭이 그대로다(남는 폭은 칸 뒤로 가서 화살표와 숫자가 붙어 있다). 환경 태그도 줄어들거나 꺾이지 않는다. 요소는 줄어들지 않고, 공간이 모자라면 제목·접속 대상이 먼저 말줄임으로 준다. **머리글 폭이 360px 이하이면 통째로 숨긴다**(좁은 분할 칸).
- 툴팁(접근성 이름도 같음): `전송량 (세션을 연 뒤) — 보냄(입력) {보낸 양} · 받음(출력) {받은 양}` — 여기는 기존 바이트 표기(dialogs.md §1.5, `12.3 KB`).
```

### 5.11 `docs/decisions.md` — 표 끝에 추가

```markdown
| D160 | 2026-11-27 | **창 머리글의 세션 전송량**: 세션마다 연결에서 읽은(받음 ↓)·쓴(보냄 ↑) 바이트를 atomic 으로 세고(화면 출력·ZMODEM, 입력·로그인 스크립트·자동 실행 명령·ZMODEM 올리기·중단 바이트열), 세션을 연 뒤 누적·재접속해도 이어서. 백엔드가 **1초마다 바뀐 세션만** `session.traffic` `{id:{in,out}}` 이벤트 하나로 보낸다. 프런트는 작업 영역 저장소와 따로 둔 `useTraffic`(바뀐 세션 항목만 새 객체)에 넣고 머리글의 `PaneTraffic` 이 자기 세션만 구독. 표시 `↑1.2 K ↓34 M` — 5글자 이하 짧은 표기, 화살표+숫자 칸 고정 6ch·왼쪽 맞춤·고정폭 숫자, `flex: none`, 머리글 360px 이하면 컨테이너 질의로 숨김, 끊긴 세션은 흐리게 마지막 값. 함께 좁은 칸에서 환경 태그가 두 줄로 꺾이던 것도 고침(`flex: none; nowrap`) | docs/import-plans/2026-11-27-20-session-traffic-title.md, spec/ipc.md §4, spec/behavior/session.md §8, spec/ui/panes.md §2.7. 사용자 질문("제목줄에 입력 출력 전송량 표시는 성능에 영향?") → 권장안(atomic + 1초 묶음) 수락, "제목줄이 이상하지 않도록". 기각: 출력마다 이벤트(세션 수 × 출력 빈도만큼 재렌더), `session.state` 에 싣기(세션 목록 전체 재렌더·상태 이벤트 의미 흐림), 프런트에서 `session.data` 글자 수 세기(디코딩 뒤 UTF-16 글자라 바이트가 아니고 입력·ZMODEM 을 못 셈), Transport 를 감싸 세기(SFTP·포워딩이 `Transport()` 형 단언으로 기능을 찾아 깨짐), 재접속마다 0 부터(자동 재접속 때 값이 사라져 헷갈림), 오른쪽 맞춤 숫자(`↓  5 B` 처럼 화살표와 숫자가 떨어짐 — 캡처로 확인), 켜고 끄는 설정(요청 없음, 필요하면 후속) |
```

### 5.12 `CHANGELOG.md` — "미배포" 절

"새 기능" — `포트 포워딩 전송량` 줄 다음:

```markdown
- **세션 전송량**: 창 머리글 오른쪽 버튼들 앞에 그 세션이 보낸 양(↑ 입력)과 받은 양(↓ 출력)을 `↑1.2 K ↓34 M` 처럼 보인다. 세션을 연 뒤 누적, 1초마다 갱신, 마우스를 올리면 정확한 값. 폭이 고정이라 값이 바뀌어도 머리글이 흔들리지 않고, 좁은 창(머리글 360px 이하)에서는 숨는다. (D160)
```

"수정" — 맨 앞:

```markdown
- **좁은 창에서 머리글의 환경 태그("운영" 등)가 두 줄로 꺾이던 문제**: 태그는 줄지 않고 제목·접속 대상이 먼저 말줄임으로 준다. (D160)
```

### 5.13 `CLAUDE.md` 문서 인덱스 — import-plans 행, `…Esc 는 맨 위 겹만 D159.` 다음

```markdown
이어서 [-20-](./docs/import-plans/2026-11-27-20-session-traffic-title.md): **창 머리글에 세션별 전송량**(`↑보냄 ↓받음`, 세션을 연 뒤 누적, 백엔드가 1초마다 바뀐 세션만 `session.traffic` 으로, 폭 고정·머리글 360px 이하면 숨김, 환경 태그 꺾임도 고침) D160 — 도움말·매뉴얼(설명서·시나리오·PDF)도 함께 고침.
```


## 6. 검증 결과 (2026-11-27)

| 항목 | 결과 |
|---|---|
| `go vet ./...`, `go test -race ./...` | 통과(19개 패키지) |
| `npx tsc --noEmit`, vitest | 통과(67개 파일, 504개) |
| `wails build` (v2.17.1, `go run`) | `build/bin/dariterm.exe` |
| 화면 캡처(vite + 가짜 백엔드 + Edge, 1280px) | 창 1개·2개(머리글 약 500px): `↑9.9 K ↓987 M` / `↑999 B ↓5 B` 모두 **폭 80px 그대로**(값을 바꾸기 전후 같음). 창 3개(머리글 333px): 전송량 숨김, 환경 태그 한 줄. 처음 안(오른쪽 맞춤)은 `↓  5 B` 처럼 벌어져 왼쪽 맞춤으로 바꿈 |

## H. 수동 검증 (앱)

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | SSH 서버 연결 뒤 `ls -al` | 머리글 오른쪽에 `↑… ↓…`, 1초 안에 ↓ 가 늘어난다 | |
| 2 | `cat` 으로 큰 파일 출력 | 숫자만 바뀌고 머리글의 제목·버튼 위치가 움직이지 않는다. 터미널이 느려지지 않는다 | |
| 3 | 마우스 올리기 | 툴팁 `전송량 (세션을 연 뒤) — 보냄(입력) … · 받음(출력) …` | |
| 4 | 창을 3개 이상으로 나눠 좁게 | 전송량이 숨고 환경 태그는 한 줄 | |
| 5 | 서버 쪽에서 끊기 → 자동 재접속 | 끊긴 동안 흐리게 마지막 값, 재접속 뒤 이어서 늘어난다 | |
| 6 | ZMODEM `sz` 로 파일 받기 | ↓ 가 파일 크기만큼 는다 | |
| 7 | 탭 여러 개에서 세션 10개 이상 | 앱 반응이 전과 같다 | |

## 체크리스트

- [x] 첨부(그림·매뉴얼) · [x] 백엔드 카운터·주기 보고 · [x] 프런트 저장소·머리글 · [x] 문구 10개 언어 · [x] 도움말 · [x] 매뉴얼(설명서·시나리오·PDF) · [x] 명세 · [x] 테스트 · [x] 빌드 · [ ] §H
