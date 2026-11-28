// 포트 포워딩 표시용 계산 (spec/ui/panes.md §2.6, D137).
import { describe, expect, it } from 'vitest';
import { failedCount, forwardsText } from './forwards';
import type { ForwardInfo } from '../ipc/types';

const LIST: ForwardInfo[] = [
  { type: 'L', bind: '127.0.0.1:25095', dest: '100.102.205.95:22' },
  { type: 'D', bind: '127.0.0.1:1080' },
  { type: 'R', bind: '0.0.0.0:9000', dest: 'localhost:9000', error: 'address already in use' },
];

describe('forwards', () => {
  it('실패한 수를 센다', () => {
    expect(failedCount(LIST)).toBe(1);
    expect(failedCount(undefined)).toBe(0);
  });

  it('복사용 글자는 한 줄에 하나, 듣는 주소 폭을 맞춘다', () => {
    const text = forwardsText(LIST, 'SOCKS', (e) => `실패: ${e}`);
    expect(text.split('\n')).toEqual([
      'L  127.0.0.1:25095  →  100.102.205.95:22',
      'D  127.0.0.1:1080   →  SOCKS',
      'R  0.0.0.0:9000     ✕  실패: address already in use',
    ]);
  });
});
