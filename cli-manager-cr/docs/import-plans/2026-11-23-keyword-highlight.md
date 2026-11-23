# 터미널 키워드 강조 (`highlightRules`) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D113). **실제 화면에서의 모습은 확인하지 않았다**(수동 검증으로 남음) · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 8번. 앞 문서 [log-line-timestamp](./2026-11-23-log-line-timestamp.md), 뒤 문서 [zmodem](./2026-11-23-zmodem.md). 같은 방식(강조 얹기)을 쓰는 앞선 문서: [2026-11-18-pane-find-button.md](./2026-11-18-pane-find-button.md)(찾기 강조 D96).

---

## 0. 한 줄 요약

설정 > 화면에 **"키워드 강조"** 규칙 목록을 더했다. 터미널 출력에 `ERROR`, `FAIL`, `refused` 같은 글자가 나오면 고른 색으로 표시한다. 정규식·대소문자 구분·바탕 칠하기를 규칙마다 고를 수 있다. 화면 표시만 바꾸므로 복사한 글자·찾기·로그에는 영향이 없다.

## 1. 배경

- 터미널 출력은 서버가 준 색 그대로만 보였다. 색이 없는 로그(`tail -f app.log`, 장비 콘솔)에서 오류 줄을 눈으로 찾아야 했다.
- SecureCRT·Xshell·MobaXterm 에 모두 있는 기능이다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 어디서 하나 | **프런트**, xterm.js 의 강조(decoration)를 일치 구간 위에 얹는다 | 받은 출력에 색 코드를 끼워 넣으면 커서 위치·복사한 글자·로그가 다 바뀐다. 강조는 그리기만 덧입힌다(찾기 강조 D96 과 같은 방식) |
| 살피는 범위 | **지금 화면에 보이는 줄만** | 스크롤백 10,000줄 전체에 강조를 걸면 느려진다(찾기 강조가 1,000개에서 멈추는 이유, KI-011). 화면 줄 수는 많아야 수십 줄 |
| 다시 그리는 때 | 출력(`onWriteParsed`)·스크롤·크기 변경·규칙 변경. 0.08초에 한 번으로 모은다 | 출력이 쏟아질 때 묶음마다 다시 그리지 않게 |
| 규칙이 없을 때 | 타이머도 걸지 않는다 | 기능을 안 쓰는 사람(기본값)에게 비용이 0 |
| 표시 | 기본은 글자색. "바탕 칠하기"를 켜면 바탕을 그 색으로, 글자는 흰색 | 글자색은 덜 거슬리고, 바탕은 눈에 확 띈다. 둘 다 쓰는 사람이 있다 |
| 색 | 6가지 고정(빨강·주황·노랑·초록·파랑·보라) | 어두운·밝은 터미널 테마 모두에서 읽히는 중간 밝기로 골랐다. 자유 색 입력은 화면이 복잡해진다 |
| 우선순위 | 목록의 위가 먼저. 겹치면 위 규칙만 | 단순하고 예측 가능하다 |
| 줄 단위 | 화면의 한 줄씩 찾는다. 폭 때문에 다음 줄로 넘어간 낱말은 못 찾는다 | 줄을 이어 붙여 찾으면 위치 계산이 복잡해진다. 키워드는 대개 짧다 |
| 전체 화면 프로그램 | vim·htop 등(대체 화면)에서는 표시하지 않는다 | xterm 이 대체 화면에서는 표식(marker)을 만들어 주지 않는다. 그런 화면은 프로그램이 직접 색을 쓴다 |
| 전각 글자 | 글 위치 → 칸 위치 표(`colOf`)로 맞춘다 | 한글은 두 칸이다. 글자 수로 세면 한글 뒤의 강조가 밀린다 |
| 한도 | 규칙 30개, 한 줄 50곳, 한 화면 400곳 | 실수로 `.` 같은 정규식을 넣어도 화면이 멈추지 않게 |
| 저장 | 설정 `highlightRules`(배열, 기본 빈 배열) | git 동기화·백업에 함께 실린다 |
| 계산과 그리기 분리 | `lib/highlight.ts`(순수 계산) / `lib/termHighlight.ts`(xterm 에 그리기) | 계산은 화면 없이 단위 테스트한다 |

### 2.1 알려진 한계

