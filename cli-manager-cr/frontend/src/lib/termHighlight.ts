// 키워드 강조를 터미널(xterm)에 그린다. 근거: spec/behavior/terminal.md §9 (D113).
// 지금 화면에 보이는 줄만 살펴 일치 구간에 강조(decoration)를 얹고, 출력·스크롤·크기 변경·규칙 변경 때 다시 한다.
// 서버로 가는 입력과 받은 출력 내용은 바꾸지 않는다(복사하면 원래 글자 그대로다).
import type { IDisposable, Terminal } from '@xterm/xterm';
import type { HighlightRule } from '../ipc/types';
import { compileRules, findMatches, lineText, matchCells, type CompiledRule } from './highlight';

/** 다시 그리기를 모으는 시간(ms). 출력이 이어지는 동안에는 이 간격으로 한 번씩만 그린다. */
export const HIGHLIGHT_REFRESH_MS = 80;
/** 한 화면에 얹는 강조 개수 한도. */
export const HIGHLIGHT_MAX_ON_SCREEN = 400;

export interface Highlighter {
  /** 규칙을 바꾼다(같은 배열이면 아무 일도 하지 않는다). */
  setRules(rules: readonly HighlightRule[] | undefined): void;
  dispose(): void;
}

/** 터미널에 키워드 강조를 붙인다. */
export function attachHighlighter(term: Terminal, initial: readonly HighlightRule[] | undefined): Highlighter {
  let source = initial;
  let rules: CompiledRule[] = compileRules(initial);
  let drawn: IDisposable[] = [];
  let timer: ReturnType<typeof setTimeout> | null = null;
  let disposed = false;

  const clear = () => {
    for (const d of drawn) d.dispose();
    drawn = [];
  };

  const draw = () => {
    timer = null;
    if (disposed) return;
    clear();
    if (rules.length === 0) return;
    const buf = term.buffer.active;
    let count = 0;
    for (let row = 0; row < term.rows && count < HIGHLIGHT_MAX_ON_SCREEN; row++) {
      const y = buf.viewportY + row;
      const line = buf.getLine(y);
      if (!line) continue;
      const { text, colOf } = lineText(line, term.cols);
      if (!text.trim()) continue;
      for (const m of findMatches(text, rules)) {
        if (count >= HIGHLIGHT_MAX_ON_SCREEN) break;
        // 표식은 일반 화면 버퍼에서만 만들 수 있다(vim·htop 같은 전체 화면 프로그램에서는 undefined → 강조하지 않는다).
        const marker = term.registerMarker(y - buf.baseY - buf.cursorY);
        if (!marker) return;
        const cells = matchCells(m, colOf);
        const deco = term.registerDecoration({
          marker,
          x: cells.x,
          width: cells.width,
          ...(m.background ? { backgroundColor: m.color, foregroundColor: '#ffffff' } : { foregroundColor: m.color }),
        });
        count++;
        // 강조를 먼저, 표식을 나중에 지운다(TerminalView 의 찾기 강조와 같은 이유).
        drawn.push({
          dispose() {
            deco?.dispose();
            marker.dispose();
          },
        });
      }
    }
  };

  const schedule = () => {
    if (disposed || timer !== null) return;
    if (rules.length === 0 && drawn.length === 0) return; // 규칙이 없으면 출력마다 타이머를 걸지 않는다
    timer = setTimeout(draw, HIGHLIGHT_REFRESH_MS);
  };

  const subs: IDisposable[] = [
    term.onWriteParsed(schedule),
    term.onScroll(schedule),
    term.onResize(schedule),
  ];
  schedule();

  return {
    setRules(next) {
      if (next === source) return;
      source = next;
      rules = compileRules(next);
      if (rules.length === 0) {
        if (timer !== null) {
          clearTimeout(timer);
          timer = null;
        }
        clear();
        return;
      }
      schedule();
    },
    dispose() {
      disposed = true;
      if (timer !== null) clearTimeout(timer);
      timer = null;
      for (const s of subs) s.dispose();
      clear();
    },
  };
}
