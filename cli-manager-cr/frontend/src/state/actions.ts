// 공통 동작: 서버 열기(새 탭·칸), 단축키 실행, 세션 이벤트 연결. 근거: spec/behavior/shortcuts.md, spec/ui/panes.md §4·§9, spec/ipc.md §4.
import { getBackend, errorMessage } from '../ipc/backend';
import type { SessionInfo } from '../ipc/types';
import { useApp } from './app';
import { useUi } from './ui';
import { addGridTab, groupOf, openSessionFor, useWorkspace, type Tab } from './workspace';
import { findPane, paneIds, type Dir, type LayoutNode } from '../lib/layout';
import { sameTargetRequest } from '../lib/sameTarget';
import type { ShortcutAction } from '../lib/keys';

/** 포커스 그룹(나뉘지 않았으면 전체)의 탭 목록. */
function focusGroupTabs(ws: { tabs: Tab[]; activeTabId: string | null }): Tab[] {
  const cur = ws.tabs.find((t) => t.id === ws.activeTabId);
  const g = cur ? groupOf(cur) : 0;
  return ws.tabs.filter((t) => groupOf(t) === g);
}

/** 상하 분할 한도(설정 maxVerticalSplits, 1~5, 기본 3). */
export function maxVerticalSplits(): number {
  const v = Number(useApp.getState().store?.settings.maxVerticalSplits);
  return Number.isFinite(v) && v >= 1 ? Math.min(5, Math.floor(v)) : 3;
}

/** 이 탭에 칸을 하나 더 쌓을 수 있는지 (탭의 칸 수 < 한도). */
export function canSplitMore(root: LayoutNode, max = maxVerticalSplits()): boolean {
  return paneIds(root).length < max;
}

/** 저장된 서버 또는 즉석 접속을 연다. where: 새 탭, 또는 포커스 칸 기준 분할/빈 칸 채우기. */
export async function openHost(
  hostId: string | null,
  where: { kind: 'tab' } | { kind: 'split'; dir: Dir; before?: boolean; paneId?: string | null } = { kind: 'tab' },
  adhoc?: unknown,
): Promise<void> {
  const info = await openSessionFor(hostId, adhoc);
  if (!info) return;
  const ws = useWorkspace.getState();
  if (where.kind === 'tab') {
    ws.openInNewTab(info);
    return;
  }
  // 분할 대상이 없으면 새 탭. 포커스 칸이 빈 칸이면 그 칸을 채운다.
  // 대상 칸을 주면 그 칸이 있는 탭(화면 나누기의 다른 그룹일 수 있다), 아니면 활성 탭의 포커스 칸.
  const active = (where.paneId && ws.tabs.find((t) => findPane(t.root, where.paneId!))) || ws.tabs.find((t) => t.id === ws.activeTabId);
  const targetId = where.paneId ?? active?.focusedPane ?? null;
  if (!targetId || !active) {
    ws.openInNewTab(info);
    return;
  }
  const pane = findPane(active.root, targetId);
  if (pane && !pane.sessionId) {
    ws.fillPane(targetId, info);
    return;
  }
  // 분할 한도에 이르면 쪼개지 않고 새 탭에서 연다 (ui/panes.md §2).
  if (!canSplitMore(active.root)) {
    useApp.getState().pushToast({ kind: 'info', text: `화면 분할은 최대 ${maxVerticalSplits()}개까지라 새 탭에서 열었습니다` });
    ws.openInNewTab(info);
    return;
  }
  ws.openInPane(targetId, where.dir, info, where.before);
}

/** 같은 대상으로 분할(panes §10): 세션 종류에 따라 같은 서버·같은 로컬 폴더·같은 즉석 대상. */
export async function splitSameAs(info: SessionInfo, dir: Dir, paneId: string): Promise<void> {
  const req = sameTargetRequest(info);
  if (!req) {
    useApp.getState().pushToast({ kind: 'info', text: '같은 대상을 다시 해석할 수 없어 분할하지 않았습니다' });
    return;
  }
  if (req.kind === 'host') await openHost(req.hostId, { kind: 'split', dir, paneId });
  else await openHost(null, { kind: 'split', dir, paneId }, req.adhoc);
}

/** 포커스 칸 기준 이동: 방향키. 위치 기준 가장 가까운 칸을 고른다(단순 구현: 칸 배열 순서). */
function moveFocus(dir: 'up' | 'down' | 'left' | 'right') {
  const ws = useWorkspace.getState();
  const tab = ws.tabs.find((t) => t.id === ws.activeTabId);
  if (!tab || !tab.focusedPane) return;
  const ids = paneIds(tab.root);
  const i = ids.indexOf(tab.focusedPane);
  const step = dir === 'left' || dir === 'up' ? -1 : 1;
  const next = ids[i + step];
  if (next) ws.focusPane(next);
}

