// 매크로의 순수 계산: 제어 문자 적기·읽기, 녹화한 입력을 단계로 나누기, 출력에서 글자 기다리기.
// 근거: spec/behavior/terminal.md §11 (D141). 실행·녹화 상태는 state/macros.ts.
import type { MacroStep, Snippet } from '../ipc/types';
import { OutputLines } from './outputLines';

/** 매크로 하나의 최대 단계 수. */
export const MACRO_STEP_MAX = 50;
/** 기다릴 글자가 나오기를 기다리는 최대 시간(ms). 넘으면 매크로를 멈춘다. */
export const MACRO_EXPECT_TIMEOUT_MS = 30_000;
/** 기다릴 글자가 없는 단계: 출력이 이만큼(ms) 조용하면 다음 입력을 보낸다. */
export const MACRO_QUIET_MS = 500;
/** 출력이 계속 이어져도 이만큼(ms) 지나면 다음 입력을 보낸다. */
export const MACRO_QUIET_MAX_MS = 10_000;
/** 기다릴 글자를 찾는 최근 출력의 최대 길이. */
const TAIL_MAX = 4000;

export function newMacroStep(p: Partial<MacroStep> = {}): MacroStep {
  return { id: `ms-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`, expect: '', send: '', enter: true, ...p };
}

/** 이 항목이 매크로인지(단계가 하나라도 있는지). */
export function isMacro(s: Snippet): s is Snippet & { steps: MacroStep[] } {
  return Array.isArray(s.steps) && s.steps.length > 0;
}

/** 저장할 단계: 보낼 것이 없는 단계(글자도 Enter 도 없음)는 빼고, 최대 개수까지만. */
export function cleanSteps(steps: readonly MacroStep[]): MacroStep[] {
  return steps.filter((s) => s.send !== '' || s.enter).slice(0, MACRO_STEP_MAX);
}

/** 매크로를 목록·검색·옛 버전에서 보일 한 덩어리 글로: 단계의 보낼 글자를 줄마다. */
export function macroBody(steps: readonly MacroStep[]): string {
  return steps.map((s) => s.send).join('\n');
}

/** 입력 글자를 편집할 수 있는 글로 적는다: 제어 문자는 `\xNN`, ESC 는 `\e`, 탭은 `\t`, 역슬래시는 `\\`. */
export function escapeKeys(raw: string): string {
  let out = '';
  for (const ch of raw) {
    const code = ch.codePointAt(0) ?? 0;
    if (ch === '\\') out += '\\\\';
    else if (code === 0x1b) out += '\\e';
    else if (code === 0x09) out += '\\t';
    else if (code === 0x0d) out += '\\r';
    else if (code === 0x0a) out += '\\n';
    else if (code < 0x20 || code === 0x7f) out += `\\x${code.toString(16).padStart(2, '0')}`;
    else out += ch;
  }
  return out;
}

/** escapeKeys 의 반대. 모르는 `\?` 는 글자 그대로 둔다. */
export function unescapeKeys(text: string): string {
  return text.replace(/\\(x[0-9a-fA-F]{2}|[etrn\\])/g, (_m, g: string) => {
    if (g === 'e') return '\x1b';
    if (g === 't') return '\t';
    if (g === 'r') return '\r';
    if (g === 'n') return '\n';
    if (g === '\\') return '\\';
    return String.fromCharCode(parseInt(g.slice(1), 16));
  });
}

/** 터미널이 서버의 질의에 스스로 답하는 입력(장치 속성·커서 위치·포커스 알림). 사용자가 친 것이 아니므로 녹화하지 않는다. */
const AUTO_REPLY = /^\x1b\[(?:[?>]?[\d;]*[cRn]|[IO])$/;

/**
 * 사용자가 친 입력을 단계로 모은다. Enter 마다 한 단계(보낸 뒤 Enter)가 되고,
 * 백스페이스는 그 줄에 친 마지막 글자를 지운다(지울 글자가 없으면 그대로 적는다).
 */
export class MacroRecorder {
  private steps: MacroStep[] = [];
  /** 지금 줄에 친 것(글자 단위. 제어 문자도 한 칸). */
  private line: string[] = [];

  feed(data: string): void {
    if (!data || AUTO_REPLY.test(data)) return;
    // 방향키 같은 ESC 시퀀스는 한 덩어리로 넣는다(백스페이스가 반쪽만 지우지 않게).
    if (data.startsWith('\x1b') && data.length > 1 && !data.includes('\r')) {
      this.line.push(data);
      return;
    }
    for (const ch of data) {
      if (ch === '\r') {
        this.flush(true);
      } else if (ch === '\n') {
        // 붙여넣기의 LF 는 CR 로 이미 왔다고 보고 버린다
      } else if ((ch === '\x7f' || ch === '\b') && this.line.length > 0 && isPrintable(this.line[this.line.length - 1])) {
        this.line.pop();
      } else {
        this.line.push(ch);
      }
    }
  }

  private flush(enter: boolean): void {
    if (this.steps.length >= MACRO_STEP_MAX) {
      this.line = [];
      return;
    }
    const send = escapeKeys(this.line.join(''));
    this.line = [];
    if (send === '' && !enter) return;
    this.steps.push(newMacroStep({ send, enter }));
  }

  /** 지금까지 모은 단계 수(진행 표시용). */
  count(): number {
    return this.steps.length + (this.line.length > 0 ? 1 : 0);
  }

  /** 녹화를 끝내고 단계를 돌려준다. Enter 없이 남은 입력은 Enter 를 붙이지 않는 단계가 된다. */
  finish(): MacroStep[] {
    this.flush(false);
    const out = this.steps;
    this.steps = [];
    return out;
  }
}

function isPrintable(s: string): boolean {
  return s.length > 0 && !s.startsWith('\x1b') && (s.codePointAt(0) ?? 0) >= 0x20 && s !== '\x7f';
}

/** 출력에서 글자를 기다리는 데 쓰는 최근 출력(ANSI 를 지운 글). 대소문자는 구분하지 않는다. */
export class OutputTail {
  private lines = new OutputLines();
  private done = '';
  private partial = '';
  /** 지금 줄에서 이미 버린 앞부분의 길이(reset 뒤에 같은 줄이 이어질 때 다시 보지 않게). */
  private skip = 0;
  /** 줄 모으개가 들고 있는 지금 줄 전체의 길이. */
  private rawPartial = 0;

  feed(chunk: string): void {
    const r = this.lines.feed(chunk);
    for (const line of r.lines) {
      this.done += `${line.slice(this.skip)}\n`;
      this.skip = 0;
    }
    if (this.done.length > TAIL_MAX) this.done = this.done.slice(this.done.length - TAIL_MAX);
    this.rawPartial = r.partial.length;
    if (this.skip > this.rawPartial) this.skip = 0; // 줄을 처음부터 다시 썼다(CR)
    this.partial = r.partial.slice(this.skip);
  }

  /** 지금까지의 출력에 needle 이 있으면 지금까지의 출력을 버리고 true. */
  consume(needle: string): boolean {
    const hay = (this.done + this.partial).toLowerCase();
    if (hay.indexOf(needle.toLowerCase()) < 0) return false;
    this.reset();
    return true;
  }

  /** 지금까지의 출력을 버린다(입력을 보낸 직후: 그 뒤의 출력만 기다린다). */
  reset(): void {
    this.done = '';
    this.partial = '';
    this.skip = this.rawPartial;
  }
}
