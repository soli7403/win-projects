# 2026-11-18 작업 6건을 공식 소스(1.0.0)에 넣기 — 설계·코드·패치·검증

**작성**: 2026-11-23 · **상태**: 적용 완료(2026-11-23, 자동 검증 통과 — 앱 수동 검증은 §6.3) · **대상**: cli-manager-cr(DariTerm) 1.0.0 · **Known issue**: KI-011(적용하면서 등록, §3.5)
**관련**: [docs/decisions.md](../decisions.md) 번호 안내(2026-11-21), [CHANGELOG.md](../../CHANGELOG.md), [docs/releases/1.0.0.md](../releases/1.0.0.md), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §4·§6

> **이 문서 하나로 적용할 수 있게 썼다.** 소스 폴더는 따로 주고받지 않는다. 6건의 원인·설계(부 A~E), 새 파일의 전체 내용(부록 P.1), 바꿀 파일의 패치(부록 P.2), 문서에 넣을 글(부록 Q)이 모두 이 안에 있다.
> 2026-11-18 에 쓴 계획서 5개는 따로 복사하지 않고 부 A~E 로 이 문서에 넣었다.

---

## 0. 한 줄 요약

git 의 공식 소스(1.0.0, 2026-11-21)에는 **2026-11-18 에 만든 기능 6건**이 들어 있지 않다. 그날 작업은 다른 작업 폴더에서 했고 저장소에 올라가지 않았다(`docs/decisions.md` 에 "D96~D100 은 이 저장소에 없다"고 적혀 있다). 이 문서는 그 6건을 공식 소스 위에 넣는 방법이다.

- 2026-11-19·20 작업(D101~D105)과 1.0.0 릴리스(D106)는 공식 소스에 이미 있다. **건드리지 않는다.**
- 패치는 2026-11-23 에 받은 공식 소스와 작업 폴더를 비교해 만들었다. 공식 소스 사본에 적용해 빌드와 테스트가 통과하는 것을 확인했다(§6.1).
- 아래 경로는 모두 `cli-manager-cr/` 기준이다.

### 문서 구성

| 부분 | 내용 |
|---|---|
| §1~§9 | 무엇을 넣는지, 적용 방법과 순서, 검증, 정할 것 |
| 부 A | 터미널 찾기 4건(돋보기 버튼, 강조 색 D96, 스크롤 막대 표시, 남는 표시) — 2026-11-18 계획서 원문 |
| 부 B | 잠금 화면 Caps Lock 안내(D97) — 원문 |
| 부 C | 서버 timeout 뒤 자동 재연결(D98) — 원문 |
| 부 D | 출력 기록 설정을 열린 세션에 곧바로 적용(D99) — 원문 |
| 부 E | 로그 내려 쓰기 주기 설정(D100) — 원문 |
| 부록 P | 새 파일 4개의 전체 내용, 바꿀 파일 33개의 패치 |
| 부록 Q | 문서에 넣을 글(decisions 행, KI-011, 시나리오 행) |

---

## 1. 넣을 6건

| # | 결정 | 기능 | 공식 소스의 지금 동작 | 설명 |
|---|---|---|---|---|
| 1 | — | 칸 머리의 돋보기 버튼으로 찾기 열기 | 버튼을 누르면 "준비 중" 안내만 나온다(`notYet`) | 부 A §A-3 |
| 2 | D96 | 터미널 찾기 강조 색·모양, 스크롤 막대 표시, 찾기를 닫으면 표시 지우기 | 현재 항목이 선택 색에 가려 잘 안 보이고, 스크롤 막대 표시가 없다 | 부 A §A-4~§A-6 |
| 3 | D97 | 잠금 화면의 Caps Lock 안내 | 안내 없음 | 부 B |
| 4 | D98 | 서버가 셸을 끝냈을 때 사용자가 끝낸 것인지 가려 자동 재접속 | 종료 코드와 함께 끝나면 항상 닫힘 + Enter 안내(유휴 로그아웃도 다시 붙지 않음) | 부 C |
| 5 | D99 | "터미널 출력 기록"을 바꾸면 열려 있는 세션에 곧바로 적용 | 켠 뒤 새로 여는 세션부터만 기록 | 부 D |
| 6 | D100 | 로그 내려 쓰기 주기 설정 `sessionLogSyncSec`(0~5초, 기본 2) | 설정 없음. 기록 중인 로그가 탐색기에 0바이트로 보일 수 있다 | 부 E |

4·5·6번은 같은 파일(`control.go`, `log.go`, `session.go`, `app.go`)을 함께 고치므로 한 번에 넣는다.

### 부 A~E 를 읽을 때

- 부 A~E 는 2026-11-18 에 작업 폴더에서 쓴 계획서 원문이다. 상태 줄의 "코드 수정 완료"는 작업 폴더 이야기이고, **공식 소스에는 아직 적용되지 않았다.**
- 원문의 "수정 전" 코드는 공식 소스의 지금 코드와 같다. 다만 원문의 줄 번호와 `#L123` 링크는 작업 폴더 기준이라 공식 소스에서는 몇 줄 어긋날 수 있다.
- 부 D 의 코드는 D99 시점 것이고 부 E(D100)가 같은 함수를 다시 고친다. **두 부의 코드가 다르면 부록 P 의 패치가 최종이다.**
- 절 번호에는 부 글자를 붙였다(부 A 의 3절 = §A-3). 부 안에서 파일 이름 없이 쓴 "§7.2" 같은 번호는 그 부의 절이다.
- 원문에 적힌 테스트 실행 결과와 빌드 시각은 2026-11-18 작업 폴더에서의 기록이다.

---

## 2. 코드 적용

### 2.1 새 파일 4개 — 부록 P.1 의 내용으로 만든다

| 파일 | 건 |
|---|---|
| `frontend/src/lib/findHighlight.ts` | 2 |
| `frontend/src/lib/findHighlight.test.ts` | 2 |
| `frontend/src/lib/useCapsLock.ts` | 3 |
| `internal/session/log_sync_windows_test.go` | 6 |

### 2.2 바꿀 파일 33개 — 부록 P.2 의 패치를 적용한다

| 파일 | 건 | 바뀌는 것 |
|---|---|---|
| `frontend/src/screens/Panes.tsx` | 1 | 돋보기 버튼 `onClick` 이 `FIND_EVENT` 를 보낸다. 쓰지 않게 된 `notYet` import 삭제 |
| `frontend/src/screens/TerminalView.tsx` | 2 | `FindBar` 의 현재 항목 강조(`markActive`·`clearActive`), 색 값, `overviewRuler` 옵션, 테마의 `overviewRulerBorder` |
| `frontend/src/screens/Lock.tsx` | 3 | `useCapsLock`, 안내 줄 3곳 |
| `frontend/src/screens/Lock.test.tsx` | 3 | "Caps Lock 안내 (D97)" 테스트 4개 |
| `frontend/src/styles/app.css` | 3 | `.caps-note` 한 줄 |
| `frontend/src/screens/SettingsConnection.tsx` | 6 | "로그 파일 갱신 주기 (초)" 필드 |
| `frontend/src/ipc/types.ts` | 6 | `sessionLogSyncSec` |
| `frontend/src/lib/defaults.ts` | 6 | `sessionLogSyncSec: 2` |
| `frontend/src/ipc/contract/store.get.json` | 6 | `"sessionLogSyncSec": 2` |
| `frontend/src/ipc/contract/store.get.afterNullPatch.json` | 6 | `"sessionLogSyncSec": 2` |
| `internal/store/store.go` | 6 | 기본값 `"sessionLogSyncSec": 2` |
| `internal/session/types.go` | 6 | `Config.LogSyncSec` |
| `internal/session/log.go` | 6 | `ClampLogSyncSec`, `SetSyncSec`, `syncNowLocked`, `flushTimer` 의 내려 쓰기 |
| `internal/session/log_test.go` | 5·6 | `TestSetLogging…` 2개, `TestLogSync…`·`TestClampLogSyncSec`·`TestSessionLogSyncSecApplies` |
| `internal/session/control.go` | 4·5·6 | `Input` 의 `tLastInput` 기록, `SetLogging`, `SetLogSyncSec` |
| `internal/session/manager.go` | 5·6 | `Manager.SetLogging`, `Manager.SetLogSyncSec` |
| `internal/session/session.go` | 4·5·6 | `tConnected`·`tLastInput`·`iLogSyncSec`, `userEndedLocked`, `onRemoteExited` 의 재접속 분기, `start` 가 `SetLogging(true)` 호출 |
| `internal/session/exit_unlock_test.go` | 4 | `TestRemoteExit…` 3개, `backdateForTest` |
| `internal/app/app.go` | 5·6 | `store.patch` 뒤 `SetLogging`·`SetLogSyncSec` 호출, `logSyncSec()` |
| `internal/app/channels_session.go` | 6 | 세션 설정에 `LogSyncSec` 한 줄 |
| `locales/*.json` (10개) | 3·5·6 | `lock.capsLock`, `conn.logSyncSec`, `conn.logSyncSecHelp` 추가, `conn.sessionLogDesc` 문구 변경 |
| `spec/behavior/session.md` | 4·5·6 | §3 안내 줄, §4 사용자 종료와 서버 종료 구분, §6 설정 변경 즉시 적용·내려 쓰기 주기 |
| `spec/data/store.md` | 6 | `sessionLogSyncSec` 행 |
| `spec/ui/settings.md` | 5·6 | 세션 로그 설명 문구, "로그 파일 갱신 주기 (초)" 필드 |

### 2.3 패치를 적용하는 방법

**전제**: 위 파일이 git 에서 받은 1.0.0 그대로여야 한다. 받은 뒤 고친 파일이 있으면 그 파일은 패치 대신 내용을 보고 손으로 넣는다.

1. 부록 P.2 의 파일별 `diff` 블록을 하나씩(또는 모두 이어 붙여) `p.patch` 같은 파일로 저장한다. UTF-8, 줄바꿈 LF, 블록 안 내용 그대로. 블록의 마지막 줄이 빈 줄이나 공백 한 칸짜리 줄이어도 지우지 않는다.
2. `cli-manager-cr/` 에서 실행한다.

```bash
git -c core.autocrlf=false apply --check p.patch   # 먼저 걸리는 곳이 없는지 본다
git -c core.autocrlf=false apply p.patch
```

- 공식 소스의 파일과 이 문서의 패치는 둘 다 줄바꿈이 LF 다.
- `-c core.autocrlf=false` 를 붙이는 이유: Windows 의 git 은 보통 `core.autocrlf=true` 라서, 그냥 적용하면 고친 파일 전체가 CRLF 로 바뀔 수 있다. 그러면 `internal/app` 의 `TestContractChannels` 가 실패한다(고정 파일 `frontend/src/ipc/contract/*.json` 을 바이트로 비교한다). 이 문서를 만들 때 실제로 겪었다.
- 작업 폴더의 파일이 CRLF 로 체크아웃돼 있어 패치가 맞지 않으면 `--ignore-whitespace` 를 더한다.
- `git apply` 를 쓰지 않고 손으로 넣어도 된다. 패치의 `-` 줄을 지우고 `+` 줄을 넣는다. 각 변경의 이유는 부 A~E 에 있다.

---

## 3. 문서 고치기 (손으로)

아래 문서는 공식 소스에 1.0.0·D101~D105 내용이 들어가 있어 패치로 만들지 않았다. 넣을 글의 전체는 부록 Q 에 있다.

### 3.1 `docs/scenarios.md` · `docs/scenarios.en.md` (4·5번)

| 행 | 바꿀 내용 |
|---|---|
| 6-11 출력 기록 | 부록 Q.3 의 행으로 바꾼다("켜면 열려 있는 연결도 그때부터…") |
| 12-1a 서버가 세션을 끝냄 | 지금 문구는 "유휴 자동 로그아웃·`exit` 로 셸이 끝나면 다시 붙지 않고"라 D98 과 반대다. 아래 문구로 바꾼다 |

12-1a 새 문구(한국어, 넷째 칸):

> `exit`·`logout` 처럼 사용자가 끝내면(끝나기 전 10초 안에 입력이 있었거나 연결된 지 10초가 안 됐을 때) 다시 붙지 않고 "서버가 세션을 끝냈습니다." + "Enter 키를 누르면 다시 연결합니다." 입력 없이 있다가 서버가 끝내면(유휴 자동 로그아웃, 관리자의 세션 종료, 서버 재시작) 끊김처럼 자동 재연결.

12-1a 새 문구(영어):

> When you end the shell yourself (`exit`, `logout` — there was input within the last 10 s, or the connection was under 10 s old), DariTerm does not log back in; it shows "The server ended the session." + "Press Enter to reconnect." When the server ends an idle session (idle auto-logout, an admin closing it, a server restart), it reconnects automatically like a dropped connection.

시나리오 문서로 PDF 를 만드는지 `tools/manual/build.mjs` 에서 확인하고, 만든다면 다시 만든다.

### 3.2 `docs/decisions.md`

1. D101 행 바로 위에 부록 Q.1 의 **D96~D100 행 5개**를 넣는다.
2. 7행 번호 안내를 고친다.
   - before: `D96~D100 은 이 저장소에 없다. 2026-11-19·20 계획서(docs/import-plans)가 매긴 번호 D101~D105 를 계획서와 어긋나지 않게 그대로 쓴다.`
   - after: `D96~D100(2026-11-18)은 1.0.0 에 들어가지 않았고 docs/import-plans/2026-11-23-import-1118-features.md 로 뒤에 넣었다. 그래서 날짜는 D101~D106 보다 앞서지만 들어간 버전은 뒤다.`


### 3.3 `CHANGELOG.md` · `docs/store/listing.md`

1.0.0 은 이미 배포본을 만들었으므로 1.0.0 항목은 건드리지 않는다. 맨 위에 다음 버전 항목을 새로 만든다. **버전 번호는 정해야 한다**(§7). 넣을 내용:

- 추가: 잠금 화면에서 Caps Lock 이 켜져 있으면 알려 준다(D97) · 설정 > 연결 "로그 파일 갱신 주기 (초)"(0~5, 기본 2). 기록 중인 로그 파일의 크기가 탐색기에 0바이트로 보이던 것이 이 주기마다 갱신된다(D100)
- 개선: 터미널 찾기에서 현재 항목이 빨강으로 뚜렷하게 보이고 스크롤 막대에 위치가 표시된다(D96) · "터미널 출력 기록"을 켜고 끄면 열려 있는 연결에도 곧바로 적용된다(D99) · 서버의 유휴 자동 로그아웃 뒤 자동으로 다시 연결한다(D98). **달라 보이는 점**: 입력 없이 10초 넘게 있다가 서버가 세션을 끝내면 이제 다시 붙는다. `exit` 로 끝내면 전처럼 닫힌다
- 수정: 칸 머리의 돋보기 버튼이 찾기를 열지 않던 문제

`docs/store/listing.md` 의 "이 버전의 새로운 기능" 문구(영어·한국어)도 같은 버전으로 더한다.

### 3.4 프로젝트 `CLAUDE.md` 의 `docs/import-plans/` 행

"2026-11-17 의 2건도 구현 완료(…)" 뒤에 아래 문장을 넣는다.

> 2026-11-18 의 6건(터미널 찾기 수정 4건 — 버튼·강조 색 D96·스크롤 막대 표시·남는 표시, 잠금 화면 Caps Lock 안내 D97, 서버 timeout 뒤 자동 재연결 D98, 출력 기록 설정을 열린 세션에 곧바로 적용 D99, 로그 내려 쓰기 주기 설정 D100)은 1.0.0 에 빠져 있어 2026-11-23-import-1118-features.md 한 문서로 넣는다(설계·코드·패치 포함).

적용이 끝나면 "넣는다"를 "넣었다"로 고친다.

### 3.5 known-issues

1. `.claude/known-issues.md` 의 `## ✅ ACTIVE` 아래 `(없음)` 을 지우고 부록 Q.2 의 **KI-011** 을 넣는다.
2. 루트 `../CLAUDE.md` 의 "다음 번호: KI-011" 을 "KI-012" 로 고친다.

---

## 4. 건드리지 않는 것

| 항목 | 이유 |
|---|---|
| `wails.json` 버전 1.0.0, `docs/releases/1.0.0.md`, CHANGELOG 의 1.0.0 항목 | 이미 낸 릴리스 기록이다 |
| D101~D105 의 코드·테스트·명세 (`Shell.tsx`, `splitDropHint.*`, `TabBar.tsx`, `hosts.ts`, `host_look.go`, `internal/sftpfs/*`, `spec/ui/shell.md`, `spec/ui/tokens.md`, `spec/protocols.md`, `spec/behavior/file-panel.md`) | 공식 소스에 이미 있다. 6건과 겹치는 파일이 없다 |
| `docs/manual/*` | 사용 설명서에는 6건 설명이 원래 없다. 넣을지는 다음 버전 문서 작업에서 정한다 |
| `go.mod` · `go.sum` | 6건은 새 의존성을 쓰지 않는다. wails 버전은 §7 |

---

## 5. 순서

여러 단계 작업이므로 시작할 때 `.claude/wip/2026-11-23-import-1118-features.md` 를 만든다(공통 규칙 §4).

패치를 한꺼번에 적용해도 되고 아래처럼 나눠도 된다. 나누면 묶음마다 §6.2 를 돌린다.

### 5.1 묶음 A — 프런트 (1·2·3번)

1. 새 파일 `findHighlight.ts`, `findHighlight.test.ts`, `useCapsLock.ts` (부록 P.1)
2. 패치: `Panes.tsx`, `TerminalView.tsx`, `Lock.tsx`, `Lock.test.tsx`, `app.css`
3. 패치: `locales/*.json` 10개 (5·6번 문구도 같이 들어간다. 쓰는 코드는 묶음 B 에서 들어온다)

### 5.2 묶음 B — 세션·로그 (4·5·6번)

1. 새 파일 `internal/session/log_sync_windows_test.go` (부록 P.1)
2. 패치: `internal/session/` 7개, `internal/store/store.go`, `internal/app/app.go`, `internal/app/channels_session.go`
3. 패치: `types.ts`, `defaults.ts`, contract JSON 2개, `SettingsConnection.tsx`
4. 패치: `spec/` 3개

묶음 B 의 Go 파일은 서로 물려 있어 일부만 넣으면 빌드되지 않는다.

### 5.3 묶음 C — 문서

§3.1~§3.5. 끝나면 이 문서의 상태 줄을 "적용 완료"로 고친다.

---

## 6. 검증

### 6.1 이 문서를 만들 때 한 확인 (2026-11-23)

이 문서에서 부록 P 의 새 파일 4개와 패치 33개를 다시 뽑아, 공식 소스(1.0.0) 사본에 `git -c core.autocrlf=false apply` 로 적용하고 확인했다.

| 확인 | 결과 |
|---|---|
| `git apply --check` | 33개 모두 걸리는 곳 없음 |
| 적용한 파일을 작업 폴더의 파일과 바이트 비교 | 37개 중 36개가 같다. `internal/app/channels_session.go` 만 다르고, 다른 점은 `hostRecord` 의 `M_szGroupId` 필드 위치뿐이다(공식 소스 것을 그대로 둔 것) |
| `go build ./...`, `go vet ./...` | `internal/...` 통과. 루트 패키지는 `frontend/dist` 가 없어 걸렸다(프런트를 빌드하지 않은 사본이라 그렇다. 6건과 무관) |
| `go test ./...` (Go 1.27.1, Windows) | `internal/` 의 테스트 있는 패키지 14개 모두 통과. 루트 패키지는 위와 같은 이유로 실행되지 않음 |
| `npm run typecheck` | 통과 |
| `npm test` (vitest, PowerShell) | 파일 40개, 테스트 259개 모두 통과 |

프런트 검사는 작업 폴더의 `node_modules` 를 빌려 돌렸다(두 폴더의 `package.json` 은 같다). **앱을 빌드해 실행해 보지는 않았다.** §3 의 문서 고치기는 적용하지 않은 상태였다.

### 6.2 자동

| 명령 | 기대 |
|---|---|
| `go build ./...` | 통과 |
| `go vet ./...` | 통과 |
| `go test ./internal/session/... ./internal/app/... ./internal/store/...` | 통과. 새 테스트: `TestRemoteExitAfterIdleAutoReconnects`, `TestRemoteExitRightAfterInputStaysClosed`, `TestRemoteExitAfterIdleWithAutoReconnectOff`, `TestSetLoggingOnOpenSession`, `TestSetLoggingIgnoresClosedSession`, `TestLogSyncZeroWritesImmediately`, `TestLogSyncSecSwitch`, `TestClampLogSyncSec`, `TestSessionLogSyncSecApplies`, Windows 에서는 `log_sync_windows_test.go` 의 테스트 |
| `go test ./...` | 통과 (D101~D105 테스트가 깨지지 않았는지) |
| `frontend/` 에서 `npm run typecheck` | 통과. `noUnusedLocals` 가 켜져 있어 `Panes.tsx` 에 `notYet` import 가 남아 있으면 여기서 걸린다 |
| `frontend/` 에서 `npm test` | 통과. 새 테스트: `findHighlight.test.ts`, `Lock.test.tsx` 의 "Caps Lock 안내 (D97)" 4개. 계약 테스트가 `sessionLogSyncSec` 를 본다 |

작업 PC 에서는 vitest 를 Git Bash 에서 돌리면 모든 테스트가 "failed to find the current suite" 로 실패했다. PowerShell 에서 돌린다.

### 6.3 수동 (빌드한 앱)

6번(D100)만 작업 폴더에서 사용자 확인이 끝났다. 나머지는 작업 폴더에서도 수동 검증을 하지 않았으므로 적용 뒤에 처음 확인하는 것이다.

| 건 | 확인할 것 | 절차 |
|---|---|---|
| 1 | 칸 머리 돋보기를 누르면 찾기 막대가 열린다 | 부 A §A-9.2 |
| 2 | 현재 항목은 빨강, 나머지는 옅은 주황. 스크롤 막대에 표시가 나오고 찾기를 닫으면 사라진다 | 부 A §A-9.2 |
| 3 | 잠금 해제·처음 설정·복구 재설정 화면에서 Caps Lock 을 켜면 안내가 나오고 끄면 사라진다 | 부 B §B-7.2 |
| 4 | `TMOUT=20` 을 준 서버에서 입력 없이 기다리면 자동 재연결, `exit` 를 치면 닫힘 + Enter 안내 | 부 C §C-6.2 |
| 5 | 세션을 연 채 "터미널 출력 기록"을 켜면 `logs` 폴더에 새 파일이 생기고, 끄면 기록이 멈춘다 | 부 D §D-7.2 |
| 6 | 설정 > 연결에 "로그 파일 갱신 주기 (초)"가 보이고, 값대로 탐색기의 파일 크기가 갱신된다. 0 이면 즉시 | 부 E §E-9.3 |

---

## 7. 정할 것

| 항목 | 내용 | 권장 |
|---|---|---|
| 다음 버전 번호 | 1.0.0 은 배포본을 이미 만들었다. 6건은 그 뒤 버전에 들어간다 | **1.1.0** — 새 설정 항목과 동작 변경(D98)이 있어 수정 버전(1.0.1)보다 기능 버전이 맞다 |
| wails 버전 | 공식 `go.mod` 는 wails v2.10.2 다. 작업 PC 에서는 v2.10.2 CLI 가 Go 1.27 에서 빌드에 실패해 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` 로 빌드했고, 작업 폴더의 `go.mod` 도 v2.17.1 이었다 | 6건과 따로 다룬다. 공식 소스에서 빌드가 되는지 먼저 확인하고, 안 되면 `go.mod` 를 올리는 결정을 따로 남긴다 |

---

## 8. 영향 범위·회귀 위험

| 위험 | 내용 | 막는 방법 |
|---|---|---|
| 패치가 맞지 않음 | 패치는 2026-11-23 에 받은 1.0.0 소스 기준이다. 그 뒤 해당 파일이 바뀌었으면 걸린다 | `git apply --check` 로 먼저 본다. 걸리는 파일은 부 A~E 를 보고 손으로 넣는다 |
| D98 동작 변경 | 유휴 로그아웃 서버에서는 로그아웃될 때마다 다시 로그인한다 | 설정 "연결이 끊기면 다시 연결"을 끄면 멈춘다. CHANGELOG 에 "달라 보이는 점"으로 적는다(§3.3) |
| 설정 파일 호환 | `sessionLogSyncSec` 가 없는 기존 설정 파일 | 기본값 2 로 읽는다(`store.go` 기본값 표). 데이터 파일 형식은 그대로다 |
| KI-011 | 찾기 막대가 열린 동안 클립보드가 덮일 수 있다. 2번과 함께 들어오는 알려진 문제다 | 고치지 않고 known-issues 에 등록한다(§3.5) |
| 줄바꿈 | `core.autocrlf=true` 인 git 으로 적용하면 파일이 CRLF 로 바뀌어 계약 테스트가 실패한다 | §2.3 의 `-c core.autocrlf=false` |

기능별 영향 범위는 부 A §A-11, 부 B §B-8, 부 C §C-7, 부 D §D-6, 부 E §E-8.

---

## 9. 체크리스트

- [ ] `git -c core.autocrlf=false apply --check` 통과 확인
- [ ] 묶음 A (프런트 1·2·3번) + 자동 검증
- [ ] 묶음 B (세션·로그 4·5·6번) + 자동 검증
- [ ] 묶음 C (문서, KI-011, 다음 번호 KI-012)
- [ ] 다음 버전 번호 결정 → CHANGELOG·스토어 문구
- [ ] 빌드 확인(wails 버전)
- [ ] 수동 검증 6건
- [ ] 이 문서 상태 줄 갱신, wip 를 archive 로 이동

---

## 부 A. 터미널 찾기 수정 4건 (찾기 버튼 · 강조 색·모양 · 스크롤 막대 표시 · 남는 표시) — 원인·수정·검증 — 1·2번

**작성**: 2026-11-18 (같은 날 §4~§7 추가) · **상태**: 코드 수정 완료(2026-11-18, D96), 앱 수동 검증은 남음(§9.2), 남은 문제는 KI-011(§7) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/ui/panes.md](../../spec/ui/panes.md) §2(머리글 버튼)·§2.4(찾기 막대), [spec/behavior/shortcuts.md](../../spec/behavior/shortcuts.md), [spec/acceptance/03-panes-terminal.md](../../spec/acceptance/03-panes-terminal.md) T12, [frontend/src/screens/Panes.tsx](../../frontend/src/screens/Panes.tsx), [frontend/src/screens/TerminalView.tsx](../../frontend/src/screens/TerminalView.tsx), [frontend/src/lib/findHighlight.ts](../../frontend/src/lib/findHighlight.ts), [known-issues KI-011](../../.claude/known-issues.md), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 파일 이름은 처음 다룬 문제(찾기 버튼)에서 왔다. 같은 날 이어진 찾기 관련 수정을 모두 이 문서에 모았다.

---

### A-0. 한눈에 보기

2026-11-18 하루 동안 터미널 찾기에서 네 가지를 고쳤다. 순서대로 발견됐고, 뒤의 두 건은 앞의 수정을 확인하다가 나왔다.

| # | 문제 | 원인 | 수정 | 절 |
|---|---|---|---|---|
| 1 | 칸 머리글의 돋보기 버튼이 "찾기: 다음 단계에서 구현됩니다" 안내만 띄움 | 버튼에 임시 처리(`notYet`)가 남아 있었다. 단축키·우클릭 메뉴는 정상 | 버튼도 같은 찾기 요청을 보낸다 | §3 |
| 2 | 현재 항목보다 나머지 찾은 항목이 더 눈에 띔 | 검색 애드온은 현재 항목을 "선택"으로 표시하는데, 현재 항목 색이 선택 색(회청색) 아래에 깔렸다 | 찾은 항목은 옅게, 현재 항목은 선택 위에 진한 빨강 + 흰 글자 + 테두리 (D96) | §4 |
| 3 | 스크롤 막대에 찾기 표시가 안 나옴 | 표시 색은 넘기고 있었지만 터미널의 표시 옵션(`overviewRuler`)을 켜지 않았다 | 옵션을 켜고 현재 항목 표시를 추가 | §5 |
| 4 | 찾기를 닫아도 스크롤 막대에 주황 표시가 한 자리에 남음 | 2번에서 넣은 현재 항목 강조를 지우는 순서가 틀려 xterm 안에 강조가 남았다(오늘 만든 버그) | 강조를 먼저, 줄 표식을 나중에 지운다 | §6 |

고치지 않고 남긴 것: 출력이 이어지는 세션에서 찾기를 쓸 때의 문제 네 가지(클립보드 덮임, 강조 갱신 지연, 마우스 선택이 바뀜, 강조 1000개 한도) → **KI-011**, §7.

---

### A-1. 발견 경위

| 시각 순 | 사용자 보고 | 결과 |
|---|---|---|
| 1 | "terminal에서 찾기가 구현이 안된 것인가?" | 기능은 있고 버튼 연결만 빠져 있었다 → §3 |
| 2 | 스크린샷과 함께 "현재 선택된 것이 눈에 띄어야 하는데, 찾아진 것이 더 눈에 띈다. 색과 모양을 조절해줘" | §4 |
| 3 | "로그가 계속 출력중에 찾기를 하면 어떻게 되나?", "1000개까지만 찾기 된다는 것인가?", "10000개로 늘리면?" | 코드로 확인해 답하고 KI-011 등록 → §7 |
| 4 | "스크롤 막대에서 안보인다" | §5 |
| 5 | "1개 찾아졌는데 스크롤 막대에는 2개로 보인다", "찾기가 없어도 표시된다" | §6 |

---

### A-2. 찾기의 구조 (수정 전후 공통)

찾기 막대는 칸마다 `TerminalView` 가 가지고 있고, **창 이벤트 `dariterm:find`**(detail = 칸 id)를 받으면 자기 칸일 때 연다. 검색과 강조는 xterm.js 의 검색 애드온(`@xterm/addon-search` 0.16, xterm 6.0)이 한다.

```
[단축키]   Ctrl+Shift+F
  → lib/keys.ts:38            KeyF → { action: 'find' }
  → state/actions.ts:149-151  포커스된 칸 id 로 'dariterm:find' 이벤트 발생

[우클릭]   칸 메뉴 "찾기"
  → screens/Panes.tsx:195     FIND_EVENT 이벤트 발생 (detail = pane.id)

[버튼]     칸 머리글 돋보기
  → screens/Panes.tsx:241     FIND_EVENT 이벤트 발생 (§3 에서 고침)

[수신]     screens/TerminalView.tsx:48       FIND_EVENT = 'dariterm:find'
           screens/TerminalView.tsx:462-468  자기 칸 id 면 setFindOpen(true)
           screens/TerminalView.tsx:531      findOpen 이면 <FindBar> 표시
           screens/TerminalView.tsx:157-254  FindBar (검색 애드온 + 현재 항목 강조)
```

애드온의 동작 중 이 문서에서 중요한 것(애드온 소스 `node_modules/@xterm/addon-search/src/` 로 확인):

- 찾은 항목마다 "강조(decoration)"를 하나씩 만든다. 강조는 버퍼 줄에 붙는 "표식(marker)"에 매달려 있어 줄이 밀려도 따라간다.
- **현재 항목은 터미널의 "선택 영역"으로 표시한다**(`terminal.select`). 따로 현재 항목용 강조도 만들지만 기본 층(bottom)이라 선택 색 아래에 그려진다.
- 강조는 버퍼 맨 위(가장 오래된 줄)부터 **1000개까지만** 만든다. 다음/이전 이동은 개수 제한이 없다.
- 출력이 올 때마다 0.2초 뒤(출력이 이어지면 계속 미룸) 전체를 다시 찾아 강조를 다시 만들고 현재 항목을 다시 선택한다.

---

### A-3. 수정 1 — 돋보기 버튼이 찾기를 열지 않음

#### A-3.1 원인

1. [TerminalView.tsx:47](../../frontend/src/screens/TerminalView.tsx#L47) 의 주석은 "찾기 요청 이벤트. **단축키·칸 머리글·우클릭 메뉴**가 보낸다" 로, 세 경로 모두 같은 이벤트를 보내는 것이 의도다.
2. 단축키([actions.ts:149-151](../../frontend/src/state/actions.ts#L149-L151))와 우클릭 메뉴([Panes.tsx:195](../../frontend/src/screens/Panes.tsx#L195))는 이벤트를 보냈다.
3. 머리글 버튼([Panes.tsx:241](../../frontend/src/screens/Panes.tsx#L241))은 찾기 막대를 만들기 전에 넣어 둔 임시 처리 `notYet(tr('pane.find'))` 가 그대로 남아 있었다. `notYet` 은 [lib/notYet.ts](../../frontend/src/lib/notYet.ts) 의 안내 토스트(`notYet.text` = "{label}: 다음 단계에서 구현됩니다")만 띄운다.
4. 기능 누락이 아니라 **버튼 한 곳의 연결 누락**이다. 명세 [ui/panes.md](../../spec/ui/panes.md) §2 는 머리글에 찾기 버튼(툴팁 "찾기 (Ctrl+Shift+F)")을 두라고 한다.

#### A-3.2 수정 — `frontend/src/screens/Panes.tsx`

```tsx
// before
{hasSession && (
  <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => notYet(tr('pane.find'))}><Search size={12} /></button>
)}

// after
{hasSession && (
  <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id }))}><Search size={12} /></button>
)}
```

안 쓰게 된 `import { notYet } from '../lib/notYet';` 줄을 지웠다(이 파일에서 쓰는 곳이 버튼 하나뿐이었다). `FIND_EVENT` 는 이미 import 하고 있었다(11행).

---

### A-4. 수정 2 — 현재 항목이 눈에 띄지 않음 (D96)

#### A-4.1 증상

찾은 항목들은 진한 주황 바탕인데, 정작 현재 항목은 흐린 회청색이었다. 명세 §2.4 는 "찾은 항목은 주황 계열, 현재 항목은 빨강 계열"이다.

#### A-4.2 원인

수정 전 강조 옵션:

```tsx
const opts = { caseSensitive: false, decorations: { matchOverviewRuler: '#f59e0b', activeMatchColorOverviewRuler: '#ef4444', matchBackground: '#f59e0b55', activeMatchBackground: '#ef444488' } };
```

1. 애드온은 현재 항목을 **선택 영역**으로 만든다. xterm 은 칸의 색을 "아래층 강조 → 선택 → 위층 강조" 순으로 덧칠한다(`node_modules/@xterm/addon-webgl/src/CellColorResolver.ts`). 애드온의 현재 항목 강조는 아래층이라 **선택 색(테마의 `selectionBackground`, DariTerm Dark 는 `#2f4a6b`)에 덮인다.** `activeMatchBackground` 로 준 빨강은 화면에 나오지 않는다.
2. 나머지 항목은 선택이 아니라서 `matchBackground` 주황이 그대로 보인다. 그래서 현재 항목보다 더 튄다.
3. 색 값에 투명도를 붙인 8자리(`#f59e0b55`)를 쓰고 있었는데, 애드온은 `#RRGGBB` 6자리만 받는다고 정해 두었다(typings 주석). 의도한 "옅은 주황"이 되지 않고 진하게 칠해졌다.

#### A-4.3 설계 결정

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 현재 항목 위에 "위층(top)" 강조를 직접 얹는다** | 선택 색 위에 그려지므로 바탕·글자색을 마음대로 정할 수 있다. 애드온은 그대로 쓴다 | **채택** |
| B. 찾기 동안 테마의 선택 색을 빨강으로 바꾼다 | 코드는 짧지만 그동안 마우스로 선택한 것도 빨갛게 되고, 설정 변경 시 테마 덮어쓰기와 충돌한다 | 기각 |
| C. 강조의 DOM 요소에 CSS 로 색을 입힌다 | 요소가 글자 위에 있어 글자를 가리거나 색이 탁해진다. 애드온이 현재 항목 요소에 구분 클래스를 붙이지 않는다 | 기각 |

**정한 모양**

| | 바탕 | 글자 | 테두리 |
|---|---|---|---|
| 찾은 항목 | 터미널 배경에 주황 `#f59e0b` 을 **32% 섞은 색** (테마마다 계산) | 원래 글자색 | 없음 |
| 현재 항목 | 진한 빨강 `#dc2626` | 흰색 `#ffffff` | 테마 글자색 1px |

찾은 항목 색을 테마 배경에 섞어 만드는 이유: 고정된 어두운 주황은 밝은 테마에서 글자가 안 읽히고, 고정된 밝은 주황은 어두운 테마에서 현재 항목보다 튄다. 섞으면 어느 테마에서나 배경에 가깝고 글자가 읽힌다(테스트로 6개 테마 확인, §9.1).

