// 끌어 놓기 복사 작업 목록 상태. 근거: spec/behavior/file-panel.md §3, §5(복사 작업 갱신 이벤트).
import { create } from 'zustand';

export interface CopyJob {
  id: string;
  srcSession: string;
  srcTitle: string;
  srcDir: string;
  dstSession: string;
  dstTitle: string;
  dstDir: string;
  /** 첫 항목 이름 / 끌어 온 항목 수 */
  name: string;
  count: number;
  files: number;
  filesDone: number;
  bytes: number;
  /** 서버 → 서버면 받기 + 올리기 합(원본 크기의 2배) */
  total: number;
  relay: boolean;
  phase: 'scan' | 'download' | 'upload' | 'copy';
  state: 'queued' | 'running' | 'done' | 'error' | 'canceled';
  error?: string;
  startedAt: number;
  endedAt?: number;
}

export const isActiveJob = (j: CopyJob): boolean => j.state === 'queued' || j.state === 'running';

interface CopyJobState {
  /** 오래된 것이 앞 */
  items: CopyJob[];
  upsert: (j: CopyJob) => void;
  remove: (ids: string[]) => void;
}

export const useCopyJobs = create<CopyJobState>((set) => ({
  items: [],
  upsert(j) {
    set((s) => {
      const idx = s.items.findIndex((x) => x.id === j.id);
      if (idx < 0) return { items: [...s.items, j] };
      const next = s.items.slice();
      next[idx] = j;
      return { items: next };
    });
  },
  remove(ids) {
    set((s) => ({ items: s.items.filter((x) => !ids.includes(x.id)) }));
  },
}));
