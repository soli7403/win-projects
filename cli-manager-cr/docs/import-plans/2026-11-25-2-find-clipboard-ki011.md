# 찾기 막대를 연 동안 클립보드가 덮이고 강조가 늦던 문제 (KI-011) — 원인·수정·코드·검증

> **적용 메모 (2026-11-25, 이 PC)**: 이 문서의 코드·테스트·명세는 2026-11-23 에 KI-011 수정(D120-1)으로 이미 들어가 있었다(새 파일 4개가 같고 패치는 이미 적용된 상태). 그래서 따로 적용할 것이 없었고, 결정 번호도 D120-1 로 적었다.

**작성**: 2026-11-25 · **상태**: 코드 수정·자동 테스트 완료(D120-1). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: KI-011 (이 문서로 resolved)
**관련**: 찾기 강조를 만든 문서 [2026-11-18-pane-find-button.md](./2026-11-18-pane-find-button.md)(D96), 명세 [spec/ui/panes.md](../../spec/ui/panes.md) §2.4 · [spec/ui/settings.md](../../spec/ui/settings.md) §2, 결정 [docs/decisions.md](../decisions.md) D120-1, 같은 날의 다른 문서 [zmodem-lrzsz-abort-stall-tmux](./2026-11-25-1-zmodem-lrzsz-abort-stall-tmux.md)

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체·패치) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 패치의 "고치기 전"은 2026-11-25 아침의 소스다.

---

## 0. 한 줄 요약

터미널 찾기 막대(Ctrl+Shift+F)를 열어 둔 동안 네 가지가 잘못됐다. 뿌리는 하나다 — **검색 애드온이 출력이 올 때마다 다시 찾으면서 터미널의 "선택"을 바꾸고, 앱의 자동 복사는 누가 바꾼 선택인지 가리지 않고 복사했다.**

| # | 고치기 전 | 고친 뒤 |
|---|---|---|
| 1 | "선택하면 자동 복사"가 켜져 있으면(기본값) 찾을 글을 칠 때·[다음]·[이전] 때마다, 그리고 **출력이 0.2초 멈출 때마다** 찾은 글자가 클립보드에 복사돼 복사해 둔 내용이 덮였다 | 자동 복사는 **마우스로 선택을 끝냈을 때만**(왼쪽 버튼을 뗄 때 한 번). 찾기는 클립보드를 건드리지 않는다 |
| 2 | 출력이 0.2초 넘게 멈추지 않으면 새로 들어온 줄의 찾은 항목이 강조되지 않았다 | 화면에 보이는 줄은 **0.1초마다** 다시 그린다(출력이 이어져도 미루지 않는다) |
| 3 | 찾기 막대가 열린 동안 마우스로 다른 곳을 선택해도 출력이 멈추면 선택이 찾은 항목으로 바뀌었다 | 찾기는 [다음]·[이전]·입력 때만 선택을 바꾼다. 마우스로 한 선택은 남는다 |
| 4 | 강조는 버퍼 **맨 위(가장 오래된 줄)** 부터 1000개까지. 흔한 낱말이면 최근 줄이 강조되지 않았다 | 화면에 보이는 줄은 항상 강조. 스크롤 막대 표시는 **최근 줄부터** 1000줄 |

바뀐 것은 프런트엔드뿐이다: 새 파일 2개(+ 시험 2개), `TerminalView.tsx`·`Panes.tsx` 수정.

## 1. 배경

### 1.1 경위

- 2026-11-18: 찾기 강조(D96)를 만든 날, 사용자 질문 "로그가 계속 출력중에 찾기를 하면 어떻게 되나?"에 답하려고 코드를 읽다가 발견해 KI-011 로 적어 두었다(앱에서 재현해 본 것은 아니고 코드로 확인).
- 2026-11-25: 사용자 요청 "KI-011와 KI-012를 수정해주고 …". (같은 수정이 다른 작업 폴더에는 2026-11-23 에 들어가 있었고, 그 코드를 이 폴더에 옮겼다. 그쪽 번호로는 D120-1 이다.)

### 1.2 용어

| 말 | 뜻 |
|---|---|
| 찾기 막대 | 칸 위에 뜨는 입력 막대(입력창 + [이전] [다음] [닫기]). `TerminalView.tsx` 의 `FindBar` |
| 찾은 항목 | 찾는 글과 일치하는 자리 전부. 주황 계열 바탕으로 강조한다 |
| 현재 항목 | [다음]·[이전] 으로 옮겨 가는, 지금 가리키는 한 자리. 빨강 바탕 + 스크롤 막대 표시(D96) |
| 검색 애드온 | `@xterm/addon-search` 0.16. xterm.js 의 공식 찾기 부가 기능 |
| 강조(decoration) | xterm.js 가 글자 칸 위에 덧입히는 표시(바탕색·스크롤 막대 표시). 화면 내용(버퍼)은 바꾸지 않는다 |
| 표식(marker) | 버퍼의 한 줄을 가리키는 책갈피. 줄이 위로 밀려도 따라간다. 강조는 표식에 붙는다 |
| 터미널 선택 | 마우스로 끌어 고른 영역(`term.getSelection()`). 터미널 하나에 하나뿐이다 |
| 자동 복사 | 설정 `copyOnSelect`(기본 켜짐) — "드래그로 선택하면 즉시 클립보드에 복사합니다 (PuTTY 방식)" |

## 2. 원인 분석 (Cause)

### 2.1 애드온이 하는 일

고치기 전 `FindBar` 는 애드온에 찾기와 강조를 모두 맡겼다:

```ts
// 고치기 전 (frontend/src/screens/TerminalView.tsx)
const matchBg = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
const opts = { caseSensitive: false, decorations: { matchOverviewRuler: FIND_RULER_MATCH, activeMatchColorOverviewRuler: FIND_RULER_ACTIVE, matchBackground: matchBg, activeMatchBackground: FIND_ACTIVE_BG } };
const find = (text: string, dir: 1 | -1) => {
  const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
  …
};
```

애드온은 (공개 문서와 관찰한 동작으로)

1. **현재 항목을 터미널의 "선택"으로 표시한다** — `findNext`·`findPrevious` 는 찾은 자리를 `term.select(…)` 로 고른다. 별도의 "현재 항목" 개념이 없다.
2. `decorations` 옵션을 받으면 **찾은 항목 전부에 강조를 붙이고**, 그것을 최신으로 유지하려고 **출력이 올 때마다 0.2초 뒤 버퍼 전체를 다시 찾는다**. 그 0.2초 안에 출력이 또 오면 다시 0.2초를 미룬다(debounce).
3. 다시 찾을 때 **현재 항목도 다시 고른다** → 선택이 다시 바뀐다.
4. 강조는 버퍼 맨 위에서부터 찾아 **1000개**에서 멈춘다.

### 2.2 앱의 자동 복사

```ts
// 고치기 전 (frontend/src/screens/TerminalView.tsx)
// 선택하면 자동 복사
t.onSelectionChange(() => {
  const st2 = useApp.getState().store?.settings;
  if (st2?.copyOnSelect && t.getSelection()) void copySelection(t);
});
```

`onSelectionChange` 는 **누가 선택을 바꿨는지 알려 주지 않는다**. 마우스로 끈 것이든 애드온이 `select` 한 것이든 똑같이 온다. 그리고 마우스로 끄는 동안에도 선택이 넓어질 때마다 와서, 한 번 끄는 동안 클립보드에 여러 번 썼다.

### 2.3 둘이 만나면

