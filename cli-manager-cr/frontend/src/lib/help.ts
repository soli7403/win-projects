// 도움말 본문 읽기·검색(D155). 본문은 locales/help/<언어>.md 에 두고(코드에 문구를 두지 않는다, D75),
// 그 언어 파일이 없으면 영어, 그것도 없으면 한국어를 쓴다. 근거: spec/ui/help.md.
//
// 형식(작은 마크다운): "## 제목 {#id}" 가 항목 하나. ### 소제목, 문단, "- "·"1. " 목록(두 칸 이상 들여 쓴 줄은 바로 위 항목의 하위),
// 표(| a | b |, 둘째 줄은 구분선), "> " 안내, ``` 코드 블록, 글자 안의 **굵게**·`코드`. 맨 앞의 <!-- --> 주석은 버린다.
// "Q: 질문" 줄과 그 아래 줄들(빈 줄까지)은 질문·답 한 묶음(자주 묻는 질문, D157).
// "![설명](파일.png)" 한 줄은 그림(locales/help/img/<언어>/파일, D158). 그림 파일이 없으면 그리지 않는다.

export type HelpBlock =
  | { kind: 'h'; text: string }
  | { kind: 'p'; text: string }
  | { kind: 'ul' | 'ol'; items: { text: string; sub: string[] }[] }
  | { kind: 'table'; head: string[]; rows: string[][] }
  | { kind: 'note'; text: string }
  | { kind: 'code'; text: string }
  | { kind: 'qa'; q: string; a: string }
  | { kind: 'img'; src: string; alt: string };

export interface HelpTopic {
  id: string;
  title: string;
  blocks: HelpBlock[];
  /** 검색용: 제목과 본문의 글자(꾸밈 기호를 뺀 것, 소문자). */
  haystack: string;
}

const TOPIC_RE = /^##\s+(.+?)\s*\{#([a-z0-9-]+)\}\s*$/;
/** 그림 줄. 파일 이름은 소문자·숫자·하이픈과 확장자만(경로를 벗어나지 못하게). */
const IMG_RE = /^!\[([^\]]*)\]\(([a-z0-9-]+\.(?:png|webp|jpg))\)$/;
const QA_RE = /^Q:\s+(.+)$/;

