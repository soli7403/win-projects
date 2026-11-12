// Shared types between main / preload / renderer

export type Env = 'prod' | 'stage' | 'dev' | 'device' | 'none'
export type Protocol = 'ssh' | 'telnet' | 'serial' | 'local'
export type AuthType = 'password' | 'key' | 'agent' | 'ask'

export interface Group {
  id: string
  parentId: string | null
  name: string
  color?: string
  env?: Env
  collapsed?: boolean
  sort: number
}

export interface PortForward {
  id: string
  type: 'L' | 'R'
  bindHost: string
  bindPort: number
  destHost: string
  destPort: number
}

export interface SerialOptions {
  path: string
  baudRate: number
  dataBits: 5 | 6 | 7 | 8
  parity: 'none' | 'even' | 'odd' | 'mark' | 'space'
  stopBits: 1 | 1.5 | 2
  flowControl: 'none' | 'rtscts' | 'xonxoff'
  enterSends: 'CR' | 'LF' | 'CRLF'
  localEcho: boolean
}

/** 로컬 터미널 — 이 PC 의 셸. 빈 값이면 기본 셸 / 홈 폴더 */
export interface LocalOptions {
  shell?: string
  cwd?: string
}

/** 점프 호스트 "직접 입력" — 별도 서버로 등록하지 않고 이 서버 설정 안에 둔 베스천 (jumpHostId === 'direct') */
export interface JumpDirect {
  host: string
  port: number
  username: string
  authType: 'password' | 'key'
  /** 볼트 항목 (이름: "<서버 별칭> 베스천") — 대상 서버 credentialId 와 별개 */
  credentialId?: string | null
}

export const JUMP_DIRECT = 'direct'

export interface Host {
  id: string
  groupId: string | null
  alias: string
  protocol: Protocol
  host: string
  port: number
  username: string
  authType: AuthType
  credentialId?: string | null
  jumpHostId?: string | null
  jumpDirect?: JumpDirect
  color?: string
  env?: Env
  tags: string[]
  encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'latin1'
  termType: string
  startupCommand?: string
  /** 3.3 세션 유지(tmux) — 원격 SSH, 로컬(macOS·Linux, Windows 는 WSL 셸) */
  keepSession?: boolean
  /** tmux 세션 이름 — 비우면 서버 별칭/로컬 폴더 이름 */
  tmuxName?: string
  keepaliveSec: number
  forwards: PortForward[]
  serial?: SerialOptions
  local?: LocalOptions
  notes?: string
  favorite?: boolean
  lastUsedAt?: number
  sort: number
}

export interface Snippet {
  id: string
  name: string
  body: string
  sendEnter: boolean
}

export interface KnownHost {
  hostPort: string // host:port
  keyType: string
  fingerprint: string // SHA256:base64
  addedAt: number
}

export interface Settings {
  theme: 'dark' | 'light'
  terminalTheme: string
  fontFamily: string
  fontSize: number
  lineHeight: number
  cursorStyle: 'block' | 'bar' | 'underline'
  cursorBlink: boolean
  scrollback: number
  copyOnSelect: boolean
  rightClickPaste: boolean
  ctrlVPaste: boolean
  confirmMultilinePaste: boolean
  guardDangerousOnProd: boolean
  dangerousPatterns: string[]
  autoReconnect: boolean
  autoLockMinutes: number
  uploadDir: string
  uploadCleanupDays: number
  /** 서버 → 서버 복사 때 거쳐 가는 이 PC 의 임시 폴더 (비면 OS 임시 폴더) */
  transferTempDir: string
  insertPathQuote: boolean
  sessionLog: boolean
  bell: boolean
  /** Shift+Enter 로 줄바꿈(ESC+CR) 보내기 */
  shiftEnterNewline: boolean
  /** 창 크기·위치·최대화 상태 기억 */
  rememberWindow: boolean
  /** 앱을 닫을 때 열려 있던 탭·분할을 저장하고 다음 실행 때 다시 연다 */
  restoreLayout: boolean
  /** 새 버전 자동 확인 (시작 시 + 12시간마다) */
  updateCheck: boolean
  /** 3.11 git 동기화 저장소 주소 / 자동 동기화 */
  syncRepo: string
  syncAuto: boolean
  /** 원격의 OSC 52 "클립보드에 쓰기" 요청 허용 */
  osc52Copy: boolean
  sidebarWidth: number
  rightPanelWidth: number
}

/** 3.4: 저장하는 탭·분할 배치 — 서버 식별자(또는 로컬 폴더)와 구조만. 비밀번호·화면 내용은 없다 */
export type SavedNode =
  | { type: 'pane'; hostId?: string; local?: { cwd?: string }; empty?: boolean }
  | { type: 'split'; dir: 'row' | 'col'; sizes: number[]; children: SavedNode[] }
export interface SavedLayout {
  tabs: { title?: string; broadcast?: boolean; root: SavedNode }[]
  active: number
}

export interface StoreData {
  version: number
  groups: Group[]
  hosts: Host[]
  snippets: Snippet[]
  knownHosts: KnownHost[]
  settings: Settings
  layout?: SavedLayout
}

export interface CredentialMeta {
  id: string
  name: string
  kind: 'password' | 'key'
  username?: string
  updatedAt: number
}

export interface CredentialSecret {
  username?: string
  password?: string
  privateKey?: string
  passphrase?: string
}

export interface CredentialInput extends CredentialSecret {
  id?: string
  name: string
  kind: 'password' | 'key'
}

