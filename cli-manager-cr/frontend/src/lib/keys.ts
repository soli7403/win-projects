// 앱 단축키(순수 함수): 명령 표, 누른 키 → 조합 글자, 조합 → 명령 찾기, 키 바꾸기 규칙.
// 근거: spec/behavior/shortcuts.md §1~§2·§5 (D126).
// 문자 키는 e.code(물리 키)로 판단하므로 한글 입력기가 켜져 있어도 동작한다.
import type { MsgId, Vars } from '../i18n';

export type ShortcutAction =
  | 'quickToggle' | 'quickOpen' | 'newLocal' | 'closePane' | 'splitRight' | 'splitDown' | 'zoom'
  | 'find' | 'sftp' | 'broadcast' | 'sidebar' | 'tabLeft' | 'tabRight' | 'tabNext' | 'tabPrev'
  | 'tabIndex' | 'focusMove' | 'fontUp' | 'fontDown' | 'fontReset' | 'fullscreen'
  | 'macroRecord' | 'macroReplay' | 'macroStop';

export type ShortcutArg = number | 'last' | 'up' | 'down' | 'left' | 'right';

export interface KeyLike {
  code: string;
  key: string;
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  metaKey?: boolean;
}

/** 키를 바꿀 수 있는 명령 하나. keys 는 기본 조합(예: 'Ctrl+Shift+KeyT'). */
export interface ShortcutCommand {
  id: string;
  action: ShortcutAction;
  arg?: ShortcutArg;
  keys: string;
  /** 이름의 문구 키(D75)와 자리표시자 값. */
  label: MsgId;
  labelVars?: Vars;
}

/** 설정 `shortcuts`: 기본값에서 바꾼 명령만 담는다. 값이 빈 글자면 "키 없음". */
export type ShortcutOverrides = Record<string, string>;

/** 설정 화면에 보이는 순서. 같은 조합이 두 명령에 걸리면 앞의 것이 이긴다. */
export const SHORTCUT_COMMANDS: ShortcutCommand[] = [
  { id: 'quick', action: 'quickToggle', keys: 'Ctrl+KeyK', label: 'sc.quick' },
  { id: 'newTab', action: 'quickToggle', keys: 'Ctrl+Shift+KeyT', label: 'sc.newTab' },
  { id: 'quickAlt', action: 'quickToggle', keys: 'Ctrl+Shift+KeyN', label: 'sc.quickAlt' },
  { id: 'quickOpen', action: 'quickOpen', keys: 'Ctrl+Shift+KeyP', label: 'sc.quickOpen' },
  { id: 'newLocal', action: 'newLocal', keys: 'Ctrl+Alt+KeyT', label: 'sc.newLocal' },
  { id: 'closePane', action: 'closePane', keys: 'Ctrl+Shift+KeyW', label: 'sc.closeTab' },
  { id: 'splitRight', action: 'splitRight', keys: 'Ctrl+Shift+KeyD', label: 'sc.splitRight' },
  { id: 'splitDown', action: 'splitDown', keys: 'Ctrl+Shift+KeyE', label: 'sc.splitDown' },
  { id: 'zoom', action: 'zoom', keys: 'Ctrl+Shift+Enter', label: 'sc.zoomPane' },
  { id: 'focusUp', action: 'focusMove', arg: 'up', keys: 'Alt+ArrowUp', label: 'sc.focusUp' },
  { id: 'focusDown', action: 'focusMove', arg: 'down', keys: 'Alt+ArrowDown', label: 'sc.focusDown' },
  { id: 'focusLeft', action: 'focusMove', arg: 'left', keys: 'Alt+ArrowLeft', label: 'sc.focusLeft' },
  { id: 'focusRight', action: 'focusMove', arg: 'right', keys: 'Alt+ArrowRight', label: 'sc.focusRight' },
  { id: 'find', action: 'find', keys: 'Ctrl+Shift+KeyF', label: 'sc.find' },
  { id: 'sftp', action: 'sftp', keys: 'Ctrl+Shift+KeyS', label: 'sc.sftp' },
  { id: 'broadcast', action: 'broadcast', keys: 'Ctrl+Shift+KeyB', label: 'sc.broadcast' },
  { id: 'sidebar', action: 'sidebar', keys: 'Ctrl+Shift+KeyL', label: 'sc.sidebar' },
  { id: 'tabNext', action: 'tabNext', keys: 'Ctrl+Tab', label: 'sc.tabNext' },
  { id: 'tabPrev', action: 'tabPrev', keys: 'Ctrl+Shift+Tab', label: 'sc.tabPrev' },
  { id: 'tabLeft', action: 'tabLeft', keys: 'Ctrl+Shift+ArrowLeft', label: 'sc.tabLeft' },
  { id: 'tabRight', action: 'tabRight', keys: 'Ctrl+Shift+ArrowRight', label: 'sc.tabRight' },
  ...[1, 2, 3, 4, 5, 6, 7, 8].map((n): ShortcutCommand => (
    { id: `tab${n}`, action: 'tabIndex', arg: n - 1, keys: `Ctrl+Digit${n}`, label: 'sc.tabN', labelVars: { n } }
  )),
  { id: 'tabLast', action: 'tabIndex', arg: 'last', keys: 'Ctrl+Digit9', label: 'sc.tabLast' },
  { id: 'fontUp', action: 'fontUp', keys: 'Ctrl+Equal', label: 'sc.fontUp' },
  { id: 'fontDown', action: 'fontDown', keys: 'Ctrl+Minus', label: 'sc.fontDown' },
  { id: 'fontReset', action: 'fontReset', keys: 'Ctrl+Digit0', label: 'sc.fontReset' },
  { id: 'fullscreen', action: 'fullscreen', keys: 'F11', label: 'sc.fullscreen' },
  // 매크로(D147): 기본 키는 없다 — vi·tmux 와 겹치지 않게. 쓰려면 설정 > 단축키에서 키를 준다.
  { id: 'macroRecord', action: 'macroRecord', keys: '', label: 'sc.macroRecord' },
  { id: 'macroReplay', action: 'macroReplay', keys: '', label: 'sc.macroReplay' },
  { id: 'macroStop', action: 'macroStop', keys: '', label: 'sc.macroStop' },
];

