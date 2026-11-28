# 출력 트리거 — 출력에 글자가 나오면 알림·소리·자동 응답, 설정의 "출력 규칙" 구역

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D140). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 키워드 강조 D113. 명세 [spec/behavior/terminal.md](../../spec/behavior/terminal.md) §10·§10.1, [spec/ui/settings.md](../../spec/ui/settings.md) §2b, [spec/data/store.md](../../spec/data/store.md) §4
**적용 순서**: 2026-11-27 의 4번째. 5번(매크로)이 이 문서의 `outputHub.ts`·`outputLines.ts` 를 쓴다.

---

## 0. 한 줄 요약

터미널 출력의 한 줄에 정해 둔 글자가 나오면 **알림**(화면 구석), **소리**, **자동 응답**(그 연결에 글자 입력)을 한다. 안 보이는 탭의 출력도 본다. 설정에 **"출력 규칙"** 구역을 새로 두고 키워드 강조를 그리로 옮겼다.

| 쓰임 | 규칙 |
|---|---|
| 긴 빌드가 끝나면 알기 | 찾을 글자 `BUILD SUCCESS` → 알림 + 소리 |
| 다른 탭의 오류 알기 | 찾을 글자 `ERROR` → 알림 |
| 확인 질문 넘기기 | 찾을 글자 `[y/N]` → 자동 응답 `y` + Enter |

## 1. 배경

2026-11-27 사용자 요청("권장 순서대로 구현")의 셋째 항목. 원본 앱에 없던 기능이라 명세도 이번에 새로 썼다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 실행하는 곳 | **프런트**(백엔드가 보낸 출력 묶음을 받는 지점) | 설정·알림·소리가 모두 프런트에 있다. 안 보이는 탭의 출력도 같은 길로 온다 |
| 화면과의 관계 | xterm 과 따로 계산(`outputHub`) | 탭이 한 번도 그려지지 않았어도 돌고, 듣는 쪽이 느려도 화면 출력(흐름 제어 ack)을 막지 않는다 |
| 찾는 단위 | **줄**. 끝난 줄은 바로, 멈춘 줄(프롬프트)은 0.3초 조용해진 뒤 | 한 줄에 여러 번 울리지 않게 하고, 줄바꿈 없는 확인 질문도 잡는다 |
| 입력한 줄의 에코 | 살피지 않는다(그 줄이 끝날 때까지) | `grep error` 를 치는 동안 `error` 규칙이 울리지 않게. 명령 모음·매크로·자동 응답이 보낸 입력도 같다 |
| 전체 화면 프로그램 | 살피지 않는다 | vim·less 안의 글자는 줄 단위 출력이 아니다(키워드 강조와 같은 규칙) |
| 반복 제한 | 한 줄에 규칙당 한 번, 같은 규칙은 1초에 한 번 | 로그가 쏟아질 때 토스트가 넘치지 않게(토스트는 5개까지만 보인다) |
| 자동 응답의 폭주 | 10초에 5번을 넘으면 그 세션에서 그 규칙을 끈다 | 응답의 에코나 다시 나온 질문이 규칙에 또 맞아 끝없이 도는 것을 막는다 |
| 운영 서버 | "운영 서버 위험 명령 확인"이 켜져 있으면 자동 응답을 **보내지 않고** 알린다 | 운영 서버의 `[y/N]` 에 확인 없이 `y` 를 치는 것은 위험 명령 확인과 같은 무게다. 그 설정 하나로 함께 끄고 켠다 |
| 적용 범위 | 모든 세션(서버별 규칙 없음) | 키워드 강조와 같다. 서버별 규칙은 필요해지면 더한다 |
| 설정 위치 | 새 구역 **"출력 규칙"**(단축키 다음): 키워드 강조 + 출력 트리거 | 둘 다 "찾을 글자 + 할 일"이라 나란히 두는 편이 덜 헷갈린다. "화면" 구역도 짧아진다 |
| 소리 | 설정의 "벨 소리"와 무관하게 낸다 | 규칙에서 소리를 켠 것 자체가 뜻이다 |

알려 둘 것:
- 알림은 앱 안의 토스트다. 앱이 다른 창 뒤에 있을 때의 Windows 알림은 아직 없다(소리는 난다).
- Claude Code 같은 화면을 다시 그리는 CLI 는 줄 단위로 출력하지 않아 규칙이 맞지 않을 수 있다.

### 2.1 영향 범위

- **설정 데이터**: `triggerRules`(빈 배열) 추가. 계약 고정 파일 `frontend/src/ipc/contract/store.get*.json` 을 다시 만들었다(`go test ./internal/app -run TestContract -update`).
- **설정 화면**: 구역이 10개 → 11개. 키워드 강조가 "화면"에서 "출력 규칙"으로 옮겨졌다(항목의 모양·데이터는 그대로).
- **터미널 입력 경로**: 사용자가 친 입력과 앱이 보낸 입력을 `outputHub` 에 알린다(서버로 가는 글자는 그대로).

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 내용 |
|---|---|---|
| `frontend/src/lib/outputLines.ts` | 코드 | **새 파일** — 출력 조각 → ANSI 를 지운 줄(조각 경계·CR·백스페이스·대체 화면) |
| `frontend/src/lib/trigger.ts` | 코드 | **새 파일** — 규칙 → 정규식, `TriggerWatcher`(줄당 한 번·쿨다운·폭주 차단·입력 에코 건너뛰기) |
| `frontend/src/lib/outputHub.ts` | 코드 | **새 파일** — 출력·입력을 듣는 창구 |
| `frontend/src/lib/beep.ts` | 코드 | **새 파일** — `TerminalView` 에 있던 소리 함수를 옮김 |
| `frontend/src/state/triggers.ts` | 코드 | **새 파일** — 실행(토스트·소리·입력), 세션별 감시, 0.3초 타이머 |
| `frontend/src/screens/SettingsRules.tsx` | 코드 | **새 파일** — "출력 규칙" 구역(키워드 강조 + 출력 트리거 표) |
| `frontend/src/screens/Settings.tsx`, `SettingsAppearance.tsx`, `state/ui.ts` | 코드 | 구역 추가, 키워드 강조를 옮김 |
| `frontend/src/state/actions.ts`, `App.tsx`, `screens/TerminalView.tsx`, `lib/termSend.ts` | 코드 | 출력·입력을 `outputHub` 에 알림, 시작 때 구독 |
| `frontend/src/ipc/types.ts`, `lib/defaults.ts`, `internal/store/store.go` | 코드 | `TriggerRule`, 설정 기본값 |
| `frontend/src/lib/outputLines.test.ts`, `trigger.test.ts`, `state/automation.test.ts`, `screens/Automation.test.tsx` | 테스트 | **새 파일**(줄 7개, 규칙 11개, 실행 6개, 설정 화면 2개) |
| `frontend/src/screens/SettingsLog.test.tsx`, `SettingsShortcuts.test.tsx` | 테스트 | 구역 목록 기대값 10개 → 11개 |
| `locales/*.json` | 문구 | `settings.sec.rules`, `trig.*` 10개 |
| `spec/behavior/terminal.md`, `spec/ui/settings.md`, `spec/data/store.md`, `spec/acceptance/04-hosts-sftp-settings-sync.md`(G8b~G8d) | 명세 | 수정 |

아래 `diff` 는 **오늘(2026-11-27) 작업을 시작하기 전의 파일**과 지금 파일을 견준 것이다. 이 폴더는 git 저장소가 아니라서, 오늘 적용한 수정 기록을 거꾸로 돌려 고치기 전 파일을 되살린 뒤 뽑았다(되살린 것에 수정을 다시 적용하면 지금 파일과 같아지는 것을 확인했다). 한 파일을 오늘의 여러 문서가 고친 경우, 각 문서에는 그 문서의 몫이 든 조각만 싣는다 — 조각 안의 문맥 줄에 다른 문서의 코드가 섞여 있을 수 있고, 줄 번호는 다섯 문서를 모두 적용한 최종 파일 기준이다. 새 파일은 최종 내용 전체를 싣는다(같은 날의 추가 지시 D142·D143 까지 반영된 것).

## B. 코드

### 먼저 읽을 곳

[frontend/src/lib/trigger.ts](../../frontend/src/lib/trigger.ts) `TriggerWatcher`:

```ts
feed(chunk, rules)   // 끝난 줄마다 match(). 줄이 끝나면 firedOnLine 과 typing 을 지운다
idle(rules)          // 출력이 조용해졌을 때: 멈춘 줄(partial)을 match()
noteInput(data)      // 글자가 든 입력이면 typing = true → 그 줄의 에코는 건너뛴다
```

`match` 는 규칙마다 ① 이 줄에서 이미 실행했는지 ② 이 세션에서 꺼졌는지 ③ 1초 쿨다운 ④ 자동 응답이면 10초 창의 횟수를 본다.

[frontend/src/state/triggers.ts](../../frontend/src/state/triggers.ts) `run`: 알림 → 소리(한 번) → 자동 응답(꺼졌으면 오류 토스트, 운영 서버면 정보 토스트, 아니면 `termInput` + `emitAppInput`).

[frontend/src/state/actions.ts](../../frontend/src/state/actions.ts): `session.data` 를 받으면 화면에 쓰고(`writeOutput`) 이어서 `emitOutput` 으로 알린다.

### `frontend/src/lib/outputLines.ts` — 새 파일 (전체)

