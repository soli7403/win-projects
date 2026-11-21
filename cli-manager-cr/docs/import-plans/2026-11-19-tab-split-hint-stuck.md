# 탭 끌기 화면 나누기 안내 2건 (취소해도 남음 · 너무 빨리 표시됨) — 원인·수정·검증

**작성**: 2026-11-19 · **상태**: 코드 수정 완료(2026-11-19, D101·D102), 자동 테스트 통과, 앱 수동 검증은 남음(§7.2) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/ui/shell.md](../../spec/ui/shell.md) §2.3(탭 그룹, 화면 나누기), [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md) G8, [frontend/src/screens/Shell.tsx](../../frontend/src/screens/Shell.tsx), [frontend/src/screens/splitDropHint.ts](../../frontend/src/screens/splitDropHint.ts), [frontend/src/screens/splitDropHint.test.tsx](../../frontend/src/screens/splitDropHint.test.tsx), [frontend/src/screens/TabBar.tsx](../../frontend/src/screens/TabBar.tsx), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 파일 이름은 처음 다룬 문제(취소해도 남음)에서 왔다. 같은 날 이어진 요청(가장자리에서만 표시)은 §9 에 있고, §5.1 의 코드와 §7 의 검증은 **두 수정을 모두 넣은 최종 상태**다.

---

## 0. 한눈에 보기

| 항목 | 내용 |
|---|---|
| 증상 | 탭을 끌면 작업 영역 오른쪽 절반에 점선 상자("여기에 놓으면 좌우로 나눕니다")가 나타나는데, 끌기를 취소해도 상자가 사라지지 않는다 |
| 원인 | 상자를 지우는 코드가 "작업 영역까지 올라온 놓기"와 "상자에서 포인터가 나감" 두 경우에만 있었다. 끌기가 그 밖의 방식으로 끝나면(다른 탭 위에 놓기, Esc, 작업 영역 밖에서 놓기) 지우는 코드가 실행되지 않았다 |
| 수정 | 상자의 표시 상태를 작은 훅(`useSplitDropHint`)으로 옮기고, 상자가 보이는 동안 **창 전체에서 끌기 종료(`dragend`)와 놓기(`drop`)를 직접 듣고** 지운다. 영역을 벗어났는지는 이벤트가 난 요소가 아니라 **포인터 좌표**로 판단한다 |
| 바뀐 파일 | `frontend/src/screens/splitDropHint.ts`(새로 만듦), `frontend/src/screens/splitDropHint.test.tsx`(새로 만듦), `frontend/src/screens/Shell.tsx`(수정) |
| 추가 변경(§9) | 안내가 너무 빨리 나온다는 요청으로, 탭이 **오른쪽 가장자리(너비의 10%, 최소 100px)** 에 들어갈 때만 켠다. 탭 막대 위에서는 켜지 않고, 켜진 뒤 왼쪽 절반으로 가면 지운다 (D102) |
| 검증 | `tsc --noEmit` 통과, 프런트 테스트 39개 파일 249개 통과(새 테스트 16개 포함), `build/bin/dariterm.exe` 빌드 완료. **실제 앱에서 끌어 보는 확인은 아직 하지 않았다**(§7.2) |
| 명세 | D101 은 명세 그대로(끌기가 끝나면 사라지는 것이 "끄는 동안 나타난다"의 뜻). D102 는 사용자 결정이라 [ui/shell.md](../../spec/ui/shell.md) §2.3 과 수용 G8 을 고치고 G8a 를 추가했다 |

---

## 1. 발견 경위

2026-11-19 사용자 보고(스크린샷 첨부): "탭을 선택해서 드래그 하면 점선의 오른쪽 패널모양이 생기는데, 드래그한 탭을 취소해도 패널 모양이 없어지지 않는다."

스크린샷에서는 탭 4개(로컬 터미널, mds-컴파일, loading-tester1(200), DC1_MSA01(213))가 열린 나뉘지 않은 화면의 오른쪽 절반에 점선 상자와 안내 문구가 떠 있고, 끌고 있는 탭은 없다.

---

## 2. 안내 상자의 구조 (수정 전)

안내 상자는 [Shell.tsx](../../frontend/src/screens/Shell.tsx) 의 작업 영역(`<main class="workspace">`) 안에 그린다. 표시 여부는 `Shell` 의 상태 `dragHint`(참·거짓) 하나였다.

