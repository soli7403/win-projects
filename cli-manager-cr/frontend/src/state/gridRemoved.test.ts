// 바둑판(그리드) 배열 기능을 없앤 것을 지킨다 (D116, spec/ui/panes.md §6).
// 이 기능은 한 탭의 창 수 한도와 화면 나누기 2개를 무시해 그 뒤 동작을 어긋나게 했다. 다시 들어오지 않게 막는다.
import { describe, expect, it } from 'vitest';
import * as layout from '../lib/layout';
import * as workspace from './workspace';
import { LOCALES } from '../i18n';

const SOURCES = import.meta.glob('../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

describe('바둑판 배열 제거 (D116)', () => {
  it('작업 공간에 모든 탭을 한 탭으로 모으는 동작이 없다', () => {
    expect('gridTabs' in workspace.useWorkspace.getState()).toBe(false);
    expect('addGridTab' in workspace).toBe(false);
    expect('gridLayout' in layout).toBe(false);
  });

  it('메뉴·명령 문구가 어느 언어에도 없다', () => {
    for (const [lang, dict] of Object.entries(LOCALES)) {
      for (const key of ['tab.grid', 'quick.grid', 'sb.openFolderGrid']) {
        expect(dict && key in dict, `${lang}: ${key}`).toBe(false);
      }
    }
  });

  it('화면 코드(탭 메뉴·폴더 메뉴·빠른 접속)가 그 동작을 부르지 않는다', () => {
    const offenders = Object.entries(SOURCES)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([, code]) => /gridTabs|openFolderGrid|addGridTab|gridLayout|'c-grid'/.test(code))
      .map(([path]) => path);
    expect(offenders).toEqual([]);
  });

  it('열 위치 선택 대화상자도 없다 (D117): 한도를 보지 않고 아무 창이나 네 방향으로 나누던 화면', () => {
    const offenders = Object.entries(SOURCES)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([path, code]) => /LocationPicker/.test(path) || /LocationPicker|openLocation|locationPicker|sb\.chooseLocation/.test(code))
      .map(([path]) => path);
    expect(offenders).toEqual([]);
    for (const [lang, dict] of Object.entries(LOCALES)) {
      for (const key of ['sb.chooseLocation', 'loc.title', 'loc.here', 'loc.splitOpen']) {
        expect(dict && key in dict, `${lang}: ${key}`).toBe(false);
      }
    }
  });

  it('화면 나누기(탭 그룹)는 왼쪽·오른쪽 두 개뿐이다', () => {
    expect(workspace.useWorkspace.getState().activeByGroup).toHaveLength(2);
  });
});