```ts
// 터미널 출력 조각을 "글자만 남긴 줄"로 바꾼다. 출력 트리거와 매크로가 출력에서 글자를 찾는 데 쓴다.
// 근거: spec/behavior/terminal.md §10.1 (D140). 화면(xterm)과는 따로 돈다 — 안 보이는 탭의 출력도 본다.

/** 한 줄로 모으는 최대 길이. 넘으면 앞을 버린다(줄바꿈 없이 이어지는 출력에서 메모리를 지킨다). */
export const LINE_MAX = 2000;

export interface FeedResult {
  /** 이번 조각에서 끝난 줄들(줄바꿈으로 끝난 것). */
  lines: string[];
  /** 아직 끝나지 않은 지금 줄(프롬프트처럼 줄바꿈 없이 멈춘 글). */
  partial: string;
}

type State = 'text' | 'esc' | 'csi' | 'osc' | 'oscEsc' | 'skip1';

/**
 * 출력 조각을 받아 ANSI 제어 시퀀스를 지우고 줄로 나눈다. 조각이 시퀀스 중간에서 잘려도 이어서 본다.
 *   - `\n` 으로 줄이 끝난다. `\r\n` 은 줄바꿈 하나다.
 *   - `\r` 뒤에 다른 글자가 오면 그 줄을 처음부터 다시 쓴 것으로 본다(진행 막대 등).
 *   - 백스페이스는 마지막 글자를 지운다. 그 밖의 제어 문자는 버린다(탭은 빈칸).
 *   - 전체 화면 프로그램(대체 화면: vim·less·htop)에 들어가 있는 동안의 글자는 버린다.
 */
export class OutputLines {
  private state: State = 'text';
  private params = '';
  private cur = '';
  private pendingCr = false;
  /** 대체 화면(전체 화면 프로그램) 안인지. */
  alt = false;

  feed(chunk: string): FeedResult {
    const lines: string[] = [];
    for (let i = 0; i < chunk.length; i++) {
      const ch = chunk[i];
      const code = chunk.charCodeAt(i);
      switch (this.state) {
        case 'esc':
          if (ch === '[') {
            this.state = 'csi';
            this.params = '';
          } else if (ch === ']') this.state = 'osc';
          else if (ch === '(' || ch === ')' || ch === '#' || ch === '%') this.state = 'skip1';
          else this.state = 'text';
          continue;
        case 'skip1':
          this.state = 'text';
          continue;
        case 'csi':
          if (code >= 0x40 && code <= 0x7e) {
            this.onCsi(this.params, ch);
            this.state = 'text';
          } else if (this.params.length < 64) this.params += ch;
          continue;
        case 'osc':
          if (code === 0x07) this.state = 'text';
          else if (code === 0x1b) this.state = 'oscEsc';
          continue;
        case 'oscEsc':
          this.state = ch === '\\' ? 'text' : 'osc';
          continue;
        default:
          break;
      }
      if (code === 0x1b) {
        this.state = 'esc';
        continue;
      }
      if (ch === '\n') {
        this.pendingCr = false;
        if (!this.alt) lines.push(this.cur);
        this.cur = '';
        continue;
      }
      if (ch === '\r') {
        this.pendingCr = true;
        continue;
      }
      if (this.pendingCr) {
        // CR 뒤에 줄바꿈이 아닌 글자: 같은 줄을 처음부터 다시 쓴다.
        this.pendingCr = false;
        this.cur = '';
      }
      if (ch === '\b' || code === 0x7f) {
        this.cur = this.cur.slice(0, -1);
        continue;
      }
      if (code < 0x20) {
        if (ch === '\t') this.push(' ');
        continue;
      }
      this.push(ch);
    }
    return { lines, partial: this.alt ? '' : this.cur };
  }

  private push(ch: string): void {
    if (this.alt) return;
    this.cur += ch;
    if (this.cur.length > LINE_MAX) this.cur = this.cur.slice(this.cur.length - LINE_MAX);
  }

  /** 대체 화면 켜기·끄기(CSI ? 47 · 1047 · 1049 h/l)만 본다. */
  private onCsi(params: string, final: string): void {
    if ((final !== 'h' && final !== 'l') || !params.startsWith('?')) return;
    const modes = params.slice(1).split(';');
    if (modes.some((m) => m === '47' || m === '1047' || m === '1049')) {
      this.alt = final === 'h';
      this.cur = '';
      this.pendingCr = false;
    }
  }
}
```

### `frontend/src/lib/outputHub.ts` — 새 파일 (전체)

```ts
// 세션 출력·입력을 지켜보는 곳(출력 트리거, 매크로)이 듣는 창구. 근거: spec/behavior/terminal.md §10.1 (D140).
// 터미널 화면에 쓰는 경로(state/workspace.ts writeOutput)와는 따로다 — 듣는 쪽이 느려도 화면 출력을 막지 않는다.

type OutputListener = (sessionId: string, text: string) => void;
/** fromUser: 사용자가 터미널에 직접 친 입력이면 true, 앱이 대신 보낸 입력(명령 모음·매크로·자동 응답)이면 false. */
type InputListener = (sessionId: string, data: string, fromUser: boolean) => void;

const outputListeners = new Set<OutputListener>();
const inputListeners = new Set<InputListener>();

/** 백엔드의 session.data 묶음마다 부른다. 듣는 쪽의 오류는 다른 쪽에 번지지 않게 삼킨다. */
export function emitOutput(sessionId: string, text: string): void {
  for (const fn of outputListeners) {
    try {
      fn(sessionId, text);
    } catch {
      /* 듣는 쪽 오류는 출력 흐름과 무관 */
    }
  }
}

function emitInput(sessionId: string, data: string, fromUser: boolean): void {
  for (const fn of inputListeners) {
    try {
      fn(sessionId, data, fromUser);
    } catch {
      /* 무시 */
    }
  }
}

/** 사용자가 터미널에 친 입력마다 부른다(TerminalView). */
export function emitUserInput(sessionId: string, data: string): void {
  emitInput(sessionId, data, true);
}

/** 앱이 대신 보낸 입력(명령 모음·매크로·자동 응답)마다 부른다. 그 에코에 출력 트리거가 반응하지 않게 알린다. */
export function emitAppInput(sessionId: string, data: string): void {
  emitInput(sessionId, data, false);
}

export function onOutput(fn: OutputListener): () => void {
  outputListeners.add(fn);
  return () => outputListeners.delete(fn);
}

export function onInput(fn: InputListener): () => void {
  inputListeners.add(fn);
  return () => inputListeners.delete(fn);
}
```

### `frontend/src/lib/trigger.ts` — 새 파일 (전체)

```ts
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
```

### `frontend/src/lib/beep.ts` — 새 파일 (전체)

```ts
// 짧은 알림 소리(터미널 벨, 출력 트리거). 소리를 낼 수 없는 환경에서는 조용히 넘어간다.
export function beep(): void {
  try {
    const ctx = new AudioContext();
    const osc = ctx.createOscillator();
    osc.frequency.value = 880;
    osc.connect(ctx.destination);
    osc.start();
    osc.stop(ctx.currentTime + 0.08);
    osc.onended = () => void ctx.close();
  } catch {
    /* 소리를 낼 수 없으면 무시 */
  }
}
```

### `frontend/src/state/triggers.ts` — 새 파일 (전체)

```ts
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
function currentRules(): CompiledTrigger[] {
  const rules = useApp.getState().store?.settings.triggerRules;
  if (rules !== compiledFrom) {
    compiledFrom = rules;
    compiled = compileTriggers(Array.isArray(rules) ? rules : []);
  }
  return compiled;
}

function watchOf(sessionId: string): Watch {
  let w = watches.get(sessionId);
  if (!w) {
    w = { watcher: new TriggerWatcher(), idle: null };
    watches.set(sessionId, w);
  }
  return w;
}

function run(sessionId: string, fires: TriggerFire[]): void {
  if (fires.length === 0) return;
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  if (!info) return;
  const name = relabel(info.title) || t('tab.session');
  const push = useApp.getState().pushToast;
  const guardProd = !!useApp.getState().store?.settings.guardDangerousOnProd && info.env === 'prod';
  let sounded = false;
  for (const f of fires) {
    const line = f.line.length > LINE_SHOWN_MAX ? `${f.line.slice(0, LINE_SHOWN_MAX)}…` : f.line;
    if (f.rule.notify) push({ kind: 'info', text: t('trig.toast', { name, line }) });
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
}

/** 출력 조각 하나를 살핀다. 끝난 줄은 바로, 줄바꿈 없이 멈춘 줄은 출력이 조용해진 뒤에 본다. */
export function feedTriggers(sessionId: string, text: string): void {
  const rules = currentRules();
  if (rules.length === 0 && !watches.has(sessionId)) return;
  const w = watchOf(sessionId);
  run(sessionId, w.watcher.feed(text, rules));
  if (w.idle) clearTimeout(w.idle);
  w.idle = setTimeout(() => {
    w.idle = null;
    run(sessionId, w.watcher.idle(currentRules()));
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
}
```

### `frontend/src/screens/SettingsRules.tsx` — 새 파일 (전체)

