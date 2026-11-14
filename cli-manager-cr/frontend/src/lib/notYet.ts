// 아직 구현되지 않은 동작을 누르면 알린다(개발 중 표시).
import { useApp } from '../state/app';
import { t } from '../i18n';

export function notYet(label: string): void {
  useApp.getState().pushToast({ kind: 'info', text: t('notYet.text', { label }) });
}
