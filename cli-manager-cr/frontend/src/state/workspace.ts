// 작업 영역 상태: 탭 목록, 탭별 칸 트리, 세션 정보와 활동 표시. 근거: spec/ui/panes.md, spec/ui/shell.md §2.
import { create } from 'zustand';
import { getBackend, errorMessage } from '../ipc/backend';
import type { SessionInfo } from '../ipc/types';
import {
  findPane, findPaneBySession, gridLayout, movePaneToEdge, newId, newPane, paneIds, removePane, resizeSplit,
  sessionsIn, splitPane, swapSessions, type Dir, type LayoutNode, type PaneNode,
} from '../lib/layout';
import { useApp } from './app';
import { relabel, t } from '../i18n';
import { ackOutput } from '../lib/outputAck';

export interface Tab {
  id: string;
  /** 사용자가 정한 이름. 없으면 포커스 칸 세션 제목 → "세션". */
  title?: string;
  broadcast: boolean;
  /** 탭 그룹(화면 나누기). 오른쪽 그룹이면 1, 아니면 없음(왼쪽). ui/shell.md §2.3 */
  group?: 1;
  root: LayoutNode;
  focusedPane: string | null;
  zoomed: string | null;
}

export interface SessionEntry {
  info: SessionInfo;
  /** 비활성 탭에서 새 출력이 있으면 true(상태 점 "활동"). */
  activity: boolean;
}

interface WorkspaceState {
  tabs: Tab[];
  /** 포커스 그룹의 선택 탭 id. */
  activeTabId: string | null;
  /** 그룹별 선택 탭 id([왼쪽, 오른쪽]). */
  activeByGroup: [string | null, string | null];
  /** 나뉘었을 때 왼쪽 그룹의 너비 비율(%, 20~80). */
  groupRatio: number;
  setGroupRatio: (pct: number) => void;
  sessions: Record<string, SessionEntry>;
  /** 열린 세션의 터미널 기록기(xterm 인스턴스 등록). 출력 기록과 크기 맞춤에 쓴다.
   *  done 은 xterm 이 text 를 다 쓴 뒤 불린다(출력 흐름 제어 ack, behavior/session.md §8). */
  writers: Record<string, Writer>;
  /** 세션 터미널의 현재 열×행(상태 표시줄 표시용). */
  sizes: Record<string, { cols: number; rows: number }>;
  setSize: (sessionId: string, cols: number, rows: number) => void;

  registerWriter: (sessionId: string, fn: Writer | null) => void;
  /** 백엔드의 session.data 묶음 하나. 다 쓴 뒤 session.ack 로 알린다. 숨은 세션은 100ms 단위로 몰아서 쓴다. */
  writeOutput: (sessionId: string, text: string) => void;
  /** 숨은 탭에서 모아 둔 출력을 지금 쓴다(탭이 보이게 될 때). */
  flushDeferred: (sessionId: string) => void;
  upsertSession: (info: SessionInfo) => void;
  removeSession: (sessionId: string) => void;

  activateTab: (tabId: string) => void;
  renameTab: (tabId: string, title: string | null) => void;
  /** 탭 순서 변경. group 을 주면 그 그룹으로 옮기며 선택 탭이 된다. */
  moveTab: (tabId: string, toIndex: number, group?: 0 | 1) => void;
  /** 탭을 다른 그룹으로 옮긴다(탭이 1개뿐이면 무시). */
  moveTabToGroup: (tabId: string, group: 0 | 1) => void;
  /** 화면 나누기 해제: 모든 탭을 왼쪽 그룹으로. */
  mergeGroups: () => void;
  closeTab: (tabId: string, force?: boolean) => Promise<void>;
  focusPane: (paneId: string) => void;
  /** 세션을 새 탭으로 연다. */
  openInNewTab: (info: SessionInfo) => void;
  /** 포커스된 칸(또는 지정 칸)에 분할로 새 세션을 붙인다. before 면 앞. */
  openInPane: (targetPaneId: string | null, dir: Dir, info: SessionInfo, before?: boolean) => void;
  /** 빈 칸 채우기 등 칸에 세션을 넣는다(구조 불변). */
  fillPane: (paneId: string, info: SessionInfo) => void;
  closePane: (paneId: string, force?: boolean) => Promise<void>;
  resizeSplit: (tabId: string, splitId: string, boundary: number, delta: number) => void;
  swapPanes: (tabId: string, a: string, b: string) => void;
  movePane: (tabId: string, moving: string, target: string, dir: Dir, before: boolean) => void;
  toggleBroadcast: (tabId: string) => void;
  toggleZoom: (tabId: string) => void;
  gridTabs: () => void;
  /** 빈 칸 생성(분할 시 빈 칸을 먼저 놓을 때). */
  splitEmpty: (paneId: string, dir: Dir, before?: boolean) => void;
}

