// 기본값과 고정 목록. 근거: spec/data/store.md §4·§6, spec/ui/tokens.md §6.
import type { EnvKind, Settings } from '../ipc/types';
import type { MsgId } from '../i18n';

export const DANGEROUS_PATTERNS: string[] = [
  String.raw`rm\s+-[a-zA-Z]*r[a-zA-Z]*f`,
  String.raw`rm\s+-[a-zA-Z]*f[a-zA-Z]*r`,
  String.raw`\breboot\b`,
  String.raw`\bshutdown\b`,
  String.raw`\bhalt\b`,
  String.raw`\bpoweroff\b`,
  String.raw`mkfs`,
  String.raw`dd\s+if=`,
  String.raw`\bDROP\s+(TABLE|DATABASE)\b`,
  String.raw`\bTRUNCATE\b`,
  String.raw`systemctl\s+(stop|restart)`,
];

export const DEFAULT_SETTINGS: Settings = {
  language: '',
  theme: 'dark',
  terminalTheme: 'DariTerm Dark',
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
  dangerousPatterns: [...DANGEROUS_PATTERNS],
  highlightRules: [],
  triggerRules: [],
  triggerOsNotify: true,
  autoReconnect: true,
  autoLockMinutes: 15,
  uploadDir: '~/.dariterm/uploads',
  uploadCleanupDays: 7,
  transferTempDir: '',
  maxVerticalSplits: 3,
  insertPathQuote: true,
  zmodemEnabled: true,
  zmodemDownloadDir: '',
  sessionLog: false,
  sessionLogKeepDays: 30,
  sessionLogSyncSec: 2,
  sessionLogTimestamp: false,
  sessionLogAnsi: false,
  shortcutsEnabled: true,
  shortcuts: {},
  bell: false,
  shiftEnterNewline: true,
  rememberWindow: true,
  restoreLayout: true,
  updateCheck: true,
  syncRepo: '',
  syncAuto: true,
  osc52Copy: true,
  sidebarWidth: 260,
  favoritesCollapsed: false,
  recentCollapsed: false,
  rightPanelWidth: 340,
  usage: '',
  usageSince: 0,
  licenseKey: '',
  domainNoticeDone: false,
};

/** label 은 문구 키(D75) — 그릴 때 t() 로 바꾼다. */
export const ENV_INFO: Record<EnvKind, { label: MsgId; color: string }> = {
  prod: { label: 'env.prod', color: '#ef4444' },
  stage: { label: 'env.stage', color: '#f59e0b' },
  perf: { label: 'env.perf', color: '#a855f7' },
  dev: { label: 'env.dev', color: '#22c55e' },
  device: { label: 'env.device', color: '#3b82f6' },
  none: { label: 'env.none', color: '#64748b' },
};

/** 색 선택 팔레트(14색, 순서 유지). 맨 앞의 "자동"은 색 값을 지운다. */
export const COLOR_PALETTE: string[] = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308', '#22c55e', '#10b981', '#14b8a6',
  '#06b6d4', '#3b82f6', '#6366f1', '#8b5cf6', '#d946ef', '#ec4899', '#64748b',
];
