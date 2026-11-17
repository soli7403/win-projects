// 우측 패널 수용 시나리오(spec/acceptance/04-hosts-sftp-settings-sync.md F1~F11) 테스트.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { setBackend, getBackend } from '../ipc/backend';
import { createFakeBackend } from '../ipc/fakeBackend';
import { useWorkspace } from '../state/workspace';
import { useUi } from '../state/ui';
import { useApp } from '../state/app';
import { useTransfers } from '../state/transfers';
import { newPane, newId } from '../lib/layout';
import { SftpPanel } from './SftpPanel';
import { SnippetsPanel } from './SnippetsPanel';
import { PreviewDialog } from './PreviewDialog';
import { SftpDock, clampWidth } from './RightPanel';
import { sendSnippet } from '../lib/termSend';
import type { SessionInfo } from '../ipc/types';

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** 세션 하나를 포커스된 칸으로 열어 둔다. ssh=false 면 telnet. */
async function openSession(ssh: boolean): Promise<SessionInfo> {
  const b = getBackend();
  const info = await b.invoke<SessionInfo>('session.open', [
    { adhoc: { host: '10.0.0.5', protocol: ssh ? 'ssh' : 'telnet', username: 'demo' } },
  ]);
  await wait(50); // 가짜 백엔드는 30ms 뒤 연결됨으로 바꾼다.
  const connected: SessionInfo = { ...info, state: 'connected', connectedAt: Date.now(), canSftp: ssh };
  useWorkspace.getState().openInNewTab(connected);
  return connected;
}