#### A-4.4 수정 — 새 파일 `frontend/src/lib/findHighlight.ts`

색·비율 상수와 순수 함수 두 개를 둔다. 값을 바꾸려면 이 파일 맨 위 상수만 고치면 된다.

```ts
export const FIND_MATCH_COLOR = '#f59e0b';   // 찾은 항목의 기준 색
export const FIND_MATCH_RATIO = 0.32;        // 터미널 배경에 섞는 비율
export const FIND_ACTIVE_BG = '#dc2626';     // 현재 항목 바탕
export const FIND_ACTIVE_FG = '#ffffff';     // 현재 항목 글자
export const FIND_RULER_MATCH = '#f59e0b';   // 스크롤 막대 표시(찾은 항목)
export const FIND_RULER_ACTIVE = '#ef4444';  // 스크롤 막대 표시(현재 항목)
export const FIND_RULER_WIDTH = 14;          // 스크롤 막대 표시 폭(px), §5

/** `#RRGGBB` 두 색을 섞는다(ratio = over 의 비중). 형식이 아니면 over 를 그대로 돌려준다. */
export function mixHex(base: string, over: string, ratio: number): string

/** 선택 범위(시작 칸 포함, 끝 칸 제외, 0부터 세는 버퍼 좌표)를 줄별 범위로 나눈다. */
export function rowRanges(start: CellPos, end: CellPos, cols: number): RowRange[]
```

`rowRanges` 가 필요한 이유: 일치 항목이 줄 끝에서 다음 줄로 이어지면 강조를 줄마다 하나씩 만들어야 한다.

#### A-4.5 수정 — `TerminalView.tsx` 의 `FindBar`

현재 위치 [TerminalView.tsx:157-254](../../frontend/src/screens/TerminalView.tsx#L157-L254). 아래는 수정 후 전체(§6 의 삭제 순서 수정까지 반영).

```tsx
function FindBar({ term, paneId, onClose }: { term: Terminal; paneId: string; onClose: () => void }) {
  const [q, setQ] = useState('');
  const addon = useRef<SearchAddon | null>(null);
  // 현재 항목 위에 얹는 강조(줄마다 하나).
  const active = useRef<IDisposable[]>([]);
  void paneId;
  const clearActive = () => {
    for (const d of active.current) d.dispose();
    active.current = [];
  };
  const markActive = () => {
    clearActive();
    const sel = term.getSelectionPosition();          // 애드온이 방금 선택한 현재 항목
    if (!sel) return;
    const buf = term.buffer.active;
    const border = term.options.theme?.foreground ?? FIND_ACTIVE_FG;
    for (const r of rowRanges(sel.start, sel.end, term.cols)) {
      const marker = term.registerMarker(r.row - buf.baseY - buf.cursorY);
      const deco = term.registerDecoration({ marker, x: r.x, width: r.width, backgroundColor: FIND_ACTIVE_BG, foregroundColor: FIND_ACTIVE_FG, layer: 'top', overviewRulerOptions: { color: FIND_RULER_ACTIVE, position: 'full' } });
      deco?.onRender((el) => {
        el.style.outline = `1px solid ${border}`;
      });
      // 반드시 강조를 먼저, 표식을 나중에 지운다 (§6).
      active.current.push({
        dispose() {
          deco?.dispose();
          marker.dispose();
        },
      });
    }
  };
  useEffect(() => {
    const a = new SearchAddon();
    term.loadAddon(a);
    addon.current = a;
    // 터미널을 눌러 선택이 풀리면 현재 항목 강조도 지운다.
    const sub = term.onSelectionChange(() => {
      if (!term.hasSelection()) clearActive();
    });
    return () => {
      sub.dispose();
      clearActive();
      a.clearDecorations();
    };
  }, [term]);
  const matchBg = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: FIND_RULER_MATCH, activeMatchColorOverviewRuler: FIND_RULER_ACTIVE, matchBackground: matchBg, activeMatchBackground: FIND_ACTIVE_BG } };
  const find = (text: string, dir: 1 | -1) => {
    const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
    if (found) markActive();
    else clearActive();
  };
  const next = (dir: 1 | -1) => {
    if (!q) return;
    find(q, dir);
  };
  const close = () => {
    clearActive();
    addon.current?.clearDecorations();
    onClose();
    term.focus();
  };
  // 아래 JSX(입력창·이전·다음·닫기)는 그대로. 입력이 바뀔 때 find(값, 1), 비면 clearActive + clearDecorations.
}
```

바뀐 점 요약:
- 찾기(글자 입력·다음·이전)가 성공할 때마다 `markActive` 가 선택 위치를 읽어 **위층 강조**를 새로 얹는다. 실패하면 지운다.
- 터미널을 눌러 선택이 풀리면, 그리고 찾기 막대를 닫으면 현재 항목 강조를 지운다.
- 색은 모두 6자리 `#RRGGBB` 다.

import 추가: `import { Terminal, type IDisposable } from '@xterm/xterm';`, `findHighlight` 의 상수·함수.

#### A-4.6 알려진 빈틈 (고치지 않음)

현재 항목 줄이 스크롤백 한도를 넘어 지워지거나 터미널을 눌러 선택을 푼 뒤 출력이 0.2초 멈추면, 애드온이 스스로 가장 최근 일치 항목을 다시 선택한다. 이때는 `markActive` 가 불리지 않아 그 항목이 예전처럼 회청색으로 보인다. 다음/이전을 누르면 바로잡힌다. 고치려면 애드온의 결과 변경 알림(`onDidChangeResults`)마다 `markActive` 를 부르면 된다.

---

### A-5. 수정 3 — 스크롤 막대에 표시가 안 나옴

#### A-5.1 원인

명세 §2.4 는 "스크롤 막대에도 표시"다. 찾기 쪽은 표시 색(`matchOverviewRuler` 등)을 넘기고 있었지만, xterm 은 터미널 옵션 `overviewRuler.width` 가 없으면 표시를 그리지 않는다(`OverviewRulerRenderer` 의 폭이 0). 터미널을 만드는 곳에 이 옵션이 없었다.

#### A-5.2 수정 — `TerminalView.tsx` `createKeptTerm`

```tsx
// after (터미널 옵션에 추가)
    allowProposedApi: true,
    // 찾기 결과를 스크롤 막대 자리에 표시한다(ui/panes.md §2.4). 폭은 xterm 스크롤 막대 기본 폭과 같아 칸 수가 바뀌지 않는다.
    overviewRuler: { width: FIND_RULER_WIDTH },
    theme: {
      overviewRulerBorder: theme.background,
      background: theme.background,
      …
```

- 폭 14px 는 xterm 스크롤 막대의 기본 폭(`DEFAULT_SCROLL_BAR_WIDTH`)과 같다. 맞춤 애드온과 스크롤 막대가 같은 값을 쓰므로 **터미널 칸 수는 바뀌지 않는다.**
- 표시 영역 왼쪽 경계선은 기본이 검정이라 터미널 배경색으로 맞췄다. 설정에서 테마를 바꿀 때 적용하는 줄([TerminalView.tsx:519](../../frontend/src/screens/TerminalView.tsx#L519))에도 `overviewRulerBorder: theme.background` 를 더했다.
- 현재 항목 표시: 애드온은 이미 표시가 있는 줄에는 현재 항목 표시를 따로 찍지 않는다. 그래서 §4.5 의 위층 강조에 `overviewRulerOptions: { color: FIND_RULER_ACTIVE, position: 'full' }` 를 주어 **폭 전체를 가로지르는 빨강 선**으로 나오게 했다. 찾은 항목은 가운데 주황 띠다.

#### A-5.3 표시가 뜻하는 것

표시는 화면이 아니라 **스크롤백을 포함한 버퍼 전체**에서 그 줄이 몇 % 지점인지로 찍힌다. 그래서:

- 화면에 일치 항목이 하나만 보여도, 지나간 줄에 더 있으면 표시는 여러 개다.
- 출력이 이어지면 표시도 움직인다. 스크롤백이 차기 전에는 전체 줄 수가 늘어 표시가 위로 몰리고, 가득 찬 뒤에는 줄이 지워지는 만큼 올라가다가 사라진다.
- 강조가 없는 항목(1000개 한도 밖, 출력 중 아직 갱신 안 된 줄)은 표시도 없다.

---

### A-6. 수정 4 — 찾기를 닫아도 스크롤 막대에 표시가 남음

#### A-6.1 증상

§5 를 넣은 뒤, 일치 항목이 하나인데 스크롤 막대에 주황 표시가 하나 더 보였다. 그 표시는 스크롤해도 움직이지 않았고 찾기 막대를 닫은 뒤에도 남았다.

#### A-6.2 원인 — §4 에서 만든 버그

처음에는 현재 항목 강조를 지울 때 **표식(marker)만** 지웠다("표식을 지우면 강조도 같이 지워진다"고 보고).

```tsx
// 잘못된 코드 (§4 의 첫 구현)
active.current.push(marker);      // clearActive 가 marker.dispose() 만 부른다
```

xterm 안에서 일어나는 일:

1. `Marker.dispose()` 는 **줄 번호를 -1 로 바꾼 뒤** 폐기 알림을 보낸다(`node_modules/@xterm/xterm/src/common/buffer/Marker.ts`).
2. 알림을 받은 강조가 자기를 목록에서 지우려 한다. 목록은 줄 번호로 정렬돼 있고 이진 탐색으로 찾는데(`SortedList.delete`), 줄 번호가 이미 -1 이라 **제자리에서 찾지 못하고 삭제가 실패한다.** 실패하면 "강조가 지워졌다"는 알림도 나가지 않는다.
3. 강조가 줄 번호 -1 인 채 목록 중간에 남는다. 목록의 정렬이 깨져, 그 뒤로는 **애드온의 주황 강조를 지우는 탐색도 실패**할 수 있다. 실패한 강조도 같은 식으로 줄 번호 -1 로 남는다.
4. 남은 강조는 줄에 묶여 있지 않아 스크롤해도 움직이지 않고, 찾기를 닫아도 지워지지 않는다. 스크롤 막대 표시는 남은 강조까지 그린다.

2번(표식을 먼저 지우면 강조가 줄 번호 -1 로 목록에 남는다)은 임시 테스트로 재현해 확인했다: 강조 3개를 만들고 마지막 것의 표식을 먼저 지우면 목록에 `[0, 2, -1]` 이 남고, 나머지를 올바른 순서로 지워도 `[-1]` 이 남았다. 사용자 스크린샷의 그 주황 표시가 정확히 이 남은 강조인지는 화면으로 대조하지 못했다(증상은 일치한다).

#### A-6.3 수정

강조를 먼저, 표식을 나중에 지운다. 애드온 자신이 쓰는 순서와 같다.

```tsx
// after
active.current.push({
  dispose() {
    deco?.dispose();    // 줄 번호가 살아 있을 때 목록에서 빠진다
    marker.dispose();
  },
});
```

**주의(다음에 강조를 직접 다룰 때)**: xterm 6.0 에서 강조가 붙은 표식을 먼저 지우면 강조가 샌다. 항상 강조부터 지운다.

수정 전 빌드로 열어 둔 세션에는 남은 강조가 그대로 있다. 앱을 다시 띄우면 없어진다.

---

### A-7. 고치지 않고 남긴 것 — KI-011

출력이 이어지는 세션에서 찾기를 쓰면 생기는 문제다. 코드로 확인했고 앱에서 재현해 보지는 않았다. [known-issues.md](../../.claude/known-issues.md) 에 KI-011(active)로 등록했다.

| # | 문제 | 원인 | 고친다면 |
|---|---|---|---|
| 1 | "선택 시 자동 복사"가 켜져 있으면(기본값) 찾기에서 글자를 치거나 다음/이전을 누를 때마다 현재 항목이 클립보드에 복사된다. 출력 중인 세션에서는 출력이 0.2초 멈출 때마다 다시 복사된다 | 애드온이 현재 항목을 "선택"으로 만들고, 자동 복사([TerminalView.tsx](../../frontend/src/screens/TerminalView.tsx) 의 `onSelectionChange`)는 누가 선택했는지 가리지 않는다 | 찾기가 만든 선택은 자동 복사에서 뺀다 (권장) |
| 2 | 출력이 0.2초 넘게 멈추지 않으면 새로 들어온 줄의 일치 항목이 강조되지 않는다 | 애드온의 갱신이 출력 뒤 0.2초 조용해야 돈다 | 애드온 밖에서 주기 갱신을 직접 구현 |
| 3 | 찾기 막대가 열린 동안 마우스로 다른 곳을 선택해도 출력이 멈추면 선택이 일치 항목으로 바뀐다 | 갱신 때 애드온이 현재 항목을 다시 선택한다 | 2번과 같이 |
| 4 | 강조(와 스크롤 막대 표시)는 가장 오래된 줄부터 1000개까지만. 흔한 낱말이면 최근 줄은 강조 없이 이동만 된다 | 애드온의 기본 한도 `highlightLimit` | 한도를 올리거나 화면 근처만 칠하도록 직접 구현 |

**한도를 10000 으로 올리면**(검토만 함, 적용하지 않음): 일치 항목이 그만큼 있을 때만 비용이 든다. 강조 하나마다 버퍼에 표식이 붙고, 스크롤백이 가득 찬 뒤에는 줄이 하나 지워질 때마다 모든 표식이 한 번씩 갱신된다. 출력이 멈출 때마다 전부 지우고 다시 만드는 일은 개수의 제곱에 가까운 부분이 있다. 스크롤 막대 표시도 화면이 스크롤될 때마다 모든 강조를 훑는다. 고속 로그 세션에서 찾기를 열어 두면 느려질 수 있다. 시간을 재 보지는 않았다. 올린다면 3000 정도부터 체감을 확인하는 것을 권한다.

---

### A-8. 수정한 파일

| 파일 | 내용 | 절 |
|---|---|---|
| `frontend/src/screens/Panes.tsx` | 돋보기 버튼이 `FIND_EVENT` 를 보냄, `notYet` import 삭제 | §3 |
| `frontend/src/screens/TerminalView.tsx` | `FindBar`: 현재 항목 위층 강조(`markActive`/`clearActive`), 색 옵션, 삭제 순서. `createKeptTerm`·설정 적용: `overviewRuler`, `overviewRulerBorder` | §4~§6 |
| `frontend/src/lib/findHighlight.ts` (신규) | 색·비율·폭 상수, `mixHex`, `rowRanges` | §4.4 |
| `frontend/src/lib/findHighlight.test.ts` (신규) | 테스트 6개 | §9.1 |
| `docs/decisions.md` | D96 (강조 색·모양) | — |
| `.claude/known-issues.md`, 루트 `CLAUDE.md` | KI-011 등록, 다음 번호 KI-012 | §7 |

백엔드·IPC·문구 파일은 바꾸지 않았다. 명세는 §2.4 의 "주황 계열 / 빨강 계열 / 스크롤 막대에도 표시" 범위 안이라 고치지 않았다. D96 은 색·모양만 적고 있고, 스크롤 막대 표시 켜기(§5)와 삭제 순서(§6)는 이 문서에만 있다.

---

### A-9. 검증

#### A-9.1 자동 (2026-11-18 실행)

```bash
cd frontend
npm ci                      # node_modules 가 없어 잠금 파일대로 설치
npx tsc --noEmit            # 통과
npx vitest run              # 테스트 파일 38개, 233개 모두 통과
```

`findHighlight.test.ts` 가 확인하는 것:

| 테스트 | 내용 |
|---|---|
| `mixHex` 비율 | 0 / 1 / 0.5 에서 기대 색, 결과가 6자리 형식 |
| `mixHex` 형식 오류 | `#RRGGBB` 가 아니면 덧칠 색을 그대로 |
| 6개 터미널 테마 | 찾은 항목 바탕 위의 글자 대비가 3 을 넘고, 현재 항목 바탕과 흰 글자의 대비가 4.5 를 넘고, 찾은 항목 바탕이 원색 주황보다 배경에 가깝다 |
| `rowRanges` | 한 줄 범위, 줄을 넘는 범위(줄마다 분할), 빈 조각 제거 |

자동 테스트가 **확인하지 않는 것**: 돋보기 버튼 클릭(`Panes` 화면 테스트 파일이 없다), `FindBar` 의 강조 생성·삭제, 스크롤 막대 표시. 이들은 §9.2 로 확인해야 한다. §6 의 재현 테스트는 xterm 내부를 들여다보는 임시 파일이라 지웠다.

#### A-9.2 수동 (`build/bin/dariterm.exe`) — 아직 하지 않음

실행 파일은 2026-11-18 10:00 빌드에 위 수정이 모두 들어 있다(§10 의 빌드 방법).

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 세션이 열린 칸의 머리글 돋보기 버튼 클릭 | 찾기 막대가 뜨고 입력창에 포커스. 안내 토스트는 뜨지 않음 |
| 2 | 여러 번 나오는 검색어 입력 | 현재 항목은 빨강 바탕·흰 글자·테두리, 나머지는 옅은 주황 바탕. 현재 항목이 가장 눈에 띔 |
| 3 | Enter / Shift+Enter, [다음] / [이전] | 빨강 강조가 다음 / 이전 항목으로 옮겨 감. 이전 자리는 옅은 주황으로 돌아옴 |
| 4 | 밝은 터미널 테마(DariTerm Light)로 바꿔 2번 반복 | 찾은 항목 위 글자가 읽히고 현재 항목이 구분됨 |
| 5 | 스크롤 막대 자리 확인 | 찾은 항목은 가운데 주황 띠, 현재 항목은 가로 빨강 선. 터미널 칸 수는 수정 전과 같음 |
| 6 | 한 번만 나오는 검색어로 찾은 뒤 Esc | 강조와 스크롤 막대 표시가 모두 사라짐. **한 자리에 남는 표시가 없음** |
| 7 | 6번을 검색어를 바꿔 가며 10번쯤 반복하고 스크롤 | 남는 표시 없음 |
| 8 | 칸을 둘로 나눈 뒤 포커스가 없는 칸의 돋보기 클릭 | 누른 칸에만 찾기 막대가 뜸 |
| 9 | `Ctrl+Shift+F`, 우클릭 → "찾기" | 전과 같이 열림 |
| 10 | 줄 끝에서 다음 줄로 이어지는 문자열 찾기 | 두 줄 모두 빨강 강조 |

6·7번에서 표시가 남으면 §6 외의 원인이 더 있는 것이다.

---

### A-10. 빌드 메모

이 PC 에는 `wails` 명령이 없고, go.mod 와 같은 v2.10.2 CLI 는 Go 1.27.1 에서 내부 오류로 실패한다. 호스트에 설치하지 않고 아래로 빌드했다. go.mod·go.sum 은 바뀌지 않고 버전 불일치 경고만 나온다.

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build    # → build/bin/dariterm.exe
```

---

### A-11. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 돋보기 버튼 | 안내 토스트 대신 찾기 막대가 열린다(의도한 변화) | §9.2 의 1 |
| 단축키·우클릭 메뉴 | 여는 경로의 코드는 그대로 | §9.2 의 9 |
| 강조 색 | 모든 터미널 테마에서 달라진다(의도한 변화). 색은 `findHighlight.ts` 상수로 조정 | §9.1, §9.2 의 2·4 |
| 현재 항목 강조의 수명 | 찾기 성공 때 만들고, 다음 찾기·선택 해제·막대 닫기·칸 정리 때 지운다. 줄이 스크롤백에서 지워지면 xterm 이 지운다 | §9.2 의 3·6 |
| 스크롤 막대 영역 | 모든 터미널에 14px 표시 영역이 생긴다. 폭이 스크롤 막대 기본 폭과 같아 칸 수는 그대로이고, 찾기를 쓰지 않으면 아무것도 그려지지 않는다 | §9.2 의 5 |
| 터미널을 가리는 프로그램(vim 등 대체 화면) | xterm 이 대체 화면에서는 표시 영역을 숨긴다 | — |
| 백엔드·IPC·데이터 파일·문구 | 변경 없음 | — |

---

### A-12. 남은 정리 거리 (이번에 하지 않음)

| 항목 | 상태 | 비고 |
|---|---|---|
| KI-011 의 1번(찾기 중 클립보드 덮임) | active | 권장 수정. §7 |
| §4.6 의 빈틈(애드온이 스스로 다시 선택한 항목이 회청색) | 미수정 | `onDidChangeResults` 에서 `markActive` |
| 찾기 막대에 "현재 순번/전체 개수" 표시 | 제안만 함 | 스크롤 막대 표시의 뜻(§5.3)을 헷갈리지 않게 해 준다. 명세의 찾기 막대 구성에 없는 요소 |
| [frontend/src/lib/notYet.ts](../../frontend/src/lib/notYet.ts), `locales/*.json` 의 `notYet.text` | 쓰는 곳이 없어짐 | 다른 미구현 기능에 다시 쓸 수 있어 그대로 뒀다 |
| 돋보기 버튼·`FindBar` 자동 테스트 | 없음 | 버튼은 "클릭 → `dariterm:find` 이벤트가 그 칸 id 로 발생" 하나면 된다 |

---

### A-13. 체크리스트

- [x] 1. 돋보기 버튼: 원인 확인, `Panes.tsx` 수정 (§3)
- [x] 2. 강조 색·모양: `findHighlight.ts` 신규, `FindBar` 위층 강조, D96 (§4)
- [x] 3. 스크롤 막대 표시: `overviewRuler` 옵션, 현재 항목 표시 (§5)
- [x] 4. 남는 표시: 강조 → 표식 순서로 삭제 (§6)
- [x] 5. KI-011 등록 (§7)
- [x] 6. `npx tsc --noEmit`, `npx vitest run` 통과, 실행 파일 빌드 (§9.1, §10)
- [ ] 7. 수동 검증 §9.2 (1·2·5·6·7 필수)
- [ ] 8. (선택) §12 의 정리 거리

---

### A-부록 A. 최종 코드 (2026-11-18 작업 종료 시점의 실제 파일 내용)

본문의 코드는 설명을 위해 일부를 줄였다. 아래는 파일에서 그대로 옮긴 것이다.

#### A-A.1 `frontend/src/lib/findHighlight.ts` (전체)

```ts
// 터미널 찾기 강조의 색·범위 계산. 근거: spec/ui/panes.md §2.4 (찾은 항목 주황 계열, 현재 항목 빨강 계열).
// 찾은 항목은 터미널 배경에 주황을 옅게 섞어 낮추고, 현재 항목은 진한 빨강 바탕 + 흰 글자 + 테두리로 올린다 (D96).

/** 찾은 항목(현재 항목 제외)의 기준 색과 섞는 비율. */
export const FIND_MATCH_COLOR = '#f59e0b';
export const FIND_MATCH_RATIO = 0.32;
/** 현재 항목의 바탕·글자 색. */
export const FIND_ACTIVE_BG = '#dc2626';
export const FIND_ACTIVE_FG = '#ffffff';
/** 스크롤 막대 표시 색. */
export const FIND_RULER_MATCH = '#f59e0b';
export const FIND_RULER_ACTIVE = '#ef4444';
/** 스크롤 막대 자리의 표시 폭(px). xterm 스크롤 막대 기본 폭과 같게 둔다. */
export const FIND_RULER_WIDTH = 14;

const HEX6 = /^#[0-9a-fA-F]{6}$/;

/** `#RRGGBB` 두 색을 섞는다(ratio = over 의 비중, 0~1). 형식이 아니면 over 를 그대로 돌려준다. */
export function mixHex(base: string, over: string, ratio: number): string {
  if (!HEX6.test(base) || !HEX6.test(over)) return over;
  const r = Math.min(1, Math.max(0, ratio));
  let out = '#';
  for (let i = 1; i < 7; i += 2) {
    const b = parseInt(base.slice(i, i + 2), 16);
    const o = parseInt(over.slice(i, i + 2), 16);
    out += Math.round(b + (o - b) * r).toString(16).padStart(2, '0');
  }
  return out;
}

export interface CellPos { x: number; y: number }
export interface RowRange { row: number; x: number; width: number }

/**
 * 선택 범위(시작 칸 포함, 끝 칸 제외, 0부터 세는 버퍼 좌표)를 줄별 범위로 나눈다.
 * 일치 항목이 줄 끝에서 다음 줄로 이어지면 줄마다 하나씩 나온다.
 */
export function rowRanges(start: CellPos, end: CellPos, cols: number): RowRange[] {
  const out: RowRange[] = [];
  for (let row = start.y; row <= end.y; row++) {
    const x0 = row === start.y ? start.x : 0;
    const x1 = row === end.y ? end.x : cols;
    if (x1 > x0) out.push({ row, x: x0, width: x1 - x0 });
  }
  return out;
}
```

#### A-A.2 `frontend/src/lib/findHighlight.test.ts` (전체)

```ts
import { describe, expect, it } from 'vitest';
import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_MATCH_COLOR, FIND_MATCH_RATIO } from './findHighlight';
import { TERM_THEMES } from './termThemes';

function luminance(hex: string): number {
  const ch = [1, 3, 5].map((i) => {
    const v = parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * ch[0] + 0.7152 * ch[1] + 0.0722 * ch[2];
}
function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

describe('findHighlight', () => {
  it('mixHex 는 두 색을 비율대로 섞는다', () => {
    expect(mixHex('#000000', '#ffffff', 0)).toBe('#000000');
    expect(mixHex('#000000', '#ffffff', 1)).toBe('#ffffff');
    expect(mixHex('#000000', '#ffffff', 0.5)).toBe('#808080');
    expect(mixHex('#0d1117', '#f59e0b', 0.32)).toMatch(/^#[0-9a-f]{6}$/);
  });

  it('mixHex 는 형식이 아닌 색이면 덧칠 색을 그대로 준다', () => {
    expect(mixHex('rgb(0,0,0)', '#f59e0b', 0.3)).toBe('#f59e0b');
  });

  it('모든 터미널 테마에서 찾은 항목 위의 글자가 읽히고, 찾은 항목은 원색보다 덜 튄다', () => {
    for (const th of TERM_THEMES) {
      const match = mixHex(th.background, FIND_MATCH_COLOR, FIND_MATCH_RATIO);
      expect(contrast(match, th.foreground), th.name).toBeGreaterThan(3);
      expect(contrast(FIND_ACTIVE_BG, '#ffffff'), th.name).toBeGreaterThan(4.5);
      // 찾은 항목은 원색 주황보다 배경에 가깝다(덜 튄다). 현재 항목과는 색상·흰 글자·테두리로 구분한다.
      expect(contrast(match, th.background), th.name).toBeLessThan(contrast(FIND_MATCH_COLOR, th.background));
    }
  });

  it('rowRanges 는 한 줄 범위를 그대로 준다', () => {
    expect(rowRanges({ x: 3, y: 10 }, { x: 8, y: 10 }, 80)).toEqual([{ row: 10, x: 3, width: 5 }]);
  });

  it('rowRanges 는 줄을 넘는 범위를 줄마다 나눈다', () => {
    expect(rowRanges({ x: 78, y: 5 }, { x: 4, y: 7 }, 80)).toEqual([
      { row: 5, x: 78, width: 2 },
      { row: 6, x: 0, width: 80 },
      { row: 7, x: 0, width: 4 },
    ]);
  });

  it('rowRanges 는 빈 줄 조각을 버린다', () => {
    expect(rowRanges({ x: 80, y: 1 }, { x: 0, y: 2 }, 80)).toEqual([]);
  });
});
```

#### A-A.3 `frontend/src/screens/TerminalView.tsx` — import (찾기 관련 줄)

```tsx
import { Terminal, type IDisposable } from '@xterm/xterm';
import { SearchAddon } from '@xterm/addon-search';
import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_ACTIVE_FG, FIND_MATCH_COLOR, FIND_MATCH_RATIO, FIND_RULER_MATCH, FIND_RULER_ACTIVE, FIND_RULER_WIDTH } from '../lib/findHighlight';
```

#### A-A.4 `TerminalView.tsx` — `FIND_EVENT` 와 `FindBar` (전체, 157~254행)

```tsx
/** 찾기 요청 이벤트(칸 id 가 detail). 단축키·칸 머리글·우클릭 메뉴가 보낸다. */
export const FIND_EVENT = 'dariterm:find';

function FindBar({ term, paneId, onClose }: { term: Terminal; paneId: string; onClose: () => void }) {
  const [q, setQ] = useState('');
  const addon = useRef<SearchAddon | null>(null);
  // 현재 항목 위에 얹는 강조(줄마다 하나). 검색 애드온은 현재 항목을 "선택"으로 표시하는데,
  // 애드온의 현재 항목 색은 선택 색 아래에 깔려 보이지 않는다. 선택 위(top)에 직접 그린다 (D96).
  const active = useRef<IDisposable[]>([]);
  void paneId;
  const clearActive = () => {
    for (const d of active.current) d.dispose();
    active.current = [];
  };
  const markActive = () => {
    clearActive();
    const sel = term.getSelectionPosition();
    if (!sel) return;
    const buf = term.buffer.active;
    const border = term.options.theme?.foreground ?? FIND_ACTIVE_FG;
    for (const r of rowRanges(sel.start, sel.end, term.cols)) {
      const marker = term.registerMarker(r.row - buf.baseY - buf.cursorY);
      const deco = term.registerDecoration({ marker, x: r.x, width: r.width, backgroundColor: FIND_ACTIVE_BG, foregroundColor: FIND_ACTIVE_FG, layer: 'top', overviewRulerOptions: { color: FIND_RULER_ACTIVE, position: 'full' } });
      deco?.onRender((el) => {
        el.style.outline = `1px solid ${border}`;
      });
      // 반드시 강조를 먼저, 표식을 나중에 지운다. 표식을 먼저 지우면 xterm 이 그 강조를 목록에서
      // 찾지 못해 남기고(줄 번호가 -1 로 바뀐 뒤라서), 다른 강조의 삭제까지 실패하게 만든다.
      active.current.push({
        dispose() {
          deco?.dispose();
          marker.dispose();
        },
      });
    }
  };
  useEffect(() => {
    const a = new SearchAddon();
    term.loadAddon(a);
    addon.current = a;
    // 터미널을 눌러 선택이 풀리면 현재 항목 강조도 지운다.
    const sub = term.onSelectionChange(() => {
      if (!term.hasSelection()) clearActive();
    });
    return () => {
      sub.dispose();
      clearActive();
      a.clearDecorations();
    };
    // clearActive 는 ref 만 다뤄 렌더마다 새로 만들어져도 같다.
  }, [term]);
  // 찾은 항목: 터미널 배경에 주황을 옅게 섞은 바탕(테마가 밝든 어둡든 글자가 읽힌다). 색은 #RRGGBB 만 받는다.
  const matchBg = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: FIND_RULER_MATCH, activeMatchColorOverviewRuler: FIND_RULER_ACTIVE, matchBackground: matchBg, activeMatchBackground: FIND_ACTIVE_BG } };
  const find = (text: string, dir: 1 | -1) => {
    const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
    if (found) markActive();
    else clearActive();
  };
  const next = (dir: 1 | -1) => {
    if (!q) return;
    find(q, dir);
  };
  const close = () => {
    clearActive();
    addon.current?.clearDecorations();
    onClose();
    term.focus();
  };
  return (
    <div className="find-bar" role="search" onMouseDown={(e) => e.stopPropagation()}>
      <input
        autoFocus
        className="input"
        placeholder={tr('find.placeholder')}
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          if (e.target.value) find(e.target.value, 1);
          else {
            clearActive();
            addon.current?.clearDecorations();
          }
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            close();
          } else if (e.key === 'Enter') {
            e.preventDefault();
            next(e.shiftKey ? -1 : 1);
          }
        }}
      />
      <button className="btn small-btn" onClick={() => next(-1)}>{tr('find.prev')}</button>
      <button className="btn small-btn" onClick={() => next(1)}>{tr('find.next')}</button>
      <button className="btn small-btn" onClick={close}>{tr('common.close')}</button>
    </div>
  );
}
```

#### A-A.5 `TerminalView.tsx` — 터미널을 만드는 옵션 (`createKeptTerm`, 스크롤 막대 표시 포함)

```tsx
function createKeptTerm(sessionId: string, paneId: string, host: HTMLElement): KeptTerm {
  const el = document.createElement('div');
  el.className = 'term-keep';
  const tracker = new LineTracker();
  const st = useApp.getState().store?.settings;
  const theme = resolveTermTheme(st?.terminalTheme);
  const t = new Terminal({
    fontFamily: st?.fontFamily,
    fontSize: st?.fontSize ?? 14,
    lineHeight: st?.lineHeight ?? 1.15,
    cursorStyle: st?.cursorStyle ?? 'block',
    cursorBlink: st?.cursorBlink ?? true,
    scrollback: st?.scrollback ?? 10000,
    rightClickSelectsWord: false,
    macOptionClickForcesSelection: true,
    altClickMovesCursor: true,
    drawBoldTextInBrightColors: true,
    minimumContrastRatio: 1,
    smoothScrollDuration: 0,
    allowProposedApi: true,
    // 찾기 결과를 스크롤 막대 자리에 표시한다(ui/panes.md §2.4). 폭은 xterm 스크롤 막대 기본 폭과 같아 칸 수가 바뀌지 않는다.
    overviewRuler: { width: FIND_RULER_WIDTH },
    theme: {
      overviewRulerBorder: theme.background,
      background: theme.background,
      foreground: theme.foreground,
      cursor: theme.cursor,
      cursorAccent: theme.background,
      selectionBackground: theme.selectionBackground,
      black: theme.black, red: theme.red, green: theme.green, yellow: theme.yellow,
      blue: theme.blue, magenta: theme.magenta, cyan: theme.cyan, white: theme.white,
      brightBlack: theme.brightBlack, brightRed: theme.brightRed, brightGreen: theme.brightGreen,
      brightYellow: theme.brightYellow, brightBlue: theme.brightBlue, brightMagenta: theme.brightMagenta,
      brightCyan: theme.brightCyan, brightWhite: theme.brightWhite,
    },
  });
```

#### A-A.6 `TerminalView.tsx` — 찾기 요청 수신과 찾기 막대 표시, 설정 변경 시 테마 적용

```tsx
  const [findOpen, setFindOpen] = useState(false);
  const [term, setTerm] = useState<Terminal | null>(null);

  // 찾기 요청 수신(이 칸에 대한 요청만)
  useEffect(() => {
    const on = (e: Event) => {
      if ((e as CustomEvent<string>).detail === paneId) setFindOpen(true);
    };
    window.addEventListener(FIND_EVENT, on);
    return () => window.removeEventListener(FIND_EVENT, on);
  }, [paneId]);
  …
  // 설정 변경은 열린 터미널 모두에 즉시 적용
  useEffect(() => {
    const t = termRef.current;
    if (!t || !settings) return;
    const theme = resolveTermTheme(settings.terminalTheme);
    t.options.fontFamily = settings.fontFamily;
    t.options.fontSize = settings.fontSize;
    t.options.lineHeight = settings.lineHeight;
    t.options.cursorStyle = settings.cursorStyle;
    t.options.cursorBlink = settings.cursorBlink;
    t.options.scrollback = settings.scrollback;
    t.options.theme = { ...t.options.theme, background: theme.background, foreground: theme.foreground, cursor: theme.cursor, selectionBackground: theme.selectionBackground, overviewRulerBorder: theme.background };
    fitRef.current?.fit();
  }, [settings]);
  …
      {findOpen && term && <FindBar term={term} paneId={paneId} onClose={() => setFindOpen(false)} />}
```


#### A-A.7 `frontend/src/screens/Panes.tsx` — 우클릭 메뉴의 찾기 항목과 머리글 돋보기 버튼

```tsx
    { label: tr('pane.find'), shortcut: 'Ctrl+Shift+F', disabled: !hasSession, onSelect: () => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id })) },
  …
        {hasSession && (
          <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id }))}><Search size={12} /></button>
        )}
