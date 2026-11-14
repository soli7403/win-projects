// 사용 용도·라이선스 판단(D78): 개인용은 기한 없음, 업무용은 30일 평가 뒤·키 문제 때 시작 알림, 회사 PC 안내.
import { describe, it, expect } from 'vitest';
import { startupNotice, trialDaysLeft, usageView, type LicenseStatus } from './license';

const DAY = 86_400_000;
const now = Date.UTC(2026, 10, 14);
const base = { usage: '' as const, usageSince: 0, licenseKey: '', domainNoticeDone: false };
const valid: LicenseStatus = { state: 'valid', id: 'DT-1', org: 'ACME', seats: 5 };

describe('사용 형태', () => {
  it('키가 없으면 개인용, 기한 없음', () => {
    expect(usageView({ ...base, usage: 'personal' }, { state: 'none' }, now)).toEqual({ kind: 'personal' });
    expect(startupNotice({ ...base, usage: 'personal', usageSince: now - 999 * DAY }, null, false, now)).toBeNull();
  });

  it('업무용 평가: 30일 남은 날, 지나면 시작 알림', () => {
    expect(trialDaysLeft(now - 10 * DAY, now)).toBe(20);
    expect(usageView({ ...base, usage: 'business', usageSince: now - 10 * DAY }, null, now)).toEqual({ kind: 'trial', daysLeft: 20 });
    expect(startupNotice({ ...base, usage: 'business', usageSince: now - 31 * DAY }, null, false, now)).toBe('trialOver');
  });

  it('유효한 키는 업무용 라이선스(회사·사용 허가 수), 만료·버전 밖은 시작 알림', () => {
    const s = { ...base, usage: 'business' as const, usageSince: now - 99 * DAY, licenseKey: 'DT1.x' };
    expect(usageView(s, valid, now)).toEqual({ kind: 'licensed', status: valid });
    expect(startupNotice(s, valid, true, now)).toBeNull();
    expect(startupNotice(s, { ...valid, state: 'expired', exp: '2026-01-01' }, false, now)).toBe('licenseProblem');
    expect(startupNotice(s, { ...valid, state: 'version', maxMajor: 1 }, false, now)).toBe('licenseProblem');
  });

  it('개인용인데 회사 도메인 PC 면 한 번만 안내', () => {
    expect(startupNotice({ ...base, usage: 'personal' }, null, true, now)).toBe('domain');
    expect(startupNotice({ ...base, usage: 'personal', domainNoticeDone: true }, null, true, now)).toBeNull();
  });
});
