// 터미널 찾기 강조의 색·범위 계산. 근거: spec/ui/panes.md §2.4 (찾은 항목 주황 계열, 현재 항목 빨강 계열).
// 찾은 항목은 터미널 배경에 주황을 옅게 섞어 낮추고, 현재 항목은 진한 빨강 바탕 + 흰 글자 + 테두리로 올린다 (D96).

/** 찾은 항목(현재 항목 제외)의 기준 색과 섞는 비율. */
export const FIND_MATCH_COLOR = '#f59e0b';
export const FIND_MATCH_RATIO = 0.32;
/** 현재 항목의 바탕·글자 색. */
export const FIND_ACTIVE_BG = '#dc2626';
export const FIND_ACTIVE_FG = '#ffffff';
/** 스크롤 막대 표시 색. */
export const FIND_RULER_MATCH = '#f59e0b';
export const FIND_RULER_ACTIVE = '#ef4444';
/** 스크롤 막대 자리의 표시 폭(px). xterm 스크롤 막대 기본 폭과 같게 둔다. */
export const FIND_RULER_WIDTH = 14;

const HEX6 = /^#[0-9a-fA-F]{6}$/;

/** `#RRGGBB` 두 색을 섞는다(ratio = over 의 비중, 0~1). 형식이 아니면 over 를 그대로 돌려준다. */
export function mixHex(base: string, over: string, ratio: number): string {
  if (!HEX6.test(base) || !HEX6.test(over)) return over;
  const r = Math.min(1, Math.max(0, ratio));
  let out = '#';
  for (let i = 1; i < 7; i += 2) {
    const b = parseInt(base.slice(i, i + 2), 16);
    const o = parseInt(over.slice(i, i + 2), 16);
    out += Math.round(b + (o - b) * r).toString(16).padStart(2, '0');
  }
  return out;
}

export interface CellPos { x: number; y: number }
export interface RowRange { row: number; x: number; width: number }

/**
 * 선택 범위(시작 칸 포함, 끝 칸 제외, 0부터 세는 버퍼 좌표)를 줄별 범위로 나눈다.
 * 일치 항목이 줄 끝에서 다음 줄로 이어지면 줄마다 하나씩 나온다.
 */
export function rowRanges(start: CellPos, end: CellPos, cols: number): RowRange[] {
  const out: RowRange[] = [];
  for (let row = start.y; row <= end.y; row++) {
    const x0 = row === start.y ? start.x : 0;
    const x1 = row === end.y ? end.x : cols;
    if (x1 > x0) out.push({ row, x: x0, width: x1 - x0 });
  }
  return out;
}
