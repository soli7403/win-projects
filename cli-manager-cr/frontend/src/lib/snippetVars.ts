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
