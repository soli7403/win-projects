# 바둑판(그리드) 배열 제거 — 창 수 한도를 무시해 그 뒤 동작이 어긋나던 문제 — 원인·수정·코드·검증

**작성**: 2026-11-23 · **상태**: 코드 수정·자동 테스트·빌드 완료(D116). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(보고 즉시 수정해 등록하지 않음)
**관련**: [spec/ui/panes.md](../../spec/ui/panes.md) §2.1·§6, [spec/ui/shell.md](../../spec/ui/shell.md) §2.2·§2.3, [spec/ui/sidebar.md](../../spec/ui/sidebar.md) §8, [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §2, 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol. 같은 날 문서 묶음의 마지막: 앞 문서 [store-update](./2026-11-23-store-update.md).

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

"바둑판으로" 기능(모든 탭을 한 탭에 모으기, 폴더의 서버를 한 탭에 모두 열기)을 **없앴다.** 이 기능은 앱이 지키는 두 한도 — **화면 나누기 2개, 탭 하나의 창 수(기본 3)** — 를 무시하고 모든 창을 한 탭에 몰아넣어, 한 번 쓰고 나면 나누기·새로 열기·화면 나누기가 모두 어긋났다.

| 들어가는 곳 | 문구 | 처리 |
|---|---|---|
| 탭 우클릭 메뉴 | 모든 탭을 한 화면에 바둑판으로 | 삭제 |
| 사이드바 폴더 우클릭 메뉴 | 폴더 서버 모두 바둑판으로 열기 | 삭제(바로 아래 구분선도) |
| 빠른 연결(Ctrl+K)의 `>` 명령 | 모든 탭을 바둑판으로 | 삭제 |

요청은 "오른쪽 마우스 메뉴에서 빼 달라"였다. 빠른 연결 명령은 우클릭 메뉴는 아니지만 **같은 함수를 부르는 같은 문제**라 함께 뺐다(남겨 두면 같은 고장이 Ctrl+K 로 다시 난다).

## 1. 사용자 보고와 확인한 사실

2026-11-23 사용자 보고: "오른쪽 마우스 메뉴에 바둑판 배열을 빼줘. 이걸 동작하면 전체 기능이 이상해진다. 현재에서는 좌우 패널 2개만 생성되도록 되어 있고, 상하는 3개씩 총 한 화면에 6개까지 보여진다. 맞나 확인하고 수정해 달라."

### 1.1 "좌우 2개 × 상하 3개 = 6개"가 맞는가 — **맞다(기본 설정 기준)**

| 한도 | 값 | 코드 |
|---|---|---|
| 화면 나누기(탭 그룹) | 왼쪽·오른쪽 **2개 고정** | [workspace.ts](../../frontend/src/state/workspace.ts) 의 `activeByGroup: [왼쪽 선택 탭, 오른쪽 선택 탭]`, `moveTabToGroup(tabId, 0 \| 1)`. 명세 ui/shell.md §2.3 "그룹은 최대 2개다" |
| 탭 하나의 창 수 | 설정 "최대 화면 분할 (상하)" `maxVerticalSplits`, **기본 3**, 1~5 | [actions.ts:20-29](../../frontend/src/state/actions.ts#L20-L29) `maxVerticalSplits()`·`canSplitMore()` — `탭의 창 수 < 한도` 일 때만 더 나눈다 |
| 한 화면에 보이는 창 | 그룹 2 × 그룹마다 선택된 탭의 창 3 = **6개** | 화면에는 그룹마다 선택된 탭 하나만 보인다 |

덧붙일 점 두 가지:

- 설정을 5 로 올리면 2 × 5 = 10개까지다. "6개"는 기본값일 때의 수다.
- 한도는 "상하로 3개"가 아니라 **"탭의 창 수 3개"** 로 센다. 창 우클릭의 "오른쪽으로 나누기"(Ctrl+Shift+D)로 좌우로 나눈 창도 같은 수에 들어간다. 창 머리글에는 [아래로 나누기] 버튼만 있어서 보통은 상하로 쌓인다.

한도를 지키는 곳(모두 `canSplitMore` 를 본다):

| 곳 | 한도에 이르면 |
|---|---|
| 창 머리글 [아래로 나누기] — [Panes.tsx:246](../../frontend/src/screens/Panes.tsx#L246) | 버튼 비활성, 도움말 "아래로 나누기 (최대 N개)" |
| 창 우클릭 "오른쪽으로/아래로 나누기 (같은 서버)" — [Panes.tsx:197-198](../../frontend/src/screens/Panes.tsx#L197-L198) | 항목 비활성 |
| Ctrl+Shift+D / E — [actions.ts](../../frontend/src/state/actions.ts) `runShortcut` | 안내 토스트 "화면 나누기는 최대 N개까지입니다" |
| 나눠서 열기(빠른 연결의 Ctrl/Shift+Enter, 가장자리에 끌어 놓기) — [actions.ts](../../frontend/src/state/actions.ts) `openHostAt` | 나누지 않고 새 탭에서 열고 토스트 |

### 1.2 바둑판 기능은 이 한도를 **하나도 보지 않았다**

지운 코드(§B 의 diff 에 전체가 있다):

```ts
// frontend/src/state/workspace.ts (지움)
gridTabs() {
  const s = get();
  const panes: PaneNode[] = [];
  for (const t of s.tabs) collectPanes(t.root, panes);     // ① 모든 탭의 모든 창을 모은다 (개수 제한 없음)
  if (panes.length < 2) return;
  const tab: Tab = { id: newId('t'), title: t('host.localTerminal'), broadcast: false,
                     root: gridLayout(panes), focusedPane: panes[0].id, zoomed: null };   // ② 새 탭 하나
  set({ tabs: [tab], activeTabId: tab.id, activeByGroup: [tab.id, null] });               // ③ 탭 목록을 통째로 바꾼다
},

// frontend/src/lib/layout.ts (지움)
export function gridLayout(panes: PaneNode[]): LayoutNode {
  …
  const cols = Math.ceil(Math.sqrt(panes.length));          // ④ 열 수 = √n 올림 → 좌우로도 나뉜다
  …
}
```

## 2. 원인 분석 (Cause)

바둑판을 한 번 실행하면 생기는 일을 코드로 따라가면:

| # | 일어나는 일 | 코드의 어느 줄 | 사용자에게 보이는 모습 |
|---|---|---|---|
| 1 | 한 탭에 창이 한도보다 많이 들어간다 | ① 개수를 보지 않고 모은다. `canSplitMore` 를 부르지 않는다 | 탭 6개(창 6개)를 모으면 한 탭에 창 6개. 한도는 3 |
| 2 | 그 뒤로 **그 탭에서 아무것도 더 나눌 수 없다** | `canSplitMore(root)` = `창 수 < 3` 이 거짓 | [아래로 나누기]가 비활성, Ctrl+Shift+D/E 는 "최대 3개까지입니다", 나눠서 열기는 새 탭으로 빠진다. 창을 3개 미만으로 닫아야 풀린다 |
| 3 | 화면 나누기가 풀린다 | ③ `activeByGroup: [tab.id, null]` — 오른쪽 그룹이 사라진다. 새 탭에는 `group` 이 없다 | 좌우로 나눠 쓰던 화면이 하나로 합쳐진다 |
| 4 | 모든 탭이 사라지고 탭 이름이 없어진다 | ③ `tabs: [tab]`, ② 이름은 "로컬 터미널" 고정 | 탭마다 붙였던 이름, 탭별 일괄 입력 상태, 크게 보기 상태가 모두 없어진다. SSH 창만 있어도 탭 이름이 "로컬 터미널" |
| 5 | 창이 좌우로도 쪼개진다 | ④ 열 수 = ⌈√n⌉ | 창 4개 → 2×2, 6개 → 3열 2행. 앱의 다른 곳은 좌우 나누기를 화면 나누기(그룹)에 맡기고 창은 상하로만 쌓게 하는데(머리글의 좌우 나누기 버튼을 뺀 이유, 2026-11-12), 여기서만 좁은 창이 여럿 생긴다 |
| 6 | **되돌릴 방법이 없다** | 원래 탭 구성을 어디에도 남기지 않는다 | "창마다 탭으로 나누기"로 탭을 다시 만들 수는 있지만 이름·그룹·묶음은 돌아오지 않는다 |
| 7 | 다음 실행까지 이어진다 | 화면 배치 저장([layoutSave.ts](../../frontend/src/state/layoutSave.ts))이 지금 탭 구성을 그대로 저장 | "마지막 화면 배치 복원"이 켜져 있으면 다음 실행에도 한도를 넘은 탭이 열린다 |

"폴더 서버 모두 바둑판으로 열기"(`openFolderGrid` → `addGridTab`)도 같은 `gridLayout` 을 쓴다. 폴더에 서버가 10대면 **창 10개짜리 탭**이 생긴다(16대까지는 묻지도 않는다). 위 표의 1·2·5번이 그대로 일어난다.

**왜 이런 코드가 있었나**: 이 기능은 처음 명세(ui/panes.md §6)에 있던 것이고, 창 수 한도(`maxVerticalSplits`)와 화면 나누기(탭 그룹)는 그 뒤에 들어왔다(2026-11-11 ~ 11-12). 한도를 넣을 때 나누기·열기 경로에는 검사를 넣었지만 바둑판 경로는 빠졌다. 명세도 §2.1 의 한도와 §6 의 그리드가 서로 어긋난 채였다.

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| **A. 기능을 없앤다** | 세 들어가는 곳과 뒤의 코드를 지운다 | **고름.** 사용자 요청 그대로이고, "2 × 3" 구조와 맞지 않는 기능을 남길 이유가 없다 |
| B. 한도에 맞게 고친다 | 모을 때 창을 한도(탭당 3)씩 나눠 여러 탭으로, 좌우는 그룹 2개로 | 이름이 "바둑판"인데 결과가 바둑판이 아니게 된다. 요청 범위를 넘는다 |
| C. 메뉴만 지우고 코드는 둔다 | 항목 세 줄만 삭제 | 쓰지 않는 코드와 테스트·문구 7개가 남는다. 누군가 다시 연결하면 같은 고장 |

A 를 고르면서 **잃는 것**: "폴더의 서버를 한 번에 모두 열기"가 없어진다. 폴더의 서버는 하나씩 열어야 한다. 한 번에 여는 기능이 필요하면 한도에 맞는 방식(서버마다 탭 하나)으로 따로 만들 수 있다 — 이번에는 만들지 않았다(§J).

되돌리기: 코드만 지웠고 저장 데이터 형식은 건드리지 않았다. 이 문서의 diff 를 거꾸로 적용하면 돌아온다.

### 3.1 지운 것

| 파일 | 지운 것 |
|---|---|
| `frontend/src/screens/TabBar.tsx` | 메뉴 항목 `tab.grid`, `gridTabs` 가져오기 |
| `frontend/src/screens/Sidebar.tsx` | 메뉴 항목 `sb.openFolderGrid` 와 그 아래 구분선, `openFolderGrid` 가져오기 |
| `frontend/src/screens/QuickPalette.tsx` | 명령 `c-grid`, 쓰지 않게 된 `useWorkspace` 가져오기 |
| `frontend/src/state/workspace.ts` | `gridTabs`(인터페이스와 구현), `addGridTab`, `gridLayout` 가져오기 |
| `frontend/src/state/actions.ts` | `openFolderGrid`, `addGridTab` 가져오기 |
| `frontend/src/lib/layout.ts` | `gridLayout` |
| `frontend/src/lib/layout.test.ts` | "바둑판 모으기" 테스트 1개 |
| `locales/*.json` (10개) | `tab.grid`, `quick.grid`, `sb.openFolderGrid`, `folder.noServers`, `folder.openManyTitle`, `folder.openManyBody`, `folder.fallbackName` — 뒤의 4개는 `openFolderGrid` 만 쓰던 문구다 |

남긴 것: `collectPanes`("창마다 탭으로 나누기"가 쓴다), `equalSizes`(나누기가 쓴다), "창마다 탭으로 나누기" 메뉴.

### 3.2 고치지 않은 것 (범위 밖)

- **이미 한도를 넘은 탭**: 전에 바둑판으로 만든 탭이 저장된 배치에 남아 있으면 그대로 복원된다. 자동으로 쪼개지 않는다. 그 탭에서 "창마다 탭으로 나누기"를 하거나 창을 닫으면 정상 범위로 돌아온다.
- **한도를 세는 방식**("상하 3개"가 아니라 "탭의 창 수 3개"): §1.1 에 적은 대로다. 바꾸지 않았다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/lib/layout.ts` | 코드 | 수정 |
| `frontend/src/screens/QuickPalette.tsx` | 코드 | 수정 |
| `frontend/src/screens/Sidebar.tsx` | 코드 | 수정 |
| `frontend/src/screens/TabBar.tsx` | 코드 | 수정 |
| `frontend/src/state/actions.ts` | 코드 | 수정 |
| `frontend/src/state/workspace.ts` | 코드 | 수정 |
| `frontend/src/lib/layout.test.ts` | 테스트 | 수정 |
| `frontend/src/state/gridRemoved.test.ts` | 테스트 | 새 파일 |
| `spec/acceptance/02-shell-sidebar.md` | 명세 | 수정 |
| `spec/acceptance/03-panes-terminal.md` | 명세 | 수정 |
| `spec/ui/dialogs.md` | 명세 | 수정 |
| `spec/ui/panes.md` | 명세 | 수정 |
| `spec/ui/shell.md` | 명세 | 수정 |
| `spec/ui/sidebar.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 이 기능이 고친 파일은 앞 문서들과 겹치지 않는다(문구 파일 제외).

## B. 코드

### `frontend/src/lib/layout.ts` — 수정

```diff
--- a/frontend/src/lib/layout.ts
+++ b/frontend/src/lib/layout.ts
@@ -164,20 +164,6 @@
   return splitPane(without, targetId, dir, moving, before);
 }
 
-/** 그리드 배치: 열 수 = ceil(sqrt(n)). 행마다 가로 분할, 행들을 세로 분할(모두 균등). */
-export function gridLayout(panes: PaneNode[]): LayoutNode {
-  if (panes.length === 0) return newPane(null);
-  if (panes.length === 1) return panes[0];
-  const cols = Math.ceil(Math.sqrt(panes.length));
-  const rows: LayoutNode[] = [];
-  for (let i = 0; i < panes.length; i += cols) {
-    const chunk = panes.slice(i, i + cols);
-    rows.push(chunk.length === 1 ? chunk[0] : { type: 'split', id: newId('s'), dir: 'row', sizes: equalSizes(chunk.length), children: chunk });
-  }
-  if (rows.length === 1) return rows[0];
-  return { type: 'split', id: newId('s'), dir: 'col', sizes: equalSizes(rows.length), children: rows };
-}
-
 export type Edge = 'left' | 'right' | 'top' | 'bottom' | 'center';
 
 /**
```

### `frontend/src/screens/QuickPalette.tsx` — 수정

```diff
--- a/frontend/src/screens/QuickPalette.tsx
+++ b/frontend/src/screens/QuickPalette.tsx
@@ -4,7 +4,6 @@
 import { PROTOCOL_ICON as PROTO_ICON, protocolIconColor } from '../ui/protocolIcon';
 import { useApp } from '../state/app';
 import { useUi } from '../state/ui';
-import { useWorkspace } from '../state/workspace';
 import { openHost, runShortcut } from '../state/actions';
 import { rankHosts, isLocalPrefix, parseDirect } from '../lib/quick';
 import { effectiveColor, recentHosts } from '../lib/hosts';
@@ -41,7 +40,6 @@
         { kind: 'cmd', key: 'c-import', label: t('quick.import'), run: () => useUi.getState().openImportDialog() },
         { kind: 'cmd', key: 'c-settings', label: t('quick.openSettings'), run: () => useUi.getState().openSettings() },
         { kind: 'cmd', key: 'c-theme', label: t('quick.toggleTheme', { theme: store?.settings.theme === 'dark' ? t('appearance.light') : t('appearance.dark') }), run: () => toggleTheme() },
-        { kind: 'cmd', key: 'c-grid', label: t('quick.grid'), run: () => useWorkspace.getState().gridTabs() },
         { kind: 'cmd', key: 'c-lock', label: t('quick.lock'), run: () => void getBackend().invoke('vault.lock').catch((e) => useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) })) },
       ];
       return cmds.filter((c) => c.kind === 'cmd' && c.label.toLowerCase().includes(word));
```

### `frontend/src/screens/Sidebar.tsx` — 수정

```diff
--- a/frontend/src/screens/Sidebar.tsx
+++ b/frontend/src/screens/Sidebar.tsx
@@ -8,7 +8,7 @@
 import { useApp } from '../state/app';
 import { useUi, type MenuItem } from '../state/ui';
 import { effectiveColor } from '../lib/hosts';
-import { openFolderGrid, openHost } from '../state/actions';
+import { openHost } from '../state/actions';
 import { HOST_MIME } from './Panes';
 import { focusActiveTerminal } from './TerminalView';
 import { rangeSelection, toggleSelection } from '../lib/multiSelect';
@@ -292,8 +292,6 @@
   ];
 
   const folderMenu = (g: Group): MenuItem[] => [
-    { label: tr('sb.openFolderGrid'), onSelect: () => void openFolderGrid(g.id) },
-    { separator: true },
     { label: tr('sb.addServerHere'), onSelect: () => openEditor({ kind: 'host', hostId: null, groupId: g.id }) },
     { label: tr('sb.newSubfolder'), onSelect: () => openEditor({ kind: 'folder', folderId: null, parentId: g.id }) },
     { label: tr('folder.edit'), onSelect: () => openEditor({ kind: 'folder', folderId: g.id, parentId: g.parentId }) },
```

### `frontend/src/screens/TabBar.tsx` — 수정

```diff
--- a/frontend/src/screens/TabBar.tsx
+++ b/frontend/src/screens/TabBar.tsx
@@ -62,7 +62,6 @@
   const renameTab = useWorkspace((s) => s.renameTab);
   const moveTab = useWorkspace((s) => s.moveTab);
   const toggleBroadcast = useWorkspace((s) => s.toggleBroadcast);
-  const gridTabs = useWorkspace((s) => s.gridTabs);
   const openMenu = useUi((s) => s.openMenu);
   const prompt = useUi((s) => s.prompt);
   // 파일 패널 항목을 끌고 탭 위에 있을 때: 0.5초 머물면 그 탭으로 전환, 놓으면 그 탭 세션의 마지막 폴더로 복사 (file-panel.md §2.2).
@@ -108,7 +107,6 @@
       ...(split ? [{ label: tr('tab.merge'), onSelect: () => mergeGroups() } as MenuItem] : []),
       { label: t.broadcast ? tr('tab.broadcastOff') : tr('tab.broadcastOn'), shortcut: 'Ctrl+Shift+B', disabled: panes < 2, onSelect: () => toggleBroadcast(t.id) },
       { label: tr('tab.detach'), disabled: panes < 2, onSelect: () => detachPanes(t.id) },
-      { label: tr('tab.grid'), onSelect: () => gridTabs() },
       { separator: true },
       { label: tr('tab.closeOthers'), onSelect: () => void closeOthers(t) },
       { label: tr('tab.close'), shortcut: 'Ctrl+Shift+W', danger: true, onSelect: () => void confirmClose(t) },
```

### `frontend/src/state/actions.ts` — 수정

```diff
--- a/frontend/src/state/actions.ts
+++ b/frontend/src/state/actions.ts
@@ -3,7 +3,7 @@
 import type { SessionInfo } from '../ipc/types';
 import { useApp } from './app';
 import { useUi } from './ui';
-import { addGridTab, groupOf, openSessionFor, useWorkspace, type Tab } from './workspace';
+import { groupOf, openSessionFor, useWorkspace, type Tab } from './workspace';
 import { findPane, paneIds, type Dir, type LayoutNode } from '../lib/layout';
 import { sameTargetRequest } from '../lib/sameTarget';
 import type { ShortcutAction } from '../lib/keys';
@@ -225,34 +225,6 @@
   useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
 }
 
-/** 폴더 전체를 그리드로 연다(panes §6). 하위 폴더 포함, sort 오름차순, 16개 초과는 확인. */
-export async function openFolderGrid(groupId: string): Promise<void> {
-  const st = useApp.getState().store;
-  if (!st) return;
-  const group = st.groups.find((g) => g.id === groupId);
-  const ids = new Set<string>([groupId]);
-  let grew = true;
-  while (grew) {
-    grew = false;
-    for (const g of st.groups) if (g.parentId && ids.has(g.parentId) && !ids.has(g.id)) { ids.add(g.id); grew = true; }
-  }
-  const hosts = st.hosts.filter((h) => h.groupId && ids.has(h.groupId)).sort((a, b) => a.sort - b.sort);
-  if (hosts.length === 0) {
-    useApp.getState().pushToast({ kind: 'info', text: t('folder.noServers') });
-    return;
-  }
-  if (hosts.length > 16) {
-    const ok = await useUi.getState().confirm({ title: t('folder.openManyTitle'), body: t('folder.openManyBody', { count: hosts.length }), okText: t('common.ok') });
-    if (!ok) return;
-  }
-  const infos: SessionInfo[] = [];
-  for (const h of hosts) {
-    const info = await openSessionFor(h.id);
-    if (info) infos.push(info);
-  }
-  addGridTab(group?.name ?? t('folder.fallbackName'), infos);
-}
-
 /** 칸을 다른 탭으로 옮기지 않는다(panes §7 은 같은 탭 안만). 탭 분리는 detachPanes 를 쓴다. */
 export { detachPanes } from './workspace';
 
```

### `frontend/src/state/workspace.ts` — 수정

```diff
--- a/frontend/src/state/workspace.ts
+++ b/frontend/src/state/workspace.ts
@@ -3,7 +3,7 @@
 import { getBackend, errorMessage } from '../ipc/backend';
 import type { SessionInfo } from '../ipc/types';
 import {
-  findPane, findPaneBySession, gridLayout, movePaneToEdge, newId, newPane, paneIds, removePane, resizeSplit,
+  findPane, findPaneBySession, movePaneToEdge, newId, newPane, paneIds, removePane, resizeSplit,
   sessionsIn, splitPane, swapSessions, type Dir, type LayoutNode, type PaneNode,
 } from '../lib/layout';
 import { useApp } from './app';
@@ -75,7 +75,6 @@
   movePane: (tabId: string, moving: string, target: string, dir: Dir, before: boolean) => void;
   toggleBroadcast: (tabId: string) => void;
   toggleZoom: (tabId: string) => void;
-  gridTabs: () => void;
   /** 빈 칸 생성(분할 시 빈 칸을 먼저 놓을 때). */
   splitEmpty: (paneId: string, dir: Dir, before?: boolean) => void;
 }
@@ -475,15 +474,6 @@
       tabs: patchTab(s.tabs, tabId, (x) => ({ ...x, zoomed: x.zoomed ? null : x.focusedPane })),
     }));
   },
-
-  gridTabs() {
-    const s = get();
-    const panes: PaneNode[] = [];
-    for (const t of s.tabs) collectPanes(t.root, panes);
-    if (panes.length < 2) return;
-    const tab: Tab = { id: newId('t'), title: t('host.localTerminal'), broadcast: false, root: gridLayout(panes), focusedPane: panes[0].id, zoomed: null };
-    set({ tabs: [tab], activeTabId: tab.id, activeByGroup: [tab.id, null] });
-  },
 }));
 
 /** 칸의 세션 id 를 바꾼다(null 이면 빈 칸). */
@@ -527,17 +517,6 @@
   useWorkspace.setState(settle(tabs, s.activeByGroup, created[0].id));
 }
 
-/** 세션들을 그리드 한 탭으로 연다(panes §6 폴더 전체 그리드). infos 는 이미 연 세션들. */
-export function addGridTab(title: string, infos: SessionInfo[]): void {
-  if (infos.length === 0) return;
-  const s = useWorkspace.getState();
-  for (const info of infos) s.upsertSession(info);
-  const panes = infos.map((i) => newPane(i.id));
-  const root = gridLayout(panes);
-  const tab: Tab = { id: newId('t'), title, broadcast: false, root, focusedPane: panes[0].id, zoomed: null };
-  useWorkspace.setState((st) => withNewTab(st, tab));
-}
-
 /** 복원: 저장된 탭 목록으로 교체한다(layout 복원, lifecycle §3.2). */
 export function replaceTabs(tabs: Tab[], activeIndex: number, infos: SessionInfo[], activeIndex1?: number, ratio?: number): void {
   for (const info of infos) useWorkspace.getState().upsertSession(info);
```


## C. 테스트 코드

### `frontend/src/lib/layout.test.ts` — 수정

```diff
--- a/frontend/src/lib/layout.test.ts
+++ b/frontend/src/lib/layout.test.ts
@@ -1,7 +1,7 @@
 // 칸·분할 규칙 테스트. 근거: spec/ui/panes.md §3~§7.
 import { describe, expect, it } from 'vitest';
 import {
-  edgeToSplit, edgeZone, findPaneBySession, gridLayout, MIN_PCT, movePaneToEdge, newPane, paneIds,
+  edgeToSplit, edgeZone, findPaneBySession, MIN_PCT, movePaneToEdge, newPane, paneIds,
   removePane, resizeSplit, splitPane, swapSessions, type LayoutNode, type SplitNode,
 } from './layout';
 
@@ -100,16 +100,3 @@
   });
 });
 
-describe('바둑판 모으기 (panes.md §6)', () => {
-  it('열 수는 ceil(sqrt(n)), 창 n개가 모두 남는다', () => {
-    for (const n of [2, 3, 4, 5, 9]) {
-      const panes = Array.from({ length: n }, (_, i) => newPane(`s${i}`));
-      const root = gridLayout(panes);
-      expect(paneIds(root)).toHaveLength(n);
-    }
-    const four = gridLayout(Array.from({ length: 4 }, (_, i) => newPane(`s${i}`))) as SplitNode;
-    expect(four.dir).toBe('col');
-    expect((four.children[0] as SplitNode).children).toHaveLength(2);
-  });
-});
-
```

### `frontend/src/state/gridRemoved.test.ts` — 새 파일 (전체)

```ts
// 바둑판(그리드) 배열 기능을 없앤 것을 지킨다 (D116, spec/ui/panes.md §6).
// 이 기능은 한 탭의 창 수 한도와 화면 나누기 2개를 무시해 그 뒤 동작을 어긋나게 했다. 다시 들어오지 않게 막는다.
import { describe, expect, it } from 'vitest';
import * as layout from '../lib/layout';
import * as workspace from './workspace';
import { LOCALES } from '../i18n';

const SOURCES = import.meta.glob('../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

describe('바둑판 배열 제거 (D116)', () => {
  it('작업 공간에 모든 탭을 한 탭으로 모으는 동작이 없다', () => {
    expect('gridTabs' in workspace.useWorkspace.getState()).toBe(false);
    expect('addGridTab' in workspace).toBe(false);
    expect('gridLayout' in layout).toBe(false);
  });

  it('메뉴·명령 문구가 어느 언어에도 없다', () => {
    for (const [lang, dict] of Object.entries(LOCALES)) {
      for (const key of ['tab.grid', 'quick.grid', 'sb.openFolderGrid']) {
        expect(dict && key in dict, `${lang}: ${key}`).toBe(false);
      }
    }
  });

  it('화면 코드(탭 메뉴·폴더 메뉴·빠른 접속)가 그 동작을 부르지 않는다', () => {
    const offenders = Object.entries(SOURCES)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([, code]) => /gridTabs|openFolderGrid|addGridTab|gridLayout|'c-grid'/.test(code))
      .map(([path]) => path);
    expect(offenders).toEqual([]);
  });

  it('화면 나누기(탭 그룹)는 왼쪽·오른쪽 두 개뿐이다', () => {
    expect(workspace.useWorkspace.getState().activeByGroup).toHaveLength(2);
  });
});
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`folder.noServers`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`folder.openManyTitle`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`folder.openManyBody`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`folder.fallbackName`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`tab.grid`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`quick.grid`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`sb.openFolderGrid`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.


## F. 명세

### `spec/acceptance/02-shell-sidebar.md` — 수정

```diff
--- a/spec/acceptance/02-shell-sidebar.md
+++ b/spec/acceptance/02-shell-sidebar.md
@@ -5,7 +5,7 @@
 ## 제목 표시줄·탭·상태 표시줄 (ui/shell.md)
 | # | 조건 | 동작 | 기대 결과 |
 |---|---|---|---|
-| S1 | 탭 1개 | 오른쪽 버튼 줄 확인 | "모든 탭을 바둑판으로" 버튼이 없다. 2개 이상이면 나타난다 |
+| S1 | 탭이 몇 개든 | 탭 우클릭 메뉴, 폴더 우클릭 메뉴, 빠른 접속의 `>` 명령 목록 확인 | 바둑판(그리드)으로 모으거나 여는 항목이 **어디에도 없다** (2026-11-23, D116) |
 | S2 | 볼트 열림 | 왼쪽 위 아이콘 묶음 확인 | 잠그기 버튼이 없다. 상태 표시줄의 "잠금 해제됨"을 누르면 잠금 화면이 된다 (D67) |
 | S3 | 탭 더블클릭 | 이름 입력 후 확인 | 탭 제목이 바뀐다. 빈 값으로 확인하면 자동 제목으로 돌아간다 |
 | S4 | 비활성 탭의 세션에 새 출력 | 탭 확인 | 상태 점이 "활동" 색(강조색)으로 바뀌고, 탭을 열면 사라진다 |
```

### `spec/acceptance/03-panes-terminal.md` — 수정

```diff
--- a/spec/acceptance/03-panes-terminal.md
+++ b/spec/acceptance/03-panes-terminal.md
@@ -18,8 +18,9 @@
 | P11 | Ctrl+Shift+Enter(칸 2개 이상) | | 포커스된 칸만 크게 보이고 다시 누르면 원복 |
 | P12 | Alt+→ | | 오른쪽 가장 가까운 칸으로 포커스 이동, 없으면 변화 없음 |
 | P13 | 동시 입력 켜기(칸 2개 이상) | 한 칸에 입력 | 탭의 연결된 모든 칸에 입력 |
-| P14 | 탭 3개 | "바둑판로 모으기" | 모든 칸이 한 탭 "로컬 터미널" 에 열 수=⌈√n⌉ 그리드로 모인다 |
-| P15 | 폴더 우클릭 → 그리드로 열기, 서버 20개 | | "20개 서버에 동시에 연결할까요?" 확인 |
+| P14 | (삭제 2026-11-23, D116) | ~~"바둑판로 모으기"~~ | 기능을 없앴다 |
+| P15 | (삭제 2026-11-23, D116) | ~~폴더 우클릭 → 그리드로 열기~~ | 기능을 없앴다 |
+| P15a | 기본 설정(최대 화면 분할 3), 화면을 좌우 둘로 나눔 | 양쪽 탭에서 각각 아래로 나누기를 되풀이 | 탭마다 창 3개에서 [아래로 나누기]가 비활성이 된다. 한 화면에 보이는 창은 최대 6개다 (D116) |
 
 ## 터미널 (behavior/terminal.md)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/ui/dialogs.md` — 수정

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -45,8 +45,9 @@
 4. PuTTY / SSH config 가져오기 — 가져오기 대화상자
 5. 설정 열기
 6. 테마 전환 (현재가 다크면 "라이트", 라이트면 "다크"를 괄호에 표시) — 앱 테마와 터미널 테마를 함께 바꾼다(다크↔`DariTerm Dark`/라이트↔`DariTerm Light`)
-7. 모든 탭을 그리드로 모으기
-8. 앱 잠그기
+7. 앱 잠그기
+
+("모든 탭을 그리드로 모으기" 명령은 2026-11-23 에 없앴다 — D116, ui/panes.md §6.)
 
 **(b) 직접 접속 주소처럼 보이는 입력**: `[ssh://|telnet://][사용자@]호스트[:포트]` 형식이면서 `@`가 있거나, IPv4 모양이거나, 점(.)이 있거나, `ssh`/`telnet`으로 시작할 때, 목록 맨 위에 "바로 연결: {입력}" 항목(부제 "저장하지 않고 한 번만 연결")을 둔다. 프로토콜 기본값은 ssh, 포트 생략 시 기본 포트.
 
```

### `spec/ui/panes.md` — 수정

```diff
--- a/spec/ui/panes.md
+++ b/spec/ui/panes.md
@@ -104,9 +104,13 @@
 
 ## 6. 그리드 모으기 / 분리
 
-- **그리드로 모으기**(제목 표시줄 버튼, 탭 우클릭 메뉴, 빠른 접속 명령): 모든 탭의 모든 칸을 한 탭으로 모은다. 칸이 2개 미만이면 아무 일도 하지 않는다. 새 탭 이름은 "로컬 터미널"이고 다른 탭은 모두 사라진다(세션은 유지).
-- **그리드 배치 규칙**: 칸 n개 → 열 수 = n의 제곱근을 올림. 칸을 열 수만큼씩 끊어 행을 만들고, 한 행에 칸이 여럿이면 가로 분할(균등), 행이 여럿이면 세로 분할(균등). 행이 하나면 그 행 자체, 칸이 하나면 칸 자체.
-- **폴더 전체를 그리드로 열기**(사이드바 폴더): 그 폴더와 모든 하위 폴더의 서버를 정렬 순서(sort 오름차순)로 모두 접속하여 그리드로 배치한 새 탭을 만든다. 탭 이름은 폴더 이름. 서버가 없으면 정보 토스트 "이 폴더에 서버가 없습니다". **16개를 넘으면** 확인한다(제목 "여러 서버 열기", 본문 "{n}개 서버에 동시에 연결할까요?"). 접속에 실패한 서버는 조용히 건너뛴다.
+> **2026-11-23 삭제 (D116)**: 아래의 "그리드로 모으기"·"그리드 배치 규칙"·"폴더 전체를 그리드로 열기"는 **기능을 없앴다**(탭 우클릭 메뉴, 폴더 우클릭 메뉴, 빠른 접속 명령 모두). 이 기능은 한 탭의 칸 수 한도(§2.1 6번, 설정 `maxVerticalSplits`)와 화면 나누기 2개(ui/shell.md §2.3)를 무시하고 모든 칸을 한 탭에 넣어, 그 뒤로 나누기·새로 열기·화면 나누기가 어긋났다. 세 문장은 기록으로만 남긴다 — **구현하지 않는다.** "패널을 각각 탭으로 분리"는 그대로 있다.
+>
+> **한 화면에 보이는 칸 수**: 화면 나누기(왼쪽·오른쪽 그룹) 최대 2개 × 그룹마다 선택된 탭의 칸 최대 `maxVerticalSplits`개(기본 3, 설정 1~5) = 기본 **6개**(설정을 5 로 올리면 10개).
+
+- ~~**그리드로 모으기**~~(삭제)(제목 표시줄 버튼, 탭 우클릭 메뉴, 빠른 접속 명령): 모든 탭의 모든 칸을 한 탭으로 모은다. 칸이 2개 미만이면 아무 일도 하지 않는다. 새 탭 이름은 "로컬 터미널"이고 다른 탭은 모두 사라진다(세션은 유지).
+- ~~**그리드 배치 규칙**~~(삭제): 칸 n개 → 열 수 = n의 제곱근을 올림. 칸을 열 수만큼씩 끊어 행을 만들고, 한 행에 칸이 여럿이면 가로 분할(균등), 행이 여럿이면 세로 분할(균등). 행이 하나면 그 행 자체, 칸이 하나면 칸 자체.
+- ~~**폴더 전체를 그리드로 열기**~~(삭제)(사이드바 폴더): 그 폴더와 모든 하위 폴더의 서버를 정렬 순서(sort 오름차순)로 모두 접속하여 그리드로 배치한 새 탭을 만든다. 탭 이름은 폴더 이름. 서버가 없으면 정보 토스트 "이 폴더에 서버가 없습니다". **16개를 넘으면** 확인한다(제목 "여러 서버 열기", 본문 "{n}개 서버에 동시에 연결할까요?"). 접속에 실패한 서버는 조용히 건너뛴다.
 - **패널을 각각 탭으로 분리**: 탭 우클릭 메뉴(ui/shell.md §2.2).
 
 ## 7. 끌어 놓기 (칸 이동·서버 놓기)
```

### `spec/ui/shell.md` — 수정

```diff
--- a/spec/ui/shell.md
+++ b/spec/ui/shell.md
@@ -46,7 +46,6 @@
 | 탭 이름 바꾸기 | 더블클릭과 같음 | |
 | 이 탭의 모든 패널에 동시 입력 / 동시 입력 끄기 (현재 상태에 따라 문구 전환) | 동시 입력 토글. 단축키 Ctrl+Shift+B 표기 | 칸이 2개 미만 |
 | 패널을 각각 탭으로 분리 | 이 탭의 칸마다 새 탭을 만들어 원래 탭 자리에 순서대로 끼우고 첫 번째를 활성화 | 칸이 2개 미만 |
-| 모든 탭을 이 창 그리드로 모으기 | 모든 탭의 칸을 한 탭으로 모아 그리드로 배치 (ui/panes.md §6) | |
 | ―구분선― | | |
 | 다른 탭 모두 닫기 | 이 탭을 제외한 모든 탭을 확인 없이 닫는다 | |
 | 탭 닫기 (Ctrl+Shift+W 표기, 위험색) | 탭 닫기 규칙(ui/panes.md §5) | |
@@ -61,7 +60,7 @@
 - **옮기기**: 탭을 다른 그룹의 탭 막대(탭 위 또는 빈 곳)로 끌어 놓거나, 탭 우클릭의 "왼쪽/오른쪽 그룹으로 이동"을 쓴다. 옮겨진 탭은 대상 그룹의 선택 탭이 된다. 세션은 끊기지 않는다.
 - **합치기**: 한 그룹의 탭이 모두 없어지면(닫거나 옮겨서) 자동으로 나뉨이 풀려 하나가 된다. 탭 우클릭의 "한 화면으로 합치기"는 모든 탭을 왼쪽 그룹으로 합친다.
 - **순서 변경·닫기·전환**: 탭 순서 변경, 탭 닫기 후 다음 탭 선택(같은 그룹 안), Ctrl+Tab 류 탭 전환과 Ctrl+Shift+←/→ 이동은 **포커스 그룹 안의 탭**만 대상으로 한다.
-- "모든 탭을 한 화면에 바둑판으로"는 나뉨을 풀고 한 탭으로 만든다.
+- ("모든 탭을 한 화면에 바둑판으로" 항목은 2026-11-23 에 없앴다 — D116, ui/panes.md §6.)
 - 저장 레이아웃(data/store.md §5)에 그룹 정보가 함께 저장·복원된다.
 
 ## 3. 상태 표시줄
```

### `spec/ui/sidebar.md` — 수정

```diff
--- a/spec/ui/sidebar.md
+++ b/spec/ui/sidebar.md
@@ -72,8 +72,6 @@
 
 | 항목 | 동작 |
 |---|---|
-| 폴더 전체를 그리드로 열기 | ui/panes.md §6 |
-| ―구분선― | |
 | 이 폴더에 새 서버 | 서버 편집(이 폴더 지정) |
 | 하위 폴더 만들기 | 폴더 편집(상위=이 폴더) |
 | 폴더 편집 (이름·색·환경) | 폴더 편집 |
```



---

## G. 검증 (2026-11-23 실행)

### G.1 자동 테스트

새 파일 [frontend/src/state/gridRemoved.test.ts](../../frontend/src/state/gridRemoved.test.ts) — 이 기능이 다시 들어오지 않게 막는다.

| 테스트 | 확인하는 것 |
|---|---|
| 작업 공간에 모든 탭을 한 탭으로 모으는 동작이 없다 | 상태에 `gridTabs` 가 없다 / `workspace` 모듈에 `addGridTab` 이 없다 / `layout` 모듈에 `gridLayout` 이 없다 |
| 메뉴·명령 문구가 어느 언어에도 없다 | 10개 언어 모두에 `tab.grid`·`quick.grid`·`sb.openFolderGrid` 가 없다 |
| 화면 코드가 그 동작을 부르지 않는다 | 테스트가 아닌 모든 `.ts`/`.tsx` 소스에 `gridTabs`·`openFolderGrid`·`addGridTab`·`gridLayout`·`'c-grid'` 가 없다 |
| 화면 나누기(탭 그룹)는 왼쪽·오른쪽 두 개뿐이다 | `activeByGroup` 길이 2 |

기존 테스트: [layout.test.ts](../../frontend/src/lib/layout.test.ts) 에서 "바둑판 모으기" 1개를 지웠고 나머지는 그대로 통과. 문구 검사([i18n.test.ts](../../frontend/src/i18n/i18n.test.ts))는 10개 언어의 키가 같은지 본다 — 통과.

### G.2 실행 결과

```bash
(cd frontend && npx tsc --noEmit)      # 통과 (쓰지 않는 가져오기 없음)
(cd frontend && npx vitest run)        # PowerShell 에서. 43개 파일 278개 통과 (전 42개 275개: 1개 지우고 4개 추가)
go test ./internal/i18n/               # 통과 (백엔드 쪽 문구 검사)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

**자동 테스트가 확인하지 못하는 것**: 메뉴가 실제 화면에서 어떻게 보이는지(항목이 빠진 자리, 구분선). 그리고 §1.1 의 창 수 한도(`canSplitMore`) 자체에는 직접 테스트가 없다 — 이번에 바꾸지 않은 코드이고, 그 함수가 든 `actions.ts` 는 터미널 화면 모듈을 끌어와 단위 테스트하기 어렵다(정리 거리 §J).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 탭을 2개 이상 열고 탭을 우클릭 | "모든 탭을 한 화면에 바둑판으로"가 없다. "창마다 탭으로 나누기" 다음이 바로 구분선과 "나머지 탭 닫기" |
| 2 | 사이드바의 폴더를 우클릭 | "폴더 서버 모두 바둑판으로 열기"가 없다. 메뉴가 "이 폴더에 서버 추가"로 시작하고 맨 위에 구분선이 남아 있지 않다 |
| 3 | Ctrl+K → `>` 입력 | 명령 목록에 "모든 탭을 바둑판으로"가 없다. "앱 잠그기"는 있다 |
| 4 | 기본 설정에서 화면을 좌우로 나누고, 양쪽 탭에서 [아래로 나누기]를 되풀이 | 탭마다 창 3개에서 버튼이 비활성. 한 화면에 창 6개 |
| 5 | 설정 > 화면 > 최대 화면 분할을 5 로 | 탭마다 5개까지, 한 화면에 10개 |
| 6 | (전에 바둑판으로 만든 탭이 복원돼 있다면) 그 탭 우클릭 → "창마다 탭으로 나누기" | 창마다 탭이 되고, 그 뒤 나누기가 정상으로 된다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 나누기·열기·화면 나누기 | 지운 코드는 세 들어가는 곳에서만 불렸다. 다른 경로는 건드리지 않았다 | `tsc`(쓰지 않는 것·없는 것 모두 오류), 기존 테스트 277개 |
| "창마다 탭으로 나누기" | `collectPanes` 를 같이 쓰던 기능. 남겼다 | 기존 동작 그대로 |
| 폴더의 서버를 한 번에 열기 | **없어진다**(기능 손실) | §3 에 적음. 설명서·시나리오 문서에서 지웠다 |
| 저장 데이터 | 형식 변화 없음. 전에 만든 큰 탭은 그대로 복원된다 | §3.2 |
| 문구 | 7개 키를 10개 언어에서 지웠다. 다른 곳에서 쓰지 않는 것을 확인(`grep`)했다 | 문구 검사 통과 |
| 스토어 등록 정보·스크린샷 | [docs/store/listing.md](../store/listing.md) 에는 바둑판 언급이 없다. 스크린샷 `01-split-broadcast` 가 이 기능으로 찍은 화면인지는 확인하지 않았다 | 다음 스토어 제출 때 확인 |

## J. 남은 정리 거리 (이번에 하지 않음)

| 항목 | 비고 |
|---|---|
| 폴더의 서버를 한 번에 열기(한도에 맞게) | 서버마다 탭 하나로 여는 "폴더 서버 모두 탭으로 열기". 필요하면 따로 만든다 |
| 창 수 한도의 직접 테스트 | `canSplitMore` 를 화면 모듈과 떨어진 곳(`lib/layout.ts`)으로 옮기면 단위 테스트할 수 있다 |
| 한도를 넘은 탭을 복원할 때 정리 | 저장된 배치의 창 수가 한도를 넘으면 탭을 나눠 복원 |
| 한도의 이름 | 설정 이름은 "상하"인데 실제로는 방향과 상관없이 탭의 창 수를 센다(§1.1) |

## K. 체크리스트

- [x] 1. 확인: 화면 나누기 2개, 탭의 창 수 한도 기본 3, 한 화면 6개(§1.1). 바둑판이 한도를 보지 않음(§1.2, §2)
- [x] 2. 세 들어가는 곳 삭제(TabBar·Sidebar·QuickPalette)
- [x] 3. 뒤의 코드 삭제(`gridTabs`, `addGridTab`, `openFolderGrid`, `gridLayout`)와 테스트 1개
- [x] 4. 문구 7개 삭제(10개 언어)
- [x] 5. 명세 6곳(ui/panes.md §6, ui/shell.md §2.2·§2.3, ui/sidebar.md §8, ui/dialogs.md §2, 수용 S1·P14·P15·P15a), 설명서 ko·en, 시나리오 ko·en, decisions D116
- [x] 6. 되돌아오지 않게 막는 테스트 4개, `tsc`·`vitest`(278개) 통과, 빌드
- [ ] 7. 수동 검증 1~6
