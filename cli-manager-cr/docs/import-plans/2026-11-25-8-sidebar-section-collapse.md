# 서버 목록의 "즐겨찾기"·"최근 사용" 구역 접기 (`favoritesCollapsed`·`recentCollapsed`) + "서버" 구역 위 구분선 — 배경·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D131, 구분선은 D132). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 명세 [spec/ui/sidebar.md](../../spec/ui/sidebar.md) §3, [spec/data/store.md](../../spec/data/store.md) §4
**적용 순서**: 2026-11-25 의 8번째. **3번**(단축키 설정 — 계약 고정 파일·`store.go`·`types.ts`·`defaults.ts` 의 같은 자리 근처를 고친다)과 **7번**(왼쪽 열 너비 — `Sidebar.tsx`·`app.css` 를 고친다)을 적용한 뒤에 적용한다.

> 이 문서 하나로 **무엇을 바꿨는지 → 왜 그렇게 정했는지 → 코드 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

서버 목록 위쪽의 **"즐겨찾기"와 "최근 사용" 구역을 제목을 눌러 접을 수 있다.** 접어 둔 상태는 설정에 저장돼 앱을 다시 열어도 그대로다. 그리고 그 아래 **"서버" 구역 제목 위에 가로 구분선**을 넣어 위의 두 구역과 나눴다.

```
펴짐(기본)                       접힘
▾ ☆ 즐겨찾기                    ▸ ☆ 즐겨찾기
    db-main                      ▸ ◷ 최근 사용
▾ ◷ 최근 사용                    ──────────────────   ← 구분선
    kai-test-vm-02               서버 · 39
    DC1_MSD01                      ▸ mds-…
    …(최대 5개)
──────────────────   ← 구분선
서버 · 39
  ▸ mds-…
```

| | 바꾸기 전 | 바꾼 뒤 |
|---|---|---|
| 구역 제목 | 글자만(누를 수 없음) | 줄 전체가 버튼. 앞에 화살표 ▾/▸ |
| 접기 | 없음 | 구역마다 따로 접고 편다 |
| 저장 | — | 설정 `favoritesCollapsed`·`recentCollapsed`(기본 false) |
| "서버" 구역과의 경계 | 여백뿐 | "서버 · n" 제목 위에 가로선(위에 구역이 있을 때만) |

## 1. 사용자 요청

2026-11-25: "'최근 사용' 서버 리스트 항목을 폴더처럼 닫게 해달라는 요청이 있었다. 가능한가? 닫힌 상태는 저장되어야 한다." → (가능하다고 답하고 두 가지를 물음: 저장 위치, "즐겨찾기"도 포함할지. 권장안은 설정에 저장 + 즐겨찾기 포함) → "수정하고 마찬가지로 md 파일 만들어줘."

같은 날 이어서: "서버 위에 라인을 하나 만들어서 즐겨찾기 & 최근 사용과 약간 구분되게 해줘." → (구분선을 넣음, D132) → "2026-11-25-8-sidebar-section-collapse.md 파일에 함께 추가해줘." 그래서 구분선도 이 문서에 함께 싣는다.

배경: "최근 사용"은 서버가 7개 이상이고 접속 기록이 있으면 서버 목록 맨 위에 최대 5줄을 차지한다. 폴더 트리를 주로 쓰는 사람에게는 같은 서버가 위에 한 번 더 나와 자리를 차지한다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 접는 방법 | 구역 **제목 줄을 누른다**(줄 전체가 버튼) | 폴더와 같은 손동작. 작은 화살표만 누르게 하면 맞히기 어렵다 |
| 표시 | 제목 맨 앞에 폴더 줄과 같은 화살표: 펴짐 ▾, 접힘 ▸. 그 뒤에 원래 아이콘(별·시계)과 이름 | "폴더처럼"이라는 요청. 접을 수 있다는 것이 보인다 |
| 접혔을 때 | 제목만 남고 그 구역의 서버 줄이 사라진다 | — |
| 대상 | "최근 사용"과 **"즐겨찾기"** | 바로 위에 같은 모양의 구역이다. 한쪽만 접히면 어색하다(사용자에게 묻고 권장안으로 진행) |
| "서버" 구역 | 접지 않는다 | 접으면 서버 목록이 통째로 사라진다. 요청에도 없다 |
| **저장** | 설정(`store.json`)의 `favoritesCollapsed`·`recentCollapsed`, 기본 **false** | 폴더의 접힘(`collapsed`)도 저장 데이터에 있다. 앱을 다시 열 때 확실히 남는다. 기존 사용자는 달라지지 않는다(펴진 채) |
| 이 PC 에만 기억하는 안 | 고르지 않았다 | 명령 모음 높이처럼 웹뷰 저장소에 두는 방법인데, 웹뷰 데이터가 지워지면 사라진다. "저장되어야 한다"는 요구에는 설정이 맞다 |
| 동기화 | git 동기화를 켜면 다른 PC 에도 같은 접힘이 적용된다 | 설정이 통째로 동기화되는 것의 결과다. 다른 화면 설정(너비 등)도 그렇다 |
| 저장하는 때 | 누르는 즉시 | 다른 설정과 같다 |
| 검색 중 | 전과 같다: 두 구역이 통째로 숨는다. 검색을 지우면 접힘 상태 그대로 돌아온다 | 검색은 접힘 값을 건드리지 않는다 |
| 값이 없을 때 | 펴진 것으로 본다 | 옛 저장 파일 |
| 접힌 구역의 개수 표시 | 넣지 않았다 | 요청에 없다. 최근 사용은 늘 5개 이하다 |
| 문구 | 더하지 않았다 | 버튼의 이름이 곧 구역 이름("최근 사용")이고 펴짐/접힘은 `aria-expanded` 로 알린다 |
| 키보드 | 버튼이라 Tab 으로 포커스가 가고 Enter·Space 로 접고 편다 | 따로 만든 것이 없다(버튼의 기본 동작) |

