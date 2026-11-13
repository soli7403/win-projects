// 빠른 접속 검색 순위. 근거: spec/ui/dialogs.md §2.1 (d).
import { describe, expect, it } from 'vitest';
import { rankHosts } from './quick';
import type { Host } from '../ipc/types';

const h = (over: Partial<Host>): Host => ({
  id: 'x', groupId: null, alias: 'a', protocol: 'ssh', host: 'h', port: 22, username: '', authType: 'password',
  tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 0, ...over,
});

describe('빠른 연결 서버 검색', () => {
  const hosts = [
    h({ id: '1', alias: 'web-prod', host: '10.0.0.1' }),
    h({ id: '2', alias: 'db', host: 'web.internal' }),
    h({ id: '3', alias: 'zeta', host: '1.1.1.1', lastUsedAt: 5 }),
  ];
  it('이름이 낱말로 시작하는 것이 먼저, 모든 낱말이 맞아야 함', () => {
    expect(rankHosts(hosts, [], 'web').map((x) => x.id)).toEqual(['1', '2']);
    expect(rankHosts(hosts, [], 'web prod').map((x) => x.id)).toEqual(['1']);
    expect(rankHosts(hosts, [], 'nomatch')).toEqual([]);
  });
  it('입력이 비면 모두 같은 점수, 최근 사용 순', () => {
    expect(rankHosts(hosts, [], '').map((x) => x.id)[0]).toBe('3');
  });
});
