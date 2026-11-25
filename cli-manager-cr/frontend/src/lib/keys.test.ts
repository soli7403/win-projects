// 단축키 규칙(D126): 기본 키는 전과 같고, 키를 바꾸면 그 키로만 동작하며, 겹치면 먼저 쓰던 명령이 비워지고, 전체를 끄면 아무것도 걸리지 않는다.
// 근거: spec/behavior/shortcuts.md §2·§5.
import { describe, expect, it } from 'vitest';
import {
  SHORTCUT_COMMANDS, assignShortcut, buildKeyMap, comboOf, comboProblem, comboText, effectiveKeys, shortcutFor, type KeyLike,
} from './keys';

function key(init: Partial<KeyLike>): KeyLike {
  return { code: '', key: '', ctrlKey: false, shiftKey: false, altKey: false, ...init };
}
const ctrl = (code: string) => key({ code, ctrlKey: true });
const ctrlShift = (code: string) => key({ code, ctrlKey: true, shiftKey: true });

describe('기본 단축키 (shortcuts.md §2)', () => {
  it('표의 키가 전과 같은 동작에 걸린다', () => {
    const cases: [KeyLike, string, unknown?][] = [
      [ctrl('KeyK'), 'quickToggle'],
      [ctrlShift('KeyT'), 'quickToggle'],
      [ctrlShift('KeyN'), 'quickToggle'],
      [ctrlShift('KeyP'), 'quickOpen'],
      [key({ code: 'KeyT', ctrlKey: true, altKey: true }), 'newLocal'],
      [ctrlShift('KeyW'), 'closePane'],
      [ctrlShift('KeyD'), 'splitRight'],
      [ctrlShift('KeyE'), 'splitDown'],
      [ctrlShift('Enter'), 'zoom'],
      [ctrlShift('NumpadEnter'), 'zoom'],
      [ctrlShift('KeyF'), 'find'],
      [ctrlShift('KeyS'), 'sftp'],
      [ctrlShift('KeyB'), 'broadcast'],
      [ctrlShift('KeyL'), 'sidebar'],
      [ctrlShift('ArrowLeft'), 'tabLeft'],
      [ctrlShift('ArrowRight'), 'tabRight'],
      [ctrl('Tab'), 'tabNext'],
      [ctrlShift('Tab'), 'tabPrev'],
      [ctrl('Digit1'), 'tabIndex', 0],
      [ctrl('Digit8'), 'tabIndex', 7],
      [ctrl('Numpad3'), 'tabIndex', 2],
      [ctrl('Digit9'), 'tabIndex', 'last'],
      [key({ code: 'ArrowUp', altKey: true }), 'focusMove', 'up'],
      [key({ code: 'ArrowRight', altKey: true }), 'focusMove', 'right'],
      [ctrl('Equal'), 'fontUp'],
      [ctrl('NumpadAdd'), 'fontUp'],
      [ctrl('Minus'), 'fontDown'],
      [ctrl('NumpadSubtract'), 'fontDown'],
      [ctrl('Digit0'), 'fontReset'],
      [key({ code: 'F11' }), 'fullscreen'],
    ];
    for (const [e, action, arg] of cases) {
      const hit = shortcutFor(e);
      expect(hit?.action, comboOf(e)).toBe(action);
      if (arg !== undefined) expect(hit?.arg, comboOf(e)).toBe(arg);
    }
  });

  it('표에 없는 조합은 걸리지 않는다(터미널이 받는다)', () => {
    for (const e of [ctrl('KeyA'), ctrl('KeyC'), ctrl('KeyV'), ctrlShift('KeyK'), ctrlShift('KeyC'), ctrlShift('Digit1'), key({ code: 'KeyK' }), key({ code: 'ArrowUp', altKey: true, shiftKey: true })]) {
      expect(shortcutFor(e), comboOf(e)).toBeNull();
    }
  });

  it('조합 키만 눌렀거나 Win 키가 끼면 조합이 아니다', () => {
    expect(comboOf(key({ code: 'ControlLeft', ctrlKey: true }))).toBe('');
    expect(comboOf(key({ code: 'ShiftRight', shiftKey: true, ctrlKey: true }))).toBe('');
    expect(comboOf(key({ code: 'KeyK', ctrlKey: true, metaKey: true }))).toBe('');
    expect(comboOf(key({ code: 'KeyK', ctrlKey: true, altKey: true, shiftKey: true }))).toBe('Ctrl+Alt+Shift+KeyK');
  });

  it('기본 키는 서로 겹치지 않고 모두 고를 수 있는 조합이다', () => {
    const keys = SHORTCUT_COMMANDS.map((c) => c.keys);
    expect(new Set(keys).size).toBe(keys.length);
    expect(new Set(SHORTCUT_COMMANDS.map((c) => c.id)).size).toBe(SHORTCUT_COMMANDS.length);
    for (const c of SHORTCUT_COMMANDS) expect(comboProblem(c.keys), c.id).toBeNull();
    expect(SHORTCUT_COMMANDS).toHaveLength(34);
  });
});

describe('화면 표기', () => {
  it('조합 글자를 읽기 쉬운 표기로', () => {
    expect(comboText('Ctrl+Shift+KeyT')).toBe('Ctrl+Shift+T');
    expect(comboText('Ctrl+Digit9')).toBe('Ctrl+9');
    expect(comboText('Alt+ArrowUp')).toBe('Alt+↑');
    expect(comboText('Ctrl+Equal')).toBe('Ctrl+=');
    expect(comboText('Ctrl+Shift+Enter')).toBe('Ctrl+Shift+Enter');
    expect(comboText('F11')).toBe('F11');
    expect(comboText('')).toBe('');
  });
});