- 정규식이 아주 느린 모양(예: `(a+)+$`)이면 화면이 버벅일 수 있다. 줄 길이가 화면 폭으로 제한되고 보이는 줄만 보므로 영향은 작지만, 막는 장치는 없다.
- 찾기 막대를 연 동안의 KI-011(출력이 이어지면 찾기 강조가 늦게 갱신됨)과는 따로 동작한다. 키워드 강조는 0.08초마다 갱신된다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/lib/defaults.ts` | 코드 | 수정 |
| `frontend/src/lib/highlight.ts` | 코드 | 새 파일 |
| `frontend/src/lib/termHighlight.ts` | 코드 | 새 파일 |
| `frontend/src/screens/SettingsAppearance.tsx` | 코드 | 수정 |
| `frontend/src/screens/SettingsHighlight.tsx` | 코드 | 새 파일 |
| `frontend/src/screens/TerminalView.tsx` | 코드 | 수정 |
| `internal/store/store.go` | 코드 | 수정 |
| `frontend/src/lib/highlight.test.ts` | 테스트 | 새 파일 |
| `frontend/src/lib/termHighlight.test.ts` | 테스트 | 새 파일 |
| `frontend/src/screens/TerminalView.test.tsx` | 테스트 | 수정 |
| `frontend/src/ipc/contract/store.get.afterNullPatch.json` | 계약 고정 파일 | 수정 |
| `frontend/src/ipc/contract/store.get.json` | 계약 고정 파일 | 수정 |
| `spec/acceptance/04-hosts-sftp-settings-sync.md` | 명세 | 수정 |
| `spec/behavior/terminal.md` | 명세 | 수정 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/ui/settings.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/ipc/contract/store.get.afterNullPatch.json`, `frontend/src/ipc/contract/store.get.json`, `frontend/src/ipc/types.ts`, `frontend/src/lib/defaults.ts`, `internal/store/store.go`, `spec/acceptance/04-hosts-sftp-settings-sync.md`, `spec/data/store.md`, `spec/ui/settings.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -66,6 +66,20 @@
   sendEnter: boolean;
 }
 
+/** 키워드 강조 규칙 하나(behavior/terminal.md §9). */
+export interface HighlightRule {
+  id: string;
+  /** 찾을 글자(regex 가 true 면 정규식). */
+  pattern: string;
+  /** 강조 색 `#RRGGBB`. */
+  color: string;
+  regex: boolean;
+  caseSensitive: boolean;
+  /** true 면 글자색 대신 바탕을 칠한다(글자는 흰색). */
+  background: boolean;
+  enabled: boolean;
+}
+
 export interface KnownHost {
   hostPort: string;
   keyType: string;
@@ -90,6 +104,8 @@
   confirmMultilinePaste: boolean;
   guardDangerousOnProd: boolean;
   dangerousPatterns: string[];
+  /** 터미널 출력의 키워드 강조 규칙. 앞에 있는 규칙이 우선이다 (behavior/terminal.md §9). */
+  highlightRules: HighlightRule[];
   autoReconnect: boolean;
   autoLockMinutes: number;
   uploadDir: string;
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -32,6 +32,7 @@
   confirmMultilinePaste: true,
   guardDangerousOnProd: true,
   dangerousPatterns: [...DANGEROUS_PATTERNS],
+  highlightRules: [],
   autoReconnect: true,
   autoLockMinutes: 15,
   uploadDir: '~/.dariterm/uploads',
```

### `frontend/src/lib/highlight.ts` — 새 파일 (전체)

```ts
// 키워드 강조의 순수 계산: 규칙 → 정규식, 한 줄에서 일치 구간 찾기, 글자 위치 → 터미널 칸 위치.
// 근거: spec/behavior/terminal.md §9 (D113). 화면(xterm)에 그리는 부분은 termHighlight.ts.
import type { HighlightRule } from '../ipc/types';

/** 고를 수 있는 강조 색(이름은 문구 키 hl.color.<id>). 어두운·밝은 터미널 테마 모두에서 읽히는 중간 밝기다. */
export const HIGHLIGHT_COLORS: { id: string; hex: string }[] = [
  { id: 'red', hex: '#ef4444' },
  { id: 'orange', hex: '#f97316' },
  { id: 'yellow', hex: '#eab308' },
  { id: 'green', hex: '#22c55e' },
  { id: 'blue', hex: '#3b82f6' },
  { id: 'purple', hex: '#a855f7' },
];
export const HIGHLIGHT_DEFAULT_COLOR = HIGHLIGHT_COLORS[0].hex;

/** 규칙 수와 한 줄의 일치 수 한도. 넘는 것은 무시한다(출력이 많은 화면에서 그리기 비용을 막는다). */
export const HIGHLIGHT_MAX_RULES = 30;
export const HIGHLIGHT_MAX_PER_LINE = 50;

const HEX6 = /^#[0-9a-fA-F]{6}$/;

export interface CompiledRule {
  re: RegExp;
  color: string;
  /** 글자색 대신 바탕색으로 칠한다. */
  background: boolean;
}

export interface Match {
  /** 줄 글의 시작 위치(포함)와 끝 위치(제외), UTF-16 단위. */
  start: number;
  end: number;
  color: string;
  background: boolean;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 규칙의 패턴이 쓸 수 있는지(빈 값·틀린 정규식이 아닌지). 설정 화면이 입력 칸에 오류를 표시할 때 쓴다. */
export function isValidPattern(pattern: string, regex: boolean): boolean {
  if (!pattern) return false;
  if (!regex) return true;
  try {
    new RegExp(pattern);
    return true;
  } catch {
    return false;
  }
}

/** 켜져 있고 쓸 수 있는 규칙만 정규식으로 만든다. 앞에 있는 규칙이 우선이다. */
export function compileRules(rules: readonly HighlightRule[] | undefined): CompiledRule[] {
  const out: CompiledRule[] = [];
  for (const r of rules ?? []) {
    if (out.length >= HIGHLIGHT_MAX_RULES) break;
    if (!r || !r.enabled || typeof r.pattern !== 'string' || !isValidPattern(r.pattern, !!r.regex)) continue;
    try {
      out.push({
        re: new RegExp(r.regex ? r.pattern : escapeRegExp(r.pattern), r.caseSensitive ? 'g' : 'gi'),
        color: HEX6.test(r.color) ? r.color : HIGHLIGHT_DEFAULT_COLOR,
        background: !!r.background,
      });
    } catch {
      // 만들 수 없는 정규식은 건너뛴다
    }
  }
  return out;
}

/**
 * 한 줄의 글에서 일치 구간을 찾는다. 겹치면 앞 규칙이 이기고, 뒤 규칙의 겹치는 일치는 버린다.
 * 결과는 시작 위치 순이다. 빈 일치는 건너뛴다.
 */
export function findMatches(text: string, rules: readonly CompiledRule[]): Match[] {
  const out: Match[] = [];
  if (!text) return out;
  for (const rule of rules) {
    rule.re.lastIndex = 0;
    let guard = 0;
    for (let m = rule.re.exec(text); m && guard < HIGHLIGHT_MAX_PER_LINE; m = rule.re.exec(text)) {
      guard++;
      if (m[0].length === 0) {
        rule.re.lastIndex++; // 빈 일치(a* 등): 제자리에서 맴돌지 않게 한 칸 나아간다
        if (rule.re.lastIndex > text.length) break;
        continue;
      }
      const start = m.index;
      const end = start + m[0].length;
      if (!out.some((o) => start < o.end && o.start < end)) out.push({ start, end, color: rule.color, background: rule.background });
    }
  }
  return out.sort((a, b) => a.start - b.start).slice(0, HIGHLIGHT_MAX_PER_LINE);
}

/** xterm 버퍼 줄에서 쓰는 부분만 뽑은 모양(테스트에서 가짜로 만들 수 있게). */
export interface CellLine {
  length: number;
  getCell(x: number): { getChars(): string; getWidth(): number } | undefined;
}

/**
 * 버퍼 줄을 글로 바꾸고, 글의 각 위치가 몇 번째 칸인지 적은 표를 만든다.
 * 한글 같은 전각 글자는 두 칸을 차지하고 둘째 칸은 폭 0 으로 들어 있으므로 건너뛴다.
 * colOf[i] = text[i] 가 시작하는 칸, colOf[text.length] = 글이 끝난 다음 칸.
 */
export function lineText(line: CellLine, cols: number): { text: string; colOf: number[] } {
  let text = '';
  const colOf: number[] = [];
  let x = 0;
  const n = Math.min(cols, line.length);
  while (x < n) {
    const cell = line.getCell(x);
    const width = cell?.getWidth() ?? 1;
    if (width === 0) {
      x++;
      continue;
    }
    const chars = cell?.getChars() || ' ';
    for (let i = 0; i < chars.length; i++) colOf.push(x);
    text += chars;
    x += width;
  }
  colOf.push(x);
  return { text, colOf };
}

/** 일치 구간(글 위치)을 터미널 칸 범위로 바꾼다. */
export function matchCells(m: Match, colOf: readonly number[]): { x: number; width: number } {
  const x = colOf[m.start] ?? 0;
  const end = colOf[m.end] ?? colOf[colOf.length - 1] ?? x;
  return { x, width: Math.max(1, end - x) };
}

/** 새 규칙의 기본값. */
export function newHighlightRule(): HighlightRule {
  return { id: crypto.randomUUID(), pattern: '', color: HIGHLIGHT_DEFAULT_COLOR, regex: false, caseSensitive: false, background: false, enabled: true };
}
```

### `frontend/src/lib/termHighlight.ts` — 새 파일 (전체)

```ts
// 키워드 강조를 터미널(xterm)에 그린다. 근거: spec/behavior/terminal.md §9 (D113).
// 지금 화면에 보이는 줄만 살펴 일치 구간에 강조(decoration)를 얹고, 출력·스크롤·크기 변경·규칙 변경 때 다시 한다.
// 서버로 가는 입력과 받은 출력 내용은 바꾸지 않는다(복사하면 원래 글자 그대로다).
import type { IDisposable, Terminal } from '@xterm/xterm';
import type { HighlightRule } from '../ipc/types';
import { compileRules, findMatches, lineText, matchCells, type CompiledRule } from './highlight';

/** 다시 그리기를 모으는 시간(ms). 출력이 이어지는 동안에는 이 간격으로 한 번씩만 그린다. */
export const HIGHLIGHT_REFRESH_MS = 80;
/** 한 화면에 얹는 강조 개수 한도. */
export const HIGHLIGHT_MAX_ON_SCREEN = 400;

export interface Highlighter {
  /** 규칙을 바꾼다(같은 배열이면 아무 일도 하지 않는다). */
  setRules(rules: readonly HighlightRule[] | undefined): void;
  dispose(): void;
}

/** 터미널에 키워드 강조를 붙인다. */
export function attachHighlighter(term: Terminal, initial: readonly HighlightRule[] | undefined): Highlighter {
  let source = initial;
  let rules: CompiledRule[] = compileRules(initial);
  let drawn: IDisposable[] = [];
  let timer: ReturnType<typeof setTimeout> | null = null;
  let disposed = false;

  const clear = () => {
    for (const d of drawn) d.dispose();
    drawn = [];
  };

  const draw = () => {
    timer = null;
    if (disposed) return;
    clear();
    if (rules.length === 0) return;
    const buf = term.buffer.active;
    let count = 0;
    for (let row = 0; row < term.rows && count < HIGHLIGHT_MAX_ON_SCREEN; row++) {
      const y = buf.viewportY + row;
      const line = buf.getLine(y);
      if (!line) continue;
      const { text, colOf } = lineText(line, term.cols);
      if (!text.trim()) continue;
      for (const m of findMatches(text, rules)) {
        if (count >= HIGHLIGHT_MAX_ON_SCREEN) break;
        // 표식은 일반 화면 버퍼에서만 만들 수 있다(vim·htop 같은 전체 화면 프로그램에서는 undefined → 강조하지 않는다).
        const marker = term.registerMarker(y - buf.baseY - buf.cursorY);
        if (!marker) return;
        const cells = matchCells(m, colOf);
        const deco = term.registerDecoration({
          marker,
          x: cells.x,
          width: cells.width,
          ...(m.background ? { backgroundColor: m.color, foregroundColor: '#ffffff' } : { foregroundColor: m.color }),
        });
        count++;
        // 강조를 먼저, 표식을 나중에 지운다(TerminalView 의 찾기 강조와 같은 이유).
        drawn.push({
          dispose() {
            deco?.dispose();
            marker.dispose();
          },
        });
      }
    }
  };

  const schedule = () => {
    if (disposed || timer !== null) return;
    if (rules.length === 0 && drawn.length === 0) return; // 규칙이 없으면 출력마다 타이머를 걸지 않는다
    timer = setTimeout(draw, HIGHLIGHT_REFRESH_MS);
  };

  const subs: IDisposable[] = [
    term.onWriteParsed(schedule),
    term.onScroll(schedule),
    term.onResize(schedule),
  ];
  schedule();

  return {
    setRules(next) {
      if (next === source) return;
      source = next;
      rules = compileRules(next);
      if (rules.length === 0) {
        if (timer !== null) {
          clearTimeout(timer);
          timer = null;
        }
        clear();
        return;
      }
      schedule();
    },
    dispose() {
      disposed = true;
      if (timer !== null) clearTimeout(timer);
      timer = null;
      for (const s of subs) s.dispose();
      clear();
    },
  };
}
```

### `frontend/src/screens/SettingsAppearance.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsAppearance.tsx
+++ b/frontend/src/screens/SettingsAppearance.tsx
@@ -6,6 +6,7 @@
 import { useApp } from '../state/app';
 import { saveSettings } from '../state/settings';
 import { NumberField, useSetting, Group } from './Settings';
+import { SettingsHighlight } from './SettingsHighlight';
 import { LANGS, isLang, t, useLang } from '../i18n';
 
 export function SettingsAppearance() {
@@ -150,6 +151,7 @@
         <ToggleRow title={t('appearance.cursorBlink')} checked={cursorBlink} onChange={setCursorBlink} />
         <ToggleRow title={t('appearance.bell')} checked={bell} onChange={setBell} />
       </Group>
+      <SettingsHighlight />
     </>
   );
 }
```

### `frontend/src/screens/SettingsHighlight.tsx` — 새 파일 (전체)

```tsx
// 설정 1 · 화면의 "키워드 강조" 부분. 근거: spec/ui/settings.md §1 키워드 강조, spec/behavior/terminal.md §9 (D113).
import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import type { HighlightRule } from '../ipc/types';
import { Field, Switch } from '../ui/Controls';
import { HIGHLIGHT_COLORS, HIGHLIGHT_MAX_RULES, isValidPattern, newHighlightRule } from '../lib/highlight';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { t, type MsgId } from '../i18n';

/** 작은 체크 상자 + 글자. */
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="small" style={{ display: 'inline-flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap' }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

export function SettingsHighlight() {
  const saved = useApp((s) => s.store?.settings.highlightRules);
  const [rows, setRows] = useState<HighlightRule[]>(() => (Array.isArray(saved) ? saved : []));
  const [prev, setPrev] = useState(saved);
  // 밖에서 바뀌면(동기화·백업 가져오기) 화면을 맞춘다. 이 화면이 저장한 것은 같은 배열이라 다시 맞추지 않는다.
  if (prev !== saved) {
    setPrev(saved);
    setRows(Array.isArray(saved) ? saved : []);
  }

  /** 화면을 바꾸고, save 면 곧바로 저장한다(패턴은 입력 칸을 벗어날 때 저장). */
  const apply = (next: HighlightRule[], save: boolean) => {
    setRows(next);
    if (save) void saveSettings({ highlightRules: next });
  };
  const patch = (id: string, p: Partial<HighlightRule>, save = true) => apply(rows.map((r) => (r.id === id ? { ...r, ...p } : r)), save);

  return (
    <Field label={t('hl.title')} help={t('hl.help')}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {rows.map((r) => {
          const bad = r.pattern !== '' && !isValidPattern(r.pattern, r.regex);
          return (
            <div key={r.id} className="hl-row" style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Switch checked={r.enabled} onChange={(v) => patch(r.id, { enabled: v })} label={t('hl.enabled')} />
              <input
                className="input mono"
                aria-label={t('hl.pattern')}
                aria-invalid={bad}
                placeholder={t('hl.pattern')}
                style={{ flex: '1 1 160px', minWidth: 120, borderColor: bad ? 'var(--danger)' : undefined }}
                value={r.pattern}
                onChange={(e) => patch(r.id, { pattern: e.target.value }, false)}
                onBlur={() => void saveSettings({ highlightRules: rows })}
              />
              <span role="radiogroup" aria-label={t('hl.color')} style={{ display: 'inline-flex', gap: 4 }}>
                {HIGHLIGHT_COLORS.map((c) => (
                  <button
                    key={c.id}
                    type="button"
                    role="radio"
                    aria-checked={r.color === c.hex}
                    title={t(`hl.color.${c.id}` as MsgId)}
                    aria-label={t(`hl.color.${c.id}` as MsgId)}
                    onClick={() => patch(r.id, { color: c.hex })}
                    style={{
                      width: 18,
                      height: 18,
                      borderRadius: 99,
                      background: c.hex,
                      border: r.color === c.hex ? '2px solid var(--fg-1)' : '2px solid transparent',
                      padding: 0,
                      cursor: 'pointer',
                    }}
                  />
                ))}
              </span>
              <Check label={t('hl.background')} checked={r.background} onChange={(v) => patch(r.id, { background: v })} />
              <Check label={t('hl.regex')} checked={r.regex} onChange={(v) => patch(r.id, { regex: v })} />
              <Check label={t('hl.case')} checked={r.caseSensitive} onChange={(v) => patch(r.id, { caseSensitive: v })} />
              <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => apply(rows.filter((x) => x.id !== r.id), true)}>
                <Trash2 size={14} />
              </button>
            </div>
          );
        })}
        <div>
          <button type="button" className="btn small-btn" disabled={rows.length >= HIGHLIGHT_MAX_RULES} onClick={() => apply([...rows, newHighlightRule()], true)}>
            {t('hl.add')}
          </button>
        </div>
      </div>
    </Field>
  );
}
```

### `frontend/src/screens/TerminalView.tsx` — 수정

```diff
--- a/frontend/src/screens/TerminalView.tsx
+++ b/frontend/src/screens/TerminalView.tsx
@@ -15,6 +15,7 @@
 import { decodeOsc52 } from '../lib/osc52';
 import { shortcutFor } from '../lib/keys';
 import { resolveTermTheme } from '../lib/termThemes';
+import { attachHighlighter, type Highlighter } from '../lib/termHighlight';
 import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_ACTIVE_FG, FIND_MATCH_COLOR, FIND_MATCH_RATIO, FIND_RULER_MATCH, FIND_RULER_ACTIVE, FIND_RULER_WIDTH } from '../lib/findHighlight';
 import { DANGEROUS_PATTERNS } from '../lib/defaults';
 import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';
