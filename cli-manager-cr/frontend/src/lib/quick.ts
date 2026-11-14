// 빠른 접속 입력 해석. 근거: spec/ui/dialogs.md §2.1.

export interface DirectTarget {
  protocol: 'ssh' | 'telnet';
  host: string;
  port: number;
  username?: string;
}

/**
 * 직접 접속 주소처럼 보이는지 판정하고 해석한다.
 * `[ssh://|telnet://][사용자@]호스트[:포트]` 이면서 @ 가 있거나, IPv4 모양이거나, 점(.)이 있거나,
 * ssh/telnet 으로 시작할 때만 해당. 포트 생략 시 기본 포트(ssh 22, telnet 23).
 */
export function parseDirect(input: string): DirectTarget | null {
  const raw = input.trim();
  if (!raw) return null;
  const m = /^(?:(ssh|telnet):\/\/)?(?:([^@\s/]+)@)?(\[[0-9a-fA-F:]+\]|[^:@\s/]+)(?::(\d{1,5}))?$/.exec(raw);
  if (!m) return null;
  const [, scheme, user, hostRaw, portRaw] = m;
  const isIpv4 = /^\d{1,3}(\.\d{1,3}){3}$/.test(hostRaw);
  const looksDirect = !!user || isIpv4 || hostRaw.includes('.') || /^(ssh|telnet)/i.test(raw);
  if (!looksDirect) return null;
  const protocol: 'ssh' | 'telnet' = scheme?.toLowerCase() === 'telnet' || /^telnet/i.test(raw) ? 'telnet' : 'ssh';
  const def = protocol === 'telnet' ? 23 : 22;
  const port = portRaw ? Number(portRaw) : def;
  if (port < 1 || port > 65535) return null;
  return { protocol, host: hostRaw.replace(/^\[|\]$/g, ''), port, username: user };
}

/** 로컬 터미널 항목을 보여 줄 입력인지: "로컬", "local", "터미널", "term", "shell", "셸" 로 시작. */
export function isLocalPrefix(input: string): boolean {
  const s = input.trim().toLowerCase();
  if (!s) return false;
  return ['로컬', 'local', '터미널', 'term', 'shell', '셸'].some((p) => s.startsWith(p)); // i18n-ok 입력 별칭 // i18n-ok 입력 별칭
}

import type { Group, Host } from '../ipc/types';

/** 저장된 서버 검색(dialogs.md §2.1 (d)). 낱말마다 점수를 더하고, 하나라도 0 이면 제외. 최대 50개. */
export function rankHosts(hosts: Host[], groups: Group[], query: string): Host[] {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  const folderName = (h: Host) => groups.find((g) => g.id === h.groupId)?.name ?? '';
  const scoreWord = (h: Host, w: string): number => {
    const name = h.alias.toLowerCase();
    if (name.split(/[\s\-_.]+/).some((p) => p.startsWith(w)) || name.startsWith(w)) return 6;
    if (name.includes(w)) return 5;
    if (h.host.toLowerCase().includes(w)) return 4;
    if (folderName(h).toLowerCase().includes(w) || h.tags.some((t) => t.toLowerCase().includes(w))) return 3;
    if ((h.username ?? '').toLowerCase().includes(w)) return 2;
    // 이름에 글자가 순서대로 흩어져 포함(퍼지)
    let i = 0;
    for (const ch of name) if (i < w.length && ch === w[i]) i++;
    return i === w.length ? 1 : 0;
  };
  const scored = hosts
    .map((h) => {
      if (words.length === 0) return { h, s: 0 };
      let total = 0;
      for (const w of words) {
        const s = scoreWord(h, w);
        if (s === 0) return null;
        total += s;
      }
      return { h, s: total };
    })
    .filter((x): x is { h: Host; s: number } => x !== null);
  scored.sort((a, b) => b.s - a.s || (b.h.lastUsedAt ?? 0) - (a.h.lastUsedAt ?? 0) || a.h.alias.localeCompare(b.h.alias, 'ko'));
  return scored.slice(0, 50).map((x) => x.h);
}
