// 정보 화면 고지(D63): 참고한 프로젝트 BrightTerm v1.1.0 의 MIT 저작권 고지와 허가 문구 전문이 앱 안에 있어야 한다.
// 사용 조건(D73): 개인 무료 · 업무용 유료 요약과 LICENSE 전문, 설정된 외부 링크만 버튼으로.
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { SettingsAbout } from './SettingsAbout';
import { aboutLinks, safeUrl, LICENSE_CONTACT } from '../lib/links';

function setup() {
  const b: Backend = {
    invoke: async (ch: string) => (ch === 'app.info' ? { version: '1.0.0', dataDir: 'C:\\data' } : undefined) as never,
    termInput: () => undefined,
    termResize: () => undefined,
    reply: () => undefined,
    on: () => () => undefined,
  };
  setBackend(b);
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store });
  return render(<SettingsAbout />);
}

/** 요약 문구(summary)로 펼침 상자를 찾아 그 안의 본문을 돌려준다. */
function detailText(container: HTMLElement, summary: string): string {
  const d = [...container.querySelectorAll('details')].find((x) => x.querySelector('summary')?.textContent === summary);
  return d?.querySelector('.detail-box')?.textContent ?? '';
}

describe('정보 화면 — 참고한 프로젝트 고지 (D63)', () => {
  it('원저작자 저작권 줄과 MIT 허가 문구를 보인다', async () => {
    const { container } = setup();
    expect(await screen.findByText('참고한 프로젝트')).toBeTruthy();
    expect(screen.getByText('BrightTerm v1.1.0 — MIT · Copyright (c) 2026 Dany Kim')).toBeTruthy();
    const full = detailText(container, '라이선스 전문 보기');
    expect(full).toContain('Copyright (c) 2026 Dany Kim');
    expect(full).toContain('The above copyright notice and this permission notice shall be included in all');
    // 내 코드는 독점 라이선스(D64), 오픈소스 구성 요소 고지가 앱 안에 있다.
    expect(screen.getByText('독점 소프트웨어 · Copyright © 2026 kai. All rights reserved.')).toBeTruthy();
    const oss = detailText(container, '오픈소스 라이선스 보기');
    expect(oss).toContain('github.com/wailsapp/wails/v2');
    expect(oss).toContain('@xterm/xterm');
  });
});

describe('정보 화면 — 사용 조건 (D73)', () => {
  it('개인 무료·업무용 유료 요약과 LICENSE 전문을 보인다', async () => {
    const { container } = setup();
    expect(await screen.findByText('사용 조건')).toBeTruthy();
    expect(screen.getByText('개인 사용은 무료 · 업무 사용은 사용자별 업무용 라이선스')).toBeTruthy();
    const full = detailText(container, '사용 조건 전문 보기');
    expect(full).toContain('2.1 개인 사용(무료)');
    expect(full).toContain('2.2 Business use (paid)');
  });

  it('업무용 라이선스 문의 연락처를 보이고, LICENSE 전문의 연락처와 같다 (D74)', async () => {
    const { container } = setup();
    await screen.findByText('사용 조건');
    expect(screen.getByText(LICENSE_CONTACT)).toBeTruthy();
    expect(screen.getByRole('button', { name: '복사' })).toBeTruthy();
    expect(detailText(container, '사용 조건 전문 보기')).toContain(`업무용 라이선스 구매·문의: ${LICENSE_CONTACT}`);
  });

  it('주소가 없으면 링크 버튼을 숨긴다(테스트 빌드는 VITE_DARITERM_*_URL 미설정)', async () => {
    setup();
    await screen.findByText('사용 조건');
    expect(screen.queryByRole('button', { name: '후원하기' })).toBeNull();
    expect(screen.queryByRole('button', { name: '업무용 라이선스 구매' })).toBeNull();
  });
});

describe('외부 링크 설정 (lib/links)', () => {
  it('http(s) 주소만 쓰고, 설정된 것만 구매 → 후원 → 홈페이지 순서로', () => {
    expect(safeUrl(' https://example.com/buy ')).toBe('https://example.com/buy');
    expect(safeUrl('javascript:alert(1)')).toBe('');
    expect(safeUrl(undefined)).toBe('');
    const links = aboutLinks({
      VITE_DARITERM_HOMEPAGE_URL: 'https://dariterm.example',
      VITE_DARITERM_DONATE_URL: 'https://ko-fi.example/dariterm',
      VITE_DARITERM_PURCHASE_URL: 'file:///c:/x',
    });
    expect(links.map((l) => l.key)).toEqual(['donate', 'homepage']);
  });
});
