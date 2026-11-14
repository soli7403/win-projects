// 탭·분할 레이아웃 저장과 복원. 근거: spec/behavior/lifecycle.md §3, spec/data/store.md §5.
// 저장: 0.3초 모아서, 복원이 끝나기 전에는 저장하지 않고, JSON 이 같으면 다시 쓰지 않는다.
// 복원: 볼트가 열린 뒤 한 번만. 저장된 서버가 있으면 접속하고, 실패·삭제된 칸은 제외한다.
import type { Host, SessionInfo, Store } from '../ipc/types';
import { useApp } from './app';
import { openSessionFor, replaceTabs, useWorkspace, type Tab } from './workspace';
import { newPane, newId, paneIds, type LayoutNode, type SplitNode } from '../lib/layout';
import { t } from '../i18n';

type SavedNode =
  | { type: 'pane'; empty?: boolean; hostId?: string; local?: { cwd?: string } }
  | { type: 'split'; dir: 'row' | 'col'; sizes: number[]; children: SavedNode[] };

interface SavedTab {
  title?: string;
  broadcast?: boolean;
  group?: 1;
  root: SavedNode;
}

interface SavedLayout {
  tabs: SavedTab[];
  active: number;
  /** 오른쪽 그룹의 선택 탭 인덱스(나뉘었을 때). */
  active1?: number;
  /** 왼쪽 그룹 너비 비율(%). 50 이면 생략. */
  ratio?: number;
}

let restored = false;
let attempted = false;
let lastJson = '';
let timer: ReturnType<typeof setTimeout> | undefined;

/** 크기를 남은 자식 합이 100 이 되도록 비례 재조정. */
function renorm(sizes: number[]): number[] {
  const total = sizes.reduce((a, b) => a + b, 0) || 1;
  return sizes.map((s) => (s * 100) / total);
}

/** 저장 가능한 노드로 바꾼다. 저장하지 않을 칸(즉석 SSH 등)은 null. */
function saveNode(node: LayoutNode, hosts: Host[], sessions: Record<string, { info: SessionInfo }>): SavedNode | null {
  if (node.type === 'pane') {
    if (!node.sessionId) return { type: 'pane', empty: true };
    const info = sessions[node.sessionId]?.info;
    if (!info) return { type: 'pane', empty: true };
    if (info.hostId && hosts.some((h) => h.id === info.hostId)) return { type: 'pane', hostId: info.hostId };
    if (info.protocol === 'local') return { type: 'pane', local: { cwd: info.target.replace(/^[^·]+ · /, '') } };
    return null;
  }
  const kids: { node: SavedNode; size: number }[] = [];
  node.children.forEach((c, i) => {
    const s = saveNode(c, hosts, sessions);
    if (s) kids.push({ node: s, size: node.sizes[i] });
  });
  if (kids.length === 0) return null;
  if (kids.length === 1) return kids[0].node;
  return { type: 'split', dir: node.dir, sizes: renorm(kids.map((k) => k.size)), children: kids.map((k) => k.node) };
}

export function serializeLayout(): SavedLayout {
  const s = useWorkspace.getState();
  const st = useApp.getState().store;
  const hosts = st?.hosts ?? [];
  const tabs: SavedTab[] = [];
  let active = 0;
  let active1: number | undefined;
  for (const t of s.tabs) {
    const root = saveNode(t.root, hosts, s.sessions);
    if (!root) continue;
    if (t.id === s.activeByGroup[0]) active = tabs.length;
    if (t.group === 1 && t.id === s.activeByGroup[1]) active1 = tabs.length;
    tabs.push({ ...(t.title ? { title: t.title } : {}), ...(t.broadcast ? { broadcast: true } : {}), ...(t.group === 1 ? { group: 1 as const } : {}), root });
  }
  const ratio = s.groupRatio;
  return { tabs, active, ...(active1 !== undefined ? { active1 } : {}), ...(active1 !== undefined && ratio !== 50 ? { ratio } : {}) };
}

/** 저장된 노드를 실제 칸으로 만든다. 접속에 실패한 칸은 null. */
async function restoreNode(node: SavedNode, hosts: Host[], infos: SessionInfo[]): Promise<LayoutNode | null> {
  if (node.type === 'pane') {
    if (node.empty) return newPane(null);
    if (node.hostId) {
      if (!hosts.some((h) => h.id === node.hostId)) return null;
      const info = await openSessionFor(node.hostId);
      if (!info) return null;
      infos.push(info);
      return newPane(info.id);
    }
    if (node.local) {
      const info = await openSessionFor(null, { host: '', protocol: 'local', cwd: node.local.cwd || undefined });
      if (!info) return null;
      infos.push(info);
      return newPane(info.id);
    }
    return null;
  }
  const kids: { n: LayoutNode; size: number }[] = [];
  for (let i = 0; i < node.children.length; i++) {
    const n = await restoreNode(node.children[i], hosts, infos);
    if (n) kids.push({ n, size: node.sizes[i] });
  }
  if (kids.length === 0) return null;
  if (kids.length === 1) return kids[0].n;
  const split: SplitNode = { type: 'split', id: newId('s'), dir: node.dir, sizes: renorm(kids.map((k) => k.size)), children: kids.map((k) => k.n) };
  return split;
}

async function restore(store: Store): Promise<void> {
  const saved = (store as Store & { layout?: SavedLayout }).layout;
  if (!store.settings.restoreLayout || !saved || !saved.tabs?.length) return;
  const hosts = store.hosts;
  const tabs: Tab[] = [];
  const infos: SessionInfo[] = [];
  let activeIndex = 0;
  let activeIndex1: number | undefined;
  for (let i = 0; i < saved.tabs.length; i++) {
    const st = saved.tabs[i];
    const root = await restoreNode(st.root, hosts, infos);
    if (!root) continue;
    const ids = paneIds(root);
    if (i === saved.active) activeIndex = tabs.length;
    if (i === saved.active1) activeIndex1 = tabs.length;
    tabs.push({
      id: newId('t'),
      // 예전 빌드가 저장한 그리드 탭 이름("전체 보기")은 새 이름으로 바꿔 복원한다.
      title: st.title === '전체 보기' ? t('host.localTerminal') : st.title, // i18n-ok 옛 저장 이름
      broadcast: !!st.broadcast,
      group: st.group === 1 ? 1 : undefined,
      root,
      focusedPane: ids[0] ?? null,
      zoomed: null,
    });
  }
  if (tabs.length) replaceTabs(tabs, activeIndex, infos, activeIndex1, saved.ratio);
}

async function save(): Promise<void> {
  const store = useApp.getState().store;
  if (!store || !store.settings.restoreLayout || !restored) return;
  const layout = serializeLayout();
  const json = JSON.stringify(layout);
  if (json === lastJson) return;
  lastJson = json;
  await useApp.getState().patchStore({ layout } as Partial<Store>);
}

/** 앱 시작 시 한 번 호출. 볼트가 열리면 복원하고, 이후 레이아웃 변화를 저장한다. 구독 해제 함수를 돌려준다. */
export function startLayoutPersistence(): () => void {
  const tryRestore = () => {
    const st = useApp.getState();
    if (attempted || !st.booted || !st.vault?.unlocked || !st.store) return;
    attempted = true;
    void restore(st.store).finally(() => {
      restored = true;
      lastJson = JSON.stringify(serializeLayout());
    });
  };
  const offApp = useApp.subscribe(tryRestore);
  const offWs = useWorkspace.subscribe(() => {
    if (!restored) return;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => void save(), 300);
  });
  tryRestore();
  return () => {
    offApp();
    offWs();
    if (timer) clearTimeout(timer);
  };
}

