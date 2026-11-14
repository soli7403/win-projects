// 같은 대상으로 하나 더 열기: 세션 정보에서 열기 요청을 만든다. 근거: spec/ui/panes.md §10.
import type { SessionInfo } from '../ipc/types';

export type OpenRequest =
  | { kind: 'host'; hostId: string }
  | { kind: 'adhoc'; adhoc: { host: string; protocol: 'local' | 'ssh' | 'telnet'; cwd?: string; port?: number; username?: string } };

/**
 * 세션에서 "같은 곳으로" 여는 요청을 만든다.
 * - 저장된 서버 → 같은 서버(hostId)
 * - 로컬 세션 → 같은 시작 폴더(대상 `로컬 · 폴더` 에서 접두를 뗀 값, `~` 이면 생략)
 * - 즉석 SSH/텔넷 → 대상 `[사용자@]호스트:포트` 를 다시 해석
 * 해석할 수 없으면 null(아무 동작 없음).
 */
export function sameTargetRequest(info: SessionInfo): OpenRequest | null {
  if (info.hostId) return { kind: 'host', hostId: info.hostId };
  if (info.protocol === 'local') {
    const cwd = info.target.replace(/^[^·]+ · /, '').trim(); // 접두(언어마다 다름) 떼기
    return { kind: 'adhoc', adhoc: { host: '', protocol: 'local', ...(cwd && cwd !== '~' ? { cwd } : {}) } };
  }
  if (info.protocol === 'ssh' || info.protocol === 'telnet') {
    const m = /^(?:(.+)@)?([^@:\s]+)(?::(\d{1,5}))?$/.exec(info.target.trim());
    if (!m) return null;
    const [, username, host, port] = m;
    return {
      kind: 'adhoc',
      adhoc: { host, protocol: info.protocol, ...(username ? { username } : {}), ...(port ? { port: Number(port) } : {}) },
    };
  }
  return null;
}
