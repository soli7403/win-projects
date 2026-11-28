// 매크로 실행·녹화 상태. 근거: spec/behavior/terminal.md §11 (D141). 계산은 lib/macro.ts.
import { create } from 'zustand';
import { getBackend } from '../ipc/backend';
import type { MacroStep, Snippet } from '../ipc/types';
import {
  isMacro, MACRO_EXPECT_TIMEOUT_MS, MACRO_QUIET_MAX_MS, MACRO_QUIET_MS, MacroRecorder, OutputTail, unescapeKeys,
} from '../lib/macro';
import { DANGEROUS_PATTERNS } from '../lib/defaults';
import { emitAppInput, onInput, onOutput } from '../lib/outputHub';
import { fillVars } from '../lib/snippetVars';
import { matchesDangerous } from '../lib/termGuard';
import { focusedSessionId, focusTerminal, inputTargets, notifyNoFocus } from '../lib/termSend';
import { relabel, t } from '../i18n';
import { useApp } from './app';
import { useUi } from './ui';
import { useWorkspace } from './workspace';

export interface MacroRun {
  /** 실행 중인 매크로(명령 모음 항목)의 id 와 이름. */
  macroId: string;
  name: string;
  /** 지금 기다리는 단계(0부터)와 전체 단계 수. */
  step: number;
  total: number;
}

interface MacroState {
  /** 세션 id → 그 세션에서 실행 중인 매크로. 세션마다 하나만 돈다. */
  running: Record<string, MacroRun>;
  /** 녹화 중인 세션과 지금까지 모은 단계 수. 녹화는 한 번에 하나만. */
  recording: { sessionId: string; count: number } | null;
  /** 방금 끝난 녹화의 단계. 명령 모음 패널이 가져가 편집 창을 연다(가져가면 null). */
  recorded: MacroStep[] | null;
}

export const useMacros = create<MacroState>(() => ({ running: {}, recording: null, recorded: null }));

interface Job {
  steps: MacroStep[];
  values: Record<string, string>;
  index: number;
  tail: OutputTail;
  /** 지금 단계의 시한(기다릴 글자) 또는 조용함 타이머. */
  timer: ReturnType<typeof setTimeout> | null;
  /** 조용함을 기다리기 시작한 때(상한 계산). */
  quietSince: number;
  /** 여러 창에서 함께 실행 중이면 참: 창마다 "완료" 안내를 띄우지 않는다(D146). */
  quiet: boolean;
}

const jobs = new Map<string, Job>();
let recorder: MacroRecorder | null = null;
/** 마지막으로 실행한 매크로와 그때의 변수 값(단축키 "마지막 매크로 다시 실행", D147). 앱이 켜져 있는 동안만 기억한다. */
let lastRun: { macroId: string; values: Record<string, string> } | null = null;

function setRun(sessionId: string, run: MacroRun | null): void {
  useMacros.setState((s) => {
    const running = { ...s.running };
    if (run) running[sessionId] = run;
    else delete running[sessionId];
    return { running };
  });
}

function clearTimer(job: Job): void {
  if (job.timer) clearTimeout(job.timer);
  job.timer = null;
}

function finish(sessionId: string, toast?: { kind: 'ok' | 'error' | 'info'; text: string }): void {
  const job = jobs.get(sessionId);
  if (!job) return;
  clearTimer(job);
  jobs.delete(sessionId);
  setRun(sessionId, null);
  if (toast) useApp.getState().pushToast(toast);
}

function nameOf(sessionId: string): string {
  return useMacros.getState().running[sessionId]?.name ?? '';
}

/** 지금 단계의 입력을 보내고 다음 단계로 간다. */
function sendStep(sessionId: string, job: Job): void {
  clearTimer(job);
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  if (!info || info.state !== 'connected') {
    finish(sessionId, { kind: 'error', text: t('macro.disconnected', { name: nameOf(sessionId) }) });
    return;
  }
  const step = job.steps[job.index];
  const text = unescapeKeys(fillVars(step.send, job.values)) + (step.enter ? '\r' : '');
  job.tail.reset(); // 보낸 뒤의 출력만 다음 단계의 조건으로 본다
  if (text) {
    getBackend().termInput(sessionId, text);
    emitAppInput(sessionId, text);
  }
  job.index++;
  if (job.index >= job.steps.length) {
    finish(sessionId, job.quiet ? undefined : { kind: 'ok', text: t('macro.done', { name: nameOf(sessionId) }) });
    return;
  }
  const cur = useMacros.getState().running[sessionId];
  if (cur) setRun(sessionId, { ...cur, step: job.index });
  armStep(sessionId, job);
}