| 증상 | 길 |
|---|---|
| 1 클립보드가 덮인다 | 글을 침·[다음] → 애드온이 현재 항목을 선택 → `onSelectionChange` → 복사. 출력이 이어지는 세션에서는 0.2초 쉴 때마다 애드온이 다시 찾아 다시 선택 → 또 복사. 다른 프로그램에서 복사해 둔 것이 사라진다 |
| 2 강조가 늦다 | 애드온의 다시 찾기가 debounce 라 출력이 0.2초 넘게 쉬지 않으면(예: `tail -f`, 빌드 로그) 영영 실행되지 않는다. 멈추면 한꺼번에 갱신 |
| 3 선택이 바뀐다 | 찾기 막대를 연 채 마우스로 다른 곳을 선택 → 출력이 쉬면 애드온이 현재 항목을 다시 선택 |
| 4 최근 줄이 강조되지 않는다 | 강조가 맨 위부터 1000개. `INFO` 같은 흔한 낱말이면 수천 줄 위에서 한도가 찬다. [다음]·[이전] 이동은 된다 |

### 2.4 확인

- 코드로: 위 두 조각과 애드온의 공개 옵션(`decorations` 를 주면 강조를 유지한다).
- 고친 뒤 시험으로: `copyOnSelect.test.ts` 의 "코드가 바꾼 선택(찾기의 현재 항목)은 복사하지 않는다", `termFind.test.ts` 의 "출력이 쉬지 않고 이어져도 정해진 간격마다 다시 그린다(미루지 않는다)", "버퍼 맨 위가 아니라 최근 줄부터 스크롤 막대에 표시하고, 한도를 넘지 않는다".

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| A | 애드온이 선택을 바꾸는 동안만 자동 복사를 잠깐 끈다(깃발) | 기각 — 1번만 풀린다. 애드온이 언제 다시 찾는지(0.2초 뒤, 미뤄질 수 있음) 밖에서 알 수 없어 깃발을 정확히 켜고 끌 수 없다 |
| B | 애드온의 안쪽(비공개 메서드·타이머)을 막는다 | 기각 — 애드온 버전이 바뀌면 조용히 깨진다 |
| C | 찾기를 전부 새로 쓴다(애드온을 뺀다) | 기각 — 현재 항목으로 옮겨 가기(줄 넘김, 넓은 글자, 스크롤해 보여 주기)까지 다시 만들어야 한다 |
| **D (고름)** | 애드온에는 **`decorations` 를 넘기지 않는다** → 애드온은 [다음]·[이전]·입력 때 현재 항목으로 옮겨 선택하는 일만 한다(출력마다 다시 찾지 않는다). **찾은 항목 표시는 앱이 직접 그린다**(`termFind.ts`). 자동 복사는 **마우스 선택에만** 반응한다(`copyOnSelect.ts`) | 애드온의 공개 기능만 쓴다. 증상 네 가지가 모두 풀린다 |

### 3.1 세부 결정 — 찾은 항목 그리기 (`termFind.ts`)

| 항목 | 값 | 까닭 |
|---|---|---|
| 바탕 강조를 그리는 범위 | **화면에 보이는 줄만** | 버퍼 전체(수만 줄)에 강조를 붙이면 느리다. 안 보이는 줄의 바탕은 볼 수 없다 |
| 바탕을 다시 그리는 간격 | 0.1초 (`FIND_VIEW_REFRESH_MS`), **throttle** | debounce 가 아니라 throttle 이라 출력이 쉬지 않아도 0.1초마다 그린다(증상 2) |
| 한 화면의 바탕 한도 | 400곳 (`FIND_MAX_ON_SCREEN`) | 한 글자를 찾을 때 화면이 강조로 뒤덮여 느려지는 것을 막는다 |
| 스크롤 막대 표시 범위 | 버퍼 전체에서 **최근 줄부터** 1000줄 (`FIND_RULER_MAX_ROWS`) | 최근 것이 더 쓸모 있다(증상 4). 표시는 줄마다 하나 |
| 스크롤 막대를 다시 만드는 간격 | 1초 (`FIND_RULER_REFRESH_MS`) | 버퍼 전체를 훑는 일이라 드물게. 찾은 줄들이 같고 버퍼가 밀리지 않았으면 다시 만들지 않는다 |
| 빨리 거르기 | 줄을 글로 바꿔(`translateToString`) 찾는 글이 있는 줄만 칸 단위로 계산 | 대부분의 줄은 일치하지 않는다 |
| 이어진 줄 | 화면 폭 때문에 다음 줄로 넘어간 글(`isWrapped`)은 한 줄로 이어 붙여 찾는다 | 줄 끝에 걸친 낱말도 찾는다. 강조는 줄마다 나눠 칠한다(`rowRanges`, D96 의 것) |
| 넓은 글자 | 칸 너비 2 인 글자(한글 등)는 글 위치 → 칸 위치 표로 바꾼다 | 강조 폭이 어긋나지 않게 |
| 대소문자 | 구분하지 않는다. 특수 문자는 글자 그대로 | 명세 §2.4. 애드온에도 `caseSensitive: false` |
| 글이 바뀌면 | 곧바로 그린다(0.1초 기다리지 않는다) | 한 글자 칠 때마다 반응이 보이게 |
| 다시 그릴 계기 | 출력(`onWriteParsed`)·스크롤(`onScroll`)·크기 변경(`onResize`) | 스크롤하면 0.1초 뒤 새로 보이는 줄에 바탕이 칠해진다 |
| 대체 화면(vim 등) | 표식을 만들 수 없으면 그리지 않는다 | xterm.js 가 대체 화면에서는 표식을 주지 않는다 |
| 현재 항목 | 그대로(D96 — 빨강 바탕 + 스크롤 막대 빨강) | `FindBar` 의 `markActive` 가 선택 영역 위에 직접 얹는다 |

### 3.2 세부 결정 — 자동 복사 (`copyOnSelect.ts`)

| 항목 | 값 | 까닭 |
|---|---|---|
| 언제 복사하나 | 터미널 **안에서** 왼쪽 버튼을 누르고, **어디서든** 뗐을 때, 선택이 있으면 한 번 | 끌기·두 번 클릭(낱말)·세 번 클릭(줄) 모두 버튼을 떼는 것으로 끝난다. 끌다가 터미널 밖에서 떼도 된다 |
| 읽는 때 | `mouseup` 다음 박자(`setTimeout 0`) | xterm.js 가 같은 `mouseup` 으로 선택을 마무리한 뒤에 읽는다 |
| 설정 | 버튼을 뗄 때마다 읽는다 | 설정을 바꾸면 열려 있는 터미널에도 곧바로 적용 |
| 칸 메뉴 "모두 선택" | `selectAllInPane` 이 선택한 뒤 따로 복사 | 마우스 선택이 아니지만 사용자가 고른 선택이다. 자동 복사가 켜져 있으면 복사되는 것이 자연스럽다 |
| 코드가 바꾼 선택 | 복사하지 않는다 | 찾기의 현재 항목 등 |
| 달라지는 점 | 끄는 동안 여러 번 쓰던 것이 **뗄 때 한 번**이 된다 | 클립보드에 남는 결과는 같다 |

키보드로 선택을 넓히는 기능은 이 앱에 없다(Shift+방향키는 서버로 간다).

### 3.3 영향 범위

- 프런트엔드만. 백엔드·IPC·설정 키·문구는 그대로다.
- 찾기 막대의 모양·단축키·[다음]·[이전] 동작은 그대로다.
- 달라 보이는 점: ① 스크롤 막대의 찾은 위치 표시가 "맨 위부터 1000개"에서 "최근 1000줄"로. ② 스크롤해 올라가면 찾은 항목의 바탕이 0.1초 뒤에 칠해진다(전에는 미리 칠해져 있었다 — 단 1000개까지).

