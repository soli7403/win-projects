// 칸 우클릭 메뉴의 세션 도구: 시리얼은 HEX·XMODEM (D163·D164), SSH 는 공개 키 등록 (D162).
// 근거: spec/ui/panes.md §2.3, spec/behavior/serial-tools.md, spec/behavior/ssh-keys.md §3.
import { describe, expect, it, vi } from 'vitest';
import { paneToolItems } from './Panes';
import type { SessionInfo } from '../ipc/types';

const base = { id: 's1', hostId: null, title: 't', target: 'COM3', state: 'connected', canSftp: false } as unknown as SessionInfo;
const labels = (items: ReturnType<typeof paneToolItems>) => items.filter((i) => !('separator' in i && i.separator)).map((i) => (i as { label: string }).label);

describe('paneToolItems', () => {
  it('시리얼: HEX 보기·보내기와 XMODEM 보내기(128·1K)·받기·취소', () => {
    const act = { invoke: vi.fn(), openHex: vi.fn(), openKey: vi.fn() };
    const items = paneToolItems({ ...base, protocol: 'serial' }, act);
    expect(labels(items)).toEqual(['HEX 로 보기', 'HEX 보내기…', 'XMODEM 보내기…', 'XMODEM-1K 보내기…', 'XMODEM 받기…', 'XMODEM 전송 취소']);
    const pick = (label: string) => items.find((i) => (i as { label?: string }).label === label) as { onSelect: () => void; disabled?: boolean };
    pick('HEX 로 보기').onSelect();
    expect(act.invoke).toHaveBeenCalledWith('session.setHexView', [true]);
    pick('XMODEM-1K 보내기…').onSelect();
    expect(act.invoke).toHaveBeenCalledWith('session.xmodemSend', [{ oneK: true }]);
    pick('XMODEM 받기…').onSelect();
    expect(act.invoke).toHaveBeenCalledWith('session.xmodemReceive');
    pick('HEX 보내기…').onSelect();
    expect(act.openHex).toHaveBeenCalled();
  });

  it('HEX 로 보는 중이면 "글자로 보기" 이고 누르면 끈다', () => {
    const act = { invoke: vi.fn(), openHex: vi.fn(), openKey: vi.fn() };
    const items = paneToolItems({ ...base, protocol: 'serial', hexView: true }, act);
    const item = items.find((i) => (i as { label?: string }).label === '글자로 보기') as { onSelect: () => void };
    item.onSelect();
    expect(act.invoke).toHaveBeenCalledWith('session.setHexView', [false]);
  });

  it('연결되지 않았으면 모두 누를 수 없다', () => {
    const items = paneToolItems({ ...base, protocol: 'serial', state: 'reconnecting' } as SessionInfo, { invoke: vi.fn(), openHex: vi.fn(), openKey: vi.fn() });
    expect(items.filter((i) => 'label' in i).every((i) => (i as { disabled?: boolean }).disabled)).toBe(true);
  });

  it('SSH 는 공개 키 등록만, 로컬·텔넷은 없음', () => {
    const act = { invoke: vi.fn(), openHex: vi.fn(), openKey: vi.fn() };
    const items = paneToolItems({ ...base, protocol: 'ssh' }, act);
    expect(labels(items)).toEqual(['공개 키를 이 서버에 등록…']);
    (items[0] as { onSelect: () => void }).onSelect();
    expect(act.openKey).toHaveBeenCalled();
    expect(paneToolItems({ ...base, protocol: 'local' }, act)).toEqual([]);
    expect(paneToolItems({ ...base, protocol: 'telnet' }, act)).toEqual([]);
    expect(paneToolItems(undefined, act)).toEqual([]);
  });
});
