# 명령 모음 변수 — `{{이름}}` 을 실행할 때 물어 채운다

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D138). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 명세 [spec/behavior/terminal.md](../../spec/behavior/terminal.md) §7.1, [spec/ui/sftp-snippets.md](../../spec/ui/sftp-snippets.md) §4.3, [spec/data/store.md](../../spec/data/store.md) §7
**적용 순서**: 2026-11-27 의 2번째. 5번(매크로)이 이 문서의 `snippetVars.ts`·`VarsDialog.tsx` 를 쓴다.

---

## 0. 한 줄 요약

명령 모음의 명령에 `{{이름}}` 을 적어 두면, 누를 때 **값을 묻는 창**이 뜨고 채운 명령이 입력된다. `{{이름:기본값}}` 으로 기본값을 줄 수 있다.

```
저장한 명령:   systemctl restart {{서비스}} --no-block {{옵션:-q}}
누르면:        [서비스: ______ ]  [옵션: -q ]   입력될 내용: systemctl restart nginx --no-block -q
```

## 1. 배경

2026-11-27 사용자 요청("권장 순서대로 구현")의 첫 항목. 다른 터미널 프로그램과 견주어 빠진 기능을 추린 목록에서, 지금 있는 명령 모음에 가장 적게 손대고 얹을 수 있는 것이다. 원본 앱에 없던 기능이라 명세도 이번에 새로 썼다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 표기 | `{{이름}}`, `{{이름:기본값}}` | 셸 문법(`$VAR`, `${VAR}`)과 겹치지 않는다 |
| 이름에 쓸 수 있는 글자 | 글자·숫자·밑줄로 시작, 그 뒤 빈칸·하이픈 허용(한글 포함, 40자까지) | "서비스 이름" 같은 이름을 쓰게 한다 |
| 변수가 아닌 것 | 점으로 시작하는 `{{.Names}}`, 앞에 `\` 를 붙인 `\{{이름}}` | `docker ps --format "{{.Names}}"` 를 망가뜨리지 않는다. 그 밖의 경우는 `\` 로 피한다 |
| 저장 형식 | 그대로(본문은 글자일 뿐) | 데이터·동기화·백업이 바뀌지 않는다. 옛 버전에서는 `{{…}}` 가 그대로 입력된다 |
| 지난 값 | 앱이 켜져 있는 동안만 기억, 파일에 남기지 않음 | 값에 비밀번호가 들어갈 수 있다 |
| 빈 칸 | 빈 글자를 넣는다(기본값으로 돌아가지 않음) | 기본값은 칸에 미리 채워져 있어, 지웠다면 뜻이 있는 것이다 |
| 값 안의 `{{…}}` | 다시 풀지 않는다 | 값에 따라 결과가 달라지는 것을 막는다 |
| 일괄 입력 | 값은 한 번 묻고 모든 대상 창에 같은 글자를 보낸다 | 지금의 명령 입력 규칙과 같다 |

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 내용 |
|---|---|---|
| `frontend/src/lib/snippetVars.ts` | 코드 | **새 파일** — `parseVars`, `parseVarsAll`, `fillVars`, 지난 값 기억 |
| `frontend/src/screens/VarsDialog.tsx` | 코드 | **새 파일** — 값을 묻는 창(미리 보기 포함) |
| `frontend/src/screens/SnippetsPanel.tsx` | 코드 | 누르면 변수가 있는지 보고 창을 띄움, 편집 창 도움말 |
| `frontend/src/styles/app.css` | 스타일 | `.vars-preview` |
| `frontend/src/lib/snippetVars.test.ts` | 테스트 | **새 파일**(10개) |
| `frontend/src/screens/Automation.test.tsx` | 테스트 | "명령 모음 변수" 3개 |
| `locales/*.json` | 문구 | `snip.varHelp`, `vars.run`, `vars.preview` |
| `spec/behavior/terminal.md`, `spec/ui/sftp-snippets.md`, `spec/data/store.md`, `spec/acceptance/04-hosts-sftp-settings-sync.md`(F12) | 명세 | 수정 |

백엔드(Go)는 바꾸지 않았다.

아래 `diff` 는 **오늘(2026-11-27) 작업을 시작하기 전의 파일**과 지금 파일을 견준 것이다. 이 폴더는 git 저장소가 아니라서, 오늘 적용한 수정 기록을 거꾸로 돌려 고치기 전 파일을 되살린 뒤 뽑았다(되살린 것에 수정을 다시 적용하면 지금 파일과 같아지는 것을 확인했다). 한 파일을 오늘의 여러 문서가 고친 경우, 각 문서에는 그 문서의 몫이 든 조각만 싣는다 — 조각 안의 문맥 줄에 다른 문서의 코드가 섞여 있을 수 있고, 줄 번호는 다섯 문서를 모두 적용한 최종 파일 기준이다. 새 파일은 최종 내용 전체를 싣는다(같은 날의 추가 지시 D142·D143 까지 반영된 것).

## B. 코드

### 먼저 읽을 곳

[frontend/src/lib/snippetVars.ts](../../frontend/src/lib/snippetVars.ts):

```ts
// (\\?) 앞의 역슬래시 · 이름 · (선택) :기본값
const VAR_RE = /(\\?)\{\{\s*([\p{L}\p{N}_][\p{L}\p{N}_ -]*?)\s*(?::([^}\r\n]*))?\}\}/gu;
```

- `parseVars(body)`: 나온 순서대로, 같은 이름은 한 번. 기본값은 처음 적힌 것.
- `fillVars(body, values)`: 값이 없는 변수는 기본값, `\{{` 는 `{{` 로. 바꾸기 함수를 써서 값 안의 `$1` 같은 글자가 풀리지 않는다.

[frontend/src/screens/SnippetsPanel.tsx](../../frontend/src/screens/SnippetsPanel.tsx) `onRun`: 포커스된 세션이 없으면 묻기 전에 안내 → 변수가 있으면 `VarsDialog` → `fillVars` 뒤 `sendSnippet`.

문구에 `{{host}}` 같은 예시를 그대로 적으면 문구의 자리표시자(`{이름}`)로 읽히므로, 예시는 자리표시자 값으로 넘긴다: `t('snip.varHelp', { a: '{{host}}', b: '{{port:22}}' })`.

### `frontend/src/lib/snippetVars.ts` — 새 파일 (전체)

```ts
// 명령 모음의 변수: 본문의 {{이름}} · {{이름:기본값}} 을 실행할 때 물어 채운다.
// 근거: spec/behavior/terminal.md §7.1 (D138).

export interface SnippetVar {
  name: string;
  /** {{이름:기본값}} 의 기본값. 없으면 빈 글자. */
  def: string;
}

