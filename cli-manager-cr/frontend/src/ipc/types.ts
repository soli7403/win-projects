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
  /** L 로컬 · R 원격 · D 동적(SOCKS 프록시, 대상 없음) (protocols.md §2.5). */
  type: 'L' | 'R' | 'D';
  bindHost: string;
  bindPort: number;
  destHost: string;
  destPort: number;
}

/** 로그인 스크립트의 한 단계(behavior/session.md §5.1, D139). */
export interface LoginStep {
  id: string;
  /** 서버 출력에서 기다릴 글자. 비면 기다리지 않고 바로 보낸다. */
  expect: string;
  /** 보낼 글자(뒤에 Enter 가 붙는다). `{{password}}` 는 로그인 비밀번호로 바뀐다. */
  send: string;
  /** 있으면 send 대신 이 보관함 항목(비밀번호 종류)의 비밀번호를 보낸다(D148). 파일에는 id 만 남는다. */
  secretId?: string;
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
  /** 프록시 경유 설정(store.md §3 proxy). 없으면 바로 연결. */
  proxy?: unknown;
  /** SSH 에이전트 전달(protocols.md §2.8). 켜면 이 PC 에이전트의 키를 서버에서 쓸 수 있다. */
  agentForward?: boolean;
  color?: string;
  env?: EnvKind;
  tags: string[];
  encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'euc-jp' | 'gbk' | 'big5' | 'latin1';
  termType: string;
  startupCommand?: string;
  /** 접속 직후 실행하는 로그인 스크립트(store.md §3 loginScript). 로컬 터미널에는 없다. */
  loginScript?: LoginStep[];
  /** 이 서버에 연결한 창에서만 쓰는 출력 트리거 규칙(D149). 설정의 규칙과 함께 동작한다. */
  triggerRules?: TriggerRule[];
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
  /** 있으면 이 항목은 매크로다: 단계를 순서대로, 출력을 기다려 가며 실행한다 (store.md §7). */
  steps?: MacroStep[];
}

/** 키워드 강조 규칙 하나(behavior/terminal.md §9). */
export interface HighlightRule {
  id: string;
  /** 찾을 글자(regex 가 true 면 정규식). */
  pattern: string;
  /** 강조 색 `#RRGGBB`. */
  color: string;
  regex: boolean;
  caseSensitive: boolean;
  /** true 면 글자색 대신 바탕을 칠한다(글자는 흰색). */
  background: boolean;
  enabled: boolean;
}

/** 출력 트리거 규칙 하나(behavior/terminal.md §10, D140). */
export interface TriggerRule {
  id: string;
  /** 찾을 글자(regex 가 true 면 정규식). 출력의 한 줄에 대해 찾는다. */
  pattern: string;
  regex: boolean;
  caseSensitive: boolean;
  /** 알림(화면 구석의 안내)을 띄운다. */
  notify: boolean;
  /** 짧은 소리를 낸다. */
  sound: boolean;
  /** 자동 응답을 보낸다. */
  replyOn: boolean;
  /** 자동 응답으로 입력할 글자. */
  reply: string;
  /** 자동 응답 뒤에 Enter 를 붙인다. */
  replyEnter: boolean;
  enabled: boolean;
}

