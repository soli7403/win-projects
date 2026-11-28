// 설정의 "단축키" 구역(D126): 정보에서 떼어 낸 구역, 전체 켜기/끄기, 키 바꾸기(겹치면 먼저 것을 빈 값으로 저장), 기본값으로.
// 근거: spec/ui/settings.md §2a·§8, spec/behavior/shortcuts.md §5.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import { keyCapture } from '../lib/keys';
import { startShortcutRows } from '../lib/shortcuts';
import { activeShortcut } from '../state/shortcuts';
import type { Settings as SettingsT, Store } from '../ipc/types';
import { Settings } from './Settings';
import { Welcome } from './Welcome';

const invoke = vi.fn(async () => undefined);

function setStore(settings: Partial<SettingsT> = {}) {
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, ...settings } } as Store });
}

beforeEach(() => {
  invoke.mockClear();
  keyCapture.active = false;
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({ toasts: [] });
  useUi.setState({ settings: { open: false, section: 'appearance' } });
  setStore();
});

function openSection(section: 'shortcuts' | 'about') {
  useUi.setState({ settings: { open: true, section } });
  return render(<Settings />);
}

/** store.patch 로 보낸 설정 값들(보낸 순서대로). */
function patched(key: string): unknown[] {
  return invoke.mock.calls
    .filter((c) => (c as unknown[])[0] === 'store.patch')
    .map((c) => ((c as unknown[])[1] as [{ settings: Record<string, unknown> }])[0].settings[key]);
}

/** 그 명령의 키 칸(이름: 키). */
const keyBox = (name: string, keys: string) => screen.getByRole('button', { name: `${name}: ${keys}` });

describe('설정 > 단축키 (D126)', () => {
  it('구역 목록에 "입력" 다음으로 "단축키", 그다음 "출력 규칙"이 있다 (11개)', () => {
    openSection('shortcuts');
    const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
    expect(names).toEqual(['화면', '입력', '단축키', '출력 규칙', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
  });

  it('"정보"에는 단축키 표가 없다', async () => {
    invoke.mockImplementation((async (ch: string) => (ch === 'app.info' ? { version: '1.0.0', dataDir: 'C:\\data' } : undefined)) as never);
    openSection('about');
    await screen.findByText('데이터 폴더');
    expect(screen.queryByText('터미널에서 찾기')).toBeNull();
    expect(screen.queryByText('Ctrl+Shift+F')).toBeNull();
    invoke.mockImplementation(async () => undefined);
  });

  it('명령 37개가 기본 키와 함께 보이고, 터미널 안의 고정 키 3개는 따로 보인다', () => {
    const { container } = openSection('shortcuts');
    expect(container.querySelectorAll('.sc-edit-row')).toHaveLength(37);
    expect(screen.getByText('마지막 매크로 다시 실행')).toBeTruthy();
    expect(keyBox('빠른 연결 / 명령', 'Ctrl+K')).toBeTruthy();
    expect(keyBox('이전 탭', 'Ctrl+Shift+Tab')).toBeTruthy();
    expect(keyBox('3번째 탭', 'Ctrl+3')).toBeTruthy();
    expect(keyBox('위쪽 창으로', 'Alt+↑')).toBeTruthy();
    expect(keyBox('창 최대화/복원', 'F11')).toBeTruthy();
    expect(screen.getByText('터미널 안에서 쓰는 키')).toBeTruthy();
    expect(screen.getByText('Ctrl+V')).toBeTruthy();
    expect(screen.getByText('Ctrl+Alt+V')).toBeTruthy();
    // 기본 상태에서는 되돌릴 것이 없다.
    expect((screen.getByRole('button', { name: '모두 기본값으로' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('"단축키 사용"은 기본 켜짐이고, 끄면 shortcutsEnabled: false 로 저장된다', async () => {
    expect(DEFAULT_SETTINGS.shortcutsEnabled).toBe(true);
    openSection('shortcuts');
    const sw = screen.getByRole('switch', { name: '단축키 사용' });
    expect(sw.getAttribute('aria-checked')).toBe('true');
    fireEvent.click(sw);
    await waitFor(() => expect(patched('shortcutsEnabled')).toEqual([false]));
  });

  it('키 칸을 누르고 새 키를 누르면 그 명령만 바뀌어 저장된다', async () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    expect(keyBox('터미널에서 찾기', '키를 누르세요…')).toBeTruthy();
    expect(keyCapture.active).toBe(true);
    // 조합 키만 누른 것은 기다린다.
    fireEvent.keyDown(window, { code: 'ControlLeft', key: 'Control', ctrlKey: true });
    expect(patched('shortcuts')).toEqual([]);
    fireEvent.keyDown(window, { code: 'KeyF', key: 'f', ctrlKey: true, altKey: true });
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: 'Ctrl+Alt+KeyF' }]));
    expect(keyCapture.active).toBe(false);
  });

  it('다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장하고 알린다', async () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'KeyK', key: 'k', ctrlKey: true });
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: 'Ctrl+KeyK', quick: '' }]));
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('Ctrl+K 를 쓰던 "빠른 연결 / 명령"의 단축키를 비웠습니다');
  });

  it('비워진 명령은 "없음"으로 보이고, 기본값으로 누르면 되찾으며 그 키를 쓰던 명령이 비워진다', async () => {
    setStore({ shortcuts: { find: 'Ctrl+KeyK', quick: '' } });
    openSection('shortcuts');
    expect(keyBox('빠른 연결 / 명령', '없음')).toBeTruthy();
    expect(keyBox('터미널에서 찾기', 'Ctrl+K')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '빠른 연결 / 명령: 기본값으로' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: '' }]));
  });

  it('[키 지우기]는 빈 값으로 저장한다', async () => {
    setStore({ shortcuts: { sftp: 'Ctrl+Alt+KeyS' } });
    openSection('shortcuts');
    fireEvent.click(screen.getByRole('button', { name: 'SFTP 창: 키 지우기' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ sftp: '' }]));
  });

  it('[모두 기본값으로]는 빈 표로 저장한다', async () => {
    // (가짜 백엔드는 저장 뒤 설정을 다시 돌려주지 않으므로 따로 연다.)
    setStore({ shortcuts: { sftp: 'Ctrl+Alt+KeyS' } });
    openSection('shortcuts');
    fireEvent.click(screen.getByRole('button', { name: '모두 기본값으로' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{}]));
  });

  it('고를 수 없는 키는 저장하지 않고 까닭을 알린 뒤 계속 기다린다', () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'KeyA', key: 'a' });
    fireEvent.keyDown(window, { code: 'KeyV', key: 'v', ctrlKey: true });
    expect(patched('shortcuts')).toEqual([]);
    const texts = useApp.getState().toasts.map((x) => x.text);
    expect(texts).toContain('Ctrl 또는 Alt 와 함께 누르는 키나 F1~F12 만 쓸 수 있습니다');
    expect(texts).toContain('Ctrl+V 는 터미널의 복사·붙여넣기에 쓰는 키라 고를 수 없습니다');
    expect(keyBox('터미널에서 찾기', '키를 누르세요…')).toBeTruthy();
  });

  it('키를 받는 중의 Esc 는 받기만 취소하고 설정 창은 닫지 않는다', () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'Escape', key: 'Escape' });
    expect(useUi.getState().settings.open).toBe(true);
    expect(keyBox('터미널에서 찾기', 'Ctrl+Shift+F')).toBeTruthy();
    expect(patched('shortcuts')).toEqual([]);
    // 받는 중이 아니면 Esc 는 전처럼 설정을 닫는다.
    fireEvent.keyDown(window, { code: 'Escape', key: 'Escape' });
    expect(useUi.getState().settings.open).toBe(false);
  });
});

