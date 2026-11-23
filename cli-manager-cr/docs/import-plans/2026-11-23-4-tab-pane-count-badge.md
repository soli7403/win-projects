# 탭의 창 수 배지(숫자)가 잘 보이지 않는 문제 — 원인·수정·코드·검증

**작성**: 2026-11-23 · **상태**: 코드 수정·자동 테스트·빌드 완료(D122). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(보고 즉시 수정해 등록하지 않음)
**관련**: [spec/ui/tokens.md](../../spec/ui/tokens.md) §3 "탭 칸 수 배지", [spec/ui/shell.md](../../spec/ui/shell.md) §2.1(탭의 구성), 탭의 세션 색 배경은 [2026-11-20-active-tab-color-bar.md](./2026-11-20-active-tab-color-bar.md)(D103)

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

한 탭에 창(세션)이 여러 개 있으면 탭 제목 옆에 창 수가 작은 알약 모양 배지로 나온다. 이 **숫자가 배경에 묻혀 잘 보이지 않았다.** 흐린 글자색을 어두운 면 배경 위에 썼고, 그 면 배경이 탭 배경과 거의 같은 색이었기 때문이다. 글자를 기본 글자색·굵게로, 바탕을 "글자색을 18% 섞은 반투명"으로 바꿨다.

| | 고치기 전 | 고친 뒤 |
|---|---|---|
| 글자 | 흐린 글자색(`--fg-3`), 보통 굵기 | **기본 글자색(`--fg`), 굵게(600)** |
| 바탕 | 면 배경 2(`--panel-2`, 불투명) | **기본 글자색 18% 반투명** — 탭 배경 위에 겹친다 |
| 숫자와 바탕의 명도 대비 | 다크 **3.49:1**, 라이트 **2.71:1** | 다크 **6.1~8.9:1**, 라이트 **8.4~10.7:1** |
| 배지와 탭 배경의 대비(알약이 보이는 정도) | 다크 1.10:1, 라이트 1.14:1 (세션 색 배경 위에서는 라이트 1.01:1 — 사실상 안 보임) | 다크 1.59:1, 라이트 1.43:1 |

읽기 쉬운 글자의 기준은 보통 4.5:1 이다(WCAG AA). 전에는 두 테마 모두 못 미쳤다.

## 1. 사용자 보고

2026-11-23, 스크린샷과 함께: "탭에 세션이 여러 개 있는 경우, 숫자가 잘 표시되지 않는다. 여러 테마에서도 숫자가 잘 보이도록 수정."

스크린샷: 다크 테마, 초록 세션 색이 깔린 선택된 탭 `DC1_MSP01(215)` 옆의 `2` 가 어두운 회색 알약 안에 흐리게 보인다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 창 수 배지 | 탭에 창이 2개 이상일 때 제목 옆에 나오는 숫자. 코드의 `.tab-badge` |
| 세션 색 배경 | 탭 배경에 옅게 깔리는 세션의 색(서버·폴더·환경 색). 선택된 탭은 14%, 선택되지 않은 탭은 7%(D103) |
| 앱 테마 | 다크·라이트 두 가지. 탭 막대의 색은 앱 테마를 따른다. 터미널 색상(DariTerm Dark, Solarized 등 6종)은 터미널 안쪽만 바꾸고 탭 막대에는 영향이 없다 |
| 명도 대비 | 두 색의 밝기 차이를 1:1(같음)~21:1(검정·흰색)로 나타낸 값 |

## 2. 원인 분석 (Cause)

고치기 전 스타일([frontend/src/styles/app.css](../../frontend/src/styles/app.css)):

```css
.tab-badge { font-size: 11px; color: var(--fg-3); background: var(--panel-2); border-radius: 8px; padding: 0 5px; }
```

| # | 문제 | 왜 | 수치 |
|---|---|---|---|
| 1 | **글자색이 가장 흐린 단계다** | `--fg-3` 은 자리표시자·비활성 안내에 쓰는 색이다. 11px 숫자 한 글자에 쓰기에는 너무 흐리다 | 숫자 ↔ 바탕: 다크 `#6b7589` on `#1c2030` = 3.49:1, 라이트 `#8a93a3` on `#eef0f4` = 2.71:1 |
| 2 | **바탕이 탭 배경과 거의 같다** | `--panel-2` 는 선택된 탭의 배경 `--panel` 보다 한 단계 밝을(어두울) 뿐이다. 알약 모양이 보이지 않아 숫자만 떠 있다 | 바탕 ↔ 탭 배경: 다크 1.10:1, 라이트 1.14:1 |
| 3 | **바탕이 불투명한 고정 색이다** | 탭에는 세션 색이 옅게 깔리는데(D103) 배지는 그 위에 회색 덩어리로 얹힌다. 색이 깔린 탭에서는 배지 바탕이 탭 배경보다 어둡거나 같아진다 | 초록 14% 배경 위: 다크 1.15:1, 라이트 1.01:1 |
| 4 | **굵기가 보통이다** | 탭 제목은 12.5px 인데 배지는 11px 보통 굵기라 획이 가늘다 | — |