```

---

## 부 B. 잠금 화면에서 Caps Lock 이 켜져 있는지 알려 주기 — 설계·코드·검증 — 3번 (D97)

**작성**: 2026-11-18 · **상태**: 코드 완료(2026-11-18, D97), 앱 수동 검증은 남음(§7.2) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(새 기능)
**관련**: [spec/ui/lock.md](../../spec/ui/lock.md), [frontend/src/screens/Lock.tsx](../../frontend/src/screens/Lock.tsx), [frontend/src/lib/useCapsLock.ts](../../frontend/src/lib/useCapsLock.ts), [frontend/src/screens/Lock.test.tsx](../../frontend/src/screens/Lock.test.tsx), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md)

> 이 문서 하나로 **무엇을 왜 넣었는지 → 어떻게 만들었는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 규모가 작아(새 파일 1개 + 화면 1개 + 문구) WIP 는 만들지 않았다.

---

### B-0. 한 줄 요약

잠금 화면에서 비밀번호를 칠 때 Caps Lock 이 켜져 있으면 입력창 아래에 경고색 한 줄 **"Caps Lock이 켜져 있습니다"** 를 보여 준다. 끄면 사라진다. 입력을 막거나 글자를 바꾸지는 않는다.

| 화면(잠금 화면의 모드) | 안내 |
|---|---|
| 잠금 해제 | 보임 |
| 처음 설정(비밀번호 만들기) | 보임 |
| 복구 코드로 재설정 | 보임 |
| 복구 코드 표시(비밀번호 입력 없음) | 안 봄 |

---

### B-1. 요청

2026-11-18 사용자 요청: "잠김 화면에서 비번 입력할 때, caps lock 켜 있는지 확인하게 해줘."

비밀번호 입력창은 글자가 가려져 있어서 Caps Lock 이 켜진 채로 치면 틀린 이유를 알기 어렵다.

---

### B-2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 상태를 읽는 방법 | 키·마우스 이벤트의 `getModifierState('CapsLock')` | 웹뷰(브라우저)에는 Caps Lock 상태를 따로 묻는 방법이 없고, 이벤트에 실려 올 때만 알 수 있다 |
| 듣는 이벤트 | 창의 `keydown`, `keyup`, `mousedown`, `mousemove` (캡처 단계) | 키만 들으면 첫 글자를 치기 전에는 알 수 없다. 마우스 이벤트에도 같은 정보가 실려 오므로, 마우스를 움직이는 순간부터 알 수 있다. `keyup` 도 듣는 것은 Caps Lock 키 자체를 눌렀을 때 바뀐 상태를 확실히 잡기 위함이다 |
| 첫 이벤트 전 | "꺼짐"으로 본다 | 알 수 없는 상태에서 경고를 띄우면 틀린 안내가 된다 |
| 창이 포커스를 잃을 때 | 안내를 지운다 | 다른 창에 있는 동안 Caps Lock 을 바꾸면 알 수 없다. 돌아와서 첫 이벤트에 다시 읽는다 |
| 적용 범위 | 잠금 화면의 비밀번호 입력 모드 3개 | 요청은 "잠김 화면에서 비번 입력할 때". 같은 화면의 처음 설정·복구도 비밀번호를 치는 곳이라 함께 넣었다. 서버 비밀번호 입력 등 다른 화면에는 넣지 않았다 |
| 표시 | 입력창과 오류 문구 사이의 한 줄, 경고색 바탕 + 아이콘 | 오류(빨강)와 구분되는 주의 수준 |
| 백엔드로 묻는 안(Win32 `GetKeyState`) | 기각 | OS 별 코드가 늘어난다. 이벤트로 충분하다 |

**알아 둘 한계**: 잠금 화면이 뜬 직후 키도 마우스도 건드리지 않은 순간에는 Caps Lock 이 켜져 있어도 안내가 없다. 마우스를 움직이거나 키를 하나 누르면 바로 나타난다.

---

### B-3. 코드 — 새 파일 `frontend/src/lib/useCapsLock.ts` (전체)

```ts
// Caps Lock 이 켜져 있는지 알려 주는 훅. 비밀번호 입력 화면에서 쓴다 (D97).
// 브라우저는 Caps Lock 상태를 따로 물어볼 방법이 없고, 키·마우스 이벤트에 실려 올 때만 알 수 있다.
// 그래서 키를 누르고 뗄 때와 마우스를 움직이거나 누를 때 상태를 읽는다(첫 이벤트 전에는 "꺼짐"으로 본다).
import { useEffect, useState } from 'react';

type ModifierEvent = { getModifierState?: (key: string) => boolean };

/** 이벤트에서 Caps Lock 상태를 읽는다. 알 수 없으면 null. */
export function capsLockOf(e: ModifierEvent): boolean | null {
  return typeof e.getModifierState === 'function' ? e.getModifierState('CapsLock') : null;
}

const EVENTS = ['keydown', 'keyup', 'mousedown', 'mousemove'] as const;