describe('고를 수 없는 조합 (shortcuts.md §5)', () => {
  it('Ctrl·Alt 없이 누른 보통 키는 안 되고 F 키는 된다', () => {
    expect(comboProblem('KeyA')).toBe('modifier');
    expect(comboProblem('Shift+KeyA')).toBe('modifier');
    expect(comboProblem('Shift+Enter')).toBe('modifier');
    expect(comboProblem('F5')).toBeNull();
    expect(comboProblem('Shift+F12')).toBeNull();
    expect(comboProblem('Alt+KeyX')).toBeNull();
    expect(comboProblem('Ctrl+KeyB')).toBeNull();
  });

  it('터미널의 복사·붙여넣기 키는 안 된다', () => {
    for (const c of ['Ctrl+KeyV', 'Ctrl+Shift+KeyV', 'Ctrl+Shift+KeyC', 'Ctrl+Alt+KeyV', 'Shift+Insert']) expect(comboProblem(c), c).toBe('reserved');
  });
});

describe('키 바꾸기 (shortcuts.md §5)', () => {
  it('바꾸면 새 키로만 동작하고 옛 키는 풀린다', () => {
    const { overrides, displaced } = assignShortcut({}, 'find', 'Ctrl+Alt+KeyF');
    expect(overrides).toEqual({ find: 'Ctrl+Alt+KeyF' });
    expect(displaced).toEqual([]);
    const map = buildKeyMap(true, overrides);
    expect(shortcutFor(key({ code: 'KeyF', ctrlKey: true, altKey: true }), map)?.action).toBe('find');
    expect(shortcutFor(ctrlShift('KeyF'), map)).toBeNull();
    expect(shortcutFor(ctrl('KeyK'), map)?.action).toBe('quickToggle');
  });

  it('다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장한다', () => {
    const { overrides, displaced } = assignShortcut({}, 'find', 'Ctrl+KeyK');
    expect(overrides).toEqual({ find: 'Ctrl+KeyK', quick: '' });
    expect(displaced).toEqual(['quick']);
    const keys = effectiveKeys(overrides);
    expect(keys.find).toBe('Ctrl+KeyK');
    expect(keys.quick).toBe('');
    const map = buildKeyMap(true, overrides);
    expect(shortcutFor(ctrl('KeyK'), map)?.id).toBe('find');
  });

  it('이미 바꿔 둔 키끼리 겹쳐도 먼저 쓰던 쪽이 비워진다', () => {
    const a = assignShortcut({}, 'find', 'Ctrl+Alt+KeyX').overrides;
    const b = assignShortcut(a, 'sftp', 'Ctrl+Alt+KeyX');
    expect(b.overrides).toEqual({ find: '', sftp: 'Ctrl+Alt+KeyX' });
    expect(b.displaced).toEqual(['find']);
  });

  it('지우면 빈 값, 기본값으로 되돌리면 설정에서 빠진다', () => {
    const cleared = assignShortcut({}, 'quick', '').overrides;
    expect(cleared).toEqual({ quick: '' });
    expect(shortcutFor(ctrl('KeyK'), buildKeyMap(true, cleared))).toBeNull();
    expect(assignShortcut(cleared, 'quick', 'Ctrl+KeyK').overrides).toEqual({});
  });

  it('기본값으로 되돌릴 때 그 키를 다른 명령이 쓰고 있으면 그 명령이 비워진다', () => {
    const taken = assignShortcut({}, 'find', 'Ctrl+KeyK').overrides; // { find: Ctrl+K, quick: '' }
    const back = assignShortcut(taken, 'quick', 'Ctrl+KeyK');
    expect(back.overrides).toEqual({ find: '' });
    expect(back.displaced).toEqual(['find']);
  });

  it('모르는 명령·글자가 아닌 값은 버리고, 손으로 겹치게 고친 파일은 표의 앞 명령이 이긴다', () => {
    expect(assignShortcut({ nope: 'Ctrl+KeyQ', find: 3 as unknown as string }, 'sftp', 'Ctrl+Alt+KeyS').overrides).toEqual({ sftp: 'Ctrl+Alt+KeyS' });
    const keys = effectiveKeys({ find: 'Ctrl+KeyK' }); // quick 의 기본 키와 겹친다
    expect(keys.quick).toBe('Ctrl+KeyK');
    expect(keys.find).toBe('');
  });
});

describe('전체 끄기 (shortcuts.md §5)', () => {
  it('끄면 어떤 키도 앱 단축키가 아니다', () => {
    const off = buildKeyMap(false, { find: 'Ctrl+Alt+KeyF' });
    expect(off.size).toBe(0);
    for (const c of SHORTCUT_COMMANDS) {
      const parts = c.keys.split('+');
      const code = parts.pop()!;
      const e = key({ code, ctrlKey: parts.includes('Ctrl'), altKey: parts.includes('Alt'), shiftKey: parts.includes('Shift') });
      expect(shortcutFor(e), c.id).not.toBeNull();
      expect(shortcutFor(e, off), c.id).toBeNull();
    }
  });
});
