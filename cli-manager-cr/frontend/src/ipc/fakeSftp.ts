// 가짜 백엔드의 SFTP 채널(개발·테스트 전용). 채널 계약은 spec/ipc.md §2 "SFTP·붙여넣기", 동작은 spec/protocols.md §3.
// 메모리 안의 가짜 파일 트리만 다룬다. 모든 이름·내용은 예시 데이터다.
import type { SessionInfo } from './types';
import { isImageName } from '../lib/sftpPath';

export interface FakeNode {
  name: string;
  path: string;
  isDir: boolean;
  isLink?: boolean;
  size: number;
  mtime: number;
  perm: number;
  content?: string;
}

export interface FakeSftpDeps {
  /** 세션 정보(없으면 undefined). */
  sessionOf: (id: string) => SessionInfo | undefined;
  /** 전송 이벤트 발행. */
  emit: (event: 'transfer.update', info: unknown) => void;
}

const HOME = '/home/demo';
const DAY = 86400000;

function seedTree(): Map<string, FakeNode> {
  const now = Date.now();
  const nodes: FakeNode[] = [
    { name: 'logs', path: `${HOME}/logs`, isDir: true, size: 0, mtime: now - DAY, perm: 0o755 },
    { name: 'app.conf', path: `${HOME}/app.conf`, isDir: false, size: 2048, mtime: now - 3600000, perm: 0o644, content: 'port = 8080\nname = demo\n' },
    { name: 'shot.png', path: `${HOME}/shot.png`, isDir: false, size: 178 * 1024, mtime: new Date(2023, 10, 15).getTime(), perm: 0o644 },
    { name: 'big.txt', path: `${HOME}/big.txt`, isDir: false, size: 600 * 1024, mtime: now - 7200000, perm: 0o644, content: 'x'.repeat(64) },
    { name: 'current', path: `${HOME}/current`, isDir: true, isLink: true, size: 0, mtime: now - DAY * 3, perm: 0o777 },
  ];
  const map = new Map<string, FakeNode>();
  map.set(HOME, { name: 'demo', path: HOME, isDir: true, size: 0, mtime: now, perm: 0o755 });
  for (const n of nodes) map.set(n.path, n);
  return map;
}

/** 권한 문자열(drwxr-xr-x 형식). 첫 글자: 링크 l, 폴더 d, 그 외 -. */
export function permString(n: FakeNode): string {
  const first = n.isLink ? 'l' : n.isDir ? 'd' : '-';
  const bits = [0o400, 0o200, 0o100, 0o40, 0o20, 0o10, 0o4, 0o2, 0o1];
  const chars = 'rwxrwxrwx';
  return first + bits.map((b, i) => (n.perm & b ? chars[i] : '-')).join('');
}

function normalize(path: string): string {
  if (!path || path === '~') return HOME;
  if (path.startsWith('~/')) return HOME + path.slice(1);
  if (path.length > 1 && path.endsWith('/')) return path.slice(0, -1);
  return path;
}

const PNG_1x1 =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==';

