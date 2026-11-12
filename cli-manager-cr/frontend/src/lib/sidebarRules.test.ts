// 수용 시나리오 B1·B2·B3·B5·B6·B7·B8·B10 의 규칙 부분. 근거: spec/acceptance/02-shell-sidebar.md, spec/ui/sidebar.md.
import { describe, expect, it } from 'vitest';
import type { Group, Host } from '../ipc/types';
import {
  canDropGroup, clampSidebarWidth, connectedCountByHost, duplicateHostRecord, folderDeletePlan, matchesQuery, moveHost, sidebarSubtitle,
} from './tree';

const g = (id: string, parentId: string | null, name: string, sort = 0): Group => ({ id, parentId, name, sort });
const h = (id: string, alias: string, groupId: string | null, extra: Partial<Host> = {}): Host => ({
  id, alias, groupId, protocol: 'ssh', host: `${alias}.example`, port: 22, username: 'root',
  authType: 'password', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30,
  forwards: [], sort: 0, ...extra,
});

const groups: Group[] = [g('a', null, '운영'), g('b', 'a', '웹'), g('c', null, '개발')];

describe('사이드바 규칙', () => {
  it('B1: 공백으로 나눈 모든 낱말이 맞아야 일치', () => {
    const x = h('1', 'web01', 'a', { tags: ['nginx'] });
    expect(matchesQuery(x, groups, 'web 10')).toBe(false);
    expect(matchesQuery(x, groups, 'web nginx')).toBe(true);
    expect(matchesQuery(x, groups, '운영')).toBe(true);
  });

  it('B2: 일치 없음은 모든 낱말 규칙의 결과로 나온다', () => {
    expect(matchesQuery(h('1', 'db', null), groups, 'zzz')).toBe(false);
  });

  it('부제는 [사용자@]주소 이고 포트를 붙이지 않는다', () => {
    expect(sidebarSubtitle(h('1', 'x', null, { port: 2200 }))).toBe('root@x.example');
  });

  it('B5: 서버를 폴더로 옮긴다. 이미 그 폴더면 같은 배열', () => {
    const hosts = [h('1', 'web', null)];
    expect(moveHost(hosts, '1', 'c')[0].groupId).toBe('c');
    expect(moveHost(hosts, '1', null)).toBe(hosts);
  });

  it('B6: 폴더를 자기 하위 폴더로 옮기면 무시', () => {
    expect(canDropGroup(groups, 'a', 'b')).toBe(false);
    expect(canDropGroup(groups, 'a', 'a')).toBe(false);
    expect(canDropGroup(groups, 'a', 'c')).toBe(true);
    expect(canDropGroup(groups, 'b', null)).toBe(true);
  });

  it('B7: 폴더 삭제는 하위 폴더까지 지우고 안의 서버는 상위 폴더로 옮긴다', () => {
    const top = g('p', null, '상위');
    const mid = g('m', 'p', '중간');
    const deep = g('d', 'm', '깊이');
    const hosts = [h('1', 'a', 'm'), h('2', 'b', 'd'), h('3', 'c', 'p')];
    const plan = folderDeletePlan([top, mid, deep], hosts, 'm');
    expect(plan.groups.map((x) => x.id)).toEqual(['p']);
    expect(plan.hosts.find((x) => x.id === '1')?.groupId).toBe('p');
    expect(plan.hosts.find((x) => x.id === '2')?.groupId).toBe('p');
    expect(plan.hosts.find((x) => x.id === '3')?.groupId).toBe('p');
    const rootPlan = folderDeletePlan([top], [h('4', 'z', 'p')], 'p');
    expect(rootPlan.hosts[0].groupId).toBeNull();
  });

  it('B8: 복제는 새 id, " (복사본)", 최근 사용 기록 없음', () => {
    const src = h('1', 'web', null, { lastUsedAt: 123 });
    const copy = duplicateHostRecord(src, 'new', 9);
    expect(copy.id).toBe('new');
    expect(copy.alias).toBe('web (복사본)');
    expect(copy.lastUsedAt).toBeUndefined();
  });

  it('B10: 사이드바 너비는 200~520 으로 제한', () => {
    expect(clampSidebarWidth(800)).toBe(520);
    expect(clampSidebarWidth(100)).toBe(200);
    expect(clampSidebarWidth(300)).toBe(300);
  });
});

describe('서버 행 연결 점 규칙(§5)', () => {
  it('connected 인 세션만 센다: 접속 중·재접속·닫힘·오류는 제외', () => {
    const mk = (hostId: string, state: string) => ({ info: { hostId, state } });
    const counts = connectedCountByHost({
      a: mk('h1', 'connected'),
      b: mk('h1', 'connected'),
      c: mk('h1', 'error'),
      d: mk('h2', 'connecting'),
      e: mk('h2', 'reconnecting'),
      f: mk('h3', 'closed'),
      g: mk('h4', 'connected'),
      h: { info: { hostId: null, state: 'connected' } },
    });
    expect(counts).toEqual({ h1: 2, h4: 1 });
  });
});
