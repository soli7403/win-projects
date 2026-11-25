# 명령 모음을 켜면 왼쪽 화면이 어긋나고 너비가 따로 노는 문제 — 원인·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D130). 실제 브라우저 엔진(Edge 헤드리스)으로 고치기 전·후의 너비를 재어 확인했다. 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(발견 즉시 수정)
**관련**: 명령 모음을 서버 목록 아래에 붙인 결정 D55. 명세 [spec/ui/sidebar.md](../../spec/ui/sidebar.md) §1, [spec/ui/sftp-snippets.md](../../spec/ui/sftp-snippets.md) §1, [spec/ui/shell.md](../../spec/ui/shell.md) §1
**적용 순서**: 2026-11-25 의 7번째. 앞의 문서들과 고치는 파일이 겹치지 않는다(`CHANGELOG.md`·`docs/decisions.md`·`CLAUDE.md` 제외).

> 이 문서 하나로 **무엇이 문제였는지 → 왜 그랬는지 → 어떻게 고쳤는지 → 코드 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

왼쪽 위의 명령 모음 버튼(`<>`)을 켜면 왼쪽 열이 서버 목록보다 넓어져 화면이 어긋났다. 이제 **서버 목록과 명령 모음이 늘 같은 너비**이고, 너비를 끌어 바꾸면 **함께** 바뀐다.

| | 고치기 전 | 고친 뒤 |
|---|---|---|
| 명령 모음을 켰을 때 | 왼쪽 열이 약 358px 로 벌어짐. 서버 목록은 제 너비(예: 290px)로 남아 오른쪽에 빈 띠와 세로선 | 열·서버 목록·명령 모음이 모두 설정한 너비 |
| 서버 목록 너비를 끌어 바꿀 때 | 서버 목록만 바뀌고 명령 모음은 그대로 | 함께 바뀐다 |
| 너비 조절 손잡이가 잡히는 곳 | 서버 목록 높이만큼 | 왼쪽 열 전체 높이(명령 모음 옆 포함) |

실제 브라우저 엔진으로 잰 값(너비 설정 290px, 명령 0개):

```
고치기 전:  왼쪽 열 358   서버 목록 290   명령 모음 358   작업 영역 시작 358
고친 뒤:    왼쪽 열 290   서버 목록 289   명령 모음 289   작업 영역 시작 290     (289 = 열의 오른쪽 테두리 1px 을 뺀 안쪽)
```

## 1. 사용자 보고

2026-11-25(화면 사진과 함께): "명령 모음을 켜면 화면이 깨지고 명령 모음은 서버 트리와 함께 움직이지 않는다. 수정해줘."

사진에서 보이는 것:
- 서버 목록(검색 칸·버튼·트리)은 약 290px 인데 그 오른쪽에 세로선이 서 있고, 세로선과 터미널 사이에 약 60px 의 빈 띠가 있다.
- 아래의 명령 모음은 그 빈 띠까지 차지해 터미널 바로 옆(약 350px)까지 넓다.
- 명령이 하나도 없어 안내 문구 "자주 쓰는 명령을 저장해 두고 클릭 한 번으로 실행하세요."가 한 줄로 보인다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 왼쪽 열 | 화면 왼쪽의 세로 영역. 위에서부터 아이콘 묶음 · 서버 목록 · (켜져 있으면) 명령 모음 |
| 명령 모음 | 자주 쓰는 명령을 저장해 두는 패널. 코드에서는 "스니펫"(`snippet`)이라 부른다 |
| 너비 조절 손잡이 | 왼쪽 열의 오른쪽 가장자리. 끌어서 너비를 바꾼다(200~520px, 설정 `sidebarWidth`) |
| 펼친 길이 | 글을 줄바꿈 없이 한 줄로 놓았을 때의 길이. 브라우저가 "내용에 맞춘 너비"를 정할 때 쓴다 |

## 2. 원인

너비를 **서버 목록 혼자** 갖고 있었고, 왼쪽 열 자체는 너비가 정해져 있지 않았다.

```
.leftcol              ← 너비를 정하지 않음(flex: none) → 안의 것 중 가장 넓은 것에 맞춰진다
 ├ .toolrow           아이콘 묶음
 └ .leftcol-body
     ├ .sidebar       ← style="width: 290px"  (여기에만 너비가 있었다)
     └ .snippet-dock  ← 너비 없음 → 내용의 펼친 길이만큼 넓어지려 한다
```

| 단계 | 일어난 일 |
|---|---|
| ① | 명령 모음이 켜지면 `.snippet-dock` 이 열에 들어온다. 너비가 없으므로 내용에 맞추려 한다 |
| ② | 명령이 0개일 때의 안내 문구는 펼친 길이가 약 318px 이고 좌우 여백 20px 씩을 더해 **358px** 이다 |
| ③ | 왼쪽 열은 안의 것 중 가장 넓은 것(358px)에 맞춰 넓어진다 → 터미널이 오른쪽으로 밀린다 |
| ④ | 서버 목록은 제 너비(290px)를 고집해 왼쪽에 붙어 있고, 오른쪽 테두리선이 열 가운데에 선다 → 빈 띠 |
| ⑤ | 손잡이로 너비를 바꾸면 서버 목록의 290 만 바뀐다. 358 보다 좁은 동안 명령 모음은 꼼짝하지 않는다 |

서버 목록을 358px 보다 넓게 해 두면 문제가 보이지 않는다(열이 서버 목록 너비에 맞춰지고 명령 모음이 그만큼 늘어난다). 명령이 하나라도 있으면 안내 문구가 없어 열이 벌어지지 않는다. 그래서 **명령을 등록하지 않은 채, 서버 목록이 358px 보다 좁을 때**만 드러났다(기본 너비 260px 이 여기에 든다).