/** enabled 가 참인 동안 창의 키·마우스 이벤트로 Caps Lock 상태를 따라간다. */
export function useCapsLock(enabled = true): boolean {
  const [on, setOn] = useState(false);
  useEffect(() => {
    if (!enabled) {
      setOn(false);
      return;
    }
    const read = (e: Event) => {
      const state = capsLockOf(e as unknown as ModifierEvent);
      if (state !== null) setOn(state);
    };
    // 창을 벗어난 동안 바뀐 상태는 알 수 없으므로 모른다고 보고 끈다. 돌아와서 첫 이벤트에 다시 읽는다.
    const reset = () => setOn(false);
    for (const name of EVENTS) window.addEventListener(name, read, true);
    window.addEventListener('blur', reset);
    return () => {
      for (const name of EVENTS) window.removeEventListener(name, read, true);
      window.removeEventListener('blur', reset);
    };
  }, [enabled]);
  return on;
}
```

읽는 법:
- `capsLockOf` 는 `getModifierState` 가 없는 이벤트(일부 합성 이벤트)를 "모름(null)"으로 돌려준다. 모르면 상태를 바꾸지 않는다.
- 캡처 단계(`true`)로 듣는다. 입력창이나 다른 요소가 이벤트 전파를 막아도 받는다.
- `enabled` 가 거짓이 되면 리스너를 떼고 상태를 끈다(복구 코드 표시 화면).
- `mousemove` 는 자주 오지만 값이 같으면 React 가 다시 그리지 않는다(`setOn` 에 같은 값).

---

### B-4. 코드 — `frontend/src/screens/Lock.tsx`

#### B-4.1 import

```tsx
// before
import { Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';

// after
import { ArrowBigUpDash, Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';
import { useCapsLock } from '../lib/useCapsLock';
```

#### B-4.2 상태와 안내 요소 (`return` 바로 위, [Lock.tsx:146-151](../../frontend/src/screens/Lock.tsx#L146-L151))

```tsx
  const score = passwordScore(pw);
  // 비밀번호를 입력하는 화면(잠금 해제·처음 설정·복구)에서만 Caps Lock 을 살핀다 (D97).
  const capsOn = useCapsLock(mode !== 'recovery');
  const capsNote = capsOn && (
    <div className="caps-note" role="status"><ArrowBigUpDash size={14} />{t('lock.capsLock')}</div>
  );
```

#### B-4.3 세 폼에 넣기

세 폼(`unlock`, `setup`, `recover`) 모두 오류 문구 줄 바로 위에 넣었다.

```tsx
// before (세 곳 모두 같은 줄)
            <div className="err-text" role="alert">{error}</div>

// after
            {capsNote}
            <div className="err-text" role="alert">{error}</div>
```

위치: 잠금 해제 [Lock.tsx:176](../../frontend/src/screens/Lock.tsx#L176), 처음 설정 [Lock.tsx:229](../../frontend/src/screens/Lock.tsx#L229), 복구 [Lock.tsx:289](../../frontend/src/screens/Lock.tsx#L289).

잠금 해제 폼의 해당 부분(수정 후):

```tsx
            <input
              className="input"
              type="password"
              placeholder={t('lock.pw')}
              autoComplete="off"
              autoFocus
              value={pw}
              onChange={(e) => setPw(e.target.value)}
              aria-label={t('lock.pw')}
            />
            {capsNote}
            <div className="err-text" role="alert">{error}</div>
            <div className="lock-actions">
```

`role="status"` 는 화면 낭독기가 안내가 나타날 때 읽게 한다. 오류 문구의 `role="alert"` 보다 낮은 수준이다.

---

### B-5. 코드 — 스타일과 문구

#### B-5.1 `frontend/src/styles/app.css` (`.lock-actions` 위에 한 줄)

```css
.caps-note { display: flex; align-items: center; gap: 6px; padding: 6px 10px; border-radius: 6px; background: var(--warn-soft); color: var(--warn); font-size: 12px; font-weight: 600; }
```

색은 기존 토큰 `--warn`(`#f59e0b`), `--warn-soft`(`rgba(245, 158, 11, 0.14)`)를 쓴다([tokens.css](../../frontend/src/styles/tokens.css)).

#### B-5.2 `locales/*.json` — 키 `lock.capsLock` (10개 언어, `lock.forgot` 다음 줄)

| 언어 | 문구 |
|---|---|
| ko | Caps Lock이 켜져 있습니다 |
| en | Caps Lock is on |
| ja | Caps Lock がオンになっています |
| zh | Caps Lock 已开启 |
| tw | Caps Lock 已開啟 |
| de | Die Feststelltaste ist aktiviert |
| fr | Le verrouillage des majuscules est activé |
| es | Bloq Mayús está activado |
| pt | Caps Lock está ativado |
| ru | Включён Caps Lock |

한국어·영어 외에는 이 수정을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

---

### B-6. 명세·기록

- `docs/decisions.md` **D97**.
- 명세 `spec/ui/lock.md` 에는 없는 추가 요소다. 명세는 고치지 않았다(다음에 명세를 손볼 때 잠금 화면 항목에 넣는다).
- 백엔드·IPC·데이터 파일은 바꾸지 않았다.

---

### B-7. 검증

#### B-7.1 자동 (2026-11-18 실행) — `frontend/src/screens/Lock.test.tsx` 에 추가한 4개 (전체)

```tsx
describe('Caps Lock 안내 (D97)', () => {
  beforeEach(() => setBackend(null));

  it('Caps Lock 이 켜진 채 키를 누르면 안내가 보이고, 끄면 사라진다', async () => {
    await setup(true);
    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
    fireEvent.keyUp(window, { key: 'CapsLock', modifierCapsLock: false });
    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
  });

  it('키를 누르기 전이라도 마우스를 움직이면 상태를 안다', async () => {
    await setup(true);
    fireEvent.mouseMove(window, { modifierCapsLock: true });
    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
  });

  it('창이 포커스를 잃으면 안내를 지운다', async () => {
    await setup(true);
    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
    fireEvent.blur(window);
    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
  });

  it('처음 설정 화면에서도 보인다', async () => {
    await setup(false);
    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
  });
});
```

`setup(true)` 는 볼트를 만든 뒤 잠근 상태(잠금 해제 화면), `setup(false)` 는 볼트가 없는 상태(처음 설정 화면)로 `Lock` 을 그리는 기존 도우미다.

```bash
cd frontend
npx tsc --noEmit     # 통과
npx vitest run       # 38개 파일, 233개 통과 (이 4개 포함)
```

자동 테스트가 확인하지 않는 것: 실제 키보드의 Caps Lock 과 웹뷰가 주는 값이 맞는지, 복구 화면에서의 표시, 화면 모양.

#### B-7.2 수동 (`build/bin/dariterm.exe`) — 아직 하지 않음

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 앱을 잠근 뒤 Caps Lock 을 켜고 비밀번호 칸에 글자를 친다 | 입력창 아래에 주황 바탕의 "Caps Lock이 켜져 있습니다" |
| 2 | Caps Lock 키를 다시 눌러 끈다 | 안내가 바로 사라진다 |
| 3 | Caps Lock 을 켠 채 앱을 잠그고, 키를 누르지 않고 마우스만 움직인다 | 안내가 나타난다 |
| 4 | 안내가 보이는 채로 다른 창을 눌렀다가 돌아온다 | 돌아온 직후에는 안내가 없고, 마우스를 움직이거나 키를 누르면 다시 나타난다 |
| 5 | "비밀번호를 잊으셨나요?" → 복구 화면에서 Caps Lock 을 켠다 | 안내가 보인다 |
| 6 | 볼트가 없는 새 데이터 폴더로 실행(처음 설정)해 Caps Lock 을 켠다 | 안내가 보인다 |
| 7 | 자동 잠금으로 세션 위에 덮인 잠금 화면에서 1번 반복 | 같은 안내 |
| 8 | 틀린 비밀번호로 오류 문구가 뜬 상태에서 Caps Lock 을 켠다 | 안내와 오류 문구가 함께 보이고 배치가 깨지지 않는다 |

---

### B-8. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 잠금 화면 배치 | 안내가 나타날 때 한 줄만큼 아래 요소가 밀린다 | 안내는 Caps Lock 이 켜졌을 때만 생긴다 |
| 성능 | 잠금 화면이 떠 있는 동안 창의 `mousemove` 를 듣는다 | 값이 같으면 다시 그리지 않는다. 잠금 화면이 사라지면 리스너도 떼어진다 |
| 다른 화면 | `useCapsLock` 은 `Lock` 에서만 쓴다 | — |
| 원격 데스크톱·가상 키보드 | 호스트와 원격의 Caps Lock 이 어긋나면 웹뷰가 받는 값이 실제와 다를 수 있다 | 확인하지 않았다 |
| 백엔드·IPC | 변경 없음 | — |

---

### B-9. 남은 정리 거리 (이번에 하지 않음)

| 항목 | 비고 |
|---|---|
| 다른 비밀번호 입력에도 같은 안내 | 서버 비밀번호 질문 창, 설정 > 보안의 비밀번호 변경. `useCapsLock` 을 그대로 쓸 수 있다 |
| 명세 `spec/ui/lock.md` 반영 | §6 |
| 8개 언어 문구 검수 | §5.2 |

---

### B-10. 체크리스트

- [x] 1. `useCapsLock.ts` 신규 (§3)
- [x] 2. `Lock.tsx`: 훅 사용, 세 폼에 안내 (§4)
- [x] 3. `app.css` `.caps-note`, 문구 10개 언어 (§5)
- [x] 4. decisions D97 (§6)
- [x] 5. 테스트 4개 추가, `tsc`·`vitest` 통과, 실행 파일 빌드 (§7.1)
- [ ] 6. 수동 검증 §7.2 (1·2·3 필수)
- [ ] 7. (선택) §9 의 정리 거리

---

## 부 C. 서버가 timeout 으로 끝낸 SSH 세션이 자동 재연결되지 않음 — 원인·수정·검증 — 4번 (D98)

**작성**: 2026-11-18 · **상태**: 코드 수정 완료(2026-11-18, D98), 실제 서버 검증은 남음(§6.2) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(발견 즉시 수정해 등록하지 않음)
**관련**: [spec/behavior/session.md](../../spec/behavior/session.md) §3·§4, [internal/session/session.go](../../internal/session/session.go), [internal/session/control.go](../../internal/session/control.go), [internal/sshconn/shell.go](../../internal/sshconn/shell.go), 앞선 문서 [2026-11-17-reconnect-password-after-screensaver.md](./2026-11-17-reconnect-password-after-screensaver.md)(안 F, D93), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 규모가 작아(Go 파일 2개 + 테스트 1개 + 명세 1곳) WIP 는 만들지 않았다.

---

### C-0. 한 줄 요약

2026-11-17 수정(D93)에서 "서버가 종료 코드와 함께 셸을 끝내면 자동 재연결하지 않는다"로 바꿨다. 사용자가 `exit` 한 세션에 다시 붙지 않게 하려는 것이었는데, **서버의 유휴 timeout 로그아웃도 똑같이 종료 코드와 함께 끝나서** timeout 으로 끝난 세션까지 재연결되지 않게 됐다.

수정: 종료 코드로는 둘을 구분할 수 없으므로 **끝나기 직전에 사용자 입력이 있었는지**로 가린다. 직전 입력이 있었으면 사용자가 끝낸 것(닫힘 유지), 입력 없이 있다가 끝났으면 서버가 끝낸 것(자동 재연결)으로 본다.

| 상황 | 전 (D93) | 후 (D98) |
|---|---|---|
| 사용자가 `exit`·`logout` 입력 | 닫힘, Enter 로 재연결 | 그대로 |
| 서버의 유휴 timeout 로그아웃 | 닫힘, Enter 로 재연결 | **자동 재연결** |
| 관리자가 세션을 종료, 서버 재시작(셸이 신호로 끝남) | 닫힘, Enter 로 재연결 | **자동 재연결** |
| 로그인 직후(10초 안) 셸이 끝남 | 닫힘 | 그대로 |
| 네트워크 단절·keepalive 실패(종료 코드 없음) | 자동 재연결 | 그대로 |

---

### C-1. 발견 경위

2026-11-18 사용자 보고: "명시적인 logout 을 수정하면서 서버에서 timeout 나서 끊어진 경우에, 재연결 기능이 있는데 재연결이 안 된다. exit 명령으로 끊어진 경우와 timeout 나서 끊어진 경우가 구분이 안 되면 exit 인 경우에도 재연결하도록 해 달라."

---

### C-2. 원인 분석 (Cause)

#### C-2.1 코드 경로

```
서버가 셸을 끝냄
  → sshconn.Shell 의 대기 고루틴          internal/sshconn/shell.go:64-71
      session.Wait() 결과를 isNormalExit 로 판정     shell.go:81-87
        오류 없음(종료 코드 0) / *ssh.ExitError(0 아닌 코드·신호)  → 정상 종료
        *ssh.ExitMissingError(종료 코드 없음) / 전송 오류            → 끊김
  → Shell.Read                            shell.go:91-100
        정상 종료 → transport.ErrRemoteExited
        끊김      → transport.ErrRemoteClosed
  → Session.readLoop                      internal/session/session.go:325-349
        ErrRemoteExited → onRemoteExited    ← 문제 지점
        그 밖           → onDisconnected (자동 재연결 예약)
```

#### C-2.2 왜 timeout 이 재연결되지 않았는가

1. 수정 전 `onRemoteExited` 는 `ErrRemoteExited` 이면 **무조건** 상태를 `closed` 로 두고 "Enter 키를 누르면 다시 연결합니다." 만 출력했다. 자동 재연결 예약(`scheduleRetryLocked`)을 부르지 않았다.
2. `ErrRemoteExited` 는 "셸이 종료 코드 또는 신호와 함께 끝났다"는 뜻이다. 사용자의 `exit` 만이 아니라 **셸 스스로의 유휴 timeout 로그아웃**(예: bash 의 `TMOUT`)도 셸이 정상적으로 끝나며 종료 코드를 보내므로 같은 길로 들어온다.
3. D93 을 정할 때 유휴 로그아웃을 "정상 종료"에 일부러 넣었다(오래 둔 뒤 재접속하면서 암호 창이 뜨는 증상을 줄이려는 목적). 그 증상은 같은 날 D94(앱 잠금 중에는 질문 창을 띄우지 않고 잠금 해제를 기다림)로 따로 해결됐으므로, 유휴 로그아웃을 재연결에서 뺄 이유가 남지 않았다.

#### C-2.3 왜 종료 코드로는 구분할 수 없는가

- `exit`(인자 없음)와 유휴 timeout 로그아웃은 둘 다 셸이 스스로 끝나며 종료 코드를 보낸다. SSH 프로토콜이 클라이언트에 알려 주는 것은 종료 코드(또는 신호)뿐이고, "왜 끝났는지"는 없다.
- 이 판단은 bash 의 일반적인 동작에 근거한 것이고, 요청자의 서버에서 timeout 이 어떤 방식으로 끊는지는 확인하지 않았다(§6.2 의 확인 항목).
- 서버가 **연결 자체를 끊는 방식**(sshd 의 유휴 차단, 방화벽)이면 종료 코드 없이 끝나므로 원래부터 끊김으로 처리되어 자동 재연결됐다. 이번 증상이 나왔다는 것은 요청자의 서버가 셸 timeout 방식이라는 뜻으로 본다.

---

### C-3. 설계 결정

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 끝나기 직전 사용자 입력 여부로 구분** | 유휴 timeout 은 입력이 한참 없을 때만 일어나므로 정확도가 높다. 틀려도 `exit` 뒤 재연결되는 쪽으로만 틀린다(요청자가 허용한 동작). 변경이 작다 | **채택** |
| B. 정상 종료도 전부 자동 재연결(요청자의 차선책) | 가장 단순하지만 `exit` 할 때마다 2초 뒤 다시 붙는다 | 기각 |
| C. 종료 직전 화면 문구("auto-logout" 등)로 구분 | 셸 종류·언어 설정마다 문구가 다르고 출력 조각 경계 처리도 필요 | 기각 |

**규칙 (명세 §4 에 반영한 내용)**

SSH 셸이 종료 코드·신호와 함께 끝나면 `● 서버가 세션을 끝냈습니다.` 를 출력하고 남은 SSH 연결을 닫은 뒤:

- **사용자가 끝낸 것**으로 보는 경우 → 상태 `closed`, 시도 횟수 0, `● Enter 키를 누르면 다시 연결합니다.`
  - 끝나기 전 **10초 안에** 이 세션으로 보낸 사용자 입력이 있었다, 또는
  - **연결된 지 10초가 안 됐다**(로그인 직후 셸이 끝나는 서버에 2초마다 다시 붙는 것을 막는다).
- **그 밖**(입력 없이 있다가 끝남) → 끊김과 같은 자동 재연결 규칙(2, 4, 8, 16, 30초, 최대 5회). 자동 재연결 조건을 충족하지 않으면(설정 꺼짐·횟수 초과) 위와 같이 `closed`.

접속 직후 자동으로 보내는 입력(세션 유지 명령·자동 실행 명령)은 `Input` 을 거치지 않으므로 사용자 입력으로 세지 않는다.

**10초로 정한 근거**: `exit` 는 보통 1초 안에 끝난다. 로그아웃 스크립트가 느린 서버를 감안해 넉넉히 잡았다. 유휴 timeout 은 대개 분 단위라 입력 뒤 10초 안에 일어나지 않는다. 값은 상수 `G_cdurUserExitWindow` 하나다.

---

### C-4. 수정 내용

#### C-4.1 `internal/session/session.go` — 시각 두 개와 기준 상수

`Session` 구조체에 필드 두 개를 더하고 상수를 정의했다.

```go
	tConnected   time.Time // 지금 연결이 맺어진 시각 (서버 종료가 사용자 종료인지 가릴 때 쓴다)
	tLastInput   time.Time // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
}

// G_cdurUserExitWindow 는 서버가 셸을 끝냈을 때 "사용자가 끝낸 것"으로 보는 시간 창이다 (session.md §4).
// 끝나기 전 이 시간 안에 사용자 입력이 있었거나, 연결된 지 이 시간이 안 됐으면 사용자가 끝낸 것으로 본다.
// 그 밖(입력 없이 한참 있다가 끝남)은 서버의 유휴 로그아웃 같은 서버 쪽 종료로 보고 자동 재접속한다.
const G_cdurUserExitWindow = 10 * time.Second
```

#### C-4.2 `session.go` `connect` — 연결될 때 시각 기록

```go
// before
	pSession.info.M_i64ConnectedAt = time.Now().UnixMilli()

// after
	tNow := time.Now()
	pSession.info.M_i64ConnectedAt = tNow.UnixMilli()
	pSession.tConnected = tNow
	pSession.tLastInput = time.Time{}   // 이전 연결의 입력 시각은 버린다
```

#### C-4.3 `session.go` `onRemoteExited` — 누가 끝냈는지 가려 처리

```go
// before
func (pSession *Session) onRemoteExited(iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	conn := pSession.transport
	pSession.transport = nil
	pSession.attempts = 0
	pSession.info.M_szState = G_cszStateClosed
	pSession.info.M_szMessage = ""
	pSession.writeOutputUnlockedNotice(MsgRemoteExited())
	pSession.writeOutputUnlockedNotice(MsgGiveUp())
	pSession.mtxData.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	pSession.publishState()
}

// after
// userEndedLocked 는 mtxData 를 잡은 상태에서, 방금 끝난 셸을 사용자가 끝낸 것으로 볼지 판단한다 (session.md §4).
func (pSession *Session) userEndedLocked(tNow time.Time) bool {
	if tNow.Sub(pSession.tConnected) < G_cdurUserExitWindow {
		return true
	}
	return !pSession.tLastInput.IsZero() && tNow.Sub(pSession.tLastInput) < G_cdurUserExitWindow
}

func (pSession *Session) onRemoteExited(iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	conn := pSession.transport
	pSession.transport = nil
	pSession.writeOutputUnlockedNotice(MsgRemoteExited())
	bRetry := !pSession.userEndedLocked(time.Now()) &&
		CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
	if bRetry {
		pSession.scheduleRetryLocked()          // 끊김과 같은 예약(상태 reconnecting + 안내 줄)
	} else {
		pSession.attempts = 0
		pSession.info.M_szState = G_cszStateClosed
		pSession.info.M_szMessage = ""
		pSession.writeOutputUnlockedNotice(MsgGiveUp())
	}
	pSession.mtxData.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	pSession.publishState()
}
```

현재 위치: `userEndedLocked` [session.go:286](../../internal/session/session.go#L286), `onRemoteExited` [session.go:298](../../internal/session/session.go#L298).

#### C-4.4 `internal/session/control.go` `Input` — 입력 시각 기록

```go
// before
	transport := pSession.transport
	szState := pSession.info.M_szState
	pSession.mtxData.Unlock()

// after
	transport := pSession.transport
	szState := pSession.info.M_szState
	if transport != nil && szText != "" {
		pSession.tLastInput = time.Now()
	}
	pSession.mtxData.Unlock()
```

import 에 `"time"` 을 더했다. 위치: [control.go:17-27](../../internal/session/control.go#L17-L27).

#### C-4.5 바꾸지 않은 것

- `internal/sshconn/shell.go` 의 정상 종료/끊김 판정, `transport.ErrRemoteExited` 는 그대로다.
- 안내 문구(`MsgRemoteExited`, `MsgGiveUp`, `MsgReconnectScheduled`)와 `locales/*.json` 은 그대로다. 문구를 새로 만들지 않고 기존 두 줄을 조합했다.
- 프런트엔드·IPC 는 그대로다.

#### C-4.6 명세

`spec/behavior/session.md` §3 의 정상 종료 안내 줄 설명과 §4 의 "서버가 정상 종료한 세션" 항목을 §3 의 규칙으로 고쳤다(개정 일자 2026-11-18 표기).

---

### C-5. 화면에 보이는 모습

**timeout 으로 끝난 경우(후)**

```
● 서버가 세션을 끝냈습니다.
● 2초 후 자동 재연결합니다... (Enter: 즉시 재연결)
● {제목} ({대상}) 연결 중...
```

**`exit` 를 친 경우(전과 같음)**

```
● 서버가 세션을 끝냈습니다.
● Enter 키를 누르면 다시 연결합니다.
```

---

### C-6. 검증

#### C-6.1 자동 (2026-11-18 실행)

[internal/session/exit_unlock_test.go](../../internal/session/exit_unlock_test.go) 에 테스트 3개와 시각을 과거로 옮기는 도우미 `backdateForTest` 를 더했다.

| 테스트 | 확인하는 것 |
|---|---|
| `TestRemoteExitAfterIdleAutoReconnects` (신규) | 연결·마지막 입력이 1분 전인 세션이 정상 종료 → "서버가 세션을 끝냈습니다." + 자동 재연결 예약 → 두 번째 접속 성공, 남은 연결 닫힘 |
| `TestRemoteExitRightAfterInputStaysClosed` (신규) | 1분 전에 연결된 세션에 `exit\r` 입력 직후 정상 종료 → `closed`, 2.5초 동안 재접속 없음 |
| `TestRemoteExitAfterIdleWithAutoReconnectOff` (신규) | 자동 재연결 설정이 꺼져 있으면 유휴 뒤 종료도 `closed` + Enter 안내 |
| `TestRemoteExitDoesNotAutoReconnect` (기존, 그대로 통과) | 연결 직후(10초 안) 정상 종료 → `closed`, 재접속 없음, Enter 로는 재연결 |
| `TestDropStillAutoReconnects` (기존) | 끊김은 전과 같이 자동 재연결 |

```bash
go build ./...
go test ./...        # 전부 통과
wails build          # build/bin/dariterm.exe (2026-11-18 10:00)
```

`go test -race` 는 이 PC 에 cgo 컴파일러가 없어 실행하지 못했다. 새 필드 두 개는 모두 `mtxData` 를 잡은 상태에서만 읽고 쓴다.

#### C-6.2 수동 (실제 서버) — 아직 하지 않음

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 유휴 timeout 이 있는 서버에 접속하고 입력 없이 timeout 까지 기다린다 | "서버가 세션을 끝냈습니다." 다음에 "2초 후 자동 재연결합니다..." 가 나오고 다시 로그인된다 |
| 2 | 접속 후 10초 넘게 지난 뒤 `exit` 입력 | "Enter 키를 누르면 다시 연결합니다." 가 나오고 자동 재연결하지 않는다 |
| 3 | 2번 상태에서 Enter | 다시 연결된다 |
| 4 | 접속하자마자 `exit` | 닫힘(재연결 없음) |
| 5 | 설정 "연결이 끊기면 다시 연결"을 끄고 1번 반복 | 닫힘 + Enter 안내 |
| 6 | 앱을 잠근 상태에서 1번 반복 | 암호 창이 뜨지 않고 "앱 잠금을 해제하면 다시 연결합니다." 로 기다린다(D94) |
| 7 | 1번에서 timeout 직전 화면의 마지막 줄을 적어 둔다 | 셸 timeout 방식인지(문구가 있음) 연결 차단 방식인지(문구 없음) 확인용 |

---

### C-7. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 유휴 timeout 이 있는 서버 | 로그아웃될 때마다 다시 로그인한다. 세션이 timeout 주기마다 새로 맺어지고, 서버의 로그인 기록이 늘어난다 | 의도한 동작(요청 사항). 원치 않으면 설정 "연결이 끊기면 다시 연결"을 끈다. 지금은 전체 설정 하나뿐이고 서버별 설정은 없다 |
| `exit` 가 10초 넘게 걸리는 서버 | 사용자가 끝냈는데 자동 재연결된다 | 요청자가 허용한 방향의 오판. 기준은 `G_cdurUserExitWindow` 로 조정 |
| 입력 뒤 10초 안에 서버가 끝낸 경우(관리자 종료가 우연히 겹침 등) | 자동 재연결하지 않고 닫힘으로 남는다 | Enter 로 재연결 |
| 사용자가 치지 않은 입력 | 터미널이 서버의 질의에 자동으로 답하는 바이트(커서 위치 보고 등)와 일괄 입력도 `Input` 으로 들어와 "사용자 입력"으로 세어진다. 유휴 프롬프트에서는 보통 이런 입력이 없다 | 문제가 보고되면 입력 출처를 구분한다 |
| 로그인 직후 끝나는 서버 | 10초 규칙으로 닫힘에 머문다(반복 접속 없음) | `TestRemoteExitDoesNotAutoReconnect` |
| 연결 10초 뒤부터 주기적으로 끝내는 서버 | 끝날 때마다 재연결한다(연결에 성공하면 시도 횟수가 0 으로 돌아가므로 5회 한도에 걸리지 않는다). 끊김도 원래 같은 성질이다 | 간격이 최소 10초 + 2초라 폭주하지는 않는다 |
| 세션 유지(tmux) 서버 | timeout 뒤 재연결하면 세션 유지 명령이 다시 실행되어 tmux 에 다시 붙는다 | 끊김 뒤 재연결과 같은 경로 |
| 텔넷·시리얼·로컬 셸 | `ErrRemoteExited` 를 내지 않으므로 영향 없음 | — |

---

### C-8. 기록

| 파일 | 내용 |
|---|---|
| `spec/behavior/session.md` §3·§4 | 사용자가 끝낸 경우 / 서버 쪽에서 끝낸 경우 규칙 |
| `docs/decisions.md` | **D98** (D93 개정) |
| `.claude/known-issues.md` | 변경 없음 |

---

### C-9. 체크리스트

- [x] 1. 원인 확인 (§2)
- [x] 2. `session.go`: 필드·상수, `connect` 시각 기록, `userEndedLocked`, `onRemoteExited` (§4.1~4.3)
- [x] 3. `control.go`: `Input` 입력 시각 기록 (§4.4)
- [x] 4. 테스트 3개 추가, `go build ./... && go test ./...` 통과 (§6.1)
- [x] 5. 명세 §3·§4, decisions D98 (§8)
- [x] 6. `wails build`
- [ ] 7. 실제 서버 수동 검증 §6.2 (1·2·6 필수)
- [ ] 8. `go test -race ./internal/session/` (cgo 컴파일러가 있는 환경에서)

---

### C-부록 A. 최종 코드 (2026-11-18 작업 종료 시점의 실제 파일 내용)

같은 날 뒤이은 수정(D99·D100)으로 같은 파일에 줄이 더해졌다. 아래는 그것까지 반영된 최종 내용이다.

#### C-A.1 `internal/session/session.go` — `Session` 구조체와 기준 상수

```go
// Session 은 열려 있는 접속 하나다. 상태 전이·재접속·출력 전달을 맡는다.
type Session struct {
	mtxData      sync.Mutex
	cfg          OpenConfig
	info         Info
	emit         EmitFunc
	transport    Transport
	gen          int
	attempts     int
	wasConnected bool
	userClosed   bool
	iCols        int
	iRows        int
	timerRetry   *time.Timer
	timerStartup *time.Timer
	decoder      *Decoder
	batcher      *Batcher
	credit       *Credit // 출력 흐름 제어 (session.md §8)
	logger       *Logger
	dialing      bool      // Dialer 를 부르는 중 (같은 세션에 접속이 두 번 겹치지 않게)
	waitUnlock   bool      // 앱 잠금 해제를 기다리는 중 (ErrWaitUnlock, session.md §4)
	tConnected   time.Time // 지금 연결이 맺어진 시각 (서버 종료가 사용자 종료인지 가릴 때 쓴다)
	tLastInput   time.Time // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
	iLogSyncSec  int       // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
}

// G_cdurUserExitWindow 는 서버가 셸을 끝냈을 때 "사용자가 끝낸 것"으로 보는 시간 창이다 (session.md §4).
// 끝나기 전 이 시간 안에 사용자 입력이 있었거나, 연결된 지 이 시간이 안 됐으면 사용자가 끝낸 것으로 본다.
// 그 밖(입력 없이 한참 있다가 끝남)은 서버의 유휴 로그아웃 같은 서버 쪽 종료로 보고 자동 재접속한다.
const G_cdurUserExitWindow = 10 * time.Second
```

#### C-A.2 `session.go` — `connect` 의 연결 성공 부분 (시각 기록)

```go
	pSession.transport = conn
	pSession.attempts = 0
	pSession.wasConnected = true
	pSession.info.M_szState = G_cszStateConnected
	pSession.info.M_szMessage = ""
	tNow := time.Now()
	pSession.info.M_i64ConnectedAt = tNow.UnixMilli()
	pSession.tConnected = tNow
	pSession.tLastInput = time.Time{}
	iCols, iRows := pSession.iCols, pSession.iRows
	pSession.mtxData.Unlock()

	if iCols > 0 && iRows > 0 {
		_ = conn.Resize(iCols, iRows)
	}
	if !bLocal {
		// 접속 중 안내 줄을 지운다 (session.md §3).
		pSession.writeOutput(G_cszClearLine)
	}
	pSession.publishState()
	go pSession.readLoop(conn, iGen)
	pSession.scheduleStartup(iGen)
}
```


#### C-A.3 `session.go` — `onDisconnected`, `userEndedLocked`, `onRemoteExited`, `readLoop`

```go
// onDisconnected 는 연결이 끊긴 것을 처리한다. 오래된 세대의 알림과 사용자 닫기 뒤의 알림은 무시한다.
func (pSession *Session) onDisconnected(szReason string, iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	pSession.transport = nil
	pSession.writeOutputUnlockedNotice(MsgDisconnected(szReason))
	bAllowed := CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
	if bAllowed {
		pSession.scheduleRetryLocked()
		pSession.mtxData.Unlock()
		pSession.publishState()
		return
	}
	pSession.attempts = 0
	pSession.info.M_szState = G_cszStateClosed
	pSession.info.M_szMessage = ""
	pSession.writeOutputUnlockedNotice(MsgGiveUp())
	pSession.mtxData.Unlock()
	pSession.publishState()
}

// userEndedLocked 는 mtxData 를 잡은 상태에서, 방금 끝난 셸을 사용자가 끝낸 것으로 볼지 판단한다 (session.md §4).
// 셸의 유휴 로그아웃과 사용자의 exit 는 종료 코드로는 구분되지 않는다(둘 다 종료 코드와 함께 끝난다).
// 유휴 로그아웃은 입력이 한참 없을 때만 일어나므로, 끝나기 직전에 입력이 있었으면 사용자가 끝낸 것이다.
// 연결하자마자 끝나는 서버(로그인 직후 셸이 끝나는 계정 등)를 계속 다시 접속하지 않도록 그 경우도 여기에 넣는다.
func (pSession *Session) userEndedLocked(tNow time.Time) bool {
	if tNow.Sub(pSession.tConnected) < G_cdurUserExitWindow {
		return true
	}
	return !pSession.tLastInput.IsZero() && tNow.Sub(pSession.tLastInput) < G_cdurUserExitWindow
}

// onRemoteExited 는 서버가 셸을 종료 코드·신호와 함께 끝낸 것을 처리한다 (session.md §4).
//   - 사용자가 끝낸 것(exit·logout, userEndedLocked)이면 자동 재접속하지 않고 닫힘 + 안내 두 줄.
//   - 서버 쪽에서 끝낸 것(유휴 로그아웃, 관리자의 세션 종료, 서버 재시작)이면 끊김처럼 자동 재접속을 예약한다.
//
// Enter·[재연결] 로 다시 연결하는 동작은 그대로다. 남은 연결(SSH 접속·keepalive·포워딩)은 닫는다.
func (pSession *Session) onRemoteExited(iGen int) {
	pSession.mtxData.Lock()
	if iGen != pSession.gen || pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	conn := pSession.transport
	pSession.transport = nil
	pSession.writeOutputUnlockedNotice(MsgRemoteExited())
	bRetry := !pSession.userEndedLocked(time.Now()) &&
		CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
	if bRetry {
		pSession.scheduleRetryLocked()
	} else {
		pSession.attempts = 0
		pSession.info.M_szState = G_cszStateClosed
		pSession.info.M_szMessage = ""
		pSession.writeOutputUnlockedNotice(MsgGiveUp())
	}
	pSession.mtxData.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	pSession.publishState()
}

// readLoop 는 연결의 출력을 읽어 화면과 로그로 보낸다. 끝나면 끊김 처리한다.
func (pSession *Session) readLoop(conn Transport, iGen int) {
	bBuf := make([]byte, 32*1024)
	for {
		// 프런트가 아직 처리하지 않은 출력이 상한을 넘으면 여기서 멈춘다 (session.md §8 출력 흐름 제어).
		// 읽기를 멈추면 TCP 윈도우·PTY 파이프가 차서 상대 프로세스가 스스로 느려진다.
		pSession.credit.Wait()
		nRead, errRead := conn.Read(bBuf)
		if nRead > 0 && pSession.isCurrent(iGen) { // 닫힌 뒤 읽힌 조각은 버린다
			pSession.writeOutput(pSession.decoder.Decode(bBuf[:nRead]))
		}
		if errRead != nil {
			if errors.Is(errRead, transport.ErrRemoteExited) {
				pSession.onRemoteExited(iGen)
				return
			}
			szReason := ""
			if !errors.Is(errRead, io.EOF) {
				szReason = errRead.Error()
			}
			pSession.onDisconnected(szReason, iGen)
			return
		}
	}
}

```

#### C-A.4 `internal/session/control.go` — `Input`

```go
// Input 은 터미널 입력(UTF-8)을 서버 인코딩으로 바꿔 보낸다 (session.md §8).
// 연결이 없으면 버린다. 다만 끊긴 세션에서 CR(Enter)이 오면 즉시 재접속한다 (session.md §4 수동 재접속).
func (pSession *Session) Input(szText string) {
	pSession.mtxData.Lock()
	transport := pSession.transport
	szState := pSession.info.M_szState
	if transport != nil && szText != "" {
		pSession.tLastInput = time.Now() // 서버가 셸을 끝냈을 때 사용자가 끝낸 것인지 가리는 데 쓴다 (session.md §4)
	}
	pSession.mtxData.Unlock()
	if transport != nil {
		_ = transport.Write(Encode(pSession.cfg.Encoding, szText))
		return
	}
	if strings.Contains(szText, "\r") && szState != G_cszStateConnecting {
		pSession.Reconnect()
	}
}
```

#### C-A.5 `internal/session/exit_unlock_test.go` — 정상 종료 관련 테스트 4개와 도우미

```go
// exitingTransport 는 서버가 셸을 정상 종료한 것처럼 Read 에서 transport.ErrRemoteExited 를 돌려준다.
type exitingTransport struct {
	*fakeTransport
	chExit  chan struct{}
	iClosed atomic.Int32
}

func newExitingTransport() *exitingTransport {
	return &exitingTransport{fakeTransport: newFakeTransport(), chExit: make(chan struct{})}
}

func (pFake *exitingTransport) Read(bBuf []byte) (int, error) {
	select {
	case bData := <-pFake.chIn:
		return copy(bBuf, bData), nil
	case <-pFake.chExit:
		return 0, transport.ErrRemoteExited
	case <-pFake.chClosed:
		return 0, transport.ErrRemoteClosed
	}
}

func (pFake *exitingTransport) Close() error {
	pFake.iClosed.Add(1)
	pFake.drop()
	return nil
}

// 서버가 셸을 정상 종료하면 자동 재접속하지 않고 닫힘 + 안내 두 줄. Enter 로는 다시 연결한다 (session.md §4, 안 F).
func TestRemoteExitDoesNotAutoReconnect(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	second := newFakeTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return second, nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "Enter 키를 누르면 다시 연결합니다") }, time.Second)
	szOut := rec.output()
	if !strings.Contains(szOut, "서버가 세션을 끝냈습니다.") {
		t.Fatalf("정상 종료 안내가 없음: %q", szOut)
	}
	if strings.Contains(szOut, "자동 재연결") || strings.Contains(szOut, "연결이 끊어졌습니다") {
		t.Fatalf("정상 종료는 끊김·재접속 안내가 없어야 함: %q", szOut)
	}
	if first.iClosed.Load() == 0 {
		t.Fatal("남은 연결을 닫아야 함")
	}
	time.Sleep(2500 * time.Millisecond) // 끊김이었다면 2초 뒤 재접속했을 시간
	if pDial.calls() != 1 {
		t.Fatalf("자동 재접속하면 안 됨, 호출 %d회", pDial.calls())
	}
	if pSession.Snapshot().M_szMessage != "" || pSession.attemptsForTest() != 0 {
		t.Fatal("재시도 횟수·상태 메시지는 비어 있어야 함")
	}

	pSession.Input("\r")
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	if pDial.calls() != 2 {
		t.Fatalf("Enter 로 한 번 다시 접속해야 함, 호출 %d회", pDial.calls())
	}
}

// 입력 없이 한참 있다가 서버가 셸을 끝내면(유휴 로그아웃 등) 끊김처럼 자동 재접속한다 (session.md §4, D98).
func TestRemoteExitAfterIdleAutoReconnects(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.Input("ls\r")
	pSession.backdateForTest(time.Minute) // 연결도 마지막 입력도 1분 전

	close(first.chExit)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "초 후 자동 재연결합니다") }, time.Second)
	szOut := rec.output()
	if !strings.Contains(szOut, "서버가 세션을 끝냈습니다.") {
		t.Fatalf("서버 종료 안내가 없음: %q", szOut)
	}
	if first.iClosed.Load() == 0 {
		t.Fatal("남은 연결을 닫아야 함")
	}
	waitUntil(t, func() bool { return pDial.calls() == 2 && lastState(pSession) == G_cszStateConnected }, 4*time.Second)
}

// 끝나기 직전에 사용자 입력이 있었으면(exit·logout) 오래 연결된 세션이어도 자동 재접속하지 않는다 (session.md §4, D98).
func TestRemoteExitRightAfterInputStaysClosed(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
		if iCall == 1 {
			return first, nil
		}
		return newFakeTransport(), nil
	}}
	info, _ := pManager.Open(sshConfig(pDial.dial, true))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.backdateForTest(time.Minute)
	pSession.Input("exit\r") // 방금 입력

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	time.Sleep(2500 * time.Millisecond)
	if pDial.calls() != 1 {
		t.Fatalf("사용자가 끝낸 세션은 자동 재접속하면 안 됨, 호출 %d회", pDial.calls())
	}
	if strings.Contains(rec.output(), "자동 재연결") {
		t.Fatalf("재접속 안내가 없어야 함: %q", rec.output())
	}
}

// 자동 재접속 설정이 꺼져 있으면 유휴 뒤 서버 종료도 닫힘으로 둔다.
func TestRemoteExitAfterIdleWithAutoReconnectOff(t *testing.T) {
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	first := newExitingTransport()
	pDial := &countingDialer{fnNext: func(int) (Transport, error) { return first, nil }}
	info, _ := pManager.Open(sshConfig(pDial.dial, false))
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
	pSession.backdateForTest(time.Minute)

	close(first.chExit)
	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "Enter 키를 누르면 다시 연결합니다") }, time.Second)
}

// backdateForTest 는 연결 시각과 마지막 입력 시각(있으면)을 durAgo 만큼 과거로 옮긴다.
func (pSession *Session) backdateForTest(durAgo time.Duration) {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	pSession.tConnected = pSession.tConnected.Add(-durAgo)
	if !pSession.tLastInput.IsZero() {
		pSession.tLastInput = pSession.tLastInput.Add(-durAgo)
	}
}
```

---

## 부 D. "터미널 출력 기록"을 켜도 열려 있는 세션은 기록되지 않음 — 원인·수정·검증 — 5번 (D99)

**작성**: 2026-11-18 · **상태**: 코드 수정 완료(2026-11-18, D99), 앱 수동 검증은 남음(§7.2) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(발견 즉시 수정해 등록하지 않음)
**관련**: [spec/behavior/session.md](../../spec/behavior/session.md) §6, [spec/ui/settings.md](../../spec/ui/settings.md), [internal/session/control.go](../../internal/session/control.go), [internal/session/manager.go](../../internal/session/manager.go), [internal/session/session.go](../../internal/session/session.go), [internal/app/app.go](../../internal/app/app.go), 앞선 문서 [2026-11-16-log-filename-collision.md](./2026-11-16-log-filename-collision.md)(KI-001, D85), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 규모가 작아(Go 파일 4개 + 테스트 1개 + 문구·명세) WIP 는 만들지 않았다.

---

### D-0. 한 줄 요약

설정 "터미널 출력 기록"은 **세션을 열 때 한 번만** 읽혔다. 그래서 설정을 켜도 이미 열려 있던 세션은 기록되지 않았고, 로그 폴더에 파일이 생기지 않아 기능이 고장 난 것처럼 보였다. 반대로 설정을 꺼도 열려 있던 세션은 계속 기록됐다.

수정: 설정이 바뀌면 **열려 있는 세션에도 곧바로 적용**한다.

| 상황 | 전 | 후 (D99) |
|---|---|---|
| 꺼진 상태에서 켬, 열려 있는 세션 | 기록 안 함(파일 없음). 앱을 다시 켜거나 세션을 새로 열어야 기록 | **그 순간부터** 새 파일에 기록 |
| 켜진 상태에서 끔, 열려 있는 세션 | 세션을 닫을 때까지 계속 기록 | **기록을 멈추고 파일을 닫음** |
| 껐다가 다시 켬 | — | 새 파일(이전 파일에 이어 쓰지 않음) |
| 새로 여는 세션 | 그때의 설정대로 처음부터 기록 | 그대로 |
| 재접속(자동·Enter·[재연결]) | 같은 파일에 이어 씀 | 그대로 |

---

### D-1. 발견 경위

2026-11-18 사용자 보고: "터미널 출력 기록을 켰는데 `AppData\Roaming\DariTerm\data\logs` 폴더에 trace log 가 저장되지 않는다."

확인한 사실:

| 확인 | 결과 |
|---|---|
| `store.json` 의 설정 | `sessionLog: true`, 10:05:39 에 저장됨 |
| 실행 중인 앱 | `C:\Program Files\DariTerm\dariterm.exe`, 10:01:21 시작(설정을 켜기 전). 그 뒤 재시작 없음 |
| `logs` 폴더 | 비어 있고 수정 시각이 2026-11-17 08:45 그대로 → 설정을 켠 뒤 파일을 만들려는 시도 자체가 없었음. 폴더 권한은 정상 |
| 기록 기능 자체 | 임시 테스트(앱 서비스에 설정을 켜고 로컬 세션을 엶)에서 `logs\로컬 터미널-20261118-101500.log` 가 만들어짐 → 정상 |

사용자가 앱을 껐다 켜자 파일이 만들어졌다. 이어진 제안: "off 상태에서 on 을 하면 현재 창에 대해서 로그 파일을 새로 만들어야 하지 않을까?"

---

### D-2. 원인 분석 (Cause)

#### D-2.1 코드 경로 (수정 전)

```
세션 열기 (session.open)
  → app.Service.openSession              internal/app/channels_session.go:158
      cfg.LogEnabled = settingBool("sessionLog", false)   :185   ← 여는 순간의 설정값을 복사
  → session.Manager.Open → Session.start  internal/session/session.go:68
      if cfg.LogEnabled { OpenLog(...) }                   ← 여기서만 파일을 연다
  → Session.writeOutput                                    ← logger 가 있으면 쓴다
  → Session.Close                                          ← 여기서만 파일을 닫는다

설정 변경 (store.patch)
  → app.Service 의 store.patch 처리       internal/app/app.go:147
      저장 + 언어 적용 + (보관 기간이 바뀌면) 로그 정리   ← sessionLog 는 보지 않는다
```

#### D-2.2 왜 기록되지 않았는가

1. 설정값은 세션을 여는 순간 `cfg.LogEnabled` 로 **복사**되고, 로그 파일은 `start` 에서 한 번만 열린다.
2. `store.patch` 는 `sessionLog` 가 바뀌어도 열려 있는 세션에 아무것도 알리지 않았다.
3. 그래서 설정을 켜기 전에 열린 세션은 닫을 때까지 `logger` 가 없다. 재접속도 같은 세션이라 파일을 열지 않는다.

이는 버그라기보다 정해 둔 동작이었다. 명세 §6 이 "설정을 바꾸면 새 세션부터 적용"이었고, 설정 화면의 설명도 "(켠 뒤 새로 여는 연결부터)"였다. 다만 사용자는 켜면 바로 기록된다고 기대했고, 폴더가 비어 있어 고장으로 보였다.

#### D-2.3 같은 구조에서 나오는 반대쪽 문제

설정을 **꺼도** 열려 있던 세션은 닫을 때까지 계속 기록됐다([2026-11-16-log-filename-collision.md](./2026-11-16-log-filename-collision.md) §4.4 의 5번 "이미 열린 세션은 계속 기록"). 설정이 꺼져 있다고 알고 있는 동안 출력이 파일에 남는다.

---

### D-3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 켤 때 열려 있는 세션 | 그 시각의 이름으로 **새 파일**을 만들어 그때부터 기록 | 사용자 제안. 세션 하나 = 파일 하나 규칙(KI-001)과 이름 규칙을 그대로 쓴다 |
| 켜기 전 화면 내용 | 넣지 않는다 | 백엔드는 지난 출력을 보관하지 않는다(화면 내용은 프런트의 터미널에만 있다) |
| 끌 때 열려 있는 세션 | 기록을 멈추고 파일을 닫는다 | 요청 범위 밖이지만 권장안으로 함께 바꿨다. 꺼져 있는데 기록이 이어지면 사용자가 모르는 채 출력이 남는다 |
| 껐다가 다시 켤 때 | 새 파일 | 중간에 빠진 구간이 있는 파일을 이어 쓰면 읽는 사람이 헷갈린다 |
| 이미 기록 중인 세션에 "켜기"가 다시 옴 | 아무 일 없음 | 파일이 둘로 나뉘지 않게 |
| 닫힌 세션 | 건드리지 않음 | 닫은 뒤 빈 파일이 생기지 않게 |
| 동기화·백업 가져오기로 설정이 바뀐 경우 | 다루지 않는다(새 세션부터 적용) | 그 경로는 `store.patch` 를 거치지 않는다. 드문 경우라 범위에서 뺐다 |

---

### D-4. 수정 내용

#### D-4.1 `internal/session/control.go` — `Session.SetLogging` (신규)

위치 [control.go:109](../../internal/session/control.go#L109). import 에 `"os"` 를 더했다.

```go
// SetLogging 은 이 세션의 출력 기록을 켜거나 끈다 (session.md §6).
// 켜면 그 순간부터의 출력을 새 로그 파일에 쓴다(이미 기록 중이면 그대로). 끄면 남은 것을 쓰고 파일을 닫는다.
// 세션을 열 때(start)와 설정 "터미널 출력 기록"을 바꿀 때 부른다. 닫힌 세션은 건드리지 않는다.
func (pSession *Session) SetLogging(bEnabled bool) {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	if !bEnabled {
		pLogger := pSession.logger
		pSession.logger = nil
		pSession.mtxData.Unlock()
		if pLogger != nil {
			_ = pLogger.Close()
		}
		return
	}
	bSkip := pSession.logger != nil || pSession.cfg.DataDir == ""
	pSession.mtxData.Unlock()
	if bSkip {
		return
	}
	// 파일 만들기는 잠금 밖에서 한다(디스크를 기다리는 동안 출력이 막히지 않게).
	pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
	if errLog != nil {
		return // 열기 실패는 조용히 넘어간다: 기록 없이 세션은 계속 쓴다
	}
	pSession.mtxData.Lock()
	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
	if !bLost {
		pSession.logger = pLogger
	}
	pSession.mtxData.Unlock()
	if bLost {
		_ = pLogger.Close()
		_ = os.Remove(pLogger.Path()) // 방금 만든 빈 파일은 남기지 않는다
	}
}
```

짚어 둘 점:
- 파일을 만드는 동안에는 세션 잠금을 잡지 않는다. 느린 디스크에서도 그 세션의 출력이 멈추지 않는다.
- 잠금 밖에서 만든 뒤 다시 확인한다. 그사이 세션이 닫혔거나 다른 호출이 먼저 파일을 열었으면 방금 만든 파일을 닫고 지운다(빈 파일이 남지 않게).
- 끈 뒤 늦게 도착한 쓰기는 `Logger.Write` 가 닫힌 것을 보고 버린다(기존 동작).

#### D-4.2 `internal/session/session.go` — `start` 가 같은 길을 쓴다

```go
// before
	if pSession.cfg.LogEnabled && pSession.cfg.DataDir != "" {
		pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
		if errLog == nil {
			pSession.mtxData.Lock()
			pSession.logger = pLogger
			pSession.mtxData.Unlock()
		}
	}

// after
	if pSession.cfg.LogEnabled {
		pSession.SetLogging(true)
	}
```

파일을 여는 곳을 하나로 모았다. 세션을 여는 순간에 설정 변경이 겹쳐도 파일이 두 개 열리지 않는다.

#### D-4.3 `internal/session/manager.go` — `Manager.SetLogging` (신규)

위치 [manager.go:94](../../internal/session/manager.go#L94).

```go
// SetLogging 은 열려 있는 모든 세션의 출력 기록을 켜거나 끈다 (설정 "터미널 출력 기록"을 바꿨을 때, session.md §6).
func (pManager *Manager) SetLogging(bEnabled bool) {
	pManager.mtxData.Lock()
	slSessions := make([]*Session, 0, len(pManager.mapSessions))
	for _, pSession := range pManager.mapSessions {
		slSessions = append(slSessions, pSession)
	}
	pManager.mtxData.Unlock()
	for _, pSession := range slSessions {
		pSession.SetLogging(bEnabled)
	}
}
```

#### D-4.4 `internal/app/app.go` — `store.patch` 에서 값이 바뀌면 적용

위치 [app.go:147-165](../../internal/app/app.go#L147-L165).

```go
// before
		iKeepBefore := pService.logKeepDays()
		errPatch := pService.pStore.Patch(mapPatch)
		pService.applyLanguage()

// after
		iKeepBefore := pService.logKeepDays()
		bLogBefore := pService.settingBool("sessionLog", false)
		errPatch := pService.pStore.Patch(mapPatch)
		pService.applyLanguage()
		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
			// "터미널 출력 기록"을 바꾸면 열려 있는 세션에도 곧바로 적용한다 (session.md §6, D99).
			pService.pSessions.SetLogging(bLogNow)
		}
```

보관 기간(`sessionLogKeepDays`)이 바뀌면 곧바로 정리하는 기존 처리와 같은 방식이다. 값이 실제로 바뀐 경우에만 부르므로 다른 설정을 저장할 때는 아무 일도 없다.

#### D-4.5 문구 — `locales/*.json` `conn.sessionLogDesc` (10개 언어)

| 언어 | 전 | 후 |
|---|---|---|
| ko | 터미널 출력을 텍스트 파일로 남깁니다 (켠 뒤 새로 여는 연결부터) | 터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록) |
| en | Saves terminal output to text files (from connections opened after turning this on) | Saves terminal output to text files (open connections start recording when you turn this on) |

나머지 8개 언어(ja, zh, tw, de, fr, es, pt, ru)도 같은 뜻으로 바꿨다. 한국어·영어 외에는 이 수정을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

#### D-4.6 명세·문서

| 파일 | 내용 |
|---|---|
| `spec/behavior/session.md` §6 | "설정을 바꾸면 새 세션부터 적용" → "설정을 바꾸면 열려 있는 세션에도 곧바로 적용"(켜기·끄기·다시 켜기 규칙, 2026-11-18 표기) |
| `spec/ui/settings.md` | 세션 로그 저장 항목의 설명 문구 |
| `docs/scenarios.md`, `docs/scenarios.en.md` 6-11 | 앱 동작 설명 |
| `docs/manual/DariTerm-user-manual.ko.html`, `.en.html` | 같은 행 |
| `docs/decisions.md` | **D99** |

---

### D-5. 사용자가 보는 모습

- 탭 두 개가 열려 있는 상태에서 설정 > 연결 > "터미널 출력 기록"을 켠다 → `logs` 폴더에 `{별칭}-{켠 시각}.log` 가 탭마다 하나씩 바로 생긴다. 같은 별칭이면 두 번째는 `-2` 가 붙는다.
- 파일 내용은 약 1초 단위로 모아서 쓴다. 켠 직후 출력이 없으면 파일은 0바이트다.
- 끄면 파일이 닫혀 다른 프로그램에서 옮기거나 지울 수 있다.
- 터미널 화면에는 기록 시작·중지 안내를 따로 내지 않는다.

---

### D-6. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 설정을 끈 사용자 | 열려 있는 세션의 기록이 그 자리에서 멈춘다(전에는 닫을 때까지 이어짐). 의도한 변화 | 설명서·시나리오에 "끄면 기록을 멈춤" 반영 |
| 세션이 많을 때 켜기 | 세션 수만큼 파일을 만든다. 세션마다 잠금 밖에서 차례로 만든다 | 파일 만들기는 가볍다. `store.patch` 응답이 그만큼 늦어질 수 있다 |
| 켜기와 세션 열기·닫기가 겹침 | 파일이 둘 열리거나 닫힌 세션에 빈 파일이 남을 수 있었다 | `SetLogging` 의 재확인 + 빈 파일 삭제, `start` 도 같은 함수 사용 |
| 오래된 로그 정리(`logclean`) | 기록 중인 파일 목록(`IsLogOpen`)은 `OpenLog`/`Close` 가 관리하므로 그대로 맞다 | `TestSetLoggingOnOpenSession` 이 열림·닫힘 등록을 확인 |
| 파일 열기 실패 | 전과 같이 조용히 넘어간다(기록 없이 세션 사용). 사용자는 실패를 알 수 없다 | 이번 범위 밖. §8 |
| 동기화·백업 가져오기로 설정이 바뀜 | 열려 있는 세션에는 적용되지 않는다 | §3 에 적은 한계 |
| 프런트엔드·IPC 계약 | 변경 없음(문구만) | — |

---

### D-7. 검증

#### D-7.1 자동 (2026-11-18 실행)

[internal/session/log_test.go](../../internal/session/log_test.go) 에 테스트 2개를 더했다.

| 테스트 | 확인하는 것 |
|---|---|
| `TestSetLoggingOnOpenSession` | 꺼진 채로 연 세션 → 출력이 와도 파일 없음 → 켜면 파일 1개(두 번 켜도 1개), 기록 중인 파일로 등록 → 그 뒤 출력이 기록됨 → 끄면 등록 해제 → 그 뒤 출력은 기록되지 않음 → 파일에는 켜져 있던 동안의 출력만 있음 → 다시 켜면 파일이 하나 더 생김 |
| `TestSetLoggingIgnoresClosedSession` | 닫힌 세션에 켜기를 불러도 파일이 생기지 않음 |
| `TestSessionLogIsPerSession` (기존) | 켜진 채로 여는 세션은 전과 같이 세션마다 파일 하나 (`start` → `SetLogging` 경로) |

```bash
go build ./...
go test ./...                       # 전부 통과
(cd frontend && npx vitest run)     # 233개 통과 (문구 파일 변경 확인)
```

- `go test ./...` 첫 실행에서 `internal/clipboard` 의 `TestClipboardInfoIsFastForText` 가 한 번 실패했다. 실제 시스템 클립보드를 쓰는 테스트이고, 다시 두 번 돌려 모두 통과했다. 이번 수정과는 무관하다.
- `go test -race` 는 이 PC 에 cgo 컴파일러가 없어 실행하지 못했다.
- `store.patch` → `Manager.SetLogging` 연결(§4.4)을 직접 확인하는 테스트는 없다. §7.2 의 1번으로 확인한다.

#### D-7.2 수동 (`build/bin/dariterm.exe`, 2026-11-18 10:21 빌드) — 아직 하지 않음

설치본(`C:\Program Files\DariTerm`)은 바꾸지 않았다. 빌드 방법은 이 문서 부 A(2026-11-18-pane-find-button.md) §10.

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정을 끈 채로 탭 2개를 연 뒤 "터미널 출력 기록"을 켠다 | `%APPDATA%\DariTerm\data\logs` 에 파일 2개가 바로 생긴다 |
| 2 | 각 탭에서 `echo tab1` / `echo tab2` | 약 1초 뒤 각 파일에 자기 줄만 있다. 켜기 전에 화면에 있던 내용은 없다 |
| 3 | 설정을 끈 뒤 `echo off` | 파일이 더 커지지 않는다. 파일을 다른 프로그램에서 지우거나 옮길 수 있다 |
| 4 | 다시 켠다 | 탭마다 새 파일이 하나씩 더 생긴다 |
| 5 | 켠 상태에서 새 탭 | 전과 같이 파일 하나가 생기고 처음부터 기록된다 |
| 6 | 켠 상태에서 세션을 끊었다가 재접속 | 새 파일 없이 같은 파일에 이어 쓴다 |
| 7 | 다른 설정(글자 크기 등)을 바꾼다 | 로그 파일이 새로 생기지 않는다 |
| 8 | 설정 화면의 설명 문구 | "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)" |

---

### D-8. 남은 정리 거리 (이번에 하지 않음)

| 항목 | 비고 |
|---|---|
| 로그 파일 열기 실패를 사용자에게 알리기 | 지금은 조용히 무시한다. 폴더 권한·디스크 문제일 때 "켰는데 안 남는다"를 다시 겪게 된다 |
| 동기화·백업 가져오기로 설정이 바뀐 경우의 즉시 적용 | 설정을 통째로 바꾸는 경로에서도 같은 비교를 하면 된다 |
| `store.patch` 경로 테스트 | 앱 서비스에 로컬 세션을 열고 설정을 켜서 파일이 생기는지 확인하는 테스트 |
| 8개 언어 문구 검수 | §4.5 |

---

### D-9. 체크리스트

- [x] 1. 원인 확인 (§1, §2)
- [x] 2. `control.go` `Session.SetLogging`, `session.go` `start`, `manager.go` `Manager.SetLogging` (§4.1~4.3)
- [x] 3. `app.go` `store.patch` 에서 값이 바뀌면 적용 (§4.4)
- [x] 4. 문구 10개 언어, 명세 §6·ui/settings.md, 시나리오·설명서, decisions D99 (§4.5, §4.6)
- [x] 5. 테스트 2개 추가, `go build ./... && go test ./...`, 프런트 테스트 통과 (§7.1)
- [x] 6. `wails build`
- [ ] 7. 수동 검증 §7.2 (1·2·3 필수)
- [ ] 8. (선택) §8 의 정리 거리

---

### D-부록 A. 최종 코드 (2026-11-18 작업 종료 시점의 실제 파일 내용)

§4.1 의 `SetLogging` 은 같은 날 뒤이은 수정(D100, 이 문서 부 E(2026-11-18-session-log-sync-interval.md))에서 로그를 등록하기 전에 내려 쓰기 주기를 전하는 한 줄(`pLogger.SetSyncSec(...)`)이 더해졌다. 아래가 최종 내용이다.

#### D-A.1 `internal/session/control.go` — `SetLogging`

```go
// SetLogging 은 이 세션의 출력 기록을 켜거나 끈다 (session.md §6).
// 켜면 그 순간부터의 출력을 새 로그 파일에 쓴다(이미 기록 중이면 그대로). 끄면 남은 것을 쓰고 파일을 닫는다.
// 세션을 열 때(start)와 설정 "터미널 출력 기록"을 바꿀 때 부른다. 닫힌 세션은 건드리지 않는다.
func (pSession *Session) SetLogging(bEnabled bool) {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	if !bEnabled {
		pLogger := pSession.logger
		pSession.logger = nil
		pSession.mtxData.Unlock()
		if pLogger != nil {
			_ = pLogger.Close()
		}
		return
	}
	bSkip := pSession.logger != nil || pSession.cfg.DataDir == ""
	pSession.mtxData.Unlock()
	if bSkip {
		return
	}
	// 파일 만들기는 잠금 밖에서 한다(디스크를 기다리는 동안 출력이 막히지 않게).
	pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
	if errLog != nil {
		return // 열기 실패는 조용히 넘어간다: 기록 없이 세션은 계속 쓴다
	}
	pSession.mtxData.Lock()
	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
	if !bLost {
		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
		pSession.logger = pLogger
	}
	pSession.mtxData.Unlock()
	if bLost {
		_ = pLogger.Close()
		_ = os.Remove(pLogger.Path()) // 방금 만든 빈 파일은 남기지 않는다
	}
}
```

#### D-A.2 `internal/session/session.go` — `start`

```go
// start 는 OpenDelay 뒤에 첫 접속을 시작한다. 로그가 켜져 있으면 연다.
func (pSession *Session) start() {
	if pSession.cfg.LogEnabled {
		pSession.SetLogging(true)
	}
	durDelay := pSession.cfg.OpenDelay
	if durDelay <= 0 {
		durDelay = G_cdurDefaultOpenDelay
	}
	time.AfterFunc(durDelay, pSession.connect)
}
```

#### D-A.3 `internal/session/manager.go` — `Manager.SetLogging`

```go
// SetLogging 은 열려 있는 모든 세션의 출력 기록을 켜거나 끈다 (설정 "터미널 출력 기록"을 바꿨을 때, session.md §6).
func (pManager *Manager) SetLogging(bEnabled bool) {
	pManager.mtxData.Lock()
	slSessions := make([]*Session, 0, len(pManager.mapSessions))
	for _, pSession := range pManager.mapSessions {
		slSessions = append(slSessions, pSession)
	}
	pManager.mtxData.Unlock()
	for _, pSession := range slSessions {
		pSession.SetLogging(bEnabled)
	}
}
```

#### D-A.4 `internal/app/app.go` — `store.patch` 처리 (D99 의 기록 켜기/끄기와 D100 의 주기 적용 포함)

```go
	pService.pRouter.Handle("store.patch", func(slArgs []json.RawMessage) (any, error) {
		var mapPatch map[string]any
		if errArg := argInto(slArgs, 0, &mapPatch); errArg != nil {
			return nil, errArg
		}
		iKeepBefore := pService.logKeepDays()
		bLogBefore := pService.settingBool("sessionLog", false)
		iSyncBefore := pService.logSyncSec()
		errPatch := pService.pStore.Patch(mapPatch)
		pService.applyLanguage()
		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
			// "터미널 출력 기록"을 바꾸면 열려 있는 세션에도 곧바로 적용한다 (session.md §6, D99).
			pService.pSessions.SetLogging(bLogNow)
		}
		if iSyncNow := pService.logSyncSec(); iSyncNow != iSyncBefore {
			// 내려 쓰기 주기도 열려 있는 세션에 곧바로 적용한다 (session.md §6, D100).
			pService.pSessions.SetLogSyncSec(iSyncNow)
		}
		if pService.logKeepDays() != iKeepBefore {
			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
		}
		return nil, errPatch
	})
```

#### D-A.5 `internal/session/log_test.go` — 추가한 테스트 2개

```go
// 열려 있는 세션에서 기록을 켜면 그때부터 새 파일에 쓰고, 끄면 파일을 닫고 더 쓰지 않는다 (session.md §6, D99).
func TestSetLoggingOnOpenSession(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web" // LogEnabled 는 꺼진 채로 연다
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	fnFiles := func() []string {
		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
		return slFiles
	}

	pFake.chIn <- []byte("before\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "before") }, time.Second)
	if len(fnFiles()) != 0 {
		t.Fatalf("꺼져 있으면 파일이 없어야 함: %v", fnFiles())
	}

	pManager.SetLogging(true)
	pManager.SetLogging(true) // 두 번 켜도 파일은 하나
	if len(fnFiles()) != 1 {
		t.Fatalf("켜면 파일 하나가 생겨야 함: %v", fnFiles())
	}
	szPath := fnFiles()[0]
	if !IsLogOpen(szPath) {
		t.Fatal("기록 중인 파일로 등록돼야 함")
	}
	pFake.chIn <- []byte("during\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "during") }, time.Second)

	pManager.SetLogging(false)
	if IsLogOpen(szPath) {
		t.Fatal("끄면 파일을 닫아야 함")
	}
	pFake.chIn <- []byte("after\r\n")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "after") }, time.Second)

	bData, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	szLog := string(bData)
	if !strings.Contains(szLog, "during") || strings.Contains(szLog, "before") || strings.Contains(szLog, "after") {
		t.Fatalf("켜져 있던 동안의 출력만 있어야 함: %q", szLog)
	}

	pManager.SetLogging(true) // 다시 켜면 새 파일
	if len(fnFiles()) != 2 {
		t.Fatalf("다시 켜면 파일이 하나 더 생겨야 함: %v", fnFiles())
	}
}

// 닫힌 세션은 기록을 켜도 파일을 만들지 않는다.
func TestSetLoggingIgnoresClosedSession(t *testing.T) {
	szDir := t.TempDir()
	cfg := sshConfig(func(func(string)) (Transport, error) { return newFakeTransport(), nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	pSession := newSession(cfg, func(string, ...any) {})
	pSession.start()
	pSession.Close()
	pSession.SetLogging(true)
	if slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "*")); len(slFiles) != 0 {
		t.Fatalf("닫힌 세션은 파일을 만들지 않음: %v", slFiles)
	}
}

```

#### D-A.6 `locales/*.json` — `conn.sessionLogDesc` (10개 언어)

```json
  "conn.sessionLogDesc": "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)",
  "conn.sessionLogDesc": "Saves terminal output to text files (open connections start recording when you turn this on)",
  "conn.sessionLogDesc": "ターミナル出力をテキストファイルに残します（オンにすると開いている接続もその時点から記録）",
  "conn.sessionLogDesc": "将终端输出保存为文本文件（开启后，已打开的连接也从此刻开始记录）",
  "conn.sessionLogDesc": "將終端機輸出存成文字檔（開啟後，已開啟的連線也從此刻開始記錄）",
  "conn.sessionLogDesc": "Speichert die Terminalausgabe als Textdatei (offene Verbindungen werden ab dem Einschalten mit aufgezeichnet)",
  "conn.sessionLogDesc": "Enregistre la sortie du terminal dans des fichiers texte (les connexions ouvertes sont enregistrées dès l'activation)",
  "conn.sessionLogDesc": "Guarda la salida del terminal en archivos de texto (las conexiones abiertas se registran desde que se activa)",
  "conn.sessionLogDesc": "Salva a saída do terminal em arquivos de texto (as conexões abertas passam a ser registradas ao ativar)",
  "conn.sessionLogDesc": "Сохраняет вывод терминала в текстовые файлы (открытые подключения записываются с момента включения)",
```

---

## 부 E. 기록 중인 세션 로그가 탐색기에 0바이트로 보임 → 내려 쓰기 주기 설정 `sessionLogSyncSec` — 원인·수정·코드·검증 — 6번 (D100)

**작성**: 2026-11-18 · **상태**: 완료(2026-11-18, D100). 사용자가 앱에서 정상 동작을 확인함(§9.3) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음(발견 즉시 수정해 등록하지 않음)
**관련**: [spec/behavior/session.md](../../spec/behavior/session.md) §6, [spec/data/store.md](../../spec/data/store.md), [spec/ui/settings.md](../../spec/ui/settings.md), [internal/session/log.go](../../internal/session/log.go), [internal/session/control.go](../../internal/session/control.go), [internal/app/app.go](../../internal/app/app.go), 앞선 문서 이 문서 부 D(2026-11-18-session-log-toggle-open-sessions.md)(D99), [2026-11-16-session-transport-issues.md](./2026-11-16-session-transport-issues.md)(KI-004, 로그 버퍼·1초 쓰기 D88), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 작업 기록(WIP)은 `.claude/wip/archive/2026-11-18-session-log-sync-interval.md` 에 있다.

---

### E-0. 한 줄 요약

터미널 출력 기록은 정상으로 저장되고 있었는데, **탐색기에는 파일 크기가 0, 수정 시각이 만든 시각 그대로** 보여서 저장이 안 되는 것처럼 보였다. Windows(NTFS)는 프로그램이 파일을 열어 둔 동안 폴더 목록의 크기·수정 시각을 갱신하지 않고, **디스크까지 내려 쓰거나(fsync) 닫을 때**만 반영한다. 앱은 세션 동안 로그 파일을 계속 열어 두고 파일에 쓰기만 했다.

수정: 기록 중인 로그를 주기적으로 **내려 쓴다.** 주기는 새 설정 `sessionLogSyncSec`(설정 > 연결 > "로그 파일 갱신 주기 (초)")로 정한다.

| 값 | 파일에 쓰기 | 디스크까지 내려 쓰기 | 탐색기에 크기가 보이는 시점 |
|---|---|---|---|
| **0** | 출력을 받는 즉시(모으지 않음) | 쓸 때마다 바로 | 곧바로 |
| **1~5** | 약 1초마다(전과 같음) | 설정한 초마다 한 번, 새로 쓴 내용이 있을 때만 | 늦어도 (설정 초 + 1)초 뒤 |
| (수정 전) | 약 1초마다 | 하지 않음 | 세션을 닫을 때 |

기본값 2, 범위 0~5. 값을 바꾸면 기록 중인 세션에도 곧바로 적용된다.

---

### E-1. 발견 경위

2026-11-18 사용자 보고: "프로그램에서 포커스를 잃으면 로그 파일 저장 안 되나? 처음에 잠깐 되다가 dariterm 이 백그라운드로 갔다 온 후로 로그가 저장이 안 되어 있네?"

#### E-1.1 확인한 사실 (10:39~10:40, 앱은 10:23:07 에 시작해 실행 중)

`%APPDATA%\DariTerm\data\logs` 의 파일을 폴더 목록으로 본 값과, 파일을 직접 열어 잰 값을 비교했다.

| 파일 | 폴더 목록의 크기 | 폴더 목록의 수정 시각 | 실제 크기 | 실제 마지막 기록 |
|---|---|---|---|---|
| `DC1_MSA01(213)-20261118-102313.log` | 0 | 10:23:13 | 131,417,790 | 10:33:55 |
| `loading-tester1(200)-20261118-102313.log` | 0 | 10:23:13 | 5,942,932 | 10:33:56 |
| `DC1_MSA01(213)-20261118-102313-2.log` | 0 | 10:23:13 | 411 | 10:38:17 |
| `mds-컴파일-20261118-102313.log` | 0 | 10:23:13 | 665 | 10:24:29 |
| `로컬 터미널-20261118-102313.log` | 0 | 10:23:13 | 134 | 10:24:23 |

- 기록은 되고 있었다(131MB). 폴더 목록만 0바이트였다.
- 앞선 실행(10:15:45 에 만든 파일들)은 크기가 보였다. 그 세션들이 닫혀서 반영된 것이다. 이것이 "처음에 잠깐 되다가"로 보인 이유다.
- 큰 파일 둘이 같은 초(10:33:56)에 멈춘 것은 부하 테스트가 끝나 서버 출력이 멈춘 것으로 보았다(로그 마지막 줄 시각 `01:33:56.926Z` = 한국 시각 10:33:56). 이 부분은 파일만 보고 한 추정이다.

실제 크기를 잰 방법(다른 프로그램이 쓰고 있는 파일을 읽기 공유로 연다):

```powershell
$fs = [IO.File]::Open($path, 'Open', 'Read', 'ReadWrite,Delete'); $fs.Length; $fs.Close()
```

#### E-1.2 포커스와는 무관

로그는 백엔드(Go)가 서버에서 읽는 즉시 쓴다([session.go](../../internal/session/session.go) `writeOutput` → `Logger.Write`). 창이 앞에 있는지는 보지 않는다. 창이 뒤에 있으면 화면 처리가 느려져 백엔드가 읽는 속도를 늦출 수는 있지만(출력 흐름 제어, session.md §8), 읽은 것은 빠짐없이 기록된다.

---

### E-2. 원인 분석 (Cause)

#### E-2.1 코드 경로 (수정 전)

```
Session.writeOutput                    internal/session/session.go
  → Logger.Write                       internal/session/log.go
      bufio.Writer(64KB) 에 쌓고, 1초 타이머가 없으면 건다
  → Logger.flushTimer (1초 뒤)
      pWriter.Flush()                  ← 파일에 쓰기(WriteFile)까지만 한다
  → Logger.Close (세션을 닫을 때)
      Flush + pFile.Close()            ← 여기서야 폴더 목록이 갱신된다
```

`Flush` 는 프로그램 버퍼를 운영체제에 넘기는 것까지다. 파일 내용은 이때부터 다른 프로그램이 읽을 수 있다. 그러나 **디스크까지 내려 쓰기(`FlushFileBuffers`, Go 의 `(*os.File).Sync`)는 한 번도 부르지 않았다.**

#### E-2.2 왜 0바이트로 보이는가

NTFS 는 파일 크기·수정 시각을 두 군데에 둔다. 파일 자체의 기록과, 그 파일이 든 **폴더의 목록 항목**이다. 탐색기와 `dir` 은 폴더 목록 항목을 읽는다. 열려 있는 파일에 쓰기만 하면 폴더 목록 항목은 갱신되지 않고, 내려 쓰거나 핸들을 닫을 때 갱신된다.

#### E-2.3 이 PC 에서의 실험 (2026-11-18)

임시 폴더에 파일을 만들어 한 핸들로 쓰면서 폴더 목록의 크기를 봤다.

```powershell
$fs = [IO.File]::Open($p, 'CreateNew', 'Write', 'ReadWrite,Delete'); $b = New-Object byte[] 100000
$fs.Write($b, 0, $b.Length); $fs.Flush($false)   # 쓰기만
$fs.Flush($true)                                  # FlushFileBuffers
```

| 단계 | 폴더 목록의 크기 |
|---|---|
| 100KB 쓰기만 함 | 0 |
| 내려 쓰기 호출 | 100,000 |
| 3초 기다림 | 100,000 |
| 다른 핸들로 열었다 닫음 | 100,000 |
| 100KB 더 쓰고 내려 쓰기 호출 | 200,000 |
| 쓰던 핸들을 닫음 | 200,000 |

결론: **파일을 닫지 않아도 내려 쓰기만 하면 폴더 목록이 갱신된다.**

---

### E-3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 방식 | 파일을 연 채로 주기적으로 내려 쓴다 | §2.3 으로 확인. 닫았다 다시 여는 안은 그 틈에 다른 프로그램이 파일을 잡거나 지울 수 있어 기각 |
| 주기 | 설정으로 뺀다: 0~5초, 기본 2 | 사용자 지정 |
| 0 의 뜻 | 받는 즉시 파일에 쓰고 내려 쓴다 | 사용자 지정("부하가 걸리더라도 바로바로") |
| 1~5 의 뜻 | 파일 쓰기는 1초 그대로, 내려 쓰기는 설정한 초마다 | 사용자 지정 |
| 내려 쓸 조건 | 마지막 내려 쓰기 뒤에 새로 쓴 내용이 있을 때만 | 조용한 세션에서 쓸데없는 디스크 동기화를 하지 않는다 |
| 출력이 멈춘 뒤 | 남은 내용이 있으면 주기가 됐을 때 한 번 내려 쓰고 타이머를 멈춘다 | 마지막 출력이 영영 0바이트로 남지 않게 |
| 1~5 에서 내려 쓰기 위치 | 로그 잠금 **밖** | 디스크를 기다리는 동안 그 세션의 출력 기록이 막히지 않는다(D88 의 취지) |
| 0 에서 내려 쓰기 위치 | 로그 잠금 **안**, 쓰는 자리에서 바로 | 순서가 단순하고 "받는 즉시"를 그대로 지킨다. 그만큼 출력이 느려질 수 있다(사용자가 고른 동작) |
| 설정 변경 | 기록 중인 세션에도 곧바로 적용 | "터미널 출력 기록" 켜기/끄기(D99)와 같은 방식 |
| 범위 밖 값 | 0 또는 5 로 맞춘다 | store.json 을 손으로 고친 경우 대비 |

**타이밍 (1~5)**: 쓰기 타이머는 1초다. 타이머가 돌 때마다 버퍼를 파일에 쓰고, 마지막 내려 쓰기로부터 설정한 초가 지났으면 내려 쓴다. 아직이면 1초 뒤 다시 돈다. 그래서 내려 쓰기는 많아야 설정한 초에 한 번이고, 어떤 출력이든 늦어도 (설정 초 + 1)초 안에 탐색기에 반영된다.

---

### E-4. 수정 내용 — 백엔드

#### E-4.1 `internal/session/log.go` — 상수와 범위 맞춤

```go
const (
	…
	// G_cdurLogFlush 는 버퍼에 쌓인 로그를 파일에 쓰는 주기다 (내려 쓰기 주기가 0 이 아닐 때).
	G_cdurLogFlush = time.Second
	// 내려 쓰기(파일을 연 채로 디스크까지 쓰기, fsync) 주기의 범위와 기본값(초)이다 (설정 sessionLogSyncSec, session.md §6).
	// 0 이면 출력을 받는 즉시 파일에 쓰고 내려 쓴다. Windows 탐색기는 내려 쓰거나 닫아야 크기·수정 시각을 갱신한다.
	G_ciLogSyncSecMin     = 0
	G_ciLogSyncSecMax     = 5
	G_ciLogSyncSecDefault = 2
	…
)

// ClampLogSyncSec 는 내려 쓰기 주기를 허용 범위(0~5초)로 맞춘다.
func ClampLogSyncSec(iSec int) int {
	if iSec < G_ciLogSyncSecMin {
		return G_ciLogSyncSecMin
	}
	if iSec > G_ciLogSyncSecMax {
		return G_ciLogSyncSecMax
	}
	return iSec
}
```

#### E-4.2 `log.go` — `Logger` 필드

```go
// before
type Logger struct {
	mtxData    sync.Mutex
	szPath     string // 열린 파일의 절대 경로 (열린 로그 집합의 키)
	pFile      *os.File
	pWriter    *bufio.Writer
	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
	timerFlush *time.Timer // 버퍼에 내용이 있을 때만 걸린다
}

// after
type Logger struct {
	mtxData    sync.Mutex
	szPath     string // 열린 파일의 절대 경로 (열린 로그 집합의 키)
	pFile      *os.File
	pWriter    *bufio.Writer
	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
	timerFlush *time.Timer // 버퍼에 내용이 있거나 아직 내려 쓰지 않은 내용이 있을 때만 걸린다
	iSyncSec   int         // 내려 쓰기 주기(초). 0 = 즉시
	bUnsynced  bool        // 파일에는 썼지만 아직 내려 쓰지 않은 내용이 있다
	tLastSync  time.Time   // 마지막으로 내려 쓴 시각 (처음에는 연 시각)
}
```

`OpenLog` 는 서명을 바꾸지 않고, 만든 `Logger` 에 기본 주기와 연 시각을 넣는다([log.go:133-134](../../internal/session/log.go#L133-L134)).

```go
// before
return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes)}, nil

// after
return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes),
	iSyncSec: G_ciLogSyncSecDefault, tLastSync: time.Now()}, nil
```

#### E-4.3 `log.go` — `Write`, `syncNowLocked`, `SetSyncSec`, `flushTimer`

현재 위치 [log.go:176-249](../../internal/session/log.go#L176-L249).

```go
// before
func (pLogger *Logger) Write(szText string) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pWriter == nil {
		return
	}
	szBody, szTail := splitAnsiTail(pLogger.szCarry + szText)
	pLogger.szCarry = szTail
	if szBody == "" {
		return
	}
	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
	if pLogger.timerFlush == nil {
		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
	}
}

// flushTimer 는 주기 flush 다. 닫힌 뒤에는 아무 일도 하지 않는다.
func (pLogger *Logger) flushTimer() {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	pLogger.timerFlush = nil
	if pLogger.pWriter != nil {
		_ = pLogger.pWriter.Flush()
	}
}
```

```go
// after
// Write 는 출력 조각을 정리해 버퍼에 쓴다. 끝이 미완성 시퀀스면 보관했다가 다음 조각과 이어 처리한다.
func (pLogger *Logger) Write(szText string) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pWriter == nil {
		return
	}
	szBody, szTail := splitAnsiTail(pLogger.szCarry + szText)
	pLogger.szCarry = szTail
	if szBody == "" {
		return
	}
	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
	if pLogger.iSyncSec == 0 {
		// 즉시 모드: 부하가 걸리더라도 받는 대로 파일에 쓰고 내려 쓴다 (사용자가 고른 동작).
		pLogger.syncNowLocked()
		return
	}
	if pLogger.timerFlush == nil {
		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
	}
}

// syncNowLocked 는 mtxData 를 잡은 상태에서 버퍼를 파일에 쓰고 곧바로 내려 쓴다.
func (pLogger *Logger) syncNowLocked() {
	if pLogger.pWriter == nil {
		return
	}
	_ = pLogger.pWriter.Flush()
	_ = pLogger.pFile.Sync()
	pLogger.bUnsynced = false
	pLogger.tLastSync = time.Now()
}

// SetSyncSec 은 내려 쓰기 주기(초, 0~5)를 바꾼다. 0 으로 바꾸면 쌓인 것을 곧바로 쓰고 내려 쓴다.
func (pLogger *Logger) SetSyncSec(iSec int) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	pLogger.iSyncSec = ClampLogSyncSec(iSec)
	if pLogger.iSyncSec == 0 {
		pLogger.syncNowLocked()
	}
}

// flushTimer 는 주기 처리다: 버퍼를 파일에 쓰고, 내려 쓰기 주기가 됐으면 내려 쓴다.
// 아직 내려 쓰지 않은 내용이 남아 있으면 다음 주기에 다시 돈다. 닫힌 뒤에는 아무 일도 하지 않는다.
func (pLogger *Logger) flushTimer() {
	pLogger.mtxData.Lock()
	pLogger.timerFlush = nil
	if pLogger.pWriter == nil {
		pLogger.mtxData.Unlock()
		return
	}
	if pLogger.pWriter.Buffered() > 0 {
		_ = pLogger.pWriter.Flush()
		pLogger.bUnsynced = true
	}
	var pSync *os.File
	if pLogger.bUnsynced {
		if time.Since(pLogger.tLastSync) >= time.Duration(pLogger.iSyncSec)*time.Second {
			pSync = pLogger.pFile
			pLogger.bUnsynced = false
			pLogger.tLastSync = time.Now()
		} else {
			pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
		}
	}
	pLogger.mtxData.Unlock()
	if pSync != nil {
		// 내려 쓰기는 잠금 밖에서 한다: 디스크를 기다리는 동안 출력 기록(Write)이 막히지 않는다.
		// 그사이 닫혔으면 오류가 날 뿐이다(닫기가 이미 다 썼다).
		_ = pSync.Sync()
	}
}
```

읽는 법:

- **0 모드**(`Write` 안): `WriteString` 뒤에 곧바로 `Flush` + `Sync`. 타이머를 걸지 않는다. 미완성 ANSI 시퀀스 꼬리(`szCarry`)는 전과 같이 다음 조각을 기다린다.
- **1~5 모드**: `Write` 는 타이머만 건다. `flushTimer` 가 1초 뒤에 돈다.
  - 버퍼에 내용이 있으면 파일에 쓰고 `bUnsynced` 를 켠다.
  - `bUnsynced` 이고 마지막 내려 쓰기에서 주기가 지났으면, 잠금을 푼 뒤 `Sync` 한다.
  - `bUnsynced` 인데 주기가 안 됐으면 타이머를 다시 건다. 출력이 멈춰도 이 타이머가 주기까지 돌다가 내려 쓰고 멈춘다.
  - 타이머가 도는 중에 값이 0 으로 바뀌었으면 `time.Duration(0)` 과 비교하므로 곧바로 내려 쓴다.
- **잠금 밖 `Sync` 와 `Close` 의 겹침**: `Close` 가 먼저 파일을 닫으면 `Sync` 는 오류를 돌려줄 뿐이다(`*os.File` 은 동시에 불러도 안전하다). `Close` 는 남은 것을 다 쓰고 닫으므로 잃는 내용은 없다.
- `Close` 는 바꾸지 않았다. 닫으면 운영체제가 폴더 목록을 갱신한다.

#### E-4.4 `internal/session/types.go` — 여는 설정에 주기 추가

```go
type OpenConfig struct {
	…
	LogEnabled    bool
	LogAlias      string
	// LogSyncSec 은 세션 로그의 내려 쓰기 주기(초, 0~5)다. 0 이면 받는 즉시 쓰고 내려 쓴다 (session.md §6).
	LogSyncSec int
	…
}
```

주의: 이 필드의 0 값은 "즉시 모드"다. `OpenConfig` 를 만드는 쪽이 값을 넣지 않으면 즉시 모드가 된다. 앱은 항상 설정값을 넣는다(§4.7). 세션 테스트 대부분은 넣지 않으므로 즉시 모드로 돈다.

#### E-4.5 `internal/session/session.go` — 세션이 주기를 기억한다

```go
type Session struct {
	…
	tLastInput   time.Time // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
	iLogSyncSec  int       // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
}

func newSession(cfg OpenConfig, emit EmitFunc) *Session {
	pSession := &Session{
		cfg:     cfg,
		info:    cfg.Info,
		emit:    emit,
		decoder: NewDecoder(cfg.Encoding),

		iLogSyncSec: ClampLogSyncSec(cfg.LogSyncSec),
	}
	…
```

세션이 값을 따로 들고 있는 이유: 기록이 꺼져 있는 동안 주기 설정이 바뀌어도, 나중에 기록을 켤 때 새 값을 써야 한다.

#### E-4.6 `internal/session/control.go` — 로그를 열 때 전달, 바뀌면 적용

`SetLogging`(D99 에서 만든 함수)의 파일 등록 부분에 한 줄을 더했다([control.go:137](../../internal/session/control.go#L137)).

```go
	pSession.mtxData.Lock()
	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
	if !bLost {
		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
		pSession.logger = pLogger
	}
	pSession.mtxData.Unlock()
```

세션 잠금 안에서 부르는 이유: 그 사이에 주기 설정이 바뀌어도 옛 값이 들어가지 않게 하려는 것이다. 방금 만든 빈 파일이라 0 모드여도 내려 쓸 것이 없다.

새 함수([control.go:149](../../internal/session/control.go#L149)):

```go
// SetLogSyncSec 은 이 세션 로그의 내려 쓰기 주기(초, 0~5)를 바꾼다. 기록 중이면 곧바로 적용하고,
// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogSyncSec, session.md §6).
func (pSession *Session) SetLogSyncSec(iSec int) {
	iSec = ClampLogSyncSec(iSec)
	pSession.mtxData.Lock()
	pSession.iLogSyncSec = iSec
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.SetSyncSec(iSec)
	}
}
```

#### E-4.7 `internal/session/manager.go` — 열린 세션 전체에 적용

```go
// SetLogSyncSec 은 열려 있는 모든 세션의 로그 내려 쓰기 주기를 바꾼다 (설정 sessionLogSyncSec 을 바꿨을 때).
func (pManager *Manager) SetLogSyncSec(iSec int) {
	pManager.mtxData.Lock()
	slSessions := make([]*Session, 0, len(pManager.mapSessions))
	for _, pSession := range pManager.mapSessions {
		slSessions = append(slSessions, pSession)
	}
	pManager.mtxData.Unlock()
	for _, pSession := range slSessions {
		pSession.SetLogSyncSec(iSec)
	}
}
```

#### E-4.8 `internal/app` — 설정 읽기, 열 때 전달, 바뀌면 적용

[app.go](../../internal/app/app.go):

```go
// logSyncSec 은 세션 로그의 내려 쓰기 주기 설정이다 (초, 0~5, 0 = 받는 즉시 쓰고 내려 쓴다).
func (pService *Service) logSyncSec() int {
	return session.ClampLogSyncSec(pService.settingInt("sessionLogSyncSec", session.G_ciLogSyncSecDefault))
}
```

`store.patch` 처리([app.go:147-169](../../internal/app/app.go#L147-L169)) 전체:

```go
	pService.pRouter.Handle("store.patch", func(slArgs []json.RawMessage) (any, error) {
		var mapPatch map[string]any
		if errArg := argInto(slArgs, 0, &mapPatch); errArg != nil {
			return nil, errArg
		}
		iKeepBefore := pService.logKeepDays()
		bLogBefore := pService.settingBool("sessionLog", false)
		iSyncBefore := pService.logSyncSec()                    // ← 추가
		errPatch := pService.pStore.Patch(mapPatch)
		pService.applyLanguage()
		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
			// "터미널 출력 기록"을 바꾸면 열려 있는 세션에도 곧바로 적용한다 (session.md §6, D99).
			pService.pSessions.SetLogging(bLogNow)
		}
		if iSyncNow := pService.logSyncSec(); iSyncNow != iSyncBefore {   // ← 추가
			// 내려 쓰기 주기도 열려 있는 세션에 곧바로 적용한다 (session.md §6, D100).
			pService.pSessions.SetLogSyncSec(iSyncNow)
		}
		if pService.logKeepDays() != iKeepBefore {
			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
		}
		return nil, errPatch
	})
```

세션을 열 때([channels_session.go:187](../../internal/app/channels_session.go#L187)):

```go
		LogEnabled:    pService.settingBool("sessionLog", false),
		LogAlias:      record.M_szAlias,
		LogSyncSec:    pService.logSyncSec(),      // ← 추가
```

#### E-4.9 `internal/store/store.go` — 기본값

```go
	"sessionLog":         false,
	"sessionLogKeepDays": 30,
	"sessionLogSyncSec":  2,      // ← 추가
```

기존 `store.json` 에 이 키가 없어도 기본값 2 로 읽힌다(데이터 이전 작업 없음).

---

### E-5. 수정 내용 — 프런트엔드·문구

#### E-5.1 타입·기본값·계약 파일

```ts
// frontend/src/ipc/types.ts
  sessionLogKeepDays: number;
  /** 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 받는 즉시 쓰고 내려 쓴다 (behavior/session.md §6). */
  sessionLogSyncSec: number;

// frontend/src/lib/defaults.ts
  sessionLogKeepDays: 30,
  sessionLogSyncSec: 2,
```

`frontend/src/ipc/contract/store.get.json`, `store.get.afterNullPatch.json` 의 `settings` 에 `"sessionLogSyncSec": 2` 를 더했다. 이 두 파일은 백엔드의 `store.get` 결과와 맞는지 계약 테스트가 비교한다.

#### E-5.2 설정 화면 — `frontend/src/screens/SettingsConnection.tsx`

"로그 보관 기간 (일)" 아래, [로그 폴더 열기] 위에 필드를 더했다.

```tsx
  const [keepDays] = useSetting('sessionLogKeepDays');
  const [syncSec] = useSetting('sessionLogSyncSec');
  …
      <Field label={t('conn.logSyncSec')} help={t('conn.logSyncSecHelp')}>
        <NumberField
          label={t('conn.logSyncSec')}
          value={syncSec}
          min={0}
          max={5}
          fallback={2}
          onCommit={(v) => void saveSettings({ sessionLogSyncSec: v })}
        />
      </Field>
```

`NumberField` 는 입력을 벗어날 때(또는 Enter) 범위를 0~5 로 맞추고, 비우면 2 로 저장한다.

#### E-5.3 문구 — `locales/*.json` (10개 언어)

| 키 | ko | en |
|---|---|---|
| `conn.logSyncSec` | 로그 파일 갱신 주기 (초) | Log file update interval (seconds) |
| `conn.logSyncSecHelp` | 기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다. | How often a log being recorded is flushed all the way to disk (0–5). File Explorer shows the file size and modified time only after such a flush. 0 writes and flushes every piece of output immediately, which increases disk load. Otherwise output is written to the file every second. |

나머지 8개 언어(ja, zh, tw, de, fr, es, pt, ru)도 같은 뜻으로 넣었다. 한국어·영어 외에는 이 수정을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

---

### E-6. 명세·기록

| 파일 | 내용 |
|---|---|
| `spec/behavior/session.md` §6 | "내려 쓰기 주기" 항목 추가(설정 이름, 범위, 0 과 1~5 의 동작, 곧바로 적용, 2026-11-18 표기) |
| `spec/data/store.md` §4 | 설정 표에 `sessionLogSyncSec` 행(기본 2) |
| `spec/ui/settings.md` §4 | 연결 화면에 "로그 파일 갱신 주기 (초)" 필드와 도움말 |
| `docs/decisions.md` | **D100** |
| `.claude/wip/archive/2026-11-18-session-log-sync-interval.md` | 작업 기록 |

---

### E-7. 동작 예시

설정 2초, 출력이 계속 나오는 세션(시각은 초):

```
t=0.0  출력 도착 → 버퍼, 타이머(1초) 시작
t=1.0  타이머: 파일에 쓰기. 마지막 내려 쓰기(연 시각)에서 2초 미만이면 타이머 다시
t=2.0  타이머: 파일에 쓰기 + 내려 쓰기  → 탐색기 크기 갱신
t=3.0  타이머: 파일에 쓰기 (2초 미만) → 타이머 다시
t=4.0  타이머: 파일에 쓰기 + 내려 쓰기
…
```

설정 5초, 출력이 한 번 나오고 멈춤:

```
t=0.0  출력 도착
t=1.0  타이머: 파일에 쓰기. 아직 5초가 안 됐으면 타이머 다시 (내용은 없지만 내려 쓰기를 기다린다)
…      1초마다 확인만 한다
t≤6.0  내려 쓰기 → 타이머 멈춤 (다음 출력이 올 때까지 아무 일도 하지 않는다)
```

설정 0:

```
출력 도착 → 파일에 쓰기 + 내려 쓰기 (그 자리에서, 끝날 때까지 그 세션의 다음 출력 처리가 기다린다)
```

---

### E-8. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 디스크 부하(1~5) | 기록 중이고 출력이 있는 세션마다 설정한 초에 한 번 동기화한다. 조용한 세션은 하지 않는다 | 부하를 재 보지 않았다. 문제가 되면 값을 5 로 올린다 |
| 디스크 부하·출력 지연(0) | 출력 조각마다 동기화한다. 고속 출력 세션에서는 로그 기록이 읽기 고루틴을 붙잡아 화면 출력과 서버 쪽 프로그램까지 느려질 수 있다 | 사용자가 고른 동작. 도움말에 부하를 적었다 |
| 비정상 종료 시 유실 | 전에는 운영체제 캐시에만 있던 내용이 주기마다 디스크에 내려가므로 잃는 양이 줄어든다(개선) | — |
| 기존 사용자 | `store.json` 에 키가 없으면 기본 2초가 적용된다. 전에는 내려 쓰기가 없었으므로 동작이 바뀐다(의도한 변화) | — |
| Windows 외 OS | `Sync` 는 fsync 로 동작한다. 폴더 목록 문제는 Windows 의 것이지만 유실 감소 효과는 같다 | OS 별 분기 없음 |
| 잠금 순서 | 세션 잠금 → 로그 잠금 순서만 쓴다(`SetLogging` 안의 `SetSyncSec`, 기존 `writeOutputUnlockedNotice`). 반대 순서는 없다 | — |
| 동기화·백업 가져오기로 설정이 바뀜 | `store.patch` 를 거치지 않아 기록 중인 세션에는 적용되지 않는다(새로 기록을 시작하는 세션부터) | D99 와 같은 한계 |
| 설정 화면에서 소수 입력 | `NumberField` 에 `step` 을 주지 않아 `2.5` 같은 값이 저장될 수 있다. 백엔드는 소수점 아래를 버려 2 로 쓴다 | 기능 문제는 없다. §10 |
| 오래된 로그 정리(`logclean`) | 기록 중인 파일은 전과 같이 건너뛴다. 내려 쓰기로 수정 시각이 실제에 가깝게 유지된다 | — |

---

### E-9. 검증

#### E-9.1 자동 테스트 (2026-11-18 실행)

[internal/session/log_test.go](../../internal/session/log_test.go):

```go
// 내려 쓰기 주기 0: 받는 즉시 파일에 쓴다(1초 주기를 기다리지 않는다) (session.md §6, D100).
func TestLogSyncZeroWritesImmediately(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(0)
	pLogger.Write("now\n")
	bData, _ := os.ReadFile(pLogger.Path())
	if string(bData) != "now\n" {
		t.Fatalf("즉시 모드는 바로 파일에 있어야 함: %q", string(bData))
	}
}

// 버퍼에 쌓인 채로 주기를 0 으로 바꾸면 곧바로 파일에 쓴다. 0 이 아니면 지금처럼 약 1초 뒤에 쓴다.
func TestLogSyncSecSwitch(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(3)
	pLogger.Write("buffered\n")
	if bData, _ := os.ReadFile(pLogger.Path()); len(bData) != 0 {
		t.Fatalf("0 이 아니면 바로 쓰지 않음: %q", string(bData))
	}
	pLogger.SetSyncSec(0)
	if bData, _ := os.ReadFile(pLogger.Path()); string(bData) != "buffered\n" {
		t.Fatalf("0 으로 바꾸면 곧바로 써야 함: %q", string(bData))
	}
	pLogger.SetSyncSec(5)
	pLogger.Write("later\n")
	waitUntil(t, func() bool {
		bData, _ := os.ReadFile(pLogger.Path())
		return string(bData) == "buffered\nlater\n"
	}, 2*time.Second)
}

// 범위 밖 값은 0~5 로 맞춘다.
func TestClampLogSyncSec(t *testing.T) {
	for iIn, iWant := range map[int]int{-3: 0, 0: 0, 2: 2, 5: 5, 9: 5} {
		if iGot := ClampLogSyncSec(iIn); iGot != iWant {
			t.Fatalf("ClampLogSyncSec(%d) = %d, 기대 %d", iIn, iGot, iWant)
		}
	}
}

// 세션에 준 주기는 기록을 켤 때 로그에 전해지고, 바꾸면 기록 중인 로그에도 곧바로 적용된다.
func TestSessionLogSyncSecApplies(t *testing.T) {
	szDir := t.TempDir()
	pFake := newFakeTransport()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
	cfg.DataDir = szDir
	cfg.LogAlias = "web"
	cfg.LogEnabled = true
	cfg.LogSyncSec = 4
	info, _ := pManager.Open(cfg)
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	if iGot := pSession.logSyncSecForTest(); iGot != 4 {
		t.Fatalf("열 때 준 주기가 로그에 전해져야 함: %d", iGot)
	}
	pManager.SetLogSyncSec(0)
	if iGot := pSession.logSyncSecForTest(); iGot != 0 {
		t.Fatalf("바꾼 주기가 기록 중인 로그에 적용돼야 함: %d", iGot)
	}
	pManager.SetLogging(false)
	pManager.SetLogSyncSec(1)
	pManager.SetLogging(true)
	if iGot := pSession.logSyncSecForTest(); iGot != 1 {
		t.Fatalf("꺼져 있을 때 바꾼 주기도 다시 켤 때 써야 함: %d", iGot)
	}
}
```

Windows 전용 [internal/session/log_sync_windows_test.go](../../internal/session/log_sync_windows_test.go) — 이 문서의 문제 자체(폴더 목록의 크기)를 확인한다:

```go
//go:build windows

// listedSize 는 폴더 목록(탐색기가 보는 값)에 적힌 파일 크기다. Windows 는 파일이 열려 있는 동안
// 내려 쓰기(fsync)나 닫기 전에는 이 값을 갱신하지 않는다.
func listedSize(t *testing.T, szPath string) int64 {
	t.Helper()
	slEntries, errRead := os.ReadDir(filepath.Dir(szPath))
	if errRead != nil {
		t.Fatal(errRead)
	}
	for _, entry := range slEntries {
		if entry.Name() == filepath.Base(szPath) {
			pInfo, errInfo := entry.Info()
			if errInfo != nil {
				t.Fatal(errInfo)
			}
			return pInfo.Size()
		}
	}
	t.Fatalf("목록에 없음: %s", szPath)
	return 0
}

// 내려 쓰기 주기가 지나면 파일을 닫지 않아도 폴더 목록의 크기가 실제 크기로 바뀐다 (D100).
func TestLogSyncUpdatesListedSize(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(1)
	pLogger.Write("0123456789\n")
	waitUntil(t, func() bool { return listedSize(t, pLogger.Path()) == 11 }, 4*time.Second)

	pLogger.SetSyncSec(0)
	pLogger.Write("abc\n")
	if iSize := listedSize(t, pLogger.Path()); iSize != 15 {
		t.Fatalf("즉시 모드는 쓰자마자 목록 크기가 맞아야 함: %d", iSize)
	}
}
```

| 테스트 | 확인하는 것 |
|---|---|
| `TestLogSyncZeroWritesImmediately` | 0 모드는 1초를 기다리지 않고 바로 파일에 있다 |
| `TestLogSyncSecSwitch` | 1~5 에서는 바로 쓰지 않는다 / 0 으로 바꾸면 쌓인 것을 곧바로 쓴다 / 다시 5 로 바꾸면 약 1초 뒤에 쓴다 |
| `TestClampLogSyncSec` | 범위 밖 값 맞춤 |
| `TestSessionLogSyncSecApplies` | 여는 설정의 주기가 로그에 전해진다 / 바꾸면 기록 중인 로그에 적용 / 꺼져 있을 때 바꾼 값도 다시 켤 때 쓴다 |
| `TestLogSyncUpdatesListedSize` (Windows) | 파일을 닫지 않아도 주기가 지나면 폴더 목록 크기가 맞다 / 0 모드는 쓰자마자 맞다 |
| `TestLogFlushesPeriodically` 등 기존 로그 테스트 | 그대로 통과(기본 2초 모드에서 1초 쓰기 유지) |

#### E-9.2 실행 결과

```bash
go build ./...
go test ./...                       # 전부 통과
(cd frontend && npx tsc --noEmit)   # 통과
(cd frontend && npx vitest run)     # 38개 파일, 233개 통과 (계약 파일·문구 포함)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe, 2026-11-18 10:59
```

- `go test -race` 는 이 PC 에 cgo 컴파일러가 없어 실행하지 못했다.
- `store.patch` 에서 주기가 바뀌었을 때 세션까지 이어지는 경로(§4.8)를 직접 확인하는 자동 테스트는 없다. 세션·관리자 쪽은 `TestSessionLogSyncSecApplies` 가 확인한다.
- 디스크 부하와 0 모드의 출력 지연은 재지 않았다.

#### E-9.3 수동 확인

2026-11-18 사용자가 앱에서 정상 동작을 확인했다("정상동작 확인했다"). 어떤 값·절차로 확인했는지는 전달받지 못했다. 다시 확인할 때의 절차:

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | "터미널 출력 기록"을 켜고 설정 2초(기본)로 세션에서 출력을 낸다 | 3초 안에 탐색기(F5)에서 파일 크기와 수정 시각이 바뀐다. 세션은 열린 채다 |
| 2 | 출력을 멈춘다 | 마지막 출력까지 반영된 크기에서 멈춘다 |
| 3 | 설정을 5 로 바꾸고 출력 | 크기가 5~6초 간격으로 바뀐다(열려 있는 세션에 바로 적용) |
| 4 | 설정을 0 으로 바꾸고 출력 | 줄이 나올 때마다 크기가 바뀐다 |
| 5 | 0 인 채로 고속 출력(큰 로그 `cat` 등) | 기록은 빠짐없이 되고, 화면 출력이 눈에 띄게 느려질 수 있다 |
| 6 | 필드를 비우고 벗어남 / 9 입력 | 2 / 5 로 저장된다 |
| 7 | 출력이 없는 세션을 5분 둔다 | 파일 수정 시각이 바뀌지 않는다(쓸데없는 내려 쓰기가 없다) |

---

### E-10. 남은 정리 거리 (이번에 하지 않음)

| 항목 | 비고 |
|---|---|
| 설정 화면 숫자 필드에 정수만 받기 | `NumberField` 에 `step={1}` 을 주면 된다. "로그 보관 기간 (일)" 필드도 같다 |
| `store.patch` 경로 테스트 | 앱 서비스에 로컬 세션을 열고 주기를 바꿔 로그에 적용되는지 확인 |
| 디스크 부하 측정 | 고속 출력 세션 여러 개에서 1초·2초·0 을 비교 |
| 동기화·백업 가져오기로 설정이 바뀐 경우의 즉시 적용 | D99 문서 §8 과 같은 항목 |
| 8개 언어 문구 검수 | §5.3 |
| 로그 파일 열기 실패를 사용자에게 알리기 | D99 문서 §8 과 같은 항목 |

---

### E-11. 체크리스트

- [x] 1. 원인 확인: 실제 크기와 폴더 목록 비교, NTFS 실험 (§1, §2)
- [x] 2. `log.go`: 상수·`ClampLogSyncSec`, `Logger` 필드, `Write`·`syncNowLocked`·`SetSyncSec`·`flushTimer` (§4.1~4.3)
- [x] 3. `types.go`·`session.go`·`control.go`·`manager.go`: 주기 전달과 적용 (§4.4~4.7)
- [x] 4. `app.go`·`channels_session.go`·`store.go`: 설정 읽기·열 때 전달·바뀌면 적용·기본값 (§4.8, §4.9)
- [x] 5. 프런트: 타입·기본값·계약 파일·설정 화면 필드, 문구 10개 언어 (§5)
- [x] 6. 명세 3곳, decisions D100, WIP 보관 (§6)
- [x] 7. 테스트 5개 추가, `go test ./...`·`tsc`·`vitest` 통과, 빌드 (§9.1, §9.2)
- [x] 8. 사용자 수동 확인 (§9.3)
- [ ] 9. (선택) §10 의 정리 거리, `go test -race`

---

### E-부록 A. 문구 전체 (`locales/*.json`, 10개 언어)

순서: ko, en, ja, zh, tw, de, fr, es, pt, ru.

```json
  "conn.logSyncSec": "로그 파일 갱신 주기 (초)",
  "conn.logSyncSecHelp": "기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다.",
  "conn.logSyncSec": "Log file update interval (seconds)",
  "conn.logSyncSecHelp": "How often a log being recorded is flushed all the way to disk (0–5). File Explorer shows the file size and modified time only after such a flush. 0 writes and flushes every piece of output immediately, which increases disk load. Otherwise output is written to the file every second.",
  "conn.logSyncSec": "ログファイルの更新間隔（秒）",
  "conn.logSyncSecHelp": "記録中のログをディスクまで書き出す間隔です（0～5）。書き出すまでエクスプローラーにはファイルサイズと更新日時が反映されません。0 にすると出力を受け取るたびに書き込んで書き出すため、ディスク負荷が大きくなります。0 以外ではファイルへの書き込みは 1 秒ごとです。",
  "conn.logSyncSec": "日志文件更新间隔（秒）",
  "conn.logSyncSecHelp": "将正在记录的日志完全写入磁盘的间隔（0～5）。写入磁盘后，资源管理器才会显示文件大小和修改时间。设为 0 时，每收到输出就立即写入并落盘，磁盘负载会增大。不为 0 时，每秒写入文件一次。",
  "conn.logSyncSec": "記錄檔更新間隔（秒）",
  "conn.logSyncSecHelp": "將記錄中的記錄檔完全寫入磁碟的間隔（0～5）。寫入磁碟後，檔案總管才會顯示檔案大小與修改時間。設為 0 時，每收到輸出就立即寫入並落盤，磁碟負載會增加。不為 0 時，每秒寫入檔案一次。",
  "conn.logSyncSec": "Aktualisierungsintervall der Protokolldatei (Sekunden)",
  "conn.logSyncSecHelp": "Wie oft ein laufendes Protokoll vollständig auf den Datenträger geschrieben wird (0–5). Erst danach zeigt der Explorer Dateigröße und Änderungszeit an. Bei 0 wird jede Ausgabe sofort geschrieben und auf den Datenträger übertragen, was die Datenträgerlast erhöht. Sonst wird jede Sekunde in die Datei geschrieben.",
  "conn.logSyncSec": "Intervalle de mise à jour du fichier journal (secondes)",
  "conn.logSyncSecHelp": "Fréquence à laquelle un journal en cours d'enregistrement est écrit jusqu'au disque (0–5). L'Explorateur n'affiche la taille et la date de modification qu'après cette écriture. Avec 0, chaque sortie est écrite et vidée sur le disque immédiatement, ce qui augmente la charge du disque. Sinon, l'écriture dans le fichier a lieu chaque seconde.",
  "conn.logSyncSec": "Intervalo de actualización del archivo de registro (segundos)",
  "conn.logSyncSecHelp": "Cada cuánto se vuelca al disco un registro en curso (0–5). El Explorador solo muestra el tamaño y la fecha de modificación después de ese volcado. Con 0, cada salida se escribe y se vuelca al disco de inmediato, lo que aumenta la carga del disco. En otro caso, se escribe en el archivo cada segundo.",
  "conn.logSyncSec": "Intervalo de atualização do arquivo de log (segundos)",
  "conn.logSyncSecHelp": "Com que frequência um log em gravação é descarregado até o disco (0–5). O Explorador só mostra o tamanho e a data de modificação depois dessa descarga. Com 0, cada saída é gravada e descarregada no disco imediatamente, o que aumenta a carga do disco. Caso contrário, a gravação no arquivo ocorre a cada segundo.",
  "conn.logSyncSec": "Интервал обновления файла журнала (секунды)",
  "conn.logSyncSecHelp": "Как часто записываемый журнал сбрасывается на диск (0–5). Проводник показывает размер файла и время изменения только после такого сброса. При 0 каждый вывод сразу записывается и сбрасывается на диск, что увеличивает нагрузку на диск. Иначе запись в файл выполняется раз в секунду.",
```

### E-부록 B. `internal/session/control.go` — `SetLogging`·`SetLogSyncSec` (최종)

```go
// SetLogging 은 이 세션의 출력 기록을 켜거나 끈다 (session.md §6).
// 켜면 그 순간부터의 출력을 새 로그 파일에 쓴다(이미 기록 중이면 그대로). 끄면 남은 것을 쓰고 파일을 닫는다.
// 세션을 열 때(start)와 설정 "터미널 출력 기록"을 바꿀 때 부른다. 닫힌 세션은 건드리지 않는다.
func (pSession *Session) SetLogging(bEnabled bool) {
	pSession.mtxData.Lock()
	if pSession.userClosed {
		pSession.mtxData.Unlock()
		return
	}
	if !bEnabled {
		pLogger := pSession.logger
		pSession.logger = nil
		pSession.mtxData.Unlock()
		if pLogger != nil {
			_ = pLogger.Close()
		}
		return
	}
	bSkip := pSession.logger != nil || pSession.cfg.DataDir == ""
	pSession.mtxData.Unlock()
	if bSkip {
		return
	}
	// 파일 만들기는 잠금 밖에서 한다(디스크를 기다리는 동안 출력이 막히지 않게).
	pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
	if errLog != nil {
		return // 열기 실패는 조용히 넘어간다: 기록 없이 세션은 계속 쓴다
	}
	pSession.mtxData.Lock()
	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
	if !bLost {
		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
		pSession.logger = pLogger
	}
	pSession.mtxData.Unlock()
	if bLost {
		_ = pLogger.Close()
		_ = os.Remove(pLogger.Path()) // 방금 만든 빈 파일은 남기지 않는다
	}
}

// SetLogSyncSec 은 이 세션 로그의 내려 쓰기 주기(초, 0~5)를 바꾼다. 기록 중이면 곧바로 적용하고,
// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogSyncSec, session.md §6).
func (pSession *Session) SetLogSyncSec(iSec int) {
	iSec = ClampLogSyncSec(iSec)
	pSession.mtxData.Lock()
	pSession.iLogSyncSec = iSec
	pLogger := pSession.logger
	pSession.mtxData.Unlock()
	if pLogger != nil {
		pLogger.SetSyncSec(iSec)
	}
}
```

---

## 부록 P. 코드

코드 블록은 백틱 네 개로 감쌌다. 블록 안의 내용만 복사한다. 줄바꿈은 LF 다(공식 소스의 파일과 같다).

### P.1 새 파일 4개 (전체 내용)

#### `frontend/src/lib/findHighlight.ts`

````ts
// 터미널 찾기 강조의 색·범위 계산. 근거: spec/ui/panes.md §2.4 (찾은 항목 주황 계열, 현재 항목 빨강 계열).
// 찾은 항목은 터미널 배경에 주황을 옅게 섞어 낮추고, 현재 항목은 진한 빨강 바탕 + 흰 글자 + 테두리로 올린다 (D96).

/** 찾은 항목(현재 항목 제외)의 기준 색과 섞는 비율. */
export const FIND_MATCH_COLOR = '#f59e0b';
export const FIND_MATCH_RATIO = 0.32;
/** 현재 항목의 바탕·글자 색. */
export const FIND_ACTIVE_BG = '#dc2626';
export const FIND_ACTIVE_FG = '#ffffff';
/** 스크롤 막대 표시 색. */
export const FIND_RULER_MATCH = '#f59e0b';
export const FIND_RULER_ACTIVE = '#ef4444';
/** 스크롤 막대 자리의 표시 폭(px). xterm 스크롤 막대 기본 폭과 같게 둔다. */
export const FIND_RULER_WIDTH = 14;

const HEX6 = /^#[0-9a-fA-F]{6}$/;

/** `#RRGGBB` 두 색을 섞는다(ratio = over 의 비중, 0~1). 형식이 아니면 over 를 그대로 돌려준다. */
export function mixHex(base: string, over: string, ratio: number): string {
  if (!HEX6.test(base) || !HEX6.test(over)) return over;
  const r = Math.min(1, Math.max(0, ratio));
  let out = '#';
  for (let i = 1; i < 7; i += 2) {
    const b = parseInt(base.slice(i, i + 2), 16);
    const o = parseInt(over.slice(i, i + 2), 16);
    out += Math.round(b + (o - b) * r).toString(16).padStart(2, '0');
  }
  return out;
}

export interface CellPos { x: number; y: number }
export interface RowRange { row: number; x: number; width: number }

/**
 * 선택 범위(시작 칸 포함, 끝 칸 제외, 0부터 세는 버퍼 좌표)를 줄별 범위로 나눈다.
 * 일치 항목이 줄 끝에서 다음 줄로 이어지면 줄마다 하나씩 나온다.
 */
export function rowRanges(start: CellPos, end: CellPos, cols: number): RowRange[] {
  const out: RowRange[] = [];
  for (let row = start.y; row <= end.y; row++) {
    const x0 = row === start.y ? start.x : 0;
    const x1 = row === end.y ? end.x : cols;
    if (x1 > x0) out.push({ row, x: x0, width: x1 - x0 });
  }
  return out;
}
````

#### `frontend/src/lib/findHighlight.test.ts`

````ts
import { describe, expect, it } from 'vitest';
import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_MATCH_COLOR, FIND_MATCH_RATIO } from './findHighlight';
import { TERM_THEMES } from './termThemes';

function luminance(hex: string): number {
  const ch = [1, 3, 5].map((i) => {
    const v = parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * ch[0] + 0.7152 * ch[1] + 0.0722 * ch[2];
}
function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

describe('findHighlight', () => {
  it('mixHex 는 두 색을 비율대로 섞는다', () => {
    expect(mixHex('#000000', '#ffffff', 0)).toBe('#000000');
    expect(mixHex('#000000', '#ffffff', 1)).toBe('#ffffff');
    expect(mixHex('#000000', '#ffffff', 0.5)).toBe('#808080');
    expect(mixHex('#0d1117', '#f59e0b', 0.32)).toMatch(/^#[0-9a-f]{6}$/);
  });

  it('mixHex 는 형식이 아닌 색이면 덧칠 색을 그대로 준다', () => {
    expect(mixHex('rgb(0,0,0)', '#f59e0b', 0.3)).toBe('#f59e0b');
  });

  it('모든 터미널 테마에서 찾은 항목 위의 글자가 읽히고, 찾은 항목은 원색보다 덜 튄다', () => {
    for (const th of TERM_THEMES) {
      const match = mixHex(th.background, FIND_MATCH_COLOR, FIND_MATCH_RATIO);
      expect(contrast(match, th.foreground), th.name).toBeGreaterThan(3);
      expect(contrast(FIND_ACTIVE_BG, '#ffffff'), th.name).toBeGreaterThan(4.5);
      // 찾은 항목은 원색 주황보다 배경에 가깝다(덜 튄다). 현재 항목과는 색상·흰 글자·테두리로 구분한다.
      expect(contrast(match, th.background), th.name).toBeLessThan(contrast(FIND_MATCH_COLOR, th.background));
    }
  });

  it('rowRanges 는 한 줄 범위를 그대로 준다', () => {
    expect(rowRanges({ x: 3, y: 10 }, { x: 8, y: 10 }, 80)).toEqual([{ row: 10, x: 3, width: 5 }]);
  });

  it('rowRanges 는 줄을 넘는 범위를 줄마다 나눈다', () => {
    expect(rowRanges({ x: 78, y: 5 }, { x: 4, y: 7 }, 80)).toEqual([
      { row: 5, x: 78, width: 2 },
      { row: 6, x: 0, width: 80 },
      { row: 7, x: 0, width: 4 },
    ]);
  });

  it('rowRanges 는 빈 줄 조각을 버린다', () => {
    expect(rowRanges({ x: 80, y: 1 }, { x: 0, y: 2 }, 80)).toEqual([]);
  });
});
````

#### `frontend/src/lib/useCapsLock.ts`

````ts
// Caps Lock 이 켜져 있는지 알려 주는 훅. 비밀번호 입력 화면에서 쓴다 (D97).
// 브라우저는 Caps Lock 상태를 따로 물어볼 방법이 없고, 키·마우스 이벤트에 실려 올 때만 알 수 있다.
// 그래서 키를 누르고 뗄 때와 마우스를 움직이거나 누를 때 상태를 읽는다(첫 이벤트 전에는 "꺼짐"으로 본다).
import { useEffect, useState } from 'react';

type ModifierEvent = { getModifierState?: (key: string) => boolean };

/** 이벤트에서 Caps Lock 상태를 읽는다. 알 수 없으면 null. */
export function capsLockOf(e: ModifierEvent): boolean | null {
  return typeof e.getModifierState === 'function' ? e.getModifierState('CapsLock') : null;
}

const EVENTS = ['keydown', 'keyup', 'mousedown', 'mousemove'] as const;

/** enabled 가 참인 동안 창의 키·마우스 이벤트로 Caps Lock 상태를 따라간다. */
export function useCapsLock(enabled = true): boolean {
  const [on, setOn] = useState(false);
  useEffect(() => {
    if (!enabled) {
      setOn(false);
      return;
    }
    const read = (e: Event) => {
      const state = capsLockOf(e as unknown as ModifierEvent);
      if (state !== null) setOn(state);
    };
    // 창을 벗어난 동안 바뀐 상태는 알 수 없으므로 모른다고 보고 끈다. 돌아와서 첫 이벤트에 다시 읽는다.
    const reset = () => setOn(false);
    for (const name of EVENTS) window.addEventListener(name, read, true);
    window.addEventListener('blur', reset);
    return () => {
      for (const name of EVENTS) window.removeEventListener(name, read, true);
      window.removeEventListener('blur', reset);
    };
  }, [enabled]);
  return on;
}
````

#### `internal/session/log_sync_windows_test.go`

````go
//go:build windows

package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// listedSize 는 폴더 목록(탐색기가 보는 값)에 적힌 파일 크기다. Windows 는 파일이 열려 있는 동안
// 내려 쓰기(fsync)나 닫기 전에는 이 값을 갱신하지 않는다.
func listedSize(t *testing.T, szPath string) int64 {
	t.Helper()
	slEntries, errRead := os.ReadDir(filepath.Dir(szPath))
	if errRead != nil {
		t.Fatal(errRead)
	}
	for _, entry := range slEntries {
		if entry.Name() == filepath.Base(szPath) {
			pInfo, errInfo := entry.Info()
			if errInfo != nil {
				t.Fatal(errInfo)
			}
			return pInfo.Size()
		}
	}
	t.Fatalf("목록에 없음: %s", szPath)
	return 0
}

// 내려 쓰기 주기가 지나면 파일을 닫지 않아도 폴더 목록의 크기가 실제 크기로 바뀐다 (D100).
func TestLogSyncUpdatesListedSize(t *testing.T) {
	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pLogger.Close()
	pLogger.SetSyncSec(1)
	pLogger.Write("0123456789\n")
	waitUntil(t, func() bool { return listedSize(t, pLogger.Path()) == 11 }, 4*time.Second)

	pLogger.SetSyncSec(0)
	pLogger.Write("abc\n")
	if iSize := listedSize(t, pLogger.Path()); iSize != 15 {
		t.Fatalf("즉시 모드는 쓰자마자 목록 크기가 맞아야 함: %d", iSize)
	}
}
````

### P.2 바꿀 파일의 패치 (공식 소스 1.0.0 → 적용 후)

`diff -u` 형식이다. `a/` 는 공식 소스의 지금 파일, `b/` 는 적용 뒤 파일. 적용 방법은 §2.3.

#### 묶음 A — 프런트 (1·2·3번)

##### `frontend/src/screens/Panes.tsx`

````diff
--- a/frontend/src/screens/Panes.tsx
+++ b/frontend/src/screens/Panes.tsx
@@ -10,7 +10,6 @@
 import { ENV_INFO } from '../lib/defaults';
 import { TerminalView, termHandles, pasteFromClipboard, targetsFor, FIND_EVENT } from './TerminalView';
 import { forceImagePaste, uploadAndInsert } from '../state/paste';
-import { notYet } from '../lib/notYet';
 import { getBackend, errorMessage } from '../ipc/backend';
 import type { MenuItem } from '../state/ui';
 import { isFileDrag, hoverStart, hoverEnd, dropFiles } from './fileDrag';
@@ -239,7 +238,7 @@
           <button className="icon-btn small" title={info?.protocol === 'local' ? tr('pane.imageLocalTitle') : tr('pane.imageServerTitle')} aria-label={tr('pane.pasteImage')} onClick={() => { if (pane.sessionId) void forceImagePaste(pane.sessionId, targetsFor(pane.id, pane.sessionId)); }}><ImageIcon size={12} /></button>
         )}
         {hasSession && (
-          <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => notYet(tr('pane.find'))}><Search size={12} /></button>
+          <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id }))}><Search size={12} /></button>
         )}
         {hasSession && (
           <>
````

##### `frontend/src/screens/TerminalView.tsx`

````diff
--- a/frontend/src/screens/TerminalView.tsx
+++ b/frontend/src/screens/TerminalView.tsx
@@ -1,6 +1,6 @@
 // 터미널 칸 본문(xterm.js). 근거: spec/behavior/terminal.md §1~§5, §6, §8, spec/behavior/shortcuts.md §2~§3, spec/ui/panes.md §2.3~§2.4.
 import { useEffect, useRef, useState } from 'react';
-import { Terminal } from '@xterm/xterm';
+import { Terminal, type IDisposable } from '@xterm/xterm';
 import { FitAddon } from '@xterm/addon-fit';
 import { SearchAddon } from '@xterm/addon-search';
 import { WebLinksAddon } from '@xterm/addon-web-links';
@@ -15,6 +15,7 @@
 import { decodeOsc52 } from '../lib/osc52';
 import { shortcutFor } from '../lib/keys';
 import { resolveTermTheme } from '../lib/termThemes';
+import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_ACTIVE_FG, FIND_MATCH_COLOR, FIND_MATCH_RATIO, FIND_RULER_MATCH, FIND_RULER_ACTIVE, FIND_RULER_WIDTH } from '../lib/findHighlight';
 import { DANGEROUS_PATTERNS } from '../lib/defaults';
 import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';
 // 이 파일은 t 를 터미널 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
@@ -156,20 +157,65 @@
 function FindBar({ term, paneId, onClose }: { term: Terminal; paneId: string; onClose: () => void }) {
   const [q, setQ] = useState('');
   const addon = useRef<SearchAddon | null>(null);
+  // 현재 항목 위에 얹는 강조(줄마다 하나). 검색 애드온은 현재 항목을 "선택"으로 표시하는데,
+  // 애드온의 현재 항목 색은 선택 색 아래에 깔려 보이지 않는다. 선택 위(top)에 직접 그린다 (D96).
+  const active = useRef<IDisposable[]>([]);
   void paneId;
+  const clearActive = () => {
+    for (const d of active.current) d.dispose();
+    active.current = [];
+  };
+  const markActive = () => {
+    clearActive();
+    const sel = term.getSelectionPosition();
+    if (!sel) return;
+    const buf = term.buffer.active;
+    const border = term.options.theme?.foreground ?? FIND_ACTIVE_FG;
+    for (const r of rowRanges(sel.start, sel.end, term.cols)) {
+      const marker = term.registerMarker(r.row - buf.baseY - buf.cursorY);
+      const deco = term.registerDecoration({ marker, x: r.x, width: r.width, backgroundColor: FIND_ACTIVE_BG, foregroundColor: FIND_ACTIVE_FG, layer: 'top', overviewRulerOptions: { color: FIND_RULER_ACTIVE, position: 'full' } });
+      deco?.onRender((el) => {
+        el.style.outline = `1px solid ${border}`;
+      });
+      // 반드시 강조를 먼저, 표식을 나중에 지운다. 표식을 먼저 지우면 xterm 이 그 강조를 목록에서
+      // 찾지 못해 남기고(줄 번호가 -1 로 바뀐 뒤라서), 다른 강조의 삭제까지 실패하게 만든다.
+      active.current.push({
+        dispose() {
+          deco?.dispose();
+          marker.dispose();
+        },
+      });
+    }
+  };
   useEffect(() => {
     const a = new SearchAddon();
     term.loadAddon(a);
     addon.current = a;
-    return () => a.clearDecorations();
+    // 터미널을 눌러 선택이 풀리면 현재 항목 강조도 지운다.
+    const sub = term.onSelectionChange(() => {
+      if (!term.hasSelection()) clearActive();
+    });
+    return () => {
+      sub.dispose();
+      clearActive();
+      a.clearDecorations();
+    };
+    // clearActive 는 ref 만 다뤄 렌더마다 새로 만들어져도 같다.
   }, [term]);
-  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: '#f59e0b', activeMatchColorOverviewRuler: '#ef4444', matchBackground: '#f59e0b55', activeMatchBackground: '#ef444488' } };
+  // 찾은 항목: 터미널 배경에 주황을 옅게 섞은 바탕(테마가 밝든 어둡든 글자가 읽힌다). 색은 #RRGGBB 만 받는다.
+  const matchBg = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
+  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: FIND_RULER_MATCH, activeMatchColorOverviewRuler: FIND_RULER_ACTIVE, matchBackground: matchBg, activeMatchBackground: FIND_ACTIVE_BG } };
+  const find = (text: string, dir: 1 | -1) => {
+    const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
+    if (found) markActive();
+    else clearActive();
+  };
   const next = (dir: 1 | -1) => {
     if (!q) return;
-    if (dir === 1) addon.current?.findNext(q, opts);
-    else addon.current?.findPrevious(q, opts);
+    find(q, dir);
   };
   const close = () => {
+    clearActive();
     addon.current?.clearDecorations();
     onClose();
     term.focus();
@@ -183,8 +229,11 @@
         value={q}
         onChange={(e) => {
           setQ(e.target.value);
-          if (e.target.value) addon.current?.findNext(e.target.value, opts);
-          else addon.current?.clearDecorations();
+          if (e.target.value) find(e.target.value, 1);
+          else {
+            clearActive();
+            addon.current?.clearDecorations();
+          }
         }}
         onKeyDown={(e) => {
           if (e.key === 'Escape') {
@@ -251,7 +300,10 @@
     minimumContrastRatio: 1,
     smoothScrollDuration: 0,
     allowProposedApi: true,
+    // 찾기 결과를 스크롤 막대 자리에 표시한다(ui/panes.md §2.4). 폭은 xterm 스크롤 막대 기본 폭과 같아 칸 수가 바뀌지 않는다.
+    overviewRuler: { width: FIND_RULER_WIDTH },
     theme: {
+      overviewRulerBorder: theme.background,
       background: theme.background,
       foreground: theme.foreground,
       cursor: theme.cursor,
@@ -464,7 +516,7 @@
     t.options.cursorStyle = settings.cursorStyle;
     t.options.cursorBlink = settings.cursorBlink;
     t.options.scrollback = settings.scrollback;
-    t.options.theme = { ...t.options.theme, background: theme.background, foreground: theme.foreground, cursor: theme.cursor, selectionBackground: theme.selectionBackground };
+    t.options.theme = { ...t.options.theme, background: theme.background, foreground: theme.foreground, cursor: theme.cursor, selectionBackground: theme.selectionBackground, overviewRulerBorder: theme.background };
     fitRef.current?.fit();
   }, [settings]);
 
````

##### `frontend/src/screens/Lock.tsx`

````diff
--- a/frontend/src/screens/Lock.tsx
+++ b/frontend/src/screens/Lock.tsx
@@ -1,7 +1,8 @@
 // 잠금 화면: 잠금 해제 / 최초 설정 / 복구 코드 표시 / 복구 코드로 재설정.
 // 근거: spec/ui/lock.md 전체, spec/acceptance/01-lock-vault.md 의 L1~L10.
 import { useState, type FormEvent } from 'react';
-import { Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';
+import { ArrowBigUpDash, Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';
+import { useCapsLock } from '../lib/useCapsLock';
 import { getBackend, errorMessage } from '../ipc/backend';
 import { useApp } from '../state/app';
 import { LogoMark } from '../ui/Logo';
@@ -143,6 +144,11 @@
   };
 
   const score = passwordScore(pw);
+  // 비밀번호를 입력하는 화면(잠금 해제·처음 설정·복구)에서만 Caps Lock 을 살핀다 (D97).
+  const capsOn = useCapsLock(mode !== 'recovery');
+  const capsNote = capsOn && (
+    <div className="caps-note" role="status"><ArrowBigUpDash size={14} />{t('lock.capsLock')}</div>
+  );
 
   return (
     <div className={`lock-screen${overlay ? ' overlay-mode' : ''}`}>
@@ -167,6 +173,7 @@
               onChange={(e) => setPw(e.target.value)}
               aria-label={t('lock.pw')}
             />
+            {capsNote}
             <div className="err-text" role="alert">{error}</div>
             <div className="lock-actions">
               <button className="btn primary" type="submit" disabled={!pw || busy}>
@@ -219,6 +226,7 @@
               onChange={(e) => setPw2(e.target.value)}
               aria-label={t('setup.pw2')}
             />
+            {capsNote}
             <div className="err-text" role="alert">{error}</div>
             <button className="btn primary" type="submit" disabled={busy}>
               {busy ? t('setup.creating') : t('setup.create')}
@@ -278,6 +286,7 @@
               onChange={(e) => setNewPw2(e.target.value)}
               aria-label={t('sec.newPwAgain')}
             />
+            {capsNote}
             <div className="err-text" role="alert">{error}</div>
             <button className="btn primary" type="submit" disabled={busy}>{t('recover.submit')}</button>
             <div className="center-row">
````

##### `frontend/src/screens/Lock.test.tsx`

````diff
--- a/frontend/src/screens/Lock.test.tsx
+++ b/frontend/src/screens/Lock.test.tsx
@@ -89,3 +89,35 @@
     await waitFor(() => expect(input.value).toBe(''));
   });
 });
+
+describe('Caps Lock 안내 (D97)', () => {
+  beforeEach(() => setBackend(null));
+
+  it('Caps Lock 이 켜진 채 키를 누르면 안내가 보이고, 끄면 사라진다', async () => {
+    await setup(true);
+    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
+    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
+    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
+    fireEvent.keyUp(window, { key: 'CapsLock', modifierCapsLock: false });
+    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
+  });
+
+  it('키를 누르기 전이라도 마우스를 움직이면 상태를 안다', async () => {
+    await setup(true);
+    fireEvent.mouseMove(window, { modifierCapsLock: true });
+    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
+  });
+
+  it('창이 포커스를 잃으면 안내를 지운다', async () => {
+    await setup(true);
+    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
+    fireEvent.blur(window);
+    expect(screen.queryByText('Caps Lock이 켜져 있습니다')).toBeNull();
+  });
+
+  it('처음 설정 화면에서도 보인다', async () => {
+    await setup(false);
+    fireEvent.keyDown(window, { key: 'A', modifierCapsLock: true });
+    expect(screen.getByText('Caps Lock이 켜져 있습니다')).toBeTruthy();
+  });
+});
````

##### `frontend/src/styles/app.css`

````diff
--- a/frontend/src/styles/app.css
+++ b/frontend/src/styles/app.css
@@ -177,6 +177,7 @@
 .usage-item { display: flex; flex-direction: column; gap: 3px; padding: 10px 12px; }
 .usage-item .usage-title { font-weight: 600; }
 .usage-item .usage-desc { font-size: 12px; color: var(--fg-2); font-weight: 400; line-height: 1.5; }
+.caps-note { display: flex; align-items: center; gap: 6px; padding: 6px 10px; border-radius: 6px; background: var(--warn-soft); color: var(--warn); font-size: 12px; font-weight: 600; }
 .lock-actions { display: flex; flex-direction: column; gap: 10px; }
 .lock-actions .btn { justify-content: center; }
 .center-row { display: flex; justify-content: center; }
````

##### `locales/de.json`

````diff
--- a/locales/de.json
+++ b/locales/de.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Letzte Anordnung wiederherstellen",
   "conn.restoreLayoutDesc": "Speichert beim Beenden offene Tabs und die Anordnung und verbindet beim nächsten Start erneut. Passwörter und Bildschirminhalte werden nicht gespeichert, einmalige SSH-Verbindungen außerhalb der Liste nicht wiederhergestellt",
   "conn.sessionLog": "Terminalausgabe protokollieren",
-  "conn.sessionLogDesc": "Speichert die Terminalausgabe als Textdatei (ab Verbindungen, die danach geöffnet werden)",
+  "conn.sessionLogDesc": "Speichert die Terminalausgabe als Textdatei (offene Verbindungen werden ab dem Einschalten mit aufgezeichnet)",
   "conn.logKeepDays": "Sitzungsprotokolle aufbewahren (Tage)",
   "conn.logKeepDaysHelp": "Sitzungsprotokolle, die älter als diese Anzahl Tage sind, werden automatisch gelöscht. 0 = nie löschen.",
+  "conn.logSyncSec": "Aktualisierungsintervall der Protokolldatei (Sekunden)",
+  "conn.logSyncSecHelp": "Wie oft ein laufendes Protokoll vollständig auf den Datenträger geschrieben wird (0–5). Erst danach zeigt der Explorer Dateigröße und Änderungszeit an. Bei 0 wird jede Ausgabe sofort geschrieben und auf den Datenträger übertragen, was die Datenträgerlast erhöht. Sonst wird jede Sekunde in die Datei geschrieben.",
   "conn.openLogs": "Protokollordner öffnen",
   "backup.saved": "Sicherung gespeichert: {path}",
   "backup.import": "Sicherung importieren",
@@ -523,6 +525,7 @@
   "lock.open": "Entsperren",
   "lock.osUnlock": "Mit Windows-Konto entsperren",
   "lock.forgot": "Passwort vergessen? Mit Wiederherstellungscode zurücksetzen",
+  "lock.capsLock": "Die Feststelltaste ist aktiviert",
   "setup.title": "DariTerm einrichten",
   "setup.desc": "Server-IDs, Passwörter und Schlüssel werden mit diesem App-Sperrpasswort per AES-256 verschlüsselt und nur auf diesem PC gespeichert. Das Passwort wird nirgendwohin gesendet.",
   "setup.pw": "App-Sperrpasswort (mindestens 8 Zeichen)",
````

##### `locales/en.json`

````diff
--- a/locales/en.json
+++ b/locales/en.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Restore last layout",
   "conn.restoreLayoutDesc": "Saves open tabs and pane layout on exit and reconnects to the same servers next time. Passwords and screen contents are not saved, and one-off SSH connections not in the list are not restored",
   "conn.sessionLog": "Log terminal output",
-  "conn.sessionLogDesc": "Saves terminal output to text files (from connections opened after turning this on)",
+  "conn.sessionLogDesc": "Saves terminal output to text files (open connections start recording when you turn this on)",
   "conn.logKeepDays": "Keep session logs (days)",
   "conn.logKeepDaysHelp": "Session log files older than this many days are deleted automatically. 0 keeps them forever.",
+  "conn.logSyncSec": "Log file update interval (seconds)",
+  "conn.logSyncSecHelp": "How often a log being recorded is flushed all the way to disk (0–5). File Explorer shows the file size and modified time only after such a flush. 0 writes and flushes every piece of output immediately, which increases disk load. Otherwise output is written to the file every second.",
   "conn.openLogs": "Open log folder",
   "backup.saved": "Backup saved: {path}",
   "backup.import": "Import backup",
@@ -523,6 +525,7 @@
   "lock.open": "Unlock",
   "lock.osUnlock": "Unlock with Windows account",
   "lock.forgot": "Forgot the password? Reset with recovery code",
+  "lock.capsLock": "Caps Lock is on",
   "setup.title": "DariTerm setup",
   "setup.desc": "Server IDs, passwords and keys are encrypted with AES-256 using this app lock password and stored only on this PC. The app lock password is never sent anywhere.",
   "setup.pw": "App lock password (at least 8 characters)",
````

##### `locales/es.json`

````diff
--- a/locales/es.json
+++ b/locales/es.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Restaurar la última disposición",
   "conn.restoreLayoutDesc": "Guarda las pestañas y la disposición al salir y vuelve a conectar a los mismos servidores la próxima vez. No guarda contraseñas ni el contenido de la pantalla, y no restaura conexiones SSH puntuales que no estén en la lista",
   "conn.sessionLog": "Registrar la salida del terminal",
-  "conn.sessionLogDesc": "Guarda la salida del terminal en archivos de texto (desde las conexiones abiertas después)",
+  "conn.sessionLogDesc": "Guarda la salida del terminal en archivos de texto (las conexiones abiertas se registran desde que se activa)",
   "conn.logKeepDays": "Conservar registros de sesión (días)",
   "conn.logKeepDaysHelp": "Los registros de sesión con más días que este número se eliminan automáticamente. 0 = no eliminar nunca.",
+  "conn.logSyncSec": "Intervalo de actualización del archivo de registro (segundos)",
+  "conn.logSyncSecHelp": "Cada cuánto se vuelca al disco un registro en curso (0–5). El Explorador solo muestra el tamaño y la fecha de modificación después de ese volcado. Con 0, cada salida se escribe y se vuelca al disco de inmediato, lo que aumenta la carga del disco. En otro caso, se escribe en el archivo cada segundo.",
   "conn.openLogs": "Abrir carpeta de registros",
   "backup.saved": "Copia guardada: {path}",
   "backup.import": "Importar copia de seguridad",
@@ -523,6 +525,7 @@
   "lock.open": "Desbloquear",
   "lock.osUnlock": "Desbloquear con la cuenta de Windows",
   "lock.forgot": "¿Olvidó la contraseña? Restablecer con el código de recuperación",
+  "lock.capsLock": "Bloq Mayús está activado",
   "setup.title": "Configuración inicial de DariTerm",
   "setup.desc": "Los ID, contraseñas y claves de los servidores se cifran con AES-256 usando esta contraseña y se guardan solo en este PC. La contraseña nunca se envía a ningún sitio.",
   "setup.pw": "Contraseña de bloqueo (mínimo 8 caracteres)",
````

##### `locales/fr.json`

````diff
--- a/locales/fr.json
+++ b/locales/fr.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Restaurer la dernière disposition",
   "conn.restoreLayoutDesc": "Enregistre les onglets et la disposition à la fermeture et se reconnecte aux mêmes serveurs au prochain lancement. Les mots de passe et le contenu de l’écran ne sont pas enregistrés, et les connexions SSH ponctuelles hors liste ne sont pas restaurées",
   "conn.sessionLog": "Journaliser la sortie du terminal",
-  "conn.sessionLogDesc": "Enregistre la sortie du terminal dans des fichiers texte (pour les connexions ouvertes ensuite)",
+  "conn.sessionLogDesc": "Enregistre la sortie du terminal dans des fichiers texte (les connexions ouvertes sont enregistrées dès l'activation)",
   "conn.logKeepDays": "Conserver les journaux de session (jours)",
   "conn.logKeepDaysHelp": "Les journaux de session plus anciens que ce nombre de jours sont supprimés automatiquement. 0 = ne jamais supprimer.",
+  "conn.logSyncSec": "Intervalle de mise à jour du fichier journal (secondes)",
+  "conn.logSyncSecHelp": "Fréquence à laquelle un journal en cours d'enregistrement est écrit jusqu'au disque (0–5). L'Explorateur n'affiche la taille et la date de modification qu'après cette écriture. Avec 0, chaque sortie est écrite et vidée sur le disque immédiatement, ce qui augmente la charge du disque. Sinon, l'écriture dans le fichier a lieu chaque seconde.",
   "conn.openLogs": "Ouvrir le dossier des journaux",
   "backup.saved": "Sauvegarde enregistrée : {path}",
   "backup.import": "Importer une sauvegarde",
@@ -523,6 +525,7 @@
   "lock.open": "Déverrouiller",
   "lock.osUnlock": "Déverrouiller avec le compte Windows",
   "lock.forgot": "Mot de passe oublié ? Réinitialiser avec le code de récupération",
+  "lock.capsLock": "Le verrouillage des majuscules est activé",
   "setup.title": "Configuration de DariTerm",
   "setup.desc": "Les identifiants, mots de passe et clés des serveurs sont chiffrés en AES-256 avec ce mot de passe et stockés uniquement sur ce PC. Le mot de passe n’est jamais transmis.",
   "setup.pw": "Mot de passe de verrouillage (8 caractères min.)",
````

##### `locales/ja.json`

````diff
--- a/locales/ja.json
+++ b/locales/ja.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "前回の画面配置を復元",
   "conn.restoreLayoutDesc": "終了時に開いているタブとペイン配置を保存し、次回起動時に同じサーバーへ再接続します。パスワードや画面の内容は保存せず、一覧にない一時的な SSH 接続は復元しません",
   "conn.sessionLog": "ターミナル出力の記録",
-  "conn.sessionLogDesc": "ターミナル出力をテキストファイルに残します（オンにした後に開く接続から）",
+  "conn.sessionLogDesc": "ターミナル出力をテキストファイルに残します（オンにすると開いている接続もその時点から記録）",
   "conn.logKeepDays": "ログの保存期間 (日)",
   "conn.logKeepDaysHelp": "この日数より古いセッションログファイルを自動で削除します。0 の場合は削除しません。",
+  "conn.logSyncSec": "ログファイルの更新間隔（秒）",
+  "conn.logSyncSecHelp": "記録中のログをディスクまで書き出す間隔です（0～5）。書き出すまでエクスプローラーにはファイルサイズと更新日時が反映されません。0 にすると出力を受け取るたびに書き込んで書き出すため、ディスク負荷が大きくなります。0 以外ではファイルへの書き込みは 1 秒ごとです。",
   "conn.openLogs": "ログフォルダーを開く",
   "backup.saved": "バックアップを保存しました: {path}",
   "backup.import": "バックアップを読み込む",
@@ -523,6 +525,7 @@
   "lock.open": "開く",
   "lock.osUnlock": "Windows アカウントでロック解除",
   "lock.forgot": "パスワードをお忘れですか？復旧コードで再設定",
+  "lock.capsLock": "Caps Lock がオンになっています",
   "setup.title": "DariTerm の初期設定",
   "setup.desc": "サーバーの ID・パスワード・キーはこのアプリロック用パスワードで AES-256 暗号化され、この PC にのみ保存されます。アプリロック用パスワードはどこにも送信されません。",
   "setup.pw": "アプリロック用パスワード（8文字以上）",
````

##### `locales/ko.json`

````diff
--- a/locales/ko.json
+++ b/locales/ko.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "마지막 화면 배치 복원",
   "conn.restoreLayoutDesc": "종료할 때 열린 탭과 창 배치를 저장하고, 다음 실행 때 같은 서버로 다시 연결합니다. 비밀번호·화면 내용은 저장하지 않으며, 목록에 없는 일회성 SSH 연결은 복원하지 않습니다",
   "conn.sessionLog": "터미널 출력 기록",
-  "conn.sessionLogDesc": "터미널 출력을 텍스트 파일로 남깁니다 (켠 뒤 새로 여는 연결부터)",
+  "conn.sessionLogDesc": "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)",
   "conn.logKeepDays": "로그 보관 기간 (일)",
   "conn.logKeepDaysHelp": "이 일수보다 오래된 세션 로그 파일을 자동으로 지웁니다. 0이면 지우지 않습니다.",
+  "conn.logSyncSec": "로그 파일 갱신 주기 (초)",
+  "conn.logSyncSecHelp": "기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다.",
   "conn.openLogs": "로그 폴더 열기",
   "backup.saved": "백업 저장: {path}",
   "backup.import": "백업 가져오기",
@@ -523,6 +525,7 @@
   "lock.open": "열기",
   "lock.osUnlock": "Windows 계정으로 잠금 해제",
   "lock.forgot": "비밀번호를 잊으셨나요? 복구 코드로 재설정",
+  "lock.capsLock": "Caps Lock이 켜져 있습니다",
   "setup.title": "DariTerm 처음 설정",
   "setup.desc": "서버 ID·비밀번호·키는 이 앱 잠금 비밀번호로 AES-256 암호화되어 이 PC에만 저장됩니다. 앱 잠금 비밀번호는 어디에도 전송되지 않습니다.",
   "setup.pw": "앱 잠금 비밀번호 (8자 이상)",
````

##### `locales/pt.json`

````diff
--- a/locales/pt.json
+++ b/locales/pt.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Restaurar o último layout",
   "conn.restoreLayoutDesc": "Salva abas e layout ao sair e reconecta aos mesmos servidores na próxima vez. Senhas e conteúdo da tela não são salvos, e conexões SSH avulsas fora da lista não são restauradas",
   "conn.sessionLog": "Registrar a saída do terminal",
-  "conn.sessionLogDesc": "Salva a saída do terminal em arquivos de texto (a partir das conexões abertas depois)",
+  "conn.sessionLogDesc": "Salva a saída do terminal em arquivos de texto (as conexões abertas passam a ser registradas ao ativar)",
   "conn.logKeepDays": "Manter logs de sessão (dias)",
   "conn.logKeepDaysHelp": "Arquivos de log de sessão mais antigos que este número de dias são excluídos automaticamente. 0 = nunca excluir.",
+  "conn.logSyncSec": "Intervalo de atualização do arquivo de log (segundos)",
+  "conn.logSyncSecHelp": "Com que frequência um log em gravação é descarregado até o disco (0–5). O Explorador só mostra o tamanho e a data de modificação depois dessa descarga. Com 0, cada saída é gravada e descarregada no disco imediatamente, o que aumenta a carga do disco. Caso contrário, a gravação no arquivo ocorre a cada segundo.",
   "conn.openLogs": "Abrir pasta de logs",
   "backup.saved": "Backup salvo: {path}",
   "backup.import": "Importar backup",
@@ -523,6 +525,7 @@
   "lock.open": "Desbloquear",
   "lock.osUnlock": "Desbloquear com a conta do Windows",
   "lock.forgot": "Esqueceu a senha? Redefinir com o código de recuperação",
+  "lock.capsLock": "Caps Lock está ativado",
   "setup.title": "Configuração inicial do DariTerm",
   "setup.desc": "IDs, senhas e chaves dos servidores são criptografados com AES-256 usando esta senha e salvos apenas neste PC. A senha nunca é enviada a lugar nenhum.",
   "setup.pw": "Senha de bloqueio (mínimo 8 caracteres)",
````

##### `locales/ru.json`

````diff
--- a/locales/ru.json
+++ b/locales/ru.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "Восстанавливать последнюю раскладку",
   "conn.restoreLayoutDesc": "При выходе сохраняет вкладки и раскладку и при следующем запуске переподключается к тем же серверам. Пароли и содержимое экрана не сохраняются, разовые SSH-подключения вне списка не восстанавливаются",
   "conn.sessionLog": "Записывать вывод терминала",
-  "conn.sessionLogDesc": "Сохраняет вывод терминала в текстовые файлы (для подключений, открытых после включения)",
+  "conn.sessionLogDesc": "Сохраняет вывод терминала в текстовые файлы (открытые подключения записываются с момента включения)",
   "conn.logKeepDays": "Хранить журналы сеансов (дней)",
   "conn.logKeepDaysHelp": "Файлы журналов сеансов старше указанного числа дней удаляются автоматически. 0 — не удалять.",
+  "conn.logSyncSec": "Интервал обновления файла журнала (секунды)",
+  "conn.logSyncSecHelp": "Как часто записываемый журнал сбрасывается на диск (0–5). Проводник показывает размер файла и время изменения только после такого сброса. При 0 каждый вывод сразу записывается и сбрасывается на диск, что увеличивает нагрузку на диск. Иначе запись в файл выполняется раз в секунду.",
   "conn.openLogs": "Открыть папку журналов",
   "backup.saved": "Резервная копия сохранена: {path}",
   "backup.import": "Импорт резервной копии",
@@ -523,6 +525,7 @@
   "lock.open": "Открыть",
   "lock.osUnlock": "Разблокировать учётной записью Windows",
   "lock.forgot": "Забыли пароль? Сбросить кодом восстановления",
+  "lock.capsLock": "Включён Caps Lock",
   "setup.title": "Первичная настройка DariTerm",
   "setup.desc": "Логины, пароли и ключи серверов шифруются AES-256 этим паролем и хранятся только на этом ПК. Пароль блокировки никуда не передаётся.",
   "setup.pw": "Пароль блокировки (не менее 8 символов)",
````

##### `locales/tw.json`

````diff
--- a/locales/tw.json
+++ b/locales/tw.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "還原上次的版面配置",
   "conn.restoreLayoutDesc": "結束時儲存開啟的分頁與窗格配置，下次啟動時重新連線到相同伺服器。不儲存密碼與畫面內容，也不還原清單外的一次性 SSH 連線",
   "conn.sessionLog": "記錄終端機輸出",
-  "conn.sessionLogDesc": "將終端機輸出存成文字檔（從開啟後新建的連線開始）",
+  "conn.sessionLogDesc": "將終端機輸出存成文字檔（開啟後，已開啟的連線也從此刻開始記錄）",
   "conn.logKeepDays": "記錄保留天數",
   "conn.logKeepDaysHelp": "自動刪除早於此天數的工作階段記錄檔。0 表示不刪除。",
+  "conn.logSyncSec": "記錄檔更新間隔（秒）",
+  "conn.logSyncSecHelp": "將記錄中的記錄檔完全寫入磁碟的間隔（0～5）。寫入磁碟後，檔案總管才會顯示檔案大小與修改時間。設為 0 時，每收到輸出就立即寫入並落盤，磁碟負載會增加。不為 0 時，每秒寫入檔案一次。",
   "conn.openLogs": "開啟記錄資料夾",
   "backup.saved": "備份已儲存：{path}",
   "backup.import": "匯入備份",
@@ -523,6 +525,7 @@
   "lock.open": "開啟",
   "lock.osUnlock": "使用 Windows 帳戶解鎖",
   "lock.forgot": "忘記密碼？使用復原碼重設",
+  "lock.capsLock": "Caps Lock 已開啟",
   "setup.title": "DariTerm 初始設定",
   "setup.desc": "伺服器 ID、密碼與金鑰會以此應用程式鎖定密碼進行 AES-256 加密，僅儲存在本機。應用程式鎖定密碼不會傳送到任何地方。",
   "setup.pw": "應用程式鎖定密碼（至少 8 個字元）",
````

##### `locales/zh.json`

````diff
--- a/locales/zh.json
+++ b/locales/zh.json
@@ -166,9 +166,11 @@
   "conn.restoreLayout": "恢复上次的布局",
   "conn.restoreLayoutDesc": "退出时保存打开的标签页和窗格布局，下次启动时重新连接相同服务器。不保存密码和屏幕内容，也不恢复列表外的一次性 SSH 连接",
   "conn.sessionLog": "记录终端输出",
-  "conn.sessionLogDesc": "将终端输出保存为文本文件（从开启后新建的连接开始）",
+  "conn.sessionLogDesc": "将终端输出保存为文本文件（开启后，已打开的连接也从此刻开始记录）",
   "conn.logKeepDays": "日志保留天数",
   "conn.logKeepDaysHelp": "自动删除早于此天数的会话日志文件。0 表示不删除。",
+  "conn.logSyncSec": "日志文件更新间隔（秒）",
+  "conn.logSyncSecHelp": "将正在记录的日志完全写入磁盘的间隔（0～5）。写入磁盘后，资源管理器才会显示文件大小和修改时间。设为 0 时，每收到输出就立即写入并落盘，磁盘负载会增大。不为 0 时，每秒写入文件一次。",
   "conn.openLogs": "打开日志文件夹",
   "backup.saved": "备份已保存：{path}",
   "backup.import": "导入备份",
@@ -523,6 +525,7 @@
   "lock.open": "打开",
   "lock.osUnlock": "使用 Windows 账户解锁",
   "lock.forgot": "忘记密码？使用恢复码重置",
+  "lock.capsLock": "Caps Lock 已开启",
   "setup.title": "DariTerm 初始设置",
   "setup.desc": "服务器 ID、密码和密钥会用此应用锁密码进行 AES-256 加密，仅保存在本机。应用锁密码不会发送到任何地方。",
   "setup.pw": "应用锁密码（至少 8 个字符）",
````

#### 묶음 B — 세션·로그 (4·5·6번)

##### `internal/session/types.go`

````diff
--- a/internal/session/types.go
+++ b/internal/session/types.go
@@ -52,6 +52,8 @@
 	DataDir       string
 	LogEnabled    bool
 	LogAlias      string
+	// LogSyncSec 은 세션 로그의 내려 쓰기 주기(초, 0~5)다. 0 이면 받는 즉시 쓰고 내려 쓴다 (session.md §6).
+	LogSyncSec int
 	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
 	StartupInput string
 	// StartupDelay 는 접속 후 StartupInput 을 보내기까지의 대기다 (tmux 500ms, 자동 실행 명령 400ms).
````

##### `internal/session/log.go`

````diff
--- a/internal/session/log.go
+++ b/internal/session/log.go
@@ -19,8 +19,13 @@
 	G_ciLogNameMaxTries = 1000
 	// G_ciLogBufferBytes 는 로그 쓰기 버퍼 크기다. G_cdurLogFlush 마다, 그리고 닫을 때 비운다.
 	G_ciLogBufferBytes = 64 * 1024
-	// G_cdurLogFlush 는 버퍼에 쌓인 로그를 디스크에 쓰는 주기다.
+	// G_cdurLogFlush 는 버퍼에 쌓인 로그를 파일에 쓰는 주기다 (내려 쓰기 주기가 0 이 아닐 때).
 	G_cdurLogFlush = time.Second
+	// 내려 쓰기(파일을 연 채로 디스크까지 쓰기, fsync) 주기의 범위와 기본값(초)이다 (설정 sessionLogSyncSec, session.md §6).
+	// 0 이면 출력을 받는 즉시 파일에 쓰고 내려 쓴다. Windows 탐색기는 내려 쓰거나 닫아야 크기·수정 시각을 갱신한다.
+	G_ciLogSyncSecMin     = 0
+	G_ciLogSyncSecMax     = 5
+	G_ciLogSyncSecDefault = 2
 	// g_ciLogCarryMax 는 미완성 ANSI 시퀀스로 보관하는 최대 길이다. 넘으면 시퀀스가 아니라고 보고 그대로 기록한다.
 	g_ciLogCarryMax = 256
 	// g_ciLogAliasMaxRunes 는 로그 파일 이름에 쓰는 별칭의 최대 길이다 (경로 260자 한도 대비).
@@ -51,6 +56,17 @@
 	return bOpen
 }
 
+// ClampLogSyncSec 는 내려 쓰기 주기를 허용 범위(0~5초)로 맞춘다.
+func ClampLogSyncSec(iSec int) int {
+	if iSec < G_ciLogSyncSecMin {
+		return G_ciLogSyncSecMin
+	}
+	if iSec > G_ciLogSyncSecMax {
+		return G_ciLogSyncSecMax
+	}
+	return iSec
+}
+
 // StripForLog 는 로그에 남길 텍스트를 만든다: ANSI 시퀀스와 CR 을 제거한다.
 func StripForLog(szText string) string {
 	szOut := g_reAnsi.ReplaceAllString(szText, "")
@@ -81,14 +97,18 @@
 }
 
 // Logger 는 세션 로그 파일에 쓴다. ANSI 시퀀스가 묶음 경계에서 잘려도 지울 수 있도록 꼬리를 보관하고,
-// 디스크 쓰기는 버퍼에 모아 G_cdurLogFlush 마다 한 번 한다 (읽기 고루틴이 디스크를 기다리지 않게).
+// 파일 쓰기는 버퍼에 모아 G_cdurLogFlush 마다 한 번 하고, 내려 쓰기는 iSyncSec 초마다 한 번 한다
+// (읽기 고루틴이 디스크를 기다리지 않게). iSyncSec 이 0 이면 받는 즉시 쓰고 내려 쓴다.
 type Logger struct {
 	mtxData    sync.Mutex
 	szPath     string // 열린 파일의 절대 경로 (열린 로그 집합의 키)
 	pFile      *os.File
 	pWriter    *bufio.Writer
 	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
-	timerFlush *time.Timer // 버퍼에 내용이 있을 때만 걸린다
+	timerFlush *time.Timer // 버퍼에 내용이 있거나 아직 내려 쓰지 않은 내용이 있을 때만 걸린다
+	iSyncSec   int         // 내려 쓰기 주기(초). 0 = 즉시
+	bUnsynced  bool        // 파일에는 썼지만 아직 내려 쓰지 않은 내용이 있다
+	tLastSync  time.Time   // 마지막으로 내려 쓴 시각 (처음에는 연 시각)
 }
 
 // OpenLog 는 데이터 폴더의 logs 아래 이 세션만의 로그 파일을 새로 만든다 (소유자 전용 권한, session.md §6).
@@ -110,7 +130,8 @@
 			g_mtxOpenLogs.Lock()
 			g_mOpenLogs[szPath] = struct{}{}
 			g_mtxOpenLogs.Unlock()
-			return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes)}, nil
+			return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes),
+				iSyncSec: G_ciLogSyncSecDefault, tLastSync: time.Now()}, nil
 		}
 		if !errors.Is(errOpen, fs.ErrExist) {
 			return nil, errOpen
@@ -165,18 +186,65 @@
 		return
 	}
 	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
+	if pLogger.iSyncSec == 0 {
+		// 즉시 모드: 부하가 걸리더라도 받는 대로 파일에 쓰고 내려 쓴다 (사용자가 고른 동작).
+		pLogger.syncNowLocked()
+		return
+	}
 	if pLogger.timerFlush == nil {
 		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
 	}
 }
 
-// flushTimer 는 주기 flush 다. 닫힌 뒤에는 아무 일도 하지 않는다.
-func (pLogger *Logger) flushTimer() {
+// syncNowLocked 는 mtxData 를 잡은 상태에서 버퍼를 파일에 쓰고 곧바로 내려 쓴다.
+func (pLogger *Logger) syncNowLocked() {
+	if pLogger.pWriter == nil {
+		return
+	}
+	_ = pLogger.pWriter.Flush()
+	_ = pLogger.pFile.Sync()
+	pLogger.bUnsynced = false
+	pLogger.tLastSync = time.Now()
+}
+
+// SetSyncSec 은 내려 쓰기 주기(초, 0~5)를 바꾼다. 0 으로 바꾸면 쌓인 것을 곧바로 쓰고 내려 쓴다.
+func (pLogger *Logger) SetSyncSec(iSec int) {
 	pLogger.mtxData.Lock()
 	defer pLogger.mtxData.Unlock()
+	pLogger.iSyncSec = ClampLogSyncSec(iSec)
+	if pLogger.iSyncSec == 0 {
+		pLogger.syncNowLocked()
+	}
+}
+
+// flushTimer 는 주기 처리다: 버퍼를 파일에 쓰고, 내려 쓰기 주기가 됐으면 내려 쓴다.
+// 아직 내려 쓰지 않은 내용이 남아 있으면 다음 주기에 다시 돈다. 닫힌 뒤에는 아무 일도 하지 않는다.
+func (pLogger *Logger) flushTimer() {
+	pLogger.mtxData.Lock()
 	pLogger.timerFlush = nil
-	if pLogger.pWriter != nil {
+	if pLogger.pWriter == nil {
+		pLogger.mtxData.Unlock()
+		return
+	}
+	if pLogger.pWriter.Buffered() > 0 {
 		_ = pLogger.pWriter.Flush()
+		pLogger.bUnsynced = true
+	}
+	var pSync *os.File
+	if pLogger.bUnsynced {
+		if time.Since(pLogger.tLastSync) >= time.Duration(pLogger.iSyncSec)*time.Second {
+			pSync = pLogger.pFile
+			pLogger.bUnsynced = false
+			pLogger.tLastSync = time.Now()
+		} else {
+			pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
+		}
+	}
+	pLogger.mtxData.Unlock()
+	if pSync != nil {
+		// 내려 쓰기는 잠금 밖에서 한다: 디스크를 기다리는 동안 출력 기록(Write)이 막히지 않는다.
+		// 그사이 닫혔으면 오류가 날 뿐이다(닫기가 이미 다 썼다).
+		_ = pSync.Sync()
 	}
 }
 
````

##### `internal/session/log_test.go`

````diff
--- a/internal/session/log_test.go
+++ b/internal/session/log_test.go
@@ -89,6 +89,79 @@
 	pSecond.Close()
 }
 
+// 열려 있는 세션에서 기록을 켜면 그때부터 새 파일에 쓰고, 끄면 파일을 닫고 더 쓰지 않는다 (session.md §6, D99).
+func TestSetLoggingOnOpenSession(t *testing.T) {
+	szDir := t.TempDir()
+	pFake := newFakeTransport()
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
+	cfg.DataDir = szDir
+	cfg.LogAlias = "web" // LogEnabled 는 꺼진 채로 연다
+	info, _ := pManager.Open(cfg)
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
+	fnFiles := func() []string {
+		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
+		return slFiles
+	}
+
+	pFake.chIn <- []byte("before\r\n")
+	waitUntil(t, func() bool { return strings.Contains(rec.output(), "before") }, time.Second)
+	if len(fnFiles()) != 0 {
+		t.Fatalf("꺼져 있으면 파일이 없어야 함: %v", fnFiles())
+	}
+
+	pManager.SetLogging(true)
+	pManager.SetLogging(true) // 두 번 켜도 파일은 하나
+	if len(fnFiles()) != 1 {
+		t.Fatalf("켜면 파일 하나가 생겨야 함: %v", fnFiles())
+	}
+	szPath := fnFiles()[0]
+	if !IsLogOpen(szPath) {
+		t.Fatal("기록 중인 파일로 등록돼야 함")
+	}
+	pFake.chIn <- []byte("during\r\n")
+	waitUntil(t, func() bool { return strings.Contains(rec.output(), "during") }, time.Second)
+
+	pManager.SetLogging(false)
+	if IsLogOpen(szPath) {
+		t.Fatal("끄면 파일을 닫아야 함")
+	}
+	pFake.chIn <- []byte("after\r\n")
+	waitUntil(t, func() bool { return strings.Contains(rec.output(), "after") }, time.Second)
+
+	bData, errRead := os.ReadFile(szPath)
+	if errRead != nil {
+		t.Fatal(errRead)
+	}
+	szLog := string(bData)
+	if !strings.Contains(szLog, "during") || strings.Contains(szLog, "before") || strings.Contains(szLog, "after") {
+		t.Fatalf("켜져 있던 동안의 출력만 있어야 함: %q", szLog)
+	}
+
+	pManager.SetLogging(true) // 다시 켜면 새 파일
+	if len(fnFiles()) != 2 {
+		t.Fatalf("다시 켜면 파일이 하나 더 생겨야 함: %v", fnFiles())
+	}
+}
+
+// 닫힌 세션은 기록을 켜도 파일을 만들지 않는다.
+func TestSetLoggingIgnoresClosedSession(t *testing.T) {
+	szDir := t.TempDir()
+	cfg := sshConfig(func(func(string)) (Transport, error) { return newFakeTransport(), nil }, false)
+	cfg.DataDir = szDir
+	cfg.LogAlias = "web"
+	pSession := newSession(cfg, func(string, ...any) {})
+	pSession.start()
+	pSession.Close()
+	pSession.SetLogging(true)
+	if slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "*")); len(slFiles) != 0 {
+		t.Fatalf("닫힌 세션은 파일을 만들지 않음: %v", slFiles)
+	}
+}
+
 // 묶음 경계에서 잘린 ANSI 시퀀스도 지워진다 (통합 계획 §3).
 func TestLogStripsSequenceSplitAcrossWrites(t *testing.T) {
 	szDir := t.TempDir()
@@ -204,3 +277,94 @@
 		t.Fatalf("cp949 확장 한글: %q", szGot)
 	}
 }
+
+// 내려 쓰기 주기 0: 받는 즉시 파일에 쓴다(1초 주기를 기다리지 않는다) (session.md §6, D100).
+func TestLogSyncZeroWritesImmediately(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	defer pLogger.Close()
+	pLogger.SetSyncSec(0)
+	pLogger.Write("now\n")
+	bData, _ := os.ReadFile(pLogger.Path())
+	if string(bData) != "now\n" {
+		t.Fatalf("즉시 모드는 바로 파일에 있어야 함: %q", string(bData))
+	}
+}
+
+// 버퍼에 쌓인 채로 주기를 0 으로 바꾸면 곧바로 파일에 쓴다. 0 이 아니면 지금처럼 약 1초 뒤에 쓴다.
+func TestLogSyncSecSwitch(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	defer pLogger.Close()
+	pLogger.SetSyncSec(3)
+	pLogger.Write("buffered\n")
+	if bData, _ := os.ReadFile(pLogger.Path()); len(bData) != 0 {
+		t.Fatalf("0 이 아니면 바로 쓰지 않음: %q", string(bData))
+	}
+	pLogger.SetSyncSec(0)
+	if bData, _ := os.ReadFile(pLogger.Path()); string(bData) != "buffered\n" {
+		t.Fatalf("0 으로 바꾸면 곧바로 써야 함: %q", string(bData))
+	}
+	pLogger.SetSyncSec(5)
+	pLogger.Write("later\n")
+	waitUntil(t, func() bool {
+		bData, _ := os.ReadFile(pLogger.Path())
+		return string(bData) == "buffered\nlater\n"
+	}, 2*time.Second)
+}
+
+// 범위 밖 값은 0~5 로 맞춘다.
+func TestClampLogSyncSec(t *testing.T) {
+	for iIn, iWant := range map[int]int{-3: 0, 0: 0, 2: 2, 5: 5, 9: 5} {
+		if iGot := ClampLogSyncSec(iIn); iGot != iWant {
+			t.Fatalf("ClampLogSyncSec(%d) = %d, 기대 %d", iIn, iGot, iWant)
+		}
+	}
+}
+
+// 세션에 준 주기는 기록을 켤 때 로그에 전해지고, 바꾸면 기록 중인 로그에도 곧바로 적용된다.
+func TestSessionLogSyncSecApplies(t *testing.T) {
+	szDir := t.TempDir()
+	pFake := newFakeTransport()
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
+	cfg.DataDir = szDir
+	cfg.LogAlias = "web"
+	cfg.LogEnabled = true
+	cfg.LogSyncSec = 4
+	info, _ := pManager.Open(cfg)
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
+	if iGot := pSession.logSyncSecForTest(); iGot != 4 {
+		t.Fatalf("열 때 준 주기가 로그에 전해져야 함: %d", iGot)
+	}
+	pManager.SetLogSyncSec(0)
+	if iGot := pSession.logSyncSecForTest(); iGot != 0 {
+		t.Fatalf("바꾼 주기가 기록 중인 로그에 적용돼야 함: %d", iGot)
+	}
+	pManager.SetLogging(false)
+	pManager.SetLogSyncSec(1)
+	pManager.SetLogging(true)
+	if iGot := pSession.logSyncSecForTest(); iGot != 1 {
+		t.Fatalf("꺼져 있을 때 바꾼 주기도 다시 켤 때 써야 함: %d", iGot)
+	}
+}
+
+// logSyncSecForTest 는 지금 기록 중인 로그의 내려 쓰기 주기다 (기록 중이 아니면 -1).
+func (pSession *Session) logSyncSecForTest() int {
+	pSession.mtxData.Lock()
+	pLogger := pSession.logger
+	pSession.mtxData.Unlock()
+	if pLogger == nil {
+		return -1
+	}
+	pLogger.mtxData.Lock()
+	defer pLogger.mtxData.Unlock()
+	return pLogger.iSyncSec
+}
````

##### `internal/session/control.go`

````diff
--- a/internal/session/control.go
+++ b/internal/session/control.go
@@ -1,7 +1,9 @@
 package session
 
 import (
+	"os"
 	"strings"
+	"time"
 )
 
 // Snapshot 은 현재 세션 정보의 복사본을 돌려준다.
@@ -17,6 +19,9 @@
 	pSession.mtxData.Lock()
 	transport := pSession.transport
 	szState := pSession.info.M_szState
+	if transport != nil && szText != "" {
+		pSession.tLastInput = time.Now() // 서버가 셸을 끝냈을 때 사용자가 끝낸 것인지 가리는 데 쓴다 (session.md §4)
+	}
 	pSession.mtxData.Unlock()
 	if transport != nil {
 		_ = transport.Write(Encode(pSession.cfg.Encoding, szText))
@@ -98,6 +103,60 @@
 	pSession.publishState()
 }
 
+// SetLogging 은 이 세션의 출력 기록을 켜거나 끈다 (session.md §6).
+// 켜면 그 순간부터의 출력을 새 로그 파일에 쓴다(이미 기록 중이면 그대로). 끄면 남은 것을 쓰고 파일을 닫는다.
+// 세션을 열 때(start)와 설정 "터미널 출력 기록"을 바꿀 때 부른다. 닫힌 세션은 건드리지 않는다.
+func (pSession *Session) SetLogging(bEnabled bool) {
+	pSession.mtxData.Lock()
+	if pSession.userClosed {
+		pSession.mtxData.Unlock()
+		return
+	}
+	if !bEnabled {
+		pLogger := pSession.logger
+		pSession.logger = nil
+		pSession.mtxData.Unlock()
+		if pLogger != nil {
+			_ = pLogger.Close()
+		}
+		return
+	}
+	bSkip := pSession.logger != nil || pSession.cfg.DataDir == ""
+	pSession.mtxData.Unlock()
+	if bSkip {
+		return
+	}
+	// 파일 만들기는 잠금 밖에서 한다(디스크를 기다리는 동안 출력이 막히지 않게).
+	pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
+	if errLog != nil {
+		return // 열기 실패는 조용히 넘어간다: 기록 없이 세션은 계속 쓴다
+	}
+	pSession.mtxData.Lock()
+	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
+	if !bLost {
+		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
+		pSession.logger = pLogger
+	}
+	pSession.mtxData.Unlock()
+	if bLost {
+		_ = pLogger.Close()
+		_ = os.Remove(pLogger.Path()) // 방금 만든 빈 파일은 남기지 않는다
+	}
+}
+
+// SetLogSyncSec 은 이 세션 로그의 내려 쓰기 주기(초, 0~5)를 바꾼다. 기록 중이면 곧바로 적용하고,
+// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogSyncSec, session.md §6).
+func (pSession *Session) SetLogSyncSec(iSec int) {
+	iSec = ClampLogSyncSec(iSec)
+	pSession.mtxData.Lock()
+	pSession.iLogSyncSec = iSec
+	pLogger := pSession.logger
+	pSession.mtxData.Unlock()
+	if pLogger != nil {
+		pLogger.SetSyncSec(iSec)
+	}
+}
+
 // AckOutput 은 프런트가 출력 묶음 iCount 개를 처리했음을 반영한다 (spec/ipc.md session.ack).
 func (pSession *Session) AckOutput(iCount int) {
 	if iCount <= 0 {
````

##### `internal/session/manager.go`

````diff
--- a/internal/session/manager.go
+++ b/internal/session/manager.go
@@ -90,6 +90,32 @@
 	return iCount
 }
 
+// SetLogging 은 열려 있는 모든 세션의 출력 기록을 켜거나 끈다 (설정 "터미널 출력 기록"을 바꿨을 때, session.md §6).
+func (pManager *Manager) SetLogging(bEnabled bool) {
+	pManager.mtxData.Lock()
+	slSessions := make([]*Session, 0, len(pManager.mapSessions))
+	for _, pSession := range pManager.mapSessions {
+		slSessions = append(slSessions, pSession)
+	}
+	pManager.mtxData.Unlock()
+	for _, pSession := range slSessions {
+		pSession.SetLogging(bEnabled)
+	}
+}
+
+// SetLogSyncSec 은 열려 있는 모든 세션의 로그 내려 쓰기 주기를 바꾼다 (설정 sessionLogSyncSec 을 바꿨을 때).
+func (pManager *Manager) SetLogSyncSec(iSec int) {
+	pManager.mtxData.Lock()
+	slSessions := make([]*Session, 0, len(pManager.mapSessions))
+	for _, pSession := range pManager.mapSessions {
+		slSessions = append(slSessions, pSession)
+	}
+	pManager.mtxData.Unlock()
+	for _, pSession := range slSessions {
+		pSession.SetLogSyncSec(iSec)
+	}
+}
+
 // Count 는 열린 세션 수를 돌려준다 (종료 확인 문구에 쓴다).
 func (pManager *Manager) Count() int {
 	pManager.mtxData.Lock()
````

##### `internal/session/session.go`

````diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -34,10 +34,18 @@
 	batcher      *Batcher
 	credit       *Credit // 출력 흐름 제어 (session.md §8)
 	logger       *Logger
-	dialing      bool // Dialer 를 부르는 중 (같은 세션에 접속이 두 번 겹치지 않게)
-	waitUnlock   bool // 앱 잠금 해제를 기다리는 중 (ErrWaitUnlock, session.md §4)
+	dialing      bool      // Dialer 를 부르는 중 (같은 세션에 접속이 두 번 겹치지 않게)
+	waitUnlock   bool      // 앱 잠금 해제를 기다리는 중 (ErrWaitUnlock, session.md §4)
+	tConnected   time.Time // 지금 연결이 맺어진 시각 (서버 종료가 사용자 종료인지 가릴 때 쓴다)
+	tLastInput   time.Time // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
+	iLogSyncSec  int       // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
 }
 
+// G_cdurUserExitWindow 는 서버가 셸을 끝냈을 때 "사용자가 끝낸 것"으로 보는 시간 창이다 (session.md §4).
+// 끝나기 전 이 시간 안에 사용자 입력이 있었거나, 연결된 지 이 시간이 안 됐으면 사용자가 끝낸 것으로 본다.
+// 그 밖(입력 없이 한참 있다가 끝남)은 서버의 유휴 로그아웃 같은 서버 쪽 종료로 보고 자동 재접속한다.
+const G_cdurUserExitWindow = 10 * time.Second
+
 // newSession 은 세션을 만든다. 접속은 start 에서 시작한다.
 func newSession(cfg OpenConfig, emit EmitFunc) *Session {
 	pSession := &Session{
@@ -45,6 +53,8 @@
 		info:    cfg.Info,
 		emit:    emit,
 		decoder: NewDecoder(cfg.Encoding),
+
+		iLogSyncSec: ClampLogSyncSec(cfg.LogSyncSec),
 	}
 	pSession.info.M_szState = G_cszStateConnecting
 	szId := cfg.Info.M_szId
@@ -59,13 +69,8 @@
 
 // start 는 OpenDelay 뒤에 첫 접속을 시작한다. 로그가 켜져 있으면 연다.
 func (pSession *Session) start() {
-	if pSession.cfg.LogEnabled && pSession.cfg.DataDir != "" {
-		pLogger, errLog := OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())
-		if errLog == nil {
-			pSession.mtxData.Lock()
-			pSession.logger = pLogger
-			pSession.mtxData.Unlock()
-		}
+	if pSession.cfg.LogEnabled {
+		pSession.SetLogging(true)
 	}
 	durDelay := pSession.cfg.OpenDelay
 	if durDelay <= 0 {
@@ -152,7 +157,10 @@
 	pSession.wasConnected = true
 	pSession.info.M_szState = G_cszStateConnected
 	pSession.info.M_szMessage = ""
-	pSession.info.M_i64ConnectedAt = time.Now().UnixMilli()
+	tNow := time.Now()
+	pSession.info.M_i64ConnectedAt = tNow.UnixMilli()
+	pSession.tConnected = tNow
+	pSession.tLastInput = time.Time{}
 	iCols, iRows := pSession.iCols, pSession.iRows
 	pSession.mtxData.Unlock()
 
@@ -269,7 +277,21 @@
 	pSession.publishState()
 }
 
-// onRemoteExited 는 서버가 셸을 정상적으로 끝낸 것을 처리한다: 자동 재접속하지 않고 닫힘 + 안내 두 줄 (session.md §4).
+// userEndedLocked 는 mtxData 를 잡은 상태에서, 방금 끝난 셸을 사용자가 끝낸 것으로 볼지 판단한다 (session.md §4).
+// 셸의 유휴 로그아웃과 사용자의 exit 는 종료 코드로는 구분되지 않는다(둘 다 종료 코드와 함께 끝난다).
+// 유휴 로그아웃은 입력이 한참 없을 때만 일어나므로, 끝나기 직전에 입력이 있었으면 사용자가 끝낸 것이다.
+// 연결하자마자 끝나는 서버(로그인 직후 셸이 끝나는 계정 등)를 계속 다시 접속하지 않도록 그 경우도 여기에 넣는다.
+func (pSession *Session) userEndedLocked(tNow time.Time) bool {
+	if tNow.Sub(pSession.tConnected) < G_cdurUserExitWindow {
+		return true
+	}
+	return !pSession.tLastInput.IsZero() && tNow.Sub(pSession.tLastInput) < G_cdurUserExitWindow
+}
+
+// onRemoteExited 는 서버가 셸을 종료 코드·신호와 함께 끝낸 것을 처리한다 (session.md §4).
+//   - 사용자가 끝낸 것(exit·logout, userEndedLocked)이면 자동 재접속하지 않고 닫힘 + 안내 두 줄.
+//   - 서버 쪽에서 끝낸 것(유휴 로그아웃, 관리자의 세션 종료, 서버 재시작)이면 끊김처럼 자동 재접속을 예약한다.
+//
 // Enter·[재연결] 로 다시 연결하는 동작은 그대로다. 남은 연결(SSH 접속·keepalive·포워딩)은 닫는다.
 func (pSession *Session) onRemoteExited(iGen int) {
 	pSession.mtxData.Lock()
@@ -279,11 +301,17 @@
 	}
 	conn := pSession.transport
 	pSession.transport = nil
-	pSession.attempts = 0
-	pSession.info.M_szState = G_cszStateClosed
-	pSession.info.M_szMessage = ""
 	pSession.writeOutputUnlockedNotice(MsgRemoteExited())
-	pSession.writeOutputUnlockedNotice(MsgGiveUp())
+	bRetry := !pSession.userEndedLocked(time.Now()) &&
+		CanAutoReconnect(pSession.cfg.AutoReconnect, pSession.cfg.IsSerial, true, pSession.attempts)
+	if bRetry {
+		pSession.scheduleRetryLocked()
+	} else {
+		pSession.attempts = 0
+		pSession.info.M_szState = G_cszStateClosed
+		pSession.info.M_szMessage = ""
+		pSession.writeOutputUnlockedNotice(MsgGiveUp())
+	}
 	pSession.mtxData.Unlock()
 	if conn != nil {
 		_ = conn.Close()
````

##### `internal/session/exit_unlock_test.go`

````diff
--- a/internal/session/exit_unlock_test.go
+++ b/internal/session/exit_unlock_test.go
@@ -108,6 +108,82 @@
 	}
 }
 
+// 입력 없이 한참 있다가 서버가 셸을 끝내면(유휴 로그아웃 등) 끊김처럼 자동 재접속한다 (session.md §4, D98).
+func TestRemoteExitAfterIdleAutoReconnects(t *testing.T) {
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	first := newExitingTransport()
+	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
+		if iCall == 1 {
+			return first, nil
+		}
+		return newFakeTransport(), nil
+	}}
+	info, _ := pManager.Open(sshConfig(pDial.dial, true))
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
+	pSession.Input("ls\r")
+	pSession.backdateForTest(time.Minute) // 연결도 마지막 입력도 1분 전
+
+	close(first.chExit)
+	waitUntil(t, func() bool { return strings.Contains(rec.output(), "초 후 자동 재연결합니다") }, time.Second)
+	szOut := rec.output()
+	if !strings.Contains(szOut, "서버가 세션을 끝냈습니다.") {
+		t.Fatalf("서버 종료 안내가 없음: %q", szOut)
+	}
+	if first.iClosed.Load() == 0 {
+		t.Fatal("남은 연결을 닫아야 함")
+	}
+	waitUntil(t, func() bool { return pDial.calls() == 2 && lastState(pSession) == G_cszStateConnected }, 4*time.Second)
+}
+
+// 끝나기 직전에 사용자 입력이 있었으면(exit·logout) 오래 연결된 세션이어도 자동 재접속하지 않는다 (session.md §4, D98).
+func TestRemoteExitRightAfterInputStaysClosed(t *testing.T) {
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	first := newExitingTransport()
+	pDial := &countingDialer{fnNext: func(iCall int) (Transport, error) {
+		if iCall == 1 {
+			return first, nil
+		}
+		return newFakeTransport(), nil
+	}}
+	info, _ := pManager.Open(sshConfig(pDial.dial, true))
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
+	pSession.backdateForTest(time.Minute)
+	pSession.Input("exit\r") // 방금 입력
+
+	close(first.chExit)
+	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
+	time.Sleep(2500 * time.Millisecond)
+	if pDial.calls() != 1 {
+		t.Fatalf("사용자가 끝낸 세션은 자동 재접속하면 안 됨, 호출 %d회", pDial.calls())
+	}
+	if strings.Contains(rec.output(), "자동 재연결") {
+		t.Fatalf("재접속 안내가 없어야 함: %q", rec.output())
+	}
+}
+
+// 자동 재접속 설정이 꺼져 있으면 유휴 뒤 서버 종료도 닫힘으로 둔다.
+func TestRemoteExitAfterIdleWithAutoReconnectOff(t *testing.T) {
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	first := newExitingTransport()
+	pDial := &countingDialer{fnNext: func(int) (Transport, error) { return first, nil }}
+	info, _ := pManager.Open(sshConfig(pDial.dial, false))
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateConnected }, time.Second)
+	pSession.backdateForTest(time.Minute)
+
+	close(first.chExit)
+	waitUntil(t, func() bool { return lastState(pSession) == G_cszStateClosed }, time.Second)
+	waitUntil(t, func() bool { return strings.Contains(rec.output(), "Enter 키를 누르면 다시 연결합니다") }, time.Second)
+}
+
 // 끊김(정상 종료가 아님)은 지금처럼 자동 재접속을 예약한다.
 func TestDropStillAutoReconnects(t *testing.T) {
 	rec := &eventRecorder{}
@@ -231,6 +307,16 @@
 	}
 }
 