명세([spec/ui/tokens.md](../../spec/ui/tokens.md) §3)가 "11px, 흐린 글자, 면 배경 2, 반경 8" 이라고 적고 있었고 구현은 그대로 따랐다. 세션 색 배경(D103)이 들어오기 전에 정한 값이다. 이번에 사용자 지시로 명세를 고쳤다(§F).

## 3. 수정 방안과 고른 것 (Fix Plan)

| 안 | 내용 | 판단 |
|---|---|---|
| **A. 글자색 기반 반투명 바탕 + 기본 글자색** | 글자 `--fg`, 바탕 `color-mix(in srgb, var(--fg) 18%, transparent)` | **고름.** 바탕이 탭 배경 위에 겹치므로 탭 배경이 무엇이든(선택·비선택, 세션 색, 다크·라이트) 늘 "배경보다 글자색 쪽으로 18% 다가간 색"이 된다. 테마가 늘어도 토큰만 있으면 따라온다 |
| B. 강조색 채움 + 흰 글자 | 알림 숫자처럼 파란 알약 | 눈에 가장 잘 띄지만 "읽지 않은 알림"처럼 읽힌다. 세션 색(빨강·초록 등) 배경 위에 파랑이 얹혀 색이 부딪친다 |
| C. 글자색만 `--fg-2` 로 올린다 | 한 줄 수정 | 다크 6.5:1 로 오르지만 알약이 안 보이는 문제(2·3번)가 남는다 |

### 3.1 세부 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 섞는 비율 | 18% | 가장 불리한 조합(다크, 노랑 14% 배경)에서도 숫자 대비 6.1:1 이고 알약이 탭 배경과 1.4:1 이상 차이 난다. 더 올리면 알약은 뚜렷해지지만 숫자 대비가 떨어진다 |
| 글자색 | 선택되지 않은 탭에서도 `--fg` | 선택되지 않은 탭의 제목은 `--fg-2` 다. 배지까지 흐리면 고치기 전과 같아진다 |
| 굵기 | 600 | 11px 에서 획이 또렷하다 |
| 크기 | 글자 11px 그대로, 높이 16px·최소 폭 16px, 숫자 가운데 | 한 자리 숫자도 찌그러지지 않은 알약이 된다. 탭 높이(32px)는 그대로 |
| 숫자 폭 | `tabular-nums` | 창 수가 바뀔 때 배지 폭이 흔들리지 않는다 |
| 줄어들지 않기 | `flex: none` | 제목이 길어 탭이 꽉 차도 배지가 눌리지 않는다(제목이 줄임표로 줄어든다) |
| 마크업 | 그대로(`<span className="tab-badge">`) | 스타일만의 문제다 |

### 3.2 영향 범위

스타일 파일의 규칙 하나와 명세 한 줄. 코드(TSX·Go)·문구·저장 형식은 바뀌지 않는다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/styles/app.css` | 스타일 | 수정 |
| `frontend/src/styles/tabBadge.contrast.test.ts` | 테스트 | 새 파일 |
| `spec/ui/tokens.md` | 명세 | 수정 |
| `docs/decisions.md` | 문서 | D122 추가 |

아래의 `diff` 는 **같은 날의 앞 문서들(D106~D121)을 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `frontend/src/styles/app.css` — 수정

```diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -328,7 +328,13 @@
 }
 .empty-pane { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; text-align: center; color: var(--fg-2); }
 .tab.drop { outline: 1px dashed var(--accent); }
-.tab-badge { font-size: 11px; color: var(--fg-3); background: var(--panel-2); border-radius: 8px; padding: 0 5px; }
+/* 탭의 창 수 배지 (D122): 기본 글자색 + 그 글자색을 18% 섞은 반투명 바탕.
+   바탕을 탭 배경 위에 겹치므로 선택·비선택, 세션 색 배경, 다크·라이트 어디서나 글자와 알약 모양이 보인다. */
+.tab-badge {
+  flex: none; min-width: 16px; height: 16px; line-height: 16px; box-sizing: border-box; padding: 0 5px;
+  border-radius: 8px; text-align: center; font-size: 11px; font-weight: 600; font-variant-numeric: tabular-nums;
+  color: var(--fg); background: color-mix(in srgb, var(--fg) 18%, transparent);
+}
 .tab .dot { margin-right: 2px; }
 .dot.reconnecting { background: var(--warn); }
 .dot.error { background: var(--danger); }
