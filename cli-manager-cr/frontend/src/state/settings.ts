// 설정 저장 도우미: 값을 바꾸면 즉시 store.patch 로 저장한다(저장 버튼 없음). 근거: spec/ui/settings.md 머리말.
// 연속 변경이 서로 덮지 않도록, 저장 중인 값을 기준으로 병합한다.
import type { Settings } from '../ipc/types';
import { useApp } from './app';

let pending: Settings | null = null;

export async function saveSettings(patch: Partial<Settings>): Promise<void> {
  const cur = useApp.getState().store?.settings;
  if (!cur) return;
  const next = { ...(pending ?? cur), ...patch } as Settings;
  pending = next;
  try {
    await useApp.getState().patchStore({ settings: next });
  } finally {
    if (pending === next) pending = null;
  }
}
