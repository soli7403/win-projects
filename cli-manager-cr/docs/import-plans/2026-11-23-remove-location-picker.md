# "열 위치 선택" 메뉴와 대화상자 제거 — 창 수 한도를 보지 않던 두 번째 경로 — 원인·수정·코드·검증

**작성**: 2026-11-23 · **상태**: 코드 수정·자동 테스트·빌드 완료(D117). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(요청 즉시 수정해 등록하지 않음)
**관련**: 바로 앞 문서 [remove-grid-layout](./2026-11-23-remove-grid-layout.md)(D116, 같은 문제의 첫 번째 경로 — 한도의 설명은 그 문서 §1.1), [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §3, [spec/ui/sidebar.md](../../spec/ui/sidebar.md) §7, [spec/ui/panes.md](../../spec/ui/panes.md) §2.1

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

사이드바에서 서버를 우클릭하면 나오던 **"열 위치 선택…"** 항목과 그것이 띄우던 대화상자(열린 탭들의 창 배치를 작게 그려 놓고, 아무 창이나 상·하·좌·우로 나눠 여는 화면)를 **없앴다.** 바둑판 배열(D116)과 마찬가지로 탭의 창 수 한도를 보지 않았다.

서버를 여는 위치는 우클릭 메뉴에 남은 세 가지로 고른다. 셋 다 한도를 지킨다.

| 남은 항목 | 동작 | 한도에 이르면 |
|---|---|---|
| 새 탭으로 열기 (Enter) | 새 탭 | — |
| 오른쪽 화면에 열기 | 오른쪽 그룹(화면 나누기)의 새 탭 | 그룹은 2개뿐이라 넘칠 일이 없다 |
| 아래 창으로 열기 | 지금 탭의 포커스 창 아래에 나눠 열기 | 나누지 않고 새 탭에서 열고 안내 토스트 |

## 1. 사용자 요청

2026-11-23, 바둑판 배열을 빼 달라는 요청에 이어: "열 위치 선택 메뉴도 없애줘."

## 2. 원인 분석 (Cause)

지운 파일 `frontend/src/screens/LocationPicker.tsx` 의 핵심(전체는 §B):

```tsx
const place = async (paneId: string | null, dir: Dir, before: boolean, empty: boolean) => {
  close();
  const info = await openSessionFor(loc.hostId);      // 세션을 먼저 연다
  if (!info) return;
  const ws = useWorkspace.getState();
  if (empty && paneId) ws.fillPane(paneId, info);
  else if (paneId) ws.openInPane(paneId, dir, info, before);   // ← 한도 검사 없이 곧바로 나눈다
  else ws.openInNewTab(info);
};
```

| # | 문제 | 왜 | 보이는 모습 |
|---|---|---|---|
| 1 | **탭의 창 수 한도를 넘길 수 있다** | 다른 "나눠서 열기" 경로는 모두 [actions.ts](../../frontend/src/state/actions.ts) `openHostAt` 을 거치고 거기서 `canSplitMore` 를 본다. 이 화면만 작업 공간의 `openInPane` 을 직접 불렀다 | 창이 이미 3개인 탭의 창에서 화살표를 누르면 4개, 5개… 로 늘어난다. 그 뒤 그 탭은 [아래로 나누기]가 비활성이 되고 Ctrl+Shift+D/E 가 "최대 3개까지입니다"만 낸다 |
| 2 | **네 방향 모두 나눈다** | 화살표가 위·아래·왼쪽·오른쪽 네 개 | 좌우로 쪼갠 좁은 창이 생긴다. 앱의 다른 곳은 좌우를 화면 나누기(그룹 2개)에 맡기고 창은 상하로 쌓게 한다 |
| 3 | **보이지 않는 탭에도 넣는다** | 열린 모든 탭을 그려 놓고 고르게 한다 | 지금 보고 있지 않은 탭이 한도를 넘은 채로 남는다. 나중에 그 탭을 열었을 때 이유를 알기 어렵다 |

바둑판 배열과 뿌리가 같다: 이 화면도 처음 명세(ui/dialogs.md §3)에 있던 것이고, 창 수 한도와 화면 나누기가 나중에 들어오면서 이 경로에는 검사가 들어가지 않았다.

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| **A. 없앤다** | 메뉴 항목, 대화상자, 상태, 스타일, 문구를 지운다 | **고름.** 사용자 요청 그대로. 남은 세 항목으로 "새 탭 / 오른쪽 화면 / 아래 창"을 모두 고를 수 있다 |
| B. 한도를 지키게 고친다 | `place` 가 `openHostAt` 을 거치게 하고, 한도에 이른 탭의 화살표를 비활성으로 | 기능은 살지만 요청과 다르다. 좌우 화살표 문제(2번)는 따로 남는다 |

잃는 것: **지금 포커스가 없는 창**이나 **다른 탭**을 골라 그 옆에 여는 방법이 메뉴에서 없어진다. 같은 일은 그 창을 먼저 눌러 포커스를 준 뒤 "아래 창으로 열기"를 하거나, 서버를 그 창의 가장자리로 끌어 놓으면 된다(끌어 놓기는 한도를 지킨다).

### 3.1 지운 것

| 파일 | 지운 것 |
|---|---|
| `frontend/src/screens/LocationPicker.tsx` | 파일 전체 |
| `frontend/src/screens/Sidebar.tsx` | 서버 메뉴 항목 `sb.chooseLocation` |
| `frontend/src/App.tsx` | 가져오기와 `<LocationPicker />` |
| `frontend/src/state/ui.ts` | `locationPicker`, `openLocation`, `closeLocation` |
| `frontend/src/styles/app.css` | `.loc-card`, `.loc-mini`, `.loc-pane`, `.loc-arrows`(2줄) |
| `locales/*.json` (10개) | `sb.chooseLocation`, `loc.title`, `loc.noTabs`, `loc.openHereTitle`, `loc.here`, `loc.splitOpen`, 그리고 이 화면의 화살표만 쓰던 `split.up`·`split.left`·`split.right` |

남긴 것: `loc.newTab`("새 탭으로 열기" — 서버 메뉴의 첫 항목이 쓴다), `split.down`(창 머리글의 [아래로 나누기]가 쓴다).

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/App.tsx` | 코드 | 수정 |
| `frontend/src/screens/LocationPicker.tsx` | 코드 | 삭제 |
| `frontend/src/screens/Sidebar.tsx` | 코드 | 수정 |
| `frontend/src/state/ui.ts` | 코드 | 수정 |
| `frontend/src/styles/app.css` | 코드 | 수정 |
| `frontend/src/state/gridRemoved.test.ts` | 테스트 | 수정 |
| `spec/ui/dialogs.md` | 명세 | 수정 |
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

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/screens/Sidebar.tsx`, `frontend/src/state/gridRemoved.test.ts`, `spec/ui/dialogs.md`, `spec/ui/sidebar.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/App.tsx` — 수정

```diff
--- a/frontend/src/App.tsx
+++ b/frontend/src/App.tsx
@@ -17,7 +17,6 @@
 import { FolderEditor } from './screens/FolderEditor';
 import { QuickPalette } from './screens/QuickPalette';
 import { RequestDialogs } from './screens/RequestDialogs';
-import { LocationPicker } from './screens/LocationPicker';
 import { startLayoutPersistence } from "./state/layoutSave";
 import { startImportSuggestion } from "./state/importSuggest";
 import { subscribeUpdateEvents, useUpdate } from './state/update';
@@ -117,7 +116,6 @@
       {!locked && editor?.kind === 'host' && <HostEditor key={`h-${editor.hostId ?? 'new'}`} hostId={editor.hostId} groupId={editor.groupId} />}
       {!locked && editor?.kind === 'folder' && <FolderEditor key={`f-${editor.folderId ?? 'new'}`} folderId={editor.folderId} parentId={editor.parentId} />}
       {!locked && <QuickPalette />}
-      {!locked && <LocationPicker />}
       {!locked && <RequestDialogs />}
       {!locked && <ErrorBoundary variant="dialog" label={t('settings.title')} onDismiss={() => useUi.getState().closeSettings()}><Settings /></ErrorBoundary>}
       {!locked && <ErrorBoundary variant="dialog" label={t('import.title')} onDismiss={() => useUi.getState().closeImportDialog()}><ImportDialog /></ErrorBoundary>}
```

### `frontend/src/screens/LocationPicker.tsx` — 삭제 (파일을 지운다. 아래는 지우기 전의 전체 내용 — 되돌릴 때 쓴다)

```tsx
// 위치 골라서 연결 대화상자. 근거: spec/ui/dialogs.md §3.
// 열린 탭마다 분할 구조를 축소해 그리고, 칸의 방향 화살표(분할) 또는 "여기에"(빈 칸)로 연다.
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp } from 'lucide-react';
import type { LayoutNode } from '../lib/layout';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace, openSessionFor, tabTitle, type Tab } from '../state/workspace';
import { openHost } from '../state/actions';
import { Modal } from '../ui/Modal';
import type { Dir } from '../lib/layout';
import { relabel, t } from '../i18n';

export function LocationPicker() {
  const loc = useUi((s) => s.locationPicker);
  const close = useUi((s) => s.closeLocation);
  const tabs = useWorkspace((s) => s.tabs);
  const sessions = useWorkspace((s) => s.sessions);
  const host = useApp((s) => s.store?.hosts.find((h) => h.id === loc?.hostId));
  if (!loc || !host) return null;

  const place = async (paneId: string | null, dir: Dir, before: boolean, empty: boolean) => {
    close();
    const info = await openSessionFor(loc.hostId);
    if (!info) return;
    const ws = useWorkspace.getState();
    if (empty && paneId) ws.fillPane(paneId, info);
    else if (paneId) ws.openInPane(paneId, dir, info, before);
    else ws.openInNewTab(info);
  };

  return (
    <Modal title={t('loc.title', { name: host.alias })} size="lg" onClose={close}
      foot={<button className="btn primary" onClick={() => { close(); void openHost(loc.hostId); }}>{t('loc.newTab')}</button>}>
      {tabs.length === 0 && <div className="faint">{t('loc.noTabs')}</div>}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 14 }}>
        {tabs.map((t) => (
          <div key={t.id} className="loc-card">
            <div className="faint small" style={{ marginBottom: 6 }}>{tabTitle(t, sessions)}</div>
            <div className="loc-mini">
              <MiniNode tab={t} node={t.root} onPlace={place} sessions={sessions} />
            </div>
          </div>
        ))}
      </div>
    </Modal>
  );
}

type Sessions = ReturnType<typeof useWorkspace.getState>['sessions'];

function MiniNode({ tab, node, onPlace, sessions }: {
  tab: Tab;
  node: LayoutNode;
  onPlace: (paneId: string | null, dir: Dir, before: boolean, empty: boolean) => void;
  sessions: Sessions;
}) {
  if (node.type === 'pane') {
    const title = node.sessionId ? relabel(sessions[node.sessionId]?.info.title) ?? t('tab.session') : null;
    if (!title) {
      return (
        <div className="loc-pane">
          <button className="btn small-btn" title={t('loc.openHereTitle')} onClick={() => onPlace(node.id, 'row', false, true)}>{t('loc.here')}</button>
        </div>
      );
    }
    return (
      <div className="loc-pane">
        <span className="small">{title}</span>
        <div className="loc-arrows">
          <button className="icon-btn small" title={t('loc.splitOpen')} aria-label={t('split.up')} onClick={() => onPlace(node.id, 'col', true, false)}><ArrowUp size={12} /></button>
          <button className="icon-btn small" title={t('loc.splitOpen')} aria-label={t('split.left')} onClick={() => onPlace(node.id, 'row', true, false)}><ArrowLeft size={12} /></button>
          <button className="icon-btn small" title={t('loc.splitOpen')} aria-label={t('split.right')} onClick={() => onPlace(node.id, 'row', false, false)}><ArrowRight size={12} /></button>
          <button className="icon-btn small" title={t('loc.splitOpen')} aria-label={t('split.down')} onClick={() => onPlace(node.id, 'col', false, false)}><ArrowDown size={12} /></button>
        </div>
      </div>
    );
  }
  const horizontal = node.dir === 'row';
  return (
    <div style={{ display: 'flex', flexDirection: horizontal ? 'row' : 'column', gap: 3, flex: 1, minHeight: 0, minWidth: 0 }}>
      {node.children.map((c, i) => (
        <div key={c.id} style={{ flex: `0 0 calc(${node.sizes[i]}% - 2px)`, display: 'flex', minWidth: 0, minHeight: 0 }}>
          <MiniNode tab={tab} node={c} onPlace={onPlace} sessions={sessions} />
        </div>
      ))}
    </div>
  );
}
```

### `frontend/src/screens/Sidebar.tsx` — 수정

```diff
--- a/frontend/src/screens/Sidebar.tsx
+++ b/frontend/src/screens/Sidebar.tsx
@@ -281,7 +281,6 @@
     { label: tr('loc.newTab'), shortcut: 'Enter', onSelect: () => void openHost(h.id) },
     { label: tr('sb.openRight'), onSelect: () => void openHost(h.id, { kind: 'rightGroup' }) },
     { label: tr('sb.openBelow'), onSelect: () => void openHost(h.id, { kind: 'split', dir: 'col' }) },
-    { label: tr('sb.chooseLocation'), onSelect: () => useUi.getState().openLocation(h.id) },
     { separator: true },
     { label: tr('common.edit'), shortcut: 'F2', onSelect: () => openEditor({ kind: 'host', hostId: h.id, groupId: null }) },
     { label: tr('sb.duplicate'), onSelect: () => void duplicateHost(h) },
```

### `frontend/src/state/ui.ts` — 수정

```diff
--- a/frontend/src/state/ui.ts
+++ b/frontend/src/state/ui.ts
@@ -74,9 +74,6 @@
   openPreview: (sessionId: string, path: string) => void;
   closePreview: () => void;
   /** 위치 선택 대화상자: 서버 id. null 이면 닫힘. */
-  locationPicker: { hostId: string } | null;
-  openLocation: (hostId: string) => void;
-  closeLocation: () => void;
   /** 빠른 접속 팔레트 상태. targetPane 이 있으면 그 빈 칸에서 연다. */
   quick: { open: boolean; targetPane: string | null };
   openQuick: (targetPane?: string | null) => void;
@@ -156,9 +153,6 @@
   preview: null,
   openPreview: (sessionId, path) => set({ preview: { sessionId, path } }),
   closePreview: () => set({ preview: null }),
-  locationPicker: null,
-  openLocation: (hostId) => set({ locationPicker: { hostId } }),
-  closeLocation: () => set({ locationPicker: null }),
   quick: { open: false, targetPane: null },
   openQuick: (targetPane = null) => set({ quick: { open: true, targetPane } }),
   closeQuick: () => set((s) => ({ quick: { ...s.quick, open: false } })),
```

### `frontend/src/styles/app.css` — 수정

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -357,9 +357,4 @@
   padding: 5px; border-radius: 8px; background: var(--panel); border: 1px solid var(--border); box-shadow: var(--shadow);
 }
 .find-bar .input { width: 220px; height: 28px; }