---

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 바뀐 것 |
|---|---|---|
| `frontend/src/lib/termFind.ts` | **새 파일** (241줄) | `attachFindMarks` — 찾은 항목의 바탕·스크롤 막대 표시를 직접 그린다. `logicalLine`·`findInText` |
| `frontend/src/lib/termFind.test.ts` | **새 파일** (153줄) | 시험 8개 |
| `frontend/src/lib/copyOnSelect.ts` | **새 파일** (42줄) | `attachCopyOnMouseSelect` — 마우스로 선택을 끝냈을 때만 복사 |
| `frontend/src/lib/copyOnSelect.test.ts` | **새 파일** (72줄) | 시험 4개 |
| `frontend/src/screens/TerminalView.tsx` | 수정 (+33 −12) | `FindBar` 가 애드온에 `decorations` 를 넘기지 않고 `attachFindMarks` 를 쓴다. 자동 복사를 `attachCopyOnMouseSelect` 로. `selectAllInPane` 추가. 정리할 때 애드온·리스너 해제 |
| `frontend/src/screens/Panes.tsx` | 수정 (+2 −2) | 칸 메뉴 "모두 선택"이 `selectAllInPane` 을 부른다 |
| `spec/ui/panes.md` | 수정 (+2) | §2.4 찾기 막대 |
| `spec/ui/settings.md` | 수정 (+1) | §2 선택하면 자동 복사 |
| `docs/decisions.md`, `.claude/known-issues.md`, `CHANGELOG.md`, `CLAUDE.md` | 수정 | D120-1, KI-011 → resolved, 미배포 절, 인덱스 |

의존성은 바뀌지 않았다(`@xterm/addon-search` 0.16 그대로).

---

## B. 코드

### `frontend/src/lib/copyOnSelect.ts` — 새 파일 (전체)

```ts
// "선택하면 자동 복사"(spec/ui/settings.md: 드래그로 선택하면 즉시 클립보드에 복사).
// 사용자가 마우스로 선택을 끝냈을 때(왼쪽 버튼을 뗄 때) 한 번만 복사한다. 앱이 코드로 바꾼 선택
// (찾기의 현재 항목 등)은 복사하지 않는다 — KI-011: 찾기가 선택을 바꿀 때마다 클립보드가 덮이던 문제.
import type { IDisposable } from '@xterm/xterm';

export interface SelectionSource {
  hasSelection(): boolean;
  getSelection(): string;
}

/**
 * el 안에서 왼쪽 버튼을 누른 뒤 (어디서든) 뗐을 때 선택이 있으면 copy 를 부른다.
 * enabled() 가 거짓이면 복사하지 않는다(설정을 그때그때 읽는다).
 */
export function attachCopyOnMouseSelect(
  el: HTMLElement,
  term: SelectionSource,
  enabled: () => boolean,
  copy: () => void,
  win: Pick<Window, 'addEventListener' | 'removeEventListener'> = window,
): IDisposable {
  let pressed = false;
  const onDown = (ev: MouseEvent) => {
    if (ev.button === 0) pressed = true;
  };
  const onUp = (ev: MouseEvent) => {
    if (ev.button !== 0 || !pressed) return;
    pressed = false;
    // xterm 이 같은 mouseup 으로 선택을 마무리한 뒤에 읽는다.
    setTimeout(() => {
      if (enabled() && term.hasSelection() && term.getSelection()) copy();
    }, 0);
  };
  el.addEventListener('mousedown', onDown, true);
  win.addEventListener('mouseup', onUp, true);
  return {
    dispose() {
      el.removeEventListener('mousedown', onDown, true);
      win.removeEventListener('mouseup', onUp, true);
    },
  };
}
```

- `mousedown` 은 터미널 요소에서, `mouseup` 은 창에서 듣는다(끌다가 밖에서 떼는 경우). 둘 다 캡처 단계라 xterm.js 가 이벤트를 먹어도 듣는다.
- 터미널 밖에서 누른 클릭의 `mouseup` 은 `pressed` 가 거짓이라 세지 않는다. 그래서 찾기 막대의 [다음] 버튼을 눌렀을 때(애드온이 선택을 바꾼다) 복사하지 않는다.
- `win` 인자는 시험에서 가짜 창을 넣으려는 것이다.

### `frontend/src/lib/termFind.ts` — 새 파일 (전체)

