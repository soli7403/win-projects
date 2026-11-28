# 포트 포워딩 정보 표시 — 터미널 안내를 줄마다, 창 머리글 버튼과 정보 창

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D136·D137, 같은 날 상태 문구와 연결 수 D142 — §4, 머리글 버튼의 테두리 D143 — §5). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 포트 포워딩 D109(동적). 명세 [spec/protocols.md](../../spec/protocols.md) §2.5, [spec/behavior/session.md](../../spec/behavior/session.md) §1, [spec/ui/panes.md](../../spec/ui/panes.md) §2.6, [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §7
**적용 순서**: 2026-11-27 의 1번째. 2~5번과 `locales/*.json`·`frontend/src/ipc/types.ts`·`frontend/src/styles/app.css` 가 겹치므로 번호 순서대로 적용한다.

---

## 0. 한 줄 요약

포워딩이 많은 서버에 연결하면 안내가 **한 줄로 붙어** 나와 읽을 수 없었다. 이제 **포워딩마다 한 줄씩 들여 써서** 나오고, 창 머리글에 **포워딩 버튼**(개수)이 생겨 누르면 **표로 된 정보 창**이 뜬다.

| | 고치기 전 | 고친 뒤 |
|---|---|---|
| 터미널 안내 | `포워딩 127.0.0.1:25095 → 100.102.205.95:22포워딩 127.0.0.1:37443 → …` (줄바꿈 없이 이어짐, 색 없음) | 첫 줄 `● 포트 포워딩 22개`, 그 아래 포워딩마다 한 줄(빈칸 4개 들여쓰기, 화살표가 세로로 맞음) |
| 실패한 포워딩 | 같은 줄에 섞여 있어 눈에 띄지 않음 | 첫 줄에 실패 수(노랑), 그 줄은 빨강 `✕  실패: …` |
| 설정됐는지 확인 | 화면을 위로 올려 안내 줄을 찾아야 함 | 창 머리글의 `⇄ 22` 버튼(테두리가 강조색, D143). 누르면 종류·듣는 주소·대상·상태 표 |
| 쓰이고 있는지 확인 | 알 수 없음 | 상태 칸에 `동작 중`, 그 포워딩으로 이어져 있는 연결이 있으면 `동작 중 · 연결 N개`(창을 열 때 읽음, [새로고침]으로 다시 읽음, D142) |

고친 뒤의 터미널 모양:

```
● 포트 포워딩 3개
    L  127.0.0.1:25095  →  100.102.205.95:22
    L  127.0.0.1:80     →  10.0.0.1:8080
    D  127.0.0.1:1080   →  SOCKS 프록시
```

## 1. 사용자 요청

2026-11-27(화면 사진과 함께): "포트포워딩 정보를 인덴테이션 해서 보기 편하게 보여줘." / "포트포워딩과 같은 설정이 되었는지를 나타내는 메뉴나 버튼이 있어 누르면 모달 팝업 형태로 포트포워딩 정보를 보여준다"(사진에서 창 머리글의 접속 대상 오른쪽을 가리킴).

## 2. 원인 (안내가 한 줄로 붙던 까닭)

- 접속 함수가 받는 `notify` 는 **받은 글자를 그대로** 터미널에 쓴다(`Session.writeOutput`). 세션이 직접 내는 안내(`● …` + 색 + CRLF)와 달리 꾸며 주지 않는다.
- `sshconn.StartForwards` 는 포워딩마다 `notice("포워딩 A → B")` 를 불렀다 — 색도 줄바꿈도 없는 글자라 다음 안내가 바로 뒤에 붙었다.
- 접속이 끝나면 세션이 "접속 중" 줄을 지우려고 `\r` + 줄 지우기를 보내는데, 붙어 나온 안내의 **마지막 화면 줄**이 이때 지워졌다(사진에서 끝이 잘려 보이는 까닭).

## 3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 안내를 만드는 곳 | `sshconn` 은 결과만 돌려주고(`[]ForwardResult`), 글은 `session.MsgForwards` 가 한 번에 만든다 | 줄 맞춤(가장 긴 주소에 폭 맞추기)은 전체를 알아야 한다. 같은 결과를 화면에도 쓴다 |
| 줄 모양 | 첫 줄 `● 포트 포워딩 N개`, 항목은 빈칸 4개 + `종류  듣는 주소  →  대상` | 사용자 요청(들여쓰기). 다른 안내 줄과 같은 `●` 머리 |
| 화면에 알리는 길 | 세션 정보에 `forwards` 필드를 더한다(`session.state` 에 실려 간다) | 새 채널·이벤트가 필요 없다. 재접속하면 그때 결과로 바뀐다 |
| 세션이 결과를 얻는 길 | 연결(`Transport`)이 `ForwardReporter` 를 구현하면 접속 성공 때 읽는다 | 접속 함수(`Dialer`)의 모양을 바꾸지 않는다(테스트 4곳이 그 모양을 쓴다) |
| 버튼 위치·모양 | 머리글의 접속 대상 오른쪽, 알약 모양 `⇄ N`. 실패가 있으면 경고색 | 사진에서 가리킨 자리. 오른쪽 아이콘 묶음(동작 버튼)과 구분된다 |
| 언제 보이나 | 연결됨이고 포워딩이 하나 이상일 때만 | 끊긴 동안에는 듣는 소켓이 닫혀 있다 |
| 정보 창 | 읽기 전용 표 + [복사] | 고치는 곳은 서버 편집 > 포트 포워딩 하나로 둔다(도움말에 적음) |

고르지 않은 안: 모든 접속 안내(경유 서버·프록시·키 오류 등)를 한꺼번에 줄 단위로 바꾸기. 그 안내들은 한 번에 하나만 나와 붙는 문제가 드러나지 않고, 일부는 접속이 끝나면 지워지는 것을 전제로 쓰였다. 이번에는 요청 받은 포워딩만 고쳤다.

### 3.1 영향 범위

- **IPC**: 세션 정보에 선택 필드 `forwards` 가 늘었다(없으면 싣지 않는다). D142 에서 항목에 `conns` 를 더하고 채널 `session.forwards`(id → 지금 목록)를 새로 두었다. 다른 채널은 그대로.
- **문구**: `be.fwd.active`·`be.fwd.dynamicActive`·`be.fwd.localFailed`·`be.fwd.remoteFailed`·`be.fwd.dynamicFailed` 를 지우고 `be.fwd.header`·`be.fwd.headerFailed`·`be.fwd.socks`·`be.fwd.failed` 와 `fwdinfo.*` 18개를 넣었다(10개 언어). D142 에서 `fwdinfo.listening` 의 값을 "대기 중" → "동작 중"으로 바꾸고 `fwdinfo.conns`("연결 {count}개")를 더했다(모두 19개). [새로고침] 버튼은 있던 키 `sftp.refresh` 를 쓴다.
- **세션 로그**: 안내가 여러 줄이 되므로 로그에도 줄마다 남는다.

## 4. 추가: 상태 문구 "동작 중"과 연결 수 (2026-11-27, D142)

사용자 질문("포트포워딩 상태가 왜 대기중인거지?")과 지시("동작중으로 해줘. 실제 사용 여부까지 보여 줄 수 있는 것이 부담이 되지 않는다면 개수도 보여줘").

- **문구**: 성공한 포워딩의 상태 "대기 중" → **"동작 중"**. "대기 중"은 듣는 소켓이 접속을 기다린다는 뜻이었지만 아직 준비가 안 된 것으로 읽혔다.
- **연결 수**: 그 포워딩으로 지금 이어져 있는 연결이 있으면 `동작 중 · 연결 N개`. 첫 줄에도 전체 합을 보인다.

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 세는 방법 | 듣는 소켓을 감싸(`countingListener`) 연결을 받을 때 +1, 그 연결을 닫을 때 -1 | 중계 코드(`acceptLoop`·`serveSocks`)를 고치지 않는다. L·R·D 가 같은 길로 세어진다. 드는 비용은 연결마다 원자 덧셈 두 번 |
| 세는 때 | 접속이 들어올 때와 닫힐 때만(콜백). 주기적으로 세거나 훑지 않는다 | 사용자 지시("trigger 나 callback 으로"). 연결이 없으면 비용도 없다 |
| 화면에 알리는 길 | 새 채널 `session.forwards`(id → 지금 목록). 정보 창을 **열 때 한 번** 묻고, 창의 [새로고침]을 누르면 다시 묻는다 | 사용자 지시("1초마다 갱신할 필요 없고, 열면 그때 읽어오면 된다" — 처음에는 1초마다 묻게 만들었다가 뺐다). 세션 정보(`session.state`)에 싣지 않은 것은 연결이 오갈 때마다 모든 화면이 다시 그려지기 때문이다 |
| 머리글 버튼 | 연결 수를 보이지 않는다(포워딩 개수만) | 버튼에까지 보이려면 창이 닫혀 있어도 계속 알려야 한다 |
| 전송량 | 세지 않는다 | 요청은 사용 여부였다. 바이트 수는 복사 경로에 손을 대야 한다 |

더한 파일: `internal/sshconn/forward_count.go`(새)·`forward_count_test.go`(새, 3개), `internal/sshconn/keepalive_forward.go`(`ForwardResult.Active`), `internal/session/{types,control}.go`(`conns`, `Session.Forwards`), `internal/app/{ssh_app,channels_session}.go`(채널), `frontend/src/screens/ForwardsDialog.tsx`(열 때 읽기, [새로고침]), `frontend/src/ipc/{types,fakeBackend}.ts`, `locales/*.json`(`fwdinfo.listening` 값 변경, `fwdinfo.conns` 추가), 명세 `spec/ipc.md`·`spec/protocols.md`·`spec/behavior/session.md`·`spec/ui/dialogs.md`·수용 P17a.

## 5. 추가: 머리글 버튼의 테두리 (2026-11-27, D143)

사용자 지시: "제목줄의 포트포워딩 버튼에도 테두리만 눈에 띄게 변경해줘." 버튼 테두리를 `--border-strong` 에서 **강조색(`--accent`)** 으로 바꿨다. 글자색·바탕은 그대로다. 실패가 있을 때의 경고색(테두리·글자)은 전과 같다. 바꾼 곳은 `frontend/src/styles/app.css` 의 `.pane-fwd` 한 줄과 명세 `spec/ui/panes.md` §2.6.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 내용 |
|---|---|---|
| `internal/sshconn/keepalive_forward.go` | 코드 | `StartForwards` 가 `([]io.Closer, []ForwardResult)` 를 돌려준다. 안내를 직접 내지 않는다 |
| `internal/session/types.go` | 코드 | `Info.M_slForwards`, `ForwardInfo`, `ForwardReporter` |
| `internal/session/messages.go` | 코드 | `MsgForwards` — 여러 줄 안내 |
| `internal/session/session.go` | 코드 | 접속 성공 때 `ForwardReporter` 로 `forwards` 를 채운다 |
| `internal/app/ssh_app.go` | 코드 | `sshSession.Forwards`, `forwardInfos`, 안내를 한 번에 `notify` |
| `frontend/src/ipc/types.ts` | 코드 | `SessionInfo.forwards`, `ForwardInfo` |
| `frontend/src/lib/forwards.ts` | 코드 | **새 파일** — 실패 수, 복사용 글자 |
| `frontend/src/screens/ForwardsDialog.tsx` | 코드 | **새 파일** — 정보 창. D142: 열 때 `session.forwards` 를 한 번 읽어 연결 수 표시, [새로고침] |
| `internal/sshconn/forward_count.go` | 코드 | **새 파일**(D142) — 듣는 소켓을 감싸 열려 있는 연결 수를 센다 |
| `internal/session/control.go`, `internal/app/channels_session.go` | 코드 | D142 — `Session.Forwards`, 채널 `session.forwards` |
| `frontend/src/ipc/fakeBackend.ts` | 코드 | D142 — 가짜 백엔드의 `session.forwards`(빈 목록) |
| `frontend/src/screens/Panes.tsx` | 코드 | 머리글 버튼, 우클릭 메뉴 항목 |
| `frontend/src/styles/app.css` | 스타일 | `.pane-fwd`(D143: 테두리 `--accent`), `.fwd-info` |
| `internal/session/forwards_test.go` | 테스트 | **새 파일**(4개) |
| `internal/sshconn/forward_count_test.go` | 테스트 | **새 파일**(D142, 3개) — 열릴 때 늘고 닫힐 때 줄어듦, 대상에 닿지 못한 연결, 실패한 포워딩은 0 |
| `frontend/src/lib/forwards.test.ts`, `frontend/src/screens/Automation.test.tsx` | 테스트 | **새 파일**(계산 2개, 정보 창 2개 — 표, 그리고 D142 의 "열 때 한 번 읽고 주기적으로 묻지 않는다·[새로고침]") |
| `locales/*.json` | 문구 | 위 §3.1 |
| `spec/protocols.md`, `spec/behavior/session.md`, `spec/ipc.md`, `spec/ui/panes.md`, `spec/ui/dialogs.md`, `spec/acceptance/03-panes-terminal.md` | 명세 | 수정 |

아래 `diff` 는 **오늘(2026-11-27) 작업을 시작하기 전의 파일**과 지금 파일을 견준 것이다. 이 폴더는 git 저장소가 아니라서, 오늘 적용한 수정 기록을 거꾸로 돌려 고치기 전 파일을 되살린 뒤 뽑았다(되살린 것에 수정을 다시 적용하면 지금 파일과 같아지는 것을 확인했다). 한 파일을 오늘의 여러 문서가 고친 경우, 각 문서에는 그 문서의 몫이 든 조각만 싣는다 — 조각 안의 문맥 줄에 다른 문서의 코드가 섞여 있을 수 있고, 줄 번호는 다섯 문서를 모두 적용한 최종 파일 기준이다. 새 파일은 최종 내용 전체를 싣는다(같은 날의 추가 지시 D142·D143 까지 반영된 것).

## B. 코드

### 먼저 읽을 곳

안내 만들기 — [internal/session/messages.go](../../internal/session/messages.go) `MsgForwards`:

```go
szPad := strings.Repeat(" ", iWidth-len([]rune(fw.M_szBind)))   // 가장 긴 듣는 주소에 맞춘다
szLine := "    " + fw.M_szType + "  " + fw.M_szBind + szPad
// 실패: 빨강 "…  ✕  실패: {오류}"   /   D: "…  →  SOCKS 프록시"   /   그 밖: "…  →  {대상}"
```

세션 정보 채우기 — [internal/session/session.go](../../internal/session/session.go) `connect`:

```go
pSession.info.M_slForwards = nil
if reporter, bOk := conn.(ForwardReporter); bOk {
	pSession.info.M_slForwards = reporter.Forwards()
}
```

머리글 버튼 — [frontend/src/screens/Panes.tsx](../../frontend/src/screens/Panes.tsx): `info.state === 'connected'` 일 때의 `info.forwards` 로 `.pane-fwd` 버튼을 그리고, 누르면 `ForwardsDialog` 를 연다.

### `internal/sshconn/forward_count.go` — 새 파일 (전체)

```go
package sshconn

import (
	"net"
	"sync"
	"sync/atomic"
)

// countingListener 는 받아들인 연결 중 아직 열려 있는 것의 수를 센다 (포트 포워딩의 "연결 N개" 표시, D142).
// 포워딩의 대기 소켓을 이것으로 감싸면 중계 코드(acceptLoop·serveSocks)를 고치지 않고 셀 수 있다:
// 연결을 받을 때 하나 늘리고, 그 연결을 닫을 때 하나 줄인다.
type countingListener struct {
	net.Listener
	pActive *atomic.Int64
}

// Accept 는 연결을 받아 수를 하나 늘리고, 닫힐 때 줄이는 연결로 감싸 돌려준다.
func (pListener *countingListener) Accept() (net.Conn, error) {
	conn, errAccept := pListener.Listener.Accept()
	if errAccept != nil {
		return nil, errAccept
	}
	pListener.pActive.Add(1)
	return &countedConn{Conn: conn, pActive: pListener.pActive}, nil
}

// countedConn 은 닫힐 때 수를 한 번만 줄이는 연결이다 (Close 가 여러 번 불려도 한 번).
type countedConn struct {
	net.Conn
	pActive   *atomic.Int64
	onceClose sync.Once
}

func (pConn *countedConn) Close() error {
	pConn.onceClose.Do(func() { pConn.pActive.Add(-1) })
	return pConn.Conn.Close()
}
```

### `frontend/src/lib/forwards.ts` — 새 파일 (전체)

```ts
// 포트 포워딩 표시용 계산. 근거: spec/ui/panes.md §2.6, spec/protocols.md §2.5 (D136·D137).
import type { ForwardInfo } from '../ipc/types';

/** 실패한 포워딩 수. */
export function failedCount(list: readonly ForwardInfo[] | undefined): number {
  return (list ?? []).filter((f) => !!f.error).length;
}

/**
 * 포워딩 목록을 복사용 글자로 만든다: 한 줄에 하나, 듣는 주소의 폭을 맞춘다.
 * socks 는 D 의 대상 자리에 적을 글자, failed 는 실패 줄의 까닭 앞에 붙일 글자를 만든다.
 */
export function forwardsText(list: readonly ForwardInfo[], socks: string, failed: (error: string) => string): string {
  const width = Math.max(0, ...list.map((f) => f.bind.length));
  return list
    .map((f) => {
      const head = `${f.type}  ${f.bind.padEnd(width)}`;
      if (f.error) return `${head}  ✕  ${failed(f.error)}`;
      return `${head}  →  ${f.type === 'D' ? socks : f.dest ?? ''}`;
    })
    .join('\n');
}
```

### `frontend/src/screens/ForwardsDialog.tsx` — 새 파일 (전체)

```tsx
// 포트 포워딩 정보 창: 지금 연결에 설정된 포워딩과 그 결과를 표로 보인다.
// 근거: spec/ui/panes.md §2.6, spec/ui/dialogs.md §7 (D137).
import { useCallback, useEffect, useState } from 'react';
import { ArrowLeftRight } from 'lucide-react';
import { Modal } from '../ui/Modal';
import type { ForwardInfo } from '../ipc/types';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { failedCount, forwardsText } from '../lib/forwards';
import { t, type MsgId } from '../i18n';

const KIND: Record<ForwardInfo['type'], MsgId> = { L: 'fwdinfo.kindL', R: 'fwdinfo.kindR', D: 'fwdinfo.kindD' };

export function ForwardsDialog({ sessionId, title, target, forwards: initial, onClose }: {
  /** 연결 수를 물을 세션. 없으면 묻지 않고 받은 목록만 보인다. */
  sessionId?: string;
  title: string;
  target: string;
  forwards: ForwardInfo[];
  onClose: () => void;
}) {
  // 지금 이어져 있는 연결 수는 세션 정보에 실려 오지 않는다(상태가 바뀔 때만 오므로).
  // 창을 열 때 한 번 읽는다. 주기적으로 다시 묻지 않는다 — 다시 보려면 [새로고침].
  const [live, setLive] = useState<ForwardInfo[] | null>(null);
  const load = useCallback(() => {
    if (!sessionId) return;
    void getBackend().invoke<ForwardInfo[]>('session.forwards', [sessionId])
      .then((list) => {
        if (Array.isArray(list) && list.length > 0) setLive(list);
      })
      .catch(() => undefined);
  }, [sessionId]);
  useEffect(() => load(), [load]);
  const forwards = live ?? initial;
  const failed = failedCount(forwards);
  const conns = forwards.reduce((sum, f) => sum + (f.conns ?? 0), 0);
  const copy = () => {
    const text = forwardsText(forwards, t('fwdinfo.socks'), (error) => t('fwdinfo.failed', { error }));
    void getBackend().invoke('clip.writeText', [text])
      .then(() => useApp.getState().pushToast({ kind: 'ok', text: t('fwdinfo.copied') }))
      .catch(() => undefined);
  };
  return (
    <Modal
      title={t('fwdinfo.title', { name: title })}
      icon={ArrowLeftRight}
      size="md"
      onClose={onClose}
      foot={
        <>
          {sessionId && <button className="btn" style={{ marginRight: 'auto' }} onClick={load}>{t('sftp.refresh')}</button>}
          <button className="btn" onClick={copy}>{t('common.copy')}</button>
          {/* 열리면 여기에 포커스를 둔다: 창이 떠 있는 동안 치는 키가 뒤의 터미널로 가지 않게. */}
          <button className="btn primary" autoFocus onClick={onClose}>{t('common.close')}</button>
        </>
      }
    >
      <div className="faint small" style={{ marginBottom: 10 }}>
        {target} · {failed > 0 ? t('fwdinfo.summaryFailed', { count: forwards.length, failed }) : t('fwdinfo.summary', { count: forwards.length })}
        {conns > 0 && ` · ${t('fwdinfo.conns', { count: conns })}`}
      </div>
      <table className="fwd-info">
        <thead>
          <tr>
            <th>{t('fwdinfo.colKind')}</th>
            <th>{t('fwdinfo.colBind')}</th>
            <th>{t('fwdinfo.colDest')}</th>
            <th>{t('fwdinfo.colState')}</th>
          </tr>
        </thead>
        <tbody>
          {forwards.map((f, i) => (
            <tr key={`${f.type}-${f.bind}-${i}`} className={f.error ? 'failed' : undefined}>
              <td title={t(KIND[f.type])}><b>{f.type}</b> <span className="faint">{t(KIND[f.type])}</span></td>
              <td className="mono selectable">{f.bind}</td>
              <td className="mono selectable">{f.type === 'D' ? <span className="faint">{t('fwdinfo.socks')}</span> : f.dest}</td>
              <td>{f.error ? <span className="fwd-bad" title={f.error}>{t('fwdinfo.failed', { error: f.error })}</span> : <span className="fwd-ok">{t('fwdinfo.listening')}{(f.conns ?? 0) > 0 && <b>{` · ${t('fwdinfo.conns', { count: f.conns ?? 0 })}`}</b>}</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="help" style={{ marginTop: 10 }}>{t('fwdinfo.help')}</div>
    </Modal>
  );
}
```

### `frontend/src/ipc/fakeBackend.ts` — 수정

```diff
--- a/frontend/src/ipc/fakeBackend.ts
+++ b/frontend/src/ipc/fakeBackend.ts
@@ -244,6 +244,7 @@
     },
     'session.reconnect': () => undefined,
     'session.ack': () => undefined,
+    'session.forwards': () => [],
     'clip.writeText': () => undefined,
     'clip.readText': () => '',
     'clip.info': () => ({ type: 'empty' }),
```

### `frontend/src/ipc/types.ts` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -197,6 +243,21 @@
   message?: string;
   canSftp: boolean;
   keepSession?: boolean;
+  /** 지금 연결에 설정한 포트 포워딩과 그 결과(session.md §1). 없으면 필드가 없다. */
+  forwards?: ForwardInfo[];
+}
+
+/** 포트 포워딩 하나의 설정 결과(session.md §1 forwards, D136). */
+export interface ForwardInfo {
+  type: 'L' | 'R' | 'D';
+  /** 듣는 주소 `호스트:포트` (L·D 는 이 PC, R 은 서버). */
+  bind: string;
+  /** 대상 `호스트:포트`. D 는 없다. */
+  dest?: string;
+  /** 실패했으면 그 까닭. */
+  error?: string;
+  /** 이 포워딩으로 지금 이어져 있는 연결 수(session.forwards 로 물었을 때의 값, D142). */
+  conns?: number;
 }
 
 export interface SyncStatus {
```

### `frontend/src/screens/Panes.tsx` — 수정

```diff
--- a/frontend/src/screens/Panes.tsx
+++ b/frontend/src/screens/Panes.tsx
@@ -1,6 +1,6 @@
 // 칸·분할 화면. 근거: spec/ui/panes.md §2, §4(크기 조절), §7(끌어 놓기), spec/ui/tokens.md §3(칸·분할선).
-import { useRef, type DragEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
-import { Link, Rows2, Square, RotateCw, X, Search, ZoomIn, Image as ImageIcon } from 'lucide-react';
+import { useRef, useState, type DragEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
+import { Link, Rows2, Square, RotateCw, X, Search, ZoomIn, Image as ImageIcon, ArrowLeftRight } from 'lucide-react';
 import type { LayoutNode, PaneNode, SplitNode, Dir } from '../lib/layout';
 import { edgeToSplit, edgeZone, findPane, paneIds, type Edge } from '../lib/layout';
 import { groupOf, useWorkspace, type Tab } from "../state/workspace";
@@ -16,6 +16,8 @@
 // 이 파일은 t 를 지역 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
 import { relabel, t as tr } from '../i18n';
 import { useShortcutLabels } from '../state/shortcuts';
+import { ForwardsDialog } from './ForwardsDialog';
+import { failedCount } from '../lib/forwards';
 
 const PANE_MIME = 'application/x-dariterm-pane';
 /** 사이드바 서버 끌어 놓기 MIME(에이전트 B 가 dragstart 에서 설정). */
@@ -210,6 +212,10 @@
   };
 
   const hasSession = !!info;
+  // 포트 포워딩 표시(§2.6): 연결된 동안, 설정된 포워딩이 있을 때만 머리글에 버튼을 둔다.
+  const forwards = info?.state === 'connected' ? info.forwards ?? [] : [];
+  const fwdFailed = failedCount(forwards);
+  const [fwdOpen, setFwdOpen] = useState(false);
   const reconnect = async () => {
     if (!info) return;
     try {
@@ -230,6 +236,7 @@
     { label: tr('sc.splitRight'), shortcut: scLabels.splitRight || undefined, disabled: !info || !canSplit, onSelect: () => void splitSame('row') },
     { label: tr('sc.splitDown'), shortcut: scLabels.splitDown || undefined, disabled: !info || !canSplit, onSelect: () => void splitSame('col') },
     { label: info?.protocol === 'local' ? tr('pane.openFiles') : tr('pane.openSftp'), shortcut: scLabels.sftp || undefined, disabled: !info?.canSftp && info?.protocol !== 'local', onSelect: () => { focusPane(pane.id); useUi.getState().setSftpOpen(groupOf(tab), true); } },
+    ...(forwards.length ? [{ label: tr('fwdinfo.menu', { count: forwards.length }), onSelect: () => setFwdOpen(true) }] : []),
     { label: tr('pane.reconnect'), disabled: !hasSession || entry?.info.state === 'connected', onSelect: () => void reconnect() },
     { separator: true },
     { label: tr('pane.close'), danger: true, onSelect: () => void closePane(pane.id) },
@@ -263,6 +270,17 @@
         <span className="pane-title">{relabel(info?.title) ?? tr('pane.empty')}</span>
         {linkInput && hasSession && <Link size={12} color="var(--warn)" aria-label={tr('pane.broadcasting')} className="pane-link" />}
         {info && <span className="faint small pane-target">{info.target}</span>}
+        {forwards.length > 0 && (
+          <button
+            className={`pane-fwd${fwdFailed ? ' failed' : ''}`}
+            title={fwdFailed ? tr('fwdinfo.btnFailed', { count: forwards.length, failed: fwdFailed }) : tr('fwdinfo.btn', { count: forwards.length })}
+            aria-label={tr('fwdinfo.btn', { count: forwards.length })}
+            onClick={() => setFwdOpen(true)}
+          >
+            <ArrowLeftRight size={11} />
+            {forwards.length}
+          </button>
+        )}
         <span className="spacer" />
         {hasSession && entry?.info.state !== 'connected' && (
           <button className="icon-btn small" title={tr('pane.reconnect')} aria-label={tr('pane.reconnect')} onClick={() => void reconnect()}><RotateCw size={12} /></button>
@@ -303,6 +321,12 @@
         )}
         {total >= 2 && hasSession && <div className="pane-watermark">{relabel(info?.title)}</div>}
       </div>
+      {fwdOpen && info && forwards.length > 0 && (
+        // 창 안의 클릭·우클릭이 칸으로 번지지 않게 막는다(칸 메뉴가 뜨거나 끌기가 시작되지 않게).
+        <div onMouseDown={(e) => e.stopPropagation()} onContextMenu={(e) => e.stopPropagation()}>
+          <ForwardsDialog sessionId={info.id} title={relabel(info.title) ?? ''} target={info.target} forwards={forwards} onClose={() => setFwdOpen(false)} />
+        </div>
+      )}
     </div>
   );
 }
```

### `frontend/src/styles/app.css` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 2, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -326,6 +329,25 @@
 .pane-head .pane-target { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
 .pane-head .spacer { flex: 1; }
 .env-tag { font-size: 10px; font-weight: 700; padding: 0 5px; border-radius: 4px; color: #fff; }
+/* 변수 값을 묻는 창의 미리 보기 (ui/sftp-snippets.md §4.3). */
+.vars-preview {
+  margin: 0; padding: 8px 10px; border-radius: 6px; border: 1px solid var(--border); background: var(--bg);
+  font-size: 12px; line-height: 1.5; white-space: pre-wrap; word-break: break-all; max-height: 140px; overflow: auto;
+}
+/* 창 머리글의 포트 포워딩 버튼(아이콘 + 개수)과 정보 창의 표. 근거: spec/ui/panes.md §2.6 (D137). */
+.pane-fwd {
+  flex: none; display: inline-flex; align-items: center; gap: 4px; height: 18px; padding: 0 7px; border-radius: 9px;
+  border: 1px solid var(--accent); background: transparent; color: var(--fg-2); font: inherit; font-size: 11px; font-weight: 600; cursor: pointer;
+}
+.pane-fwd:hover { background: var(--hover); color: var(--fg); }
+.pane-fwd.failed { border-color: var(--warn); color: var(--warn); }
+.fwd-info { width: 100%; border-collapse: collapse; font-size: 12.5px; }
+.fwd-info th { text-align: left; font-weight: 600; color: var(--fg-2); padding: 6px 8px; border-bottom: 1px solid var(--border); white-space: nowrap; }
+.fwd-info td { padding: 6px 8px; border-bottom: 1px solid var(--border); vertical-align: top; }
+.fwd-info td:nth-child(1), .fwd-info td:nth-child(2) { white-space: nowrap; }
+.fwd-info tr.failed td.mono { color: var(--fg-3); }
+.fwd-ok { color: var(--ok, #22c55e); }
+.fwd-bad { color: var(--danger); word-break: break-word; }
 .pane-body { flex: 1; min-height: 0; position: relative; display: flex; padding: 4px 2px 2px 8px; background: #0d1117; }
 .term-host { flex: 1; min-width: 0; min-height: 0; display: flex; }
 /* 세션별로 살려 두는 터미널 DOM(칸이 다시 그려질 때 떼었다 붙인다). */
```

### `internal/app/channels_session.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -155,6 +174,18 @@
 		}
 		pSession.AckOutput(iCount)
 		return nil, nil
+	})
+	// session.forwards: 포트 포워딩 정보 창을 열 때(와 [새로고침]을 누를 때) 묻는다 (지금 이어져 있는 연결 수, D142).
+	pRouter.Handle("session.forwards", func(slArgs []json.RawMessage) (any, error) {
+		szId, errArg := argString(slArgs, 0)
+		if errArg != nil {
+			return nil, errArg
+		}
+		pSession, errGet := pService.pSessions.Get(szId)
+		if errGet != nil {
+			return []session.ForwardInfo{}, nil // 이미 닫힌 세션: 빈 목록 (창은 곧 닫힌다)
+		}
+		return pSession.Forwards(), nil
 	})
 	pRouter.Handle("local.tmuxAvailable", func(slArgs []json.RawMessage) (any, error) {
 		return session.TmuxAvailable(), nil
```

### `internal/app/ssh_app.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/app/ssh_app.go
+++ b/internal/app/ssh_app.go
@@ -21,8 +21,20 @@
 // sshSession 은 SSH 셸과 그 연결·포워딩을 함께 닫는 전송이다.
 type sshSession struct {
 	*sshconn.Shell
-	conn    *sshconn.Conn
-	closers []io.Closer
+	conn     *sshconn.Conn
+	closers  []io.Closer
+	forwards []sshconn.ForwardResult
+	password string // 로그인에 쓴 비밀번호 (로그인 스크립트의 {{password}} 용, 세션 사본과 같은 수명)
+}
+
+// LoginSecret 은 이 서버에 로그인할 때 쓴 비밀번호다 (session.SecretProvider). 키·에이전트로 로그인했으면 빈 글자다.
+func (pSess *sshSession) LoginSecret() string {
+	return pSess.password
+}
+
+// Forwards 는 이 연결에 설정한 포트 포워딩과 그 결과, 지금 이어져 있는 연결 수다 (session.ForwardReporter).
+func (pSess *sshSession) Forwards() []session.ForwardInfo {
+	return forwardInfos(pSess.forwards)
 }
 
 // Close 는 셸, 포워딩 소켓, 연결을 모두 닫는다.
@@ -72,8 +84,12 @@
 			_ = conn.Close()
 			return nil, errShell
 		}
-		sess := &sshSession{Shell: shell, conn: conn}
-		sess.closers = conn.StartForwards(forwardsOf(record), notify)
+		sess := &sshSession{Shell: shell, conn: conn, password: tgt.Password}
+		sess.closers, sess.forwards = conn.StartForwards(forwardsOf(record))
+		// 결과는 들여 쓴 여러 줄로 한 번에 알린다 (protocols.md §2.5, D136).
+		if szBlock := session.MsgForwards(sess.Forwards()); szBlock != "" {
+			notify(szBlock)
+		}
 		conn.StartKeepalive(record.M_iKeepaliveSec, func() { _ = sess.Close() })
 		if allowSave {
 			pService.afterSshLogin(record, conn, notify)
@@ -234,6 +250,19 @@
 		}
 	}
 	return pProxy
