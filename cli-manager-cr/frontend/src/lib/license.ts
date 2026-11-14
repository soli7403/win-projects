// 사용 용도·업무용 라이선스 판단(D78). 근거: spec/behavior/license.md.
// - 개인용: 기한 없음. 라이선스 키가 없으면 개인용으로 본다.
// - 업무용(키 없음): 30일 평가. 지나면 시작할 때 알림 한 번([확인]만), 계속 정상 실행.
// - 업무용(키 있음): 키 만료일이 지났거나 버전이 범위 밖이면 시작할 때 알림 한 번.
// - 개인용인데 회사 도메인 가입 PC: 한 번만 묻는다(업무용으로 바꾸기 / 개인용이 맞습니다).
import type { Settings } from '../ipc/types';

export const TRIAL_DAYS = 30;
const DAY = 86_400_000;

/** license.check 결과. */
export interface LicenseStatus {
  state: 'none' | 'valid' | 'invalid' | 'expired' | 'version';
  id?: string;
  org?: string;
  seats?: number;
  maxMajor?: number;
  exp?: string;
}

/** 화면에 보일 사용 형태. */
export type UsageView =
  | { kind: 'personal' }
  | { kind: 'trial'; daysLeft: number }
  | { kind: 'trialOver' }
  | { kind: 'licensed'; status: LicenseStatus }
  | { kind: 'licenseProblem'; status: LicenseStatus };

/** 업무용 평가 남은 날(올림, 0 이하이면 끝남). */
export function trialDaysLeft(usageSince: number, now: number): number {
  if (!usageSince) return TRIAL_DAYS;
  return Math.ceil((usageSince + TRIAL_DAYS * DAY - now) / DAY);
}

/** 설정과 키 확인 결과로 지금 사용 형태를 정한다. 키가 없으면 개인용 또는 업무용 평가. */
export function usageView(s: Pick<Settings, 'usage' | 'usageSince' | 'licenseKey'>, status: LicenseStatus | null, now: number): UsageView {
  if (s.licenseKey && status && status.state !== 'none') {
    return status.state === 'valid' ? { kind: 'licensed', status } : { kind: 'licenseProblem', status };
  }
  if (s.usage === 'business') {
    const left = trialDaysLeft(s.usageSince, now);
    return left > 0 ? { kind: 'trial', daysLeft: left } : { kind: 'trialOver' };
  }
  return { kind: 'personal' };
}

export type StartupNotice = 'trialOver' | 'licenseProblem' | 'domain' | null;

/** 시작할 때 보일 알림(없으면 null). 한 번 실행에 한 번만 보인다(부르는 쪽이 지킨다). */
export function startupNotice(
  s: Pick<Settings, 'usage' | 'usageSince' | 'licenseKey' | 'domainNoticeDone'>,
  status: LicenseStatus | null,
  domainJoined: boolean,
  now: number,
): StartupNotice {
  const view = usageView(s, status, now);
  if (view.kind === 'trialOver') return 'trialOver';
  if (view.kind === 'licenseProblem') return 'licenseProblem';
  if (view.kind === 'personal' && domainJoined && !s.domainNoticeDone) return 'domain';
  return null;
}
