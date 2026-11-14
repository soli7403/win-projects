// 설정 4 · 연결. 근거: spec/ui/settings.md §4, spec/behavior/lifecycle.md §3.
import { FolderOpen } from 'lucide-react';
import { ToggleRow } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useSetting, Group } from './Settings';
import { t } from '../i18n';

export function SettingsConnection() {
  const [autoReconnect, setAutoReconnect] = useSetting('autoReconnect');
  const [restoreLayout, setRestoreLayout] = useSetting('restoreLayout');
  const [sessionLog, setSessionLog] = useSetting('sessionLog');
  const pushToast = useApp((s) => s.pushToast);

  const openLogs = async () => {
    try {
      await getBackend().invoke('app.openLogs');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <>
      <Group>
        <ToggleRow
          title={t('conn.autoReconnect')}
          desc={t('conn.autoReconnectDesc')}
          checked={autoReconnect}
          onChange={setAutoReconnect}
        />
        <ToggleRow
          title={t('conn.restoreLayout')}
          desc={t('conn.restoreLayoutDesc')}
          checked={restoreLayout}
          onChange={setRestoreLayout}
        />
        <ToggleRow
          title={t('conn.sessionLog')}
          desc={t('conn.sessionLogDesc')}
          checked={sessionLog}
          onChange={setSessionLog}
        />
      </Group>
      <div>
        <button className="btn" type="button" onClick={() => void openLogs()}>
          <FolderOpen size={14} /> {t('conn.openLogs')}
        </button>
      </div>
    </>
  );
}
