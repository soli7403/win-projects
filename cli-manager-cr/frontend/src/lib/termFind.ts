// 터미널 찾기의 "찾은 항목" 표시(현재 항목 제외). 근거: spec/ui/panes.md §2.4, KI-011.
// 검색 애드온에 강조를 맡기면 애드온이 출력마다 다시 찾으며 터미널 선택을 바꾼다(자동 복사·사용자 선택을 덮음).
// 그래서 애드온은 현재 항목 이동에만 쓰고, 찾은 항목의 바탕·스크롤 막대 표시는 여기서 직접 그린다.
// - 화면에 보이는 줄: 바탕색 강조. FIND_VIEW_REFRESH_MS 간격(출력이 이어져도 미루지 않는다).
// - 스크롤 막대: 버퍼 전체에서 찾되 최근 줄부터 FIND_RULER_MAX_ROWS 줄까지, FIND_RULER_REFRESH_MS 간격.
// - 화면 폭 때문에 다음 줄로 이어진 글(isWrapped)도 한 줄로 이어 붙여 찾는다.
import type { IDisposable, Terminal } from '@xterm/xterm';
import { rowRanges } from './findHighlight';

export const FIND_VIEW_REFRESH_MS = 100;
export const FIND_RULER_REFRESH_MS = 1000;
/** 화면 하나에 얹는 바탕 강조 한도. */
export const FIND_MAX_ON_SCREEN = 400;
/** 스크롤 막대에 표시하는 줄 수 한도(최근 줄부터). */
export const FIND_RULER_MAX_ROWS = 1000;
/** 이어진 한 줄에서 찾는 일치 수 한도. */
const FIND_MAX_PER_LINE = 200;

export interface FindColors {
  /** 찾은 항목의 바탕(#RRGGBB). */
  matchBackground: string;
  /** 스크롤 막대 표시 색. */
  ruler: string;
}

export interface FindMarks {
  /** 찾는 글을 바꾼다. 빈 글이면 표시를 모두 지운다. */
  setQuery(q: string): void;
  dispose(): void;
}

interface CellLine {
  readonly length: number;
  readonly isWrapped: boolean;
  getCell(x: number): { getChars(): string; getWidth(): number } | undefined;
  translateToString(trimRight?: boolean): string;
}

interface Pos { x: number; y: number }

/** 버퍼 줄 y 에서 시작하는 이어진 줄의 글과 글 위치별 칸 위치. 다음 줄 번호도 돌려준다. */
export function logicalLine(getLine: (y: number) => CellLine | undefined, y: number, cols: number): { text: string; pos: Pos[]; next: number } {
  let text = '';
  const pos: Pos[] = [];
  let row = y;
  let line = getLine(row);
  let lastEnd: Pos = { x: 0, y };
  while (line) {
    let x = 0;
    const n = Math.min(cols, line.length);
    while (x < n) {
      const cell = line.getCell(x);
      const width = cell?.getWidth() ?? 1;
      if (width === 0) {
        x++;
        continue;
      }
      const chars = cell?.getChars() || ' ';
      for (let i = 0; i < chars.length; i++) pos.push({ x, y: row });
      text += chars;
      x += width;
    }
    lastEnd = { x, y: row };
    row++;
    line = getLine(row);
    if (!line?.isWrapped) break;
  }
  pos.push(lastEnd);
  return { text, pos, next: Math.max(row, y + 1) };
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 글에서 q 의 일치 구간(대소문자 무시)을 찾는다. */
export function findInText(text: string, q: string): { start: number; end: number }[] {
  if (!q) return [];
  const re = new RegExp(escapeRegExp(q), 'gi');
  const out: { start: number; end: number }[] = [];
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) && out.length < FIND_MAX_PER_LINE) {
    if (m[0].length === 0) {
      re.lastIndex++;
      continue;
    }
    out.push({ start: m.index, end: m.index + m[0].length });
  }
  return out;
}