-.loc-card { border: 1px solid var(--border); border-radius: 10px; padding: 8px; background: var(--panel); }
-.loc-mini { height: 170px; display: flex; border-radius: 6px; background: var(--panel-2); padding: 3px; }
-.loc-pane { flex: 1; min-width: 0; min-height: 0; border-radius: 6px; background: var(--bg); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; overflow: hidden; }
-.loc-arrows { display: flex; gap: 2px; }
-.loc-arrows .icon-btn:hover:not(:disabled) { background: var(--accent); color: #fff; }
-
+
```


## C. 테스트 코드

### `frontend/src/state/gridRemoved.test.ts` — 수정

```diff
--- a/frontend/src/state/gridRemoved.test.ts
+++ b/frontend/src/state/gridRemoved.test.ts
@@ -30,6 +30,19 @@
     expect(offenders).toEqual([]);
   });
 
+  it('열 위치 선택 대화상자도 없다 (D117): 한도를 보지 않고 아무 창이나 네 방향으로 나누던 화면', () => {
+    const offenders = Object.entries(SOURCES)
+      .filter(([path]) => !/\.test\.tsx?$/.test(path))
+      .filter(([path, code]) => /LocationPicker/.test(path) || /LocationPicker|openLocation|locationPicker|sb\.chooseLocation/.test(code))
+      .map(([path]) => path);
+    expect(offenders).toEqual([]);
+    for (const [lang, dict] of Object.entries(LOCALES)) {
+      for (const key of ['sb.chooseLocation', 'loc.title', 'loc.here', 'loc.splitOpen']) {
+        expect(dict && key in dict, `${lang}: ${key}`).toBe(false);
+      }
+    }
+  });
+
   it('화면 나누기(탭 그룹)는 왼쪽·오른쪽 두 개뿐이다', () => {
     expect(workspace.useWorkspace.getState().activeByGroup).toHaveLength(2);
   });
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`loc.title`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`loc.noTabs`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`loc.openHereTitle`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`loc.here`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`loc.splitOpen`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`split.up`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`split.left`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`split.right`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.

