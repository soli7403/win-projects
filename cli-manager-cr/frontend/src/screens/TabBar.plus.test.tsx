// 탭 막대의 + 로 연 세션은 그 + 가 속한 그룹에 생긴다 (D119, spec/ui/shell.md §2.3, 수용 G5a·G5b).
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { groupOf, useWorkspace, type Tab } from '../state/workspace';
import { useUi } from '../state/ui';
import { setBackend, type Backend } from '../ipc/backend';
import type { SessionInfo } from '../ipc/types';
import { TabBar } from './TabBar';
import { QuickPalette } from './QuickPalette';

function tab(id: string, group?: 1): Tab {
  return { id, broadcast: false, group, root: { type: 'pane', id: `p-${id}`, sessionId: `s-${id}` }, focusedPane: `p-${id}`, zoomed: null };
}

function session(id: string) {
  return { info: { id, title: id, state: 'connected' } as unknown as SessionInfo, activity: false };
}

const ws = () => useWorkspace.getState();
let opened = 0;

beforeEach(() => {
  opened = 0;
  const invoke = vi.fn(async (channel: string) => {
    if (channel !== 'session.open') return undefined;
    opened += 1;
    return { id: `s-new${opened}`, title: `new${opened}`, state: 'connected' };
  });
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  // 왼쪽 그룹에 a, 오른쪽 그룹에 b. 포커스는 오른쪽(b).
  useWorkspace.setState({
    tabs: [tab('a'), tab('b', 1)],
    activeTabId: 'b',
    activeByGroup: ['a', 'b'],
    sessions: { 's-a': session('s-a'), 's-b': session('s-b') },
    writers: {},
  });
  useUi.setState({ quick: { open: false, targetPane: null, group: null } });
});

afterEach(() => {
  setBackend(null);
});

function renderSplit() {
  render(
    <>
      <div data-testid="left"><TabBar group={0} /></div>
      <div data-testid="right"><TabBar group={1} /></div>
      <QuickPalette />
    </>,
  );
}

function clickPlus(side: 'left' | 'right') {
  fireEvent.click(within(screen.getByTestId(side)).getByLabelText('새로 연결'));
}

/** 빠른 연결 창에서 맨 위 항목(서버가 없으므로 "새 로컬 터미널")을 실행한다. */
function runFirst(init: { ctrlKey?: boolean; shiftKey?: boolean } = {}) {
  fireEvent.keyDown(screen.getByLabelText('빠른 연결 입력'), { key: 'Enter', ...init });
}

describe('탭 막대의 + (D119)', () => {
  it('G5a: 포커스가 오른쪽일 때 왼쪽 + 로 열면 왼쪽 그룹에 생기고 포커스가 왼쪽으로 온다', async () => {
    renderSplit();
    clickPlus('left');
    expect(useUi.getState().quick).toMatchObject({ open: true, group: 0 });
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    const nt = ws().tabs[2];
    expect(groupOf(nt)).toBe(0);
    expect(ws().activeByGroup).toEqual([nt.id, 'b']);
    expect(ws().activeTabId).toBe(nt.id);
  });

  it('포커스가 왼쪽일 때 오른쪽 + 로 열면 오른쪽 그룹에 생긴다', async () => {
    ws().activateTab('a');
    renderSplit();
    clickPlus('right');
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    const nt = ws().tabs[2];
    expect(groupOf(nt)).toBe(1);
    expect(ws().activeByGroup).toEqual(['a', nt.id]);
  });

  it('G5b: 왼쪽 + 로 연 창에서 Shift+Enter(아래로 나누기)는 왼쪽 그룹의 선택 탭을 나눈다', async () => {
    renderSplit();
    clickPlus('left');
    runFirst({ shiftKey: true });
    await waitFor(() => expect(Object.keys(ws().sessions)).toHaveLength(3));
    expect(ws().tabs).toHaveLength(2);
    const a = ws().tabs.find((t) => t.id === 'a')!;
    const b = ws().tabs.find((t) => t.id === 'b')!;
    expect(a.root.type).toBe('split');
    expect(b.root.type).toBe('pane');
  });

  it('단축키로 연 빠른 연결(그룹 없음)은 지금처럼 포커스 그룹에 연다', async () => {
    renderSplit();
    useUi.getState().openQuick(null);
    await screen.findByLabelText('빠른 연결 입력');
    runFirst();
    await waitFor(() => expect(ws().tabs).toHaveLength(3));
    expect(groupOf(ws().tabs[2])).toBe(1);
  });
});

describe('openInNewTab 의 그룹 지정 (D119)', () => {
  const info = (id: string) => ({ id, title: id, state: 'connected' }) as unknown as SessionInfo;

  it('그룹을 주면 포커스와 관계없이 그 그룹에 붙고 선택된다', () => {
    ws().openInNewTab(info('x'), 0);
    const x = ws().tabs[2];
    expect(groupOf(x)).toBe(0);
    expect(ws().activeTabId).toBe(x.id);
  });

  it('오른쪽을 줬는데 나뉨이 이미 풀렸으면 왼쪽에 붙는다(새 탭 하나로 다시 나뉘지 않는다)', () => {
    ws().mergeGroups();
    ws().openInNewTab(info('x'), 1);
    expect(ws().tabs.every((t) => groupOf(t) === 0)).toBe(true);
  });
});