+// backdateForTest 는 연결 시각과 마지막 입력 시각(있으면)을 durAgo 만큼 과거로 옮긴다.
+func (pSession *Session) backdateForTest(durAgo time.Duration) {
+	pSession.mtxData.Lock()
+	defer pSession.mtxData.Unlock()
+	pSession.tConnected = pSession.tConnected.Add(-durAgo)
+	if !pSession.tLastInput.IsZero() {
+		pSession.tLastInput = pSession.tLastInput.Add(-durAgo)
+	}
+}
+
 func (pSession *Session) attemptsForTest() int {
 	pSession.mtxData.Lock()
 	defer pSession.mtxData.Unlock()
````

##### `internal/store/store.go`

````diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -50,6 +50,7 @@
 	"insertPathQuote":    true,
 	"sessionLog":         false,
 	"sessionLogKeepDays": 30,
+	"sessionLogSyncSec":  2,
 	"bell":               false,
 	"shiftEnterNewline":  true,
 	"rememberWindow":     true,
````

##### `internal/app/app.go`

````diff
--- a/internal/app/app.go
+++ b/internal/app/app.go
@@ -150,8 +150,18 @@
 			return nil, errArg
 		}
 		iKeepBefore := pService.logKeepDays()
+		bLogBefore := pService.settingBool("sessionLog", false)
+		iSyncBefore := pService.logSyncSec()
 		errPatch := pService.pStore.Patch(mapPatch)
 		pService.applyLanguage()
