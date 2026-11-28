// SSH 키 만들기·공개 키 보기·서버 등록, HEX 보내기 (D162·D163).
// 근거: spec/ui/settings.md §6.3, spec/ui/dialogs.md §8·§9, spec/behavior/ssh-keys.md, spec/behavior/serial-tools.md.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Host, Store } from '../ipc/types';
import { KeyGenerateDialog, KeyInstallDialog, PublicKeyDialog } from './KeyDialogs';
import { HexSendDialog, hexByteCount } from './HexSendDialog';

const PUB = { publicKey: 'ssh-ed25519 AAAAC3Nza user@pc', fingerprint: 'SHA256:abc' };
let invoke: ReturnType<typeof vi.fn>;

function useBackend(handlers: Record<string, (args: unknown[]) => unknown>) {
  invoke = vi.fn(async (channel: string, args: unknown[] = []) => {
    const fn = handlers[channel];
    return fn ? fn(args) : undefined;
  });
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
}

const host: Host = { id: 'h1', alias: 'web', host: '10.0.0.1', port: 22, username: 'root', protocol: 'ssh', authType: 'password', credentialId: 'pw1' } as Host;

beforeEach(() => {
  useApp.setState({ store: { version: 1, groups: [], hosts: [host], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store, toasts: [] } as never);
});

describe('SSH 키 만들기 (D162)', () => {
  it('이름이 있어야 만들 수 있고, 종류·키 암호를 보낸다', async () => {
    useBackend({ 'key.generate': () => ({ cred: { id: 'k1', name: '노트북', kind: 'key', updatedAt: 1 }, ...PUB }) });
    const onDone = vi.fn();
    render(<KeyGenerateDialog onCancel={() => undefined} onDone={onDone} />);
    const create = screen.getByRole('button', { name: '만들기' });
    expect((create as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText('이름'), { target: { value: ' 노트북 ' } });
    fireEvent.click(screen.getByRole('button', { name: 'RSA 4096 (오래된 서버용)' }));
    fireEvent.change(screen.getByLabelText('키 암호 (선택)'), { target: { value: 'pp' } });
    // 키 암호를 넣으면 확인 칸이 생기고, 다르면 만들 수 없다.
    expect((create as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText('키 암호 확인'), { target: { value: 'px' } });
    expect(screen.getByText('키 암호가 서로 다릅니다')).toBeTruthy();
    fireEvent.change(screen.getByLabelText('키 암호 확인'), { target: { value: 'pp' } });
    fireEvent.click(create);
    await waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(invoke).toHaveBeenCalledWith('key.generate', [{ name: '노트북', type: 'rsa', bits: 4096, passphrase: 'pp' }]);
  });

  it('실패하면 창에 까닭을 보이고 다시 누를 수 있다', async () => {
    useBackend({ 'key.generate': () => { throw new Error('앱이 잠겨 있습니다'); } });
    render(<KeyGenerateDialog onCancel={() => undefined} onDone={() => undefined} />);
    fireEvent.change(screen.getByLabelText('이름'), { target: { value: 'k' } });
    fireEvent.click(screen.getByRole('button', { name: '만들기' }));
    expect(await screen.findByText('앱이 잠겨 있습니다')).toBeTruthy();
    expect((screen.getByRole('button', { name: '만들기' }) as HTMLButtonElement).disabled).toBe(false);
  });
});

describe('공개 키 창 (D162)', () => {
  it('계정 id 로 공개 키를 읽어 보이고 복사한다', async () => {
    useBackend({ 'key.publicKey': () => PUB });
    render(<PublicKeyDialog credId="k1" name="노트북" onClose={() => undefined} />);
    expect(await screen.findByDisplayValue(PUB.publicKey)).toBeTruthy();
    expect(screen.getByText('SHA256:abc')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '복사' }));
    await waitFor(() => expect(invoke).toHaveBeenCalledWith('clip.writeText', [PUB.publicKey]));
    expect(invoke).toHaveBeenCalledWith('key.publicKey', ['k1']);
  });
});

describe('공개 키 서버 등록 (D162)', () => {
  it('키 계정만 고를 수 있고, 등록하면 서버를 이 키로 로그인하게 바꾼다', async () => {
    useBackend({
      'cred.list': () => [{ id: 'pw1', name: '비번', kind: 'password', updatedAt: 1 }, { id: 'k1', name: '노트북', kind: 'key', updatedAt: 1 }],
      'key.install': () => ({ status: 'added' }),
    });
    const onClose = vi.fn();
    render(<KeyInstallDialog sessionId="s1" target="root@10.0.0.1:22" hostId="h1" onClose={onClose} />);
    const select = await screen.findByLabelText('등록할 키');
    expect(Array.from((select as HTMLSelectElement).options).map((o) => o.textContent)).toEqual(['노트북']);
    fireEvent.click(screen.getByRole('button', { name: '등록' }));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(invoke).toHaveBeenCalledWith('key.install', ['s1', 'k1']);
    const patch = invoke.mock.calls.find((c) => c[0] === 'store.patch');
    expect(patch?.[1]).toEqual([{ hosts: [{ ...host, authType: 'key', credentialId: 'k1' }] }]);
  });

  it('"다음 접속부터 이 키로" 를 끄면 서버 설정을 바꾸지 않는다', async () => {
    useBackend({ 'cred.list': () => [{ id: 'k1', name: '노트북', kind: 'key', updatedAt: 1 }], 'key.install': () => ({ status: 'exists' }) });
    const onClose = vi.fn();
    render(<KeyInstallDialog sessionId="s1" target="t" hostId="h1" onClose={onClose} />);
    await screen.findByLabelText('등록할 키');
    fireEvent.click(screen.getByRole('switch'));
    fireEvent.click(screen.getByRole('button', { name: '등록' }));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(invoke.mock.calls.some((c) => c[0] === 'store.patch')).toBe(false);
  });

  it('키가 없으면 안내하고 등록 단추가 꺼진다', async () => {
    useBackend({ 'cred.list': () => [] });
    render(<KeyInstallDialog sessionId="s1" target="t" onClose={() => undefined} />);
    expect(await screen.findByText('보관함에 SSH 키가 없습니다. 먼저 키를 만드세요.')).toBeTruthy();
    expect((screen.getByRole('button', { name: '등록' }) as HTMLButtonElement).disabled).toBe(true);
  });
});

describe('HEX 보내기 (D163)', () => {
  it('바이트 수를 세는 규칙이 백엔드와 같다', () => {
    expect(hexByteCount('01 03 0a FF')).toBe(4);
    expect(hexByteCount('01030AFF')).toBe(4);
    expect(hexByteCount('0x01,0x03;0X0a')).toBe(3);
    expect(hexByteCount('a b')).toBe(2);
    for (const bad of ['', '  ', '123', '0g', '0x', '+1']) expect(hexByteCount(bad)).toBeNull();
  });

  it('16진이 아니면 보낼 수 없고, 맞으면 입력 그대로 보낸다', async () => {
    useBackend({ 'session.sendHex': () => 3 });
    render(<HexSendDialog sessionId="s1" onClose={() => undefined} />);
    const input = screen.getByRole('textbox', { name: 'HEX 보내기' });
    fireEvent.change(input, { target: { value: '01 zz' } });
    expect((screen.getByRole('button', { name: '보내기' }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(input, { target: { value: '01 03 FF' } });
    expect(screen.getByText('3바이트')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '보내기' }));
    await waitFor(() => expect(invoke).toHaveBeenCalledWith('session.sendHex', ['s1', '01 03 FF']));
  });
});
