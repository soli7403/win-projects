// 왼쪽 열이 안의 내용 때문에 넓어지지 않게 하는 스타일 규칙 (D130, spec/ui/sidebar.md §1).
// 화면 테스트(jsdom)는 배치를 계산하지 않으므로, 규칙이 스타일 파일에 있는지를 파일에서 확인한다. 동작은 screens/Shell.leftcol.test.tsx.
import { describe, expect, it } from 'vitest';
// 테스트 실행기는 가져온 css 를 빈 내용으로 바꾸므로(?raw 도) 파일을 직접 읽는다. 이 프로젝트에는 node 타입이 없다.
// @ts-expect-error node:fs 의 타입 선언이 없다
import { readFileSync } from 'node:fs';

/** 이 파일 옆의 스타일 파일을 글자로 읽는다(줄 끝은 LF 로). */
function readStyle(name: string): string {
  const path = decodeURIComponent(new URL(name, import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
  return String(readFileSync(path, 'utf8')).replace(/\r\n/g, '\n');
}

const APP_CSS = readStyle('./app.css');

/** 스타일 파일에서 선택자 하나의 선언 글(중괄호 안)을 모두 이어 돌려준다. */
function cssOf(selector: string): string {
  const esc = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return [...APP_CSS.matchAll(new RegExp(`(?:^|\\n)${esc}\\s*\\{([^}]*)\\}`, 'g'))].map((m) => m[1]).join(' ');
}

describe('왼쪽 열 스타일 (D130)', () => {
  it('왼쪽 열은 줄지도 늘지도 않고(flex: none) 손잡이의 기준이며, 오른쪽 테두리를 갖는다', () => {
    const css = cssOf('.leftcol');
    expect(css).toMatch(/flex:\s*none/);
    expect(css).toMatch(/position:\s*relative/);
    expect(css).toMatch(/border-right:/);
  });

  it('명령 모음·서버 목록은 내용이 길어도 열을 넓히지 않는다(min-width: 0, 명령 모음은 넘치는 것을 자른다)', () => {
    expect(cssOf('.snippet-dock')).toMatch(/min-width:\s*0/);
    expect(cssOf('.snippet-dock')).toMatch(/overflow:\s*hidden/);
    expect(cssOf('.leftcol-body')).toMatch(/min-width:\s*0/);
    expect(cssOf('.sidebar')).toMatch(/min-width:\s*0/);
  });

  it('열 안의 서버 목록은 오른쪽 테두리를 따로 그리지 않는다(테두리가 열 가운데에 서지 않는다)', () => {
    expect(cssOf('.leftcol-body > .sidebar')).toMatch(/border-right:\s*none/);
    expect(cssOf('.sidebar')).not.toMatch(/(^|[;\s])width:/);
  });
});