/** 터미널에 찾은 항목 표시를 붙인다. */
export function attachFindMarks(term: Terminal, colors: FindColors): FindMarks {
  let query = '';
  let viewDrawn: IDisposable[] = [];
  let rulerDrawn: IDisposable[] = [];
  let rulerKey = '';
  let viewTimer: ReturnType<typeof setTimeout> | null = null;
  let rulerTimer: ReturnType<typeof setTimeout> | null = null;
  let disposed = false;

  const clearList = (list: IDisposable[]) => {
    for (const d of list) d.dispose();
  };

  // 표식은 커서 기준 상대 줄로 만든다. 대체 화면(vim 등)에서는 undefined.
  const mark = (row: number, x: number, width: number, bg: boolean, ruler: boolean): IDisposable | null => {
    const buf = term.buffer.active;
    const marker = term.registerMarker(row - buf.baseY - buf.cursorY);
    if (!marker) return null;
    const deco = term.registerDecoration({
      marker,
      x,
      width: Math.max(1, width),
      ...(bg ? { backgroundColor: colors.matchBackground } : {}),
      ...(ruler ? { overviewRulerOptions: { color: colors.ruler, position: 'center' as const } } : {}),
    });
    // 강조를 먼저, 표식을 나중에 지운다(TerminalView 의 현재 항목 강조와 같은 이유).
    return {
      dispose() {
        deco?.dispose();
        marker.dispose();
      },
    };
  };

  const cols = () => term.cols;
  const getLine = (y: number) => term.buffer.active.getLine(y) as CellLine | undefined;

  const drawView = () => {
    viewTimer = null;
    if (disposed) return;
    clearList(viewDrawn);
    viewDrawn = [];
    if (!query) return;
    const buf = term.buffer.active;
    const top = buf.viewportY;
    const bottom = top + term.rows; // 제외
    // 화면 맨 윗줄이 이어진 줄의 중간이면 그 줄의 시작부터 본다.
    let y = top;
    while (y > 0 && getLine(y)?.isWrapped) y--;
    let count = 0;
    while (y < bottom && count < FIND_MAX_ON_SCREEN) {
      const ll = logicalLine(getLine, y, cols());
      for (const m of findInText(ll.text, query)) {
        const s = ll.pos[m.start];
        const e = ll.pos[m.end];
        if (!s || !e) continue;
        for (const r of rowRanges(s, e, cols())) {
          if (r.row < top || r.row >= bottom || count >= FIND_MAX_ON_SCREEN) continue;
          const d = mark(r.row, r.x, r.width, true, false);
          if (!d) return;
          viewDrawn.push(d);
          count++;
        }
      }
      y = ll.next;
    }
  };

  const drawRuler = () => {
    rulerTimer = null;
    if (disposed) return;
    const rows: number[] = [];
    if (query) {
      const buf = term.buffer.active;
      // 최근 줄부터: 맨 아래에서 이어진 줄의 시작을 찾아 위로 올라간다.
      let end = buf.length; // 제외
      while (end > 0 && rows.length < FIND_RULER_MAX_ROWS) {
        let start = end - 1;
        while (start > 0 && getLine(start)?.isWrapped) start--;
        // 먼저 빠른 글 변환으로 거른다(칸 단위로 위치를 계산하는 것은 일치하는 줄만).
        let quick = '';
        for (let y = start; y < end; y++) quick += getLine(y)?.translateToString(false) ?? '';
        if (!quick.toLowerCase().includes(query.toLowerCase())) {
          end = start;
          continue;
        }
        const ll = logicalLine(getLine, start, cols());
        const hit = findInText(ll.text, query);
        const seen = new Set<number>();
        for (const m of hit) {
          const s = ll.pos[m.start];
          if (s && !seen.has(s.y)) seen.add(s.y);
        }
        for (const r of [...seen].sort((a, b) => b - a)) if (rows.length < FIND_RULER_MAX_ROWS) rows.push(r);
        end = start;
      }
    }
    // 같은 줄들이면(그리고 버퍼가 밀리지 않았으면) 다시 만들지 않는다.
    const key = `${term.buffer.active.baseY}:${rows.join(',')}`;
    if (key === rulerKey) return;
    rulerKey = key;
    clearList(rulerDrawn);
    rulerDrawn = [];
    for (const r of rows) {
      const d = mark(r, 0, 1, false, true);
      if (!d) break;
      rulerDrawn.push(d);
    }
  };

  const schedule = () => {
    if (disposed || !query) return;
    if (viewTimer === null) viewTimer = setTimeout(drawView, FIND_VIEW_REFRESH_MS);
    if (rulerTimer === null) rulerTimer = setTimeout(drawRuler, FIND_RULER_REFRESH_MS);
  };

  const stopTimers = () => {
    if (viewTimer !== null) clearTimeout(viewTimer);
    if (rulerTimer !== null) clearTimeout(rulerTimer);
    viewTimer = null;
    rulerTimer = null;
  };

  const subs: IDisposable[] = [
    term.onWriteParsed(schedule),
    term.onScroll(schedule),
    term.onResize(schedule),
  ];

  return {
    setQuery(q) {
      if (q === query) return;
      query = q;
      stopTimers();
      // 글이 바뀌면 곧바로 그린다(입력할 때마다 0.1초 기다리지 않게).
      drawView();
      drawRuler();
    },
    dispose() {
      disposed = true;
      stopTimers();
      for (const s of subs) s.dispose();
      clearList(viewDrawn);
      clearList(rulerDrawn);
      viewDrawn = [];
      rulerDrawn = [];
    },
  };
}
