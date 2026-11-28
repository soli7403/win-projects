// 출력 트리거의 순수 계산 (spec/behavior/terminal.md §10, D140).
import { describe, expect, it } from 'vitest';
import type { TriggerRule } from '../ipc/types';
import { compileTriggers, newTriggerRule, TRIGGER_COOLDOWN_MS, TRIGGER_LOOP_MAX, TRIGGER_MAX_RULES, TriggerWatcher } from './trigger';

function rule(p: Partial<TriggerRule>): TriggerRule {
  return { ...newTriggerRule(), id: p.pattern ?? 'r', ...p };
}

describe('compileTriggers', () => {
  it('꺼진 규칙·빈 글자·틀린 정규식·하는 일이 없는 규칙은 뺀다', () => {
    const out = compileTriggers([
      rule({ pattern: 'a', enabled: false }),
      rule({ pattern: '' }),
      rule({ pattern: '(', regex: true }),
      rule({ pattern: 'b', notify: false, sound: false, replyOn: false }),
      rule({ pattern: 'ok' }),
    ]);
    expect(out.map((r) => r.id)).toEqual(['ok']);
  });

  it('자동 응답은 켰을 때만, Enter 를 붙이면 CR 을 더한다', () => {
    const [off, on, enterOnly, noEnter] = compileTriggers([
      rule({ id: '1', pattern: 'a', reply: 'y', replyOn: false }),
      rule({ id: '2', pattern: 'a', reply: 'y', replyOn: true }),
      rule({ id: '3', pattern: 'a', reply: '', replyOn: true, notify: false }),
      rule({ id: '4', pattern: 'a', reply: 'q', replyOn: true, replyEnter: false }),
    ]);
    expect(off.reply).toBe('');
    expect(on.reply).toBe('y\r');
    expect(enterOnly.reply).toBe('\r');
    expect(noEnter.reply).toBe('q');
  });

  it('글자는 그대로(정규식 기호를 풀지 않고), 기본은 대소문자 구분 없이 찾는다', () => {
    const [plain, cased] = compileTriggers([rule({ id: '1', pattern: '[y/N]' }), rule({ id: '2', pattern: 'Error', caseSensitive: true })]);
    expect(plain.re.test('Continue? [Y/n]')).toBe(true);
    expect(cased.re.test('error')).toBe(false);
  });

  it('규칙 수 한도', () => {
    const many = Array.from({ length: TRIGGER_MAX_RULES + 5 }, (_, i) => rule({ id: `r${i}`, pattern: `p${i}` }));
    expect(compileTriggers(many)).toHaveLength(TRIGGER_MAX_RULES);
  });
});

describe('TriggerWatcher', () => {
  const rules = compileTriggers([rule({ id: 'err', pattern: 'error' }), rule({ id: 'yn', pattern: '[y/n]', replyOn: true, reply: 'y', notify: false })]);

  it('끝난 줄에서 맞는 규칙을 실행한다', () => {
    let now = 0;
    const w = new TriggerWatcher(() => now);
    const fires = w.feed('ok\r\n\x1b[31mERROR: disk full\x1b[0m\r\n', rules);
    expect(fires.map((f) => [f.rule.id, f.line])).toEqual([['err', 'ERROR: disk full']]);
  });

  it('줄바꿈 없이 멈춘 줄(프롬프트)은 idle 에서 실행하고, 그 줄이 끝나도 다시 실행하지 않는다', () => {
    let now = 0;
    const w = new TriggerWatcher(() => now);
    expect(w.feed('Continue? [y/N] ', rules)).toEqual([]);
    expect(w.idle(rules).map((f) => f.rule.id)).toEqual(['yn']);
    expect(w.idle(rules)).toEqual([]);
    now += 5000;
    expect(w.feed('y\r\n', rules)).toEqual([]);
  });

  it('같은 규칙은 쿨다운 동안 다시 실행하지 않는다', () => {
    let now = 0;
    const w = new TriggerWatcher(() => now);
    expect(w.feed('error 1\n', rules)).toHaveLength(1);
    now += TRIGGER_COOLDOWN_MS - 1;
    expect(w.feed('error 2\n', rules)).toHaveLength(0);
    now += 2;
    expect(w.feed('error 3\n', rules)).toHaveLength(1);
  });

  it('사용자가 치는 줄의 에코에는 반응하지 않고, 줄이 바뀌면 다시 살핀다', () => {
    let now = 0;
    const w = new TriggerWatcher(() => now);
    w.noteInput('grep error log');
    expect(w.feed('$ grep error log', rules)).toEqual([]);
    expect(w.idle(rules)).toEqual([]);
    w.noteInput('\r');
    expect(w.feed('\r\n', rules)).toEqual([]); // 에코 줄 자체는 건너뛴다
    expect(w.feed('app.log: error at line 3\r\n', rules).map((f) => f.rule.id)).toEqual(['err']);
  });

  it('터미널의 자동 응답(ESC 로 시작)은 사용자 입력으로 세지 않는다', () => {
    const w = new TriggerWatcher(() => 0);
    w.noteInput('\x1b[?1;2c');
    expect(w.feed('error\n', rules)).toHaveLength(1);
  });

  it('자동 응답이 짧은 시간에 너무 자주 반복되면 그 규칙을 끈다', () => {
    let now = 0;
    const w = new TriggerWatcher(() => now);
    const muted: boolean[] = [];
    for (let i = 0; i < TRIGGER_LOOP_MAX + 2; i++) {
      now += TRIGGER_COOLDOWN_MS + 1;
      for (const f of w.feed('again? [y/n]\n', rules)) muted.push(f.muted);
    }
    // 한도까지는 보내고, 한도를 넘는 순간 한 번 muted 로 알린 뒤에는 조용하다.
    expect(muted).toEqual([...Array(TRIGGER_LOOP_MAX).fill(false), true]);
  });

  it('전체 화면 프로그램 안의 글자에는 반응하지 않는다', () => {
    const w = new TriggerWatcher(() => 0);
    expect(w.feed('\x1b[?1049herror in editor\n', rules)).toEqual([]);
  });
});
