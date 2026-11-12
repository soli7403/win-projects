// 칸·분할 규칙 테스트. 근거: spec/ui/panes.md §3~§7.
import { describe, expect, it } from 'vitest';
import {
  edgeToSplit, edgeZone, findPaneBySession, gridLayout, MIN_PCT, movePaneToEdge, newPane, paneIds,
  removePane, resizeSplit, splitPane, swapSessions, type LayoutNode, type SplitNode,
} from './layout';

const sum = (xs: number[]) => xs.reduce((a, b) => a + b, 0);

describe('분할 (panes.md §4)', () => {
  it('같은 방향 분할의 직접 자식이면 형제로 끼우고 균등 분할', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    let root: LayoutNode = splitPane(a, a.id, 'row', b);
    const c = newPane('s3');
    root = splitPane(root, b.id, 'row', c);
    const split = root as SplitNode;
    expect(split.children).toHaveLength(3);
    expect(split.sizes.every((s) => Math.abs(s - 100 / 3) < 1e-9)).toBe(true);
  });

  it('다른 방향이면 대상 칸 자리를 50/50 분할로 바꾼다(before 지정 가능)', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    const root = splitPane(a, a.id, 'col', b, true) as SplitNode;
    expect(root.dir).toBe('col');
    expect(root.sizes).toEqual([50, 50]);
    expect((root.children[0] as { sessionId: string }).sessionId).toBe('s2');
  });
});

describe('칸 제거 (panes.md §4 칸 제거, §5.1)', () => {
  it('자식이 하나 남으면 그 자식이 분할을 대신하고, 남은 크기는 합 100', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    const c = newPane('s3');
    let root: LayoutNode = splitPane(a, a.id, 'row', b);
    root = splitPane(root, b.id, 'row', c);
    root = removePane(root, b.id)!;
    expect(root.type).toBe('split');
    expect(sum((root as SplitNode).sizes)).toBeCloseTo(100);
    root = removePane(root, a.id)!;
    expect(root.type).toBe('pane');
    expect((root as { id: string }).id).toBe(c.id);
  });

  it('마지막 칸을 지우면 null(탭 닫힘)', () => {
    expect(removePane(newPane('s1'), 'nope')).not.toBeNull();
    const only = newPane('s1');
    expect(removePane(only, only.id)).toBeNull();
  });
});

describe('크기 조절 (panes.md §4 크기 조절)', () => {
  it('분할선을 끌면 양옆만 바뀌고 각 칸은 최소 8%', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    const root = splitPane(a, a.id, 'row', b) as SplitNode;
    const moved = resizeSplit(root, root.id, 0, 90) as SplitNode;
    expect(moved.sizes[1]).toBe(MIN_PCT);
    expect(moved.sizes[0]).toBe(100 - MIN_PCT);
    const back = resizeSplit(root, root.id, 0, -10) as SplitNode;
    expect(back.sizes[0]).toBeCloseTo(40);
  });
});

describe('세션 맞바꿈·칸 이동 (panes.md §7)', () => {
  it('가운데 놓기는 세션만 맞바꾼다(구조 불변)', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    const root = splitPane(a, a.id, 'row', b);
    const sw = swapSessions(root, a.id, b.id);
    expect(findPaneBySession(sw, 's2')?.id).toBe(a.id);
    expect(paneIds(sw)).toEqual(paneIds(root));
  });

  it('가장자리 놓기는 떼어 대상 옆에 붙인다', () => {
    const a = newPane('s1');
    const b = newPane('s2');
    const c = newPane('s3');
    let root: LayoutNode = splitPane(a, a.id, 'row', b);
    root = movePaneToEdge(root, b.id, a.id, 'col', false);
    expect(root.type).toBe('split');
    const sp = root as SplitNode;
    expect(sp.dir).toBe('col');
    expect(paneIds(root)).toEqual([a.id, b.id]);
    // 같은 탭의 칸 자신 위에는 무시
    expect(movePaneToEdge(root, a.id, a.id, 'row', true)).toBe(root);
    void c;
  });

  it('구역 판정: 가장자리 22% 이하면 그 가장자리, 아니면 가운데', () => {
    expect(edgeZone(0.5, 0.5)).toBe('center');
    expect(edgeZone(0.1, 0.5)).toBe('left');
    expect(edgeZone(0.9, 0.5)).toBe('right');
    expect(edgeZone(0.5, 0.05)).toBe('top');
    expect(edgeZone(0.5, 0.95)).toBe('bottom');
    expect(edgeToSplit('left')).toEqual({ dir: 'row', before: true });
    expect(edgeToSplit('center')).toBeNull();
  });
});

describe('그리드 모으기 (panes.md §6)', () => {
  it('열 수는 ceil(sqrt(n)), 칸 n개가 모두 남는다', () => {
    for (const n of [2, 3, 4, 5, 9]) {
      const panes = Array.from({ length: n }, (_, i) => newPane(`s${i}`));
      const root = gridLayout(panes);
      expect(paneIds(root)).toHaveLength(n);
    }
    const four = gridLayout(Array.from({ length: 4 }, (_, i) => newPane(`s${i}`))) as SplitNode;
    expect(four.dir).toBe('col');
    expect((four.children[0] as SplitNode).children).toHaveLength(2);
  });
});
