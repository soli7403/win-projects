# 매크로 단축키 — 녹화 시작/정지 · 마지막 매크로 다시 실행 · 중지

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D147). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-25-3-shortcut-settings.md](./2026-11-25-3-shortcut-settings.md)(D126 단축키 설정), [2026-11-27-5-macros.md](./2026-11-27-5-macros.md)(D141 "기본 단축키는 주지 않는다 … 필요한 사람은 나중에 설정에서"), [-8-](./2026-11-27-8-macro-broadcast.md)

---

## 0. 한 줄 요약

설정 > 단축키에 매크로 명령 세 개를 더했다. **기본 키는 없다** — 쓰려는 사람이 키를 준다. "다시 실행"은 이 앱 실행 중 마지막으로 실행한 매크로를 그때의 변수 값으로 지금 입력 대상(일괄 입력이면 탭의 모든 창)에서 다시 돌린다.

## 1. 결정

| 명령 id | 이름 | 동작 | 기본 키 |
|---|---|---|---|
| `macroRecord` | 매크로 녹화 시작/정지 | `toggleRecording()` — 명령 모음의 [매크로 녹화], 빠른 연결 `>` 명령과 같다 | 없음 |
| `macroReplay` | 마지막 매크로 다시 실행 | `replayLastMacro()` | 없음 |
| `macroStop` | 지금 창의 매크로 중지 | `stopMacrosOnTargets()` — 지금 입력 대상에서 도는 매크로를 모두 멈춘다 | 없음 |

| 항목 | 결정 | 까닭 |
|---|---|---|
| 기본 키 | 주지 않는다 | 앱 단축키가 이미 34개다. vi·tmux 와 겹치는 키가 는다(D141 과 같은 까닭) |
| 매크로마다 단축키 | 하지 않는다 | 단축키 표가 저장 데이터(명령 모음)에 따라 늘고, git 동기화로 두 PC 가 같은 키를 다른 매크로에 주면 겹친다. "다시 실행" 하나로 반복 실행을 덮는다 |
| 마지막 실행 | 메모리에만(`lastRun = { macroId, values }`), 파일에 남기지 않는다 | 변수 값에 비밀이 들 수 있다(D138 과 같은 규칙) |
| 다시 실행할 때 변수 | 묻지 않고 그때의 값 | 단축키로 빠르게 반복하려는 것이다. 값을 바꾸려면 목록에서 누른다 |
| 마지막 매크로가 지워졌으면 | `마지막으로 실행한 매크로가 지워졌습니다`, 기억을 지움 | |
| 실행한 적이 없으면 | `아직 실행한 매크로가 없습니다` | |
| 중지할 것이 없으면 | `이 창에서 실행 중인 매크로가 없습니다` | 키를 눌렀는데 아무 일도 없으면 키가 먹지 않은 것처럼 보인다 |

## 2. 구현

### 2.1 명령 표 — `frontend/src/lib/keys.ts`

```ts
export type ShortcutAction =
  | … | 'fontUp' | 'fontDown' | 'fontReset' | 'fullscreen'
  | 'macroRecord' | 'macroReplay' | 'macroStop';

  { id: 'fullscreen', action: 'fullscreen', keys: 'F11', label: 'sc.fullscreen' },
  // 매크로(D147): 기본 키는 없다 — vi·tmux 와 겹치지 않게. 쓰려면 설정 > 단축키에서 키를 준다.
  { id: 'macroRecord', action: 'macroRecord', keys: '', label: 'sc.macroRecord' },
  { id: 'macroReplay', action: 'macroReplay', keys: '', label: 'sc.macroReplay' },
  { id: 'macroStop', action: 'macroStop', keys: '', label: 'sc.macroStop' },
```

기본 키가 빈 글자인 명령은 기존 코드가 이미 맞게 다룬다(고칠 것 없음):

| 함수 | 빈 기본 키에서의 동작 |
|---|---|
| `effectiveKeys` | 저장된 값이 없으면 `c.keys`(빈 글자) → `out[id] = ''`, `used` 에 넣지 않는다 |
| `buildKeyMap` | `keys[c.id]` 가 빈 글자면 표에 넣지 않는다 → 어떤 조합에도 걸리지 않는다 |
| `assignShortcut` | 키를 주면 `next[id] = combo`, 비우면 기본(빈 글자)과 같아 `delete next[id]` |
| 설정 > 단축키 화면 | 키 칸이 빈 상태("없음")로 보이고, 누르면 키를 받는다 |

### 2.2 실행 — `frontend/src/state/actions.ts`

```ts
import { replayLastMacro, stopMacrosOnTargets, toggleRecording } from './macros';

    case 'macroRecord':
      toggleRecording();
      return;
    case 'macroReplay':
      replayLastMacro();
      return;
    case 'macroStop':
      stopMacrosOnTargets();
      return;
```

### 2.3 `frontend/src/state/macros.ts`