```tsx
// 설정 · 출력 규칙: 터미널 출력에 반응하는 규칙을 한곳에 둔다 — 키워드 강조(색)와 출력 트리거(알림·소리·자동 응답).
// 근거: spec/ui/settings.md §2b, spec/behavior/terminal.md §9·§10 (D140).
import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import type { TriggerRule } from '../ipc/types';
import { Field, Switch } from '../ui/Controls';
import { isValidPattern } from '../lib/highlight';
import { newTriggerRule, TRIGGER_MAX_RULES } from '../lib/trigger';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { SettingsHighlight } from './SettingsHighlight';
import { t } from '../i18n';

function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="small" style={{ display: 'inline-flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap' }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

export function SettingsTriggers() {
  const saved = useApp((s) => s.store?.settings.triggerRules);
  const [rows, setRows] = useState<TriggerRule[]>(() => (Array.isArray(saved) ? saved : []));
  const [prev, setPrev] = useState(saved);
  // 밖에서 바뀌면(동기화·백업 가져오기) 화면을 맞춘다.
  if (prev !== saved) {
    setPrev(saved);
    setRows(Array.isArray(saved) ? saved : []);
  }

  /** 화면을 바꾸고, save 면 곧바로 저장한다(글자 칸은 벗어날 때 저장). */
  const apply = (next: TriggerRule[], save: boolean) => {
    setRows(next);
    if (save) void saveSettings({ triggerRules: next });
  };
  const patch = (id: string, p: Partial<TriggerRule>, save = true) => apply(rows.map((r) => (r.id === id ? { ...r, ...p } : r)), save);
  const saveNow = () => void saveSettings({ triggerRules: rows });

  return (
    <Field label={t('trig.title')} help={t('trig.help')}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {rows.map((r) => {
          const bad = r.pattern !== '' && !isValidPattern(r.pattern, r.regex);
          return (
            <div key={r.id} className="trig-row">
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <Switch checked={r.enabled} onChange={(v) => patch(r.id, { enabled: v })} label={t('hl.enabled')} />
                <input
                  className="input mono"
                  aria-label={t('hl.pattern')}
                  aria-invalid={bad}
                  placeholder={t('hl.pattern')}
                  style={{ flex: '1 1 180px', minWidth: 120, borderColor: bad ? 'var(--danger)' : undefined }}
                  value={r.pattern}
                  onChange={(e) => patch(r.id, { pattern: e.target.value }, false)}
                  onBlur={saveNow}
                />
                <Check label={t('hl.regex')} checked={r.regex} onChange={(v) => patch(r.id, { regex: v })} />
                <Check label={t('hl.case')} checked={r.caseSensitive} onChange={(v) => patch(r.id, { caseSensitive: v })} />
                <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => apply(rows.filter((x) => x.id !== r.id), true)}>
                  <Trash2 size={14} />
                </button>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', paddingLeft: 44 }}>
                <Check label={t('trig.notify')} checked={r.notify} onChange={(v) => patch(r.id, { notify: v })} />
                <Check label={t('trig.sound')} checked={r.sound} onChange={(v) => patch(r.id, { sound: v })} />
                <Check label={t('trig.reply')} checked={r.replyOn} onChange={(v) => patch(r.id, { replyOn: v })} />
                {r.replyOn && (
                  <>
                    <input
                      className="input mono"
                      aria-label={t('trig.replyText')}
                      placeholder={t('trig.replyText')}
                      style={{ flex: '1 1 140px', minWidth: 100 }}
                      value={r.reply}
                      onChange={(e) => patch(r.id, { reply: e.target.value }, false)}
                      onBlur={saveNow}
                    />
                    <Check label={t('trig.replyEnter')} checked={r.replyEnter} onChange={(v) => patch(r.id, { replyEnter: v })} />
                  </>
                )}
              </div>
            </div>
          );
        })}
        <div>
          <button type="button" className="btn small-btn" disabled={rows.length >= TRIGGER_MAX_RULES} onClick={() => apply([...rows, newTriggerRule()], true)}>
            {t('hl.add')}
          </button>
        </div>
      </div>
    </Field>
  );
}

export function SettingsRules() {
  return (
    <>
      <SettingsHighlight />
      <SettingsTriggers />
    </>
  );
}
```

### `frontend/src/App.tsx` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/App.tsx
+++ b/frontend/src/App.tsx
@@ -7,6 +7,8 @@
 import { useRequests, type UiRequest } from './state/requests';
 import { useWorkspace } from './state/workspace';
 import { subscribeSessionEvents, runShortcut } from './state/actions';
+import { startTriggers } from './state/triggers';
+import { startMacros } from './state/macros';
 import { keyCapture } from './lib/keys';
 import { activeShortcut } from './state/shortcuts';
 import { ContextMenuHost } from './ui/ContextMenu';
@@ -58,6 +60,8 @@
       offs = [
         subscribeAppEvents(),
         subscribeSessionEvents(),
+        startTriggers(),
+        startMacros(),
         subscribeUpdateEvents(),
         b.on('ui.request', (r: UiRequest) => useRequests.getState().push(r)),
       ];
```

### `frontend/src/ipc/types.ts` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -78,6 +91,37 @@
   /** true 면 글자색 대신 바탕을 칠한다(글자는 흰색). */
   background: boolean;
   enabled: boolean;
+}
+
+/** 출력 트리거 규칙 하나(behavior/terminal.md §10, D140). */
+export interface TriggerRule {
+  id: string;
+  /** 찾을 글자(regex 가 true 면 정규식). 출력의 한 줄에 대해 찾는다. */
+  pattern: string;
+  regex: boolean;
+  caseSensitive: boolean;
+  /** 알림(화면 구석의 안내)을 띄운다. */
+  notify: boolean;
+  /** 짧은 소리를 낸다. */
+  sound: boolean;
+  /** 자동 응답을 보낸다. */
+  replyOn: boolean;
+  /** 자동 응답으로 입력할 글자. */
+  reply: string;
+  /** 자동 응답 뒤에 Enter 를 붙인다. */
+  replyEnter: boolean;
+  enabled: boolean;
+}
+
+/** 매크로의 한 단계(behavior/terminal.md §11, D141). */
+export interface MacroStep {
+  id: string;
+  /** 보내기 전에 출력에서 기다릴 글자. 비면 출력이 잠잠해질 때까지 기다린다. */
+  expect: string;
+  /** 보낼 글자. 제어 문자는 `\x03` 처럼 적는다. `{{이름}}` 변수를 쓸 수 있다. */
+  send: string;
+  /** 보낸 뒤 Enter 를 붙인다. */
+  enter: boolean;
 }
 
 export interface KnownHost {
@@ -106,6 +150,8 @@
   dangerousPatterns: string[];
   /** 터미널 출력의 키워드 강조 규칙. 앞에 있는 규칙이 우선이다 (behavior/terminal.md §9). */
   highlightRules: HighlightRule[];
+  /** 출력 트리거 규칙 (behavior/terminal.md §10). */
+  triggerRules: TriggerRule[];
   autoReconnect: boolean;
   autoLockMinutes: number;
   uploadDir: string;
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -33,6 +33,7 @@
   guardDangerousOnProd: true,
   dangerousPatterns: [...DANGEROUS_PATTERNS],
   highlightRules: [],
+  triggerRules: [],
   autoReconnect: true,
   autoLockMinutes: 15,
   uploadDir: '~/.dariterm/uploads',
```

### `frontend/src/lib/termSend.ts` — 수정

```diff
--- a/frontend/src/lib/termSend.ts
+++ b/frontend/src/lib/termSend.ts
@@ -1,6 +1,7 @@
 // 포커스된 칸(동시 입력 규칙 포함)으로 텍스트를 보낸다. 근거: spec/behavior/terminal.md §7, spec/ui/panes.md §5(동시 입력).
 // TerminalView 안의 입력 대상 규칙과 같은 내용을 여기서 직접 구현한다(그 함수는 칸 안에 갇혀 있어 호출하지 않는다).
 import { getBackend } from '../ipc/backend';
+import { emitAppInput } from './outputHub';
 import { useApp } from '../state/app';
 import { useWorkspace } from '../state/workspace';
 import { findPane, sessionsIn } from './layout';
@@ -29,7 +30,10 @@
   const sid = focusedSessionId();
   if (!sid) return false;
   const text = body.replace(/\r?\n/g, '\r') + (sendEnter ? '\r' : '');
-  for (const target of inputTargets()) getBackend().termInput(target, text);
+  for (const target of inputTargets()) {
+    getBackend().termInput(target, text);
+    emitAppInput(target, text);
+  }
   focusTerminal();
   return true;
 }
```

### `frontend/src/screens/Settings.tsx` — 수정

