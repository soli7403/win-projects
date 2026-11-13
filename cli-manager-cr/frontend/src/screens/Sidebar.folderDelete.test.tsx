// 폴더 삭제(D58): 폴더 삭제 확인 → 안에 서버가 있으면 서버까지 지울지 한 번 더 묻는다.
// [서버까지 삭제] = 폴더·하위 폴더의 서버 삭제, [아니오] = 지금처럼 서버를 상위 폴더로 옮김,
// [취소]·X·Esc = 아무것도 삭제하지 않음.
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store, Host, Group } from '../ipc/types';
import { Sidebar } from './Sidebar';
import { DialogHost } from '../ui/DialogHost';

function host(id: string, groupId: string | null): Host {
  return {
    id, groupId, alias: `srv-${id}`, protocol: 'ssh', host: `10.0.0.${id}`, port: 22, username: 'root',
    authType: 'password', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: Number(id),
  } as Host;
}

const GROUPS: Group[] = [
  { id: 'p', parentId: null, name: '회사', sort: 0 },
  { id: 'c', parentId: 'p', name: '하위', sort: 1 },
];

function setup(hosts: Host[]): () => Store {
  let store: Store = {
    version: 1, groups: GROUPS, hosts, snippets: [], knownHosts: [],
    settings: { ...DEFAULT_SETTINGS },
  } as Store;
  const b: Backend = {
    async invoke(channel: string, args: unknown[] = []): Promise<any> {
      if (channel === 'store.get') return structuredClone(store);
      if (channel === 'store.patch') {
        store = { ...store, ...(args[0] as object) } as Store;
        return undefined;
      }
      return [];
    },
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  useApp.setState({ booted: true, store: structuredClone(store), toasts: [] });
  useUi.setState({ dialogs: [], menu: null });
  return () => store;
}

/** "회사" 폴더 우클릭 메뉴의 [폴더 삭제]를 누른다. */
function clickDeleteFolder(): void {
  fireEvent.contextMenu(screen.getByText('회사'));
  const item = useUi.getState().menu?.items.find((x) => x.label === '폴더 삭제');
  expect(item).toBeTruthy();
  item!.onSelect!();
}

async function answer(v: unknown): Promise<void> {
  await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
  useUi.getState().settleDialog(v);
}

describe('폴더 삭제 — 안의 서버 처리 (D58)', () => {
  beforeEach(() => {
    useUi.setState({ dialogs: [], menu: null });
  });

  it('[서버까지 삭제]면 폴더·하위 폴더 안의 서버를 지우고 바깥 서버는 남긴다', async () => {
    const store = setup([host('1', 'p'), host('2', 'c'), host('3', null)]);
    render(<Sidebar />);
    clickDeleteFolder();
    await answer(true);
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
    const q = useUi.getState().dialogs[0];
    expect(q.kind).toBe('ask');
    if (q.kind === 'ask') {
      expect(q.opts.body).toBe('"회사" 폴더 안의 서버 2개(하위 폴더 포함)도 모두 삭제할까요?');
      expect(q.opts.okText).toBe('서버까지 삭제');
      expect(q.opts.noText).toBe('아니오 (폴더만 삭제)');
    }
    useUi.getState().settleDialog('yes');
    await waitFor(() => expect(store().groups).toEqual([]));
    expect(store().hosts.map((h) => h.id)).toEqual(['3']);
  });

  it('[아니오]면 지금처럼 폴더만 지우고 서버는 상위(최상위)로 옮긴다', async () => {
    const store = setup([host('1', 'p'), host('2', 'c')]);
    render(<Sidebar />);
    clickDeleteFolder();
    await answer(true);
    await answer('no');
    await waitFor(() => expect(store().groups).toEqual([]));
    expect(store().hosts.map((h) => [h.id, h.groupId])).toEqual([['1', null], ['2', null]]);
  });

  it('두 번째 질문 창의 X 를 누르면 폴더도 서버도 지우지 않는다', async () => {
    const store = setup([host('1', 'p'), host('2', 'c')]);
    render(<><Sidebar /><DialogHost /></>);
    clickDeleteFolder();
    await answer(true);
    await screen.findByText('"회사" 폴더 안의 서버 2개(하위 폴더 포함)도 모두 삭제할까요?');
    fireEvent.click(screen.getByRole('button', { name: '닫기' }));
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(0));
    await new Promise((r) => setTimeout(r, 20));
    expect(store().groups).toHaveLength(2);
    expect(store().hosts.map((h) => [h.id, h.groupId])).toEqual([['1', 'p'], ['2', 'c']]);
  });

  it('두 번째 질문 창에서 Esc 도 취소다', async () => {
    const store = setup([host('1', 'p')]);
    render(<><Sidebar /><DialogHost /></>);
    clickDeleteFolder();
    await answer(true);
    await screen.findByText('"회사" 폴더 안의 서버 1개(하위 폴더 포함)도 모두 삭제할까요?');
    fireEvent.keyDown(window, { key: 'Escape' });
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(0));
    await new Promise((r) => setTimeout(r, 20));
    expect(store().groups).toHaveLength(2);
    expect(store().hosts[0].groupId).toBe('p');
  });

  it('첫 확인에서 취소하면 아무것도 바뀌지 않는다', async () => {
    const store = setup([host('1', 'p')]);
    render(<Sidebar />);
    clickDeleteFolder();
    await answer(false);
    await new Promise((r) => setTimeout(r, 20));
    expect(useUi.getState().dialogs.length).toBe(0);
    expect(store().groups).toHaveLength(2);
    expect(store().hosts[0].groupId).toBe('p');
  });

  it('안에 서버가 없으면 두 번째 질문 없이 폴더만 지운다', async () => {
    const store = setup([host('3', null)]);
    render(<Sidebar />);
    clickDeleteFolder();
    await answer(true);
    await waitFor(() => expect(store().groups).toEqual([]));
    expect(useUi.getState().dialogs.length).toBe(0);
    expect(store().hosts.map((h) => h.id)).toEqual(['3']);
  });
});
