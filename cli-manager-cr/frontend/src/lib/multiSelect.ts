// 사이드바 서버 여러 개 선택 규칙(D59): Ctrl+클릭 = 하나씩 넣고 빼기, Shift+클릭 = 화면 순서상 범위.

/** Ctrl+클릭: base 에 id 가 있으면 빼고, 없으면 끝에 넣는다. */
export function toggleSelection(base: string[], id: string): string[] {
  return base.includes(id) ? base.filter((x) => x !== id) : [...base, id];
}

/**
 * Shift+클릭: 화면에 보이는 순서(order)에서 anchor 부터 id 까지(양끝 포함).
 * 둘 중 하나라도 화면에 없으면 null(호출자가 일반 클릭으로 처리).
 */
export function rangeSelection(order: string[], anchor: string | null, id: string): string[] | null {
  if (!anchor) return null;
  const i = order.indexOf(anchor);
  const j = order.indexOf(id);
  if (i < 0 || j < 0) return null;
  return order.slice(Math.min(i, j), Math.max(i, j) + 1);
}