```diff
--- a/frontend/src/screens/Settings.tsx
+++ b/frontend/src/screens/Settings.tsx
@@ -1,6 +1,6 @@
-// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 10개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
+// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 11개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
 import { useState, type ReactNode, type KeyboardEvent } from 'react';
-import { Palette, Keyboard, Command, ImageUp, Plug, ScrollText, RefreshCw, Shield, Archive, Info } from 'lucide-react';
+import { Palette, Keyboard, Command, ImageUp, Plug, ScrollText, RefreshCw, Shield, Archive, Info, Zap } from 'lucide-react';
 import { Modal } from '../ui/Modal';
 import { useUi, type SettingsSection } from '../state/ui';
 import { useApp } from '../state/app';
@@ -9,6 +9,7 @@
 import { SettingsAppearance } from './SettingsAppearance';
 import { SettingsInput } from './SettingsInput';
 import { SettingsShortcuts } from './SettingsShortcuts';
+import { SettingsRules } from './SettingsRules';
 import { SettingsFiles } from './SettingsFiles';
 import { SettingsConnection } from './SettingsConnection';
 import { SettingsLog } from './SettingsLog';
@@ -23,6 +24,7 @@
   { id: 'appearance', label: 'settings.sec.appearance', icon: Palette },
   { id: 'input', label: 'settings.sec.input', icon: Keyboard },
   { id: 'shortcuts', label: 'settings.sec.shortcuts', icon: Command },
+  { id: 'rules', label: 'settings.sec.rules', icon: Zap },
   { id: 'files', label: 'settings.sec.files', icon: ImageUp },
   { id: 'connection', label: 'settings.sec.connection', icon: Plug },
   { id: 'log', label: 'settings.sec.log', icon: ScrollText },
@@ -101,6 +103,7 @@
     appearance: <SettingsAppearance />,
     input: <SettingsInput />,
     shortcuts: <SettingsShortcuts />,
+    rules: <SettingsRules />,
     files: <SettingsFiles />,
     connection: <SettingsConnection />,
     log: <SettingsLog />,
```

### `frontend/src/screens/SettingsAppearance.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsAppearance.tsx
+++ b/frontend/src/screens/SettingsAppearance.tsx
@@ -1,5 +1,3 @@
-import { SettingsHighlight } from './SettingsHighlight';
-      <SettingsHighlight />
 // 설정 1 · 모양. 근거: spec/ui/settings.md §1, spec/ui/tokens.md §6, spec/acceptance/04 G1·G2·G8.
 import { useState } from 'react';
 import { ToggleRow, Segmented, Field } from '../ui/Controls';
```

### `frontend/src/screens/SettingsHighlight.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsHighlight.tsx
+++ b/frontend/src/screens/SettingsHighlight.tsx
@@ -1,4 +1,4 @@
-// 설정 1 · 화면의 "키워드 강조" 부분. 근거: spec/ui/settings.md §1 키워드 강조, spec/behavior/terminal.md §9 (D113).
+// 설정 · 출력 규칙의 "키워드 강조" 부분(D140 에서 "화면" 구역에서 옮김). 근거: spec/ui/settings.md §2b, spec/behavior/terminal.md §9 (D113).
 import { useState } from 'react';
 import { Trash2 } from 'lucide-react';
 import type { HighlightRule } from '../ipc/types';
```

### `frontend/src/screens/TerminalView.tsx` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/screens/TerminalView.tsx
+++ b/frontend/src/screens/TerminalView.tsx
@@ -1,17 +1,3 @@
-function beep(): void {
-  try {
-    const ctx = new AudioContext();
-    const osc = ctx.createOscillator();
-    osc.frequency.value = 880;
-    osc.connect(ctx.destination);
-    osc.start();
-    osc.stop(ctx.currentTime + 0.08);
-    osc.onended = () => void ctx.close();
-  } catch {
-    /* 소리를 낼 수 없으면 무시 */
-  }
-}
-
 // 터미널 칸 본문(xterm.js). 근거: spec/behavior/terminal.md §1~§5, §6, §8, spec/behavior/shortcuts.md §2~§3, spec/ui/panes.md §2.3~§2.4.
 import { useEffect, useRef, useState } from 'react';
 import { Terminal, type IDisposable } from '@xterm/xterm';
@@ -37,6 +23,8 @@
 import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';
 // 이 파일은 t 를 터미널 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
 import { relabel, t as tr } from '../i18n';
+import { beep } from '../lib/beep';
+import { emitUserInput } from '../lib/outputHub';
 
 interface Props {
   paneId: string;
@@ -378,7 +366,11 @@
       }
     }
     tracker.feed(data);
-    for (const sid of targetsFor(k.paneId, sessionId)) getBackend().termInput(sid, data);
+    for (const sid of targetsFor(k.paneId, sessionId)) {
+      getBackend().termInput(sid, data);
+      // 사용자가 친 입력을 알린다(출력 트리거가 에코에 반응하지 않게, 매크로 녹화).
+      emitUserInput(sid, data);
+    }
   });
 
   // 선택하면 자동 복사: 마우스로 선택을 끝냈을 때만(찾기가 바꾼 선택은 복사하지 않는다, KI-011).
```

### `frontend/src/state/actions.ts` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/state/actions.ts
+++ b/frontend/src/state/actions.ts
@@ -1,5 +1,6 @@
 // 공통 동작: 서버 열기(새 탭·칸), 단축키 실행, 세션 이벤트 연결. 근거: spec/behavior/shortcuts.md, spec/ui/panes.md §4·§9, spec/ipc.md §4.
 import { getBackend, errorMessage } from '../ipc/backend';
+import { emitOutput } from '../lib/outputHub';
 import type { SessionInfo } from '../ipc/types';
 import { useApp } from './app';
 import { useUi } from './ui';
@@ -216,7 +217,11 @@
       ws.upsertSession(info);
       if (info.state === 'error' && info.message) useApp.getState().pushToast({ kind: 'error', text: info.message });
     }),
-    b.on('session.data', (id: string, text: string) => useWorkspace.getState().writeOutput(id, text)),
+    b.on('session.data', (id: string, text: string) => {
+      useWorkspace.getState().writeOutput(id, text);
+      // 출력을 지켜보는 쪽(출력 트리거·매크로)에도 알린다 (terminal.md §10.1).
+      emitOutput(id, text);
+    }),
   ];
   return () => offs.forEach((off) => off());
 }
```

### `frontend/src/state/ui.ts` — 수정

```diff
--- a/frontend/src/state/ui.ts
+++ b/frontend/src/state/ui.ts
@@ -45,7 +45,7 @@
   | { kind: 'host'; hostId: string | null; groupId: string | null }
   | { kind: 'folder'; folderId: string | null; parentId: string | null };
 
-export type SettingsSection = 'appearance' | 'input' | 'shortcuts' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';
+export type SettingsSection = 'appearance' | 'input' | 'shortcuts' | 'rules' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';
 
 interface UiState {
   /** 설정 화면(크게 보기 모달). section 은 열 때 보일 구역(기본 모양). */
```

### `frontend/src/styles/settings.css` — 수정

```diff
--- a/frontend/src/styles/settings.css
+++ b/frontend/src/styles/settings.css
@@ -36,6 +36,8 @@
 .cred-row + .cred-row { border-top: 1px solid var(--border); }
 .cred-row .cred-main { flex: 1; min-width: 0; }
 .cred-row .cred-sub { font-size: 11.5px; color: var(--fg-3); }
+/* 출력 트리거 규칙 한 건(두 줄: 찾을 글자 / 할 일). 근거: spec/ui/settings.md §2b. */
+.trig-row { display: flex; flex-direction: column; gap: 6px; padding: 8px 10px; border: 1px solid var(--border); border-radius: 8px; }
 .pill-soft { font-size: 10.5px; font-weight: 700; padding: 1px 7px; border-radius: 10px; background: var(--panel-2); color: var(--fg-2); margin-left: 6px; }
 .iconlike { display: inline-flex; align-items: center; gap: 6px; }
 .set-table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -42,6 +42,7 @@
 		`systemctl\s+(stop|restart)`,
 	},
 	"highlightRules":      []any{}, // 키워드 강조 규칙 (behavior/terminal.md §9, D113)
+	"triggerRules":        []any{}, // 출력 트리거 규칙 (behavior/terminal.md §10, D140)
 	"autoReconnect":       true,
 	"autoLockMinutes":     15,
 	"uploadDir":           "~/.dariterm/uploads",
