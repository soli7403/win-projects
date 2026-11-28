// 출력 조각 → 글자만 남긴 줄 (spec/behavior/terminal.md §10.1, D140).
import { describe, expect, it } from 'vitest';
import { LINE_MAX, OutputLines } from './outputLines';

describe('OutputLines', () => {
  it('줄바꿈으로 줄을 나누고 끝나지 않은 줄은 partial 로', () => {
    const o = new OutputLines();
    expect(o.feed('hello\r\nworld\r\n$ ')).toEqual({ lines: ['hello', 'world'], partial: '$ ' });
    expect(o.feed('ls')).toEqual({ lines: [], partial: '$ ls' });
  });

  it('색·커서 시퀀스와 OSC(창 제목)를 지운다', () => {
    const o = new OutputLines();
    const r = o.feed('\x1b]0;user@host\x07\x1b[1;31mERROR\x1b[0m: disk \x1b[Kfull\n');
    expect(r.lines).toEqual(['ERROR: disk full']);
  });

  it('시퀀스가 조각 경계에서 잘려도 이어서 지운다', () => {
    const o = new OutputLines();
    expect(o.feed('ab\x1b[3').partial).toBe('ab');
    expect(o.feed('1mcd\x1b]0;ti').partial).toBe('abcd');
    expect(o.feed('tle\x1b\\ef\n').lines).toEqual(['abcdef']);
  });

  it('CR 뒤에 글자가 오면 그 줄을 처음부터 다시 쓴 것으로 본다', () => {
    const o = new OutputLines();
    expect(o.feed('10%\r50%\r100%').partial).toBe('100%');
    expect(o.feed('\r\ndone\n').lines).toEqual(['100%', 'done']);
  });

  it('백스페이스는 마지막 글자를 지우고 탭은 빈칸', () => {
    const o = new OutputLines();
    expect(o.feed('abx\bc\td\n').lines).toEqual(['abc d']);
  });

  it('전체 화면 프로그램(대체 화면) 안의 글자는 버린다', () => {
    const o = new OutputLines();
    expect(o.feed('before\n\x1b[?1049hERROR in vim\nmore').lines).toEqual(['before']);
    expect(o.alt).toBe(true);
    const r = o.feed('\x1b[?1049l$ ');
    expect(o.alt).toBe(false);
    expect(r).toEqual({ lines: [], partial: '$ ' });
  });

  it('줄바꿈 없이 이어지는 출력은 뒤쪽만 남긴다', () => {
    const o = new OutputLines();
    const r = o.feed('x'.repeat(LINE_MAX + 50) + 'END');
    expect(r.partial.length).toBe(LINE_MAX);
    expect(r.partial.endsWith('END')).toBe(true);
  });
});
