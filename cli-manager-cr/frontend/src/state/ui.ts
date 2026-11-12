// 대화상자 큐와 컨텍스트 메뉴 상태. 근거: spec/ui/dialogs.md §1.
import { create } from 'zustand';

export interface ConfirmOptions {
  title: string;
  body: string;
  detail?: string;
  okText?: string;
  danger?: boolean;
}

export interface PromptOptions {
  title: string;
  label?: string;
  value?: string;
  okText?: string;
}

type DialogReq =
  | { kind: 'confirm'; opts: ConfirmOptions; resolve: (v: boolean) => void }
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

export type SettingsSection = 'appearance' | 'input' | 'files' | 'connection' | 'sync' | 'security' | 'backup' | 'about';

interface UiState {
  /** 설정 화면(크게 보기 모달). section 은 열 때 보일 구역(기본 모양). */
  settings: { open: boolean; section: SettingsSection };
  openSettings: (section?: SettingsSection) => void;
  closeSettings: () => void;
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
  /** 미리보기 대화상자 대상(세션·경로). null 이면 닫힘(spec/ui/sftp-snippets.md §3). */
  preview: { sessionId: string; path: string } | null;
  openPreview: (sessionId: string, path: string) => void;
  closePreview: () => void;
  /** 위치 선택 대화상자: 서버 id. null 이면 닫힘. */
  locationPicker: { hostId: string } | null;
  openLocation: (hostId: string) => void;
  closeLocation: () => void;
  /** 빠른 접속 팔레트 상태. targetPane 이 있으면 그 빈 칸에서 연다. */
  quick: { open: boolean; targetPane: string | null };
  openQuick: (targetPane?: string | null) => void;
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
  /** 맨 앞 대화상자를 끝낸다(호스트가 호출). */
  settleDialog: (value: unknown) => void;
}

export const useUi = create<UiState>((set, get) => ({
  settings: { open: false, section: 'appearance' },
  openSettings: (section = 'appearance') => set({ settings: { open: true, section } }),
  closeSettings: () => set((s) => ({ settings: { ...s.settings, open: false } })),
  importOpen: false,
  openImportDialog: () => set({ importOpen: true }),
  closeImportDialog: () => set({ importOpen: false }),
  sftpOpen: [false, false],
  setSftpOpen: (group, open) => set((s) => ({ sftpOpen: group === 0 ? [open, s.sftpOpen[1]] : [s.sftpOpen[0], open] })),
  toggleSftp: (group) => set((s) => ({ sftpOpen: group === 0 ? [!s.sftpOpen[0], s.sftpOpen[1]] : [s.sftpOpen[0], !s.sftpOpen[1]] })),
  snippetsOpen: false,
  setSnippetsOpen: (open) => set({ snippetsOpen: open }),
  preview: null,
  openPreview: (sessionId, path) => set({ preview: { sessionId, path } }),
  closePreview: () => set({ preview: null }),
  locationPicker: null,
  openLocation: (hostId) => set({ locationPicker: { hostId } }),
  closeLocation: () => set({ locationPicker: null }),
  quick: { open: false, targetPane: null },
  openQuick: (targetPane = null) => set({ quick: { open: true, targetPane } }),
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
  settleDialog(value) {
    const [first, ...rest] = get().dialogs;
    if (!first) return;
    set({ dialogs: rest });
    if (first.kind === 'confirm') first.resolve(value === true);
    else first.resolve(typeof value === 'string' ? value : null);
  },
}));
