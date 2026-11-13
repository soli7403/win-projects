import { describe, it, expect } from 'vitest';
import { toggleSelection, rangeSelection } from './multiSelect';
import { moveHosts } from './tree';
import type { Host } from '../ipc/types';

describe('서버 여러 개 선택 (D59)', () => {
  it('Ctrl+클릭은 넣고 뺀다', () => {
    expect(toggleSelection(['a'], 'b')).toEqual(['a', 'b']);
    expect(toggleSelection(['a', 'b'], 'a')).toEqual(['b']);
  });

  it('Shift+클릭은 화면 순서의 범위(양방향)', () => {
    const order = ['a', 'b', 'c', 'd'];
    expect(rangeSelection(order, 'b', 'd')).toEqual(['b', 'c', 'd']);
    expect(rangeSelection(order, 'd', 'b')).toEqual(['b', 'c', 'd']);
    expect(rangeSelection(order, null, 'b')).toBeNull();
    expect(rangeSelection(order, 'x', 'b')).toBeNull();
  });

  it('moveHosts 는 고른 서버만 옮기고, 바뀔 게 없으면 같은 배열', () => {
    const h = (id: string, groupId: string | null) => ({ id, groupId }) as Host;
    const hosts = [h('1', null), h('2', 'x'), h('3', 'y')];
    const moved = moveHosts(hosts, ['1', '2'], 'y');
    expect(moved.map((x) => x.groupId)).toEqual(['y', 'y', 'y']);
    expect(moveHosts(hosts, ['3'], 'y')).toBe(hosts);
  });
});
