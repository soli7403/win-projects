// 정보 화면의 외부 링크(구매·후원·홈페이지)와 사용 조건 요약. 근거: spec/ui/settings.md §8, D73.
// 주소는 코드에 박지 않고 빌드 설정으로 받는다: frontend/.env(또는 빌드 환경 변수)의
//   VITE_DARITERM_PURCHASE_URL, VITE_DARITERM_DONATE_URL, VITE_DARITERM_HOMEPAGE_URL
// 비어 있거나 http(s) 주소가 아니면 그 버튼을 숨긴다(백엔드 app.openExternal 도 http(s) 만 연다).

import type { MsgId } from '../i18n';

export interface ExternalLink {
  key: 'purchase' | 'donate' | 'homepage';
  /** 문구 키(D75). */
  label: MsgId;
  url: string;
}

/** http:// 또는 https:// 로 시작하는 주소만 쓴다. 그 외는 빈 문자열. */
export function safeUrl(value: unknown): string {
  const s = typeof value === 'string' ? value.trim() : '';
  return /^https?:\/\//i.test(s) ? s : '';
}

/** 주소가 설정된 링크만 [구매하기] → [후원하기] → [홈페이지] 순서로 돌려준다. */
export function aboutLinks(env: Record<string, unknown> = import.meta.env): ExternalLink[] {
  const all: ExternalLink[] = [
    { key: 'purchase', label: 'about.buyLicense', url: safeUrl(env.VITE_DARITERM_PURCHASE_URL) },
    { key: 'donate', label: 'about.donate', url: safeUrl(env.VITE_DARITERM_DONATE_URL) },
    { key: 'homepage', label: 'about.homepage', url: safeUrl(env.VITE_DARITERM_HOMEPAGE_URL) },
  ];
  return all.filter((l) => l.url !== '');
}

/** 업무용 라이선스 구매·문의 연락처(D74). 구매처가 생기기 전까지 이 주소로 받는다. LICENSE §2.4 와 같아야 한다. */
export const LICENSE_CONTACT = 'soli97403@gmail.com';

/** 정보 화면의 사용 조건 요약(LICENSE §2 와 같은 뜻). */
export const LICENSE_SUMMARY: { personal: MsgId; business: MsgId } = {
  personal: 'about.licensePersonal',
  business: 'about.licenseBusiness',
};
