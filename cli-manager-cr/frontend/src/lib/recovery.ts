// 복구 코드 형식. 근거: spec/ui/lock.md §3·§4, spec/data/vault.md.

/** 입력 정규화: 하이픈·공백 제거 후 대문자. */
export function normalizeRecoveryInput(raw: string): string {
  return raw.replace(/[\s-]/g, '').toUpperCase();
}

/** 영숫자 24자를 4자씩 하이픈으로 나눈다. */
export function formatRecoveryCode(code: string): string {
  const s = normalizeRecoveryInput(code);
  const groups: string[] = [];
  for (let i = 0; i < s.length; i += 4) groups.push(s.slice(i, i + 4));
  return groups.join('-');
}
