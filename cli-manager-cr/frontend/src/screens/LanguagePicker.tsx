// 첫 실행 언어 선택(D75). 근거: spec/behavior/i18n.md §2.
// settings.language 가 비어 있으면 잠금·설정 화면보다 먼저 보인다. OS 언어가 미리 골라져 있고,
// 고르면 바로 그 언어로 바뀌어 보이며, [계속]을 누르면 저장한다(나중에 설정 > 모양 > 언어에서 바꿀 수 있다).
import { useState } from 'react';
import { LANGS, useLang, useT, type Lang } from '../i18n';
import { LogoMark } from '../ui/Logo';
import { saveSettings } from '../state/settings';
import { detectedLang } from '../state/language';

export function LanguagePicker() {
  const t = useT();
  const [picked, setPicked] = useState<Lang>(() => useLang.getState().lang ?? detectedLang());
  const [busy, setBusy] = useState(false);

  const pick = (id: Lang) => {
    setPicked(id);
    useLang.getState().setLang(id);
  };

  const submit = async () => {
    setBusy(true);
    try {
      await saveSettings({ language: picked });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="lock-screen">
      <div className="lock-card" role="region" aria-label={t('lang.pickerAria')}>
        <div className="center-row"><LogoMark size={40} /></div>
        <h1>{t('lang.title')}</h1>
        <p className="desc">{t('lang.hint')}</p>
        <div className="lang-list" role="radiogroup" aria-label={t('lang.label')}>
          {LANGS.map((l) => (
            <button
              key={l.id}
              type="button"
              role="radio"
              aria-checked={picked === l.id}
              className={`lang-item${picked === l.id ? ' on' : ''}`}
              lang={l.tag}
              onClick={() => pick(l.id)}
            >
              {l.name}
            </button>
          ))}
        </div>
        <div className="lock-actions">
          <button className="btn primary" type="button" disabled={busy} onClick={() => void submit()}>
            {t('common.continue')}
          </button>
        </div>
      </div>
    </div>
  );
}