/** 터미널 기록기. done 은 다 쓴 뒤 부른다(없으면 ack 를 바로 보낸다). */
export type Writer = (text: string, done?: () => void) => void;

/** 세션이 지금 화면에 보이는 탭(왼쪽·오른쪽 그룹의 선택 탭 또는 활성 탭) 안에 있는지. */
function isSessionVisible(s: WorkspaceState, sessionId: string): boolean {
  for (const tab of s.tabs) {
    if (tab.id !== s.activeByGroup[0] && tab.id !== s.activeByGroup[1] && tab.id !== s.activeTabId) continue;
    if (findPaneBySession(tab.root, sessionId)) return true;
  }
  return false;
}

/** 숨은 세션 출력의 몰아쓰기 주기(ms, behavior/session.md §8). */
export const HIDDEN_FLUSH_MS = 100;

/** 숨은 세션의 출력 대기분(세션 id → 조각들, 타이머). 스토어 밖에 둔다(매 묶음마다 상태 갱신을 피함). */
const deferred = new Map<string, { parts: string[]; timer: ReturnType<typeof setTimeout> | null }>();

/** 묶음 count 개를 쓴다: 다 쓴 뒤 ack. writer 가 done 을 부르지 않는 구현이면(인자 1개) 바로 ack. */
function writeWithAck(sessionId: string, w: Writer, text: string, count: number): void {
  if (w.length < 2) {
    w(text);
    ackOutput(sessionId, count);
    return;
  }
  w(text, () => ackOutput(sessionId, count));
}

function writeDeferred(sessionId: string, w: Writer): void {
  const d = deferred.get(sessionId);
  if (!d) return;
  if (d.timer) {
    clearTimeout(d.timer);
    d.timer = null;
  }
  if (d.parts.length === 0) return;
  const count = d.parts.length;
  const text = d.parts.join('');
  d.parts = [];
  writeWithAck(sessionId, w, text, count);
}

/** 세션이 사라지거나 writer 가 빠질 때: 대기분을 버리고 ack 만 보낸다(백엔드가 읽기를 멈추지 않게). */
function dropDeferred(sessionId: string): void {
  const d = deferred.get(sessionId);
  if (!d) return;
  if (d.timer) clearTimeout(d.timer);
  if (d.parts.length) ackOutput(sessionId, d.parts.length);
  deferred.delete(sessionId);
}

/** 테스트용: 몰아쓰기 대기분을 비운다(타이머 포함, ack 없음). */
export function resetDeferredForTest(): void {
  for (const d of deferred.values()) if (d.timer) clearTimeout(d.timer);
  deferred.clear();
}

function tabOf(state: WorkspaceState, tabId: string): Tab | undefined {
  return state.tabs.find((t) => t.id === tabId);
}

function tabIdOfPane(tabs: Tab[], paneId: string): string | undefined {
  return tabs.find((t) => findPane(t.root, paneId))?.id;
}

function patchTab(tabs: Tab[], tabId: string, fn: (t: Tab) => Tab): Tab[] {
  return tabs.map((t) => (t.id === tabId ? fn(t) : t));
}

export type GroupIdx = 0 | 1;
export const groupOf = (t: Tab): GroupIdx => (t.group === 1 ? 1 : 0);
export const isSplit = (tabs: Tab[]): boolean => tabs.some((t) => t.group === 1);

interface Placement {
  tabs: Tab[];
  activeByGroup: [string | null, string | null];
  activeTabId: string | null;
}

