// 도움말 후속(D156~D158): 목록 정렬, 자주 묻는 질문 카드, 그림.
import { afterEach, describe, expect, it } from 'vitest';
import ko from '../../../locales/help/ko.md?raw';
import en from '../../../locales/help/en.md?raw';
import { clampHelpSize, helpImage, loadHelpSize, loadHelpSort, saveHelpSize, parseHelp, plainText, saveHelpSort, searchHelp, sortTopics, type HelpBlock, type HelpTopic } from './help';

const kt = parseHelp(ko);
const et = parseHelp(en);
const titles = (list: HelpTopic[]) => list.map((x) => plainText(x.title));

describe('목록 정렬 (D156)', () => {
  it('기능 순서는 본문 파일 순서 그대로이고, 원래 배열을 바꾸지 않는다', () => {
    const before = kt.map((x) => x.id);
    const out = sortTopics(kt, 'feature', 'ko');
    expect(out.map((x) => x.id)).toEqual(before);
    expect(out).not.toBe(kt);
    sortTopics(kt, 'name', 'ko');
    expect(kt.map((x) => x.id)).toEqual(before);
  });

  it('한국어 이름순은 가나다순이다', () => {
    const out = titles(sortTopics(kt, 'name', 'ko'));
    const coll = new Intl.Collator('ko', { numeric: true, sensitivity: 'base' });
    expect(out).toEqual([...out].sort(coll.compare));
    // 가나다: "개인용…" 이 "화면 둘러보기" 보다 앞, "화면…" 은 뒤쪽
    expect(out.indexOf('개인용·업무용과 업데이트')).toBeLessThan(out.indexOf('화면 둘러보기'));
    expect(out[0]).toBe('개인용·업무용과 업데이트');
  });

  it('영어 이름순은 ABC 순이고 대소문자를 가리지 않는다', () => {
    const out = titles(sortTopics(et, 'name', 'en'));
    for (let i = 1; i < out.length; i++) expect(out[i - 1].localeCompare(out[i], 'en', { sensitivity: 'base' })).toBeLessThanOrEqual(0);
    expect(out[0].toLowerCase().startsWith('a')).toBe(true);
  });

  it('BCP 47 이 아닌 앱 언어 코드(tw)나 이상한 코드도 오류 없이 정렬한다', () => {
    expect(sortTopics(et, 'name', 'tw').length).toBe(et.length);
    expect(sortTopics(et, 'name', '!!').length).toBe(et.length);
  });

  describe('고른 순서 기억', () => {
    afterEach(() => localStorage.removeItem('dariterm.helpSort'));
    it('저장한 값을 읽고, 없거나 모르는 값이면 기능 순서', () => {
      expect(loadHelpSort()).toBe('feature');
      saveHelpSort('name');
      expect(loadHelpSort()).toBe('name');
      localStorage.setItem('dariterm.helpSort', 'zzz');
      expect(loadHelpSort()).toBe('feature');
    });
  });
});

describe('자주 묻는 질문 카드 (D157)', () => {
  const qa = (list: HelpTopic[]) => list.find((x) => x.id === 'faq')!.blocks.filter((b): b is Extract<HelpBlock, { kind: 'qa' }> => b.kind === 'qa');

  it('한국어·영어 모두 질문마다 카드 하나, 질문과 답이 비지 않는다', () => {
    for (const list of [kt, et]) {
      const cards = qa(list);
      expect(cards.length).toBeGreaterThanOrEqual(9);
      for (const c of cards) {
        expect(c.q).not.toBe('');
        expect(c.a).not.toBe('');
      }
    }
    expect(qa(kt).length).toBe(qa(et).length);
  });

  it('예전 형식(**질문** 문단)이 남아 있지 않다', () => {
    for (const list of [kt, et]) {
      const faq = list.find((x) => x.id === 'faq')!;
      expect(faq.blocks.some((b) => b.kind === 'p' && b.text.startsWith('**'))).toBe(false);
    }
  });

  it('"Q:" 다음 줄들이 답이 되고, 빈 줄·다음 Q 에서 끝난다', () => {
    const [t] = parseHelp('## T {#t}\n\nQ: 하나?\n답 첫 줄\n둘째 줄\nQ: 둘?\n답2\n\n문단');
    expect(t.blocks).toEqual([
      { kind: 'qa', q: '하나?', a: '답 첫 줄 둘째 줄' },
      { kind: 'qa', q: '둘?', a: '답2' },
      { kind: 'p', text: '문단' },
    ]);
  });

  it('질문과 답의 글자로 검색된다', () => {
    expect(searchHelp(kt, '화면 보호기').map((h) => h.topic.id)).toContain('faq');
    expect(searchHelp(et, 'screen saver').map((h) => h.topic.id)).toContain('faq');
  });
});

