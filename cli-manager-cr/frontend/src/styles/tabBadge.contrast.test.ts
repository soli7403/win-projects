// 탭의 창 수 배지는 어느 테마·어느 탭 배경에서도 숫자가 읽혀야 한다 (D122, spec/ui/tokens.md §3 "탭 칸 수 배지").
// 스타일 파일의 실제 값으로 명도 대비(WCAG)를 계산한다: 다크·라이트 × 탭 배경(막대 바탕·선택 탭) × 세션 색 배경(없음·7%·14%).
import { describe, expect, it } from 'vitest';
// 테스트 실행기는 가져온 css 를 빈 내용으로 바꾸므로(?raw 도) 파일을 직접 읽는다. 이 프로젝트에는 node 타입이 없다.
// @ts-expect-error node:fs 의 타입 선언이 없다
import { readFileSync } from 'node:fs';
import { COLOR_PALETTE } from '../lib/defaults';

/** 이 파일 옆의 스타일 파일을 글자로 읽는다. */
function readStyle(name: string): string {
  const path = decodeURIComponent(new URL(name, import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
  return String(readFileSync(path, 'utf8'));
}

const appCss = readStyle('./app.css');
const tokensCss = readStyle('./tokens.css');

type Rgb = [number, number, number];

function hex(s: string): Rgb {
  const m = /^#([0-9a-f]{6})$/i.exec(s.trim());
  if (!m) throw new Error(`색 형식: ${s}`);
  const n = parseInt(m[1], 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

/** a 를 pct% 만큼 b 위에 섞는다(color-mix(in srgb, a pct%, transparent) 를 b 위에 겹친 결과와 같다). */
function over(a: Rgb, pct: number, b: Rgb): Rgb {
  const k = pct / 100;
  return [0, 1, 2].map((i) => a[i] * k + b[i] * (1 - k)) as Rgb;
}

function luminance(c: Rgb): number {
  const [r, g, b] = c.map((v) => {
    const x = v / 255;
    return x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function contrast(a: Rgb, b: Rgb): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

/** tokens.css 의 한 블록(:root 또는 라이트)에서 색 토큰을 읽는다. */
function tokens(block: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const m of block.matchAll(/(--[a-z0-9-]+):\s*(#[0-9a-f]{6})\s*;/gi)) out[m[1]] = m[2];
  return out;
}

const lightAt = tokensCss.indexOf(":root[data-theme='light']");
const DARK = tokens(tokensCss.slice(0, lightAt));
const LIGHT = { ...DARK, ...tokens(tokensCss.slice(lightAt)) };

const rule = /\.tab-badge\s*\{([^}]*)\}/.exec(appCss)?.[1] ?? '';

describe('탭의 창 수 배지 (D122)', () => {
  it('글자는 기본 글자색, 바탕은 그 색을 섞은 반투명이다(흐린 글자·불투명 면 배경을 쓰지 않는다)', () => {
    expect(rule).toMatch(/color:\s*var\(--fg\)\s*;/);
    expect(rule).toMatch(/background:\s*color-mix\(in srgb,\s*var\(--fg\)\s*\d+%,\s*transparent\)/);
    expect(rule).not.toContain('--fg-3');
    expect(rule).not.toContain('--panel-2');
    expect(rule).toMatch(/font-weight:\s*600/);
    expect(rule).toMatch(/flex:\s*none/);
  });

  it('다크·라이트, 선택·비선택, 세션 색 배경 어디서나 숫자와 바탕의 대비가 4.5:1 이상이다', () => {
    const pct = Number(/var\(--fg\)\s*(\d+)%/.exec(rule)![1]);
    const worst: { ratio: number; where: string } = { ratio: Infinity, where: '' };
    for (const [themeName, tk] of [['dark', DARK], ['light', LIGHT]] as const) {
      const fg = hex(tk['--fg']);
      // 탭이 놓이는 바탕: 제목 줄(--bg), 열 안 탭 막대와 선택된 탭(--panel).
      for (const baseName of ['--bg', '--panel']) {
        const base = hex(tk[baseName]);
        // 세션 색 배경: 없음, 비선택 7%, 선택 14% (TabBar.tsx 의 TINT_INACTIVE·TINT_ACTIVE). 색이 없는 선택 탭은 강조색.
        for (const color of [null, tk['--accent'], ...COLOR_PALETTE]) {
          for (const tint of color ? [7, 14] : [0]) {
            const tabBg = color ? over(hex(color), tint, base) : base;
            const badgeBg = over(fg, pct, tabBg);
            const ratio = contrast(fg, badgeBg);
            if (ratio < worst.ratio) Object.assign(worst, { ratio, where: `${themeName} ${baseName} ${color ?? '색 없음'} ${tint}%` });
          }
        }
      }
    }
    expect(worst.ratio, `가장 낮은 곳: ${worst.where}`).toBeGreaterThanOrEqual(4.5);
  });

  it('배지가 탭 배경과 구분된다(바탕과 탭 배경의 대비 1.15:1 이상)', () => {
    const pct = Number(/var\(--fg\)\s*(\d+)%/.exec(rule)![1]);
    for (const tk of [DARK, LIGHT]) {
      for (const baseName of ['--bg', '--panel']) {
        const base = hex(tk[baseName]);
        expect(contrast(over(hex(tk['--fg']), pct, base), base)).toBeGreaterThanOrEqual(1.15);
      }
    }
  });

  it('고치기 전 값(흐린 글자 + 면 배경 2)은 두 테마 모두 4.5:1 에 못 미쳤다 — 되돌아가지 않게 남겨 둔다', () => {
    for (const tk of [DARK, LIGHT]) {
      expect(contrast(hex(tk['--fg-3']), hex(tk['--panel-2']))).toBeLessThan(4.5);
    }
  });
});