/** 꾸밈 기호(** ` )를 뺀 글자. */
export function plainText(s: string): string {
  return s.replace(/\*\*([^*]+)\*\*/g, '$1').replace(/`([^`]+)`/g, '$1');
}

function splitRow(line: string): string[] {
  return line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map((c) => c.trim());
}

function blockText(b: HelpBlock): string {
  switch (b.kind) {
    case 'ul':
    case 'ol':
      return b.items.map((i) => [i.text, ...i.sub].join(' ')).join(' ');
    case 'table':
      return [...b.head, ...b.rows.flat()].join(' ');
    case 'qa':
      return `${b.q} ${b.a}`;
    case 'img':
      return b.alt;
    default:
      return b.text;
  }
}

/** 도움말 파일을 항목 목록으로 읽는다. 제목 줄({#id})이 없는 앞부분은 버린다. */
export function parseHelp(src: string): HelpTopic[] {
  const lines = src.replace(/<!--[\s\S]*?-->/g, '').replace(/\r\n/g, '\n').split('\n');
  const topics: HelpTopic[] = [];
  let cur: HelpTopic | null = null;
  let i = 0;
  const push = (b: HelpBlock) => cur?.blocks.push(b);
  while (i < lines.length) {
    const line = lines[i];
    const trimmed = line.trim();
    const m = TOPIC_RE.exec(trimmed);
    if (m) {
      cur = { id: m[2], title: m[1], blocks: [], haystack: '' };
      topics.push(cur);
      i++;
      continue;
    }
    if (!cur || trimmed === '') {
      i++;
      continue;
    }
    if (trimmed.startsWith('```')) {
      const body: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trim().startsWith('```')) body.push(lines[i++]);
      i++; // 닫는 ```
      push({ kind: 'code', text: body.join('\n') });
      continue;
    }
    if (trimmed.startsWith('### ')) {
      push({ kind: 'h', text: trimmed.slice(4) });
      i++;
      continue;
    }
    if (trimmed.startsWith('|')) {
      const rows: string[][] = [];
      while (i < lines.length && lines[i].trim().startsWith('|')) {
        const cells = splitRow(lines[i]);
        if (!cells.every((c) => /^:?-{2,}:?$/.test(c))) rows.push(cells);
        i++;
      }
      push({ kind: 'table', head: rows[0] ?? [], rows: rows.slice(1) });
      continue;
    }
    const img = IMG_RE.exec(trimmed);
    if (img) {
      push({ kind: 'img', alt: img[1], src: img[2] });
      i++;
      continue;
    }
    const qa = QA_RE.exec(trimmed);
    if (qa) {
      // 답: 다음 줄부터 빈 줄 전까지(다음 "Q:" 나 항목 제목이 오면 거기서 끝).
      const body: string[] = [];
      i++;
      while (i < lines.length) {
        const t = lines[i].trim();
        if (t === '' || QA_RE.test(t) || TOPIC_RE.test(t)) break;
        body.push(t);
        i++;
      }
      push({ kind: 'qa', q: qa[1], a: body.join(' ') });
      continue;
    }
    if (trimmed.startsWith('> ')) {
      const body: string[] = [];
      while (i < lines.length && lines[i].trim().startsWith('>')) body.push(lines[i++].trim().replace(/^>\s?/, ''));
      push({ kind: 'note', text: body.join(' ') });
      continue;
    }
    const listOf = (s: string): 'ul' | 'ol' | null => (/^- /.test(s) ? 'ul' : /^\d+\. /.test(s) ? 'ol' : null);
    const kind = !/^\s/.test(line) ? listOf(trimmed) : null;
    if (kind) {
      const items: { text: string; sub: string[] }[] = [];
      while (i < lines.length) {
        const l = lines[i];
        const t = l.trim();
        if (t === '') break;
        if (/^\s{2,}/.test(l) && items.length > 0) {
          // 들여 쓴 줄: 바로 위 항목의 하위 목록(또는 이어지는 글)
          items[items.length - 1].sub.push(t.replace(/^(- |\d+\. )/, ''));
          i++;
          continue;
        }
        if (listOf(t) !== kind) break;
        items.push({ text: t.replace(/^(- |\d+\. )/, ''), sub: [] });
        i++;
      }
      push({ kind, items });
      continue;
    }
    // 문단: 빈 줄이나 다른 블록이 나올 때까지 잇는다.
    const body: string[] = [];
    while (i < lines.length) {
      const t = lines[i].trim();
      if (t === '' || TOPIC_RE.test(t) || IMG_RE.test(t) || QA_RE.test(t) || t.startsWith('```') || t.startsWith('### ') || t.startsWith('|') || t.startsWith('> ') || (!/^\s/.test(lines[i]) && listOf(t))) break;
      body.push(t);
      i++;
    }
    push({ kind: 'p', text: body.join(' ') });
  }
  for (const t of topics) t.haystack = plainText([t.title, ...t.blocks.map(blockText)].join(' ')).toLowerCase();
  return topics;
}

/** 검색어를 낱말로 나눈다(공백 기준, 소문자, 빈 것 제외). */
export function searchTerms(query: string): string[] {
  return query.toLowerCase().split(/\s+/).filter(Boolean);
}

export interface HelpHit {
  topic: HelpTopic;
  /** 본문에 나온 횟수(낱말마다 센 합). 제목에 나오면 크게 더한다. */
  score: number;
  /** 첫 번째로 맞은 자리 둘레의 글(목록에 보일 미리 보기). */
  snippet: string;
}

