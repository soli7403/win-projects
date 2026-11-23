# 창 머리글의 최대화·복원 아이콘이 구분되지 않는 문제 — 원인·수정·코드·검증

**작성**: 2026-11-23 · **상태**: 코드 수정·자동 테스트·빌드 완료(D121). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(보고 즉시 수정해 등록하지 않음)
**관련**: [spec/ui/panes.md](../../spec/ui/panes.md) §2.1 7번(크게 보기/원래대로)

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

탭 안의 창이 2개 이상일 때 창 머리글에 나오는 **[최대화] / [복원] 버튼의 두 아이콘이 거의 같아 보였다.** 두 가지를 바꿨다.

| | 고치기 전 | 고친 뒤 |
|---|---|---|
| 평소(최대화 버튼) | 바깥을 향한 대각선 화살표 두 개 | **빈 사각형 하나** |
| 크게 보는 중(복원 버튼) | 안쪽을 향한 대각선 화살표 두 개 | **겹친 사각형 두 개** + **켜진 버튼 표시**(강조색 옅은 바탕, 강조색 아이콘) |

Windows 창의 최대화·복원 버튼과 같은 관례다. 툴팁·단축키(Ctrl+Shift+Enter)·동작은 그대로다.

## 1. 사용자 보고

2026-11-23: "최대화 최소화 아이콘이 구분이 잘 되지 않는다." → 방법 세 가지(모양 바꾸기 / 켜진 상태 표시 / 크기 키우기)를 제안했고 사용자가 **1번(모양)과 2번(켜진 표시)** 을 골랐다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 창(칸) | 탭 안을 나눈 한 조각. 창마다 위에 머리글(제목 + 아이콘 버튼 줄)이 있다 |
| 크게 보기(최대화) | 그 탭에서 한 창만 탭 전체 크기로 보이게 하는 것. 다른 창은 닫히지 않고 가려질 뿐이다. 코드에서는 `tab.zoomed` 에 그 창의 id 가 들어 있다 |
| 원래대로(복원) | 크게 보기를 풀어 창들이 다시 나뉘어 보이게 하는 것 |

앱 창 자체의 최소화·최대화 버튼(Windows 제목 표시줄)은 운영체제가 그리는 것이라 이 문서와 관계없다.

## 2. 원인 분석 (Cause)

고치기 전 코드([frontend/src/screens/Panes.tsx](../../frontend/src/screens/Panes.tsx) 의 창 머리글):

```tsx
<button className="icon-btn small" title={tab.zoomed ? tr('pane.restoreTitle') : tr('pane.maximizeTitle')} …>
  {tab.zoomed ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
</button>
```

| # | 문제 | 왜 |
|---|---|---|
| 1 | **두 아이콘의 윤곽이 같다** | `Maximize2` 와 `Minimize2` 는 둘 다 "대각선 화살표 두 개"이고 화살촉이 바깥을 향하느냐 안쪽을 향하느냐만 다르다. 선의 자리와 전체 모양이 같다 |
| 2 | **너무 작다** | 12px 로 그린다. 이 크기에서 화살촉은 2~3px 이라 방향이 읽히지 않는다 |
| 3 | **상태 표시가 아이콘뿐이다** | 크게 보는 중인지 아닌지를 알려 주는 것이 그 아이콘 하나였다. 색·바탕은 두 상태가 같았다. 머리글의 다른 토글(동시 입력 등)은 켜지면 `.on` 으로 바탕색이 바뀌는데 이 버튼만 빠져 있었다 |

