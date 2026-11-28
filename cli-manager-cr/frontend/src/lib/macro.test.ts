// 매크로의 순수 계산 (spec/behavior/terminal.md §11, D141).
import { describe, expect, it } from 'vitest';
import { cleanSteps, escapeKeys, isMacro, MACRO_STEP_MAX, macroBody, MacroRecorder, newMacroStep, OutputTail, unescapeKeys } from './macro';
import { cleanLoginScript, LOGIN_STEP_MAX, moveStep, newLoginStep } from './loginScript';

describe('제어 문자 적기·읽기', () => {
  it('제어 문자는 \\xNN, ESC 는 \\e, 탭은 \\t, 역슬래시는 두 번', () => {
    expect(escapeKeys('a\x03b\x1b[A\tc\\d')).toBe('a\\x03b\\e[A\\tc\\\\d');
  });

  it('적은 것을 다시 읽으면 원래 글자', () => {
    const raw = 'ls -l\x03\x1b[A\t\\n 한글 \x7f';
    expect(unescapeKeys(escapeKeys(raw))).toBe(raw);
  });

  it('모르는 역슬래시 표기는 그대로 둔다', () => {
    expect(unescapeKeys('C:\\work \\q \\x4')).toBe('C:\\work \\q \\x4');
    expect(unescapeKeys('\\x03\\r\\n\\e')).toBe('\x03\r\n\x1b');
  });
});

describe('MacroRecorder', () => {
  it('Enter 마다 한 단계, 남은 입력은 Enter 없는 단계', () => {
    const r = new MacroRecorder();
    r.feed('c');
    r.feed('d /var/log\r');
    r.feed('tail -f app.log\rq');
    const steps = r.finish();
    expect(steps.map((s) => [s.send, s.enter])).toEqual([['cd /var/log', true], ['tail -f app.log', true], ['q', false]]);
  });

  it('백스페이스는 친 글자를 지운다', () => {
    const r = new MacroRecorder();
    r.feed('lss');
    r.feed('\x7f');
    r.feed(' -la\r');
    expect(r.finish().map((s) => s.send)).toEqual(['ls -la']);
  });

  it('제어 문자와 방향키는 적어 둔다(방향키는 백스페이스로 반쪽만 지워지지 않는다)', () => {
    const r = new MacroRecorder();
    r.feed('\x03');
    r.feed('\x1b[A');
    r.feed('\x7f');
    r.feed('\r');
    expect(r.finish().map((s) => s.send)).toEqual(['\\x03\\e[A\\x7f']);
  });

  it('터미널이 스스로 보내는 응답은 기록하지 않는다', () => {
    const r = new MacroRecorder();
    r.feed('\x1b[?1;2c');
    r.feed('\x1b[24;80R');
    r.feed('\x1b[I');
    expect(r.count()).toBe(0);
    expect(r.finish()).toEqual([]);
  });

  it('빈 Enter 도 한 단계다(Enter 만 보낸다)', () => {
    const r = new MacroRecorder();
    r.feed('\r');
    expect(r.finish().map((s) => [s.send, s.enter])).toEqual([['', true]]);
  });

  it('단계 수 한도', () => {
    const r = new MacroRecorder();
    for (let i = 0; i < MACRO_STEP_MAX + 10; i++) r.feed(`echo ${i}\r`);
    expect(r.finish()).toHaveLength(MACRO_STEP_MAX);
  });
});

describe('OutputTail', () => {
  it('출력에서 글자를 찾으면(대소문자 무시) 그때까지의 출력을 버린다', () => {
    const tail = new OutputTail();
    tail.feed('Last login\r\nuser@host:~$ ');
    expect(tail.consume('Password')).toBe(false);
    expect(tail.consume('$ ')).toBe(true);
    expect(tail.consume('$ ')).toBe(false);
  });

  it('버린 뒤 같은 줄에 이어진 출력에서 앞부분을 다시 보지 않는다', () => {
    const tail = new OutputTail();
    tail.feed('host$ ');
    expect(tail.consume('$ ')).toBe(true);
    tail.feed('sudo -i'); // 같은 줄에 에코가 이어진다
    expect(tail.consume('$ ')).toBe(false);
    tail.feed('\r\n[sudo] password for user: ');
    expect(tail.consume('password')).toBe(true);
    tail.feed('\r\nhost# ');
    expect(tail.consume('password')).toBe(false);
    expect(tail.consume('# ')).toBe(true);
  });

  it('reset 뒤의 출력만 본다', () => {
    const tail = new OutputTail();
    tail.feed('ready\r\n$ ');
    tail.reset();
    expect(tail.consume('ready')).toBe(false);
    tail.feed('ls\r\nready\r\n');
    expect(tail.consume('ready')).toBe(true);
  });
});

describe('매크로·로그인 스크립트 단계 다듬기', () => {
  it('보낼 것이 없는 매크로 단계는 뺀다', () => {
    const steps = [newMacroStep({ send: 'a' }), newMacroStep({ send: '', enter: false }), newMacroStep({ send: '', enter: true })];
    expect(cleanSteps(steps).map((s) => [s.send, s.enter])).toEqual([['a', true], ['', true]]);
    expect(macroBody(cleanSteps(steps))).toBe('a\n');
  });

  it('단계가 있어야 매크로다', () => {
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true })).toBe(false);
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true, steps: [] })).toBe(false);
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true, steps: [newMacroStep({ send: 'ls' })] })).toBe(true);
  });

  it('로그인 스크립트: 빈 단계는 빼고, 하나도 없으면 필드를 두지 않는다', () => {
    expect(cleanLoginScript([newLoginStep()])).toBeUndefined();
    expect(cleanLoginScript(undefined)).toBeUndefined();
    const many = Array.from({ length: LOGIN_STEP_MAX + 3 }, () => ({ ...newLoginStep(), expect: '$', send: 'x' }));
    expect(cleanLoginScript(many)).toHaveLength(LOGIN_STEP_MAX);
  });

  it('단계 옮기기', () => {
    expect(moveStep(['a', 'b', 'c'], 0, 1)).toEqual(['b', 'a', 'c']);
    expect(moveStep(['a', 'b', 'c'], 2, -1)).toEqual(['a', 'c', 'b']);
    const same = ['a', 'b'];
    expect(moveStep(same, 0, -1)).toBe(same);
  });
});
