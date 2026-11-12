import { describe, expect, it } from 'vitest';
import { formatBytes, formatDate } from './format';

describe('표시 형식 (spec/ui/dialogs.md §1.5)', () => {
  it('바이트 단위 경계', () => {
    expect(formatBytes(1023)).toBe('1023 B');
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1024 * 1024 * 1.5)).toBe('1.5 MB');
    expect(formatBytes(1024 ** 3 * 2)).toBe('2.00 GB');
  });
  it('날짜: 값 0 이면 빈 칸, 올해는 MM-DD HH:mm, 다른 해는 YYYY-MM-DD', () => {
    expect(formatDate(0)).toBe('');
    const now = new Date(2026, 5, 3, 12, 0);
    expect(formatDate(new Date(2026, 0, 2, 9, 5).getTime(), now)).toBe('01-02 09:05');
    expect(formatDate(new Date(2025, 0, 2).getTime(), now)).toBe('2025-01-02');
  });
});