/** 변수 이름의 최대 길이. */
export const VAR_NAME_MAX = 40;

/**
 * 변수 자리. 이름은 글자·숫자·밑줄로 시작하고 그 뒤로 글자·숫자·밑줄·빈칸·하이픈이 올 수 있다.
 * 점으로 시작하는 `{{.Names}}`(docker·kubectl 의 출력 형식)는 변수가 아니다. 앞에 `\` 가 있으면 글자 그대로다.
 */
const VAR_RE = /(\\?)\{\{\s*([\p{L}\p{N}_][\p{L}\p{N}_ -]*?)\s*(?::([^}\r\n]*))?\}\}/gu;

/** 본문에 나온 변수를 나온 순서대로(같은 이름은 한 번만) 돌려준다. 기본값은 처음 적힌 것을 쓴다. */
export function parseVars(body: string): SnippetVar[] {
  const out: SnippetVar[] = [];
  for (const m of body.matchAll(VAR_RE)) {
    if (m[1]) continue; // \{{…}} 는 글자 그대로
    const name = m[2];
    if (name.length > VAR_NAME_MAX) continue;
    const found = out.find((v) => v.name === name);
    if (!found) out.push({ name, def: m[3] ?? '' });
    else if (!found.def && m[3]) found.def = m[3];
  }
  return out;
}

/** 여러 본문(매크로의 단계들)에 나온 변수를 합친다. */
export function parseVarsAll(bodies: readonly string[]): SnippetVar[] {
  return parseVars(bodies.join('\n'));
}

/** 변수 자리를 값으로 바꾼다. 값이 없는 변수는 기본값(없으면 빈 글자)을 쓴다. `\{{` 는 `{{` 로 남긴다. */
export function fillVars(body: string, values: Readonly<Record<string, string>>): string {
  return body.replace(VAR_RE, (whole, esc: string, name: string, def: string | undefined) => {
    if (esc) return whole.slice(1);
    if (name.length > VAR_NAME_MAX) return whole;
    return Object.prototype.hasOwnProperty.call(values, name) ? values[name] : def ?? '';
  });
}

/** 마지막으로 넣은 값(명령 id → 이름 → 값). 앱을 끄면 사라진다 — 비밀번호 같은 값이 파일에 남지 않게. */
const lastValues = new Map<string, Record<string, string>>();

/** 물어볼 때 칸에 미리 채울 값: 지난번에 넣은 값, 없으면 기본값. */
export function initialValues(id: string, vars: readonly SnippetVar[]): Record<string, string> {
  const last = lastValues.get(id) ?? {};
  return Object.fromEntries(vars.map((v) => [v.name, last[v.name] ?? v.def]));
}

export function rememberValues(id: string, values: Record<string, string>): void {
  lastValues.set(id, { ...values });
}

/** 테스트용. */
export function resetRememberedValues(): void {
  lastValues.clear();
}
```

### `frontend/src/screens/VarsDialog.tsx` — 새 파일 (전체)

```tsx
// 명령·매크로를 실행하기 전에 변수 값을 묻는 창. 근거: spec/ui/sftp-snippets.md §4.3 (D138).
import { useState, type FormEvent } from 'react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import type { SnippetVar } from '../lib/snippetVars';
import { t } from '../i18n';

