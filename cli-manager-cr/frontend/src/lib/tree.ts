// 사이드바 트리 계산(순수 함수). 근거: spec/ui/sidebar.md §2~§4, spec/ui/host-editor.md §2.
import type { Group, Host } from '../ipc/types';

/** 정렬: 정렬 키 오름차순, 같으면 이름 순. */
function bySortThenName<T extends { sort: number }>(nameOf: (x: T) => string) {
  return (a: T, b: T) => a.sort - b.sort || nameOf(a).localeCompare(nameOf(b), 'ko');
}

export const sortGroups = (gs: Group[]) => [...gs].sort(bySortThenName<Group>((g) => g.name));
export const sortHosts = (hs: Host[]) => [...hs].sort(bySortThenName<Host>((h) => h.alias));

/** 검색 대상 글자: 이름·주소·사용자명·태그·폴더 이름·메모를 이어 붙인다(소문자). */
export function searchHaystack(h: Host, groups: Group[]): string {
  const folder = groups.find((g) => g.id === h.groupId)?.name ?? '';
  return [h.alias, h.host, h.username, h.tags.join(' '), folder, h.notes ?? ''].join(' ').toLowerCase();
}

/** 공백으로 나눈 모든 낱말이 포함되어야 일치. 검색어가 비면 전부 일치. */
export function matchesQuery(h: Host, groups: Group[], query: string): boolean {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) return true;
  const hay = searchHaystack(h, groups);
  return words.every((w) => hay.includes(w));
}

/** 폴더 아래(하위 폴더 포함) 모든 폴더 id. 자기 자신 포함. */
export function descendantGroupIds(groups: Group[], rootId: string): Set<string> {
  const out = new Set<string>([rootId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const g of groups) {
      if (g.parentId && out.has(g.parentId) && !out.has(g.id)) {
        out.add(g.id);
        grew = true;
      }
    }
  }
  return out;
}

/** 서버가 속한 폴더가 실제로 있는지(없으면 최상위 취급). */
export function effectiveGroupId(h: Host, groups: Group[]): string | null {
  return h.groupId && groups.some((g) => g.id === h.groupId) ? h.groupId : null;
}

/** 폴더 안(하위 폴더 포함)의 서버 수. filter 가 있으면 일치하는 것만 센다. */
export function countHostsIn(groups: Group[], hosts: Host[], groupId: string, filter: (h: Host) => boolean = () => true): number {
  const ids = descendantGroupIds(groups, groupId);
  return hosts.filter((h) => h.groupId && ids.has(h.groupId) && filter(h)).length;
}

/** 사이드바 행의 부제(접속 대상). 시리얼=장치 경로, 로컬=`로컬 · 시작 폴더 또는 ~`, 그 외=`[사용자@]주소`. */
export function sidebarSubtitle(h: Host): string {
  if (h.protocol === 'local') return `로컬 · ${h.local?.cwd || '~'}`;
  if (h.protocol === 'serial') return (h.serial as { path?: string } | undefined)?.path ?? h.host;
  // 부제는 포트를 붙이지 않는다(§5). 포트는 툴팁에서만 보인다.
  return `${h.username ? h.username + '@' : ''}${h.host}`;
}

/** 대상 문자열에 포트를 붙인 형태(툴팁·상세). 기본 포트는 생략. */
export function fullTarget(h: Host): string {
  const def = h.protocol === 'telnet' ? 23 : 22;
  const port = h.protocol === 'ssh' || h.protocol === 'telnet' ? (h.port && h.port !== def ? `:${h.port}` : '') : '';
  return `${h.username ? h.username + '@' : ''}${h.host}${port}`;
}

/** 폴더 편집에서 상위 폴더 후보가 될 수 있는지(자기 자신과 하위 폴더 제외). */
export function validParentChoices(groups: Group[], editingId: string | null): Group[] {
  if (!editingId) return sortGroups(groups);
  const banned = descendantGroupIds(groups, editingId);
  return sortGroups(groups).filter((g) => !banned.has(g.id));
}

/** `상위 / 하위` 경로 문자열. */
export function groupPath(groups: Group[], id: string | null): string {
  const parts: string[] = [];
  let cur = id ? groups.find((g) => g.id === id) : undefined;
  while (cur) {
    parts.unshift(cur.name);
    cur = cur.parentId ? groups.find((g) => g.id === cur!.parentId) : undefined;
  }
  return parts.join(' / ');
}

/** 끌어 놓기 결과(§9): 폴더를 자기 자신이나 하위 폴더 안으로 옮기면 null(무시). */
export function canDropGroup(groups: Group[], dragGroupId: string, targetGroupId: string | null): boolean {
  if (targetGroupId === null) return true;
  return !descendantGroupIds(groups, dragGroupId).has(targetGroupId);
}

/** 서버를 폴더(또는 최상위)로 옮긴 결과. 이미 그 폴더에 있으면 같은 배열을 돌려준다. */
export function moveHost(hosts: Host[], hostId: string, targetGroupId: string | null): Host[] {
  const h = hosts.find((x) => x.id === hostId);
  if (!h || (h.groupId ?? null) === targetGroupId) return hosts;
  return hosts.map((x) => (x.id === hostId ? { ...x, groupId: targetGroupId } : x));
}

/** 폴더 삭제 계획(§8, B7): 하위 폴더까지 지우고, 안의 서버는 삭제된 폴더의 상위 폴더로 옮긴다. */
export function folderDeletePlan(groups: Group[], hosts: Host[], folderId: string): { groups: Group[]; hosts: Host[] } {
  const doomed = descendantGroupIds(groups, folderId);
  const target = groups.find((g) => g.id === folderId)?.parentId ?? null;
  return {
    groups: groups.filter((g) => !doomed.has(g.id)),
    hosts: hosts.map((h) => (h.groupId && doomed.has(h.groupId) ? { ...h, groupId: target } : h)),
  };
}

/** 서버 복제(§7, B8): 새 id, 이름에 " (복사본)", 최근 사용 기록 지움. */
export function duplicateHostRecord(h: Host, newId: string, sort: number): Host {
  return { ...h, id: newId, alias: `${h.alias} (복사본)`, lastUsedAt: undefined, sort };
}

/** 사이드바 너비 범위(§1.3): 200~520px. */
export const clampSidebarWidth = (w: number) => Math.max(200, Math.min(520, w));

/** 서버별 연결된 세션 수(§5 초록 점): 상태가 connected 인 세션만 센다. 접속 중·재접속·끊김·오류는 제외. */
export function connectedCountByHost(sessions: Record<string, { info: { hostId: string | null; state: string } }>): Record<string, number> {
  const out: Record<string, number> = {};
  for (const e of Object.values(sessions)) {
    const hid = e.info.hostId;
    if (hid && e.info.state === 'connected') out[hid] = (out[hid] ?? 0) + 1;
  }
  return out;
}
