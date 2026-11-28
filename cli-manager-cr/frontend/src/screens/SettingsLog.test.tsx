// 설정의 "로그" 구역(D120): 세션 로그 항목은 "연결"이 아니라 "로그"에 있고, "ANSI 그대로 저장"은 기본 꺼짐이며 누르면 저장된다.
// 근거: spec/ui/settings.md §4·§4a, spec/behavior/session.md §6.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { Settings } from './Settings';

const invoke = vi.fn(async () => undefined);

/** 로그 구역에만 있어야 하는 항목 이름. */
const LOG_ITEMS = ['터미널 출력 기록', '로그 줄마다 시각 표시', 'ANSI 그대로 저장', '로그 보관 기간 (일)', '로그 파일 갱신 주기 (초)'];

beforeEach(() => {
  invoke.mockClear();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store });
});

function openSection(section: 'connection' | 'log') {
  useUi.setState({ settings: { open: true, section } });
  render(<Settings />);
}

/** store.patch 로 보낸 설정 값들(보낸 순서대로). */
function patched(key: string): unknown[] {
  return invoke.mock.calls
    .filter((c) => (c as unknown[])[0] === 'store.patch')
    .map((c) => ((c as unknown[])[1] as [{ settings: Record<string, unknown> }])[0].settings[key]);
}

describe('설정 > 로그 (D120)', () => {
  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (11개)', () => {
    openSection('connection');
    const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
    expect(names).toEqual(['화면', '입력', '단축키', '출력 규칙', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
  });

  it('"연결"에는 로그 항목이 없고 자동 재연결·배치 복원만 남는다', () => {
    openSection('connection');
    for (const name of LOG_ITEMS) expect(screen.queryByText(name), name).toBeNull();
    expect(screen.queryByText('로그 폴더 열기')).toBeNull();
    expect(screen.getAllByRole('switch')).toHaveLength(2);
  });

  it('"로그"에 로그 항목이 모두 있다', () => {
    openSection('log');
    for (const name of LOG_ITEMS) expect(screen.getAllByText(name).length, name).toBeGreaterThan(0);
    expect(screen.getByText('로그 폴더 열기')).toBeTruthy();
    expect(screen.getAllByRole('switch')).toHaveLength(3);
  });

  it('"연결"에서 "로그"를 누르면 로그 구역으로 바뀐다', () => {
    openSection('connection');
    fireEvent.click(within(screen.getByRole('navigation')).getByText('로그'));
    expect(screen.getByText('ANSI 그대로 저장')).toBeTruthy();
  });

  it('"ANSI 그대로 저장"은 기본 꺼짐이고, 누르면 sessionLogAnsi: true 로 저장된다', async () => {
    expect(DEFAULT_SETTINGS.sessionLogAnsi).toBe(false);
    openSection('log');
    const sw = screen.getByRole('switch', { name: 'ANSI 그대로 저장' });
    expect(sw.getAttribute('aria-checked')).toBe('false');
    fireEvent.click(sw);
    await waitFor(() => expect(patched('sessionLogAnsi')).toEqual([true]));
  });

  it('[로그 폴더 열기]는 app.openLogs 를 부른다', async () => {
    openSection('log');
    fireEvent.click(screen.getByText('로그 폴더 열기'));
    await waitFor(() => expect(invoke).toHaveBeenCalledWith('app.openLogs'));
  });
});
