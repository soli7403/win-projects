// 새 버전 상태: 상태 표시줄 "새 버전" 버튼과 update.available 이벤트 처리. 근거: spec/behavior/lifecycle.md §6, spec/ui/shell.md §3.
import { create } from 'zustand';
import { getBackend, errorMessage } from '../ipc/backend';
import type { UpdateInfo, UpdateCheckResult } from '../ipc/c1types';
import { useApp } from './app';

interface UpdateState {
  available: UpdateInfo | null;
  setAvailable: (u: UpdateInfo | null) => void;
  /** 처음 한 번 마지막 발견 상태를 읽는다. */
  loadStatus: () => Promise<void>;
  /** 설정의 [지금 확인]: 결과를 토스트로 알린다. */
  checkNow: () => Promise<void>;
}

export const useUpdate = create<UpdateState>((set) => ({
  available: null,
  setAvailable: (u) => set({ available: u }),
  async loadStatus() {
    try {
      const r = await getBackend().invoke<{ status?: string; version: string; url: string } | null>('update.status');
      if (r && r.version) set({ available: { version: r.version, url: r.url } });
    } catch {
      /* 조회 실패는 조용히 넘어간다 */
    }
  },
  async checkNow() {
    const push = useApp.getState().pushToast;
    try {
      const r = await getBackend().invoke<UpdateCheckResult>('update.check');
      if (r.status === 'newer') {
        set({ available: { version: r.version ?? '', url: r.url ?? '' } });
        push({ kind: 'info', text: `새 버전 ${r.version} 이 있습니다` });
      } else if (r.status === 'latest') push({ kind: 'ok', text: '최신 버전입니다' });
      else if (r.status === 'unconfigured') push({ kind: 'ok', text: '배포 저장소가 설정되지 않은 빌드입니다' });
      else push({ kind: 'error', text: '확인하지 못했습니다 (네트워크를 확인하세요)' });
    } catch (e) {
      push({ kind: 'error', text: `확인하지 못했습니다 (${errorMessage(e)})` });
    }
  },
}));

/** update.available 이벤트를 받아 상태와 안내 토스트를 갱신한다. 구독 해제 함수를 돌려준다. */
export function subscribeUpdateEvents(): () => void {
  const off = getBackend().on('update.available', (u: UpdateInfo) => {
    useUpdate.getState().setAvailable({ version: u.version, url: u.url });
    if (u.announce) {
      useApp.getState().pushToast({ kind: 'info', text: `새 버전 ${u.version} 이 있습니다 (상태 표시줄을 눌러 확인)` });
    }
  });
  return off;
}
