// 탭 그룹(화면 나누기) 상태 규칙 시험. 근거: spec/ui/shell.md §2.3, acceptance G1~G9.
import { beforeEach, describe, expect, it } from 'vitest';
import type { SessionInfo } from '../ipc/types';
import { groupOf, isSplit, useWorkspace } from './workspace';

const info = (id: string) => ({ id, title: id, state: 'connected' }) as unknown as SessionInfo;
const ws = () => useWorkspace.getState();

beforeEach(() => {
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], groupRatio: 50, sessions: {}, writers: {} });
});

function open3(): string[] {
  ['a', 'b', 'c'].forEach((n) => ws().openInNewTab(info(n)));
  return ws().tabs.map((t) => t.id);
}

describe('탭 그룹', () => {
  it('G2: 탭이 1개뿐이면 나눌 수 없다', () => {
    ws().openInNewTab(info('a'));
    ws().moveTabToGroup(ws().tabs[0].id, 1);
    expect(isSplit(ws().tabs)).toBe(false);
  });

  it('G1: 오른쪽 그룹으로 옮기면 나뉘고 각 그룹이 선택 탭을 가진다', () => {
    const [a, b] = open3();
    ws().moveTabToGroup(b, 1);
    expect(isSplit(ws().tabs)).toBe(true);
    expect(ws().activeByGroup[1]).toBe(b);
    expect(ws().activeByGroup[0]).not.toBe(b);
    expect(groupOf(ws().tabs.find((t) => t.id === a)!)).toBe(0);
    expect(ws().activeTabId).toBe(b);
  });

  it('G5: 새 탭은 포커스 그룹에 생긴다', () => {
    const [, b] = open3();
    ws().moveTabToGroup(b, 1);
    ws().activateTab(b);
    ws().openInNewTab(info('d'));
    const d = ws().tabs[ws().tabs.length - 1];
    expect(groupOf(d)).toBe(1);
    expect(ws().activeByGroup[1]).toBe(d.id);
  });

  it('G3: 다른 그룹 막대로 끌어 옮기면 대상 그룹의 선택 탭이 된다', () => {
    const [a, b] = open3();
    ws().moveTabToGroup(b, 1);
    ws().moveTab(b, 0, 0);
    expect(groupOf(ws().tabs.find((t) => t.id === b)!)).toBe(0);
    expect(ws().activeByGroup[0]).toBe(b);
    expect(a).toBeTruthy();
  });

  it('G4: 한 그룹이 비면 나뉨이 풀린다', async () => {
    const [, b] = open3();
    ws().moveTabToGroup(b, 1);
    await ws().closeTab(b);
    expect(isSplit(ws().tabs)).toBe(false);
    expect(ws().activeByGroup[1]).toBeNull();
    expect(ws().activeTabId).not.toBeNull();
  });

  it('왼쪽 그룹이 모두 비면 오른쪽 탭들이 왼쪽으로 합쳐진다', async () => {
    const [a, b, c] = open3();
    ws().moveTabToGroup(c, 1);
    await ws().closeTab(a);
    await ws().closeTab(b);
    expect(ws().tabs.map((t) => t.id)).toEqual([c]);
    expect(isSplit(ws().tabs)).toBe(false);
    expect(ws().activeByGroup[0]).toBe(c);
  });

  it('탭을 닫으면 같은 그룹의 이웃 탭이 선택된다', async () => {
    const [a, b, c] = open3();
    ws().moveTabToGroup(b, 1);
    ws().moveTabToGroup(c, 1);
    ws().activateTab(b);
    await ws().closeTab(b);
    expect(ws().activeByGroup[1]).toBe(c);
    expect(ws().activeByGroup[0]).toBe(a);
  });

  it('G9: 화면 나누기 해제는 모든 탭을 왼쪽 그룹으로 합친다', () => {
    const [, b] = open3();
    ws().moveTabToGroup(b, 1);
    ws().mergeGroups();
    expect(isSplit(ws().tabs)).toBe(false);
    expect(ws().tabs.length).toBe(3);
  });

  it('경계를 끌어 정한 비율은 20~80% 로 제한된다', () => {
    ws().setGroupRatio(5);
    expect(ws().groupRatio).toBe(20);
    ws().setGroupRatio(95);
    expect(ws().groupRatio).toBe(80);
    ws().setGroupRatio(37.26);
    expect(ws().groupRatio).toBe(37.3);
  });
});