describe('그림 (D158)', () => {
  it('"![설명](파일.png)" 한 줄은 그림 블록, 설명은 검색에 들어간다', () => {
    const [t] = parseHelp('## T {#t}\n\n앞 문단\n![서버 편집 화면](servers.png)\n뒤 문단');
    expect(t.blocks).toEqual([
      { kind: 'p', text: '앞 문단' },
      { kind: 'img', src: 'servers.png', alt: '서버 편집 화면' },
      { kind: 'p', text: '뒤 문단' },
    ]);
    expect(t.haystack).toContain('서버 편집 화면');
  });

  it('경로·대문자·다른 확장자가 든 파일 이름은 그림이 아니다(글자로 남는다)', () => {
    for (const bad of ['![x](../a.png)', '![x](img/a.png)', '![x](A.png)', '![x](a.gif)', '![x](a.png) 뒤 글']) {
      const [t] = parseHelp(`## T {#t}\n\n${bad}`);
      expect(t.blocks[0].kind, bad).toBe('p');
    }
  });

  it('그림 주소는 그 언어 → 영어 → 한국어, 없으면 null', () => {
    const imgs = { 'ko/a.png': '/k/a', 'en/a.png': '/e/a', 'en/b.png': '/e/b', 'ko/c.png': '/k/c' };
    expect(helpImage('ko', 'a.png', imgs)).toBe('/k/a');
    expect(helpImage('ko', 'b.png', imgs)).toBe('/e/b');
    expect(helpImage('de', 'a.png', imgs)).toBe('/e/a');
    expect(helpImage('de', 'c.png', imgs)).toBe('/k/c');
    expect(helpImage('ko', 'zzz.png', imgs)).toBeNull();
  });

  it('본문이 가리키는 그림 파일이 한국어·영어 모두 있다(자동 캡처 결과)', () => {
    for (const [lang, list] of [['ko', kt], ['en', et]] as const) {
      const imgs = list.flatMap((x) => x.blocks).filter((b): b is Extract<HelpBlock, { kind: 'img' }> => b.kind === 'img');
      expect(imgs.length, lang).toBeGreaterThanOrEqual(12);
      for (const b of imgs) {
        const url = helpImage(lang, b.src);
        expect(url, `${lang}/${b.src}`).not.toBeNull();
        expect(url!.includes(`/${lang}/`), `${lang}/${b.src} → ${url}`).toBe(true);
      }
    }
  });

  it('한국어·영어의 그림은 같은 항목에 같은 파일 이름이다', () => {
    const files = (list: HelpTopic[]) => list.map((x) => x.blocks.filter((b) => b.kind === 'img').map((b) => (b as { src: string }).src).join(','));
    expect(files(et)).toEqual(files(kt));
  });
});

describe('창 크기 (D159)', () => {
  afterEach(() => localStorage.removeItem('dariterm.helpSize'));

  it('창 안으로 자른다: 최소 600×380, 최대 (너비−40)×(높이×0.92−16)', () => {
    expect(clampHelpSize({ w: 900, h: 600 }, 1600, 1000)).toEqual({ w: 900, h: 600 });
    expect(clampHelpSize({ w: 100, h: 100 }, 1600, 1000)).toEqual({ w: 600, h: 380 });
    expect(clampHelpSize({ w: 5000, h: 5000 }, 1600, 1000)).toEqual({ w: 1560, h: 904 });
    // 창이 최소보다 작으면 최소가 이긴다
    expect(clampHelpSize({ w: 5000, h: 5000 }, 500, 300)).toEqual({ w: 600, h: 380 });
    expect(clampHelpSize({ w: Number.NaN, h: 500 }, 1600, 1000)).toEqual({ w: 600, h: 500 });
  });

  it('저장·읽기, 깨진 값은 null, null 저장은 지우기', () => {
    expect(loadHelpSize()).toBeNull();
    saveHelpSize({ w: 900.4, h: 600.6 });
    expect(loadHelpSize()).toEqual({ w: 900, h: 601 });
    localStorage.setItem('dariterm.helpSize', '{"w":"x","h":1}');
    expect(loadHelpSize()).toBeNull();
    localStorage.setItem('dariterm.helpSize', 'not json');
    expect(loadHelpSize()).toBeNull();
    saveHelpSize({ w: 900, h: 600 });
    saveHelpSize(null);
    expect(localStorage.getItem('dariterm.helpSize')).toBeNull();
  });
});