고치기 전 코드:

- [frontend/src/screens/Sidebar.tsx](../../frontend/src/screens/Sidebar.tsx): `<aside className="sidebar" style={{ width }}>` — 너비가 서버 목록의 인라인 스타일. 손잡이(`.side-resize`)도 서버 목록 안.
- [frontend/src/styles/app.css](../../frontend/src/styles/app.css): `.leftcol { … flex: none; … }` — 너비 없음. `.snippet-dock` 에 너비 제한 없음.

## 3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 너비를 가진 곳 | **왼쪽 열**(`.leftcol` 의 인라인 `width` = 설정 `sidebarWidth`) | 열 안의 모든 것이 한 너비를 따른다. 너비를 두 곳(서버 목록·명령 모음)에 따로 주면 다시 어긋날 수 있다 |
| 서버 목록·명령 모음 | 너비를 따로 주지 않고 열을 채운다 | 세로로 쌓인 것들은 기본으로 열 너비에 맞춰 늘어난다 |
| 내용이 열을 넓히지 못하게 | 열·열 본문·서버 목록·명령 모음에 `min-width: 0`, 명령 모음에 `overflow: hidden` | 긴 글이 있어도 열 너비는 그대로. 안내 문구는 줄을 바꿔 보인다 |
| 오른쪽 테두리선 | 열이 한 번 그린다. 열 안의 서버 목록은 자기 테두리를 그리지 않는다 | 선이 열 전체 높이로 이어진다(전에는 서버 목록 높이만큼만 있었다) |
| 손잡이 | 서버 목록에서 **왼쪽 열로 옮겼다**(`SidebarResizer`) | 명령 모음 옆에서도 잡힌다. "함께 움직인다"는 요구에 맞는다 |
| 손잡이를 눌렀다 떼기만 함 | 저장하지 않는다 | 값이 바뀌지 않았는데 저장할 까닭이 없다(전에는 저장했다) |
| 설정 값 | 그대로 `sidebarWidth`(200~520, 기본 260) | 뜻만 "서버 목록 너비"에서 "왼쪽 열 너비"로 넓어진다. 저장된 값을 그대로 쓴다 |
| 서버 목록을 접었을 때 | 그대로(세로 막대 38px) | 건드리지 않았다 |

고르지 않은 안: 명령 모음에도 같은 너비를 인라인으로 주기. 가장 적게 고치지만 너비가 두 곳에 있게 되고, 손잡이가 서버 목록 높이에서만 잡히는 것이 남는다.

고친 뒤의 구조:

```
.leftcol  style="width: 290px"   ← 너비는 여기 하나. 오른쪽 테두리. 손잡이의 기준
 ├ .toolrow
 ├ .leftcol-body
 │   ├ .sidebar        (너비 없음 — 열을 채운다)
 │   └ .snippet-dock   (너비 없음 — 열을 채운다, 넘치면 자른다)
 └ .side-resize        ← 손잡이: 열 전체 높이
```

### 3.1 영향 범위

- **저장 형식·IPC**: 바뀌지 않았다.
- **겉모습**: 아이콘 묶음 줄 오른쪽에도 테두리선이 이어진다(전에는 서버 목록부터 시작). 서버 목록 안쪽 너비가 1px 줄었다(테두리가 열로 옮겨 갔지만 열 너비에 포함되므로 전과 같은 자리).
- **터미널**: 명령 모음을 켜고 끌 때 작업 영역 너비가 더는 바뀌지 않는다(전에는 열이 벌어지며 터미널이 좁아졌다).

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/screens/Shell.tsx` | 코드 | 수정(열에 너비, 손잡이) |
| `frontend/src/screens/Sidebar.tsx` | 코드 | 수정(인라인 너비·손잡이를 뺌, `useSidebarWidth`·`SidebarResizer` 추가) |
| `frontend/src/styles/app.css` | 스타일 | 수정 |
| `frontend/src/screens/Shell.leftcol.test.tsx` | 테스트 | **새 파일**(6개) |
| `frontend/src/styles/leftcol.layout.test.ts` | 테스트 | **새 파일**(3개) |
| `spec/ui/sidebar.md`, `spec/ui/shell.md`, `spec/ui/sftp-snippets.md` | 명세 | 수정 |
| `spec/acceptance/02-shell-sidebar.md` | 명세 | 수정(B10a·B10b) |
| `CHANGELOG.md`, `docs/decisions.md`(D130), `CLAUDE.md` | 문서 | 수정 |

백엔드(Go)와 문구는 바꾸지 않았다.

## B. 코드

### `frontend/src/screens/Shell.tsx` — 수정

```diff
--- a/frontend/src/screens/Shell.tsx
+++ b/frontend/src/screens/Shell.tsx
@@ -15,7 +15,7 @@
 import { PanelHost, SftpDock } from './RightPanel';
 import { CopyJobs } from './CopyJobs';
 import { SnippetsPanel } from './SnippetsPanel';
-import { Sidebar } from './Sidebar';
+import { Sidebar, SidebarResizer, useSidebarWidth } from './Sidebar';
 import { UpdateBadge } from '../ui/UpdateBadge';
 // 이 파일은 t 를 지역 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
 import { relabel, t as tr, type MsgId } from '../i18n';