```


## C. 테스트 코드

### `frontend/src/styles/tabBadge.contrast.test.ts` — 새 파일

```ts
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
```


## F. 명세

### `spec/ui/tokens.md` — 수정

```diff
--- a/spec/ui/tokens.md
+++ b/spec/ui/tokens.md
@@ -44,7 +44,7 @@
 | 제목 표시줄 높이 | 38px (아래 테두리 1px). 오른쪽에 기본 창 컨트롤용 여백 약 140px |
 | 탭 | 높이 32, 최소 폭 120 / 최대 폭 220, 위쪽 모서리 반경 8, 안쪽 여백 좌 12 우 8, 항목 간격 7. 활성 탭은 면 배경 1 + 테두리, 아래 선을 지워 본문과 이어 보이게 함. 색 막대 높이 3(탭 위쪽, 선택된 탭에만 — ui/shell.md §2.1). 탭 사이 간격 2, 탭 줄 위 여백 5 |
 | 탭 닫기 버튼 | 20×20, 평소 투명, 탭에 마우스를 올리거나 활성이면 0.7 투명도 |
-| 탭 칸 수 배지 | 11px, 흐린 글자, 면 배경 2, 반경 8 |
+| 탭 칸 수 배지 | 11px 굵게(600), **기본 글자색**, 배경은 **기본 글자색을 18% 섞은 반투명**(탭 배경 위에 겹침), 높이 16·최소 폭 16, 반경 8, 숫자 가운데. 줄어들지 않는다. 선택·비선택 탭, 세션 색 배경, 다크·라이트 모두에서 숫자와 배경의 명도 대비가 4.5:1 이상이어야 한다 (2026-11-23 사용자 지시, D122. 전에는 흐린 글자 + 면 배경 2 여서 숫자가 잘 보이지 않았다) |
 | 상태 점 | 8px 원. 연결됨=성공색(+2px 옅은 후광), 접속 중/재접속=경고색 + 1초 주기 깜빡임(투명도 0.35까지), 끊김=흐린 글자색, 오류=위험색, 활동=강조색 |
 | 아이콘 버튼 | 28×28(작은 것 22×22), 반경 6(작은 것 5), 평소 보조 글자색, 호버 시 호버 배경, 켜짐(on)=강조 옅은 배경+강조색, 비활성 투명도 0.35 |
 | 버튼 | 높이 32(작은 것 26, 12px), 좌우 14(작은 것 10), 반경 7, 테두리 강한 테두리, 배경 면 배경 2, 글자 굵게 500. 주 버튼=강조색 채움+흰 글자, 위험 버튼=위험색 채움+흰 글자, 비활성 투명도 0.5 |
```

---

## G. 검증 (2026-11-23 실행)

새 테스트 파일 [frontend/src/styles/tabBadge.contrast.test.ts](../../frontend/src/styles/tabBadge.contrast.test.ts) (4개). 스타일 파일(`app.css`, `tokens.css`)을 직접 읽어 **실제 값으로** 명도 대비를 계산한다.

| 테스트 | 확인하는 것 |
|---|---|
| 글자는 기본 글자색, 바탕은 그 색을 섞은 반투명이다 | `.tab-badge` 규칙이 `color: var(--fg)`, `background: color-mix(in srgb, var(--fg) N%, transparent)`, 굵기 600, `flex: none` 을 쓰고 `--fg-3`·`--panel-2` 를 쓰지 않는다 |
| 어디서나 숫자와 바탕의 대비가 4.5:1 이상 | 다크·라이트 × 탭 바탕 2가지(제목 줄 `--bg`, 탭 막대·선택 탭 `--panel`) × 세션 색 15가지(팔레트 14색 + 강조색) × 배경 농도(7%·14%) + 색 없음 = **124가지 조합**의 최저값이 4.5 이상. 지금 최저는 6.14:1(다크, 노랑 `#eab308` 14%) |
| 배지가 탭 배경과 구분된다 | 배지 바탕과 탭 배경의 대비가 두 테마 모두 1.15:1 이상(지금 최저 1.42) |
| 고치기 전 값은 4.5:1 에 못 미쳤다 | `--fg-3` on `--panel-2` 가 두 테마 모두 4.5 미만 — 누가 예전 값으로 되돌리면 첫 번째 테스트와 함께 이유가 드러난다 |

조합별 수치(계산값):

| 테마 | 고치기 전(숫자 ↔ 바탕) | 고친 뒤 최저 | 고친 뒤 최고 |
|---|---|---|---|
| 다크 | 3.49:1 | 6.14:1 (선택 탭, 노랑 14%) | 8.86:1 (제목 줄, 색 없음) |
| 라이트 | 2.71:1 | 8.36:1 (제목 줄, 파랑 14%) | 10.74:1 (선택 탭, 색 없음) |

