// 포커스된 칸(동시 입력 규칙 포함)으로 텍스트를 보낸다. 근거: spec/behavior/terminal.md §7, spec/ui/panes.md §5(동시 입력).
// TerminalView 안의 입력 대상 규칙과 같은 내용을 여기서 직접 구현한다(그 함수는 칸 안에 갇혀 있어 호출하지 않는다).
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useWorkspace } from '../state/workspace';
import { findPane, sessionsIn } from './layout';

/** 포커스된 칸의 세션 id. 없으면 null. */
export function focusedSessionId(): string | null {
  const s = useWorkspace.getState();
  const tab = s.tabs.find((t) => t.id === s.activeTabId);
  if (!tab || !tab.focusedPane) return null;
  return findPane(tab.root, tab.focusedPane)?.sessionId ?? null;
}

/** 입력 대상 세션 목록: 동시 입력이 켜진 탭이면 탭의 연결된 모든 세션, 아니면 포커스 세션만. */
export function inputTargets(): string[] {
  const s = useWorkspace.getState();
  const tab = s.tabs.find((t) => t.id === s.activeTabId);
  const sid = focusedSessionId();
  if (!tab || !sid) return [];
  if (!tab.broadcast) return [sid];
  return sessionsIn(tab.root).filter((x) => !!s.sessions[x]);
}

/** 스니펫 본문 입력. 줄바꿈은 CR 로 바꾸고, sendEnter 면 끝에 CR 을 더한다. 포커스 세션이 없으면 false. */
export function sendSnippet(body: string, sendEnter: boolean): boolean {
  const sid = focusedSessionId();
  if (!sid) return false;
  const text = body.replace(/\r?\n/g, '\r') + (sendEnter ? '\r' : '');
  for (const target of inputTargets()) getBackend().termInput(target, text);
  focusTerminal();
  return true;
}

/** 원문 텍스트를 입력 대상에 보내고 터미널에 포커스(경로 입력·cd 용). 포커스 세션이 없으면 토스트. */
export function sendText(text: string): boolean {
  if (!focusedSessionId()) {
    useApp.getState().pushToast({ kind: 'info', text: '먼저 세션을 선택하세요' });
    return false;
  }
  for (const target of inputTargets()) getBackend().termInput(target, text);
  focusTerminal();
  return true;
}

/** 현재 포커스된 칸의 터미널에 키보드 포커스를 준다(xterm 의 입력 요소). */
export function focusTerminal(): void {
  const pane = document.querySelector<HTMLElement>('.pane.focused .xterm-helper-textarea');
  pane?.focus();
}

export function notifyNoFocus(): void {
  useApp.getState().pushToast({ kind: 'info', text: '먼저 세션을 선택하세요' });
}