@@ -255,6 +255,8 @@
 export function Shell() {
   const sidebarOpen = useUi((s) => s.sidebarOpen);
   const snippetRatio = useUi((s) => s.snippetRatio);
+  // 왼쪽 열의 너비: 열 전체(아이콘 묶음·서버 목록·명령 모음)가 같은 너비를 쓴다 (D130).
+  const sidebarWidth = useSidebarWidth();
   const leftRef = useRef<HTMLDivElement>(null);
   const split = useWorkspace((s) => isSplit(s.tabs));
   const ratio = useWorkspace((s) => s.groupRatio);
@@ -268,7 +270,7 @@
     <div className="app">
       <div className="body">
         {sidebarOpen ? (
-          <div className="leftcol">
+          <div className="leftcol" style={{ width: sidebarWidth }}>
             <ToolRow vertical={false} />
             {/* 서버 목록 + 스니펫이 나눠 쓰는 영역. 비율은 SnippetDivider 로 바꾼다. */}
             <div
@@ -286,6 +288,8 @@
                 </>
               )}
             </div>
+            {/* 너비 조절 손잡이: 열 전체 높이(서버 목록 + 명령 모음)에서 잡힌다. */}
+            <SidebarResizer />
           </div>
         ) : (
           <div className="rail">
```

### `frontend/src/screens/Sidebar.tsx` — 수정

```diff
--- a/frontend/src/screens/Sidebar.tsx
+++ b/frontend/src/screens/Sidebar.tsx
@@ -160,7 +160,6 @@
   const groups = store?.groups ?? [];
   const hosts = store?.hosts ?? [];
   const searching = query.trim().length > 0;
-  const width = store?.settings.sidebarWidth ?? 260;
 
   const matched = useMemo(() => hosts.filter((h) => matchesQuery(h, groups, query)), [hosts, groups, query]);
   // 지워진 서버는 선택에서 뺀다. 2개 이상일 때만 여러 개 선택 상태.
@@ -318,24 +317,6 @@
     if (e.key === 'Enter') void openHost(h.id, e.ctrlKey ? { kind: 'rightGroup' } : e.shiftKey ? { kind: 'split', dir: 'col' } : { kind: 'tab' });
   };
 
-  const onResizeDown = (e: RPointerEvent<HTMLDivElement>) => {
-    const startX = e.clientX;
-    const startW = width;
-    let latest = startW;
-    const move = (ev: PointerEvent) => {
-      latest = clampSidebarWidth(startW + ev.clientX - startX);
-      useApp.setState((s) => (s.store ? { store: { ...s.store, settings: { ...s.store.settings, sidebarWidth: latest } } } : s));
-    };
-    const up = () => {
-      window.removeEventListener('pointermove', move);
-      window.removeEventListener('pointerup', up);
-      const st = useApp.getState().store;
-      if (st) void patchStore({ settings: { ...st.settings, sidebarWidth: latest } });
-    };
-    window.addEventListener('pointermove', move);
-    window.addEventListener('pointerup', up);
-  };
-
   const hostRow = (h: Host, depth: number, opts: { selectable?: boolean } = { selectable: true }) => {
     const Icon = PROTOCOL_ICON[h.protocol];
     const color = effectiveColor(h, groups);
@@ -493,7 +474,7 @@
   const noMatch = searching && matched.length === 0 && hosts.length > 0;
 
   return (
-    <aside className="sidebar" style={{ width }} aria-label={tr('shell.servers')} onKeyDown={onKey} tabIndex={-1}>
+    <aside className="sidebar" aria-label={tr('shell.servers')} onKeyDown={onKey} tabIndex={-1}>
       <div className="side-head">
         <div className="search-box">
           <Search size={14} />
@@ -562,8 +543,39 @@
           <div className="faint small" style={{ padding: '8px 10px' }}>{tr('sb.noMatch', { query: query.trim() })}</div>
         )}
       </div>
-
-      <div className="side-resize" onPointerDown={onResizeDown} role="separator" aria-orientation="vertical" aria-label={tr('sb.width')} />
     </aside>
   );
 }
