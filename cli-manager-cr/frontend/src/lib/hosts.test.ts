import { describe, expect, it } from 'vitest';
import { effectiveColor, recentHosts } from './hosts';
import type { Group, Host } from '../ipc/types';

const base = (over: Partial<Host>): Host => ({
  id: 'x', groupId: null, alias: 'a', protocol: 'ssh', host: 'h', port: 22, username: 'u', authType: 'password',
  tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 0, ...over,
});

describe('서버 색 우선순위 (spec/ui/sidebar.md 색 규칙, session.md §1, D105)', () => {
  const groups: Group[] = [
    { id: 'g1', parentId: null, name: '운영', env: 'prod', sort: 0 },
    { id: 'g2', parentId: 'g1', name: '하위', color: '#22c55e', sort: 0 },
    { id: 'g3', parentId: 'g4', name: '보라 아래 운영', env: 'prod', sort: 0 },
    { id: 'g4', parentId: null, name: '보라', color: '#8b5cf6', sort: 0 },
    { id: 'g5', parentId: 'g4', name: '빈 폴더', sort: 0 },
    { id: 'c1', parentId: 'c2', name: '순환1', sort: 0 },
    { id: 'c2', parentId: 'c1', name: '순환2', sort: 0 },
  ];
  it('서버 색이 가장 먼저', () => {
    expect(effectiveColor(base({ color: '#123456', groupId: 'g1', env: 'dev' }), groups)).toBe('#123456');
  });
  it('서버 환경색이 폴더 색보다 먼저 (none 은 지정 안 함)', () => {
    expect(effectiveColor(base({ env: 'dev', groupId: 'g4' }), groups)).toBe('#22c55e');
    expect(effectiveColor(base({ env: 'none', groupId: 'g4' }), groups)).toBe('#8b5cf6');
    expect(effectiveColor(base({ env: 'none' }), groups)).toBeUndefined();
  });
  it('폴더를 올라가며 처음 나오는 폴더 색/환경색', () => {
    expect(effectiveColor(base({ groupId: 'g2' }), groups)).toBe('#22c55e');
    expect(effectiveColor(base({ groupId: 'g1' }), groups)).toBe('#ef4444');
    // 가까운 폴더의 환경색이 먼 폴더의 색보다 먼저
    expect(effectiveColor(base({ groupId: 'g3' }), groups)).toBe('#ef4444');
    // 색 없는 폴더는 건너뛰고 상위 폴더 색
    expect(effectiveColor(base({ groupId: 'g5' }), groups)).toBe('#8b5cf6');
  });
  it('없으면 undefined (폴더 없음·없어진 폴더·순환)', () => {
    expect(effectiveColor(base({}), groups)).toBeUndefined();
    expect(effectiveColor(base({ groupId: 'gone' }), groups)).toBeUndefined();
    expect(effectiveColor(base({ groupId: 'c1' }), groups)).toBeUndefined();
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
