# 매크로를 일괄 입력 대상 모두에서 실행

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D146). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-5-macros.md](./2026-11-27-5-macros.md)(D141 — "일괄 입력 대상 모두에서 실행"을 보류했던 곳), [-9-](./2026-11-27-9-macro-shortcuts.md)(같은 함수를 단축키가 쓴다)

---

## 0. 한 줄 요약

매크로는 포커스된 창 하나에서만 돌았다. 이제 명령 모음과 같은 **입력 대상**에서 돈다 — 일괄 입력(Ctrl+Shift+B)이 켜진 탭이면 그 탭의 **연결된 모든 창**에서 **창마다 따로**(각자 자기 출력을 기다리며) 실행한다. 여러 서버에 같은 절차(점검·배포)를 돌리는 것이 매크로의 주 용도다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 대상 | `lib/termSend.ts` 의 `inputTargets()` 와 같다 — 탭의 `broadcast` 가 켜져 있으면 그 탭의 세션 모두, 아니면 포커스 창 | 명령(스니펫)·직접 입력과 같은 규칙이라 헷갈리지 않는다 |
| 연결 안 된 창 | 여러 창일 때는 연결됨인 창만 고른다 | 끊긴 창에서는 어차피 시작하지 못한다 |
| 진행 방식 | 창마다 독립 작업(`jobs` Map 의 세션별 Job) | 서버마다 응답 속도가 다르다. 한 창 기준으로 모두에 보내면 앞 단계가 안 끝난 창에 다음 입력이 들어간다 |
| 이미 매크로가 도는 창 | 건너뛰고 `이미 매크로가 도는 창 {n}개는 건너뛰었습니다` | 한 창에 매크로 둘이 섞이면 안 된다 |
| 안내 | 시작할 때 `매크로 {이름} 을(를) 창 {n}개에서 실행합니다` 한 번. 창마다의 **완료** 토스트는 띄우지 않는다. 시간 초과·끊김·중지 토스트의 이름은 `{매크로} · {세션 제목}` | 창이 10개면 완료 토스트가 10개 쌓인다. 오류는 어느 창인지 알아야 한다 |
| 운영 서버 확인 | 대상 중 **하나라도** 운영(`env: prod`)이고 위험 명령이 들었으면 한 번 묻는다. 취소하면 **어디에도** 보내지 않는다 | 일부 창에만 보내면 서버 사이 상태가 어긋난다 |
| 창 하나일 때 | 전과 똑같다(완료 토스트 포함) | 바뀐 것을 느끼지 않게 |

## 2. 구현 — `frontend/src/state/macros.ts`

### 2.1 Job 에 `quiet`

```ts
interface Job {
  // … 전과 같음
  /** 여러 창에서 함께 실행 중이면 참: 창마다 "완료" 안내를 띄우지 않는다(D146). */
  quiet: boolean;
}

// sendStep 의 마지막 단계
if (job.index >= job.steps.length) {
  finish(sessionId, job.quiet ? undefined : { kind: 'ok', text: t('macro.done', { name: nameOf(sessionId) }) });
  return;
}

export function runMacro(sessionId, macroId, name, steps, values = {}, opts: { quiet?: boolean } = {}): boolean {
  // …
  const job: Job = { steps: [...steps], values, index: 0, tail: new OutputTail(), timer: null, quietSince: Date.now(), quiet: !!opts.quiet };
```

### 2.2 `runMacroOnTargets` (새 함수)

```ts
/**
 * 매크로를 지금 입력 대상에서 실행한다(D146): 일괄 입력이 켜진 탭이면 그 탭의 연결된 창 모두, 아니면 지금 창 하나.
 * 운영 서버 창이 하나라도 있고 위험 명령이 들었으면 먼저 묻는다(붙여넣기와 같은 규칙).
 * 여러 창이면 창마다 따로 돌고(창마다 출력을 기다린다), 이미 매크로가 도는 창은 건너뛴다. 시작한 창 수를 돌려준다.
 */
export async function runMacroOnTargets(s: Snippet, values: Record<string, string> = {}): Promise<number> {
  if (!isMacro(s)) return 0;
  const focused = focusedSessionId();
  if (!focused) {
    notifyNoFocus();
    return 0;
  }
  const ws = useWorkspace.getState();
  const all = inputTargets();
  const multi = all.length > 1;
  const targets = multi ? all.filter((sid) => ws.sessions[sid]?.info.state === 'connected') : [focused];
  const st = useApp.getState().store?.settings;
  const text = s.steps.map((x) => fillVars(x.send, values)).join('\n');
  const prod = targets.map((sid) => ws.sessions[sid]?.info).find((info) => info?.env === 'prod');
  if (st?.guardDangerousOnProd && prod && matchesDangerous(text, st.dangerousPatterns ?? DANGEROUS_PATTERNS)) {
    const ok = await useUi.getState().confirm({
      title: t('run.dangerTitle'),
      body: t('run.dangerBody', { name: relabel(prod.title) }),
      detail: text.slice(0, 2000),
      okText: t('run.ok'),
      danger: true,
    });
    if (!ok) return 0;
  }
  const name = s.name || t('snip.noName');
  lastRun = { macroId: s.id, values };          // 단축키 "다시 실행"용 (-9-)
  if (!multi) {
    const ok = runMacro(focused, s.id, name, s.steps, values);
    if (ok) focusTerminal();
    return ok ? 1 : 0;
  }
  let started = 0;
  let busy = 0;
  for (const sid of targets) {
    if (jobs.has(sid)) {
      busy++;
      continue;
    }
    // 여러 창에서 돌 때는 안내에 창 이름을 붙여 어느 창의 일인지 보이게 한다.
    const label = `${name} · ${relabel(ws.sessions[sid]?.info.title) || t('tab.session')}`;
    if (runMacro(sid, s.id, label, s.steps, values, { quiet: true })) started++;
  }
  const push = useApp.getState().pushToast;
  if (started > 0) push({ kind: 'info', text: t('macro.broadcastStarted', { name, count: started }) });
  if (busy > 0) push({ kind: 'info', text: t('macro.broadcastBusy', { count: busy }) });
  if (started === 0 && busy === 0) push({ kind: 'info', text: t('macro.needConnected') });
  if (started > 0) focusTerminal();
  return started;
}
```

