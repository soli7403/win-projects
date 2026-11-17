import { describe, expect, it } from 'vitest';
import { createInputQueue } from './inputQueue';

// 끝나는 시점을 테스트가 정하는 가짜 전송 함수.
function makeSender() {
  const calls: { id: string; text: string; resolve: () => void; reject: (e: unknown) => void }[] = [];
  const send = (id: string, text: string) =>
    new Promise<void>((resolve, reject) => {
      calls.push({ id, text, resolve, reject });
    });
  return { calls, send };
}

const flush = () => new Promise((r) => setTimeout(r, 0));

describe('createInputQueue', () => {
  it('넣은 순서대로 한 번에 하나씩 보낸다', async () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s', 'a');
    expect(calls.map((c) => c.text)).toEqual(['a']);
    input('s', 'b');
    expect(calls).toHaveLength(1); // 앞 호출이 끝나기 전에는 보내지 않는다
    calls[0].resolve();
    await flush();
    expect(calls.map((c) => c.text)).toEqual(['a', 'b']);
  });

  it('기다리는 동안 들어온 입력은 이어 붙여 한 번에 보낸다', async () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s', 'a');
    input('s', 'b');
    input('s', 'c');
    calls[0].resolve();
    await flush();
    expect(calls.map((c) => c.text)).toEqual(['a', 'bc']);
  });

  it('세션마다 줄이 따로라 다른 세션은 기다리지 않는다', () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s1', 'a');
    input('s2', 'x');
    expect(calls.map((c) => `${c.id}:${c.text}`)).toEqual(['s1:a', 's2:x']);
  });

  it('전송이 거부되거나 예외를 던져도 다음 입력은 간다', async () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s', 'a');
    input('s', 'b');
    calls[0].reject(new Error('fail'));
    await flush();
    expect(calls.map((c) => c.text)).toEqual(['a', 'b']);

    let n = 0;
    const sent: string[] = [];
    const throwing = createInputQueue((_id, text) => {
      n++;
      if (n === 1) throw new Error('sync fail');
      sent.push(text);
    });
    throwing('s', 'a');
    throwing('s', 'b');
    await flush();
    expect(sent).toEqual(['b']);
  });

  it('대기 입력이 없어지면 줄을 지우고 다음 입력은 다시 바로 간다', async () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s', 'a');
    calls[0].resolve();
    await flush();
    input('s', 'b');
    expect(calls.map((c) => c.text)).toEqual(['a', 'b']); // 기다리지 않고 바로
  });

  it('빈 문자열은 보내지 않는다', () => {
    const { calls, send } = makeSender();
    const input = createInputQueue(send);
    input('s', '');
    expect(calls).toHaveLength(0);
  });

  it('전송 함수가 Promise 가 아닌 값을 돌려줘도 동작한다', async () => {
    const sent: string[] = [];
    const input = createInputQueue((_id, text) => {
      sent.push(text);
    });
    input('s', 'a');
    input('s', 'b');
    input('s', 'c');
    await flush();
    expect(sent).toEqual(['a', 'bc']);
    input('s', 'd');
    await flush();
    expect(sent).toEqual(['a', 'bc', 'd']);
  });
});
