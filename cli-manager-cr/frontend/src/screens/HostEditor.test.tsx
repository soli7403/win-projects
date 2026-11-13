// 수용 시나리오 H4(볼트 잠김 저장 차단), H8(공유 계정 전파), H11([저장 후 연결]).
// 근거: spec/acceptance/04-hosts-sftp-settings-sync.md, spec/ui/host-editor.md §1.2·§1.5.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { Backend, EventName } from '../ipc/backend';
import { setBackend } from '../ipc/backend';
import type { CredMeta, Host, Store } from '../ipc/types';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import { HostEditor } from './HostEditor';

function host(id: string, alias: string, extra: Partial<Host> = {}): Host {
  return {
    id, alias, groupId: null, protocol: 'ssh', host: `${alias}.example`, port: 22, username: 'deploy',
    authType: 'password', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30,
    forwards: [], sort: 1, ...extra,
  };
}

/** 테스트용 백엔드: 메모리 저장소와 호출 기록. 화면이 쓰는 채널만 구현한다. */
function testBackend(initial: Store, creds: CredMeta[]) {
  let store: Store = structuredClone(initial);
  const calls: { channel: string; args: unknown[] }[] = [];
  const saved: CredMeta[] = [...creds];
  const invoke = vi.fn(async (channel: string, args: unknown[] = []) => {
    calls.push({ channel, args });
    switch (channel) {
      case 'store.get': return structuredClone(store);
      case 'store.patch': store = { ...store, ...(args[0] as object) } as Store; return undefined;
      case 'cred.list': return saved.map((c) => ({ ...c }));
      case 'cred.save': {
        const input = args[0] as { id?: string; name: string; kind: 'password' | 'key'; username?: string };
        const meta: CredMeta = { id: input.id ?? `c-new-${saved.length + 1}`, name: input.name, kind: input.kind, username: input.username, updatedAt: Date.now() };
        const i = saved.findIndex((c) => c.id === meta.id);
        if (i >= 0) saved[i] = meta; else saved.push(meta);
        return meta;
      }
      case 'local.shells': return [{ path: 'pwsh.exe', label: 'PowerShell 7' }];
      case 'local.tmuxAvailable': return false;
      case 'serial.list': return [];
      case 'session.open': return { id: 's-1', hostId: (args[0] as { hostId?: string }).hostId ?? null, title: 'web', protocol: 'ssh', target: '', state: 'connecting', canSftp: false };
      default: return undefined;
    }
  });
  const backend: Backend = {
    invoke: invoke as unknown as Backend['invoke'],
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: (_e: EventName, _h: (...a: unknown[]) => void) => () => undefined,
  };
  return { backend, calls, getStore: () => store };
}

function seedApp(store: Store, unlocked: boolean) {
  useApp.setState({
    booted: true,
    store,
    vault: { initialized: true, unlocked, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
  });
}

const base = (hosts: Host[]): Store => ({ version: 1, groups: [], hosts, snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS }, layout: undefined });

describe('서버 편집 화면 수용 시나리오', () => {
  beforeEach(() => {
    useUi.setState({ editor: null, dialogs: [], menu: null });
    useWorkspace.setState({ tabs: [], activeTabId: null, sessions: {}, writers: {} });
  });

  it('H4: 보관함이 잠겨 있으면 비밀번호 저장을 막고 대화상자를 닫지 않는다', async () => {
    const t = testBackend(base([]), []);
    setBackend(t.backend);
    seedApp(base([]), false);
    await act(async () => { await useApp.getState().refreshStore(); });
    useUi.getState().openEditor({ kind: 'host', hostId: null, groupId: null });
    render(<HostEditor hostId={null} groupId={null} />);

    fireEvent.change(screen.getByPlaceholderText(/10.0.0.11/), { target: { value: '10.0.0.1' } });
    fireEvent.click(screen.getByRole('tab', { name: '로그인' }));
    fireEvent.change(screen.getByPlaceholderText('비밀번호'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: '저장' }));

    await waitFor(() => expect(screen.getByText('보관함이 잠겨 있어 비밀번호를 저장할 수 없습니다')).toBeTruthy());
    expect(useUi.getState().editor).not.toBeNull();
    expect(t.calls.some((c) => c.channel === 'store.patch')).toBe(false);
    expect(t.calls.some((c) => c.channel === 'cred.save')).toBe(false);
  });

  it('H8: 같은 계정을 쓰는 서버는 공유 계정으로 열리고, 저장해도 계정이 새로 생기지 않는다', async () => {
    const creds: CredMeta[] = [{ id: 'c-1', name: '배포 계정', kind: 'password', username: 'deploy', updatedAt: 1 }];
    const a = host('h-a', 'web1', { credentialId: 'c-1' });
    const b = host('h-b', 'web2', { credentialId: 'c-1' });
    const t = testBackend(base([a, b]), creds);
    setBackend(t.backend);
    seedApp(base([a, b]), true);
    await act(async () => { await useApp.getState().refreshStore(); });
    useUi.getState().openEditor({ kind: 'host', hostId: 'h-a', groupId: null });
    render(<HostEditor hostId="h-a" groupId={null} />);

    await waitFor(() => expect(screen.getByRole('tab', { name: '로그인' })).toBeTruthy());
    fireEvent.click(screen.getByRole('tab', { name: '로그인' }));
    await waitFor(() => {
      const sel = screen.getByRole('combobox') as HTMLSelectElement;
      expect(sel.value).toBe('c-1');
    });
    fireEvent.click(screen.getByRole('button', { name: '저장' }));

    await waitFor(() => expect(t.calls.some((c) => c.channel === 'store.patch')).toBe(true));
    expect(t.calls.some((c) => c.channel === 'cred.save')).toBe(false);
    const saved = t.getStore().hosts.find((h) => h.id === 'h-a');
    expect(saved?.credentialId).toBe('c-1');
  });

  it('H11: [저장 후 연결]은 저장 뒤 대화상자를 닫고 새 탭으로 연결한다', async () => {
    const t = testBackend(base([]), []);
    setBackend(t.backend);
    seedApp(base([]), true);
    await act(async () => { await useApp.getState().refreshStore(); });
    useUi.getState().openEditor({ kind: 'host', hostId: null, groupId: null });
    render(<HostEditor hostId={null} groupId={null} />);

    fireEvent.change(screen.getByPlaceholderText(/10.0.0.11/), { target: { value: '10.0.0.5' } });
    fireEvent.click(screen.getByRole('button', { name: '저장하고 연결' }));

    await waitFor(() => expect(useUi.getState().editor).toBeNull());
    expect(t.getStore().hosts.some((h) => h.host === '10.0.0.5')).toBe(true);
    await waitFor(() => expect(t.calls.some((c) => c.channel === 'session.open')).toBe(true));
    await waitFor(() => expect(useWorkspace.getState().tabs.length).toBe(1));
  });
});