export function VarsDialog({ title, vars, initial, preview, onCancel, onRun }: {
  title: string;
  vars: SnippetVar[];
  initial: Record<string, string>;
  /** 지금 값으로 채운 명령(미리 보기). */
  preview: (values: Record<string, string>) => string;
  onCancel: () => void;
  onRun: (values: Record<string, string>) => void;
}) {
  const [values, setValues] = useState<Record<string, string>>(initial);
  const submit = (e?: FormEvent) => {
    e?.preventDefault();
    onRun(values);
  };
  return (
    <Modal
      title={title}
      size="sm"
      onClose={onCancel}
      foot={
        <>
          <button className="btn" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn primary" onClick={() => submit()}>{t('vars.run')}</button>
        </>
      }
    >
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {vars.map((v, i) => (
          <Field key={v.name} label={v.name}>
            <input
              className="input mono"
              autoFocus={i === 0}
              aria-label={v.name}
              placeholder={v.def}
              value={values[v.name] ?? ''}
              onChange={(e) => setValues((cur) => ({ ...cur, [v.name]: e.target.value }))}
            />
          </Field>
        ))}
        {/* Enter 로 실행되게 보이지 않는 제출 버튼을 둔다 */}
        <button type="submit" style={{ display: 'none' }} aria-hidden="true" tabIndex={-1} />
      </form>
      <Field label={t('vars.preview')}>
        <pre className="vars-preview mono selectable">{preview(values)}</pre>
      </Field>
    </Modal>
  );
}
```

### `frontend/src/screens/SnippetsPanel.tsx` — 수정

이 파일은 5번 문서(매크로)에서 전체를 다시 썼으므로 **전체 코드는 5번 문서 §B** 에 있다. 이 문서의 몫은 다음 세 군데다.

```tsx
// ① 항목을 눌렀을 때: 변수가 있으면 바로 보내지 않고 값을 묻는다
const onRun = (s: Snippet) => {
  if (!focusedSessionId()) {        // 보낼 창이 없으면 값을 묻기 전에 알린다
    notifyNoFocus();
    return;
  }
  if (varsOf(s).length > 0) {       // varsOf: 명령은 parseVars(s.body), 매크로는 모든 단계의 보낼 글자
    setAsking(s);
    return;
  }
  void execute(s, {});
};

// ② 값 창에서 [실행]: 지난 값으로 기억하고, 채워서 보낸다
const onRunWith = (s: Snippet, values: Record<string, string>) => {
  rememberValues(s.id, values);
  setAsking(null);
  void execute(s, values);          // 명령이면 sendSnippet(fillVars(s.body, values), s.sendEnter)
};

// ③ 값 창
{asking && (
  <VarsDialog
    title={asking.name || t('snip.noName')}
    vars={varsOf(asking)}
    initial={initialValues(asking.id, varsOf(asking))}
    preview={(values) => previewOf(asking, values)}
    onCancel={() => setAsking(null)}
    onRun={(values) => onRunWith(asking, values)}
  />
)}
```

명령 편집 창의 "명령" 도움말에는 `t('snip.varHelp', { a: '{{host}}', b: '{{port:22}}' })` 를 덧붙였다.

### `frontend/src/styles/app.css` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 2, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -326,6 +329,25 @@
 .pane-head .pane-target { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
 .pane-head .spacer { flex: 1; }
 .env-tag { font-size: 10px; font-weight: 700; padding: 0 5px; border-radius: 4px; color: #fff; }
+/* 변수 값을 묻는 창의 미리 보기 (ui/sftp-snippets.md §4.3). */
+.vars-preview {
+  margin: 0; padding: 8px 10px; border-radius: 6px; border: 1px solid var(--border); background: var(--bg);
+  font-size: 12px; line-height: 1.5; white-space: pre-wrap; word-break: break-all; max-height: 140px; overflow: auto;
+}
+/* 창 머리글의 포트 포워딩 버튼(아이콘 + 개수)과 정보 창의 표. 근거: spec/ui/panes.md §2.6 (D137). */
+.pane-fwd {
+  flex: none; display: inline-flex; align-items: center; gap: 4px; height: 18px; padding: 0 7px; border-radius: 9px;
+  border: 1px solid var(--accent); background: transparent; color: var(--fg-2); font: inherit; font-size: 11px; font-weight: 600; cursor: pointer;
+}
+.pane-fwd:hover { background: var(--hover); color: var(--fg); }
+.pane-fwd.failed { border-color: var(--warn); color: var(--warn); }
+.fwd-info { width: 100%; border-collapse: collapse; font-size: 12.5px; }
+.fwd-info th { text-align: left; font-weight: 600; color: var(--fg-2); padding: 6px 8px; border-bottom: 1px solid var(--border); white-space: nowrap; }
+.fwd-info td { padding: 6px 8px; border-bottom: 1px solid var(--border); vertical-align: top; }
+.fwd-info td:nth-child(1), .fwd-info td:nth-child(2) { white-space: nowrap; }
+.fwd-info tr.failed td.mono { color: var(--fg-3); }
+.fwd-ok { color: var(--ok, #22c55e); }
+.fwd-bad { color: var(--danger); word-break: break-word; }
 .pane-body { flex: 1; min-height: 0; position: relative; display: flex; padding: 4px 2px 2px 8px; background: #0d1117; }
 .term-host { flex: 1; min-width: 0; min-height: 0; display: flex; }
 /* 세션별로 살려 두는 터미널 DOM(칸이 다시 그려질 때 떼었다 붙인다). */
```

