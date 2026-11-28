// 포트 포워딩 표시용 계산. 근거: spec/ui/panes.md §2.6, spec/protocols.md §2.5 (D136·D137·D151).
import type { ForwardInfo } from '../ipc/types';

/** 실패한 포워딩 수. */
export function failedCount(list: readonly ForwardInfo[] | undefined): number {
  return (list ?? []).filter((f) => !!f.error).length;
}

/**
 * 포워딩 목록을 복사용 글자로 만든다: 한 줄에 하나, 듣는 주소의 폭을 맞춘다.
 * socks 는 D 의 대상 자리에 적을 글자, failed 는 실패 줄의 까닭 앞에 붙일 글자를 만든다.
 */
export function forwardsText(list: readonly ForwardInfo[], socks: string, failed: (error: string) => string): string {
  const width = Math.max(0, ...list.map((f) => f.bind.length));
  return list
    .map((f) => {
      const head = `${f.type}  ${f.bind.padEnd(width)}`;
      if (f.error) return `${head}  ✕  ${failed(f.error)}`;
      return `${head}  →  ${f.type === 'D' ? socks : f.dest ?? ''}`;
    })
    .join('\n');
}

/** 목록 전체의 전송량 합(D151). 값이 없는 항목(옛 백엔드·실패)은 0 으로 본다. */
export function trafficTotal(list: readonly ForwardInfo[] | undefined): { up: number; down: number } {
  return (list ?? []).reduce((sum, f) => ({ up: sum.up + (f.up ?? 0), down: sum.down + (f.down ?? 0) }), { up: 0, down: 0 });
}