/** 그룹·선택 탭 상태를 일관되게 맞춘다: 빈 그룹 합치기, 선택 탭 보정, 포커스 탭 보정. */
export function settle(tabs: Tab[], by: [string | null, string | null], focus: string | null): Placement {
  let ts = tabs;
  let ab: [string | null, string | null] = [by[0], by[1]];
  if (ts.some((t) => groupOf(t) === 1) && !ts.some((t) => groupOf(t) === 0)) {
    ts = ts.map((t) => ({ ...t, group: undefined }));
    ab = [ab[1] ?? ab[0], null];
  }
  for (const g of [0, 1] as const) {
    const inG = ts.filter((t) => groupOf(t) === g);
    if (!inG.some((t) => t.id === ab[g])) ab[g] = inG[0]?.id ?? null;
  }
  const f = focus && ts.some((t) => t.id === focus) ? focus : (ab[0] ?? ab[1]);
  if (f) ab[groupOf(ts.find((t) => t.id === f)!)] = f;
  return { tabs: ts, activeByGroup: ab, activeTabId: f ?? null };
}

/** 새 탭을 포커스 그룹에 붙이고 선택한다. */
function withNewTab(s: WorkspaceState, tab: Tab): Placement {
  const cur = s.tabs.find((t) => t.id === s.activeTabId);
  const nt: Tab = { ...tab, group: cur && groupOf(cur) === 1 ? 1 : undefined };
  return settle([...s.tabs, nt], s.activeByGroup, nt.id);
}

/** 탭 이름(표시용): 사용자 이름 → 포커스 칸 세션 제목 → "세션". */
export function tabTitle(tab: Tab, sessions: Record<string, SessionEntry>): string {
  if (tab.title) return relabel(tab.title);
  const pane = tab.focusedPane ? findPane(tab.root, tab.focusedPane) : undefined;
  const sid = pane?.sessionId;
  if (sid && sessions[sid]) return relabel(sessions[sid].info.title);
  return t('tab.session');
}

