// 탭 안 칸·분할 트리(순수 함수). 근거: spec/ui/panes.md §1, §4~§7.
// 노드는 칸(pane) 또는 분할(split). 분할 크기는 % 이며 합은 100.

export type Dir = 'row' | 'col';

export interface PaneNode {
  type: 'pane';
  id: string;
  /** 세션 id. 빈 칸이면 null. */
  sessionId: string | null;
}

export interface SplitNode {
  type: 'split';
  id: string;
  dir: Dir;
  sizes: number[];
  children: LayoutNode[];
}

export type LayoutNode = PaneNode | SplitNode;

export const MIN_PCT = 8;
let seq = 0;
export const newId = (p: string) => `${p}-${Date.now().toString(36)}-${(++seq).toString(36)}`;

export const newPane = (sessionId: string | null): PaneNode => ({ type: 'pane', id: newId('p'), sessionId });

export function equalSizes(n: number): number[] {
  return Array.from({ length: n }, () => 100 / n);
}

export function paneIds(node: LayoutNode): string[] {
  return node.type === 'pane' ? [node.id] : node.children.flatMap(paneIds);
}

export function findPane(node: LayoutNode, id: string): PaneNode | undefined {
  if (node.type === 'pane') return node.id === id ? node : undefined;
  for (const c of node.children) {
    const f = findPane(c, id);
    if (f) return f;
  }
  return undefined;
}

export function sessionsIn(node: LayoutNode): string[] {
  return node.type === 'pane'
    ? node.sessionId ? [node.sessionId] : []
    : node.children.flatMap(sessionsIn);
}

export function findPaneBySession(node: LayoutNode, sessionId: string): PaneNode | undefined {
  if (node.type === 'pane') return node.sessionId === sessionId ? node : undefined;
  for (const c of node.children) {
    const f = findPaneBySession(c, sessionId);
    if (f) return f;
  }
  return undefined;
}

/** 칸 교체(대상 id 를 변환 함수 결과로 바꾼다). */
function mapPane(node: LayoutNode, id: string, fn: (p: PaneNode) => LayoutNode): LayoutNode {
  if (node.type === 'pane') return node.id === id ? fn(node) : node;
  return { ...node, children: node.children.map((c) => mapPane(c, id, fn)) };
}

/**
 * 칸을 분할한다. 대상 칸이 같은 방향 분할의 직접 자식이면 형제로 끼우고 모든 자식을 균등하게 나눈다.
 * 아니면 대상 칸 자리를 [기존, 새] (또는 before 면 [새, 기존]) 분할로 바꾼다(50/50).
 */
export function splitPane(root: LayoutNode, targetId: string, dir: Dir, added: PaneNode, before = false): LayoutNode {
  // 부모 분할 탐색
  const tryInsertSibling = (node: LayoutNode): LayoutNode | null => {
    if (node.type === 'pane') return null;
    const idx = node.children.findIndex((c) => c.type === 'pane' && c.id === targetId);
    if (idx >= 0 && node.dir === dir) {
      const children = [...node.children];
      children.splice(before ? idx : idx + 1, 0, added);
      return { ...node, children, sizes: equalSizes(children.length) };
    }
    for (let i = 0; i < node.children.length; i++) {
      const r = tryInsertSibling(node.children[i]);
      if (r) {
        const children = [...node.children];
        children[i] = r;
        return { ...node, children };
      }
    }
    return null;
  };
  const sib = tryInsertSibling(root);
  if (sib) return sib;
  return mapPane(root, targetId, (target) => ({
    type: 'split',
    id: newId('s'),
    dir,
    sizes: [50, 50],
    children: before ? [added, target] : [target, added],
  }));
}

/**
 * 칸 제거. 분할에 자식이 하나 남으면 그 자식이 분할을 대신하고, 남은 자식 크기는 합 100 으로 비례 재조정.
 * 트리가 비면 null.
 */
