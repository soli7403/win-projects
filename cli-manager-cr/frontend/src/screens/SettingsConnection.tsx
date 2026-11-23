// 설정 4 · 연결. 근거: spec/ui/settings.md §4, spec/behavior/lifecycle.md §3.
// 세션 로그 항목은 "로그" 구역(SettingsLog.tsx)으로 옮겼다 (D120).
import { ToggleRow } from '../ui/Controls';
import { useSetting, Group } from './Settings';
import { t } from '../i18n';

export function SettingsConnection() {
  const [autoReconnect, setAutoReconnect] = useSetting('autoReconnect');
  const [restoreLayout, setRestoreLayout] = useSetting('restoreLayout');

  return (
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
    </Group>
  );
}
