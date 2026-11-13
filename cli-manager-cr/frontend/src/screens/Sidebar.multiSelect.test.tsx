// 서버 여러 개 선택(D59): Ctrl/Shift+클릭으로 고르고, 메뉴는 삭제 하나뿐, 끌면 모두 폴더로 옮긴다.
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store, Host, Group } from '../ipc/types';
import { Sidebar } from './Sidebar';

function host(id: string, groupId: string | null): Host {
  return {
    id, groupId, alias: `srv-${id}`, protocol: 'ssh', host: `10.0.0.${id}`, port: 22, username: 'root',
    authType: 'password', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: Number(id),
  } as Host;
}

const GROUPS: Group[] = [{ id: 'g', parentId: null, name: '대상', sort: 0 }];

function setup(hosts: Host[]): () => Store {
  let store: Store = { version: 1, groups: GROUPS, hosts, snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store;
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
  return () => store;
}

const row = (name: string) => screen.getByText(name).closest('[role="treeitem"]') as HTMLElement;

function fakeTransfer() {
  const data: Record<string, string> = {};
  return {
    data,
    effectAllowed: '',
    dropEffect: '',
    types: [] as string[],
    setData(k: string, v: string) { data[k] = v; },
    getData(k: string) { return data[k] ?? ''; },
    setDragImage() {},
  };
}

beforeEach(() => {
  useUi.setState({ dialogs: [], menu: null, selectedHostId: null });
});

describe('서버 여러 개 선택 (D59)', () => {
  it('Ctrl+클릭 2개 → 우클릭 메뉴는 "삭제 (2개)" 하나뿐, 확인하면 둘 다 삭제', async () => {
    const store = setup([host('1', null), host('2', null), host('3', null)]);
    render(<Sidebar />);
    fireEvent.click(row('srv-1'));
    fireEvent.click(row('srv-3'), { ctrlKey: true });
    expect(screen.getByText('서버 2개 선택')).toBeTruthy();
    fireEvent.contextMenu(row('srv-3'));
    const items = useUi.getState().menu?.items ?? [];
    expect(items.map((x) => x.label)).toEqual(['삭제 (2개)']);
    items[0].onSelect!();
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
    useUi.getState().settleDialog(true);
    await waitFor(() => expect(store().hosts.map((h) => h.id)).toEqual(['2']));
  });

  it('선택 밖의 서버를 우클릭하면 원래 메뉴', () => {
    setup([host('1', null), host('2', null), host('3', null)]);
    render(<Sidebar />);
    fireEvent.click(row('srv-1'));
    fireEvent.click(row('srv-2'), { ctrlKey: true });
    fireEvent.contextMenu(row('srv-3'));
    expect(useUi.getState().menu?.items.some((x) => x.label === '편집')).toBe(true);
  });

  it('Shift+클릭 범위를 끌어 폴더에 놓으면 모두 그 폴더로 옮긴다', async () => {
    const store = setup([host('1', null), host('2', null), host('3', null), host('4', null)]);
    render(<Sidebar />);
    fireEvent.click(row('srv-1'));
    fireEvent.click(row('srv-3'), { shiftKey: true });
    expect(screen.getByText('서버 3개 선택')).toBeTruthy();
    const dt = fakeTransfer();
    fireEvent.dragStart(row('srv-2'), { dataTransfer: dt });
    fireEvent.dragOver(row('대상'), { dataTransfer: dt });
    fireEvent.drop(row('대상'), { dataTransfer: dt });
    await waitFor(() => expect(store().hosts.filter((h) => h.groupId === 'g').map((h) => h.id)).toEqual(['1', '2', '3']));
    expect(store().hosts.find((h) => h.id === '4')?.groupId).toBeNull();
    // 여러 개를 끌 때는 칸에 놓아 접속하는 데이터(HOST_MIME)를 싣지 않는다.
    expect(Object.keys(dt.data)).toEqual(['text/plain']);
  });

  it('여러 개 선택 상태에서 Delete 는 일괄 삭제 확인, Esc 는 선택 해제', async () => {
    setup([host('1', null), host('2', null)]);
    render(<Sidebar />);
    fireEvent.click(row('srv-1'));
    fireEvent.click(row('srv-2'), { ctrlKey: true });
    fireEvent.keyDown(row('srv-2'), { key: 'Delete' });
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
    const d = useUi.getState().dialogs[0];
    if (d.kind === 'confirm') expect(d.opts.body).toBe('선택한 서버 2개를 삭제할까요?');
    useUi.getState().settleDialog(false);
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(0));
    fireEvent.keyDown(row('srv-2'), { key: 'Escape' });
    expect(screen.queryByText('서버 2개 선택')).toBeNull();
  });
});
