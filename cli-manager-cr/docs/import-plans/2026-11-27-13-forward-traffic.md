# 포트 포워딩 전송량

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D151). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-1-forward-info.md](./2026-11-27-1-forward-info.md) §4(D142 — 연결 수, "전송량 표시(보류)")

---

## 0. 한 줄 요약

포워딩 정보 창에 포워딩마다 **보낸·받은 바이트**(접속한 뒤 누적)와 그 합계를 보인다. 연결 수를 세려고 이미 감싸 두었던 연결(`countedConn`)이 Read·Write 에서 바이트도 함께 센다. 중계 코드는 고치지 않았다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 무엇을 세나 | 포워딩이 **받아들인 쪽** 연결의 읽기·쓰기 바이트 | 듣는 소켓을 이미 감싸고 있다(D142). 대상 쪽 연결은 L 은 SSH 채널, R 은 이 PC 의 TCP 라 종류마다 감싸는 곳이 달라진다 |
| 방향 이름 | `up` = 받아들인 쪽 프로그램이 **보낸** 것(그 연결에서 읽은 것), `down` = 그 프로그램이 **받은** 것(그 연결에 쓴 것). 화면은 `↑ … · ↓ …` | L 이면 이 PC 의 프로그램 기준 업로드·다운로드와 같다 |
| 범위 | 접속한 뒤부터 누적, 연결이 닫혀도 줄지 않는다. 재접속하면 새로 센다 | 연결 수는 "지금", 전송량은 "지금까지" |
| D(동적) | SOCKS 협상 몇 바이트도 함께 센다 | 따로 빼려면 SOCKS 코드에 손을 대야 한다. 차이가 작다 |
| 언제 보이나 | 창을 열 때·[새로고침] 때 읽는다(연결 수와 같이) | D142 의 "주기적으로 묻지 않는다" |
| 초당 속도 | 하지 않는다 | 주기적으로 물어야 한다 |

## 2. 구현

### 2.1 `internal/sshconn/forward_count.go` (전체)

```go
// forwardStats 는 포워딩 하나의 집계다: 지금 열려 있는 연결 수(D142)와, 접속한 뒤 지금까지 오간 바이트 수(D151).
// 바이트는 이 포워딩이 받아들인 쪽(L·D 는 이 PC 의 프로그램, R 은 서버 쪽 프로그램) 연결 기준이다:
//   - Up   = 그 연결에서 읽은 바이트 (프로그램 → 대상으로 보낸 것)
//   - Down = 그 연결에 쓴 바이트   (대상 → 프로그램으로 받은 것)
//
// 동적(D) 포워딩은 SOCKS 협상 몇 바이트도 함께 센다.
type forwardStats struct {
	active atomic.Int64
	up     atomic.Int64
	down   atomic.Int64
}

type countingListener struct {
	net.Listener
	pStats *forwardStats
}

func (pListener *countingListener) Accept() (net.Conn, error) {
	conn, errAccept := pListener.Listener.Accept()
	if errAccept != nil {
		return nil, errAccept
	}
	pListener.pStats.active.Add(1)
	return &countedConn{Conn: conn, pStats: pListener.pStats}, nil
}

type countedConn struct {
	net.Conn
	pStats    *forwardStats
	onceClose sync.Once
}

func (pConn *countedConn) Read(bBuf []byte) (int, error) {
	nRead, errRead := pConn.Conn.Read(bBuf)
	if nRead > 0 {
		pConn.pStats.up.Add(int64(nRead))
	}
	return nRead, errRead
}

func (pConn *countedConn) Write(bBuf []byte) (int, error) {
	nWritten, errWrite := pConn.Conn.Write(bBuf)
	if nWritten > 0 {
		pConn.pStats.down.Add(int64(nWritten))
	}
	return nWritten, errWrite
}

func (pConn *countedConn) Close() error {
	pConn.onceClose.Do(func() { pConn.pStats.active.Add(-1) })
	return pConn.Conn.Close()
}
```

`io.Copy` 가 `countedConn` 의 `ReadFrom`/`WriteTo` 지름길을 쓸 수 없게 되지 않나? — `countedConn` 은 `net.Conn` 을 품기만 하고 `ReadFrom`·`WriteTo` 를 드러내지 않는다(품은 필드의 메서드 집합은 `net.Conn` 인터페이스의 것뿐). 그래서 `io.Copy` 는 늘 우리의 `Read`·`Write` 를 거친다. 바이트 수가 빠지지 않는다(시험으로 확인).

