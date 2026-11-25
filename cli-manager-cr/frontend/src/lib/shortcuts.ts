// 단축키 표(화면 표시용). 근거: spec/behavior/shortcuts.md §4·§5.
// 시작 화면은 12줄(지금 쓰는 키로), 설정의 "단축키" 구역은 명령 전체(lib/keys.ts)와 아래 고정 키를 보여 준다.
import { t, type MsgId } from '../i18n';
import { SHORTCUT_COMMANDS, comboText } from './keys';

export interface ShortcutItem {
  name: MsgId;
  keys: string;
  /** 키 표기에 번역할 말이 있으면(예: 선택) 그 문구 키. */
  keysId?: MsgId;
}

/** 표시용 [이름, 키] 문자열. */
export function shortcutText(s: ShortcutItem): { name: string; keys: string } {
  return { name: t(s.name), keys: s.keysId ? t(s.keysId) : s.keys };
}

/** 터미널 안에서 쓰는 고정 키. 바꿀 수 없고 "단축키 사용"을 꺼도 동작한다. */
export const FIXED_SHORTCUTS: ShortcutItem[] = [
  { name: 'sc.paste', keys: 'Ctrl+V' },
  { name: 'sc.pasteImage', keys: 'Ctrl+Alt+V' },
  { name: 'sc.copy', keys: '', keysId: 'sc.copyKeys' },
];

/** 시작 화면의 한 줄: 명령 하나, 또는 여러 명령을 묶은 줄(묶음이 모두 기본 키면 묶음 표기를 쓴다). */
type StartRow =
  | { id: string }
  | { name: MsgId; ids: string[]; all: string; allId?: MsgId };

const TAB_IDS = ['tabNext', 'tab1', 'tab2', 'tab3', 'tab4', 'tab5', 'tab6', 'tab7', 'tab8', 'tabLast'];

const START_ROWS: StartRow[] = [
  { id: 'quick' },
  { id: 'newTab' },
  { id: 'newLocal' },
  { id: 'closePane' },
  { id: 'splitRight' },
  { id: 'splitDown' },
  { name: 'sc.movePane', ids: ['focusUp', 'focusDown', 'focusLeft', 'focusRight'], all: '', allId: 'sc.altArrows' },
  { id: 'zoom' },
  { name: 'sc.moveTab', ids: TAB_IDS, all: 'Ctrl+Tab / Ctrl+1~9' },
];

const COMMANDS = new Map(SHORTCUT_COMMANDS.map((c) => [c.id, c]));

/**
 * 시작 화면의 단축키 표. labels 는 명령 id → 지금 키 표기(state/shortcuts 의 useShortcutLabels).
 * 키가 없는 줄은 빼고(전체를 끄면 앱 단축키 줄이 모두 빠진다), 끝에 고정 키 3줄을 붙인다.
 */
export function startShortcutRows(labels: Record<string, string>): { name: string; keys: string }[] {
  const rows: { name: string; keys: string }[] = [];
  for (const row of START_ROWS) {
    if ('id' in row) {
      const cmd = COMMANDS.get(row.id);
      if (cmd && labels[row.id]) rows.push({ name: t(cmd.label, cmd.labelVars), keys: labels[row.id] });
      continue;
    }
    const set = row.ids.map((id) => labels[id]).filter(Boolean);
    if (set.length === 0) continue;
    const untouched = row.ids.every((id) => labels[id] === comboText(COMMANDS.get(id)?.keys ?? ''));
    const keys = untouched ? (row.allId ? t(row.allId) : row.all) : set.length > 1 ? `${set[0]} …` : set[0];
    rows.push({ name: t(row.name), keys });
  }
  return [...rows, ...FIXED_SHORTCUTS.map(shortcutText)];
}