```

## C. 테스트 코드

### `frontend/src/lib/outputLines.test.ts` — 새 파일 (전체)

```ts
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
```

### `frontend/src/lib/trigger.test.ts` — 새 파일 (전체)

```ts
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
```

### `frontend/src/state/automation.test.ts` — 새 파일 (이 문서의 몫: `describe('출력 트리거 실행')`, 전체는 5번 문서 §C)

```ts
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
```

### `frontend/src/screens/Automation.test.tsx` — 새 파일 (이 문서의 몫: `describe('설정 · 출력 규칙 (D140)')`, 전체는 5번 문서 §C)

```tsx
describe('설정 · 출력 규칙 (D140)', () => {
  it('키워드 강조와 출력 트리거가 한 구역에 있고, 트리거 규칙을 추가하면 저장된다', async () => {
    seed();
    render(<SettingsRules />);
    expect(screen.getByText('키워드 강조')).toBeTruthy();
    expect(screen.getByText('출력 트리거')).toBeTruthy();
    const add = screen.getAllByRole('button', { name: '+ 규칙 추가' });
    expect(add).toHaveLength(2);
    await act(async () => { fireEvent.click(add[1]); });
    const rules = patches.at(-1)?.settings?.triggerRules;
    expect(rules).toHaveLength(1);
    expect(rules?.[0]).toMatchObject({ notify: true, sound: false, replyOn: false, enabled: true });
  });

  it('자동 응답을 켜야 응답 글자 칸이 보인다', async () => {
    seed();
    render(<SettingsRules />);
    await act(async () => { fireEvent.click(screen.getAllByRole('button', { name: '+ 규칙 추가' })[1]); });
    expect(screen.queryByLabelText('응답으로 입력할 글자')).toBeNull();
    await act(async () => { fireEvent.click(screen.getByLabelText('자동 응답')); });
    expect(screen.getByLabelText('응답으로 입력할 글자')).toBeTruthy();
    expect(patches.at(-1)?.settings?.triggerRules?.[0].replyOn).toBe(true);
  });
});
```

### `frontend/src/screens/SettingsLog.test.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsLog.test.tsx
+++ b/frontend/src/screens/SettingsLog.test.tsx
@@ -33,10 +33,10 @@
 }
 
 describe('설정 > 로그 (D120)', () => {
-  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (10개)', () => {
+  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (11개)', () => {
     openSection('connection');
     const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
-    expect(names).toEqual(['화면', '입력', '단축키', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
+    expect(names).toEqual(['화면', '입력', '단축키', '출력 규칙', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
   });
 
   it('"연결"에는 로그 항목이 없고 자동 재연결·배치 복원만 남는다', () => {
```

### `frontend/src/screens/SettingsShortcuts.test.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsShortcuts.test.tsx
+++ b/frontend/src/screens/SettingsShortcuts.test.tsx
@@ -44,10 +44,10 @@
 const keyBox = (name: string, keys: string) => screen.getByRole('button', { name: `${name}: ${keys}` });
 
 describe('설정 > 단축키 (D126)', () => {
-  it('구역 목록에 "입력" 다음으로 "단축키"가 있다 (10개)', () => {
+  it('구역 목록에 "입력" 다음으로 "단축키", 그다음 "출력 규칙"이 있다 (11개)', () => {
     openSection('shortcuts');
     const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
-    expect(names).toEqual(['화면', '입력', '단축키', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
+    expect(names).toEqual(['화면', '입력', '단축키', '출력 규칙', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
   });
 
   it('"정보"에는 단축키 표가 없다', async () => {
```

## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

설정 기본값에 `triggerRules` 가 늘어 `go test ./internal/app -run TestContract -update` 로 다시 만들었다.

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -56,6 +56,7 @@
     "terminalTheme": "DariTerm Dark",
     "theme": "dark",
     "transferTempDir": "",
+    "triggerRules": [],
     "updateCheck": true,
     "uploadCleanupDays": 7,
     "uploadDir": "~/.dariterm/uploads",
```

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -56,6 +56,7 @@
     "terminalTheme": "DariTerm Dark",
     "theme": "dark",
     "transferTempDir": "",
+    "triggerRules": [],
     "updateCheck": true,
     "uploadCleanupDays": 7,
     "uploadDir": "~/.dariterm/uploads",
```

## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다. 자리표시자(`{이름}`)는 언어마다 같은 이름을 쓴다(`frontend: npx vitest run src/i18n`, `go test ./internal/i18n` 이 확인한다).

**`settings.sec.rules`** (추가)

```json
  "settings.sec.rules": "출력 규칙",
  "settings.sec.rules": "Output rules",
  "settings.sec.rules": "出力ルール",
  "settings.sec.rules": "输出规则",
  "settings.sec.rules": "輸出規則",
  "settings.sec.rules": "Ausgaberegeln",
  "settings.sec.rules": "Règles de sortie",
  "settings.sec.rules": "Reglas de salida",
  "settings.sec.rules": "Regras de saída",
  "settings.sec.rules": "Правила вывода",
```

**`trig.title`** (추가)

```json
  "trig.title": "출력 트리거",
  "trig.title": "Output triggers",
  "trig.title": "出力トリガー",
  "trig.title": "输出触发器",
  "trig.title": "輸出觸發器",
  "trig.title": "Ausgabe-Trigger",
  "trig.title": "Déclencheurs de sortie",
  "trig.title": "Disparadores de salida",
  "trig.title": "Gatilhos de saída",
  "trig.title": "Триггеры вывода",
```

**`trig.help`** (추가)

```json
  "trig.help": "터미널 출력의 한 줄에 이 글자가 나오면 알림·소리·자동 응답을 실행합니다. 열려 있는 모든 연결에 적용되고, 보이지 않는 탭의 출력도 살핍니다. 한 줄에서 규칙마다 한 번, 같은 규칙은 1초에 한 번만 실행합니다. vim 같은 전체 화면 프로그램 안에서와 직접 입력하는 줄에서는 실행하지 않습니다. 자동 응답은 그 글자가 나온 연결에 곧바로 입력되므로 조심해서 쓰세요. 운영 서버에는 보내지 않고(입력 > 운영 서버 위험 명령 확인이 켜져 있을 때), 10초에 5번 넘게 반복되면 그 연결에서는 꺼집니다.",
  "trig.help": "Runs a notification, a sound or an automatic reply when a line of terminal output contains this text. Applies to every open connection, including output in tabs you are not looking at. Each rule runs once per line and at most once per second. Nothing runs inside full-screen programs such as vim or on the line you are typing. An automatic reply is typed straight into the connection where the text appeared, so use it with care. It is not sent to production servers (while Input > Confirm dangerous commands on production servers is on), and it turns off for a connection if it repeats more than 5 times in 10 seconds.",
  "trig.help": "ターミナル出力の 1 行にこの文字が現れると、通知・音・自動応答を実行します。開いているすべての接続に適用され、表示していないタブの出力も調べます。1 行につきルールごとに 1 回、同じルールは 1 秒に 1 回だけ実行します。vim のような全画面プログラムの中や、自分で入力している行では実行しません。自動応答はその文字が現れた接続にそのまま入力されるため、注意して使ってください。本番サーバーには送らず (入力 > 本番サーバーでの危険なコマンドの確認がオンのとき)、10 秒間に 5 回を超えて繰り返すとその接続ではオフになります。",
  "trig.help": "终端输出的某一行出现这些文字时，执行通知、声音或自动应答。适用于所有已打开的连接，也会检查未显示的标签页的输出。每条规则在一行中只执行一次，同一规则每秒最多执行一次。在 vim 等全屏程序中以及您正在输入的行上不会执行。自动应答会直接输入到出现该文字的连接中，请谨慎使用。不会发送到生产服务器（在“输入 > 生产服务器危险命令确认”开启时），10 秒内重复超过 5 次则会在该连接中关闭。",
  "trig.help": "終端機輸出的某一行出現這些文字時，執行通知、聲音或自動回應。適用於所有已開啟的連線，也會檢查未顯示的分頁的輸出。每條規則在一行中只執行一次，同一規則每秒最多執行一次。在 vim 等全螢幕程式中以及您正在輸入的行上不會執行。自動回應會直接輸入到出現該文字的連線中，請謹慎使用。不會傳送到正式伺服器（在「輸入 > 正式伺服器危險命令確認」開啟時），10 秒內重複超過 5 次則會在該連線中關閉。",
  "trig.help": "Führt eine Benachrichtigung, einen Ton oder eine automatische Antwort aus, wenn eine Zeile der Terminalausgabe diesen Text enthält. Gilt für alle offenen Verbindungen, auch für die Ausgabe in nicht sichtbaren Tabs. Jede Regel läuft einmal pro Zeile und höchstens einmal pro Sekunde. In Vollbildprogrammen wie vim und in der Zeile, die Sie gerade tippen, läuft nichts. Eine automatische Antwort wird direkt in die Verbindung eingegeben, in der der Text erschien – verwenden Sie sie mit Vorsicht. An Produktionsserver wird sie nicht gesendet (solange Eingabe > Gefährliche Befehle auf Produktionsservern bestätigen aktiv ist), und sie wird für eine Verbindung abgeschaltet, wenn sie sich mehr als 5-mal in 10 Sekunden wiederholt.",
  "trig.help": "Exécute une notification, un son ou une réponse automatique lorsqu'une ligne de la sortie du terminal contient ce texte. S'applique à toutes les connexions ouvertes, y compris la sortie des onglets non affichés. Chaque règle s'exécute une fois par ligne et au plus une fois par seconde. Rien ne s'exécute dans les programmes plein écran comme vim ni sur la ligne que vous tapez. Une réponse automatique est saisie directement dans la connexion où le texte est apparu : utilisez-la avec prudence. Elle n'est pas envoyée aux serveurs de production (tant que Saisie > Confirmer les commandes dangereuses sur les serveurs de production est activé) et se désactive pour une connexion si elle se répète plus de 5 fois en 10 secondes.",
  "trig.help": "Ejecuta una notificación, un sonido o una respuesta automática cuando una línea de la salida del terminal contiene este texto. Se aplica a todas las conexiones abiertas, incluida la salida de las pestañas que no está viendo. Cada regla se ejecuta una vez por línea y como máximo una vez por segundo. No se ejecuta nada dentro de programas a pantalla completa como vim ni en la línea que está escribiendo. Una respuesta automática se escribe directamente en la conexión donde apareció el texto, así que úsela con cuidado. No se envía a servidores de producción (mientras Entrada > Confirmar comandos peligrosos en servidores de producción esté activado) y se desactiva para una conexión si se repite más de 5 veces en 10 segundos.",
  "trig.help": "Executa uma notificação, um som ou uma resposta automática quando uma linha da saída do terminal contém este texto. Vale para todas as conexões abertas, inclusive a saída de abas que você não está vendo. Cada regra é executada uma vez por linha e no máximo uma vez por segundo. Nada é executado dentro de programas em tela cheia como o vim nem na linha que você está digitando. Uma resposta automática é digitada diretamente na conexão em que o texto apareceu, portanto use com cuidado. Ela não é enviada a servidores de produção (enquanto Entrada > Confirmar comandos perigosos em servidores de produção estiver ativado) e é desativada para uma conexão se repetir mais de 5 vezes em 10 segundos.",
  "trig.help": "Выполняет уведомление, звук или автоответ, когда строка вывода терминала содержит этот текст. Действует для всех открытых подключений, включая вывод во вкладках, которые вы не видите. Каждое правило срабатывает один раз на строку и не чаще раза в секунду. Внутри полноэкранных программ, таких как vim, и в строке, которую вы набираете, ничего не выполняется. Автоответ вводится прямо в то подключение, где появился текст, поэтому используйте его осторожно. На рабочие серверы он не отправляется (пока включено «Ввод > Подтверждать опасные команды на рабочих серверах») и отключается для подключения, если повторяется более 5 раз за 10 секунд.",
```

**`trig.notify`** (추가)

```json
  "trig.notify": "알림",
  "trig.notify": "Notify",
  "trig.notify": "通知",
  "trig.notify": "通知",
  "trig.notify": "通知",
  "trig.notify": "Benachrichtigen",
  "trig.notify": "Notifier",
  "trig.notify": "Notificar",
  "trig.notify": "Notificar",
  "trig.notify": "Уведомить",
```

**`trig.sound`** (추가)

```json
  "trig.sound": "소리",
  "trig.sound": "Sound",
  "trig.sound": "音",
  "trig.sound": "声音",
  "trig.sound": "聲音",
  "trig.sound": "Ton",
  "trig.sound": "Son",
  "trig.sound": "Sonido",
  "trig.sound": "Som",
  "trig.sound": "Звук",
```

**`trig.reply`** (추가)

```json
  "trig.reply": "자동 응답",
  "trig.reply": "Auto reply",
  "trig.reply": "自動応答",
  "trig.reply": "自动应答",
  "trig.reply": "自動回應",
  "trig.reply": "Automatisch antworten",
  "trig.reply": "Réponse auto",
  "trig.reply": "Respuesta automática",
  "trig.reply": "Resposta automática",
  "trig.reply": "Автоответ",
```

**`trig.replyText`** (추가)

```json
  "trig.replyText": "응답으로 입력할 글자",
  "trig.replyText": "Text to type as the reply",
  "trig.replyText": "応答として入力する文字",
  "trig.replyText": "作为应答输入的文字",
  "trig.replyText": "作為回應輸入的文字",
  "trig.replyText": "Als Antwort einzugebender Text",
  "trig.replyText": "Texte à saisir en réponse",
  "trig.replyText": "Texto que se escribirá como respuesta",
  "trig.replyText": "Texto a digitar como resposta",
  "trig.replyText": "Текст ответа",
```

**`trig.replyEnter`** (추가)

```json
  "trig.replyEnter": "Enter 붙이기",
  "trig.replyEnter": "Add Enter",
  "trig.replyEnter": "Enter を付ける",
  "trig.replyEnter": "附加 Enter",
  "trig.replyEnter": "附加 Enter",
  "trig.replyEnter": "Enter anhängen",
  "trig.replyEnter": "Ajouter Entrée",
  "trig.replyEnter": "Añadir Enter",
  "trig.replyEnter": "Adicionar Enter",
  "trig.replyEnter": "Добавить Enter",
```

**`trig.toast`** (추가)

```json
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name}：{line}",
  "trig.toast": "{name}：{line}",
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name} : {line}",
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name}: {line}",
  "trig.toast": "{name}: {line}",
```

**`trig.muted`** (추가)

```json
  "trig.muted": "{name}: 자동 응답이 너무 자주 반복돼 이 연결에서는 껐습니다",
  "trig.muted": "{name}: the automatic reply repeated too often and was turned off for this connection",
  "trig.muted": "{name}: 自動応答の繰り返しが多すぎるため、この接続ではオフにしました",
  "trig.muted": "{name}：自动应答重复过于频繁，已在此连接中关闭",
  "trig.muted": "{name}：自動回應重複過於頻繁，已在此連線中關閉",
  "trig.muted": "{name}: Die automatische Antwort hat sich zu oft wiederholt und wurde für diese Verbindung abgeschaltet",
  "trig.muted": "{name} : la réponse automatique s'est trop répétée et a été désactivée pour cette connexion",
  "trig.muted": "{name}: la respuesta automática se repitió demasiado y se desactivó para esta conexión",
  "trig.muted": "{name}: a resposta automática se repetiu demais e foi desativada para esta conexão",
  "trig.muted": "{name}: автоответ повторялся слишком часто и отключён для этого подключения",
```

**`trig.prodSkipped`** (추가)

```json
  "trig.prodSkipped": "{name}: 운영 서버라 자동 응답을 보내지 않았습니다",
  "trig.prodSkipped": "{name}: the automatic reply was not sent because this is a production server",
  "trig.prodSkipped": "{name}: 本番サーバーのため自動応答を送りませんでした",
  "trig.prodSkipped": "{name}：这是生产服务器，未发送自动应答",
  "trig.prodSkipped": "{name}：這是正式伺服器，未傳送自動回應",
  "trig.prodSkipped": "{name}: Die automatische Antwort wurde nicht gesendet, weil dies ein Produktionsserver ist",
  "trig.prodSkipped": "{name} : la réponse automatique n'a pas été envoyée car il s'agit d'un serveur de production",
  "trig.prodSkipped": "{name}: no se envió la respuesta automática porque es un servidor de producción",
  "trig.prodSkipped": "{name}: a resposta automática não foi enviada porque este é um servidor de produção",
  "trig.prodSkipped": "{name}: автоответ не отправлен, так как это рабочий сервер",
```

## F. 명세

원본 앱에 없던 기능이라 명세를 이 작업에서 새로 썼다(아래 조각의 `+` 줄이 새 명세다). 줄이 길어 가로로 넘치면 파일을 직접 여는 편이 읽기 쉽다.

### `spec/acceptance/04-hosts-sftp-settings-sync.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/acceptance/04-hosts-sftp-settings-sync.md
+++ b/spec/acceptance/04-hosts-sftp-settings-sync.md
@@ -44,6 +51,9 @@
 | G7 | 업데이트 [업데이트 확인], 배포 저장소 미지정 | | "배포 저장소가 설정되지 않은 빌드입니다" |
 | G8 | [기본 크기로] | | 창이 1440×900 가운데 |
 | G8a | 키워드 강조 규칙 `error`(빨강), `디스크`(노랑, 바탕 칠하기) | 터미널에 `디스크 Error: disk ERROR` 출력 | `Error`·`ERROR` 는 빨간 글자, `디스크` 는 노란 바탕에 흰 글자. 복사하면 원래 글자 그대로. 규칙을 끄거나 지우면 곧바로 원래 색 (D113) |
+| G8b | 설정 구역 목록 | | 11개: 화면·입력·단축키·**출력 규칙**·파일 전송·연결·로그·git 동기화·보안·백업·정보. 키워드 강조는 "출력 규칙"에 있다 (D140) |
+| G8c | 출력 트리거 규칙 `error`(알림) | 숨은 탭의 세션에 `ERROR: disk full` 출력 | 토스트 `{세션 제목}: ERROR: disk full`. 같은 줄로 다시 울리지 않는다 (D140) |
+| G8d | 출력 트리거 규칙 `[y/n]`(자동 응답 `y` + Enter) | 일반 서버 세션에 `Proceed? [y/N] ` 출력 후 멈춤 | 0.3초 뒤 그 세션에 `y`+Enter 입력. 운영 서버 세션이면 보내지 않고 "운영 서버라 자동 응답을 보내지 않았습니다" (D140) |
 
 ## 가져오기 (import.md, ui/dialogs.md §5)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/behavior/terminal.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/behavior/terminal.md
+++ b/spec/behavior/terminal.md
@@ -95,7 +106,7 @@
 
 ## 9. 키워드 강조 (2026-11-23, D113)
 
-- 설정 `highlightRules`(설정 > 화면 > 키워드 강조)의 규칙마다, 터미널 출력에서 그 글자가 나오는 자리를 색으로 표시한다. 규칙: 찾을 글자(`pattern`), 색(`color`), 정규식 여부(`regex`), 대소문자 구분(`caseSensitive`, 기본 구분 안 함), 바탕 칠하기(`background`), 사용 여부(`enabled`).
+- 설정 `highlightRules`(설정 > 출력 규칙 > 키워드 강조 — 2026-11-27 D140 에 "화면" 구역에서 옮김)의 규칙마다, 터미널 출력에서 그 글자가 나오는 자리를 색으로 표시한다. 규칙: 찾을 글자(`pattern`), 색(`color`), 정규식 여부(`regex`), 대소문자 구분(`caseSensitive`, 기본 구분 안 함), 바탕 칠하기(`background`), 사용 여부(`enabled`).
 - **표시 방식**: 기본은 일치한 글자의 **글자색**을 규칙의 색으로 바꾼다. 바탕 칠하기를 켠 규칙은 바탕을 규칙의 색으로 칠하고 글자는 흰색으로 한다.
 - **우선순위**: 목록에서 위에 있는 규칙이 먼저다. 한 줄에서 두 규칙의 일치가 겹치면 위 규칙의 것만 표시한다.
 - **화면 표시만 바꾼다**: 받은 출력 내용은 바꾸지 않는다. 복사한 글자, 찾기, 세션 로그에는 영향이 없다. 찾기의 강조와 마우스 선택은 키워드 강조 위에 보인다.
```

### `spec/data/store.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -89,6 +90,7 @@
 | guardDangerousOnProd | true | 운영 환경 위험 명령 확인 |
 | dangerousPatterns | 아래 목록 | 정규식 문자열 배열 |
 | highlightRules | 빈 배열 | 키워드 강조 규칙 배열(2026-11-23, D113). 항목: id, pattern(찾을 글자 또는 정규식), color(`#RRGGBB`), regex, caseSensitive, background(바탕 칠하기), enabled. 순서가 우선순위 ([../behavior/terminal.md](../behavior/terminal.md) §9) |
+| triggerRules | 빈 배열 | 출력 트리거 규칙 배열(2026-11-27, D140). 항목: id, pattern(찾을 글자 또는 정규식), regex, caseSensitive, notify(알림), sound(소리), replyOn(자동 응답 사용), reply(응답으로 입력할 글자), replyEnter(Enter 붙이기), enabled. 최대 30개 ([../behavior/terminal.md](../behavior/terminal.md) §10) |
 | autoReconnect | true | |
 | autoLockMinutes | 15 | 0 이면 자동 잠금 없음 |
 | uploadDir | `~/.dariterm/uploads`  | 붙여넣기 이미지·파일 업로드 폴더 |
```

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -2,24 +2,27 @@
 
 > 작성 방식: 기존 앱 실행 관찰 + 열람. 작성일 2026-11-10. 상태: **draft**. 기본값·키 이름은 data/store.md §4.
 
-크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 10개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
+크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 11개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
 
 | 순서 | 구역 | 아이콘 |
 |---|---|---|
 | 1 | 모양 | 팔레트 |
 | 2 | 입력·붙여넣기 | 키보드 |
 | 3 | 단축키 | 명령 키(⌘ 모양) |
-| 4 | 이미지·파일 전송 | 이미지 업로드 |
-| 5 | 연결 | 플러그 |
-| 6 | 로그 | 두루마리 문서 |
-| 7 | 동기화 (git) | 새로고침 |
-| 8 | 보안·계정 | 방패 |
-| 9 | 백업 | 보관함 |
-| 10 | 정보 | 정보 |
+| 4 | 출력 규칙 | 번개 |
+| 5 | 이미지·파일 전송 | 이미지 업로드 |
+| 6 | 연결 | 플러그 |
+| 7 | 로그 | 두루마리 문서 |
+| 8 | 동기화 (git) | 새로고침 |
+| 9 | 보안·계정 | 방패 |
+| 10 | 백업 | 보관함 |
+| 11 | 정보 | 정보 |
 
 > 2026-11-23 (D120): "연결"에 있던 세션 로그 항목을 **"로그" 구역**으로 떼어 냈다. 아래 절 번호는 다른 문서·코드가 가리키고 있어 바꾸지 않았다 — 로그 구역은 **§4a** 다(화면의 순서는 위 표).
 >
 > 2026-11-25 (D126): "정보"에 있던 단축키 표를 **"단축키" 구역**으로 떼어 내고, 전체 켜기/끄기와 키 바꾸기를 넣었다. 단축키 구역은 **§2a** 다.
+
+> 2026-11-27 (D140): **"출력 규칙" 구역**을 새로 두고, "모양"에 있던 키워드 강조를 그리로 옮기고 출력 트리거를 넣었다. 출력 규칙 구역은 **§2b** 다.
 
 ## 1. 모양
 - **언어**(맨 위, 목록 선택, D75): 도움말 "화면과 안내 문구의 언어입니다. 바꾸면 바로 적용됩니다." 목록은 behavior/i18n.md §1 의 10개(그 언어로 쓴 이름). 지금 화면 언어가 골라져 있다.
@@ -33,6 +36,9 @@
   - **스크롤백 (줄)**(숫자, 500~200000으로 제한, 비면 10000)
   - **최대 화면 분할 (상하)**(세그먼트 1·2·3·4·5, 기본 3, 도움말 "한 탭에서 위아래로 나눌 수 있는 최대 창 수입니다. 도달하면 [아래로 나누기] 버튼이 비활성화됩니다.") — 2026-11-12 추가, ui/panes.md §2
 - 카드: 토글 **커서 깜빡임**, 토글 **벨 소리**.
+
+(아래 "키워드 강조" 항목은 2026-11-27 D140 에 **"출력 규칙" 구역(§2b)으로 옮겼다**. 항목의 모양은 그대로다.)
+
 - **키워드 강조**(2026-11-23, D113, 동작은 behavior/terminal.md §9): 도움말 "터미널 출력에서 이 글자가 나오면 색으로 표시합니다. 위에 있는 규칙이 우선입니다. 화면 표시만 바꾸며 복사한 글자와 로그에는 영향이 없습니다. vim 같은 전체 화면 프로그램 안에서는 표시하지 않습니다." 규칙마다 한 줄: 사용 스위치 / "찾을 글자" 입력(고정폭, 틀린 정규식이면 빨간 테두리) / 색 6개(빨강 `#ef4444`·주황 `#f97316`·노랑 `#eab308`·초록 `#22c55e`·파랑 `#3b82f6`·보라 `#a855f7`, 18px 원, 고른 색에 테두리) / 체크 "바탕 칠하기" / 체크 "정규식" / 체크 "대소문자 구분" / [삭제](휴지통). 아래에 [+ 규칙 추가](30개면 비활성): 새 규칙은 빈 글자, 빨강, 체크 모두 꺼짐, 사용 켜짐. 스위치·색·체크·삭제·추가는 누르는 즉시 저장하고, 찾을 글자는 입력 칸을 벗어날 때 저장한다.
 
 ## 2. 입력·붙여넣기
@@ -58,6 +64,18 @@
   - 다른 명령이 쓰던 조합을 고르면 그대로 저장하고, 먼저 쓰던 명령은 "없음"이 되며 정보 토스트 '{키} 를 쓰던 "{이름}"의 단축키를 비웠습니다'.
   - "단축키 사용"이 꺼져 있으면 목록의 이름·키가 흐리게 보인다(바꾸는 것은 된다).
 - 소제목 **터미널 안에서 쓰는 키** + 도움말 '바꿀 수 없고, "단축키 사용"을 꺼도 동작합니다. Ctrl+V 붙여넣기는 설정 > 입력에서 끌 수 있습니다' + 표 3줄: 붙여넣기 (이미지·파일 자동 업로드) Ctrl+V · 클립보드 이미지 → 서버 Ctrl+Alt+V · 복사 Ctrl+Shift+C / 선택.
+
+## 2b. 출력 규칙
+(2026-11-27, D140. 터미널 출력에 반응하는 규칙을 한곳에 둔다.)
+
+위에서 아래로 **키워드 강조**(§1 끝에 적힌 항목 그대로)와 **출력 트리거** 두 항목.
+
+- **출력 트리거**(동작은 behavior/terminal.md §10): 도움말 "터미널 출력의 한 줄에 이 글자가 나오면 알림·소리·자동 응답을 실행합니다. 열려 있는 모든 연결에 적용되고, 보이지 않는 탭의 출력도 살핍니다. 한 줄에서 규칙마다 한 번, 같은 규칙은 1초에 한 번만 실행합니다. vim 같은 전체 화면 프로그램 안에서와 직접 입력하는 줄에서는 실행하지 않습니다. 자동 응답은 그 글자가 나온 연결에 곧바로 입력되므로 조심해서 쓰세요. 운영 서버에는 보내지 않고(입력 > 운영 서버 위험 명령 확인이 켜져 있을 때), 10초에 5번 넘게 반복되면 그 연결에서는 꺼집니다."
+  - 규칙마다 테두리 상자 하나, 두 줄.
+    - 첫 줄: 사용 스위치 / "찾을 글자" 입력(고정폭, 틀린 정규식이면 빨간 테두리) / 체크 "정규식" / 체크 "대소문자 구분" / [삭제](휴지통).
+    - 둘째 줄(스위치 폭만큼 들여 씀): 체크 "알림" / 체크 "소리" / 체크 "자동 응답". 자동 응답을 켜면 그 오른쪽에 "응답으로 입력할 글자" 입력(고정폭)과 체크 "Enter 붙이기"가 나타난다.
+  - [+ 규칙 추가](30개면 비활성): 새 규칙은 빈 글자, 알림만 켜짐, 자동 응답 꺼짐(Enter 붙이기는 켜진 채), 사용 켜짐.
+  - 스위치·체크·삭제·추가는 누르는 즉시 저장하고, 글자 칸은 벗어날 때 저장한다.
 
 ## 3. 이미지·파일 전송
 - 안내 박스(이미지 아이콘): "SSH 터미널에서 **Ctrl+V**로 이미지(캡처)를 붙여넣거나 파일을 터미널로 끌어다 놓으면, 서버의 업로드 폴더로 전송한 뒤 그 **경로를 터미널에 입력**합니다. 서버에서 실행 중인 **Claude Code** 등 CLI는 이 경로로 이미지를 읽을 수 있습니다. 예: `이 화면의 오류 원인 알려줘` 입력 후 Ctrl+V."
```

## G. 검증 (2026-11-27 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [outputLines.test.ts](../../frontend/src/lib/outputLines.test.ts) 7개 | 줄 나누기와 멈춘 줄 / 색·커서·창 제목 시퀀스 지우기 / **시퀀스가 조각 경계에서 잘려도** 이어서 지우기 / CR 로 다시 쓴 줄(진행 막대) / 백스페이스·탭 / 전체 화면 프로그램 안의 글자 버리기 / 줄 길이 상한 |
| [trigger.test.ts](../../frontend/src/lib/trigger.test.ts) `compileTriggers` 4개 | 꺼진 규칙·빈 글자·틀린 정규식·할 일 없는 규칙 빼기 / 자동 응답은 켰을 때만, Enter 붙이기, Enter 만 보내기 / 보통 글자는 정규식 기호를 풀지 않음(`[y/N]`) / 30개 한도 |
| 같은 파일 `TriggerWatcher` 7개 | 끝난 줄에서 실행 / **멈춘 줄은 `idle` 에서 실행하고 그 줄이 끝나도 다시 실행하지 않음** / 1초 쿨다운 / 사용자가 치는 줄의 에코는 건너뛰고 줄이 바뀌면 다시 살핌 / 터미널의 자동 응답은 입력으로 세지 않음 / **자동 응답 폭주 차단**(한도까지 보내고 넘는 순간 한 번 알린 뒤 조용) / 전체 화면 프로그램 안은 살피지 않음 |
| [automation.test.ts](../../frontend/src/state/automation.test.ts) "출력 트리거 실행" 6개 | 알림 토스트 문구 / 프롬프트에는 0.3초 뒤 그 세션에 입력 / **운영 서버에는 보내지 않고 알림** / 운영 서버 확인 설정을 끄면 보냄 / 입력 중인 줄의 에코에는 반응하지 않음 / 규칙이 없으면 아무 일도 하지 않음 |
| [Automation.test.tsx](../../frontend/src/screens/Automation.test.tsx) "설정 · 출력 규칙" 2개 | 키워드 강조와 출력 트리거가 한 구역에 있고 규칙을 추가하면 저장된다 / 자동 응답을 켜야 응답 글자 칸이 보인다 |
| `SettingsLog.test.tsx`·`SettingsShortcuts.test.tsx`(기대값 수정) | 구역 목록이 11개이고 "단축키" 다음이 "출력 규칙" |
| 계약 테스트 | `store.get` 의 `triggerRules: []` |
| 기존 `highlight.test.ts`·`termHighlight.test.ts`·`TerminalView.test.tsx` | 키워드 강조의 동작은 그대로(화면에서의 위치만 옮김), 벨 소리 함수를 옮긴 뒤에도 터미널이 그대로 |

**자동 테스트가 확인하지 못하는 것**: 실제 소리, 실제 서버 출력에서의 줄 나누기(특히 화면을 다시 그리는 CLI), 토스트가 5개를 넘을 때의 모양, 출력이 초당 수 MB 일 때의 부하.

전체 실행 결과(2026-11-27, 다섯 문서와 추가 지시 D142·D143 을 모두 적용한 상태):

```bash
go vet ./...                              # 경고 없음
go test ./...                             # 19개 패키지 모두 ok
(cd frontend && npx tsc --noEmit)         # 오류 없음
(cd frontend && npm test)                 # PowerShell 에서. 61개 파일 439개 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

**`-race` 검사 (같은 날 추가)**: 처음에는 이 PC 에 gcc 가 없어 돌리지 못했다(`-race requires cgo`). 같은 날 gcc 를 설치하고(D144, [2026-11-27-6-gcc-race-test.md](./2026-11-27-6-gcc-race-test.md)) `go test -race ./...` 를 돌려 19개 패키지 모두 통과했다 — 경쟁 보고 없음.

```
(frontend) npm test
go test ./internal/app ./internal/store      # 설정 기본값·계약 고정 파일
```

## H. 수동 검증 (앱에서)

| # | 할 일 | 기대 |
|---|---|---|
| 1 | 설정을 열어 구역 목록 확인 | "단축키" 다음에 "출력 규칙". 그 안에 키워드 강조(전에 넣은 규칙이 그대로)와 출력 트리거 |
| 2 | 규칙 `hello`(알림) 추가 → 터미널에서 `echo hello` | 명령을 치는 동안에는 조용하고, 출력 줄 `hello` 에서 토스트 `{세션}: hello` 한 번 |
| 3 | 2번을 다른 탭에서 실행해 두고 탭을 옮김(`sleep 5; echo hello`) | 안 보이는 탭이어도 토스트가 뜬다 |
| 4 | 규칙에 소리를 켬 | 짧은 소리. 설정 > 화면의 "벨 소리"가 꺼져 있어도 난다 |
| 5 | 규칙 `[y/N]`(자동 응답 `y`, Enter 붙이기) → `read -p "go? [y/N] " a; echo $a` | 잠깐 뒤 `y` 가 입력되고 `y` 가 출력된다 |
| 6 | 5번을 환경이 "운영"인 서버에서 | 입력되지 않고 "운영 서버라 자동 응답을 보내지 않았습니다" |
| 7 | 자동 응답이 다시 같은 질문을 부르게 함(`while true; do read -p "[y/N] " a; done`) | 5번쯤 응답한 뒤 "너무 자주 반복돼 이 연결에서는 껐습니다", 그 뒤로 조용 |
| 8 | `vim` 안에 `hello` 를 적음 | 울리지 않는다 |
| 9 | 틀린 정규식(`(`)을 넣고 정규식을 체크 | 칸에 빨간 테두리, 그 규칙은 동작하지 않는다 |
| 10 | 규칙을 끄거나 지움 | 곧바로 멈춘다(다시 연결하지 않아도) |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 규칙이 없는 사용자(기본) | 출력 묶음마다 `emitOutput` 이 한 번 더 불린다 | 규칙이 없고 그 세션의 감시가 없으면 `feedTriggers` 가 곧바로 돌아온다(줄 나누기도 하지 않는다). `automation.test.ts` 마지막 |
| 출력이 많은 세션 + 규칙 있음 | 묶음마다 글자를 한 번 훑어 줄로 나누고 줄마다 규칙을 대 본다 | 줄 길이 2000자·규칙 30개 한도. 화면 쓰기(xterm)와 따로 돌아 흐름 제어(ack)는 막지 않는다. 부하는 재 보지 않았다 |
| 자동 응답이 잘못된 곳에 입력됨 | 글자가 나온 세션에 확인 없이 입력한다 | 기본은 꺼짐(새 규칙은 알림만). 운영 서버에는 보내지 않는다. 폭주하면 그 세션에서 끈다. 도움말에 "조심해서 쓰세요" |
| 자동 응답의 에코가 다시 규칙에 맞음 | 끝없이 돌 수 있다 | 보낸 입력의 줄은 살피지 않는다 + 1초 쿨다운 + 10초 5번 한도 |
| 사용자가 치는 글자에 반응 | `grep error` 를 치는 동안 울릴 수 있다 | 입력을 보낸 줄은 그 줄이 끝날 때까지 살피지 않는다. 다만 입력 없이 화살표(↑)로 불러온 지난 명령의 에코에는 반응할 수 있다(ESC 로 시작하는 입력은 세지 않으므로) |
| 설정 화면에서 키워드 강조가 사라진 것처럼 보임 | "화면" 구역에서 없어졌다 | "출력 규칙"으로 옮겼다. 저장해 둔 규칙은 그대로. 변경 이력·설명서에 적었다 |
| 설정 구역 수 | 10 → 11개. 내비게이션이 길어진다 | 큰 창(820px)의 왼쪽 목록에 들어간다. 기대값을 고정해 둔 테스트 2개를 고쳤다 |
| 저장 데이터 | 설정에 `triggerRules` 가 생긴다. 옛 버전 앱은 무시한다 | git 동기화는 설정을 싣지 않으므로 PC 마다 따로다(키워드 강조와 같다) |
| 벨 소리 | `beep` 을 `TerminalView.tsx` 에서 `lib/beep.ts` 로 옮겼다 | 내용은 같다. `TerminalView.test.tsx` 통과 |
| 알림 | 앱 안의 토스트뿐이다(앱이 뒤에 있으면 보이지 않는다) | 소리는 난다. Windows 알림은 후속 과제 |

## J. 체크리스트

- [x] `lib/outputLines.ts`(출력 → 줄), `lib/trigger.ts`(규칙·감시), `lib/outputHub.ts`(출력·입력 창구), `lib/beep.ts`
- [x] `state/triggers.ts`: 세션별 감시, 0.3초 타이머, 알림·소리·자동 응답, 운영 서버 차단, 폭주 알림
- [x] 출력·입력 연결: `state/actions.ts`(session.data), `TerminalView.tsx`(친 입력), `lib/termSend.ts`(명령 모음이 보낸 입력), `App.tsx`(구독)
- [x] 설정: 새 구역 "출력 규칙"(`SettingsRules.tsx`), 키워드 강조를 "화면"에서 옮김, 구역 11개
- [x] 타입(`TriggerRule`)·기본값·백엔드 기본값(`store.go`)·계약 파일 2개
- [x] 문구 11개(10개 언어)
- [x] 명세 behavior/terminal.md §9(위치)·§10·§10.1 · data/store.md §4 · ui/settings.md 표·§2b · 수용 G8b~G8d, decisions D140
- [x] 자동 테스트 26개 추가, 기대값 2개 수정, 전체 통과
- [ ] 수동 검증 1~10 (실제 화면)