```ts
// 터미널 찾기의 "찾은 항목" 표시(현재 항목 제외). 근거: spec/ui/panes.md §2.4, KI-011.
// 검색 애드온에 강조를 맡기면 애드온이 출력마다 다시 찾으며 터미널 선택을 바꾼다(자동 복사·사용자 선택을 덮음).
// 그래서 애드온은 현재 항목 이동에만 쓰고, 찾은 항목의 바탕·스크롤 막대 표시는 여기서 직접 그린다.
// - 화면에 보이는 줄: 바탕색 강조. FIND_VIEW_REFRESH_MS 간격(출력이 이어져도 미루지 않는다).
// - 스크롤 막대: 버퍼 전체에서 찾되 최근 줄부터 FIND_RULER_MAX_ROWS 줄까지, FIND_RULER_REFRESH_MS 간격.
// - 화면 폭 때문에 다음 줄로 이어진 글(isWrapped)도 한 줄로 이어 붙여 찾는다.
import type { IDisposable, Terminal } from '@xterm/xterm';
import { rowRanges } from './findHighlight';

export const FIND_VIEW_REFRESH_MS = 100;
export const FIND_RULER_REFRESH_MS = 1000;
/** 화면 하나에 얹는 바탕 강조 한도. */
export const FIND_MAX_ON_SCREEN = 400;
/** 스크롤 막대에 표시하는 줄 수 한도(최근 줄부터). */
export const FIND_RULER_MAX_ROWS = 1000;
/** 이어진 한 줄에서 찾는 일치 수 한도. */
const FIND_MAX_PER_LINE = 200;

export interface FindColors {
  /** 찾은 항목의 바탕(#RRGGBB). */
  matchBackground: string;
  /** 스크롤 막대 표시 색. */
  ruler: string;
}

export interface FindMarks {
  /** 찾는 글을 바꾼다. 빈 글이면 표시를 모두 지운다. */
  setQuery(q: string): void;
  dispose(): void;
}

interface CellLine {
  readonly length: number;
  readonly isWrapped: boolean;
  getCell(x: number): { getChars(): string; getWidth(): number } | undefined;
  translateToString(trimRight?: boolean): string;
}

interface Pos { x: number; y: number }

/** 버퍼 줄 y 에서 시작하는 이어진 줄의 글과 글 위치별 칸 위치. 다음 줄 번호도 돌려준다. */
export function logicalLine(getLine: (y: number) => CellLine | undefined, y: number, cols: number): { text: string; pos: Pos[]; next: number } {
  let text = '';
  const pos: Pos[] = [];
  let row = y;
  let line = getLine(row);
  let lastEnd: Pos = { x: 0, y };
  while (line) {
    let x = 0;
    const n = Math.min(cols, line.length);
    while (x < n) {
      const cell = line.getCell(x);
      const width = cell?.getWidth() ?? 1;
      if (width === 0) {
        x++;
        continue;
      }
      const chars = cell?.getChars() || ' ';
      for (let i = 0; i < chars.length; i++) pos.push({ x, y: row });
      text += chars;
      x += width;
    }
    lastEnd = { x, y: row };
    row++;
    line = getLine(row);
    if (!line?.isWrapped) break;
  }
  pos.push(lastEnd);
  return { text, pos, next: Math.max(row, y + 1) };
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 글에서 q 의 일치 구간(대소문자 무시)을 찾는다. */
export function findInText(text: string, q: string): { start: number; end: number }[] {
  if (!q) return [];
  const re = new RegExp(escapeRegExp(q), 'gi');
  const out: { start: number; end: number }[] = [];
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) && out.length < FIND_MAX_PER_LINE) {
    if (m[0].length === 0) {
      re.lastIndex++;
      continue;
    }
    out.push({ start: m.index, end: m.index + m[0].length });
  }
  return out;
}

/** 터미널에 찾은 항목 표시를 붙인다. */
export function attachFindMarks(term: Terminal, colors: FindColors): FindMarks {
  let query = '';
  let viewDrawn: IDisposable[] = [];
  let rulerDrawn: IDisposable[] = [];
  let rulerKey = '';
  let viewTimer: ReturnType<typeof setTimeout> | null = null;
  let rulerTimer: ReturnType<typeof setTimeout> | null = null;
  let disposed = false;

  const clearList = (list: IDisposable[]) => {
    for (const d of list) d.dispose();
  };

  // 표식은 커서 기준 상대 줄로 만든다. 대체 화면(vim 등)에서는 undefined.
  const mark = (row: number, x: number, width: number, bg: boolean, ruler: boolean): IDisposable | null => {
    const buf = term.buffer.active;
    const marker = term.registerMarker(row - buf.baseY - buf.cursorY);
    if (!marker) return null;
    const deco = term.registerDecoration({
      marker,
      x,
      width: Math.max(1, width),
      ...(bg ? { backgroundColor: colors.matchBackground } : {}),
      ...(ruler ? { overviewRulerOptions: { color: colors.ruler, position: 'center' as const } } : {}),
    });
    // 강조를 먼저, 표식을 나중에 지운다(TerminalView 의 현재 항목 강조와 같은 이유).
    return {
      dispose() {
        deco?.dispose();
        marker.dispose();
      },
    };
  };

  const cols = () => term.cols;
  const getLine = (y: number) => term.buffer.active.getLine(y) as CellLine | undefined;

  const drawView = () => {
    viewTimer = null;
    if (disposed) return;
    clearList(viewDrawn);
    viewDrawn = [];
    if (!query) return;
    const buf = term.buffer.active;
    const top = buf.viewportY;
    const bottom = top + term.rows; // 제외
    // 화면 맨 윗줄이 이어진 줄의 중간이면 그 줄의 시작부터 본다.
    let y = top;
    while (y > 0 && getLine(y)?.isWrapped) y--;
    let count = 0;
    while (y < bottom && count < FIND_MAX_ON_SCREEN) {
      const ll = logicalLine(getLine, y, cols());
      for (const m of findInText(ll.text, query)) {
        const s = ll.pos[m.start];
        const e = ll.pos[m.end];
        if (!s || !e) continue;
        for (const r of rowRanges(s, e, cols())) {
          if (r.row < top || r.row >= bottom || count >= FIND_MAX_ON_SCREEN) continue;
          const d = mark(r.row, r.x, r.width, true, false);
          if (!d) return;
          viewDrawn.push(d);
          count++;
        }
      }
      y = ll.next;
    }
  };

  const drawRuler = () => {
    rulerTimer = null;
    if (disposed) return;
    const rows: number[] = [];
    if (query) {
      const buf = term.buffer.active;
      // 최근 줄부터: 맨 아래에서 이어진 줄의 시작을 찾아 위로 올라간다.
      let end = buf.length; // 제외
      while (end > 0 && rows.length < FIND_RULER_MAX_ROWS) {
        let start = end - 1;
        while (start > 0 && getLine(start)?.isWrapped) start--;
        // 먼저 빠른 글 변환으로 거른다(칸 단위로 위치를 계산하는 것은 일치하는 줄만).
        let quick = '';
        for (let y = start; y < end; y++) quick += getLine(y)?.translateToString(false) ?? '';
        if (!quick.toLowerCase().includes(query.toLowerCase())) {
          end = start;
          continue;
        }
        const ll = logicalLine(getLine, start, cols());
        const hit = findInText(ll.text, query);
        const seen = new Set<number>();
        for (const m of hit) {
          const s = ll.pos[m.start];
          if (s && !seen.has(s.y)) seen.add(s.y);
        }
        for (const r of [...seen].sort((a, b) => b - a)) if (rows.length < FIND_RULER_MAX_ROWS) rows.push(r);
        end = start;
      }
    }
    // 같은 줄들이면(그리고 버퍼가 밀리지 않았으면) 다시 만들지 않는다.
    const key = `${term.buffer.active.baseY}:${rows.join(',')}`;
    if (key === rulerKey) return;
    rulerKey = key;
    clearList(rulerDrawn);
    rulerDrawn = [];
    for (const r of rows) {
      const d = mark(r, 0, 1, false, true);
      if (!d) break;
      rulerDrawn.push(d);
    }
  };

  const schedule = () => {
    if (disposed || !query) return;
    if (viewTimer === null) viewTimer = setTimeout(drawView, FIND_VIEW_REFRESH_MS);
    if (rulerTimer === null) rulerTimer = setTimeout(drawRuler, FIND_RULER_REFRESH_MS);
  };

  const stopTimers = () => {
    if (viewTimer !== null) clearTimeout(viewTimer);
    if (rulerTimer !== null) clearTimeout(rulerTimer);
    viewTimer = null;
    rulerTimer = null;
  };

  const subs: IDisposable[] = [
    term.onWriteParsed(schedule),
    term.onScroll(schedule),
    term.onResize(schedule),
  ];

  return {
    setQuery(q) {
      if (q === query) return;
      query = q;
      stopTimers();
      // 글이 바뀌면 곧바로 그린다(입력할 때마다 0.1초 기다리지 않게).
      drawView();
      drawRuler();
    },
    dispose() {
      disposed = true;
      stopTimers();
      for (const s of subs) s.dispose();
      clearList(viewDrawn);
      clearList(rulerDrawn);
      viewDrawn = [];
      rulerDrawn = [];
    },
  };
}
```

읽는 순서:

1. `logicalLine` — 버퍼 줄 `y` 에서 시작해 `isWrapped` 인 다음 줄들을 이어 붙인 글과, **글의 i 번째 글자가 어느 줄 어느 칸인지**의 표(`pos`)를 만든다. 너비 0 인 칸(넓은 글자의 뒤 칸)은 건너뛴다. 표의 마지막 항목은 글 끝의 위치다(일치 구간의 끝을 칸으로 바꿀 때 쓴다).
2. `findInText` — 정규식의 특수 문자를 글자로 바꿔 대소문자 무시로 찾는다. 한 줄에 200개까지.
3. `attachFindMarks` 의 `mark` — 표식은 "커서 줄 기준 몇 줄 위"로 만든다(xterm.js 의 `registerMarker` 가 그렇게 받는다). 강조를 먼저 지우고 표식을 나중에 지운다.
4. `drawView` — 보이는 줄만. 맨 윗줄이 이어진 줄의 중간이면 그 줄의 시작으로 올라가서 본다(위에서 시작해 화면으로 넘어온 낱말).
5. `drawRuler` — 맨 아래에서 위로. 찾은 줄 번호들을 `baseY:줄,줄,…` 글로 만들어 지난번과 같으면 아무것도 하지 않는다.
6. `schedule` — 타이머가 **이미 걸려 있으면 다시 걸지 않는다**. 이것이 throttle 이다: 출력이 계속 와도 처음 건 타이머가 0.1초 뒤 그리고, 그다음 출력이 새 타이머를 건다.

### `frontend/src/screens/TerminalView.tsx` — 수정

