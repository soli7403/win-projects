# 탭 막대의 + 로 연 세션이 다른 쪽 화면에 생기는 문제 — 원인·수정·코드·검증

**작성**: 2026-11-23 · **상태**: 코드 수정·자동 테스트·빌드 완료(D119). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(보고 즉시 수정해 등록하지 않음)
**관련**: [spec/ui/shell.md](../../spec/ui/shell.md) §2.3(탭 그룹), [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §2.2(빠른 접속 조작), [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md) G5a~G5c

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

화면을 좌우로 나눈 상태에서 **오른쪽에 포커스가 있을 때 왼쪽 탭 막대의 + 를 눌러 세션을 열면 오른쪽에 생겼다.** + 버튼이 "어느 쪽 탭 막대의 버튼인지"를 넘기지 않았고, 새 탭은 언제나 포커스가 있는 쪽에 붙었기 때문이다. 이제 + 로 띄운 빠른 접속은 그 + 가 속한 쪽(그룹)을 기억하고 **그쪽에** 연다.

| 빠른 접속을 띄운 방법 | 여는 곳 (고친 뒤) | 고치기 전 |
|---|---|---|
| 왼쪽 탭 막대의 + | 왼쪽 그룹 | 포커스 그룹 |
| 오른쪽 탭 막대의 + | 오른쪽 그룹 | 포커스 그룹 |
| 단축키(Ctrl+K 등), 시작 화면의 [빠른 접속] | 포커스 그룹 | 포커스 그룹 (그대로) |
| 빈 창의 [빠른 접속…] | 그 빈 창 | 그 빈 창 (그대로) |

## 1. 사용자 보고

2026-11-23: "좌우 패널이 열려있고, 오른쪽이 포커스가 있다. 이때 왼쪽 패널의 + 를 눌러서 세션을 추가하면 왼쪽에 추가되는 것이 아니고 오른쪽에 추가된다. + 가 속한 패널에 생성이 되도록 수정."

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 그룹 | 화면을 좌우로 나눴을 때의 한쪽(왼쪽 = 0, 오른쪽 = 1). 그룹마다 자기 탭 막대와 선택된 탭이 있다 |
| 포커스 그룹 | 마지막으로 탭이나 창을 누른 쪽. 열 위쪽에 강조 선이 보인다. 코드에서는 `activeTabId` 가 가리키는 탭의 그룹 |
| 빠른 접속 | + 버튼이나 Ctrl+K 로 뜨는 검색 창. 서버를 골라 Enter 로 연다 |

## 2. 원인 분석 (Cause)

호출 경로(고치기 전):

```
TabBar 의 + 클릭                      frontend/src/screens/TabBar.tsx
  └ openQuick(null)                   ← 어느 그룹의 + 인지 넘기지 않는다
      └ 빠른 접속에서 Enter           frontend/src/screens/QuickPalette.tsx  runRow
          └ openHost(hostId, { kind: 'tab' })
              └ openHostAt → ws.openInNewTab(info)        frontend/src/state/actions.ts
                  └ withNewTab(s, tab)                    frontend/src/state/workspace.ts
                        cur = activeTabId 의 탭           ← 포커스 그룹
                        새 탭의 group = cur 의 그룹
```

| # | 어디 | 무엇이 문제인가 |
|---|---|---|
| 1 | `TabBar.tsx` 의 + 버튼 | 탭 막대는 자기 그룹 번호(`g`)를 알고 있는데(SFTP 버튼은 `toggleSftp(g)` 로 넘긴다) + 버튼은 `openQuick(null)` 만 불렀다 |
| 2 | `ui.ts` 의 빠른 접속 상태 | `{ open, targetPane }` 뿐이라 그룹을 실어 둘 자리가 없었다 |
| 3 | `workspace.ts` `withNewTab` | 새 탭의 그룹을 항상 `activeTabId`(포커스 그룹)에서 정했다. 그룹을 받아들이는 길이 없었다 |

+ 를 눌러도 포커스 그룹은 바뀌지 않는다(탭이나 창을 눌러야 바뀐다). 그래서 오른쪽에 포커스를 둔 채 왼쪽 + 를 누르면 1 → 3 을 거쳐 오른쪽에 붙었다.

같은 이유로 그 빠른 접속에서 Ctrl+Enter·Shift+Enter(나눠 열기)를 해도 **오른쪽 탭**이 나뉘었다(`openHostAt` 이 대상 창이 없으면 활성 탭의 포커스 창을 쓴다).

명세와의 관계: ui/shell.md §2.3 은 "상태 표시줄·단축키·사이드바에서 서버를 열 때 새 탭은 포커스 그룹에 생긴다"고만 적었고 + 버튼은 따로 정하지 않았다. 구현은 그 문장을 + 에도 그대로 적용한 것이다. 이번에 사용자 지시로 + 의 규칙을 명세에 더했다(§F).

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| **A. 그룹을 끝까지 넘긴다** | + 가 `openQuick(null, g)` 로 그룹을 넘기고, 빠른 접속 상태가 그것을 들고 있다가 `openHost(…, { kind: 'tab', group })` → `openInNewTab(info, group)` 으로 전한다 | **고름.** 세션이 열리는 동안(비동기) 사용자가 다른 곳을 눌러도 결과가 바뀌지 않는다. 취소하면 아무것도 바뀌지 않는다 |
| B. + 를 누를 때 그 그룹으로 포커스를 옮긴다 | + 클릭에서 그 그룹의 선택 탭을 `activateTab` 한 뒤 지금처럼 연다 | 한 줄이면 되지만 Esc 로 취소해도 포커스 그룹이 바뀐 채 남고, 키 입력 포커스(터미널)는 원래 쪽에 있어 강조 선과 실제 입력 대상이 어긋난다 |

### 3.1 함께 정한 것

| # | 경우 | 동작 | 이유 |
|---|---|---|---|
| 1 | + 로 띄운 빠른 접속에서 **나눠 열기**(Ctrl+Enter·Shift+Enter) | 그 그룹에서 선택된 탭의 포커스 창을 나눈다 | 새 탭만 그쪽에 열리고 나누기는 반대쪽에서 일어나면 같은 문제가 남는다 |
| 2 | + 로 띄운 빠른 접속의 명령 모드 **"새 로컬 터미널"** | 그 그룹의 새 탭 | 같은 창에서 연 것은 같은 쪽에 |
| 3 | 오른쪽 + 로 띄운 뒤, 세션이 열리기 전에 **나뉨이 풀린 경우**(오른쪽 탭을 모두 닫음) | 왼쪽(하나뿐인 화면)에 붙는다 | 그대로 오른쪽에 붙이면 새 탭 하나 때문에 화면이 다시 나뉜다 |
| 4 | 나눠 열기가 **창 수 한도**에 걸려 새 탭으로 바뀔 때 | 그 새 탭을 나누려던 탭과 같은 그룹에 둔다 | 1번과 짝. 사이드바의 "아래 창으로 열기"는 나누려던 탭이 곧 포커스 탭이라 결과가 전과 같다 |
| 5 | + 를 누른 것만으로 포커스 그룹을 바꾸는가 | 바꾸지 않는다. 세션이 열리면 그 새 탭이 선택되면서 그 그룹이 포커스가 된다 | 안 B 를 고르지 않은 이유와 같다 |

### 3.2 영향 범위

프런트엔드만 고쳤다. 백엔드(Go)·IPC·저장 형식·문구는 바뀌지 않는다. `openInNewTab` 과 `openQuick` 에 **선택 인자**를 더한 것이라 기존 호출(인자 없음)은 전과 똑같이 동작한다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/state/ui.ts` | 코드 | 수정 |
| `frontend/src/state/workspace.ts` | 코드 | 수정 |
| `frontend/src/state/actions.ts` | 코드 | 수정 |
| `frontend/src/screens/TabBar.tsx` | 코드 | 수정 |
| `frontend/src/screens/QuickPalette.tsx` | 코드 | 수정 |
| `frontend/src/screens/TabBar.plus.test.tsx` | 테스트 | 새 파일 |
| `spec/ui/shell.md` | 명세 | 수정 |
| `spec/ui/dialogs.md` | 명세 | 수정 |
| `spec/acceptance/02-shell-sidebar.md` | 명세 | 수정 |
| `docs/decisions.md` | 문서 | D119 추가 |

아래의 `diff` 는 **같은 날의 앞 문서들(D106~D118)을 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `frontend/src/state/ui.ts` — 수정

```diff
--- a/frontend/src/state/ui.ts
+++ b/frontend/src/state/ui.ts
@@ -74,9 +74,10 @@
   openPreview: (sessionId: string, path: string) => void;
   closePreview: () => void;
   /** 위치 선택 대화상자: 서버 id. null 이면 닫힘. */
-  /** 빠른 접속 팔레트 상태. targetPane 이 있으면 그 빈 칸에서 연다. */
-  quick: { open: boolean; targetPane: string | null };
-  openQuick: (targetPane?: string | null) => void;
+  /** 빠른 접속 팔레트 상태. targetPane 이 있으면 그 빈 칸에서 연다.
+   *  group 이 있으면(탭 막대의 + 로 열었을 때) 그 탭 그룹에 연다. 없으면 포커스 그룹 (D119). */
+  quick: { open: boolean; targetPane: string | null; group: 0 | 1 | null };
+  openQuick: (targetPane?: string | null, group?: 0 | 1 | null) => void;
   closeQuick: () => void;
   /** 사이드바에서 선택된 서버 id. */
   selectedHostId: string | null;
@@ -153,8 +154,8 @@
   preview: null,
   openPreview: (sessionId, path) => set({ preview: { sessionId, path } }),
   closePreview: () => set({ preview: null }),
-  quick: { open: false, targetPane: null },
-  openQuick: (targetPane = null) => set({ quick: { open: true, targetPane } }),
+  quick: { open: false, targetPane: null, group: null },
+  openQuick: (targetPane = null, group = null) => set({ quick: { open: true, targetPane, group } }),
   closeQuick: () => set((s) => ({ quick: { ...s.quick, open: false } })),
   selectedHostId: null,
   setSelectedHost: (id) => set({ selectedHostId: id }),
```

### `frontend/src/state/workspace.ts` — 수정

```diff
--- a/frontend/src/state/workspace.ts
+++ b/frontend/src/state/workspace.ts
@@ -63,8 +63,8 @@
   mergeGroups: () => void;
   closeTab: (tabId: string, force?: boolean) => Promise<void>;
   focusPane: (paneId: string) => void;
-  /** 세션을 새 탭으로 연다. */
-  openInNewTab: (info: SessionInfo) => void;
+  /** 세션을 새 탭으로 연다. group 을 주면 그 그룹에, 아니면 포커스 그룹에. */
+  openInNewTab: (info: SessionInfo, group?: GroupIdx) => void;
   /** 포커스된 칸(또는 지정 칸)에 분할로 새 세션을 붙인다. before 면 앞. */
   openInPane: (targetPaneId: string | null, dir: Dir, info: SessionInfo, before?: boolean) => void;
   /** 빈 칸 채우기 등 칸에 세션을 넣는다(구조 불변). */
@@ -175,10 +175,14 @@
   return { tabs: ts, activeByGroup: ab, activeTabId: f ?? null };
 }
 
