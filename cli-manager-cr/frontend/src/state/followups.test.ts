// 2026-11-27 후속 기능들의 실행 시험: 트리거의 Windows 알림(D145) 등.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Host, Settings, SessionInfo, Snippet, Store, TriggerRule } from '../ipc/types';
import { setBackend, type Backend } from '../ipc/backend';
import { emitOutput } from '../lib/outputHub';
import { newTriggerRule } from '../lib/trigger';
import { newMacroStep } from '../lib/macro';
import type { LayoutNode } from '../lib/layout';
import { useApp } from './app';
import { replayLastMacro, resetMacrosForTest, runMacro, runMacroOnTargets, startMacros, stopMacrosOnTargets, useMacros } from './macros';
import { resetTriggersForTest, startTriggers } from './triggers';
import { useUi } from './ui';
import { useWorkspace } from './workspace';

const termInput = vi.fn();
let invoke = vi.fn(async (..._a: unknown[]): Promise<unknown> => undefined);
let stop: Array<() => void> = [];

function session(id: string, p: Partial<SessionInfo> = {}): SessionInfo {
  return { id, hostId: null, title: id, protocol: 'ssh', target: 'u@h:22', state: 'connected', canSftp: true, ...p };
}

function setSettings(rules: TriggerRule[], extra: Partial<Settings> = {}): void {
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { triggerRules: rules, guardDangerousOnProd: true, triggerOsNotify: true, ...extra } as Settings } as Store });
}

const rule = (p: Partial<TriggerRule>): TriggerRule => ({ ...newTriggerRule(), ...p });
const notifyCalls = () => invoke.mock.calls.filter((c) => c[0] === 'app.notify').map((c) => c[1]);

beforeEach(() => {
  vi.useFakeTimers();
  termInput.mockClear();
  invoke = vi.fn(async (..._a: unknown[]): Promise<unknown> => true);
  setBackend({ invoke, termInput, termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], sessions: {}, writers: {} });
  useWorkspace.getState().upsertSession(session('s1'));
  useApp.setState({ toasts: [] });
  setSettings([]);
  resetTriggersForTest();
  resetMacrosForTest();
  stop = [startTriggers(), startMacros()];
});

afterEach(() => {
  stop.forEach((off) => off());
  vi.restoreAllMocks();
  vi.useRealTimers();
  setBackend(null);
});

describe('트리거의 Windows 알림 (D145)', () => {
  it('앱 창에 포커스가 없을 때만, 한 번에 맞은 줄을 알림 하나로 보낸다', () => {
    setSettings([rule({ id: 'e', pattern: 'ERR' }), rule({ id: 'f', pattern: 'FAIL' })]);
    const focus = vi.spyOn(document, 'hasFocus').mockReturnValue(true);
    emitOutput('s1', 'ERR one\r\n');
    expect(notifyCalls()).toEqual([]);

    focus.mockReturnValue(false);
    vi.advanceTimersByTime(1100); // 같은 규칙 1초 제한을 넘긴다
    emitOutput('s1', 'ERR two\r\nFAIL three\r\n');
    expect(notifyCalls()).toEqual([['DariTerm · s1', 'ERR two\nFAIL three']]);
  });

  it('설정에서 끄면 보내지 않고, 한 번 실패하면 이번 실행에서는 다시 시도하지 않는다', async () => {
    vi.spyOn(document, 'hasFocus').mockReturnValue(false);
    setSettings([rule({ id: 'e', pattern: 'ERR' })], { triggerOsNotify: false });
    emitOutput('s1', 'ERR off\r\n');
    expect(notifyCalls()).toEqual([]);

    setSettings([rule({ id: 'e', pattern: 'ERR' })]);
    invoke.mockImplementation(async () => {
      throw new Error('unavailable');
    });
    vi.advanceTimersByTime(1100);
    emitOutput('s1', 'ERR first\r\n');
    expect(notifyCalls()).toHaveLength(1);
    await vi.waitFor(() => Promise.resolve()); // 실패가 기억되도록 약속을 흘려보낸다
    await Promise.resolve();
    vi.advanceTimersByTime(1100);
    emitOutput('s1', 'ERR again\r\n');
    expect(notifyCalls()).toHaveLength(1);
  });
});