구분선(D132):

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 자리 | "서버 · n" 제목 **위** | 요청 그대로. 위의 두 구역(즐겨찾기·최근 사용)과 서버 목록 사이 |
| 모양 | 테두리색(`--border`) 1px 가로선, 목록 안쪽 너비. 선 위 6px, 선 아래 12px 을 띄운다 | 다른 경계선(열 테두리, 폴더 사이)과 같은 색. "약간 구분"이라 굵게 하지 않았다 |
| 언제 | 위에 즐겨찾기나 최근 사용 구역이 **보이는 동안만**. 그 구역이 접혀 있어도 제목은 남으므로 선은 있다 | 구역을 접으면 제목 세 줄이 붙어 보여 어디부터 서버 목록인지 흐려진다 — 그때 선이 가장 쓸모 있다 |
| 위에 구역이 없을 때 | 선이 없다(서버 6개 이하이고 즐겨찾기 없음, 검색 중) | 늘 그으면 검색 칸 바로 아래에 뜻 없는 선이 생긴다 |
| 만드는 방법 | "서버" 제목 줄에 클래스 `divided` 를 붙이고 스타일에서 위 테두리를 준다 | 요소를 더하지 않는다. 구역이 사라지면 선도 함께 사라진다 |

흐름:

```
"최근 사용" 제목 누름
  └ saveSettings({ recentCollapsed: !지금 값 })  → store.patch → 저장 데이터를 다시 읽음
       └ 서버 목록이 다시 그려짐: recentCollapsed 면 제목만, 아니면 제목 + 서버 줄
앱을 다시 열면 store.get 의 settings.recentCollapsed 로 그린다
```

### 2.1 영향 범위

- **저장 형식**: `settings` 에 키 둘이 늘었다. 옛 파일에는 없으므로 기본값(false)으로 채워진다.
- **IPC**: 채널은 그대로. `store.get` 응답의 `settings` 에 키가 둘 늘어 계약 고정 파일 2개를 고쳤다.
- **겉모습**: 두 구역의 제목 앞에 화살표가 생겨 제목 글자가 약 16px 오른쪽으로 간다. 위에 구역이 있을 때 "서버" 제목이 선과 여백만큼(약 9px) 아래로 내려간다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/screens/Sidebar.tsx` | 코드 | 수정 |
| `frontend/src/styles/app.css` | 스타일 | 수정 |
| `frontend/src/ipc/types.ts`, `frontend/src/lib/defaults.ts` | 코드 | 수정(설정 2개) |
| `internal/store/store.go` | 코드(Go) | 수정(기본값 2줄) |
| `frontend/src/ipc/contract/store.get.json`, `store.get.afterNullPatch.json` | 계약 고정 파일 | 수정 |
| `frontend/src/screens/Sidebar.sections.test.tsx` | 테스트 | **새 파일**(9개) |
| `spec/ui/sidebar.md`, `spec/data/store.md`, `spec/acceptance/02-shell-sidebar.md` | 명세 | 수정 |
| `docs/scenarios.md`, `docs/scenarios.en.md` | 문서 | 수정(3-8) |
| `CHANGELOG.md`, `docs/decisions.md`(D131·D132), `CLAUDE.md` | 문서 | 수정 |

아래의 `diff` 는 **7번 문서까지 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `frontend/src/screens/Sidebar.tsx` — 수정

```diff
--- a/frontend/src/screens/Sidebar.tsx
+++ b/frontend/src/screens/Sidebar.tsx
@@ -21,6 +21,7 @@
 import { useWorkspace } from '../state/workspace';
 import { findPane, findPaneBySession, sessionsIn, type LayoutNode } from '../lib/layout';
 // 이 파일은 t 를 탭 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
+import { saveSettings } from '../state/settings';
 import { t as tr } from '../i18n';
 
 /** 이 서버의 열린 칸 목록(모든 탭, 탭 순서대로). */
@@ -471,6 +472,21 @@
     .sort((a, b) => (b.lastUsedAt ?? 0) - (a.lastUsedAt ?? 0))
     .slice(0, 5);
   const showRecent = !searching && hosts.length > 6 && recent.length > 0;
