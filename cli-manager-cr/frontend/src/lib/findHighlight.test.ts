import { describe, expect, it } from 'vitest';
import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_MATCH_COLOR, FIND_MATCH_RATIO } from './findHighlight';
import { TERM_THEMES } from './termThemes';

function luminance(hex: string): number {
  const ch = [1, 3, 5].map((i) => {
    const v = parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * ch[0] + 0.7152 * ch[1] + 0.0722 * ch[2];
}
function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

describe('findHighlight', () => {
  it('mixHex 는 두 색을 비율대로 섞는다', () => {
    expect(mixHex('#000000', '#ffffff', 0)).toBe('#000000');
    expect(mixHex('#000000', '#ffffff', 1)).toBe('#ffffff');
    expect(mixHex('#000000', '#ffffff', 0.5)).toBe('#808080');
    expect(mixHex('#0d1117', '#f59e0b', 0.32)).toMatch(/^#[0-9a-f]{6}$/);
  });

  it('mixHex 는 형식이 아닌 색이면 덧칠 색을 그대로 준다', () => {
    expect(mixHex('rgb(0,0,0)', '#f59e0b', 0.3)).toBe('#f59e0b');
  });

  it('모든 터미널 테마에서 찾은 항목 위의 글자가 읽히고, 찾은 항목은 원색보다 덜 튄다', () => {
    for (const th of TERM_THEMES) {
      const match = mixHex(th.background, FIND_MATCH_COLOR, FIND_MATCH_RATIO);
      expect(contrast(match, th.foreground), th.name).toBeGreaterThan(3);
      expect(contrast(FIND_ACTIVE_BG, '#ffffff'), th.name).toBeGreaterThan(4.5);
      // 찾은 항목은 원색 주황보다 배경에 가깝다(덜 튄다). 현재 항목과는 색상·흰 글자·테두리로 구분한다.
      expect(contrast(match, th.background), th.name).toBeLessThan(contrast(FIND_MATCH_COLOR, th.background));
    }
  });

  it('rowRanges 는 한 줄 범위를 그대로 준다', () => {
    expect(rowRanges({ x: 3, y: 10 }, { x: 8, y: 10 }, 80)).toEqual([{ row: 10, x: 3, width: 5 }]);
  });

  it('rowRanges 는 줄을 넘는 범위를 줄마다 나눈다', () => {
    expect(rowRanges({ x: 78, y: 5 }, { x: 4, y: 7 }, 80)).toEqual([
      { row: 5, x: 78, width: 2 },
      { row: 6, x: 0, width: 80 },
      { row: 7, x: 0, width: 4 },
    ]);
  });

  it('rowRanges 는 빈 줄 조각을 버린다', () => {
    expect(rowRanges({ x: 80, y: 1 }, { x: 0, y: 2 }, 80)).toEqual([]);
  });
});