+
+/** 기본 왼쪽 열 너비(px). 저장된 값이 없을 때 쓴다 (data/store.md §4 sidebarWidth). */
+export const SIDEBAR_WIDTH_DEFAULT = 260;
+
+/** 왼쪽 열(아이콘 묶음 + 서버 목록 + 명령 모음)의 너비. 서버 목록만이 아니라 열 전체가 이 너비다 (ui/sidebar.md §1, D130). */
+export function useSidebarWidth(): number {
+  return useApp((s) => s.store?.settings.sidebarWidth ?? SIDEBAR_WIDTH_DEFAULT);
+}
+
+/**
+ * 왼쪽 열의 오른쪽 가장자리: 좌우로 끌어 열 너비를 바꾼다(200~520px). 끄는 동안은 화면에만 반영하고 놓을 때 저장한다.
+ * 왼쪽 열(.leftcol) 안에 두어 서버 목록과 명령 모음 높이 전체에서 잡힌다.
+ */
+export function SidebarResizer() {
+  const onResizeDown = (e: RPointerEvent<HTMLDivElement>) => {
+    const startX = e.clientX;
+    const startW = useApp.getState().store?.settings.sidebarWidth ?? SIDEBAR_WIDTH_DEFAULT;
+    let latest = startW;
+    const move = (ev: PointerEvent) => {
+      latest = clampSidebarWidth(startW + ev.clientX - startX);
+      useApp.setState((s) => (s.store ? { store: { ...s.store, settings: { ...s.store.settings, sidebarWidth: latest } } } : s));
+    };
+    const up = () => {
+      window.removeEventListener('pointermove', move);
+      window.removeEventListener('pointerup', up);
+      const st = useApp.getState().store;
+      if (st && latest !== startW) void useApp.getState().patchStore({ settings: { ...st.settings, sidebarWidth: latest } });
+    };
+    window.addEventListener('pointermove', move);
+    window.addEventListener('pointerup', up);
+  };
+  return <div className="side-resize" onPointerDown={onResizeDown} role="separator" aria-orientation="vertical" aria-label={tr('sb.width')} />;
+}
```

### `frontend/src/styles/app.css` — 수정

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -186,8 +186,11 @@
 .app { height: 100vh; display: flex; flex-direction: column; }
 .toolrow { display: flex; align-items: center; gap: 4px; padding: 6px 8px; flex: none; }
 .toolrow.vertical { flex-direction: column; padding: 6px 0; }
-.leftcol { display: flex; flex-direction: column; flex: none; min-height: 0; background: var(--bg); }
-.leftcol-body { flex: 1 1 0; min-height: 0; display: flex; flex-direction: column; }
+/* 왼쪽 열: 너비는 열이 갖고(인라인 width = 설정 sidebarWidth) 안의 서버 목록·명령 모음은 그 너비를 채운다.
+   안의 내용(긴 문구 등)이 열을 넓히지 못하게 min-width 0 · overflow hidden 을 둔다 (D130). */
+.leftcol { position: relative; display: flex; flex-direction: column; flex: none; min-width: 0; min-height: 0; background: var(--bg); border-right: 1px solid var(--border); }
+.leftcol-body { flex: 1 1 0; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
+.leftcol-body > .sidebar { border-right: none; }
 /* 서버 목록 : 스니펫 = (100 − 비율) : 비율. 스니펫이 닫히면 서버 목록이 전부 차지. */
 .leftcol-body > .sidebar { flex: var(--side-flex, 1) 1 0; min-height: 80px; }
 .snippet-divider { flex: none; height: 6px; margin: -3px 0; cursor: row-resize; position: relative; z-index: 6; }
@@ -224,7 +227,7 @@
 .tabgroup-main { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
 .tab-sftp { margin-left: auto; margin-top: 5px; flex: none; }
 .tab-sftp.on { background: var(--accent-soft); }
-.snippet-dock { flex: 1 1 0; min-height: 80px; display: flex; flex-direction: column; border-top: 1px solid var(--border); background: var(--bg); }
+.snippet-dock { flex: 1 1 0; min-width: 0; min-height: 80px; overflow: hidden; display: flex; flex-direction: column; border-top: 1px solid var(--border); background: var(--bg); }
 .tabbar.in-group { height: 38px; flex: none; border-bottom: 1px solid var(--border); background: var(--panel); padding: 0 4px; }
 .split-drop {
   position: absolute; top: 4px; right: 4px; bottom: 4px; width: calc(50% - 4px); z-index: 20;
@@ -255,7 +258,7 @@
 .shortcut-table span:nth-child(odd) { text-align: right; }
 
 /* 사이드바. 근거: spec/ui/tokens.md §3(사이드바 행·섹션 제목). */
-.sidebar { position: relative; display: flex; flex-direction: column; min-height: 0; background: var(--bg); border-right: 1px solid var(--border); outline: none; }
+.sidebar { position: relative; display: flex; flex-direction: column; min-width: 0; min-height: 0; background: var(--bg); border-right: 1px solid var(--border); outline: none; }
 .side-head { padding: 10px; display: flex; flex-direction: column; gap: 8px; }
 .search-box {
   display: flex; align-items: center; gap: 6px; height: 32px; padding: 0 10px; border-radius: 7px;
```


## C. 테스트 코드

### `frontend/src/screens/Shell.leftcol.test.tsx` — 새 파일