/**
 * 모든 낱말이 들어 있는 항목만, 많이 맞은 순서로 돌려준다(같으면 원래 순서).
 * 제목에 들어 있으면 본문보다 앞에 온다. 검색어가 비면 빈 배열.
 */
export function searchHelp(topics: readonly HelpTopic[], query: string): HelpHit[] {
  const terms = searchTerms(query);
  if (terms.length === 0) return [];
  const hits: (HelpHit & { order: number })[] = [];
  topics.forEach((topic, order) => {
    if (!terms.every((w) => topic.haystack.includes(w))) return;
    const title = topic.title.toLowerCase();
    let score = 0;
    for (const w of terms) {
      if (title.includes(w)) score += 100;
      score += topic.haystack.split(w).length - 1;
    }
    const at = topic.haystack.indexOf(terms[0], title.length);
    const from = Math.max(0, (at < 0 ? 0 : at) - 30);
    const raw = plainText([topic.title, ...topic.blocks.map(blockText)].join(' '));
    const snippet = (from > 0 ? '…' : '') + raw.slice(from, from + 90).trim() + (raw.length > from + 90 ? '…' : '');
    hits.push({ topic, score, snippet, order });
  });
  hits.sort((a, b) => b.score - a.score || a.order - b.order);
  return hits.map(({ topic, score, snippet }) => ({ topic, score, snippet }));
}

/** 글을 검색 낱말 기준으로 조각낸다: [글, 맞은 곳인지]. 대소문자 무시. */
export function markParts(text: string, terms: readonly string[]): [string, boolean][] {
  if (terms.length === 0 || !text) return [[text, false]];
  const re = new RegExp(`(${terms.map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`, 'gi');
  return text.split(re).filter((s) => s !== '').map((s): [string, boolean] => [s, terms.includes(s.toLowerCase())]);
}

const FILES = import.meta.glob('../../../locales/help/*.md', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>;
const SOURCES: Record<string, string> = Object.fromEntries(
  Object.entries(FILES).map(([path, src]) => [path.replace(/^.*\/([^/]+)\.md$/, '$1'), src]),
);
const cache = new Map<string, HelpTopic[]>();

// 그림: locales/help/img/<언어>/<파일>. 빌드 때 exe 에 함께 묶인다(URL 만 받고, 그리기는 <img>).
const IMAGE_FILES = import.meta.glob('../../../locales/help/img/*/*.{png,webp,jpg}', { eager: true, query: '?url', import: 'default' }) as Record<string, string>;
const IMAGES: Record<string, string> = Object.fromEntries(
  Object.entries(IMAGE_FILES).map(([path, url]) => [path.replace(/^.*\/img\/([^/]+)\/([^/]+)$/, '$1/$2'), url]),
);

/** 그림 주소: 그 언어 → 영어 → 한국어 순으로 찾는다. 없으면 null(그 그림은 그리지 않는다). */
export function helpImage(lang: string, file: string, images: Record<string, string> = IMAGES): string | null {
  return images[`${lang}/${file}`] ?? images[`en/${file}`] ?? images[`ko/${file}`] ?? null;
}

/** 목록 순서: 기능 순서(본문 파일의 순서) 또는 이름순(그 언어의 가나다·ABC). D156. */
export type HelpSort = 'feature' | 'name';

// 앱 언어 코드 → 정렬에 쓸 로캘. tw 는 BCP 47 코드가 아니라서 바꾼다.
const COLLATE_LOCALE: Record<string, string> = { tw: 'zh-TW' };

/** 항목을 고른 순서로. 이름순은 그 언어의 정렬 규칙(Intl.Collator), 숫자는 크기로. 원래 배열은 건드리지 않는다. */
export function sortTopics(topics: readonly HelpTopic[], sort: HelpSort, lang: string): HelpTopic[] {
  if (sort !== 'name') return [...topics];
  let coll: Intl.Collator;
  try {
    coll = new Intl.Collator(COLLATE_LOCALE[lang] ?? lang, { numeric: true, sensitivity: 'base' });
  } catch {
    coll = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });
  }
  return [...topics].sort((a, b) => coll.compare(plainText(a.title), plainText(b.title)));
}

