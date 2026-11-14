// 다국어(D75). 근거: spec/behavior/i18n.md.
// 문구는 코드에 두지 않고 저장소 루트 locales/<언어>.json 에 둔다(백엔드와 같은 파일). 문구를 고칠 때는 그 파일만 고친다.
// 기준 파일은 locales/ko.json — 키 목록이 여기서 정해지고(MsgId 타입), 다른 언어 파일은 같은 키를 가진다.
// 화면 코드는 t('hostEditor.title') · t('session.connecting', { name }) 처럼 키로 부른다. 자리표시자는 {이름}.
import { create } from 'zustand';
import ko from '../../../locales/ko.json';

/** 지원 언어. 순서는 언어 선택 목록 순서. id 는 locales/<id>.json 파일 이름. */
export type Lang = 'en' | 'ko' | 'ja' | 'zh' | 'tw' | 'de' | 'fr' | 'es' | 'pt' | 'ru';

export const LANGS: { id: Lang; name: string; tag: string }[] = [
  { id: 'en', name: 'English', tag: 'en' },
  { id: 'ko', name: '한국어', tag: 'ko' },
  { id: 'ja', name: '日本語', tag: 'ja' },
  { id: 'zh', name: '简体中文', tag: 'zh-CN' },
  { id: 'tw', name: '繁體中文', tag: 'zh-TW' },
  { id: 'de', name: 'Deutsch', tag: 'de' },
  { id: 'fr', name: 'Français', tag: 'fr' },
  { id: 'es', name: 'Español', tag: 'es' },
  { id: 'pt', name: 'Português (Brasil)', tag: 'pt-BR' },
  { id: 'ru', name: 'Русский', tag: 'ru' },
];

/** 문구 키(locales/ko.json 의 키). 없는 키를 쓰면 tsc 가 잡는다. */
export type MsgId = keyof typeof ko;
type Dict = Partial<Record<string, string>>;

const FILES = import.meta.glob('../../../locales/*.json', { eager: true, import: 'default' }) as Record<string, Dict>;
/** 언어별 문구 표. 파일 이름(확장자 뺀 것)이 언어 id. */
export const LOCALES: Partial<Record<Lang, Dict>> = Object.fromEntries(
  Object.entries(FILES).map(([path, dict]) => [path.replace(/^.*\/([^/]+)\.json$/, '$1'), dict]),
);

export const FALLBACK_LANG: Lang = 'en';

export function isLang(v: unknown): v is Lang {
  return typeof v === 'string' && LANGS.some((l) => l.id === v);
}

/**
 * OS·브라우저 언어 태그(예: ko-KR, zh-Hant-TW, pt_BR.UTF-8)를 지원 언어로 맞춘다.
 * 앞에서부터 처음 맞는 것을 쓰고, 하나도 없으면 영어. 백엔드 i18n.Match 와 같은 규칙.
 */
export function matchLang(tags: readonly (string | undefined | null)[]): Lang {
  for (const raw of tags) {
    const tag = String(raw ?? '').trim().toLowerCase().replace(/_/g, '-').split('.')[0];
    if (!tag) continue;
    const parts = tag.split('-');
    const primary = parts[0];
    if (primary === 'zh') {
      const traditional = parts.some((p) => p === 'hant' || p === 'tw' || p === 'hk' || p === 'mo');
      return traditional ? 'tw' : 'zh';
    }
    if (primary === 'pt') return 'pt';
    if (isLang(primary) && primary !== 'tw' && primary !== 'zh') return primary;
  }
  return FALLBACK_LANG;
}

interface LangState {
  lang: Lang;
  setLang: (lang: Lang) => void;
}

/** 지금 화면 언어. App 이 구독해 바뀌면 전체를 다시 그린다. */
export const useLang = create<LangState>((set) => ({
  lang: FALLBACK_LANG,
  setLang: (lang) => {
    set({ lang });
    if (typeof document !== 'undefined') document.documentElement.lang = LANGS.find((l) => l.id === lang)?.tag ?? 'en';
  },
}));

/** 지금 언어의 BCP 47 태그(날짜·숫자 표시용, 예: toLocaleString(langTag())). */
export function langTag(): string {
  const lang = useLang.getState().lang;
  return LANGS.find((l) => l.id === lang)?.tag ?? 'en';
}

export type Vars = Record<string, string | number>;

function fill(text: string, vars?: Vars): string {
  if (!vars) return text;
  return text.replace(/\{(\w+)\}/g, (m, name: string) => (name in vars ? String(vars[name]) : m));
}

/** 키를 지금 언어 문구로 바꾼다. 그 언어에 없으면 영어 → 한국어 → 키 순서로 대신한다. */
export function t(id: MsgId, vars?: Vars): string {
  const lang = useLang.getState().lang;
  const text = LOCALES[lang]?.[id] ?? LOCALES.en?.[id] ?? LOCALES.ko?.[id] ?? id;
  return fill(text, vars);
}

/** 앱이 자동으로 붙여 저장하는 이름(탭·세션 제목)의 문구 키. 저장된 글자는 만들 때의 언어로 남는다. */
const AUTO_NAMES: MsgId[] = ['host.localTerminal'];

/**
 * 저장된 이름이 어느 언어로든 자동 이름(예: "로컬 터미널", "Local terminal")이면 지금 언어 문구로 바꿔 보인다.
 * 사용자가 직접 지은 이름은 그대로 둔다. 저장 데이터는 바꾸지 않는다(그릴 때만).
 */
export function relabel(text: string): string;
export function relabel(text: string | undefined): string | undefined;
export function relabel(text: string | undefined): string | undefined {
  if (!text) return text;
  for (const id of AUTO_NAMES) {
    if (Object.values(LOCALES).some((d) => d?.[id] === text)) return t(id);
  }
  return text;
}

/** 화면 컴포넌트용: 언어가 바뀌면 다시 그리도록 구독하고 t 를 돌려준다. */
export function useT(): typeof t {
  useLang((s) => s.lang);
  return t;
}