export const useWorkspace = create<WorkspaceState>((set, get) => ({
  tabs: [],
  activeTabId: null,
  activeByGroup: [null, null],
  groupRatio: 50,
  setGroupRatio(pct) {
    set({ groupRatio: Math.max(20, Math.min(80, Math.round(pct * 10) / 10)) });
  },
  sessions: {},
  writers: {},
  sizes: {},
  setSize(sessionId, cols, rows) {
    set((s) => ({ sizes: { ...s.sizes, [sessionId]: { cols, rows } } }));
  },

  registerWriter(sessionId, fn) {
    if (!fn) dropDeferred(sessionId);
    set((s) => {
      const writers = { ...s.writers };
      if (fn) writers[sessionId] = fn;
      else delete writers[sessionId];
      return { writers };
    });
  },

  writeOutput(sessionId, text) {
    const s = get();
    const w = s.writers[sessionId];
    if (!w) {
      // 터미널이 아직 없거나 이미 버려짐: 출력은 버리되 ack 는 보낸다(안 보내면 백엔드가 읽기를 멈춘다).
      ackOutput(sessionId);
      return;
    }
    if (isSessionVisible(s, sessionId)) {
      // 보이는 세션이 다시 보이게 된 직후라면 모아 둔 것부터 쓴다(순서 유지).
      if (deferred.get(sessionId)?.parts.length) writeDeferred(sessionId, w);
      writeWithAck(sessionId, w, text, 1);
      return;
    }
    // 숨은 탭: 100ms 단위로 몰아서 쓴다(보이는 탭에 프레임 시간을 양보).
    let d = deferred.get(sessionId);
    if (!d) {
      d = { parts: [], timer: null };
      deferred.set(sessionId, d);
    }
    d.parts.push(text);
    if (!d.timer) {
      d.timer = setTimeout(() => {
        const entry = deferred.get(sessionId);
        if (entry) entry.timer = null;
        const w2 = get().writers[sessionId];
        if (w2) writeDeferred(sessionId, w2);
        else dropDeferred(sessionId);
      }, HIDDEN_FLUSH_MS);
    }
    // 활성 탭에 보이지 않는 세션이면 활동 표시를 켠다(이미 켜져 있으면 상태 갱신 없음).
    const entry = s.sessions[sessionId];
    if (entry && !entry.activity) {
      set((st) => ({ sessions: { ...st.sessions, [sessionId]: { ...st.sessions[sessionId], activity: true } } }));
    }
  },

  flushDeferred(sessionId) {
    const w = get().writers[sessionId];
    if (w) writeDeferred(sessionId, w);
    else dropDeferred(sessionId);
  },

  upsertSession(info) {
    set((s) => ({
      sessions: { ...s.sessions, [info.id]: { info, activity: s.sessions[info.id]?.activity ?? false } },
    }));
  },

  removeSession(sessionId) {
    dropDeferred(sessionId);
    set((s) => {
      const sessions = { ...s.sessions };
      delete sessions[sessionId];
      const tabs = s.tabs.map((t) => {
        const hit = findPaneBySession(t.root, sessionId);
        if (!hit) return t;
        const root = mapSession(t.root, sessionId);
        return { ...t, root };
      });
      return { sessions, tabs };
    });
  },

  activateTab(tabId) {
    set((s) => ({
      ...settle(s.tabs, s.activeByGroup, tabId),
      tabs: patchTab(s.tabs, tabId, (t) => {
        // 탭을 활성화하면 포커스 칸의 활동 표시를 지운다.
        const sid = t.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId : null;
        if (sid && s.sessions[sid]?.activity) {
          useWorkspace.setState((x) => ({ sessions: { ...x.sessions, [sid]: { ...x.sessions[sid], activity: false } } }));
        }
        return t;
      }),
    }));
  },

  renameTab(tabId, title) {
    set((s) => ({ tabs: patchTab(s.tabs, tabId, (t) => ({ ...t, title: title && title.trim() ? title.trim() : undefined })) }));
  },

  moveTab(tabId, toIndex, group) {
    set((s) => {
      const from = s.tabs.findIndex((t) => t.id === tabId);
      if (from < 0) return s;
      const list = [...s.tabs];
      const [item] = list.splice(from, 1);
      const moved = group === undefined ? item : { ...item, group: group === 1 ? (1 as const) : undefined };
      list.splice(Math.max(0, Math.min(toIndex, list.length)), 0, moved);
      if (group === undefined) return { tabs: list };
      return settle(list, s.activeByGroup, tabId);
    });
  },

  moveTabToGroup(tabId, group) {
    set((s) => {
      const t = s.tabs.find((x) => x.id === tabId);
      if (!t || groupOf(t) === group) return s;
      if (s.tabs.length < 2) return s;
      const list = patchTab(s.tabs, tabId, (x) => ({ ...x, group: group === 1 ? (1 as const) : undefined }));
      return settle(list, s.activeByGroup, tabId);
    });
  },

  mergeGroups() {
    set((s) => settle(s.tabs.map((t) => ({ ...t, group: undefined })), s.activeByGroup, s.activeTabId));
  },

  async closeTab(tabId, force = false) {
    const s = get();
    const tab = tabOf(s, tabId);
    if (!tab) return;
    for (const sid of sessionsIn(tab.root)) {
      try {
        await getBackend().invoke('session.close', [sid]);
      } catch {
        /* 닫기 실패는 무시(이미 끊긴 세션일 수 있음) */
      }
      get().removeSession(sid);
    }
    void force;
    // 같은 그룹에서 같은 위치(없으면 마지막) 탭을 활성화(§5.1 5번)
    const g = groupOf(tab);
    const idx = s.tabs.filter((t) => groupOf(t) === g).findIndex((t) => t.id === tabId);
    const tabs = s.tabs.filter((t) => t.id !== tabId);
    const remaining = tabs.filter((t) => groupOf(t) === g);
    const by: [string | null, string | null] = [s.activeByGroup[0], s.activeByGroup[1]];
    if (by[g] === tabId) by[g] = remaining.length ? remaining[Math.min(idx, remaining.length - 1)].id : null;
    const focus = s.activeTabId === tabId ? by[g] : s.activeTabId;
    set(settle(tabs, by, focus));
  },

  focusPane(paneId) {
    set((s) => {
      const tid = tabIdOfPane(s.tabs, paneId);
      if (!tid) return s;
      const tabs = patchTab(s.tabs, tid, (t) => ({ ...t, focusedPane: paneId }));
      const sid = findPane(tabOf({ ...s, tabs } as WorkspaceState, tid)!.root, paneId)?.sessionId;
      const sessions = sid && s.sessions[sid]?.activity ? { ...s.sessions, [sid]: { ...s.sessions[sid], activity: false } } : s.sessions;
      return { ...settle(tabs, s.activeByGroup, tid), sessions };
    });
  },

  openInNewTab(info) {
    get().upsertSession(info);
    const pane: PaneNode = newPane(info.id);
    const tab: Tab = { id: newId('t'), broadcast: false, root: pane, focusedPane: pane.id, zoomed: null };
    set((s) => withNewTab(s, tab));
  },

  openInPane(targetPaneId, dir, info, before = false) {
    get().upsertSession(info);
    const s = get();
    const target = targetPaneId ?? s.tabs.find((t) => t.id === s.activeTabId)?.focusedPane ?? null;
    const tid = target ? tabIdOfPane(s.tabs, target) : s.activeTabId ?? undefined;
    if (!tid || !target) {
      get().openInNewTab(info);
      return;
    }
    const added = newPane(info.id);
    set((st) => ({
      tabs: patchTab(st.tabs, tid, (t) => ({
        ...t,
        root: splitPane(t.root, target, dir, added, before),
        focusedPane: added.id,
        zoomed: null,
      })),
    }));
  },

  splitEmpty(paneId, dir, before = false) {
    const tid = tabIdOfPane(get().tabs, paneId);
    if (!tid) return;
    const added = newPane(null);
    set((st) => ({
      tabs: patchTab(st.tabs, tid, (t) => ({ ...t, root: splitPane(t.root, paneId, dir, added, before), focusedPane: added.id })),
    }));
  },

  fillPane(paneId, info) {
    get().upsertSession(info);
    set((st) => ({
      tabs: st.tabs.map((t) => {
        if (!findPane(t.root, paneId)) return t;
        return { ...t, root: setPaneSession(t.root, paneId, info.id), focusedPane: paneId };
      }),
    }));
  },

  async closePane(paneId, force = false) {
    void force;
    const s = get();
    const tid = tabIdOfPane(s.tabs, paneId);
    if (!tid) return;
    const tab = tabOf(s, tid)!;
    const pane = findPane(tab.root, paneId)!;
    const sid = pane.sessionId;
    if (sid) {
      try {
        await getBackend().invoke('session.close', [sid]);
      } catch {
        /* 무시 */
      }
      get().removeSession(sid);
    }
    // 세션이 있든 없든 칸을 바로 제거한다(빈 칸으로 남기지 않음, §5.1 3번)
    const after = removePane(get().tabs.find((t) => t.id === tid)!.root, paneId);
    if (!after) {
      await get().closeTab(tid);
      return;
    }
    set((st) => ({
      tabs: patchTab(st.tabs, tid, (t) => ({
        ...t,
        root: after,
        zoomed: null,
        focusedPane: t.focusedPane && findPane(after, t.focusedPane) ? t.focusedPane : paneIds(after)[0] ?? null,
      })),
    }));
  },

  resizeSplit(tabId, splitId, boundary, delta) {
    set((s) => ({ tabs: patchTab(s.tabs, tabId, (t) => ({ ...t, root: resizeSplit(t.root, splitId, boundary, delta) })) }));
  },

  swapPanes(tabId, a, b) {
    set((s) => ({ tabs: patchTab(s.tabs, tabId, (t) => ({ ...t, root: swapSessions(t.root, a, b), focusedPane: a })) }));
  },

  movePane(tabId, moving, target, dir, before) {
    set((s) => ({
      tabs: patchTab(s.tabs, tabId, (t) => ({ ...t, root: movePaneToEdge(t.root, moving, target, dir, before), focusedPane: moving, zoomed: null })),
    }));
  },

  toggleBroadcast(tabId) {
    const tab = tabOf(get(), tabId);
    if (!tab) return;
    if (!tab.broadcast && paneIds(tab.root).length < 2) {
      useApp.getState().pushToast({ kind: 'info', text: t('broadcast.needTwo') });
      return;
    }
    set((s) => ({ tabs: patchTab(s.tabs, tabId, (x) => ({ ...x, broadcast: !x.broadcast })) }));
    const on = !tab.broadcast;
    useApp.getState().pushToast({
      kind: 'info',
      text: on ? t('broadcast.on') : t('broadcast.off'),
    });
  },

  toggleZoom(tabId) {
    const t = tabOf(get(), tabId);
    if (!t || paneIds(t.root).length < 2) return;
    set((s) => ({
      tabs: patchTab(s.tabs, tabId, (x) => ({ ...x, zoomed: x.zoomed ? null : x.focusedPane })),
    }));
  },

  gridTabs() {
    const s = get();
    const panes: PaneNode[] = [];
    for (const t of s.tabs) collectPanes(t.root, panes);
    if (panes.length < 2) return;
    const tab: Tab = { id: newId('t'), title: t('host.localTerminal'), broadcast: false, root: gridLayout(panes), focusedPane: panes[0].id, zoomed: null };
    set({ tabs: [tab], activeTabId: tab.id, activeByGroup: [tab.id, null] });
  },
}));