**`sb.chooseLocation`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.


## F. 명세

### `spec/ui/dialogs.md` — 수정

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -67,6 +67,8 @@
 
 ## 3. 위치 골라서 연결 (위치 선택)
 
+> **2026-11-23 삭제 (D117)**: 이 대화상자와 서버 우클릭 메뉴의 "열 위치 선택…" 항목을 **없앴다.** 이 화면은 열린 탭의 아무 창이나 상·하·좌·우로 나눠 열 수 있었는데, 탭의 창 수 한도(ui/panes.md §2.1 6번, 설정 `maxVerticalSplits`)를 보지 않았다 — 바둑판 배열(D116)과 같은 문제다. 아래 설명은 기록으로만 남긴다. **구현하지 않는다.** 서버를 여는 위치는 우클릭 메뉴의 "새 탭으로 열기"·"오른쪽 화면에 열기"·"아래 창으로 열기"로 고른다(셋 다 한도를 지킨다).
+
 서버 우클릭 메뉴의 "열 위치 선택…"에서 열린다. 크게 보기. 제목 "위치 골라서 연결 — {서버 이름}".
 - 위쪽 버튼 [새 탭으로 열기]
 - 열린 탭이 없으면 "열린 탭이 없습니다. 새 탭으로 여세요."
```

### `spec/ui/sidebar.md` — 수정

```diff
--- a/spec/ui/sidebar.md
+++ b/spec/ui/sidebar.md
@@ -57,7 +57,6 @@
 | 새 탭에서 연결 | Enter | |
 | 오른쪽으로 분할해 연결 | | 오른쪽 탭 그룹에 새 탭으로 (D56) |
 | 아래로 분할해 연결 | | |