```bash
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 47개 파일 299개 통과 (앞 문서까지 46개 파일 295개 + 이 문서 4개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

Go 코드는 바뀌지 않아 `go test` 는 다시 돌리지 않았다.

테스트에 관한 메모: 테스트 실행기(vitest)는 가져온 CSS 를 빈 내용으로 바꾼다(`?raw` 로 가져와도). 그래서 `node:fs` 로 파일을 직접 읽는다. 이 프로젝트에는 node 타입 선언이 없어 그 줄에 `@ts-expect-error` 를 달았다.

자동 테스트가 확인하지 못하는 것: **눈으로 봤을 때 실제로 잘 보이는지**(이 수정의 목적). 계산은 색 값만 본다. 11px 굵은 숫자가 실제 글꼴에서 어떻게 그려지는지, 알약의 크기·자리가 탭 안에서 어울리는지는 화면에서 봐야 한다.

## H. 수동 검증 (남은 일)

준비: 한 탭에서 창 머리글의 [아래로 나누기]로 창을 2개 이상 만든다(탭 제목 옆에 숫자가 생긴다).

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 다크 테마, 선택된 탭 | 숫자가 또렷하고 알약 모양이 탭 배경과 구분된다 |
| 2 | 다크 테마, 다른 탭을 선택해 그 탭을 선택되지 않은 상태로 | 제목은 흐려지지만 숫자는 계속 또렷하다 |
| 3 | 설정 > 화면에서 라이트 테마로 바꿔 1·2 를 다시 | 숫자가 진한 글자, 알약이 옅은 회색으로 보인다 |
| 4 | 세션 색이 있는 서버(운영=빨강, 개발=초록, 노랑·파랑 등)로 1~3 을 다시 | 탭에 색이 깔려도 숫자와 알약이 보인다. 알약이 탭 색을 살짝 머금는다 |
| 5 | 화면을 좌우로 나눠 두 그룹의 탭 막대에서 본다 | 두 쪽 모두 같다 |
| 6 | 창을 3개, 그리고 설정의 한도를 5로 올려 5개까지 | 숫자가 바뀌어도 알약 폭이 흔들리지 않는다 |
| 7 | 탭 제목이 긴 서버(탭 최대 폭 220px 을 채움) | 제목이 줄임표로 줄고 배지는 눌리지 않는다. 닫기(X) 버튼과 겹치지 않는다 |
| 8 | 동시 입력을 켠 탭(경고색 아이콘이 배지 앞에 나온다) | 아이콘·배지·닫기 버튼이 나란히 보인다 |
| 9 | 터미널 색상을 Solarized·Monokai 등으로 바꾼다 | 탭 막대와 배지는 달라지지 않는다(앱 테마만 따른다) |
| 10 | 창이 하나뿐인 탭 | 배지가 없다(전과 같다) |

알약이 너무 진하거나 옅게 느껴지면 `app.css` 의 `18%` 한 곳만 바꾸면 된다(테스트가 대비 4.5:1 과 구분 1.15:1 을 지키는 범위를 알려 준다 — 대략 8%~29%).

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 탭의 폭·배치 | 배지에 높이 16px·최소 폭 16px·`flex: none` 이 생겼다. 전에는 글자 높이만큼(약 15px)이고 눌릴 수 있었다. 탭 높이는 32px 이라 넘치지 않는다 | 수동 7·8 |
| 다른 배지 | `.tab-badge` 는 탭 막대에서만 쓴다([TabBar.tsx](../../frontend/src/screens/TabBar.tsx)). 창 머리글의 "유지" 배지·상태 표시줄의 동시 입력 배지는 다른 규칙이다 | `grep` 으로 확인 |
| `color-mix` 지원 | 앱의 다른 곳(탭의 세션 색 배경, 아이콘 버튼 등)에서 이미 쓰고 있다 | — |
| 테마가 늘 때 | 새 테마가 `--fg` 와 배경 토큰을 정하면 배지는 따라온다. 대비가 낮은 테마(글자색과 배경이 가까운)는 테스트의 계산 대상에 넣어야 걸러진다 | 테스트는 지금 다크·라이트만 본다 |
| 스토어 스크린샷 | 탭 배지가 찍힌 스크린샷이 있으면 모양이 달라진다 | 확인하지 않았다 |

되돌리기: §B 의 `-` 줄로 되돌리고 테스트 파일을 지운다.

## J. 체크리스트

- [x] 원인 확인(흐린 글자 + 탭 배경과 거의 같은 불투명 바탕, 대비 다크 3.49·라이트 2.71)
- [x] `app.css` 의 `.tab-badge` 수정
- [x] 테스트 4개 추가(124가지 조합의 대비 계산), `tsc`·`vitest`(299개) 통과, 빌드
- [x] 명세 ui/tokens.md §3, decisions D122
- [ ] 수동 검증 1~10