+}
+
+// forwardInfos 는 포워딩 설정 결과를 세션 정보에 싣는 모양으로 바꾼다.
+func forwardInfos(slResults []sshconn.ForwardResult) []session.ForwardInfo {
+	slInfos := make([]session.ForwardInfo, 0, len(slResults))
+	for _, result := range slResults {
+		info := session.ForwardInfo{M_szType: result.Type, M_szBind: result.Bind, M_szDest: result.Dest, M_iConns: result.Active()}
+		if result.Err != nil {
+			info.M_szError = result.Err.Error()
+		}
+		slInfos = append(slInfos, info)
+	}
+	return slInfos
 }
 
 // forwardsOf 는 서버의 포워딩 설정을 연결 쪽 형식으로 바꾼다.
```

### `internal/session/control.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/control.go
+++ b/internal/session/control.go
@@ -198,6 +204,17 @@
 	pSession.credit.Ack(iCount)
 }
 
+// Forwards 는 지금 연결의 포트 포워딩과 그 순간의 연결 수를 돌려준다 (spec/ipc.md session.forwards, D142).
+// 연결이 없거나 포워딩을 알려 주지 않는 프로토콜이면 빈 목록이다.
+func (pSession *Session) Forwards() []ForwardInfo {
+	if reporter, bOk := pSession.Transport().(ForwardReporter); bOk {
+		if slForwards := reporter.Forwards(); slForwards != nil {
+			return slForwards
+		}
+	}
+	return []ForwardInfo{}
+}
+
 // Transport 는 현재 연결된 전송을 돌려준다 (연결이 없으면 nil). SFTP 처럼 프로토콜 기능을 찾을 때 쓴다.
 func (pSession *Session) Transport() Transport {
 	pSession.mtxData.Lock()
```

### `internal/session/messages.go` — 수정

```diff
--- a/internal/session/messages.go
+++ b/internal/session/messages.go
@@ -82,6 +82,47 @@
 // MsgTmuxMissing 은 tmux 가 없을 때 스크립트가 내는 안내다 (session.md §7 1번).
 func MsgTmuxMissing() string {
 	return notice(G_cszColorWarn, i18n.T("be.session.tmuxMissing"))
+}
+
+// MsgForwards 는 포트 포워딩 설정 결과를 여러 줄로 적는다 (protocols.md §2.5, D136).
+// 첫 줄은 개수(실패가 있으면 실패 수도), 그 아래에 포워딩마다 한 줄씩 들여 쓴다. 듣는 주소는 폭을 맞춘다.
+// 실패한 줄은 빨강이다. 포워딩이 없으면 빈 글자다.
+func MsgForwards(slForwards []ForwardInfo) string {
+	if len(slForwards) == 0 {
+		return ""
+	}
+	iWidth, iFailed := 0, 0
+	for _, fw := range slForwards {
+		if iLen := len([]rune(fw.M_szBind)); iLen > iWidth {
+			iWidth = iLen
+		}
+		if fw.M_szError != "" {
+			iFailed++
+		}
+	}
+	szHead := i18n.T("be.fwd.header", "count", len(slForwards))
+	szColor := G_cszColorInfo
+	if iFailed > 0 {
+		szHead = i18n.T("be.fwd.headerFailed", "count", len(slForwards), "failed", iFailed)
+		szColor = G_cszColorWarn
+	}
+	var sb strings.Builder
+	sb.WriteString(notice(szColor, szHead))
+	for _, fw := range slForwards {
+		szPad := strings.Repeat(" ", iWidth-len([]rune(fw.M_szBind)))
+		szLine := "    " + fw.M_szType + "  " + fw.M_szBind + szPad
+		switch {
+		case fw.M_szError != "":
+			sb.WriteString(G_cszColorError + szLine + "  ✕  " + i18n.T("be.fwd.failed", "error", fw.M_szError) + G_cszColorReset + "\r\n")
+			continue
+		case fw.M_szType == "D":
+			szLine += "  →  " + i18n.T("be.fwd.socks")
+		default:
+			szLine += "  →  " + fw.M_szDest
+		}
+		sb.WriteString(G_cszColorInfo + szLine + G_cszColorReset + "\r\n")
+	}
+	return sb.String()
 }
 
 // g_slErrorRules 는 오류 문구 번역표다 (session.md §3). 위에서 아래 순서로 첫 일치를 쓴다.
```

### `internal/session/session.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -170,6 +172,10 @@
 	pSession.info.M_i64ConnectedAt = tNow.UnixMilli()
 	pSession.tConnected = tNow
 	pSession.tLastInput = time.Time{}
+	pSession.info.M_slForwards = nil
+	if reporter, bOk := conn.(ForwardReporter); bOk {
+		pSession.info.M_slForwards = reporter.Forwards()
+	}
 	iCols, iRows := pSession.iCols, pSession.iRows
 	pSession.mtxData.Unlock()
 
```

### `internal/session/types.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/types.go
+++ b/internal/session/types.go
@@ -40,6 +40,23 @@
 	M_szMessage      string  `json:"message,omitempty"`
 	M_bCanSftp       bool    `json:"canSftp"`
 	M_bKeepSession   bool    `json:"keepSession,omitempty"`
+	// M_slForwards 는 지금 연결에 설정한 포트 포워딩과 그 결과다 (session.md §1, D136). 없으면 싣지 않는다.
+	M_slForwards []ForwardInfo `json:"forwards,omitempty"`
+}
+
+// ForwardInfo 는 포트 포워딩 하나의 설정 결과다 (session.md §1 forwards).
+type ForwardInfo struct {
+	M_szType  string `json:"type"`            // L | R | D
+	M_szBind  string `json:"bind"`            // 듣는 주소 "호스트:포트"
+	M_szDest  string `json:"dest,omitempty"`  // 대상 "호스트:포트" (D 는 없음)
+	M_szError string `json:"error,omitempty"` // 실패했으면 그 까닭, 성공이면 없음
+	M_iConns  int    `json:"conns"`           // 이 포워딩으로 지금 이어져 있는 연결 수 (D142)
+}
+
+// ForwardReporter 는 포트 포워딩 결과를 알려 줄 수 있는 Transport 다 (SSH).
+// 세션은 접속에 성공하면 이것으로 Info 의 forwards 를 채운다.
+type ForwardReporter interface {
+	Forwards() []ForwardInfo
 }
 
 // OpenConfig 는 세션 하나를 여는 설정이다.