/** 탭 하나에 세션들을 나란히 둔다. 첫 칸이 포커스. */
function tabWith(sessionIds: string[], broadcast: boolean): void {
  const panes: LayoutNode[] = sessionIds.map((sid, i) => ({ type: 'pane', id: `p${i}`, sessionId: sid }));
  const root: LayoutNode = { type: 'split', id: 'sp', dir: 'row', sizes: panes.map(() => 100 / panes.length), children: panes } as LayoutNode;
  useWorkspace.setState({ tabs: [{ id: 't1', broadcast, root, focusedPane: 'p0', zoomed: null }], activeTabId: 't1' });
}

const macro = (send: string[], p: Partial<Snippet> = {}): Snippet => ({
  id: 'm1', name: '점검', body: send.join('\n'), sendEnter: true, steps: send.map((x) => newMacroStep({ send: x })), ...p,
});
const sentTo = (sid: string) => termInput.mock.calls.filter((c) => c[0] === sid).map((c) => c[1]);

describe('매크로를 일괄 입력 대상 모두에서 (D146)', () => {
  it('일괄 입력이 켜진 탭이면 연결된 창 모두에서 따로 돈다. 창마다 완료 안내는 띄우지 않는다', async () => {
    useWorkspace.getState().upsertSession(session('s2'));
    useWorkspace.getState().upsertSession(session('s3', { state: 'closed' }));
    tabWith(['s1', 's2', 's3'], true);
    expect(await runMacroOnTargets(macro(['uptime']))).toBe(2);
    expect(sentTo('s1')).toEqual(['uptime\r']);
    expect(sentTo('s2')).toEqual(['uptime\r']);
    expect(sentTo('s3')).toEqual([]);
    const texts = useApp.getState().toasts.map((x) => x.text);
    expect(texts).toEqual(['매크로 점검 을(를) 창 2개에서 실행합니다']);
  });

  it('일괄 입력이 꺼져 있으면 지금 창 하나에서만', async () => {
    useWorkspace.getState().upsertSession(session('s2'));
    tabWith(['s1', 's2'], false);
    expect(await runMacroOnTargets(macro(['uptime']))).toBe(1);
    expect(sentTo('s1')).toEqual(['uptime\r']);
    expect(sentTo('s2')).toEqual([]);
  });

  it('이미 매크로가 도는 창은 건너뛰고 알린다', async () => {
    useWorkspace.getState().upsertSession(session('s2'));
    tabWith(['s1', 's2'], true);
    runMacro('s2', 'other', '다른 것', [newMacroStep({ send: 'a' }), newMacroStep({ send: 'b', expect: 'never' })]);
    termInput.mockClear();
    expect(await runMacroOnTargets(macro(['uptime', 'w']))).toBe(1);
    expect(sentTo('s1')).toEqual(['uptime\r']);
    expect(sentTo('s2')).toEqual([]);
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('이미 매크로가 도는 창 1개는 건너뛰었습니다');
    expect(useMacros.getState().running.s2?.macroId).toBe('other');
  });

  it('대상 중 운영 서버가 있고 위험 명령이 들었으면 먼저 묻고, 취소하면 어디에도 보내지 않는다', async () => {
    useWorkspace.getState().upsertSession(session('s2', { env: 'prod' }));
    tabWith(['s1', 's2'], true);
    setSettings([], { dangerousPatterns: ['rm\\s+-rf'] });
    const confirm = vi.fn(async () => false);
    const prev = useUi.getState().confirm;
    useUi.setState({ confirm });
    try {
      expect(await runMacroOnTargets(macro(['rm -rf /tmp/x']))).toBe(0);
    } finally {
      useUi.setState({ confirm: prev });
    }
    expect(confirm).toHaveBeenCalledTimes(1);
    expect(termInput).not.toHaveBeenCalled();
  });
});

