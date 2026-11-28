// 자동화 기능의 화면 검사: 명령 모음 변수(D138), 로그인 스크립트 탭(D139), 출력 트리거 설정(D140),
// 매크로 편집(D141), 포트 포워딩 정보 창(D137).
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import type { Backend } from '../ipc/backend';
import { setBackend } from '../ipc/backend';
import type { Host, SessionInfo, Snippet, Store } from '../ipc/types';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { resetMacrosForTest, useMacros } from '../state/macros';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import { newMacroStep } from '../lib/macro';
import { resetRememberedValues } from '../lib/snippetVars';
import { SnippetsPanel } from './SnippetsPanel';
import { HostEditor } from './HostEditor';
import { SettingsRules } from './SettingsRules';
import { ForwardsDialog } from './ForwardsDialog';

const termInput = vi.fn();
let store: Store;
let patches: Partial<Store>[];

function seed(p: Partial<Store> = {}): void {
  store = { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS }, layout: undefined, ...p };
  patches = [];
  const invoke = vi.fn(async (channel: string, args: unknown[] = []) => {
    switch (channel) {
      case 'store.get': return structuredClone(store);
      case 'store.patch':
        patches.push(args[0] as Partial<Store>);
        store = { ...store, ...(args[0] as object) } as Store;
        return undefined;
      case 'cred.list': return [];
      case 'local.shells': return [];
      case 'local.tmuxAvailable': return false;
      case 'serial.list': return [];
      default: return undefined;
    }
  });
  setBackend({ invoke: invoke as unknown as Backend['invoke'], termInput, termResize: () => undefined, reply: () => undefined, on: () => () => undefined });
  useApp.setState({ booted: true, store, vault: { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false }, toasts: [] });
}

function openSession(p: Partial<SessionInfo> = {}): SessionInfo {
  const info: SessionInfo = { id: 's1', hostId: null, title: 'web-01', protocol: 'ssh', target: 'deploy@10.0.0.5:22', state: 'connected', canSftp: true, ...p };
  useWorkspace.getState().openInNewTab(info);
  return info;
}

beforeEach(() => {
  termInput.mockClear();
  resetMacrosForTest();
  resetRememberedValues();
  useUi.setState({ editor: null, dialogs: [], menu: null });
  useWorkspace.setState({ tabs: [], activeTabId: null, activeByGroup: [null, null], sessions: {}, writers: {} });
});