덧붙여, 크게 보는 중에는 다른 창이 화면에서 사라지는데 그 사실을 알려 주는 표시가 머리글에 없었다(탭의 창 수 배지만 남는다).

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| **1. 아이콘 모양을 다르게** | 최대화 = 빈 사각형 하나, 복원 = 겹친 사각형 두 개 | **고름(사용자 선택).** 윤곽 자체가 달라 12px 에서도 구분된다. Windows 창 버튼과 같아 따로 배울 것이 없다 |
| **2. 켜진 상태를 버튼에 표시** | 크게 보는 중이면 버튼에 `.on`(강조색 옅은 바탕 + 강조색 아이콘) | **고름(사용자 선택).** 이미 있는 스타일을 쓴다. 다른 창이 가려져 있다는 것도 함께 알린다 |
| 3. 아이콘을 키운다 | 12px → 14px | 고르지 않음. 모양이 같으면 효과가 작고 머리글의 다른 아이콘(찾기·나누기·닫기)과 크기가 어긋난다 |

### 3.1 세부 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 최대화 아이콘 | lucide 의 `Square` (12px) | 이미 쓰는 아이콘 묶음에 있다 |
| 복원 아이콘 | 직접 그린 작은 SVG(`RestoreIcon`): 앞 사각형 + 뒤 사각형의 보이는 두 변 | lucide 의 `Copy` 가 비슷한 모양이지만 "복사"로 읽힌다. 뒤 사각형을 오른쪽 위로 두어 Windows 의 복원 버튼과 같은 방향으로 그렸다. 선 굵기(2)·둥근 끝·24 격자는 lucide 와 같게 맞췄다 |
| 켜진 표시 | 기존 `.icon-btn.on` 그대로 (새 스타일 없음) | 동시 입력·SFTP 버튼 등 다른 켜진 아이콘과 같은 모양이어야 한다 |
| 버튼을 따로 떼기 | `PaneZoomButton({ zoomed, onToggle })` 로 내보냄 | 창 전체(`PaneView`)는 터미널까지 그려야 해서 테스트가 무겁다. 버튼만 떼면 두 상태를 바로 그려 확인할 수 있다 |
| 접근성 | `aria-pressed` 를 더했다. 이름(`aria-label`)은 지금처럼 "최대화" | 토글 버튼의 표준 표현이다. 문구를 새로 만들지 않는다 |
| 문구·단축키·동작 | 그대로 | 보이는 모양만 바꾼다 |

### 3.2 영향 범위

프런트엔드의 파일 하나(`Panes.tsx`)와 명세 한 줄. 스타일 파일·문구·백엔드·저장 형식은 바뀌지 않는다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/screens/Panes.tsx` | 코드 | 수정 |
| `frontend/src/screens/Panes.zoomButton.test.tsx` | 테스트 | 새 파일 |
| `spec/ui/panes.md` | 명세 | 수정 |
| `docs/decisions.md` | 문서 | D121 추가 |

아래의 `diff` 는 **같은 날의 앞 문서들(D106~D120)을 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `frontend/src/screens/Panes.tsx` — 수정

```diff
--- a/frontend/src/screens/Panes.tsx
+++ b/frontend/src/screens/Panes.tsx
@@ -1,6 +1,6 @@
 // 칸·분할 화면. 근거: spec/ui/panes.md §2, §4(크기 조절), §7(끌어 놓기), spec/ui/tokens.md §3(칸·분할선).
 import { useRef, type DragEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
-import { Link, Rows2, Maximize2, Minimize2, RotateCw, X, Search, ZoomIn, Image as ImageIcon } from 'lucide-react';
+import { Link, Rows2, Square, RotateCw, X, Search, ZoomIn, Image as ImageIcon } from 'lucide-react';
 import type { LayoutNode, PaneNode, SplitNode, Dir } from '../lib/layout';
 import { edgeToSplit, edgeZone, findPane, paneIds, type Edge } from '../lib/layout';
 import { groupOf, useWorkspace, type Tab } from "../state/workspace";
@@ -88,6 +88,37 @@
   );
 }
 
