// 사이드바의 "즐겨찾기"·"최근 사용" 구역 접기 (D131): 제목을 누르면 접히고, 접힘은 설정에 저장돼 다시 열어도 유지된다.
// 근거: spec/ui/sidebar.md §3, spec/data/store.md §4.
import { beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Host, Settings, Store } from '../ipc/types';
import { Sidebar } from './Sidebar';

/** 서버 8개(6개를 넘어야 "최근 사용"이 보인다). 앞의 3개는 최근 사용 기록이 있고 그중 하나는 즐겨찾기. */
const HOSTS = Array.from({ length: 8 }, (_, i) => ({
  id: `h${i}`,
  groupId: null,
  alias: `srv-${i}`,
  protocol: 'ssh',
  host: `10.0.0.${i + 1}`,
  port: 22,
  username: 'ops',
  authType: 'password',
  tags: [],
  order: i,
  favorite: i === 0,
  lastUsedAt: i < 3 ? 1000 + i : undefined,
})) as unknown as Host[];

/** 가짜 백엔드: store.patch 를 받아 두고 store.get 으로 돌려준다. 보낸 설정 조각을 돌려주는 함수를 준다. */
function setup(settings: Partial<Settings> = {}, hosts: Host[] = HOSTS) {
  let store = { version: 1, groups: [], hosts, snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, ...settings } } as Store;
  const patches: Partial<Settings>[] = [];
  const b: Backend = {
    async invoke(channel: string, args: unknown[] = []): Promise<any> {
      if (channel === 'store.get') return structuredClone(store);
      if (channel === 'store.patch') {
        const patch = args[0] as { settings?: Settings };
        if (patch.settings) patches.push(patch.settings);
        store = { ...store, ...(patch as object) } as Store;
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
  return { patches, saved: () => store.settings };
}

const head = (name: string) => screen.getByRole('button', { name });
/** 이름이 name 인 서버 줄의 수(구역마다 한 줄씩 나온다). */
const rows = (name: string) => screen.queryAllByText(name).length;

beforeEach(() => {
  useUi.setState({ dialogs: [], menu: null, selectedHostId: null });
});

describe('사이드바 구역 접기 (D131)', () => {
  it('기본은 둘 다 펴져 있다: 즐겨찾기·최근 사용의 서버가 구역에도 보인다', () => {
    expect(DEFAULT_SETTINGS.favoritesCollapsed).toBe(false);
    expect(DEFAULT_SETTINGS.recentCollapsed).toBe(false);
    setup();
    render(<Sidebar />);
    expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('true');
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('true');
    expect(rows('srv-0')).toBe(3); // 즐겨찾기 + 최근 사용 + 서버 목록
    expect(rows('srv-1')).toBe(2); // 최근 사용 + 서버 목록
    expect(rows('srv-5')).toBe(1); // 서버 목록만
  });

  it('"최근 사용" 제목을 누르면 그 구역의 서버가 사라지고 recentCollapsed: true 로 저장된다', async () => {
    const { patches, saved } = setup();
    render(<Sidebar />);
    fireEvent.click(head('최근 사용'));
    await waitFor(() => expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false'));
    expect(rows('srv-1')).toBe(1); // 서버 목록에만 남는다
    expect(rows('srv-0')).toBe(2); // 즐겨찾기 구역은 그대로
    expect(patches.map((p) => p.recentCollapsed)).toEqual([true]);
    expect(saved().recentCollapsed).toBe(true);
    expect(saved().favoritesCollapsed).toBe(false);
    // 서버 목록 구역과 제목은 그대로다.
    expect(screen.getByText(/^서버 · 8$/)).toBeTruthy();
  });

  it('다시 누르면 펴지고 false 로 저장된다', async () => {
    const { patches } = setup({ recentCollapsed: true });
    render(<Sidebar />);
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
    expect(rows('srv-1')).toBe(1);
    fireEvent.click(head('최근 사용'));
    await waitFor(() => expect(rows('srv-1')).toBe(2));
    expect(patches.map((p) => p.recentCollapsed)).toEqual([false]);
  });

  it('"즐겨찾기"도 같은 방식으로 접히고 favoritesCollapsed 로 따로 저장된다', async () => {
    const { patches, saved } = setup();
    render(<Sidebar />);
    fireEvent.click(head('즐겨찾기'));
    await waitFor(() => expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('false'));
    expect(rows('srv-0')).toBe(2); // 최근 사용 + 서버 목록
    expect(patches.map((p) => p.favoritesCollapsed)).toEqual([true]);
    expect(saved().recentCollapsed).toBe(false);
  });

  it('저장된 접힘 상태로 다시 그려도 접혀 있다(앱을 다시 열었을 때)', () => {
    setup({ favoritesCollapsed: true, recentCollapsed: true });
    render(<Sidebar />);
    expect(head('즐겨찾기').getAttribute('aria-expanded')).toBe('false');
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
    expect(rows('srv-0')).toBe(1);
    expect(rows('srv-2')).toBe(1);
  });

  it('접혀 있어도 검색하면 구역 자체가 숨고, 검색을 지우면 접힌 채로 돌아온다', () => {
    setup({ recentCollapsed: true });
    render(<Sidebar />);
    const search = screen.getByPlaceholderText('서버 찾기');
    fireEvent.change(search, { target: { value: 'srv-1' } });
    expect(screen.queryByRole('button', { name: '최근 사용' })).toBeNull();
    expect(screen.queryByRole('button', { name: '즐겨찾기' })).toBeNull();
    expect(rows('srv-1')).toBe(1);
    fireEvent.change(search, { target: { value: '' } });
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('false');
  });

  it('구역이 나올 조건이 아니면 제목도 없다(서버 6개 이하, 즐겨찾기 없음)', () => {
    setup({}, HOSTS.slice(1, 6));
    render(<Sidebar />);
    expect(screen.queryByRole('button', { name: '최근 사용' })).toBeNull();
    expect(screen.queryByRole('button', { name: '즐겨찾기' })).toBeNull();
  });

  it('"서버" 제목 위의 구분선: 위에 즐겨찾기·최근 사용 구역이 있을 때만 있다 (D132)', () => {
    const serversTitle = () => screen.getByText(/^서버 · \d+$/);
    setup();
    const first = render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(true);
    // 검색 중에는 위의 구역이 숨으므로 선도 없다.
    fireEvent.change(screen.getByPlaceholderText('서버 찾기'), { target: { value: 'srv' } });
    expect(serversTitle().classList.contains('divided')).toBe(false);
    first.unmount();
    // 위의 구역을 모두 접어도 제목은 남으므로 선은 있다.
    setup({ favoritesCollapsed: true, recentCollapsed: true });
    const second = render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(true);
    second.unmount();
    // 위에 구역이 없으면(서버 6개 이하, 즐겨찾기 없음) 선이 없다.
    setup({}, HOSTS.slice(1, 6));
    render(<Sidebar />);
    expect(serversTitle().classList.contains('divided')).toBe(false);
  });

  it('옛 저장 파일처럼 값이 없으면 펴진 것으로 본다', () => {
    setup({ favoritesCollapsed: undefined as unknown as boolean, recentCollapsed: undefined as unknown as boolean });
    render(<Sidebar />);
    expect(head('최근 사용').getAttribute('aria-expanded')).toBe('true');
    expect(rows('srv-1')).toBe(2);
  });
});
