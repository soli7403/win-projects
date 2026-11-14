// 첫 실행 용도 선택(D78). 근거: spec/behavior/license.md §2.
// 언어 선택 다음, 잠금·처음 설정 화면보다 먼저 보인다(settings.usage 가 비어 있을 때).
// 개인용(기한 없음) / 업무용(30일 평가). 라이선스 키가 있으면 여기서 넣을 수 있다.
import { useState } from 'react';
import { t } from '../i18n';
import { LogoMark } from '../ui/Logo';
import { applyKey, setUsage } from '../state/license';

export function UsagePicker() {
  const [picked, setPicked] = useState<'personal' | 'business'>('personal');
  const [key, setKey] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      if (key.trim()) {
        const st = await applyKey(key);
        if (st.state === 'invalid' || st.state === 'none') {
          setError(t('lic.keyInvalid'));
          return;
        }
        return; // 키가 맞으면 업무용으로 저장됨
      }
      await setUsage(picked);
    } finally {
      setBusy(false);
    }
  };

  const option = (id: 'personal' | 'business', title: string, desc: string) => (
    <button type="button" role="radio" aria-checked={picked === id} className={`lang-item usage-item${picked === id ? ' on' : ''}`} onClick={() => setPicked(id)}>
      <span className="usage-title">{title}</span>
      <span className="usage-desc">{desc}</span>
    </button>
  );

  return (
    <div className="lock-screen">
      <div className="lock-card" role="region" aria-label={t('lic.usageTitle')}>
        <div className="center-row"><LogoMark size={40} /></div>
        <h1>{t('lic.usageTitle')}</h1>
        <div className="usage-list" role="radiogroup" aria-label={t('lic.usageTitle')}>
          {option('personal', t('lic.personal'), t('lic.personalDesc'))}
          {option('business', t('lic.business'), t('lic.businessDesc'))}
        </div>
        <label className="field">
          <span className="field-label">{t('lic.keyOptional')}</span>
          <input className="input mono" value={key} placeholder="DT1.…" aria-label={t('lic.key')} onChange={(e) => setKey(e.target.value)} />
        </label>
        {error && <div className="err-text" role="alert">{error}</div>}
        <p className="desc">{t('lic.changeLater')}</p>
        <div className="lock-actions">
          <button className="btn primary" type="button" disabled={busy} onClick={() => void submit()}>{t('common.continue')}</button>
        </div>
      </div>
    </div>
  );
}
