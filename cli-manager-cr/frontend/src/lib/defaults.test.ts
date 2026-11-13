import { describe, expect, it } from 'vitest';
import { DEFAULT_SETTINGS, DANGEROUS_PATTERNS, COLOR_PALETTE } from './defaults';

describe('기본값 (spec/data/store.md §4·§6)', () => {
  it('설정 기본값 수치', () => {
    expect(DEFAULT_SETTINGS.fontSize).toBe(14);
    expect(DEFAULT_SETTINGS.autoLockMinutes).toBe(15);
    expect(DEFAULT_SETTINGS.sidebarWidth).toBe(260);
    expect(DEFAULT_SETTINGS.rightPanelWidth).toBe(340);
    expect(DEFAULT_SETTINGS.terminalTheme).toBe('DariTerm Dark');
  });
  it('위험 명령 패턴 11개, 팔레트 14색', () => {
    expect(DANGEROUS_PATTERNS).toHaveLength(11);
    expect(COLOR_PALETTE).toHaveLength(14);
  });
});