## C. 테스트 코드

### `frontend/src/lib/snippetVars.test.ts` — 새 파일 (전체)

```ts
// 명령 모음 변수 (spec/behavior/terminal.md §7.1, D138).
import { beforeEach, describe, expect, it } from 'vitest';
import { fillVars, initialValues, parseVars, parseVarsAll, rememberValues, resetRememberedValues } from './snippetVars';

beforeEach(() => resetRememberedValues());

describe('parseVars', () => {
  it('나온 순서대로, 같은 이름은 한 번만', () => {
    expect(parseVars('ssh {{user}}@{{host}} -p {{port:22}}; echo {{host}}')).toEqual([
      { name: 'user', def: '' },
      { name: 'host', def: '' },
      { name: 'port', def: '22' },
    ]);
  });

  it('한글 이름·빈칸·하이픈을 받고 앞뒤 빈칸은 뺀다', () => {
    expect(parseVars('systemctl restart {{ 서비스 이름 }} {{log-level:info}}')).toEqual([
      { name: '서비스 이름', def: '' },
      { name: 'log-level', def: 'info' },
    ]);
  });

  it('점으로 시작하는 출력 형식과 \\{{ 는 변수가 아니다', () => {
    expect(parseVars('docker ps --format "{{.Names}} {{ .Status }}"')).toEqual([]);
    expect(parseVars('echo \\{{name}}')).toEqual([]);
  });

  it('기본값은 처음 적힌 것을 쓴다', () => {
    expect(parseVars('{{a}} {{a:1}} {{a:2}}')).toEqual([{ name: 'a', def: '1' }]);
  });

  it('여러 본문을 합친다', () => {
    expect(parseVarsAll(['cd {{dir}}', 'tail -f {{file:app.log}}', 'ls {{dir}}']).map((v) => v.name)).toEqual(['dir', 'file']);
  });
});

describe('fillVars', () => {
  it('값으로 바꾸고, 값이 없으면 기본값', () => {
    expect(fillVars('ssh {{user}}@{{host}} -p {{port:22}}', { user: 'root', host: '10.0.0.1' })).toBe('ssh root@10.0.0.1 -p 22');
  });

  it('빈 값을 넣으면 기본값이 아니라 빈 글자', () => {
    expect(fillVars('a{{x:1}}b', { x: '' })).toBe('ab');
  });

  it('변수가 아닌 것은 그대로, \\{{ 는 {{ 로', () => {
    expect(fillVars('docker ps --format "{{.Names}}" \\{{name}}', { name: 'x' })).toBe('docker ps --format "{{.Names}}" {{name}}');
  });

  it('값 안의 {{ 를 다시 풀지 않는다', () => {
    expect(fillVars('{{a}}', { a: '{{b}}', b: 'no' })).toBe('{{b}}');
  });
});

describe('마지막 값 기억', () => {
  it('지난번 값을 먼저, 없으면 기본값', () => {
    const vars = parseVars('{{host}} {{port:22}}');
    expect(initialValues('s1', vars)).toEqual({ host: '', port: '22' });
    rememberValues('s1', { host: 'web-01', port: '2222' });
    expect(initialValues('s1', vars)).toEqual({ host: 'web-01', port: '2222' });
    expect(initialValues('s2', vars)).toEqual({ host: '', port: '22' });
  });
});
```

### `frontend/src/screens/Automation.test.tsx` — 새 파일 (이 문서의 몫: `describe('명령 모음 변수 (D138)')`, 전체는 5번 문서 §C)

```tsx
describe('명령 모음 변수 (D138)', () => {
  it('변수가 있는 명령은 값을 물은 뒤 채워서 보낸다', () => {
    const snip: Snippet = { id: 'sn1', name: '재시작', body: 'systemctl restart {{서비스}} --no-block {{옵션:-q}}', sendEnter: true };
    seed({ snippets: [snip] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('재시작'));
    expect(termInput).not.toHaveBeenCalled();

    const dialog = screen.getByRole('dialog', { name: '재시작' });
    const option = within(dialog).getByLabelText('옵션') as HTMLInputElement;
    expect(option.value).toBe('-q'); // 기본값이 미리 채워진다
    fireEvent.change(within(dialog).getByLabelText('서비스'), { target: { value: 'nginx' } });
    expect(within(dialog).getByText('systemctl restart nginx --no-block -q')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: '실행' }));

    expect(termInput).toHaveBeenCalledWith('s1', 'systemctl restart nginx --no-block -q\r');
    expect(screen.queryByRole('dialog', { name: '재시작' })).toBeNull();
  });

  it('변수가 없는 명령은 묻지 않고 바로 보낸다', () => {
    seed({ snippets: [{ id: 'sn1', name: '가동 시간', body: 'uptime', sendEnter: true }] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('가동 시간'));
    expect(termInput).toHaveBeenCalledWith('s1', 'uptime\r');
  });

  it('다음에 물을 때는 지난번 값을 채워 둔다', () => {
    seed({ snippets: [{ id: 'sn1', name: '접속', body: 'ssh {{host}}', sendEnter: true }] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('접속'));
    fireEvent.change(screen.getByLabelText('host'), { target: { value: 'db-01' } });
    fireEvent.click(screen.getByRole('button', { name: '실행' }));
    fireEvent.click(screen.getByText('접속'));
    expect((screen.getByLabelText('host') as HTMLInputElement).value).toBe('db-01');
  });
});
```

## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다. 자리표시자(`{이름}`)는 언어마다 같은 이름을 쓴다(`frontend: npx vitest run src/i18n`, `go test ./internal/i18n` 이 확인한다).

**`snip.varHelp`** (추가)

```json
  "snip.varHelp": "{a} 처럼 적으면 실행할 때 값을 묻습니다 (기본값은 {b})",
  "snip.varHelp": "Write {a} to be asked for a value when it runs (default value: {b})",
  "snip.varHelp": "{a} と書くと実行時に値を尋ねます (既定値は {b})",
  "snip.varHelp": "写成 {a} 会在运行时询问值（默认值写法：{b}）",
  "snip.varHelp": "寫成 {a} 會在執行時詢問值（預設值寫法：{b}）",
  "snip.varHelp": "Mit {a} wird beim Ausführen nach einem Wert gefragt (Standardwert: {b})",
  "snip.varHelp": "Écrivez {a} pour qu'une valeur soit demandée à l'exécution (valeur par défaut : {b})",
  "snip.varHelp": "Escriba {a} para que se pida un valor al ejecutar (valor predeterminado: {b})",
  "snip.varHelp": "Escreva {a} para que um valor seja pedido ao executar (valor padrão: {b})",
  "snip.varHelp": "Напишите {a}, чтобы при запуске запрашивалось значение (значение по умолчанию: {b})",
```

**`vars.run`** (추가)

```json
  "vars.run": "실행",
  "vars.run": "Run",
  "vars.run": "実行",
  "vars.run": "运行",
  "vars.run": "執行",
  "vars.run": "Ausführen",
  "vars.run": "Exécuter",
  "vars.run": "Ejecutar",
  "vars.run": "Executar",
  "vars.run": "Выполнить",
```

**`vars.preview`** (추가)

```json
  "vars.preview": "입력될 내용",
  "vars.preview": "Text to be typed",
  "vars.preview": "入力される内容",
  "vars.preview": "将输入的内容",
  "vars.preview": "將輸入的內容",
  "vars.preview": "Einzugebender Text",
  "vars.preview": "Texte qui sera saisi",
  "vars.preview": "Texto que se escribirá",
  "vars.preview": "Texto que será digitado",
  "vars.preview": "Будет введено",
```

## F. 명세

원본 앱에 없던 기능이라 명세를 이 작업에서 새로 썼다(아래 조각의 `+` 줄이 새 명세다). 줄이 길어 가로로 넘치면 파일을 직접 여는 편이 읽기 쉽다.

### `spec/acceptance/04-hosts-sftp-settings-sync.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/acceptance/04-hosts-sftp-settings-sync.md
+++ b/spec/acceptance/04-hosts-sftp-settings-sync.md
@@ -31,6 +34,10 @@
 | F9 | 파일 업로드 완료 | | 현재 폴더 목록이 자동 갱신 |
 | F10 | 스니펫 저장, 명령 비움 | | [저장] 비활성 |
 | F11 | 스니펫 클릭, 동시 입력 탭 | | 모든 칸에 본문+Enter 입력 |
+| F12 | 본문 `systemctl restart {{서비스}} {{옵션:-q}}` 인 명령 클릭 | 값 창에서 서비스=`nginx`, [실행] | 값 창이 먼저 뜨고(옵션 칸에 `-q`), `systemctl restart nginx -q`+Enter 가 입력된다. `{{.Names}}` 는 변수가 아니다 (D138) |
+| F13 | [새 매크로]: 1단계 `cd /var/log`, 2단계 기다릴 글자 `$ `·보낼 글자 `tail -f app.log`, 저장 후 클릭 | 1단계 뒤 프롬프트 출력 | 목록에 `매크로 2단계` 배지. 1단계가 바로 입력되고, `$ ` 가 나온 뒤 2단계가 입력된다. 도는 동안 `실행 중 2/2` 와 [매크로 중지] (D141) |
+| F13a | 명령 하나와 매크로 하나가 있는 목록 | 왼쪽 위 `<>` 아이콘에 마우스를 올림 | 툴팁 "매크로&명령 모음". 명령 항목에는 초록 배지 `명령`, 매크로 항목에는 강조색 배지 `매크로 N단계` — 한 항목에 둘 다 붙지 않는다 (D143) |
+| F14 | [매크로 녹화] → 창에 `uptime`⏎ `df -h`⏎ 입력 → 정지 | | 상태 표시줄에 녹화 배지. 정지하면 두 단계가 든 "새 매크로" 편집 창(경고 박스 있음) (D141) |
 
 ## 설정 (ui/settings.md)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/behavior/terminal.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/behavior/terminal.md
+++ b/spec/behavior/terminal.md
@@ -88,6 +88,17 @@
 - 대상: 현재 포커스된 칸의 세션(동시 입력 규칙 적용). 포커스된 세션이 없으면 정보 토스트 "먼저 세션을 선택하세요".
 - 본문의 모든 줄바꿈을 CR 로 바꿔 보내고, "입력하고 바로 실행"이 켜져 있으면 끝에 CR 을 더한다. 입력 후 터미널에 포커스.
 