/** 매크로의 한 단계(behavior/terminal.md §11, D141). */
export interface MacroStep {
  id: string;
  /** 보내기 전에 출력에서 기다릴 글자. 비면 출력이 잠잠해질 때까지 기다린다. */
  expect: string;
  /** 보낼 글자. 제어 문자는 `\x03` 처럼 적는다. `{{이름}}` 변수를 쓸 수 있다. */
  send: string;
  /** 보낸 뒤 Enter 를 붙인다. */
  enter: boolean;
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
  /** 터미널 출력의 키워드 강조 규칙. 앞에 있는 규칙이 우선이다 (behavior/terminal.md §9). */
  highlightRules: HighlightRule[];
  /** 출력 트리거 규칙 (behavior/terminal.md §10). */
  triggerRules: TriggerRule[];
  /** 알림을 켠 트리거가 맞았을 때 앱 창에 포커스가 없으면 Windows 알림도 띄운다 (D145). */
  triggerOsNotify: boolean;
  autoReconnect: boolean;
  autoLockMinutes: number;
  uploadDir: string;
  uploadCleanupDays: number;
  /** 서버 ↔ 서버 복사 때 거쳐 가는 이 PC 임시 폴더. 비면 OS 임시 폴더 (behavior/file-panel.md §4). */
  transferTempDir: string;
  /** 탭 하나의 최대 칸 수(상하 분할 한도), 1~5 (ui/panes.md §2). */
  maxVerticalSplits: number;
  insertPathQuote: boolean;
  /** 서버에서 sz·rz 를 실행하면 ZMODEM 으로 파일을 주고받는다 (behavior/zmodem.md). */
  zmodemEnabled: boolean;
  /** ZMODEM 으로 받은 파일을 둘 이 PC 폴더. 비면 OS 의 다운로드 폴더. */
  zmodemDownloadDir: string;
  sessionLog: boolean;
  /** 세션 로그 보관 일수. 0 이면 지우지 않음 (behavior/session.md §6). */
  sessionLogKeepDays: number;
  /** 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 받는 즉시 쓰고 내려 쓴다 (behavior/session.md §6). */
  sessionLogSyncSec: number;
  /** 세션 로그의 줄마다 앞에 시각([YYYY-MM-DD HH:mm:ss.SSS])을 붙인다 (behavior/session.md §6). */
  sessionLogTimestamp: boolean;
  /** 세션 로그에 ANSI 시퀀스(색·커서 이동)와 CR 을 지우지 않고 받은 그대로 남긴다 (behavior/session.md §6, D120). */
  sessionLogAnsi: boolean;
  /** 앱 단축키 전체를 쓸지. 끄면 그 키들은 터미널(서버)로 간다 (behavior/shortcuts.md §5, D126). */
  shortcutsEnabled: boolean;
  /** 기본값에서 바꾼 단축키만: 명령 id → 조합('Ctrl+Shift+KeyT'). 빈 글자는 키 없음 (behavior/shortcuts.md §5). */
  shortcuts: Record<string, string>;
  bell: boolean;
  shiftEnterNewline: boolean;
  rememberWindow: boolean;
  restoreLayout: boolean;
  updateCheck: boolean;
  syncRepo: string;
  syncAuto: boolean;
  osc52Copy: boolean;
  sidebarWidth: number;
  /** 사이드바의 "즐겨찾기" 구역을 접어 두었는지 (ui/sidebar.md §3, D131). */
  favoritesCollapsed: boolean;
  /** 사이드바의 "최근 사용" 구역을 접어 두었는지 (ui/sidebar.md §3, D131). */
  recentCollapsed: boolean;
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
  /** 지금 연결에 설정한 포트 포워딩과 그 결과(session.md §1). 없으면 필드가 없다. */
  forwards?: ForwardInfo[];
}

/** 포트 포워딩 하나의 설정 결과(session.md §1 forwards, D136). */
export interface ForwardInfo {
  type: 'L' | 'R' | 'D';
  /** 듣는 주소 `호스트:포트` (L·D 는 이 PC, R 은 서버). */
  bind: string;
  /** 대상 `호스트:포트`. D 는 없다. */
  dest?: string;
  /** 실패했으면 그 까닭. */
  error?: string;
  /** 이 포워딩으로 지금 이어져 있는 연결 수(session.forwards 로 물었을 때의 값, D142). */
  conns?: number;
  /** 접속한 뒤 받아들인 쪽 프로그램이 보낸 바이트(D151). */
  up?: number;
  /** 접속한 뒤 받아들인 쪽 프로그램이 받은 바이트(D151). */
  down?: number;
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