describe('명령 모음 변수 (D138)', () => {
  it('변수가 있는 명령은 값을 물은 뒤 채워서 보낸다', () => {
    const snip: Snippet = { id: 'sn1', name: '재시작', body: 'systemctl restart {{서비스}} --no-block {{옵션:-q}}', sendEnter: true };
    seed({ snippets: [snip] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('재시작'));
    expect(termInput).not.toHaveBeenCalled();

    const dialog = screen.getByRole('dialog', { name: '재시작' });
    const option = within(dialog).getByLabelText('옵션') as HTMLInputElement;
    expect(option.value).toBe('-q'); // 기본값이 미리 채워진다
    fireEvent.change(within(dialog).getByLabelText('서비스'), { target: { value: 'nginx' } });
    expect(within(dialog).getByText('systemctl restart nginx --no-block -q')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: '실행' }));

    expect(termInput).toHaveBeenCalledWith('s1', 'systemctl restart nginx --no-block -q\r');
    expect(screen.queryByRole('dialog', { name: '재시작' })).toBeNull();
  });

  it('변수가 없는 명령은 묻지 않고 바로 보낸다', () => {
    seed({ snippets: [{ id: 'sn1', name: '가동 시간', body: 'uptime', sendEnter: true }] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('가동 시간'));
    expect(termInput).toHaveBeenCalledWith('s1', 'uptime\r');
  });

  it('다음에 물을 때는 지난번 값을 채워 둔다', () => {
    seed({ snippets: [{ id: 'sn1', name: '접속', body: 'ssh {{host}}', sendEnter: true }] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('접속'));
    fireEvent.change(screen.getByLabelText('host'), { target: { value: 'db-01' } });
    fireEvent.click(screen.getByRole('button', { name: '실행' }));
    fireEvent.click(screen.getByText('접속'));
    expect((screen.getByLabelText('host') as HTMLInputElement).value).toBe('db-01');
  });
});

describe('매크로 (D141)', () => {
  it('새 매크로를 단계와 함께 저장하면 목록에 단계 수 표시가 붙는다', async () => {
    seed();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByRole('button', { name: '새 매크로' }));
    const dialog = screen.getByRole('dialog', { name: '새 매크로' });
    const save = within(dialog).getByRole('button', { name: '저장' }) as HTMLButtonElement;
    fireEvent.change(within(dialog).getByPlaceholderText('예: 배포 후 로그 확인'), { target: { value: '로그 보기' } });
    fireEvent.change(within(dialog).getByLabelText('1단계 보낼 글자'), { target: { value: 'cd /var/log' } });
    fireEvent.click(within(dialog).getByRole('button', { name: '+ 단계 추가' }));
    fireEvent.change(within(dialog).getByLabelText('2단계 기다릴 글자'), { target: { value: '$ ' } });
    fireEvent.change(within(dialog).getByLabelText('2단계 보낼 글자'), { target: { value: 'tail -f app.log' } });
    expect(save.disabled).toBe(false);
    await act(async () => { fireEvent.click(save); });

    const saved = patches.at(-1)?.snippets?.[0];
    expect(saved?.name).toBe('로그 보기');
    expect(saved?.steps?.map((s) => [s.expect, s.send, s.enter])).toEqual([['', 'cd /var/log', true], ['$ ', 'tail -f app.log', true]]);
    expect(saved?.body).toBe('cd /var/log\ntail -f app.log'); // 옛 버전·검색용 본문
    await waitFor(() => expect(screen.getByText('매크로 2단계')).toBeTruthy());
  });

  it('명령에는 "명령" 배지가, 매크로에는 단계 수 배지가 서로 다른 모양으로 붙는다 (D143)', () => {
    seed({ snippets: [
      { id: 'c1', name: '가동 시간', body: 'uptime', sendEnter: true },
      { id: 'm1', name: '루트', body: 'sudo -i', sendEnter: true, steps: [newMacroStep({ send: 'sudo -i' })] },
    ] });
    render(<SnippetsPanel />);
    const cmd = screen.getByText('명령');
    const macro = screen.getByText('매크로 1단계');
    expect(cmd.className).toBe('snip-badge cmd');
    expect(macro.className).toBe('snip-badge');
    expect(screen.getAllByText('명령')).toHaveLength(1); // 매크로 항목에는 붙지 않는다
  });

  it('매크로를 누르면 지금 창에서 실행한다', () => {
    const macro: Snippet = { id: 'm1', name: '루트', body: 'sudo -i', sendEnter: true, steps: [newMacroStep({ send: 'sudo -i' }), newMacroStep({ expect: '# ', send: 'id' })] };
    seed({ snippets: [macro] });
    openSession();
    render(<SnippetsPanel />);
    fireEvent.click(screen.getByText('루트'));
    expect(termInput).toHaveBeenCalledWith('s1', 'sudo -i\r');
    expect(useMacros.getState().running.s1).toMatchObject({ macroId: 'm1', step: 1, total: 2 });
    expect(screen.getByText('실행 중 2/2')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '매크로 중지' }));
    expect(useMacros.getState().running.s1).toBeUndefined();
  });

  it('녹화가 끝나면 모은 단계로 편집 창이 열린다', () => {
    seed();
    render(<SnippetsPanel />);
    act(() => useMacros.setState({ recorded: [newMacroStep({ send: 'uptime' })] }));
    const dialog = screen.getByRole('dialog', { name: '새 매크로' });
    expect((within(dialog).getByLabelText('1단계 보낼 글자') as HTMLInputElement).value).toBe('uptime');
    expect(within(dialog).getByText(/녹화한 입력입니다/)).toBeTruthy();
    expect(useMacros.getState().recorded).toBeNull();
  });
});

