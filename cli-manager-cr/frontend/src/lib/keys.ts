// 앱 단축키 매핑(순수 함수). 근거: spec/behavior/shortcuts.md §1~§2.
// 문자 키는 e.code(물리 키)로 판단하므로 한글 입력기가 켜져 있어도 동작한다.

export type ShortcutAction =
  | 'quickToggle' | 'quickOpen' | 'newLocal' | 'closePane' | 'splitRight' | 'splitDown' | 'zoom'
  | 'find' | 'sftp' | 'broadcast' | 'sidebar' | 'tabLeft' | 'tabRight' | 'tabNext' | 'tabPrev'
  | 'tabIndex' | 'focusMove' | 'fontUp' | 'fontDown' | 'fontReset' | 'fullscreen';

export interface KeyLike {
  code: string;
  key: string;
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
}

/** 단축키 동작을 돌려준다. 해당 없으면 null. tabIndex 는 0부터의 번호(Ctrl+1 → 0, Ctrl+9 → 'last'). */
export function shortcutFor(e: KeyLike): { action: ShortcutAction; arg?: number | 'last' | 'up' | 'down' | 'left' | 'right' } | null {
  if (e.code === 'F11') return { action: 'fullscreen' };
  const ctrl = e.ctrlKey && !e.altKey;
  if (e.altKey && !e.ctrlKey && !e.shiftKey) {
    const dirs: Record<string, 'up' | 'down' | 'left' | 'right'> = { ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right' };
    if (dirs[e.code]) return { action: 'focusMove', arg: dirs[e.code] };
    return null;
  }
  if (e.ctrlKey && e.altKey && !e.shiftKey && e.code === 'KeyT') return { action: 'newLocal' };
  if (!ctrl) return null;
  if (e.shiftKey) {
    switch (e.code) {
      case 'KeyK': return null;
      case 'KeyT': return { action: 'quickToggle' };
      case 'KeyN': return { action: 'quickToggle' };
      case 'KeyP': return { action: 'quickOpen' };
      case 'KeyW': return { action: 'closePane' };
      case 'KeyD': return { action: 'splitRight' };
      case 'KeyE': return { action: 'splitDown' };
      case 'Enter': case 'NumpadEnter': return { action: 'zoom' };
      case 'KeyF': return { action: 'find' };
      case 'KeyS': return { action: 'sftp' };
      case 'KeyB': return { action: 'broadcast' };
      case 'KeyL': return { action: 'sidebar' };
      case 'ArrowLeft': return { action: 'tabLeft' };
      case 'ArrowRight': return { action: 'tabRight' };
      default: return null;
    }
  }
  switch (e.code) {
    case 'KeyK': return { action: 'quickToggle' };
    case 'Tab': return { action: 'tabNext' };
    case 'Equal': case 'NumpadAdd': return { action: 'fontUp' };
    case 'Minus': case 'NumpadSubtract': return { action: 'fontDown' };
    case 'Digit0': case 'Numpad0': return { action: 'fontReset' };
    default: break;
  }
  const digit = /^(?:Digit|Numpad)([1-9])$/.exec(e.code);
  if (digit) {
    const n = Number(digit[1]);
    return { action: 'tabIndex', arg: n === 9 ? 'last' : n - 1 };
  }
  return null;
}
