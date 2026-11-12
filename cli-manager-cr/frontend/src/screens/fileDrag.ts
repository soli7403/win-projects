// 파일 패널 끌어 놓기 복사: 끄는 항목, 탭·칸 위에 머물면 전환, 놓으면 복사 작업 시작.
// 근거: spec/behavior/file-panel.md §2.
import type { DragEvent } from 'react';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { findPane } from '../lib/layout';

export const FILE_MIME = 'application/x-termgo-files';

interface FileDrag {
  sessionId: string;
  /** 끄는 항목들이 있는 폴더 */
  dir: string;
  paths: string[];
}

let g_drag: FileDrag | null = null;

/** 세션별 파일 패널에서 마지막으로 본 폴더(패널을 닫아도 유지). 탭·칸에 바로 놓을 때의 대상. */
export const lastDir = new Map<string, string>();

export function isFileDrag(e: DragEvent): boolean {
  return !!g_drag && Array.from(e.dataTransfer.types).includes(FILE_MIME);
}

export function startFileDrag(e: DragEvent, d: FileDrag): void {
  g_drag = d;
  e.dataTransfer.setData(FILE_MIME, '1');
  e.dataTransfer.effectAllowed = 'copy';
}

// 끌기 시작한 행은 탭이 바뀌면 사라져 dragend 를 못 받을 수 있어 창에서 정리한다.
function endDrag(): void {
  setTimeout(() => {
    g_drag = null;
  }, 0);
  hoverEnd();
}
if (typeof window !== 'undefined') {
  window.addEventListener('dragend', endDrag);
  window.addEventListener('drop', endDrag);
}

let g_hoverKey = '';
let g_hoverTimer: ReturnType<typeof setTimeout> | undefined;

/** 끄는 중 같은 대상 위에 0.5초 머물면 fn 을 부른다(탭 전환·칸 포커스). */
export function hoverStart(key: string, fn: () => void): void {
  if (g_hoverKey === key) return;
  hoverEnd();
  g_hoverKey = key;
  g_hoverTimer = setTimeout(fn, 500);
}

export function hoverEnd(key?: string): void {
  if (key && key !== g_hoverKey) return;
  clearTimeout(g_hoverTimer);
  g_hoverKey = '';
}

/** 파일 패널을 쓸 수 있는 세션(연결된 SSH 또는 로컬 터미널)인지. */
export function canBrowse(sid: string | null | undefined): boolean {
  const info = sid ? useWorkspace.getState().sessions[sid]?.info : undefined;
  return !!info && info.state === 'connected' && ((info.canSftp && info.protocol === 'ssh') || info.protocol === 'local');
}

/** 탭의 포커스 칸 세션. */
export function tabSessionId(tabId: string): string | null {
  const t = useWorkspace.getState().tabs.find((x) => x.id === tabId);
  if (!t?.focusedPane) return null;
  return findPane(t.root, t.focusedPane)?.sessionId || null;
}

/**
 * 끄는 항목을 dstSid 의 dir 로 복사하는 작업을 시작한다(dir 이 비면 그 세션의 마지막 폴더, 없으면 시작 폴더).
 * 같은 이름이 있으면 확인 후 덮어쓴다. 진행은 창 맨 아래 복사 목록에 한 줄로 나온다.
 */
export async function dropFiles(dstSid: string | null, dir: string): Promise<void> {
  const d = g_drag;
  g_drag = null;
  hoverEnd();
  if (!d || d.paths.length === 0 || !dstSid) return;
  const toast = useApp.getState().pushToast;
  if (!canBrowse(dstSid)) {
    toast({ kind: 'error', text: '이 탭에는 복사할 수 없습니다 (연결된 SSH 또는 로컬 터미널만 가능)' });
    return;
  }
  const target = dir || lastDir.get(dstSid) || '';
  // 같은 폴더에 놓거나 끌던 폴더 자신 위에 놓으면 무시한다.
  if (d.sessionId === dstSid && (d.dir === target || d.paths.includes(target))) return;
  try {
    const b = getBackend();
    const r = await b.invoke<{ jobId: string; conflicts: string[] }>('sftp.copy', [d.sessionId, d.paths, dstSid, target, false]);
    if (r.conflicts.length > 0) {
      const names = r.conflicts.slice(0, 5).join(', ') + (r.conflicts.length > 5 ? ` 외 ${r.conflicts.length - 5}개` : '');
      const ok = await useUi.getState().confirm({
        title: '덮어쓰기',
        body: `대상 폴더에 같은 이름이 있습니다: ${names}`,
        detail: '덮어쓸까요?',
        okText: '덮어쓰기',
        danger: true,
      });
      if (ok) await b.invoke('sftp.copy', [d.sessionId, d.paths, dstSid, target, true]);
    }
  } catch (e) {
    toast({ kind: 'error', text: `복사 실패: ${errorMessage(e)}` });
  }
}

export function cancelCopy(jobId: string): void {
  void getBackend().invoke('sftp.cancelCopy', [jobId]).catch(() => undefined);
}