```diff
--- a/frontend/src/screens/TerminalView.tsx
+++ b/frontend/src/screens/TerminalView.tsx
@@ -15,8 +15,10 @@
 import { decodeOsc52 } from '../lib/osc52';
 import { shortcutFor } from '../lib/keys';
 import { resolveTermTheme } from '../lib/termThemes';
 import { attachHighlighter, type Highlighter } from '../lib/termHighlight';
+import { attachFindMarks, type FindMarks } from '../lib/termFind';
+import { attachCopyOnMouseSelect } from '../lib/copyOnSelect';
 import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_ACTIVE_FG, FIND_MATCH_COLOR, FIND_MATCH_RATIO, FIND_RULER_MATCH, FIND_RULER_ACTIVE, FIND_RULER_WIDTH } from '../lib/findHighlight';
 import { DANGEROUS_PATTERNS } from '../lib/defaults';
 import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';
 // 이 파일은 t 를 터미널 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
@@ -29,8 +31,17 @@
 }
 
 /** 칸 id → xterm 인스턴스. 칸 우클릭 메뉴가 복사·모두 선택·화면 지우기에 쓴다. */
 export const termHandles = new Map<string, Terminal>();
+
+/** 메뉴의 "모두 선택". 사용자가 고른 선택이므로 "선택하면 자동 복사"가 켜져 있으면 복사한다
+ *  (자동 복사는 마우스 선택에만 반응하므로 여기서 따로 한다, KI-011). */
+export function selectAllInPane(paneId: string): void {
+  const term = termHandles.get(paneId);
+  if (!term) return;
+  term.selectAll();
+  if (useApp.getState().store?.settings.copyOnSelect && term.getSelection()) void copySelection(term);
+}
 
 /**
  * 활성 탭의 포커스 칸 터미널에 키보드 포커스를 준다. 탭 클릭·서버 열기 뒤 바로 입력할 수 있게 한다.
  * 클릭의 기본 동작(포커스 이동)과 React 그리기가 끝난 다음 프레임에 준다.
@@ -187,27 +198,37 @@
         },
       });
     }
   };
+  // 찾은 항목(현재 항목 제외)의 바탕·스크롤 막대 표시. 애드온에 강조를 맡기지 않는다(KI-011):
+  // 애드온은 강조를 맡으면 출력마다 다시 찾으며 터미널 선택을 바꿔, 자동 복사·사용자 선택을 덮었다.
+  const marks = useRef<FindMarks | null>(null);
   useEffect(() => {
     const a = new SearchAddon();
     term.loadAddon(a);
     addon.current = a;
+    // 찾은 항목: 터미널 배경에 주황을 옅게 섞은 바탕(테마가 밝든 어둡든 글자가 읽힌다). 색은 #RRGGBB 만 받는다.
+    const matchBackground = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
+    const m = attachFindMarks(term, { matchBackground, ruler: FIND_RULER_MATCH });
+    marks.current = m;
     // 터미널을 눌러 선택이 풀리면 현재 항목 강조도 지운다.
     const sub = term.onSelectionChange(() => {
       if (!term.hasSelection()) clearActive();
     });
     return () => {
       sub.dispose();
       clearActive();
-      a.clearDecorations();
+      m.dispose();
+      marks.current = null;
+      a.dispose();
+      addon.current = null;
     };
     // clearActive 는 ref 만 다뤄 렌더마다 새로 만들어져도 같다.
   }, [term]);
-  // 찾은 항목: 터미널 배경에 주황을 옅게 섞은 바탕(테마가 밝든 어둡든 글자가 읽힌다). 색은 #RRGGBB 만 받는다.
-  const matchBg = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
-  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: FIND_RULER_MATCH, activeMatchColorOverviewRuler: FIND_RULER_ACTIVE, matchBackground: matchBg, activeMatchBackground: FIND_ACTIVE_BG } };
+  // 애드온은 현재 항목으로 옮겨 선택하는 일만 한다(decorations 없음 → 출력마다 다시 찾지 않는다).
+  const opts = { caseSensitive: false };
   const find = (text: string, dir: 1 | -1) => {
+    marks.current?.setQuery(text);
     const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
     if (found) markActive();
     else clearActive();
   };
@@ -216,9 +237,9 @@
     find(q, dir);
   };
   const close = () => {
     clearActive();
-    addon.current?.clearDecorations();
+    marks.current?.setQuery('');
     onClose();
     term.focus();
   };
   return (
@@ -232,9 +253,9 @@
           setQ(e.target.value);
           if (e.target.value) find(e.target.value, 1);
           else {
             clearActive();
-            addon.current?.clearDecorations();
+            marks.current?.setQuery('');
           }
         }}
         onKeyDown={(e) => {
           if (e.key === 'Escape') {
@@ -266,8 +287,10 @@
   /** 지금 이 터미널을 보여 주는 칸(입력 대상 계산용). 칸이 바뀌면 붙일 때 갱신한다. */
   paneId: string;
   /** 키워드 강조(behavior/terminal.md §9). 설정의 규칙이 바뀌면 setRules 로 알린다. */
   highlighter: Highlighter;
+  /** 마우스 선택 자동 복사 리스너. */
+  copySub: IDisposable;
 }
 const keptTerms = new Map<string, KeptTerm>();
 
 /** WebGL 렌더러를 붙인다. 만들 수 없으면(소프트웨어 렌더링·RDP 등) 조용히 DOM 렌더러를 유지하고,
@@ -357,13 +380,10 @@
     tracker.feed(data);
     for (const sid of targetsFor(k.paneId, sessionId)) getBackend().termInput(sid, data);
   });
 
-  // 선택하면 자동 복사
-  t.onSelectionChange(() => {
-    const st2 = useApp.getState().store?.settings;
-    if (st2?.copyOnSelect && t.getSelection()) void copySelection(t);
-  });
+  // 선택하면 자동 복사: 마우스로 선택을 끝냈을 때만(찾기가 바꾼 선택은 복사하지 않는다, KI-011).
+  const copySub = attachCopyOnMouseSelect(el, t, () => !!useApp.getState().store?.settings.copyOnSelect, () => void copySelection(t));
 
   // 벨: 설정이 켜져 있을 때만 짧은 소리
   t.onBell(() => {
     if (useApp.getState().store?.settings.bell) beep();
@@ -434,9 +454,9 @@
     if (next !== st3.settings.fontSize) void useApp.getState().patchStore({ settings: { ...st3.settings, fontSize: next } });
   };
   el.addEventListener('wheel', onWheel, { passive: false });
 
-  const k: KeptTerm = { term: t, fit, el, paneId, highlighter: attachHighlighter(t, st?.highlightRules) };
+  const k: KeptTerm = { term: t, fit, el, paneId, highlighter: attachHighlighter(t, st?.highlightRules), copySub };
   keptTerms.set(sessionId, k);
   return k;
 }
 
@@ -444,8 +464,9 @@
   const k = keptTerms.get(sessionId);
   if (!k) return;
   keptTerms.delete(sessionId);
   k.highlighter.dispose();
+  k.copySub.dispose();
   k.el.remove();
   k.term.dispose();
   if (useWorkspace.getState().writers[sessionId]) useWorkspace.getState().registerWriter(sessionId, null);
 }
```

바뀐 곳마다:

- **`selectAllInPane`**(새 함수, 내보냄): 칸 메뉴의 "모두 선택". 선택한 뒤 자동 복사가 켜져 있으면 복사한다.
- **`FindBar` 의 `useEffect`**: 애드온을 붙일 때 `attachFindMarks` 도 붙인다. 찾은 항목의 바탕색 계산(`mixHex`)을 여기로 옮겼다. 정리할 때 `m.dispose()` 와 **`a.dispose()`** — 전에는 `clearDecorations()` 만 불러 애드온이 터미널에 남았다.
- **`opts`**: `{ caseSensitive: false }` 뿐이다. `decorations` 가 없으므로 애드온은 강조를 만들지 않고, **출력 뒤 다시 찾지도 않는다**.
- **`find`**: 애드온으로 옮겨 가기 전에 `marks.current?.setQuery(text)` 로 찾은 항목을 그린다.
- **닫기·입력을 비움**: `addon.current?.clearDecorations()` 대신 `marks.current?.setQuery('')`.
- **`KeptTerm.copySub`**: 자동 복사 리스너. 터미널을 버릴 때 `dispose`.
- **자동 복사**: `t.onSelectionChange(…)` 를 지우고 `attachCopyOnMouseSelect(el, t, 설정을 읽는 함수, 복사하는 함수)`.

