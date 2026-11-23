// 상태 표시줄의 "새 버전 {버전}" 버튼. 누르면 배포 페이지를 외부 브라우저로 연다.
// Microsoft Store 설치본이면 Windows 의 업데이트 확인 창을 띄운다. 근거: spec/ui/shell.md §3, spec/behavior/lifecycle.md §6.1.
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUpdate } from '../state/update';
import { t } from '../i18n';

export function UpdateBadge() {
  const info = useUpdate((s) => s.available);
  if (!info) return null;
  const open = async () => {
    try {
      if (info.store) {
        // 스토어 설치본: Windows 의 업데이트 확인 창을 띄운다(결과는 토스트로 온다). lifecycle.md §6.1
        await getBackend().invoke('update.install');
        return;
      }
      if (!info.url) return;
      await getBackend().invoke('app.openExternal', [info.url]);
    } catch (e) {
      useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  return (
    <button className="update-badge" type="button" title={t(info.store ? 'update.storeTitle' : 'update.openTitle')} onClick={() => void open()}>
      {t('update.badge', { version: info.version })}
    </button>
  );
}
