// 앱 잠금 비밀번호 강도. 근거: spec/ui/lock.md §2.

export const STRENGTH_LABELS = ['매우 약함', '약함', '보통', '강함', '매우 강함'] as const;
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
