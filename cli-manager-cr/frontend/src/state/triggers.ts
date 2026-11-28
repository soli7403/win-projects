// 출력 트리거 실행: 세션 출력에서 규칙에 맞는 줄이 나오면 알림·소리·자동 응답을 한다.
// 근거: spec/behavior/terminal.md §10 (D140). 계산은 lib/trigger.ts.
import { getBackend } from '../ipc/backend';
import type { TriggerRule } from '../ipc/types';
import { beep } from '../lib/beep';
import { emitAppInput, onInput, onOutput } from '../lib/outputHub';
import { compileTriggers, TRIGGER_IDLE_MS, TriggerWatcher, type CompiledTrigger, type TriggerFire } from '../lib/trigger';
import { relabel, t } from '../i18n';
import { useApp } from './app';
import { useWorkspace } from './workspace';

/** 알림에 보이는 줄의 최대 길이. */
const LINE_SHOWN_MAX = 120;

interface Watch {
  watcher: TriggerWatcher;
  idle: ReturnType<typeof setTimeout> | null;
}

const watches = new Map<string, Watch>();
let compiledFrom: readonly TriggerRule[] | undefined;
let compiled: CompiledTrigger[] = [];

/** 지금 설정의 규칙(배열이 바뀌었을 때만 다시 만든다). */
function globalRules(): CompiledTrigger[] {
  const rules = useApp.getState().store?.settings.triggerRules;
  if (rules !== compiledFrom) {
    compiledFrom = rules;
    compiled = compileTriggers(Array.isArray(rules) ? rules : []);
  }
  return compiled;
}

/** 서버 id → 그 서버 규칙을 만든 원본 배열과 결과(배열이 바뀌었을 때만 다시 만든다). */
const hostCompiled = new Map<string, { from: readonly TriggerRule[] | undefined; rules: CompiledTrigger[]; merged: CompiledTrigger[]; base: CompiledTrigger[] }>();

/** 이 세션에 쓸 규칙: 설정의 규칙 + 그 세션 서버의 규칙(D149). 저장된 서버가 아니면 설정의 규칙만. */
function currentRules(sessionId: string): CompiledTrigger[] {
  const base = globalRules();
  const hostId = useWorkspace.getState().sessions[sessionId]?.info.hostId;
  if (!hostId) return base;
  const own = useApp.getState().store?.hosts.find((h) => h.id === hostId)?.triggerRules;
  let entry = hostCompiled.get(hostId);
  if (!entry || entry.from !== own) {
    entry = { from: own, rules: compileTriggers(Array.isArray(own) ? own : []), merged: [], base: [] };
    hostCompiled.set(hostId, entry);
  }
  if (entry.rules.length === 0) return base;
  if (entry.base !== base || entry.merged.length === 0) {
    entry.base = base;
    entry.merged = [...base, ...entry.rules];
  }
  return entry.merged;
}

function watchOf(sessionId: string): Watch {
  let w = watches.get(sessionId);
  if (!w) {
    w = { watcher: new TriggerWatcher(), idle: null };
    watches.set(sessionId, w);
  }
  return w;
}

/** Windows 알림이 이번 실행에서 실패했으면 더 시도하지 않는다(실패할 때마다 오류를 만들지 않게). */
let osNotifyBroken = false;

/** 앱 창이 뒤에 있으면(포커스가 없으면) Windows 알림으로도 띄운다(D145). */
function notifyOs(title: string, body: string): void {
  if (osNotifyBroken) return;
  if (useApp.getState().store?.settings.triggerOsNotify === false) return;
  if (typeof document === 'undefined' || document.hasFocus()) return;
  void getBackend()
    .invoke<boolean>('app.notify', [title, body])
    .then((ok) => {
      if (ok === false) osNotifyBroken = true;
    })
    .catch(() => {
      osNotifyBroken = true;
    });
}

function run(sessionId: string, fires: TriggerFire[]): void {
  if (fires.length === 0) return;
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  if (!info) return;
  const name = relabel(info.title) || t('tab.session');
  const push = useApp.getState().pushToast;
  const guardProd = !!useApp.getState().store?.settings.guardDangerousOnProd && info.env === 'prod';
  let sounded = false;
  let osLines: string[] = [];
  for (const f of fires) {
    const line = f.line.length > LINE_SHOWN_MAX ? `${f.line.slice(0, LINE_SHOWN_MAX)}…` : f.line;
    if (f.rule.notify) {
      push({ kind: 'info', text: t('trig.toast', { name, line }) });
      osLines = [...osLines, line];
    }
    if (f.rule.sound && !sounded) {
      sounded = true;
      beep();
    }
    if (!f.rule.reply) continue;
    if (f.muted) {
      push({ kind: 'error', text: t('trig.muted', { name }) });
      continue;
    }
    if (guardProd) {
      // 운영 서버에는 확인 없이 입력하지 않는다(위험 명령 확인과 같은 설정을 따른다).
      push({ kind: 'info', text: t('trig.prodSkipped', { name }) });
      continue;
    }
    if (info.state === 'connected') {
      getBackend().termInput(sessionId, f.rule.reply);
      emitAppInput(sessionId, f.rule.reply);
    }
  }
  // 한 번에 맞은 줄들은 Windows 알림 하나로 묶는다(D145).
  if (osLines.length > 0) notifyOs(t('trig.osTitle', { name }), osLines.join('\n'));
}

/** 출력 조각 하나를 살핀다. 끝난 줄은 바로, 줄바꿈 없이 멈춘 줄은 출력이 조용해진 뒤에 본다. */
export function feedTriggers(sessionId: string, text: string): void {
  const rules = currentRules(sessionId);
  if (rules.length === 0 && !watches.has(sessionId)) return;
  const w = watchOf(sessionId);
  run(sessionId, w.watcher.feed(text, rules));
  if (w.idle) clearTimeout(w.idle);
  w.idle = setTimeout(() => {
    w.idle = null;
    run(sessionId, w.watcher.idle(currentRules(sessionId)));
  }, TRIGGER_IDLE_MS);
}

function dropWatch(sessionId: string): void {
  const w = watches.get(sessionId);
  if (!w) return;
  if (w.idle) clearTimeout(w.idle);
  watches.delete(sessionId);
}

/** 앱 시작 때 한 번 건다. 돌려준 함수는 구독을 끊는다. */
export function startTriggers(): () => void {
  const offs = [
    onOutput(feedTriggers),
    // 직접 친 입력이든 앱이 대신 보낸 입력이든, 그 줄의 에코에는 반응하지 않는다.
    onInput((sessionId, data) => watches.get(sessionId)?.watcher.noteInput(data)),
    // 닫힌 세션의 감시는 버린다.
    useWorkspace.subscribe((s) => {
      for (const sid of [...watches.keys()]) if (!s.sessions[sid]) dropWatch(sid);
    }),
  ];
  return () => {
    offs.forEach((off) => off());
    for (const sid of [...watches.keys()]) dropWatch(sid);
  };
}

/** 테스트용: 감시 상태를 모두 버린다. */
export function resetTriggersForTest(): void {
  for (const sid of [...watches.keys()]) dropWatch(sid);
  compiledFrom = undefined;
  compiled = [];
  hostCompiled.clear();
  osNotifyBroken = false;
}
