// OSC 52 쓰기 요청 해석. 근거: spec/behavior/terminal.md §8.
// 반환: 클립보드에 쓸 문자열, 무시하면 null. 읽기 요청(?)은 항상 거절한다.
const MAX_CHARS = 1_000_000;

export function decodeOsc52(payload: string): string | null {
  // payload 형식: "<선택>;<base64>" (선택은 무시)
  const semi = payload.indexOf(';');
  if (semi < 0) return null;
  const data = payload.slice(semi + 1);
  if (data === '' || data === '?') return null;
  let text: string;
  try {
    const bin = atob(data);
    const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
    text = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    return null; // 잘못된 base64 는 조용히 무시
  }
  return text.length <= MAX_CHARS ? text : null;
}