describe('서버 편집 · 자동화 탭 (D139)', () => {
  const host: Host = {
    id: 'h1', alias: 'web', groupId: null, protocol: 'ssh', host: '10.0.0.5', port: 22, username: 'deploy',
    authType: 'ask', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 1,
  };

  it('로그인 스크립트 단계를 넣고 저장하면 서버에 loginScript 로 남는다(빈 단계는 빠진다)', async () => {
    seed({ hosts: [host] });
    useUi.getState().openEditor({ kind: 'host', hostId: 'h1', groupId: null });
    render(<HostEditor hostId="h1" groupId={null} />);
    fireEvent.click(screen.getByRole('tab', { name: '자동화' }));
    expect(screen.getByText(/로그인 스크립트: 연결 직후/)).toBeTruthy();
    expect(screen.getByText(/\{\{password\}\}/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '+ 단계 추가' }));
    fireEvent.click(screen.getByRole('button', { name: '+ 단계 추가' }));
    fireEvent.change(screen.getByLabelText('1단계 기다릴 글자'), { target: { value: '$ ' } });
    fireEvent.change(screen.getByLabelText('1단계 보낼 글자'), { target: { value: 'sudo su -' } });
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '저장' })); });

    await waitFor(() => expect(patches.some((p) => p.hosts)).toBe(true));
    const saved = patches.find((p) => p.hosts)!.hosts![0];
    expect(saved.loginScript?.map((s) => [s.expect, s.send])).toEqual([['$ ', 'sudo su -']]);
  });

  it('로컬 터미널의 자동화 탭에는 서버별 출력 트리거만 있고 로그인 스크립트는 없다 (D149)', () => {
    seed({ hosts: [{ ...host, protocol: 'local', host: '', port: 0, username: '' }] });
    render(<HostEditor hostId="h1" groupId={null} />);
    expect(screen.queryByRole('tab', { name: '포트 포워딩' })).toBeNull();
    fireEvent.click(screen.getByRole('tab', { name: '자동화' }));
    expect(screen.getByText('이 서버의 출력 트리거')).toBeTruthy();
    expect(screen.queryByText(/로그인 스크립트: 연결 직후/)).toBeNull();
  });

  it('텔넷에는 자동화 탭이 있고 비밀번호 자리표시자 안내는 없다', () => {
    seed({ hosts: [{ ...host, protocol: 'telnet', port: 23 }] });
    render(<HostEditor hostId="h1" groupId={null} />);
    fireEvent.click(screen.getByRole('tab', { name: '자동화' }));
    expect(screen.queryByText(/\{\{password\}\}/)).toBeNull();
  });
});

describe('설정 · 출력 규칙 (D140)', () => {
  it('키워드 강조와 출력 트리거가 한 구역에 있고, 트리거 규칙을 추가하면 저장된다', async () => {
    seed();
    render(<SettingsRules />);
    expect(screen.getByText('키워드 강조')).toBeTruthy();
    expect(screen.getByText('출력 트리거')).toBeTruthy();
    const add = screen.getAllByRole('button', { name: '+ 규칙 추가' });
    expect(add).toHaveLength(2);
    await act(async () => { fireEvent.click(add[1]); });
    const rules = patches.at(-1)?.settings?.triggerRules;
    expect(rules).toHaveLength(1);
    expect(rules?.[0]).toMatchObject({ notify: true, sound: false, replyOn: false, enabled: true });
  });

  it('자동 응답을 켜야 응답 글자 칸이 보인다', async () => {
    seed();
    render(<SettingsRules />);
    await act(async () => { fireEvent.click(screen.getAllByRole('button', { name: '+ 규칙 추가' })[1]); });
    expect(screen.queryByLabelText('응답으로 입력할 글자')).toBeNull();
    await act(async () => { fireEvent.click(screen.getByLabelText('자동 응답')); });
    expect(screen.getByLabelText('응답으로 입력할 글자')).toBeTruthy();
    expect(patches.at(-1)?.settings?.triggerRules?.[0].replyOn).toBe(true);
  });
});

