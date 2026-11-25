// 저장된 단축키 설정(shortcutsEnabled·shortcuts)을 키 처리와 화면 표기에 잇는다. 근거: spec/behavior/shortcuts.md §5 (D126).
import { useApp } from './app';
import { buildKeyMap, comboText, effectiveKeys, shortcutFor, type KeyLike, type ShortcutHit, type ShortcutOverrides } from '../lib/keys';
import type { Settings } from '../ipc/types';

interface Resolved {
  enabled: boolean;
  overrides: ShortcutOverrides | undefined;
  map: Map<string, ShortcutHit>;
  /** 명령 id → 화면 표기('Ctrl+Shift+F'). 꺼져 있거나 키가 없으면 빈 글자. */
  labels: Record<string, string>;
}

let cache: Resolved | null = null;

function resolve(settings: Settings | undefined): Resolved {
  const enabled = settings?.shortcutsEnabled !== false;
  const raw = settings?.shortcuts;
  const overrides = raw && typeof raw === 'object' && !Array.isArray(raw) ? (raw as ShortcutOverrides) : undefined;
  if (cache && cache.enabled === enabled && cache.overrides === overrides) return cache;
  const keys = effectiveKeys(overrides);
  const labels: Record<string, string> = {};
  for (const id of Object.keys(keys)) labels[id] = enabled ? comboText(keys[id]) : '';
  cache = { enabled, overrides, map: buildKeyMap(enabled, overrides), labels };
  return cache;
}

/** 지금 설정에서 이 키 입력이 앱 단축키면 그 동작, 아니면 null(꺼져 있으면 늘 null). */
export function activeShortcut(e: KeyLike): ShortcutHit | null {
  return shortcutFor(e, resolve(useApp.getState().store?.settings).map);
}

/** 명령 id → 화면 표기. 설정이 바뀌면 다시 그린다. 메뉴의 단축키 표기는 `labels.find || undefined` 로 쓴다. */
export function useShortcutLabels(): Record<string, string> {
  return useApp((s) => resolve(s.store?.settings).labels);
}
