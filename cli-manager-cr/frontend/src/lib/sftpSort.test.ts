import { describe, expect, it } from 'vitest';
import { DEFAULT_SORT, nextSort, sortEntries } from './sftpSort';

const e = (name: string, isDir: boolean, size: number, mtime: number) => ({ name, isDir, size, mtime });
const list = [
  e('b.txt', false, 300, 30),
  e('file10.log', false, 10, 10),
  e('Zeta', true, 4096, 5),
  e('A.md', false, 300, 20),
  e('file2.log', false, 2000, 40),
  e('alpha', true, 4096, 50),
];
const names = (xs: { name: string }[]) => xs.map((x) => x.name);

describe('sortEntries', () => {
  it('이름 오름차순: 폴더 먼저, 대소문자 무시, 숫자는 값 순서', () => {
    expect(names(sortEntries(list, DEFAULT_SORT))).toEqual(['alpha', 'Zeta', 'A.md', 'b.txt', 'file2.log', 'file10.log']);
  });

  it('이름 내림차순에서도 폴더는 앞', () => {
    expect(names(sortEntries(list, { key: 'name', dir: 'desc' }))).toEqual(['Zeta', 'alpha', 'file10.log', 'file2.log', 'b.txt', 'A.md']);
  });

  it('크기: 같은 크기는 이름 오름차순, 폴더는 크기 무시하고 이름순', () => {
    expect(names(sortEntries(list, { key: 'size', dir: 'desc' }))).toEqual(['alpha', 'Zeta', 'file2.log', 'A.md', 'b.txt', 'file10.log']);
    expect(names(sortEntries(list, { key: 'size', dir: 'asc' }))).toEqual(['alpha', 'Zeta', 'file10.log', 'A.md', 'b.txt', 'file2.log']);
  });

  it('수정일: 폴더끼리도 수정일로', () => {
    expect(names(sortEntries(list, { key: 'mtime', dir: 'desc' }))).toEqual(['alpha', 'Zeta', 'file2.log', 'b.txt', 'A.md', 'file10.log']);
  });

  it('원본 배열은 바꾸지 않는다', () => {
    const copy = [...list];
    sortEntries(list, { key: 'size', dir: 'asc' });
    expect(list).toEqual(copy);
  });
});

describe('nextSort', () => {
  it('같은 열은 방향을 뒤집고, 다른 열은 이름=오름차순·크기/수정일=내림차순으로 시작', () => {
    expect(nextSort(DEFAULT_SORT, 'name')).toEqual({ key: 'name', dir: 'desc' });
    expect(nextSort(DEFAULT_SORT, 'size')).toEqual({ key: 'size', dir: 'desc' });
    expect(nextSort({ key: 'size', dir: 'desc' }, 'mtime')).toEqual({ key: 'mtime', dir: 'desc' });
    expect(nextSort({ key: 'mtime', dir: 'desc' }, 'name')).toEqual({ key: 'name', dir: 'asc' });
  });
});
