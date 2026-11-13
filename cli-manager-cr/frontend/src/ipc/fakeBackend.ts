// 개발·테스트용 가짜 백엔드. 백엔드 없이 화면을 띄우기 위한 것이며 운영 빌드에서는 불리지 않는다.
// 데이터는 모두 가짜(예시 이름·주소)이고 메모리에만 있다. 채널 동작은 spec/ipc.md §2 를 따른다.
import type { Backend, EventHandler, EventName } from './backend';
import type { CredMeta, SessionInfo, Store, VaultStatus } from './types';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import { createFakeSftp } from './fakeSftp';

const RECOVERY_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';

function randomCode(): string {
  const chars: string[] = [];
  for (let i = 0; i < 24; i++) {
    chars.push(RECOVERY_ALPHABET[Math.floor(Math.random() * RECOVERY_ALPHABET.length)]);
  }
  const s = chars.join('');
  return [0, 4, 8, 12, 16, 20].map((i) => s.slice(i, i + 4)).join('-');
}

function seedStore(): Store {
  const now = Date.now();
  return {
    version: 1,
    groups: [
      { id: 'g-prod', parentId: null, name: '운영 서버', env: 'prod', sort: 0 },
      { id: 'g-lab', parentId: null, name: '실습', sort: 1 },
    ],
    hosts: [
      mkHost('h-1', 'g-prod', 'web-prod-01', 'ssh', '10.0.0.11', 22, 'deploy', now - 1000),
      mkHost('h-2', 'g-prod', 'db-prod-01', 'ssh', '10.0.0.21', 2222, 'dba', now - 5000),
      mkHost('h-3', 'g-lab', 'lab-box', 'ssh', 'lab.example.test', 22, 'student', now - 9000),
      mkHost('h-4', null, 'switch-core', 'telnet', '192.168.1.1', 23, 'admin', now - 20000),
    ],
    snippets: [{ id: 's-1', name: '디스크 사용량', body: 'df -h', sendEnter: true }],
    knownHosts: [],
    settings: { ...DEFAULT_SETTINGS },
  };
}

/** 즉석 접속의 제목(실제 백엔드: 로컬은 "로컬 터미널", 사용자가 있으면 사용자@주소). */
function adhocTitle(a: { host: string; username?: string; protocol?: string }): string {
  if (a.protocol === 'local') return '로컬 터미널';
  return a.username ? `${a.username}@${a.host}` : a.host;
}

/** 즉석 접속의 대상 문자열(실제 백엔드: 사용자@주소:포트, 로컬은 "로컬 · 폴더"). */
function adhocTarget(a: { host: string; username?: string; port?: number; protocol?: string; cwd?: string }): string {
  if (a.protocol === 'local') return `로컬 · ${a.cwd || '~'}`;
  const port = a.port ?? (a.protocol === 'telnet' ? 23 : 22);
  return `${a.username ? a.username + '@' : ''}${a.host}:${port}`;
}

function effEnv(h: Store['hosts'][number]): SessionInfo['env'] {
  if (h.env) return h.env;
  return undefined;
}

function effColor(h: Store['hosts'][number]): string | undefined {
  return h.color;
}

function mkHost(
  id: string,
  groupId: string | null,
  alias: string,
  protocol: 'ssh' | 'telnet',
  host: string,
  port: number,
  username: string,
  lastUsedAt: number,
): Store['hosts'][number] {
  return {
    id,
    groupId,
    alias,
    protocol,
    host,
    port,
    username,
    authType: 'password',
    tags: [],
    encoding: 'utf-8',
    termType: 'xterm-256color',
    keepaliveSec: 30,
    forwards: [],
    lastUsedAt,
    sort: lastUsedAt,
  };
}

type Handler = (args: unknown[]) => unknown | Promise<unknown>;