```tsx
// 왼쪽 열 너비(D130): 너비는 서버 목록이 아니라 왼쪽 열 전체가 갖는다. 명령 모음을 켜도 열이 넓어지지 않고,
// 너비를 끌어 바꾸면 서버 목록과 명령 모음이 함께 바뀐다. 근거: spec/ui/sidebar.md §1, spec/ui/sftp-snippets.md §1.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { Shell } from './Shell';
const invoke = vi.fn(async (channel: string) => (channel === 'store.get' ? structuredClone(useApp.getState().store) : undefined));

function seed(sidebarWidth: number, snippetsOpen: boolean) {
  invoke.mockClear();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({
    booted: true,
    vault: { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
    store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, sidebarWidth } } as Store,
  });
  useUi.setState({ sidebarOpen: true, snippetsOpen });
}

beforeEach(() => seed(290, true));

describe('왼쪽 열 너비 (D130)', () => {
  it('너비는 왼쪽 열이 갖고, 서버 목록에는 따로 너비를 주지 않는다', () => {
    const { container } = render(<Shell />);
    const leftcol = container.querySelector<HTMLElement>('.leftcol')!;
    expect(leftcol.style.width).toBe('290px');
    expect(container.querySelector<HTMLElement>('.leftcol .sidebar')!.style.width).toBe('');
    expect(container.querySelector<HTMLElement>('.leftcol .snippet-dock')!.style.width).toBe('');
  });

  it('명령 모음을 켜고 꺼도 왼쪽 열 너비는 같다', () => {
    const { container } = render(<Shell />);
    expect(container.querySelector('.snippet-dock')).toBeTruthy();
    expect(container.querySelector<HTMLElement>('.leftcol')!.style.width).toBe('290px');
    act(() => useUi.getState().setSnippetsOpen(false));
    expect(container.querySelector('.snippet-dock')).toBeNull();
    expect(container.querySelector<HTMLElement>('.leftcol')!.style.width).toBe('290px');
  });

  it('너비 조절 손잡이는 서버 목록 안이 아니라 왼쪽 열에 있다(명령 모음 옆에서도 잡힌다)', () => {
    const { container } = render(<Shell />);
    const handle = container.querySelector('.side-resize')!;
    expect(handle.parentElement?.classList.contains('leftcol')).toBe(true);
    expect(container.querySelector('.sidebar .side-resize')).toBeNull();
    expect(container.querySelectorAll('.side-resize')).toHaveLength(1);
  });

  it('손잡이를 끌면 열 너비가 따라 바뀌고(200~520), 놓을 때 한 번 저장한다', () => {
    const { container } = render(<Shell />);
    const leftcol = container.querySelector<HTMLElement>('.leftcol')!;
    const handle = container.querySelector('.side-resize')!;
    // jsdom 에는 PointerEvent 가 없어 같은 이름의 MouseEvent 로 보낸다.
    fireEvent(handle, new MouseEvent('pointerdown', { bubbles: true, clientX: 290 }));
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 350 })); });
    expect(leftcol.style.width).toBe('350px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 2000 })); });
    expect(leftcol.style.width).toBe('520px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 0 })); });
    expect(leftcol.style.width).toBe('200px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 330 })); });
    expect(invoke.mock.calls.filter((c) => c[0] === 'store.patch')).toHaveLength(0); // 끄는 동안은 저장하지 않는다
    act(() => { window.dispatchEvent(new MouseEvent('pointerup', { clientX: 330 })); });
    const patches = invoke.mock.calls.filter((c) => c[0] === 'store.patch') as unknown as [string, [{ settings: { sidebarWidth: number } }]][];
    expect(patches).toHaveLength(1);
    expect(patches[0][1][0].settings.sidebarWidth).toBe(330);
    // 놓은 뒤의 움직임은 반영하지 않는다.
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 500 })); });
    expect(useApp.getState().store?.settings.sidebarWidth).toBe(330);
  });

  it('끌지 않고 눌렀다 떼기만 하면 저장하지 않는다', () => {
    const { container } = render(<Shell />);
    fireEvent(container.querySelector('.side-resize')!, new MouseEvent('pointerdown', { bubbles: true, clientX: 290 }));
    act(() => { window.dispatchEvent(new MouseEvent('pointerup', { clientX: 290 })); });
    expect(invoke.mock.calls.filter((c) => c[0] === 'store.patch')).toHaveLength(0);
  });

  it('서버 목록을 접으면 왼쪽 열 대신 세로 막대가 남는다(너비 손잡이 없음)', () => {
    const { container } = render(<Shell />);
    act(() => useUi.setState({ sidebarOpen: false }));
    expect(container.querySelector('.leftcol')).toBeNull();
    expect(container.querySelector('.rail')).toBeTruthy();
    expect(container.querySelector('.side-resize')).toBeNull();
  });
});
```

### `frontend/src/styles/leftcol.layout.test.ts` — 새 파일

```ts
// 왼쪽 열이 안의 내용 때문에 넓어지지 않게 하는 스타일 규칙 (D130, spec/ui/sidebar.md §1).
// 화면 테스트(jsdom)는 배치를 계산하지 않으므로, 규칙이 스타일 파일에 있는지를 파일에서 확인한다. 동작은 screens/Shell.leftcol.test.tsx.
import { describe, expect, it } from 'vitest';
// 테스트 실행기는 가져온 css 를 빈 내용으로 바꾸므로(?raw 도) 파일을 직접 읽는다. 이 프로젝트에는 node 타입이 없다.
// @ts-expect-error node:fs 의 타입 선언이 없다
import { readFileSync } from 'node:fs';

/** 이 파일 옆의 스타일 파일을 글자로 읽는다(줄 끝은 LF 로). */
function readStyle(name: string): string {
  const path = decodeURIComponent(new URL(name, import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
  return String(readFileSync(path, 'utf8')).replace(/\r\n/g, '\n');
}

const APP_CSS = readStyle('./app.css');

/** 스타일 파일에서 선택자 하나의 선언 글(중괄호 안)을 모두 이어 돌려준다. */
function cssOf(selector: string): string {
  const esc = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return [...APP_CSS.matchAll(new RegExp(`(?:^|\\n)${esc}\\s*\\{([^}]*)\\}`, 'g'))].map((m) => m[1]).join(' ');
}

describe('왼쪽 열 스타일 (D130)', () => {
  it('왼쪽 열은 줄지도 늘지도 않고(flex: none) 손잡이의 기준이며, 오른쪽 테두리를 갖는다', () => {
    const css = cssOf('.leftcol');
    expect(css).toMatch(/flex:\s*none/);
    expect(css).toMatch(/position:\s*relative/);
    expect(css).toMatch(/border-right:/);
  });

  it('명령 모음·서버 목록은 내용이 길어도 열을 넓히지 않는다(min-width: 0, 명령 모음은 넘치는 것을 자른다)', () => {
    expect(cssOf('.snippet-dock')).toMatch(/min-width:\s*0/);
    expect(cssOf('.snippet-dock')).toMatch(/overflow:\s*hidden/);
    expect(cssOf('.leftcol-body')).toMatch(/min-width:\s*0/);
    expect(cssOf('.sidebar')).toMatch(/min-width:\s*0/);
  });

  it('열 안의 서버 목록은 오른쪽 테두리를 따로 그리지 않는다(테두리가 열 가운데에 서지 않는다)', () => {
    expect(cssOf('.leftcol-body > .sidebar')).toMatch(/border-right:\s*none/);
    expect(cssOf('.sidebar')).not.toMatch(/(^|[;\s])width:/);
  });
});
```


