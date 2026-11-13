// C1 화면 수용 시나리오 중 화면 쪽 검증. 근거: spec/acceptance/04 (G1·G2·G3·G7, I6, Y4).
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/react';
import { NumberField } from './Settings';
import { sameHostExists } from './ImportDialog';
import { CRED_IN_URL } from './SettingsSync';
import { resolveTermTheme, TERM_THEMES } from '../lib/termThemes';

describe('설정 숫자 입력 보정 (G2)', () => {
  it('범위를 넘는 값은 최댓값으로 제한한다 (100 → 32)', () => {
    const onCommit = vi.fn();
    const { getByLabelText } = render(
      <NumberField label="글자 크기" value={14} min={8} max={32} fallback={14} onCommit={onCommit} />,
    );
    const input = getByLabelText('글자 크기') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '100' } });
    fireEvent.blur(input);
    expect(onCommit).toHaveBeenCalledWith(32);
  });

  it('비우면 기본값(14)으로 되돌린다', () => {
    const onCommit = vi.fn();
    const { getByLabelText } = render(
      <NumberField label="글자 크기" value={20} min={8} max={32} fallback={14} onCommit={onCommit} />,
    );
    const input = getByLabelText('글자 크기') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '' } });
    fireEvent.blur(input);
    expect(onCommit).toHaveBeenCalledWith(14);
  });

  it('줄 간격은 0.05 단위로 맞춘다', () => {
    const onCommit = vi.fn();
    const { getByLabelText } = render(
      <NumberField label="줄 간격" value={1.15} min={1} max={2} step={0.05} fallback={1.15} onCommit={onCommit} />,
    );
    const input = getByLabelText('줄 간격') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '1.33' } });
    fireEvent.blur(input);
    expect(onCommit).toHaveBeenCalledWith(1.35);
  });
});

describe('앱 테마와 터미널 테마 짝 (G1 기반 데이터)', () => {
  it('DariTerm Light 테마가 목록에 있고 이름으로 찾는다', () => {
    expect(TERM_THEMES.map((t) => t.name)).toContain('DariTerm Light');
    expect(resolveTermTheme('DariTerm Light').name).toBe('DariTerm Light');
  });
});

describe('저장소 주소 비밀번호 검사 (Y4)', () => {
  it('계정:비밀번호@ 형태를 막는다', () => {
    expect(CRED_IN_URL.test('https://user:pw@host/repo.git')).toBe(true);
  });
  it('SSH 형식 주소는 막지 않는다', () => {
    expect(CRED_IN_URL.test('git@github.com:me/dariterm-sync.git')).toBe(false);
    expect(CRED_IN_URL.test('https://github.com/me/dariterm-sync.git')).toBe(false);
  });
});

describe('가져오기 초기 선택 (I6)', () => {
  it('같은 호스트·포트·사용자의 서버가 있으면 중복으로 본다', () => {
    const hosts = [{ host: '10.1.1.2', port: 22, username: 'ubuntu' }];
    const dup = { source: 'putty', name: 'a', host: { host: '10.1.1.2', port: 22, username: 'ubuntu' } } as Parameters<typeof sameHostExists>[0];
    const other = { source: 'putty', name: 'a', host: { host: '10.1.1.2', port: 2222, username: 'ubuntu' } } as Parameters<typeof sameHostExists>[0];
    expect(sameHostExists(dup, hosts)).toBe(true);
    expect(sameHostExists(other, hosts)).toBe(false);
  });
});
