// 매크로 실행·녹화와 출력 트리거 실행 (spec/behavior/terminal.md §10·§11, D140·D141).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Settings, SessionInfo, Store, TriggerRule } from '../ipc/types';
import { setBackend, type Backend } from '../ipc/backend';
import { MACRO_EXPECT_TIMEOUT_MS, MACRO_QUIET_MAX_MS, MACRO_QUIET_MS, newMacroStep } from '../lib/macro';
import { emitOutput, emitUserInput } from '../lib/outputHub';
import { newTriggerRule, TRIGGER_IDLE_MS } from '../lib/trigger';
import { useApp } from './app';
import { resetMacrosForTest, runMacro, startMacros, startRecording, stopMacro, stopRecording, useMacros } from './macros';
import { resetTriggersForTest, startTriggers } from './triggers';
import { useWorkspace } from './workspace';

const termInput = vi.fn();
let stop: Array<() => void> = [];

function session(id: string, p: Partial<SessionInfo> = {}): SessionInfo {
  return { id, hostId: null, title: id, protocol: 'ssh', target: 'u@h:22', state: 'connected', canSftp: true, ...p };
}

function setRules(rules: TriggerRule[], extra: Partial<Settings> = {}): void {
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { triggerRules: rules, guardDangerousOnProd: true, ...extra } as Settings } as Store });
}

const sent = () => termInput.mock.calls.map((c) => c as [string, string]);
const toasts = () => useApp.getState().toasts.map((x) => x.text);

beforeEach(() => {
  vi.useFakeTimers();
  termInput.mockClear();
  setBackend({ invoke: vi.fn(async () => undefined), termInput, termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], sessions: {}, writers: {} });
  useWorkspace.getState().upsertSession(session('s1'));
  useApp.setState({ toasts: [] });
  setRules([]);
  resetMacrosForTest();
  resetTriggersForTest();
  stop = [startMacros(), startTriggers()];
});

afterEach(() => {
  stop.forEach((off) => off());
  vi.useRealTimers();
  setBackend(null);
});

describe('매크로 실행', () => {
  it('첫 단계는 바로, 기다릴 글자가 없는 다음 단계는 출력이 조용해진 뒤에 보낸다', () => {
    const steps = [newMacroStep({ send: 'cd /srv' }), newMacroStep({ send: 'ls', enter: false })];
    expect(runMacro('s1', 'm1', '배포', steps)).toBe(true);
    expect(sent()).toEqual([['s1', 'cd /srv\r']]);
    expect(useMacros.getState().running.s1).toMatchObject({ macroId: 'm1', step: 1, total: 2 });

    // 출력이 이어지는 동안에는 기다린다.
    vi.advanceTimersByTime(MACRO_QUIET_MS - 100);
    emitOutput('s1', 'working...\r\n');
    vi.advanceTimersByTime(MACRO_QUIET_MS - 100);
    expect(sent()).toHaveLength(1);
    vi.advanceTimersByTime(100);
    expect(sent()[1]).toEqual(['s1', 'ls']);
    expect(useMacros.getState().running.s1).toBeUndefined();
    expect(toasts()).toContain('매크로 "배포" 완료');
  });

  it('기다릴 글자가 있으면 그 글자가 나온 뒤에 보내고, 보내기 전의 출력은 다시 쓰지 않는다', () => {
    const steps = [newMacroStep({ send: 'sudo -i' }), newMacroStep({ expect: 'password', send: '{{pw}}' }), newMacroStep({ expect: '# ', send: 'whoami' })];
    emitOutput('s1', 'old password prompt\r\n');
    runMacro('s1', 'm1', 'root', steps, { pw: 's3cret' });
    vi.advanceTimersByTime(MACRO_QUIET_MAX_MS);
    expect(sent()).toHaveLength(1); // 조용해도 글자가 나오기 전에는 보내지 않는다
    emitOutput('s1', '[sudo] Password for user: ');
    expect(sent()[1]).toEqual(['s1', 's3cret\r']);
    emitOutput('s1', '\r\nroot@host:~# ');
    expect(sent()[2]).toEqual(['s1', 'whoami\r']);
  });

  it('기다릴 글자가 시한 안에 나오지 않으면 멈추고 알린다', () => {
    runMacro('s1', 'm1', '느림', [newMacroStep({ send: 'a' }), newMacroStep({ expect: 'never', send: 'b' })]);
    vi.advanceTimersByTime(MACRO_EXPECT_TIMEOUT_MS + 1);
    expect(sent()).toHaveLength(1);
    expect(useMacros.getState().running.s1).toBeUndefined();
    expect(toasts().some((x) => x.includes('"never"'))).toBe(true);
  });

  it('출력이 그치지 않아도 상한이 지나면 다음 단계를 보낸다', () => {
    runMacro('s1', 'm1', 'tail', [newMacroStep({ send: 'tail -f x' }), newMacroStep({ send: '\\x03', enter: false })]);
    for (let t = 0; t < MACRO_QUIET_MAX_MS + 200; t += 100) {
      emitOutput('s1', 'line\r\n');
      vi.advanceTimersByTime(100);
    }
    expect(sent()[1]).toEqual(['s1', '\x03']);
  });

  it('같은 창에서 두 개를 겹쳐 실행하지 않고, 중지하면 남은 단계를 보내지 않는다', () => {
    const steps = [newMacroStep({ send: 'a' }), newMacroStep({ send: 'b' })];
    expect(runMacro('s1', 'm1', '하나', steps)).toBe(true);
    expect(runMacro('s1', 'm2', '둘', steps)).toBe(false);
    stopMacro('s1');
    vi.advanceTimersByTime(MACRO_QUIET_MAX_MS);
    expect(sent()).toEqual([['s1', 'a\r']]);
  });

  it('연결되지 않은 창에서는 실행하지 않는다', () => {
    useWorkspace.getState().upsertSession(session('s1', { state: 'closed' }));
    expect(runMacro('s1', 'm1', 'x', [newMacroStep({ send: 'a' })])).toBe(false);
    expect(sent()).toEqual([]);
  });
});

