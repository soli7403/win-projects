// 같은 대상으로 열기 규칙. 근거: spec/ui/panes.md §10.
import { describe, expect, it } from 'vitest';
import { sameTargetRequest } from './sameTarget';
import type { SessionInfo } from '../ipc/types';

const base = (over: Partial<SessionInfo>): SessionInfo => ({
  id: 's', hostId: null, title: 't', protocol: 'ssh', target: '', state: 'connected', canSftp: true, ...over,
});

describe('같은 대상으로 열기 (panes.md §10)', () => {
  it('저장된 서버는 hostId 로', () => {
    expect(sameTargetRequest(base({ hostId: 'h1' }))).toEqual({ kind: 'host', hostId: 'h1' });
  });
  it('로컬 세션은 같은 시작 폴더, ~ 이면 생략', () => {
    expect(sameTargetRequest(base({ protocol: 'local', target: '로컬 · C:\work' }))).toEqual({
      kind: 'adhoc', adhoc: { host: '', protocol: 'local', cwd: 'C:\work' },
    });
    expect(sameTargetRequest(base({ protocol: 'local', target: '로컬 · ~' }))).toEqual({
      kind: 'adhoc', adhoc: { host: '', protocol: 'local' },
    });
  });
  it('즉석 SSH 는 사용자·호스트·포트를 다시 해석', () => {
    expect(sameTargetRequest(base({ target: 'root@10.0.0.5:2222' }))).toEqual({
      kind: 'adhoc', adhoc: { host: '10.0.0.5', protocol: 'ssh', username: 'root', port: 2222 },
    });
    expect(sameTargetRequest(base({ protocol: 'telnet', target: 'sw1:23' }))).toEqual({
      kind: 'adhoc', adhoc: { host: 'sw1', protocol: 'telnet', port: 23 },
    });
  });
  it('해석할 수 없으면 null', () => {
    expect(sameTargetRequest(base({ target: '' }))).toBeNull();
  });
});
