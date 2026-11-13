// 터미널 색 테마 표 확인. 근거: spec/ui/tokens.md §6.
import { describe, expect, it } from 'vitest';
import { resolveTermTheme, TERM_THEMES } from './termThemes';

describe('터미널 색 테마 (tokens.md §6)', () => {
  it('6개 테마, 순서 유지, 이름 없으면 DariTerm Dark, 예전 접두어 호환', () => {
    expect(TERM_THEMES.map((t) => t.name)).toEqual(['DariTerm Dark', 'DariTerm Light', 'PuTTY Classic', 'Solarized Dark', 'Monokai', 'Nord']);
    expect(resolveTermTheme(undefined).name).toBe('DariTerm Dark');
    expect(resolveTermTheme('BrightTerm Light').name).toBe('DariTerm Light');
    expect(TERM_THEMES[0].background).toBe('#0d1117');
    expect(TERM_THEMES[0].brightWhite).toBe('#ffffff');
  });
});