/** 터미널 안에서 쓰는 고정 키(복사·붙여넣기). 앱 단축키로 고를 수 없다 (shortcuts.md §5). */
export const RESERVED_COMBOS: readonly string[] = ['Ctrl+KeyV', 'Ctrl+Shift+KeyV', 'Ctrl+Shift+KeyC', 'Ctrl+Alt+KeyV', 'Shift+Insert'];

const MODIFIER_CODES = /^(Control|Shift|Alt|Meta|OS)(Left|Right)?$/;

/** 숫자 패드의 같은 뜻 키는 본 자판 키로 본다(Ctrl+숫자패드 + 도 글자 키우기). */
function normalizeCode(code: string): string {
  if (code === 'NumpadEnter') return 'Enter';
  if (code === 'NumpadAdd') return 'Equal';
  if (code === 'NumpadSubtract') return 'Minus';
  const digit = /^Numpad([0-9])$/.exec(code);
  return digit ? `Digit${digit[1]}` : code;
}

/** 누른 키를 조합 글자로 바꾼다(예: 'Ctrl+Shift+KeyT'). 조합 키만 눌렀거나 Win 키가 끼면 빈 글자. */
export function comboOf(e: KeyLike): string {
  if (!e.code || MODIFIER_CODES.test(e.code) || e.metaKey) return '';
  return `${e.ctrlKey ? 'Ctrl+' : ''}${e.altKey ? 'Alt+' : ''}${e.shiftKey ? 'Shift+' : ''}${normalizeCode(e.code)}`;
}

const CODE_TEXT: Record<string, string> = {
  ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→',
  Equal: '=', Minus: '-', Comma: ',', Period: '.', Slash: '/', Backslash: '\\', Semicolon: ';', Quote: "'",
  BracketLeft: '[', BracketRight: ']', Backquote: '`', Escape: 'Esc', Delete: 'Del', Insert: 'Ins',
  PageUp: 'PgUp', PageDown: 'PgDn', NumpadMultiply: 'Num *', NumpadDivide: 'Num /', NumpadDecimal: 'Num .',
};

/** 조합 글자를 화면 표기로 바꾼다('Ctrl+Shift+KeyT' → 'Ctrl+Shift+T'). 빈 글자는 그대로 빈 글자. */
export function comboText(combo: string): string {
  if (!combo) return '';
  const parts = combo.split('+');
  const code = parts.pop() ?? '';
  const text = CODE_TEXT[code] ?? code.replace(/^Key(?=[A-Z]$)/, '').replace(/^Digit(?=[0-9]$)/, '');
  return [...parts, text].join('+');
}