### 2.2 `internal/sshconn/keepalive_forward.go`

```go
type ForwardResult struct {
	Type string; Bind string; Dest string; Err error
	// pStats 는 이 포워딩의 연결 수·전송량 집계다 (실패한 항목은 nil).
	pStats *forwardStats
}

func (result ForwardResult) Active() int { … pStats.active … }

// Traffic 은 접속한 뒤 이 포워딩으로 오간 바이트 수다 (D151). Up 은 받아들인 쪽 프로그램이 보낸 것, Down 은 받은 것.
func (result ForwardResult) Traffic() (i64Up int64, i64Down int64) {
	if result.pStats == nil {
		return 0, 0
	}
	return result.pStats.up.Load(), result.pStats.down.Load()
}

func counted(ln net.Listener) (net.Listener, *forwardStats) { … }
```

### 2.3 세션 정보 — `internal/session/types.go`, `internal/app/ssh_app.go`

```go
type ForwardInfo struct {
	…
	M_iConns  int    `json:"conns"`
	M_i64Up   int64  `json:"up"`   // 접속한 뒤 받아들인 쪽 프로그램이 보낸 바이트 (D151)
	M_i64Down int64  `json:"down"` // 접속한 뒤 받아들인 쪽 프로그램이 받은 바이트 (D151)
}

// forwardInfos
		info.M_i64Up, info.M_i64Down = result.Traffic()
```

### 2.4 화면 — `frontend/src/screens/ForwardsDialog.tsx`, `lib/forwards.ts`

- 표 머리에 **"전송량"** 열: 작은 고정폭 `↑ 1.2 MB · ↓ 34.5 KB`(바이트 표기는 기존 `formatBytes`). 실패한 줄은 비운다.
- 첫 줄 요약 끝에 합계 ` · ↑ … · ↓ …`(오간 것이 있을 때만).

```ts
/** 목록 전체의 전송량 합(D151). 값이 없는 항목(옛 백엔드·실패)은 0 으로 본다. */
export function trafficTotal(list: readonly ForwardInfo[] | undefined): { up: number; down: number } { … }
```

`ipc/types.ts` `ForwardInfo.up?`·`down?`.

### 2.5 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `fwdinfo.colTraffic` | 전송량 |
| `fwdinfo.traffic` | ↑ {up} · ↓ {down} |

## 3. 명세

| 문서 | 바꾼 것 |
|---|---|
| spec/protocols.md §2.5 | "전송량" 항목 |
| spec/ipc.md `session.forwards` | `up`·`down` |
| spec/ui/dialogs.md §7 | 요약·열 |

## 4. 테스트 — `internal/sshconn/forward_count_test.go`

| 테스트 | 확인 |
|---|---|
| TestForwardCountsOpenConnections | 기존 연결 수 시험에 더해, 두 연결로 `ping`(4)·`hello!`(6)을 왕복하면 ↑10 ↓10, 연결을 닫아도 10·10 그대로 |
| TestForwardCountDropsFailedDial | 대상에 닿지 못한 연결은 오간 것이 0 |
| TestForwardResultActiveWithoutCounter | 실패한 포워딩은 연결 수·전송량 0 |

`go test -race ./internal/sshconn/` 통과(카운터는 atomic).

## 5. 검증 결과 (2026-11-27)

전체 `go test -race`, tsc, vitest 465개, `wails build` 성공.

## 6. 성능

Read·Write 마다 atomic 더하기 한 번. 중계 버퍼(32KB)마다 한 번이라 부담이 없다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | L 포워딩(예: 로컬 8080 → 서버의 웹) 연결 뒤 브라우저로 페이지 몇 번 | 정보 창 전송량 ↑ 작고 ↓ 큼 | |
| 2 | [새로고침] | 값이 늘어 있다 | |
| 3 | 브라우저를 닫아 연결 수 0 | 전송량은 그대로 | |
| 4 | 재접속 | 0 부터 다시 | |
| 5 | D(SOCKS) 포워딩으로 브라우저 | 값이 쌓인다 | |

## 체크리스트

- [x] 카운터 · [x] 세션 정보 · [x] 화면 · [x] 문구 · [x] 명세 · [x] 테스트 · [ ] §H
