// 앱 잠금 비밀번호 강도. 근거: spec/ui/lock.md §2.

import type { MsgId } from '../i18n';

/** 강도 이름의 문구 키(D75). 그릴 때 t() 로 바꾼다. */
export const STRENGTH_LABELS: readonly MsgId[] = ['pw.veryWeak', 'pw.weak', 'pw.fair', 'pw.strong', 'pw.veryStrong'];
export const STRENGTH_COLORS = ['#ef4444', '#f97316', '#f59e0b', '#22c55e', '#10b981'] as const;

/** 조건 5개 각 1점, 최대 4로 자른다. 라벨 인덱스와 같다. */
export function passwordScore(pw: string): number {
  let s = 0;
  if (pw.length >= 8) s++;
  if (pw.length >= 12) s++;
  if (/[a-z]/.test(pw) && /[A-Z]/.test(pw)) s++;
  if (/[0-9]/.test(pw)) s++;
  if (/[^A-Za-z0-9]/.test(pw)) s++;
  return Math.min(s, 4);
}

/** 막대 너비(%) = (점수+1)×20. */
export function strengthBarWidth(score: number): string {
  return `${(score + 1) * 20}%`;
}
