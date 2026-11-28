// 도움말 본문·검색(D155). 근거: spec/ui/help.md.
import { describe, expect, it } from 'vitest';
import ko from '../../../locales/help/ko.md?raw';
import en from '../../../locales/help/en.md?raw';
import { helpFor, markParts, parseHelp, searchHelp, searchTerms, SETTINGS_HELP_TOPIC, type HelpBlock, type HelpTopic } from './help';

const kt = parseHelp(ko);
const et = parseHelp(en);
const byId = (list: HelpTopic[], id: string) => list.find((x) => x.id === id)!;

describe('본문 (D155)', () => {
  it('한국어·영어의 항목 id 가 같은 순서로 같고, 겹치지 않으며, 항목마다 본문이 있다', () => {
    expect(kt.length).toBeGreaterThanOrEqual(25);
    expect(kt.map((x) => x.id)).toEqual(et.map((x) => x.id));
    expect(new Set(kt.map((x) => x.id)).size).toBe(kt.length);
    for (const t of [...kt, ...et]) expect(t.blocks.length, t.id).toBeGreaterThan(0);
  });

  it('이번에 넣은 기능이 도움말에 있다 (D145~D154)', () => {
    expect(byId(kt, 'proxy').haystack).toContain('%h');
    expect(byId(kt, 'forward').haystack).toContain('전송량');
    expect(byId(kt, 'macros').haystack).toContain('일괄 입력');
    expect(byId(kt, 'login-script').haystack).toContain('보관함 값');
    expect(byId(kt, 'triggers').haystack).toContain('windows 알림');
    expect(byId(kt, 'import').haystack).toContain('proxyjump');
    expect(byId(kt, 'vault').haystack).toContain('화면 보호기');
    expect(byId(et, 'vault').haystack).toContain('screen saver');
  });

  it('설정 구역마다 연결된 도움말 항목이 있다', () => {
    const ids = new Set(kt.map((x) => x.id));
    for (const [section, id] of Object.entries(SETTINGS_HELP_TOPIC)) expect(ids.has(id), section).toBe(true);
  });

  it('표·목록·코드·안내를 읽는다', () => {
    const table = byId(kt, 'forward').blocks.find((b): b is Extract<HelpBlock, { kind: 'table' }> => b.kind === 'table')!;
    expect(table.head).toHaveLength(2);
    expect(table.rows).toHaveLength(3);
    const snip = byId(kt, 'snippets').blocks;
    expect(snip.some((b) => b.kind === 'code' && b.text.includes('{{서비스}}'))).toBe(true);
    expect(snip.some((b) => b.kind === 'note')).toBe(true);
    const list = byId(kt, 'servers').blocks.find((b) => b.kind === 'ol') as { items: { text: string; sub: string[] }[] };
    expect(list.items.some((item) => item.sub.length > 0)).toBe(true);
  });

  it('없는 언어는 영어 도움말을 쓰고 그렇다고 알린다', () => {
    const de = helpFor('de');
    expect(de.lang).toBe('en');
    expect(de.fallback).toBe(true);
    expect(helpFor('ko').fallback).toBe(false);
  });
});

describe('검색 (D155)', () => {
  it('모든 낱말이 든 항목만, 제목에 있으면 앞으로', () => {
    const hits = searchHelp(kt, '프록시 명령');
    expect(hits.length).toBeGreaterThan(0);
    expect(hits[0].topic.id).toBe('proxy');
    for (const h of hits) for (const w of ['프록시', '명령']) expect(h.topic.haystack).toContain(w);
  });

  it('대소문자를 가리지 않고, 빈 검색어는 결과가 없고, 없는 낱말은 빈 결과', () => {
    expect(searchHelp(et, 'PROXYCOMMAND').map((h) => h.topic.id)).toContain('proxy');
    expect(searchHelp(kt, '   ')).toEqual([]);
    expect(searchTerms('  a  B ')).toEqual(['a', 'b']);
    expect(searchHelp(kt, '없는낱말xyz')).toEqual([]);
  });

  it('미리 보기는 맞은 낱말 근처의 글이다', () => {
    const hit = searchHelp(kt, '보관함 값').find((h) => h.topic.id === 'login-script')!;
    expect(hit.snippet).toContain('보관함');
  });

  it('강조 조각: 정규식 특수 문자도 글자 그대로 찾는다', () => {
    expect(markParts('a %h b', ['%h'])).toEqual([['a ', false], ['%h', true], [' b', false]]);
    expect(markParts('x (y) z', ['(y)'])).toEqual([['x ', false], ['(y)', true], [' z', false]]);
    expect(markParts('abc', [])).toEqual([['abc', false]]);
  });
});
