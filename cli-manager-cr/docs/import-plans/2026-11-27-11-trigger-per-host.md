# 서버별 출력 트리거

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D149). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-4-output-triggers.md](./2026-11-27-4-output-triggers.md)(D140), [-7-](./2026-11-27-7-trigger-windows-notify.md)(같은 실행부)

---

## 0. 한 줄 요약

출력 트리거 규칙은 설정 하나에만 있어 모든 연결에 걸렸다. 서버 편집 > 자동화에 **"이 서버의 출력 트리거"** 를 두어, 그 서버에 연결한 창에서만 쓰는 규칙을 넣는다. 설정의 규칙과 **함께** 동작한다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 저장 위치 | 서버 항목의 `triggerRules`(모양은 설정의 규칙과 같다) | 서버를 복제·삭제·동기화·백업할 때 규칙이 따라간다. 서버를 지우면 남는 것이 없다 |
| 합치는 방법 | 설정의 규칙 **다음에** 서버 규칙 | 둘 다 쓴다. 순서는 안내 토스트의 순서에만 영향 |
| 저장되는 때 | 서버의 [저장] | 서버 편집 화면의 다른 값과 같다(설정 화면처럼 즉시 저장하지 않는다) |
| 빈 규칙 | 찾을 글자가 빈 규칙은 저장하지 않고, 하나도 없으면 필드를 쓰지 않는다 | store.json 을 작게 |
| 로컬 터미널 | 자동화 탭을 보이고, 트리거만(로그인 스크립트는 없다) | 로컬 빌드·테스트의 끝을 알고 싶을 수 있다 |
| 저장하지 않은 즉석 연결 | 설정의 규칙만 | 서버 항목이 없다 |
| 반복 제한·끄기 | 규칙 id 마다(설정 규칙과 서버 규칙이 섞여도 각자) | id 가 UUID 라 겹치지 않는다 |
| 대안 | 규칙마다 "적용 서버" 목록 — 기각(설정 화면에서 서버를 고르는 UI 가 무겁고 지운 서버 id 를 치워야 한다). 폴더별 규칙 — 보류 | |

## 2. 구현

### 2.1 실행부 — `frontend/src/state/triggers.ts`

```ts
/** 서버 id → 그 서버 규칙을 만든 원본 배열과 결과(배열이 바뀌었을 때만 다시 만든다). */
const hostCompiled = new Map<string, { from: readonly TriggerRule[] | undefined; rules: CompiledTrigger[]; merged: CompiledTrigger[]; base: CompiledTrigger[] }>();

/** 이 세션에 쓸 규칙: 설정의 규칙 + 그 세션 서버의 규칙(D149). 저장된 서버가 아니면 설정의 규칙만. */
function currentRules(sessionId: string): CompiledTrigger[] {
  const base = globalRules();
  const hostId = useWorkspace.getState().sessions[sessionId]?.info.hostId;
  if (!hostId) return base;
  const own = useApp.getState().store?.hosts.find((h) => h.id === hostId)?.triggerRules;
  let entry = hostCompiled.get(hostId);
  if (!entry || entry.from !== own) {
    entry = { from: own, rules: compileTriggers(Array.isArray(own) ? own : []), merged: [], base: [] };
    hostCompiled.set(hostId, entry);
  }
  if (entry.rules.length === 0) return base;
  if (entry.base !== base) {
    entry.base = base;
    entry.merged = [...base, ...entry.rules];
  }
  return entry.merged;
}
```

- 출력 묶음마다 부르지만 배열(저장소의 그 서버 `triggerRules`, 설정의 `triggerRules`)이 바뀌었을 때만 다시 컴파일·합친다. 규칙이 하나도 없으면 전처럼 곧바로 돌아온다.
- `feedTriggers`·idle 검사가 `currentRules(sessionId)` 를 쓴다.

### 2.2 규칙 목록 편집 화면 재사용 — `frontend/src/screens/SettingsRules.tsx`

설정의 출력 트리거 목록(행 편집·추가·삭제)을 `TriggerRuleList` 로 뽑았다.

```tsx
/**
 * 트리거 규칙 목록 편집(설정의 규칙과 서버 편집의 서버별 규칙이 같이 쓴다, D149).
 * onRows(next, commit): commit 이 참이면 바로 저장할 만한 변경(체크·추가·삭제), 거짓이면 글자 입력 중.
 * onCommit 은 글자 칸을 벗어날 때 부른다.
 */
export function TriggerRuleList({ rows, onRows, onCommit }: {
  rows: TriggerRule[];
  onRows: (next: TriggerRule[], commit: boolean) => void;
  onCommit?: () => void;
}) { … 기존 행 편집 그대로 … }
```

