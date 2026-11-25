// 왼쪽 열 너비(D130): 너비는 서버 목록이 아니라 왼쪽 열 전체가 갖는다. 명령 모음을 켜도 열이 넓어지지 않고,
// 너비를 끌어 바꾸면 서버 목록과 명령 모음이 함께 바뀐다. 근거: spec/ui/sidebar.md §1, spec/ui/sftp-snippets.md §1.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { Shell } from './Shell';
const invoke = vi.fn(async (channel: string) => (channel === 'store.get' ? structuredClone(useApp.getState().store) : undefined));

function seed(sidebarWidth: number, snippetsOpen: boolean) {
  invoke.mockClear();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({
    booted: true,
    vault: { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
    store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, sidebarWidth } } as Store,
  });
  useUi.setState({ sidebarOpen: true, snippetsOpen });
}

beforeEach(() => seed(290, true));

describe('왼쪽 열 너비 (D130)', () => {
  it('너비는 왼쪽 열이 갖고, 서버 목록에는 따로 너비를 주지 않는다', () => {
    const { container } = render(<Shell />);
    const leftcol = container.querySelector<HTMLElement>('.leftcol')!;
    expect(leftcol.style.width).toBe('290px');
    expect(container.querySelector<HTMLElement>('.leftcol .sidebar')!.style.width).toBe('');
    expect(container.querySelector<HTMLElement>('.leftcol .snippet-dock')!.style.width).toBe('');
  });

  it('명령 모음을 켜고 꺼도 왼쪽 열 너비는 같다', () => {
    const { container } = render(<Shell />);
    expect(container.querySelector('.snippet-dock')).toBeTruthy();
    expect(container.querySelector<HTMLElement>('.leftcol')!.style.width).toBe('290px');
    act(() => useUi.getState().setSnippetsOpen(false));
    expect(container.querySelector('.snippet-dock')).toBeNull();
    expect(container.querySelector<HTMLElement>('.leftcol')!.style.width).toBe('290px');
  });

  it('너비 조절 손잡이는 서버 목록 안이 아니라 왼쪽 열에 있다(명령 모음 옆에서도 잡힌다)', () => {
    const { container } = render(<Shell />);
    const handle = container.querySelector('.side-resize')!;
    expect(handle.parentElement?.classList.contains('leftcol')).toBe(true);
    expect(container.querySelector('.sidebar .side-resize')).toBeNull();
    expect(container.querySelectorAll('.side-resize')).toHaveLength(1);
  });

  it('손잡이를 끌면 열 너비가 따라 바뀌고(200~520), 놓을 때 한 번 저장한다', () => {
    const { container } = render(<Shell />);
    const leftcol = container.querySelector<HTMLElement>('.leftcol')!;
    const handle = container.querySelector('.side-resize')!;
    // jsdom 에는 PointerEvent 가 없어 같은 이름의 MouseEvent 로 보낸다.
    fireEvent(handle, new MouseEvent('pointerdown', { bubbles: true, clientX: 290 }));
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 350 })); });
    expect(leftcol.style.width).toBe('350px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 2000 })); });
    expect(leftcol.style.width).toBe('520px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 0 })); });
    expect(leftcol.style.width).toBe('200px');
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 330 })); });
    expect(invoke.mock.calls.filter((c) => c[0] === 'store.patch')).toHaveLength(0); // 끄는 동안은 저장하지 않는다
    act(() => { window.dispatchEvent(new MouseEvent('pointerup', { clientX: 330 })); });
    const patches = invoke.mock.calls.filter((c) => c[0] === 'store.patch') as unknown as [string, [{ settings: { sidebarWidth: number } }]][];
    expect(patches).toHaveLength(1);
    expect(patches[0][1][0].settings.sidebarWidth).toBe(330);
    // 놓은 뒤의 움직임은 반영하지 않는다.
    act(() => { window.dispatchEvent(new MouseEvent('pointermove', { clientX: 500 })); });
    expect(useApp.getState().store?.settings.sidebarWidth).toBe(330);
  });

  it('끌지 않고 눌렀다 떼기만 하면 저장하지 않는다', () => {
    const { container } = render(<Shell />);
    fireEvent(container.querySelector('.side-resize')!, new MouseEvent('pointerdown', { bubbles: true, clientX: 290 }));
    act(() => { window.dispatchEvent(new MouseEvent('pointerup', { clientX: 290 })); });
    expect(invoke.mock.calls.filter((c) => c[0] === 'store.patch')).toHaveLength(0);
  });

  it('서버 목록을 접으면 왼쪽 열 대신 세로 막대가 남는다(너비 손잡이 없음)', () => {
    const { container } = render(<Shell />);
    act(() => useUi.setState({ sidebarOpen: false }));
    expect(container.querySelector('.leftcol')).toBeNull();
    expect(container.querySelector('.rail')).toBeTruthy();
    expect(container.querySelector('.side-resize')).toBeNull();
  });
});