/** 지금 단계의 기다림을 건다: 기다릴 글자가 있으면 시한, 없으면 출력이 조용해지기를 기다린다. */
function armStep(sessionId: string, job: Job): void {
  clearTimer(job);
  const step = job.steps[job.index];
  if (step.expect) {
    if (job.tail.consume(step.expect)) {
      sendStep(sessionId, job);
      return;
    }
    job.timer = setTimeout(() => {
      finish(sessionId, {
        kind: 'error',
        text: t('macro.timeout', { name: nameOf(sessionId), text: step.expect, sec: Math.round(MACRO_EXPECT_TIMEOUT_MS / 1000) }),
      });
    }, MACRO_EXPECT_TIMEOUT_MS);
    return;
  }
  job.quietSince = Date.now();
  job.timer = setTimeout(() => sendStep(sessionId, job), MACRO_QUIET_MS);
}

function onSessionOutput(sessionId: string, text: string): void {
  const job = jobs.get(sessionId);
  if (!job) return;
  job.tail.feed(text);
  const step = job.steps[job.index];
  if (step.expect) {
    if (job.tail.consume(step.expect)) sendStep(sessionId, job);
    return;
  }
  // 출력이 왔다: 조용함을 다시 잰다. 다만 상한을 넘겼으면 더 기다리지 않는다.
  clearTimer(job);
  const left = MACRO_QUIET_MAX_MS - (Date.now() - job.quietSince);
  job.timer = setTimeout(() => sendStep(sessionId, job), Math.max(0, Math.min(MACRO_QUIET_MS, left)));
}

/**
 * 매크로를 세션에서 실행한다. 첫 단계는 기다릴 글자가 없으면 바로 보낸다.
 * 그 세션에서 이미 매크로가 돌고 있으면 실행하지 않고 false.
 */
export function runMacro(
  sessionId: string,
  macroId: string,
  name: string,
  steps: readonly MacroStep[],
  values: Record<string, string> = {},
  opts: { quiet?: boolean } = {},
): boolean {
  if (steps.length === 0) return false;
  if (jobs.has(sessionId)) {
    useApp.getState().pushToast({ kind: 'info', text: t('macro.busy') });
    return false;
  }
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  if (!info || info.state !== 'connected') {
    useApp.getState().pushToast({ kind: 'info', text: t('macro.needConnected') });
    return false;
  }
  const job: Job = { steps: [...steps], values, index: 0, tail: new OutputTail(), timer: null, quietSince: Date.now(), quiet: !!opts.quiet };
  jobs.set(sessionId, job);
  setRun(sessionId, { macroId, name: name || relabel(info.title), step: 0, total: steps.length });
  if (job.steps[0].expect) armStep(sessionId, job);
  else sendStep(sessionId, job);
  return true;
}

/**
 * 매크로를 지금 입력 대상에서 실행한다(D146): 일괄 입력이 켜진 탭이면 그 탭의 연결된 창 모두, 아니면 지금 창 하나.
 * 운영 서버 창이 하나라도 있고 위험 명령이 들었으면 먼저 묻는다(붙여넣기와 같은 규칙).
 * 여러 창이면 창마다 따로 돌고(창마다 출력을 기다린다), 이미 매크로가 도는 창은 건너뛴다. 시작한 창 수를 돌려준다.
 */
export async function runMacroOnTargets(s: Snippet, values: Record<string, string> = {}): Promise<number> {
  if (!isMacro(s)) return 0;
  const focused = focusedSessionId();
  if (!focused) {
    notifyNoFocus();
    return 0;
  }
  const ws = useWorkspace.getState();
  const all = inputTargets();
  const multi = all.length > 1;
  const targets = multi ? all.filter((sid) => ws.sessions[sid]?.info.state === 'connected') : [focused];
  const st = useApp.getState().store?.settings;
  const text = s.steps.map((x) => fillVars(x.send, values)).join('\n');
  const prod = targets.map((sid) => ws.sessions[sid]?.info).find((info) => info?.env === 'prod');
  if (st?.guardDangerousOnProd && prod && matchesDangerous(text, st.dangerousPatterns ?? DANGEROUS_PATTERNS)) {
    const ok = await useUi.getState().confirm({
      title: t('run.dangerTitle'),
      body: t('run.dangerBody', { name: relabel(prod.title) }),
      detail: text.slice(0, 2000),
      okText: t('run.ok'),
      danger: true,
    });
    if (!ok) return 0;
  }
  const name = s.name || t('snip.noName');
  lastRun = { macroId: s.id, values }; // 단축키 "다시 실행"용 (D147)
  if (!multi) {
    const ok = runMacro(focused, s.id, name, s.steps, values);
    if (ok) focusTerminal();
    return ok ? 1 : 0;
  }
  let started = 0;
  let busy = 0;
  for (const sid of targets) {
    if (jobs.has(sid)) {
      busy++;
      continue;
    }
    // 여러 창에서 돌 때는 안내에 창 이름을 붙여 어느 창의 일인지 보이게 한다.
    const label = `${name} · ${relabel(ws.sessions[sid]?.info.title) || t('tab.session')}`;
    if (runMacro(sid, s.id, label, s.steps, values, { quiet: true })) started++;
  }
  const push = useApp.getState().pushToast;
  if (started > 0) push({ kind: 'info', text: t('macro.broadcastStarted', { name, count: started }) });
  if (busy > 0) push({ kind: 'info', text: t('macro.broadcastBusy', { count: busy }) });
  if (started === 0 && busy === 0) push({ kind: 'info', text: t('macro.needConnected') });
  if (started > 0) focusTerminal();
  return started;
}

