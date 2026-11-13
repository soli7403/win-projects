// 정보 화면 고지(D63): 참고한 프로젝트 BrightTerm v1.1.0 의 MIT 저작권 고지와 허가 문구 전문이 앱 안에 있어야 한다.
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { SettingsAbout } from './SettingsAbout';

describe('정보 화면 — 참고한 프로젝트 고지 (D63)', () => {
  it('원저작자 저작권 줄과 MIT 허가 문구를 보인다', async () => {
    const b: Backend = {
      invoke: async (ch: string) => (ch === 'app.info' ? { version: '1.0.0', dataDir: 'C:\\data' } : undefined) as never,
      termInput: () => undefined,
      termResize: () => undefined,
      reply: () => undefined,
      on: () => () => undefined,
    };
    setBackend(b);
    useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store });
    const { container } = render(<SettingsAbout />);
    expect(await screen.findByText('참고한 프로젝트')).toBeTruthy();
    expect(screen.getByText('BrightTerm v1.1.0 — MIT · Copyright (c) 2026 Dany Kim')).toBeTruthy();
    const full = container.querySelector('details .detail-box')?.textContent ?? '';
    expect(full).toContain('Copyright (c) 2026 Dany Kim');
    expect(full).toContain('The above copyright notice and this permission notice shall be included in all');
  });
});
