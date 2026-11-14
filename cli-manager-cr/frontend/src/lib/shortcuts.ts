// 단축키 표(화면 표시용). 근거: spec/behavior/shortcuts.md §4.
// 시작 화면은 처음 12개, 설정 화면은 전체를 보여 준다. name·keysId 는 문구 키(D75), 그릴 때 shortcutText 로 바꾼다.
import { t, type MsgId } from '../i18n';

export interface ShortcutItem {
  name: MsgId;
  keys: string;
  /** 키 표기에 번역할 말이 있으면(예: 방향키) 그 문구 키. */
  keysId?: MsgId;
}

/** 표시용 [이름, 키] 문자열. */
export function shortcutText(s: ShortcutItem): { name: string; keys: string } {
  return { name: t(s.name), keys: s.keysId ? t(s.keysId) : s.keys };
}

export const SHORTCUT_LABELS: ShortcutItem[] = [
  { name: 'sc.quick', keys: 'Ctrl+K' },
  { name: 'sc.newTab', keys: 'Ctrl+Shift+T' },
  { name: 'sc.newLocal', keys: 'Ctrl+Alt+T' },
  { name: 'sc.closeTab', keys: 'Ctrl+Shift+W' },
  { name: 'sc.splitRight', keys: 'Ctrl+Shift+D' },
  { name: 'sc.splitDown', keys: 'Ctrl+Shift+E' },
  { name: 'sc.movePane', keys: '', keysId: 'sc.altArrows' },
  { name: 'sc.zoomPane', keys: 'Ctrl+Shift+Enter' },
  { name: 'sc.moveTab', keys: 'Ctrl+Tab / Ctrl+1~9' },
  { name: 'sc.paste', keys: 'Ctrl+V' },
  { name: 'sc.pasteImage', keys: 'Ctrl+Alt+V' },
  { name: 'sc.copy', keys: '', keysId: 'sc.copyKeys' },
  { name: 'sc.find', keys: 'Ctrl+Shift+F' },
  { name: 'sc.sftp', keys: 'Ctrl+Shift+S' },
  { name: 'sc.broadcast', keys: 'Ctrl+Shift+B' },
  { name: 'sc.sidebar', keys: 'Ctrl+Shift+L' },
  { name: 'sc.fontSize', keys: 'Ctrl+= / Ctrl+- / Ctrl+0' },
];

/** 시작 화면에 보이는 처음 12개. */
export const START_SHORTCUTS: ShortcutItem[] = SHORTCUT_LABELS.slice(0, 12);