export function removePane(root: LayoutNode, id: string): LayoutNode | null {
  if (root.type === 'pane') return root.id === id ? null : root;
  const children: LayoutNode[] = [];
  const sizes: number[] = [];
  let changed = false;
  root.children.forEach((c, i) => {
    const r = c.type === 'pane' && c.id === id ? null : removePane(c, id);
    if (r !== c) changed = true;
    if (r) {
      children.push(r);
      sizes.push(root.sizes[i]);
    }
  });
  if (!changed) return root;
  if (children.length === 0) return null;
  if (children.length === 1) return children[0];
  const total = sizes.reduce((x, y) => x + y, 0) || 1;
  return { ...root, children, sizes: sizes.map((x) => (x * 100) / total) };
}

/** 분할선 i (자식 i 와 i+1 사이)를 delta(%) 만큼 옮긴다. 각 자식은 MIN_PCT 이상. */
export function resizeSplit(root: LayoutNode, splitId: string, boundary: number, deltaPct: number): LayoutNode {
  if (root.type === 'pane') return root;
  if (root.id === splitId) {
    const sizes = [...root.sizes];
    const a = sizes[boundary];
    const b = sizes[boundary + 1];
    if (a === undefined || b === undefined) return root;
    let d = deltaPct;
    if (a + d < MIN_PCT) d = MIN_PCT - a;
    if (b - d < MIN_PCT) d = b - MIN_PCT;
    sizes[boundary] = a + d;
    sizes[boundary + 1] = b - d;
    return { ...root, sizes };
  }
  return { ...root, children: root.children.map((c) => resizeSplit(c, splitId, boundary, deltaPct)) };
}

/** 두 칸의 세션을 맞바꾼다(구조 불변). */
export function swapSessions(root: LayoutNode, a: string, b: string): LayoutNode {
  const pa = findPane(root, a);
  const pb = findPane(root, b);
  if (!pa || !pb) return root;
  const sa = pa.sessionId;
  const sb = pb.sessionId;
  return mapPane(mapPane(root, a, (p) => ({ ...p, sessionId: sb })), b, (p) => ({ ...p, sessionId: sa }));
}

/**
 * 칸을 떼어 대상 칸의 가장자리(dir, before)에 붙인다. 떼어낸 뒤 구조를 다시 계산한다.
 * 떼는 칸은 원래 노드를 그대로 옮기므로 id 가 유지된다.
 */
export function movePaneToEdge(root: LayoutNode, movingId: string, targetId: string, dir: Dir, before: boolean): LayoutNode {
  if (movingId === targetId) return root;
  const moving = findPane(root, movingId);
  if (!moving) return root;
  const without = removePane(root, movingId);
  if (!without || !findPane(without, targetId)) return root;
  return splitPane(without, targetId, dir, moving, before);
}

export type Edge = 'left' | 'right' | 'top' | 'bottom' | 'center';

/**
 * 놓는 구역 판정: 가장자리까지의 비율이 22% 이하이면 가장 가까운 가장자리, 아니면 가운데.
 * x, y 는 칸 안의 상대 좌표(0~1).
 */
export function edgeZone(x: number, y: number): Edge {
  const dl = x, dr = 1 - x, dt = y, db = 1 - y;
  const m = Math.min(dl, dr, dt, db);
  if (m > 0.22) return 'center';
  if (m === dl) return 'left';
  if (m === dr) return 'right';
  if (m === dt) return 'top';
  return 'bottom';
}

/** 가장자리 → (분할 방향, 새 칸이 앞인지). 가운데는 null. */
export function edgeToSplit(e: Edge): { dir: Dir; before: boolean } | null {
  switch (e) {
    case 'left': return { dir: 'row', before: true };
    case 'right': return { dir: 'row', before: false };
    case 'top': return { dir: 'col', before: true };
    case 'bottom': return { dir: 'col', before: false };
    default: return null;
  }
}