### 2.3 명령 모음 패널 — `frontend/src/screens/SnippetsPanel.tsx`

운영 서버 확인과 실행을 패널에서 상태 쪽으로 옮겼다(단축키도 같은 길을 쓰게).

```diff
-  /** 값을 채워 실제로 보낸다. 매크로는 지금 창 하나에서만 돈다(일괄 입력 대상이 아니다). */
+  /**
+   * 값을 채워 실제로 보낸다. 명령·매크로 모두 지금 입력 대상에 간다: 일괄 입력이 켜진 탭이면 그 탭의 모든 창(D146).
+   * 매크로의 운영 서버 확인은 runMacroOnTargets 가 한다.
+   */
   const execute = async (s: Snippet, values: Record<string, string>) => {
     if (!isMacro(s)) {
       if (!sendSnippet(fillVars(s.body, values), s.sendEnter)) notifyNoFocus();
       return;
     }
-    const sid = focusedSessionId();
-    … (운영 서버 확인 20줄)
-    if (runMacro(sid, s.id, s.name || t('snip.noName'), s.steps, values)) focusTerminal();
+    await runMacroOnTargets(s, values);
   };
```

### 2.4 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `macro.broadcastStarted` | 매크로 {name} 을(를) 창 {count}개에서 실행합니다 |
| `macro.broadcastBusy` | 이미 매크로가 도는 창 {count}개는 건너뛰었습니다 |

## 3. 명세

spec/behavior/terminal.md §11 "실행" 줄을 바꾸고(창 하나/여러 창의 안내), 운영 서버 확인 줄을 "대상 중 하나라도"로 고쳤다.

## 4. 테스트 — `frontend/src/state/followups.test.ts` "매크로를 일괄 입력 대상 모두에서 (D146)"

| 테스트 | 확인 |
|---|---|
| 일괄 입력이 켜진 탭이면 연결된 창 모두에서 따로 돈다. 창마다 완료 안내는 띄우지 않는다 | 창 3개 중 끊긴 1개 빼고 2개에 `uptime\r`, 시작 토스트, 완료 토스트 없음 |
| 일괄 입력이 꺼져 있으면 지금 창 하나에서만 | 포커스 창에만 |
| 이미 매크로가 도는 창은 건너뛰고 알린다 | 건너뛴 수 토스트 |
| 대상 중 운영 서버가 있고 위험 명령이 들었으면 먼저 묻고, 취소하면 어디에도 보내지 않는다 | 확인 창 → 취소 → 보낸 것 없음 |

기존 `automation.test.ts`(매크로 실행·녹화 14개)도 그대로 통과 — 창 하나일 때 동작이 같다.

## 5. 검증 결과 (2026-11-27)

tsc, vitest 465개 통과, `wails build` 성공. 앱 화면 확인은 §H.

## 6. 영향 범위와 위험

- 일괄 입력을 켠 채 매크로를 누르던 사람은 이제 모든 창에서 돈다(전에는 한 창). 명령 모음의 명령은 원래 그랬으므로 규칙이 맞춰진 것이다. 도움말·설명서에 적었다.
- 쇠사슬(보이는 모든 탭의 창에 입력, `linkInput`)은 대상 계산(`inputTargets`)에 들어 있지 않다 — 명령 모음과 같다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 한 탭에 서버 둘(Ctrl+Shift+D), 일괄 입력 켬, 매크로 `hostname` → `uptime` 실행 | 두 창 모두 두 명령이 차례로, 토스트 "창 2개에서 실행" 한 번 | |
| 2 | 1 에서 한 창의 서버를 느리게(`sleep 3` 단계 추가) | 각 창이 자기 출력을 기다려 진행한다 | |
| 3 | 한 창에서 다른 매크로를 돌리는 중 1 | 그 창은 건너뛰고 안내 | |
| 4 | 운영 서버 창을 섞고 `rm -rf /tmp/x` 단계 | 확인 창, 취소하면 아무 창에도 안 감 | |
| 5 | 일괄 입력 끄고 실행 | 포커스 창 하나, 완료 토스트 | |

## 체크리스트

- [x] runMacroOnTargets · [x] 패널 연결 · [x] 문구 · [x] 명세 · [x] 테스트 · [ ] §H
