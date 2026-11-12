import { describe, expect, it } from 'vitest';
import { formatRecoveryCode, normalizeRecoveryInput } from './recovery';

describe('복구 코드 형식 (spec/ui/lock.md §3)', () => {
  it('24자를 4자씩 하이픈으로 나눈다', () => {
    expect(formatRecoveryCode('ABCDEFGHJKLMNPQRSTUVWXYZ')).toBe('ABCD-EFGH-JKLM-NPQR-STUV-WXYZ');
  });
  it('입력 정규화는 하이픈·공백을 빼고 대문자로 만든다', () => {
    expect(normalizeRecoveryInput('abcd-efgh 1234')).toBe('ABCDEFGH1234');
  });
});