-/** 새 탭을 포커스 그룹에 붙이고 선택한다. */
-function withNewTab(s: WorkspaceState, tab: Tab): Placement {
+/**
+ * 새 탭을 붙이고 선택한다. group 을 주면 그 그룹에, 아니면 포커스 그룹에 (D119).
+ * 오른쪽 그룹을 줬는데 그 사이 나뉨이 풀렸으면(오른쪽 탭 없음) 왼쪽에 붙인다 — 새 탭 하나로 화면이 다시 나뉘지 않게.
+ */
+function withNewTab(s: WorkspaceState, tab: Tab, group?: GroupIdx): Placement {
   const cur = s.tabs.find((t) => t.id === s.activeTabId);
-  const nt: Tab = { ...tab, group: cur && groupOf(cur) === 1 ? 1 : undefined };
+  const g: GroupIdx = group === undefined ? (cur ? groupOf(cur) : 0) : group === 1 && !isSplit(s.tabs) ? 0 : group;
+  const nt: Tab = { ...tab, group: g === 1 ? 1 : undefined };
   return settle([...s.tabs, nt], s.activeByGroup, nt.id);
 }
 
@@ -360,11 +364,11 @@
     });
   },
 
-  openInNewTab(info) {
+  openInNewTab(info, group) {
     get().upsertSession(info);
     const pane: PaneNode = newPane(info.id);
     const tab: Tab = { id: newId('t'), broadcast: false, root: pane, focusedPane: pane.id, zoomed: null };
-    set((s) => withNewTab(s, tab));
+    set((s) => withNewTab(s, tab, group));
   },
 
   openInPane(targetPaneId, dir, info, before = false) {
```

### `frontend/src/state/actions.ts` — 수정

```diff
--- a/frontend/src/state/actions.ts
+++ b/frontend/src/state/actions.ts
@@ -38,11 +38,11 @@
   focusActiveTerminal();
 }
 
-/** where: 새 탭, 오른쪽 그룹의 새 탭, 또는 포커스 칸 기준 분할/빈 칸 채우기. */
+/** where: 새 탭(group 을 주면 그 그룹, 아니면 포커스 그룹), 오른쪽 그룹의 새 탭, 또는 포커스 칸 기준 분할/빈 칸 채우기. */
 async function openHostAt(
   hostId: string | null,
   where:
-    | { kind: 'tab' }
+    | { kind: 'tab'; group?: 0 | 1 }
     | { kind: 'rightGroup' }
     | { kind: 'split'; dir: Dir; before?: boolean; paneId?: string | null } = { kind: 'tab' },
   adhoc?: unknown,
@@ -51,7 +51,7 @@
   if (!info) return;
   const ws = useWorkspace.getState();
   if (where.kind === 'tab') {
-    ws.openInNewTab(info);
+    ws.openInNewTab(info, where.group);
     return;
   }
   if (where.kind === 'rightGroup') {
@@ -75,10 +75,10 @@
     ws.fillPane(targetId, info);
     return;
   }
-  // 분할 한도에 이르면 쪼개지 않고 새 탭에서 연다 (ui/panes.md §2).
+  // 분할 한도에 이르면 쪼개지 않고 새 탭에서 연다 (ui/panes.md §2). 새 탭은 나누려던 탭과 같은 그룹에 둔다.
   if (!canSplitMore(active.root)) {
     useApp.getState().pushToast({ kind: 'info', text: t('split.limitNewTab', { max: maxVerticalSplits() }) });
-    ws.openInNewTab(info);
+    ws.openInNewTab(info, groupOf(active));
     return;
   }
   ws.openInPane(targetId, where.dir, info, where.before);
```

### `frontend/src/screens/TabBar.tsx` — 수정

```diff
--- a/frontend/src/screens/TabBar.tsx
+++ b/frontend/src/screens/TabBar.tsx
@@ -236,7 +236,8 @@
           </div>
         );
       })}
