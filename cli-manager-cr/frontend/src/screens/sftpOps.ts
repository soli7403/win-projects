// SFTP 동작(백엔드 호출 + 토스트·확인 흐름). 근거: spec/ui/sftp-snippets.md §2.2~§2.5, spec/ipc.md §2.
import { getBackend, errorMessage } from '../ipc/backend';
import type { SessionInfo } from '../ipc/types';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { sendText } from '../lib/termSend';
import { cdCommandFor, isValidMode, joinWith, quoteFor } from '../lib/sftpPath';

export interface Entry {
  name: string;
  path: string;
  isDir: boolean;
  isLink: boolean;
  size: number;
  mtime: number;
  /** 권한 하위 9비트 숫자(예 0o755). */
  mode: number;
  /** 권한 문자열(drwxr-xr-x 형식). */
  perm: string;
}

export interface Listing {
  path: string;
  entries: Entry[];
  /** 상위 폴더(루트면 path 와 같음)·경로 구분자·이 PC 여부 (file-panel.md §5) */
  parent?: string;
  sep?: string;
  local?: boolean;
}

const toastErr = (e: unknown) => useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
const toastOk = (text: string) => useApp.getState().pushToast({ kind: 'ok', text });
const toastInfo = (text: string) => useApp.getState().pushToast({ kind: 'info', text });

export async function sftpHome(sid: string): Promise<string> {
  return getBackend().invoke<string>('sftp.home', [sid]);
}

export async function sftpList(sid: string, path: string): Promise<Listing> {
  return getBackend().invoke<Listing>('sftp.list', [sid, path]);
}

/** 세션 정보 조회: 포커스된 세션의 현재 상태. */
export function sessionInfo(sid: string | null): SessionInfo | undefined {
  return sid ? useWorkspace.getState().sessions[sid]?.info : undefined;
}

/** 새 폴더. 빈 이름이면 아무 일도 없다. 성공 시 true. */
export async function makeFolder(sid: string, dir: string, sep = '/'): Promise<boolean> {
  const name = await useUi.getState().prompt({ title: '새 폴더', label: '새 폴더 이름', okText: '만들기' });
  if (name === null || !name.trim()) return false;
  try {
    await getBackend().invoke('sftp.mkdir', [sid, joinWith(dir, name.trim(), sep)]);
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 업로드 대화상자 → 올리기. 올린 개수가 있으면 true(호출부가 새로고침). */
export async function uploadViaPicker(sid: string, dir: string): Promise<boolean> {
  try {
    const paths = await getBackend().invoke<string[]>('sftp.uploadPick', [sid, dir]);
    return Array.isArray(paths) && paths.length > 0;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 끌어 놓기로 받은 로컬 경로를 올린다. 경로가 없으면 아무 일도 없다(spec §2.3). */
export async function uploadLocal(sid: string, dir: string, localPaths: string[]): Promise<boolean> {
  if (localPaths.length === 0) return false;
  try {
    await getBackend().invoke('sftp.upload', [sid, localPaths, dir]);
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 다운로드: 폴더 선택 대화상자(백엔드) 후 저장. 취소면 아무 토스트도 없다. */
export async function downloadPaths(sid: string, paths: string[], local = false): Promise<void> {
  try {
    const saved = await getBackend().invoke<string | false>('sftp.download', [sid, paths]);
    if (typeof saved === 'string' && saved) toastOk(`${local ? '복사' : '다운로드'} 완료 → ${saved}`);
  } catch (e) {
    toastErr(e);
  }
}

/** 로컬 앱으로 편집: 임시로 받아 기본 앱으로 연다(저장 시 백엔드가 올린다). */
export async function editLocally(sid: string, path: string, local = false): Promise<boolean> {
  try {
    await getBackend().invoke('sftp.edit', [sid, path]);
    // 로컬 터미널은 그 파일을 기본 앱으로 열 뿐이라 안내하지 않는다 (file-panel.md §1).
    if (!local) toastInfo('편집 후 저장하면 서버에 자동 업로드됩니다');
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 이름 바꾸기. 같은 폴더(dir = 패널의 현재 폴더) 안에서 바꾼다. 성공 시 true. */
export async function renameEntry(sid: string, entry: Entry, dir: string, sep = '/'): Promise<boolean> {
  const next = await useUi.getState().prompt({ title: '이름 바꾸기', label: '이름', value: entry.name, okText: '바꾸기' });
  if (next === null || !next.trim() || next.trim() === entry.name) return false;
  try {
    await getBackend().invoke('sftp.rename', [sid, entry.path, joinWith(dir, next.trim(), sep)]);
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 권한 변경. 형식이 [0-7]{3,4} 가 아니면 적용하지 않는다(F5). */
export async function chmodEntry(sid: string, entry: Entry): Promise<boolean> {
  const current = (entry.mode & 0o777).toString(8).padStart(3, '0');
  const v = await useUi.getState().prompt({ title: '권한 변경', label: '권한 (8진수, 예: 755)', value: current, okText: '적용' });
  if (v === null || !isValidMode(v.trim())) return false;
  try {
    await getBackend().invoke('sftp.chmod', [sid, entry.path, parseInt(v.trim(), 8)]);
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 삭제 확인 후 삭제. 성공 시 true. */
export async function deleteEntries(sid: string, entries: Entry[], local = false): Promise<boolean> {
  const n = entries.length;
  const anyDir = entries.some((e) => e.isDir);
  const title = '삭제';
  const where = local ? '이 PC에서' : '서버에서';
  const body = n === 1 ? `"${entries[0].name}"을(를) ${where} 영구 삭제할까요?` : `${n}개 항목을(를) ${where} 영구 삭제할까요?`;
  const ok = await useUi.getState().confirm({
    title,
    body,
    detail: anyDir ? '폴더는 안의 내용까지 모두 삭제됩니다.' : undefined,
    okText: '삭제',
    danger: true,
  });
  if (!ok) return false;
  try {
    await getBackend().invoke('sftp.remove', [sid, entries.map((e) => e.path)]);
    return true;
  } catch (e) {
    toastErr(e);
    return false;
  }
}

/** 터미널에 경로들을 입력(공백으로 잇고 끝에 공백). 포커스 칸 대상. */
export function typePaths(paths: string[], sep = '/'): void {
  sendText(paths.map((p) => quoteFor(p, sep)).join(' ') + ' ');
}

/** 터미널에서 cd. 목록 행(폴더) 또는 현재 폴더. */
export function cdTerminal(path: string, sep = '/'): void {
  sendText(`${cdCommandFor(path, sep)}\r`);
}

/** 경로 복사(줄바꿈으로 이어 클립보드). */
export async function copyPaths(paths: string[]): Promise<void> {
  try {
    await getBackend().invoke('clip.writeText', [paths.join('\n')]);
  } catch (e) {
    toastErr(e);
  }
}
