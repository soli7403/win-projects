// 상태 표시줄의 "새 버전 {버전}" 버튼. 누르면 배포 페이지를 외부 브라우저로 연다. 근거: spec/ui/shell.md §3.
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUpdate } from '../state/update';
import { t } from '../i18n';

export function UpdateBadge() {
  const info = useUpdate((s) => s.available);
  if (!info) return null;
  const open = async () => {
    if (!info.url) return;
    try {
      await getBackend().invoke('app.openExternal', [info.url]);
    } catch (e) {
      useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  return (
    <button className="update-badge" type="button" title={t('update.openTitle')} onClick={() => void open()}>
      {t('update.badge', { version: info.version })}
    </button>
  );
}
