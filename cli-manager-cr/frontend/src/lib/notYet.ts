// 아직 구현되지 않은 동작을 누르면 알린다(개발 중 표시).
import { useApp } from '../state/app';

export function notYet(label: string): void {
  useApp.getState().pushToast({ kind: 'info', text: `${label}: 다음 단계에서 구현됩니다` });
}