## F. 명세·문서

### `spec/ui/sidebar.md` — 수정

```diff
--- a/spec/ui/sidebar.md
+++ b/spec/ui/sidebar.md
@@ -7,6 +7,9 @@
 1. **머리**: 검색 입력(돋보기 아이콘, 플레이스홀더 "서버 찾기"), 그 아래 버튼 3개 [+ 서버] [폴더 추가 아이콘 + 폴더] [가져오기]. 가져오기 버튼 툴팁 "PuTTY / SSH config 가져오기".
 2. **트리**(스크롤): 즐겨찾기 섹션 → 최근 접속 섹션 → "서버" 섹션(폴더 트리 + 폴더 없는 서버)
 3. 오른쪽 가장자리의 **너비 조절 손잡이**: 끌어서 너비 변경. **최소 200px, 최대 520px**. 끌기를 마치면 설정(sidebarWidth)에 저장하고 터미널 크기 재계산을 알린다. 기본 260px.
+   - (2026-11-25, D130) 이 너비는 서버 목록만의 것이 아니라 **왼쪽 열 전체**(아이콘 묶음 · 서버 목록 · 그 아래의 명령 모음)의 너비다. 셋은 늘 같은 너비이고, 안의 내용(긴 문구 등)이 열을 넓히지 않는다 — 넘치는 글은 줄을 바꾸거나 잘린다. 명령 모음을 켜고 꺼도 열 너비는 바뀌지 않는다.
+   - 손잡이는 **왼쪽 열의 오른쪽 가장자리 전체 높이**에서 잡힌다(서버 목록 옆에서도, 명령 모음 옆에서도). 끄는 동안은 화면에만 반영하고 놓을 때 한 번 저장한다. 끌지 않고 눌렀다 떼기만 하면 저장하지 않는다.
+   - 오른쪽 테두리 선은 왼쪽 열이 한 번 그린다(열 전체 높이).
 
 ## 2. 검색
 
```

### `spec/ui/shell.md` — 수정

```diff
--- a/spec/ui/shell.md
+++ b/spec/ui/shell.md
@@ -4,7 +4,7 @@
 
 ## 1. 전체 배치
 
-창은 운영체제 기본 프레임(제목 "DariTerm")을 쓰고, **앱 안에는 별도 제목 줄이 없다**. 프레임 아래는 왼쪽부터 **왼쪽 열(아이콘 묶음 + 서버 목록)**, **작업 영역(탭 막대 + 탭 내용)**, **그룹별 SFTP 패널(열려 있을 때만, 그룹 오른쪽)** 이고, 맨 아래가 **상태 표시줄**이다. 서버 목록 너비는 설정(sidebarWidth 260)에 저장되고 경계를 끌어 조절한다.
+창은 운영체제 기본 프레임(제목 "DariTerm")을 쓰고, **앱 안에는 별도 제목 줄이 없다**. 프레임 아래는 왼쪽부터 **왼쪽 열(아이콘 묶음 + 서버 목록 + 열려 있으면 그 아래 명령 모음 — 셋이 같은 너비, ui/sidebar.md §1)**, **작업 영역(탭 막대 + 탭 내용)**, **그룹별 SFTP 패널(열려 있을 때만, 그룹 오른쪽)** 이고, 맨 아래가 **상태 표시줄**이다. 서버 목록 너비는 설정(sidebarWidth 260)에 저장되고 경계를 끌어 조절한다.
 
 ## 2. 왼쪽 아이콘 묶음과 탭 막대
 
```

### `spec/ui/sftp-snippets.md` — 수정

```diff
--- a/spec/ui/sftp-snippets.md
+++ b/spec/ui/sftp-snippets.md
@@ -8,7 +8,7 @@
 - 그룹에 탭이 없으면 그 그룹의 SFTP 패널은 보이지 않는다(열림 상태는 기억).
 - 왼쪽 가장자리를 끌어 **너비 조절**: 최소 260px, 최대 760px, 기본 340px. 끌기를 마치면 설정(rightPanelWidth)에 저장(두 그룹이 같은 너비를 쓴다).
 - 머리: "SFTP"(폴더 트리 아이콘) 제목, 오른쪽 끝 닫기(X).
-- **스니펫 패널**은 왼쪽 **서버 목록 열의 아래쪽**에 붙는다. 기본 높이는 열의 **35%**(서버 목록 65%), 둘 사이 경계를 위아래로 끌어 **15~80%** 로 바꾸며 이 PC 에만 기억한다(두 번 누르면 기본값). 서버 목록·스니펫은 각각 최소 80px (D55). 열기/닫기: 왼쪽 아이콘 묶음의 **스니펫(코드 `<>` 아이콘)** 버튼. 서버 목록이 접혀 있을 때 누르면 서버 목록을 펴면서 스니펫을 연다.
+- **스니펫 패널**은 왼쪽 **서버 목록 열의 아래쪽**에 붙는다. 기본 높이는 열의 **35%**(서버 목록 65%), 둘 사이 경계를 위아래로 끌어 **15~80%** 로 바꾸며 이 PC 에만 기억한다(두 번 누르면 기본값). 서버 목록·스니펫은 각각 최소 80px (D55). **너비는 서버 목록과 같다**(왼쪽 열의 너비, 열의 오른쪽 가장자리를 끌면 함께 바뀐다 — ui/sidebar.md §1, 2026-11-25 D130). 열기/닫기: 왼쪽 아이콘 묶음의 **스니펫(코드 `<>` 아이콘)** 버튼. 서버 목록이 접혀 있을 때 누르면 서버 목록을 펴면서 스니펫을 연다.
 
 ## 2. SFTP 탭
 
```

