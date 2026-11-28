// 서버 저장 파이프라인: 계정 저장 → 점프 호스트 베스천 저장 → 서버 객체 조립. 근거: spec/ui/host-editor.md §1.5 3~8번.
import type { AuthType, CredMeta, Host, Protocol } from '../ipc/types';
import { getBackend } from '../ipc/backend';
import { JUMP_DIRECT, localOptions, PROXY_DEFAULT_PORT, type JumpDirect, type ProxyConf, type ProxyType, type SerialOpts } from './hostForm';
import { cleanLoginScript } from './loginScript';
import { t } from '../i18n';

export interface SaveInput {
  original: Host | null;
  hostId: string | null;
  allHosts: Host[];
  protocol: Protocol;
  alias: string;
  host: string;
  port: number;
  username: string;
  folderId: string | null;
  tags: string[];
  env: Host['env'];
  color?: string;
  notes?: string;
  authType: AuthType;
  /** 계정 저장 방식: 이 서버 전용이면 own, 공유 계정이면 shared. */
  accountMode: 'own' | 'shared';
  sharedCredId: string | null;
  password: string;
  privateKey: string;
  passphrase: string;
  encoding: Host['encoding'];
  termType: string;
  keepaliveSec: number;
  startupCommand?: string;
  /** 로그인 스크립트 단계(빈 단계는 저장할 때 뺀다). */
  loginScript?: Host['loginScript'];
  /** 이 서버의 출력 트리거 규칙(D149). */
  triggerRules?: Host['triggerRules'];
  keepSession: boolean;
  tmuxName: string;
  forwards: Host['forwards'];
  serial: SerialOpts;
  localShell: string;
  localCwd: string;
  /** 점프 호스트: '' 사용 안 함, 'direct' 직접 입력, 그 밖은 서버 id. */
  jumpChoice: string;
  jump: { host: string; port: number; username: string; authType: JumpDirect['authType']; password: string; keyText: string; passphrase: string };
  /** 프록시 경유: type 이 '' 이면 사용 안 함. password 는 새로 넣었을 때만 값이 있다. */
  proxy?: { type: ProxyType | ''; host: string; port: number; username: string; password: string; command?: string };
  /** SSH 에이전트 전달. */
  agentForward?: boolean;
}

/** 계정 저장(§1.5 5번). 기존 계정을 이 서버만 쓰면 갱신, 다른 서버와 공유 중이면 새 항목. */
async function saveAccount(
  kind: 'password' | 'key',
  name: string,
  username: string,
  secret: { password?: string; privateKey?: string; passphrase?: string },
  reuseId: string | undefined,
): Promise<string> {
  const payload = kind === 'key'
    ? { id: reuseId, name, kind, username, privateKey: secret.privateKey, passphrase: secret.passphrase || undefined }
    : { id: reuseId, name, kind, username, password: secret.password };
  const meta = await getBackend().invoke<CredMeta>('cred.save', [payload]);
  return meta.id;
}

