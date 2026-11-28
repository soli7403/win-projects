// 출력 트리거의 순수 계산: 규칙 → 정규식, 줄에서 맞는 규칙 찾기, 반복 제한.
// 근거: spec/behavior/terminal.md §10 (D140). 알림·소리·입력을 실제로 하는 쪽은 state/triggers.ts.
import type { TriggerRule } from '../ipc/types';
import { isValidPattern } from './highlight';
import { OutputLines } from './outputLines';

/** 규칙 수 한도. 넘는 규칙은 쓰지 않는다. */
export const TRIGGER_MAX_RULES = 30;
/** 같은 규칙이 한 세션에서 다시 실행되기까지의 최소 간격(ms). */
export const TRIGGER_COOLDOWN_MS = 1000;
/** 자동 응답 반복 제한: 이 시간(ms) 안에 이 횟수를 넘으면 그 세션에서 그 규칙을 끈다. */
export const TRIGGER_LOOP_WINDOW_MS = 10_000;
export const TRIGGER_LOOP_MAX = 5;
/** 줄바꿈 없이 멈춘 줄(프롬프트)은 출력이 이만큼(ms) 조용한 뒤에 살핀다. */
export const TRIGGER_IDLE_MS = 300;

export interface CompiledTrigger {
  id: string;
  re: RegExp;
  notify: boolean;
  sound: boolean;
  /** 자동 응답으로 보낼 글자(Enter 포함). 없으면 빈 글자. */
  reply: string;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 켜져 있고 쓸 수 있는 규칙만 정규식으로 만든다. 하는 일이 하나도 없는 규칙은 뺀다. */
export function compileTriggers(rules: readonly TriggerRule[] | undefined): CompiledTrigger[] {
  const out: CompiledTrigger[] = [];
  for (const r of rules ?? []) {
    if (out.length >= TRIGGER_MAX_RULES) break;
    if (!r || !r.enabled || typeof r.pattern !== 'string' || !isValidPattern(r.pattern, !!r.regex)) continue;
    // 자동 응답: 켜져 있고 보낼 것(글자 또는 Enter)이 있을 때만.
    const reply = r.replyOn ? (typeof r.reply === 'string' ? r.reply : '') + (r.replyEnter ? '\r' : '') : '';
    if (!r.notify && !r.sound && !reply) continue;
    try {
      out.push({
        id: r.id,
        re: new RegExp(r.regex ? r.pattern : escapeRegExp(r.pattern), r.caseSensitive ? '' : 'i'),
        notify: !!r.notify,
        sound: !!r.sound,
        reply,
      });
    } catch {
      // 만들 수 없는 정규식은 건너뛴다
    }
  }
  return out;
}

export interface TriggerFire {
  rule: CompiledTrigger;
  /** 맞은 줄(앞뒤 빈칸 제거). */
  line: string;
  /** 자동 응답이 너무 자주 반복돼 이번부터 이 세션에서 이 규칙을 껐다(이번 응답은 보내지 않는다). */
  muted: boolean;
}

/**
 * 세션 하나의 출력을 살펴 실행할 규칙을 고른다.
 *   - 한 줄에서 규칙마다 한 번만(같은 줄이 이어서 길어져도 다시 실행하지 않는다).
 *   - 같은 규칙은 TRIGGER_COOLDOWN_MS 에 한 번만.
 *   - 자동 응답이 TRIGGER_LOOP_WINDOW_MS 안에 TRIGGER_LOOP_MAX 번을 넘으면 그 규칙을 이 세션에서 끈다
 *     (응답의 에코가 다시 규칙에 맞아 끝없이 도는 것을 막는다).
 */
export class TriggerWatcher {
  private lines = new OutputLines();
  private firedOnLine = new Set<string>();
  private lastFire = new Map<string, number>();
  private replyTimes = new Map<string, number[]>();
  private muted = new Set<string>();
  /** 사용자가 지금 줄에 글자를 치는 중인지(에코에 반응하지 않게). 줄이 바뀌면 풀린다. */
  private typing = false;
  private partial = '';

  constructor(private now: () => number = () => Date.now()) {}

  /** 사용자가 이 세션에 보낸 입력을 알린다. 글자를 쳤으면 그 줄의 에코는 살피지 않는다. */
  noteInput(data: string): void {
    if (!data || data.startsWith('\x1b')) return;
    // Enter 로 끝나면 줄이 곧 바뀐다. 그 밖의 글자는 지금 줄에 에코된다.
    if (/[^\r\n]/.test(data)) this.typing = true;
  }

  /** 출력 조각. 이번에 끝난 줄에서 실행할 규칙을 돌려준다. */
  feed(chunk: string, rules: readonly CompiledTrigger[]): TriggerFire[] {
    const r = this.lines.feed(chunk);
    const fires: TriggerFire[] = [];
    for (const line of r.lines) {
      if (!this.typing) this.match(line, rules, fires);
      this.firedOnLine.clear();
      this.typing = false;
    }
    this.partial = r.partial;
    return fires;
  }

  /** 출력이 조용해졌을 때: 줄바꿈 없이 멈춘 지금 줄(프롬프트)을 살핀다. */
  idle(rules: readonly CompiledTrigger[]): TriggerFire[] {
    const fires: TriggerFire[] = [];
    if (!this.typing && this.partial.trim()) this.match(this.partial, rules, fires);
    return fires;
  }

  private match(line: string, rules: readonly CompiledTrigger[], out: TriggerFire[]): void {
    const text = line.trim();
    if (!text) return;
    const t = this.now();
    for (const rule of rules) {
      if (this.firedOnLine.has(rule.id) || this.muted.has(rule.id)) continue;
      if (!rule.re.test(text)) continue;
      const last = this.lastFire.get(rule.id);
      if (last !== undefined && t - last < TRIGGER_COOLDOWN_MS) continue;
      this.firedOnLine.add(rule.id);
      this.lastFire.set(rule.id, t);
      let muted = false;
      if (rule.reply) {
        const times = (this.replyTimes.get(rule.id) ?? []).filter((x) => t - x < TRIGGER_LOOP_WINDOW_MS);
        times.push(t);
        this.replyTimes.set(rule.id, times);
        if (times.length > TRIGGER_LOOP_MAX) {
          muted = true;
          this.muted.add(rule.id);
        }
      }
      out.push({ rule, line: text, muted });
    }
  }
}

/** 새 규칙의 기본값: 알림만 켜짐. */
export function newTriggerRule(): TriggerRule {
  return { id: crypto.randomUUID(), pattern: '', regex: false, caseSensitive: false, notify: true, sound: false, replyOn: false, reply: '', replyEnter: true, enabled: true };
}