### `spec/acceptance/02-shell-sidebar.md` — 수정

```diff
--- a/spec/acceptance/02-shell-sidebar.md
+++ b/spec/acceptance/02-shell-sidebar.md
@@ -40,6 +40,8 @@
 | B8 | 서버 우클릭 → 복제 | | 이름 "… (복사본)", 새 id, 최근 사용 기록 없음 |
 | B9 | 서버 선택 후 F2 / Delete | | 편집 대화상자 / 삭제 확인 |
 | B10 | 사이드바 경계를 500px→800px 로 끌기 | | 520px 에서 멈추고 설정에 저장 |
+| B10a | 사이드바 너비 290px, 명령 모음 켬(명령 0개 — 긴 안내 문구가 보임) | | 서버 목록·명령 모음·왼쪽 열이 모두 290px. 작업 영역이 290px 에서 시작한다. 켜기 전과 열 너비가 같다 (D130) |
+| B10b | 명령 모음 옆(왼쪽 열 아래쪽)의 오른쪽 가장자리를 끌기 | | 서버 목록과 명령 모음이 함께 넓어지고 좁아진다 (D130) |
 
 ## 빠른 접속 (ui/dialogs.md §2)
 | # | 조건 | 동작 | 기대 결과 |
```

`CHANGELOG.md` 의 미배포 절 "수정"에 한 줄, `docs/decisions.md` 에 D130 을 넣었다.

---

## G. 검증 (2026-11-25 실행)

### G.1 실제 브라우저 엔진으로 너비 재기

자동 테스트의 화면 환경(jsdom)은 배치를 계산하지 않아 "열이 벌어진다"를 볼 수 없다. 그래서 이 앱의 실제 스타일 파일(`tokens.css`·`app.css`·`rightpanel.css`)과 왼쪽 열과 같은 구조의 시험 페이지를 만들어, 이 앱이 쓰는 것과 같은 엔진(Edge, 헤드리스)에 넣고 너비를 쟀다. 고치기 전 것은 고치기 전의 `app.css` 와 전의 구조(서버 목록에 인라인 너비, 손잡이가 서버 목록 안)로 만들었다. 명령 0개(안내 문구가 보이는 상태), 창 1100×860.

| 너비 설정 | | 왼쪽 열 | 서버 목록 | 명령 모음 | 작업 영역 시작 | 손잡이 높이 / 열 높이 |
|---|---|---|---|---|---|---|
| 200 | 전 | **358** | 200 | **358** | 358 | 473 / 768 |
| 200 | 뒤 | 200 | 199 | 199 | 200 | 768 / 768 |
| 290 | 전 | **358** | 290 | **358** | 358 | 473 / 768 |
| 290 | 뒤 | 290 | 289 | 289 | 290 | 768 / 768 |
| 520 | 전 | 520 | 520 | 520 | 520 | 473 / 768 |
| 520 | 뒤 | 520 | 519 | 519 | 520 | 768 / 768 |

- 고치기 전: 설정이 358 보다 좁으면 열과 명령 모음이 358 로 벌어지고 서버 목록만 설정 너비다 — 사진의 모습(서버 목록 약 290, 터미널 시작 약 350)과 같다. 설정이 358 보다 넓으면(520) 문제가 없다.
- 고친 뒤: 어느 설정에서나 셋이 같은 너비다(안쪽 값이 1 작은 것은 열의 오른쪽 테두리 1px).
- 손잡이: 전에는 서버 목록 높이(473)만큼, 뒤에는 열 전체 높이(768)만큼.

이 측정은 시험 페이지로 한 것이고 앱을 띄워 본 것은 아니다. 시험 페이지의 스타일과 구조는 앱의 것과 같지만, 앱 화면에서의 확인은 §H 에 남긴다.

### G.2 프런트 — 새 파일 [frontend/src/screens/Shell.leftcol.test.tsx](../../frontend/src/screens/Shell.leftcol.test.tsx) 6개

| 테스트 | 확인하는 것 |
|---|---|
| 너비는 왼쪽 열이 갖고, 서버 목록에는 따로 너비를 주지 않는다 | `.leftcol` 의 인라인 너비 = 설정값, `.sidebar`·`.snippet-dock` 에는 인라인 너비 없음 |
| 명령 모음을 켜고 꺼도 왼쪽 열 너비는 같다 | — |
| 너비 조절 손잡이는 서버 목록 안이 아니라 왼쪽 열에 있다 | 손잡이의 부모가 `.leftcol`, 하나뿐 |
| 손잡이를 끌면 열 너비가 따라 바뀌고(200~520), 놓을 때 한 번 저장한다 | 끄는 동안 열 너비가 바뀜, 520·200 에서 멈춤, 끄는 동안 저장 0번, 놓을 때 `store.patch` 1번(`sidebarWidth: 330`), 놓은 뒤의 움직임은 무시 |
| 끌지 않고 눌렀다 떼기만 하면 저장하지 않는다 | — |
| 서버 목록을 접으면 왼쪽 열 대신 세로 막대가 남는다 | 손잡이 없음 |

### G.3 프런트 — 새 파일 [frontend/src/styles/leftcol.layout.test.ts](../../frontend/src/styles/leftcol.layout.test.ts) 3개

