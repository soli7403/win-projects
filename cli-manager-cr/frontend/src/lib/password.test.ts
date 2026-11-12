import { describe, expect, it } from 'vitest';
import { passwordScore, strengthBarWidth, STRENGTH_COLORS, STRENGTH_LABELS } from './password';

describe('마스터 비밀번호 강도 (spec/ui/lock.md §2, acceptance L4)', () => {
  it('Abcdefgh1!x2 는 매우 강함, 막대 100%, 색 #10b981', () => {
    const s = passwordScore('Abcdefgh1!x2');
    expect(STRENGTH_LABELS[s]).toBe('매우 강함');
    expect(strengthBarWidth(s)).toBe('100%');
    expect(STRENGTH_COLORS[s]).toBe('#10b981');
  });
  it('abc 는 0점 (매우 약함, 막대 20%)', () => {
    expect(passwordScore('abc')).toBe(0);
    expect(strengthBarWidth(0)).toBe('20%');
  });
  it('8자 이상 소문자만은 1점', () => {
    expect(passwordScore('abcdefgh')).toBe(1);
  });
  it('점수는 4에서 잘린다', () => {
    expect(passwordScore('Abcdefghijkl1!')).toBe(4);
  });
});
