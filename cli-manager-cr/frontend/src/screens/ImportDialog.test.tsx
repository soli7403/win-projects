// 가져오기 대화상자 테스트: 실제 백엔드 응답 모양(spec/ipc.md "가져오기 후보 객체")의 샘플을 넣는다.
// PuTTY SSH 1건, ssh config 1건(keyFile·jumpVia 포함), 시리얼 1건, 그리고 host 가 빠진 불완전한 후보 1건.
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { ImportDialog, hostOf, sameHostExists } from './ImportDialog';

const SAMPLE = [
  {
    source: 'putty',
    name: 'prod-web02',
    host: { alias: 'prod-web02', protocol: 'ssh', host: '10.1.1.2', port: 22, username: 'ubuntu', authType: 'password', keepaliveSec: 30 },
  },
  {
    source: 'ssh-config',
    name: 'bastion',
    host: { alias: 'bastion', protocol: 'ssh', host: '203.0.113.9', port: 22, username: 'ec2-user', authType: 'key', keepaliveSec: 30 },
    keyFile: '~/.ssh/bastion.pem',
  },
  {
    source: 'putty',
    name: 'serial-lab',
    host: { alias: 'serial-lab', protocol: 'serial', host: '', port: 0, authType: 'password', serial: { path: '/dev/ttyUSB0', baudRate: 115200 } },
  },
  // host 가 아예 없는 후보: 화면이 예외 없이 그려야 한다.
  { source: 'putty', name: 'broken-entry' },
];

function mockBackend(scan: unknown) {
  const calls: [string, unknown[]][] = [];
  const b: Backend = {
    async invoke(channel: string, args: unknown[] = []): Promise<any> {
      calls.push([channel, args]);
      if (channel === 'import.scan') return scan;
      if (channel === 'import.apply') return { hosts: (args[0] as unknown[]).length, keys: 0, skipped: [] };
      if (channel === 'store.get') return structuredClone(useApp.getState().store);
      if (channel === 'vault.status') return { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false };
      return undefined;
    },
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  return calls;
}

function seed(hosts: Store['hosts'] = []) {
  useApp.setState({
    booted: true,
    store: { version: 1, groups: [], hosts, snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } },
    vault: { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false },
    toasts: [],
  });
  useUi.setState({ importOpen: true, dialogs: [] });
}

beforeEach(() => {
  seed();
});

describe('가져오기 대화상자: 실제 후보 구조', () => {
  it('샘플 후보를 모두 그리고, 종류·주소·출처를 host 객체에서 읽는다', async () => {
    mockBackend(SAMPLE);
    render(<ImportDialog />);
    expect(await screen.findByText('prod-web02')).toBeTruthy();
    expect(screen.getByText('bastion')).toBeTruthy();
    expect(screen.getByText('serial-lab')).toBeTruthy();
    // PuTTY SSH 주소: 사용자@호스트:포트
    expect(screen.getByText('ubuntu@10.1.1.2:22')).toBeTruthy();
    // 시리얼 주소: 장치 경로 @ 속도
    expect(screen.getByText('/dev/ttyUSB0 @ 115200')).toBeTruthy();
    // ssh config 의 키 파일 표시
    expect(screen.getByLabelText('키 파일 있음')).toBeTruthy();
    // 출처 표기
    expect(screen.getAllByText('PuTTY').length).toBeGreaterThan(0);
    expect(screen.getAllByText('ssh config').length).toBeGreaterThan(0);
  });

  it('host 가 없는 후보도 예외 없이 그린다(종류는 SSH 기본)', async () => {
    mockBackend(SAMPLE);
    render(<ImportDialog />);
    expect(await screen.findByText('broken-entry')).toBeTruthy();
    const row = screen.getByText('broken-entry').closest('tr')!;
    expect(row.textContent).toContain('SSH');
  });

  it('초기 선택은 모두 선택(기존 서버와 같은 호스트가 없을 때)', async () => {
    mockBackend(SAMPLE);
    render(<ImportDialog />);
    await screen.findByText('prod-web02');
    expect(screen.getByText('4/4개 선택')).toBeTruthy();
  });

  it('기존 서버와 같은 호스트·포트·사용자가 있으면 초기 선택에서 뺀다', async () => {
    seed([{ id: 'h1', host: '10.1.1.2', port: 22, username: 'ubuntu' } as Store['hosts'][number]]);
    mockBackend(SAMPLE);
    render(<ImportDialog />);
    await screen.findByText('prod-web02');
    expect(screen.getByText('3/4개 선택')).toBeTruthy();
  });

  it('가져오기는 후보 객체를 변경 없이 보내고 폴더 이름을 두 번째 인자로 보낸다', async () => {
    const calls = mockBackend(SAMPLE);
    render(<ImportDialog />);
    await screen.findByText('prod-web02');
    fireEvent.click(screen.getByRole('button', { name: '4개 가져오기' }));
    await waitFor(() => expect(calls.some(([c]) => c === 'import.apply')).toBe(true));
    const apply = calls.find(([c]) => c === 'import.apply')!;
    expect(apply[1][0]).toEqual(SAMPLE);
    expect(apply[1][1]).toBe('PuTTY 가져오기');
  });
});

describe('후보 방어 읽기 헬퍼', () => {
  it('hostOf 는 host 가 없거나 문자열이면 빈 객체를 준다', () => {
    expect(hostOf({ source: 'putty', name: 'x' } as never)).toEqual({});
    expect(hostOf({ source: 'putty', name: 'x', host: 'oops' } as never)).toEqual({});
    expect(hostOf(null)).toEqual({});
  });

  it('sameHostExists 는 host 객체 안의 값으로 비교한다', () => {
    const c = SAMPLE[0] as never;
    expect(sameHostExists(c, [{ host: '10.1.1.2', port: 22, username: 'ubuntu' }])).toBe(true);
    expect(sameHostExists(c, [{ host: '10.1.1.2', port: 2222, username: 'ubuntu' }])).toBe(false);
  });
});