+### 7.1 변수 (2026-11-27, D138)
+
+원본 앱에 없던 기능이다. 한 명령을 값만 바꿔 여러 번 쓰게 한다.
+
+- 본문의 `{{이름}}` 과 `{{이름:기본값}}` 이 변수다. 이름은 글자·숫자·밑줄로 시작하고 그 뒤로 글자·숫자·밑줄·빈칸·하이픈이 올 수 있으며(한글 포함, 최대 40자) 앞뒤 빈칸은 뺀다. 기본값은 `:` 뒤부터 `}}` 앞까지(줄바꿈·`}` 제외).
+- **변수가 아닌 것**: 점으로 시작하는 `{{.Names}}`(docker·kubectl 의 출력 형식)와, 앞에 역슬래시를 붙인 `\{{이름}}`(보낼 때 `{{이름}}` 그대로).
+- 변수가 하나라도 있으면 항목을 눌렀을 때 바로 보내지 않고 **값을 묻는 창**(ui/sftp-snippets.md §4.3)을 띄운다. 같은 이름은 한 번만 묻고(나온 순서대로), 기본값은 처음 적힌 것을 쓴다. 포커스된 세션이 없으면 묻기 전에 "먼저 세션을 선택하세요".
+- 칸에는 **지난번에 넣은 값**이 먼저, 없으면 기본값이 채워져 있다. 지난 값은 앱이 켜져 있는 동안만 기억하고 파일에 남기지 않는다(비밀번호 같은 값이 남지 않게).
+- [실행]: 변수 자리를 값으로 바꿔 §7 대로 보낸다. 값 안의 `{{…}}` 는 다시 풀지 않는다. 칸을 비우면 빈 글자가 들어간다(기본값으로 돌아가지 않는다).
+- 매크로(§11)의 보낼 글자에도 같은 변수를 쓴다(모든 단계에 나온 변수를 합쳐 한 번에 묻는다).
+
 ## 8. OSC 52 (원격 → 로컬 클립보드)
 
 - 설정 "서버의 클립보드 쓰기 허용 (OSC 52)"이 꺼져 있으면 모두 무시.
```

### `spec/data/store.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -155,6 +157,9 @@
 
 id, name, body, sendEnter(불리언).
 
+- **변수**(2026-11-27, D138): body 안의 `{{이름}}` · `{{이름:기본값}}` 은 실행할 때 물어 채운다. 저장 형식은 그대로다(글자일 뿐) — behavior/terminal.md §7.1.
+- **매크로**(2026-11-27, D141): 항목에 `steps`(배열)가 있고 하나 이상이면 그 항목은 매크로다. 단계: id, expect(보내기 전에 기다릴 글자, 비면 출력이 조용해질 때까지), send(보낼 글자, 제어 문자는 `\x03` 처럼 적음), enter(Enter 붙이기). 최대 50단계. 매크로의 body 에는 단계의 send 를 줄마다 이어 적어 둔다(검색과, `steps` 를 모르는 옛 버전이 평범한 명령으로 보이게 하는 용도) — behavior/terminal.md §11.
+
 ## 8. knownHosts[]
 
 hostPort(`host:port`), keyType, fingerprint(`SHA256:` + base64), addedAt(밀리초).
```

### `spec/ui/sftp-snippets.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/ui/sftp-snippets.md
+++ b/spec/ui/sftp-snippets.md
@@ -90,3 +92,31 @@
 - 삭제 확인: 제목 "명령 삭제", 본문 `"{이름}"을(를) 삭제할까요?`, 확인 "삭제", 위험.
 - **편집 대화상자**(작게): 제목 새로 만들 때 "새 명령", 편집 시 "명령 수정". 필드 "이름"(플레이스홀더 "예: 로그 보기", 포커스), "명령"(5줄, 도움말 "여러 줄은 순서대로 입력됩니다", 플레이스홀더 `tail -f /var/log/nginx/error.log`), 토글 "입력하고 바로 실행". 새 스니펫 기본값: 이름·본문 빈 값, 입력 후 Enter 실행 켜짐. [저장]은 **명령이 비어 있으면 비활성**. 저장하면 목록에 추가(편집이면 교체) 후 닫는다.
 
