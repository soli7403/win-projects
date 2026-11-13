// C1 수용 시나리오의 화면 쪽 검증. 근거: spec/acceptance/04 (G1·G3·G5·G7·G8, Y1·Y8, I6 은 C1.test.tsx).
// 백엔드는 mock 으로 대체하고, 화면이 보내는 채널·인자와 보여 주는 토스트·대화상자를 확인한다.
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useUpdate } from '../state/update';
import { DEFAULT_SETTINGS, DANGEROUS_PATTERNS } from '../lib/defaults';
import type { Store, Host } from '../ipc/types';
import { SettingsInput } from './SettingsInput';
import { SettingsAppearance } from './SettingsAppearance';
import { SettingsSecurity } from './SettingsSecurity';
import { SettingsSync } from './SettingsSync';
import { SettingsBackup } from './SettingsBackup';

type Calls = [string, unknown[]][];

/** 메모리 저장소를 가진 mock 백엔드. 호출 기록을 돌려준다. */
function mockBackend(extra: Record<string, (args: unknown[]) => unknown> = {}): { calls: Calls; store: () => Store } {
  const calls: Calls = [];
  let store = useApp.getState().store as Store;
  const handlers: Record<string, (args: unknown[]) => unknown> = {
    'store.get': () => structuredClone(store),
    'store.patch': ([patch]) => {
      store = { ...store, ...(patch as object) } as Store;
      return undefined;
    },
    'cred.list': () => [],
    ...extra,
  };
  const b: Backend = {
    async invoke(channel: string, args: unknown[] = []): Promise<any> {
      calls.push([channel, args]);
      const h = handlers[channel];
      if (!h) throw new Error(`mock 없음: ${channel}`);
      return h(args);
    },
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  return { calls, store: () => store };
}

function seed(hosts: Host[] = []) {
  const store: Store = {
    version: 1,
    groups: [],
    hosts,
    snippets: [],
    knownHosts: [],
    settings: { ...DEFAULT_SETTINGS, dangerousPatterns: [...DANGEROUS_PATTERNS] },
  };
  useApp.setState({
    booted: true,
    store,
    vault: { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
  });
  useUi.setState({ dialogs: [] });
  return store;
}

const HOST: Host = {
  id: 'h1', groupId: null, alias: 'web01', protocol: 'ssh', host: '10.0.0.2', port: 22, username: 'root',
  authType: 'password', credentialId: 'c1', tags: [], encoding: 'utf-8', termType: 'xterm-256color',
  keepaliveSec: 30, forwards: [], sort: 0,
};

beforeEach(() => {
  seed();
});

describe('G3 위험 명령 패턴 편집', () => {
  it('줄 단위로 나누고 앞뒤 공백·빈 줄을 뺀 뒤 저장한다', async () => {
    seed();
    const m = mockBackend({ 'app.info': () => ({ platform: 'windows' }) });
    render(<SettingsInput />);
    const ta = screen.getByLabelText('위험 명령 목록') as HTMLTextAreaElement;
    fireEvent.change(ta, { target: { value: '  rm\\s+-rf  \n\n\nreboot\n   \n' } });
    fireEvent.blur(ta);
    await waitFor(() => {
      const patch = m.calls.find(([c]) => c === 'store.patch');
      expect(patch).toBeTruthy();
      const settings = (patch![1][0] as { settings: { dangerousPatterns: string[] } }).settings;
      expect(settings.dangerousPatterns).toEqual(['rm\\s+-rf', 'reboot']);
    });
  });

  it('[기본값으로] 는 기본 패턴 목록으로 되돌린다', async () => {
    seed();
    const m = mockBackend({ 'app.info': () => ({ platform: 'windows' }) });
    render(<SettingsInput />);
    fireEvent.click(screen.getByRole('button', { name: '기본값으로' }));
    await waitFor(() => {
      const patch = m.calls.find(([c]) => c === 'store.patch');
      const settings = (patch![1][0] as { settings: { dangerousPatterns: string[] } }).settings;
      expect(settings.dangerousPatterns).toEqual(DANGEROUS_PATTERNS);
    });
  });
});

describe('G1 앱 테마와 터미널 테마 짝', () => {
  it('짝 테마를 쓰는 중에 라이트로 바꾸면 DariTerm Light 로 같이 바뀐다', async () => {
    seed();
    const m = mockBackend({ 'app.info': () => ({}) });
    render(<SettingsAppearance />);
    fireEvent.click(screen.getByRole('button', { name: '라이트' }));
    await waitFor(() => {
      const patches = m.calls.filter(([c]) => c === 'store.patch');
      const last = patches[patches.length - 1][1][0] as { settings: { theme: string; terminalTheme: string } };
      expect(last.settings.theme).toBe('light');
      expect(last.settings.terminalTheme).toBe('DariTerm Light');
    });
  });
});

describe('G8 [원래 크기로]', () => {
  it('app.resetWindow 를 부른다', async () => {
    seed();
    const m = mockBackend();
    render(<SettingsAppearance />);
    fireEvent.click(screen.getByRole('button', { name: '기본 크기로' }));
    await waitFor(() => expect(m.calls.some(([c]) => c === 'app.resetWindow')).toBe(true));
  });
});

describe('G5 저장된 계정 삭제', () => {
  it('사용 중인 서버 수를 확인 문구에 넣고, 확인하면 cred.remove 를 부른다', async () => {
    seed([HOST]);
    const m = mockBackend({
      'cred.list': () => [{ id: 'c1', name: '배포 계정', kind: 'password', username: 'deploy', updatedAt: 1 }],
      'cred.remove': () => undefined,
    });
    render(<SettingsSecurity />);
    fireEvent.click(await screen.findByRole('button', { name: '삭제' }));
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
    const dlg = useUi.getState().dialogs[0];
    expect(dlg.kind).toBe('confirm');
    if (dlg.kind === 'confirm') {
      expect(dlg.opts.title).toBe('계정 삭제');
      expect(dlg.opts.body).toBe('"배포 계정" 계정을 삭제할까요? 1개 서버가 연결할 때 비밀번호를 다시 묻게 됩니다.');
      expect(dlg.opts.okText).toBe('삭제');
      expect(dlg.opts.danger).toBe(true);
    }
    useUi.getState().settleDialog(true);
    await waitFor(() => expect(m.calls.some(([c, a]) => c === 'cred.remove' && a[0] === 'c1')).toBe(true));
  });
});

describe('G7 [지금 확인] 결과 토스트', () => {
  it('배포 저장소가 없는 빌드이면 그 문구를 보인다', async () => {
    seed();
    mockBackend({ 'update.check': () => ({ status: 'unconfigured' }) });
    await useUpdate.getState().checkNow();
    expect(useApp.getState().toasts.map((t) => t.text)).toContain('배포 저장소가 설정되지 않은 빌드입니다');
  });

  it('새 버전이면 정보 토스트와 상태 표시 버전을 갱신한다', async () => {
    seed();
    mockBackend({ 'update.check': () => ({ status: 'newer', version: '0.2.0', url: 'https://example.test/r' }) });
    await useUpdate.getState().checkNow();
    expect(useApp.getState().toasts.map((t) => t.text)).toContain('새 버전 0.2.0 이 있습니다');
    expect(useUpdate.getState().available?.version).toBe('0.2.0');
  });
});

describe('Y1 백업 내보내기', () => {
  it('저장한 경로를 토스트로 보인다', async () => {
    seed();
    mockBackend({ 'backup.export': () => 'D:/백업/dariterm-backup-2026-11-10.dtbackup' });
    render(<SettingsBackup />);
    fireEvent.click(screen.getByRole('button', { name: '백업 내보내기' }));
    await waitFor(() =>
      expect(useApp.getState().toasts.map((t) => t.text)).toContain('백업 저장: D:/백업/dariterm-backup-2026-11-10.dtbackup'),
    );
  });

  it('취소(false)이면 토스트가 없다', async () => {
    seed();
    mockBackend({ 'backup.export': () => false });
    render(<SettingsBackup />);
    fireEvent.click(screen.getByRole('button', { name: '백업 내보내기' }));
    await waitFor(() => expect(useApp.getState().toasts.length).toBe(0));
  });
});

describe('Y8 저장소 볼트 불일치 카드', () => {
  it('카드를 보이고, 덮어쓰기는 확인을 거쳐 overwrite 모드로 실행한다', async () => {
    seed();
    const m = mockBackend({
      'sync.status': () => ({ configured: true, gitAvailable: true }),
      'sync.run': (args) => (args[0] === 'overwrite' ? { state: 'ok', lastSync: Date.now(), changed: true } : { state: 'vault-mismatch' }),
    });
    useApp.setState({ store: { ...useApp.getState().store!, settings: { ...DEFAULT_SETTINGS, syncRepo: 'git@example.test:me/sync.git' } } });
    render(<SettingsSync />);
    fireEvent.click(screen.getByRole('button', { name: /지금 동기화/ }));
    expect(await screen.findByText('저장소의 암호화 데이터가 이 PC 와 다릅니다')).toBeTruthy();
    expect(screen.getByRole('button', { name: '저장소 데이터 받기' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '이 PC 데이터로 덮어쓰기' }));
    await waitFor(() => expect(useUi.getState().dialogs.length).toBe(1));
    const dlg = useUi.getState().dialogs[0];
    if (dlg.kind === 'confirm') expect(dlg.opts.okText).toBe('덮어쓰기');
    useUi.getState().settleDialog(true);
    await waitFor(() => expect(m.calls.some(([c, a]) => c === 'sync.run' && a[0] === 'overwrite')).toBe(true));
  });
});