+/**
+ * "원래대로" 아이콘: 겹친 사각형 두 개(Windows 창의 복원 버튼과 같은 모양). 선 굵기·모서리는 다른 머리글 아이콘(lucide)과 맞춘다.
+ * 최대화의 빈 사각형 하나(Square)와 윤곽이 달라 12px 에서도 구분된다 (D121).
+ */
+function RestoreIcon({ size }: { size: number }) {
+  return (
+    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" data-icon="restore">
+      <rect x="3" y="8" width="13" height="13" rx="2" />
+      <path d="M8 8V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-3" />
+    </svg>
+  );
+}
+
+/**
+ * 창 머리글의 크게 보기/원래대로 버튼 (ui/panes.md §2.1 7번, D121).
+ * 평소: 빈 사각형 하나. 크게 보는 중: 겹친 사각형 두 개 + 켜진 버튼 표시(강조색 바탕) — 다른 창이 가려져 있음을 알린다.
+ */
+export function PaneZoomButton({ zoomed, onToggle }: { zoomed: boolean; onToggle: () => void }) {
+  return (
+    <button
+      className={`icon-btn small${zoomed ? ' on' : ''}`}
+      title={zoomed ? tr('pane.restoreTitle') : tr('pane.maximizeTitle')}
+      aria-label={tr('pane.maximize')}
+      aria-pressed={zoomed}
+      onClick={onToggle}
+    >
+      {zoomed ? <RestoreIcon size={12} /> : <Square size={12} data-icon="maximize" />}
+    </button>
+  );
+}
+
 function PaneView({ tab, pane, total }: { tab: Tab; pane: PaneNode; total: number }) {
   const entry = useWorkspace((s) => (pane.sessionId ? s.sessions[pane.sessionId] : undefined));
   const focused = tab.focusedPane === pane.id;
@@ -247,9 +278,7 @@
           </>
         )}
         {total >= 2 && (
-          <button className="icon-btn small" title={tab.zoomed ? tr('pane.restoreTitle') : tr('pane.maximizeTitle')} aria-label={tr('pane.maximize')} onClick={() => { focusPane(pane.id); toggleZoom(tab.id); }}>
-            {tab.zoomed ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
-          </button>
+          <PaneZoomButton zoomed={!!tab.zoomed} onToggle={() => { focusPane(pane.id); toggleZoom(tab.id); }} />
         )}
         <button
           className="icon-btn small"
```


## C. 테스트 코드

### `frontend/src/screens/Panes.zoomButton.test.tsx` — 새 파일

```tsx
// 창 머리글의 크게 보기/원래대로 버튼: 두 상태의 아이콘 윤곽이 다르고, 크게 보는 중에는 켜진 모양이다 (D121, spec/ui/panes.md §2.1 7번).
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { PaneZoomButton } from './Panes';

/** 버튼 안 아이콘이 그리는 도형의 종류와 개수(윤곽 비교용). */
function shapes(btn: HTMLElement): string {
  const svg = btn.querySelector('svg')!;
  return [...svg.children].map((c) => c.tagName.toLowerCase()).sort().join(',');
}

describe('PaneZoomButton (D121)', () => {
  it('평소: 빈 사각형 하나, 꺼진 모양, 툴팁 "최대화"', () => {
    render(<PaneZoomButton zoomed={false} onToggle={() => undefined} />);
    const btn = screen.getByRole('button', { name: '최대화' });
    expect(btn.className).not.toContain(' on');
    expect(btn.getAttribute('aria-pressed')).toBe('false');
    expect(btn.getAttribute('title')).toBe('최대화 (Ctrl+Shift+Enter)');
    expect(btn.querySelector('[data-icon="maximize"]')).toBeTruthy();
    expect(shapes(btn)).toBe('rect');
  });

  it('크게 보는 중: 겹친 사각형 두 개, 켜진 모양, 툴팁 "복원"', () => {
    render(<PaneZoomButton zoomed onToggle={() => undefined} />);
    const btn = screen.getByRole('button', { name: '최대화' });
    expect(btn.className).toContain(' on');
    expect(btn.getAttribute('aria-pressed')).toBe('true');
    expect(btn.getAttribute('title')).toBe('복원 (Ctrl+Shift+Enter)');
    expect(btn.querySelector('[data-icon="restore"]')).toBeTruthy();
    expect(shapes(btn)).toBe('path,rect');
  });

  it('두 상태의 아이콘은 서로 다른 도형이다', () => {
    const a = render(<PaneZoomButton zoomed={false} onToggle={() => undefined} />);
    const off = a.container.querySelector('svg')!.innerHTML;
    a.unmount();
    const b = render(<PaneZoomButton zoomed onToggle={() => undefined} />);
    const on = b.container.querySelector('svg')!.innerHTML;
    expect(on).not.toBe(off);
  });

  it('누르면 onToggle 을 부른다', () => {
    const onToggle = vi.fn();
    render(<PaneZoomButton zoomed={false} onToggle={onToggle} />);
    fireEvent.click(screen.getByRole('button', { name: '최대화' }));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});
```


## F. 명세

### `spec/ui/panes.md` — 수정

```diff
--- a/spec/ui/panes.md
+++ b/spec/ui/panes.md
@@ -23,7 +23,7 @@
   5. ~~오른쪽으로 분할~~ — **머리글 아이콘 없음**(2026-11-12 사용자 요청: 좌우로 늘릴 필요 없음). 칸 우클릭 메뉴·Ctrl+Shift+D 는 남아 있다.
   6. **아래로 분할** — 이 칸에 포커스를 주고 **빈 칸**을 아래에 붙인다(세션을 열지 않는다). 툴팁 "아래로 나누기 (Ctrl+Shift+E)". **탭의 칸 수가 최대 화면 분할(설정 maxVerticalSplits, 1~5, 기본 3)에 이르면 비활성**, 툴팁 "아래로 나누기 (최대 N개)".
      - 같은 한도를 칸 우클릭 메뉴 "아래로/오른쪽으로 나누기 (같은 서버)"(비활성), Ctrl+Shift+D/E(안내 토스트 "화면 나누기는 최대 N개까지입니다"), 분할해 열기(빠른 접속 Ctrl/Shift+Enter, 가장자리 끌어 놓기: 새 탭에서 열고 토스트 "화면 나누기는 최대 N개까지라 새 탭에서 열었습니다")에 적용한다. 빈 칸 채우기는 한도와 무관.
-  7. **크게 보기/원래대로** — 탭에 칸이 2개 이상일 때만. 툴팁 "최대화 (Ctrl+Shift+Enter)" / "복원 (Ctrl+Shift+Enter)"
+  7. **크게 보기/원래대로** — 탭에 칸이 2개 이상일 때만. 툴팁 "최대화 (Ctrl+Shift+Enter)" / "복원 (Ctrl+Shift+Enter)". 아이콘은 두 상태의 **윤곽이 달라야** 한다(2026-11-23 사용자 지시, D121): 평소에는 **빈 사각형 하나**, 크게 보는 중에는 **겹친 사각형 두 개**(Windows 창의 최대화·복원 버튼과 같은 관례). 크게 보는 중에는 버튼을 **켜진 모양**(강조색 옅은 바탕 + 강조색 아이콘, 다른 켜진 아이콘 버튼과 같은 모양)으로 보여 다른 칸이 가려져 있음을 알린다.
   8. **닫기** — 툴팁 "닫기"
 - 머리글은 칸이 2개 이상일 때 **끌어서 옮길 수 있다**(§7).
 
```

---

## G. 검증 (2026-11-23 실행)

새 테스트 파일 [frontend/src/screens/Panes.zoomButton.test.tsx](../../frontend/src/screens/Panes.zoomButton.test.tsx) (4개).

| 테스트 | 확인하는 것 |
|---|---|
| 평소: 빈 사각형 하나, 꺼진 모양, 툴팁 "최대화" | 클래스에 `on` 없음 / `aria-pressed="false"` / 툴팁 "최대화 (Ctrl+Shift+Enter)" / 아이콘이 사각형 하나 |
| 크게 보는 중: 겹친 사각형 두 개, 켜진 모양, 툴팁 "복원" | 클래스에 `on` / `aria-pressed="true"` / 툴팁 "복원 (Ctrl+Shift+Enter)" / 아이콘이 사각형 + 선 |
| 두 상태의 아이콘은 서로 다른 도형이다 | 두 SVG 의 내용이 다르다 |
| 누르면 onToggle 을 부른다 | 버튼을 떼어 낸 뒤에도 누름이 전해진다 |

```bash
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 46개 파일 295개 통과 (앞 문서까지 45개 파일 291개 + 이 문서 4개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

Go 코드는 바뀌지 않아 `go test` 는 다시 돌리지 않았다.

자동 테스트가 확인하지 못하는 것: **눈으로 봤을 때 실제로 구분되는지**(이 수정의 목적). 12px 에서 겹친 사각형의 선이 뭉개지지 않는지, 다크·라이트 테마에서 켜진 바탕색이 충분히 보이는지는 화면에서 봐야 한다.

## H. 수동 검증 (남은 일)

준비: 한 탭에서 창 머리글의 [아래로 나누기]로 창을 2개 이상 만든다.

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 창 머리글의 최대화 버튼을 본다 | 빈 사각형 하나. 다른 머리글 아이콘과 크기·굵기가 어울린다 |
| 2 | 그 버튼을 누른다 | 그 창이 탭 전체로 커지고, 버튼이 **겹친 사각형 두 개**로 바뀌며 **강조색 옅은 바탕**이 깔린다. 툴팁은 "복원 (Ctrl+Shift+Enter)" |
| 3 | 다시 누른다 | 창들이 다시 나뉘어 보이고 버튼이 빈 사각형·바탕 없음으로 돌아간다 |
| 4 | Ctrl+Shift+Enter 로 켜고 끈다 | 버튼 모양이 2·3 과 같이 바뀐다 |
| 5 | 라이트 테마로 바꿔 1~3 을 다시 | 두 상태가 구분되고 켜진 바탕이 보인다 |
| 6 | 화면 배율 100%·125%·150% 에서 본다 | 겹친 사각형의 두 선이 붙어 보이지 않는다 |
| 7 | 창이 하나뿐인 탭 | 버튼이 없다(전과 같다) |
| 8 | 마우스를 창 밖으로 뺀다(포커스 없는 창) | 머리글 아이콘이 흐려지는 것은 전과 같다 |

6번에서 선이 뭉개지면: `RestoreIcon` 의 앞 사각형을 `x="3" y="9" width="12" height="12"` 로 줄여 두 사각형 사이를 넓히거나, 이 버튼만 13px 로 그린다.

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 크게 보기의 동작 | 누름 처리는 그대로(`focusPane` → `toggleZoom`)이고 버튼만 컴포넌트로 뗐다 | "누르면 onToggle 을 부른다" 테스트, 수동 2~4 |
| 머리글의 다른 버튼 | 건드리지 않았다 | `tsc`, 기존 테스트 |
| 켜진 표시의 흐림 처리 | 머리글 아이콘은 창에 마우스가 없고 포커스도 없으면 55% 로 흐려진다. 크게 보는 창은 늘 포커스 창이라 흐려지지 않는다 | 수동 2 |
| 아이콘 묶음 | `Maximize2`·`Minimize2` 를 이 파일에서 더는 쓰지 않는다(가져오기에서 뺐다). `Square` 는 lucide 에 있는 것 | `tsc` 통과 |
| 스토어 스크린샷 | 창 머리글이 찍힌 스크린샷이 있으면 아이콘 모양이 달라진다 | 확인하지 않았다 |

되돌리기: §B 의 `-` 줄로 되돌리고 테스트 파일을 지운다.

## J. 체크리스트

- [x] 원인 확인(윤곽이 같은 두 아이콘, 12px, 상태 표시 없음)
- [x] `Panes.tsx`: `RestoreIcon`, `PaneZoomButton`, 머리글에서 사용
- [x] 테스트 4개 추가, `tsc`·`vitest`(295개) 통과, 빌드
- [x] 명세 ui/panes.md §2.1 7번, decisions D121
- [ ] 수동 검증 1~8
