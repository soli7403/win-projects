// 저장 데이터·IPC 페이로드 타입. 근거: spec/data/store.md, spec/ipc.md.
// 필드 이름은 데이터 호환을 위한 사실이며 spec 과 같아야 한다.

export type Protocol = 'ssh' | 'telnet' | 'serial' | 'local';
export type AuthType = 'password' | 'key' | 'agent' | 'ask';
export type EnvKind = 'prod' | 'stage' | 'perf' | 'dev' | 'device' | 'none';

export interface Group {
  id: string;
  parentId: string | null;
  name: string;
  color?: string;
  env?: EnvKind;
  collapsed?: boolean;
  sort: number;
}

export interface Forward {
  id: string;
  type: 'L' | 'R';
  bindHost: string;
  bindPort: number;
  destHost: string;
  destPort: number;
}

export interface Host {
  id: string;
  groupId: string | null;
  alias: string;
  protocol: Protocol;
  host: string;
  port: number;
  username: string;
  authType: AuthType;
  credentialId?: string | null;
  jumpHostId?: string | null;
  jumpDirect?: unknown;
  color?: string;
  env?: EnvKind;
  tags: string[];
  encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'latin1';
  termType: string;
  startupCommand?: string;
  keepSession?: boolean;
  tmuxName?: string;
  keepaliveSec: number;
  forwards: Forward[];
  serial?: unknown;
  local?: { shell?: string; cwd?: string };
  notes?: string;
  favorite?: boolean;
  lastUsedAt?: number;
  sort: number;
}

export interface Snippet {
  id: string;
  name: string;
  body: string;
  sendEnter: boolean;
}

export interface KnownHost {
  hostPort: string;
  keyType: string;
  fingerprint: string;
  addedAt: number;
}

export interface Settings {
  /** 화면 언어(D75): en·ko·ja·zh·tw·de·fr·es·pt·ru. 비면 아직 고르지 않음 → 첫 실행 언어 선택, 그동안은 OS 언어. */
  language: string;
  theme: 'dark' | 'light';
  terminalTheme: string;
  fontFamily: string;
  fontSize: number;
  lineHeight: number;
  cursorStyle: 'block' | 'bar' | 'underline';
  cursorBlink: boolean;
  scrollback: number;
  copyOnSelect: boolean;
  rightClickPaste: boolean;
  ctrlVPaste: boolean;
  confirmMultilinePaste: boolean;
  guardDangerousOnProd: boolean;
  dangerousPatterns: string[];
  autoReconnect: boolean;
  autoLockMinutes: number;
  uploadDir: string;
  uploadCleanupDays: number;
  /** 서버 ↔ 서버 복사 때 거쳐 가는 이 PC 임시 폴더. 비면 OS 임시 폴더 (behavior/file-panel.md §4). */
  transferTempDir: string;
  /** 탭 하나의 최대 칸 수(상하 분할 한도), 1~5 (ui/panes.md §2). */
  maxVerticalSplits: number;
  insertPathQuote: boolean;
  sessionLog: boolean;
  /** 세션 로그 보관 일수. 0 이면 지우지 않음 (behavior/session.md §6). */
  sessionLogKeepDays: number;
  bell: boolean;
  shiftEnterNewline: boolean;
  rememberWindow: boolean;
  restoreLayout: boolean;
  updateCheck: boolean;
  syncRepo: string;
  syncAuto: boolean;
  osc52Copy: boolean;
  sidebarWidth: number;
  rightPanelWidth: number;
  /** 사용 용도(D78): '' 아직 안 고름 / personal(기한 없음) / business(30일 평가 뒤 라이선스). */
  usage: '' | 'personal' | 'business';
  /** 업무용을 고른 시각(ms). 평가 기간 계산에 쓴다. */
  usageSince: number;
  /** 업무용 라이선스 키(DT1.…). 비면 없음. */
  licenseKey: string;
  /** 회사 PC 안내를 이미 보였는지. */
  domainNoticeDone: boolean;
  [extra: string]: unknown;
}

export interface Store {
  version: number;
  groups: Group[];
  hosts: Host[];
  snippets: Snippet[];
  knownHosts: KnownHost[];
  settings: Settings;
  layout?: unknown;
  [extra: string]: unknown;
}

export interface VaultStatus {
  initialized: boolean;
  unlocked: boolean;
  osUnlockAvailable: boolean;
  osUnlockEnabled: boolean;
  osUnlockKind?: string;
}

export interface CredMeta {
  id: string;
  name: string;
  kind: 'password' | 'key';
  username?: string;
  updatedAt: number;
}

export interface SessionInfo {
  id: string;
  hostId: string | null;
  title: string;
  protocol: Protocol;
  target: string;
  color?: string;
  env?: EnvKind;
  state: 'connecting' | 'connected' | 'reconnecting' | 'closed' | 'error';
  connectedAt?: number;
  message?: string;
  canSftp: boolean;
  keepSession?: boolean;
}

export interface SyncStatus {
  configured: boolean;
  gitAvailable: boolean;
  /** 마지막 동기화 결과. 항상 객체이며 동기화 전에는 {state: ""} (spec/ipc.md sync.status, null 금지). */
  last?: { state: string; message?: string; lastSync?: number; changed?: boolean };
}

export interface ToastPayload {
  kind: 'ok' | 'error' | 'info';
  text: string;
}

export interface UiRequestBase {
  requestId: string;
  sessionTitle?: string;
}