export interface VaultStatus {
  initialized: boolean
  unlocked: boolean
  osUnlockAvailable: boolean
  osUnlockEnabled: boolean
  osUnlockKind?: 'windows' | 'touchid' | 'keychain'
}

export type SessionState = 'connecting' | 'connected' | 'reconnecting' | 'closed' | 'error'

export interface SessionInfo {
  id: string
  hostId: string | null
  title: string
  protocol: Protocol
  target: string // user@host:port or COM3@115200
  color?: string
  env?: Env
  state: SessionState
  connectedAt?: number
  message?: string
  canSftp: boolean
  /** tmux 세션 유지 중 */
  keepSession?: boolean
}

export interface AdhocTarget {
  host: string
  /** protocol 'local' 일 때 시작 폴더 */
  cwd?: string
  port?: number
  username?: string
  protocol?: Protocol
}

export interface SftpEntry {
  name: string
  path: string
  isDir: boolean
  isLink: boolean
  size: number
  mtime: number
  mode: number
  perm: string
}

export interface Transfer {
  id: string
  sessionId: string
  name: string
  direction: 'up' | 'down'
  bytes: number
  total: number
  state: 'queued' | 'running' | 'done' | 'error' | 'canceled'
  error?: string
  remotePath: string
  localPath?: string
}

/** 파일 패널 끌어 놓기 복사 작업 (끌어 놓기 한 번 = 하나) */
export interface CopyJob {
  id: string
  srcSession: string
  srcTitle: string
  srcDir: string
  dstSession: string
  dstTitle: string
  dstDir: string
  /** 첫 항목 이름 / 끌어 온 항목 수 */
  name: string
  count: number
  /** 폴더 안까지 센 파일 수 */
  files: number
  filesDone: number
  bytes: number
  total: number
  /** 서버 → 서버 (이 PC 임시 폴더 경유) */
  relay: boolean
  phase: 'scan' | 'download' | 'upload' | 'copy'
  state: 'queued' | 'running' | 'done' | 'error' | 'canceled'
  error?: string
  startedAt: number
  endedAt?: number
}

export type UiRequest =
  | { reqId: string; kind: 'hostkey'; sessionTitle: string; hostPort: string; keyType: string; fingerprint: string; changed: boolean; oldFingerprint?: string }
  | { reqId: string; kind: 'password'; sessionTitle: string; prompt: string; username: string; allowSave: boolean; failed?: boolean }
  | { reqId: string; kind: 'keyboard'; sessionTitle: string; name: string; instructions: string; prompts: { prompt: string; echo: boolean }[] }
  | { reqId: string; kind: 'passphrase'; sessionTitle: string }
  | { reqId: string; kind: 'username'; sessionTitle: string }

export interface ImportCandidate {
  source: 'putty' | 'ssh-config' | 'aws'
  name: string
  host: Partial<Host>
  keyFile?: string
  /** 점프 호스트의 주소 — 같이 가져오는 후보나 이미 있는 서버 중 host 가 같은 것으로 잇는다 */
  jumpVia?: string
  note?: string
}

export interface ClipboardInfo {
  type: 'image' | 'files' | 'text' | 'empty'
  text?: string
  files?: string[]
  imageSize?: { width: number; height: number }
}

export const MAC_FONT_FAMILY = "Menlo, 'D2Coding', 'Apple SD Gothic Neo', monospace"

export const DEFAULT_SETTINGS: Settings = {
  theme: 'dark',
  terminalTheme: 'TermGo Dark',
  fontFamily: "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
  fontSize: 14,
  lineHeight: 1.15,
  cursorStyle: 'block',
  cursorBlink: true,
  scrollback: 10000,
  copyOnSelect: true,
  rightClickPaste: true,
  ctrlVPaste: true,
  confirmMultilinePaste: true,
  guardDangerousOnProd: true,
  dangerousPatterns: ['rm\\s+-[a-zA-Z]*r[a-zA-Z]*f', 'rm\\s+-[a-zA-Z]*f[a-zA-Z]*r', '\\breboot\\b', '\\bshutdown\\b', '\\bhalt\\b', '\\bpoweroff\\b', 'mkfs', 'dd\\s+if=', '\\bDROP\\s+(TABLE|DATABASE)\\b', '\\bTRUNCATE\\b', 'systemctl\\s+(stop|restart)'],
  autoReconnect: true,
  autoLockMinutes: 15,
  uploadDir: '~/.brightterm/uploads',
  uploadCleanupDays: 7,
  transferTempDir: '',
  insertPathQuote: true,
  sessionLog: false,
  bell: false,
  shiftEnterNewline: true,
  rememberWindow: true,
  restoreLayout: true,
  updateCheck: true,
  syncRepo: '',
  syncAuto: true,
  osc52Copy: true,
  sidebarWidth: 260,
  rightPanelWidth: 340
}

export const ENV_COLORS: Record<Env, string> = {
  prod: '#ef4444',
  stage: '#f59e0b',
  dev: '#22c55e',
  device: '#3b82f6',
  none: '#64748b'
}

export const ENV_LABELS: Record<Env, string> = {
  prod: '운영',
  stage: '스테이징',
  dev: '개발',
  device: '장비',
  none: '일반'
}

export const PALETTE = ['#ef4444', '#f97316', '#f59e0b', '#eab308', '#22c55e', '#10b981', '#14b8a6', '#06b6d4', '#3b82f6', '#6366f1', '#8b5cf6', '#d946ef', '#ec4899', '#64748b']