describe('매크로 단축키: 다시 실행·중지 (D147)', () => {
  it('마지막 매크로 다시 실행: 아직 없으면 알리고, 있으면 같은 값으로 다시', async () => {
    tabWith(['s1'], false);
    replayLastMacro();
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('아직 실행한 매크로가 없습니다');

    const m = macro(['echo {{msg}}']);
    useApp.setState((s) => ({ store: { ...s.store!, snippets: [m] } }));
    expect(await runMacroOnTargets(m, { msg: 'hi' })).toBe(1);
    expect(sentTo('s1')).toEqual(['echo hi\r']);
    replayLastMacro();
    await vi.waitFor(() => expect(sentTo('s1')).toEqual(['echo hi\r', 'echo hi\r']));
  });

  it('마지막 매크로가 지워졌으면 알린다', async () => {
    tabWith(['s1'], false);
    const m = macro(['uptime']);
    useApp.setState((s) => ({ store: { ...s.store!, snippets: [m] } }));
    await runMacroOnTargets(m);
    useApp.setState((s) => ({ store: { ...s.store!, snippets: [] } }));
    replayLastMacro();
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('마지막으로 실행한 매크로가 지워졌습니다');
  });

  it('중지: 지금 대상에서 도는 매크로를 멈추고, 없으면 알린다', async () => {
    useWorkspace.getState().upsertSession(session('s2'));
    tabWith(['s1', 's2'], true);
    stopMacrosOnTargets();
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('이 창에서 실행 중인 매크로가 없습니다');

    const m = macro(['a', 'b'], { steps: [newMacroStep({ send: 'a' }), newMacroStep({ send: 'b', expect: 'never' })] });
    expect(await runMacroOnTargets(m)).toBe(2);
    expect(Object.keys(useMacros.getState().running).sort()).toEqual(['s1', 's2']);
    stopMacrosOnTargets();
    expect(useMacros.getState().running).toEqual({});
  });
});

describe('서버별 출력 트리거 (D149)', () => {
  const withHost = (hostRules: TriggerRule[], globalRules: TriggerRule[]) => {
    setSettings(globalRules);
    useApp.setState((s) => ({ store: { ...s.store!, hosts: [{ id: 'h1', triggerRules: hostRules } as unknown as Host] } }));
  };
  const toasts = () => useApp.getState().toasts.map((x) => x.text);

  it('그 서버의 세션에서만 서버 규칙이 동작하고, 설정의 규칙은 모든 세션에서 동작한다', () => {
    useWorkspace.getState().upsertSession(session('s1', { hostId: 'h1' }));
    useWorkspace.getState().upsertSession(session('s2'));
    withHost([rule({ id: 'own', pattern: 'ONLYH1' })], [rule({ id: 'g', pattern: 'GLOBAL' })]);
    emitOutput('s1', 'ONLYH1 GLOBAL\r\n');
    emitOutput('s2', 'ONLYH1 GLOBAL\r\n');
    expect(toasts()).toEqual(['s1: ONLYH1 GLOBAL', 's1: ONLYH1 GLOBAL', 's2: ONLYH1 GLOBAL']);
  });

  it('서버 규칙을 바꾸면 다음 출력부터 새 규칙을 쓴다', () => {
    useWorkspace.getState().upsertSession(session('s1', { hostId: 'h1' }));
    withHost([rule({ id: 'old', pattern: 'OLD' })], []);
    emitOutput('s1', 'OLD one\r\n');
    expect(toasts()).toEqual(['s1: OLD one']);
    withHost([rule({ id: 'new', pattern: 'NEW' })], []);
    vi.advanceTimersByTime(1100);
    emitOutput('s1', 'OLD two\r\nNEW three\r\n');
    expect(toasts()).toEqual(['s1: OLD one', 's1: NEW three']);
  });
});
