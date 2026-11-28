// 세션별 전송량(창 머리글 표시). 근거: spec/ipc.md session.traffic, spec/ui/panes.md §2.7 (D160).
// 작업 영역 저장소(sessions)와 따로 둔다: 1초마다 오는 값이 세션 목록·탭·칸 전체를 다시 그리지 않게,
// 바뀐 세션의 항목만 새 객체로 바꾸고 각 머리글은 자기 세션 항목만 구독한다.
import { create } from 'zustand';

export interface SessionTraffic {
  /** 세션을 연 뒤 연결에서 받은 바이트(서버 → 터미널 출력). */
  in: number;
  /** 세션을 연 뒤 연결로 보낸 바이트(키 입력·붙여넣기·명령 모음 등). */
  out: number;
}

interface TrafficState {
  bySession: Record<string, SessionTraffic>;
  /** session.traffic 이벤트 값(바뀐 세션만)을 합친다. 값이 같으면 객체를 바꾸지 않는다. */
  apply: (changed: Record<string, Partial<SessionTraffic>> | null | undefined) => void;
  /** 닫힌 세션의 항목을 지운다. */
  remove: (sessionId: string) => void;
}

export const useTraffic = create<TrafficState>((set) => ({
  bySession: {},
  apply(changed) {
    if (!changed || typeof changed !== 'object') return;
    set((s) => {
      let next: Record<string, SessionTraffic> | null = null;
      for (const [id, v] of Object.entries(changed)) {
        const nIn = Number(v?.in) || 0;
        const nOut = Number(v?.out) || 0;
        const cur = s.bySession[id];
        if (cur && cur.in === nIn && cur.out === nOut) continue;
        next ??= { ...s.bySession };
        next[id] = { in: nIn, out: nOut };
      }
      return next ? { bySession: next } : s;
    });
  },
  remove(sessionId) {
    set((s) => {
      if (!(sessionId in s.bySession)) return s;
      const next = { ...s.bySession };
      delete next[sessionId];
      return { bySession: next };
    });
  },
}));

/** 세션 하나의 전송량. 아직 받은 값이 없으면 undefined. */
export function useSessionTraffic(sessionId: string | undefined): SessionTraffic | undefined {
  return useTraffic((s) => (sessionId ? s.bySession[sessionId] : undefined));
}
