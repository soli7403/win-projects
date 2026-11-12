// 서버 편집 대화상자의 순수 규칙(입력 나누기, 프로토콜 전환, 검사, 정규화). 근거: spec/ui/host-editor.md §1.1~§1.6.
// 화면 없이 단위 테스트하기 위해 React 밖에 둔다.
import type { EnvKind, Host, Protocol } from '../ipc/types';

export type SerialParity = 'none' | 'even' | 'odd' | 'mark' | 'space';
export type SerialFlow = 'none' | 'rtscts' | 'xonxoff';
export type SerialEnter = 'CR' | 'LF' | 'CRLF';

export interface SerialOpts {
  path: string;
  baudRate: number;
  dataBits: 5 | 6 | 7 | 8;
  parity: SerialParity;
  stopBits: 1 | 1.5 | 2;
  flowControl: SerialFlow;
  enterSends: SerialEnter;
  localEcho: boolean;
}

/** 시리얼 기본값(§1.6). */
export const SERIAL_DEFAULTS: SerialOpts = {
  path: 'COM1',
  baudRate: 9600,
  dataBits: 8,
  parity: 'none',
  stopBits: 1,
  flowControl: 'none',
  enterSends: 'CR',
  localEcho: false,
};

export const BAUD_RATES = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600];

/** 점프 호스트 직접 입력 정보(store.md §3 jumpDirect). */
export interface JumpDirect {
  host: string;
  port: number;
  username: string;
  authType: 'password' | 'key';
  credentialId?: string | null;
}

/** 직접 입력 베스천을 가리키는 jumpHostId 값. */
export const JUMP_DIRECT = 'direct';

const DEFAULT_PORT: Record<Protocol, number> = { ssh: 22, telnet: 23, serial: 0, local: 0 };

/** 호스트 칸 입력 나누기(§1.1): `사용자@호스트:포트`. 사용자나 포트가 있을 때만 나눈다. */
export function splitHostInput(raw: string): { host: string; port?: string; username?: string } {
  const v = raw.trim();
  const m = /^(?:([^@\s]+)@)?([^:@\s]+)(?::(\d+))?$/.exec(v);
  if (m && (m[1] || m[3])) return { host: m[2], port: m[3], username: m[1] };
  return { host: raw };
}

/** 베스천 주소 나누기(§1.3): `[사용자@]호스트[:포트]`, IPv6 는 대괄호. 사용자나 포트가 있을 때만 나눈다. */
export function splitBastionInput(raw: string): { host: string; port?: string; username?: string } {
  const v = raw.trim();
  const m = /^(?:([^@\s]+)@)?(\[[0-9a-fA-F:.]+\]|[^:@\s]+)(?::(\d+))?$/.exec(v);
  if (m && (m[1] || m[3])) {
    const host = m[2].startsWith('[') ? m[2].slice(1, -1) : m[2];
    return { host, port: m[3], username: m[1] };
  }
  return { host: raw };
}

/** 프로토콜 전환 시 포트 보정(§1.1 1·5번). */
export function portAfterProtocolChange(next: Protocol, port: number): number {
  let p = port;
  if (next === 'telnet' && p === 22) p = 23;
  if (next === 'ssh' && p === 23) p = 22;
  if (next !== 'local' && next !== 'serial' && p === 0) p = DEFAULT_PORT[next];
  return p;
}

/** 시리얼로 바꿀 때 옵션 채우기(§1.1 1번): 없으면 기본값, 환경이 비었거나 none 이면 device. */
export function serialDefaultsFor(prev: SerialOpts | undefined, env: EnvKind | undefined): { serial: SerialOpts; env: EnvKind | undefined } {
  const serial = prev ? { ...SERIAL_DEFAULTS, ...prev } : { ...SERIAL_DEFAULTS };
  const nextEnv = !env || env === 'none' ? 'device' : env;
  return { serial, env: nextEnv };
}

/** 별칭 보정(§1.5 2번). */
export function fallbackAlias(alias: string, protocol: Protocol, opts: { host: string; serialPath?: string; localCwd?: string }): string {
  const a = alias.trim();
  if (a) return a;
  if (protocol === 'serial') return opts.serialPath?.trim() || 'Serial';
  if (protocol === 'local') {
    const last = (opts.localCwd ?? '').trim().split(/[\\/]/).filter(Boolean).pop();
    return last || '로컬 터미널';
  }
  return opts.host.trim();
}

/** 저장 검사(§1.5 4번). 실패하면 이동할 탭과 문구를 돌려준다. */
export function validateHostForm(input: {
  protocol: Protocol;
  host: string;
  serialPath: string;
  authType: string;
  hasSecret: boolean;
  vaultOpen: boolean;
  jumpDirect: boolean;
  jumpAddress: string;
  jumpHasSecret: boolean;
}): { tab: 'basic' | 'auth' | 'adv'; message: string } | null {
  const isNet = input.protocol === 'ssh' || input.protocol === 'telnet';
  if (isNet && !input.host.trim()) return { tab: 'basic', message: '호스트 주소를 입력하세요' };
  if (input.protocol === 'serial' && !input.serialPath.trim()) return { tab: 'basic', message: '시리얼 포트를 선택하세요' };
  if (isNet && input.hasSecret && !input.vaultOpen) {
    return { tab: 'auth', message: '볼트가 잠겨 있어 비밀번호를 저장할 수 없습니다' };
  }
  if (input.protocol === 'ssh' && input.jumpDirect) {
    if (!input.jumpAddress.trim()) return { tab: 'adv', message: '점프 호스트 주소를 입력하세요' };
    if (input.jumpHasSecret && !input.vaultOpen) {
      return { tab: 'adv', message: '볼트가 잠겨 있어 점프 호스트 자격 증명을 저장할 수 없습니다' };
    }
  }
  return null;
}

/** PuTTY PPK v3 키인지(§1.2, H9). */
export function isPpkV3(text: string): boolean {
  return /^PuTTY-User-Key-File-3:/m.test(text.trimStart());
}
export const PPK_V3_MESSAGE = 'PPK v3 키는 PuTTYgen → Conversions → Export OpenSSH key 로 변환해 주세요';

/** 셸 경로가 WSL 인지(세션 유지 카드 표시 조건, §1.3 3번). */
export function isWslShell(path: string | undefined): boolean {
  return !!path && /(^|[\\/])wsl(\.exe)?$/i.test(path.trim());
}

/** tmux 세션 이름 기본값(§1.3 3번 도움말): 비우면 별칭, 로컬이면 폴더 이름. */
export function tmuxPlaceholder(alias: string, protocol: Protocol, localCwd: string): string {
  if (protocol === 'local') {
    const last = localCwd.trim().split(/[\\/]/).filter(Boolean).pop();
    return last || 'termgo';
  }
  return alias.trim() || 'termgo';
}

/** 폴더 편집 저장 가능 여부(H10): 이름 공백 제거 후 비어 있지 않아야 한다. */
export function canSaveFolder(name: string): boolean {
  return name.trim().length > 0;
}

/** 저장 시 쓸 로컬 옵션(§1.5 3번): 셸·시작 폴더만, 공백 제거, 비면 지정 안 함. */
export function localOptions(shell: string, cwd: string): Host['local'] {
  return { shell: shell.trim() || undefined, cwd: cwd.trim() || undefined };
}
