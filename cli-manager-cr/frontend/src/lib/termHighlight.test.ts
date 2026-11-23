// 키워드 강조를 터미널에 그리는 부분의 검사(가짜 터미널). 근거: spec/behavior/terminal.md §9 (D113).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Terminal } from '@xterm/xterm';
import type { HighlightRule } from '../ipc/types';
import { attachHighlighter, HIGHLIGHT_REFRESH_MS } from './termHighlight';

interface Deco { x: number; width: number; fg?: string; bg?: string; line: number; disposed: boolean }

/** 화면 줄(문자열 배열)을 가진 가짜 터미널. 강조와 표식을 기록한다. */
function fakeTerm(lines: string[], opts: { altScreen?: boolean } = {}) {
  const decos: Deco[] = [];
  const handlers: Record<string, (() => void)[]> = { write: [], scroll: [], resize: [] };
  let liveMarkers = 0;
  const on = (k: string) => (fn: () => void) => {
    handlers[k].push(fn);
    return { dispose: () => { handlers[k] = handlers[k].filter((f) => f !== fn); } };
  };
  const state = { lines, viewportY: 0 };
  const term = {
    rows: 3,
    cols: 20,
    buffer: {
      get active() {
        return {
          viewportY: state.viewportY,
          baseY: 0,
          cursorY: 0,
          getLine: (y: number) => {
            const s = state.lines[y];
            if (s === undefined) return undefined;
            return { length: 20, getCell: (x: number) => ({ getChars: () => s[x] ?? '', getWidth: () => 1 }) };
          },
        };
      },
    },
    registerMarker: (offset: number) => {
      if (opts.altScreen) return undefined;
      liveMarkers++;
      return { line: offset, dispose: () => { liveMarkers--; } };
    },
    registerDecoration: (o: { marker: { line: number }; x: number; width: number; foregroundColor?: string; backgroundColor?: string }) => {
      const d: Deco = { x: o.x, width: o.width, fg: o.foregroundColor, bg: o.backgroundColor, line: o.marker.line, disposed: false };
      decos.push(d);
      return { dispose: () => { d.disposed = true; } };
    },
    onWriteParsed: on('write'),
    onScroll: on('scroll'),
    onResize: on('resize'),
  };
  return {
    term: term as unknown as Terminal,
    state,
    live: () => decos.filter((d) => !d.disposed),
    liveMarkers: () => liveMarkers,
    fire: (k: 'write' | 'scroll' | 'resize') => handlers[k].forEach((f) => f()),
    handlerCount: () => handlers.write.length + handlers.scroll.length + handlers.resize.length,
  };
}

const rule = (p: Partial<HighlightRule>): HighlightRule => ({
  id: p.pattern ?? 'x', pattern: '', color: '#ef4444', regex: false, caseSensitive: false, background: false, enabled: true, ...p,
});

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('키워드 강조: 터미널에 그리기', () => {
  it('보이는 줄의 일치 구간에 강조를 얹는다(글자색 / 바탕색)', () => {
    const f = fakeTerm(['ok', 'an ERROR here', 'WARN x']);
    attachHighlighter(f.term, [rule({ pattern: 'error' }), rule({ pattern: 'warn', color: '#eab308', background: true })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => ({ line: d.line, x: d.x, width: d.width, fg: d.fg, bg: d.bg }))).toEqual([
      { line: 1, x: 3, width: 5, fg: '#ef4444', bg: undefined },
      { line: 2, x: 0, width: 4, fg: '#ffffff', bg: '#eab308' },
    ]);
  });

  it('출력이 이어져도 모아서 한 번만 다시 그리고, 이전 강조와 표식은 지운다', () => {
    const f = fakeTerm(['error', '', '']);
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(1);
    f.state.lines = ['error', 'error', 'x'];
    for (let i = 0; i < 20; i++) f.fire('write');
    expect(f.live()).toHaveLength(1); // 아직 다시 그리지 않았다
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(2);
    expect(f.liveMarkers()).toBe(2);
  });

  it('스크롤하면 새로 보이는 줄을 강조한다', () => {
    const f = fakeTerm(['a', 'b', 'c', 'error', 'e']);
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(0);
    f.state.viewportY = 2;
    f.fire('scroll');
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => d.line)).toEqual([3]);
  });

  it('규칙을 바꾸면 다시 그리고, 규칙을 비우면 모두 지운 뒤 출력에도 타이머를 걸지 않는다', () => {
    const f = fakeTerm(['error warn', '', '']);
    const h = attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    h.setRules([rule({ pattern: 'warn' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => d.x)).toEqual([6]);
    h.setRules([]);
    expect(f.live()).toHaveLength(0);
    expect(f.liveMarkers()).toBe(0);
    f.fire('write');
    expect(vi.getTimerCount()).toBe(0);
  });

  it('전체 화면 프로그램(대체 화면)에서는 강조하지 않는다', () => {
    const f = fakeTerm(['error', '', ''], { altScreen: true });
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(0);
  });

  it('dispose 하면 강조·표식·이벤트 연결을 모두 정리한다', () => {
    const f = fakeTerm(['error', '', '']);
    const h = attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    h.dispose();
    expect(f.live()).toHaveLength(0);
    expect(f.liveMarkers()).toBe(0);
    expect(f.handlerCount()).toBe(0);
    expect(vi.getTimerCount()).toBe(0);
  });
});
