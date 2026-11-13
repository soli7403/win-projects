// 이미지·파일 붙여넣기 업로드와 경로 입력. 근거: spec/behavior/terminal.md §4·§5, spec/ui/panes.md §2.5.
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from './app';
import { useWorkspace } from './workspace';

const SPECIAL = /[\s'"$`\\()&;|<>*?!#]/;

/** 경로 하나를 입력 형식으로 만든다. 따옴표 설정이 켜져 있으면 특수문자가 있을 때 작은따옴표로 감싼다. */
export function quotePath(p: string, quote: boolean): string {
  if (!quote || !SPECIAL.test(p)) return p;
  return `'${p.replace(/'/g, `'\\''`)}'`;
}

/** 경로들을 붙여넣기 방식(bracketed paste)으로 대상 세션에 입력한다. 여러 경로는 공백으로 잇고 끝에 공백 하나. */
export function insertPaths(targets: string[], paths: string[]): void {
  const quote = !!useApp.getState().store?.settings.insertPathQuote;
  const text = `${paths.map((p) => quotePath(p, quote)).join(' ')} `;
  for (const sid of targets) getBackend().termInput(sid, `\x1b[200~${text}\x1b[201~`);
}

function labelOf(kind: 'image' | 'files', count: number, name?: string): string {
  if (kind === 'image') return '클립보드 이미지';
  if (count === 1 && name) return name;
  return `파일 ${count}개`;
}

/** 세션 종류에 따라 업로드 또는 경로만 입력(terminal.md §5). */
export async function uploadAndInsert(
  sessionId: string,
  targets: string[],
  original: { kind: 'clipboardImage' } | { kind: 'files'; paths: string[] },
): Promise<void> {
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  const isLocal = info?.protocol === 'local';
  const isSftp = !!info?.canSftp && info?.state === 'connected' && info.protocol === 'ssh';
  const pushToast = useApp.getState().pushToast;
  const kind = original.kind === 'clipboardImage' ? 'image' : 'files';
  const count = original.kind === 'files' ? original.paths.length : 1;
  const name = original.kind === 'files' && count === 1 ? original.paths[0].split(/[\\/]/).pop() : undefined;
  const label = labelOf(kind, count, name);
  try {
    if (isLocal) {
      const paths = await getBackend().invoke<string[]>('local.saveForPrompt', [original]);
      insertPaths(targets, paths);
      return;
    }
    if (isSftp) {
      pushToast({ kind: 'info', text: `${label} 서버로 전송 중...` });
      const paths = await getBackend().invoke<string[]>('sftp.uploadForPrompt', [sessionId, original]);
      pushToast({ kind: 'ok', text: `${label} 전송 완료 → ${paths.join(', ')}` });
      insertPaths(targets, paths);
      return;
    }
    if (kind === 'image') {
      pushToast({ kind: 'error', text: '이미지 전송은 연결된 SSH 세션에서만 가능합니다' });
      return;
    }
    // 파일은 올리지 않고 원래 경로를 입력한다.
    if (original.kind === 'files') insertPaths(targets, original.paths);
  } catch (e) {
    pushToast({ kind: 'error', text: `전송 실패: ${errorMessage(e)}` });
  }
}

/** Ctrl+Alt+V: 클립보드 이미지를 강제로 올린다. 이미지가 없으면 토스트. */
export async function forceImagePaste(sessionId: string, targets: string[]): Promise<void> {
  const pushToast = useApp.getState().pushToast;
  try {
    const has = await getBackend().invoke<boolean>('clip.hasImage');
    if (!has) {
      pushToast({ kind: 'info', text: '클립보드에 이미지가 없습니다' });
      return;
    }
    await uploadAndInsert(sessionId, targets, { kind: 'clipboardImage' });
  } catch (e) {
    pushToast({ kind: 'error', text: errorMessage(e) });
  }
}

/** Ctrl+V 붙여넣기 흐름: 클립보드 종류(clip.info)에 따라 파일·이미지·텍스트 처리. 텍스트는 호출자가 처리한다. */
export async function clipboardKind(): Promise<{ type: 'files' | 'text' | 'image' | 'empty'; files?: string[]; text?: string }> {
  return getBackend().invoke('clip.info');
}
