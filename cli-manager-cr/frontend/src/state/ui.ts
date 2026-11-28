// 대화상자 큐와 컨텍스트 메뉴 상태. 근거: spec/ui/dialogs.md §1.
import { create } from 'zustand';

export interface ConfirmOptions {
  title: string;
  body: string;
  detail?: string;
  okText?: string;
  danger?: boolean;
  /** 알림 전용: [취소] 없이 [확인]만. 닫아도 확인과 같다. */
  noCancel?: boolean;
}

/** 예/아니오/취소 세 갈래 질문. 아니오도 뜻이 있는 동작일 때 쓴다(예: 폴더만 삭제). */
export interface AskOptions extends ConfirmOptions {
  /** [아니오] 단추 문구. */
  noText: string;
}

/** ask 결과: 'yes' = 확인 단추, 'no' = 아니오 단추, null = 취소·Esc·X·바깥 클릭. */
export type AskResult = 'yes' | 'no' | null;

export interface PromptOptions {
  title: string;
  label?: string;
  value?: string;
  okText?: string;
}

type DialogReq =
  | { kind: 'confirm'; opts: ConfirmOptions; resolve: (v: boolean) => void }
  | { kind: 'ask'; opts: AskOptions; resolve: (v: AskResult) => void }
  | { kind: 'prompt'; opts: PromptOptions; resolve: (v: string | null) => void };

export interface MenuItem {
  label?: string;
  shortcut?: string;
  danger?: boolean;
  disabled?: boolean;
  separator?: boolean;
  onSelect?: () => void;
}

export type EditorState =
  | { kind: 'host'; hostId: string | null; groupId: string | null }
  | { kind: 'folder'; folderId: string | null; parentId: string | null };

export type SettingsSection = 'appearance' | 'input' | 'shortcuts' | 'rules' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';

interface UiState {
  /** 설정 화면(크게 보기 모달). section 은 열 때 보일 구역(기본 모양). */
  settings: { open: boolean; section: SettingsSection };
  openSettings: (section?: SettingsSection) => void;
  closeSettings: () => void;
  /** 도움말 창(ui/help.md, D155). topic 은 처음 보일 항목 id, returnTo 는 닫을 때 다시 열 설정 구역. */
  help: { open: boolean; topic: string | null; returnTo: SettingsSection | null };
  openHelp: (topic?: string | null, returnTo?: SettingsSection | null) => void;
  closeHelp: () => void;
  /** 서버 가져오기 대화상자 열림 여부. */
  importOpen: boolean;
  openImportDialog: () => void;
  closeImportDialog: () => void;
  /** 탭 그룹별 SFTP 패널 열림([왼쪽, 오른쪽]). 그룹당 최대 1개(ui/sftp-snippets.md §1). */
  sftpOpen: [boolean, boolean];
  setSftpOpen: (group: 0 | 1, open: boolean) => void;
  toggleSftp: (group: 0 | 1) => void;
  /** 서버 목록 아래 절반의 스니펫 패널 열림. */
  snippetsOpen: boolean;
  setSnippetsOpen: (open: boolean) => void;
  /** 왼쪽 열에서 스니펫 패널이 차지하는 높이 비율(%). 경계를 끌어 바꾸며 이 PC 에만 기억한다. */
  snippetRatio: number;
  setSnippetRatio: (pct: number) => void;
  /** 모든 창 동시 입력(쇠사슬 아이콘). 켜지면 입력이 화면에 보이는 모든 창으로 간다. 시작할 때 항상 꺼짐 (behavior/terminal.md 동시 입력). */
  linkInput: boolean;
  toggleLinkInput: () => void;
  /** 미리보기 대화상자 대상(세션·경로). null 이면 닫힘(spec/ui/sftp-snippets.md §3). */
  preview: { sessionId: string; path: string } | null;
  openPreview: (sessionId: string, path: string) => void;
  closePreview: () => void;
  /** 위치 선택 대화상자: 서버 id. null 이면 닫힘. */
  /** 빠른 접속 팔레트 상태. targetPane 이 있으면 그 빈 칸에서 연다.
   *  group 이 있으면(탭 막대의 + 로 열었을 때) 그 탭 그룹에 연다. 없으면 포커스 그룹 (D119). */
  quick: { open: boolean; targetPane: string | null; group: 0 | 1 | null };
  openQuick: (targetPane?: string | null, group?: 0 | 1 | null) => void;
  closeQuick: () => void;
  /** 사이드바에서 선택된 서버 id. */
  selectedHostId: string | null;
  setSelectedHost: (id: string | null) => void;
  /** 열린 서버·폴더 편집 대화상자. */
  editor: EditorState | null;
  openEditor: (e: EditorState) => void;
  closeEditor: () => void;
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  dialogs: DialogReq[];
  menu: { x: number; y: number; items: MenuItem[] } | null;
  openMenu: (x: number, y: number, items: MenuItem[]) => void;
  closeMenu: () => void;
  /** 확인 대화상자. 예=true, 아니오·Esc·X·바깥 클릭=false. */
  confirm: (opts: ConfirmOptions) => Promise<boolean>;
  /** 입력 대화상자. 취소=null. */
  prompt: (opts: PromptOptions) => Promise<string | null>;
  /** 세 갈래 질문. 취소·Esc·X·바깥 클릭은 null — 아니오와 구분된다. */
  ask: (opts: AskOptions) => Promise<AskResult>;
  /** 맨 앞 대화상자를 끝낸다(호스트가 호출). */
  settleDialog: (value: unknown) => void;
}

