// 설정 8 · 정보·단축키. 근거: spec/ui/settings.md §8, spec/behavior/shortcuts.md §4, spec/acceptance G6·G7.
// 결정 D63(D8 대체): 참고한 프로젝트(BrightTerm v1.1.0, MIT)의 고지를 "참고한 프로젝트" 필드에 보인다.
// 결정 D73: 사용 조건(개인 무료 · 업무용 유료) 요약과 전문, 설정된 외부 링크([업무용 라이선스 구매]·[후원하기]·[홈페이지]).
import { useEffect, useState } from 'react';
import { ToggleRow } from '../ui/Controls';
import { LogoMark } from '../ui/Logo';
import { SHORTCUT_LABELS, shortcutText } from '../lib/shortcuts';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUpdate } from '../state/update';
import { useSetting, Group } from './Settings';
import { REFERENCE_NOTICE } from '../lib/notices';
import { aboutLinks, LICENSE_CONTACT, LICENSE_SUMMARY } from '../lib/links';
import { LicensePanel } from './LicensePanel';
// 사용 조건 전문: 저장소 루트 LICENSE 를 그대로 싣는다(vite.config 의 server.fs.allow).
import LICENSE_TEXT from '../../../LICENSE?raw';
// 오픈소스 고지 전문: 저장소 루트에서 `go run ./tools/notices` 가 만든 복사본(D64).
import THIRD_PARTY_NOTICES from '../generated/third-party-notices.md?raw';
import { t } from '../i18n';

export function SettingsAbout() {
  const [updateCheck, setUpdateCheck] = useSetting('updateCheck');
  const checkNow = useUpdate((s) => s.checkNow);
  const [info, setInfo] = useState<{ version?: string; dataDir?: string; updateEnabled?: boolean; updateSource?: string }>({});
  const pushToast = useApp((s) => s.pushToast);
  const links = aboutLinks();
  const openLink = (url: string) => {
    getBackend()
      .invoke('app.openExternal', [url])
      .catch((e) => pushToast({ kind: 'error', text: String(e?.message ?? e) }));
  };
  const copyContact = () => {
    getBackend()
      .invoke('clip.writeText', [LICENSE_CONTACT])
      .then(() => pushToast({ kind: 'ok', text: t('about.contactCopied') }))
      .catch((e) => pushToast({ kind: 'error', text: String(e?.message ?? e) }));
  };

  useEffect(() => {
    getBackend()
      .invoke<{ version?: string; dataDir?: string; updateEnabled?: boolean; updateSource?: string }>('app.info')
      .then((i) => setInfo(i ?? {}))
      .catch((e) => pushToast({ kind: 'error', text: String(e?.message ?? e) }));
    // pushToast 는 안정적인 참조다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <>
      <div style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
        <LogoMark size={44} />
        <div>
          <div style={{ fontSize: 16, fontWeight: 700 }}>DariTerm {info.version ?? ''}</div>
          <div className="faint" style={{ fontSize: 12.5 }}>{t('about.tagline')}</div>
        </div>
      </div>

      <div className="set-stack">
        <div className="set-label">{t('about.developer')}</div>
        <div>YangHun Lee (kai)</div>
      </div>
      <div className="set-stack">
        <div className="set-label">{t('about.license')}</div>
        <div>{t('about.proprietary')}</div>
      </div>
      <LicensePanel />
      <div className="set-stack">
        <div className="set-label">{t('about.terms')}</div>
        <div>{t('about.termsLine')}</div>
        <div className="faint" style={{ fontSize: 12 }}>{t(LICENSE_SUMMARY.personal)}</div>
        <div className="faint" style={{ fontSize: 12 }}>{t(LICENSE_SUMMARY.business)}</div>
        {/* D74: 구매처가 생기기 전까지 업무용 라이선스는 연락처로 문의받는다 */}
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <span>{t('about.contact')} <span style={{ userSelect: 'text' }}>{LICENSE_CONTACT}</span></span>
          <button className="btn small-btn" type="button" onClick={copyContact}>{t('common.copy')}</button>
        </div>
        <details>
          <summary className="faint" style={{ fontSize: 12, cursor: 'pointer' }}>{t('about.termsFull')}</summary>
          <div className="detail-box notices-box" style={{ marginTop: 6 }}>{LICENSE_TEXT}</div>
        </details>
        {links.length > 0 && (
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
            {links.map((l) => (
              <button key={l.key} className={`btn small-btn${l.key === 'purchase' ? ' primary' : ''}`} type="button" title={l.url} onClick={() => openLink(l.url)}>
                {t(l.label)}
              </button>
            ))}
          </div>
        )}
      </div>
      <div className="set-stack">
        <div className="set-label">{t('about.reference')}</div>
        <div>{REFERENCE_NOTICE.name} — {REFERENCE_NOTICE.summary}</div>
        <div className="faint" style={{ fontSize: 12 }}>{t(REFERENCE_NOTICE.note)}</div>
        {/* MIT 고지 전문: 접어 두고 필요할 때 펼친다 */}
        <details>
          <summary className="faint" style={{ fontSize: 12, cursor: 'pointer' }}>{t('about.licenseFull')}</summary>
          <div className="detail-box" style={{ marginTop: 6 }}>{REFERENCE_NOTICE.license}</div>
        </details>
      </div>
      <div className="set-stack">
        <div className="set-label">{t('about.oss')}</div>
        <div className="faint" style={{ fontSize: 12 }}>{t('about.ossDesc')}</div>
        <details>
          <summary className="faint" style={{ fontSize: 12, cursor: 'pointer' }}>{t('about.ossView')}</summary>
          <div className="detail-box notices-box" style={{ marginTop: 6 }}>{THIRD_PARTY_NOTICES}</div>
        </details>
      </div>

      {/* D80: 업데이트 확인은 배포 저장소가 정해진 빌드에서만 보인다(지금은 꺼짐). */}
      {info.updateEnabled && (
        <Group>
          <ToggleRow
            title={t('about.updateCheck')}
            desc={t(info.updateSource === 'store' ? 'about.updateCheckStoreDesc' : 'about.updateCheckDesc')}
            checked={updateCheck}
            onChange={setUpdateCheck}
          />
          <div style={{ paddingBottom: 10 }}>
            <button className="btn small-btn" type="button" onClick={() => void checkNow()}>{t('about.checkNow')}</button>
          </div>
        </Group>
      )}

      <div className="set-stack">
        <div className="set-label">{t('about.dataDir')}</div>
        <div className="detail-box">{info.dataDir ?? ''}</div>
      </div>

      <div>
        <div className="set-section-title" style={{ marginBottom: 8 }}>{t('about.shortcuts')}</div>
        <div className="sc-table">
          {SHORTCUT_LABELS.map((s) => (
            <ShortcutRow key={s.name} {...shortcutText(s)} />
          ))}
        </div>
      </div>
    </>
  );
}

function ShortcutRow({ name, keys }: { name: string; keys: string }) {
  return (
    <>
      <span className="sc-name">{name}</span>
      <span className="kbd">{keys}</span>
    </>
  );
}
