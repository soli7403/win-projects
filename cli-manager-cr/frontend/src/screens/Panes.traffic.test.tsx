// 창 머리글의 세션 전송량 (D160, spec/ui/panes.md §2.7): 짧은 표기·고정 폭, 바뀐 세션만 새 객체, 닫으면 지움.
import { afterEach, describe, expect, it } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import { PaneTraffic } from './Panes';
import { useTraffic } from '../state/traffic';
import { formatBytesCompact } from '../lib/format';

afterEach(() => useTraffic.setState({ bySession: {} }));

describe('formatBytesCompact (D160)', () => {
  it('늘 5글자 이하, 단위 경계에서 올림', () => {
    const cases: [number, string][] = [
      [0, '0 B'],
      [999, '999 B'],
      [1000, '1.0 K'],
      [1024 * 9.9, '9.9 K'],
      [1024 * 9.96, '10 K'],
      [1024 * 999, '999 K'],
      [1024 * 999.6, '1.0 M'],
      [1024 ** 2 * 512, '512 M'],
      [1024 ** 3 * 2.5, '2.5 G'],
      [1024 ** 4 * 3, '3.0 T'],
      [-5, '0 B'],
      [Number.NaN, '0 B'],
    ];
    for (const [n, want] of cases) expect(formatBytesCompact(n)).toBe(want);
    for (let n = 1; n < 1024 ** 5; n = Math.ceil(n * 1.37)) expect(formatBytesCompact(n).length).toBeLessThanOrEqual(5);
  });
});

describe('useTraffic (D160)', () => {
  it('바뀐 세션만 새 객체로 바꾸고, 같은 값이면 상태를 그대로 둔다', () => {
    useTraffic.getState().apply({ a: { in: 1, out: 2 }, b: { in: 3, out: 4 } });
    const first = useTraffic.getState().bySession;
    useTraffic.getState().apply({ a: { in: 10, out: 2 } });
    const second = useTraffic.getState().bySession;
    expect(second.a).toEqual({ in: 10, out: 2 });
    expect(second.b).toBe(first.b);
    useTraffic.getState().apply({ a: { in: 10, out: 2 } });
    expect(useTraffic.getState().bySession).toBe(second);
    useTraffic.getState().remove('a');
    expect(useTraffic.getState().bySession.a).toBeUndefined();
    useTraffic.getState().apply(null);
    expect(useTraffic.getState().bySession.b).toEqual({ in: 3, out: 4 });
  });
});

describe('PaneTraffic (D160)', () => {
  it('값이 없으면 0, 이벤트가 오면 ↑보냄 ↓받음 으로 바뀐다. 툴팁은 정확한 단위', () => {
    render(<PaneTraffic sessionId="s1" live />);
    const el = screen.getByTestId('pane-traffic');
    expect(el.textContent).toBe('↑0 B↓0 B');
    expect([...el.querySelectorAll('.v')].map((x) => x.textContent)).toEqual(['↑0 B', '↓0 B']);
    act(() => useTraffic.getState().apply({ s1: { in: 2048, out: 12 } }));
    expect(el.textContent).toBe('↑12 B↓2.0 K');
    expect(el.getAttribute('title')).toBe('전송량 (세션을 연 뒤) — 보냄(입력) 12 B · 받음(출력) 2.0 KB');
    expect(el.className).toBe('pane-traffic');
    // 숫자 칸은 고정 폭 클래스(제목줄이 흔들리지 않게)
    expect(el.querySelectorAll('.v')).toHaveLength(2);
  });

  it('연결이 끊긴 세션은 흐리게(idle) 마지막 값을 둔다', () => {
    useTraffic.getState().apply({ s2: { in: 5, out: 1 } });
    render(<PaneTraffic sessionId="s2" live={false} />);
    const el = screen.getByTestId('pane-traffic');
    expect(el.className).toContain('idle');
    expect(el.textContent).toBe('↑1 B↓5 B');
  });
});
