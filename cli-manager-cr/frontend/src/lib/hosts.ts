// 서버 표시 보조 함수. 근거: spec/data/store.md §2·§3·§6, spec/ui/shell.md §4, spec/ui/dialogs.md §2.1.
import type { Group, Host } from '../ipc/types';
import { ENV_INFO } from './defaults';

/** 폴더를 따라 올라가는 최대 단계 (순환하는 데이터에서도 멈춘다). */
const GROUP_DEPTH_MAX = 32;

/** 환경의 색. none(또는 없음)은 색을 주지 않는다. */
function envColor(env: Host['env']): string | undefined {
  return env && env !== 'none' ? ENV_INFO[env]?.color : undefined;
}

/**
 * 서버의 색 (spec/ui/sidebar.md "색 규칙", spec/behavior/session.md §1 color — 백엔드 resolveLook 과 같은 순서).
 * 서버 색 → 서버 환경색 → 소속 폴더부터 위로 올라가며 처음 나오는 (폴더 색 → 폴더 환경색).
 * 없으면 undefined(호출한 쪽이 기본 테두리색 사용).
 */
export function effectiveColor(host: Host, groups: Group[]): string | undefined {
  const own = host.color || envColor(host.env);
  if (own) return own;
  let gid = host.groupId;
  for (let depth = 0; gid && depth < GROUP_DEPTH_MAX; depth++) {
    const g = groups.find((x) => x.id === gid);
    if (!g) break;
    const c = g.color || envColor(g.env);
    if (c) return c;
    gid = g.parentId;
  }
  return undefined;
}

/** 대상 문자열: `user@host:port`. */
export function targetString(host: Host): string {
  // 로컬·시리얼의 대상 표기는 명세에 없다(spec/QUESTIONS.md Q2). 임시로 주소만 쓴다.
  if (host.protocol === 'local') return '';
  if (host.protocol === 'serial') return host.host;
  return `${host.username ? host.username + '@' : ''}${host.host}:${host.port}`;
}

/** 최근 서버 격자: lastUsedAt 내림차순, 같으면 이름 오름차순, 최대 9개. */
export function recentHosts(hosts: Host[], limit = 9): Host[] {
  return [...hosts]
    .sort((a, b) => (b.lastUsedAt ?? 0) - (a.lastUsedAt ?? 0) || a.alias.localeCompare(b.alias, 'ko'))
    .slice(0, limit);
}