+  // "즐겨찾기"·"최근 사용" 구역은 제목을 눌러 접는다. 접힘은 설정에 저장한다 (sidebar.md §3, D131).
+  const favoritesCollapsed = !!store?.settings.favoritesCollapsed;
+  const recentCollapsed = !!store?.settings.recentCollapsed;
+  /** 접을 수 있는 구역의 제목 줄. 누르면 접힘을 뒤집어 저장한다. */
+  const sectionHead = (key: 'favoritesCollapsed' | 'recentCollapsed', collapsed: boolean, icon: ReactNode, label: string) => (
+    <button
+      type="button"
+      className="side-section side-section-toggle"
+      aria-expanded={!collapsed}
+      onClick={() => void saveSettings({ [key]: !collapsed })}
+    >
+      {collapsed ? <ChevronRight size={12} /> : <ChevronDown size={12} />}
+      {icon} {label}
+    </button>
+  );
   const noMatch = searching && matched.length === 0 && hosts.length > 0;
 
   return (
@@ -521,17 +537,18 @@
       >
         {!searching && favorites.length > 0 && (
           <>
-            <div className="side-section"><Star size={12} /> {tr('sb.favorites')}</div>
-            {favorites.map((h) => hostRow(h, 0, { selectable: false }))}
+            {sectionHead('favoritesCollapsed', favoritesCollapsed, <Star size={12} />, tr('sb.favorites'))}
+            {!favoritesCollapsed && favorites.map((h) => hostRow(h, 0, { selectable: false }))}
           </>
         )}
         {showRecent && (
           <>
-            <div className="side-section"><Clock size={12} /> {tr('sb.recent')}</div>
-            {recent.map((h) => hostRow(h, 0, { selectable: false }))}
+            {sectionHead('recentCollapsed', recentCollapsed, <Clock size={12} />, tr('sb.recent'))}
+            {!recentCollapsed && recent.map((h) => hostRow(h, 0, { selectable: false }))}
           </>
         )}
-        <div className="side-section">{tr('sb.servers')}{hosts.length > 0 ? ` · ${hosts.length}` : ''}</div>
+        {/* 위에 즐겨찾기·최근 사용 구역이 있으면 "서버" 제목 위에 선을 그어 구분한다(접혀 있어도). D132 */}
+        <div className={`side-section${(!searching && favorites.length > 0) || showRecent ? ' divided' : ''}`}>{tr('sb.servers')}{hosts.length > 0 ? ` · ${hosts.length}` : ''}</div>
         {rootGroups.map((g) => folderRow(g, 0))}
         {looseHosts.map((h) => hostRow(h, 0))}
         {hosts.length === 0 && (
```

### `frontend/src/styles/app.css` — 수정

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -269,6 +269,12 @@
 .side-btns .btn { flex: 1 1 auto; justify-content: center; padding: 0 6px; white-space: nowrap; }
 .side-tree { flex: 1; overflow: auto; padding: 4px 6px; }
 .side-section { font-size: 11px; font-weight: 700; color: var(--fg-3); letter-spacing: 0.4px; padding: 10px 6px 4px; display: flex; align-items: center; gap: 4px; }
+/* 접을 수 있는 구역 제목(즐겨찾기·최근 사용): 글자 모양은 다른 구역 제목과 같고, 줄 전체가 버튼이다 (D131). */
+/* "서버" 구역 제목 위의 구분선: 위에 즐겨찾기·최근 사용 구역이 있을 때만 (D132). 선 위아래에 6px 씩 띄운다. */
+.side-section.divided { margin-top: 6px; padding-top: 12px; border-top: 1px solid var(--border); }
+.side-section-toggle { width: 100%; border: none; background: transparent; font-family: inherit; text-align: left; cursor: pointer; border-radius: 6px; }
+.side-section-toggle:hover { color: var(--fg-2); }
+.side-section-toggle:focus-visible { outline: 1px solid var(--accent); outline-offset: -1px; }
 .side-row {
   position: relative; height: 30px; display: flex; align-items: center; gap: 6px; padding-right: 8px;
   border-radius: 6px; margin-bottom: 6px; cursor: default; white-space: nowrap; color: var(--fg);
```

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -141,6 +141,10 @@
   syncAuto: boolean;
   osc52Copy: boolean;
   sidebarWidth: number;
+  /** 사이드바의 "즐겨찾기" 구역을 접어 두었는지 (ui/sidebar.md §3, D131). */
+  favoritesCollapsed: boolean;
+  /** 사이드바의 "최근 사용" 구역을 접어 두었는지 (ui/sidebar.md §3, D131). */
+  recentCollapsed: boolean;
   rightPanelWidth: number;
   /** 사용 용도(D78): '' 아직 안 고름 / personal(기한 없음) / business(30일 평가 뒤 라이선스). */
   usage: '' | 'personal' | 'business';
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -58,6 +58,8 @@
   syncAuto: true,
   osc52Copy: true,
   sidebarWidth: 260,
+  favoritesCollapsed: false,
+  recentCollapsed: false,
   rightPanelWidth: 340,
   usage: '',
   usageSince: 0,
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -67,6 +67,8 @@
 	"syncAuto":            true,
 	"osc52Copy":           true,
 	"sidebarWidth":        260,
+	"favoritesCollapsed":  false, // 사이드바 "즐겨찾기" 구역 접힘 (sidebar.md §3, D131)
+	"recentCollapsed":     false, // 사이드바 "최근 사용" 구역 접힘
 	"rightPanelWidth":     340,
 	// D78: 사용 용도·업무용 라이선스. usage ''=아직 안 고름 / personal / business(30일 평가), usageSince=업무용 고른 시각(ms).
 	"usage":            "",
```


## C. 테스트 코드

### `frontend/src/screens/Sidebar.sections.test.tsx` — 새 파일

```tsx
// 사이드바의 "즐겨찾기"·"최근 사용" 구역 접기 (D131): 제목을 누르면 접히고, 접힘은 설정에 저장돼 다시 열어도 유지된다.
// 근거: spec/ui/sidebar.md §3, spec/data/store.md §4.
import { beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Host, Settings, Store } from '../ipc/types';
import { Sidebar } from './Sidebar';

/** 서버 8개(6개를 넘어야 "최근 사용"이 보인다). 앞의 3개는 최근 사용 기록이 있고 그중 하나는 즐겨찾기. */
const HOSTS = Array.from({ length: 8 }, (_, i) => ({
  id: `h${i}`,
  groupId: null,
  alias: `srv-${i}`,
  protocol: 'ssh',
  host: `10.0.0.${i + 1}`,
  port: 22,
  username: 'ops',
  authType: 'password',
  tags: [],
  order: i,
  favorite: i === 0,
  lastUsedAt: i < 3 ? 1000 + i : undefined,
})) as unknown as Host[];

/** 가짜 백엔드: store.patch 를 받아 두고 store.get 으로 돌려준다. 보낸 설정 조각을 돌려주는 함수를 준다. */
function setup(settings: Partial<Settings> = {}, hosts: Host[] = HOSTS) {
  let store = { version: 1, groups: [], hosts, snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, ...settings } } as Store;
  const patches: Partial<Settings>[] = [];
  const b: Backend = {
    async invoke(channel: string, args: unknown[] = []): Promise<any> {
      if (channel === 'store.get') return structuredClone(store);
      if (channel === 'store.patch') {
        const patch = args[0] as { settings?: Settings };
        if (patch.settings) patches.push(patch.settings);
        store = { ...store, ...(patch as object) } as Store;
        return undefined;
      }
      return [];
    },
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  useApp.setState({ booted: true, store: structuredClone(store), toasts: [] });
  return { patches, saved: () => store.settings };
}

const head = (name: string) => screen.getByRole('button', { name });
/** 이름이 name 인 서버 줄의 수(구역마다 한 줄씩 나온다). */
const rows = (name: string) => screen.queryAllByText(name).length;

beforeEach(() => {
  useUi.setState({ dialogs: [], menu: null, selectedHostId: null });
});

describe('사이드바 구역 접기 (D131)', () => {
  it('기본은 둘 다 펴져 있다: 즐겨찾기·최근 사용의 서버가 구역에도 보인다', () => {
    expect(DEFAULT_SETTINGS.favoritesCollapsed).toBe(false);
    expect(DEFAULT_SETTINGS.recentCollapsed).toBe(false);
    setup();
    render(<Sidebar />);
    expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('true');
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('true');
    expect(rows('srv-0')).toBe(3); // 즐겨찾기 + 최근 사용 + 서버 목록
    expect(rows('srv-1')).toBe(2); // 최근 사용 + 서버 목록
    expect(rows('srv-5')).toBe(1); // 서버 목록만
  });

  it('"최근 사용" 제목을 누르면 그 구역의 서버가 사라지고 recentCollapsed: true 로 저장된다', async () => {
    const { patches, saved } = setup();
    render(<Sidebar />);
    fireEvent.click(head('최근 사용'));
    await waitFor(() => expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false'));
    expect(rows('srv-1')).toBe(1); // 서버 목록에만 남는다
    expect(rows('srv-0')).toBe(2); // 즐겨찾기 구역은 그대로
    expect(patches.map((p) => p.recentCollapsed)).toEqual([true]);
    expect(saved().recentCollapsed).toBe(true);
    expect(saved().favoritesCollapsed).toBe(false);
    // 서버 목록 구역과 제목은 그대로다.
    expect(screen.getByText(/^서버 · 8$/)).toBeTruthy();
  });

  it('다시 누르면 펴지고 false 로 저장된다', async () => {
    const { patches } = setup({ recentCollapsed: true });
    render(<Sidebar />);
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
    expect(rows('srv-1')).toBe(1);
    fireEvent.click(head('최근 사용'));
    await waitFor(() => expect(rows('srv-1')).toBe(2));
    expect(patches.map((p) => p.recentCollapsed)).toEqual([false]);
  });

  it('"즐겨찾기"도 같은 방식으로 접히고 favoritesCollapsed 로 따로 저장된다', async () => {
    const { patches, saved } = setup();
    render(<Sidebar />);
    fireEvent.click(head('즐겨찾기'));
    await waitFor(() => expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('false'));
    expect(rows('srv-0')).toBe(2); // 최근 사용 + 서버 목록
    expect(patches.map((p) => p.favoritesCollapsed)).toEqual([true]);
    expect(saved().recentCollapsed).toBe(false);
  });

  it('저장된 접힘 상태로 다시 그려도 접혀 있다(앱을 다시 열었을 때)', () => {
    setup({ favoritesCollapsed: true, recentCollapsed: true });
    render(<Sidebar />);
    expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('false');
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
    expect(rows('srv-0')).toBe(1);
    expect(rows('srv-2')).toBe(1);
  });

  it('접혀 있어도 검색하면 구역 자체가 숨고, 검색을 지우면 접힌 채로 돌아온다', () => {
    setup({ recentCollapsed: true });
    render(<Sidebar />);
    const search = screen.getByPlaceholderText('서버 찾기');
    fireEvent.change(search, { target: { value: 'srv-1' } });
    expect(screen.queryByRole('button', { name: '최근 사용' })).toBeNull();
    expect(screen.queryByRole('button', { name: '즐겨찾기' })).toBeNull();
    expect(rows('srv-1')).toBe(1);
    fireEvent.change(search, { target: { value: '' } });
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
  });

  it('구역이 나올 조건이 아니면 제목도 없다(서버 6개 이하, 즐겨찾기 없음)', () => {
    setup({}, HOSTS.slice(1, 6));
    render(<Sidebar />);
    expect(screen.queryByRole('button', { name: '최근 사용' })).toBeNull();
    expect(screen.queryByRole('button', { name: '즐겨찾기' })).toBeNull();
  });

  it('"서버" 제목 위의 구분선: 위에 즐겨찾기·최근 사용 구역이 있을 때만 있다 (D132)', () => {
    const serversTitle = () => screen.getByText(/^서버 · \d+$/);
    setup();
    const first = render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(true);
    // 검색 중에는 위의 구역이 숨으므로 선도 없다.
    fireEvent.change(screen.getByPlaceholderText('서버 찾기'), { target: { value: 'srv' } });
    expect(serversTitle().classList.contains('divided')).toBe(false);
    first.unmount();
    // 위의 구역을 모두 접어도 제목은 남으므로 선은 있다.
    setup({ favoritesCollapsed: true, recentCollapsed: true });
    const second = render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(true);
    second.unmount();
    // 위에 구역이 없으면(서버 6개 이하, 즐겨찾기 없음) 선이 없다.
    setup({}, HOSTS.slice(1, 6));
    render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(false);
  });

  it('옛 저장 파일처럼 값이 없으면 펴진 것으로 본다', () => {
    setup({ favoritesCollapsed: undefined as unknown as boolean, recentCollapsed: undefined as unknown as boolean });
    render(<Sidebar />);
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('true');
    expect(rows('srv-1')).toBe(2);
  });
});
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -25,6 +25,7 @@
       "systemctl\\s+(stop|restart)"
     ],
     "domainNoticeDone": false,
+    "favoritesCollapsed": false,
     "fontFamily": "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
     "fontSize": 14,
     "guardDangerousOnProd": true,
@@ -35,6 +36,7 @@
     "lineHeight": 1.15,
     "maxVerticalSplits": 3,
     "osc52Copy": true,
+    "recentCollapsed": false,
     "rememberWindow": true,
     "restoreLayout": true,
     "rightClickPaste": true,
```

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -25,6 +25,7 @@
       "systemctl\\s+(stop|restart)"
     ],
     "domainNoticeDone": false,
+    "favoritesCollapsed": false,
     "fontFamily": "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
     "fontSize": 14,
     "guardDangerousOnProd": true,
@@ -35,6 +36,7 @@
     "lineHeight": 1.15,
     "maxVerticalSplits": 3,
     "osc52Copy": true,
+    "recentCollapsed": false,
     "rememberWindow": true,
     "restoreLayout": true,
     "rightClickPaste": true,
```


## F. 명세·문서

### `spec/ui/sidebar.md` — 수정

```diff
--- a/spec/ui/sidebar.md
+++ b/spec/ui/sidebar.md
@@ -21,8 +21,9 @@
 ## 3. 섹션
 
 - **즐겨찾기**: 검색 중이 아니고 즐겨찾기 서버가 있을 때. 제목 "즐겨찾기"(별 아이콘). 이 섹션의 행은 끌 수 없고 선택 강조도 하지 않는다.
+- **구역 접기**(2026-11-25, D131 — 사용자 요청, 기존 앱에는 없던 기능): "즐겨찾기"와 "최근 사용"의 **제목 줄을 누르면 그 구역이 접히고**(제목만 남고 서버 줄이 사라진다) 다시 누르면 펴진다. 제목 줄 맨 앞에 폴더 줄과 같은 화살표(펴짐 ▾ / 접힘 ▸)가 있고, 그 뒤에 원래의 아이콘과 이름이 온다. 줄 전체가 버튼이다(키보드 포커스·Enter·Space 로도 된다). 접힘은 구역마다 따로 설정에 저장한다 — 즐겨찾기 `favoritesCollapsed`, 최근 사용 `recentCollapsed`(둘 다 기본 false). 앱을 다시 열어도 유지되고, 값이 없으면 펴진 것으로 본다. 구역이 나올 조건(아래)이 아니면 제목도 없다. 검색 중에는 지금처럼 두 구역이 통째로 숨고, 검색을 지우면 접힘 상태 그대로 돌아온다. "서버" 구역은 접지 않는다.
 - **최근 접속**: 검색 중이 아니고 **서버가 6개를 넘고**(7개 이상) 최근 사용 기록이 있는 서버가 있을 때. 제목 "최근 사용"(시계 아이콘). 최근 사용 시각 내림차순 **최대 5개**.
-- **서버**: 제목 "서버 · {전체 서버 수}"(서버가 0이면 개수 생략). 그 아래 최상위 폴더(정렬 키 오름차순, 같으면 이름 순) → 폴더 없는 서버(또는 소속 폴더가 없어진 서버)(정렬 키 오름차순, 같으면 이름 순).
+- **서버**: 제목 "서버 · {전체 서버 수}"(서버가 0이면 개수 생략). (2026-11-25, D132) 위에 즐겨찾기나 최근 사용 구역이 보이는 동안에는(접혀 있어도) 제목 **위에 가로 구분선**(테두리색 1px, 목록 안쪽 너비, 선 위아래로 6px 띄움)을 그어 위 구역들과 나눈다. 위에 구역이 없으면(검색 중 포함) 선이 없다. 그 아래 최상위 폴더(정렬 키 오름차순, 같으면 이름 순) → 폴더 없는 서버(또는 소속 폴더가 없어진 서버)(정렬 키 오름차순, 같으면 이름 순).
 - 서버가 하나도 없으면 안내: "아직 저장된 서버가 없습니다." 줄바꿈 "**+ 서버**로 추가하거나" 줄바꿈 "**가져오기**로 PuTTY 세션을 불러오세요."
 
 ## 4. 폴더 행
```

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -114,6 +114,8 @@
 | syncAuto | true | 자동 동기화 |
 | osc52Copy | true | 원격의 OSC 52 클립보드 쓰기 허용 |
 | sidebarWidth | 260 | px |
+| favoritesCollapsed | false | 사이드바 "즐겨찾기" 구역을 접어 두었는지 ([../ui/sidebar.md](../ui/sidebar.md) §3, D131) |
+| recentCollapsed | false | 사이드바 "최근 사용" 구역을 접어 두었는지 (같은 곳) |
 | rightPanelWidth | 340 | px |
 
 dangerousPatterns 기본 목록(정규식 원문, 순서 유지):
```

### `spec/acceptance/02-shell-sidebar.md` — 수정

```diff
--- a/spec/acceptance/02-shell-sidebar.md
+++ b/spec/acceptance/02-shell-sidebar.md
@@ -42,6 +42,9 @@
 | B10 | 사이드바 경계를 500px→800px 로 끌기 | | 520px 에서 멈추고 설정에 저장 |
 | B10a | 사이드바 너비 290px, 명령 모음 켬(명령 0개 — 긴 안내 문구가 보임) | | 서버 목록·명령 모음·왼쪽 열이 모두 290px. 작업 영역이 290px 에서 시작한다. 켜기 전과 열 너비가 같다 (D130) |
 | B10b | 명령 모음 옆(왼쪽 열 아래쪽)의 오른쪽 가장자리를 끌기 | | 서버 목록과 명령 모음이 함께 넓어지고 좁아진다 (D130) |
+| B11 | 서버 8개, 최근 사용 기록 3개 | "최근 사용" 제목을 누른다 | 그 구역의 서버 줄이 사라지고 화살표가 ▸ 로 바뀐다. 서버 목록은 그대로. 설정 `recentCollapsed` 가 true (D131) |
+| B12 | B11 뒤 앱을 껐다 켠다 | | "최근 사용"이 접힌 채로 나온다. 다시 누르면 펴지고 false 로 저장 |
+| B13 | "즐겨찾기" 제목을 누른다 | | 즐겨찾기 구역만 접힌다(`favoritesCollapsed`). 최근 사용은 그대로 |
 
 ## 빠른 접속 (ui/dialogs.md §2)
 | # | 조건 | 동작 | 기대 결과 |
```

### `docs/scenarios.md` — 수정

```diff
--- a/docs/scenarios.md
+++ b/docs/scenarios.md
@@ -59,7 +59,7 @@
 | 3-5 | 폴더 만들기·정리 | [폴더 추가], 폴더 우클릭(새 하위 폴더·폴더 수정·삭제) | 폴더 환경을 정하면 안의 서버가 그 색을 물려받는다. 삭제: 비어 있으면 바로, 서버가 있으면 [서버까지 삭제]/[아니오(폴더만 삭제, 서버는 위로)]/[취소]. |
 | 3-6 | 환경 표시 | 서버·폴더 환경 = 운영/검증/부하 테스트/개발/네트워크 장비 | 색 막대·알약으로 표시. **운영**은 빨간 테두리 + 위험 명령 확인(6-6). |
 | 3-7 | 찾기 | 사이드바 "서버 찾기" | 이름·주소·태그로 거른다. 없으면 "…와 일치하는 서버가 없습니다". |
-| 3-8 | 즐겨찾기·최근 | 서버 우클릭 "즐겨찾기에 추가" | 목록 위 "즐겨찾기", "최근 사용" 구역. |
+| 3-8 | 즐겨찾기·최근 | 서버 우클릭 "즐겨찾기에 추가" | 목록 위 "즐겨찾기", "최근 사용" 구역. 구역 제목을 누르면 폴더처럼 접히고, 접어 둔 상태는 저장돼 다시 열어도 그대로다. |
 | 3-9 | 옮기기 | 서버·폴더를 끌어 다른 폴더에 놓기 | 정렬·이동, 바로 저장. |
 | 3-10 | 여러 개 선택 | Ctrl+클릭 / Shift+클릭 | "서버 n개 선택" 막대. 끌어서 폴더로 옮기기, 우클릭 "삭제 (n개)". |
 | 3-11 | 서버 우클릭 | — | 새 탭으로 열기 · 오른쪽 화면에 열기 · 아래 창으로 열기 · 수정(F2) · 복사본 만들기 · 즐겨찾기 · 접속 주소 복사 · 삭제. |
```

### `docs/scenarios.en.md` — 수정

```diff
--- a/docs/scenarios.en.md
+++ b/docs/scenarios.en.md
@@ -58,7 +58,7 @@
 | 3-5 | Folders | [Add folder], right-click a folder (New subfolder · Edit folder · Delete) | Servers inherit the folder's environment color. Delete: empty folders go immediately; with servers you choose [Delete servers too] / [No (delete folder only — servers move up)] / [Cancel]. |
 | 3-6 | Environment | Environment = Production/Staging/Load test/Development/Network device | Shown as a color bar and pill. **Production** gets a red border and a dangerous-command check (6-6). |
 | 3-7 | Search | Sidebar "Search servers" | Filters by name, address and tags. Otherwise "No servers match …". |
-| 3-8 | Favorites and recent | Right-click a server "Add to favorites" | "Favorites" and "Recent" sections at the top of the list. |
+| 3-8 | Favorites and recent | Right-click a server "Add to favorites" | "Favorites" and "Recent" sections at the top of the list. Click a section title to collapse it like a folder; the collapsed state is saved and kept the next time you open the app. |
 | 3-9 | Move | Drag servers/folders onto another folder | Reorders/moves and saves right away. |
 | 3-10 | Select several | Ctrl+click / Shift+click | "N servers selected" bar. Drag them into a folder, or right-click "Delete (N)". |
 | 3-11 | Server context menu | — | Open in new tab · Open in right view · Open in pane below · Edit (F2) · Duplicate · Favorites · Copy address · Delete. |
```

`CHANGELOG.md` 의 미배포 절 "새 기능"에 한 줄, `docs/decisions.md` 에 D131·D132 를 넣었다. 문구(`locales/*.json`)는 바꾸지 않았다.

---

## G. 검증 (2026-11-25 실행)

### G.1 프런트 — 새 파일 [frontend/src/screens/Sidebar.sections.test.tsx](../../frontend/src/screens/Sidebar.sections.test.tsx) 9개

서버 8개(앞의 3개는 접속 기록이 있고 그중 하나는 즐겨찾기)로 서버 목록을 그리고, 가짜 백엔드가 `store.patch` 를 받아 두었다가 `store.get` 으로 돌려준다.

| 테스트 | 확인하는 것 |
|---|---|
| 기본은 둘 다 펴져 있다 | 기본값 false / 두 제목이 버튼이고 `aria-expanded=true` / 서버가 구역에도 보인다 |
| **"최근 사용" 제목을 누르면 접히고 `recentCollapsed: true` 로 저장된다** | 그 구역의 서버 줄이 사라짐, 즐겨찾기 구역·"서버 · 8" 은 그대로, `store.patch` 로 true 한 번, 저장된 값 확인 |
| 다시 누르면 펴지고 false 로 저장된다 | — |
| "즐겨찾기"도 같은 방식으로, `favoritesCollapsed` 로 따로 저장된다 | 최근 사용의 값은 그대로 |
| **저장된 접힘 상태로 다시 그려도 접혀 있다** | 설정에 true 가 든 채 새로 그리면(앱을 다시 연 것) 둘 다 접혀 있다 |
| 접혀 있어도 검색하면 구역 자체가 숨고, 검색을 지우면 접힌 채로 돌아온다 | — |
| 구역이 나올 조건이 아니면 제목도 없다 | 서버 6개 이하·즐겨찾기 없음 |
| "서버" 제목 위의 구분선(D132) | 위에 구역이 있으면 "서버 · n" 제목에 `divided` 가 붙는다 / 검색 중에는 없다 / 위 구역을 모두 접어도 있다 / 위에 구역이 없으면(서버 6개 이하, 즐겨찾기 없음) 없다 |
| 옛 저장 파일처럼 값이 없으면 펴진 것으로 본다 | — |

### G.2 그 밖의 자동 테스트

| 테스트 | 확인하는 것 |
|---|---|
| 계약 테스트(`internal/app`, `frontend/src/ipc/contract.test.ts`) | `store.get` 의 `favoritesCollapsed: false`·`recentCollapsed: false` 가 백엔드 응답과 프런트 고정 파일에서 같다 |
| 기존 사이드바 테스트(여러 개 선택, 폴더 삭제, 규칙) | 고치지 않고 통과 |

### G.3 실행한 명령

```bash
go build ./... && go test -count=1 ./...     # 전부 통과 (이 PC 는 cgo 가 꺼져 있어 -race 는 돌리지 못했다)
(cd frontend && npx tsc --noEmit)            # 통과
(cd frontend && npx vitest run)              # PowerShell 에서. 54개 파일 366개 통과 (앞 문서까지 53개 파일 357개 + 이 문서 9개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

자동 테스트가 확인하지 못하는 것:
- 실제 앱을 껐다 켰을 때 접힘이 남는지(테스트는 "저장 요청을 보낸다"와 "저장된 값으로 그리면 접혀 있다"를 따로 본다. 그 사이의 디스크 저장은 다른 설정과 같은 길이다).
- 화살표·제목 줄의 실제 모양과 마우스를 올렸을 때의 색, 구분선의 실제 굵기·색·위아래 간격(테스트는 선을 그리는 클래스가 붙는지만 본다).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 서버 목록을 본다(서버 7개 이상, 접속한 적 있음) | "최근 사용" 제목 앞에 ▾ 화살표. 아래에 최근 서버 최대 5개(전과 같다) |
| 2 | "최근 사용" 제목을 누른다 | 서버 줄이 사라지고 화살표가 ▸. 아래 "서버" 구역이 위로 올라온다 |
| 3 | 다시 누른다 | 펴진다 |
| 4 | 접어 둔 채 앱을 껐다 켠다 | **접힌 채로** 나온다 |
| 5 | 즐겨찾기가 있는 상태에서 "즐겨찾기" 제목을 누른다 | 즐겨찾기 구역만 접힌다. 최근 사용은 그대로 |
| 6 | 접어 둔 채 검색 칸에 글자를 넣었다 지운다 | 검색 중에는 두 구역이 없고, 지우면 접힌 채로 돌아온다 |
| 7 | 접어 둔 채 서버에 접속한다 | 접힌 채 그대로(펴지지 않는다) |
| 8 | 제목 줄에 마우스를 올린다 / Tab 으로 포커스를 준 뒤 Enter | 글자색이 조금 밝아지고 손 모양 커서 / 접히고 펴진다 |
| 9 | 데이터 폴더의 store.json | `"recentCollapsed": true` 등이 `settings` 에 있다 |
| 10 | 라이트 테마, 다른 언어(영어) | 모양이 같다. 제목은 "Recent"·"Favorites" |
| 11 | "서버 · n" 제목 위 | 가는 가로선이 있고 위의 구역과 조금 떨어져 있다. 위의 두 구역을 모두 접어도 선이 있다 |
| 12 | 검색 칸에 글자를 넣는다 / 즐겨찾기가 없고 서버가 6개 이하인 상태 | 선이 없다("서버" 제목이 검색 칸 아래에 바로 온다) |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 구역 제목의 모양 | `div` 를 `button` 으로 바꿨다. 버튼의 기본 모양(테두리·바탕·글꼴)이 묻어날 수 있다 | 스타일에서 테두리·바탕을 없애고 글꼴을 물려받게 했다. 글자 크기·굵기·색·여백은 같은 클래스(`.side-section`)를 그대로 쓴다. 수동 검증 1·8 |
| 제목의 위치 | 화살표만큼 제목 글자가 오른쪽으로 간다. "서버" 구역 제목은 그대로라 세 제목의 글자 시작 위치가 다르다 | 폴더 줄도 화살표 뒤에 이름이 온다. 거슬리면 "서버" 제목에도 같은 폭의 빈자리를 둘 수 있다 |
| 끌어 놓기 | 서버를 최상위로 끌어 놓는 영역 안에 버튼이 생겼다 | 버튼은 끌기 이벤트를 따로 다루지 않아 바깥 영역이 전처럼 받는다. 기존 끌기 테스트 통과 |
| 저장 값 | 옛 저장 파일은 두 키가 없어 false 로 채워진다. 이 버전에서 저장한 파일을 옛 버전이 읽으면 두 키를 무시한다 | 계약 테스트, `store` 테스트 |
| 동기화 | 한 PC 에서 접으면 git 동기화를 쓰는 다른 PC 에서도 접힌다 | §2 에 적었다. PC 마다 다르게 하려면 저장 위치를 바꿔야 한다 |
| 연속으로 누름 | 빠르게 여러 번 누르면 저장 요청이 겹친다 | 설정 저장 도우미(`saveSettings`)가 저장 중인 값을 기준으로 합친다(다른 토글과 같다) |
| 구분선 | 선과 여백만큼 "서버" 제목이 아래로 내려간다(약 9px) | 위에 구역이 있을 때만이다. 싫으면 `.side-section.divided` 한 줄을 지우면 된다 |
| 접속 직후 | 접속하면 최근 사용 순서가 바뀌며 목록이 다시 그려진다 | 접힘 값은 설정에서 읽으므로 그대로다. 수동 검증 7 |

되돌리기: §B 의 `-` 줄로 되돌리고 새 테스트 파일을 지운다. 저장 파일에 남은 두 키는 읽는 코드가 없으면 무시된다.

## J. 체크리스트

- [x] `Sidebar.tsx`: 구역 제목을 버튼으로(`sectionHead`), 접혔으면 서버 줄을 그리지 않음, 누르면 설정 저장
- [x] `app.css`: 버튼 제목의 모양, "서버" 제목 위 구분선(`.side-section.divided`, D132)
- [x] 설정 2개: 프런트 타입·기본값, `store.go` 기본값, 계약 고정 파일 2개
- [x] 프런트 테스트 9개 추가. `go test ./...`·`tsc`·`vitest`(366개)·빌드
- [x] 명세 ui/sidebar.md §3·data/store.md §4·수용 B11~B13, 시나리오 ko·en, CHANGELOG, decisions D131·D132, CLAUDE.md
- [ ] 수동 검증 1~12