/** 단축키 동작 실행. 조건(대화상자, 볼트 잠김)은 호출 전에 확인한다. */
export function runShortcut(action: ShortcutAction, arg?: number | 'last' | 'up' | 'down' | 'left' | 'right'): void {
  const ws = useWorkspace.getState();
  const active = ws.tabs.find((t) => t.id === ws.activeTabId);
  const focused = active?.focusedPane ?? null;
  switch (action) {
    case 'quickToggle': {
      const q = useUi.getState().quick;
      if (q.open) useUi.getState().closeQuick();
      else useUi.getState().openQuick(null);
      return;
    }
    case 'quickOpen':
      useUi.getState().openQuick(null);
      return;
    case 'newLocal':
      void openHost(null, { kind: 'tab' }, { host: '', protocol: 'local' });
      return;
    case 'closePane':
      if (focused) void ws.closePane(focused);
      return;
    case 'splitRight':
    case 'splitDown': {
      const sess = active && focused ? findPane(active.root, focused)?.sessionId : null;
      const info = sess ? ws.sessions[sess]?.info : undefined;
      if (!info || !focused) {
        useApp.getState().pushToast({ kind: 'info', text: '분할할 세션이 없습니다' });
        return;
      }
      if (active && !canSplitMore(active.root)) {
        useApp.getState().pushToast({ kind: 'info', text: `화면 분할은 최대 ${maxVerticalSplits()}개까지입니다` });
        return;
      }
      void splitSameAs(info, action === 'splitRight' ? 'row' : 'col', focused);
      return;
    }
    case 'zoom':
      if (active) ws.toggleZoom(active.id);
      return;
    case 'find':
      if (focused) window.dispatchEvent(new CustomEvent('termgo:find', { detail: focused }));
      return;
    case 'sftp': {
      // 포커스 그룹의 SFTP 패널을 열고 닫는다.
      useUi.getState().toggleSftp(active ? groupOf(active) : 0);
      return;
    }
    case 'broadcast':
      if (active) ws.toggleBroadcast(active.id);
      return;
    case 'sidebar':
      useUi.getState().toggleSidebar();
      return;
    case 'tabLeft':
    case 'tabRight':
      if (active) {
        // 포커스 그룹 안에서만 이웃 탭과 자리를 바꾼다(ui/shell.md §2.3).
        const mates = ws.tabs.filter((t) => groupOf(t) === groupOf(active));
        const i = mates.findIndex((t) => t.id === active.id);
        const nb = mates[action === 'tabLeft' ? i - 1 : i + 1];
        if (nb) ws.moveTab(active.id, ws.tabs.findIndex((t) => t.id === nb.id));
      }
      return;
    case 'tabNext':
    case 'tabPrev': {
      const mates = focusGroupTabs(ws);
      if (mates.length === 0) return;
      const i = mates.findIndex((t) => t.id === ws.activeTabId);
      const n = action === 'tabNext' ? (i + 1) % mates.length : (i - 1 + mates.length) % mates.length;
      ws.activateTab(mates[n].id);
      return;
    }
    case 'tabIndex': {
      const mates = focusGroupTabs(ws);
      const idx = arg === 'last' ? mates.length - 1 : (arg as number);
      if (typeof idx === 'number' && mates[idx]) ws.activateTab(mates[idx].id);
      return;
    }
    case 'focusMove':
      if (arg === 'up' || arg === 'down' || arg === 'left' || arg === 'right') moveFocus(arg);
      return;
    case 'fontUp':
    case 'fontDown':
    case 'fontReset': {
      const st = useApp.getState().store;
      if (!st) return;
      const cur = st.settings.fontSize;
      const next = action === 'fontReset' ? 14 : Math.max(8, Math.min(32, cur + (action === 'fontUp' ? 1 : -1)));
      void useApp.getState().patchStore({ settings: { ...st.settings, fontSize: next } });
      return;
    }
    case 'fullscreen':
      // 창 최대화 ↔ 복원 토글은 백엔드 채널이 담당한다.
      void getBackend().invoke('app.toggleFullScreen').catch(() => undefined);
      return;
    default:
      return;
  }
}

/** 세션 이벤트(session.state, session.data)를 작업 영역 상태에 연결한다. */
export function subscribeSessionEvents(): () => void {
  const b = getBackend();
  const offs = [
    b.on('session.state', (info: SessionInfo) => {
      const ws = useWorkspace.getState();
      ws.upsertSession(info);
      if (info.state === 'error' && info.message) useApp.getState().pushToast({ kind: 'error', text: info.message });
    }),
    b.on('session.data', (id: string, text: string) => useWorkspace.getState().writeOutput(id, text)),
  ];
  return () => offs.forEach((off) => off());
}

export function reportError(e: unknown): void {
  useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
}

/** 폴더 전체를 그리드로 연다(panes §6). 하위 폴더 포함, sort 오름차순, 16개 초과는 확인. */
export async function openFolderGrid(groupId: string): Promise<void> {
  const st = useApp.getState().store;
  if (!st) return;
  const group = st.groups.find((g) => g.id === groupId);
  const ids = new Set<string>([groupId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const g of st.groups) if (g.parentId && ids.has(g.parentId) && !ids.has(g.id)) { ids.add(g.id); grew = true; }
  }
  const hosts = st.hosts.filter((h) => h.groupId && ids.has(h.groupId)).sort((a, b) => a.sort - b.sort);
  if (hosts.length === 0) {
    useApp.getState().pushToast({ kind: 'info', text: '이 폴더에 서버가 없습니다' });
    return;
  }
  if (hosts.length > 16) {
    const ok = await useUi.getState().confirm({ title: '여러 서버 열기', body: `${hosts.length}개 서버에 동시에 접속할까요?`, okText: '확인' });
    if (!ok) return;
  }
  const infos: SessionInfo[] = [];
  for (const h of hosts) {
    const info = await openSessionFor(h.id);
    if (info) infos.push(info);
  }
  addGridTab(group?.name ?? '폴더', infos);
}

/** 칸을 다른 탭으로 옮기지 않는다(panes §7 은 같은 탭 안만). 탭 분리는 detachPanes 를 쓴다. */
export { detachPanes } from './workspace';
