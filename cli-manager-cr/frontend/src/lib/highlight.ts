// 키워드 강조의 순수 계산: 규칙 → 정규식, 한 줄에서 일치 구간 찾기, 글자 위치 → 터미널 칸 위치.
// 근거: spec/behavior/terminal.md §9 (D113). 화면(xterm)에 그리는 부분은 termHighlight.ts.
import type { HighlightRule } from '../ipc/types';

/** 고를 수 있는 강조 색(이름은 문구 키 hl.color.<id>). 어두운·밝은 터미널 테마 모두에서 읽히는 중간 밝기다. */
export const HIGHLIGHT_COLORS: { id: string; hex: string }[] = [
  { id: 'red', hex: '#ef4444' },
  { id: 'orange', hex: '#f97316' },
  { id: 'yellow', hex: '#eab308' },
  { id: 'green', hex: '#22c55e' },
  { id: 'blue', hex: '#3b82f6' },
  { id: 'purple', hex: '#a855f7' },
];
export const HIGHLIGHT_DEFAULT_COLOR = HIGHLIGHT_COLORS[0].hex;

/** 규칙 수와 한 줄의 일치 수 한도. 넘는 것은 무시한다(출력이 많은 화면에서 그리기 비용을 막는다). */
export const HIGHLIGHT_MAX_RULES = 30;
export const HIGHLIGHT_MAX_PER_LINE = 50;

const HEX6 = /^#[0-9a-fA-F]{6}$/;

export interface CompiledRule {
  re: RegExp;
  color: string;
  /** 글자색 대신 바탕색으로 칠한다. */
  background: boolean;
}

export interface Match {
  /** 줄 글의 시작 위치(포함)와 끝 위치(제외), UTF-16 단위. */
  start: number;
  end: number;
  color: string;
  background: boolean;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 규칙의 패턴이 쓸 수 있는지(빈 값·틀린 정규식이 아닌지). 설정 화면이 입력 칸에 오류를 표시할 때 쓴다. */
export function isValidPattern(pattern: string, regex: boolean): boolean {
  if (!pattern) return false;
  if (!regex) return true;
  try {
    new RegExp(pattern);
    return true;
  } catch {
    return false;
  }
}

/** 켜져 있고 쓸 수 있는 규칙만 정규식으로 만든다. 앞에 있는 규칙이 우선이다. */
export function compileRules(rules: readonly HighlightRule[] | undefined): CompiledRule[] {
  const out: CompiledRule[] = [];
  for (const r of rules ?? []) {
    if (out.length >= HIGHLIGHT_MAX_RULES) break;
    if (!r || !r.enabled || typeof r.pattern !== 'string' || !isValidPattern(r.pattern, !!r.regex)) continue;
    try {
      out.push({
        re: new RegExp(r.regex ? r.pattern : escapeRegExp(r.pattern), r.caseSensitive ? 'g' : 'gi'),
        color: HEX6.test(r.color) ? r.color : HIGHLIGHT_DEFAULT_COLOR,
        background: !!r.background,
      });
    } catch {
      // 만들 수 없는 정규식은 건너뛴다
    }
  }
  return out;
}

/**
 * 한 줄의 글에서 일치 구간을 찾는다. 겹치면 앞 규칙이 이기고, 뒤 규칙의 겹치는 일치는 버린다.
 * 결과는 시작 위치 순이다. 빈 일치는 건너뛴다.
 */
export function findMatches(text: string, rules: readonly CompiledRule[]): Match[] {
  const out: Match[] = [];
  if (!text) return out;
  for (const rule of rules) {
    rule.re.lastIndex = 0;
    let guard = 0;
    for (let m = rule.re.exec(text); m && guard < HIGHLIGHT_MAX_PER_LINE; m = rule.re.exec(text)) {
      guard++;
      if (m[0].length === 0) {
        rule.re.lastIndex++; // 빈 일치(a* 등): 제자리에서 맴돌지 않게 한 칸 나아간다
        if (rule.re.lastIndex > text.length) break;
        continue;
      }
      const start = m.index;
      const end = start + m[0].length;
      if (!out.some((o) => start < o.end && o.start < end)) out.push({ start, end, color: rule.color, background: rule.background });
    }
  }
  return out.sort((a, b) => a.start - b.start).slice(0, HIGHLIGHT_MAX_PER_LINE);
}

/** xterm 버퍼 줄에서 쓰는 부분만 뽑은 모양(테스트에서 가짜로 만들 수 있게). */
export interface CellLine {
  length: number;
  getCell(x: number): { getChars(): string; getWidth(): number } | undefined;
}

/**
 * 버퍼 줄을 글로 바꾸고, 글의 각 위치가 몇 번째 칸인지 적은 표를 만든다.
 * 한글 같은 전각 글자는 두 칸을 차지하고 둘째 칸은 폭 0 으로 들어 있으므로 건너뛴다.
 * colOf[i] = text[i] 가 시작하는 칸, colOf[text.length] = 글이 끝난 다음 칸.
 */
export function lineText(line: CellLine, cols: number): { text: string; colOf: number[] } {
  let text = '';
  const colOf: number[] = [];
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
    for (let i = 0; i < chars.length; i++) colOf.push(x);
    text += chars;
    x += width;
  }
  colOf.push(x);
  return { text, colOf };
}

/** 일치 구간(글 위치)을 터미널 칸 범위로 바꾼다. */
export function matchCells(m: Match, colOf: readonly number[]): { x: number; width: number } {
  const x = colOf[m.start] ?? 0;
  const end = colOf[m.end] ?? colOf[colOf.length - 1] ?? x;
  return { x, width: Math.max(1, end - x) };
}

/** 새 규칙의 기본값. */
export function newHighlightRule(): HighlightRule {
  return { id: crypto.randomUUID(), pattern: '', color: HIGHLIGHT_DEFAULT_COLOR, regex: false, caseSensitive: false, background: false, enabled: true };
}
