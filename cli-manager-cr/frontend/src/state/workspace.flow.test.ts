// 출력 흐름 제어(ack)와 숨은 세션 몰아쓰기 시험. 근거: spec/behavior/session.md §8, spec/ipc.md session.ack.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SessionInfo } from '../ipc/types';
import { setBackend, type Backend } from '../ipc/backend';
import { HIDDEN_FLUSH_MS, resetDeferredForTest, useWorkspace } from './workspace';
import { resetAcksForTest } from '../lib/outputAck';

const info = (id: string) => ({ id, title: id, state: 'connected' }) as unknown as SessionInfo;
const ws = () => useWorkspace.getState();
const invoke = vi.fn(async () => undefined);
const flushMicrotasks = () => Promise.resolve();

/** ack 호출만 모아 [id, 개수] 목록으로. */
function acks(): Array<[string, number]> {
  return invoke.mock.calls
    .filter((c) => (c as unknown[])[0] === 'session.ack')
    .map((c) => (c as unknown[])[1] as [string, number]);
}

beforeEach(() => {
  vi.useFakeTimers();
  invoke.mockClear();
  resetAcksForTest();
  resetDeferredForTest();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], groupRatio: 50, sessions: {}, writers: {} });
});

afterEach(() => {
  vi.useRealTimers();
  setBackend(null);
});

/** a, b 두 탭을 연다. 마지막에 연 b 가 보이고 a 는 숨는다. */
function openTwo(): void {
  ws().openInNewTab(info('a'));
  ws().upsertSession(info('a'));
  ws().openInNewTab(info('b'));
  ws().upsertSession(info('b'));
}

/** 쓴 텍스트를 모으고 done 을 바로 부르는 기록기. */
function recorder(): { written: string[]; fn: (text: string, done?: () => void) => void } {
  const written: string[] = [];
  return { written, fn: (text, done) => { written.push(text); done?.(); } };
}

describe('writeOutput 흐름 제어', () => {
  it('writer 가 없으면 출력은 버리고 ack 만 보낸다', async () => {
    ws().writeOutput('s1', 'abc');
    await flushMicrotasks();
    expect(acks()).toEqual([['s1', 1]]);
  });

  it('보이는 세션은 바로 쓰고, 한 틱의 done 여러 번은 ack 한 번으로 합친다', async () => {
    openTwo();
    const r = recorder();
    ws().registerWriter('b', r.fn);
    ws().writeOutput('b', 'x');
    ws().writeOutput('b', 'y');
    ws().writeOutput('b', 'z');
    expect(r.written).toEqual(['x', 'y', 'z']);
    await flushMicrotasks();
    expect(acks()).toEqual([['b', 3]]);
  });

  it('done 이 오기 전에는 ack 하지 않는다 (xterm 이 다 쓴 뒤)', async () => {
    openTwo();
    const pending: Array<() => void> = [];
    ws().registerWriter('b', (_text, done) => { if (done) pending.push(done); });
    ws().writeOutput('b', 'x');
    await flushMicrotasks();
    expect(acks()).toEqual([]);
    pending.forEach((d) => d());
    await flushMicrotasks();
    expect(acks()).toEqual([['b', 1]]);
  });

  it('숨은 세션은 100ms 뒤 한 번에 쓰고 묶음 수만큼 ack, 활동 표시를 켠다', async () => {
    openTwo();
    const r = recorder();
    ws().registerWriter('a', r.fn);
    ws().writeOutput('a', '1');
    ws().writeOutput('a', '2');
    ws().writeOutput('a', '3');
    expect(r.written).toEqual([]);
    expect(ws().sessions.a.activity).toBe(true);
    vi.advanceTimersByTime(HIDDEN_FLUSH_MS);
    expect(r.written).toEqual(['123']);
    await flushMicrotasks();
    expect(acks()).toEqual([['a', 3]]);
  });

  it('flushDeferred 는 타이머 전이라도 모아 둔 것을 바로 쓴다', () => {
    openTwo();
    const r = recorder();
    ws().registerWriter('a', r.fn);
    ws().writeOutput('a', 'p');
    ws().writeOutput('a', 'q');
    ws().flushDeferred('a');
    expect(r.written).toEqual(['pq']);
    vi.advanceTimersByTime(HIDDEN_FLUSH_MS);
    expect(r.written).toEqual(['pq']); // 타이머가 두 번 쓰지 않는다
  });

  it('숨어 있다 보이게 된 뒤의 새 출력은 모아 둔 것 다음에 쓴다 (순서 유지)', () => {
    openTwo();
    const r = recorder();
    ws().registerWriter('a', r.fn);
    ws().writeOutput('a', 'old');
    ws().activateTab(ws().tabs[0].id); // a 를 보이게
    ws().writeOutput('a', 'new');
    expect(r.written).toEqual(['old', 'new']);
  });

  it('removeSession 은 대기분을 버리고 ack 만 보낸다', async () => {
    openTwo();
    const r = recorder();
    ws().registerWriter('a', r.fn);
    ws().writeOutput('a', 'x');
    ws().writeOutput('a', 'y');
    ws().removeSession('a');
    vi.advanceTimersByTime(HIDDEN_FLUSH_MS);
    expect(r.written).toEqual([]);
    await flushMicrotasks();
    expect(acks()).toEqual([['a', 2]]);
  });

  it('done 을 받지 않는 기록기(인자 하나)는 쓰자마자 ack', async () => {
    openTwo();
    const written: string[] = [];
    ws().registerWriter('b', (text: string) => { written.push(text); });
    ws().writeOutput('b', 'k');
    expect(written).toEqual(['k']);
    await flushMicrotasks();
    expect(acks()).toEqual([['b', 1]]);
  });
});
