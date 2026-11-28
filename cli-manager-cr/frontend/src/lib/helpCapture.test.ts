// 도움말 그림 자동 캡처 손잡이(D158): 화면 저장소에 표시가 있을 때만 생긴다.
import { afterEach, describe, expect, it } from 'vitest';
import { HELP_CAPTURE_KEY, installHelpCapture } from './helpCapture';

afterEach(() => localStorage.removeItem(HELP_CAPTURE_KEY));

describe('캡처 손잡이', () => {
  it('표시가 없으면 달지 않는다(평소 앱)', () => {
    const target: Record<string, unknown> = {};
    expect(installHelpCapture(target)).toBe(false);
    expect(target.__dtHelpCapture).toBeUndefined();
  });

  it('표시가 "1" 일 때만 단다', () => {
    const target: Record<string, unknown> = {};
    localStorage.setItem(HELP_CAPTURE_KEY, 'true');
    expect(installHelpCapture(target)).toBe(false);
    localStorage.setItem(HELP_CAPTURE_KEY, '1');
    expect(installHelpCapture(target)).toBe(true);
    const h = target.__dtHelpCapture as Record<string, unknown>;
    for (const k of ['ui', 'app', 'saveSettings', 'invoke', 'openHost']) expect(h[k], k).toBeTruthy();
  });
});
