// SFTP 경로·표시 형식 순수 함수. 근거: spec/ui/sftp-snippets.md §2.2·§2.4, spec/ui/dialogs.md §1.5.

const NEEDS_QUOTE = /[\s'"$`\\()&;|<>*?!#]/;

/** 터미널에 넣을 경로: 특수 문자가 있으면 작은따옴표로 감싸고 안의 ' 는 '\'' 로 바꾼다. */
export function shellQuotePath(path: string): string {
  if (!NEEDS_QUOTE.test(path)) return path;
  return `'${path.replace(/'/g, `'\\''`)}'`;
}

/** cd 명령용: 항상 작은따옴표로 감싼다(spec §2.2 '터미널에서 cd'). */
export function cdCommand(path: string): string {
  return `cd '${path.replace(/'/g, `'\\''`)}'`;
}

/** Windows 경로(구분자 \)용 따옴표: 공백·특수 문자가 있으면 큰따옴표로 감싼다 (file-panel.md §1). */
export function quoteFor(path: string, sep: string): string {
  if (sep !== '\\') return shellQuotePath(path);
  return /[\s&()^%!'";,]/.test(path) ? `"${path}"` : path;
}

/** cd 명령: 서버는 항상 작은따옴표, Windows 로컬은 quoteFor 규칙. */
export function cdCommandFor(path: string, sep: string): string {
  return sep === '\\' ? `cd ${quoteFor(path, sep)}` : cdCommand(path);
}

/** 구분자를 지정해 이름을 폴더에 붙인다. */
export function joinWith(dir: string, name: string, sep: string): string {
  return dir.endsWith(sep) ? dir + name : dir + sep + name;
}

/** 상위 폴더. "/" 이면 그대로, 마지막 요소를 뺀 경로(비면 "/"). */
export function parentPath(path: string): string {
  if (path === '/' || path === '') return '/';
  const trimmed = path.endsWith('/') ? path.slice(0, -1) : path;
  const idx = trimmed.lastIndexOf('/');
  if (idx <= 0) return '/';
  return trimmed.slice(0, idx);
}

/** 이름을 현재 폴더에 붙인 경로. */
export function joinPath(dir: string, name: string): string {
  if (dir.endsWith('/')) return dir + name;
  return `${dir}/${name}`;
}

const IMAGE_EXT = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg']);

export function extOf(name: string): string {
  const i = name.lastIndexOf('.');
  return i < 0 ? '' : name.slice(i + 1).toLowerCase();
}

export function isImageName(name: string): boolean {
  return IMAGE_EXT.has(extOf(name));
}

/** 바이트: 1024 미만 `n B`, 1MB 미만 `x.x KB`, 1GB 미만 `x.x MB`, 그 이상 `x.xx GB`. */
export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

const pad = (n: number) => String(n).padStart(2, '0');

/** 날짜: 올해이면 `MM-DD HH:mm`, 아니면 `YYYY-MM-DD`, 0 이면 빈 문자열. */
export function formatMtime(ms: number, now: Date = new Date()): string {
  if (!ms) return '';
  const d = new Date(ms);
  const date = `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
  if (d.getFullYear() === now.getFullYear()) return `${date} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** 권한 문자열 확인: 8진수 3~4자리만 유효(spec §2.4, F5). */
export function isValidMode(s: string): boolean {
  return /^[0-7]{3,4}$/.test(s);
}

/** 전송 진행 퍼센트: 전체가 0 이면 완료일 때 100, 아니면 0. */
export function transferPercent(bytes: number, total: number, state: string): number {
  if (total <= 0) return state === 'done' ? 100 : 0;
  return Math.min(100, Math.floor((bytes / total) * 100));
}
