// 서버 표시 보조 함수. 근거: spec/data/store.md §2·§3·§6, spec/ui/shell.md §4, spec/ui/dialogs.md §2.1.
import type { Group, Host } from '../ipc/types';
import { ENV_INFO } from './defaults';

/** 서버 색 → 폴더 색 → 환경 색 순서로 고른다. 없으면 undefined(호출한 쪽이 기본 테두리색 사용). */
export function effectiveColor(host: Host, groups: Group[]): string | undefined {
  if (host.color) return host.color;
  const chain: Group[] = [];
  let gid = host.groupId;
  while (gid) {
    const g = groups.find((x) => x.id === gid);
    if (!g) break;
    chain.push(g);
    gid = g.parentId;
  }
  const folderColor = chain.find((g) => g.color)?.color;
  if (folderColor) return folderColor;
  const env = host.env ?? chain.find((g) => g.env)?.env;
  return env ? ENV_INFO[env].color : undefined;
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