-      <button className="icon-btn" title={tr('tab.newTitle')} aria-label={tr('tab.new')} onClick={() => openQuick(null)} style={{ marginLeft: 4, marginTop: 5 }}>
+      {/* + 는 이 탭 막대가 속한 그룹에 연다(포커스가 다른 그룹에 있어도, D119). */}
+      <button className="icon-btn" title={tr('tab.newTitle')} aria-label={tr('tab.new')} onClick={() => openQuick(null, g)} style={{ marginLeft: 4, marginTop: 5 }}>
         <span style={{ fontSize: 16, lineHeight: 1 }}>+</span>
       </button>
       <button
```

### `frontend/src/screens/QuickPalette.tsx` — 수정

```diff
--- a/frontend/src/screens/QuickPalette.tsx
+++ b/frontend/src/screens/QuickPalette.tsx
@@ -4,7 +4,8 @@
 import { PROTOCOL_ICON as PROTO_ICON, protocolIconColor } from '../ui/protocolIcon';
 import { useApp } from '../state/app';
 import { useUi } from '../state/ui';
-import { openHost, runShortcut } from '../state/actions';
+import { openHost } from '../state/actions';
+import { useWorkspace } from '../state/workspace';
 import { rankHosts, isLocalPrefix, parseDirect } from '../lib/quick';
 import { effectiveColor, recentHosts } from '../lib/hosts';
 import { sidebarSubtitle } from '../lib/tree';
