// 경로 입력 형식 테스트. 근거: spec/behavior/terminal.md §5.
import { describe, expect, it } from 'vitest';
import { quotePath } from './paste';

describe('경로 입력 따옴표 (terminal.md §5)', () => {
  it('특수문자나 공백이 있을 때만 작은따옴표, 안의 작은따옴표는 이스케이프', () => {
    expect(quotePath('/tmp/a.png', true)).toBe('/tmp/a.png');
    expect(quotePath('/tmp/my file.png', true)).toBe("'/tmp/my file.png'");
    expect(quotePath("it's.png", true)).toBe("'it'\\''s.png'");
    expect(quotePath('/tmp/my file.png', false)).toBe('/tmp/my file.png');
  });
});