설정 쪽은 `onRows` 에서 commit 이면 `saveSettings`, `onCommit` 은 글자 칸을 벗어날 때 저장. 목록 아래 도움말 `trig.hostHint`.

### 2.3 서버 편집 — `frontend/src/screens/HostEditor.tsx`

```tsx
  // 이 서버의 출력 트리거 규칙(D149). 서버를 저장할 때 함께 저장한다.
  const [hostTriggers, setHostTriggers] = useState(original?.triggerRules ?? []);
  …
  { id: 'auto' as Tab, label: t('host.tabAuto') },          // 모든 프로토콜에 보인다
  …
  {curTab === 'auto' && (
    <>
      {!isLocal && <LoginScriptTable … />}
      <Field label={t('trig.hostTitle')} help={t('trig.hostHelp')}>
        <TriggerRuleList rows={hostTriggers} onRows={(next) => setHostTriggers(next)} />
      </Field>
    </>
  )}
```

`frontend/src/lib/hostSave.ts`:

```ts
/** 서버별 트리거 규칙 저장값: 찾을 글자가 빈 규칙은 빼고, 하나도 없으면 undefined(필드를 남기지 않는다, D149). */
export function cleanHostTriggers(rules: Host['triggerRules']): Host['triggerRules'] {
  const out = (rules ?? []).filter((r) => r.pattern !== '');
  return out.length ? out : undefined;
}
    triggerRules: cleanHostTriggers(i.triggerRules),
```

`frontend/src/ipc/types.ts` `Host.triggerRules?: TriggerRule[]`. 백엔드는 서버 항목을 그대로 저장하므로 고칠 것이 없다(세션 열기 쪽 `hostRecord` 는 이 필드를 읽지 않는다).

### 2.4 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `trig.hostTitle` | 이 서버의 출력 트리거 |
| `trig.hostHelp` | 이 서버에 연결한 창에서만 씁니다. 설정 > 출력 규칙의 규칙(모든 연결)과 함께 동작합니다. 서버를 저장할 때 함께 저장됩니다. |
| `trig.hostHint` | 특정 서버에서만 쓸 규칙은 서버 편집 > 자동화 탭의 "이 서버의 출력 트리거"에 넣습니다. 여기 규칙은 모든 연결에 씁니다. |

## 3. 명세

| 문서 | 바꾼 것 |
|---|---|
| spec/behavior/terminal.md §10 | "서버별 규칙" |
| spec/data/store.md §3 | `triggerRules`(서버) |
| spec/ui/host-editor.md | 탭 규칙(자동화 탭은 모든 프로토콜), §1.4a 아래 항목 |
| spec/ui/settings.md §2b | 목록 아래 도움말 |

## 4. 테스트

| 파일 | 테스트 | 확인 |
|---|---|---|
| `state/followups.test.ts` | 그 서버의 세션에서만 서버 규칙이 동작하고, 설정의 규칙은 모든 세션에서 동작한다 | h1 세션에만 `ONLYH1`, 두 세션 모두 `GLOBAL` |
| 〃 | 서버 규칙을 바꾸면 다음 출력부터 새 규칙을 쓴다 | `OLD` → `NEW` 로 바꾼 뒤 `OLD` 는 더 울리지 않음 |
| `screens/Automation.test.tsx` | 로컬 터미널의 자동화 탭에는 서버별 출력 트리거만 있고 로그인 스크립트는 없다 | 기존 "로컬에는 자동화 탭이 없다"를 바꿨다 |
| 기존 `Automation.test.tsx` 설정 화면 시험 | 그대로 통과 | `TriggerRuleList` 로 뽑아도 같은 화면 |

## 5. 검증 결과 (2026-11-27)

tsc, vitest 465개, `wails build` 성공.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 서버 A 편집 > 자동화 > 이 서버의 출력 트리거 + 규칙 `MYAPP READY`(알림) → 저장 | 저장된다 | |
| 2 | 서버 A 와 B 에 연결해 둘 다 `echo MYAPP READY` | A 에서만 알림 | |
| 3 | 설정 > 출력 규칙에도 `READY` 규칙 | A·B 모두 그 알림 | |
| 4 | 로컬 터미널 서버 편집 | 자동화 탭에 트리거만 있다 | |
| 5 | 서버 A 를 복제 | 복제본에도 규칙이 있다 | |
| 6 | 찾을 글자를 비운 규칙을 두고 저장 → 다시 열기 | 빈 규칙은 사라져 있다 | |

## 체크리스트

- [x] 실행부 · [x] 목록 편집 재사용 · [x] 서버 편집·저장 · [x] 문구 · [x] 명세 · [x] 테스트(기존 1개 수정) · [ ] §H
