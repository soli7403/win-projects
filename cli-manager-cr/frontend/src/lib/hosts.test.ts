import { describe, expect, it } from 'vitest';
import { effectiveColor, recentHosts } from './hosts';
import type { Group, Host } from '../ipc/types';

const base = (over: Partial<Host>): Host => ({
  id: 'x', groupId: null, alias: 'a', protocol: 'ssh', host: 'h', port: 22, username: 'u', authType: 'password',
  tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 0, ...over,
});

describe('서버 색 우선순위 (spec/ui/shell.md §4)', () => {
  const groups: Group[] = [
    { id: 'g1', parentId: null, name: '운영', env: 'prod', sort: 0 },
    { id: 'g2', parentId: 'g1', name: '하위', color: '#22c55e', sort: 0 },
  ];
  it('서버 색이 가장 먼저', () => {
    expect(effectiveColor(base({ color: '#123456', groupId: 'g1' }), groups)).toBe('#123456');
  });
  it('폴더 색(상위 방향 탐색) 다음, 그다음 환경 색', () => {
    expect(effectiveColor(base({ groupId: 'g2' }), groups)).toBe('#22c55e');
    expect(effectiveColor(base({ groupId: 'g1' }), groups)).toBe('#ef4444');
    expect(effectiveColor(base({}), groups)).toBeUndefined();
  });
});

describe('최근 서버 (spec/ui/shell.md §4)', () => {
  it('lastUsedAt 내림차순, 같으면 이름 오름차순, 최대 9개', () => {
    const hs = [
      base({ id: '1', alias: 'b', lastUsedAt: 5 }),
      base({ id: '2', alias: 'a', lastUsedAt: 5 }),
      base({ id: '3', alias: 'c', lastUsedAt: 9 }),
    ];
    expect(recentHosts(hs).map((h) => h.id)).toEqual(['3', '2', '1']);
    const many = Array.from({ length: 12 }, (_, i) => base({ id: String(i), alias: `n${i}`, lastUsedAt: i }));
    expect(recentHosts(many)).toHaveLength(9);
  });
});