+		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
+			// "터미널 출력 기록"을 바꾸면 열려 있는 세션에도 곧바로 적용한다 (session.md §6, D99).
+			pService.pSessions.SetLogging(bLogNow)
+		}
+		if iSyncNow := pService.logSyncSec(); iSyncNow != iSyncBefore {
+			// 내려 쓰기 주기도 열려 있는 세션에 곧바로 적용한다 (session.md §6, D100).
+			pService.pSessions.SetLogSyncSec(iSyncNow)
+		}
 		if pService.logKeepDays() != iKeepBefore {
 			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
 		}
@@ -159,6 +169,11 @@
 	})
 }
 
+// logSyncSec 은 세션 로그의 내려 쓰기 주기 설정이다 (초, 0~5, 0 = 받는 즉시 쓰고 내려 쓴다).
+func (pService *Service) logSyncSec() int {
+	return session.ClampLogSyncSec(pService.settingInt("sessionLogSyncSec", session.G_ciLogSyncSecDefault))
+}
+
 // logKeepDays 는 세션 로그 보관 일수 설정이다 (0 = 지우지 않음).
 func (pService *Service) logKeepDays() int {
 	return pService.settingInt("sessionLogKeepDays", G_ciSessionLogKeepDaysDefault)
````

##### `internal/app/channels_session.go`

````diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -185,6 +185,7 @@
 		DataDir:       pService.szDataDir,
 		LogEnabled:    pService.settingBool("sessionLog", false),
 		LogAlias:      record.M_szAlias,
+		LogSyncSec:    pService.logSyncSec(),
 		OpenDelay:     0,
 	}
 	if bHasSaved {
````

##### `frontend/src/ipc/types.ts`

````diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -97,6 +97,8 @@
   sessionLog: boolean;
   /** 세션 로그 보관 일수. 0 이면 지우지 않음 (behavior/session.md §6). */
   sessionLogKeepDays: number;
+  /** 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 받는 즉시 쓰고 내려 쓴다 (behavior/session.md §6). */
+  sessionLogSyncSec: number;
   bell: boolean;
   shiftEnterNewline: boolean;
   rememberWindow: boolean;
````

##### `frontend/src/lib/defaults.ts`

````diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -41,6 +41,7 @@
   insertPathQuote: true,
   sessionLog: false,
   sessionLogKeepDays: 30,
+  sessionLogSyncSec: 2,
   bell: false,
   shiftEnterNewline: true,
   rememberWindow: true,
````

##### `frontend/src/ipc/contract/store.get.json`

````diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -41,6 +41,7 @@
     "scrollback": 10000,
     "sessionLog": false,
     "sessionLogKeepDays": 30,
+    "sessionLogSyncSec": 2,
     "shiftEnterNewline": true,
     "sidebarWidth": 260,
     "syncAuto": true,
````

##### `frontend/src/ipc/contract/store.get.afterNullPatch.json`

````diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -41,6 +41,7 @@
     "scrollback": 10000,
     "sessionLog": false,
     "sessionLogKeepDays": 30,
+    "sessionLogSyncSec": 2,
     "shiftEnterNewline": true,
     "sidebarWidth": 260,
     "syncAuto": true,
````

##### `frontend/src/screens/SettingsConnection.tsx`

````diff
--- a/frontend/src/screens/SettingsConnection.tsx
+++ b/frontend/src/screens/SettingsConnection.tsx
@@ -12,6 +12,7 @@
   const [restoreLayout, setRestoreLayout] = useSetting('restoreLayout');
   const [sessionLog, setSessionLog] = useSetting('sessionLog');
   const [keepDays] = useSetting('sessionLogKeepDays');
+  const [syncSec] = useSetting('sessionLogSyncSec');
   const pushToast = useApp((s) => s.pushToast);
 
   const openLogs = async () => {
@@ -54,6 +55,16 @@
           onCommit={(v) => void saveSettings({ sessionLogKeepDays: v })}
         />
       </Field>
+      <Field label={t('conn.logSyncSec')} help={t('conn.logSyncSecHelp')}>
+        <NumberField
+          label={t('conn.logSyncSec')}
+          value={syncSec}
+          min={0}
+          max={5}
+          fallback={2}
+          onCommit={(v) => void saveSettings({ sessionLogSyncSec: v })}
+        />
+      </Field>
       <div>
         <button className="btn" type="button" onClick={() => void openLogs()}>
           <FolderOpen size={14} /> {t('conn.openLogs')}
````

##### `spec/behavior/session.md`

````diff
--- a/spec/behavior/session.md
+++ b/spec/behavior/session.md
@@ -44,7 +44,7 @@
 - 로컬 셸 종료: `● {이유 또는 "셸이 종료되었습니다"} — Enter 키를 누르면 새 셸을 엽니다.`(회색)
 - 자동 재접속 예약: `● {n}초 후 자동 재연결합니다... (Enter: 즉시 재연결)`(회색)
 - 재접속 포기: `● Enter 키를 누르면 다시 연결합니다.`(회색)
-- 서버가 세션을 정상 종료(§4): `● 서버가 세션을 끝냈습니다.`(회색) 다음 줄에 `● Enter 키를 누르면 다시 연결합니다.`(회색). (2026-11-17)
+- 서버가 세션을 정상 종료(§4): `● 서버가 세션을 끝냈습니다.`(회색) 다음 줄에, 사용자가 끝낸 경우 `● Enter 키를 누르면 다시 연결합니다.`(회색), 서버 쪽에서 끝낸 경우 자동 재접속 예약 줄. (2026-11-17, 2026-11-18 개정)
 - 앱 잠금 해제 대기(§4): `● 앱 잠금을 해제하면 다시 연결합니다.`(회색). 대기에 들어올 때 한 번만. (2026-11-17)
 - 점프/포워딩/키/비밀번호 저장 안내는 protocols.md.
 
@@ -67,7 +67,10 @@
 - 대기 시간: `min(30, 2 × 2^(시도 횟수))` 초 → 2, 4, 8, 16, 30초(최대 5회, D62). 예약할 때 상태를 `reconnecting`, 메시지 `{n}초 후 재접속 ({시도}/8)` 로 하고 §3 안내 줄을 출력.
 - 규칙을 충족하지 않으면(자동 재접속 꺼짐·시리얼·5회 초과 등) 시도 횟수를 0 으로 되돌리고, **접속 실패였으면 상태 `error`, 끊김이었으면 `closed`** 로 하고 "Enter 키를 누르면 다시 연결합니다." 를 출력.
 - **수동 재접속**: 사용자가 칸의 [재연결] 버튼·메뉴를 쓰거나, **끊긴(연결 없는) 세션에서 Enter(CR) 입력**을 하면(접속 중이 아닐 때) 즉시 재접속한다. 대기 중인 예약 타이머는 취소. 이미 연결되어 있으면 아무 일 없음. **로컬 셸이 종료된 뒤 Enter 를 누르면 새 셸을 연다**(같은 규칙).
-- **서버가 정상 종료한 세션은 자동 재접속하지 않는다**(2026-11-17): SSH 셸이 **종료 코드(0 포함) 또는 신호와 함께** 끝났으면(서버의 유휴 자동 로그아웃, 사용자의 `exit`·`logout`, 관리자의 세션 종료) "끊김"이 아니라 정상 종료로 본다. 상태 `closed`, 시도 횟수 0, §3 의 정상 종료 안내 두 줄을 출력하고 남은 SSH 연결(포워딩·keepalive 포함)을 닫는다. Enter·[재연결] 로 다시 연결하는 것은 그대로. 종료 코드 없이 채널이 닫혔거나 전송 오류로 끝나면(네트워크 단절, keepalive 실패, 절전 복귀) 지금처럼 끊김이다. 세션 유지(tmux)를 켠 서버도 같은 규칙. 텔넷은 둘을 구분할 수 없어 끊김으로 둔다. (서버의 유휴 로그아웃을 피하려는 가짜 입력은 보내지 않는다 — 서버 보안 정책을 우회하는 동작.)
+- **사용자가 끝낸 세션은 자동 재접속하지 않고, 서버 쪽에서 끝낸 세션은 자동 재접속한다**(2026-11-17, 2026-11-18 개정): SSH 셸이 **종료 코드(0 포함) 또는 신호와 함께** 끝났으면 "끊김"이 아니라 정상 종료로 보고, 남은 SSH 연결(포워딩·keepalive 포함)을 닫은 뒤 §3 의 `서버가 세션을 끝냈습니다.` 를 출력한다. 그다음은 누가 끝냈는지에 따라 다르다. 사용자의 `exit`·`logout` 과 서버의 유휴 자동 로그아웃은 종료 코드로 구분되지 않으므로 **직전 입력 여부**로 가린다.
+  - **사용자가 끝낸 것**: 끝나기 전 **10초 안에 이 세션으로 보낸 사용자 입력이 있었거나**, **연결된 지 10초가 안 됐을 때**(로그인 직후 셸이 끝나는 서버를 계속 다시 접속하지 않기 위함). 상태 `closed`, 시도 횟수 0, `Enter 키를 누르면 다시 연결합니다.` 를 출력한다. 자동 재접속하지 않는다.
+  - **서버 쪽에서 끝낸 것**: 그 밖의 경우(입력 없이 있다가 끝남 — 유휴 자동 로그아웃, 관리자의 세션 종료, 서버 재시작). **끊김과 같은 자동 재접속 규칙**(위 조건·대기 시간)을 따른다. 자동 재접속 조건을 충족하지 않으면(설정 꺼짐 등) 사용자가 끝낸 경우와 같이 `closed` 로 둔다.
+  - Enter·[재연결] 로 다시 연결하는 것은 두 경우 모두 그대로. 접속 직후 자동으로 보내는 입력(세션 유지 명령·자동 실행 명령)은 사용자 입력으로 세지 않는다. 종료 코드 없이 채널이 닫혔거나 전송 오류로 끝나면(네트워크 단절, keepalive 실패, 절전 복귀) 지금처럼 끊김이다. 세션 유지(tmux)를 켠 서버도 같은 규칙. 텔넷은 정상 종료를 구분할 수 없어 끊김으로 둔다. (서버의 유휴 로그아웃을 피하려는 가짜 입력은 보내지 않는다 — 서버 보안 정책을 우회하는 동작.)
 - **앱 잠금 해제 대기**(2026-11-17, protocols.md §2.2): SSH 접속(처음 열기·자동·수동 재접속 모두)에 볼트가 필요한데 볼트가 잠겨 있으면 — 저장된 계정을 읽을 수 없고 세션 사본도 없거나, 사본으로 안 되어 질문 창이 필요하면 — 접속을 멈추고 상태 `reconnecting`, 메시지 `앱 잠금을 해제하면 다시 연결합니다.`, §3 안내 줄을 출력한다. **실패로 세지 않는다**(시도 횟수 불변, 오류 상태로 가지 않음, 예약 타이머 없음). 볼트가 열리면 기다리던 세션을 **곧바로** 다시 접속한다(닫힌 세션·이미 접속 중인 세션은 제외). 기다리는 중 Enter·[재연결]은 다시 시도하며, 아직 잠겨 있으면 같은 대기로 돌아간다(안내 줄은 다시 내지 않는다). 볼트를 만들지 않은 사용자는 이 규칙이 없다(지금처럼 묻는다).
 - **접속이 겹치지 않음**: 한 세션의 접속 시도는 한 번에 하나다. 접속 중에 온 Enter·[재연결]·잠금 해제 재개는 무시한다.
 - **오래된 연결의 콜백 무시**: 재접속할 때마다 연결 "세대"를 올려, 이전 연결에서 늦게 오는 출력·종료 알림은 무시한다.
@@ -80,10 +83,15 @@
 3. **세션 유지(tmux)가 적용되면**(§7) 500ms 뒤 tmux 접속 명령을 셸에 입력한다. 아니면 서버의 **자동 실행 명령**이 있으면 줄바꿈을 CR 로 바꾸고 끝에 CR 을 붙여 400ms 뒤에 입력한다.
 
 ## 6. 세션 로그
-- 설정 "터미널 출력 기록"(`sessionLog`)이 켜져 있을 때 **세션을 열 때 한 번** 로그 폴더(`<데이터 폴더>\logs`)에 `{표시 이름}-{YYYYMMDD-HHmmss}.log` 를 **새로 만든다**(권한 소유자 전용). 같은 이름이 이미 있으면 `.log` 앞에 `-2`, `-3`, … 을 붙여 없는 이름으로 만든다(세션 하나에 파일 하나). 1000번까지 시도해도 못 만들면 로그 없이 연다.
+- 설정 "터미널 출력 기록"(`sessionLog`)이 켜져 있을 때 **세션을 열 때 한 번** 로그 폴더(`<데이터 폴더>\logs`)에 `{표시 이름}-{YYYYMMDD-HHmmss}.log` 를 **새로 만든다**(권한 소유자 전용, 열린 뒤에 설정을 켠 경우는 아래). 같은 이름이 이미 있으면 `.log` 앞에 `-2`, `-3`, … 을 붙여 없는 이름으로 만든다(세션 하나에 파일 하나). 1000번까지 시도해도 못 만들면 로그 없이 연다.
 - 파일 이름의 표시 이름: 별칭의 `\ / : * ? " < > |` 와 제어 문자는 `_` 로 바꾸고, 앞뒤 공백·점을 지우며, 80자로 자른다. 비면 `session`.
 - 기록 내용: 화면 출력에서 **ANSI 이스케이프 시퀀스를 제거**하고 CR 도 제거한 텍스트. 시퀀스가 출력 묶음 경계에서 잘려도 이어 붙여 지운다(종결 없이 256바이트를 넘는 ESC 는 시퀀스로 보지 않고 그대로 둔다). 디스크에는 **약 1초 단위로 모아서** 쓰고, 세션을 닫을 때 남은 것을 쓴다(앱이 비정상 종료하면 마지막 1초 분량이 빠질 수 있다).
-- 자동·수동 재접속은 같은 파일에 이어 쓰고, 세션을 닫으면 파일을 닫는다. 설정을 바꾸면 **새 세션부터** 적용.
+- 자동·수동 재접속은 같은 파일에 이어 쓰고, 세션을 닫으면 파일을 닫는다.
+- **설정을 바꾸면 열려 있는 세션에도 곧바로 적용**(2026-11-18): 켜면 열려 있는 세션마다 그 순간의 시각으로 새 로그 파일을 만들어 **그때부터의 출력**을 기록한다(이전 출력은 넣지 않는다, 이미 기록 중인 세션은 그대로). 끄면 열려 있는 세션의 기록을 멈추고 남은 것을 쓴 뒤 파일을 닫는다. 다시 켜면 새 파일이다.
+- **내려 쓰기 주기**(2026-11-18): 설정 `sessionLogSyncSec`(초, 0~5, 기본 2, 범위 밖 값은 가까운 끝으로). 로그 파일은 세션 동안 열어 두는데, Windows 는 열려 있는 파일의 크기·수정 시각을 **디스크까지 내려 쓰거나 닫을 때만** 폴더 목록에 반영한다. 그래서 기록 중에도 파일 크기가 보이도록 주기적으로 내려 쓴다.
+  - 값이 1~5 이면: 파일 쓰기는 지금처럼 약 1초 단위, 내려 쓰기는 **그 초마다 한 번**(그동안 새로 쓴 내용이 있을 때만). 출력이 멈춘 뒤에도 남은 내용은 주기가 되면 내려 쓴다.
+  - 값이 0 이면: 출력을 받는 **즉시** 파일에 쓰고 내려 쓴다(모으지 않는다). 출력이 많은 세션에서는 디스크 부하와 출력 지연이 커질 수 있다.
+  - 설정을 바꾸면 기록 중인 세션에도 곧바로 적용한다.
 - **보관 기간**: 설정 `sessionLogKeepDays`(기본 30, 0 이면 지우지 않음). 앱은 시작 약 10초 뒤와 그 뒤 **1시간마다** 로그 폴더 바로 아래의 `.log` 파일 중 수정 시각이 그 일수보다 오래된 것을 지운다. 지금 열려 있는(기록 중인) 세션의 파일과 하위 폴더·다른 확장자 파일은 건드리지 않는다. 설정 값을 바꾸면 곧바로 한 번 정리한다. 지우지 못한 파일(다른 프로그램이 열고 있음 등)은 건너뛰고 다음 주기에 다시 본다. "터미널 출력 기록"이 꺼져 있어도 남은 파일은 정리한다.
 
 ## 7. 세션 유지 (tmux)
````

##### `spec/data/store.md`

````diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -95,6 +95,7 @@
 | insertPathQuote | true | 경로 삽입 시 따옴표 |
 | sessionLog | false | |
 | sessionLogKeepDays | 30 | 세션 로그 보관 일수. 0 이면 지우지 않음 ([../behavior/session.md](../behavior/session.md) §6) |
+| sessionLogSyncSec | 2 | 기록 중인 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 출력을 받는 즉시 쓰고 내려 쓴다 ([../behavior/session.md](../behavior/session.md) §6, 2026-11-18) |
 | bell | false | |
 | shiftEnterNewline | true | Shift+Enter 로 줄바꿈 입력 |
 | rememberWindow | true | 창 크기·위치 기억 |
````

##### `spec/ui/settings.md`

````diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -50,8 +50,9 @@
 - 카드 토글:
   1. **끊기면 자동 재접속** — "연결이 끊기면 2·4·8·16·30초로 간격을 늘리며 최대 5번 재연결합니다"
   2. **탭·분할 레이아웃 복원** — "종료할 때 열린 탭과 창 배치를 저장하고, 다음 실행 때 같은 서버로 다시 연결합니다. 비밀번호·화면 내용은 저장하지 않으며, 목록에 없는 일회성 SSH 연결은 복원하지 않습니다"
-  3. **세션 로그 저장** — "터미널 출력을 텍스트 파일로 남깁니다 (켠 뒤 새로 여는 연결부터)"
+  3. **세션 로그 저장** — "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)"
 - 필드 **로그 보관 기간 (일)**(숫자, 0~36500, 비면 30, 도움말 "이 일수보다 오래된 세션 로그 파일을 자동으로 지웁니다. 0이면 지우지 않습니다.") — 값은 `sessionLogKeepDays`(behavior/session.md §6 보관 기간).
+- 필드 **로그 파일 갱신 주기 (초)**(숫자, 0~5, 비면 2, 도움말 "기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다.") — 값은 `sessionLogSyncSec`(behavior/session.md §6 내려 쓰기 주기). (2026-11-18)
 - 버튼 [로그 폴더 열기](폴더 아이콘) → 로그 폴더를 탐색기로 연다.
 
 ## 5. 동기화 (git)
````

---

## 부록 Q. 문서에 넣을 글

### Q.1 `docs/decisions.md` — D96~D100 행 (D101 행 바로 위에)

````markdown
| D96 | 2026-11-18 | **터미널 찾기 강조의 색·모양**: 찾은 항목은 터미널 배경에 주황(`#f59e0b`)을 32% 섞은 옅은 바탕(테두리 없음), 현재 항목은 진한 빨강 바탕(`#dc2626`) + 흰 글자 + 테마 글자색 1px 테두리. 현재 항목은 검색 애드온의 색 대신 선택 영역 **위(top)** 에 직접 얹는 장식으로 그린다(`frontend/src/lib/findHighlight.ts`, `TerminalView.tsx` FindBar). 스크롤 막대 색은 그대로(주황/빨강) | 사용자 보고("현재 선택된 것이 눈에 띄어야 하는데, 찾아진 것이 더 눈에 띈다"). 원인: 애드온이 현재 항목을 선택으로 표시하고 그 색은 선택 색 아래에 깔려 회청색으로만 보였다. 색 값에 투명도(`#RRGGBBAA`)를 쓰던 것도 애드온 규격(`#RRGGBB`)에 맞췄다. 명세 ui/panes.md §2.4(주황 계열/빨강 계열) 범위 안. 찾기 동안 선택 색 자체를 바꾸는 안은 마우스 선택까지 빨갛게 돼 기각 |
| D97 | 2026-11-18 | **잠금 화면의 Caps Lock 안내**: 비밀번호를 입력하는 세 화면(잠금 해제·처음 설정·복구 코드로 재설정)에서 Caps Lock 이 켜져 있으면 입력창 아래에 경고색 한 줄("Caps Lock이 켜져 있습니다", 문구 키 `lock.capsLock`, 10개 언어)을 보인다. 상태는 창의 키 누름·뗌과 마우스 움직임·누름 이벤트에서 읽고(`frontend/src/lib/useCapsLock.ts`), 창이 포커스를 잃으면 지운다. 입력을 막거나 글자를 바꾸지는 않는다 | 사용자 요청("잠김 화면에서 비번 입력할 때, caps lock 켜 있는지 확인하게 해줘"). 웹뷰는 Caps Lock 상태를 직접 물을 수 없어 이벤트가 오기 전에는 꺼짐으로 본다. 백엔드(Win32 GetKeyState)로 묻는 안은 OS 별 코드가 늘어 기각. 명세 ui/lock.md 에는 없는 추가 요소 |
| D98 | 2026-11-18 | **서버가 셸을 끝냈을 때 누가 끝냈는지 가려 자동 재접속**(D93 개정): 종료 코드·신호와 함께 끝난 SSH 셸은, 끝나기 전 10초 안에 사용자 입력이 있었거나 연결된 지 10초가 안 됐으면 사용자가 끝낸 것(`exit`·`logout`)으로 보고 지금처럼 `closed` + Enter 안내. 그 밖(입력 없이 있다가 끝남 = 유휴 자동 로그아웃·관리자 종료·서버 재시작)은 끊김과 같은 규칙으로 자동 재접속한다(`internal/session/session.go` `userEndedLocked`·`onRemoteExited`, `control.go` `Input`). 명세 session.md §3·§4 개정 | 사용자 요청("timeout 나서 끊어진 경우 재연결이 안 된다. exit 와 구분이 안 되면 exit 인 경우에도 재연결"). 셸의 유휴 로그아웃(bash TMOUT 등)과 `exit` 는 둘 다 종료 코드와 함께 끝나 SSH 로는 구분되지 않는다. 유휴 로그아웃은 입력이 한참 없을 때만 일어나므로 직전 입력 여부로 가린다. 잘못 가려도 `exit` 뒤 재접속되는 쪽(사용자가 허용한 동작)으로만 틀린다. 종료 직전 화면 문구("auto-logout")로 가리는 안은 셸·언어마다 달라 기각, 전부 재접속하는 안은 `exit` 할 때마다 다시 붙어 기각. 연결 직후 종료를 제외한 것은 로그인하자마자 끝나는 서버에 2초마다 다시 붙는 것을 막기 위함. 유휴 로그아웃 서버에서는 로그아웃될 때마다 다시 로그인하게 된다(설정 "연결이 끊기면 다시 연결"을 끄면 멈춤) |
| D99 | 2026-11-18 | **"터미널 출력 기록" 설정을 바꾸면 열려 있는 세션에도 곧바로 적용**: 켜면 열려 있는 세션마다 그 시각의 새 로그 파일을 만들어 그때부터의 출력을 기록하고(이전 화면 내용은 넣지 않음), 끄면 열려 있는 세션의 기록을 멈추고 파일을 닫는다. 다시 켜면 새 파일. `session.Session.SetLogging`·`Manager.SetLogging`, `store.patch` 에서 값이 바뀌었을 때 호출. 설정 설명 문구(`conn.sessionLogDesc`, 10개 언어)와 명세 session.md §6·ui/settings.md, 시나리오 6-11 을 고쳤다 | 사용자 보고("켰는데 logs 폴더에 저장되지 않는다" → 켜기 전에 열린 세션이라 기록되지 않았음)와 제안("off 에서 on 을 하면 현재 창에 대해서 로그 파일을 새로 만들어야 하지 않을까"). 끄기도 같이 바꾼 것은 권장안: 설정이 꺼져 있는데 기록이 계속되면 사용자가 모르는 채 출력이 파일에 남는다. 켜기 전 화면 내용까지 넣는 안은 백엔드가 지난 출력을 갖고 있지 않아 기각. 동기화·백업 가져오기로 설정이 바뀐 경우는 다루지 않는다(새 세션부터 적용) |
| D100 | 2026-11-18 | **세션 로그 내려 쓰기 주기** 설정 `sessionLogSyncSec`(초, 0~5, 기본 2, 설정 > 연결 "로그 파일 갱신 주기 (초)"). 1~5: 파일 쓰기는 지금처럼 약 1초 단위, 디스크까지 내려 쓰기(fsync)는 그 초마다 한 번(새로 쓴 내용이 있을 때만, 잠금 밖에서). 0: 출력을 받는 즉시 파일에 쓰고 내려 쓴다(모으지 않음). 값을 바꾸면 기록 중인 세션에도 곧바로 적용. `session.Logger.SetSyncSec`·`flushTimer`, `Session/Manager.SetLogSyncSec`, `store.patch` | 사용자 보고("백그라운드로 갔다 온 뒤 로그가 저장이 안 된다") — 실제로는 저장되고 있었고, Windows 가 열려 있는 파일의 크기·수정 시각을 내려 쓰거나 닫을 때만 폴더 목록에 반영해 탐색기에 0바이트로 보인 것(이 PC 에서 실험으로 확인). 주기·범위·기본값·0 의 뜻은 사용자 지정. 닫았다 다시 여는 안은 그 틈에 다른 프로그램이 파일을 잡을 수 있어 기각. 디스크 부하는 재 보지 않았다 |
````

### Q.2 `.claude/known-issues.md` — KI-011 (ACTIVE 아래에)

````markdown
### KI-011: 찾기 막대가 열린 동안 찾은 글자가 클립보드에 계속 복사되고, 출력 중에는 강조가 늦게 갱신됨
- **Status**: active
- **Severity**: medium
- **Module**: frontend (screens/TerminalView.tsx — FindBar, 선택 시 자동 복사)
- **Trigger areas**: 터미널 찾기, 선택 시 자동 복사(copyOnSelect), 출력이 이어지는 세션
- **증상**(코드로 확인, 앱에서 재현해 보지는 않음):
  1. "선택 시 자동 복사"가 켜져 있으면(기본값) 찾기에서 글자를 칠 때·다음/이전을 누를 때마다 현재 항목이 클립보드에 복사된다. 출력이 이어지는 세션에서는 출력이 0.2초 멈출 때마다 다시 복사돼, 다른 곳에서 복사해 둔 내용이 덮인다.
  2. 출력이 0.2초 넘게 멈추지 않으면 새로 들어온 줄의 일치 항목은 강조되지 않는다(멈추면 한꺼번에 갱신).
  3. 찾기 막대가 열린 동안 마우스로 다른 곳을 선택해도, 출력이 멈추면 선택이 일치 항목으로 바뀐다.
  4. 강조는 버퍼 맨 위(가장 오래된 줄)부터 1000개까지만 한다. 흔한 낱말이면 최근 줄은 강조되지 않는다(다음/이전 이동은 된다).
- **원인**: 검색 애드온(@xterm/addon-search 0.16)이 현재 항목을 터미널 "선택"으로 표시하고, 출력이 올 때마다 0.2초 뒤(출력이 이어지면 계속 미룸) 전체를 다시 찾아 다시 선택한다. 앱의 자동 복사는 선택이 바뀌면 누가 선택했는지 가리지 않고 복사한다.
- **History**:
  - 2026-11-18: 사용자 질문("로그가 계속 출력중에 찾기를 하면 어떻게 되나?")으로 코드 확인 중 발견
````

### Q.3 시나리오 6-11 행

`docs/scenarios.md`:

````markdown
| 6-11 | 출력 기록 | 설정 > 연결 "터미널 출력 기록" 켬(기본 꺼짐) | 켜면 열려 있는 연결도 그때부터, 새 연결은 처음부터 텍스트 파일로 저장. 끄면 기록을 멈춤. [로그 폴더 열기]. |
````

`docs/scenarios.en.md`:

````markdown
| 6-11 | Log output | Settings > Connection "Log terminal output" on (off by default) | Open connections start saving output to text files from that moment, new connections from the start. Turning it off stops recording. [Open log folder]. |
````
