// 설정 > 정보의 "사용 형태"(D78): 지금 용도·라이선스 상태(회사 이름·번호·사용 허가 수·버전·만료일),
// 라이선스 키 입력칸 [적용] / [키 제거], 용도 바꾸기. 상태 표시줄의 짧은 표시도 여기서 만든다.
import { useState } from 'react';
import { t } from '../i18n';
import { Segmented } from '../ui/Controls';
import { useApp } from '../state/app';
import { applyKey, removeKey, setUsage, useLicense } from '../state/license';
import { usageView, type UsageView } from '../lib/license';

/** 지금 사용 형태(설정 + 키 확인 결과). */
export function useUsageView(): UsageView {
  const s = useApp((st) => st.store?.settings);
  const status = useLicense((st) => st.status);
  return usageView({ usage: s?.usage ?? '', usageSince: s?.usageSince ?? 0, licenseKey: s?.licenseKey ?? '' }, status, Date.now());
}

/** 상태 표시줄용 짧은 글. */
export function usageShort(v: UsageView): string {
  switch (v.kind) {
    case 'personal': return t('lic.personal');
    case 'trial': return t('lic.trialShort', { days: v.daysLeft });
    case 'trialOver': return t('lic.trialOverShort');
    case 'licensed': return t('lic.licensedShort', { org: v.status.org ?? '', seats: v.status.seats ?? 0 });
    case 'licenseProblem': return t('lic.problemShort');
  }
}

function usageLong(v: UsageView): string {
  switch (v.kind) {
    case 'personal': return t('lic.personalNoKey');
    case 'trial': return t('lic.trialLong', { days: v.daysLeft });
    case 'trialOver': return t('lic.trialOverLong');
    case 'licensed':
    case 'licenseProblem': {
      const st = v.status;
      if (st.state === 'invalid') return t('lic.keyInvalid');
      const head = t('lic.licensedLong', { org: st.org ?? '', id: st.id ?? '', seats: st.seats ?? 0 });
      const ver = st.maxMajor ? t('lic.upToVersion', { ver: st.maxMajor }) : t('lic.allVersions');
      const exp = st.exp ? t(st.state === 'expired' ? 'lic.expiredOn' : 'lic.validUntil', { date: st.exp }) : t('lic.noExpiry');
      const tail = st.state === 'version' ? ` · ${t('lic.versionOut')}` : '';
      return `${head} · ${ver} · ${exp}${tail}`;
    }
  }
}

export function LicensePanel() {
  const view = useUsageView();
  const usage = useApp((s) => s.store?.settings.usage ?? '');
  const hasKey = useApp((s) => !!s.store?.settings.licenseKey);
  const pushToast = useApp((s) => s.pushToast);
  const [key, setKey] = useState('');
  const [busy, setBusy] = useState(false);

  const apply = async () => {
    if (!key.trim()) return;
    setBusy(true);
    try {
      const st = await applyKey(key);
      if (st.state === 'invalid' || st.state === 'none') pushToast({ kind: 'error', text: t('lic.keyInvalid') });
      else {
        setKey('');
        pushToast({ kind: 'ok', text: t('lic.keyApplied', { org: st.org ?? '' }) });
      }
    } finally {
      setBusy(false);
    }
  };

  const problem = view.kind === 'trialOver' || view.kind === 'licenseProblem';
  return (
    <div className="set-stack">
      <div className="set-label">{t('lic.usageLabel')}</div>
      <div className={problem ? 'warn-text' : undefined}>{usageLong(view)}</div>
      {!hasKey && (
        <Segmented
          label={t('lic.usageLabel')}
          value={usage === 'business' ? 'business' : 'personal'}
          onChange={(v) => void setUsage(v)}
          options={[
            { value: 'personal', label: t('lic.personal') },
            { value: 'business', label: t('lic.businessTrial') },
          ]}
        />
      )}
      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <input
          className="input mono"
          style={{ flex: 1 }}
          placeholder={t('lic.keyPlaceholder')}
          aria-label={t('lic.key')}
          value={key}
          onChange={(e) => setKey(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') void apply(); }}
        />
        <button className="btn small-btn primary" type="button" disabled={busy || !key.trim()} onClick={() => void apply()}>{t('lic.apply')}</button>
        {hasKey && <button className="btn small-btn" type="button" onClick={() => void removeKey()}>{t('lic.remove')}</button>}
      </div>
      <div className="faint" style={{ fontSize: 12 }}>{t('lic.keyHelp')}</div>
    </div>
  );
}
