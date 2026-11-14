// 서버 종류별 아이콘과 색(사이드바·시작 화면·빠른 접속에서 같이 쓴다). 근거: spec/ui/sidebar.md §5, D71·D72.
// 원격 서버(SSH·텔넷)는 서버 아이콘, 시리얼은 칩, 로컬은 화면. ">_" 터미널 아이콘은 "로컬 터미널 열기" 버튼에만 쓴다.
import { Monitor, Server, Usb } from 'lucide-react';
import type { Protocol } from '../ipc/types';

export const PROTOCOL_ICON: Record<Protocol, typeof Server> = { ssh: Server, telnet: Server, serial: Usb, local: Monitor };

/** 종류별 기본 아이콘 색(테마 토큰). 서버에 색을 지정했으면 그 색이 우선한다(protocolIconColor). */
export const PROTOCOL_ICON_COLOR: Record<Protocol, string> = {
  ssh: 'var(--icon-server)',
  telnet: 'var(--icon-server)',
  serial: 'var(--icon-serial)',
  local: 'var(--icon-local)',
};

/** 서버 아이콘 색: 서버 자신의 색 → 종류별 기본 색. 환경·폴더 색은 행의 색 막대가 보여 주므로 아이콘에는 쓰지 않는다. */
export function protocolIconColor(h: { protocol: Protocol; color?: string }): string {
  return h.color || PROTOCOL_ICON_COLOR[h.protocol] || 'var(--icon-server)';
}

/** 폴더 아이콘 기본 색(폴더 색·환경 색이 없을 때). */
export const FOLDER_ICON_COLOR = 'var(--icon-folder)';