```

### `internal/sshconn/keepalive_forward.go` — 수정

```diff
--- a/internal/sshconn/keepalive_forward.go
+++ b/internal/sshconn/keepalive_forward.go
@@ -1,10 +1,10 @@
 package sshconn
 
 import (
-	"dariterm/internal/i18n"
 	"io"
 	"net"
 	"strconv"
+	"sync/atomic"
 	"time"
 )
 
@@ -67,9 +67,30 @@
 	}
 }
 
-// StartForwards 는 포워딩을 설정한다. 성공과 실패는 notice 로 알린다 (protocols.md §2.5). 세션이 끝나면 Close 로 정리된다.
-func (pConn *Conn) StartForwards(slForwards []Forward, notice func(string)) []io.Closer {
+// ForwardResult 는 포워딩 하나를 설정한 결과다. Err 가 nil 이면 대기 중이다.
+type ForwardResult struct {
+	Type string // L | R | D
+	Bind string // 듣는 주소 "호스트:포트" (L·D 는 이 PC, R 은 서버)
+	Dest string // 대상 "호스트:포트" (D 는 빈 값)
+	Err  error
+	// pActive 는 이 포워딩으로 지금 이어져 있는 연결 수다 (실패한 항목은 nil).
+	pActive *atomic.Int64
+}
+
+// Active 는 이 포워딩으로 지금 이어져 있는 연결 수다. 실패한 포워딩은 0 이다.
+func (result ForwardResult) Active() int {
+	if result.pActive == nil {
+		return 0
+	}
+	return int(result.pActive.Load())
+}
+
+// StartForwards 는 포워딩을 설정하고, 설정마다 결과를 돌려준다 (protocols.md §2.5).
+// 실패한 항목이 있어도 다음 항목으로 계속한다. 터미널 안내와 화면 표시는 부른 쪽이 결과로 만든다.
+// 세션이 끝나면 돌려준 Closer 들을 닫아 정리한다.
+func (pConn *Conn) StartForwards(slForwards []Forward) ([]io.Closer, []ForwardResult) {
 	var slClosers []io.Closer
+	slResults := make([]ForwardResult, 0, len(slForwards))
 	for _, fw := range slForwards {
 		szBind := fw.BindHost
 		if szBind == "" {
@@ -77,40 +98,44 @@
 		}
 		szBindAddr := net.JoinHostPort(szBind, strconv.Itoa(fw.BindPort))
 		szDest := net.JoinHostPort(fw.DestHost, strconv.Itoa(fw.DestPort))
-		if fw.Type == "D" {
+		result := ForwardResult{Type: "L", Bind: szBindAddr, Dest: szDest}
+		var ln net.Listener
+		switch fw.Type {
+		case "D":
 			// 동적 포워딩: 이 PC 의 포트가 SOCKS 프록시가 되고, 요청마다 대상을 SSH 연결로 잇는다 (D109).
-			ln, errListen := net.Listen("tcp", szBindAddr)
-			if errListen != nil {
-				notice(i18n.T("be.fwd.dynamicFailed", "port", fw.BindPort, "error", errListen))
-				continue
+			result.Type, result.Dest = "D", ""
+			ln, result.Err = net.Listen("tcp", szBindAddr)
+			if result.Err == nil {
+				ln, result.pActive = counted(ln)
+				go serveSocks(ln, pConn.Client.Dial)
 			}
-			notice(i18n.T("be.fwd.dynamicActive", "from", szBindAddr))
+		case "R":
+			result.Type = "R"
+			ln, result.Err = pConn.Client.Listen("tcp", szBindAddr)
+			if result.Err == nil {
+				ln, result.pActive = counted(ln)
+				go acceptLoop(ln, func() (net.Conn, error) { return net.Dial("tcp", szDest) })
+			}
+		default:
+			ln, result.Err = net.Listen("tcp", szBindAddr)
+			if result.Err == nil {
+				ln, result.pActive = counted(ln)
+				client := pConn.Client
+				go acceptLoop(ln, func() (net.Conn, error) { return client.Dial("tcp", szDest) })
+			}
+		}
+		if result.Err == nil {
 			slClosers = append(slClosers, ln)
-			go serveSocks(ln, pConn.Client.Dial)
-			continue
 		}
-		if fw.Type == "R" {
-			ln, errListen := pConn.Client.Listen("tcp", szBindAddr)
-			if errListen != nil {
-				notice(i18n.T("be.fwd.remoteFailed", "port", fw.BindPort, "error", errListen))
-				continue
-			}
-			notice(i18n.T("be.fwd.active", "from", szBindAddr, "to", szDest))
-			slClosers = append(slClosers, ln)
-			go acceptLoop(ln, func() (net.Conn, error) { return net.Dial("tcp", szDest) })
-			continue
-		}
-		ln, errListen := net.Listen("tcp", szBindAddr)
-		if errListen != nil {
-			notice(i18n.T("be.fwd.localFailed", "port", fw.BindPort, "error", errListen))
-			continue
-		}
-		notice(i18n.T("be.fwd.active", "from", szBindAddr, "to", szDest))
-		slClosers = append(slClosers, ln)
-		client := pConn.Client
-		go acceptLoop(ln, func() (net.Conn, error) { return client.Dial("tcp", szDest) })
+		slResults = append(slResults, result)
 	}
-	return slClosers
+	return slClosers, slResults
+}
+
+// counted 는 대기 소켓을 연결 수를 세는 것으로 감싼다. 닫을 때는 돌려준 소켓을 닫으면 된다.
+func counted(ln net.Listener) (net.Listener, *atomic.Int64) {
+	pActive := &atomic.Int64{}
+	return &countingListener{Listener: ln, pActive: pActive}, pActive
 }
 
 // acceptLoop 는 들어오는 연결마다 대상에 이어 양방향으로 복사한다.
```

## C. 테스트 코드

### `internal/session/forwards_test.go` — 새 파일 (전체)

```go
package session

import (
	"dariterm/internal/i18n"
	"strings"
	"testing"
	"time"
)

// 포트 포워딩 안내는 포워딩마다 한 줄씩 들여 쓰고, 듣는 주소의 폭을 맞춘다 (protocols.md §2.5, D136).
func TestMsgForwardsLines(t *testing.T) {
	i18n.SetLang("ko")
	szOut := MsgForwards([]ForwardInfo{
		{M_szType: "L", M_szBind: "127.0.0.1:25095", M_szDest: "100.102.205.95:22"},
		{M_szType: "L", M_szBind: "127.0.0.1:80", M_szDest: "10.0.0.1:8080"},
		{M_szType: "D", M_szBind: "127.0.0.1:1080"},
	})
	slLines := strings.Split(strings.TrimSuffix(szOut, "\r\n"), "\r\n")
	if len(slLines) != 4 {
		t.Fatalf("줄 수 = %d, 4 를 기대: %q", len(slLines), szOut)
	}
	if !strings.Contains(slLines[0], "● 포트 포워딩 3개") {
		t.Errorf("첫 줄 = %q", slLines[0])
	}
	if !strings.Contains(slLines[1], "    L  127.0.0.1:25095  →  100.102.205.95:22") {
		t.Errorf("둘째 줄 = %q", slLines[1])
	}
	// 짧은 주소는 가장 긴 주소(15자)에 맞춰 공백을 채운다.
	if !strings.Contains(slLines[2], "    L  127.0.0.1:80     →  10.0.0.1:8080") {
		t.Errorf("셋째 줄 = %q", slLines[2])
	}
	if !strings.Contains(slLines[3], "    D  127.0.0.1:1080   →  SOCKS 프록시") {
		t.Errorf("넷째 줄 = %q", slLines[3])
	}
	for iIndex, szLine := range slLines {
		if !strings.HasSuffix(szLine, G_cszColorReset) {
			t.Errorf("%d 번째 줄이 색을 되돌리지 않는다: %q", iIndex, szLine)
		}
	}
}

// 실패한 포워딩은 빨간 줄로 까닭을 적고, 첫 줄에 실패 수를 적는다.
func TestMsgForwardsFailed(t *testing.T) {
	i18n.SetLang("ko")
	szOut := MsgForwards([]ForwardInfo{
		{M_szType: "L", M_szBind: "127.0.0.1:3307", M_szDest: "db:3306"},
		{M_szType: "R", M_szBind: "0.0.0.0:9000", M_szDest: "localhost:9000", M_szError: "bind: address already in use"},
	})
	if !strings.Contains(szOut, G_cszColorWarn+"● 포트 포워딩 2개 (실패 1개)") {
		t.Errorf("첫 줄에 실패 수가 없다: %q", szOut)
	}
	if !strings.Contains(szOut, G_cszColorError+"    R  0.0.0.0:9000    ✕  실패: bind: address already in use") {
		t.Errorf("실패 줄이 다르다: %q", szOut)
	}
}

func TestMsgForwardsEmpty(t *testing.T) {
	if szOut := MsgForwards(nil); szOut != "" {
		t.Errorf("포워딩이 없으면 빈 글자여야 한다: %q", szOut)
	}
}

// forwardTransport 는 포워딩 결과를 알려 주는 테스트용 연결이다.
type forwardTransport struct {
	*fakeTransport
	slForwards []ForwardInfo
}

func (pFake *forwardTransport) Forwards() []ForwardInfo { return pFake.slForwards }

// 접속에 성공하면 연결이 알려 준 포워딩 결과가 세션 정보에 실린다 (session.md §1 forwards).
func TestSessionInfoCarriesForwards(t *testing.T) {
	pFake := &forwardTransport{
		fakeTransport: newFakeTransport(),
		slForwards:    []ForwardInfo{{M_szType: "L", M_szBind: "127.0.0.1:8080", M_szDest: "localhost:80"}},
	}
	pRec := &eventRecorder{}
	pManager := NewManager(pRec.emit)
	info, errOpen := pManager.Open(OpenConfig{
		Info:      Info{M_szTitle: "fwd", M_szProtocol: "ssh"},
		Dialer:    func(notify func(string)) (Transport, error) { return pFake, nil },
		OpenDelay: time.Millisecond,
	})
	if errOpen != nil {
		t.Fatalf("Open: %v", errOpen)
	}
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		pSession.mtxData.Lock()
		szState, slForwards := pSession.info.M_szState, pSession.info.M_slForwards
		pSession.mtxData.Unlock()
		if szState == G_cszStateConnected {
			if len(slForwards) != 1 || slForwards[0].M_szBind != "127.0.0.1:8080" {
				t.Fatalf("forwards = %+v", slForwards)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("연결되지 않았다")
}
```

### `internal/sshconn/forward_count_test.go` — 새 파일 (전체)

```go
package sshconn

import (
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// waitActive 는 연결 수가 iWant 가 될 때까지 기다린다.
func waitActive(t *testing.T, pActive *atomic.Int64, iWant int64) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if pActive.Load() == iWant {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("연결 수 = %d, %d 를 기대", pActive.Load(), iWant)
}

// 포워딩으로 이어진 연결이 열릴 때 늘고 닫힐 때 준다 (포트 포워딩 정보 창의 "연결 N개", D142).
func TestForwardCountsOpenConnections(t *testing.T) {
	iEchoPort := startEcho(t)
	lnRaw, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	ln, pActive := counted(lnRaw)
	t.Cleanup(func() { _ = ln.Close() })
	szDest := net.JoinHostPort("127.0.0.1", strconv.Itoa(iEchoPort))
	go acceptLoop(ln, func() (net.Conn, error) { return net.DialTimeout("tcp", szDest, 2*time.Second) })

	if result := (ForwardResult{pActive: pActive}); result.Active() != 0 {
		t.Fatalf("처음 연결 수 = %d", result.Active())
	}
	connA, errA := net.Dial("tcp", lnRaw.Addr().String())
	if errA != nil {
		t.Fatal(errA)
	}
	connB, errB := net.Dial("tcp", lnRaw.Addr().String())
	if errB != nil {
		t.Fatal(errB)
	}
	waitActive(t, pActive, 2)

	// 이어져 있는 동안 데이터가 오간다.
	if _, errWrite := connA.Write([]byte("ping")); errWrite != nil {
		t.Fatal(errWrite)
	}
	bBuf := make([]byte, 4)
	_ = connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, errRead := io.ReadFull(connA, bBuf); errRead != nil || string(bBuf) != "ping" {
		t.Fatalf("되돌아온 것 = %q, %v", bBuf, errRead)
	}

	_ = connA.Close()
	waitActive(t, pActive, 1)
	_ = connB.Close()
	waitActive(t, pActive, 0)
}

// 대상에 닿지 못한 연결도 닫히면 수에서 빠진다.
func TestForwardCountDropsFailedDial(t *testing.T) {
	lnRaw, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	ln, pActive := counted(lnRaw)
	t.Cleanup(func() { _ = ln.Close() })
	go acceptLoop(ln, func() (net.Conn, error) { return nil, io.ErrClosedPipe })
	conn, errDial := net.Dial("tcp", lnRaw.Addr().String())
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	// 서버 쪽이 곧바로 닫는다: 읽기가 끝나면 수는 0 이어야 한다.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Read(make([]byte, 1))
	waitActive(t, pActive, 0)
}

// 실패한 포워딩(세는 것이 없음)의 연결 수는 0 이다.
func TestForwardResultActiveWithoutCounter(t *testing.T) {
	if iActive := (ForwardResult{Err: io.EOF}).Active(); iActive != 0 {
		t.Fatalf("= %d", iActive)
	}
}
```

### `frontend/src/lib/forwards.test.ts` — 새 파일 (전체)

```ts
// 포트 포워딩 표시용 계산 (spec/ui/panes.md §2.6, D137).
import { describe, expect, it } from 'vitest';
import { failedCount, forwardsText } from './forwards';
import type { ForwardInfo } from '../ipc/types';

const LIST: ForwardInfo[] = [
  { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
  { type: 'D', bind: '127.0.0.1:1080' },
  { type: 'R', bind: '0.0.0.0:9000', dest: 'localhost:9000', error: 'address already in use' },
];

describe('forwards', () => {
  it('실패한 수를 센다', () => {
    expect(failedCount(LIST)).toBe(1);
    expect(failedCount(undefined)).toBe(0);
  });

  it('복사용 글자는 한 줄에 하나, 듣는 주소 폭을 맞춘다', () => {
    const text = forwardsText(LIST, 'SOCKS', (e) => `실패: ${e}`);
    expect(text.split('\n')).toEqual([
      'L  127.0.0.1:25095  →  100.102.205.95:22',
      'D  127.0.0.1:1080   →  SOCKS',
      'R  0.0.0.0:9000     ✕  실패: address already in use',
    ]);
  });
});
```

### `frontend/src/screens/Automation.test.tsx` — 새 파일 (이 문서의 몫: `describe('포트 포워딩 정보 창 (D137)')`, 전체는 5번 문서 §C)

```tsx
describe('포트 포워딩 정보 창 (D137)', () => {
  it('포워딩마다 한 줄씩 종류·듣는 주소·대상·상태를 보인다', () => {
    seed();
    render(
      <ForwardsDialog
        title="gw"
        target="samsung@100.102.204.43:22"
        onClose={() => undefined}
        forwards={[
          { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
          { type: 'D', bind: '127.0.0.1:1080' },
          { type: 'R', bind: '0.0.0.0:9000', dest: 'localhost:9000', error: 'address already in use' },
        ]}
      />,
    );
    const dialog = screen.getByRole('dialog', { name: '포트 포워딩 · gw' });
    expect(within(dialog).getByText(/3개 설정됨, 1개 실패/)).toBeTruthy();
    const rows = within(dialog).getAllByRole('row');
    expect(rows).toHaveLength(4); // 머리 + 3
    expect(within(rows[1]).getByText('127.0.0.1:25095')).toBeTruthy();
    expect(within(rows[1]).getByText('100.102.205.95:22')).toBeTruthy();
    expect(within(rows[1]).getByText('동작 중')).toBeTruthy();
    expect(within(rows[2]).getByText('SOCKS 프록시')).toBeTruthy();
    expect(within(rows[3]).getByText('실패: address already in use')).toBeTruthy();
  });

  it('열 때 연결 수를 한 번 읽어 "동작 중 · 연결 N개"로 보이고, 주기적으로 다시 묻지 않는다 (D142)', async () => {
    seed();
    const asked: unknown[][] = [];
    let conns = 2;
    setBackend({
      invoke: (async (channel: string, args: unknown[] = []) => {
        if (channel !== 'session.forwards') return undefined;
        asked.push(args);
        return [
          { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22', conns },
          { type: 'L', bind: '127.0.0.1:25096', dest: '100.102.205.96:22', conns: 0 },
        ];
      }) as unknown as Backend['invoke'],
      termInput, termResize: () => undefined, reply: () => undefined, on: () => () => undefined,
    });
    vi.useFakeTimers();
    try {
      const view = render(
        <ForwardsDialog
          sessionId="s1"
          title="gw"
          target="samsung@100.102.204.43:22"
          onClose={() => undefined}
          forwards={[
            { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
            { type: 'L', bind: '127.0.0.1:25096', dest: '100.102.205.96:22' },
          ]}
        />,
      );
      await act(async () => { await vi.advanceTimersByTimeAsync(0); });
      expect(asked[0]).toEqual(['s1']);
      let rows = screen.getAllByRole('row');
      expect(rows[1].textContent).toContain('동작 중 · 연결 2개');
      expect(rows[2].textContent).toContain('동작 중');
      expect(rows[2].textContent).not.toContain('연결');
      expect(screen.getByText(/2개 설정됨 · 연결 2개/)).toBeTruthy();

      // 시간이 지나도 다시 묻지 않는다.
      conns = 0;
      await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
      expect(asked).toHaveLength(1);
      expect(screen.getAllByRole('row')[1].textContent).toContain('연결 2개');

      // [새로고침]을 누르면 그때 다시 읽는다.
      fireEvent.click(screen.getByRole('button', { name: '새로고침' }));
      await act(async () => { await vi.advanceTimersByTimeAsync(0); });
      expect(asked).toHaveLength(2);
      rows = screen.getAllByRole('row');
      expect(rows[1].textContent).not.toContain('연결');
      view.unmount();
    } finally {
      vi.useRealTimers();
    }
  });
});
```

## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다. 자리표시자(`{이름}`)는 언어마다 같은 이름을 쓴다(`frontend: npx vitest run src/i18n`, `go test ./internal/i18n` 이 확인한다).

**지운 키**(안내를 항목마다 내던 옛 문구, 5개): `be.fwd.active` · `be.fwd.dynamicActive` · `be.fwd.localFailed` · `be.fwd.remoteFailed` · `be.fwd.dynamicFailed`

정보 창의 [새로고침] 버튼(D142)은 이미 있던 키 `sftp.refresh` 를 쓴다(새 키 없음).

**`be.fwd.header`** (추가)

```json
  "be.fwd.header": "포트 포워딩 {count}개",
  "be.fwd.header": "Port forwarding: {count}",
  "be.fwd.header": "ポート転送 {count}件",
  "be.fwd.header": "端口转发 {count} 项",
  "be.fwd.header": "連接埠轉送 {count} 項",
  "be.fwd.header": "Portweiterleitungen: {count}",
  "be.fwd.header": "Redirections de port : {count}",
  "be.fwd.header": "Reenvíos de puertos: {count}",
  "be.fwd.header": "Encaminhamentos de porta: {count}",
  "be.fwd.header": "Проброс портов: {count}",
```

**`be.fwd.headerFailed`** (추가)

```json
  "be.fwd.headerFailed": "포트 포워딩 {count}개 (실패 {failed}개)",
  "be.fwd.headerFailed": "Port forwarding: {count} ({failed} failed)",
  "be.fwd.headerFailed": "ポート転送 {count}件 (失敗 {failed}件)",
  "be.fwd.headerFailed": "端口转发 {count} 项（失败 {failed} 项）",
  "be.fwd.headerFailed": "連接埠轉送 {count} 項（失敗 {failed} 項）",
  "be.fwd.headerFailed": "Portweiterleitungen: {count} ({failed} fehlgeschlagen)",
  "be.fwd.headerFailed": "Redirections de port : {count} ({failed} en échec)",
  "be.fwd.headerFailed": "Reenvíos de puertos: {count} ({failed} con error)",
  "be.fwd.headerFailed": "Encaminhamentos de porta: {count} ({failed} com falha)",
  "be.fwd.headerFailed": "Проброс портов: {count} (ошибок: {failed})",
```

**`be.fwd.socks`** (추가)

```json
  "be.fwd.socks": "SOCKS 프록시",
  "be.fwd.socks": "SOCKS proxy",
  "be.fwd.socks": "SOCKS プロキシ",
  "be.fwd.socks": "SOCKS 代理",
  "be.fwd.socks": "SOCKS 代理",
  "be.fwd.socks": "SOCKS-Proxy",
  "be.fwd.socks": "proxy SOCKS",
  "be.fwd.socks": "proxy SOCKS",
  "be.fwd.socks": "proxy SOCKS",
  "be.fwd.socks": "SOCKS-прокси",
```

**`be.fwd.failed`** (추가)

```json
  "be.fwd.failed": "실패: {error}",
  "be.fwd.failed": "failed: {error}",
  "be.fwd.failed": "失敗: {error}",
  "be.fwd.failed": "失败：{error}",
  "be.fwd.failed": "失敗：{error}",
  "be.fwd.failed": "fehlgeschlagen: {error}",
  "be.fwd.failed": "échec : {error}",
  "be.fwd.failed": "error: {error}",
  "be.fwd.failed": "falha: {error}",
  "be.fwd.failed": "ошибка: {error}",
```

**`fwdinfo.title`** (추가)

```json
  "fwdinfo.title": "포트 포워딩 · {name}",
  "fwdinfo.title": "Port forwarding · {name}",
  "fwdinfo.title": "ポート転送 · {name}",
  "fwdinfo.title": "端口转发 · {name}",
  "fwdinfo.title": "連接埠轉送 · {name}",
  "fwdinfo.title": "Portweiterleitung · {name}",
  "fwdinfo.title": "Redirection de port · {name}",
  "fwdinfo.title": "Reenvío de puertos · {name}",
  "fwdinfo.title": "Encaminhamento de porta · {name}",
  "fwdinfo.title": "Проброс портов · {name}",
```

**`fwdinfo.summary`** (추가)

```json
  "fwdinfo.summary": "{count}개 설정됨",
  "fwdinfo.summary": "{count} configured",
  "fwdinfo.summary": "{count}件設定済み",
  "fwdinfo.summary": "已设置 {count} 项",
  "fwdinfo.summary": "已設定 {count} 項",
  "fwdinfo.summary": "{count} eingerichtet",
  "fwdinfo.summary": "{count} configurée(s)",
  "fwdinfo.summary": "{count} configurados",
  "fwdinfo.summary": "{count} configurados",
  "fwdinfo.summary": "Настроено: {count}",
```

**`fwdinfo.summaryFailed`** (추가)

```json
  "fwdinfo.summaryFailed": "{count}개 설정됨, {failed}개 실패",
  "fwdinfo.summaryFailed": "{count} configured, {failed} failed",
  "fwdinfo.summaryFailed": "{count}件設定済み、{failed}件失敗",
  "fwdinfo.summaryFailed": "已设置 {count} 项，失败 {failed} 项",
  "fwdinfo.summaryFailed": "已設定 {count} 項，失敗 {failed} 項",
  "fwdinfo.summaryFailed": "{count} eingerichtet, {failed} fehlgeschlagen",
  "fwdinfo.summaryFailed": "{count} configurée(s), {failed} en échec",
  "fwdinfo.summaryFailed": "{count} configurados, {failed} con error",
  "fwdinfo.summaryFailed": "{count} configurados, {failed} com falha",
  "fwdinfo.summaryFailed": "Настроено: {count}, ошибок: {failed}",
```

**`fwdinfo.colKind`** (추가)

```json
  "fwdinfo.colKind": "종류",
  "fwdinfo.colKind": "Type",
  "fwdinfo.colKind": "種類",
  "fwdinfo.colKind": "类型",
  "fwdinfo.colKind": "類型",
  "fwdinfo.colKind": "Typ",
  "fwdinfo.colKind": "Type",
  "fwdinfo.colKind": "Tipo",
  "fwdinfo.colKind": "Tipo",
  "fwdinfo.colKind": "Тип",
```

**`fwdinfo.colBind`** (추가)

```json
  "fwdinfo.colBind": "듣는 주소",
  "fwdinfo.colBind": "Listening on",
  "fwdinfo.colBind": "待ち受けアドレス",
  "fwdinfo.colBind": "监听地址",
  "fwdinfo.colBind": "監聽位址",
  "fwdinfo.colBind": "Lauscht auf",
  "fwdinfo.colBind": "Écoute sur",
  "fwdinfo.colBind": "Escucha en",
  "fwdinfo.colBind": "Escutando em",
  "fwdinfo.colBind": "Слушает",
```

**`fwdinfo.colDest`** (추가)

```json
  "fwdinfo.colDest": "대상",
  "fwdinfo.colDest": "Destination",
  "fwdinfo.colDest": "転送先",
  "fwdinfo.colDest": "目标",
  "fwdinfo.colDest": "目標",
  "fwdinfo.colDest": "Ziel",
  "fwdinfo.colDest": "Destination",
  "fwdinfo.colDest": "Destino",
  "fwdinfo.colDest": "Destino",
  "fwdinfo.colDest": "Назначение",
```

**`fwdinfo.colState`** (추가)

```json
  "fwdinfo.colState": "상태",
  "fwdinfo.colState": "Status",
  "fwdinfo.colState": "状態",
  "fwdinfo.colState": "状态",
  "fwdinfo.colState": "狀態",
  "fwdinfo.colState": "Status",
  "fwdinfo.colState": "État",
  "fwdinfo.colState": "Estado",
  "fwdinfo.colState": "Estado",
  "fwdinfo.colState": "Состояние",
```

**`fwdinfo.kindL`** (추가)

```json
  "fwdinfo.kindL": "로컬",
  "fwdinfo.kindL": "Local",
  "fwdinfo.kindL": "ローカル",
  "fwdinfo.kindL": "本地",
  "fwdinfo.kindL": "本機",
  "fwdinfo.kindL": "Lokal",
  "fwdinfo.kindL": "Locale",
  "fwdinfo.kindL": "Local",
  "fwdinfo.kindL": "Local",
  "fwdinfo.kindL": "Локальный",
```

**`fwdinfo.kindR`** (추가)

```json
  "fwdinfo.kindR": "원격",
  "fwdinfo.kindR": "Remote",
  "fwdinfo.kindR": "リモート",
  "fwdinfo.kindR": "远程",
  "fwdinfo.kindR": "遠端",
  "fwdinfo.kindR": "Remote",
  "fwdinfo.kindR": "Distante",
  "fwdinfo.kindR": "Remoto",
  "fwdinfo.kindR": "Remoto",
  "fwdinfo.kindR": "Удалённый",
```

**`fwdinfo.kindD`** (추가)

```json
  "fwdinfo.kindD": "동적",
  "fwdinfo.kindD": "Dynamic",
  "fwdinfo.kindD": "ダイナミック",
  "fwdinfo.kindD": "动态",
  "fwdinfo.kindD": "動態",
  "fwdinfo.kindD": "Dynamisch",
  "fwdinfo.kindD": "Dynamique",
  "fwdinfo.kindD": "Dinámico",
  "fwdinfo.kindD": "Dinâmico",
  "fwdinfo.kindD": "Динамический",
```

**`fwdinfo.socks`** (추가)

```json
  "fwdinfo.socks": "SOCKS 프록시",
  "fwdinfo.socks": "SOCKS proxy",
  "fwdinfo.socks": "SOCKS プロキシ",
  "fwdinfo.socks": "SOCKS 代理",
  "fwdinfo.socks": "SOCKS 代理",
  "fwdinfo.socks": "SOCKS-Proxy",
  "fwdinfo.socks": "Proxy SOCKS",
  "fwdinfo.socks": "Proxy SOCKS",
  "fwdinfo.socks": "Proxy SOCKS",
  "fwdinfo.socks": "SOCKS-прокси",
```

**`fwdinfo.failed`** (추가)

```json
  "fwdinfo.failed": "실패: {error}",
  "fwdinfo.failed": "Failed: {error}",
  "fwdinfo.failed": "失敗: {error}",
  "fwdinfo.failed": "失败：{error}",
  "fwdinfo.failed": "失敗：{error}",
  "fwdinfo.failed": "Fehlgeschlagen: {error}",
  "fwdinfo.failed": "Échec : {error}",
  "fwdinfo.failed": "Error: {error}",
  "fwdinfo.failed": "Falha: {error}",
  "fwdinfo.failed": "Ошибка: {error}",
```

**`fwdinfo.listening`** (추가 — D142 에서 값을 "대기 중"에서 "동작 중"으로 바꿈)

```json
  "fwdinfo.listening": "동작 중",
  "fwdinfo.listening": "Active",
  "fwdinfo.listening": "動作中",
  "fwdinfo.listening": "运行中",
  "fwdinfo.listening": "運作中",
  "fwdinfo.listening": "Aktiv",
  "fwdinfo.listening": "Active",
  "fwdinfo.listening": "Activo",
  "fwdinfo.listening": "Ativo",
  "fwdinfo.listening": "Работает",
```

**`fwdinfo.copied`** (추가)

```json
  "fwdinfo.copied": "포워딩 목록을 복사했습니다",
  "fwdinfo.copied": "Forwarding list copied",
  "fwdinfo.copied": "転送の一覧をコピーしました",
  "fwdinfo.copied": "已复制转发列表",
  "fwdinfo.copied": "已複製轉送清單",
  "fwdinfo.copied": "Weiterleitungsliste kopiert",
  "fwdinfo.copied": "Liste des redirections copiée",
  "fwdinfo.copied": "Lista de reenvíos copiada",
  "fwdinfo.copied": "Lista de encaminhamentos copiada",
  "fwdinfo.copied": "Список пробросов скопирован",
```

**`fwdinfo.help`** (추가)

```json
  "fwdinfo.help": "L: 이 PC의 듣는 주소로 접속하면 서버를 거쳐 대상으로 갑니다. R: 서버의 듣는 주소로 접속하면 이 PC를 거쳐 대상으로 갑니다. D: 듣는 주소가 SOCKS 프록시입니다. 바꾸려면 서버 편집 > 포트 포워딩에서 고친 뒤 다시 연결하세요.",
  "fwdinfo.help": "L: connections to the listening address on this PC go through the server to the destination. R: connections to the listening address on the server go through this PC to the destination. D: the listening address is a SOCKS proxy. To change these, edit the server's Port forwarding tab and reconnect.",
  "fwdinfo.help": "L: この PC の待ち受けアドレスへの接続は、サーバーを経由して転送先に届きます。R: サーバーの待ち受けアドレスへの接続は、この PC を経由して転送先に届きます。D: 待ち受けアドレスが SOCKS プロキシになります。変更するには、サーバーの編集 > ポート転送で直してから再接続してください。",
  "fwdinfo.help": "L：连接到本机的监听地址后，会经由服务器到达目标。R：连接到服务器的监听地址后，会经由本机到达目标。D：监听地址是 SOCKS 代理。如需更改，请在编辑服务器 > 端口转发中修改后重新连接。",
  "fwdinfo.help": "L：連線到本機的監聽位址後，會經由伺服器到達目標。R：連線到伺服器的監聽位址後，會經由本機到達目標。D：監聽位址是 SOCKS 代理。如需變更，請在編輯伺服器 > 連接埠轉送中修改後重新連線。",
  "fwdinfo.help": "L: Verbindungen zur lauschenden Adresse auf diesem PC gehen über den Server zum Ziel. R: Verbindungen zur lauschenden Adresse auf dem Server gehen über diesen PC zum Ziel. D: Die lauschende Adresse ist ein SOCKS-Proxy. Zum Ändern den Tab Portweiterleitung des Servers bearbeiten und neu verbinden.",
  "fwdinfo.help": "L : les connexions à l'adresse d'écoute de ce PC passent par le serveur jusqu'à la destination. R : les connexions à l'adresse d'écoute du serveur passent par ce PC jusqu'à la destination. D : l'adresse d'écoute est un proxy SOCKS. Pour modifier, changez l'onglet Redirection de port du serveur puis reconnectez-vous.",
  "fwdinfo.help": "L: las conexiones a la dirección de escucha de este PC pasan por el servidor hasta el destino. R: las conexiones a la dirección de escucha del servidor pasan por este PC hasta el destino. D: la dirección de escucha es un proxy SOCKS. Para cambiarlos, edite la pestaña Reenvío de puertos del servidor y vuelva a conectar.",
  "fwdinfo.help": "L: conexões ao endereço de escuta deste PC passam pelo servidor até o destino. R: conexões ao endereço de escuta do servidor passam por este PC até o destino. D: o endereço de escuta é um proxy SOCKS. Para alterar, edite a aba Encaminhamento de porta do servidor e reconecte.",
  "fwdinfo.help": "L: подключения к адресу на этом ПК идут через сервер к назначению. R: подключения к адресу на сервере идут через этот ПК к назначению. D: адрес работает как SOCKS-прокси. Чтобы изменить, отредактируйте вкладку «Проброс портов» сервера и переподключитесь.",
```

**`fwdinfo.btn`** (추가)

```json
  "fwdinfo.btn": "포트 포워딩 {count}개 — 눌러서 보기",
  "fwdinfo.btn": "Port forwarding: {count} — click to view",
  "fwdinfo.btn": "ポート転送 {count}件 — クリックで表示",
  "fwdinfo.btn": "端口转发 {count} 项 — 点击查看",
  "fwdinfo.btn": "連接埠轉送 {count} 項 — 點選檢視",
  "fwdinfo.btn": "Portweiterleitungen: {count} — zum Anzeigen klicken",
  "fwdinfo.btn": "Redirections de port : {count} — cliquer pour afficher",
  "fwdinfo.btn": "Reenvíos de puertos: {count} — clic para ver",
  "fwdinfo.btn": "Encaminhamentos de porta: {count} — clique para ver",
  "fwdinfo.btn": "Проброс портов: {count} — нажмите для просмотра",
```

**`fwdinfo.btnFailed`** (추가)

```json
  "fwdinfo.btnFailed": "포트 포워딩 {count}개 (실패 {failed}개) — 눌러서 보기",
  "fwdinfo.btnFailed": "Port forwarding: {count} ({failed} failed) — click to view",
  "fwdinfo.btnFailed": "ポート転送 {count}件 (失敗 {failed}件) — クリックで表示",
  "fwdinfo.btnFailed": "端口转发 {count} 项（失败 {failed} 项）— 点击查看",
  "fwdinfo.btnFailed": "連接埠轉送 {count} 項（失敗 {failed} 項）— 點選檢視",
  "fwdinfo.btnFailed": "Portweiterleitungen: {count} ({failed} fehlgeschlagen) — zum Anzeigen klicken",
  "fwdinfo.btnFailed": "Redirections de port : {count} ({failed} en échec) — cliquer pour afficher",
  "fwdinfo.btnFailed": "Reenvíos de puertos: {count} ({failed} con error) — clic para ver",
  "fwdinfo.btnFailed": "Encaminhamentos de porta: {count} ({failed} com falha) — clique para ver",
  "fwdinfo.btnFailed": "Проброс портов: {count} (ошибок: {failed}) — нажмите для просмотра",
```

**`fwdinfo.menu`** (추가)

```json
  "fwdinfo.menu": "포트 포워딩 보기 ({count}개)",
  "fwdinfo.menu": "View port forwarding ({count})",
  "fwdinfo.menu": "ポート転送を表示 ({count}件)",
  "fwdinfo.menu": "查看端口转发（{count} 项）",
  "fwdinfo.menu": "檢視連接埠轉送（{count} 項）",
  "fwdinfo.menu": "Portweiterleitungen anzeigen ({count})",
  "fwdinfo.menu": "Afficher les redirections de port ({count})",
  "fwdinfo.menu": "Ver reenvíos de puertos ({count})",
  "fwdinfo.menu": "Ver encaminhamentos de porta ({count})",
  "fwdinfo.menu": "Показать проброс портов ({count})",
```

**`fwdinfo.conns`** (추가 (D142))

```json
  "fwdinfo.conns": "연결 {count}개",
  "fwdinfo.conns": "{count} connection(s)",
  "fwdinfo.conns": "接続 {count}件",
  "fwdinfo.conns": "{count} 个连接",
  "fwdinfo.conns": "{count} 個連線",
  "fwdinfo.conns": "{count} Verbindung(en)",
  "fwdinfo.conns": "{count} connexion(s)",
  "fwdinfo.conns": "{count} conexión(es)",
  "fwdinfo.conns": "{count} conexão(ões)",
  "fwdinfo.conns": "подключений: {count}",
```

## F. 명세

원본 앱에 없던 기능이라 명세를 이 작업에서 새로 썼다(아래 조각의 `+` 줄이 새 명세다). 줄이 길어 가로로 넘치면 파일을 직접 여는 편이 읽기 쉽다.

### `spec/acceptance/03-panes-terminal.md` — 수정

```diff
--- a/spec/acceptance/03-panes-terminal.md
+++ b/spec/acceptance/03-panes-terminal.md
@@ -21,6 +21,10 @@
 | P14 | (삭제 2026-11-23, D116) | ~~"바둑판로 모으기"~~ | 기능을 없앴다 |
 | P15 | (삭제 2026-11-23, D116) | ~~폴더 우클릭 → 그리드로 열기~~ | 기능을 없앴다 |
 | P15a | 기본 설정(최대 화면 분할 3), 화면을 좌우 둘로 나눔 | 양쪽 탭에서 각각 아래로 나누기를 되풀이 | 탭마다 창 3개에서 [아래로 나누기]가 비활성이 된다. 한 화면에 보이는 창은 최대 6개다 (D116) |
+| P16 | 포트 포워딩 3개(L 2개, D 1개)가 있는 SSH 서버에 연결 | | 터미널에 `● 포트 포워딩 3개` 와 그 아래 들여 쓴 세 줄(화살표가 세로로 맞음). 창 머리글의 접속 대상 옆에 `⇄ 3` 버튼(테두리가 강조색, D143) (D136·D137) |
+| P17 | 그 버튼을 누름 | | "포트 포워딩 · {제목}" 창에 종류·듣는 주소·대상·상태 표 세 줄, [복사] 로 목록이 클립보드에 (D137) |
+| P17a | 포워딩 하나로 접속해 둠(예: `ssh -p 25095 127.0.0.1`) | 그 창을 엶 | 그 줄의 상태가 `동작 중 · 연결 1개`, 나머지는 `동작 중`. 열어 둔 채 접속을 끊어도 저절로 바뀌지 않고, [새로고침]을 누르면 `동작 중` 으로 (D142) |
+| P18 | 이미 쓰이는 포트를 듣는 포워딩이 있음 | | 첫 줄이 `(실패 1개)` 노랑, 그 줄은 빨강 `✕  실패: …`, 버튼이 경고색. 나머지 포워딩과 셸은 그대로 쓴다 (D136) |
 
 ## 터미널 (behavior/terminal.md)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/behavior/session.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/behavior/session.md
+++ b/spec/behavior/session.md
@@ -18,6 +18,7 @@
 | message | 재접속 대기 시 안내 등 |
 | canSftp | protocol 이 ssh 이면 true |
 | keepSession | 세션 유지가 적용되는 서버면 true |
+| forwards | (2026-11-27, D136) **지금 연결**에 설정한 포트 포워딩과 그 결과의 배열. 포워딩이 없는 서버·SSH 가 아닌 세션은 필드가 없다. 항목: `type`(`L`/`R`/`D`), `bind`(듣는 주소 `호스트:포트` — L·D 는 이 PC, R 은 서버), `dest`(대상 `호스트:포트`, D 는 없음), `error`(실패했으면 그 까닭, 성공이면 없음), `conns`(이 포워딩으로 지금 이어져 있는 연결 수 — 세션 정보에 실릴 때는 접속 직후라 늘 0 이고, 지금 값은 `session.forwards` 로 묻는다, 2026-11-27 D142). 접속에 성공할 때마다 새로 채운다(재접속하면 그때의 결과). 화면은 상태가 연결됨일 때만 쓴다(ui/panes.md §2.6) |
 
 **target 문자열**: SSH/텔넷 `[사용자@]호스트:포트`, 시리얼 `{장치} @ {속도}`(없으면 `? @ 9600`), 로컬 `로컬 · {시작 폴더, 비면 ~}`.
 
```

### `spec/ipc.md` — 수정

```diff
--- a/spec/ipc.md
+++ b/spec/ipc.md
@@ -48,9 +48,10 @@
 ### 세션
 | 채널 | 인자 | 결과 |
 |---|---|---|
-| `session.open` | `{hostId?, adhoc?, cols?, rows?}` (`adhoc`: host, cwd?, port?, username?, protocol?) | 세션 정보 `{id, hostId|null, title, protocol, target, color?, env?, state, connectedAt?, message?, canSftp, keepSession?}`. 서버 id 도 즉석 정보도 없으면 오류 "서버를 찾을 수 없습니다". **연결은 바로 시작하지 않고 약 30ms 뒤** 시작한다(프런트가 터미널을 등록할 시간). **로컬 터미널**: 저장된 로컬 서버는 `hostId` 로 열고(셸·시작 폴더는 서버 설정의 `local` 값), 즉석 로컬은 `adhoc: {host: "", protocol: "local", cwd?: 시작 폴더}` 로 연다 — 즉석 로컬에는 셸 지정이 없고 기본 셸을 쓴다. 즉석 SSH/텔넷은 `{host, port?, username?, protocol?}` |
+| `session.open` | `{hostId?, adhoc?, cols?, rows?}` (`adhoc`: host, cwd?, port?, username?, protocol?) | 세션 정보 `{id, hostId|null, title, protocol, target, color?, env?, state, connectedAt?, message?, canSftp, keepSession?, forwards?}`. 서버 id 도 즉석 정보도 없으면 오류 "서버를 찾을 수 없습니다". **연결은 바로 시작하지 않고 약 30ms 뒤** 시작한다(프런트가 터미널을 등록할 시간). **로컬 터미널**: 저장된 로컬 서버는 `hostId` 로 열고(셸·시작 폴더는 서버 설정의 `local` 값), 즉석 로컬은 `adhoc: {host: "", protocol: "local", cwd?: 시작 폴더}` 로 연다 — 즉석 로컬에는 셸 지정이 없고 기본 셸을 쓴다. 즉석 SSH/텔넷은 `{host, port?, username?, protocol?}` |
 | `session.close` | id | 없음. SFTP 캐시도 정리 |
 | `session.reconnect` | id | 없음 |
+| `session.forwards` | id | (2026-11-27, D142) 지금 연결의 포트 포워딩 배열 — 세션 정보의 `forwards` 와 같은 모양이고 `conns` 가 **묻는 순간의** 연결 수다. 연결이 없거나 포워딩이 없거나 모르는 id 면 빈 배열(오류 아님). 포트 포워딩 정보 창을 **열 때 한 번**, 그리고 그 창의 [새로고침]을 누를 때 묻는다(주기적으로 묻지 않는다) |
 | `session.ack` | id, 처리한 출력 묶음 수(정수) | 없음. 프런트가 `session.data` 묶음을 터미널에 다 쓴 뒤 보낸다(출력 흐름 제어, behavior/session.md §8). 모르는 id·넘치는 수는 무시 |
 | `local.shells` | | `[{path, label}]` (첫 항목이 기본) |
 | `local.tmuxAvailable` | | 불리언 |
```

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -58,8 +58,15 @@
 - 서버의 `forwards` 마다: 바인드 주소가 비면 `127.0.0.1`.
   - **L(로컬)**: 이 PC 의 바인드 주소:포트에서 대기 → 연결이 오면 SSH 연결을 통해 대상 호스트:포트로 중계.
   - **R(원격)**: 서버 쪽에서 바인드 주소:포트로 대기 → 연결이 오면 이 PC 에서 대상 호스트:포트로 중계.
-  - **D(동적)**(2026-11-23, D109): 이 PC 의 바인드 주소:포트를 **SOCKS 프록시**로 연다. 들어온 연결의 SOCKS 요청(SOCKS5 인증 없음·SOCKS4·SOCKS4a, 명령은 CONNECT 만)이 가리키는 대상으로 SSH 연결을 통해 중계한다. 대상 호스트·포트 설정은 쓰지 않는다. 도메인 이름은 이 PC 가 아니라 **서버 쪽에서** 풀린다. SOCKS5 에서 인증 없음을 내놓지 않은 클라이언트·CONNECT 가 아닌 명령·닿지 않는 대상은 SOCKS 오류 응답으로 거절하고, SOCKS 가 아닌 연결은 응답 없이 닫는다. 요청을 10초 안에 다 보내지 않는 연결은 닫는다. 프록시 자체에는 인증이 없으므로 바인드 주소를 `127.0.0.1` 밖으로 열면 그 주소에 닿는 누구나 쓸 수 있다. 성공 시 터미널에 "동적 포워딩(SOCKS 프록시) {바인드}:{포트}", 실패 시 "포트 포워딩 D{포트} 실패: {오류}".
-- 설정 성공 시 터미널에 "포워딩 {바인드}:{포트} → {대상}", 실패 시 "포트 포워딩 {L|R}{포트} 실패: {오류}" 를 보이고 다음 항목으로 계속한다. 세션이 끝나면 모든 대기 소켓을 닫는다.
+  - **D(동적)**(2026-11-23, D109): 이 PC 의 바인드 주소:포트를 **SOCKS 프록시**로 연다. 들어온 연결의 SOCKS 요청(SOCKS5 인증 없음·SOCKS4·SOCKS4a, 명령은 CONNECT 만)이 가리키는 대상으로 SSH 연결을 통해 중계한다. 대상 호스트·포트 설정은 쓰지 않는다. 도메인 이름은 이 PC 가 아니라 **서버 쪽에서** 풀린다. SOCKS5 에서 인증 없음을 내놓지 않은 클라이언트·CONNECT 가 아닌 명령·닿지 않는 대상은 SOCKS 오류 응답으로 거절하고, SOCKS 가 아닌 연결은 응답 없이 닫는다. 요청을 10초 안에 다 보내지 않는 연결은 닫는다. 프록시 자체에는 인증이 없으므로 바인드 주소를 `127.0.0.1` 밖으로 열면 그 주소에 닿는 누구나 쓸 수 있다.
+- 한 항목이 실패해도 다음 항목으로 계속한다. 세션이 끝나면 모든 대기 소켓을 닫는다.
+- **터미널 안내**(2026-11-27, D136 — 전에는 항목마다 줄바꿈 없는 글자를 내보내 한 줄로 붙어 나왔다): 모든 항목을 설정한 뒤 **여러 줄 한 덩어리**로 알린다. 포워딩이 없으면 아무것도 내지 않는다.
+  - 첫 줄: `● 포트 포워딩 {개수}개`(회색). 실패가 있으면 `● 포트 포워딩 {개수}개 (실패 {실패 수}개)`(노랑).
+  - 그 아래 항목마다 한 줄, **빈칸 4개로 들여 쓴다**: `{L|R|D}  {듣는 주소}  →  {대상}`(회색). 듣는 주소는 가장 긴 것에 맞춰 뒤를 빈칸으로 채워 화살표가 세로로 맞는다. D 는 대상 자리에 `SOCKS 프록시`.
+  - 실패한 항목: `{L|R|D}  {듣는 주소}  ✕  실패: {오류}`(빨강).
+  - 줄은 CRLF 로 끝난다(접속 중 안내 줄을 지우는 동작이 이 줄들을 지우지 않는다).
+- **연결 수**(2026-11-27, D142): 포워딩마다 듣는 소켓이 받아들인 연결 중 **아직 열려 있는 것**을 센다(받을 때 +1, 그 연결을 닫을 때 -1. D 는 SOCKS 요청을 읽는 중인 연결도 센다). `session.forwards` 로 알려 준다(spec/ipc.md).
+- 같은 결과를 세션 정보의 `forwards` 에 싣는다(behavior/session.md §1). 화면은 그것으로 창 머리글의 포워딩 버튼과 정보 창을 그린다(ui/panes.md §2.6).
 
 ### 2.7 프록시 경유 (2026-11-23, D110)
 - 서버에 `proxy` 가 있고 **경유 서버(점프 호스트)가 없으면**, 대상까지의 TCP 연결을 프록시를 거쳐 맺는다. 경유 서버가 있으면 그 서버의 `proxy` 는 쓰지 않는다(연결이 경유 서버 위에서 열리기 때문). 대신 **경유 서버 체인의 맨 바깥 서버**(이 PC 가 직접 연결하는 서버)에 `proxy` 가 있으면 그 연결이 프록시를 거친다. 직접 입력한 경유 서버에는 프록시를 둘 수 없다.
```

### `spec/ui/dialogs.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -129,3 +130,17 @@
 - 동기화 결과 토스트: 성공 "동기화했습니다"(다른 PC 변경을 받았으면 "동기화했습니다 (다른 PC 의 변경을 받았습니다)"), 오류·git 없음·잠김은 오류 토스트(메시지 없으면 "동기화하지 못했습니다").
 - 상태 변화 이벤트를 받으면 마지막 상태 표시를 즉시 갱신한다.
 
+## 7. 포트 포워딩 정보 창 (기본 크기, 2026-11-27, D137)
+
+창 머리글의 포트 포워딩 버튼 또는 칸 우클릭 메뉴로 연다(ui/panes.md §2.6). 지금 연결의 포워딩 결과(behavior/session.md §1 `forwards`)를 보인다. 읽기 전용이다.
+
+- 제목: `포트 포워딩 · {세션 제목}`, 좌우 화살표 아이콘.
+- 본문 첫 줄(흐린 작은 글씨): `{접속 대상} · {개수}개 설정됨`, 실패가 있으면 `{개수}개 설정됨, {실패 수}개 실패`. 이어져 있는 연결이 하나라도 있으면 뒤에 ` · 연결 {모든 포워딩의 합}개`.
+- 표(포워딩마다 한 줄, 설정한 순서): 머리 **종류 · 듣는 주소 · 대상 · 상태**.
+  - 종류: 굵은 글자 `L`/`R`/`D` + 흐린 이름(로컬 / 원격 / 동적).
+  - 듣는 주소·대상: 고정폭, 글자를 선택해 복사할 수 있다. D 의 대상은 흐린 글씨 "SOCKS 프록시".
+  - 상태: 성공이면 초록 **"동작 중"**(2026-11-27 사용자 지시, D142 — 처음의 "대기 중"은 아직 준비가 안 된 것으로 읽혔다). 그 포워딩으로 지금 이어져 있는 연결이 있으면 `동작 중 · 연결 {n}개`(뒷부분 굵게), 0 이면 "동작 중"만. 실패면 빨강 `실패: {오류}`(실패한 줄은 주소 글자가 흐리다).
+  - **연결 수 읽기**: 창을 **열 때 한 번** `session.forwards` 를 물어 표를 그린다(빈 배열·오류가 오면 세션 정보의 목록을 그대로 둔다). **주기적으로 다시 묻지 않는다**(2026-11-27 사용자 지시). 다시 보려면 [새로고침]. 창 머리글의 버튼에는 연결 수를 보이지 않는다.
+- 표 아래 도움말: "L: 이 PC의 듣는 주소로 접속하면 서버를 거쳐 대상으로 갑니다. R: 서버의 듣는 주소로 접속하면 이 PC를 거쳐 대상으로 갑니다. D: 듣는 주소가 SOCKS 프록시입니다. 바꾸려면 서버 편집 > 포트 포워딩에서 고친 뒤 다시 연결하세요."
+- 버튼 [새로고침](왼쪽 끝, 연결 수를 다시 읽는다) [복사] [닫기](주 버튼). [복사]는 목록을 글자로 클립보드에 넣고(한 줄에 하나, 듣는 주소 폭을 맞춤 — 터미널 안내의 줄 모양에서 들여쓰기만 뺀 것) 성공 토스트 "포워딩 목록을 복사했습니다".
+
```

### `spec/ui/panes.md` — 수정

```diff
--- a/spec/ui/panes.md
+++ b/spec/ui/panes.md
@@ -26,6 +26,7 @@
   7. **크게 보기/원래대로** — 탭에 칸이 2개 이상일 때만. 툴팁 "최대화 (Ctrl+Shift+Enter)" / "복원 (Ctrl+Shift+Enter)". 아이콘은 두 상태의 **윤곽이 달라야** 한다(2026-11-23 사용자 지시, D121): 평소에는 **빈 사각형 하나**, 크게 보는 중에는 **겹친 사각형 두 개**(Windows 창의 최대화·복원 버튼과 같은 관례). 크게 보는 중에는 버튼을 **켜진 모양**(강조색 옅은 바탕 + 강조색 아이콘, 다른 켜진 아이콘 버튼과 같은 모양)으로 보여 다른 칸이 가려져 있음을 알린다.
   8. **닫기** — 툴팁 "닫기"
 - 머리글은 칸이 2개 이상일 때 **끌어서 옮길 수 있다**(§7).
+- **포트 포워딩 버튼**(2026-11-27, D137): 접속 대상 문자열 바로 오른쪽 — §2.6.
 
 ### 2.2 본문
 - 터미널 화면. 배경색은 현재 터미널 테마의 배경색.
@@ -54,6 +55,7 @@
 | 오른쪽으로 분할 (같은 서버) | Ctrl+Shift+D | |
 | 아래로 분할 (같은 서버) | Ctrl+Shift+E | |
 | SFTP 열기 | Ctrl+Shift+S | SFTP 불가 |
+| 포트 포워딩 보기 ({개수}개) | | 연결된 세션에 설정된 포워딩이 있을 때만 **항목이 보인다**(2026-11-27, D137). 누르면 §2.6 의 정보 창 |
 | 다시 접속 | | 이미 연결됨 |
 | ―구분선― | | |
 | 패널 닫기 (위험색) | | |
@@ -65,6 +67,16 @@
   - 찾기는 현재 항목으로 옮길 때([다음]·[이전]·입력)만 터미널 선택을 바꾼다. 출력이 와도 선택을 다시 바꾸지 않으므로, 막대가 열린 동안 마우스로 한 선택은 그대로 남는다.
 - Esc 또는 닫기: 막대를 닫고 강조를 지우며 터미널에 포커스를 돌려준다(닫기 버튼은 포커스 이동 없음).
 - 여는 방법: 칸 머리글 버튼, 우클릭 메뉴, 단축키(포커스된 칸 대상).
+
+### 2.6 포트 포워딩 표시 (2026-11-27, D137)
+
+원본 앱에 없던 기능이다(사용자 요청: 포워딩이 설정됐는지 보이고, 누르면 목록이 뜨게).
+
+- **언제 보이나**: 세션 상태가 **연결됨**이고 세션 정보의 `forwards`(behavior/session.md §1)가 하나 이상일 때. 끊기면 사라지고, 다시 연결되면 그때의 결과로 다시 보인다.
+- **모양**: 머리글의 접속 대상 문자열 오른쪽에 알약 모양 버튼 — 좌우 화살표 아이콘 + 개수(예 `⇄ 22`). 높이 18px, 글자 11px 굵게. **테두리만 강조색(`--accent`)** 으로 눈에 띄게 하고 글자·바탕은 머리글의 다른 글자와 같게 둔다(2026-11-27 사용자 지시, D143 — 처음의 `--border-strong` 테두리는 머리글에 묻혀 보이지 않았다). **실패한 포워딩이 있으면** 테두리와 글자가 경고색(`--warn`).
+- 툴팁: `포트 포워딩 {개수}개 — 눌러서 보기`, 실패가 있으면 `포트 포워딩 {개수}개 (실패 {실패 수}개) — 눌러서 보기`.
+- 누르면 **포트 포워딩 정보 창**(ui/dialogs.md §7)을 연다. 칸 우클릭 메뉴의 "포트 포워딩 보기 ({개수}개)"도 같은 창을 연다.
+- 터미널 안에도 접속 직후 같은 내용이 여러 줄로 나온다(protocols.md §2.5).
 
 ### 2.5 파일 끌어 놓기
 - 칸 위로 **파일**을 끌어오면 칸 전체에 안내 오버레이가 뜬다. 문구: SFTP 가능 → "서버({업로드 폴더 설정값})로 업로드하고 경로를 입력합니다", 로컬 → "파일 경로를 입력합니다 (이미지는 저장 후 경로)", 그 외 → "파일 경로를 입력합니다". 칸 밖으로 나가면 사라진다.
```

## G. 검증 (2026-11-27 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [forwards_test.go](../../internal/session/forwards_test.go) `TestMsgForwardsLines` | 첫 줄 `● 포트 포워딩 3개`, 항목마다 빈칸 4개 들여쓰기, **짧은 주소 뒤를 빈칸으로 채워 화살표가 세로로 맞는지**, D 의 대상 자리 `SOCKS 프록시`, 줄마다 색을 되돌리는지 |
| 같은 파일 `TestMsgForwardsFailed` | 실패가 있으면 첫 줄이 노랑 `(실패 1개)`, 실패 줄은 빨강 `✕  실패: {오류}` |
| 같은 파일 `TestMsgForwardsEmpty` | 포워딩이 없으면 아무것도 내지 않는다 |
| 같은 파일 `TestSessionInfoCarriesForwards` | 연결이 `ForwardReporter` 면 접속 성공 때 세션 정보의 `forwards` 에 실린다 |
| [forward_count_test.go](../../internal/sshconn/forward_count_test.go) `TestForwardCountsOpenConnections` (D142) | 접속 2개 → 2, 데이터가 오가는 동안 유지, 하나 닫으면 1, 다 닫으면 0 |
| 같은 파일 `TestForwardCountDropsFailedDial` (D142) | 대상에 닿지 못해 곧바로 닫힌 연결도 수에서 빠진다 |
| 같은 파일 `TestForwardResultActiveWithoutCounter` (D142) | 실패한 포워딩의 연결 수는 0 |
| [forwards.test.ts](../../frontend/src/lib/forwards.test.ts) 2개 | 실패 수 세기, 복사용 글자(한 줄에 하나, 폭 맞춤) |
| [Automation.test.tsx](../../frontend/src/screens/Automation.test.tsx) "포트 포워딩 정보 창" 2개 | 표의 줄 수·종류·주소·상태("동작 중"/실패) / **열 때 한 번만 읽고 5초가 지나도 다시 묻지 않으며, [새로고침]을 누르면 다시 읽는다**(D142) |
| 기존 `internal/sshconn/socks_test.go` | SOCKS 처리 코드를 고치지 않았으므로 그대로 통과(듣는 소켓을 감싸는 방식의 회귀 확인) |

**자동 테스트가 확인하지 못하는 것**: 실제 SSH 서버에 붙어 R(원격) 포워딩의 연결 수가 맞게 세어지는지(테스트는 L 과 같은 중계 코드를 일반 TCP 로 돌린다), 머리글 버튼이 실제 화면에서 어떻게 보이는지(색·위치·좁은 창).

전체 실행 결과(2026-11-27, 다섯 문서와 추가 지시 D142·D143 을 모두 적용한 상태):

```bash
go vet ./...                              # 경고 없음
go test ./...                             # 19개 패키지 모두 ok
(cd frontend && npx tsc --noEmit)         # 오류 없음
(cd frontend && npm test)                 # PowerShell 에서. 61개 파일 439개 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

**`-race` 검사 (같은 날 추가)**: 처음에는 이 PC 에 gcc 가 없어 돌리지 못했다(`-race requires cgo`). 같은 날 gcc 를 설치하고(D144, [2026-11-27-6-gcc-race-test.md](./2026-11-27-6-gcc-race-test.md)) `go test -race ./...` 를 돌려 19개 패키지 모두 통과했다 — 경쟁 보고 없음. 연결 수를 세는 `TestForwardCountsOpenConnections` 등도 따로 돌려 통과(그 문서 §6.2).

```
go test ./internal/session -run Forwards     # 줄 모양·폭 맞춤·실패 줄·빈 목록·세션 정보에 실림
go test ./internal/sshconn -run Forward      # 연결 수 세기 (D142)
(frontend) npm test                          # forwards.test.ts, Automation.test.tsx "포트 포워딩 정보 창"
```

2026-11-27 결과: `go test ./...` 전부 통과, 프런트 437개 통과(D142·D143 뒤 439개), `wails build` 성공. `go test -race ./...` 는 처음에 gcc 가 없어 돌리지 못했고, 같은 날 gcc 설치 뒤(D144) 통과했다.

## H. 수동 검증 (앱에서)

| # | 할 일 | 기대 |
|---|---|---|
| 1 | 포워딩이 여러 개인 SSH 서버(사진의 `삼성_gw_tunneling`)에 연결 | `● 포트 포워딩 N개` 아래에 포워딩마다 한 줄, 화살표가 세로로 맞는다 |
| 2 | 창 머리글의 접속 대상 옆 `⇄ N` 버튼을 누름 | 정보 창에 N줄, 상태가 모두 "동작 중" |
| 2a | 다른 터미널에서 포워딩 포트로 접속해 둔 뒤(예: `ssh -p 25095 mds@127.0.0.1`) 창을 엶 | 그 줄이 `동작 중 · 연결 1개`, 첫 줄 끝에 `· 연결 1개`. 접속을 끊고 [새로고침]을 누르면 다시 `동작 중` |
| 3 | 정보 창 [복사] 뒤 메모장에 붙여넣기 | 한 줄에 하나, 폭이 맞는다 |
| 4 | 다른 프로그램이 쓰는 포트를 듣는 포워딩을 하나 넣고 다시 연결 | 첫 줄 노랑 `(실패 1개)`, 그 줄 빨강, 버튼 경고색. 셸은 그대로 쓴다 |
| 5 | 연결을 끊음(서버에서 `exit`) | 버튼이 사라지고, 다시 연결하면 다시 보인다 |
| 6 | 화면을 좌우로 나누고 창을 좁힘 | 버튼이 줄어들지 않고 접속 대상 글자가 먼저 잘린다 |
| 6a | 머리글의 버튼을 눈으로 봄(다크·라이트 테마 둘 다) | 테두리가 파란 강조색이라 머리글에서 눈에 띈다. 글자색은 옆의 글자와 같다 (D143) |
| 6b | 정보 창을 연 채 포워딩 접속을 끊고 기다림 → [새로고침] | 기다리는 동안은 그대로, 누르면 `동작 중` 으로 바뀐다 (D142) |
| 7 | 포워딩이 없는 서버·텔넷·로컬 터미널 | 안내도 버튼도 없다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 포워딩이 없는 서버·SSH 가 아닌 세션 | 세션 정보에 `forwards` 가 없다(싣지 않는다) | 화면은 `info.forwards ?? []` 로 읽는다. `TestMsgForwardsEmpty` |
| 터미널 안내 문구가 달라짐 | 옛 문구(`포워딩 A → B`)를 화면에서 찾던 사용자 스크립트·세션 로그 검색이 있으면 맞지 않는다 | 줄마다 나오므로 `grep` 하기는 더 쉽다. 변경 이력에 적었다 |
| `StartForwards` 의 모양 변경(안내 함수 인자를 뺌) | 부르는 곳은 `internal/app/ssh_app.go` 한 곳 | `go build ./...` |
| 접속 중 안내 줄 지우기 | 포워딩 안내가 CRLF 로 끝나므로 접속이 끝날 때의 줄 지우기가 빈 줄만 지운다(전에는 안내의 끝이 지워졌다) | `TestMsgForwardsLines` 가 줄 끝을 본다 |
| 연결 수 세기(D142) | 듣는 소켓을 감싼 것이 `Accept` 한 연결을 닫지 않고 흘리면 수가 줄지 않는다 | 중계 코드의 모든 길(`acceptLoop` 의 대상 연결 실패·정상 끝, `handleSocks` 의 `defer Close`)이 받은 연결을 닫는다. `Close` 가 여러 번 불려도 한 번만 줄인다(`sync.Once`) |
| 새 채널 `session.forwards`(D142) | 닫힌 세션·모르는 id 로 물을 수 있다 | 오류 대신 빈 목록. 화면은 빈 목록이면 세션 정보의 목록을 그대로 둔다 |
| 머리글이 좁을 때 | 버튼(`flex: none`)은 줄지 않고 접속 대상 글자가 먼저 잘린다 | 수동 검증 6 |
| 정보 창이 칸(`.pane`) 안에 그려짐 | 창 안의 클릭·우클릭이 칸으로 번져 칸 메뉴가 뜨거나 끌기가 시작될 수 있다 | 창을 감싼 요소에서 `mousedown`·`contextmenu` 가 번지지 않게 막았다. 열리면 [닫기]에 포커스를 둬 키 입력이 터미널로 가지 않는다 |
| 저장 데이터 | 바뀌지 않는다(포워딩 설정은 전과 같은 `forwards`) | — |

## J. 체크리스트

- [x] `sshconn.StartForwards` 가 안내 대신 결과(`ForwardResult`)를 돌려준다
- [x] `session.MsgForwards`(줄마다, 폭 맞춤, 실패 줄), 세션 정보 `forwards`, `ForwardReporter`
- [x] 창 머리글의 `⇄ N` 버튼(실패가 있으면 경고색), 칸 우클릭 메뉴 "포트 포워딩 보기", 정보 창(표·[복사])
- [x] (D142) 상태 문구 "동작 중", 포워딩별 연결 수 — 접속이 들어올 때·닫힐 때만 세고(`countingListener`), 창을 열 때 한 번 읽는다(`session.forwards`). [새로고침]
- [x] (D143) 머리글 버튼의 테두리만 강조색
- [x] 문구: 옛 키 5개 삭제, `be.fwd.*` 4개·`fwdinfo.*` 19개(10개 언어)
- [x] 명세 protocols.md §2.5 · behavior/session.md §1 · ipc.md · ui/panes.md §2.6 · ui/dialogs.md §7 · 수용 P16~P18·P17a, decisions D136·D137·D142·D143
- [x] 자동 테스트 11개 추가(Go 7, 프런트 4), 전체 통과
- [ ] 수동 검증 1~7, 2a, 6a, 6b (실제 서버·화면)
- [x] `go test -race` — 2026-11-27 gcc 설치 뒤 통과 (D144, [-6-](./2026-11-27-6-gcc-race-test.md))