-| 위치 골라서 연결… | | 위치 선택 대화상자(ui/dialogs.md §3) |
 | ―구분선― | | |
 | 편집 | F2 | 서버 편집 |
 | 복제 | | 같은 설정으로 새 id, 이름에 " (복사본)" 덧붙임, 최근 사용 기록은 지움 |
```



---

## G. 검증 (2026-11-23 실행)

[frontend/src/state/gridRemoved.test.ts](../../frontend/src/state/gridRemoved.test.ts) 에 테스트 하나를 더했다.

| 테스트 | 확인하는 것 |
|---|---|
| 열 위치 선택 대화상자도 없다 (D117) | 테스트가 아닌 소스 가운데 이름에 `LocationPicker` 가 든 파일이 없고, 어느 소스도 `LocationPicker`·`openLocation`·`locationPicker`·`sb.chooseLocation` 을 쓰지 않는다 / 10개 언어 모두에 `sb.chooseLocation`·`loc.title`·`loc.here`·`loc.splitOpen` 이 없다 |

```bash
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 43개 파일 279개 통과
go test ./internal/i18n/               # 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

자동 테스트가 확인하지 못하는 것: 메뉴가 실제 화면에서 어떻게 보이는지.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 사이드바의 서버를 우클릭 | "열 위치 선택…"이 없다. "아래 창으로 열기" 다음이 바로 구분선과 "수정" |
| 2 | "새 탭으로 열기", "오른쪽 화면에 열기", "아래 창으로 열기"를 차례로 | 각각 새 탭 / 오른쪽 그룹 / 포커스 창 아래에 열린다 |
| 3 | 창이 3개인 탭에서 "아래 창으로 열기" | 나누지 않고 새 탭에서 열리고 안내 토스트가 나온다 |
| 4 | 서버를 끌어 창의 아래쪽 가장자리에 놓는다 | 그 창 아래에 열린다(한도 안일 때) |
| 5 | 즐겨찾기·최근 목록의 서버를 우클릭 | 같은 메뉴(항목이 빠진 채) |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 서버 열기의 다른 경로 | 건드리지 않았다 | `tsc`, 기존 테스트 |
| 작업 공간의 `openInPane`·`fillPane` | 그대로 있다(`openHostAt` 이 쓴다) | — |
| 대화상자가 떠 있는지 보는 코드 | 이 화면은 `dialogs` 목록에 들어가지 않고 따로 상태를 가졌다. 그 상태를 읽는 다른 곳이 없는 것을 확인했다 | `grep` |
| 문구 | 9개 키를 10개 언어에서 지웠다. 쓰는 곳이 없는 것을 확인했다 | 문구 검사 통과 |
| 스토어 스크린샷 | 이 화면을 찍은 스크린샷은 없다(목록: docs/store/README.md §2 5번) | — |

## J. 체크리스트

- [x] 메뉴 항목·대화상자 파일·상태·스타일 삭제
- [x] 문구 9개 삭제(10개 언어)
- [x] 명세 ui/dialogs.md §3(삭제 표시)·ui/sidebar.md §7, 시나리오 ko·en, decisions D117
- [x] 막는 테스트 1개 추가, `tsc`·`vitest`(279개) 통과, 빌드
- [ ] 수동 검증 1~5