/**
 * 고를 수 없는 조합이면 까닭을 돌려준다.
 * modifier: Ctrl·Alt 없이 누른 보통 키(F1~F24 는 그냥도 된다) — 글자 입력과 겹친다.
 * reserved: 터미널의 복사·붙여넣기 키.
 */
export function comboProblem(combo: string): 'modifier' | 'reserved' | null {
  if (RESERVED_COMBOS.includes(combo)) return 'reserved';
  const parts = combo.split('+');
  const code = parts[parts.length - 1];
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(code)) return null;
  return parts.includes('Ctrl') || parts.includes('Alt') ? null : 'modifier';
}

/** 명령마다 지금 쓰는 조합(바꾼 것이 있으면 그것, 없으면 기본). 같은 조합이 겹치면 표의 앞 명령만 갖고 뒤는 빈 글자. */
export function effectiveKeys(overrides?: ShortcutOverrides | null): Record<string, string> {
  const out: Record<string, string> = {};
  const used = new Set<string>();
  for (const c of SHORTCUT_COMMANDS) {
    const saved = overrides?.[c.id];
    const combo = typeof saved === 'string' ? saved : c.keys;
    out[c.id] = combo && !used.has(combo) ? combo : '';
    if (combo) used.add(combo);
  }
  return out;
}

export type ShortcutHit = { action: ShortcutAction; arg?: ShortcutArg; id: string };

/** 조합 → 명령 표. enabled 가 false 면 비어 있다(전체 끄기). */
export function buildKeyMap(enabled: boolean, overrides?: ShortcutOverrides | null): Map<string, ShortcutHit> {
  const map = new Map<string, ShortcutHit>();
  if (!enabled) return map;
  const keys = effectiveKeys(overrides);
  for (const c of SHORTCUT_COMMANDS) {
    if (keys[c.id]) map.set(keys[c.id], { action: c.action, arg: c.arg, id: c.id });
  }
  return map;
}

const DEFAULT_MAP = buildKeyMap(true);

/** 단축키 동작을 돌려준다. 해당 없으면 null. map 을 주지 않으면 기본 키. tabIndex 의 arg 는 0부터의 번호 또는 'last'. */
export function shortcutFor(e: KeyLike, map: Map<string, ShortcutHit> = DEFAULT_MAP): ShortcutHit | null {
  const combo = comboOf(e);
  return (combo && map.get(combo)) || null;
}

/**
 * 명령 id 의 키를 combo 로 바꾼 새 설정 값을 돌려준다(combo 가 빈 글자면 키 없음).
 * 그 조합을 쓰던 다른 명령은 빈 값으로 저장한다 — displaced 가 그 명령 id 들.
 */
export function assignShortcut(overrides: ShortcutOverrides | null | undefined, id: string, combo: string): { overrides: ShortcutOverrides; displaced: string[] } {
  const next: ShortcutOverrides = {};
  const known = new Map(SHORTCUT_COMMANDS.map((c) => [c.id, c]));
  for (const [k, v] of Object.entries(overrides ?? {})) {
    if (known.has(k) && typeof v === 'string') next[k] = v;
  }
  const displaced: string[] = [];
  if (!known.has(id)) return { overrides: next, displaced };
  if (combo) {
    const cur = effectiveKeys(next);
    for (const c of SHORTCUT_COMMANDS) {
      if (c.id !== id && cur[c.id] === combo) {
        next[c.id] = '';
        displaced.push(c.id);
      }
    }
    // 겹쳐서 가려져 있던 뒤쪽 명령(파일을 손으로 고친 경우)도 같은 조합이면 비운다.
    for (const c of SHORTCUT_COMMANDS) {
      if (c.id !== id && !displaced.includes(c.id) && (next[c.id] ?? c.keys) === combo) next[c.id] = '';
    }
  }
  if (combo === known.get(id)!.keys) delete next[id];
  else next[id] = combo;
  return { overrides: next, displaced };
}

/** 설정 화면이 키를 받는 중이면 true. 그동안 앱 단축키와 모달의 Esc 닫기는 쉰다. */
export const keyCapture = { active: false };