beforeEach(async () => {
  setBackend(createFakeBackend());
  useWorkspace.setState({ tabs: [], activeTabId: null, sessions: {}, writers: {} });
  useUi.setState({ sftpOpen: [true, false], snippetsOpen: false, preview: null, dialogs: [], menu: null });
  useTransfers.setState({ items: [] });
  useApp.setState({ store: await getBackend().invoke('store.get') });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('F1 · Telnet 창 포커스 → SFTP 안내', () => {
  it('SSH 가 아니면 프로토콜 안내를 보인다', async () => {
    await openSession(false);
    render(<SftpPanel />);
    expect(screen.getByText('파일 창은 SSH·로컬 터미널 세션에서만 사용할 수 있습니다.')).toBeTruthy();
    expect(screen.getByText('(TELNET 세션)')).toBeTruthy();
  });

  it('포커스 세션이 없으면 세션 선택 안내를 보인다', () => {
    render(<SftpPanel />);
    expect(screen.getByText('세션을 선택하면 해당 서버(또는 이 PC)의 파일이 표시됩니다.')).toBeTruthy();
  });
});

describe('F2 · 홈 목록: 폴더 우선, 이름순', () => {
  it('폴더가 먼저 오고 이름 오름차순으로 보인다', async () => {
    await openSession(true);
    render(<SftpPanel />);
    const rows = await screen.findAllByRole('option');
    const names = rows.map((r) => r.querySelector('.name')?.textContent?.replace(/\s+$/, ''));
    // logs·current 가 폴더(current 는 링크 폴더), 그다음 파일 이름순.
    expect(names.slice(0, 2)).toEqual(['current', 'logs']);
    expect(names.slice(2)).toEqual(['app.conf', 'big.txt', 'shot.png']);
  });

  it('머리 줄을 누르면 그 열로 정렬하고, 다시 누르면 방향을 뒤집는다(폴더는 항상 앞)', async () => {
    await openSession(true);
    render(<SftpPanel />);
    await screen.findAllByRole('option');
    const namesNow = () => screen.getAllByRole('option').map((r) => r.querySelector('.name')?.textContent?.replace(/\s+$/, ''));
    const sizeHead = screen.getByRole('button', { name: '크기' });
    const nameHead = screen.getByRole('button', { name: '이름' });
    expect(nameHead.getAttribute('aria-sort')).toBe('ascending');

    fireEvent.click(nameHead);
    expect(nameHead.getAttribute('aria-sort')).toBe('descending');
    expect(namesNow()).toEqual(['logs', 'current', 'shot.png', 'big.txt', 'app.conf']);

    fireEvent.click(sizeHead); // 크기는 큰 것 먼저로 시작
    expect(sizeHead.getAttribute('aria-sort')).toBe('descending');
    expect(nameHead.getAttribute('aria-sort')).toBe('none');
    const files = namesNow().slice(2);
    expect(namesNow().slice(0, 2)).toEqual(['current', 'logs']);
    fireEvent.click(sizeHead);
    expect(namesNow().slice(2)).toEqual([...files].reverse());

    fireEvent.click(nameHead); // 다음 테스트를 위해 기본(이름 오름차순)으로 되돌린다
    expect(nameHead.getAttribute('aria-sort')).toBe('ascending');
  });
});

describe('F3·F4 · 삭제 확인 문구', () => {
  it('여러 항목은 개수 문구, 폴더가 있으면 내용 삭제 경고를 붙인다', async () => {
    const { deleteEntries } = await import('./sftpOps');
    const seen: { body?: string; detail?: string }[] = [];
    useUi.setState({
      confirm: async (opts: { body: string; detail?: string }) => {
        seen.push(opts);
        return false;
      },
    } as never);
    const mk = (name: string, isDir: boolean) => ({ name, path: `/x/${name}`, isDir, isLink: false, size: 0, mtime: 0, mode: 0o644, perm: '-rw-r--r--' });
    await deleteEntries('s-x', [mk('a', false), mk('b', false), mk('c', false)]);
    expect(seen[0].body).toBe('3개 항목을(를) 서버에서 영구 삭제할까요?');
    expect(seen[0].detail).toBeUndefined();
    await deleteEntries('s-x', [mk('logs', true)]);
    expect(seen[1].body).toBe('"logs"을(를) 서버에서 영구 삭제할까요?');
    expect(seen[1].detail).toBe('폴더는 안의 내용까지 모두 삭제됩니다.');
  });
});

describe('F5 · 권한 형식', () => {
  it('8진수 3~4자리만 허용한다', async () => {
    const { isValidMode } = await import('../lib/sftpPath');
    expect(isValidMode('755')).toBe(true);
    expect(isValidMode('0644')).toBe(true);
    expect(isValidMode('999')).toBe(false);
    expect(isValidMode('75')).toBe(false);
  });
});

describe('F6·F7 · 미리보기', () => {
  it('이미지는 그림으로, 512KB 초과 텍스트는 안내 문구로 보인다', async () => {
    const info = await openSession(true);
    useUi.getState().openPreview(info.id, '/home/demo/shot.png');
    const { unmount } = render(<PreviewDialog />);
    await waitFor(() => expect(document.querySelector('img.preview-img')).toBeTruthy());
    unmount();
    useUi.getState().openPreview(info.id, '/home/demo/big.txt');
    render(<PreviewDialog />);
    expect(await screen.findByText('미리보기를 지원하지 않는 파일입니다 (이진 파일 또는 512KB 초과).')).toBeTruthy();
  });
});

describe('F9 · 업로드 완료 시 현재 폴더 목록 갱신', () => {
  it('현재 폴더 아래로 올린 전송이 끝나면 목록을 다시 읽는다', async () => {
    const info = await openSession(true);
    render(<SftpPanel />);
    await screen.findAllByRole('option');
    const b = getBackend();
    const spy = vi.spyOn(b, 'invoke');
    act(() => {
      useTransfers.getState().upsert({
        id: 'tr-test-1', sessionId: info.id, name: 'new.txt', direction: 'upload', bytes: 10, total: 10,
        state: 'done', remotePath: '/home/demo/new.txt',
      });
    });
    await waitFor(() => expect(spy).toHaveBeenCalledWith('sftp.list', [info.id, '/home/demo']));
  });
});

describe('F10 · 명령 모음 저장: 명령이 비면 비활성', () => {
  it('명령이 비어 있으면 저장 버튼이 비활성이다', async () => {
    await openSession(true);
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByRole('button', { name: '새 명령' }));
    const save = screen.getByRole('button', { name: '저장' }) as HTMLButtonElement;
    expect(save.disabled).toBe(true);
    fireEvent.change(screen.getByPlaceholderText('tail -f /var/log/nginx/error.log'), { target: { value: 'uptime' } });
    expect((screen.getByRole('button', { name: '저장' }) as HTMLButtonElement).disabled).toBe(false);
  });

  it('명령 모음이 없으면 안내 문구를 보인다', async () => {
    await openSession(true);
    useApp.setState((s) => (s.store ? { store: { ...s.store, snippets: [] } } : s));
    render(<SnippetsPanel />);
    expect(screen.getByText('자주 쓰는 명령을 저장해 두고 클릭 한 번으로 실행하세요.')).toBeTruthy();
  });
});

describe('F11 · 명령 모음 클릭 → 일괄 입력 탭의 모든 창에 입력', () => {
  it('본문과 Enter 를 모든 연결 세션에 보낸다', async () => {
    const a = await openSession(true);
    const b2 = await openSession(true);
    const tabId = useWorkspace.getState().tabs[0].id;
    // 두 세션을 한 탭의 두 칸으로 합친다.
    useWorkspace.setState((s) => ({
      tabs: s.tabs.map((t) => (t.id === tabId ? { ...t, broadcast: true, root: { type: 'split', id: newId('sp'), dir: 'row', sizes: [50, 50], children: [newPane(a.id), newPane(b2.id)] }, focusedPane: null } : t)),
    }));
    const tab = useWorkspace.getState().tabs.find((t) => t.id === tabId)!;
    const firstPane = (tab.root as { children: { id: string }[] }).children[0].id;
    useWorkspace.getState().focusPane(firstPane);
    const input = vi.spyOn(getBackend(), 'termInput');
    expect(sendSnippet('df -h', true)).toBe(true);
    const calls = input.mock.calls.map((c) => [c[0], c[1]]);
    expect(calls).toEqual(expect.arrayContaining([[a.id, 'df -h\r'], [b2.id, 'df -h\r']]));
  });
});

describe('우측 창 폭', () => {
  it('260~760 으로 자른다', () => {
    expect(clampWidth(100)).toBe(260);
    expect(clampWidth(340)).toBe(340);
    expect(clampWidth(900)).toBe(760);
  });

  it('SFTP 도크는 닫기 버튼으로 그 그룹의 창만 닫는다', () => {
    useUi.setState({ sftpOpen: [true, true] });
    render(<SftpDock group={1} />);
    expect(document.querySelector('.rpanel')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '닫기' }));
    expect(useUi.getState().sftpOpen).toEqual([true, false]);
  });
});
