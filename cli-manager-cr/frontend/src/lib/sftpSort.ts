// SFTP·파일 창 목록 정렬. 근거: spec/ui/sftp-snippets.md §2.3 (머리 줄을 눌러 이름·크기·수정일로 정렬).
// 폴더는 정렬 방향과 관계없이 항상 파일보다 앞에 둔다. 같은 값이면 이름 오름차순.

export type SortKey = 'name' | 'size' | 'mtime';
export type SortDir = 'asc' | 'desc';
export interface SortState {
  key: SortKey;
  dir: SortDir;
}

/** 기본 정렬: 이름 오름차순(백엔드가 주는 순서와 같은 규칙). */
export const DEFAULT_SORT: SortState = { key: 'name', dir: 'asc' };

interface Sortable {
  name: string;
  isDir: boolean;
  size: number;
  mtime: number;
}

// 대소문자 무시, 숫자는 값 순서(file2 < file10).
const nameCollator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });

function compareName(a: Sortable, b: Sortable): number {
  return nameCollator.compare(a.name, b.name) || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0);
}

/** 정렬한 새 배열을 돌려준다(원본은 그대로). */
export function sortEntries<T extends Sortable>(entries: readonly T[], sort: SortState): T[] {
  const sign = sort.dir === 'asc' ? 1 : -1;
  return [...entries].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    let diff = 0;
    if (sort.key === 'size') diff = (a.isDir ? 0 : a.size) - (b.isDir ? 0 : b.size);
    else if (sort.key === 'mtime') diff = a.mtime - b.mtime;
    else diff = compareName(a, b);
    if (diff !== 0) return diff * sign;
    return compareName(a, b);
  });
}

/** 머리 줄 클릭: 같은 열이면 방향을 뒤집고, 다른 열이면 이름은 오름차순·크기와 수정일은 내림차순(큰 것·최근 것 먼저)으로 시작. */
export function nextSort(cur: SortState, key: SortKey): SortState {
  if (cur.key === key) return { key, dir: cur.dir === 'asc' ? 'desc' : 'asc' };
  return { key, dir: key === 'name' ? 'asc' : 'desc' };
}
