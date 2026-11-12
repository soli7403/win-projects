// 위험 명령 가드(줄 추적 + 패턴 검사). 근거: spec/behavior/terminal.md §4(붙여넣기 검사), §6.

/** 위험 패턴 중 하나와 일치하는지(대소문자 무시). 잘못된 정규식은 건너뛴다. */
export function matchesDangerous(text: string, patterns: string[]): boolean {
  for (const p of patterns) {
    try {
      if (new RegExp(p, 'i').test(text)) return true;
    } catch {
      /* 잘못된 패턴은 건너뜀 */
    }
  }
  return false;
}

/** 붙여넣기 줄 수: 끝의 줄바꿈 하나를 뺀 뒤 줄바꿈으로 나눈다. */
export function pasteLineCount(text: string): number {
  const t = text.endsWith('\n') ? text.slice(0, -1) : text;
  return t.length === 0 ? 0 : t.split('\n').length;
}

/**
 * 입력 줄 추적기. 일반 화면에서만 쓴다(대체 화면은 추적하지 않음).
 * unknown=true 이면 Enter 검사를 하지 않는다.
 */
export class LineTracker {
  private line = '';
  private unknown = false;

  feed(data: string): void {
    for (const ch of data) {
      const code = ch.codePointAt(0)!;
      if (ch === '\r') {
        this.reset();
      } else if (ch === '\x7f' || ch === '\b') {
        this.line = [...this.line].slice(0, -1).join('');
      } else if (ch === '\x03' || ch === '\x15') {
        // Ctrl+C, Ctrl+U: 줄을 비우고 알 수 없음 표시도 해제
        this.line = '';
        this.unknown = false;
      } else if (code >= 0x20 && code !== 0x7f) {
        this.line += ch;
      } else {
        // 그 외 제어 입력(방향키 ESC 시퀀스, 탭 등)은 알 수 없음
        this.unknown = true;
      }
    }
  }

  /** Enter 를 누르기 직전 검사할 줄. 알 수 없거나 비었으면 null. */
  current(): string | null {
    if (this.unknown) return null;
    return this.line.trim().length ? this.line : null;
  }

  reset(): void {
    this.line = '';
    this.unknown = false;
  }
}
