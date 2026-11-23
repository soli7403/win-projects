// 설정 · 로그(화면 순서 5번째). 세션 로그(터미널 출력 기록) 항목을 "연결"에서 떼어 낸 구역 (D120). 근거: spec/ui/settings.md §4a, spec/behavior/session.md §6.
import { FolderOpen } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { NumberField, useSetting, Group } from './Settings';
import { saveSettings } from '../state/settings';
import { t } from '../i18n';

export function SettingsLog() {
  const [sessionLog, setSessionLog] = useSetting('sessionLog');
  const [keepDays] = useSetting('sessionLogKeepDays');
  const [syncSec] = useSetting('sessionLogSyncSec');
  const [logStamp, setLogStamp] = useSetting('sessionLogTimestamp');
  const [logAnsi, setLogAnsi] = useSetting('sessionLogAnsi');
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
          title={t('conn.sessionLog')}
          desc={t('conn.sessionLogDesc')}
          checked={sessionLog}
          onChange={setSessionLog}
        />
        <ToggleRow
          title={t('conn.logTimestamp')}
          desc={t('conn.logTimestampDesc')}
          checked={!!logStamp}
          onChange={setLogStamp}
        />
        <ToggleRow
          title={t('conn.logAnsi')}
          desc={t('conn.logAnsiDesc')}
          checked={!!logAnsi}
          onChange={setLogAnsi}
        />
      </Group>
      <Field label={t('conn.logKeepDays')} help={t('conn.logKeepDaysHelp')}>
        <NumberField
          label={t('conn.logKeepDays')}
          value={keepDays}
          min={0}
          max={36500}
          fallback={30}
          onCommit={(v) => void saveSettings({ sessionLogKeepDays: v })}
        />
      </Field>
      <Field label={t('conn.logSyncSec')} help={t('conn.logSyncSecHelp')}>
        <NumberField
          label={t('conn.logSyncSec')}
          value={syncSec}
          min={0}
          max={5}
          fallback={2}
          onCommit={(v) => void saveSettings({ sessionLogSyncSec: v })}
        />
      </Field>
      <div>
        <button className="btn" type="button" onClick={() => void openLogs()}>
          <FolderOpen size={14} /> {t('conn.openLogs')}
        </button>
      </div>
    </>
  );
}
