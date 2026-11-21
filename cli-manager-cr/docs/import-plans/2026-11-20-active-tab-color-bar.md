# 탭을 선택해도 위쪽 색 막대가 그 탭으로 옮겨 가지 않는 문제 — 원인·수정·검증

**작성**: 2026-11-20 · **상태**: 코드 수정 완료(2026-11-20, D103), 자동 테스트 통과, 앱 수동 검증은 남음(§7.2) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/ui/shell.md](../../spec/ui/shell.md) §2.1(탭), [spec/ui/tokens.md](../../spec/ui/tokens.md) §3(탭), [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md) S4a, [frontend/src/screens/TabBar.tsx](../../frontend/src/screens/TabBar.tsx), [frontend/src/screens/TabBar.mark.test.tsx](../../frontend/src/screens/TabBar.mark.test.tsx), [frontend/src/styles/app.css](../../frontend/src/styles/app.css), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
>
> **이 문서는 두 부분이다.** 1부(§0~§9) = 탭 색 막대를 선택된 탭에만 표시(D103). **2부(§B0~§B9, 문서 뒤쪽) = 세션 색·환경을 폴더에서 물려받기(D105)** — 같은 날 이어서 보고된 "폴더가 보라색인데 터미널이 파랑으로 표시된다" 문제.

---

## 0. 한눈에 보기

| 항목 | 내용 |
|---|---|
| 증상 | 탭을 눌러 선택해도 탭 위쪽의 색 막대(스크린샷의 빨강·초록)가 선택한 탭으로 옮겨 가지 않는다. 막대는 다른 탭에 그대로 붙어 있고, 선택한 탭은 거의 구분되지 않는다 |
| 원인 | 색 막대는 선택 표시가 아니었다. **서버(또는 폴더·환경)에 색이 지정된 세션의 탭**이면 선택과 무관하게 항상 그렸고, 색이 없는 서버의 탭은 선택해도 그리지 않았다. 탭 선택 자체는 정상 동작한다 |
| 수정 | 색 막대를 **그룹에서 선택된 탭에만** 그린다. 색은 그 탭 세션의 색, 없으면 강조색(파랑). 선택되지 않은 탭은 세션 색의 옅은 배경만 남긴다 (D103) |
| 바뀐 파일 | `frontend/src/screens/TabBar.tsx`(수정), `frontend/src/screens/TabBar.mark.test.tsx`(새로 만듦) |
| 검증 | `tsc --noEmit` 통과, 프런트 테스트 40개 파일 257개 통과(새 테스트 8개 포함), `build/bin/dariterm.exe` 빌드 완료(2026-11-20 14:14). **실제 앱에서 눌러 보는 확인은 아직 하지 않았다**(§7.2) |
| 명세 | 사용자 결정이라 [ui/shell.md](../../spec/ui/shell.md) §2.1 의 "색" 항목과 [ui/tokens.md](../../spec/ui/tokens.md) 의 탭 행을 고치고 수용 S4a 를 추가했다 |

---

## 1. 발견 경위

2026-11-20 사용자 보고(스크린샷 첨부): "탭을 선택하면 패널의 윗쪽에 선택한 탭의 왼쪽의 빨강색과 오른쪽의 초록색처럼 바뀌어야 하는데, 바뀌지 않는다."

스크린샷은 화면이 좌우로 나뉜 상태다.

| 그룹 | 탭 | 위쪽 색 막대가 있는 탭 | 실제로 선택된 탭(아래 칸에 보이는 세션) |
|---|---|---|---|
| 왼쪽 | 로컬 터…, 삼성_gw…, mds-컴…, loading-t…, 25097-d… | **mds-컴…** (빨강) | **loading-t…** — 칸 머리글 `loading-tester1(200)` |
| 오른쪽 | DC1_MSA01…, 14172-dc4-ms…, 25095-dc1-ms…, 25097-dc1-ms… | **DC1_MSA01…** (초록) | **14172-dc4-ms…** — 칸 머리글 `14172-dc4-msa001` |

그룹마다 막대가 정확히 하나씩 있어 "선택된 탭 표시"처럼 보이지만, 두 그룹 모두 막대가 있는 탭과 선택된 탭이 다르다. 선택된 탭은 닫기(X) 버튼이 보이는 것 말고는 다른 탭과 거의 같다.

---

## 2. 색 막대의 구조 (수정 전)

탭 하나를 그리는 코드는 [TabBar.tsx](../../frontend/src/screens/TabBar.tsx) 의 `tabs.map(...)` 안에 있다. 수정 전:

```tsx
const active = t.id === activeId;                                                   // 이 그룹에서 선택된 탭인가
const focusedSid = t.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId : null;
const env = focusedSid ? sessions[focusedSid]?.info.color : undefined;              // 포커스 칸 세션의 색
…
<div
  role="tab"
  className={`tab${active ? ' active' : ''}…`}
  …
  style={env ? { background: `color-mix(in srgb, ${env} ${active ? 14 : 7}%, transparent)` } : undefined}
>
  {env && <span className="bar" style={{ background: env }} />}                     // ← 색 막대
  <span className={`dot ${dot}`} />
  <span className="title">{title}</span>
  …
```