describe('저장된 단축키 설정이 키 처리에 쓰인다 (D126)', () => {
  const ev = (code: string, mods: { ctrlKey?: boolean; shiftKey?: boolean; altKey?: boolean }) =>
    ({ code, key: '', ctrlKey: false, shiftKey: false, altKey: false, ...mods });

  it('기본 설정이면 기본 키가 앱 단축키다', () => {
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('quickToggle');
  });

  it('전체를 끄면 어떤 키도 앱 단축키가 아니다(터미널이 받는다)', () => {
    setStore({ shortcutsEnabled: false });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))).toBeNull();
    expect(activeShortcut(ev('F11', {}))).toBeNull();
    expect(activeShortcut(ev('ArrowUp', { altKey: true }))).toBeNull();
  });

  it('바꾼 키는 새 키로만, 비운 명령은 어떤 키로도 동작하지 않는다', () => {
    setStore({ shortcuts: { find: 'Ctrl+KeyK', quick: '' } });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('find');
    expect(activeShortcut(ev('KeyF', { ctrlKey: true, shiftKey: true }))).toBeNull();
  });

  it('옛 저장 파일처럼 값이 없으면 켜진 것으로 본다', () => {
    setStore({ shortcutsEnabled: undefined as unknown as boolean, shortcuts: undefined as unknown as Record<string, string> });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('quickToggle');
  });
});

describe('시작 화면의 단축키 표는 지금 키를 보인다 (D126)', () => {
  it('기본: 12줄, 묶음 줄은 묶음 표기', () => {
    render(<Welcome />);
    const rows = [...document.querySelectorAll('.shortcut-table > div')].map((d) => d.textContent);
    expect(rows).toHaveLength(12);
    expect(rows).toContain('창 이동Alt+방향키');
    expect(rows).toContain('탭 이동Ctrl+Tab / Ctrl+1~9');
    expect(rows).toContain('빠른 연결 / 명령Ctrl+K');
    expect(screen.getAllByText('Ctrl+K').length).toBe(2); // 안내 문장 + 표
  });

  it('바꾼 키를 보이고, 비운 줄은 뺀다', () => {
    setStore({ shortcuts: { quick: 'Ctrl+Alt+KeyK', newLocal: '', focusUp: 'Ctrl+Alt+ArrowUp' } });
    render(<Welcome />);
    const rows = [...document.querySelectorAll('.shortcut-table > div')].map((d) => d.textContent);
    expect(rows).toContain('빠른 연결 / 명령Ctrl+Alt+K');
    expect(rows.some((r) => r?.startsWith('새 로컬 터미널'))).toBe(false);
    expect(rows).toContain('창 이동Ctrl+Alt+↑ …');
  });

  it('전체를 끄면 고정 키 3줄만 남고 안내 문장의 키도 없다', () => {
    expect(startShortcutRows({}).map((r) => r.keys)).toEqual(['Ctrl+V', 'Ctrl+Alt+V', 'Ctrl+Shift+C / 선택']);
    setStore({ shortcutsEnabled: false });
    render(<Welcome />);
    expect(document.querySelectorAll('.shortcut-table > div')).toHaveLength(3);
    expect(screen.queryByText('Ctrl+K')).toBeNull();
  });
});