@@ -264,6 +265,8 @@
   el: HTMLDivElement;
   /** 지금 이 터미널을 보여 주는 칸(입력 대상 계산용). 칸이 바뀌면 붙일 때 갱신한다. */
   paneId: string;
+  /** 키워드 강조(behavior/terminal.md §9). 설정의 규칙이 바뀌면 setRules 로 알린다. */
+  highlighter: Highlighter;
 }
 const keptTerms = new Map<string, KeptTerm>();
 
@@ -432,7 +435,7 @@
   };
   el.addEventListener('wheel', onWheel, { passive: false });
 
-  const k: KeptTerm = { term: t, fit, el, paneId };
+  const k: KeptTerm = { term: t, fit, el, paneId, highlighter: attachHighlighter(t, st?.highlightRules) };
   keptTerms.set(sessionId, k);
   return k;
 }
@@ -441,10 +444,17 @@
   const k = keptTerms.get(sessionId);
   if (!k) return;
   keptTerms.delete(sessionId);
+  k.highlighter.dispose();
   k.el.remove();
   k.term.dispose();
   if (useWorkspace.getState().writers[sessionId]) useWorkspace.getState().registerWriter(sessionId, null);
 }
+
+// 키워드 강조 규칙이 바뀌면 살려 둔 모든 터미널(안 보이는 탭 포함)에 알린다.
+useApp.subscribe((s) => {
+  const rules = s.store?.settings.highlightRules;
+  for (const k of keptTerms.values()) k.highlighter.setRules(rules);
+});
 
 // 세션이 닫혀 작업 공간에서 빠지면 살려 둔 터미널도 버린다.
 useWorkspace.subscribe((s) => {
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -41,6 +41,7 @@
 		`\bTRUNCATE\b`,
 		`systemctl\s+(stop|restart)`,
 	},
+	"highlightRules":      []any{}, // 키워드 강조 규칙 (behavior/terminal.md §9, D113)
 	"autoReconnect":       true,
 	"autoLockMinutes":     15,
 	"uploadDir":           "~/.dariterm/uploads",
```


## C. 테스트 코드

### `frontend/src/lib/highlight.test.ts` — 새 파일 (전체)

```ts
// 키워드 강조의 순수 계산 검사. 근거: spec/behavior/terminal.md §9, 수용 G9 (D113).
import { describe, expect, it } from 'vitest';
import type { HighlightRule } from '../ipc/types';
import { compileRules, findMatches, HIGHLIGHT_DEFAULT_COLOR, HIGHLIGHT_MAX_PER_LINE, isValidPattern, lineText, matchCells, type CellLine } from './highlight';

const rule = (p: Partial<HighlightRule>): HighlightRule => ({
  id: p.pattern ?? 'x', pattern: '', color: '#ef4444', regex: false, caseSensitive: false, background: false, enabled: true, ...p,
});

/** 가짜 버퍼 줄: 전각 글자는 폭 2 칸 + 폭 0 칸으로 넣는다(xterm 과 같은 모양). */
function fakeLine(s: string, cols = 80): CellLine {
  const cells: { c: string; w: number }[] = [];
  for (const ch of s) {
    const wide = /[ᄀ-ᇿ　-鿿가-힣＀-｠]/.test(ch);
    cells.push({ c: ch, w: wide ? 2 : 1 });
    if (wide) cells.push({ c: '', w: 0 });
  }
  while (cells.length < cols) cells.push({ c: '', w: 1 });
  return { length: cells.length, getCell: (x) => (cells[x] ? { getChars: () => cells[x].c, getWidth: () => cells[x].w } : undefined) };
}

describe('키워드 강조: 규칙 → 일치 구간', () => {
  it('보통 글자는 대소문자를 가리지 않고 모두 찾는다', () => {
    const rules = compileRules([rule({ pattern: 'error' })]);
    expect(findMatches('ERROR: disk Error at error.log', rules).map((m) => [m.start, m.end])).toEqual([[0, 5], [12, 17], [21, 26]]);
  });

  it('대소문자 구분을 켜면 같은 모양만 찾는다', () => {
    const rules = compileRules([rule({ pattern: 'FAIL', caseSensitive: true })]);
    expect(findMatches('fail FAIL Fail', rules)).toHaveLength(1);
  });

  it('보통 글자의 특수 문자는 글자 그대로 찾는다', () => {
    const rules = compileRules([rule({ pattern: '[WARN]' })]);
    expect(findMatches('x [WARN] W', rules).map((m) => [m.start, m.end])).toEqual([[2, 8]]);
  });

  it('정규식 규칙은 정규식으로 찾고, 빈 일치는 건너뛴다', () => {
    const rules = compileRules([rule({ pattern: '\\b\\d{3}\\b', regex: true }), rule({ pattern: 'x*', regex: true, color: '#22c55e' })]);
    const got = findMatches('code 500 and 404 xx', rules);
    expect(got.map((m) => [m.start, m.end, m.color])).toEqual([[5, 8, '#ef4444'], [13, 16, '#ef4444'], [17, 19, '#22c55e']]);
  });

  it('겹치면 앞 규칙이 이긴다', () => {
    const rules = compileRules([rule({ pattern: 'connection refused', color: '#ef4444' }), rule({ pattern: 'refused', color: '#3b82f6' })]);
    expect(findMatches('ssh: connection refused; refused', rules).map((m) => [m.start, m.end, m.color]))
      .toEqual([[5, 23, '#ef4444'], [25, 32, '#3b82f6']]);
  });

  it('꺼진 규칙·빈 패턴·틀린 정규식은 쓰지 않고, 색이 형식에 안 맞으면 기본 색을 쓴다', () => {
    const rules = compileRules([
      rule({ pattern: 'a', enabled: false }),
      rule({ pattern: '' }),
      rule({ pattern: '(', regex: true }),
      rule({ pattern: 'ok', color: 'red' }),
    ]);
    expect(rules).toHaveLength(1);
    expect(rules[0].color).toBe(HIGHLIGHT_DEFAULT_COLOR);
    expect(isValidPattern('(', true)).toBe(false);
    expect(isValidPattern('(', false)).toBe(true);
    expect(isValidPattern('', false)).toBe(false);
    expect(compileRules(undefined)).toEqual([]);
  });

  it('한 줄의 일치 수는 한도까지만', () => {
    const rules = compileRules([rule({ pattern: 'a' })]);
    expect(findMatches('a'.repeat(500), rules)).toHaveLength(HIGHLIGHT_MAX_PER_LINE);
  });
});

describe('키워드 강조: 글 위치 → 터미널 칸', () => {
  it('반각 글자만 있으면 위치와 칸이 같다', () => {
    const { text, colOf } = lineText(fakeLine('abc ERROR def', 20), 20);
    expect(text.startsWith('abc ERROR def')).toBe(true);
    const [m] = findMatches(text, compileRules([rule({ pattern: 'error' })]));
    expect(matchCells(m, colOf)).toEqual({ x: 4, width: 5 });
  });

  it('한글(전각) 뒤의 일치는 칸이 밀리고, 한글 일치는 글자 수의 두 배 폭이다', () => {
    const { text, colOf } = lineText(fakeLine('디스크 오류 ERROR', 30), 30);
    const rules = compileRules([rule({ pattern: 'error' }), rule({ pattern: '오류', color: '#eab308' })]);
    const got = findMatches(text, rules).map((m) => ({ ...matchCells(m, colOf), color: m.color }));
    expect(got).toEqual([{ x: 7, width: 4, color: '#eab308' }, { x: 12, width: 5, color: '#ef4444' }]);
  });

  it('줄 끝까지 가는 일치도 폭이 맞다', () => {
    const { text, colOf } = lineText(fakeLine('xxFAIL', 6), 6);
    const [m] = findMatches(text, compileRules([rule({ pattern: 'fail' })]));
    expect(matchCells(m, colOf)).toEqual({ x: 2, width: 4 });
  });
});
```

### `frontend/src/lib/termHighlight.test.ts` — 새 파일 (전체)

```ts
// 키워드 강조를 터미널에 그리는 부분의 검사(가짜 터미널). 근거: spec/behavior/terminal.md §9 (D113).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Terminal } from '@xterm/xterm';
import type { HighlightRule } from '../ipc/types';
import { attachHighlighter, HIGHLIGHT_REFRESH_MS } from './termHighlight';

interface Deco { x: number; width: number; fg?: string; bg?: string; line: number; disposed: boolean }

/** 화면 줄(문자열 배열)을 가진 가짜 터미널. 강조와 표식을 기록한다. */
function fakeTerm(lines: string[], opts: { altScreen?: boolean } = {}) {
  const decos: Deco[] = [];
  const handlers: Record<string, (() => void)[]> = { write: [], scroll: [], resize: [] };
  let liveMarkers = 0;
  const on = (k: string) => (fn: () => void) => {
    handlers[k].push(fn);
    return { dispose: () => { handlers[k] = handlers[k].filter((f) => f !== fn); } };
  };
  const state = { lines, viewportY: 0 };
  const term = {
    rows: 3,
    cols: 20,
    buffer: {
      get active() {
        return {
          viewportY: state.viewportY,
          baseY: 0,
          cursorY: 0,
          getLine: (y: number) => {
            const s = state.lines[y];
            if (s === undefined) return undefined;
            return { length: 20, getCell: (x: number) => ({ getChars: () => s[x] ?? '', getWidth: () => 1 }) };
          },
        };
      },
    },
    registerMarker: (offset: number) => {
      if (opts.altScreen) return undefined;
      liveMarkers++;
      return { line: offset, dispose: () => { liveMarkers--; } };
    },
    registerDecoration: (o: { marker: { line: number }; x: number; width: number; foregroundColor?: string; backgroundColor?: string }) => {
      const d: Deco = { x: o.x, width: o.width, fg: o.foregroundColor, bg: o.backgroundColor, line: o.marker.line, disposed: false };
      decos.push(d);
      return { dispose: () => { d.disposed = true; } };
    },
    onWriteParsed: on('write'),
    onScroll: on('scroll'),
    onResize: on('resize'),
  };
  return {
    term: term as unknown as Terminal,
    state,
    live: () => decos.filter((d) => !d.disposed),
    liveMarkers: () => liveMarkers,
    fire: (k: 'write' | 'scroll' | 'resize') => handlers[k].forEach((f) => f()),
    handlerCount: () => handlers.write.length + handlers.scroll.length + handlers.resize.length,
  };
}

const rule = (p: Partial<HighlightRule>): HighlightRule => ({
  id: p.pattern ?? 'x', pattern: '', color: '#ef4444', regex: false, caseSensitive: false, background: false, enabled: true, ...p,
});

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('키워드 강조: 터미널에 그리기', () => {
  it('보이는 줄의 일치 구간에 강조를 얹는다(글자색 / 바탕색)', () => {
    const f = fakeTerm(['ok', 'an ERROR here', 'WARN x']);
    attachHighlighter(f.term, [rule({ pattern: 'error' }), rule({ pattern: 'warn', color: '#eab308', background: true })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => ({ line: d.line, x: d.x, width: d.width, fg: d.fg, bg: d.bg }))).toEqual([
      { line: 1, x: 3, width: 5, fg: '#ef4444', bg: undefined },
      { line: 2, x: 0, width: 4, fg: '#ffffff', bg: '#eab308' },
    ]);
  });

  it('출력이 이어져도 모아서 한 번만 다시 그리고, 이전 강조와 표식은 지운다', () => {
    const f = fakeTerm(['error', '', '']);
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(1);
    f.state.lines = ['error', 'error', 'x'];
    for (let i = 0; i < 20; i++) f.fire('write');
    expect(f.live()).toHaveLength(1); // 아직 다시 그리지 않았다
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(2);
    expect(f.liveMarkers()).toBe(2);
  });

  it('스크롤하면 새로 보이는 줄을 강조한다', () => {
    const f = fakeTerm(['a', 'b', 'c', 'error', 'e']);
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(0);
    f.state.viewportY = 2;
    f.fire('scroll');
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => d.line)).toEqual([3]);
  });

  it('규칙을 바꾸면 다시 그리고, 규칙을 비우면 모두 지운 뒤 출력에도 타이머를 걸지 않는다', () => {
    const f = fakeTerm(['error warn', '', '']);
    const h = attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    h.setRules([rule({ pattern: 'warn' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live().map((d) => d.x)).toEqual([6]);
    h.setRules([]);
    expect(f.live()).toHaveLength(0);
    expect(f.liveMarkers()).toBe(0);
    f.fire('write');
    expect(vi.getTimerCount()).toBe(0);
  });

  it('전체 화면 프로그램(대체 화면)에서는 강조하지 않는다', () => {
    const f = fakeTerm(['error', '', ''], { altScreen: true });
    attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    expect(f.live()).toHaveLength(0);
  });

  it('dispose 하면 강조·표식·이벤트 연결을 모두 정리한다', () => {
    const f = fakeTerm(['error', '', '']);
    const h = attachHighlighter(f.term, [rule({ pattern: 'error' })]);
    vi.advanceTimersByTime(HIGHLIGHT_REFRESH_MS);
    h.dispose();
    expect(f.live()).toHaveLength(0);
    expect(f.liveMarkers()).toBe(0);
    expect(f.handlerCount()).toBe(0);
    expect(vi.getTimerCount()).toBe(0);
  });
});
```

### `frontend/src/screens/TerminalView.test.tsx` — 수정

```diff
--- a/frontend/src/screens/TerminalView.test.tsx
+++ b/frontend/src/screens/TerminalView.test.tsx
@@ -30,6 +30,10 @@
   onData() { return { dispose() {} }; }
   onSelectionChange() { return { dispose() {} }; }
   onBell() { return { dispose() {} }; }
+  // 키워드 강조(termHighlight)가 듣는 이벤트. 규칙이 없으면 그리지 않으므로 버퍼는 필요 없다.
+  onWriteParsed() { return { dispose() {} }; }
+  onScroll() { return { dispose() {} }; }
+  onResize() { return { dispose() {} }; }
   attachCustomKeyEventHandler() {}
   refresh() {}
   focus() {}
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -28,6 +28,7 @@
     "fontFamily": "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
     "fontSize": 14,
     "guardDangerousOnProd": true,
+    "highlightRules": [],
     "insertPathQuote": true,
     "language": "",
     "licenseKey": "",
```

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -28,6 +28,7 @@
     "fontFamily": "'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace",
     "fontSize": 14,
     "guardDangerousOnProd": true,
+    "highlightRules": [],
     "insertPathQuote": true,
     "language": "",
     "licenseKey": "",
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`hl.title`** (추가)

```json
  "hl.title": "키워드 강조",
  "hl.title": "Keyword highlighting",
  "hl.title": "キーワードの強調表示",
  "hl.title": "关键字高亮",
  "hl.title": "關鍵字醒目提示",
  "hl.title": "Schlüsselwörter hervorheben",
  "hl.title": "Surlignage de mots-clés",
  "hl.title": "Resaltado de palabras clave",
  "hl.title": "Realce de palavras-chave",
  "hl.title": "Подсветка ключевых слов",
```

**`hl.help`** (추가)

```json
  "hl.help": "터미널 출력에서 이 글자가 나오면 색으로 표시합니다. 위에 있는 규칙이 우선입니다. 화면 표시만 바꾸며 복사한 글자와 로그에는 영향이 없습니다. vim 같은 전체 화면 프로그램 안에서는 표시하지 않습니다.",
  "hl.help": "Colors these words when they appear in terminal output. Rules higher in the list win. Only the display changes; copied text and logs are not affected. Nothing is highlighted inside full-screen programs such as vim.",
  "hl.help": "ターミナルの出力にこの文字が現れたら色を付けて表示します。上にあるルールが優先です。表示だけを変え、コピーした文字やログには影響しません。vim などの全画面プログラム内では表示しません。",
  "hl.help": "终端输出中出现这些文字时以颜色标出。靠上的规则优先。只改变显示，不影响复制的文字和日志。在 vim 等全屏程序中不标出。",
  "hl.help": "終端機輸出中出現這些文字時以顏色標示。靠上的規則優先。只改變顯示，不影響複製的文字和記錄。在 vim 等全螢幕程式中不標示。",
  "hl.help": "Färbt diese Wörter ein, wenn sie in der Terminalausgabe vorkommen. Regeln weiter oben haben Vorrang. Nur die Anzeige ändert sich; kopierter Text und Protokolle bleiben unverändert. In Vollbildprogrammen wie vim wird nichts hervorgehoben.",
  "hl.help": "Colore ces mots lorsqu'ils apparaissent dans la sortie du terminal. Les règles situées plus haut l'emportent. Seul l'affichage change ; le texte copié et les journaux ne sont pas modifiés. Rien n'est surligné dans les programmes plein écran comme vim.",
  "hl.help": "Colorea estas palabras cuando aparecen en la salida del terminal. Las reglas situadas más arriba tienen prioridad. Solo cambia la visualización; el texto copiado y los registros no se ven afectados. No se resalta nada dentro de programas a pantalla completa como vim.",
  "hl.help": "Colore estas palavras quando aparecem na saída do terminal. As regras mais acima têm prioridade. Só a exibição muda; o texto copiado e os logs não são afetados. Nada é realçado dentro de programas em tela cheia como o vim.",
  "hl.help": "Выделяет цветом эти слова в выводе терминала. Правила выше по списку имеют приоритет. Меняется только отображение; скопированный текст и журналы не затрагиваются. В полноэкранных программах вроде vim подсветка не работает.",
```

**`hl.add`** (추가)

```json
  "hl.add": "+ 규칙 추가",
  "hl.add": "+ Add rule",
  "hl.add": "+ ルールを追加",
  "hl.add": "+ 添加规则",
  "hl.add": "+ 新增規則",
  "hl.add": "+ Regel hinzufügen",
  "hl.add": "+ Ajouter une règle",
  "hl.add": "+ Añadir regla",
  "hl.add": "+ Adicionar regra",
  "hl.add": "+ Добавить правило",
```

**`hl.pattern`** (추가)

```json
  "hl.pattern": "찾을 글자",
  "hl.pattern": "Text to find",
  "hl.pattern": "検索する文字",
  "hl.pattern": "要查找的文字",
  "hl.pattern": "要尋找的文字",
  "hl.pattern": "Zu suchender Text",
  "hl.pattern": "Texte à rechercher",
  "hl.pattern": "Texto a buscar",
  "hl.pattern": "Texto a localizar",
  "hl.pattern": "Искомый текст",
```

**`hl.enabled`** (추가)

```json
  "hl.enabled": "이 규칙 사용",
  "hl.enabled": "Use this rule",
  "hl.enabled": "このルールを使用",
  "hl.enabled": "启用此规则",
  "hl.enabled": "啟用此規則",
  "hl.enabled": "Diese Regel verwenden",
  "hl.enabled": "Utiliser cette règle",
  "hl.enabled": "Usar esta regla",
  "hl.enabled": "Usar esta regra",
  "hl.enabled": "Использовать это правило",
```

**`hl.color`** (추가)

```json
  "hl.color": "강조 색",
  "hl.color": "Highlight color",
  "hl.color": "強調色",
  "hl.color": "高亮颜色",
  "hl.color": "醒目提示色彩",
  "hl.color": "Hervorhebungsfarbe",
  "hl.color": "Couleur de surlignage",
  "hl.color": "Color de resaltado",
  "hl.color": "Cor do realce",
  "hl.color": "Цвет подсветки",
```

**`hl.color.red`** (추가)

```json
  "hl.color.red": "빨강",
  "hl.color.red": "Red",
  "hl.color.red": "赤",
  "hl.color.red": "红",
  "hl.color.red": "紅",
  "hl.color.red": "Rot",
  "hl.color.red": "Rouge",
  "hl.color.red": "Rojo",
  "hl.color.red": "Vermelho",
  "hl.color.red": "Красный",
```

**`hl.color.orange`** (추가)

```json
  "hl.color.orange": "주황",
  "hl.color.orange": "Orange",
  "hl.color.orange": "オレンジ",
  "hl.color.orange": "橙",
  "hl.color.orange": "橙",
  "hl.color.orange": "Orange",
  "hl.color.orange": "Orange",
  "hl.color.orange": "Naranja",
  "hl.color.orange": "Laranja",
  "hl.color.orange": "Оранжевый",
```

**`hl.color.yellow`** (추가)

```json
  "hl.color.yellow": "노랑",
  "hl.color.yellow": "Yellow",
  "hl.color.yellow": "黄",
  "hl.color.yellow": "黄",
  "hl.color.yellow": "黃",
  "hl.color.yellow": "Gelb",
  "hl.color.yellow": "Jaune",
  "hl.color.yellow": "Amarillo",
  "hl.color.yellow": "Amarelo",
  "hl.color.yellow": "Жёлтый",
```

**`hl.color.green`** (추가)

```json
  "hl.color.green": "초록",
  "hl.color.green": "Green",
  "hl.color.green": "緑",
  "hl.color.green": "绿",
  "hl.color.green": "綠",
  "hl.color.green": "Grün",
  "hl.color.green": "Vert",
  "hl.color.green": "Verde",
  "hl.color.green": "Verde",
  "hl.color.green": "Зелёный",
```

**`hl.color.blue`** (추가)

```json
  "hl.color.blue": "파랑",
  "hl.color.blue": "Blue",
  "hl.color.blue": "青",
  "hl.color.blue": "蓝",
  "hl.color.blue": "藍",
  "hl.color.blue": "Blau",
  "hl.color.blue": "Bleu",
  "hl.color.blue": "Azul",
  "hl.color.blue": "Azul",
  "hl.color.blue": "Синий",
```

**`hl.color.purple`** (추가)

```json
  "hl.color.purple": "보라",
  "hl.color.purple": "Purple",
  "hl.color.purple": "紫",
  "hl.color.purple": "紫",
  "hl.color.purple": "紫",
  "hl.color.purple": "Violett",
  "hl.color.purple": "Violet",
  "hl.color.purple": "Morado",
  "hl.color.purple": "Roxo",
  "hl.color.purple": "Фиолетовый",
```

**`hl.background`** (추가)

```json
  "hl.background": "바탕 칠하기",
  "hl.background": "Fill background",
  "hl.background": "背景を塗る",
  "hl.background": "填充背景",
  "hl.background": "填滿背景",
  "hl.background": "Hintergrund füllen",
  "hl.background": "Remplir le fond",
  "hl.background": "Rellenar fondo",
  "hl.background": "Preencher fundo",
  "hl.background": "Заливка фона",
```

**`hl.regex`** (추가)

```json
  "hl.regex": "정규식",
  "hl.regex": "Regex",
  "hl.regex": "正規表現",
  "hl.regex": "正则",
  "hl.regex": "規則運算式",
  "hl.regex": "Regex",
  "hl.regex": "Regex",
  "hl.regex": "Regex",
  "hl.regex": "Regex",
  "hl.regex": "Regex",
```

**`hl.case`** (추가)

```json
  "hl.case": "대소문자 구분",
  "hl.case": "Match case",
  "hl.case": "大文字小文字を区別",
  "hl.case": "区分大小写",
  "hl.case": "區分大小寫",
  "hl.case": "Groß-/Kleinschreibung",
  "hl.case": "Respecter la casse",
  "hl.case": "Distinguir mayúsculas",
  "hl.case": "Diferenciar maiúsculas",
  "hl.case": "Учитывать регистр",
```


## F. 명세

### `spec/acceptance/04-hosts-sftp-settings-sync.md` — 수정

```diff
--- a/spec/acceptance/04-hosts-sftp-settings-sync.md
+++ b/spec/acceptance/04-hosts-sftp-settings-sync.md
@@ -43,6 +43,7 @@
 | G6 | 정보 화면 | | 개발자 `YangHun Lee (kai)`, 데이터 폴더 경로, 단축키 전체 표 |
 | G7 | 업데이트 [업데이트 확인], 배포 저장소 미지정 | | "배포 저장소가 설정되지 않은 빌드입니다" |
 | G8 | [기본 크기로] | | 창이 1440×900 가운데 |
+| G8a | 키워드 강조 규칙 `error`(빨강), `디스크`(노랑, 바탕 칠하기) | 터미널에 `디스크 Error: disk ERROR` 출력 | `Error`·`ERROR` 는 빨간 글자, `디스크` 는 노란 바탕에 흰 글자. 복사하면 원래 글자 그대로. 규칙을 끄거나 지우면 곧바로 원래 색 (D113) |
 
 ## 가져오기 (import.md, ui/dialogs.md §5)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/behavior/terminal.md` — 수정

```diff
--- a/spec/behavior/terminal.md
+++ b/spec/behavior/terminal.md
@@ -93,6 +93,19 @@
 - 설정 "서버의 클립보드 쓰기 허용 (OSC 52)"이 꺼져 있으면 모두 무시.
 - 켜져 있으면 **쓰기 요청만** 처리한다: 내용이 비었거나 `?`(읽기 요청)이면 무시(읽기는 항상 거절). base64 로 인코딩된 UTF-8 텍스트를 풀어 **1,000,000자 이하일 때만** 이 PC 클립보드에 쓴다. 잘못된 base64 는 조용히 무시.
 
+## 9. 키워드 강조 (2026-11-23, D113)
+
+- 설정 `highlightRules`(설정 > 화면 > 키워드 강조)의 규칙마다, 터미널 출력에서 그 글자가 나오는 자리를 색으로 표시한다. 규칙: 찾을 글자(`pattern`), 색(`color`), 정규식 여부(`regex`), 대소문자 구분(`caseSensitive`, 기본 구분 안 함), 바탕 칠하기(`background`), 사용 여부(`enabled`).
+- **표시 방식**: 기본은 일치한 글자의 **글자색**을 규칙의 색으로 바꾼다. 바탕 칠하기를 켠 규칙은 바탕을 규칙의 색으로 칠하고 글자는 흰색으로 한다.
+- **우선순위**: 목록에서 위에 있는 규칙이 먼저다. 한 줄에서 두 규칙의 일치가 겹치면 위 규칙의 것만 표시한다.
+- **화면 표시만 바꾼다**: 받은 출력 내용은 바꾸지 않는다. 복사한 글자, 찾기, 세션 로그에는 영향이 없다. 찾기의 강조와 마우스 선택은 키워드 강조 위에 보인다.
+- **범위**: 지금 화면에 보이는 줄만 살핀다(스크롤하면 그때 보이는 줄을 다시 살핀다). 줄 단위로 찾으므로 화면 폭 때문에 다음 줄로 넘어간 낱말은 찾지 못한다. vim·htop 같은 **전체 화면 프로그램 안에서는 표시하지 않는다**.
+- **다시 그리는 때**: 출력이 오거나, 스크롤하거나, 창 크기가 바뀌거나, 규칙이 바뀌었을 때. 출력이 이어지는 동안에는 약 0.08초에 한 번으로 모은다. 규칙이 하나도 없으면 아무 일도 하지 않는다.
+- **한도**: 규칙 30개, 한 줄에 50곳, 한 화면에 400곳까지. 넘는 것은 표시하지 않는다.
+- 빈 글자, 틀린 정규식, 꺼진 규칙은 쓰지 않는다. 색이 `#RRGGBB` 형식이 아니면 빨강(`#ef4444`)으로 본다. 정규식이 빈 글자에 맞는 자리는 표시하지 않는다.
+- 한글 같은 전각 글자는 두 칸으로 계산해 자리를 맞춘다.
+- 규칙을 바꾸면 열려 있는 모든 터미널(보이지 않는 탭 포함)에 곧바로 적용한다.
+
 ## 모든 창 동시 입력 (쇠사슬, 2026-11-12 사용자 요청)
 
 - 왼쪽 위 아이콘 묶음(서버 목록 · 스니펫 · **동시 입력** · 설정)에 쇠사슬 아이콘. 꺼짐 = 끊어진 쇠사슬(툴팁 "일괄 입력 꺼짐 — 누르면 보이는 모든 창에 같이 입력합니다"), 켜짐 = 이어진 쇠사슬 + 경고색 배경(warn-soft, 아이콘 warn, 툴팁 "일괄 입력 켜짐 — 보이는 모든 창에 같이 입력합니다 (누르면 끄기)").
```

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -88,6 +88,7 @@
 | confirmMultilinePaste | true | 여러 줄 붙여넣기 확인 |
 | guardDangerousOnProd | true | 운영 환경 위험 명령 확인 |
 | dangerousPatterns | 아래 목록 | 정규식 문자열 배열 |
+| highlightRules | 빈 배열 | 키워드 강조 규칙 배열(2026-11-23, D113). 항목: id, pattern(찾을 글자 또는 정규식), color(`#RRGGBB`), regex, caseSensitive, background(바탕 칠하기), enabled. 순서가 우선순위 ([../behavior/terminal.md](../behavior/terminal.md) §9) |
 | autoReconnect | true | |
 | autoLockMinutes | 15 | 0 이면 자동 잠금 없음 |
 | uploadDir | `~/.dariterm/uploads`  | 붙여넣기 이미지·파일 업로드 폴더 |
```

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -27,6 +27,7 @@
   - **스크롤백 (줄)**(숫자, 500~200000으로 제한, 비면 10000)
   - **최대 화면 분할 (상하)**(세그먼트 1·2·3·4·5, 기본 3, 도움말 "한 탭에서 위아래로 나눌 수 있는 최대 창 수입니다. 도달하면 [아래로 나누기] 버튼이 비활성화됩니다.") — 2026-11-12 추가, ui/panes.md §2
 - 카드: 토글 **커서 깜빡임**, 토글 **벨 소리**.
+- **키워드 강조**(2026-11-23, D113, 동작은 behavior/terminal.md §9): 도움말 "터미널 출력에서 이 글자가 나오면 색으로 표시합니다. 위에 있는 규칙이 우선입니다. 화면 표시만 바꾸며 복사한 글자와 로그에는 영향이 없습니다. vim 같은 전체 화면 프로그램 안에서는 표시하지 않습니다." 규칙마다 한 줄: 사용 스위치 / "찾을 글자" 입력(고정폭, 틀린 정규식이면 빨간 테두리) / 색 6개(빨강 `#ef4444`·주황 `#f97316`·노랑 `#eab308`·초록 `#22c55e`·파랑 `#3b82f6`·보라 `#a855f7`, 18px 원, 고른 색에 테두리) / 체크 "바탕 칠하기" / 체크 "정규식" / 체크 "대소문자 구분" / [삭제](휴지통). 아래에 [+ 규칙 추가](30개면 비활성): 새 규칙은 빈 글자, 빨강, 체크 모두 꺼짐, 사용 켜짐. 스위치·색·체크·삭제·추가는 누르는 즉시 저장하고, 찾을 글자는 입력 칸을 벗어날 때 저장한다.
 
 ## 2. 입력·붙여넣기
 카드 안 토글(설명 포함):
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [highlight.test.ts](../../frontend/src/lib/highlight.test.ts) "규칙 → 일치 구간" 7개 | 대소문자 무시로 모두 찾기 / 대소문자 구분 / 보통 글자의 특수 문자(`[WARN]`)는 글자 그대로 / 정규식과 빈 일치 건너뛰기 / 겹치면 앞 규칙 / 꺼진 규칙·빈 패턴·틀린 정규식 제외와 색 형식 / 한 줄 한도 |
| 같은 파일 "글 위치 → 터미널 칸" 3개 | 반각만 있을 때 / **한글 뒤의 일치는 칸이 밀리고 한글 일치는 두 배 폭** / 줄 끝까지 가는 일치 |
| [termHighlight.test.ts](../../frontend/src/lib/termHighlight.test.ts) 6개 (가짜 터미널) | 보이는 줄에 글자색·바탕색 강조를 얹는다 / 출력 20번이 와도 한 번만 다시 그리고 이전 강조·표식을 지운다 / 스크롤하면 새로 보이는 줄 / 규칙을 바꾸면 다시 그리고, 비우면 모두 지우고 타이머를 걸지 않는다 / 대체 화면에서는 강조하지 않는다 / `dispose` 가 강조·표식·이벤트 연결을 모두 정리 |
| [TerminalView.test.tsx](../../frontend/src/screens/TerminalView.test.tsx) (기존) | 가짜 터미널에 새 이벤트 3개를 더해 그대로 통과 |
| 계약 테스트 | `store.get` 의 `highlightRules: []` |

```bash
(cd frontend && npx tsc --noEmit)
(cd frontend && npx vitest run)     # PowerShell 에서. 42개 파일 275개 통과
go test ./internal/store/ ./internal/app/
```

**자동 테스트가 확인하지 못하는 것**: 실제 xterm.js(WebGL·DOM 렌더러)에서 강조가 어떻게 보이는지. 강조 API(`registerDecoration` 의 `foregroundColor`·`backgroundColor`)는 찾기 강조(D96)가 이미 쓰는 것과 같지만, 이 기능의 화면은 띄워 보지 않았다. 설정 화면(`SettingsHighlight.tsx`)도 화면 테스트가 없다.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정 > 화면 > 키워드 강조 > [+ 규칙 추가] → `error`, 빨강 → 터미널에서 `echo "an Error and ERROR"` | 두 낱말이 빨간 글자로 보인다. 출력 직후(0.1초 안) 색이 입혀진다 |
| 2 | 규칙 `디스크`, 노랑, "바탕 칠하기" → `echo "디스크 오류 error"` | `디스크` 가 노란 바탕에 흰 글자, 자리가 정확히 맞는다(한글 폭) |
| 3 | 강조된 줄을 마우스로 선택해 복사 | 원래 글자 그대로 복사된다. 선택 색이 강조 위에 보인다 |
| 4 | 긴 로그를 `cat` 한 뒤 위로 스크롤 | 스크롤해서 보이는 줄에도 강조가 붙는다 |
| 5 | 규칙의 스위치를 끈다 / 지운다 | 열려 있는 모든 탭에서 곧바로 원래 색 |
| 6 | 정규식 체크 + `\b5\d\d\b` → `echo "status 500 and 1500"` | `500` 만 강조(`1500` 은 아님) |
| 7 | 정규식 체크 + `(` | 입력 칸에 빨간 테두리, 강조는 안 된다 |
| 8 | `vim` 을 열어 `error` 를 친다 | 강조되지 않는다(프로그램 화면). 나오면 다시 강조된다 |
| 9 | 라이트 터미널 테마 | 6가지 색이 모두 읽힌다 |
| 10 | 찾기(Ctrl+Shift+F)로 강조된 낱말을 찾는다 | 찾기의 현재 항목 표시가 키워드 강조 위에 보인다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 규칙이 없는 사용자(기본) | 터미널마다 이벤트 구독 3개가 생기지만 일은 하지 않는다(`schedule` 이 곧바로 돌아온다) | `termHighlight.test.ts` 4번째 |
| 출력이 많은 세션 + 규칙 있음 | 0.08초마다 보이는 줄을 다시 살피고 강조를 다시 만든다 | 한도 400곳. 부하는 재 보지 않았다. 문제가 되면 `HIGHLIGHT_REFRESH_MS` 를 늘린다 |
| 찾기 강조 | 같은 종류의 강조를 쓰지만 서로 다른 목록으로 관리한다. 지우는 순서(강조 → 표식)도 찾기와 같게 했다 | D101 계열 문제(남는 표시)를 피한 순서 |
| 저장 데이터 | 설정에 `highlightRules` 가 생긴다. 옛 버전 앱은 무시한다 | — |
| 문구 검사 | 화면 코드에 한국어를 직접 쓰지 않았다(문구 키 사용) | `i18n.test.ts` 통과 |

## J. 체크리스트

- [x] `lib/highlight.ts`(계산), `lib/termHighlight.ts`(그리기), `screens/SettingsHighlight.tsx`(설정 화면) 새 파일
- [x] `TerminalView.tsx`: 터미널마다 강조기 붙이기·규칙 변경 알림·정리. `SettingsAppearance.tsx`: 화면에 넣기
- [x] 타입(`HighlightRule`)·기본값·계약 파일 2개, 백엔드 기본값(`store.go`)
- [x] 문구 16개 추가(10개 언어), 명세 behavior/terminal.md §9·data/store.md §4·ui/settings.md §1·수용 G8a, decisions D113
- [x] 자동 테스트 16개 추가, 전체 통과
- [ ] 수동 검증 1~10 (실제 화면)