describe('매크로 녹화', () => {
  it('녹화 중인 창의 입력만 단계로 모은다', () => {
    expect(startRecording('s1')).toBe(true);
    expect(startRecording('s1')).toBe(false);
    emitUserInput('s2', 'other\r');
    emitUserInput('s1', 'uptime\r');
    emitUserInput('s1', 'df -h');
    expect(useMacros.getState().recording).toEqual({ sessionId: 's1', count: 2 });
    const steps = stopRecording();
    expect(steps.map((s) => [s.send, s.enter])).toEqual([['uptime', true], ['df -h', false]]);
    expect(useMacros.getState().recording).toBeNull();
  });
});

describe('출력 트리거 실행', () => {
  const rule = (p: Partial<TriggerRule>): TriggerRule => ({ ...newTriggerRule(), ...p });

  it('맞는 줄이 나오면 알림을 띄운다', () => {
    setRules([rule({ id: 'e', pattern: 'error' })]);
    emitOutput('s1', 'ERROR: disk full\r\n');
    expect(toasts()).toEqual(['s1: ERROR: disk full']);
  });

  it('프롬프트에는 출력이 조용해진 뒤 자동 응답을 그 연결에 입력한다', () => {
    setRules([rule({ id: 'y', pattern: '[y/n]', notify: false, replyOn: true, reply: 'y' })]);
    emitOutput('s1', 'Proceed? [y/N] ');
    expect(sent()).toEqual([]);
    vi.advanceTimersByTime(TRIGGER_IDLE_MS + 1);
    expect(sent()).toEqual([['s1', 'y\r']]);
  });

  it('운영 서버에는 자동 응답을 보내지 않고 알린다', () => {
    useWorkspace.getState().upsertSession(session('s1', { env: 'prod' }));
    setRules([rule({ id: 'y', pattern: 'continue', notify: false, replyOn: true, reply: 'yes' })]);
    emitOutput('s1', 'continue?\r\n');
    expect(sent()).toEqual([]);
    expect(toasts()).toEqual(['s1: 운영 서버라 자동 응답을 보내지 않았습니다']);
  });

  it('운영 서버 확인 설정을 끄면 운영 서버에도 보낸다', () => {
    useWorkspace.getState().upsertSession(session('s1', { env: 'prod' }));
    setRules([rule({ id: 'y', pattern: 'continue', notify: false, replyOn: true, reply: 'yes' })], { guardDangerousOnProd: false });
    emitOutput('s1', 'continue?\r\n');
    expect(sent()).toEqual([['s1', 'yes\r']]);
  });

  it('사용자가 치고 있는 줄의 에코에는 반응하지 않는다', () => {
    setRules([rule({ id: 'e', pattern: 'error' })]);
    emitOutput('s1', '$ ');
    emitUserInput('s1', 'grep error x');
    emitOutput('s1', 'grep error x');
    vi.advanceTimersByTime(TRIGGER_IDLE_MS + 1);
    expect(toasts()).toEqual([]);
  });

  it('규칙이 없으면 아무 일도 하지 않는다', () => {
    emitOutput('s1', 'error\r\n');
    vi.advanceTimersByTime(TRIGGER_IDLE_MS + 1);
    expect(toasts()).toEqual([]);
    expect(sent()).toEqual([]);
  });
});