export function createFakeBackend(): Backend {
  let store: Store = seedStore();
  let vault = { initialized: false, unlocked: false, password: '', recoveryCode: '' };
  const creds: CredMeta[] = [{ id: 'c-1', name: '배포 계정', kind: 'password', username: 'deploy', updatedAt: Date.now() }];
  const listeners = new Map<EventName, Set<EventHandler>>();
  const sessions = new Map<string, SessionInfo>();
  let seq = 0;

  const emit = (event: EventName, ...args: unknown[]) => {
    listeners.get(event)?.forEach((h) => h(...args));
  };

  const vaultStatus = (): VaultStatus => ({
    initialized: vault.initialized,
    unlocked: vault.unlocked,
    osUnlockAvailable: true,
    osUnlockEnabled: false,
  });

  // 폴더의 환경·색은 서버 쪽 값이 없을 때 상속한다(가짜 백엔드 단순 구현).
  const groupOf = (h: Store['hosts'][number]) => store.groups.find((g) => g.id === h.groupId);
  const handlers: Record<string, Handler> = {
    'store.get': () => structuredClone(store),
    'store.patch': ([patch]) => {
      store = { ...store, ...(patch as object) } as Store;
      return undefined;
    },
    'vault.status': () => vaultStatus(),
    'vault.setup': ([pw]) => {
      if (vault.initialized) throw new Error('이미 앱 잠금 비밀번호가 설정되어 있습니다');
      vault = { initialized: true, unlocked: true, password: String(pw), recoveryCode: randomCode() };
      emit('vault.changed', true);
      return { recoveryCode: vault.recoveryCode };
    },
    'vault.unlock': ([pw]) => {
      if (!vault.initialized) return false;
      const ok = String(pw) === vault.password;
      if (ok) {
        vault.unlocked = true;
        emit('vault.changed', true);
      }
      return ok;
    },
    'vault.unlockOs': () => false,
    'vault.recover': ([code, pw]) => {
      if (String(code).toUpperCase() !== vault.recoveryCode) return false;
      vault.password = String(pw);
      vault.unlocked = true;
      emit('vault.changed', true);
      return true;
    },
    'vault.lock': () => {
      vault.unlocked = false;
      emit('vault.changed', false);
      return undefined;
    },
    'vault.changePassword': ([cur, next]) => {
      if (String(cur) !== vault.password) return false;
      vault.password = String(next);
      return true;
    },
    'vault.setOsUnlock': () => undefined,
    'cred.list': () => creds.map((c) => ({ ...c })),
    'cred.get': () => {
      if (!vault.unlocked) throw new Error('앱이 잠겨 있습니다');
      return null;
    },
    // 실제 백엔드와 같은 규칙: id 가 없으면 새로 만든다, 빠진 필드는 기존 값 유지, 빈 사용자 이름은 지운다.
    'cred.save': ([input]) => {
      const i = (input ?? {}) as { id?: string; name?: string; kind?: 'password' | 'key'; username?: string | null };
      const existing = i.id ? creds.find((c) => c.id === i.id) : undefined;
      const meta: CredMeta = {
        id: existing?.id ?? i.id ?? `c-${Date.now()}-${creds.length}`,
        name: i.name || existing?.name || '새 계정',
        kind: i.kind || existing?.kind || 'password',
        updatedAt: Date.now(),
      };
      const username = i.username === undefined || i.username === null ? existing?.username : i.username;
      if (username) meta.username = username;
      const at = creds.findIndex((c) => c.id === meta.id);
      if (at >= 0) creds.splice(at, 1);
      creds.push(meta);
      return { ...meta };
    },
    'cred.remove': ([id]) => {
      const at = creds.findIndex((c) => c.id === String(id));
      if (at >= 0) creds.splice(at, 1);
      return undefined;
    },
    'app.info': () => ({ version: '0.1.0-dev', platform: 'windows', dataDir: '(가짜 데이터)' }),
    'app.openExternal': () => undefined,
    'app.toggleFullScreen': () => undefined,
    'app.titlebar': () => undefined,
    // 파일 선택 대화상자는 가짜에서 항상 취소(null)다.
    'dialog.chooseDir': () => null,
    'dialog.readKeyFile': () => null,
    'local.shells': () => [{ path: 'pwsh.exe', label: 'PowerShell 7' }],
    'local.tmuxAvailable': () => false,
    'serial.list': () => [],
    'sync.status': () => ({ configured: false, gitAvailable: true }),
    'update.status': () => null,
    'session.open': ([args]) => {
      const a = (args ?? {}) as { hostId?: string; adhoc?: { host: string; username?: string; port?: number; protocol?: 'ssh' | 'telnet' | 'local'; cwd?: string } };
      const host = a.hostId ? store.hosts.find((h) => h.id === a.hostId) : undefined;
      if (!host && !a.adhoc) throw new Error('서버를 찾을 수 없습니다');
      const id = `s-${++seq}`;
      const info: SessionInfo = host
        ? {
            id, hostId: host.id, title: host.alias, protocol: host.protocol,
            target: `${host.username ? host.username + '@' : ''}${host.host}:${host.port}`, env: effEnv(host) ?? groupOf(host)?.env, color: effColor(host) ?? groupOf(host)?.color, state: 'connecting', canSftp: host.protocol === 'ssh',
          }
        : {
            id, hostId: null, title: adhocTitle(a.adhoc!), protocol: a.adhoc!.protocol ?? 'ssh',
            target: adhocTarget(a.adhoc!), state: 'connecting', canSftp: (a.adhoc!.protocol ?? 'ssh') === 'ssh',
          };
      sessions.set(id, info);
      // 접속은 약 30ms 뒤에 시작한다(spec/ipc.md §2). 가짜 백엔드는 곧바로 연결됨으로 보낸다.
      setTimeout(() => {
        const s = { ...info, state: 'connected' as const, connectedAt: Date.now() };
        sessions.set(id, s);
        emit('session.state', s);
        emit('session.data', id, '\x1b[32m[가짜 백엔드]\x1b[0m 연결을 흉내 냈습니다. 출력은 없습니다.\r\n');
      }, 30);
      return info;
    },
    'session.close': ([id]) => {
      sessions.delete(String(id));
      return undefined;
    },
    'session.reconnect': () => undefined,
    'clip.writeText': () => undefined,
    'clip.readText': () => '',
    'clip.info': () => ({ type: 'empty' }),
    'clip.hasImage': () => false,
    // C1 화면(설정·가져오기·동기화·백업·업데이트)용 가짜 응답. 가짜 데이터만 쓴다.
    'app.resetWindow': () => undefined,
    'app.openLogs': () => undefined,
    // 실제 백엔드 응답 모양(spec/ipc.md "가져오기 후보 객체"): 서버 정보는 host 객체 안에 있다.
    'import.scan': () => [
      { source: 'putty', name: 'prod-web02', host: { alias: 'prod-web02', protocol: 'ssh', host: '10.1.1.2', port: 22, username: 'ubuntu', authType: 'password', keepaliveSec: 30 } },
      { source: 'ssh-config', name: 'bastion', host: { alias: 'bastion', protocol: 'ssh', host: '203.0.113.9', port: 22, username: 'ec2-user', authType: 'key', keepaliveSec: 30 }, keyFile: '~/.ssh/bastion.pem' },
      { source: 'putty', name: 'serial-lab', host: { alias: 'serial-lab', protocol: 'serial', host: '', port: 0, authType: 'password', serial: { path: '/dev/ttyUSB0', baudRate: 115200 } } },
      { source: 'securecrt', name: 'gw_tunneling', folderPath: ['lab', 'tunnel'], host: { alias: 'gw_tunneling', protocol: 'ssh', host: '203.0.113.5', port: 22, username: 'ops', authType: 'password', keepaliveSec: 30 } },
    ],
    'import.scanSecureCrt': () => [
      { source: 'securecrt', name: 'web01', folderPath: ['MDS'], host: { alias: 'web01', protocol: 'ssh', host: '10.0.0.13', port: 22, username: 'ops', authType: 'password', keepaliveSec: 30 } },
    ],
    'import.scanAws': () => [
      { source: 'aws', name: 'app-01', host: { alias: 'app-01', protocol: 'ssh', host: '10.0.3.5', port: 22, username: 'ubuntu', authType: 'key', keepaliveSec: 30 }, jumpVia: '203.0.113.9', note: 'ap-northeast-2 · 키 파일 없음: app.pem' },
    ],
    'import.apply': ([list]) => ({ hosts: Array.isArray(list) ? list.length : 0, keys: 0, skipped: [] }),
    'sync.run': () => ({ state: 'ok', lastSync: Date.now(), changed: false }),
    'update.check': () => ({ status: 'latest' }),
    'backup.export': () => false,
    'backup.import': () => false,
    // SFTP 채널(fakeSftp.ts). 세션 목록은 이 파일의 sessions 를 그대로 쓴다.
    ...createFakeSftp({
      sessionOf: (id) => sessions.get(id),
      emit: (event, info) => emit(event, info),
    }),
  };

  return {
    async invoke(channel, args = []) {
      const h = handlers[channel];
      if (!h) throw new Error(`구현되지 않은 채널: ${channel}`);
      return (await h(args)) as any;
    },
    termInput(sessionId, text) {
      // 가짜 백엔드는 입력을 그대로 되돌려 준다(에코).
      emit('session.data', sessionId, text);
    },
    termResize() {
      /* 가짜 백엔드: 크기 변경은 무시 */
    },
    reply() {
      /* 가짜 백엔드: ui.request 를 만들지 않으므로 답할 것이 없다 */
    },
    on(event, handler) {
      let set = listeners.get(event);
      if (!set) {
        set = new Set();
        listeners.set(event, set);
      }
      set.add(handler);
      return () => set!.delete(handler);
    },
  };
}
