// 화면 언어 연결(D75). 근거: spec/behavior/i18n.md.
// 설정 settings.language 가 지원 언어면 그것을, 비어 있으면(첫 실행 전) OS 언어를 맞춰 쓴다. 맞는 게 없으면 영어.
import { isLang, matchLang, useLang, type Lang } from '../i18n';
import { useApp } from './app';

let osTags: string[] = [];

/** 백엔드가 알려 준 OS 언어(app.info.osLanguage)를 기억하고 언어를 다시 맞춘다. */
export function setOsLanguage(tag: unknown): void {
  osTags = typeof tag === 'string' && tag ? [tag] : [];
  applyLanguage();
}

/** OS 언어 추정: 백엔드 값 → 웹뷰(navigator.languages) 순서. */
export function detectedLang(): Lang {
  const nav = typeof navigator !== 'undefined' ? [...(navigator.languages ?? []), navigator.language] : [];
  return matchLang([...osTags, ...nav]);
}

/** 설정 값으로 화면 언어를 정한다. */
export function resolveLang(setting: unknown): Lang {
  return isLang(setting) ? setting : detectedLang();
}

/** 아직 언어를 고르지 않았는지(첫 실행 언어 선택 화면을 보일지). */
export function languageUnset(): boolean {
  const s = useApp.getState().store;
  return !!s && !isLang(s.settings.language);
}

function applyLanguage(): void {
  const next = resolveLang(useApp.getState().store?.settings.language);
  if (useLang.getState().lang !== next) useLang.getState().setLang(next);
}

/** 저장 데이터가 바뀔 때마다 언어를 맞춘다. 구독 해제 함수를 돌려준다. */
export function startLanguageSync(): () => void {
  applyLanguage();
  return useApp.subscribe((s, prev) => {
    if (s.store?.settings.language !== prev.store?.settings.language) applyLanguage();
  });
}
