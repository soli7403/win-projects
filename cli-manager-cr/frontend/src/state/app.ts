// 앱 전역 상태: 저장 데이터·볼트 상태·토스트. 백엔드 이벤트를 여기서 받아 반영한다.
import { create } from 'zustand';
import { getBackend, errorMessage } from '../ipc/backend';
import type { Store, ToastPayload, VaultStatus } from '../ipc/types';

export interface ToastItem extends ToastPayload {
  id: number;
}

const MAX_TOASTS = 5;
let toastSeq = 0;

interface AppState {
  /** 저장 데이터와 볼트 상태를 한 번 이상 읽었는지. false 면 빈 앱 틀만 보인다. */
  booted: boolean;
  store: Store | null;
  vault: VaultStatus | null;
  toasts: ToastItem[];
  /** 최초 설정 직후 복구 코드를 아직 확인하지 않았는지(확인 전에는 앱으로 들어가지 않는다). */
  recoveryPending: boolean;
  pendingRecoveryCode: string;
  setRecoveryPending: (on: boolean, code: string) => void;
  /** 볼트가 열려 있는지. */
  vaultOpen: () => boolean;
  refreshStore: () => Promise<void>;
  refreshVault: () => Promise<void>;
  boot: () => Promise<void>;
  pushToast: (t: ToastPayload) => void;
  dismissToast: (id: number) => void;
  /** 저장 데이터 일부 덮어쓰기(store.patch). */
  patchStore: (patch: Partial<Store>) => Promise<void>;
}

/**
 * 저장 데이터 방어: 배열 키가 null·누락이면 빈 배열로 채운다(백엔드 계약상 항상 배열이지만 화면이 깨지지 않도록).
 * 설정 객체가 없으면 빈 객체로 둔다(기본값 병합은 백엔드 몫).
 */
export function normalizeStore(raw: Store | null | undefined): Store {
  const s = (raw ?? {}) as Partial<Store>;
  return {
    ...s,
    version: typeof s.version === 'number' ? s.version : 1,
    groups: Array.isArray(s.groups) ? s.groups : [],
    hosts: Array.isArray(s.hosts) ? s.hosts : [],
    snippets: Array.isArray(s.snippets) ? s.snippets : [],
    knownHosts: Array.isArray(s.knownHosts) ? s.knownHosts : [],
    settings: (s.settings ?? {}) as Store['settings'],
  } as Store;
}

export const useApp = create<AppState>((set, get) => ({
  booted: false,
  store: null,
  vault: null,
  toasts: [],
  recoveryPending: false,
  pendingRecoveryCode: '',
  setRecoveryPending: (on, code) => set({ recoveryPending: on, pendingRecoveryCode: on ? code : '' }),
  vaultOpen: () => !!get().vault?.unlocked,

  async refreshStore() {
    const raw = await getBackend().invoke<Store>('store.get');
    set({ store: normalizeStore(raw) });
  },

  async refreshVault() {
    const vault = await getBackend().invoke<VaultStatus>('vault.status');
    set({ vault });
  },

  async boot() {
    try {
      await Promise.all([get().refreshStore(), get().refreshVault()]);
    } catch (e) {
      get().pushToast({ kind: 'error', text: errorMessage(e) });
    } finally {
      set({ booted: true });
    }
  },

  pushToast(t) {
    const id = ++toastSeq;
    set((s) => ({ toasts: [...s.toasts, { ...t, id }].slice(-MAX_TOASTS) }));
    // error 6초, 나머지 3.5초 뒤 사라진다(spec/ui/shell.md §5).
    setTimeout(() => get().dismissToast(id), t.kind === 'error' ? 6000 : 3500);
  },

  dismissToast(id) {
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
  },

  async patchStore(patch) {
    await getBackend().invoke('store.patch', [patch]);
    await get().refreshStore();
  },
}));

/** 백엔드 이벤트를 앱 상태에 연결한다. 구독 해제 함수를 돌려준다. */
export function subscribeAppEvents(): () => void {
  const b = getBackend();
  const offs = [
    b.on('vault.changed', () => void useApp.getState().refreshVault()),
    b.on('store.changed', () => void useApp.getState().refreshStore()),
    b.on('toast', (t: ToastPayload) => useApp.getState().pushToast(t)),
  ];
  return () => offs.forEach((off) => off());
}