/** 칸의 세션 id 를 바꾼다(null 이면 빈 칸). */
function setPaneSession(root: LayoutNode, paneId: string, sessionId: string | null): LayoutNode {
  if (root.type === 'pane') return root.id === paneId ? { ...root, sessionId } : root;
  return { ...root, children: root.children.map((c) => setPaneSession(c, paneId, sessionId)) };
}

function mapSession(root: LayoutNode, sessionId: string): LayoutNode {
  if (root.type === 'pane') return root.sessionId === sessionId ? { ...root, sessionId: null } : root;
  return { ...root, children: root.children.map((c) => mapSession(c, sessionId)) };
}

function collectPanes(node: LayoutNode, out: PaneNode[]): void {
  if (node.type === 'pane') out.push({ ...node, sessionId: node.sessionId });
  else node.children.forEach((c) => collectPanes(c, out));
}

/** 세션 열기 요청: 서버 id 로 백엔드에 열고 결과 세션 정보를 돌려준다. 실패하면 토스트. */
export async function openSessionFor(hostId: string | null, adhoc?: unknown): Promise<SessionInfo | null> {
  try {
    const info = await getBackend().invoke<SessionInfo>('session.open', [adhoc ? { adhoc } : { hostId, cols: 120, rows: 30 }]);
    return info;
  } catch (e) {
    useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
    return null;
  }
}

