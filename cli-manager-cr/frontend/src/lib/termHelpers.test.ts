// 터미널 보조 규칙 테스트. 근거: spec/behavior/terminal.md §4, §6, §8, spec/behavior/shortcuts.md, spec/ui/dialogs.md §2.1.
import { describe, expect, it } from 'vitest';
import { DANGEROUS_PATTERNS } from './defaults';
import { matchesDangerous, pasteLineCount, LineTracker } from './termGuard';
import { decodeOsc52 } from './osc52';
import { shortcutFor, type KeyLike } from './keys';
import { isLocalPrefix, parseDirect } from './quick';

const key = (over: Partial<KeyLike>): KeyLike => ({ code: '', key: '', ctrlKey: false, shiftKey: false, altKey: false, ...over });

describe('위험 명령 가드 (terminal.md §6)', () => {
  it('기본 패턴에 걸리고 대소문자를 무시한다', () => {
    expect(matchesDangerous('RM -rf /tmp/x', DANGEROUS_PATTERNS)).toBe(true);
    expect(matchesDangerous('sudo reboot', DANGEROUS_PATTERNS)).toBe(true);
    expect(matchesDangerous('ls -la', DANGEROUS_PATTERNS)).toBe(false);
  });
  it('잘못된 정규식은 건너뛴다', () => {
    expect(matchesDangerous('abc', ['(', 'abc'])).toBe(true);
  });
  it('줄 추적: Enter 전 줄 확인, Backspace·Ctrl+U 처리, 방향키는 알 수 없음', () => {
    const t = new LineTracker();
    t.feed('rebooX');
    t.feed('\x7f');
    t.feed('t');
    expect(t.current()).toBe('reboot');
    t.feed('\x15');
    expect(t.current()).toBeNull();
    t.feed('ls');
    t.feed('\x1b[A');
    expect(t.current()).toBeNull();
    t.reset();
    t.feed('shutdown now');
    expect(t.current()).toBe('shutdown now');
  });
  it('붙여넣기 줄 수: 끝 줄바꿈 하나는 세지 않는다', () => {
    expect(pasteLineCount('a\nb\n')).toBe(2);
    expect(pasteLineCount('a')).toBe(1);
    expect(pasteLineCount('')).toBe(0);
  });
});

describe('OSC 52 (terminal.md §8)', () => {
  it('base64 UTF-8 쓰기 요청을 풀고, 읽기 요청과 잘못된 base64 는 무시', () => {
    const enc = btoa(String.fromCharCode(...new TextEncoder().encode('안녕')));
    expect(decodeOsc52(`c;${enc}`)).toBe('안녕');
    expect(decodeOsc52('c;?')).toBeNull();
    expect(decodeOsc52('c;')).toBeNull();
    expect(decodeOsc52('c;!!!')).toBeNull();
  });
});

describe('단축키 매핑 (shortcuts.md §2)', () => {
  it('물리 키(code)로 판단하고 Ctrl+K 는 토글, Ctrl+Shift+P 는 열기', () => {
    expect(shortcutFor(key({ code: 'KeyK', ctrlKey: true }))?.action).toBe('quickToggle');
    expect(shortcutFor(key({ code: 'KeyP', ctrlKey: true, shiftKey: true }))?.action).toBe('quickOpen');
    expect(shortcutFor(key({ code: 'Digit9', ctrlKey: true }))?.arg).toBe('last');
    expect(shortcutFor(key({ code: 'Digit3', ctrlKey: true }))?.arg).toBe(2);
    expect(shortcutFor(key({ code: 'ArrowUp', altKey: true }))?.arg).toBe('up');
    expect(shortcutFor(key({ code: 'F11' }))?.action).toBe('fullscreen');
    expect(shortcutFor(key({ code: 'KeyA', ctrlKey: true }))).toBeNull();
  });
});

describe('빠른 연결 입력 해석 (dialogs.md §2.1)', () => {
  it('user@host:port 형식과 기본 포트', () => {
    expect(parseDirect('root@10.0.0.5:2222')).toEqual({ protocol: 'ssh', host: '10.0.0.5', port: 2222, username: 'root' });
    expect(parseDirect('telnet://sw1')).toEqual({ protocol: 'telnet', host: 'sw1', port: 23, username: undefined });
    expect(parseDirect('10.0.0.5')?.port).toBe(22);
  });
  it('직접 연결처럼 보이지 않으면 null (예: 그냥 단어)', () => {
    expect(parseDirect('webserver')).toBeNull();
    expect(parseDirect('')).toBeNull();
  });
  it('로컬 접두어는 입력의 시작 단어로 판정', () => {
    expect(isLocalPrefix('shell')).toBe(true);
    expect(isLocalPrefix('로컬 터미널')).toBe(true);
    expect(isLocalPrefix('web')).toBe(false);
  });
});
