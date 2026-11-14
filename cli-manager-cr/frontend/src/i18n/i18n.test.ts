// 다국어 검사(D75): 언어 파일(locales/*.json)이 모두 있고 ko.json 과 같은 키·자리표시자를 갖는지,
// 화면 코드에 번역을 거치지 않은 한국어 문구가 남지 않았는지 확인한다.
import { describe, it, expect, afterEach } from 'vitest';
import { LANGS, LOCALES, matchLang, relabel, t, useLang } from './index';

const SOURCES = import.meta.glob('../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

/** 화면 코드: 테스트, i18n 자체, 개발용 가짜 백엔드, 생성물은 뺀다. */
function isUiSource(path: string): boolean {
  // glob 경로는 이 폴더 기준이라 i18n 자신은 './…' 로 온다.
  return !/\.test\.tsx?$/.test(path) && !path.startsWith('./') && !/fakeBackend|fakeSftp|\/test\/|\/generated\//.test(path);
}

const HANGUL = /[가-힣]/;

function placeholders(s: string): string {
  return [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');
}

afterEach(() => useLang.getState().setLang('ko'));

describe('언어 맞추기', () => {
  it('OS 언어 태그를 지원 언어로, 없으면 영어', () => {
    expect(matchLang(['ko-KR'])).toBe('ko');
    expect(matchLang(['zh-Hans-CN'])).toBe('zh');
    expect(matchLang(['zh-TW'])).toBe('tw');
    expect(matchLang(['zh-Hant-HK'])).toBe('tw');
    expect(matchLang(['pt-PT'])).toBe('pt');
    expect(matchLang(['de_DE.UTF-8'])).toBe('de');
    expect(matchLang(['it-IT', 'fr-FR'])).toBe('fr');
    expect(matchLang(['it-IT'])).toBe('en');
    expect(matchLang([])).toBe('en');
    expect(LANGS).toHaveLength(10);
  });

  it('t() 는 지금 언어 문구로, 자리표시자를 채운다', () => {
    useLang.getState().setLang('en');
    expect(t('common.continue')).toBe('Continue');
    useLang.getState().setLang('ko');
    expect(t('common.continue')).toBe('계속');
  });
});

describe('자동 이름 다시 표시 (relabel)', () => {
  it('어느 언어로 저장된 "로컬 터미널"이든 지금 언어로, 사용자 이름은 그대로', () => {
    useLang.getState().setLang('en');
    expect(relabel('로컬 터미널')).toBe('Local terminal');
    expect(relabel('ローカルターミナル')).toBe('Local terminal');
    expect(relabel('web-prod-01')).toBe('web-prod-01');
    expect(relabel(undefined)).toBeUndefined();
    useLang.getState().setLang('ko');
    expect(relabel('Local terminal')).toBe('로컬 터미널');
  });
});

describe('언어 파일 (locales/*.json)', () => {
  const ko = LOCALES.ko ?? {};
  const keys = Object.keys(ko);

  it('10개 언어 파일이 모두 있다', () => {
    expect(Object.keys(LOCALES).sort()).toEqual(LANGS.map((l) => l.id).sort());
  });

  for (const { id } of LANGS) {
    it(`${id}.json: ko.json 과 키가 같고, 빈 문구가 없고, 자리표시자가 같다`, () => {
      const dict = LOCALES[id] ?? {};
      const missing = keys.filter((k) => !(k in dict));
      const extra = Object.keys(dict).filter((k) => !(k in ko));
      const empty = Object.entries(dict).filter(([, v]) => !v || !v.trim()).map(([k]) => k);
      const ph = keys.filter((k) => k in dict && placeholders(dict[k] ?? '') !== placeholders(ko[k] ?? '')).map((k) => `${k}: ${dict[k]}`);
      expect({ missing, extra, empty, ph }).toEqual({ missing: [], extra: [], empty: [], ph: [] });
    });
  }
});

describe('화면 코드', () => {
  it('번역을 거치지 않은 한국어 문구가 없다(주석 제외)', () => {
    const left: string[] = [];
    for (const [path, src] of Object.entries(SOURCES).filter(([p]) => isUiSource(p))) {
      let inBlock = false;
      src.split(/\r?\n/).forEach((line, i) => {
        // 의도적으로 남긴 한국어(입력 별칭·옛 저장 데이터 호환)는 그 줄에 i18n-ok 주석을 단다.
        if (line.includes('i18n-ok')) return;
        let s = line;
        if (inBlock) {
          const end = s.indexOf('*/');
          if (end < 0) return;
          s = s.slice(end + 2);
          inBlock = false;
        }
        s = s.replace(/\{\/\*.*?\*\/\}/g, '').replace(/\/\*.*?\*\//g, '');
        const open = s.indexOf('/*');
        if (open >= 0) {
          inBlock = true;
          s = s.slice(0, open);
        }
        s = s.replace(/(^|[^:'"`])\/\/.*$/, '$1');
        if (HANGUL.test(s)) left.push(`${path}:${i + 1}: ${line.trim()}`);
      });
    }
    expect(left).toEqual([]);
  });
});