/** 마지막으로 실행한 매크로를 같은 변수 값으로 지금 입력 대상에서 다시 실행한다(단축키, D147). */
export function replayLastMacro(): void {
  const push = useApp.getState().pushToast;
  if (!lastRun) {
    push({ kind: 'info', text: t('macro.noLast') });
    return;
  }
  const { macroId, values } = lastRun;
  const s = useApp.getState().store?.snippets.find((x) => x.id === macroId);
  if (!s || !isMacro(s)) {
    lastRun = null;
    push({ kind: 'info', text: t('macro.lastGone') });
    return;
  }
  void runMacroOnTargets(s, values);
}

/** 지금 입력 대상(일괄 입력이면 탭의 모든 창)에서 도는 매크로를 멈춘다(단축키, D147). 멈춘 것이 없으면 알린다. */
export function stopMacrosOnTargets(): void {
  const targets = inputTargets().filter((sid) => jobs.has(sid));
  if (targets.length === 0) {
    useApp.getState().pushToast({ kind: 'info', text: t('macro.nothingRunning') });
    return;
  }
  for (const sid of targets) stopMacro(sid);
}

/** 실행 중인 매크로를 멈춘다(사용자가 [중지]). */
export function stopMacro(sessionId: string): void {
  if (!jobs.has(sessionId)) return;
  finish(sessionId, { kind: 'info', text: t('macro.stopped', { name: nameOf(sessionId) }) });
}

/** 지금 창의 입력을 녹화하기 시작한다. 이미 녹화 중이면 false. */
export function startRecording(sessionId: string): boolean {
  if (recorder) return false;
  recorder = new MacroRecorder();
  useMacros.setState({ recording: { sessionId, count: 0 } });
  return true;
}

/** 녹화를 끝내고 모은 단계를 돌려준다. 녹화 중이 아니면 빈 배열. */
export function stopRecording(): MacroStep[] {
  const r = recorder;
  recorder = null;
  useMacros.setState({ recording: null });
  return r ? r.finish() : [];
}

/**
 * 매크로 녹화 시작/정지(명령 모음 패널·상태 표시줄·빠른 연결의 명령이 부른다).
 * 시작: 지금 창의 입력을 모으기 시작한다. 정지: 모은 단계가 있으면 명령 모음 패널을 열어 편집 창을 띄우게 한다.
 */
export function toggleRecording(): void {
  const push = useApp.getState().pushToast;
  if (useMacros.getState().recording) {
    const steps = stopRecording();
    if (steps.length === 0) {
      push({ kind: 'info', text: t('macro.recordEmpty') });
      return;
    }
    const ui = useUi.getState();
    ui.setSnippetsOpen(true);
    if (!ui.sidebarOpen) ui.toggleSidebar();
    useMacros.setState({ recorded: steps });
    return;
  }
  const sid = focusedSessionId();
  if (!sid) {
    notifyNoFocus();
    return;
  }
  if (startRecording(sid)) {
    push({ kind: 'info', text: t('macro.recordStarted') });
    focusTerminal();
  }
}

/** 앱 시작 때 한 번 건다. 돌려준 함수는 구독을 끊는다. */
export function startMacros(): () => void {
  const offs = [
    onOutput(onSessionOutput),
    onInput((sessionId, data, fromUser) => {
      // 녹화는 사용자가 직접 친 입력만 모은다(명령 모음·자동 응답이 보낸 것은 빼고).
      const rec = useMacros.getState().recording;
      if (!fromUser || !recorder || !rec || rec.sessionId !== sessionId) return;
      recorder.feed(data);
      const count = recorder.count();
      if (count !== rec.count) useMacros.setState({ recording: { sessionId, count } });
    }),
    // 세션이 닫히면 그 세션의 실행·녹화를 끝낸다.
    useWorkspace.subscribe((s) => {
      for (const sid of [...jobs.keys()]) if (!s.sessions[sid]) finish(sid);
      const rec = useMacros.getState().recording;
      if (rec && !s.sessions[rec.sessionId]) stopRecording();
    }),
  ];
  return () => {
    offs.forEach((off) => off());
    resetMacrosForTest();
  };
}

/** 테스트용: 실행·녹화 상태를 모두 버린다. */
export function resetMacrosForTest(): void {
  for (const job of jobs.values()) clearTimer(job);
  jobs.clear();
  recorder = null;
  lastRun = null;
  useMacros.setState({ running: {}, recording: null, recorded: null });
}
