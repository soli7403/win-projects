// 출력 묶음 ack 를 세션별로 모아 microtask 한 번에 보낸다 (spec/ipc.md session.ack, behavior/session.md §8 출력 흐름 제어).
// xterm 은 한 처리 주기에 여러 묶음의 완료 콜백을 연달아 부르므로 microtask 로 합치면 세션당 주기마다 1회가 된다.
import { getBackend } from '../ipc/backend';

const pending = new Map<string, number>();
let scheduled = false;

/** 세션의 출력 묶음 count 개를 처리했다고 표시한다. 실제 전송은 microtask 에서 한 번에. */
export function ackOutput(sessionId: string, count = 1): void {
  if (count <= 0) return;
  pending.set(sessionId, (pending.get(sessionId) ?? 0) + count);
  if (scheduled) return;
  scheduled = true;
  queueMicrotask(flushAcks);
}

function flushAcks(): void {
  scheduled = false;
  const entries = [...pending];
  pending.clear();
  let b: ReturnType<typeof getBackend>;
  try {
    b = getBackend();
  } catch {
    return; // 백엔드 준비 전(테스트)이면 버린다
  }
  for (const [id, n] of entries) void b.invoke('session.ack', [id, n]).catch(() => undefined);
}

/** 테스트용: 쌓인 ack 를 비운다. */
export function resetAcksForTest(): void {
  pending.clear();
  scheduled = false;
}
