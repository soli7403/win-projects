// 터미널 유지: 칸이 다시 그려져도(탭 전환·분할) 같은 xterm 을 다시 붙이고, 세션이 닫힐 때만 버린다.
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useWorkspace } from '../state/workspace';
import type { SessionInfo } from '../ipc/types';

const created: FakeTerm[] = [];

class FakeTerm {
  options: Record<string, unknown> = {};
  cols = 80;
  rows = 24;
  modes = { bracketedPasteMode: false };
  disposed = false;
  written: string[] = [];
  parser = { registerOscHandler: () => ({ dispose() {} }) };
  constructor() {
    created.push(this);
  }
  loadAddon() {}
  open(el: HTMLElement) {
    const d = document.createElement('div');
    d.className = 'xterm';
    el.appendChild(d);
  }
  write(text: string) {
    this.written.push(text);
  }
  onData() { return { dispose() {} }; }
  onSelectionChange() { return { dispose() {} }; }
  onBell() { return { dispose() {} }; }
  attachCustomKeyEventHandler() {}
  refresh() {}
  focus() {}
  getSelection() { return ''; }
  hasSelection() { return false; }
  dispose() {
    this.disposed = true;
  }
}

vi.mock('@xterm/xterm', () => ({ Terminal: FakeTerm }));
vi.mock('@xterm/addon-fit', () => ({ FitAddon: class { fit() {} } }));
vi.mock('@xterm/addon-web-links', () => ({ WebLinksAddon: class {} }));
vi.mock('@xterm/addon-search', () => ({ SearchAddon: class {} }));
vi.mock('@xterm/xterm/css/xterm.css', () => ({}));

const { TerminalView } = await import('./TerminalView');

const info = (id: string): SessionInfo => ({ id, hostId: null, title: id, protocol: 'local', target: '', state: 'connected', canSftp: false });

beforeEach(() => {
  created.length = 0;
  globalThis.ResizeObserver ??= class { observe() {} disconnect() {} unobserve() {} } as unknown as typeof ResizeObserver;
  globalThis.requestAnimationFrame ??= ((cb: FrameRequestCallback) => setTimeout(() => cb(0), 0)) as typeof requestAnimationFrame;
  const b: Backend = {
    invoke: async () => undefined as never,
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  useWorkspace.setState({ tabs: [], sessions: {}, writers: {}, activeTabId: null, activeByGroup: [null, null] });
});

describe('TerminalView 터미널 유지', () => {
  it('떼었다 다시 붙이면 같은 터미널을 쓰고, 떼어 둔 동안의 출력도 그 터미널로 간다', () => {
    useWorkspace.getState().upsertSession(info('s1'));
    const first = render(<TerminalView paneId="p1" sessionId="s1" focused />);
    expect(created).toHaveLength(1);
    first.unmount(); // 다른 탭으로 전환
    expect(created[0].disposed).toBe(false);
    useWorkspace.getState().writers.s1?.('hello');
    expect(created[0].written).toEqual(['hello']);

    // 돌아오면(다른 칸 id 로 다시 그려져도) 새로 만들지 않는다.
    const again = render(<TerminalView paneId="p2" sessionId="s1" focused />);
    expect(created).toHaveLength(1);
    expect(again.container.querySelector('.term-host .xterm')).not.toBeNull();
  });

  it('세션이 작업 공간에서 빠지면 터미널을 버린다', () => {
    useWorkspace.getState().upsertSession(info('s2'));
    const v = render(<TerminalView paneId="p1" sessionId="s2" focused />);
    v.unmount();
    useWorkspace.getState().removeSession('s2');
    expect(created[0].disposed).toBe(true);
    expect(useWorkspace.getState().writers.s2).toBeUndefined();
  });
});
