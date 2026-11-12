// 첫 실행 가져오기 제안. 근거: spec/ui/shell.md §6-5, 수용 시나리오 X7.
// 볼트가 열렸고 서버가 0개이고 "이미 제안함" 기록이 없으면 import.scan 으로 후보를 찾고,
// 후보가 있을 때만 가져오기 대화상자를 연다. 제안하면 화면 저장소에 기록해 다시 묻지 않는다.
import { getBackend } from '../ipc/backend';
import type { Store } from '../ipc/types';
import { useApp } from './app';
import { useUi } from './ui';

export const IMPORT_SUGGESTED_KEY = 'importSuggested';

/** 제안할지 판정한다(순수 함수, 테스트용). */
export function shouldSuggestImport(store: Store | null, vaultUnlocked: boolean): boolean {
  if (!store || !vaultUnlocked) return false;
  if (store.hosts.length > 0) return false;
  const ui = (store.appUi ?? {}) as Record<string, unknown>;
  return ui[IMPORT_SUGGESTED_KEY] !== true;
}

/** 앱 시작 시 한 번 호출. 조건이 맞으면 후보를 찾아 대화상자를 연다. 구독 해제 함수를 돌려준다. */
export function startImportSuggestion(): () => void {
  let checked = false;
  const check = () => {
    if (checked) return;
    const st = useApp.getState();
    if (!st.booted || !shouldSuggestImport(st.store, !!st.vault?.unlocked)) return;
    checked = true;
    void (async () => {
      try {
        const found = await getBackend().invoke<unknown[]>('import.scan');
        if (!Array.isArray(found) || found.length === 0) return;
        const cur = useApp.getState().store;
        if (!cur) return;
        const appUi = (cur.appUi ?? {}) as Record<string, unknown>;
        await useApp.getState().patchStore({ appUi: { ...appUi, [IMPORT_SUGGESTED_KEY]: true } });
        useUi.getState().openImportDialog();
      } catch {
        // 조회 실패는 조용히 넘어간다(다음 실행 때 다시 시도).

      }
    })();
  };
  const off = useApp.subscribe(check);
  check();
  return off;
}
