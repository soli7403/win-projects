// 첫 실행 가져오기 제안 판정. 근거: spec/ui/shell.md §6-5, 수용 시나리오 X7.
import { describe, expect, it } from 'vitest';
import { shouldSuggestImport } from './importSuggest';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';

const store = (over: Partial<Store>): Store => ({
  version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS }, ...over,
});

describe('첫 실행 가져오기 제안 (shell.md §6-5)', () => {
  it('보관함이 열리고 서버 0개이고 기록이 없으면 제안', () => {
    expect(shouldSuggestImport(store({}), true)).toBe(true);
  });
  it('보관함이 잠겨 있거나 서버가 있으면 제안하지 않음', () => {
    expect(shouldSuggestImport(store({}), false)).toBe(false);
    expect(shouldSuggestImport(null, true)).toBe(false);
    expect(shouldSuggestImport(store({ hosts: [{ id: 'h' } as Store['hosts'][number]] }), true)).toBe(false);
  });
  it('이미 제안한 기록이 있으면 다시 묻지 않음', () => {
    expect(shouldSuggestImport(store({ appUi: { importSuggested: true } }), true)).toBe(false);
  });
});