const SORT_KEY = 'dariterm.helpSort';

/** 마지막에 고른 순서(이 PC 의 화면 저장소). 읽을 수 없으면 기능 순서. */
export function loadHelpSort(): HelpSort {
  try {
    return globalThis.localStorage?.getItem(SORT_KEY) === 'name' ? 'name' : 'feature';
  } catch {
    return 'feature';
  }
}

export function saveHelpSort(sort: HelpSort): void {
  try {
    globalThis.localStorage?.setItem(SORT_KEY, sort);
  } catch {
    /* 저장 실패는 무시 — 다음에는 기능 순서 */
  }
}

/** 도움말 창 크기(px). null 이면 기본 크기(CSS: 820 × 80vh). D159. */
export interface HelpSize {
  w: number;
  h: number;
}

export const HELP_MIN_W = 600;
export const HELP_MIN_H = 380;
const SIZE_KEY = 'dariterm.helpSize';

/** 창에 들어가게 자른다: 최소 600×380, 최대 (창 너비 − 40) × (창 높이의 92% − 16). 창이 더 작으면 최소가 이긴다. */
export function clampHelpSize(s: HelpSize, viewW: number, viewH: number): HelpSize {
  const maxW = Math.max(HELP_MIN_W, viewW - 40);
  const maxH = Math.max(HELP_MIN_H, Math.floor(viewH * 0.92) - 16);
  const fit = (v: number, lo: number, hi: number) => Math.round(Math.min(hi, Math.max(lo, Number.isFinite(v) ? v : lo)));
  return { w: fit(s.w, HELP_MIN_W, maxW), h: fit(s.h, HELP_MIN_H, maxH) };
}

/** 마지막에 바꾼 크기(이 PC 의 화면 저장소). 없거나 깨졌으면 null(기본 크기). */
export function loadHelpSize(): HelpSize | null {
  try {
    const raw = globalThis.localStorage?.getItem(SIZE_KEY);
    if (!raw) return null;
    const v = JSON.parse(raw) as Partial<HelpSize>;
    return typeof v.w === 'number' && typeof v.h === 'number' && v.w > 0 && v.h > 0 ? { w: v.w, h: v.h } : null;
  } catch {
    return null;
  }
}

/** 크기를 저장한다. null 이면 지운다(기본 크기로). */
export function saveHelpSize(s: HelpSize | null): void {
  try {
    if (s) globalThis.localStorage?.setItem(SIZE_KEY, JSON.stringify({ w: Math.round(s.w), h: Math.round(s.h) }));
    else globalThis.localStorage?.removeItem(SIZE_KEY);
  } catch {
    /* 저장 실패는 무시 — 다음에는 기본 크기 */
  }
}

/** 그 언어의 도움말(없으면 영어 → 한국어). fallback 은 다른 언어 본문을 대신 보였는지. */
export function helpFor(lang: string): { topics: HelpTopic[]; lang: string; fallback: boolean } {
  const pick = SOURCES[lang] ? lang : SOURCES.en ? 'en' : 'ko';
  let topics = cache.get(pick);
  if (!topics) {
    topics = parseHelp(SOURCES[pick] ?? '');
    cache.set(pick, topics);
  }
  return { topics, lang: pick, fallback: pick !== lang };
}

/** 설정 구역 → 도움말 항목 id (설정 화면의 [도움말] 단추가 연다). */
export const SETTINGS_HELP_TOPIC: Record<string, string> = {
  appearance: 'settings',
  input: 'terminal',
  shortcuts: 'shortcuts',
  rules: 'triggers',
  files: 'sftp',
  connection: 'connect',
  log: 'log',
  sync: 'sync',
  security: 'vault',
  backup: 'sync',
  about: 'license',
};
