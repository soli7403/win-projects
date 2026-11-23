// 앱 루트: 백엔드 초기화 → 상태 읽기 → 뼈대 + 잠금 화면(볼트가 잠겼을 때 또는 복구 코드 확인 전).
// 근거: spec/ui/shell.md §6, spec/behavior/shortcuts.md §1.
import { useEffect } from 'react';
import { initBackend, getBackend } from './ipc/backend';
import { subscribeAppEvents, useApp } from './state/app';
import { useUi } from './state/ui';
import { useRequests, type UiRequest } from './state/requests';
import { useWorkspace } from './state/workspace';
import { subscribeSessionEvents, runShortcut } from './state/actions';
import { shortcutFor } from './lib/keys';
import { ContextMenuHost } from './ui/ContextMenu';
import { DialogHost } from './ui/DialogHost';
import { Toasts } from './ui/Toasts';
import { Lock } from './screens/Lock';
import { Shell } from './screens/Shell';
import { HostEditor } from './screens/HostEditor';
import { FolderEditor } from './screens/FolderEditor';
import { QuickPalette } from './screens/QuickPalette';
import { RequestDialogs } from './screens/RequestDialogs';
import { startLayoutPersistence } from "./state/layoutSave";
import { startImportSuggestion } from "./state/importSuggest";
import { subscribeUpdateEvents, useUpdate } from './state/update';
import { Settings } from './screens/Settings';
import { ImportDialog } from './screens/ImportDialog';
import { ErrorBoundary } from './ui/ErrorBoundary';
import { isLang, useLang, useT } from './i18n';
import { setOsLanguage, startLanguageSync } from './state/language';
import { LanguagePicker } from './screens/LanguagePicker';
import { UsagePicker } from './screens/UsagePicker';
import { runStartupNotice, useLicense } from './state/license';

export function App() {
  const t = useT();
  // 언어가 바뀌면 전체를 다시 그린다(다시 마운트하지 않으므로 터미널은 그대로). D75.
  useLang((s) => s.lang);
  const languageSet = useApp((s) => isLang(s.store?.settings.language));
  const usageSet = useApp((s) => !!s.store?.settings.usage || !!s.store?.settings.licenseKey);
  const appOpen = useApp((s) => !!s.vault?.unlocked && !s.recoveryPending);
  // 앱이 열리면(잠금 해제 뒤) 한 번만 라이선스·회사 PC 안내를 보인다. [확인]이면 그대로 쓴다. D78.
  useEffect(() => {
    if (appOpen && languageSet && usageSet) void runStartupNotice();
  }, [appOpen, languageSet, usageSet]);
  const booted = useApp((s) => s.booted);
  const vault = useApp((s) => s.vault);
  const recoveryPending = useApp((s) => s.recoveryPending);
  const theme = useApp((s) => s.store?.settings.theme ?? 'dark');
  const editor = useUi((s) => s.editor);
  const tabCount = useWorkspace((s) => s.tabs.length);

  useEffect(() => {
    let offs: (() => void)[] = [];
    let cancelled = false;
    (async () => {
      await initBackend();
      if (cancelled) return;
      const b = getBackend();
      offs = [
        subscribeAppEvents(),
        subscribeSessionEvents(),
        subscribeUpdateEvents(),
        b.on('ui.request', (r: UiRequest) => useRequests.getState().push(r)),
      ];
      // OS 언어(백엔드가 알려 줌)는 첫 실행 언어 선택의 기본값과, 고르기 전 화면 언어로 쓴다.
      const info = await b.invoke<{ osLanguage?: string; domainJoined?: boolean }>('app.info').catch(() => undefined);
      setOsLanguage(info?.osLanguage);
      useLicense.getState().setDomainJoined(!!info?.domainJoined);
      await useApp.getState().boot();
      void useLicense.getState().refresh();
      void useUpdate.getState().loadStatus();
    })();
    const stopLanguage = startLanguageSync();
    const stopLayout = startLayoutPersistence();
    const stopImport = startImportSuggestion();

    // 단축키: 볼트 잠김·대화상자 열림 조건을 따른다(shortcuts.md §1). 빠른 접속이 열려 있으면 모두 동작.
    const onKey = (e: KeyboardEvent) => {
      const action = shortcutFor(e);
      if (!action) return;
      const v = useApp.getState().vault;
      if (v?.initialized && !v.unlocked) return;
      const ui = useUi.getState();
      const busy = ui.dialogs.length > 0 || useRequests.getState().queue.length > 0 || !!ui.editor;
      if (busy && !ui.quick.open && action.action !== 'fullscreen') return;
      e.preventDefault();
      e.stopPropagation();
      runShortcut(action.action, action.arg);
    };
    window.addEventListener('keydown', onKey, true);

    return () => {
      cancelled = true;
      offs.forEach((off) => off());
      stopLanguage();
      stopLayout();
      stopImport();
      window.removeEventListener('keydown', onKey, true);
    };
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // 저장 데이터와 볼트 상태를 읽는 동안에는 빈 틀만 보인다.
  if (!booted) return <div className="app" />;
  // 첫 실행: 언어를 먼저 고른다(잠금·설정 화면보다 앞). D75.
  if (!languageSet) return <LanguagePicker />;
  // 그다음 사용 용도(개인용/업무용). D78.
  if (!usageSet) return <UsagePicker />;

  const locked = !vault?.unlocked || recoveryPending;
  return (
    <ErrorBoundary>
      <Shell />
      {locked && <Lock overlay={!recoveryPending && tabCount > 0} />}
      {!locked && editor?.kind === 'host' && <HostEditor key={`h-${editor.hostId ?? 'new'}`} hostId={editor.hostId} groupId={editor.groupId} />}
      {!locked && editor?.kind === 'folder' && <FolderEditor key={`f-${editor.folderId ?? 'new'}`} folderId={editor.folderId} parentId={editor.parentId} />}
      {!locked && <QuickPalette />}
      {!locked && <RequestDialogs />}
      {!locked && <ErrorBoundary variant="dialog" label={t('settings.title')} onDismiss={() => useUi.getState().closeSettings()}><Settings /></ErrorBoundary>}
      {!locked && <ErrorBoundary variant="dialog" label={t('import.title')} onDismiss={() => useUi.getState().closeImportDialog()}><ImportDialog /></ErrorBoundary>}
      <DialogHost />
      <ContextMenuHost />
      <Toasts />
    </ErrorBoundary>
  );
}