/** 저장을 수행하고 새 서버 객체를 돌려준다. 호출한 쪽이 목록 반영·대화상자 닫기를 맡는다. */
export async function buildHostFromForm(i: SaveInput): Promise<Host> {
  const isNet = i.protocol === 'ssh' || i.protocol === 'telnet';
  const isLocal = i.protocol === 'local';
  const isSerial = i.protocol === 'serial';
  const trimmedHost = i.host.trim();
  const trimmedUser = i.username.trim();
  const now = Date.now();

  // 1) 이 서버 전용 계정 저장.
  let credentialId: string | null = i.original?.credentialId ?? null;
  const ownSecret = isNet && i.accountMode === 'own' && ((i.authType === 'password' && i.password) || (i.authType === 'key' && i.privateKey.trim()));
  if (ownSecret) {
    const sharedElsewhere = i.allHosts.some((h) => h.id !== i.hostId && !!credentialId && h.credentialId === credentialId);
    const reuse = credentialId && !sharedElsewhere ? credentialId : undefined;
    credentialId = await saveAccount(
      i.authType === 'key' ? 'key' : 'password',
      i.alias,
      trimmedUser,
      { password: i.password, privateKey: i.privateKey.trim(), passphrase: i.passphrase },
      reuse,
    );
  } else if (isNet && i.accountMode === 'shared') {
    credentialId = i.sharedCredId;
  }
  // 매번 묻기는 저장된 계정 연결을 끊는다(§1.5 6번). 에이전트는 기존 연결 유지.
  if (isNet && i.authType === 'ask') credentialId = null;
  if (!isNet) credentialId = null;

  // 2) 점프 호스트(SSH 만, §1.5 7번).
  let jumpHostId: string | null = null;
  let jumpDirect: JumpDirect | undefined;
  if (i.protocol === 'ssh' && i.jumpChoice === JUMP_DIRECT) {
    const prevJump = i.original?.jumpDirect as JumpDirect | undefined;
    let jumpCred = prevJump?.credentialId ?? null;
    const j = i.jump;
    const secret = (j.authType === 'password' && j.password) || (j.authType === 'key' && j.keyText.trim());
    if (secret) {
      jumpCred = await saveAccount(
        j.authType === 'key' ? 'key' : 'password',
        t('host.jumpAccountName', { alias: i.alias }),
        j.username.trim(),
        { password: j.password, privateKey: j.keyText.trim(), passphrase: j.passphrase },
        jumpCred ?? undefined,
      );
    }
    jumpHostId = JUMP_DIRECT;
    jumpDirect = {
      host: j.host.trim(),
      port: j.port || 22,
      username: j.username.trim(),
      authType: j.authType,
      credentialId: jumpCred,
    };
  } else if (i.protocol === 'ssh' && i.jumpChoice) {
    jumpHostId = i.jumpChoice;
  }

  // 2-1) 프록시 경유(SSH 만). 비밀번호를 새로 넣었으면 "{별칭} 프록시" 계정으로 저장한다.
  //      사용자 이름을 비우면 인증 없는 프록시로 보고 계정 연결을 끊는다.
  let proxy: ProxyConf | undefined;
  if (i.protocol === 'ssh' && i.proxy?.type === 'command') {
    // 프록시 명령(D152): 주소·계정 없이 명령 줄만.
    const command = (i.proxy.command ?? '').trim();
    if (command) proxy = { type: 'command', host: '', port: 0, username: '', credentialId: null, command };
  } else if (i.protocol === 'ssh' && i.proxy && i.proxy.type && i.proxy.host.trim()) {
    const prev = i.original?.proxy as ProxyConf | undefined;
    const user = i.proxy.username.trim();
    let proxyCred = user ? prev?.credentialId ?? null : null;
    if (user && i.proxy.password) {
      proxyCred = await saveAccount('password', t('host.proxyAccountName', { alias: i.alias }), user, { password: i.proxy.password }, proxyCred ?? undefined);
    }
    proxy = {
      type: i.proxy.type,
      host: i.proxy.host.trim(),
      port: i.proxy.port || PROXY_DEFAULT_PORT[i.proxy.type as Exclude<ProxyType, 'command'>],
      username: user,
      credentialId: proxyCred,
    };
  }

  const base: Host = i.original ?? {
    id: crypto.randomUUID(),
    groupId: null,
    alias: '',
    protocol: i.protocol,
    host: '',
    port: 22,
    username: '',
    authType: 'password',
    tags: [],
    encoding: 'utf-8',
    termType: 'xterm-256color',
    keepaliveSec: 30,
    forwards: [],
    sort: now,
  };

  // 3) 로컬 정규화(§1.5 3번): 원격 전용 값은 비운다.
  const next: Host = {
    ...base,
    alias: i.alias,
    protocol: i.protocol,
    host: isNet ? trimmedHost : '',
    port: isNet ? i.port || (i.protocol === 'telnet' ? 23 : 22) : 0,
    username: isNet ? trimmedUser : '',
    authType: isNet ? i.authType : 'ask',
    credentialId,
    jumpHostId,
    jumpDirect,
    proxy,
    agentForward: i.protocol === 'ssh' && i.agentForward ? true : undefined,
    groupId: i.folderId || null,
    env: i.env === 'none' ? undefined : i.env,
    color: i.color,
    tags: i.tags,
    notes: i.notes?.trim() || undefined,
    encoding: isNet || isSerial ? i.encoding : 'utf-8',
    termType: i.termType.trim() || 'xterm-256color',
    keepaliveSec: isNet ? Number(i.keepaliveSec) || 0 : base.keepaliveSec,
    startupCommand: i.startupCommand?.trim() || undefined,
    loginScript: isLocal ? undefined : cleanLoginScript(i.loginScript),
    triggerRules: cleanHostTriggers(i.triggerRules),
    keepSession: i.protocol === 'ssh' || isLocal ? i.keepSession : undefined,
    tmuxName: (i.protocol === 'ssh' || isLocal) && i.keepSession ? i.tmuxName.trim() || undefined : undefined,
    forwards: i.protocol === 'ssh' ? i.forwards : [],
    serial: isSerial ? { ...i.serial, path: i.serial.path.trim() } : undefined,
    local: isLocal ? localOptions(i.localShell, i.localCwd) : undefined,
  };
  return next;
}

/** 서버 목록 반영(§1.5 8번): 새로 만들면 추가, 아니면 교체. */
/** 서버별 트리거 규칙 저장값: 찾을 글자가 빈 규칙은 빼고, 하나도 없으면 undefined(필드를 남기지 않는다, D149). */
export function cleanHostTriggers(rules: Host['triggerRules']): Host['triggerRules'] {
  const out = (rules ?? []).filter((r) => r.pattern !== '');
  return out.length ? out : undefined;
}

export function applyHost(hosts: Host[], next: Host, editing: boolean): Host[] {
  return editing ? hosts.map((h) => (h.id === next.id ? next : h)) : [...hosts, next];
}

