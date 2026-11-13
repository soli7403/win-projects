// 가짜 백엔드의 기본 채널 동작(spec/ipc.md §1·§2 의 오류 문구 포함).
import { describe, expect, it } from 'vitest';
import { createFakeBackend } from './fakeBackend';

describe('가짜 백엔드', () => {
  it('알 수 없는 채널은 명세의 오류 문구로 거부한다', async () => {
    const b = createFakeBackend();
    await expect(b.invoke('no.such')).rejects.toThrow('구현되지 않은 채널: no.such');
  });

  it('보관함 설정 후 복구 코드는 24자 영숫자이며 잠금 해제가 동작한다', async () => {
    const b = createFakeBackend();
    const res = await b.invoke<{ recoveryCode: string }>('vault.setup', ['pw-12345678']);
    expect(res.recoveryCode.replace(/-/g, '')).toHaveLength(24);
    expect(await b.invoke('vault.unlock', ['nope'])).toBe(false);
    await b.invoke('vault.lock');
    expect(await b.invoke('vault.unlock', ['pw-12345678'])).toBe(true);
  });

  it('저장 데이터는 시드 서버 4개를 가진다', async () => {
    const b = createFakeBackend();
    const store = await b.invoke('store.get');
    expect(store.hosts).toHaveLength(4);
    expect(store.settings.autoLockMinutes).toBe(15);
  });
});