### `frontend/src/screens/Panes.tsx` — 수정

```diff
--- a/frontend/src/screens/Panes.tsx
+++ b/frontend/src/screens/Panes.tsx
@@ -7,9 +7,9 @@
 import { openHost, splitSameAs, canSplitMore } from '../state/actions';
 import { useApp } from '../state/app';
 import { useUi } from '../state/ui';
 import { ENV_INFO } from '../lib/defaults';
-import { TerminalView, termHandles, pasteFromClipboard, targetsFor, FIND_EVENT } from './TerminalView';
+import { TerminalView, termHandles, pasteFromClipboard, targetsFor, selectAllInPane, FIND_EVENT } from './TerminalView';
 import { forceImagePaste, uploadAndInsert } from '../state/paste';
 import { getBackend, errorMessage } from '../ipc/backend';
 import type { MenuItem } from '../state/ui';
 import { isFileDrag, hoverStart, hoverEnd, dropFiles } from './fileDrag';
@@ -220,9 +220,9 @@
     { label: tr('common.copy'), shortcut: 'Ctrl+Shift+C', disabled: !hasSession, onSelect: () => { const sel = termHandles.get(pane.id)?.getSelection(); if (sel) void getBackend().invoke('clip.writeText', [sel]).catch(() => undefined); } },
     { label: tr('paste.ok'), shortcut: 'Ctrl+V', disabled: !hasSession, onSelect: () => { const t = termHandles.get(pane.id); if (t && pane.sessionId) void pasteFromClipboard(pane.id, pane.sessionId, t); } },
     { label: info?.canSftp || info?.protocol === 'local' ? tr('pane.pasteImagePath') : tr('pane.pasteImageServer'), shortcut: 'Ctrl+Alt+V', disabled: !hasSession || (!info?.canSftp && info?.protocol !== 'local'), onSelect: () => { if (pane.sessionId) void forceImagePaste(pane.sessionId, targetsFor(pane.id, pane.sessionId)); } },
     { separator: true },
-    { label: tr('pane.selectAll'), disabled: !hasSession, onSelect: () => termHandles.get(pane.id)?.selectAll() },
+    { label: tr('pane.selectAll'), disabled: !hasSession, onSelect: () => selectAllInPane(pane.id) },
     { label: tr('pane.clear'), disabled: !hasSession, onSelect: () => termHandles.get(pane.id)?.clear() },
     { label: tr('pane.find'), shortcut: 'Ctrl+Shift+F', disabled: !hasSession, onSelect: () => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id })) },
     { separator: true },
     { label: tr('sc.splitRight'), shortcut: 'Ctrl+Shift+D', disabled: !info || !canSplit, onSelect: () => void splitSame('row') },
```

---

## C. 테스트 코드

### `frontend/src/lib/copyOnSelect.test.ts` — 새 파일 (전체)

```ts
// "선택하면 자동 복사"는 마우스로 고른 선택만 복사한다. 근거: spec/ui/settings.md, KI-011.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { attachCopyOnMouseSelect } from './copyOnSelect';

function setup(enabled = true) {
  const el = document.createElement('div');
  document.body.appendChild(el);
  const sel = { text: '' };
  const term = { hasSelection: () => sel.text !== '', getSelection: () => sel.text };
  const copy = vi.fn();
  const sub = attachCopyOnMouseSelect(el, term, () => enabled, copy);
  const down = (target: EventTarget = el) => target.dispatchEvent(new MouseEvent('mousedown', { button: 0, bubbles: true }));
  const up = () => window.dispatchEvent(new MouseEvent('mouseup', { button: 0, bubbles: true }));
  return { el, sel, copy, sub, down, up };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  vi.useRealTimers();
  document.body.innerHTML = '';
});

describe('선택하면 자동 복사', () => {
  it('터미널에서 끌어 선택하고 버튼을 떼면 한 번 복사한다', () => {
    const f = setup();
    f.down();
    f.sel.text = 'abc';
    f.up();
    vi.runAllTimers();
    expect(f.copy).toHaveBeenCalledTimes(1);
  });

  it('코드가 바꾼 선택(찾기의 현재 항목)은 복사하지 않는다', () => {
    const f = setup();
    f.sel.text = 'found';
    // 마우스 누름 없이 버튼 뗌만 와도(다른 곳 클릭의 끝) 복사하지 않는다.
    f.up();
    vi.runAllTimers();
    expect(f.copy).not.toHaveBeenCalled();
  });

  it('터미널 밖에서 누른 클릭은 세지 않고, 선택이 없으면 복사하지 않는다', () => {
    const f = setup();
    const other = document.createElement('div');
    document.body.appendChild(other);
    f.sel.text = 'x';
    f.down(other);
    f.up();
    f.down();
    f.sel.text = '';
    f.up();
    vi.runAllTimers();
    expect(f.copy).not.toHaveBeenCalled();
  });

  it('설정이 꺼져 있으면 복사하지 않고, dispose 뒤에는 듣지 않는다', () => {
    const off = setup(false);
    off.down();
    off.sel.text = 'abc';
    off.up();
    vi.runAllTimers();
    expect(off.copy).not.toHaveBeenCalled();

    const on = setup(true);
    on.sub.dispose();
    on.down();
    on.sel.text = 'abc';
    on.up();
    vi.runAllTimers();
    expect(on.copy).not.toHaveBeenCalled();
  });
});
```

### `frontend/src/lib/termFind.test.ts` — 새 파일 (전체)

가짜 터미널(줄 배열 + `registerMarker`·`registerDecoration` 을 세는 것)로 시험한다. 실제 xterm.js 는 jsdom 에서 그리기를 하지 않는다.

