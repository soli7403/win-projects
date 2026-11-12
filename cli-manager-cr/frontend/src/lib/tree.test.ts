// 사이드바 트리 계산 테스트. 근거: spec/ui/sidebar.md §2~§5, spec/ui/host-editor.md §2.
import { describe, expect, it } from 'vitest';
import type { Group, Host } from '../ipc/types';
import { descendantGroupIds, groupPath, matchesQuery, sidebarSubtitle, validParentChoices, countHostsIn, fullTarget } from './tree';

const h = (over: Partial<Host>): Host => ({
  id: 'x', groupId: null, alias: 'a', protocol: 'ssh', host: 'h', port: 22, username: 'u', authType: 'password',
  tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 0, ...over,
});

const groups: Group[] = [
  { id: 'p', parentId: null, name: '운영', sort: 0 },
  { id: 'c', parentId: 'p', name: 'web', sort: 0 },
  { id: 'd', parentId: 'c', name: 'deep', sort: 0 },
  { id: 'o', parentId: null, name: '기타', sort: 1 },
];

describe('검색 (spec/ui/sidebar.md §2)', () => {
  it('모든 낱말이 포함되어야 일치, 폴더 이름도 검색 대상', () => {
    const host = h({ alias: 'WEB01', host: '10.0.2.14', tags: ['nginx'], groupId: 'c' });
    expect(matchesQuery(host, groups, 'web 10.0')).toBe(true);
    expect(matchesQuery(host, groups, 'web db')).toBe(false);
    expect(matchesQuery(host, groups, 'NGINX')).toBe(true);
    expect(matchesQuery(host, groups, '   ')).toBe(true);
  });
});

describe('폴더 계층 (spec/ui/sidebar.md §4)', () => {
  it('하위 폴더까지 포함한 개수', () => {
    const hs = [h({ id: '1', groupId: 'p' }), h({ id: '2', groupId: 'd' }), h({ id: '3', groupId: 'o' })];
    expect(countHostsIn(groups, hs, 'p')).toBe(2);
    expect([...descendantGroupIds(groups, 'p')].sort()).toEqual(['c', 'd', 'p']);
  });
  it('편집 중인 폴더의 하위 폴더는 상위 후보에서 뺀다', () => {
    expect(validParentChoices(groups, 'p').map((g) => g.id)).toEqual(['o']);
    expect(groupPath(groups, 'd')).toBe('운영 / web / deep');
  });
});

describe('사이드바 부제·접속 대상 (spec/ui/sidebar.md §5)', () => {
  it('로컬은 시작 폴더 또는 ~, 시리얼은 장치 경로', () => {
    expect(sidebarSubtitle(h({ protocol: 'local', local: {} }))).toBe('로컬 · ~');
    expect(sidebarSubtitle(h({ protocol: 'local', local: { cwd: 'C:\\w' } }))).toBe('로컬 · C:\\w');
    expect(sidebarSubtitle(h({ protocol: 'serial', serial: { path: 'COM3' } }))).toBe('COM3');
  });
  it('툴팁 대상은 기본 포트가 아닐 때만 :포트 를 붙인다', () => {
    expect(fullTarget(h({ port: 22 }))).toBe('u@h');
    expect(fullTarget(h({ port: 2222 }))).toBe('u@h:2222');
  });
});
