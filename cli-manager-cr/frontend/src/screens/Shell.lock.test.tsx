// 앱 잠그기 위치(D67, spec/acceptance S2): 왼쪽 위 아이콘 묶음에는 잠그기 버튼이 없고, 상태 표시줄의 "잠금 해제됨"을 누르면 잠근다.
import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { StatusBar, ToolRow } from './Shell';

function mockBackend() {
  const calls: string[] = [];
  const b: Backend = {
    async invoke(channel: string): Promise<any> {
      calls.push(channel);
      return undefined;
    },
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  return calls;
}

function seed(unlocked: boolean) {
  useApp.setState({
    booted: true,
    vault: { initialized: true, unlocked, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
  });
}

describe('앱 잠그기 위치 (D67)', () => {
  it('왼쪽 위 아이콘 묶음에는 잠그기 버튼이 없다 (가로·세로 모두)', () => {
    mockBackend();
    seed(true);
    const { unmount } = render(<ToolRow vertical={false} />);
    expect(screen.queryByRole('button', { name: /잠그기/ })).toBeNull();
    unmount();
    render(<ToolRow vertical />);
    expect(screen.queryByRole('button', { name: /잠그기/ })).toBeNull();
  });

  it('상태 표시줄의 "잠금 해제됨"을 누르면 vault.lock 을 부른다', async () => {
    const calls = mockBackend();
    seed(true);
    render(<StatusBar />);
    const btn = screen.getByRole('button', { name: /잠금 해제됨/ });
    expect(btn.getAttribute('title')).toBe('누르면 앱을 잠급니다');
    fireEvent.click(btn);
    await waitFor(() => expect(calls).toContain('vault.lock'));
  });

  it('잠겨 있으면 "앱 잠김" 표시만 있고 누를 버튼이 없다', () => {
    mockBackend();
    seed(false);
    render(<StatusBar />);
    expect(screen.getByText('앱 잠김')).toBeTruthy();
    expect(screen.queryByRole('button', { name: /잠금 해제됨/ })).toBeNull();
  });
});