describe('포트 포워딩 정보 창 (D137)', () => {
  it('포워딩마다 한 줄씩 종류·듣는 주소·대상·상태를 보인다', () => {
    seed();
    render(
      <ForwardsDialog
        title="gw"
        target="samsung@100.102.204.43:22"
        onClose={() => undefined}
        forwards={[
          { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
          { type: 'D', bind: '127.0.0.1:1080' },
          { type: 'R', bind: '0.0.0.0:9000', dest: 'localhost:9000', error: 'address already in use' },
        ]}
      />,
    );
    const dialog = screen.getByRole('dialog', { name: '포트 포워딩 · gw' });
    expect(within(dialog).getByText(/3개 설정됨, 1개 실패/)).toBeTruthy();
    const rows = within(dialog).getAllByRole('row');
    expect(rows).toHaveLength(4); // 머리 + 3
    expect(within(rows[1]).getByText('127.0.0.1:25095')).toBeTruthy();
    expect(within(rows[1]).getByText('100.102.205.95:22')).toBeTruthy();
    expect(within(rows[1]).getByText('동작 중')).toBeTruthy();
    expect(within(rows[2]).getByText('SOCKS 프록시')).toBeTruthy();
    expect(within(rows[3]).getByText('실패: address already in use')).toBeTruthy();
  });

  it('열 때 연결 수를 한 번 읽어 "동작 중 · 연결 N개"로 보이고, 주기적으로 다시 묻지 않는다 (D142)', async () => {
    seed();
    const asked: unknown[][] = [];
    let conns = 2;
    setBackend({
      invoke: (async (channel: string, args: unknown[] = []) => {
        if (channel !== 'session.forwards') return undefined;
        asked.push(args);
        return [
          { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22', conns },
          { type: 'L', bind: '127.0.0.1:25096', dest: '100.102.205.96:22', conns: 0 },
        ];
      }) as unknown as Backend['invoke'],
      termInput, termResize: () => undefined, reply: () => undefined, on: () => () => undefined,
    });
    vi.useFakeTimers();
    try {
      const view = render(
        <ForwardsDialog
          sessionId="s1"
          title="gw"
          target="samsung@100.102.204.43:22"
          onClose={() => undefined}
          forwards={[
            { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
            { type: 'L', bind: '127.0.0.1:25096', dest: '100.102.205.96:22' },
          ]}
        />,
      );
      await act(async () => { await vi.advanceTimersByTimeAsync(0); });
      expect(asked[0]).toEqual(['s1']);
      let rows = screen.getAllByRole('row');
      expect(rows[1].textContent).toContain('동작 중 · 연결 2개');
      expect(rows[2].textContent).toContain('동작 중');
      expect(rows[2].textContent).not.toContain('연결');
      expect(screen.getByText(/2개 설정됨 · 연결 2개/)).toBeTruthy();

      // 시간이 지나도 다시 묻지 않는다.
      conns = 0;
      await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
      expect(asked).toHaveLength(1);
      expect(screen.getAllByRole('row')[1].textContent).toContain('연결 2개');

      // [새로고침]을 누르면 그때 다시 읽는다.
      fireEvent.click(screen.getByRole('button', { name: '새로고침' }));
      await act(async () => { await vi.advanceTimersByTimeAsync(0); });
      expect(asked).toHaveLength(2);
      rows = screen.getAllByRole('row');
      expect(rows[1].textContent).not.toContain('연결');
      view.unmount();
    } finally {
      vi.useRealTimers();
    }
  });
});
