// 전송 목록 상태. 근거: spec/ui/sftp-snippets.md §2.6, spec/ipc.md §4(transfer.update).
import { create } from 'zustand';

export interface TransferInfo {
  id: string;
  sessionId: string;
  name: string;
  direction: 'upload' | 'download';
  bytes: number;
  total: number;
  /** 진행 중·대기 중·완료·오류·취소 등. 진행 중·대기 중이 아닌 것을 "지우기"로 지운다. */
  state: string;
  error?: string;
  remotePath: string;
  localPath?: string;
}

/** 최신순 최대 30개(spec §2.6). */
const MAX = 30;

interface TransferState {
  items: TransferInfo[];
  upsert: (t: TransferInfo) => void;
  /** 진행 중·대기 중이 아닌 전송을 지운다. */
  clearFinished: () => void;
}

const ACTIVE_STATES = new Set(['queued', 'running', 'progress', 'pending']);

export function isActiveTransfer(t: TransferInfo): boolean {
  return ACTIVE_STATES.has(t.state);
}

export const useTransfers = create<TransferState>((set) => ({
  items: [],
  upsert(t) {
    set((s) => {
      const rest = s.items.filter((x) => x.id !== t.id);
      // 새 항목은 맨 앞(최신순). 기존 항목은 갱신하면서 위치를 유지한다.
      const idx = s.items.findIndex((x) => x.id === t.id);
      const next = idx >= 0 ? s.items.map((x) => (x.id === t.id ? t : x)) : [t, ...rest];
      return { items: next.slice(0, MAX) };
    });
  },
  clearFinished() {
    set((s) => ({ items: s.items.filter((t) => isActiveTransfer(t)) }));
  },
}));