export function createFakeSftp(deps: FakeSftpDeps): Record<string, (args: unknown[]) => unknown> {
  const tree = seedTree();
  let seq = 0;

  const sshSession = (id: unknown): SessionInfo => {
    const info = deps.sessionOf(String(id));
    if (!info) throw new Error('세션을 찾을 수 없습니다');
    if (!info.canSftp || info.protocol !== 'ssh') throw new Error('SSH로 연결된 세션에서만 사용할 수 있습니다');
    if (info.state !== 'connected') throw new Error('연결되지 않았습니다');
    return info;
  };

  const need = (path: string): FakeNode => {
    const n = tree.get(path);
    if (!n) throw new Error(`경로를 찾을 수 없습니다: ${path}`);
    return n;
  };

  const childrenOf = (dir: string): FakeNode[] => {
    const prefix = dir.endsWith('/') ? dir : `${dir}/`;
    const out: FakeNode[] = [];
    for (const n of tree.values()) {
      if (n.path.startsWith(prefix) && !n.path.slice(prefix.length).includes('/') && n.path !== dir) out.push(n);
    }
    return out;
  };

  const transfer = (sessionId: string, name: string, direction: 'upload' | 'download', size: number, remotePath: string, localPath?: string) => {
    const id = `tr-${++seq}`;
    deps.emit('transfer.update', { id, sessionId, name, direction, bytes: size, total: size, state: 'done', remotePath, localPath });
    return id;
  };

  return {
    'sftp.home': ([id]) => {
      sshSession(id);
      return HOME;
    },
    'sftp.list': ([id, path]) => {
      sshSession(id);
      const p = normalize(String(path ?? ''));
      const dir = need(p);
      if (!dir.isDir) throw new Error('폴더가 아닙니다');
      const entries = childrenOf(p).map((n) => ({
        name: n.name,
        path: n.path,
        isDir: n.isDir,
        isLink: !!n.isLink,
        size: n.size,
        mtime: n.mtime,
        mode: n.perm,
        perm: permString(n),
      }));
      // 폴더 먼저, 그다음 이름 오름차순(대소문자 무시). spec/protocols.md §3 목록.
      entries.sort((a, b) => (a.isDir !== b.isDir ? (a.isDir ? -1 : 1) : a.name.toLowerCase().localeCompare(b.name.toLowerCase())));
      return { path: p, entries };
    },
    'sftp.mkdir': ([id, path]) => {
      sshSession(id);
      const p = normalize(String(path));
      if (tree.has(p)) throw new Error('이미 있습니다');
      const parent = p.slice(0, p.lastIndexOf('/')) || '/';
      need(parent);
      tree.set(p, { name: p.slice(p.lastIndexOf('/') + 1), path: p, isDir: true, size: 0, mtime: Date.now(), perm: 0o755 });
      return undefined;
    },
    'sftp.rename': ([id, from, to]) => {
      sshSession(id);
      const a = need(normalize(String(from)));
      const dest = normalize(String(to));
      if (tree.has(dest)) throw new Error('같은 이름이 이미 있습니다');
      tree.delete(a.path);
      tree.set(dest, { ...a, path: dest, name: dest.slice(dest.lastIndexOf('/') + 1) });
      return undefined;
    },
    'sftp.remove': ([id, paths]) => {
      sshSession(id);
      for (const raw of paths as string[]) {
        const p = normalize(raw);
        need(p);
        for (const k of [...tree.keys()]) if (k === p || k.startsWith(`${p}/`)) tree.delete(k);
      }
      return undefined;
    },
    'sftp.chmod': ([id, path, mode]) => {
      sshSession(id);
      need(normalize(String(path))).perm = Number(mode) & 0o777;
      return undefined;
    },
    'sftp.upload': ([id, localPaths, remoteDir]) => {
      const info = sshSession(id);
      const dir = normalize(String(remoteDir));
      need(dir);
      const out: string[] = [];
      for (const lp of localPaths as string[]) {
        const name = lp.split(/[\\/]/).pop() ?? 'file';
        const remote = `${dir}/${name}`;
        tree.set(remote, { name, path: remote, isDir: false, size: 1024, mtime: Date.now(), perm: 0o644 });
        transfer(info.id, name, 'upload', 1024, remote, lp);
        out.push(remote);
      }
      return out;
    },
    // 로컬 파일 선택 대화상자는 가짜에서 항상 취소(빈 배열).
    'sftp.uploadPick': ([id]) => {
      sshSession(id);
      return [];
    },
    'sftp.download': ([id, paths, localDir]) => {
      const info = sshSession(id);
      const dest = typeof localDir === 'string' && localDir ? localDir : 'C:\\Users\\demo\\Downloads';
      for (const raw of paths as string[]) {
        const n = need(normalize(raw));
        transfer(info.id, n.name, 'download', n.size, n.path, `${dest}\\${n.name}`);
      }
      return dest;
    },
    'sftp.preview': ([id, path]) => {
      sshSession(id);
      const n = need(normalize(String(path)));
      if (n.isDir) return null;
      if (isImageName(n.name)) {
        // 20MB 이하 이미지만 미리보기(spec/protocols.md §3). 가짜 데이터는 1x1 PNG 로 대신한다.
        return n.size <= 20 * 1024 * 1024 ? { kind: 'image', data: PNG_1x1 } : null;
      }
      // 텍스트: 512KB 이하(NUL 없음). 가짜 트리에서는 content 가 있으면 텍스트로 본다.
      if (n.size > 512 * 1024 || n.content === undefined) return null;
      return { kind: 'text', data: n.content };
    },
    // 붙여넣기 원본: 클립보드 이미지·버퍼는 업로드 폴더에 새 이름으로, 파일 목록은 같은 이름으로 올린다(가짜 트리).
    'sftp.uploadForPrompt': ([id, origin]) => {
      const info = sshSession(id);
      const o = (origin ?? {}) as { kind?: string; paths?: string[]; name?: string };
      const dir = `${HOME}/.termgo/uploads`;
      if (!tree.has(dir)) tree.set(dir, { name: 'uploads', path: dir, isDir: true, size: 0, mtime: Date.now(), perm: 0o700 });
      const names: string[] =
        o.kind === 'clipboardImage' ? ['clipboard.png']
        : o.kind === 'buffer' ? [o.name || 'buffer.bin']
        : (o.paths ?? []).map((p) => p.split(/[\/]/).pop() ?? 'file');
      const out: string[] = [];
      for (const name of names) {
        const remote = `${dir}/${name}`;
        tree.set(remote, { name, path: remote, isDir: false, size: 1024, mtime: Date.now(), perm: 0o600 });
        transfer(info.id, name, 'upload', 1024, remote, name);
        out.push(remote);
      }
      return out;
    },
    // 로컬 세션의 붙여넣기: 파일 목록은 원래 경로, 이미지·버퍼는 로컬 업로드 폴더에 저장한 경로를 돌려준다.
    'local.saveForPrompt': ([origin]) => {
      const o = (origin ?? {}) as { kind?: string; paths?: string[]; name?: string };
      if (o.kind === 'files') return o.paths ?? [];
      const name = o.kind === 'buffer' ? o.name || 'buffer.bin' : 'clipboard.png';
      return [`C:\\Users\\demo\\.termgo\\uploads\\${name}`];
    },
    'sftp.edit': ([id, path]) => {
      sshSession(id);
      const n = need(normalize(String(path)));
      return `C:\\Users\\demo\\AppData\\Local\\Temp\\termgo-edit\\demo\\${n.name}`;
    },
  };
}
