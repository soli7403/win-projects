// 수용 시나리오 L1~L7(spec/acceptance/01-lock-vault.md)을 가짜 백엔드로 확인한다.
import { beforeEach, describe, expect, it } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createFakeBackend } from '../ipc/fakeBackend';
import { setBackend, getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { Lock } from './Lock';

async function setup(initialized: boolean) {
  setBackend(createFakeBackend());
  await act(async () => {
    useApp.setState({
      booted: true,
      vault: { initialized, unlocked: false, osUnlockAvailable: false, osUnlockEnabled: false },
      recoveryPending: false,
      pendingRecoveryCode: '',
      toasts: [],
    });
  });
  if (initialized) {
    // 볼트를 만든 뒤 잠근다.
    await getBackend().invoke('vault.setup', ['Correct-Horse-9']);
    await getBackend().invoke('vault.lock');
    await act(async () => {
      await useApp.getState().refreshVault();
    });
  }
  return render(<Lock />);
}

const submitOf = (container: HTMLElement) => container.querySelector('form') as HTMLFormElement;

describe('잠금 화면 최초 설정 (L1~L4)', () => {
  beforeEach(() => setBackend(null));

  it('L1: 보관함이 없으면 시작하기 카드와 첫 입력창 포커스', async () => {
    await setup(false);
    expect(screen.getByText('DariTerm 처음 설정')).toBeTruthy();
    const pw = screen.getByPlaceholderText('마스터 비밀번호 (8자 이상)');
    expect(document.activeElement).toBe(pw);
  });

  it('L2: 8자 미만이면 오류, 보관함은 만들어지지 않는다', async () => {
    const { container } = await setup(false);
    fireEvent.change(screen.getByPlaceholderText('마스터 비밀번호 (8자 이상)'), { target: { value: 'abc' } });
    fireEvent.submit(submitOf(container));
    expect(await screen.findByText('8자 이상 입력하세요')).toBeTruthy();
    expect((await getBackend().invoke('vault.status')).initialized).toBe(false);
  });

  it('L3: 확인이 다르면 불일치 오류', async () => {
    const { container } = await setup(false);
    fireEvent.change(screen.getByPlaceholderText('마스터 비밀번호 (8자 이상)'), { target: { value: 'Abcdefgh1!x2' } });
    fireEvent.change(screen.getByPlaceholderText('비밀번호 확인'), { target: { value: 'other' } });
    fireEvent.submit(submitOf(container));
    expect(await screen.findByText('비밀번호가 일치하지 않습니다')).toBeTruthy();
  });

  it('L4: 강한 비밀번호는 강도 "매우 강함"', async () => {
    await setup(false);
    fireEvent.change(screen.getByPlaceholderText('마스터 비밀번호 (8자 이상)'), { target: { value: 'Abcdefgh1!x2' } });
    expect(screen.getByText('강도: 매우 강함')).toBeTruthy();
  });

  it('L5·L6: 만들기 후 복구 코드 화면, 체크 전에는 시작하기가 비활성', async () => {
    const { container } = await setup(false);
    const pw = 'Abcdefgh1!x2';
    fireEvent.change(screen.getByPlaceholderText('마스터 비밀번호 (8자 이상)'), { target: { value: pw } });
    fireEvent.change(screen.getByPlaceholderText('비밀번호 확인'), { target: { value: pw } });
    fireEvent.submit(submitOf(container));
    const start = await screen.findByRole('button', { name: /시작하기/ });
    expect((start as HTMLButtonElement).disabled).toBe(true);
    const box = container.querySelector('.recovery-box')!.textContent!;
    expect(box).toMatch(/^[A-Z2-9]{4}(-[A-Z2-9]{4}){5}$/);
    fireEvent.click(screen.getByRole('checkbox'));
    expect((start as HTMLButtonElement).disabled).toBe(false);
  });
});

describe('잠금 해제 (L7)', () => {
  beforeEach(() => setBackend(null));

  it('L7: 틀린 비밀번호는 오류 표시 후 입력창을 비운다', async () => {
    const { container } = await setup(true);
    const input = screen.getByPlaceholderText('마스터 비밀번호') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'wrong-pass' } });
    fireEvent.submit(submitOf(container));
    expect(await screen.findByText('비밀번호가 올바르지 않습니다')).toBeTruthy();
    await waitFor(() => expect(input.value).toBe(''));
  });
});