```ts
// 터미널 찾기의 찾은 항목 표시 검사(가짜 터미널). 근거: spec/ui/panes.md §2.4, KI-011.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Terminal } from '@xterm/xterm';
import { attachFindMarks, findInText, logicalLine, FIND_RULER_MAX_ROWS, FIND_RULER_REFRESH_MS, FIND_VIEW_REFRESH_MS } from './termFind';

interface Deco { x: number; width: number; bg?: string; ruler?: string; line: number; disposed: boolean }
interface Row { s: string; wrapped?: boolean }

const COLS = 10;

function fakeLine(r: Row) {
  return {
    length: COLS,
    isWrapped: !!r.wrapped,
    getCell: (x: number) => ({ getChars: () => r.s[x] ?? '', getWidth: () => 1 }),
    translateToString: () => r.s,
  };
}

/** 줄 배열을 가진 가짜 터미널. 표식 줄 번호는 버퍼 줄 번호 그대로 기록한다(baseY·cursorY = 0). */
function fakeTerm(rows: Row[], view: { rows: number; viewportY: number }) {
  const decos: Deco[] = [];
  const handlers: Record<string, (() => void)[]> = { write: [], scroll: [], resize: [] };
  const on = (k: string) => (fn: () => void) => {
    handlers[k].push(fn);
    return { dispose: () => { handlers[k] = handlers[k].filter((f) => f !== fn); } };
  };
  const state = { lines: rows, viewportY: view.viewportY };
  const term = {
    rows: view.rows,
    cols: COLS,
    buffer: {
      get active() {
        return {
          viewportY: state.viewportY,
          baseY: 0,
          cursorY: 0,
          length: state.lines.length,
          getLine: (y: number) => (state.lines[y] ? fakeLine(state.lines[y]) : undefined),
        };
      },
    },
    registerMarker: (offset: number) => ({ line: offset, dispose: () => undefined }),
    registerDecoration: (o: { marker: { line: number }; x: number; width: number; backgroundColor?: string; overviewRulerOptions?: { color: string } }) => {
      const d: Deco = { x: o.x, width: o.width, bg: o.backgroundColor, ruler: o.overviewRulerOptions?.color, line: o.marker.line, disposed: false };
      decos.push(d);
      return { dispose: () => { d.disposed = true; } };
    },
    onWriteParsed: on('write'),
    onScroll: on('scroll'),
    onResize: on('resize'),
  };
  return {
    term: term as unknown as Terminal,
    state,
    live: () => decos.filter((d) => !d.disposed),
    created: () => decos.length,
    fire: (k: 'write' | 'scroll' | 'resize') => handlers[k].forEach((f) => f()),
    handlerCount: () => handlers.write.length + handlers.scroll.length + handlers.resize.length,
  };
}

const colors = { matchBackground: '#553311', ruler: '#f59e0b' };

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('찾기 표시: 글 계산', () => {
  it('대소문자를 무시하고 특수 문자는 글자 그대로 찾는다', () => {
    expect(findInText('Error error ERROR', 'error')).toEqual([{ start: 0, end: 5 }, { start: 6, end: 11 }, { start: 12, end: 17 }]);
    expect(findInText('a.b axb', 'a.b')).toEqual([{ start: 0, end: 3 }]);
    expect(findInText('abc', '')).toEqual([]);
  });

  it('다음 줄로 이어진 글은 한 줄로 이어 붙인다', () => {
    const rows: Row[] = [{ s: 'xxxxxxxerr' }, { s: 'or here', wrapped: true }, { s: 'next' }];
    const ll = logicalLine((y) => (rows[y] ? fakeLine(rows[y]) : undefined), 0, COLS);
    expect(ll.text.startsWith('xxxxxxxerror here')).toBe(true);
    expect(ll.next).toBe(2);
    expect(ll.pos[7]).toEqual({ x: 7, y: 0 });
    expect(ll.pos[10]).toEqual({ x: 0, y: 1 });
  });
});

describe('찾기 표시: 터미널에 그리기', () => {
  it('글을 넣으면 곧바로 화면의 일치 항목에 바탕을 칠하고, 줄을 넘는 항목은 줄마다 칠한다', () => {
    const f = fakeTerm([{ s: 'xxxxxxxerr' }, { s: 'or ok', wrapped: true }, { s: 'ERROR' }], { rows: 3, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('error');
    const bg = f.live().filter((d) => d.bg).map((d) => ({ line: d.line, x: d.x, width: d.width }));
    expect(bg).toEqual([
      { line: 0, x: 7, width: 3 },
      { line: 1, x: 0, width: 2 },
      { line: 2, x: 0, width: 5 },
    ]);
  });

  it('출력이 쉬지 않고 이어져도 정해진 간격마다 다시 그린다(미루지 않는다)', () => {
    const f = fakeTerm([{ s: 'ok' }], { rows: 1, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('fail');
    expect(f.live().filter((d) => d.bg)).toHaveLength(0);
    f.state.lines = [{ s: 'fail' }];
    // 출력이 FIND_VIEW_REFRESH_MS 보다 짧은 간격으로 계속 온다.
    for (let i = 0; i < 5; i++) {
      f.fire('write');
      vi.advanceTimersByTime(FIND_VIEW_REFRESH_MS / 4);
    }
    expect(f.live().filter((d) => d.bg)).toHaveLength(1);
  });

  it('버퍼 맨 위가 아니라 최근 줄부터 스크롤 막대에 표시하고, 한도를 넘지 않는다', () => {
    const rows: Row[] = Array.from({ length: FIND_RULER_MAX_ROWS + 50 }, () => ({ s: 'hit' }));
    const f = fakeTerm(rows, { rows: 2, viewportY: rows.length - 2 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    const ruler = f.live().filter((d) => d.ruler).map((d) => d.line);
    expect(ruler).toHaveLength(FIND_RULER_MAX_ROWS);
    expect(Math.max(...ruler)).toBe(rows.length - 1);
    expect(Math.min(...ruler)).toBe(50);
  });

  it('스크롤 막대 표시는 같은 줄들이면 다시 만들지 않는다', () => {
    const f = fakeTerm([{ s: 'hit' }, { s: 'x' }], { rows: 2, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    const before = f.created();
    f.fire('write');
    vi.advanceTimersByTime(FIND_RULER_REFRESH_MS);
    // 화면 바탕 강조 1개만 다시 만들어지고 스크롤 막대 표시는 그대로다.
    expect(f.created() - before).toBe(1);
  });

  it('글을 비우거나 dispose 하면 표시를 모두 지우고 구독을 푼다', () => {
    const f = fakeTerm([{ s: 'hit' }], { rows: 1, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    expect(f.live().length).toBeGreaterThan(0);
    m.setQuery('');
    expect(f.live()).toHaveLength(0);
    m.setQuery('hit');
    m.dispose();
    expect(f.live()).toHaveLength(0);
    expect(f.handlerCount()).toBe(0);
  });

  it('글이 없으면 출력이 와도 타이머를 걸지 않는다', () => {
    const f = fakeTerm([{ s: 'hit' }], { rows: 1, viewportY: 0 });
    attachFindMarks(f.term, colors);
    f.fire('write');
    expect(vi.getTimerCount()).toBe(0);
  });
});
```

| 시험 | 본 것 |
|---|---|
| 대소문자를 무시하고 특수 문자는 글자 그대로 찾는다 | `findInText` |
| 다음 줄로 이어진 글은 한 줄로 이어 붙인다 | `logicalLine` — 줄 끝에 걸친 낱말의 글 위치 → 칸 위치 |
| 글을 넣으면 곧바로 화면의 일치 항목에 바탕을 칠하고, 줄을 넘는 항목은 줄마다 칠한다 | `setQuery` 가 기다리지 않고 그리는 것, 줄 넘김 |
| 출력이 쉬지 않고 이어져도 정해진 간격마다 다시 그린다(미루지 않는다) | **증상 2** — throttle |
| 버퍼 맨 위가 아니라 최근 줄부터 스크롤 막대에 표시하고, 한도를 넘지 않는다 | **증상 4** |
| 스크롤 막대 표시는 같은 줄들이면 다시 만들지 않는다 | 1초마다 수백 개를 지우고 만드는 일을 피한다 |
| 글을 비우거나 dispose 하면 표시를 모두 지우고 구독을 푼다 | 찾기 막대를 닫으면 남는 표시가 없다 |
| 글이 없으면 출력이 와도 타이머를 걸지 않는다 | 찾기 막대를 안 쓸 때는 아무 일도 하지 않는다 |

---

## F. 명세

### `spec/ui/panes.md` — 수정

```diff
--- a/spec/ui/panes.md
+++ b/spec/ui/panes.md
@@ -60,8 +60,10 @@
 
 ### 2.4 찾기 막대
 - 칸 위에 뜨는 입력 막대: 입력창(플레이스홀더 "터미널에서 찾기"), [이전] [다음] [닫기] 버튼. 열리면 입력창에 포커스.
 - 대소문자 구분 없이 검색. Enter=다음, Shift+Enter=이전. 찾은 항목은 주황 계열로 강조하고 현재 항목은 빨강 계열로 강조(스크롤 막대에도 표시).
+  - (2026-11-25, D120-1) 찾은 항목의 강조는 화면에 보이는 줄에서 0.1초 간격으로, 스크롤 막대 표시는 최근 줄부터 1000줄까지 1초 간격으로 다시 그린다 — 출력이 이어져도 미루지 않는다. 화면 폭 때문에 다음 줄로 이어진 글도 찾는다.
+  - 찾기는 현재 항목으로 옮길 때([다음]·[이전]·입력)만 터미널 선택을 바꾼다. 출력이 와도 선택을 다시 바꾸지 않으므로, 막대가 열린 동안 마우스로 한 선택은 그대로 남는다.
 - Esc 또는 닫기: 막대를 닫고 강조를 지우며 터미널에 포커스를 돌려준다(닫기 버튼은 포커스 이동 없음).
 - 여는 방법: 칸 머리글 버튼, 우클릭 메뉴, 단축키(포커스된 칸 대상).
 
 ### 2.5 파일 끌어 놓기
```

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -34,8 +34,9 @@
 
 ## 2. 입력·붙여넣기
 카드 안 토글(설명 포함):
 1. **선택하면 자동 복사** — "드래그로 선택하면 즉시 클립보드에 복사합니다 (PuTTY 방식)"