스타일 파일을 읽어, 열이 벌어지지 않게 하는 규칙이 지워지지 않았는지 본다.

| 테스트 | 확인하는 것 |
|---|---|
| 왼쪽 열은 `flex: none`·`position: relative`·오른쪽 테두리 | 너비가 줄거나 늘지 않고, 손잡이의 기준이 된다 |
| 명령 모음·서버 목록은 내용이 길어도 열을 넓히지 않는다 | `.snippet-dock`·`.leftcol-body`·`.sidebar` 의 `min-width: 0`, `.snippet-dock` 의 `overflow: hidden` |
| 열 안의 서버 목록은 오른쪽 테두리를 따로 그리지 않는다 | `.leftcol-body > .sidebar` 의 `border-right: none`, `.sidebar` 규칙에 `width` 가 없음 |

### G.4 실행한 명령

```bash
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 53개 파일 357개 통과 (앞 문서까지 51개 파일 348개 + 이 문서 9개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
# §G.1: 시험 페이지 6개를 만들어
msedge --headless=new --window-size=1100,860 --dump-dom file:///…/after-290.html   # 페이지의 스크립트가 잰 값을 읽는다
```

기존 사이드바 테스트(여러 개 선택, 폴더 삭제, 규칙)와 뼈대 테스트는 고치지 않고 통과한다.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 명령을 하나도 등록하지 않은 상태에서 왼쪽 위 `<>` 를 누른다 | 서버 목록 아래에 명령 모음이 **서버 목록과 같은 너비**로 나온다. 서버 목록 오른쪽에 빈 띠가 없고, 터미널이 오른쪽으로 밀리지 않는다 |
| 2 | 안내 문구 | "자주 쓰는 명령을 저장해 두고 클릭 한 번으로 실행하세요."가 열 너비에 맞춰 줄을 바꿔 보인다(잘리지 않는다) |
| 3 | `<>` 를 여러 번 켜고 끈다 | 터미널 너비가 바뀌지 않는다 |
| 4 | 왼쪽 열 오른쪽 가장자리를 **서버 목록 옆에서** 끈다 | 서버 목록과 명령 모음이 함께 넓어지고 좁아진다. 200px·520px 에서 멈춘다 |
| 5 | 같은 가장자리를 **명령 모음 옆에서** 끈다 | 같다(전에는 여기서는 잡히지 않았다) |
| 6 | 너비를 바꾼 뒤 앱을 껐다 켠다 | 바꾼 너비 그대로 |
| 7 | 서버 목록과 명령 모음 사이 가로 경계를 위아래로 끈다 | 높이 비율이 바뀐다(전과 같다) |
| 8 | 명령을 여러 개 등록(이름·내용이 긴 것 포함) | 명령 모음이 열 너비를 넘지 않고, 긴 글은 줄임표로 잘린다 |
| 9 | 서버 목록을 접었다 편다(`Ctrl+Shift+L`) | 접으면 세로 막대, 펴면 전의 너비 |
| 10 | 라이트 테마 | 오른쪽 테두리선이 열 전체 높이로 한 줄 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 서버 목록의 너비 | 인라인 너비를 뺐다. 열 밖에서 서버 목록을 쓰는 곳이 있으면 너비가 없어진다 | 서버 목록은 뼈대(`Shell`)에서만 쓴다. 테스트에서 따로 그릴 때는 너비가 필요 없다 |
| 손잡이의 자리 | 열로 옮겼다. 열의 오른쪽 가장자리에서 6px 폭으로 작업 영역 쪽으로 3px 걸친다(전과 같은 값) | §G.1 에서 손잡이의 왼쪽 위치가 전과 같다(너비 290 일 때 286) |
| 가로 경계와 겹침 | 서버 목록·명령 모음 사이 가로 경계의 오른쪽 끝 6px 이 손잡이와 겹친다. 그 자리는 가로 경계가 위다 | 겹치는 곳은 6×6px. 수동 검증 5·7 |
| 명령 모음의 긴 내용 | `overflow: hidden` 으로 넘치는 것을 자른다 | 명령 목록은 원래 줄임표로 자른다. 목록의 세로 스크롤은 안쪽 요소가 맡아 영향이 없다. 수동 검증 8 |
| 테두리선 | 아이콘 묶음 줄 오른쪽에도 선이 생겼다 | 열 전체가 한 덩어리로 보인다. 싫으면 `.toolrow` 쪽만 뺄 수 있다 |
| 저장 횟수 | 눌렀다 떼기만 하면 저장하지 않게 됐다 | 값이 같을 때만이다 |
| 다른 패널 | 오른쪽 SFTP 패널의 너비 조절은 건드리지 않았다 | — |

되돌리기: §B 의 `-` 줄로 되돌리고 새 테스트 파일 2개를 지운다.

## J. 체크리스트

- [x] `Shell.tsx`: 왼쪽 열에 너비, 손잡이를 열에
- [x] `Sidebar.tsx`: 인라인 너비·손잡이 빼기, `useSidebarWidth`·`SidebarResizer`
- [x] `app.css`: 열의 테두리·기준·`min-width: 0`, 명령 모음 `overflow: hidden`, 열 안 서버 목록의 테두리 없앰
- [x] 테스트 9개 추가. `tsc`·`vitest`(357개)·빌드
- [x] 실제 브라우저 엔진으로 고치기 전·후 너비 측정(§G.1)
- [x] 명세 ui/sidebar.md §1·ui/shell.md §1·ui/sftp-snippets.md §1·수용 B10a·B10b, CHANGELOG, decisions D130, CLAUDE.md
- [ ] 수동 검증 1~10
