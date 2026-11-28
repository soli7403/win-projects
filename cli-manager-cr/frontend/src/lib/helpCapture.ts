// 도움말 그림 자동 캡처(frontend/scripts/help-capture.mjs, D158)가 화면을 여는 손잡이.
// 화면 저장소에 dariterm.helpCapture=1 이 있을 때만 window.__dtHelpCapture 를 만든다. 캡처 스크립트는
// vite 개발 서버 + 가짜 백엔드(ipc/fakeBackend.ts)를 Playwright 의 빈 임시 Edge 프로필로 열 때만 이 값을 넣으므로
// 앱(WebView2)에는 생기지 않는다. 손잡이는 이미 페이지 안의 상태(zustand 저장소)를 가리킬 뿐이라 새 권한을 주지 않는다.
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { saveSettings } from '../state/settings';
import { openHost } from '../state/actions';

export const HELP_CAPTURE_KEY = 'dariterm.helpCapture';

export interface HelpCaptureHandle {
  ui: typeof useUi;
  app: typeof useApp;
  saveSettings: typeof saveSettings;
  /** 백엔드 채널 부르기(캡처 때는 가짜 백엔드). */
  invoke: (channel: string, args?: unknown[]) => Promise<unknown>;
  /** 서버를 새 탭으로 연다(가짜 백엔드는 흉내 터미널). */
  openHost: (hostId: string) => Promise<void>;
}

/** 캡처 모드면 손잡이를 window 에 단다. 저장소를 읽을 수 없으면 아무것도 하지 않는다. 달았으면 true. */
export function installHelpCapture(target: Record<string, unknown> = globalThis as unknown as Record<string, unknown>): boolean {
  let on = false;
  try {
    on = globalThis.localStorage?.getItem(HELP_CAPTURE_KEY) === '1';
  } catch {
    on = false;
  }
  if (!on) return false;
  const handle: HelpCaptureHandle = {
    ui: useUi,
    app: useApp,
    saveSettings,
    invoke: (channel, args = []) => getBackend().invoke(channel, args),
    openHost: (hostId) => openHost(hostId),
  };
  target.__dtHelpCapture = handle;
  return true;
}