```ts
/** 마지막으로 실행한 매크로와 그때의 변수 값(단축키 "마지막 매크로 다시 실행", D147). 앱이 켜져 있는 동안만 기억한다. */
let lastRun: { macroId: string; values: Record<string, string> } | null = null;
// runMacroOnTargets 가 실행할 때 lastRun = { macroId: s.id, values } (-8- §2.2)

/** 마지막으로 실행한 매크로를 같은 변수 값으로 지금 입력 대상에서 다시 실행한다(단축키, D147). */
export function replayLastMacro(): void {
  const push = useApp.getState().pushToast;
  if (!lastRun) {
    push({ kind: 'info', text: t('macro.noLast') });
    return;
  }
  const { macroId, values } = lastRun;
  const s = useApp.getState().store?.snippets.find((x) => x.id === macroId);
  if (!s || !isMacro(s)) {
    lastRun = null;
    push({ kind: 'info', text: t('macro.lastGone') });
    return;
  }
  void runMacroOnTargets(s, values);
}

/** 지금 입력 대상(일괄 입력이면 탭의 모든 창)에서 도는 매크로를 멈춘다(단축키, D147). 멈춘 것이 없으면 알린다. */
export function stopMacrosOnTargets(): void {
  const targets = inputTargets().filter((sid) => jobs.has(sid));
  if (targets.length === 0) {
    useApp.getState().pushToast({ kind: 'info', text: t('macro.nothingRunning') });
    return;
  }
  for (const sid of targets) stopMacro(sid);
}
```

`resetMacrosForTest` 는 `lastRun` 도 지운다.

### 2.4 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `sc.macroRecord` | 매크로 녹화 시작/정지 |
| `sc.macroReplay` | 마지막 매크로 다시 실행 |
| `sc.macroStop` | 지금 창의 매크로 중지 |
| `macro.noLast` | 아직 실행한 매크로가 없습니다 |
| `macro.lastGone` | 마지막으로 실행한 매크로가 지워졌습니다 |
| `macro.nothingRunning` | 이 창에서 실행 중인 매크로가 없습니다 |

## 3. 명세

| 문서 | 바꾼 것 |
|---|---|
| spec/behavior/shortcuts.md §5.1 | 대상 34 → 37개, 표에 35~37(기본 키 "(없음)") |
| spec/behavior/terminal.md §11 | "단축키" 항목 |

## 4. 테스트

| 파일 | 테스트 | 확인 |
|---|---|---|
| `lib/keys.test.ts` | 기본 키는 서로 겹치지 않고 모두 고를 수 있는 조합이다 (매크로 명령 3개는 기본 키가 없다) | 37개, 빈 키 명령 셋의 id, 나머지는 겹치지 않음 |
| 〃 | 기본 키가 없는 명령도 설정에서 키를 주면 동작하고, 빈 키는 어떤 조합에도 걸리지 않는다 | `Ctrl+Alt+KeyM` 을 주면 `macroReplay`, 표에 빈 키 없음 |
| 〃 | 끄면 어떤 키도 앱 단축키가 아니다 | 빈 키 명령은 건너뛰고 검사(기존 테스트 수정) |
| `screens/SettingsShortcuts.test.tsx` | 명령 37개가 … 보인다 | 행 37개, "마지막 매크로 다시 실행" 이 보임(기존 34 → 37 로 수정) |
| `state/followups.test.ts` | 마지막 매크로 다시 실행: 아직 없으면 알리고, 있으면 같은 값으로 다시 | `echo {{msg}}` 를 `hi` 로 실행 → 다시 실행이 `echo hi\r` |
| 〃 | 마지막 매크로가 지워졌으면 알린다 | |
| 〃 | 중지: 지금 대상에서 도는 매크로를 멈추고, 없으면 알린다 | 두 창의 매크로가 함께 멈춤 |

## 5. 검증 결과 (2026-11-27)

tsc, vitest 465개, `wails build` 성공.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 설정 > 단축키 맨 아래 | 매크로 명령 셋이 키 없이 보인다 | |
| 2 | "마지막 매크로 다시 실행"에 Ctrl+Alt+M | 저장된다. 다른 명령과 겹치면 그 명령이 비워진다 | |
| 3 | 매크로를 한 번 실행한 뒤 Ctrl+Alt+M | 같은 매크로가 다시 돈다(변수는 묻지 않는다) | |
| 4 | 앱을 다시 켜고 바로 Ctrl+Alt+M | "아직 실행한 매크로가 없습니다" | |
| 5 | 녹화 명령에 키를 주고 눌러 녹화 → 다시 눌러 정지 | 편집 창이 열린다 | |
| 6 | 긴 매크로 실행 중 중지 키 | 멈추고 "중지했습니다" | |

## 체크리스트

- [x] 명령 표 · [x] 실행 · [x] 문구 · [x] 명세 · [x] 테스트(기존 2개 수정 포함) · [ ] §H
