// 백엔드 인증·확인 요청(ui.request) 큐. 근거: spec/ipc.md §3, spec/ui/dialogs.md §4.
// 요청은 순서대로 한 번에 하나씩 보이며, 응답은 Reply(requestId, 값JSON) 으로 돌려준다.
import { create } from 'zustand';
import { getBackend } from '../ipc/backend';

export type UiRequest =
  | { requestId: string; kind: 'hostkey'; sessionTitle: string; hostPort: string; keyType: string; fingerprint: string; changed: boolean; oldFingerprint?: string }
  | { requestId: string; kind: 'password'; sessionTitle: string; prompt: string; username: string; allowSave: boolean; failed?: boolean }
  | { requestId: string; kind: 'keyboard'; sessionTitle: string; name: string; instructions: string; prompts: { prompt: string; echo: boolean }[] }
  | { requestId: string; kind: 'passphrase'; sessionTitle: string }
  | { requestId: string; kind: 'username'; sessionTitle: string };

interface RequestState {
  queue: UiRequest[];
  push: (r: UiRequest) => void;
  /** 맨 앞 요청에 답한다. value=null 이면 취소. */
  answer: (value: unknown) => void;
}

export const useRequests = create<RequestState>((set, get) => ({
  queue: [],
  push: (r) => set((s) => ({ queue: [...s.queue, r] })),
  answer(value) {
    const [first, ...rest] = get().queue;
    if (!first) return;
    set({ queue: rest });
    getBackend().reply(first.requestId, value);
  },
}));
