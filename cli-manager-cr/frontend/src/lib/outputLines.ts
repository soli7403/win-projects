// 터미널 출력 조각을 "글자만 남긴 줄"로 바꾼다. 출력 트리거와 매크로가 출력에서 글자를 찾는 데 쓴다.
// 근거: spec/behavior/terminal.md §10.1 (D140). 화면(xterm)과는 따로 돈다 — 안 보이는 탭의 출력도 본다.

/** 한 줄로 모으는 최대 길이. 넘으면 앞을 버린다(줄바꿈 없이 이어지는 출력에서 메모리를 지킨다). */
export const LINE_MAX = 2000;

export interface FeedResult {
  /** 이번 조각에서 끝난 줄들(줄바꿈으로 끝난 것). */
  lines: string[];
  /** 아직 끝나지 않은 지금 줄(프롬프트처럼 줄바꿈 없이 멈춘 글). */
  partial: string;
}

type State = 'text' | 'esc' | 'csi' | 'osc' | 'oscEsc' | 'skip1';

/**
 * 출력 조각을 받아 ANSI 제어 시퀀스를 지우고 줄로 나눈다. 조각이 시퀀스 중간에서 잘려도 이어서 본다.
 *   - `\n` 으로 줄이 끝난다. `\r\n` 은 줄바꿈 하나다.
 *   - `\r` 뒤에 다른 글자가 오면 그 줄을 처음부터 다시 쓴 것으로 본다(진행 막대 등).
 *   - 백스페이스는 마지막 글자를 지운다. 그 밖의 제어 문자는 버린다(탭은 빈칸).
 *   - 전체 화면 프로그램(대체 화면: vim·less·htop)에 들어가 있는 동안의 글자는 버린다.
 */
export class OutputLines {
  private state: State = 'text';
  private params = '';
  private cur = '';
  private pendingCr = false;
  /** 대체 화면(전체 화면 프로그램) 안인지. */
  alt = false;

  feed(chunk: string): FeedResult {
    const lines: string[] = [];
    for (let i = 0; i < chunk.length; i++) {
      const ch = chunk[i];
      const code = chunk.charCodeAt(i);
      switch (this.state) {
        case 'esc':
          if (ch === '[') {
            this.state = 'csi';
            this.params = '';
          } else if (ch === ']') this.state = 'osc';
          else if (ch === '(' || ch === ')' || ch === '#' || ch === '%') this.state = 'skip1';
          else this.state = 'text';
          continue;
        case 'skip1':
          this.state = 'text';
          continue;
        case 'csi':
          if (code >= 0x40 && code <= 0x7e) {
            this.onCsi(this.params, ch);
            this.state = 'text';
          } else if (this.params.length < 64) this.params += ch;
          continue;
        case 'osc':
          if (code === 0x07) this.state = 'text';
          else if (code === 0x1b) this.state = 'oscEsc';
          continue;
        case 'oscEsc':
          this.state = ch === '\\' ? 'text' : 'osc';
          continue;
        default:
          break;
      }
      if (code === 0x1b) {
        this.state = 'esc';
        continue;
      }
      if (ch === '\n') {
        this.pendingCr = false;
        if (!this.alt) lines.push(this.cur);
        this.cur = '';
        continue;
      }
      if (ch === '\r') {
        this.pendingCr = true;
        continue;
      }
      if (this.pendingCr) {
        // CR 뒤에 줄바꿈이 아닌 글자: 같은 줄을 처음부터 다시 쓴다.
        this.pendingCr = false;
        this.cur = '';
      }
      if (ch === '\b' || code === 0x7f) {
        this.cur = this.cur.slice(0, -1);
        continue;
      }
      if (code < 0x20) {
        if (ch === '\t') this.push(' ');
        continue;
      }
      this.push(ch);
    }
    return { lines, partial: this.alt ? '' : this.cur };
  }

  private push(ch: string): void {
    if (this.alt) return;
    this.cur += ch;
    if (this.cur.length > LINE_MAX) this.cur = this.cur.slice(this.cur.length - LINE_MAX);
  }

  /** 대체 화면 켜기·끄기(CSI ? 47 · 1047 · 1049 h/l)만 본다. */
  private onCsi(params: string, final: string): void {
    if ((final !== 'h' && final !== 'l') || !params.startsWith('?')) return;
    const modes = params.slice(1).split(';');
    if (modes.some((m) => m === '47' || m === '1047' || m === '1049')) {
      this.alt = final === 'h';
      this.cur = '';
      this.pendingCr = false;
    }
  }
}