@@ -34,7 +35,7 @@
     if (cmdMode) {
       const word = input.trimStart().slice(1).trim().toLowerCase();
       const cmds: Row[] = [
-        { kind: 'cmd', key: 'c-local', label: t('sc.newLocal'), sub: t('quick.newTab'), run: () => runShortcut('newLocal') },
+        { kind: 'cmd', key: 'c-local', label: t('sc.newLocal'), sub: t('quick.newTab'), run: () => void openHost(null, { kind: 'tab', group: quick.group ?? undefined }, { host: '', protocol: 'local' }) },
         { kind: 'cmd', key: 'c-new-host', label: t('welcome.newServer'), run: () => openEditor({ kind: 'host', hostId: null, groupId: null }) },
         { kind: 'cmd', key: 'c-new-folder', label: t('quick.newFolder'), run: () => openEditor({ kind: 'folder', folderId: null, parentId: null }) },
         { kind: 'cmd', key: 'c-import', label: t('quick.import'), run: () => useUi.getState().openImportDialog() },
@@ -73,7 +74,7 @@
     return out;
     // toggleTheme 은 안정적인 함수 참조가 필요 없다(렌더마다 새로 만들어도 무방).
     // eslint-disable-next-line react-hooks/exhaustive-deps
-  }, [input, hosts, groups, cmdMode, store?.settings.theme]);
+  }, [input, hosts, groups, cmdMode, store?.settings.theme, quick.group]);
 
   useEffect(() => {
     setSel(0);
@@ -92,10 +93,14 @@
   const runRow = (r: Row, mod: 'tab' | 'right' | 'down') => {
     // 빈 칸의 [빠른 접속…]으로 열었으면(대상 칸 있음) 보조키 없는 Enter·클릭은 그 칸을 채운다 (panes.md §3).
     // openHost 는 대상 칸이 빈 칸이면 분할하지 않고 채운다.
+    // 탭 막대의 + 로 열었으면(그룹 있음) 그 그룹에 연다: 새 탭은 그 그룹에, 나누기는 그 그룹에서 선택된 탭의 포커스 칸에 (D119).
+    const ws = useWorkspace.getState();
+    const groupTabId = quick.group === null ? null : ws.activeByGroup[quick.group];
+    const groupPane = groupTabId ? ws.tabs.find((x) => x.id === groupTabId)?.focusedPane ?? null : null;
     const where =
       mod === 'tab' && !quick.targetPane
-        ? { kind: 'tab' as const }
-        : { kind: 'split' as const, dir: mod === 'right' ? ('row' as const) : ('col' as const), paneId: quick.targetPane };
+        ? { kind: 'tab' as const, group: quick.group ?? undefined }
+        : { kind: 'split' as const, dir: mod === 'right' ? ('row' as const) : ('col' as const), paneId: quick.targetPane ?? groupPane };
     close();
     if (r.kind === 'cmd') {
       r.run();
```


## C. 테스트 코드

### `frontend/src/screens/TabBar.plus.test.tsx` — 새 파일

```tsx
// 탭 막대의 + 로 연 세션은 그 + 가 속한 그룹에 생긴다 (D119, spec/ui/shell.md §2.3, 수용 G5a·G5b).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { groupOf, useWorkspace, type Tab } from '../state/workspace';
import { useUi } from '../state/ui';
import { setBackend, type Backend } from '../ipc/backend';
import type { SessionInfo } from '../ipc/types';
import { TabBar } from './TabBar';
import { QuickPalette } from './QuickPalette';

function tab(id: string, group?: 1): Tab {
  return { id, broadcast: false, group, root: { type: 'pane', id: `p-${id}`, sessionId: `s-${id}` }, focusedPane: `p-${id}`, zoomed: null };
}

function session(id: string) {
  return { info: { id, title: id, state: 'connected' } as unknown as SessionInfo, activity: false };
}

const ws = () => useWorkspace.getState();
let opened = 0;

beforeEach(() => {
  opened = 0;
  const invoke = vi.fn(async (channel: string) => {
    if (channel !== 'session.open') return undefined;
    opened += 1;
    return { id: `s-new${opened}`, title: `new${opened}`, state: 'connected' };
  });
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  // 왼쪽 그룹에 a, 오른쪽 그룹에 b. 포커스는 오른쪽(b).
  useWorkspace.setState({
    tabs: [tab('a'), tab('b', 1)],
    activeTabId: 'b',
    activeByGroup: ['a', 'b'],
    sessions: { 's-a': session('s-a'), 's-b': session('s-b') },
    writers: {},
  });
  useUi.setState({ quick: { open: false, targetPane: null, group: null } });
});

afterEach(() => {
  setBackend(null);
});

function renderSplit() {
  render(
    <>
      <div data-testid="left"><TabBar group={0} /></div>
      <div data-testid="right"><TabBar group={1} /></div>
      <QuickPalette />
    </>,
  );
}

function clickPlus(side: 'left' | 'right') {
  fireEvent.click(within(screen.getByTestId(side)).getByLabelText('새로 연결'));
}

/** 빠른 연결 창에서 맨 위 항목(서버가 없으므로 "새 로컬 터미널")을 실행한다. */
function runFirst(init: { ctrlKey?: boolean; shiftKey?: boolean } = {}) {
  fireEvent.keyDown(screen.getByLabelText('빠른 연결 입력'), { key: 'Enter', ...init });
}

describe('탭 막대의 + (D119)', () => {
  it('G5a: 포커스가 오른쪽일 때 왼쪽 + 로 열면 왼쪽 그룹에 생기고 포커스가 왼쪽으로 온다', async () => {
    renderSplit();
    clickPlus('left');
    expect(useUi.getState().quick).toMatchObject({ open: true, group: 0 });
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    const nt = ws().tabs[2];
    expect(groupOf(nt)).toBe(0);
    expect(ws().activeByGroup).toEqual([nt.id, 'b']);
    expect(ws().activeTabId).toBe(nt.id);
  });

  it('포커스가 왼쪽일 때 오른쪽 + 로 열면 오른쪽 그룹에 생긴다', async () => {
    ws().activateTab('a');
    renderSplit();
    clickPlus('right');
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    const nt = ws().tabs[2];
    expect(groupOf(nt)).toBe(1);
    expect(ws().activeByGroup).toEqual(['a', nt.id]);
  });

  it('G5b: 왼쪽 + 로 연 창에서 Shift+Enter(아래로 나누기)는 왼쪽 그룹의 선택 탭을 나눈다', async () => {
    renderSplit();
    clickPlus('left');
    runFirst({ shiftKey: true });
    await waitFor(() => expect(Object.keys(ws().sessions)).toHaveLength(3));
    expect(ws().tabs).toHaveLength(2);
    const a = ws().tabs.find((t) => t.id === 'a')!;
    const b = ws().tabs.find((t) => t.id === 'b')!;
    expect(a.root.type).toBe('split');
    expect(b.root.type).toBe('pane');
  });

  it('단축키로 연 빠른 연결(그룹 없음)은 지금처럼 포커스 그룹에 연다', async () => {
    renderSplit();
    useUi.getState().openQuick(null);
    await screen.findByLabelText('빠른 연결 입력');
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    expect(groupOf(ws().tabs[2])).toBe(1);
  });
});

describe('openInNewTab 의 그룹 지정 (D119)', () => {
  const info = (id: string) => ({ id, title: id, state: 'connected' }) as unknown as SessionInfo;

  it('그룹을 주면 포커스와 관계없이 그 그룹에 붙고 선택된다', () => {
    ws().openInNewTab(info('x'), 0);
    const x = ws().tabs[2];
    expect(groupOf(x)).toBe(0);
    expect(ws().activeTabId).toBe(x.id);
  });

  it('오른쪽을 줬는데 나뉨이 이미 풀렸으면 왼쪽에 붙는다(새 탭 하나로 다시 나뉘지 않는다)', () => {
    ws().mergeGroups();
    ws().openInNewTab(info('x'), 1);
    expect(ws().tabs.every((t) => groupOf(t) === 0)).toBe(true);
  });
});
```


## F. 명세

### `spec/ui/shell.md` — 수정

```diff
--- a/spec/ui/shell.md
+++ b/spec/ui/shell.md
@@ -57,6 +57,7 @@
 - **나누기**: 탭 우클릭 메뉴의 "오른쪽 화면으로 보내기", 또는 탭을 끌어 작업 영역의 오른쪽 절반에 놓는다. 끄는 탭이 **작업 영역의 오른쪽 가장자리(너비의 10%, 최소 100px)** 에 들어가면 오른쪽 절반에 "여기에 놓으면 좌우로 나눕니다" 안내가 나타나고, 그 안내에 놓으면 나뉜다. 탭 막대 위에서는 안내를 띄우지 않는다(탭 순서 바꾸기). 안내는 포인터가 오른쪽 절반에 있는 동안 유지되고, 왼쪽 절반으로 가거나 끌기가 끝나면(놓기·취소) 사라진다(사용자 결정 2026-11-19, D101·D102). 탭이 1개뿐이면 나눌 수 없다.
 - **나뉜 모양**: 작업 영역이 같은 너비의 두 열이 되고, **각 열이 자기 탭 막대를 맨 위에 가지며** 그 아래에 그 그룹에서 선택된 탭의 내용을 바로 보여 준다. 각 열의 탭 막대 끝에는 + 버튼이 있다. 두 열 사이 경계(가운데)를 **끌어서** 좌우 너비 비율을 바꿀 수 있다(각 20~80%, 경계를 더블클릭하면 50:50). 비율은 저장 레이아웃에 함께 저장·복원된다.
 - **그룹별 선택**: 두 그룹은 각각 자기 선택 탭을 기억한다. 탭을 누르거나 그 안의 칸을 누르면 그 그룹이 **포커스 그룹**이 되고(열 위쪽에 강조 선), 상태 표시줄·단축키·사이드바에서 서버를 열 때 새 탭은 **포커스 그룹**에 생긴다.
+- **+ 버튼은 자기 그룹에 연다** (2026-11-23 사용자 지시, D119): 열의 탭 막대 끝 + 버튼으로 빠른 접속을 열어 연결하면, 포커스가 다른 그룹에 있어도 새 탭은 **그 + 버튼이 속한 그룹**에 생기고 그 그룹이 포커스 그룹이 된다. 그 빠른 접속에서 나눠 열기(Ctrl+Enter·Shift+Enter)를 고르면 그 그룹에서 선택된 탭의 포커스 칸을 나눈다. + 를 눌러 빠른 접속을 띄운 것만으로는 포커스 그룹이 바뀌지 않는다(취소하면 그대로). 단축키(Ctrl+K 등)로 연 빠른 접속은 포커스 그룹에 연다.
 - **옮기기**: 탭을 다른 그룹의 탭 막대(탭 위 또는 빈 곳)로 끌어 놓거나, 탭 우클릭의 "왼쪽/오른쪽 그룹으로 이동"을 쓴다. 옮겨진 탭은 대상 그룹의 선택 탭이 된다. 세션은 끊기지 않는다.
 - **합치기**: 한 그룹의 탭이 모두 없어지면(닫거나 옮겨서) 자동으로 나뉨이 풀려 하나가 된다. 탭 우클릭의 "한 화면으로 합치기"는 모든 탭을 왼쪽 그룹으로 합친다.
 - **순서 변경·닫기·전환**: 탭 순서 변경, 탭 닫기 후 다음 탭 선택(같은 그룹 안), Ctrl+Tab 류 탭 전환과 Ctrl+Shift+←/→ 이동은 **포커스 그룹 안의 탭**만 대상으로 한다.
```

### `spec/ui/dialogs.md` — 수정

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -61,6 +61,7 @@
 ### 2.2 조작
 - ↑/↓: 선택 이동(끝에서 멈춤, 선택 항목이 보이도록 스크롤). 마우스를 올려도 선택이 따라간다. 입력이 바뀌면 선택은 맨 위로.
 - **Enter: 새 탭에서 열기 / Ctrl+Enter: 오른쪽 분할 / Shift+Enter: 아래 분할.** 항목을 마우스로 클릭할 때도 같은 보조키 규칙.
+  - 화면이 나뉘어 있을 때 어느 그룹에 여는가: 탭 막대의 + 버튼으로 띄웠으면 **그 버튼이 속한 그룹**(새 탭은 그 그룹에, 나누기는 그 그룹에서 선택된 탭의 포커스 칸), 그 밖(단축키·시작 화면 버튼)에는 포커스 그룹 (ui/shell.md §2.3, D119). 명령 모드의 "새 로컬 터미널"도 같다.
 - Esc 또는 바깥 클릭: 닫기. 실행하면 대부분 닫힌다(새 서버 등록·새 폴더·가져오기·설정은 다른 대화상자로 바뀜).
 - 선택된 항목 오른쪽에 `Enter` 키 표시.
 - 바닥 안내줄: `Enter 새 탭` · `Ctrl+Enter 오른쪽 나누기` · `Shift+Enter 아래 나누기` · `> 명령`
```

### `spec/acceptance/02-shell-sidebar.md` — 수정

탭 그룹 표의 G5 와 G6 사이에 세 줄을 넣는다.

```diff
 | G5 | 나뉨, 오른쪽 열 선택 | 사이드바에서 서버 열기 | 새 탭이 오른쪽 그룹에 생기고 선택된다 |
+| G5a | 나뉨, 오른쪽 열 선택 | **왼쪽** 탭 막대의 + → 빠른 접속에서 서버를 Enter | 새 탭이 **왼쪽** 그룹에 생기고 선택되며 왼쪽이 포커스 그룹이 된다. 오른쪽 그룹의 선택 탭은 그대로 (D119) |
+| G5b | 나뉨, 오른쪽 열 선택 | 왼쪽 탭 막대의 + → 빠른 접속에서 Shift+Enter | 왼쪽 그룹에서 선택된 탭의 포커스 칸 아래에 나눠 열린다. 오른쪽 탭은 나뉘지 않는다 |
+| G5c | 나뉨, 오른쪽 열 선택 | 왼쪽 탭 막대의 + → Esc | 아무것도 열리지 않고 포커스 그룹은 오른쪽 그대로 |
 | G6 | 나뉨 | Ctrl+Tab | 포커스 그룹 안의 탭만 순환한다 |
```

---

## G. 검증 (2026-11-23 실행)

새 테스트 파일 [frontend/src/screens/TabBar.plus.test.tsx](../../frontend/src/screens/TabBar.plus.test.tsx) (6개). 왼쪽·오른쪽 탭 막대와 빠른 접속 창을 함께 그리고, 세션 열기는 가짜 백엔드가 받는다.

| 테스트 | 확인하는 것 |
|---|---|
| G5a: 포커스가 오른쪽일 때 왼쪽 + 로 열면 왼쪽 그룹에 생긴다 | + 클릭 뒤 빠른 접속 상태의 그룹이 0 / Enter 뒤 새 탭의 그룹이 0, 왼쪽의 선택 탭이 새 탭, 오른쪽의 선택 탭은 그대로, 포커스가 새 탭 |
| 포커스가 왼쪽일 때 오른쪽 + 로 열면 오른쪽 그룹에 생긴다 | 반대 방향 |
| G5b: 왼쪽 + 로 연 창에서 Shift+Enter 는 왼쪽 그룹의 선택 탭을 나눈다 | 탭 수는 그대로 2, 왼쪽 탭만 나뉘고 오른쪽 탭은 창 하나 |
| 단축키로 연 빠른 연결(그룹 없음)은 포커스 그룹에 연다 | 전의 동작이 그대로인지 |
| 그룹을 주면 포커스와 관계없이 그 그룹에 붙고 선택된다 | `openInNewTab(info, 0)` |
| 오른쪽을 줬는데 나뉨이 이미 풀렸으면 왼쪽에 붙는다 | §3.1 3번 — 새 탭 하나로 다시 나뉘지 않는다 |

```bash
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 44개 파일 285개 통과 (앞 문서까지 43개 파일 279개 + 이 문서 6개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

Go 코드는 바뀌지 않아 `go test` 는 다시 돌리지 않았다.

자동 테스트가 확인하지 못하는 것: 실제 화면에서 열 위쪽 강조 선이 새 탭 쪽으로 옮겨 가는지, 열린 터미널에 키 입력 포커스가 가는지, 마우스로 항목을 눌러 열 때(테스트는 Enter 만).

## H. 수동 검증 (남은 일)

준비: 탭 두 개를 열고 한 탭을 우클릭 → "오른쪽 화면으로 보내기"로 화면을 나눈다.

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 오른쪽 터미널을 눌러 포커스를 오른쪽에 둔다 → **왼쪽** 탭 막대의 + → 서버를 골라 Enter | 새 탭이 **왼쪽**에 생기고 선택된다. 강조 선이 왼쪽으로 옮겨 가고 바로 입력할 수 있다. 오른쪽 탭은 그대로 |
| 2 | 왼쪽에 포커스 → **오른쪽** 탭 막대의 + → 서버를 마우스로 클릭 | 새 탭이 오른쪽에 생긴다 |
| 3 | 오른쪽에 포커스 → 왼쪽 + → Shift+Enter | 왼쪽에서 선택된 탭의 창 아래에 나눠 열린다. 오른쪽은 나뉘지 않는다 |
| 4 | 오른쪽에 포커스 → 왼쪽 + → Esc | 아무것도 열리지 않는다. 강조 선은 오른쪽 그대로이고 오른쪽 터미널에 계속 입력된다 |
| 5 | 오른쪽에 포커스 → Ctrl+K → Enter | 새 탭이 오른쪽(포커스 그룹)에 생긴다 — 전과 같다 |
| 6 | 오른쪽에 포커스 → 왼쪽 + → `>` 입력 → "새 로컬 터미널" | 로컬 터미널이 왼쪽에 열린다 |
| 7 | 왼쪽 탭의 창을 한도(기본 3개)까지 나눈다 → 오른쪽에 포커스 → 왼쪽 + → Shift+Enter | 나누지 않고 새 탭에서 열리며 안내 토스트가 나온다. 그 새 탭은 **왼쪽**에 생긴다 |
| 8 | 나누지 않은 화면에서 + → Enter | 전과 같이 새 탭이 열린다 |
| 9 | 사이드바에서 서버 더블클릭, 우클릭 "오른쪽 화면에 열기"·"아래 창으로 열기" | 전과 같다(포커스 그룹 / 오른쪽 그룹 / 포커스 창 아래) |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 새 탭을 여는 다른 경로(사이드바, 단축키, 시작 화면, 로컬 터미널, 복원) | `openInNewTab` 의 그룹 인자를 주지 않으므로 전과 같은 줄을 탄다 | 기존 테스트(G5 등) 통과, 새 테스트 "단축키로 연 빠른 연결" |
| 빈 창의 [빠른 접속…] | `openQuick(pane.id)` — 그룹은 `null`, 대상 창이 우선한다 | `runRow` 가 `targetPane ?? groupPane` 순서로 본다 |
| 나누기가 한도에 걸릴 때의 새 탭 | 이제 나누려던 탭의 그룹에 생긴다. 대상 창을 따로 준 경우(창 머리글의 [아래로 나누기]는 한도에서 비활성이라 해당 없음, 서버를 다른 그룹 창의 가장자리에 끌어 놓기)만 결과가 달라질 수 있고, 달라진 쪽이 사용자가 가리킨 쪽이다 | 수동 검증 7 |
| 빠른 접속 상태의 모양 | `quick` 에 `group` 필드가 늘었다. 이 상태를 직접 만들어 넣는 곳은 `ui.ts` 뿐이다 | `tsc` 통과 |
| 명령 모드 "새 로컬 터미널" | 단축키 실행 함수를 거치지 않고 같은 일을 직접 한다(`openHost(null, { kind: 'tab', group }, { host: '', protocol: 'local' })`) | 수동 검증 6 |

되돌리기: §B 의 `-` 줄로 되돌리고 테스트 파일을 지운다. 저장 데이터에는 흔적이 없다.

## J. 체크리스트

- [x] 원인 확인(+ 가 그룹을 넘기지 않음, 새 탭은 항상 포커스 그룹)
- [x] 코드 수정 5개 파일
- [x] 테스트 6개 추가, `tsc`·`vitest`(285개) 통과, 빌드
- [x] 명세 ui/shell.md §2.3·ui/dialogs.md §2.2·수용 G5a~G5c, decisions D119
- [ ] 수동 검증 1~9