/** 패널을 각각 탭으로 분리(panes §6, shell §2.2): 원래 탭 자리에 칸 수만큼 탭을 끼우고 첫 번째를 활성화. */
export function detachPanes(tabId: string): void {
  const s = useWorkspace.getState();
  const idx = s.tabs.findIndex((t) => t.id === tabId);
  const tab = s.tabs[idx];
  if (!tab) return;
  const panes: PaneNode[] = [];
  collectPanes(tab.root, panes);
  if (panes.length < 2) return;
  const created: Tab[] = panes.map((p) => ({ id: newId('t'), broadcast: false, group: tab.group, root: p, focusedPane: p.id, zoomed: null }));
  const tabs = [...s.tabs.slice(0, idx), ...created, ...s.tabs.slice(idx + 1)];
  useWorkspace.setState(settle(tabs, s.activeByGroup, created[0].id));
}

/** 세션들을 그리드 한 탭으로 연다(panes §6 폴더 전체 그리드). infos 는 이미 연 세션들. */
export function addGridTab(title: string, infos: SessionInfo[]): void {
  if (infos.length === 0) return;
  const s = useWorkspace.getState();
  for (const info of infos) s.upsertSession(info);
  const panes = infos.map((i) => newPane(i.id));
  const root = gridLayout(panes);
  const tab: Tab = { id: newId('t'), title, broadcast: false, root, focusedPane: panes[0].id, zoomed: null };
  useWorkspace.setState((st) => withNewTab(st, tab));
}

/** 복원: 저장된 탭 목록으로 교체한다(layout 복원, lifecycle §3.2). */
export function replaceTabs(tabs: Tab[], activeIndex: number, infos: SessionInfo[], activeIndex1?: number, ratio?: number): void {
  for (const info of infos) useWorkspace.getState().upsertSession(info);
  const at = (i: number | undefined) => (i === undefined ? null : tabs[i]?.id ?? null);
  const active = tabs[Math.min(Math.max(activeIndex, 0), tabs.length - 1)];
  useWorkspace.setState({ ...settle(tabs, [at(activeIndex), at(activeIndex1)], active ? active.id : null), ...(typeof ratio === 'number' ? { groupRatio: Math.max(20, Math.min(80, ratio)) } : {}) });
}