+### 4.3 변수 값을 묻는 창 (작게, 2026-11-27, D138)
+
+동작은 behavior/terminal.md §7.1. 변수가 있는 명령·매크로를 눌렀을 때 뜬다.
+
+- 제목: 항목 이름(비면 "(이름 없음)").
+- 변수마다 필드 하나(나온 순서대로): 라벨 = 변수 이름, 고정폭 입력, 플레이스홀더 = 기본값. 첫 칸에 포커스. 지난번 값(없으면 기본값)이 채워져 있다.
+- 그 아래 필드 "입력될 내용": 지금 값으로 채운 명령을 고정폭 상자에 보인다(값을 바꾸면 바로 바뀐다, 길면 140px 에서 스크롤). 매크로는 단계의 보낼 글자를 줄마다.
+- 버튼 [취소] [실행](주 버튼). 입력 칸에서 Enter 도 실행. Esc·바깥 클릭은 취소(아무것도 보내지 않는다).
+- 명령 편집 대화상자의 "명령" 도움말에 변수 쓰는 법을 덧붙인다: "여러 줄은 순서대로 입력됩니다 · {{host}} 처럼 적으면 실행할 때 값을 묻습니다 (기본값은 {{port:22}})".
+
+### 4.4 매크로 (2026-11-27, D141)
+
+동작은 behavior/terminal.md §11.
+
+- **종류 배지**(2026-11-27 사용자 지시, D143): 목록의 **모든 항목** 이름 오른쪽에 작은 배지를 둔다 — 매크로는 `매크로 {n}단계`(강조색 글자 + 강조색 옅은 바탕), 명령은 `명령`(초록 `--ok` 글자 + 초록 옅은 바탕). 모양은 같고 색만 다르다.
+- **패널 이름**: 왼쪽 아이콘 묶음의 버튼 툴팁과 패널의 이름은 **"매크로&명령 모음"** 이다(D143, 전에는 "명령 모음").
+- **목록**: 매크로 항목은 이름 오른쪽에 작은 배지 `매크로 {n}단계`(강조색 옅은 바탕), 아래 미리보기는 단계의 보낼 글자를 ` ⏎ ` 로 이은 것. 툴팁 "누르면 지금 창에서 매크로를 실행합니다". 검색은 명령과 같다(이름 + 본문).
+- **실행 중**: 그 매크로가 어느 세션에서든 돌고 있으면 미리보기 자리에 `실행 중 {지금 단계}/{전체}` 를 보이고, 오른쪽의 [수정]·[삭제] 대신 **[매크로 중지]**(네모 아이콘)를 둔다.
+- **[새 매크로]**(도구 막대): 빈 단계 하나가 있는 편집 창을 연다.
+- **[매크로 녹화]**(도구 막대, 동그라미 아이콘): 툴팁 "매크로 녹화 — 지금 창에 치는 입력을 기록합니다". 누르면 포커스된 세션의 녹화를 시작하고 정보 토스트 "매크로 녹화 시작 — 지금 창에 치는 입력을 기록합니다. 비밀번호는 치지 마세요." 포커스된 세션이 없으면 "먼저 세션을 선택하세요". 녹화 중에는 버튼이 **네모 아이콘 + 위험색 옅은 바탕**으로 바뀌고 툴팁 "녹화 정지 ({n}단계)". 다시 누르면 정지하고 편집 창이 열린다. 상태 표시줄의 배지(ui/shell.md §3)와 빠른 접속의 명령(ui/dialogs.md §2.1)으로도 시작·정지한다.
+- **편집 창**(기본 크기, 바깥 클릭으로 닫히지 않는다): 제목 "새 매크로" / "매크로 수정".
+  - 녹화로 연 경우 맨 위 경고 박스 "녹화한 입력입니다. 비밀번호를 쳤다면 그 단계를 지우세요 — 매크로는 암호화되지 않고 저장됩니다."
+  - 필드 "이름"(플레이스홀더 "예: 배포 후 로그 확인", 포커스).
+  - 필드 "단계": 표 — 머리 `#` · 기다릴 글자 · 보낼 글자 · Enter · (버튼). 행마다 번호, "기다릴 글자"(고정폭, 플레이스홀더 "(비우면 출력이 멈출 때까지)"), "보낼 글자"(고정폭), 체크 Enter, [위로] [아래로] [삭제]. 아래 [+ 단계 추가](50단계면 비활성).
+  - 도움말: "위에서 아래로 실행합니다. 기다릴 글자가 있으면 출력에 그 글자가 나온 뒤에 보내고(30초 안에 안 나오면 멈춤), 비우면 출력이 0.5초 동안 조용해진 뒤에 보냅니다. 첫 단계는 기다릴 글자가 없으면 바로 보냅니다. {{host}} 처럼 적으면 실행할 때 값을 묻고, 제어 문자는 \x03(Ctrl+C)처럼 적습니다. 매크로는 지금 창 하나에서만 실행됩니다."
+  - [저장]은 보낼 것이 있는 단계(글자 또는 Enter)가 하나도 없으면 비활성. 저장하면 보낼 것이 없는 단계는 빠진다.
+- 삭제 확인의 제목은 "매크로 삭제"(본문·버튼은 명령과 같다).
+
```

## G. 검증 (2026-11-27 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [snippetVars.test.ts](../../frontend/src/lib/snippetVars.test.ts) `parseVars` 5개 | 나온 순서·같은 이름 한 번 / 한글 이름·빈칸·하이픈, 앞뒤 빈칸 제거 / **`{{.Names}}` 와 `\{{…}}` 는 변수가 아님** / 기본값은 처음 적힌 것 / 여러 본문 합치기(매크로용) |
| 같은 파일 `fillVars` 4개 | 값으로 바꾸고 없으면 기본값 / 빈 값을 넣으면 빈 글자(기본값으로 돌아가지 않음) / 변수가 아닌 것은 그대로, `\{{` 는 `{{` 로 / **값 안의 `{{…}}` 를 다시 풀지 않음** |
| 같은 파일 "마지막 값 기억" 1개 | 지난 값이 먼저, 다른 명령에는 섞이지 않음 |
| [Automation.test.tsx](../../frontend/src/screens/Automation.test.tsx) "명령 모음 변수" 3개 | 누르면 바로 보내지 않고 값 창이 뜨며 기본값이 채워져 있다, 미리 보기가 값에 따라 바뀐다, [실행]하면 채운 명령 + Enter 가 간다 / 변수가 없으면 묻지 않는다 / 다시 누르면 지난 값이 채워져 있다 |
| 기존 `RightPanel.test.tsx` F10·F11 | [새 명령] 버튼·저장 비활성·일괄 입력이 그대로 동작(패널을 다시 썼어도) |

**자동 테스트가 확인하지 못하는 것**: 값 창에서 Enter 로 실행되는 것(보이지 않는 제출 버튼), 창이 닫힌 뒤 터미널에 포커스가 돌아가는 것.

전체 실행 결과(2026-11-27, 다섯 문서와 추가 지시 D142·D143 을 모두 적용한 상태):

```bash
go vet ./...                              # 경고 없음
go test ./...                             # 19개 패키지 모두 ok
(cd frontend && npx tsc --noEmit)         # 오류 없음
(cd frontend && npm test)                 # PowerShell 에서. 61개 파일 439개 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

**`-race` 검사 (같은 날 추가)**: 처음에는 이 PC 에 gcc 가 없어 돌리지 못했다(`-race requires cgo`). 같은 날 gcc 를 설치하고(D144, [2026-11-27-6-gcc-race-test.md](./2026-11-27-6-gcc-race-test.md)) `go test -race ./...` 를 돌려 19개 패키지 모두 통과했다 — 경쟁 보고 없음. 이 문서의 변경은 프런트만이라 Go 경쟁 검사와 직접 관계는 없다.

```
(frontend) npm test      # snippetVars.test.ts, Automation.test.tsx "명령 모음 변수"
```

## H. 수동 검증 (앱에서)

| # | 할 일 | 기대 |
|---|---|---|
| 1 | 명령 `ssh {{user:root}}@{{host}}` 를 저장하고 누름 | 값 창이 뜬다. user 칸에 `root` 가 채워져 있고 첫 칸(user)에 포커스 |
| 2 | host 에 값을 넣는 동안 | "입력될 내용"이 바로 바뀐다 |
| 3 | Enter | 채운 명령이 지금 창에 입력되고 터미널에 포커스 |
| 4 | 같은 명령을 다시 누름 | 지난번 값이 채워져 있다. 앱을 다시 켜면 기본값으로 돌아온다 |
| 5 | Esc 또는 [취소] | 아무것도 입력되지 않는다 |
| 6 | `docker ps --format "{{.Names}}"` 를 저장하고 누름 | 묻지 않고 그대로 입력된다 |
| 7 | 일괄 입력을 켠 탭에서 변수 명령을 누름 | 한 번 묻고 모든 창에 같은 명령 |
| 8 | 연결된 창이 없을 때 누름 | 값 창이 뜨지 않고 "먼저 세션을 선택하세요" |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 이미 저장해 둔 명령에 `{{…}}` 가 있는 경우 | 전에는 글자 그대로 입력됐는데 이제 값을 묻는다 | 점으로 시작하는 출력 형식(`{{.Names}}`)은 변수가 아니다. 그 밖(예: Jinja `{{ var }}`)은 앞에 `\` 를 붙여 피한다. 설명서·편집 창 도움말에 적었다 |
| 옛 버전 앱·git 동기화 | 옛 버전은 `{{…}}` 를 글자 그대로 입력한다 | 저장 형식을 바꾸지 않았으므로 깨지지는 않는다 |
| 문구의 자리표시자 | 도움말에 `{{host}}` 를 직접 적으면 문구의 `{이름}` 자리표시자로 읽혀 언어 간 검사에 걸린다 | 예시를 자리표시자 값으로 넘긴다(`{a}`, `{b}`) |
| 값에 든 특수 글자 | `$1`·`$&` 같은 글자가 바꾸기 규칙으로 풀릴 수 있다 | 바꾸기 함수를 쓴다(글자 그대로 들어간다) |
| 지난 값 | 메모리에만 둔다(앱을 끄면 사라짐) | 파일·동기화에 남지 않는다 |
| 백엔드 | 바꾸지 않았다 | — |

## J. 체크리스트

- [x] `lib/snippetVars.ts`(찾기·채우기·지난 값), `screens/VarsDialog.tsx`(값 창 + 미리 보기)
- [x] `SnippetsPanel.tsx`: 누르면 변수가 있는지 보고 값 창을 띄움, 편집 창 도움말
- [x] 문구 3개(10개 언어), 스타일 `.vars-preview`
- [x] 명세 behavior/terminal.md §7.1 · ui/sftp-snippets.md §4.3 · data/store.md §7 · 수용 F12, decisions D138
- [x] 자동 테스트 13개 추가, 전체 통과
- [ ] 수동 검증 1~8 (실제 화면)
