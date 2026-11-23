// 키워드 강조의 순수 계산 검사. 근거: spec/behavior/terminal.md §9, 수용 G9 (D113).
import { describe, expect, it } from 'vitest';
import type { HighlightRule } from '../ipc/types';
import { compileRules, findMatches, HIGHLIGHT_DEFAULT_COLOR, HIGHLIGHT_MAX_PER_LINE, isValidPattern, lineText, matchCells, type CellLine } from './highlight';

const rule = (p: Partial<HighlightRule>): HighlightRule => ({
  id: p.pattern ?? 'x', pattern: '', color: '#ef4444', regex: false, caseSensitive: false, background: false, enabled: true, ...p,
});

/** 가짜 버퍼 줄: 전각 글자는 폭 2 칸 + 폭 0 칸으로 넣는다(xterm 과 같은 모양). */
function fakeLine(s: string, cols = 80): CellLine {
  const cells: { c: string; w: number }[] = [];
  for (const ch of s) {
    const wide = /[ᄀ-ᇿ　-鿿가-힣＀-｠]/.test(ch);
    cells.push({ c: ch, w: wide ? 2 : 1 });
    if (wide) cells.push({ c: '', w: 0 });
  }
  while (cells.length < cols) cells.push({ c: '', w: 1 });
  return { length: cells.length, getCell: (x) => (cells[x] ? { getChars: () => cells[x].c, getWidth: () => cells[x].w } : undefined) };
}

describe('키워드 강조: 규칙 → 일치 구간', () => {
  it('보통 글자는 대소문자를 가리지 않고 모두 찾는다', () => {
    const rules = compileRules([rule({ pattern: 'error' })]);
    expect(findMatches('ERROR: disk Error at error.log', rules).map((m) => [m.start, m.end])).toEqual([[0, 5], [12, 17], [21, 26]]);
  });

  it('대소문자 구분을 켜면 같은 모양만 찾는다', () => {
    const rules = compileRules([rule({ pattern: 'FAIL', caseSensitive: true })]);
    expect(findMatches('fail FAIL Fail', rules)).toHaveLength(1);
  });

  it('보통 글자의 특수 문자는 글자 그대로 찾는다', () => {
    const rules = compileRules([rule({ pattern: '[WARN]' })]);
    expect(findMatches('x [WARN] W', rules).map((m) => [m.start, m.end])).toEqual([[2, 8]]);
  });

  it('정규식 규칙은 정규식으로 찾고, 빈 일치는 건너뛴다', () => {
    const rules = compileRules([rule({ pattern: '\\b\\d{3}\\b', regex: true }), rule({ pattern: 'x*', regex: true, color: '#22c55e' })]);
    const got = findMatches('code 500 and 404 xx', rules);
    expect(got.map((m) => [m.start, m.end, m.color])).toEqual([[5, 8, '#ef4444'], [13, 16, '#ef4444'], [17, 19, '#22c55e']]);
  });

  it('겹치면 앞 규칙이 이긴다', () => {
    const rules = compileRules([rule({ pattern: 'connection refused', color: '#ef4444' }), rule({ pattern: 'refused', color: '#3b82f6' })]);
    expect(findMatches('ssh: connection refused; refused', rules).map((m) => [m.start, m.end, m.color]))
      .toEqual([[5, 23, '#ef4444'], [25, 32, '#3b82f6']]);
  });

  it('꺼진 규칙·빈 패턴·틀린 정규식은 쓰지 않고, 색이 형식에 안 맞으면 기본 색을 쓴다', () => {
    const rules = compileRules([
      rule({ pattern: 'a', enabled: false }),
      rule({ pattern: '' }),
      rule({ pattern: '(', regex: true }),
      rule({ pattern: 'ok', color: 'red' }),
    ]);
    expect(rules).toHaveLength(1);
    expect(rules[0].color).toBe(HIGHLIGHT_DEFAULT_COLOR);
    expect(isValidPattern('(', true)).toBe(false);
    expect(isValidPattern('(', false)).toBe(true);
    expect(isValidPattern('', false)).toBe(false);
    expect(compileRules(undefined)).toEqual([]);
  });

  it('한 줄의 일치 수는 한도까지만', () => {
    const rules = compileRules([rule({ pattern: 'a' })]);
    expect(findMatches('a'.repeat(500), rules)).toHaveLength(HIGHLIGHT_MAX_PER_LINE);
  });
});

describe('키워드 강조: 글 위치 → 터미널 칸', () => {
  it('반각 글자만 있으면 위치와 칸이 같다', () => {
    const { text, colOf } = lineText(fakeLine('abc ERROR def', 20), 20);
    expect(text.startsWith('abc ERROR def')).toBe(true);
    const [m] = findMatches(text, compileRules([rule({ pattern: 'error' })]));
    expect(matchCells(m, colOf)).toEqual({ x: 4, width: 5 });
  });

  it('한글(전각) 뒤의 일치는 칸이 밀리고, 한글 일치는 글자 수의 두 배 폭이다', () => {
    const { text, colOf } = lineText(fakeLine('디스크 오류 ERROR', 30), 30);
    const rules = compileRules([rule({ pattern: 'error' }), rule({ pattern: '오류', color: '#eab308' })]);
    const got = findMatches(text, rules).map((m) => ({ ...matchCells(m, colOf), color: m.color }));
    expect(got).toEqual([{ x: 7, width: 4, color: '#eab308' }, { x: 12, width: 5, color: '#ef4444' }]);
  });

  it('줄 끝까지 가는 일치도 폭이 맞다', () => {
    const { text, colOf } = lineText(fakeLine('xxFAIL', 6), 6);
    const [m] = findMatches(text, compileRules([rule({ pattern: 'fail' })]));
    expect(matchCells(m, colOf)).toEqual({ x: 2, width: 4 });
  });
});
