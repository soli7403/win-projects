// 라이선스 상태 연결(D78): 저장된 키를 백엔드(license.check, 오프라인 확인)로 확인해 두고,
// 앱이 열린 뒤 한 번만 시작 알림을 보인다. 아무 정보도 밖으로 보내지 않는다.
import { create } from 'zustand';
import { getBackend } from '../ipc/backend';
import { t } from '../i18n';
import { startupNotice, type LicenseStatus } from '../lib/license';
import { useApp } from './app';
import { saveSettings } from './settings';
import { useUi } from './ui';

interface LicenseState {
  status: LicenseStatus | null;
  domainJoined: boolean;
  setDomainJoined: (v: boolean) => void;
  refresh: () => Promise<void>;
}

export const useLicense = create<LicenseState>((set) => ({
  status: null,
  domainJoined: false,
  setDomainJoined: (v) => set({ domainJoined: v }),
  async refresh() {
    const key = useApp.getState().store?.settings.licenseKey ?? '';
    try {
      const status = await getBackend().invoke<LicenseStatus>('license.check', [key]);
      set({ status: status ?? { state: 'none' } });
    } catch {
      set({ status: { state: key ? 'invalid' : 'none' } });
    }
  },
}));

/** 키를 확인한다(저장하지 않음). */
export async function checkKey(key: string): Promise<LicenseStatus> {
  return (await getBackend().invoke<LicenseStatus>('license.check', [key])) ?? { state: 'invalid' };
}

/** 키를 저장하고 업무용으로 바꾼다. 형식·서명이 틀리면 저장하지 않고 상태를 돌려준다. */
export async function applyKey(key: string): Promise<LicenseStatus> {
  const trimmed = key.replace(/\s+/g, '');
  const status = await checkKey(trimmed);
  if (status.state === 'invalid' || status.state === 'none') return status;
  await saveSettings({ licenseKey: trimmed, usage: 'business' });
  await useLicense.getState().refresh();
  return status;
}

export async function removeKey(): Promise<void> {
  await saveSettings({ licenseKey: '' });
  await useLicense.getState().refresh();
}

/** 용도를 바꾼다. 업무용으로 처음 바꾸면 평가 시작 시각을 남긴다. */
export async function setUsage(usage: 'personal' | 'business'): Promise<void> {
  const s = useApp.getState().store?.settings;
  const patch: Parameters<typeof saveSettings>[0] = { usage };
  if (usage === 'business' && !s?.usageSince) patch.usageSince = Date.now();
  await saveSettings(patch);
}

let noticeShown = false;

/** 앱이 열린 뒤(잠금 해제 후) 한 번만 부른다: 평가 끝남·키 문제·회사 PC 안내. [확인]이면 그대로 쓴다. */
export async function runStartupNotice(): Promise<void> {
  if (noticeShown) return;
  noticeShown = true;
  await useLicense.getState().refresh();
  const s = useApp.getState().store?.settings;
  if (!s) return;
  const { status, domainJoined } = useLicense.getState();
  const kind = startupNotice(s, status, domainJoined, Date.now());
  const ui = useUi.getState();
  if (kind === 'trialOver') {
    await ui.confirm({ title: t('lic.noticeTitle'), body: t('lic.trialOverBody'), detail: t('lic.whereToEnter'), okText: t('common.ok'), noCancel: true });
  } else if (kind === 'licenseProblem' && status) {
    const body = status.state === 'expired' ? t('lic.expiredBody', { date: status.exp ?? '' })
      : status.state === 'version' ? t('lic.versionBody', { ver: status.maxMajor ?? 0 })
      : t('lic.invalidBody');
    await ui.confirm({ title: t('lic.noticeTitle'), body, detail: t('lic.whereToEnter'), okText: t('common.ok'), noCancel: true });
  } else if (kind === 'domain') {
    const r = await ui.ask({ title: t('lic.domainTitle'), body: t('lic.domainBody'), okText: t('lic.switchBusiness'), noText: t('lic.stayPersonal') });
    if (r === 'yes') await setUsage('business');
    if (r !== null) await saveSettings({ domainNoticeDone: true });
  }
}
