// 정보 화면의 외부 링크(구매·후원·홈페이지)와 사용 조건 요약. 근거: spec/ui/settings.md §8, D73.
// 주소는 코드에 박지 않고 빌드 설정으로 받는다: frontend/.env(또는 빌드 환경 변수)의
//   VITE_DARITERM_PURCHASE_URL, VITE_DARITERM_DONATE_URL, VITE_DARITERM_HOMEPAGE_URL
// 비어 있거나 http(s) 주소가 아니면 그 버튼을 숨긴다(백엔드 app.openExternal 도 http(s) 만 연다).

export interface ExternalLink {
  key: 'purchase' | 'donate' | 'homepage';
  label: string;
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
    { key: 'purchase', label: '업무용 라이선스 구매', url: safeUrl(env.VITE_DARITERM_PURCHASE_URL) },
    { key: 'donate', label: '후원하기', url: safeUrl(env.VITE_DARITERM_DONATE_URL) },
    { key: 'homepage', label: '홈페이지', url: safeUrl(env.VITE_DARITERM_HOMEPAGE_URL) },
  ];
  return all.filter((l) => l.url !== '');
}

/** 업무용 라이선스 구매·문의 연락처(D74). 구매처가 생기기 전까지 이 주소로 받는다. LICENSE §2.4 와 같아야 한다. */
export const LICENSE_CONTACT = 'soli97403@gmail.com';

/** 정보 화면의 사용 조건 요약(LICENSE §2 와 같은 뜻). */
export const LICENSE_SUMMARY = {
  personal: '개인이 업무가 아닌 목적(학습, 개인 프로젝트, 취미 등)으로 쓰면 무료입니다.',
  business: '회사·기관·단체의 업무나 영리 목적의 일에 쓰려면 사용하는 사람마다 업무용 라이선스가 필요합니다. 검토용으로 30일까지 무료로 써 볼 수 있습니다.',
};