막대의 모양은 [app.css:204](../../frontend/src/styles/app.css#L204) — 탭 위쪽 가장자리의 3px 띠다.

```css
.tab .bar { position: absolute; top: 0; left: 0; right: 0; height: 3px; border-radius: 8px 8px 0 0; }
```

세션의 색(`info.color`)은 백엔드가 정한다(spec/behavior/session.md): 서버 색 → 환경 색 → 폴더 색/환경 색 순, 어느 것도 없으면 없음.

---

## 3. 원인

막대를 그리는 조건이 `env`(세션에 색이 있는가) 하나뿐이고, **`active`(선택됐는가)는 조건에 없다.** `active` 는 배경을 섞는 비율(14% / 7%)에만 쓰였다.

| 탭 | 세션 색 | 선택 | 수정 전 표시 |
|---|---|---|---|
| mds-컴… | 빨강 | 아님 | 빨강 막대 + 빨강 7% 배경 → **선택된 것처럼 보임** |
| loading-t… | 없음 | **선택됨** | 막대 없음, 배경 색칠 없음 → **선택된 것처럼 안 보임** |
| DC1_MSA01… | 초록 | 아님 | 초록 막대 + 초록 7% 배경 |
| 14172-dc4-ms… | 없음 | **선택됨** | 막대 없음 |

즉 탭 선택은 정상이다(`activateTab` → `activeByGroup` 갱신 → 아래 칸이 바뀜, 스크린샷에서도 칸은 선택한 탭의 것이다). 문제는 **막대가 선택을 따라가는 표시가 아니었다**는 것이다.

### 3.1 선택된 탭이 유난히 안 보인 이유

선택된 탭의 기본 표시는 [app.css:203](../../frontend/src/styles/app.css#L203) `.tab.active { background: var(--panel); border-color: var(--border); … }` 이다. 그런데 화면을 나눴을 때의 탭 막대도 [app.css:228](../../frontend/src/styles/app.css#L228) `.tabbar.in-group { … background: var(--panel); … }` 로 **같은 색**이라, 선택된 탭의 배경이 탭 막대에 묻힌다. 남는 차이는 1px 테두리와 글자색·X 버튼뿐이다. 나뉘지 않은 화면에서는 탭 막대 배경이 달라 이 문제가 덜하다.

### 3.2 명세와의 관계

수정 전 코드는 당시 명세([ui/shell.md](../../spec/ui/shell.md) §2.1 "포커스된 칸 세션에 색이 있으면 … 그 색의 막대를 두고")대로였다. 구현 오류가 아니라 **명세의 표시 방식이 쓰는 사람의 기대와 달랐던 것**이라, 사용자 결정으로 명세를 바꾼다(§4, §8).

---

## 4. 설계 결정 (D103)

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 막대를 선택된 탭에만 그린다. 색은 세션 색, 없으면 강조색** | 그룹마다 막대가 정확히 하나라 "막대 = 선택된 탭"이 된다(사용자가 스크린샷에서 읽은 뜻 그대로). 서버 색은 선택된 탭의 막대 색과, 선택되지 않은 탭의 옅은 배경으로 계속 보인다 | **채택** |
| B. 색 있는 탭의 막대는 그대로 두고, 선택된 탭에 강조색 막대를 더한다 | 스크린샷의 왼쪽 그룹이라면 빨강(mds-컴)과 파랑(loading-t) 막대가 함께 보인다. 어느 것이 선택인지 다시 헷갈린다 | 기각 |
| C. 막대는 그대로 두고 선택된 탭의 배경·테두리만 진하게 한다 | 사용자가 요청한 것은 "빨강·초록처럼" 위쪽이 바뀌는 것이다. 막대가 선택과 무관하게 남아 오해가 그대로다 | 기각 |

세부 결정:

1. **색이 없는 세션의 선택된 탭은 강조색(`--accent`)**. 칸의 포커스 테두리([app.css:304](../../frontend/src/styles/app.css#L304))와 같은 색이라 "지금 보는 것"의 표시가 탭과 칸에서 일치한다.
2. **선택된 탭은 배경도 14% 로 칠한다**(색이 없으면 강조색 14%). §3.1 의 "탭 막대에 묻힘"이 함께 풀린다. 비율은 수정 전 값 그대로다.
3. **선택되지 않은 탭은 세션 색의 7% 배경만 남긴다**(수정 전 값 그대로, 막대만 뺌). 어느 탭이 어느 서버 색인지는 계속 알 수 있다. 색이 없으면 아무 표시도 없다.
4. **판단을 작은 순수 함수(`tabMark`)로 뺀다**. 조건 네 가지(선택 × 색 유무)를 화면 없이 테스트하려는 것이다.
5. CSS 는 바꾸지 않는다. 막대의 크기·위치는 그대로다.

---

## 5. 수정 내용

### 5.1 `frontend/src/screens/TabBar.tsx`

새로 넣은 상수와 함수(`TAB_MIME`·`tabDot` 아래, `TabBar` 위):

```tsx
/** 선택되지 않은 탭의 배경에 섞는 세션 색 비율(%)과 선택된 탭의 비율 (shell.md §2.1). */
const TINT_INACTIVE = 7;
const TINT_ACTIVE = 14;

/**
 * 탭의 색 표시(shell.md §2.1, D103).
 * 색 막대(bar)는 그룹에서 선택된 탭에만 둔다: 세션 색, 없으면 강조색.
 * 선택되지 않은 탭은 세션 색이 있을 때 옅은 배경(tint)만 가진다.
 */
export function tabMark(active: boolean, color: string | undefined): { bar?: string; tint?: string } {
  const c = color || (active ? 'var(--accent)' : undefined);
  if (!c) return {};
  return {
    bar: active ? c : undefined,
    tint: `color-mix(in srgb, ${c} ${active ? TINT_ACTIVE : TINT_INACTIVE}%, transparent)`,
  };
}
```

탭을 그리는 곳:

```tsx
// before
const active = t.id === activeId;
const focusedSid = t.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId : null;
const env = focusedSid ? sessions[focusedSid]?.info.color : undefined;
…
  style={env ? { background: `color-mix(in srgb, ${env} ${active ? 14 : 7}%, transparent)` } : undefined}
>
  {env && <span className="bar" style={{ background: env }} />}

// after
const active = t.id === activeId;
const focusedSid = t.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId : null;
const mark = tabMark(active, focusedSid ? sessions[focusedSid]?.info.color : undefined);
…
  style={mark.tint ? { background: mark.tint } : undefined}
>
  {mark.bar && <span className="bar" style={{ background: mark.bar }} />}
```

읽는 법:

- `activeId` 는 `activeByGroup[g]`(이 탭 막대가 맡은 그룹의 선택 탭)이다. 그래서 화면을 나누면 **그룹마다 하나씩** 막대가 생긴다. 포커스가 없는 쪽 그룹의 선택 탭에도 막대가 있다(그 그룹에서 지금 보이는 탭이므로).
- `color || …` — 빈 문자열도 "색 없음"으로 본다.
- 탭 선택(`onMouseDown` → `activate`)과 그 밖의 처리기는 건드리지 않았다.

### 5.2 수정 뒤 표시 (§3 의 표와 대응)

| 탭 | 세션 색 | 선택 | 수정 뒤 표시 |
|---|---|---|---|
| mds-컴… | 빨강 | 아님 | 막대 없음, 빨강 7% 배경 |
| loading-t… | 없음 | **선택됨** | **강조색(파랑) 막대** + 강조색 14% 배경 |
| DC1_MSA01… | 초록 | 아님 | 막대 없음, 초록 7% 배경 |
| 14172-dc4-ms… | 없음 | **선택됨** | **강조색 막대** + 강조색 14% 배경 |
| (mds-컴… 을 누르면) | 빨강 | **선택됨** | **빨강 막대** + 빨강 14% 배경, loading-t… 의 막대는 사라짐 |

### 5.3 새 파일 — `frontend/src/screens/TabBar.mark.test.tsx` (전체)

```tsx
// 탭의 색 막대는 그룹에서 선택된 탭에만 있다 (D103, spec/ui/shell.md §2.1, 수용 S4a).
import { beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { useWorkspace, type Tab } from '../state/workspace';
import type { SessionInfo } from '../ipc/types';
import { TabBar, tabMark } from './TabBar';

function tab(id: string, sessionId: string, group?: 1): Tab {
  return { id, broadcast: false, group, root: { type: 'pane', id: `p-${id}`, sessionId }, focusedPane: `p-${id}`, zoomed: null };
}

function session(id: string, title: string, color?: string) {
  return { info: { id, title, state: 'connected', color } as unknown as SessionInfo, activity: false };
}

/** 색 막대가 있는 탭의 제목들. */
function barred(): string[] {
  return screen
    .getAllByRole('tab')
    .filter((el) => el.querySelector('.bar'))
    .map((el) => el.getAttribute('title') ?? '');
}

function press(title: string) {
  fireEvent.mouseDown(screen.getByTitle(title), { button: 0 });
}

beforeEach(() => {
  useWorkspace.setState({
    tabs: [tab('a', 's-a'), tab('b', 's-b'), tab('c', 's-c')],
    activeTabId: 'a',
    activeByGroup: ['a', null],
    sessions: { 's-a': session('s-a', 'red', '#ef4444'), 's-b': session('s-b', 'plain'), 's-c': session('s-c', 'green', '#22c55e') },
  });
});

describe('tabMark', () => {
  it('선택된 탭: 세션 색의 막대와 14% 배경', () => {
    expect(tabMark(true, '#ef4444')).toEqual({ bar: '#ef4444', tint: 'color-mix(in srgb, #ef4444 14%, transparent)' });
  });
  it('선택된 탭에 세션 색이 없으면 강조색을 쓴다', () => {
    expect(tabMark(true, undefined)).toEqual({ bar: 'var(--accent)', tint: 'color-mix(in srgb, var(--accent) 14%, transparent)' });
    expect(tabMark(true, '').bar).toBe('var(--accent)');
  });
  it('선택되지 않은 탭: 막대 없이 세션 색의 7% 배경만', () => {
    expect(tabMark(false, '#ef4444')).toEqual({ bar: undefined, tint: 'color-mix(in srgb, #ef4444 7%, transparent)' });
  });
  it('선택되지 않았고 세션 색도 없으면 아무 표시도 없다', () => {
    expect(tabMark(false, undefined)).toEqual({});
  });
});

describe('탭 색 막대 (D103)', () => {
  it('선택된 탭에만 색 막대가 있다 (색이 있는 다른 탭에는 없다)', () => {
    render(<TabBar />);
    expect(barred()).toEqual(['red']);
  });

  it('세션 색이 없는 탭을 선택하면 막대가 그 탭으로 옮겨 간다', () => {
    render(<TabBar />);
    press('plain');
    expect(barred()).toEqual(['plain']);
  });

  it('다른 탭을 선택할 때마다 막대가 따라간다', () => {
    render(<TabBar />);
    press('green');
    expect(barred()).toEqual(['green']);
    press('red');
    expect(barred()).toEqual(['red']);
  });

  it('화면을 나누면 그룹마다 선택된 탭 하나씩에 막대가 있다', () => {
    useWorkspace.setState({
      tabs: [tab('a', 's-a'), tab('b', 's-b'), tab('c', 's-c', 1)],
      activeTabId: 'b',
      activeByGroup: ['b', 'c'],
    });
    render(
      <>
        <TabBar group={0} />
        <TabBar group={1} />
      </>,
    );
    expect(barred()).toEqual(['plain', 'green']);
    press('red');
    expect(barred()).toEqual(['red', 'green']);
  });
});
```

---

## 6. 영향 범위와 회귀 위험

| 영역 | 영향 |
|---|---|
| 탭 선택·닫기·이름 바꾸기·끌기([TabBar.tsx](../../frontend/src/screens/TabBar.tsx)) | 처리기를 건드리지 않았다. 바뀐 것은 탭의 인라인 배경과 막대 요소의 유무뿐이다 |
| 색이 있는 서버의 선택되지 않은 탭 | **막대가 없어진다**(의도한 변경). 옅은 배경(7%)은 그대로라 색 구분은 남지만 전보다 약하다 |
| 색이 없는 서버의 선택된 탭 | 강조색 막대와 강조색 14% 배경이 **새로 생긴다**(의도한 변경) |
| 칸 머리글·칸 테두리([Panes.tsx](../../frontend/src/screens/Panes.tsx)), 사이드바·시작 화면의 색 막대 | 바뀌지 않았다 |
| 파일 끌기 중 탭 강조(`.tab.file-over`), 탭 끌기 중 점선(`.tab.drop`) | 클래스·CSS 그대로 |
| 위험 | 선택되지 않은 색 탭이 7% 배경만으로는 눈에 덜 띌 수 있다 → §7.2 의 4번으로 확인, 약하면 `TINT_INACTIVE` 를 올린다 |

백엔드(Go), 데이터 파일, 문구(`locales/`), CSS 는 바뀌지 않았다.

---

## 7. 검증

### 7.1 자동 (2026-11-20 실행)

| 명령 (`frontend/` 에서, PowerShell) | 결과 |
|---|---|
| `npx tsc --noEmit` | 통과 (exit 0) |
| `npx vitest run` | 40개 파일, 257개 테스트 통과 |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` (`cli-manager-cr/` 에서) | `build/bin/dariterm.exe` 생성 (2026-11-20 14:14) |

새 테스트 [TabBar.mark.test.tsx](../../frontend/src/screens/TabBar.mark.test.tsx) 8개 — 탭 3개(빨강 색 서버 `red`, 색 없는 서버 `plain`, 초록 색 서버 `green`)를 놓고 확인한다.

| 테스트 | 확인하는 것 | 대응 |
|---|---|---|
| tabMark: 선택된 탭 — 세션 색의 막대와 14% 배경 | 색 있는 탭을 선택 | §4 결정 2 |
| tabMark: 선택된 탭에 세션 색이 없으면 강조색 | `undefined`·빈 문자열 | §4 결정 1 |
| tabMark: 선택되지 않은 탭 — 막대 없이 7% 배경만 | | §4 결정 3 |
| tabMark: 선택되지 않았고 색도 없으면 표시 없음 | | §4 결정 3 |
| 선택된 탭에만 색 막대가 있다 | 선택되지 않은 `green` 에는 막대가 없다(수정 전에는 있었다) | §3 |
| 세션 색이 없는 탭을 선택하면 막대가 옮겨 간다 | `plain` 을 누름 — **보고된 증상** | §1, S4a |
| 다른 탭을 선택할 때마다 막대가 따라간다 | `green` → `red` | S4a |
| 화면을 나누면 그룹마다 하나씩 | 왼쪽 `plain`·오른쪽 `green`, 왼쪽에서 `red` 를 눌러도 오른쪽은 그대로 | 스크린샷의 상태 |

수정 전 코드에 같은 테스트를 돌려 실패하는지는 확인하지 않았다(화면 테스트 4개는 수정 전 조건 `env && …` 로는 `['red', 'green']` 이 나와 실패할 것으로 본다 — 코드로 판단한 것이고 실행해 보지는 않았다).

### 7.2 수동 (남음) — `build/bin/dariterm.exe` (2026-11-20 14:14 빌드)

준비: 색이 있는 서버 탭과 색이 없는 서버 탭을 섞어 4개 이상 열고, 화면을 좌우로 나눈다(스크린샷의 상태).

| # | 조작 | 기대 |
|---|---|---|
| 1 | 그대로 본다 | 그룹마다 **선택된 탭 하나에만** 위쪽 색 막대가 있다 |
| 2 | 색이 없는 서버의 탭을 누른다 | 막대가 그 탭으로 옮겨 가고 파란색(강조색)이다. 전에 선택됐던 탭의 막대는 사라진다 |
| 3 | 색이 있는 서버의 탭을 누른다 | 그 서버 색의 막대가 그 탭에 생긴다 |
| 4 | 선택되지 않은 색 있는 탭을 본다 | 막대는 없고 옅은 배경색만 있다. 어느 서버 색인지 알아볼 수 있는지 본다 |
| 5 | 반대쪽 그룹의 탭을 누른다 | 그 그룹의 막대만 옮겨 간다. 다른 그룹의 막대는 그대로 |
| 6 | 화면 나누기를 풀고(탭 우클릭 > 합치기) 탭을 눌러 본다 | 막대가 하나만 있고 선택을 따라간다 |
| 7 | Ctrl+Tab 등 단축키로 탭을 바꾼다 | 막대가 따라간다 |
| 8 | 칸이 여러 개인 탭에서 색이 다른 칸을 누른다(포커스 칸 변경) | 그 탭의 막대 색이 포커스 칸 세션의 색으로 바뀐다 |
| 9 | 밝은 테마로 바꿔 1~4 를 본다 | 막대와 배경이 보인다 |

---

## 8. 명세

사용자 결정이라 명세에 반영했다.

- [spec/ui/shell.md](../../spec/ui/shell.md) §2.1 "색": 막대는 선택된 탭에만(세션 색, 없으면 강조색, 배경 14%), 선택되지 않은 탭은 세션 색의 7% 배경만, 나뉜 화면에서는 그룹마다 하나. 탭 구성 요소 나열에서 "(색 막대, 선택)"을 뺐다(막대는 왼쪽 요소가 아니라 위쪽 가장자리의 띠다 — 이전 문장 "탭 왼쪽에 … 막대"는 tokens.md 의 "탭 위쪽"과도 어긋나 있었다).
- [spec/ui/tokens.md](../../spec/ui/tokens.md) 탭 행: "색 막대 높이 3(탭 위쪽, 선택된 탭에만)".
- [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md): S4a 추가.
- [docs/decisions.md](../decisions.md): D103.

---

## 9. 남은 일

- §7.2 수동 검증.
- 선택되지 않은 색 탭의 표시가 약하면 `TabBar.tsx` 의 `TINT_INACTIVE`(지금 7)를 올린다. 선택된 탭의 배경이 과하면 `TINT_ACTIVE`(지금 14)를 내린다.

---
---

# 2부 — 폴더에 색을 지정해도 터미널 칸 테두리가 파랑으로 나오는 문제 (세션 색·환경을 폴더에서 물려받기, D105)

**작성**: 2026-11-20 · **상태**: 코드 수정 완료(2026-11-20, D105), 자동 테스트 통과, 앱 수동 검증은 남음(§B7.2)
**관련**: [spec/behavior/session.md](../../spec/behavior/session.md) §1(color·env), [spec/ui/sidebar.md](../../spec/ui/sidebar.md) "색 규칙", [spec/data/store.md](../../spec/data/store.md) §6, [internal/app/host_look.go](../../internal/app/host_look.go), [internal/app/channels_session.go](../../internal/app/channels_session.go), [internal/app/host_look_test.go](../../internal/app/host_look_test.go), [frontend/src/lib/hosts.ts](../../frontend/src/lib/hosts.ts), [frontend/src/lib/hosts.test.ts](../../frontend/src/lib/hosts.test.ts), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 1부(D103)는 탭의 색 막대를 **어느 탭에** 그리는가를 고쳤다. 2부는 그 막대와 칸 테두리가 쓰는 **세션 색 자체가 어떻게 정해지는가**를 고친다. 1부 §2 의 "세션의 색(`info.color`)은 백엔드가 정한다: 서버 색 → 환경 색 → 폴더 색/환경 색 순"은 명세의 규칙이고, 2부 수정 전에는 백엔드가 이 가운데 서버 색만 구현하고 있었다.
> 2부의 절 번호는 1부와 겹치지 않게 `B` 를 붙인다(§B0 ~ §B9).

---

## B0. 한눈에 보기

| 항목 | 내용 |
|---|---|
| 증상 | 폴더 `mds-삼성` 에 보라색을 지정했고 그 안의 서버 `삼성_gw_tunneling` 은 색·환경을 지정하지 않았다. 사이드바의 색 막대는 보라인데, 그 서버를 연 **터미널 칸의 테두리는 파랑(강조색)** 이다 |
| 원인 | 세션을 열 때 백엔드가 **서버 자신의 색·환경만** 세션 정보에 넣고 폴더를 보지 않았다. 사이드바는 프런트에서 따로 폴더까지 따라 올라가 계산해, 같은 서버의 색이 두 곳에서 달랐다 |
| 함께 있던 문제 | ① **환경도 물려받지 않았다** — 폴더만 "운영"이고 서버에 환경이 없으면 그 세션은 운영으로 취급되지 않아 빨간 테두리·"운영" 배지·위험 명령 확인이 모두 빠졌다. ② 프런트의 색 순서(서버 색 → 폴더 색 → 환경색)가 명세(서버 색 → 서버 환경색 → 폴더)와 달랐다 |
| 수정 | 백엔드에 `resolveLook` 을 넣어 세션의 색·환경을 명세대로 정한다. 프런트 `effectiveColor` 도 같은 순서로 맞춘다 (D105) |
| 바뀐 파일 | `internal/app/host_look.go`·`host_look_test.go`(새로 만듦), `internal/app/channels_session.go`, `frontend/src/lib/hosts.ts`, `frontend/src/lib/hosts.test.ts`(수정) |
| 검증 | `go build ./...`, `go test ./...` 통과. `tsc --noEmit` 통과, 프런트 테스트 40개 파일 259개 통과. 수정 전 동작으로 되돌리면 새 Go 테스트가 실패하는 것을 확인. `build/bin/dariterm.exe` 빌드(2026-11-20 15:14). **실제 앱에서 보는 확인은 아직 하지 않았다**(§B7.2) |
| 명세 | 바꾸지 않았다. 명세는 처음부터 폴더를 물려받게 돼 있었고 구현이 따르지 않았다 |

---

## B1. 발견 경위

2026-11-20 사용자 보고(스크린샷 2장): "폴더가 보라색인데 파랑색으로 터미널이 표시된다. 정상인가?" 그리고 서버 편집 화면 — 폴더 `mds-삼성`, 환경 "기본", 색상 "자동"(점선 칸). 그 화면의 문구는 "색상 (지정하지 않으면 환경/폴더 색을 따름)" 이다.

---

## B2. 색이 정해지는 두 경로 (수정 전)

같은 서버의 색을 두 곳에서 따로 계산했다.

| 쓰는 곳 | 계산하는 곳 | 수정 전 규칙 |
|---|---|---|
| 사이드바·시작 화면 카드·빠른 접속의 색 막대 | 프런트 `effectiveColor` ([hosts.ts](../../frontend/src/lib/hosts.ts)) | 서버 색 → 폴더를 올라가며 처음 나오는 폴더 색 → (서버 환경 또는 처음 나오는 폴더 환경)의 색 |
| 터미널 칸 테두리, 탭의 색 막대·배경, 상태 표시줄 제목, "운영" 판정 | 백엔드가 세션을 열 때 정해 세션 정보(`color`·`env`)로 보낸 값 | **서버 자신의 `color`·`env` 그대로** |

백엔드 쪽 코드([channels_session.go](../../internal/app/channels_session.go) `openSession`, 수정 전):

```go
	if bHasSaved {
		szId := record.M_szId
		cfg.Info.M_pszHostId = &szId
		cfg.Info.M_szColor = record.M_szColor   // ← 서버에 지정한 색만
		cfg.Info.M_szEnv = record.M_szEnv       // ← 서버에 지정한 환경만
	} else {
		cfg.Info.M_szEnv = "none"
	}
```

칸 테두리는 이 값을 이렇게 쓴다([Panes.tsx](../../frontend/src/screens/Panes.tsx)):

```tsx
style={{ borderColor: isProd ? 'var(--danger)' : focused ? (color ?? 'var(--accent)') : undefined, … }}
```

운영이면 빨강, 아니면 포커스된 칸은 세션 색(없으면 강조색 파랑), 그 밖은 기본 테두리색이다.

---

## B3. 원인

명세는 세션의 색·환경을 이렇게 정한다([session.md](../../spec/behavior/session.md) §1).

| 필드 | 값 |
|---|---|
| color | 저장된 서버만: 서버 색 → 환경색 → 폴더(상위 최대 32단계) 색/환경색. 없으면 없음 |
| env | 저장된 서버만: 서버 환경(`none` 제외) → 폴더(상위 최대 32단계)의 환경 → `none`. 즉석 연결은 `none` |

백엔드는 이 가운데 "서버 색"과 "서버 환경"만 구현했다. 폴더를 읽는 코드가 세션 열기 경로에 없었다.

보고된 경우를 따라가면:

1. `삼성_gw_tunneling` 은 색 없음, 환경 없음(화면의 "기본"은 저장할 때 값을 비운다).
2. 세션 정보의 `color` 가 비어 나간다.
3. 칸은 `color ?? 'var(--accent)'` 에서 강조색 파랑을 쓴다.
4. 사이드바는 `effectiveColor` 가 폴더 `mds-삼성` 의 보라를 찾아 보라 막대를 그린다.

### B3.1 환경도 같은 문제

| 설정 | 명세 | 수정 전 |
|---|---|---|
| 폴더 = 운영, 서버 환경 없음 | 세션 환경 `prod` | 세션 환경 없음 |

프런트의 "운영" 판정은 모두 세션 정보의 `env` 를 본다. 그래서 수정 전에는 위 설정의 서버에서 다음이 빠졌다.

- 칸의 빨간 테두리와 "운영" 배지 ([Panes.tsx](../../frontend/src/screens/Panes.tsx))
- 위험 명령 붙여넣기 확인, Enter 전 확인 ([TerminalView.tsx:129](../../frontend/src/screens/TerminalView.tsx#L129), [TerminalView.tsx:337](../../frontend/src/screens/TerminalView.tsx#L337) — 둘 다 `info?.env === 'prod'`)

폴더 편집 화면의 환경 도움말은 "폴더 안의 서버가 이 환경 색을 물려받습니다"이다. 색뿐 아니라 운영 보호까지 빠지는 것은 사용자가 화면만 보고는 알 수 없었다.

### B3.2 프런트의 색 순서도 명세와 달랐다

[sidebar.md](../../spec/ui/sidebar.md) "색 규칙": 서버 자신의 색 → 서버 환경색(`none` 제외) → 소속 폴더의 색/환경색 → 상위 폴더로 올라가며 처음 나오는 것 → 없음.

| 경우 | 명세 | 수정 전 프런트 |
|---|---|---|
| 서버 환경 = 개발, 폴더 색 = 보라 | 초록(서버 환경색) | 보라(폴더 색) |
| 폴더 A(환경 운영, 색 없음) 의 상위 폴더 B(색 보라) | 빨강(가까운 폴더의 환경색) | 보라(먼 폴더의 색) |

백엔드만 명세대로 고치면 이 두 경우에 사이드바와 칸이 다시 어긋나므로 함께 고친다.

---

## B4. 설계 결정 (D105)

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 백엔드가 세션을 열 때 폴더를 따라가 색·환경을 정한다. 프런트 계산도 같은 순서로 맞춘다** | 명세 그대로다. "운영" 판정이 세션 정보 하나에서 나오므로 보호 기능이 함께 고쳐진다 | **채택** |
| B. 프런트가 세션의 `hostId` 로 서버·폴더를 찾아 그때그때 계산한다 | 폴더 색을 바꾸면 열린 칸에도 곧바로 반영된다. 그러나 세션 정보의 `color`·`env` 계약(명세 session.md §1)을 버리게 되고, 서버가 삭제된 뒤 열린 세션의 색을 알 수 없다 | 기각 |

세부 결정:

1. **색 순서는 명세대로**: 서버 색 → 서버 환경색 → 폴더를 올라가며 처음 나오는 (폴더 색 → 그 폴더의 환경색). 서버 편집 화면 문구 "환경/폴더 색을 따름"의 순서와도 같다.
2. **환경 `none`·빈 값·모르는 값은 "지정 안 함"** 으로 보고 다음 단계로 넘어간다. `none` 에는 표에 회색(`#64748b`)이 있지만 색을 주지 않는다 — 주면 환경을 지정하지 않은 서버가 모두 회색 테두리가 되고 폴더 색을 가린다.
3. **폴더는 32단계까지만** 올라간다(명세 값). 폴더가 서로를 상위로 가리키는 데이터에서도 멈춘다.
4. **세션 환경은 항상 값이 있다**: 없으면 `none`(명세). 수정 전에는 저장된 서버에 환경이 없으면 필드가 아예 빠졌다. 프런트는 둘을 같게 다룬다([Panes.tsx:95](../../frontend/src/screens/Panes.tsx#L95)).
5. **색·환경은 세션을 열 때 한 번 정한다**(전과 같음). 열린 뒤에 서버·폴더의 색을 바꿔도 그 칸은 바뀌지 않고, 다시 열면 반영된다. 사이드바는 곧바로 바뀐다.

---

## B5. 수정 내용

### B5.1 새 파일 — `internal/app/host_look.go` (전체)

```go
package app

import "encoding/json"

// 세션의 색과 환경을 정한다 (spec/behavior/session.md §1 의 color·env, spec/data/store.md §6).

// G_ciGroupDepthMax 는 폴더를 따라 올라가는 최대 단계다 (순환하는 데이터에서도 멈춘다).
const G_ciGroupDepthMax = 32

// G_cszEnvNone 은 환경이 지정되지 않았음을 뜻하는 값이다.
const G_cszEnvNone = "none"

// g_mapEnvColor 는 환경의 색이다. none 은 색을 주지 않는다.
var g_mapEnvColor = map[string]string{
	"prod":   "#ef4444",
	"stage":  "#f59e0b",
	"perf":   "#a855f7",
	"dev":    "#22c55e",
	"device": "#3b82f6",
}

// groupRecord 는 저장된 폴더에서 색·환경을 정하는 데 쓰는 필드다 (data/store.md §2).
type groupRecord struct {
	M_szId       string `json:"id"`
	M_szParentId string `json:"parentId"`
	M_szColor    string `json:"color"`
	M_szEnv      string `json:"env"`
}

// loadGroups 는 저장된 폴더 목록을 읽는다. 없거나 읽을 수 없으면 빈 목록.
func (pService *Service) loadGroups() []groupRecord {
	rawGroups, bOk := pService.pStore.Get("groups")
	if !bOk {
		return nil
	}
	var slGroups []groupRecord
	if errUnmarshal := json.Unmarshal(rawGroups, &slGroups); errUnmarshal != nil {
		return nil
	}
	return slGroups
}

// resolveLook 은 저장된 서버의 세션 색과 환경을 정한다.
//   - 색: 서버 색 → 서버 환경색 → 소속 폴더부터 위로 올라가며 처음 나오는 (폴더 색 → 폴더 환경색). 없으면 빈 값.
//   - 환경: 서버 환경 → 소속 폴더부터 위로 올라가며 처음 나오는 폴더 환경 → none.
//
// 환경 none(또는 빈 값·모르는 값)은 지정하지 않은 것으로 본다.
func resolveLook(record hostRecord, slGroups []groupRecord) (string, string) {
	szColor, szEnv := record.M_szColor, ""
	if g_mapEnvColor[record.M_szEnv] != "" {
		szEnv = record.M_szEnv
		if szColor == "" {
			szColor = g_mapEnvColor[szEnv]
		}
	}
	szGroupId := record.M_szGroupId
	for iDepth := 0; iDepth < G_ciGroupDepthMax && szGroupId != "" && (szColor == "" || szEnv == ""); iDepth++ {
		pGroup := findGroup(slGroups, szGroupId)
		if pGroup == nil {
			break
		}
		szGroupEnvColor := g_mapEnvColor[pGroup.M_szEnv]
		if szColor == "" {
			szColor = pGroup.M_szColor
			if szColor == "" {
				szColor = szGroupEnvColor
			}
		}
		if szEnv == "" && szGroupEnvColor != "" {
			szEnv = pGroup.M_szEnv
		}
		szGroupId = pGroup.M_szParentId
	}
	if szEnv == "" {
		szEnv = G_cszEnvNone
	}
	return szColor, szEnv
}

func findGroup(slGroups []groupRecord, szId string) *groupRecord {
	for iIdx := range slGroups {
		if slGroups[iIdx].M_szId == szId {
			return &slGroups[iIdx]
		}
	}
	return nil
}
```

읽는 법:

- 색과 환경은 **따로** 정해진다. 색은 가까운 폴더에서 찾았어도 환경은 더 위 폴더에서 올 수 있다(반복 조건 `szColor == "" || szEnv == ""`).
- 환경색 표는 [store.md](../../spec/data/store.md) §6 의 값이고 프런트의 `ENV_INFO` 와 같다. 표가 두 곳(Go·TypeScript)에 있으므로 환경을 더하거나 색을 바꾸면 **둘 다** 고쳐야 한다.
- 저장 데이터의 `parentId`·`groupId` 는 `null` 일 수 있다. Go 의 JSON 해석은 `null` 을 빈 문자열로 두므로 "없음"과 같게 다뤄진다.

### B5.2 `internal/app/channels_session.go`

```go
// hostRecord 에 필드 추가
	M_szGroupId        string            `json:"groupId"`
```

```go
// openSession — before
		cfg.Info.M_szColor = record.M_szColor
		cfg.Info.M_szEnv = record.M_szEnv

// after
		// 색·환경은 서버에 없으면 폴더 것을 물려받는다 (session.md §1, D105).
		cfg.Info.M_szColor, cfg.Info.M_szEnv = resolveLook(record, pService.loadGroups())
```

즉석 연결(`else` 쪽 `M_szEnv = "none"`)은 그대로다.

### B5.3 `frontend/src/lib/hosts.ts`

```ts
// before
/** 서버 색 → 폴더 색 → 환경 색 순서로 고른다. 없으면 undefined(호출한 쪽이 기본 테두리색 사용). */
export function effectiveColor(host: Host, groups: Group[]): string | undefined {
  if (host.color) return host.color;
  const chain: Group[] = [];
  let gid = host.groupId;
  while (gid) {
    const g = groups.find((x) => x.id === gid);
    if (!g) break;
    chain.push(g);
    gid = g.parentId;
  }
  const folderColor = chain.find((g) => g.color)?.color;
  if (folderColor) return folderColor;
  const env = host.env ?? chain.find((g) => g.env)?.env;
  return env ? ENV_INFO[env].color : undefined;
}
```

```ts
// after
/** 폴더를 따라 올라가는 최대 단계 (순환하는 데이터에서도 멈춘다). */
const GROUP_DEPTH_MAX = 32;

/** 환경의 색. none(또는 없음)은 색을 주지 않는다. */
function envColor(env: Host['env']): string | undefined {
  return env && env !== 'none' ? ENV_INFO[env]?.color : undefined;
}

/**
 * 서버의 색 (spec/ui/sidebar.md "색 규칙", spec/behavior/session.md §1 color — 백엔드 resolveLook 과 같은 순서).
 * 서버 색 → 서버 환경색 → 소속 폴더부터 위로 올라가며 처음 나오는 (폴더 색 → 폴더 환경색).
 * 없으면 undefined(호출한 쪽이 기본 테두리색 사용).
 */
export function effectiveColor(host: Host, groups: Group[]): string | undefined {
  const own = host.color || envColor(host.env);
  if (own) return own;
  let gid = host.groupId;
  for (let depth = 0; gid && depth < GROUP_DEPTH_MAX; depth++) {
    const g = groups.find((x) => x.id === gid);
    if (!g) break;
    const c = g.color || envColor(g.env);
    if (c) return c;
    gid = g.parentId;
  }
  return undefined;
}
```

수정 전 코드에서 함께 고쳐진 것:

- 순서가 명세와 같아졌다(§B3.2 의 두 경우).
- 수정 전에는 폴더가 서로를 상위로 가리키면 `while (gid)` 가 **끝나지 않았다**(화면이 멈춤). 32단계에서 멈춘다. 상위 폴더 선택 목록은 자기 하위 폴더를 걸러 주지 않는다고 명세에 적혀 있어([host-editor.md](../../spec/ui/host-editor.md) "알려진 한계") 실제로 만들 수 있는 데이터다.
- 수정 전에는 환경이 `none` 으로 저장된 서버·폴더가 회색(`#64748b`)을 받았다. 이제 색을 주지 않는다.

### B5.4 수정 뒤 표시 (보고된 경우)

| 서버 | 설정 | 사이드바 막대 | 칸 테두리(포커스) | 탭 막대(선택) |
|---|---|---|---|---|
| 삼성_gw_tunneling | 색 없음·환경 없음, 폴더 `mds-삼성` 보라 | 보라 (전과 같음) | **보라** (전에는 파랑) | **보라** (전에는 파랑) |
| mds-컴파일 | 서버 색 빨강, 환경 개발 | 빨강 | 빨강 (전과 같음) | 빨강 |
| 색도 폴더도 없는 서버 | — | 없음 | 파랑(강조색, 전과 같음) | 파랑 |

---

## B6. 영향 범위와 회귀 위험

| 영역 | 영향 |
|---|---|
| 칸 테두리·칸 머리글 배지, 탭의 색 막대·배경, 상태 표시줄 제목 색 | 서버에 색·환경이 없고 폴더에 있으면 **폴더 것이 새로 반영된다**(의도한 변경) |
| 운영 서버 보호(빨간 테두리, 위험 명령 확인) | **폴더만 운영인 서버도 이제 운영으로 취급된다**(의도한 변경). 전에는 확인 없이 실행되던 명령에 확인 창이 뜨게 된다 |
| 사이드바·시작 화면·빠른 접속의 색 막대 | §B3.2 의 두 경우에 색이 바뀐다. 환경이 `none` 으로 저장된 항목의 회색 막대가 없어진다 |
| 세션 정보의 `env` | 저장된 서버는 이제 항상 값이 있다(`none` 포함). 전에는 없을 때 필드가 빠졌다 |
| IPC 계약, 데이터 파일, 문구(`locales/`), CSS | 바뀌지 않았다 |
| 이미 열려 있는 세션 | 바뀌지 않는다. 앱을 다시 실행하거나 세션을 다시 열어야 새 값이 들어간다 |

위험:

1. **운영 판정이 넓어진다.** 폴더를 "운영"으로 지정해 두고 그 안에 실제로는 운영이 아닌 서버를 넣어 둔 경우, 그 서버에도 빨간 테두리와 확인 창이 생긴다. 서버 편집에서 환경을 다른 값(개발 등)으로 지정하면 서버 값이 먼저 적용된다. 다만 "기본"은 "지정 안 함"이라 폴더의 운영을 끌 수 없다.
2. **환경색 표가 두 곳에 있다**(§B5.1). 한쪽만 고치면 사이드바와 칸의 색이 다시 어긋난다.
3. **세션 복원**(앱을 다시 열 때 탭 복원)으로 열리는 세션도 같은 `openSession` 을 지난다고 보았으나, 복원 경로를 따로 추적해 확인하지는 않았다 → §B7.2 의 7번.

---

## B7. 검증

### B7.1 자동 (2026-11-20 실행)

| 명령 | 결과 |
|---|---|
| `go build ./...` (`cli-manager-cr/`) | 통과 |
| `go vet ./internal/app/` | 통과 |
| `go test -count=1 ./...` | 전부 통과 |
| `npx tsc --noEmit` (`frontend/`, PowerShell) | 통과 (exit 0) |
| `npx vitest run` (`frontend/`, PowerShell) | 40개 파일, 259개 테스트 통과 |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` | `build/bin/dariterm.exe` 생성 (2026-11-20 15:14) |

새 Go 테스트 [host_look_test.go](../../internal/app/host_look_test.go) 3개.

| 테스트 | 확인하는 것 |
|---|---|
| `TestResolveLook` | 12가지 경우: 폴더 색 물려받기(**보고된 경우**), 서버 환경이 `none` 이어도 물려받기, 서버 색 우선, 서버 환경색이 폴더 색보다 먼저, 가까운 폴더의 환경색이 먼 폴더의 색보다 먼저, 빈 폴더 건너뛰기, 서버 환경이 폴더 환경보다 먼저, 아무것도 없음, 폴더 없는 서버, 없어진 폴더, 모르는 환경 값, 순환하는 폴더 |
| `TestLookRecordsFromStoreJson` | 저장 데이터 형식(`parentId`·`groupId` 가 `null`) 그대로 읽어 같은 결과 |
| `TestSessionOpenInheritsFolderLook` | `store.patch` 로 폴더·서버를 넣고 `session.open` 을 불러, 돌아온 세션 정보의 `color`·`env` 확인: 보라 폴더 안 서버 → `#8b5cf6`·`none`, 운영 폴더 안 서버 → `#ef4444`·`prod`, 폴더 없는 서버 → 색 없음·`none` |

**수정 전 동작과의 대조**: `openSession` 의 그 줄을 수정 전 코드(`record.M_szColor, record.M_szEnv`)로 잠깐 되돌려 `TestSessionOpenInheritsFolderLook` 을 돌렸더니 세 경우 모두 실패했다(색·환경이 빈 값). 확인 뒤 되돌렸다.

프런트 [hosts.test.ts](../../frontend/src/lib/hosts.test.ts) 의 색 우선순위 테스트를 2개에서 4개로 다시 썼다: 서버 색 우선, 서버 환경색이 폴더 색보다 먼저(`none` 은 지정 안 함), 폴더를 올라가며 처음 나오는 색/환경색, 없으면 `undefined`(폴더 없음·없어진 폴더·순환). 수정 전 `effectiveColor` 로 이 테스트를 돌려 보지는 않았다(순환 경우는 끝나지 않을 것으로 본다 — 코드로 판단한 것이다).

### B7.2 수동 (남음) — `build/bin/dariterm.exe` (2026-11-20 15:14 빌드)

앱을 **완전히 끝내고 새 실행 파일로 다시 연다**(이미 열려 있던 세션은 옛 값을 갖고 있다).

| # | 조작 | 기대 |
|---|---|---|
| 1 | 보라 폴더 `mds-삼성` 의 `삼성_gw_tunneling`(색·환경 없음)을 연다 | 칸 테두리가 **보라**, 선택된 탭의 위쪽 막대와 옅은 배경도 보라. 사이드바 막대와 같은 색 |
| 2 | 서버 색이 있는 서버(`mds-컴파일`, 빨강)를 연다 | 전처럼 그 서버 색 |
| 3 | 색도 폴더 색도 없는 서버를 연다 | 전처럼 파랑(강조색) |
| 4 | 폴더 환경을 "운영"으로 하고 그 안의 환경 없는 서버를 연다 | 빨간 테두리 + "운영" 배지. `rm -rf /tmp/x` 입력 후 Enter → "운영 서버 위험 명령" 확인 창 |
| 5 | 4 의 서버에 환경 "개발"을 지정하고 다시 연다 | 운영 표시·확인 창 없음, 초록(개발 환경색) |
| 6 | 폴더 색을 다른 색으로 바꾼다 | 사이드바는 곧바로 바뀌고, 열린 칸은 그대로. 세션을 닫고 다시 열면 새 색 |
| 7 | 탭을 연 채 앱을 끝내고 다시 연다(세션 복원) | 복원된 칸도 1 과 같은 색 |
| 8 | 빠른 접속(즉석 연결)으로 연다 | 전처럼 파랑, 운영 표시 없음 |
| 9 | 하위 폴더(색 없음) 안의 서버를 연다. 상위 폴더에 색이 있다 | 상위 폴더 색 |

---

## B8. 명세

명세는 바꾸지 않았다. [session.md](../../spec/behavior/session.md) §1 과 [sidebar.md](../../spec/ui/sidebar.md) "색 규칙"이 이미 이 동작을 정하고 있었다.

명세끼리 표현이 다른 곳이 하나 있다. [shell.md](../../spec/ui/shell.md) 는 색의 출처를 "서버 색·폴더 색·환경 색"으로 나열하는데, 순서를 정한 문장이 아니라 출처를 든 것으로 읽고 session.md·sidebar.md 의 순서를 따랐다.

- [docs/decisions.md](../decisions.md): D105.

---

## B9. 남은 일

- §B7.2 수동 검증.
- 열린 세션에 색·환경 변경을 곧바로 반영할지는 정하지 않았다(지금은 다시 열어야 한다, §B4 의 5).
- 폴더의 "운영"을 서버에서 끄는 방법이 없다(§B6 위험 1). 필요하면 서버 환경에 "폴더 값을 따르지 않음"에 해당하는 선택을 따로 둬야 한다.
