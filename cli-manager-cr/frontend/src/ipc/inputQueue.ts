// 세션별 터미널 입력 줄 세우기. 근거: spec/ipc.md §1(한 세션의 입력은 보낸 순서대로 전송에 쓰인다).
// Wails v2 는 화면에서 온 호출마다 고루틴을 따로 띄워, 연달아 보낸 호출의 처리 순서를 보장하지 않는다.
// 한 세션의 호출을 한 번에 하나만 띄워 순서를 지키고, 기다리는 동안 온 입력은 이어 붙여 다음에 한 번에 보낸다.
// 입력은 바이트 흐름이라 이어 붙여도 서버가 받는 내용은 같다.

/** 실제 전송 함수. 끝나면 풀리는 Promise(또는 동기 값)를 돌려준다. */
export type InputSender = (sessionId: string, text: string) => unknown;

export function createInputQueue(send: InputSender): (sessionId: string, text: string) => void {
  // 값 = 앞 호출이 끝나기를 기다리는 동안 쌓인 입력. 항목이 있으면 "보내는 중".
  const pending = new Map<string, string>();

  const pump = (sessionId: string, chunk: string): void => {
    let done: Promise<unknown>;
    try {
      done = Promise.resolve(send(sessionId, chunk));
    } catch {
      done = Promise.resolve();
    }
    // 실패해도 다음 입력은 보낸다(실패한 입력은 예전처럼 조용히 버려진다).
    done
      .catch(() => undefined)
      .then(() => {
        const next = pending.get(sessionId);
        if (next === undefined) return;
        if (next) {
          pending.set(sessionId, '');
          pump(sessionId, next);
        } else {
          pending.delete(sessionId); // 한가해지면 지운다(닫힌 세션의 항목이 남지 않게)
        }
      });
  };

  return (sessionId, text) => {
    if (!text) return;
    const queued = pending.get(sessionId);
    if (queued !== undefined) {
      pending.set(sessionId, queued + text); // 보내는 중: 뒤에 이어 붙인다
      return;
    }
    pending.set(sessionId, '');
    pump(sessionId, text);
  };
}
