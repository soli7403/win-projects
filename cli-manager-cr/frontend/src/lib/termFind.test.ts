// 터미널 찾기의 찾은 항목 표시 검사(가짜 터미널). 근거: spec/ui/panes.md §2.4, KI-011.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Terminal } from '@xterm/xterm';
import { attachFindMarks, findInText, logicalLine, FIND_RULER_MAX_ROWS, FIND_RULER_REFRESH_MS, FIND_VIEW_REFRESH_MS } from './termFind';

interface Deco { x: number; width: number; bg?: string; ruler?: string; line: number; disposed: boolean }
interface Row { s: string; wrapped?: boolean }

const COLS = 10;

function fakeLine(r: Row) {
  return {
    length: COLS,
    isWrapped: !!r.wrapped,
    getCell: (x: number) => ({ getChars: () => r.s[x] ?? '', getWidth: () => 1 }),
    translateToString: () => r.s,
  };
}

/** 줄 배열을 가진 가짜 터미널. 표식 줄 번호는 버퍼 줄 번호 그대로 기록한다(baseY·cursorY = 0). */
function fakeTerm(rows: Row[], view: { rows: number; viewportY: number }) {
  const decos: Deco[] = [];
  const handlers: Record<string, (() => void)[]> = { write: [], scroll: [], resize: [] };
  const on = (k: string) => (fn: () => void) => {
    handlers[k].push(fn);
    return { dispose: () => { handlers[k] = handlers[k].filter((f) => f !== fn); } };
  };
  const state = { lines: rows, viewportY: view.viewportY };
  const term = {
    rows: view.rows,
    cols: COLS,
    buffer: {
      get active() {
        return {
          viewportY: state.viewportY,
          baseY: 0,
          cursorY: 0,
          length: state.lines.length,
          getLine: (y: number) => (state.lines[y] ? fakeLine(state.lines[y]) : undefined),
        };
      },
    },
    registerMarker: (offset: number) => ({ line: offset, dispose: () => undefined }),
    registerDecoration: (o: { marker: { line: number }; x: number; width: number; backgroundColor?: string; overviewRulerOptions?: { color: string } }) => {
      const d: Deco = { x: o.x, width: o.width, bg: o.backgroundColor, ruler: o.overviewRulerOptions?.color, line: o.marker.line, disposed: false };
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
    created: () => decos.length,
    fire: (k: 'write' | 'scroll' | 'resize') => handlers[k].forEach((f) => f()),
    handlerCount: () => handlers.write.length + handlers.scroll.length + handlers.resize.length,
  };
}

const colors = { matchBackground: '#553311', ruler: '#f59e0b' };

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('찾기 표시: 글 계산', () => {
  it('대소문자를 무시하고 특수 문자는 글자 그대로 찾는다', () => {
    expect(findInText('Error error ERROR', 'error')).toEqual([{ start: 0, end: 5 }, { start: 6, end: 11 }, { start: 12, end: 17 }]);
    expect(findInText('a.b axb', 'a.b')).toEqual([{ start: 0, end: 3 }]);
    expect(findInText('abc', '')).toEqual([]);
  });

  it('다음 줄로 이어진 글은 한 줄로 이어 붙인다', () => {
    const rows: Row[] = [{ s: 'xxxxxxxerr' }, { s: 'or here', wrapped: true }, { s: 'next' }];
    const ll = logicalLine((y) => (rows[y] ? fakeLine(rows[y]) : undefined), 0, COLS);
    expect(ll.text.startsWith('xxxxxxxerror here')).toBe(true);
    expect(ll.next).toBe(2);
    expect(ll.pos[7]).toEqual({ x: 7, y: 0 });
    expect(ll.pos[10]).toEqual({ x: 0, y: 1 });
  });
});

describe('찾기 표시: 터미널에 그리기', () => {
  it('글을 넣으면 곧바로 화면의 일치 항목에 바탕을 칠하고, 줄을 넘는 항목은 줄마다 칠한다', () => {
    const f = fakeTerm([{ s: 'xxxxxxxerr' }, { s: 'or ok', wrapped: true }, { s: 'ERROR' }], { rows: 3, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('error');
    const bg = f.live().filter((d) => d.bg).map((d) => ({ line: d.line, x: d.x, width: d.width }));
    expect(bg).toEqual([
      { line: 0, x: 7, width: 3 },
      { line: 1, x: 0, width: 2 },
      { line: 2, x: 0, width: 5 },
    ]);
  });

  it('출력이 쉬지 않고 이어져도 정해진 간격마다 다시 그린다(미루지 않는다)', () => {
    const f = fakeTerm([{ s: 'ok' }], { rows: 1, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('fail');
    expect(f.live().filter((d) => d.bg)).toHaveLength(0);
    f.state.lines = [{ s: 'fail' }];
    // 출력이 FIND_VIEW_REFRESH_MS 보다 짧은 간격으로 계속 온다.
    for (let i = 0; i < 5; i++) {
      f.fire('write');
      vi.advanceTimersByTime(FIND_VIEW_REFRESH_MS / 4);
    }
    expect(f.live().filter((d) => d.bg)).toHaveLength(1);
  });

  it('버퍼 맨 위가 아니라 최근 줄부터 스크롤 막대에 표시하고, 한도를 넘지 않는다', () => {
    const rows: Row[] = Array.from({ length: FIND_RULER_MAX_ROWS + 50 }, () => ({ s: 'hit' }));
    const f = fakeTerm(rows, { rows: 2, viewportY: rows.length - 2 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    const ruler = f.live().filter((d) => d.ruler).map((d) => d.line);
    expect(ruler).toHaveLength(FIND_RULER_MAX_ROWS);
    expect(Math.max(...ruler)).toBe(rows.length - 1);
    expect(Math.min(...ruler)).toBe(50);
  });

  it('스크롤 막대 표시는 같은 줄들이면 다시 만들지 않는다', () => {
    const f = fakeTerm([{ s: 'hit' }, { s: 'x' }], { rows: 2, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    const before = f.created();
    f.fire('write');
    vi.advanceTimersByTime(FIND_RULER_REFRESH_MS);
    // 화면 바탕 강조 1개만 다시 만들어지고 스크롤 막대 표시는 그대로다.
    expect(f.created() - before).toBe(1);
  });

  it('글을 비우거나 dispose 하면 표시를 모두 지우고 구독을 푼다', () => {
    const f = fakeTerm([{ s: 'hit' }], { rows: 1, viewportY: 0 });
    const m = attachFindMarks(f.term, colors);
    m.setQuery('hit');
    expect(f.live().length).toBeGreaterThan(0);
    m.setQuery('');
    expect(f.live()).toHaveLength(0);
    m.setQuery('hit');
    m.dispose();
    expect(f.live()).toHaveLength(0);
    expect(f.handlerCount()).toBe(0);
  });

  it('글이 없으면 출력이 와도 타이머를 걸지 않는다', () => {
    const f = fakeTerm([{ s: 'hit' }], { rows: 1, viewportY: 0 });
    attachFindMarks(f.term, colors);
    f.fire('write');
    expect(vi.getTimerCount()).toBe(0);
  });
});