/** 스니펫 패널 높이 비율 기본값·범위(%). 서버 목록이 더 길게 보이도록 기본은 3분의 1 남짓. */
export const SNIPPET_RATIO_DEFAULT = 35;
const SNIPPET_RATIO_MIN = 15;
const SNIPPET_RATIO_MAX = 80;
const SNIPPET_RATIO_KEY = 'dariterm.snippetRatio';

export function clampSnippetRatio(pct: number): number {
  if (!Number.isFinite(pct)) return SNIPPET_RATIO_DEFAULT;
  return Math.min(SNIPPET_RATIO_MAX, Math.max(SNIPPET_RATIO_MIN, pct));
}

// 저장소가 막혀 있어도(사생활 모드 등) 기본값으로 동작한다.
function loadSnippetRatio(): number {
  try {
    const raw = globalThis.localStorage?.getItem(SNIPPET_RATIO_KEY);
    return raw ? clampSnippetRatio(Number(raw)) : SNIPPET_RATIO_DEFAULT;
  } catch {
    return SNIPPET_RATIO_DEFAULT;
  }
}

function saveSnippetRatio(pct: number): void {
  try {
    globalThis.localStorage?.setItem(SNIPPET_RATIO_KEY, String(Math.round(pct)));
  } catch {
    /* 저장 실패는 무시 — 다음 실행 때 기본값 */
  }
}

export const useUi = create<UiState>((set, get) => ({
  settings: { open: false, section: 'appearance' },
  openSettings: (section = 'appearance') => set({ settings: { open: true, section } }),
  closeSettings: () => set((s) => ({ settings: { ...s.settings, open: false } })),
  help: { open: false, topic: null, returnTo: null },
  // 설정에서 열면 설정을 닫고 도움말만 띄운다(창이 겹치면 Esc 가 둘을 함께 닫는다). 닫을 때 설정으로 돌아간다.
  openHelp: (topic = null, returnTo = null) =>
    set((s) => ({ help: { open: true, topic, returnTo }, settings: returnTo ? { ...s.settings, open: false } : s.settings })),
  closeHelp: () =>
    set((s) => ({ help: { open: false, topic: null, returnTo: null }, settings: s.help.returnTo ? { open: true, section: s.help.returnTo } : s.settings })),
  importOpen: false,
  openImportDialog: () => set({ importOpen: true }),
  closeImportDialog: () => set({ importOpen: false }),
  sftpOpen: [false, false],
  setSftpOpen: (group, open) => set((s) => ({ sftpOpen: group === 0 ? [open, s.sftpOpen[1]] : [s.sftpOpen[0], open] })),
  toggleSftp: (group) => set((s) => ({ sftpOpen: group === 0 ? [!s.sftpOpen[0], s.sftpOpen[1]] : [s.sftpOpen[0], !s.sftpOpen[1]] })),
  snippetsOpen: false,
  setSnippetsOpen: (open) => set({ snippetsOpen: open }),
  snippetRatio: loadSnippetRatio(),
  setSnippetRatio: (pct) => {
    const v = clampSnippetRatio(pct);
    saveSnippetRatio(v);
    set({ snippetRatio: v });
  },
  linkInput: false,
  toggleLinkInput: () => set((s) => ({ linkInput: !s.linkInput })),
  preview: null,
  openPreview: (sessionId, path) => set({ preview: { sessionId, path } }),
  closePreview: () => set({ preview: null }),
  quick: { open: false, targetPane: null, group: null },
  openQuick: (targetPane = null, group = null) => set({ quick: { open: true, targetPane, group } }),
  closeQuick: () => set((s) => ({ quick: { ...s.quick, open: false } })),
  selectedHostId: null,
  setSelectedHost: (id) => set({ selectedHostId: id }),
  editor: null,
  openEditor: (e) => set({ editor: e }),
  closeEditor: () => set({ editor: null }),
  sidebarOpen: true,
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  dialogs: [],
  menu: null,
  openMenu: (x, y, items) => set({ menu: { x, y, items } }),
  closeMenu: () => set({ menu: null }),
  confirm: (opts) =>
    new Promise<boolean>((resolve) => set((s) => ({ dialogs: [...s.dialogs, { kind: 'confirm', opts, resolve }] }))),
  prompt: (opts) =>
    new Promise<string | null>((resolve) => set((s) => ({ dialogs: [...s.dialogs, { kind: 'prompt', opts, resolve }] }))),
  ask: (opts) =>
    new Promise<AskResult>((resolve) => set((s) => ({ dialogs: [...s.dialogs, { kind: 'ask', opts, resolve }] }))),
  settleDialog(value) {
    const [first, ...rest] = get().dialogs;
    if (!first) return;
    set({ dialogs: rest });
    if (first.kind === 'confirm') first.resolve(value === true);
    else if (first.kind === 'ask') first.resolve(value === 'yes' || value === 'no' ? value : null);
    else first.resolve(typeof value === 'string' ? value : null);
  },
}));
