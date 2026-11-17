// 프런트↔백엔드 어댑터. 근거: spec/ipc.md §1, §4.
// 백엔드는 Wails 가 노출하는 전역 객체로 호출한다. 전역 객체의 정확한 이름과 이벤트 구독 함수는
// 명세에 없으므로(spec/QUESTIONS.md Q1) 아래 실제 어댑터는 Wails v2 관례(window.go.main.App, window.runtime)를 가정한다.
// 해당 전역이 없으면 가짜 백엔드(fakeBackend.ts)를 쓴다(개발 서버·테스트).

import { t } from '../i18n';
import { createInputQueue } from './inputQueue';

export type EventName =
  | 'session.data'
  | 'session.state'
  | 'ui.request'
  | 'transfer.update'
  | 'copyjob.update'
  | 'vault.changed'
  | 'store.changed'
  | 'toast'
  | 'sync.status'
  | 'update.available';

export type EventHandler = (...args: any[]) => void;

export interface Backend {
  /** 요청-응답. 값이 없으면 undefined. 오류는 Error 로 거부된다. */
  invoke<T = any>(channel: string, args?: unknown[]): Promise<T>;
  /** 터미널 입력. 응답 없음. */
  termInput(sessionId: string, text: string): void;
  /** 터미널 크기. 응답 없음. */
  termResize(sessionId: string, cols: number, rows: number): void;
  /** 백엔드가 낸 ui.request 에 대한 답. 응답 없음. */
  reply(requestId: string, value: unknown): void;
  /** 이벤트 구독. 구독 해제 함수를 돌려준다. */
  on(event: EventName, handler: EventHandler): () => void;
}

interface WailsAppBinding {
  Invoke(channel: string, argsJson: string): Promise<string>;
  /** Wails 는 백엔드 처리가 끝나면 풀리는 Promise 를 돌려준다. */
  TermInput(sessionId: string, text: string): Promise<void> | void;
  TermResize(sessionId: string, cols: number, rows: number): void;
  Reply(requestId: string, valueJson: string): void;
}

interface WailsRuntime {
  EventsOn(event: string, cb: (...data: any[]) => void): () => void;
}

declare global {
  interface Window {
    go?: { main?: { App?: WailsAppBinding } };
    runtime?: WailsRuntime;
  }
}

function errorMessage(e: unknown): string {
  if (typeof e === 'string') return e;
  if (e && typeof e === 'object' && 'message' in e) return String((e as { message: unknown }).message);
  return String(e);
}

/** Wails 전역이 있을 때의 실제 어댑터. */
export function createWailsBackend(app: WailsAppBinding, runtime: WailsRuntime | undefined): Backend {
  // 세션별로 한 번에 한 호출만 띄워 입력 순서를 지킨다(inputQueue.ts).
  const sendInput = createInputQueue((sessionId, text) => app.TermInput(sessionId, text));
  return {
    async invoke<T = any>(channel: string, args: unknown[] = []): Promise<T> {
      let raw: string;
      try {
        raw = await app.Invoke(channel, JSON.stringify(args));
      } catch (e) {
        throw new Error(errorMessage(e));
      }
      return (raw ? JSON.parse(raw) : undefined) as T;
    },
    termInput: (sessionId, text) => sendInput(sessionId, text),
    termResize: (sessionId, cols, rows) => app.TermResize(sessionId, cols, rows),
    reply: (requestId, value) => app.Reply(requestId, JSON.stringify(value ?? null)),
    on(event, handler) {
      if (!runtime) return () => undefined;
      return runtime.EventsOn(event, handler);
    },
  };
}

let current: Backend | null = null;

/** 현재 백엔드. 앱 시작 시 initBackend 로 정한 뒤 사용한다. */
export function getBackend(): Backend {
  if (!current) throw new Error(t('backend.notReady'));
  return current;
}

export function setBackend(b: Backend | null): void {
  current = b;
}

export function isRealBackend(): boolean {
  return typeof window !== 'undefined' && !!window.go?.main?.App;
}

/**
 * Wails 전역이 있으면 실제 어댑터, 없으면 개발·테스트용 가짜 백엔드를 고른다.
 * 가짜 백엔드 모듈은 필요할 때만 불러온다(운영 번들에서 분리).
 */
let initPromise: Promise<void> | null = null;

export function initBackend(): Promise<void> {
  // StrictMode 의 이중 실행에도 백엔드를 한 번만 만든다.
  if (!initPromise) {
    initPromise = (async () => {
      if (isRealBackend()) {
        current = createWailsBackend(window.go!.main!.App!, window.runtime);
        return;
      }
      const { createFakeBackend } = await import('./fakeBackend');
      current = createFakeBackend();
    })();
  }
  return initPromise;
}

export { errorMessage };