+   - (2026-11-25, D120-1) 마우스로 선택을 끝냈을 때(끌기·두 번/세 번 클릭 뒤 왼쪽 버튼을 뗄 때) 한 번 복사하고, 칸 메뉴의 "모두 선택"도 복사한다. 앱이 코드로 바꾼 선택(찾기의 현재 항목)은 복사하지 않는다.
 2. **오른쪽 클릭으로 붙여넣기** — "끄면 오른쪽 클릭에 메뉴가 뜹니다. 켜져 있어도 Shift+오른쪽 클릭은 메뉴입니다"
 3. **Ctrl+V로 붙여넣기** (Windows 에서만 표시) — "끄면 Ctrl+V 는 그대로 서버로 보내고, 붙여넣기는 Ctrl+Shift+V 로 합니다"
 4. **Shift+Enter 로 줄바꿈** — "실행하지 않고 줄만 바꿉니다. Claude Code 같은 CLI의 여러 줄 입력용입니다 (Alt+Enter 와 같은 값 전송)"
 5. **원격 복사 허용 (OSC 52)** — "서버 쪽 tmux·vim·Claude Code 등의 클립보드 쓰기 요청을 이 PC 클립보드에 반영합니다. 읽기 요청은 항상 거부합니다"
```

---

## G. 검증 (2026-11-25 실행)

환경: Windows 11, Node v26.10.0. vitest 는 PowerShell 에서 돌린다(Git Bash 에서는 모든 시험이 "failed to find the current suite" 로 실패한다).

```
cd frontend
npx tsc --noEmit        통과
npx vitest run          Test Files  49 passed (49)
                        Tests  311 passed (311)
```

고치기 전에는 47개 파일 299개였다(새 파일 2개, 시험 12개가 늘었다).

`build/bin/dariterm.exe` 는 **다시 빌드하지 않았다**. 앱에서 확인하려면 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build`.

## H. 수동 검증 (남은 일)

설정 > 입력·붙여넣기 > "선택하면 자동 복사"가 **켜진** 상태(기본값)에서 한다.

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 메모장에서 `CLIP-TEST` 를 복사 → 터미널에서 Ctrl+Shift+F → `e` 를 친다 → [다음] 여러 번 → 메모장에 붙여넣기 | `CLIP-TEST` 가 붙는다(**찾기가 클립보드를 바꾸지 않았다**) |
| 2 | `while true; do date; sleep 0.05; done` 을 돌려 두고 → 메모장에서 `CLIP-TEST` 복사 → 찾기에 `2026` → 10초 기다림 → 메모장에 붙여넣기 | `CLIP-TEST` 가 붙는다. 새로 올라오는 줄의 `2026` 이 **계속 강조된다**(출력이 멈추지 않아도) |
| 3 | 2번 상태에서 마우스로 화면의 다른 글을 끌어 선택 | 그 글이 복사된다. 선택이 그대로 남고, **찾은 항목으로 바뀌지 않는다** |
| 4 | `seq 1 5000` → 찾기에 `1` | 화면에 보이는 `1` 이 모두 강조된다(마지막 줄 근처 포함). 스크롤 막대에 표시가 **아래쪽(최근)** 에 몰려 있다 |
| 5 | 4번에서 위로 스크롤 | 0.1초쯤 뒤 새로 보이는 줄의 `1` 에 바탕이 칠해진다 |
| 6 | 창을 좁혀 긴 줄이 다음 줄로 넘어가게 한 뒤, 줄 끝에 걸친 낱말을 찾는다 | 두 줄에 걸쳐 강조된다 |
| 7 | [다음]·[이전] | 현재 항목이 빨강으로 옮겨 간다(전과 같다). 스크롤 막대에 빨강 표시 |
| 8 | 찾기 막대를 닫는다(Esc) | 강조와 스크롤 막대 표시가 모두 사라진다 |
| 9 | 낱말을 두 번 클릭 / 줄을 세 번 클릭 / 끌다가 터미널 밖에서 버튼을 뗌 | 각각 선택된 글이 복사된다 |
| 10 | 칸 우클릭(또는 Shift+우클릭) → "모두 선택" → 메모장에 붙여넣기 | 화면 내용 전체가 붙는다 |
| 11 | "선택하면 자동 복사"를 끄고 9·10번 | 복사되지 않는다(Ctrl+Shift+C 는 된다) |
| 12 | `vim` 을 연 채 찾기 | 오류 없이 동작한다(대체 화면에서는 찾은 항목 바탕이 안 그려질 수 있다 — 현재 항목 이동은 된다) |
| 13 | 다크·라이트 테마, 터미널 색상 테마를 바꿔 가며 4번 | 찾은 항목의 글자가 읽힌다(바탕은 터미널 배경에 주황을 섞은 색, D96) |

## I. 영향 범위·회귀 위험

| 위험 | 정도 | 대비 |
|---|---|---|
| 마우스가 아닌 방법으로 한 선택이 복사되지 않는다 | 낮음 | 이 앱에서 사용자가 선택하는 방법은 마우스와 칸 메뉴 "모두 선택"뿐이고 둘 다 복사한다 |
| 터치·펜 입력 | 낮음 | 브라우저가 터치를 마우스 이벤트로도 보낸다. 앱은 Windows 데스크톱용이다 |
| 출력이 아주 빠를 때 0.1초마다 화면을 훑는 비용 | 낮음 | 보이는 줄(수십 줄)만 훑고 한 화면 400곳에서 멈춘다. 찾기 막대에 글이 없으면 타이머를 걸지 않는다 |
| 스크롤 막대를 1초마다 버퍼 전체 훑기 | 낮음 | 글로 바꿔 빨리 거르고, 찾은 줄이 1000줄이면 멈춘다. 결과가 같으면 다시 만들지 않는다 |
| 찾은 항목이 1000줄을 넘으면 오래된 쪽은 스크롤 막대에 안 나온다 | 알려진 한계 | [다음]·[이전] 이동은 버퍼 전체에서 된다(애드온) |
| 애드온 버전을 올렸을 때 | 낮음 | 공개 기능(`findNext`·`findPrevious`·`caseSensitive`)만 쓴다 |
| 키워드 강조(D113)와 겹침 | 없음 | 서로 다른 강조다. 둘 다 표식에 붙고 버퍼를 바꾸지 않는다 |

되돌리기: 이 문서의 패치를 거꾸로 적용하고 새 파일 4개를 지운다.

## J. 체크리스트

- [x] `termFind.ts`·`copyOnSelect.ts` 와 시험
- [x] `TerminalView.tsx`·`Panes.tsx`
- [x] `tsc --noEmit`, vitest 49파일 311개
- [x] 명세 ui/panes.md §2.4 · ui/settings.md §2, decisions D120-1, known-issues KI-011 resolved, CHANGELOG, CLAUDE.md
- [ ] 앱 빌드 후 수동 검증(§H)
- [ ] 커밋(사용자 요청 시)
