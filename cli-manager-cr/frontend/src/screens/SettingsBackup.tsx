// 설정 7 · 백업. 근거: spec/ui/settings.md §7, spec/data/sync-backup.md §3, spec/acceptance Y1·Y2·Y3.
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { t } from '../i18n';

export function SettingsBackup() {
  const pushToast = useApp((s) => s.pushToast);
  const confirm = useUi((s) => s.confirm);

  const exportBackup = async () => {
    try {
      const path = await getBackend().invoke<string | false>('backup.export');
      if (path) pushToast({ kind: 'ok', text: t('backup.saved', { path }) });
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  const importBackup = async () => {
    const ok = await confirm({
      title: t('backup.import'),
      body: t('backup.importBody'),
      okText: t('backup.importOk'),
      danger: true,
    });
    if (!ok) return;
    try {
      const done = await getBackend().invoke<boolean>('backup.import');
      if (!done) return;
      await useApp.getState().refreshStore();
      await useApp.getState().refreshVault();
      pushToast({ kind: 'ok', text: t('backup.imported') });
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <>
      <div className="set-notice">
        <div>
          {t('backup.desc').split('{encrypted}')[0]}<b>{t('backup.encrypted')}</b>{t('backup.desc').split('{encrypted}')[1]}
        </div>
      </div>
      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" type="button" onClick={() => void exportBackup()}>{t('backup.export')}</button>
        <button className="btn" type="button" onClick={() => void importBackup()}>{t('backup.import')}</button>
      </div>
    </>
  );
}