```
<main class="workspace">            ← 끌기 이벤트 처리기 3개(위로 끌림·나감·놓임)가 여기 달려 있다
  <div class="tabgroups">
    탭 막대(TabBar)                 ← 탭 하나하나가 끌 수 있는 요소
    칸(PaneTree) 또는 시작 화면
  </div>
  {dragHint && <div class="split-drop">여기에 놓으면 좌우로 나눕니다</div>}
</main>
```

상자의 모양은 [app.css:229-233](../../frontend/src/styles/app.css#L229-L233) 이다. 작업 영역의 오른쪽 절반을 `position: absolute; z-index: 20` 으로 덮는다(탭 막대의 오른쪽 절반도 덮는다).

수정 전 코드(Shell.tsx):

```tsx
const [dragHint, setDragHint] = useState(false);
const isTabDrag = (e: React.DragEvent) => !split && canSplit && e.dataTransfer.types.includes(TAB_MIME);
…
<main
  className="workspace"
  onDragOver={(e) => {
    if (isTabDrag(e)) {
      e.preventDefault();
      setDragHint(true);                                   // (켜기) 탭을 끌어 작업 영역 위에 있으면
    }
  }}
  onDragLeave={(e) => {
    if (e.currentTarget === e.target) setDragHint(false);  // (끄기 1) main 자신에서 난 dragleave 일 때만
  }}
  onDrop={() => setDragHint(false)}                        // (끄기 2) 놓기가 main 까지 올라왔을 때
>
  …
  {dragHint && (
    <div
      className="split-drop"
      onDragOver={(e) => e.preventDefault()}
      onDragLeave={() => setDragHint(false)}               // (끄기 3) 상자에서 포인터가 나갔을 때
      onDrop={(e) => {
        const id = e.dataTransfer.getData(TAB_MIME);
        setDragHint(false);
        if (!id) return;
        e.preventDefault();
        useWorkspace.getState().moveTabToGroup(id, 1);     // 오른쪽 그룹으로 보내 화면을 나눈다
      }}
    >
      <span>{tr('shell.dropSplit')}</span>
    </div>
  )}
</main>
```

켜는 조건: 화면이 나뉘어 있지 않고(`!split`), 탭이 2개 이상이고(`canSplit`), 끄는 것이 탭(`TAB_MIME` = `application/x-dariterm-tab`)일 때. **탭을 조금이라도 끌면 포인터가 탭 막대 위에 있으므로 곧바로 켜진다**(탭 막대도 작업 영역 안이라 `dragover` 가 `main` 까지 올라온다).

---

## 3. 원인

끄는 경로가 세 개뿐이고, 셋 다 끌기가 "작업 영역 안에서 정상적으로 놓이는" 경우를 전제로 한다. 브라우저의 끌어 놓기는 아래처럼 끝날 수도 있는데 그때는 세 경로 어느 것도 실행되지 않는다.

| # | 끌기가 끝나는 방식 | 브라우저가 보내는 이벤트 | 수정 전 동작 |
|---|---|---|---|
| A | **다른 탭(또는 자기 자신) 위에 놓기** — 탭 순서 바꾸기, 또는 조금 끌다가 제자리에 놓기 | 탭 요소에 `drop` | 탭의 놓기 처리가 전파를 막는다([TabBar.tsx:116](../../frontend/src/screens/TabBar.tsx#L116) `e.stopPropagation()`). `drop` 이 `main` 까지 올라오지 않아 (끄기 2)가 실행되지 않는다 → **남음** |
| B | **Esc 로 취소** | `drop` 은 없다. 끌기 시작한 탭에 `dragend` 만 온다 | `dragend` 를 듣는 곳이 없다 → **남음** |
| C | **작업 영역 밖으로 끌고 나가 놓기** — 서버 목록, 상태 표시줄, 창 밖 | 나갈 때 마지막으로 있던 자식 요소에 `dragleave`, 그 뒤 탭에 `dragend` | `dragleave` 의 대상(`e.target`)이 `main` 이 아니라 그 안의 자식(탭·터미널 등)이라 (끄기 1)의 조건 `e.currentTarget === e.target` 이 거짓 → **남음** |

(끄기 1)의 조건이 참이 되는 것은 포인터가 `main` 자신(안쪽 여백 4px)에 직접 있다가 나갈 때뿐이라, 사실상 거의 동작하지 않았다.

**한번 남으면** 다음에 탭을 끌어 정상적으로 놓을 때까지 계속 보인다. 상자가 `z-index: 20` 으로 오른쪽 절반을 덮고 있어, 그동안 오른쪽 절반의 터미널·탭은 마우스로 누를 수 없다(상자가 마우스 입력을 받는다).

> 위 A~C 는 코드를 읽고 확인한 것이다. 사용자가 말한 "취소"가 A·B·C 중 어느 것이었는지는 묻지 않았다(셋 다 같은 결과다). A 가 가장 흔하다 — 탭을 눌러 조금만 움직였다 놓아도 해당한다.

### 3.1 함께 있던 문제 (코드상)

(끄기 3) `split-drop` 의 `onDragLeave` 는 조건 없이 끈다. 포인터가 상자 안의 글자(`<span>`) 위로 들어갈 때도 상자 자신에게 `dragleave` 가 오므로, 그 순간 상자가 꺼졌다가 다음 `dragover` 에서 다시 켜진다(깜빡임). 화면에서 실제로 깜빡이는지는 확인하지 않았다. 이번 수정으로 이 처리기는 없어졌다.

---

## 4. 설계 결정 (D101)

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 상자가 보이는 동안 창에서 `dragend`·`drop` 을 캡처 단계로 듣고 지운다** | 끌기가 어떻게 끝나든 둘 중 하나는 반드시 온다. 자식이 전파를 막아도 캡처 단계라 받는다. 이 프로젝트의 파일 끌기([fileDrag.ts:35-45](../../frontend/src/screens/fileDrag.ts#L35-L45))가 이미 같은 방식을 쓴다 | **채택** |
| B. 탭의 `onDragEnd` 에서 지운다 | 탭 막대가 `Shell` 의 상태를 알아야 해서 상태를 전역 저장소로 옮겨야 한다. 놓는 순간 탭이 다른 그룹으로 옮겨져 요소가 새로 만들어지면 `dragend` 를 못 받을 수 있다 | 기각 |
| C. `TabBar` 의 `stopPropagation` 을 없앤다 | A 경우만 고쳐지고 B·C 는 그대로다. 놓기가 탭 막대의 "빈 곳에 놓기" 처리까지 올라가 탭이 두 번 옮겨진다 | 기각 |

세부 결정:

1. **지우기를 한 박자 미룬다(`setTimeout 0`)**. 창의 캡처 처리기는 상자 자신의 `drop` 처리보다 먼저 실행된다. 그 자리에서 바로 지우면 React 가 상자를 화면에서 뺀 뒤라 상자의 `drop` 처리(화면 나누기)가 실행되지 않을 수 있다. `fileDrag.ts` 도 같은 이유로 미룬다.
2. **영역을 벗어났는지는 포인터 좌표로 판단한다**. `dragleave` 는 자식 사이를 오갈 때마다 올라온다. "어디로 갔는지"를 알려 주는 `relatedTarget` 은 웹뷰 엔진에 따라 비어 있다(WebKit 계열). 공통 규칙의 OS 중립(../CLAUDE.md §8)에 맞춰, 이벤트의 포인터 좌표가 작업 영역 사각형 밖일 때만 지운다. 창 밖으로 나갈 때는 좌표가 (0,0) 또는 창 밖 값으로 오는데, 작업 영역은 왼쪽에 아이콘 묶음(또는 서버 목록)이 있어 (0,0)을 포함하지 않는다. 경계값도 밖으로 본다(`<=`, `>=`).
3. **상자 자신의 `onDragLeave` 는 없앤다**(§3.1). 상자에서 난 `dragleave` 도 `main` 으로 올라와 2번의 좌표 판단을 받는다.
4. **상태를 훅 파일로 분리한다**. `Shell` 전체를 그리지 않고 이 동작만 테스트하려는 것이다.

이 단계(D101)에서는 켜는 조건을 바꾸지 않았다(작업 영역 위 어디서든 켜짐). 켜는 조건은 같은 날 §9(D102)에서 바꿨다.

---

## 5. 수정 내용

### 5.1 새 파일 — `frontend/src/screens/splitDropHint.ts` (전체, §9 까지 넣은 최종)

```ts
// 탭을 끄는 동안 작업 영역 오른쪽 절반에 보이는 "여기에 놓으면 좌우로 나눕니다" 안내의 표시 상태.
// 근거: spec/ui/shell.md §2.3(나누기), spec/acceptance/02-shell-sidebar.md G8.
import { useEffect, useState, type DragEvent } from 'react';

/** 안내를 켜는 오른쪽 가장자리 띠의 너비: 영역 너비의 이 비율, 단 아래 최소값(px)보다 좁지 않게 (D102). */
export const SPLIT_EDGE_RATIO = 0.1;
export const SPLIT_EDGE_MIN_PX = 100;

export interface SplitDropHint {
  /** 안내를 보일지 */
  show: boolean;
  hide: () => void;
  /** 작업 영역(안내가 덮는 영역 전체)에 다는 처리기 */
  zone: {
    onDragOver: (e: DragEvent<HTMLElement>) => void;
    onDragLeave: (e: DragEvent<HTMLElement>) => void;
    onDrop: () => void;
  };
}

/**
 * mime 을 가진 끌기가 작업 영역의 오른쪽 가장자리 띠에 들어오면 안내를 보인다.
 * 한번 보이면 포인터가 오른쪽 절반(안내 상자 자리)에 있는 동안 유지하고, 왼쪽 절반으로 가면 지운다.
 * ignore(CSS 선택자)에 맞는 요소 위에서는 켜지 않는다(탭 막대: 탭 순서 바꾸기를 가리지 않도록).
 * enabled 가 false 면(이미 나뉘어 있거나 탭이 1개) 보이지 않는다.
 *
 * 끌기는 놓기 말고도 여러 방식으로 끝난다(Esc 취소, 놓을 수 없는 곳·창 밖에서 놓기,
 * 전파를 막는 자식 위에 놓기). 어느 경우든 지워지도록 창에서 dragend·drop 을 직접 듣는다.
 */
export function useSplitDropHint(enabled: boolean, mime: string, ignore?: string): SplitDropHint {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (!show) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    // 캡처 단계라 자식이 전파를 막아도 받는다. 곧바로 지우면 안내 상자가 자기 drop 을
    // 처리하기 전에 화면에서 빠질 수 있어 이벤트 처리가 끝난 뒤로 미룬다.
    const clear = () => {
      clearTimeout(timer);
      timer = setTimeout(() => setShow(false), 0);
    };
    window.addEventListener('dragend', clear, true);
    window.addEventListener('drop', clear, true);
    return () => {
      clearTimeout(timer);
      window.removeEventListener('dragend', clear, true);
      window.removeEventListener('drop', clear, true);
    };
  }, [show]);

  return {
    show: show && enabled,
    hide: () => setShow(false),
    zone: {
      onDragOver: (e) => {
        if (!enabled || !Array.from(e.dataTransfer.types).includes(mime)) return;
        e.preventDefault();
        const r = e.currentTarget.getBoundingClientRect();
        if (show) {
          // 안내 상자는 오른쪽 절반을 덮는다. 그 밖(왼쪽 절반)으로 나가면 지운다.
          if (e.clientX < r.left + r.width / 2) setShow(false);
          return;
        }
        const edge = Math.max(SPLIT_EDGE_MIN_PX, r.width * SPLIT_EDGE_RATIO);
        if (e.clientX < r.right - edge) return;
        if (ignore && (e.target as Element).closest?.(ignore)) return;
        setShow(true);
      },
      // 자식 사이를 오갈 때도 dragleave 가 올라오므로, 포인터가 영역 밖으로 나갔을 때만 지운다.
      // relatedTarget 은 웹뷰 엔진에 따라 비어 있어 좌표로 판단한다.
      onDragLeave: (e) => {
        const r = e.currentTarget.getBoundingClientRect();
        if (e.clientX <= r.left || e.clientX >= r.right || e.clientY <= r.top || e.clientY >= r.bottom) setShow(false);
      },
      onDrop: () => setShow(false),
    },
  };
}
```

읽는 법:

- 창 처리기는 **상자가 보이는 동안에만** 달려 있다(`show` 가 참일 때 등록, 거짓이 되면 해제). 평소에는 아무것도 듣지 않는다.
- `show: show && enabled` — 끄는 도중 화면이 나뉘는 등 조건이 바뀌면 상태가 남아 있어도 그리지 않는다.
- `zone.onDrop` 은 수정 전의 (끄기 2)와 같다. 창 처리기가 있어 없어도 되지만, 미루지 않고 바로 지우는 경로로 남겼다.

### 5.2 `frontend/src/screens/Shell.tsx`

가져오기 한 줄 추가:

```tsx
import { useSplitDropHint } from './splitDropHint';
```

상태 선언(`Shell` 함수 안):

```tsx
// before
const [dragHint, setDragHint] = useState(false);
const groupsRef = useRef<HTMLDivElement>(null);
const isTabDrag = (e: React.DragEvent) => !split && canSplit && e.dataTransfer.types.includes(TAB_MIME);

// after
// 탭을 끄는 동안의 "여기에 놓으면 좌우로 나눕니다" 안내. 끌기가 어떻게 끝나든 지워진다(splitDropHint.ts).
const dragHint = useSplitDropHint(!split && canSplit, TAB_MIME);   // §9 에서 세 번째 인자 '.tabbar' 추가
const groupsRef = useRef<HTMLDivElement>(null);
```

작업 영역과 상자:

```tsx
// before
<main
  className="workspace"
  onDragOver={(e) => {
    if (isTabDrag(e)) {
      e.preventDefault();
      setDragHint(true);
    }
  }}
  onDragLeave={(e) => {
    if (e.currentTarget === e.target) setDragHint(false);
  }}
  onDrop={() => setDragHint(false)}
>
  …
  {dragHint && (
    <div
      className="split-drop"
      onDragOver={(e) => e.preventDefault()}
      onDragLeave={() => setDragHint(false)}
      onDrop={(e) => {
        const id = e.dataTransfer.getData(TAB_MIME);
        setDragHint(false);
        …

// after
<main className="workspace" {...dragHint.zone}>
  …
  {dragHint.show && (
    <div
      className="split-drop"
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => {
        const id = e.dataTransfer.getData(TAB_MIME);
        dragHint.hide();
        …
```

상자의 놓기 처리(탭을 오른쪽 그룹으로 보내기)와 문구·모양(CSS)은 바꾸지 않았다. `useState` 는 이 파일의 다른 곳(`useNow`)에서 계속 쓴다.

### 5.3 수정 뒤 동작 (§3 의 표와 대응)

| # | 끌기가 끝나는 방식 | 지우는 경로 |
|---|---|---|
| A | 다른 탭·자기 자신 위에 놓기 | 창의 `drop`(캡처) → 다음 박자에 지움. 탭 순서 바꾸기는 그대로 동작 |
| B | Esc 취소 | 창의 `dragend`(캡처) → 지움 |
| C | 작업 영역 밖으로 나가 놓기 | 나가는 순간 `dragleave` 의 좌표가 영역 밖 → 바로 지움. 놓을 때 `dragend` 로 한 번 더 |
| — | 상자에 놓기(화면 나누기) | 상자의 `onDrop` 이 바로 지우고 탭을 오른쪽으로 보냄 |
| — | 끄는 중 오른쪽 절반 안에서 움직임 | 지우지 않음 |
| — | 끄는 중 왼쪽 절반으로 감 | `dragover` 에서 지움(§9) |
| — | 영역 밖으로 나갔다가 다시 들어옴 | 나갈 때 지워지고, 오른쪽 가장자리로 들어오면 다시 보임 |

---

## 6. 영향 범위와 회귀 위험

| 영역 | 영향 |
|---|---|
| 탭 순서 바꾸기·다른 그룹으로 옮기기([TabBar.tsx](../../frontend/src/screens/TabBar.tsx)) | 코드를 건드리지 않았다. 창 처리기는 캡처 단계에서 듣기만 하고 `preventDefault`·`stopPropagation` 을 부르지 않는다 |
| 칸 끌기·서버 끌기([Panes.tsx](../../frontend/src/screens/Panes.tsx)), 파일 끌기([fileDrag.ts](../../frontend/src/screens/fileDrag.ts)) | `TAB_MIME` 이 아니면 안내가 켜지지 않고, 켜지지 않으면 창 처리기도 달리지 않는다 |
| 화면 나누기(상자에 놓기) | 지우기를 미루므로 상자의 놓기 처리가 먼저 실행된다(테스트 "안내 상자에 놓으면 나누기가 실행되고 지워진다") |
| 위험 | `dragleave` 의 좌표가 실제 WebView2 에서 기대대로 오는지는 자동 테스트로 확인할 수 없다(테스트 환경 jsdom 은 끌기 이벤트에 좌표를 넣어 주지 않아 테스트가 좌표를 직접 붙인다). 좌표가 틀리게 오더라도 `dragend`·`drop` 경로가 있어 상자가 **남는** 일은 없고, 최악은 끄는 중 깜빡임이다 → §7.2 의 5·6번으로 확인 |

백엔드(Go), 데이터 파일, 문구(`locales/`), CSS 는 바뀌지 않았다.

---

## 7. 검증

### 7.1 자동 (2026-11-19 실행)

| 명령 (`frontend/` 에서, PowerShell) | 결과 |
|---|---|
| `npx tsc --noEmit` | 통과 (exit 0) |
| `npx vitest run` | 39개 파일, 249개 테스트 통과 |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` (`cli-manager-cr/` 에서) | `build/bin/dariterm.exe` 생성 |

> 이 PC 에서 vitest 를 Git Bash 로 돌리면 테스트 내용과 무관하게 "Vitest failed to find the current suite" 로 모든 파일이 실패한다(기존 테스트도 같다). PowerShell 에서 돌리면 정상이다.

새 테스트 [splitDropHint.test.tsx](../../frontend/src/screens/splitDropHint.test.tsx) 16개 — `Shell` 의 작업 영역과 같은 구조(영역 안에 탭 막대와 전파를 막는 탭·내용·안내 상자)를 만든 작은 화면으로 확인한다. 영역은 (100,50)~(500,350), 너비 400 이라 가장자리 띠는 100px(x ≥ 400), 가운데는 x = 300 이다.

| 테스트 | 확인하는 것 | 대응 |
|---|---|---|
| 오른쪽 가장자리 띠에 들어가면 보인다 | x = 450 | §9, G8 |
| 띠의 경계(오른쪽 끝 − 100px)에서부터 보인다 | x = 399 는 안 보이고 400 은 보임 | §9 |
| 왼쪽 절반이나 오른쪽 절반의 띠 밖에서는 보이지 않는다 | x = 200, 350 | §9 |
| 영역이 넓으면 띠는 너비의 10% 다 | 너비 2000 → x = 1799 안 보임, 1800 보임 | §9 |
| 탭 막대 위에서는 가장자리라도 보이지 않는다 | 탭 순서 바꾸기를 가리지 않음 | §9 |
| 보인 뒤에는 오른쪽 절반 안에서 띠 밖으로 나와도 유지된다 | 깜빡임 없음 | §9 |
| 보인 뒤 왼쪽 절반으로 가면 지워지고, 다시 띠에 들어가면 보인다 | 오른쪽 절반의 띠 밖으로 돌아온 것만으로는 다시 안 켜짐 | §9 |
| 탭이 아닌 것을 끌면 보이지 않는다 | `Files` 끌기 | — |
| 나눌 수 없는 상태(enabled=false)면 보이지 않는다 | 이미 나뉨·탭 1개 | — |
| 끌기를 취소하면(dragend 만 온다) 지워진다 | Esc·놓을 수 없는 곳 | §3 B |
| 전파를 막는 자식(탭) 위에 놓아도 지워진다 | 탭 위에 놓기 | §3 A |
| 영역 밖으로 나가면 지워진다 (자식에서 난 dragleave 라도) | 좌표 (520,200) | §3 C |
| 창 밖으로 나가면(좌표 0,0) 지워진다 | 창 밖 | §3 C |
| 영역 안에서 자식 사이를 오가는 dragleave 로는 지워지지 않는다 | 깜빡임 없음 | §3.1 |
| 안내 상자에 놓으면 나누기가 실행되고 지워진다 | 놓기 처리가 먼저 실행됨 | §4 결정 1 |
| 지워진 뒤 다시 끌면 다시 보인다 | 창 처리기 재등록 | — |

수정 전 코드에 같은 테스트를 돌려 실패하는지는 확인하지 않았다(훅이 새 파일이라 수정 전 코드에는 대상이 없다).

### 7.2 수동 (남음) — `build/bin/dariterm.exe` (2026-11-19 11:30 빌드)

준비: 탭 2개 이상, 화면은 나뉘지 않은 상태.

| # | 조작 | 기대 |
|---|---|---|
| 1 | 탭을 눌러 조금 끌다가 제자리에 놓는다 / 다른 탭 위에 놓아 순서를 바꾼다 | 상자가 **나타나지 않는다**. 순서 바꾸기는 정상 |
| 2 | 탭을 맨 오른쪽 탭 자리(탭 막대의 오른쪽 끝)로 끈다 | 상자가 나타나지 않는다 |
| 3 | 탭을 터미널 영역으로 끌어 내려 오른쪽으로 천천히 움직인다 | 오른쪽 끝에서 약 100px(넓은 창은 너비의 10%) 안에 들어가는 순간 상자가 나타난다. 그 전에는 없다 |
| 4 | 3 에서 상자가 보이는 채로 상자 안(오른쪽 절반)을 움직인다 | 계속 보이고 깜빡이지 않는다 |
| 5 | 4 에서 왼쪽 절반으로 끈다 | 상자가 사라진다. 다시 오른쪽 가장자리로 가면 나타난다 |
| 6 | 상자가 보이는 채로 **Esc** | 상자가 사라진다 |
| 7 | 상자가 보이는 채로 창 밖(오른쪽)으로 끌고 나가 놓는다 | 나가는 순간 사라지고 놓은 뒤에도 없다 |
| 8 | 상자 안에 놓는다 | 화면이 좌우로 나뉘고 그 탭이 오른쪽 그룹으로 간다(G8) |
| 9 | 6·7 뒤에 오른쪽 절반의 터미널을 마우스로 누른다 | 정상적으로 눌린다 |
| 10 | 칸 머리글 끌기, 서버 목록에서 서버 끌기, 파일 창에서 파일 끌기 | 수정 전과 같다(상자가 나타나지 않는다) |

---

## 8. 남은 일

- §7.2 수동 검증.
- 가장자리 띠의 너비(10%, 최소 100px)가 손에 맞는지는 써 보고 정한다. 바꿀 곳은 `splitDropHint.ts` 의 상수 두 개(`SPLIT_EDGE_RATIO`, `SPLIT_EDGE_MIN_PX`)다.

---

## 9. 추가 변경 — 안내를 오른쪽 가장자리에서만 표시 (D102)

### 9.1 요청

§5 의 수정을 본 뒤 사용자 요청(2026-11-19): "오른쪽 분할이 너무 빨리 표시된다. 오른쪽에 100 픽셀 정도(고정된 길이) 또는 전체 화면에서 10% 정도 영역에 들어가면 표시하는 것은 어떤가?"

### 9.2 왜 빨리 나왔나

켜는 조건이 "탭을 끄는 포인터가 작업 영역 위에 있다" 뿐이었다(§2). 탭 막대도 작업 영역 안이라, 탭을 누르고 몇 픽셀만 움직여도 켜졌다. 켜진 상자는 탭 막대의 오른쪽 절반까지 덮어서, 탭 순서만 바꾸려 해도 오른쪽 절반의 탭 자리에는 놓을 수 없었다(상자에 놓여 화면이 나뉜다).

### 9.3 결정

| 항목 | 결정 | 이유 |
|---|---|---|
| 켜는 곳 | 작업 영역 오른쪽 끝에서 `max(100px, 너비의 10%)` 안 | 사용자가 든 두 기준을 함께 쓴다. 좁은 창(너비 800 → 10% 는 80px)에서는 100px 로 받쳐 띠가 너무 얇아지지 않고, 넓은 화면(너비 2400 → 240px)에서는 끝까지 가지 않아도 된다 |
| 탭 막대 | 탭 막대(`.tabbar`) 위에서는 켜지 않는다 | 맨 오른쪽 탭 자리로 순서를 바꿀 때 포인터가 띠 안에 들어가도 상자가 가리지 않게 |
| 켜진 뒤 | 포인터가 오른쪽 절반(상자 자리)에 있는 동안 유지, 왼쪽 절반으로 가면 지움 | 상자는 오른쪽 절반 크기이고 그 전체가 놓을 수 있는 곳이다. 띠를 벗어나자마자 지우면 보이는 상자와 놓을 수 있는 곳이 달라지고 띠 경계에서 깜빡인다 |
| 다시 켜기 | 왼쪽으로 가서 지워진 뒤에는 다시 띠에 들어가야 켜진다 | 켜는 조건을 한 가지로 유지 |
| 상자 크기·문구 | 그대로(오른쪽 절반, "여기에 놓으면 좌우로 나눕니다") | 놓았을 때의 결과(좌우 절반)를 미리 보여 주는 모양이다 |

켜진 상자는 여전히 탭 막대의 오른쪽 절반을 덮는다. 다만 이제는 사용자가 터미널 영역의 오른쪽 가장자리까지 끌고 가야 켜지므로, 탭 순서를 바꾸는 동작과 겹치지 않는다.

### 9.4 코드 — `frontend/src/screens/splitDropHint.ts`

상수와 인자 추가:

```ts
/** 안내를 켜는 오른쪽 가장자리 띠의 너비: 영역 너비의 이 비율, 단 아래 최소값(px)보다 좁지 않게 (D102). */
export const SPLIT_EDGE_RATIO = 0.1;
export const SPLIT_EDGE_MIN_PX = 100;

// before
export function useSplitDropHint(enabled: boolean, mime: string): SplitDropHint {
// after
export function useSplitDropHint(enabled: boolean, mime: string, ignore?: string): SplitDropHint {
```

켜는 처리:

```ts
// before
onDragOver: (e) => {
  if (!enabled || !Array.from(e.dataTransfer.types).includes(mime)) return;
  e.preventDefault();
  setShow(true);
},

// after
onDragOver: (e) => {
  if (!enabled || !Array.from(e.dataTransfer.types).includes(mime)) return;
  e.preventDefault();
  const r = e.currentTarget.getBoundingClientRect();
  if (show) {
    // 안내 상자는 오른쪽 절반을 덮는다. 그 밖(왼쪽 절반)으로 나가면 지운다.
    if (e.clientX < r.left + r.width / 2) setShow(false);
    return;
  }
  const edge = Math.max(SPLIT_EDGE_MIN_PX, r.width * SPLIT_EDGE_RATIO);
  if (e.clientX < r.right - edge) return;
  if (ignore && (e.target as Element).closest?.(ignore)) return;
  setShow(true);
},
```

`Shell.tsx`:

```tsx
// before
const dragHint = useSplitDropHint(!split && canSplit, TAB_MIME);
// after
const dragHint = useSplitDropHint(!split && canSplit, TAB_MIME, '.tabbar');
```

읽는 법:

- `r` 은 작업 영역(`main`)의 사각형이다. 상자의 왼쪽 끝은 CSS(`right: 4px; width: calc(50% - 4px)`)로 정확히 영역의 가운데(`r.left + r.width / 2`)라, "왼쪽 절반으로 나감"의 기준과 일치한다.
- 상자 위의 `dragover` 도 `main` 으로 올라오므로, 상자가 켜진 동안의 판단도 이 처리기 하나가 한다.
- `e.preventDefault()` 는 띠 밖에서도 그대로 부른다(수정 전과 같다). 띠 밖에 놓으면 아무 일도 일어나지 않고 `dragend`·`drop` 으로 끝난다.
- `.tabbar` 는 [TabBar.tsx](../../frontend/src/screens/TabBar.tsx) 의 탭 막대 클래스다(탭, + 버튼, SFTP 버튼, 빈 곳 포함).

### 9.5 명세

사용자 결정이라 명세에 반영했다.

- [spec/ui/shell.md](../../spec/ui/shell.md) §2.3 "나누기": 가장자리(10%, 최소 100px)에서 안내가 나타남, 탭 막대 위에서는 띄우지 않음, 오른쪽 절반에 있는 동안 유지, 왼쪽 절반으로 가거나 끌기가 끝나면 사라짐.
- [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md) G8 개정, G8a(취소·밖에 놓기·왼쪽으로 끌기 → 사라짐) 추가.
