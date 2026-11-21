// 탭의 색 막대는 그룹에서 선택된 탭에만 있다 (D103, spec/ui/shell.md §2.1, 수용 S4a).
import { beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { useWorkspace, type Tab } from '../state/workspace';
import type { SessionInfo } from '../ipc/types';
import { TabBar, tabMark } from './TabBar';

function tab(id: string, sessionId: string, group?: 1): Tab {
  return { id, broadcast: false, group, root: { type: 'pane', id: `p-${id}`, sessionId }, focusedPane: `p-${id}`, zoomed: null };
}

function session(id: string, title: string, color?: string) {
  return { info: { id, title, state: 'connected', color } as unknown as SessionInfo, activity: false };
}

/** 색 막대가 있는 탭의 제목들. */
function barred(): string[] {
  return screen
    .getAllByRole('tab')
    .filter((el) => el.querySelector('.bar'))
    .map((el) => el.getAttribute('title') ?? '');
}

function press(title: string) {
  fireEvent.mouseDown(screen.getByTitle(title), { button: 0 });
}

beforeEach(() => {
  useWorkspace.setState({
    tabs: [tab('a', 's-a'), tab('b', 's-b'), tab('c', 's-c')],
    activeTabId: 'a',
    activeByGroup: ['a', null],
    sessions: { 's-a': session('s-a', 'red', '#ef4444'), 's-b': session('s-b', 'plain'), 's-c': session('s-c', 'green', '#22c55e') },
  });
});

describe('tabMark', () => {
  it('선택된 탭: 세션 색의 막대와 14% 배경', () => {
    expect(tabMark(true, '#ef4444')).toEqual({ bar: '#ef4444', tint: 'color-mix(in srgb, #ef4444 14%, transparent)' });
  });
  it('선택된 탭에 세션 색이 없으면 강조색을 쓴다', () => {
    expect(tabMark(true, undefined)).toEqual({ bar: 'var(--accent)', tint: 'color-mix(in srgb, var(--accent) 14%, transparent)' });
    expect(tabMark(true, '').bar).toBe('var(--accent)');
  });
  it('선택되지 않은 탭: 막대 없이 세션 색의 7% 배경만', () => {
    expect(tabMark(false, '#ef4444')).toEqual({ bar: undefined, tint: 'color-mix(in srgb, #ef4444 7%, transparent)' });
  });
  it('선택되지 않았고 세션 색도 없으면 아무 표시도 없다', () => {
    expect(tabMark(false, undefined)).toEqual({});
  });
});

describe('탭 색 막대 (D103)', () => {
  it('선택된 탭에만 색 막대가 있다 (색이 있는 다른 탭에는 없다)', () => {
    render(<TabBar />);
    expect(barred()).toEqual(['red']);
  });

  it('세션 색이 없는 탭을 선택하면 막대가 그 탭으로 옮겨 간다', () => {
    render(<TabBar />);
    press('plain');
    expect(barred()).toEqual(['plain']);
  });

  it('다른 탭을 선택할 때마다 막대가 따라간다', () => {
    render(<TabBar />);
    press('green');
    expect(barred()).toEqual(['green']);
    press('red');
    expect(barred()).toEqual(['red']);
  });

  it('화면을 나누면 그룹마다 선택된 탭 하나씩에 막대가 있다', () => {
    useWorkspace.setState({
      tabs: [tab('a', 's-a'), tab('b', 's-b'), tab('c', 's-c', 1)],
      activeTabId: 'b',
      activeByGroup: ['b', 'c'],
    });
    render(
      <>
        <TabBar group={0} />
        <TabBar group={1} />
      </>,
    );
    expect(barred()).toEqual(['plain', 'green']);
    press('red');
    expect(barred()).toEqual(['red', 'green']);
  });
});
