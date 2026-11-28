// 세션 출력·입력을 지켜보는 곳(출력 트리거, 매크로)이 듣는 창구. 근거: spec/behavior/terminal.md §10.1 (D140).
// 터미널 화면에 쓰는 경로(state/workspace.ts writeOutput)와는 따로다 — 듣는 쪽이 느려도 화면 출력을 막지 않는다.

type OutputListener = (sessionId: string, text: string) => void;
/** fromUser: 사용자가 터미널에 직접 친 입력이면 true, 앱이 대신 보낸 입력(명령 모음·매크로·자동 응답)이면 false. */
type InputListener = (sessionId: string, data: string, fromUser: boolean) => void;

const outputListeners = new Set<OutputListener>();
const inputListeners = new Set<InputListener>();

/** 백엔드의 session.data 묶음마다 부른다. 듣는 쪽의 오류는 다른 쪽에 번지지 않게 삼킨다. */
export function emitOutput(sessionId: string, text: string): void {
  for (const fn of outputListeners) {
    try {
      fn(sessionId, text);
    } catch {
      /* 듣는 쪽 오류는 출력 흐름과 무관 */
    }
  }
}

function emitInput(sessionId: string, data: string, fromUser: boolean): void {
  for (const fn of inputListeners) {
    try {
      fn(sessionId, data, fromUser);
    } catch {
      /* 무시 */
    }
  }
}

/** 사용자가 터미널에 친 입력마다 부른다(TerminalView). */
export function emitUserInput(sessionId: string, data: string): void {
  emitInput(sessionId, data, true);
}

/** 앱이 대신 보낸 입력(명령 모음·매크로·자동 응답)마다 부른다. 그 에코에 출력 트리거가 반응하지 않게 알린다. */
export function emitAppInput(sessionId: string, data: string): void {
  emitInput(sessionId, data, false);
}

export function onOutput(fn: OutputListener): () => void {
  outputListeners.add(fn);
  return () => outputListeners.delete(fn);
}

export function onInput(fn: InputListener): () => void {
  inputListeners.add(fn);
  return () => inputListeners.delete(fn);
}
