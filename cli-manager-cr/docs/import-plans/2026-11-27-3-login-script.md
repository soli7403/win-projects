# 로그인 스크립트 — 접속 직후 출력을 기다렸다가 입력을 보낸다

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D139). 실제 서버에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 접속 후 자동 실행 명령·tmux 접속 줄([spec/behavior/session.md](../../spec/behavior/session.md) §5·§7). 명세 session.md §5.1, [spec/ui/host-editor.md](../../spec/ui/host-editor.md) §1.4a, [spec/data/store.md](../../spec/data/store.md) §3
**적용 순서**: 2026-11-27 의 3번째. 1번과 `internal/session/session.go`·`internal/app/ssh_app.go` 가 겹친다.

---

## 0. 한 줄 요약

서버마다 **"이 글자가 나오면 → 이것을 입력"** 단계를 적어 두면, 연결할 때마다 순서대로 실행한다. `sudo su -` 뒤의 비밀번호, 장비의 `enable`, 접속 메뉴 선택 같은 일을 자동으로 넘긴다.

| # | 기다릴 글자 | 보낼 글자 |
|---|---|---|
| 1 | `$ ` | `sudo su -` |
| 2 | `password` | `{{password}}` ← 이 서버에 로그인할 때 쓴 비밀번호로 바뀐다 |
| 3 | `# ` | `cd /var/log` |

"연결 후 실행할 명령"(고급 탭)과의 차이: 그 명령은 0.4초 뒤에 **기다리지 않고** 한 번 보낸다. 로그인 스크립트는 **출력을 보고** 보낸다. 둘 다 있으면 스크립트가 끝난 뒤에 그 명령이 실행된다.

## 1. 배경

2026-11-27 사용자 요청("권장 순서대로 구현")의 둘째 항목. Xshell·SecureCRT 에서 옮겨 오는 사람이 가장 먼저 찾는 기능이다. 원본 앱에 없던 기능이라 명세도 이번에 새로 썼다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 실행하는 곳 | **백엔드**(세션의 읽기 루프) | 접속 직후의 첫 출력을 놓치지 않고, 탭이 안 보이거나 앱이 잠겨 재접속할 때도 돈다. 비밀번호를 화면 쪽으로 보내지 않아도 된다 |
| 조건 | 글자 포함 여부(정규식 아님), 대소문자 구분 없음 | `Password:`·`password:` 를 가리지 않게. 정규식은 필요해지면 더한다 |
| 한 번 맞은 글자 | 다음 단계에 다시 쓰지 않는다 | 같은 `$ ` 를 두 번 기다리는 스크립트가 프롬프트 하나에 두 단계를 넘기지 않게 |
| 비밀번호 | `{{password}}` → 로그인에 쓴 비밀번호 | 보낼 글자는 평문으로 저장되므로 비밀번호를 적지 않게 할 길이 필요하다. 세션이 이미 들고 있는 사본(D94 의 세션 계정 사본과 같은 수명)을 쓴다 |
| 비밀번호가 없을 때 | 멈춘다(빈 비밀번호를 보내지 않는다) | 빈 비밀번호가 여러 번 가면 계정이 잠길 수 있다 |
| 시한 | 단계마다 20초, 넘으면 멈추고 알림 | 프롬프트가 다르면 영영 기다리지 않게 |
| 사용자 입력 | 입력이 오면 멈춘다(ESC 로 시작하는 입력 제외) | 손으로 넘기는 동안 스크립트가 뒤늦게 비밀번호를 엉뚱한 곳에 치는 것을 막는다. 터미널의 자동 응답(커서 위치 등)은 ESC 로 시작한다 |
| 접속 후 명령·tmux | 스크립트가 **끝까지** 간 뒤에 보낸다. 멈추면 보내지 않는다 | 스크립트가 권한을 바꾼 뒤에 명령이 실행돼야 한다. 멈췄으면 상태를 알 수 없다 |
| 재접속 | 처음부터 다시 | 새 셸이다 |
| 화면 위치 | 서버 편집의 새 탭 "자동화" | 고급 탭이 이미 길다. 로컬 터미널에는 탭을 숨긴다 |
| 단계 수 | 20 | 넉넉하고, 잘못 만든 긴 스크립트를 막는다 |

알려 둘 것:
- **보낼 글자는 평문**으로 store.json 에 남고 git 동기화·백업에도 들어간다. 탭에 경고를 적었다. 비밀번호를 금고에 넣어 쓰는 것은 `{{password}}`(로그인 비밀번호) 하나만 지원한다 — 다른 비밀번호(예: 로그인과 다른 `su` 비밀번호)는 아직 금고에 둘 길이 없다.
- 텔넷은 저장된 계정이 없어 `{{password}}` 가 채워지지 않는다(탭에 그 안내를 보이지 않는다).
- PuTTY·SecureCRT·Xshell 가져오기는 로그인 스크립트를 아직 옮기지 않는다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 내용 |
|---|---|---|
| `internal/session/loginscript.go` | 코드 | **새 파일** — `LoginStep`, `loginRunner`(진행 상태), 세션 연결(`startLogin`·`feedLogin`·시한·취소) |
| `internal/session/session.go` | 코드 | 접속 성공 때 `startLogin`(읽기 루프보다 먼저), 출력마다 `feedLogin` |
| `internal/session/control.go` | 코드 | 입력이 오면 스크립트 취소, 닫을 때 정리 |
| `internal/session/types.go` | 코드 | `OpenConfig.LoginScript` |
| `internal/app/channels_session.go` | 코드 | 서버 기록의 `loginScript` 읽기(`loginScriptOf`) |
| `internal/app/ssh_app.go` | 코드 | `sshSession.LoginSecret`(로그인 비밀번호) |
| `frontend/src/ipc/types.ts` | 코드 | `LoginStep`, `Host.loginScript` |
| `frontend/src/lib/loginScript.ts` | 코드 | **새 파일** — 빈 단계 빼기, 단계 옮기기 |
| `frontend/src/screens/LoginScriptTable.tsx` | 코드 | **새 파일** — 단계 표 |
| `frontend/src/screens/HostEditor.tsx`, `frontend/src/lib/hostSave.ts` | 코드 | "자동화" 탭, 저장 |
| `internal/session/loginscript_test.go` | 테스트 | **새 파일**(9개) |
| `frontend/src/lib/macro.test.ts`, `frontend/src/screens/Automation.test.tsx` | 테스트 | 단계 다듬기 2개, 자동화 탭 3개 |
| `locales/*.json` | 문구 | `host.tabAuto`, `login.*` 12개, `be.login.*` 3개 |
| `spec/behavior/session.md`, `spec/data/store.md`, `spec/ui/host-editor.md`, `spec/acceptance/04-hosts-sftp-settings-sync.md`(H12~H14) | 명세 | 수정 |

아래 `diff` 는 **오늘(2026-11-27) 작업을 시작하기 전의 파일**과 지금 파일을 견준 것이다. 이 폴더는 git 저장소가 아니라서, 오늘 적용한 수정 기록을 거꾸로 돌려 고치기 전 파일을 되살린 뒤 뽑았다(되살린 것에 수정을 다시 적용하면 지금 파일과 같아지는 것을 확인했다). 한 파일을 오늘의 여러 문서가 고친 경우, 각 문서에는 그 문서의 몫이 든 조각만 싣는다 — 조각 안의 문맥 줄에 다른 문서의 코드가 섞여 있을 수 있고, 줄 번호는 다섯 문서를 모두 적용한 최종 파일 기준이다. 새 파일은 최종 내용 전체를 싣는다(같은 날의 추가 지시 D142·D143 까지 반영된 것).

## B. 코드

### 먼저 읽을 곳

[internal/session/loginscript.go](../../internal/session/loginscript.go):

```go
// feed: 출력 조각을 받아 이번에 조건이 맞은 단계들의 "보낼 글자"를 돌려준다.
szBody, szCarry := splitAnsiTail(pRunner.szCarry + szText)          // 조각 끝의 반쪽 시퀀스는 다음 조각으로
pRunner.szTail += strings.ToLower(g_reAnsi.ReplaceAllString(szBody, ""))
for pRunner.iStep < len(pRunner.slSteps) {
	// expect 가 비면 바로, 아니면 szTail 에서 찾아 그 뒤로 잘라 낸다(다음 단계에 다시 쓰지 않게)
}
```

- `startLogin` 은 `go readLoop` **앞**에서 부른다 — 접속 직후 이미 와 있는 첫 프롬프트를 읽기 루프가 넘겨주기 전에 스크립트가 걸려 있어야 한다.
- `feedLogin` 이 `{{password}}` 를 바꿀 때 연결이 `SecretProvider` 가 아니거나 값이 비면 멈춘다.
- 시한은 단계가 넘어갈 때마다 새로 건다(`armLoginTimeout`). 콜백은 그사이 단계가 넘어갔는지 확인한다.
- 스크립트가 끝나면 `scheduleStartup` 으로 접속 후 입력을 건다(스크립트가 없으면 전처럼 접속 직후에 건다).

### `internal/session/loginscript.go` — 새 파일 (전체)

```go
package session

import (
	"dariterm/internal/i18n"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// 로그인 스크립트 (session.md §5.1, D139): 접속 직후 서버 출력에서 "기다릴 글자"가 나오면 "보낼 글자"를 입력한다.
// 단계는 위에서 아래로 한 번씩만 실행한다. 접속(재접속 포함)마다 처음부터 다시 한다.

// LoginStep 은 로그인 스크립트의 한 단계다 (data/store.md §3 loginScript).
type LoginStep struct {
	Expect string // 기다릴 글자. 비면 기다리지 않고 바로 보낸다
	Send   string // 보낼 글자. 뒤에 Enter(CR)를 붙여 보낸다
}

// G_ciLoginStepMax 는 로그인 스크립트의 최대 단계 수다. 넘는 단계는 버린다.
const G_ciLoginStepMax = 20

// G_durLoginStepTimeout 은 한 단계가 기다리는 최대 시간이다. 넘으면 스크립트를 멈춘다.
// 테스트에서 줄일 수 있게 변수로 둔다.
var G_durLoginStepTimeout = 20 * time.Second

// G_cszLoginPassword 는 보낼 글자에 쓰는 자리표시자다: 이 서버에 로그인할 때 쓴 비밀번호로 바뀐다.
const G_cszLoginPassword = "{{password}}"

// g_ciLoginTailMax 는 기다릴 글자를 찾는 최근 출력의 최대 길이(바이트)다.
const g_ciLoginTailMax = 4096

// SecretProvider 는 로그인에 쓴 비밀번호를 알려 줄 수 있는 Transport 다 (SSH).
// 로그인 스크립트의 {{password}} 를 채우는 데만 쓴다. 없으면 빈 글자를 돌려준다.
type SecretProvider interface {
	LoginSecret() string
}

// TrimLoginSteps 는 저장된 단계에서 쓸 수 있는 것만 남긴다: 기다릴 글자와 보낼 글자가 모두 빈 단계는 빼고,
// G_ciLoginStepMax 개를 넘는 단계는 버린다.
func TrimLoginSteps(slSteps []LoginStep) []LoginStep {
	slOut := make([]LoginStep, 0, len(slSteps))
	for _, step := range slSteps {
		if step.Expect == "" && step.Send == "" {
			continue
		}
		if len(slOut) >= G_ciLoginStepMax {
			break
		}
		slOut = append(slOut, step)
	}
	return slOut
}

// loginRunner 는 접속 하나에서 로그인 스크립트의 진행 상태다. 접속마다 새로 만든다.
type loginRunner struct {
	mtxData sync.Mutex
	slSteps []LoginStep
	iStep   int    // 다음에 실행할 단계
	szCarry string // 조각 끝의 미완성 ANSI 시퀀스 (다음 조각과 이어 본다)
	szTail  string // ANSI 를 지운 최근 출력 (소문자). 맞은 자리까지는 잘라 낸다
	bDone   bool   // 끝났거나 멈췄다
}

func newLoginRunner(slSteps []LoginStep) *loginRunner {
	return &loginRunner{slSteps: slSteps}
}

// feed 는 출력 조각을 받아, 이번에 조건이 맞은 단계들의 "보낼 글자"를 순서대로 돌려준다.
// bFinished 는 이 호출로 마지막 단계까지 끝났다는 뜻이다. 대소문자는 구분하지 않는다.
// 한 번 맞은 글자는 다음 단계의 조건에 다시 쓰지 않는다(같은 프롬프트가 두 단계를 한꺼번에 넘기지 않게).
func (pRunner *loginRunner) feed(szText string) (slSends []string, bFinished bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	if pRunner.bDone {
		return nil, false
	}
	szBody, szCarry := splitAnsiTail(pRunner.szCarry + szText)
	pRunner.szCarry = szCarry
	pRunner.szTail += strings.ToLower(g_reAnsi.ReplaceAllString(szBody, ""))
	if iOver := len(pRunner.szTail) - g_ciLoginTailMax; iOver > 0 {
		for iOver < len(pRunner.szTail) && !utf8.RuneStart(pRunner.szTail[iOver]) {
			iOver++
		}
		pRunner.szTail = pRunner.szTail[iOver:]
	}
	for pRunner.iStep < len(pRunner.slSteps) {
		step := pRunner.slSteps[pRunner.iStep]
		if step.Expect != "" {
			szNeedle := strings.ToLower(step.Expect)
			iAt := strings.Index(pRunner.szTail, szNeedle)
			if iAt < 0 {
				break
			}
			pRunner.szTail = pRunner.szTail[iAt+len(szNeedle):]
		}
		slSends = append(slSends, step.Send)
		pRunner.iStep++
	}
	if pRunner.iStep >= len(pRunner.slSteps) {
		pRunner.bDone = true
		return slSends, true
	}
	return slSends, false
}

// stop 은 스크립트를 멈춘다. 멈추기 전에 진행 중이었으면 기다리던 단계의 "기다릴 글자"와 true 를 돌려준다.
func (pRunner *loginRunner) stop() (szWaiting string, bWasActive bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	if pRunner.bDone {
		return "", false
	}
	pRunner.bDone = true
	if pRunner.iStep < len(pRunner.slSteps) {
		szWaiting = pRunner.slSteps[pRunner.iStep].Expect
	}
	return szWaiting, true
}

// stepIndex 는 다음에 실행할 단계 번호와 끝났는지를 돌려준다.
func (pRunner *loginRunner) stepIndex() (int, bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	return pRunner.iStep, pRunner.bDone
}

// startLogin 은 접속 직후의 자동 입력을 시작한다. 로그인 스크립트가 있으면 그것을 먼저 끝까지 실행한 뒤에
// 접속 후 입력(tmux 한 줄 또는 자동 실행 명령)을 보내고, 없으면 접속 후 입력만 예약한다 (session.md §5·§5.1).
func (pSession *Session) startLogin(conn Transport, iGen int) {
	if len(pSession.cfg.LoginScript) == 0 {
		pSession.scheduleStartup(iGen)
		return
	}
	pRunner := newLoginRunner(pSession.cfg.LoginScript)
	pSession.mtxData.Lock()
	if iGen != pSession.gen {
		pSession.mtxData.Unlock()
		return
	}
	pSession.login = pRunner
	pSession.mtxData.Unlock()
	pSession.armLoginTimeout(pRunner, iGen)
	// 첫 단계의 기다릴 글자가 비어 있으면 출력 없이도 바로 보낸다.
	pSession.feedLogin(conn, "", iGen)
}

// feedLogin 은 읽기 루프가 화면 출력 조각마다 부른다. 조건이 맞은 단계의 글자를 서버로 보낸다.
func (pSession *Session) feedLogin(conn Transport, szText string, iGen int) {
	pSession.mtxData.Lock()
	pRunner := pSession.login
	pSession.mtxData.Unlock()
	if pRunner == nil {
		return
	}
	slSends, bFinished := pRunner.feed(szText)
	for _, szSend := range slSends {
		if strings.Contains(szSend, G_cszLoginPassword) {
			szSecret := ""
			if provider, bOk := conn.(SecretProvider); bOk {
				szSecret = provider.LoginSecret()
			}
			if szSecret == "" {
				// 넣을 비밀번호가 없다: 빈 비밀번호를 보내 계정이 잠기는 일이 없게 여기서 멈춘다.
				pRunner.stop()
				pSession.clearLogin(pRunner)
				pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T("be.login.noPassword")))
				return
			}
			szSend = strings.ReplaceAll(szSend, G_cszLoginPassword, szSecret)
		}
		pSession.mtxData.Lock()
		bCurrent := iGen == pSession.gen && pSession.transport != nil
		pSession.mtxData.Unlock()
		if !bCurrent {
			return
		}
		_ = conn.Write(Encode(pSession.cfg.Encoding, szSend+"\r"))
	}
	if bFinished {
		pSession.clearLogin(pRunner)
		pSession.scheduleStartup(iGen)
		return
	}
	if len(slSends) > 0 {
		pSession.armLoginTimeout(pRunner, iGen) // 다음 단계의 시한을 새로 잰다
	}
}

// armLoginTimeout 은 지금 단계의 시한을 건다. 시한 안에 다음 단계로 넘어가지 못하면 스크립트를 멈추고 알린다.
func (pSession *Session) armLoginTimeout(pRunner *loginRunner, iGen int) {
	iStep, bDone := pRunner.stepIndex()
	if bDone {
		return
	}
	durTimeout := G_durLoginStepTimeout
	timer := time.AfterFunc(durTimeout, func() {
		if iNow, _ := pRunner.stepIndex(); iNow != iStep {
			return // 그사이 다음 단계로 넘어갔다
		}
		szWaiting, bWasActive := pRunner.stop()
		if !bWasActive || !pSession.isCurrent(iGen) {
			return
		}
		pSession.clearLogin(pRunner)
		pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T("be.login.timeout", "text", szWaiting, "sec", int(durTimeout/time.Second))))
	})
	pSession.mtxData.Lock()
	if pSession.timerLogin != nil {
		pSession.timerLogin.Stop()
	}
	pSession.timerLogin = timer
	pSession.mtxData.Unlock()
}

// clearLogin 은 끝났거나 멈춘 스크립트를 세션에서 뗀다 (다른 접속의 스크립트로 바뀌었으면 건드리지 않는다).
func (pSession *Session) clearLogin(pRunner *loginRunner) {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	if pSession.login != pRunner {
		return
	}
	pSession.login = nil
	if pSession.timerLogin != nil {
		pSession.timerLogin.Stop()
		pSession.timerLogin = nil
	}
}

// cancelLoginOnInput 은 사용자가 직접 입력하면 진행 중인 로그인 스크립트를 멈춘다.
// 프롬프트가 예상과 달라 사용자가 손으로 넘기는 동안 스크립트가 뒤늦게 글자를 보내는 일을 막는다.
// ESC 로 시작하는 입력은 세지 않는다: 터미널이 서버의 질의에 스스로 답하는 응답(커서 위치 등)이 그 모양이다.
func (pSession *Session) cancelLoginOnInput(szText string) {
	if szText == "" || strings.HasPrefix(szText, "\x1b") {
		return
	}
	pSession.mtxData.Lock()
	pRunner := pSession.login
	pSession.mtxData.Unlock()
	if pRunner == nil {
		return
	}
	if _, bWasActive := pRunner.stop(); !bWasActive {
		return
	}
	pSession.clearLogin(pRunner)
	pSession.writeOutput("\r\n" + notice(G_cszColorInfo, i18n.T("be.login.cancelled")))
}
```

### `frontend/src/lib/loginScript.ts` — 새 파일 (전체)

```ts
// 로그인 스크립트 편집용 계산. 근거: spec/ui/host-editor.md §1.4a, spec/behavior/session.md §5.1 (D139).
import type { LoginStep } from '../ipc/types';

/** 최대 단계 수(백엔드 G_ciLoginStepMax 와 같다). */
export const LOGIN_STEP_MAX = 20;
/** 한 단계가 기다리는 시간(초, 백엔드 G_durLoginStepTimeout 과 같다). 도움말 표시용. */
export const LOGIN_STEP_TIMEOUT_SEC = 20;
/** 보낼 글자에 쓰는 자리표시자: 이 서버에 로그인할 때 쓴 비밀번호로 바뀐다. */
export const LOGIN_PASSWORD_TOKEN = '{{password}}';

export function newLoginStep(): LoginStep {
  return { id: `ls-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`, expect: '', send: '' };
}

/** 저장할 단계: 기다릴 글자와 보낼 글자가 모두 빈 단계는 빼고, 최대 개수까지만. 하나도 없으면 undefined. */
export function cleanLoginScript(steps: readonly LoginStep[] | undefined): LoginStep[] | undefined {
  const out = (steps ?? []).filter((s) => s.expect !== '' || s.send !== '').slice(0, LOGIN_STEP_MAX);
  return out.length ? out : undefined;
}

/** i 번째 단계를 dir(-1 위, +1 아래)로 옮긴 새 배열. 옮길 수 없으면 같은 배열. */
export function moveStep<T>(steps: readonly T[], i: number, dir: -1 | 1): T[] {
  const j = i + dir;
  if (i < 0 || i >= steps.length || j < 0 || j >= steps.length) return steps as T[];
  const next = [...steps];
  [next[i], next[j]] = [next[j], next[i]];
  return next;
}
```

### `frontend/src/screens/LoginScriptTable.tsx` — 새 파일 (전체)

```tsx
// 서버 편집 "자동화" 탭: 로그인 스크립트 단계 표. 근거: spec/ui/host-editor.md §1.4a (D139).
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';
import type { LoginStep } from '../ipc/types';
import { LOGIN_PASSWORD_TOKEN, LOGIN_STEP_MAX, LOGIN_STEP_TIMEOUT_SEC, moveStep, newLoginStep } from '../lib/loginScript';
import { t } from '../i18n';

export function LoginScriptTable({ rows, onChange, isSsh }: { rows: LoginStep[]; onChange: (rows: LoginStep[]) => void; isSsh: boolean }) {
  const patch = (id: string, p: Partial<LoginStep>) => onChange(rows.map((r) => (r.id === id ? { ...r, ...p } : r)));
  return (
    <>
      <div className="notice info">{t('login.help', { sec: LOGIN_STEP_TIMEOUT_SEC })}</div>
      <div className="notice warn">
        {t('login.plainWarn')}
        {isSsh && ` ${t('login.passwordHint', { token: LOGIN_PASSWORD_TOKEN })}`}
      </div>
      {rows.length > 0 && (
        <table className="fwd-table" style={{ width: '100%', borderCollapse: 'separate', borderSpacing: '0 6px' }}>
          <thead>
            <tr className="faint small" style={{ textAlign: 'left' }}>
              <th style={{ width: 26 }}>#</th>
              <th>{t('login.expect')}</th>
              <th>{t('login.send')}</th>
              <th style={{ width: 96 }} />
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={r.id}>
                <td className="faint small">{i + 1}</td>
                <td>
                  <input className="input mono" aria-label={t('login.expectN', { n: i + 1 })} placeholder={t('login.expectPlaceholder')} value={r.expect} onChange={(e) => patch(r.id, { expect: e.target.value })} />
                </td>
                <td>
                  <input className="input mono" aria-label={t('login.sendN', { n: i + 1 })} placeholder={t('login.sendPlaceholder')} value={r.send} onChange={(e) => patch(r.id, { send: e.target.value })} />
                </td>
                <td style={{ whiteSpace: 'nowrap' }}>
                  <button type="button" className="icon-btn" aria-label={t('login.up')} title={t('login.up')} disabled={i === 0} onClick={() => onChange(moveStep(rows, i, -1))}>
                    <ArrowUp size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('login.down')} title={t('login.down')} disabled={i === rows.length - 1} onClick={() => onChange(moveStep(rows, i, 1))}>
                    <ArrowDown size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => onChange(rows.filter((x) => x.id !== r.id))}>
                    <Trash2 size={14} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <div>
        <button type="button" className="btn small-btn" disabled={rows.length >= LOGIN_STEP_MAX} onClick={() => onChange([...rows, newLoginStep()])}>
          {t('login.add')}
        </button>
      </div>
    </>
  );
}
```

### `frontend/src/ipc/types.ts` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -23,6 +23,15 @@
   bindPort: number;
   destHost: string;
   destPort: number;
+}
+
+/** 로그인 스크립트의 한 단계(behavior/session.md §5.1, D139). */
+export interface LoginStep {
+  id: string;
+  /** 서버 출력에서 기다릴 글자. 비면 기다리지 않고 바로 보낸다. */
+  expect: string;
+  /** 보낼 글자(뒤에 Enter 가 붙는다). `{{password}}` 는 로그인 비밀번호로 바뀐다. */
+  send: string;
 }
 
 export interface Host {
@@ -47,6 +56,8 @@
   encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'euc-jp' | 'gbk' | 'big5' | 'latin1';
   termType: string;
   startupCommand?: string;
+  /** 접속 직후 실행하는 로그인 스크립트(store.md §3 loginScript). 로컬 터미널에는 없다. */
+  loginScript?: LoginStep[];
   keepSession?: boolean;
   tmuxName?: string;
   keepaliveSec: number;
```

### `frontend/src/lib/hostSave.ts` — 수정

```diff
--- a/frontend/src/lib/hostSave.ts
+++ b/frontend/src/lib/hostSave.ts
@@ -2,6 +2,7 @@
 import type { AuthType, CredMeta, Host, Protocol } from '../ipc/types';
 import { getBackend } from '../ipc/backend';
 import { JUMP_DIRECT, localOptions, PROXY_DEFAULT_PORT, type JumpDirect, type ProxyConf, type ProxyType, type SerialOpts } from './hostForm';
+import { cleanLoginScript } from './loginScript';
 import { t } from '../i18n';
 
 export interface SaveInput {
@@ -29,6 +30,8 @@
   termType: string;
   keepaliveSec: number;
   startupCommand?: string;
+  /** 로그인 스크립트 단계(빈 단계는 저장할 때 뺀다). */
+  loginScript?: Host['loginScript'];
   keepSession: boolean;
   tmuxName: string;
   forwards: Host['forwards'];
@@ -176,6 +179,7 @@
     termType: i.termType.trim() || 'xterm-256color',
     keepaliveSec: isNet ? Number(i.keepaliveSec) || 0 : base.keepaliveSec,
     startupCommand: i.startupCommand?.trim() || undefined,
+    loginScript: isLocal ? undefined : cleanLoginScript(i.loginScript),
     keepSession: i.protocol === 'ssh' || isLocal ? i.keepSession : undefined,
     tmuxName: (i.protocol === 'ssh' || isLocal) && i.keepSession ? i.tmuxName.trim() || undefined : undefined,
     forwards: i.protocol === 'ssh' ? i.forwards : [],
```

### `frontend/src/screens/HostEditor.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditor.tsx
+++ b/frontend/src/screens/HostEditor.tsx
@@ -15,9 +15,10 @@
 } from '../lib/hostForm';
 import { applyHost, buildHostFromForm } from '../lib/hostSave';
 import { ForwardsTable, JumpDirectCard, ProxyCard, SerialFields } from './HostEditorParts';
+import { LoginScriptTable } from './LoginScriptTable';
 import { t } from '../i18n';
 
-type Tab = 'basic' | 'auth' | 'adv' | 'fwd';
+type Tab = 'basic' | 'auth' | 'adv' | 'fwd' | 'auto';
 
 // 선택 목록은 그릴 때 언어에 맞춰 만든다(D75).
 const PROTOCOLS = (): { value: Protocol; label: string }[] => [
@@ -106,6 +107,8 @@
   const [termType, setTermType] = useState(original?.termType ?? 'xterm-256color');
   const [keepalive, setKeepalive] = useState(String(original?.keepaliveSec ?? 30));
   const [startup, setStartup] = useState(original?.startupCommand ?? '');
+  // 자동화(§1.4a): 로그인 스크립트
+  const [loginScript, setLoginScript] = useState(original?.loginScript ?? []);
   const [keepSession, setKeepSession] = useState<boolean | undefined>(original?.keepSession);
   const [tmuxName, setTmuxName] = useState(original?.tmuxName ?? '');
   const [tmuxAvail, setTmuxAvail] = useState(false);
@@ -193,6 +196,7 @@
     }
     if (next !== 'ssh' && tab === 'auth') setTab('basic');
     if (next !== 'ssh' && tab === 'fwd') setTab('basic');
+    if (next === 'local' && tab === 'auto') setTab('basic');
   };
 
   // 호스트 칸 입력 나누기(§1.1).
@@ -281,6 +285,7 @@
         termType,
         keepaliveSec: Number(keepalive) || 0,
         startupCommand: startup,
+        loginScript,
         keepSession: !!tmuxOn,
         tmuxName,
         forwards,
@@ -317,6 +322,7 @@
     ...(isSsh ? [{ id: 'auth' as Tab, label: t('host.tabAuth') }] : []),
     { id: 'adv', label: t('host.tabAdv') },
     ...(isSsh ? [{ id: 'fwd' as Tab, label: t('host.tabFwd') }] : []),
+    ...(!isLocal ? [{ id: 'auto' as Tab, label: t('host.tabAuto') }] : []),
   ];
   const curTab = tabs.some((t) => t.id === tab) ? tab : 'basic';
   const shellOptions = [
@@ -595,6 +601,8 @@
         )}
 
         {curTab === 'fwd' && isSsh && <ForwardsTable rows={forwards} onChange={setForwards} />}
+
+        {curTab === 'auto' && !isLocal && <LoginScriptTable rows={loginScript} onChange={setLoginScript} isSsh={isSsh} />}
       </div>
     </Modal>
   );
```

### `internal/app/channels_session.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -40,6 +40,25 @@
 	M_slForwards       []forwardRecord   `json:"forwards"`
 	M_pProxy           *proxyRecord      `json:"proxy"`
 	M_bAgentForward    bool              `json:"agentForward"` // SSH 에이전트 전달 (D111)
+	M_slLoginScript    []loginStepRecord `json:"loginScript"`  // 로그인 스크립트 (D139)
+}
+
+// loginStepRecord 는 로그인 스크립트의 한 단계다 (data/store.md §3 loginScript).
+type loginStepRecord struct {
+	M_szExpect string `json:"expect"`
+	M_szSend   string `json:"send"`
+}
+
+// loginScriptOf 는 서버의 로그인 스크립트를 세션 쪽 형식으로 바꾼다. 로컬 터미널에는 쓰지 않는다.
+func loginScriptOf(record hostRecord) []session.LoginStep {
+	if record.M_szProtocol == "local" {
+		return nil
+	}
+	slSteps := make([]session.LoginStep, 0, len(record.M_slLoginScript))
+	for _, step := range record.M_slLoginScript {
+		slSteps = append(slSteps, session.LoginStep{Expect: step.M_szExpect, Send: step.M_szSend})
+	}
+	return session.TrimLoginSteps(slSteps)
 }
 
 // proxyRecord 는 서버의 프록시 경유 설정이다 (data/store.md §3 proxy, D110).
@@ -213,6 +244,7 @@
 	}
 	cfg.Info.M_szTarget = targetOf(record)
 	cfg.StartupInput, cfg.StartupDelay = startupInputOf(record)
+	cfg.LoginScript = loginScriptOf(record)
 	info, errOpen := pService.pSessions.Open(cfg)
 	if errOpen == nil && args.M_iCols > 0 && args.M_iRows > 0 {
 		// 열 때 지정한 크기는 접속 직후 프로토콜에 알린다 (session.md §8).
```

### `internal/app/ssh_app.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/app/ssh_app.go
+++ b/internal/app/ssh_app.go
@@ -21,8 +21,20 @@
 // sshSession 은 SSH 셸과 그 연결·포워딩을 함께 닫는 전송이다.
 type sshSession struct {
 	*sshconn.Shell
-	conn    *sshconn.Conn
-	closers []io.Closer
+	conn     *sshconn.Conn
+	closers  []io.Closer
+	forwards []sshconn.ForwardResult
+	password string // 로그인에 쓴 비밀번호 (로그인 스크립트의 {{password}} 용, 세션 사본과 같은 수명)
+}
+
+// LoginSecret 은 이 서버에 로그인할 때 쓴 비밀번호다 (session.SecretProvider). 키·에이전트로 로그인했으면 빈 글자다.
+func (pSess *sshSession) LoginSecret() string {
+	return pSess.password
+}
+
+// Forwards 는 이 연결에 설정한 포트 포워딩과 그 결과, 지금 이어져 있는 연결 수다 (session.ForwardReporter).
+func (pSess *sshSession) Forwards() []session.ForwardInfo {
+	return forwardInfos(pSess.forwards)
 }
 
 // Close 는 셸, 포워딩 소켓, 연결을 모두 닫는다.
```

### `internal/session/control.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/control.go
+++ b/internal/session/control.go
@@ -24,6 +24,7 @@
 	}
 	pSession.mtxData.Unlock()
 	if transport != nil {
+		pSession.cancelLoginOnInput(szText) // 사용자가 직접 입력하면 로그인 스크립트는 멈춘다 (session.md §5.1)
 		if pSession.zActive.Load() {
 			// ZMODEM 전송 중: 입력이 전송 데이터에 섞이면 안 되므로 보내지 않는다. Ctrl+C 는 전송 취소다 (zmodem.md).
 			if strings.Contains(szText, "\x03") {
@@ -88,6 +89,11 @@
 		pSession.timerStartup.Stop()
 		pSession.timerStartup = nil
 	}
+	if pSession.timerLogin != nil {
+		pSession.timerLogin.Stop()
+		pSession.timerLogin = nil
+	}
+	pSession.login = nil
 	transport := pSession.transport
 	pSession.transport = nil
 	pSession.info.M_szState = G_cszStateClosed
```

### `internal/session/session.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -33,6 +33,8 @@
 	iRows        int
 	timerRetry   *time.Timer
 	timerStartup *time.Timer
+	timerLogin   *time.Timer  // 로그인 스크립트의 단계 시한 (session.md §5.1)
+	login        *loginRunner // 지금 연결에서 진행 중인 로그인 스크립트 (없으면 nil)
 	decoder      *Decoder
 	batcher      *Batcher
 	credit       *Credit // 출력 흐름 제어 (session.md §8)
@@ -181,8 +187,9 @@
 		pSession.writeOutput(G_cszClearLine)
 	}
 	pSession.publishState()
+	// 읽기 루프보다 먼저 건다: 접속 직후 이미 와 있는 출력(첫 프롬프트)을 스크립트가 놓치지 않게.
+	pSession.startLogin(conn, iGen)
 	go pSession.readLoop(conn, iGen)
-	pSession.scheduleStartup(iGen)
 }
 
 // onConnectFailed 는 접속 실패를 처리한다: 자동 재접속 규칙이 맞으면 예약, 아니면 error 상태.
@@ -367,7 +374,9 @@
 				}
 			}
 			if len(bScreen) > 0 {
-				pSession.writeOutput(pSession.decoder.Decode(bScreen))
+				szScreen := pSession.decoder.Decode(bScreen)
+				pSession.writeOutput(szScreen)
+				pSession.feedLogin(conn, szScreen, iGen)
 			}
 			if bMangled && time.Since(tmMangledNotice) >= G_durZmodemMangledNotice {
 				// 시작 표시가 깨져서 왔다(tmux·screen 안): 전송은 될 수 없으니 까닭과 끝내는 방법을 알린다.
```

### `internal/session/types.go` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/internal/session/types.go
+++ b/internal/session/types.go
@@ -62,6 +79,9 @@
 	Zmodem *ZmodemHooks
 	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
 	StartupInput string
+	// LoginScript 는 접속 직후 실행하는 로그인 스크립트다 (session.md §5.1, D139). 비면 쓰지 않는다.
+	// 있으면 StartupInput 은 스크립트가 끝까지 실행된 뒤에 보낸다.
+	LoginScript []LoginStep
 	// StartupDelay 는 접속 후 StartupInput 을 보내기까지의 대기다 (tmux 500ms, 자동 실행 명령 400ms).
 	StartupDelay time.Duration
 	// OpenDelay 는 열기 후 접속을 시작하기까지의 대기다 (기본 30ms, 프런트가 터미널을 등록할 시간).
```

## C. 테스트 코드

### `internal/session/loginscript_test.go` — 새 파일 (전체)

```go
package session

import (
	"dariterm/internal/i18n"
	"strings"
	"testing"
	"time"
)

// 단계는 순서대로, 기다릴 글자가 나왔을 때만 넘어간다 (session.md §5.1).
func TestLoginRunnerSteps(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{
		{Expect: "$ ", Send: "sudo su -"},
		{Expect: "password", Send: "secret"},
		{Expect: "# ", Send: "cd /var/log"},
	})
	if slSends, _ := pRunner.feed("Last login: Thu\r\n"); len(slSends) != 0 {
		t.Fatalf("프롬프트 전에 보냈다: %v", slSends)
	}
	slSends, bFinished := pRunner.feed("[user@host ~]$ ")
	if len(slSends) != 1 || slSends[0] != "sudo su -" || bFinished {
		t.Fatalf("1단계 = %v, 끝=%v", slSends, bFinished)
	}
	// 대소문자를 가리지 않고, 조각이 나뉘어 와도 찾는다.
	if slSends, _ = pRunner.feed("[sudo] Pass"); len(slSends) != 0 {
		t.Fatalf("반쪽에서 보냈다: %v", slSends)
	}
	slSends, _ = pRunner.feed("word for user: ")
	if len(slSends) != 1 || slSends[0] != "secret" {
		t.Fatalf("2단계 = %v", slSends)
	}
	slSends, bFinished = pRunner.feed("\r\n[root@host ~]# ")
	if len(slSends) != 1 || slSends[0] != "cd /var/log" || !bFinished {
		t.Fatalf("3단계 = %v, 끝=%v", slSends, bFinished)
	}
	if slSends, _ = pRunner.feed("# # # "); len(slSends) != 0 {
		t.Fatalf("끝난 뒤에 보냈다: %v", slSends)
	}
}

// 한 번 맞은 글자는 다음 단계에 다시 쓰이지 않는다: 같은 프롬프트를 두 번 기다리면 두 번 나와야 한다.
func TestLoginRunnerDoesNotReuseMatch(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Expect: "$ ", Send: "a"}, {Expect: "$ ", Send: "b"}})
	slSends, bFinished := pRunner.feed("host$ ")
	if len(slSends) != 1 || bFinished {
		t.Fatalf("첫 프롬프트에서 = %v, 끝=%v", slSends, bFinished)
	}
	slSends, bFinished = pRunner.feed("host$ ")
	if len(slSends) != 1 || slSends[0] != "b" || !bFinished {
		t.Fatalf("둘째 프롬프트에서 = %v, 끝=%v", slSends, bFinished)
	}
}

// 색 시퀀스가 글자 사이에 끼어 있거나 조각 경계에서 잘려도 찾는다.
func TestLoginRunnerIgnoresAnsi(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Expect: "login:", Send: "admin"}})
	if slSends, _ := pRunner.feed("\x1b[1mlog\x1b[3"); len(slSends) != 0 {
		t.Fatalf("반쪽에서 보냈다: %v", slSends)
	}
	slSends, bFinished := pRunner.feed("2min:\x1b[0m ")
	if len(slSends) != 1 || !bFinished {
		t.Fatalf("= %v, 끝=%v", slSends, bFinished)
	}
}

// 기다릴 글자가 빈 단계는 출력 없이 바로 보낸다.
func TestLoginRunnerEmptyExpect(t *testing.T) {
	pRunner := newLoginRunner([]LoginStep{{Send: ""}, {Expect: ">", Send: "enable"}})
	slSends, bFinished := pRunner.feed("")
	if len(slSends) != 1 || slSends[0] != "" || bFinished {
		t.Fatalf("= %v, 끝=%v", slSends, bFinished)
	}
}

func TestTrimLoginSteps(t *testing.T) {
	slIn := []LoginStep{{}, {Expect: "a"}, {Send: "b"}}
	for iIndex := 0; iIndex < 30; iIndex++ {
		slIn = append(slIn, LoginStep{Expect: "x", Send: "y"})
	}
	slOut := TrimLoginSteps(slIn)
	if len(slOut) != G_ciLoginStepMax || slOut[0].Expect != "a" || slOut[1].Send != "b" {
		t.Fatalf("개수 = %d, 앞 = %+v", len(slOut), slOut[:2])
	}
}

// secretTransport 는 로그인 비밀번호를 알려 주는 테스트용 연결이다.
type secretTransport struct {
	*fakeTransport
	szSecret string
}

func (pFake *secretTransport) LoginSecret() string { return pFake.szSecret }

func (pFake *fakeTransport) writes() []string {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return append([]string(nil), pFake.slWrites...)
}

// waitWrites 는 보낸 입력이 iCount 개가 될 때까지 기다린다.
func waitWrites(t *testing.T, pFake *fakeTransport, iCount int) []string {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if slWrites := pFake.writes(); len(slWrites) >= iCount {
			return slWrites
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("입력 %d개를 기다렸지만 %v", iCount, pFake.writes())
	return nil
}

func openLoginSession(t *testing.T, conn Transport, cfg OpenConfig) (*Manager, *eventRecorder, string) {
	t.Helper()
	pRec := &eventRecorder{}
	pManager := NewManager(pRec.emit)
	cfg.Info = Info{M_szTitle: "login", M_szProtocol: "ssh"}
	cfg.Dialer = func(notify func(string)) (Transport, error) { return conn, nil }
	cfg.OpenDelay = time.Millisecond
	info, errOpen := pManager.Open(cfg)
	if errOpen != nil {
		t.Fatalf("Open: %v", errOpen)
	}
	return pManager, pRec, info.M_szId
}

// 세션은 출력에서 기다릴 글자를 찾아 보내고, {{password}} 를 로그인 비밀번호로 바꾸고,
// 스크립트가 끝난 뒤에 접속 후 입력을 보낸다.
func TestSessionRunsLoginScript(t *testing.T) {
	pFake := &secretTransport{fakeTransport: newFakeTransport(), szSecret: "pw!"}
	pManager, _, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "$ ", Send: "sudo su -"}, {Expect: "password", Send: G_cszLoginPassword}},
		StartupInput: "cd /srv\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)

	pFake.chIn <- []byte("Welcome\r\nuser@host:~$ ")
	if slWrites := waitWrites(t, pFake.fakeTransport, 1); slWrites[0] != "sudo su -\r" {
		t.Fatalf("1단계 입력 = %q", slWrites[0])
	}
	pFake.chIn <- []byte("[sudo] password for user: ")
	slWrites := waitWrites(t, pFake.fakeTransport, 3)
	if slWrites[1] != "pw!\r" {
		t.Errorf("2단계 입력 = %q", slWrites[1])
	}
	if slWrites[2] != "cd /srv\r" {
		t.Errorf("스크립트 뒤 접속 후 입력 = %q", slWrites[2])
	}
}

// 넣을 비밀번호가 없으면 빈 비밀번호를 보내지 않고 멈춘다.
func TestSessionLoginScriptStopsWithoutPassword(t *testing.T) {
	i18n.SetLang("ko")
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "password", Send: G_cszLoginPassword}},
		StartupInput: "never\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)
	pFake.chIn <- []byte("Password: ")
	waitOutput(t, pRec, "저장된 비밀번호가 없어")
	if slWrites := pFake.writes(); len(slWrites) != 0 {
		t.Fatalf("보내면 안 된다: %v", slWrites)
	}
}

// 기다릴 글자가 시한 안에 나오지 않으면 멈추고 알린다. 접속 후 입력도 보내지 않는다.
func TestSessionLoginScriptTimeout(t *testing.T) {
	i18n.SetLang("ko")
	durOld := G_durLoginStepTimeout
	G_durLoginStepTimeout = 40 * time.Millisecond
	defer func() { G_durLoginStepTimeout = durOld }()
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript:  []LoginStep{{Expect: "never-shown", Send: "x"}},
		StartupInput: "never\r",
		StartupDelay: time.Millisecond,
	})
	defer pManager.Close(szId)
	pFake.chIn <- []byte("something else\r\n")
	waitOutput(t, pRec, "never-shown")
	time.Sleep(20 * time.Millisecond)
	if slWrites := pFake.writes(); len(slWrites) != 0 {
		t.Fatalf("보내면 안 된다: %v", slWrites)
	}
}

// 사용자가 직접 입력하면 스크립트는 멈춘다. 터미널의 자동 응답(ESC 로 시작)은 멈추지 않는다.
func TestSessionLoginScriptCancelledByInput(t *testing.T) {
	i18n.SetLang("ko")
	pFake := newFakeTransport()
	pManager, pRec, szId := openLoginSession(t, pFake, OpenConfig{
		LoginScript: []LoginStep{{Expect: "password", Send: "secret"}},
	})
	defer pManager.Close(szId)
	pSession, _ := pManager.Get(szId)
	waitConnected(t, pSession)

	pSession.Input("\x1b[?1;2c") // 터미널의 자동 응답
	pSession.mtxData.Lock()
	bStill := pSession.login != nil
	pSession.mtxData.Unlock()
	if !bStill {
		t.Fatal("자동 응답에 스크립트가 멈췄다")
	}
	pSession.Input("ls\r")
	waitOutput(t, pRec, "로그인 스크립트를 멈췄습니다")
	pFake.chIn <- []byte("Password: ")
	time.Sleep(30 * time.Millisecond)
	for _, szWrite := range pFake.writes() {
		if strings.Contains(szWrite, "secret") {
			t.Fatalf("멈춘 뒤에 보냈다: %q", szWrite)
		}
	}
}

func waitConnected(t *testing.T, pSession *Session) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if pSession.Snapshot().M_szState == G_cszStateConnected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("연결되지 않았다")
}

// waitOutput 은 화면 출력에 szWant 가 나올 때까지 기다린다.
func waitOutput(t *testing.T, pRec *eventRecorder, szWant string) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		pRec.mtxData.Lock()
		szAll := strings.Join(pRec.slData, "")
		pRec.mtxData.Unlock()
		if strings.Contains(szAll, szWant) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	t.Fatalf("출력에 %q 가 없다: %q", szWant, strings.Join(pRec.slData, ""))
}
```

### `frontend/src/screens/Automation.test.tsx` — 새 파일 (이 문서의 몫: `describe('서버 편집 · 자동화 탭 (D139)')`, 전체는 5번 문서 §C)

```tsx
describe('서버 편집 · 자동화 탭 (D139)', () => {
  const host: Host = {
    id: 'h1', alias: 'web', groupId: null, protocol: 'ssh', host: '10.0.0.5', port: 22, username: 'deploy',
    authType: 'ask', tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 1,
  };

  it('로그인 스크립트 단계를 넣고 저장하면 서버에 loginScript 로 남는다(빈 단계는 빠진다)', async () => {
    seed({ hosts: [host] });
    useUi.getState().openEditor({ kind: 'host', hostId: 'h1', groupId: null });
    render(<HostEditor hostId="h1" groupId={null} />);
    fireEvent.click(screen.getByRole('tab', { name: '자동화' }));
    expect(screen.getByText(/로그인 스크립트: 연결 직후/)).toBeTruthy();
    expect(screen.getByText(/\{\{password\}\}/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '+ 단계 추가' }));
    fireEvent.click(screen.getByRole('button', { name: '+ 단계 추가' }));
    fireEvent.change(screen.getByLabelText('1단계 기다릴 글자'), { target: { value: '$ ' } });
    fireEvent.change(screen.getByLabelText('1단계 보낼 글자'), { target: { value: 'sudo su -' } });
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '저장' })); });

    await waitFor(() => expect(patches.some((p) => p.hosts)).toBe(true));
    const saved = patches.find((p) => p.hosts)!.hosts![0];
    expect(saved.loginScript?.map((s) => [s.expect, s.send])).toEqual([['$ ', 'sudo su -']]);
  });

  it('로컬 터미널에는 자동화 탭이 없다', () => {
    seed({ hosts: [{ ...host, protocol: 'local', host: '', port: 0, username: '' }] });
    render(<HostEditor hostId="h1" groupId={null} />);
    expect(screen.queryByRole('tab', { name: '자동화' })).toBeNull();
    expect(screen.queryByRole('tab', { name: '포트 포워딩' })).toBeNull();
  });

  it('텔넷에는 자동화 탭이 있고 비밀번호 자리표시자 안내는 없다', () => {
    seed({ hosts: [{ ...host, protocol: 'telnet', port: 23 }] });
    render(<HostEditor hostId="h1" groupId={null} />);
    fireEvent.click(screen.getByRole('tab', { name: '자동화' }));
    expect(screen.queryByText(/\{\{password\}\}/)).toBeNull();
  });
});
```

### `frontend/src/lib/macro.test.ts` — 새 파일 (이 문서의 몫: `describe('매크로·로그인 스크립트 단계 다듬기')`, 전체는 5번 문서 §C)

```ts
describe('매크로·로그인 스크립트 단계 다듬기', () => {
  it('보낼 것이 없는 매크로 단계는 뺀다', () => {
    const steps = [newMacroStep({ send: 'a' }), newMacroStep({ send: '', enter: false }), newMacroStep({ send: '', enter: true })];
    expect(cleanSteps(steps).map((s) => [s.send, s.enter])).toEqual([['a', true], ['', true]]);
    expect(macroBody(cleanSteps(steps))).toBe('a\n');
  });

  it('단계가 있어야 매크로다', () => {
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true })).toBe(false);
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true, steps: [] })).toBe(false);
    expect(isMacro({ id: '1', name: '', body: 'ls', sendEnter: true, steps: [newMacroStep({ send: 'ls' })] })).toBe(true);
  });

  it('로그인 스크립트: 빈 단계는 빼고, 하나도 없으면 필드를 두지 않는다', () => {
    expect(cleanLoginScript([newLoginStep()])).toBeUndefined();
    expect(cleanLoginScript(undefined)).toBeUndefined();
    const many = Array.from({ length: LOGIN_STEP_MAX + 3 }, () => ({ ...newLoginStep(), expect: '$', send: 'x' }));
    expect(cleanLoginScript(many)).toHaveLength(LOGIN_STEP_MAX);
  });

  it('단계 옮기기', () => {
    expect(moveStep(['a', 'b', 'c'], 0, 1)).toEqual(['b', 'a', 'c']);
    expect(moveStep(['a', 'b', 'c'], 2, -1)).toEqual(['a', 'c', 'b']);
    const same = ['a', 'b'];
    expect(moveStep(same, 0, -1)).toBe(same);
  });
});
```

## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다. 자리표시자(`{이름}`)는 언어마다 같은 이름을 쓴다(`frontend: npx vitest run src/i18n`, `go test ./internal/i18n` 이 확인한다).

**`host.tabAuto`** (추가)

```json
  "host.tabAuto": "자동화",
  "host.tabAuto": "Automation",
  "host.tabAuto": "自動化",
  "host.tabAuto": "自动化",
  "host.tabAuto": "自動化",
  "host.tabAuto": "Automatisierung",
  "host.tabAuto": "Automatisation",
  "host.tabAuto": "Automatización",
  "host.tabAuto": "Automação",
  "host.tabAuto": "Автоматизация",
```

**`login.help`** (추가)

```json
  "login.help": "로그인 스크립트: 연결 직후, 서버 출력에 \"기다릴 글자\"가 나오면 \"보낼 글자\"와 Enter를 입력합니다. 위에서 아래로 한 번씩 실행하고, 다시 연결할 때마다 처음부터 합니다. 기다릴 글자를 비우면 바로 보내며, 대소문자는 구분하지 않습니다. {sec}초 안에 나오지 않거나 직접 입력하면 멈춥니다. 고급 탭의 \"연결 후 실행할 명령\"은 스크립트가 끝난 뒤에 실행됩니다.",
  "login.help": "Login script: right after connecting, when the \"wait for\" text appears in the server output, the \"send\" text and Enter are typed. Steps run once each from top to bottom, and start over on every reconnect. An empty \"wait for\" sends immediately; matching ignores case. The script stops if the text does not appear within {sec} seconds or if you type. The \"command to run after connecting\" on the Advanced tab runs after the script finishes.",
  "login.help": "ログインスクリプト: 接続直後、サーバーの出力に「待つ文字」が現れると「送る文字」と Enter を入力します。上から順に 1 回ずつ実行し、再接続のたびに最初からやり直します。待つ文字が空ならすぐに送り、大文字と小文字は区別しません。{sec} 秒以内に現れない場合や自分で入力した場合は停止します。詳細タブの「接続後に実行するコマンド」はスクリプトの終了後に実行されます。",
  "login.help": "登录脚本：连接后，服务器输出中出现“等待的文字”时，输入“发送的文字”和 Enter。各步骤自上而下各执行一次，每次重新连接都从头开始。等待的文字留空则立即发送，不区分大小写。{sec} 秒内未出现或您自己输入时会停止。高级选项卡中的“连接后运行的命令”在脚本结束后执行。",
  "login.help": "登入指令碼：連線後，伺服器輸出中出現「等待的文字」時，輸入「傳送的文字」和 Enter。各步驟由上而下各執行一次，每次重新連線都從頭開始。等待的文字留空則立即傳送，不區分大小寫。{sec} 秒內未出現或您自行輸入時會停止。進階分頁中的「連線後執行的命令」在指令碼結束後執行。",
  "login.help": "Anmeldeskript: Direkt nach dem Verbinden wird, sobald der Text unter „Warten auf“ in der Serverausgabe erscheint, der Text unter „Senden“ mit Enter eingegeben. Die Schritte laufen je einmal von oben nach unten und beginnen bei jeder Neuverbindung von vorn. Ein leeres „Warten auf“ sendet sofort; Groß-/Kleinschreibung wird nicht beachtet. Das Skript hält an, wenn der Text nicht innerhalb von {sec} Sekunden erscheint oder Sie selbst tippen. Der „Befehl nach dem Verbinden“ im Tab Erweitert wird nach dem Skript ausgeführt.",
  "login.help": "Script de connexion : juste après la connexion, quand le texte « attendre » apparaît dans la sortie du serveur, le texte « envoyer » suivi d'Entrée est saisi. Les étapes s'exécutent une fois chacune de haut en bas et reprennent du début à chaque reconnexion. Un « attendre » vide envoie immédiatement ; la casse est ignorée. Le script s'arrête si le texte n'apparaît pas en {sec} secondes ou si vous tapez. La « commande à exécuter après la connexion » de l'onglet Avancé s'exécute après le script.",
  "login.help": "Script de inicio de sesión: justo después de conectar, cuando el texto «esperar» aparece en la salida del servidor, se escribe el texto «enviar» y Enter. Los pasos se ejecutan una vez cada uno de arriba abajo y vuelven a empezar en cada reconexión. Un «esperar» vacío envía de inmediato; no se distinguen mayúsculas. El script se detiene si el texto no aparece en {sec} segundos o si usted escribe. El «comando a ejecutar tras conectar» de la pestaña Avanzado se ejecuta al terminar el script.",
  "login.help": "Script de login: logo após conectar, quando o texto \"aguardar\" aparece na saída do servidor, o texto \"enviar\" e Enter são digitados. Os passos são executados uma vez cada, de cima para baixo, e recomeçam a cada reconexão. Um \"aguardar\" vazio envia imediatamente; maiúsculas e minúsculas não são diferenciadas. O script para se o texto não aparecer em {sec} segundos ou se você digitar. O \"comando a executar após conectar\" da aba Avançado é executado depois do script.",
  "login.help": "Сценарий входа: сразу после подключения, когда в выводе сервера появляется текст «ждать», вводится текст «отправить» и Enter. Шаги выполняются по одному разу сверху вниз и начинаются заново при каждом переподключении. Пустое поле «ждать» отправляет сразу; регистр не учитывается. Сценарий останавливается, если текст не появился за {sec} с или если вы начали вводить сами. «Команда после подключения» на вкладке «Дополнительно» выполняется после сценария.",
```

**`login.plainWarn`** (추가)

```json
  "login.plainWarn": "보낼 글자는 암호화되지 않은 채 서버 목록에 저장되고 git 동기화·백업에도 들어갑니다. 비밀번호를 직접 적지 마세요.",
  "login.plainWarn": "The text to send is stored unencrypted in the server list and is included in git sync and backups. Do not type passwords here.",
  "login.plainWarn": "送る文字は暗号化されずにサーバー一覧に保存され、git 同期やバックアップにも含まれます。パスワードを直接書かないでください。",
  "login.plainWarn": "发送的文字会以未加密的形式保存在服务器列表中，并包含在 git 同步和备份里。请不要直接写入密码。",
  "login.plainWarn": "傳送的文字會以未加密的形式儲存在伺服器清單中，並包含在 git 同步和備份裡。請不要直接寫入密碼。",
  "login.plainWarn": "Der zu sendende Text wird unverschlüsselt in der Serverliste gespeichert und ist in git-Synchronisierung und Backups enthalten. Geben Sie hier keine Passwörter ein.",
  "login.plainWarn": "Le texte à envoyer est enregistré non chiffré dans la liste des serveurs et inclus dans la synchronisation git et les sauvegardes. N'y saisissez pas de mot de passe.",
  "login.plainWarn": "El texto a enviar se guarda sin cifrar en la lista de servidores y se incluye en la sincronización git y las copias de seguridad. No escriba contraseñas aquí.",
  "login.plainWarn": "O texto a enviar é salvo sem criptografia na lista de servidores e incluído na sincronização git e nos backups. Não digite senhas aqui.",
  "login.plainWarn": "Отправляемый текст хранится в списке серверов без шифрования и попадает в синхронизацию git и резервные копии. Не вводите сюда пароли.",
```

**`login.passwordHint`** (추가)

```json
  "login.passwordHint": "대신 {token} 을 적으면 이 서버에 로그인할 때 쓴 비밀번호로 바뀝니다 (비밀번호로 로그인한 경우만).",
  "login.passwordHint": "Write {token} instead; it is replaced with the password used to log in to this server (password logins only).",
  "login.passwordHint": "代わりに {token} と書くと、このサーバーへのログインに使ったパスワードに置き換わります (パスワードでログインした場合のみ)。",
  "login.passwordHint": "请改写 {token}，它会被替换为登录此服务器时使用的密码（仅限密码登录）。",
  "login.passwordHint": "請改寫 {token}，它會被替換為登入此伺服器時使用的密碼（僅限密碼登入）。",
  "login.passwordHint": "Schreiben Sie stattdessen {token}; es wird durch das Passwort ersetzt, mit dem Sie sich an diesem Server angemeldet haben (nur bei Passwortanmeldung).",
  "login.passwordHint": "Écrivez plutôt {token} ; il est remplacé par le mot de passe utilisé pour vous connecter à ce serveur (connexion par mot de passe uniquement).",
  "login.passwordHint": "Escriba {token} en su lugar; se sustituye por la contraseña usada para iniciar sesión en este servidor (solo inicios con contraseña).",
  "login.passwordHint": "Escreva {token} no lugar; ele é substituído pela senha usada para entrar neste servidor (somente logins por senha).",
  "login.passwordHint": "Вместо этого напишите {token} — он заменяется паролем, с которым выполнен вход на этот сервер (только при входе по паролю).",
```

**`login.expect`** (추가)

```json
  "login.expect": "기다릴 글자",
  "login.expect": "Wait for",
  "login.expect": "待つ文字",
  "login.expect": "等待的文字",
  "login.expect": "等待的文字",
  "login.expect": "Warten auf",
  "login.expect": "Attendre",
  "login.expect": "Esperar",
  "login.expect": "Aguardar",
  "login.expect": "Ждать",
```

**`login.send`** (추가)

```json
  "login.send": "보낼 글자",
  "login.send": "Send",
  "login.send": "送る文字",
  "login.send": "发送的文字",
  "login.send": "傳送的文字",
  "login.send": "Senden",
  "login.send": "Envoyer",
  "login.send": "Enviar",
  "login.send": "Enviar",
  "login.send": "Отправить",
```

**`login.expectN`** (추가)

```json
  "login.expectN": "{n}단계 기다릴 글자",
  "login.expectN": "Step {n}: wait for",
  "login.expectN": "ステップ {n} の待つ文字",
  "login.expectN": "第 {n} 步等待的文字",
  "login.expectN": "第 {n} 步等待的文字",
  "login.expectN": "Schritt {n}: warten auf",
  "login.expectN": "Étape {n} : attendre",
  "login.expectN": "Paso {n}: esperar",
  "login.expectN": "Passo {n}: aguardar",
  "login.expectN": "Шаг {n}: ждать",
```

**`login.sendN`** (추가)

```json
  "login.sendN": "{n}단계 보낼 글자",
  "login.sendN": "Step {n}: send",
  "login.sendN": "ステップ {n} の送る文字",
  "login.sendN": "第 {n} 步发送的文字",
  "login.sendN": "第 {n} 步傳送的文字",
  "login.sendN": "Schritt {n}: senden",
  "login.sendN": "Étape {n} : envoyer",
  "login.sendN": "Paso {n}: enviar",
  "login.sendN": "Passo {n}: enviar",
  "login.sendN": "Шаг {n}: отправить",
```

**`login.expectPlaceholder`** (추가)

```json
  "login.expectPlaceholder": "예: password:",
  "login.expectPlaceholder": "e.g. password:",
  "login.expectPlaceholder": "例: password:",
  "login.expectPlaceholder": "例如：password:",
  "login.expectPlaceholder": "例如：password:",
  "login.expectPlaceholder": "z. B. password:",
  "login.expectPlaceholder": "ex. : password:",
  "login.expectPlaceholder": "p. ej.: password:",
  "login.expectPlaceholder": "ex.: password:",
  "login.expectPlaceholder": "напр.: password:",
```

**`login.sendPlaceholder`** (추가)

```json
  "login.sendPlaceholder": "예: sudo su -",
  "login.sendPlaceholder": "e.g. sudo su -",
  "login.sendPlaceholder": "例: sudo su -",
  "login.sendPlaceholder": "例如：sudo su -",
  "login.sendPlaceholder": "例如：sudo su -",
  "login.sendPlaceholder": "z. B. sudo su -",
  "login.sendPlaceholder": "ex. : sudo su -",
  "login.sendPlaceholder": "p. ej.: sudo su -",
  "login.sendPlaceholder": "ex.: sudo su -",
  "login.sendPlaceholder": "напр.: sudo su -",
```

**`login.up`** (추가)

```json
  "login.up": "위로",
  "login.up": "Move up",
  "login.up": "上へ",
  "login.up": "上移",
  "login.up": "上移",
  "login.up": "Nach oben",
  "login.up": "Monter",
  "login.up": "Subir",
  "login.up": "Mover para cima",
  "login.up": "Вверх",
```

**`login.down`** (추가)

```json
  "login.down": "아래로",
  "login.down": "Move down",
  "login.down": "下へ",
  "login.down": "下移",
  "login.down": "下移",
  "login.down": "Nach unten",
  "login.down": "Descendre",
  "login.down": "Bajar",
  "login.down": "Mover para baixo",
  "login.down": "Вниз",
```

**`login.add`** (추가)

```json
  "login.add": "+ 단계 추가",
  "login.add": "+ Add step",
  "login.add": "+ ステップを追加",
  "login.add": "+ 添加步骤",
  "login.add": "+ 新增步驟",
  "login.add": "+ Schritt hinzufügen",
  "login.add": "+ Ajouter une étape",
  "login.add": "+ Añadir paso",
  "login.add": "+ Adicionar passo",
  "login.add": "+ Добавить шаг",
```

**`be.login.noPassword`** (추가)

```json
  "be.login.noPassword": "로그인 스크립트: 저장된 비밀번호가 없어 멈췄습니다 ({{password}} 는 이 서버에 비밀번호로 로그인했을 때만 채워집니다)",
  "be.login.noPassword": "Login script: stopped because there is no saved password ({{password}} is filled only when you logged in to this server with a password)",
  "be.login.noPassword": "ログインスクリプト: 保存されたパスワードがないため停止しました ({{password}} はこのサーバーにパスワードでログインした場合のみ入ります)",
  "be.login.noPassword": "登录脚本：没有已保存的密码，已停止（仅在用密码登录此服务器时才会填入 {{password}}）",
  "be.login.noPassword": "登入指令碼：沒有已儲存的密碼，已停止（僅在以密碼登入此伺服器時才會填入 {{password}}）",
  "be.login.noPassword": "Anmeldeskript: angehalten, weil kein gespeichertes Passwort vorhanden ist ({{password}} wird nur gefüllt, wenn Sie sich an diesem Server mit einem Passwort angemeldet haben)",
  "be.login.noPassword": "Script de connexion : arrêté car aucun mot de passe n'est enregistré ({{password}} n'est renseigné que si vous vous êtes connecté à ce serveur avec un mot de passe)",
  "be.login.noPassword": "Script de inicio de sesión: detenido porque no hay contraseña guardada ({{password}} solo se rellena si inició sesión en este servidor con contraseña)",
  "be.login.noPassword": "Script de login: interrompido porque não há senha salva ({{password}} só é preenchido quando você entrou neste servidor com senha)",
  "be.login.noPassword": "Сценарий входа: остановлен, так как нет сохранённого пароля ({{password}} подставляется, только если вход на этот сервер выполнен по паролю)",
```

**`be.login.timeout`** (추가)

```json
  "be.login.timeout": "로그인 스크립트: \"{text}\" 을(를) {sec}초 동안 기다렸지만 나오지 않아 멈췄습니다",
  "be.login.timeout": "Login script: stopped after waiting {sec}s for \"{text}\"",
  "be.login.timeout": "ログインスクリプト: 「{text}」を {sec} 秒待ちましたが出力されなかったため停止しました",
  "be.login.timeout": "登录脚本：等待“{text}”{sec} 秒仍未出现，已停止",
  "be.login.timeout": "登入指令碼：等待「{text}」{sec} 秒仍未出現，已停止",
  "be.login.timeout": "Anmeldeskript: nach {sec} s Warten auf „{text}“ angehalten",
  "be.login.timeout": "Script de connexion : arrêté après {sec} s d'attente de « {text} »",
  "be.login.timeout": "Script de inicio de sesión: detenido tras esperar {sec} s a «{text}»",
  "be.login.timeout": "Script de login: interrompido após esperar {sec} s por \"{text}\"",
  "be.login.timeout": "Сценарий входа: остановлен после {sec} с ожидания «{text}»",
```

**`be.login.cancelled`** (추가)

```json
  "be.login.cancelled": "입력이 있어 로그인 스크립트를 멈췄습니다",
  "be.login.cancelled": "Login script stopped because you typed",
  "be.login.cancelled": "入力があったため、ログインスクリプトを停止しました",
  "be.login.cancelled": "检测到输入，登录脚本已停止",
  "be.login.cancelled": "偵測到輸入，登入指令碼已停止",
  "be.login.cancelled": "Anmeldeskript wegen Ihrer Eingabe angehalten",
  "be.login.cancelled": "Script de connexion arrêté car vous avez saisi du texte",
  "be.login.cancelled": "Script de inicio de sesión detenido porque escribió",
  "be.login.cancelled": "Script de login interrompido porque você digitou",
  "be.login.cancelled": "Сценарий входа остановлен из-за вашего ввода",
```

## F. 명세

원본 앱에 없던 기능이라 명세를 이 작업에서 새로 썼다(아래 조각의 `+` 줄이 새 명세다). 줄이 길어 가로로 넘치면 파일을 직접 여는 편이 읽기 쉽다.

### `spec/acceptance/04-hosts-sftp-settings-sync.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/acceptance/04-hosts-sftp-settings-sync.md
+++ b/spec/acceptance/04-hosts-sftp-settings-sync.md
@@ -16,6 +16,9 @@
 | H9 | 키 파일이 PuTTY PPK(형식 2 또는 3, 암호 유무 무관) | 불러와 저장하고 연결 | 변환 안내 없이 불러와지고, 그 키로 공개 키 인증을 한다. 암호가 걸린 키는 "키 암호 입력"을 묻는다(틀리면 최대 3번) (D106) |
 | H10 | 폴더 편집, 이름 비움 | | [저장] 비활성 |
 | H11 | [저장하고 연결] | | 저장 후 대화상자가 닫히고 새 탭 연결 |
+| H12 | 자동화 탭에서 단계 2개 추가, 첫 단계만 채우고 [저장] | | 서버의 `loginScript` 에 채운 단계 하나만 남는다(빈 단계는 빠짐). 로컬 터미널에는 자동화 탭이 없다 (D139) |
+| H13 | 로그인 스크립트 `$ ` → `sudo su -`, `password` → `{{password}}` 인 SSH 서버(비밀번호 로그인)에 연결 | 프롬프트, 이어서 `[sudo] password for …:` 출력 | `sudo su -`+Enter, 이어서 로그인 비밀번호+Enter 가 입력된다. 그 뒤에 "연결 후 실행할 명령"이 실행된다 (D139) |
+| H14 | H13 에서 키로 로그인한 서버 | `password` 프롬프트 출력 | 아무것도 보내지 않고 "로그인 스크립트: 저장된 비밀번호가 없어 멈췄습니다 …" 안내 (D139) |
 
 ## SFTP·스니펫 (ui/sftp-snippets.md, protocols.md §3)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/behavior/session.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 1, 3번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/behavior/session.md
+++ b/spec/behavior/session.md
@@ -81,6 +82,21 @@
 1. 저장된 서버면 서버의 `lastUsedAt` 갱신.
 2. 설정 "터미널 출력 기록"이 켜져 있으면 로그 파일을 연다(§6).
 3. **세션 유지(tmux)가 적용되면**(§7) 500ms 뒤 tmux 접속 명령을 셸에 입력한다. 아니면 서버의 **자동 실행 명령**이 있으면 줄바꿈을 CR 로 바꾸고 끝에 CR 을 붙여 400ms 뒤에 입력한다.
+4. 서버에 **로그인 스크립트**(§5.1)가 있으면 3번의 입력은 **스크립트가 마지막 단계까지 끝난 뒤에**(그때부터 500ms·400ms 뒤) 보낸다. 스크립트가 멈추면(시한·비밀번호 없음·사용자 입력) 3번의 입력은 보내지 않는다.
+
+## 5.1 로그인 스크립트 (2026-11-27, D139)
+
+접속 직후 서버 출력을 보고 정해 둔 글자를 차례로 입력한다(`sudo su -` 뒤 비밀번호, 장비의 `enable`, 메뉴 선택 등). 원본 앱에 없던 기능이다.
+
+- **데이터**: 서버의 `loginScript`(data/store.md §3) — 단계 배열. 단계는 `expect`(기다릴 글자)와 `send`(보낼 글자). 둘 다 빈 단계는 버리고, **20단계**를 넘는 것도 버린다. 로컬 터미널에는 쓰지 않는다(SSH·텔넷·시리얼).
+- **실행**: 접속에 성공하면(재접속 포함) 첫 단계부터 한 번씩 순서대로 한다. 서버 출력(ANSI 제어 시퀀스를 지운 글)에 지금 단계의 `expect` 가 나오면 `send` 뒤에 CR 을 붙여 보내고 다음 단계로 간다. **대소문자는 구분하지 않는다.** `expect` 가 비면 기다리지 않고 바로 보낸다(`send` 도 비면 Enter 만). 접속 직후 이미 와 있는 출력(첫 프롬프트)도 본다.
+- **한 번 맞은 글자는 다음 단계에 다시 쓰지 않는다**: 단계가 넘어가면 그 글자까지의 출력을 버리고 그 뒤의 출력에서 찾는다(같은 `$ ` 를 두 번 기다리면 프롬프트가 두 번 나와야 한다). 찾는 범위는 최근 출력 4096바이트.
+- **`{{password}}`**: `send` 안의 이 글자는 **이 서버에 로그인할 때 쓴 비밀번호**(볼트에서 읽은 것 또는 사용자가 입력해 인증에 성공한 것, protocols.md §2.2 의 세션 사본과 같은 수명)로 바꿔 보낸다. SSH 에서 비밀번호로 로그인한 경우에만 값이 있다. 값이 없으면 **빈 비밀번호를 보내지 않고** 스크립트를 멈추며 터미널에 `● 로그인 스크립트: 저장된 비밀번호가 없어 멈췄습니다 ({{password}} 는 이 서버에 비밀번호로 로그인했을 때만 채워집니다)`(노랑).
+- **시한**: 한 단계가 **20초** 안에 넘어가지 못하면 멈추고 `● 로그인 스크립트: "{기다리던 글자}" 을(를) 20초 동안 기다렸지만 나오지 않아 멈췄습니다`(노랑).
+- **사용자 입력**: 스크립트가 도는 동안 사용자가 그 세션에 입력하면 멈추고 `● 입력이 있어 로그인 스크립트를 멈췄습니다`(회색). 프롬프트가 예상과 달라 손으로 넘기는 동안 스크립트가 뒤늦게 글자를 보내는 것을 막는다. ESC 로 시작하는 입력은 세지 않는다(터미널이 서버의 질의에 스스로 답하는 응답이 그 모양이다).
+- 스크립트가 보내는 글자는 화면에 따로 표시하지 않는다(서버가 에코하면 보인다). 세션 로그에는 서버 출력만 남는다.
+- 세션을 닫거나 연결이 끊기면 스크립트도 끝난다. 다시 연결되면 처음부터 다시 한다.
+- **보안**: `send` 는 store.json 에 평문으로 남고 git 동기화·백업에도 들어간다. 화면 도움말에 비밀번호를 직접 적지 말고 `{{password}}` 를 쓰라고 적는다(ui/host-editor.md §1.4a).
 
 ## 6. 세션 로그
 - 설정 "터미널 출력 기록"(`sessionLog`)이 켜져 있을 때 **세션을 열 때 한 번** 로그 폴더(`<데이터 폴더>\logs`)에 `{표시 이름}-{YYYYMMDD-HHmmss}.log` 를 **새로 만든다**(권한 소유자 전용, 열린 뒤에 설정을 켠 경우는 아래). 같은 이름이 이미 있으면 `.log` 앞에 `-2`, `-3`, … 을 붙여 없는 이름으로 만든다(세션 하나에 파일 하나). 1000번까지 시도해도 못 만들면 로그 없이 연다.
```

### `spec/data/store.md` — 수정

오늘의 다른 문서도 이 파일을 고친다(문서 2, 3, 4, 5번). 아래는 **이 문서의 몫이 든 조각만** 추린 것이다.

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -52,6 +52,7 @@
 | encoding | `utf-8` `euc-kr` `cp949` `shift_jis` `euc-jp` `gbk` `big5` `latin1` (`euc-jp`·`gbk`·`big5` 는 2026-11-23 D118) | 새 서버 기본 `utf-8` |
 | termType | 문자열 | 새 서버 기본 `xterm-256color` |
 | startupCommand | 문자열(선택) | 접속 직후 보낼 명령 |
+| loginScript | 배열(선택) | 로그인 스크립트(2026-11-27, D139, behavior/session.md §5.1). 항목: id, expect(기다릴 글자, 비면 바로 보냄), send(보낼 글자. `{{password}}` 는 로그인 비밀번호로 바뀐다). 최대 20단계. 단계가 없으면 필드를 두지 않는다. 로컬 터미널에는 없다. **평문으로 저장된다** |
 | keepSession | 불리언(선택) | 세션 유지(tmux) |
 | tmuxName | 문자열(선택) | 비우면 서버 별칭(로컬은 폴더 이름) |
 | keepaliveSec | 숫자 | 새 서버 기본 30 |
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -6,7 +6,7 @@
 
 - 제목: 새 서버면 "서버 추가", 편집이면 "서버 편집 · {표시 이름}".
 - 바닥(왼쪽에 오류 메시지가 있으면 빨간 글씨로, 오른쪽에 버튼): [취소] [저장] [저장하고 연결](주 버튼).
-- 본문 위에 **탭 막대**: 기본 · 인증 · 고급 · 포트 포워딩. **프로토콜이 SSH 가 아니면 "로그인"과 "포트 포워딩" 탭은 숨긴다.** 본문 최소 높이 380px.
+- 본문 위에 **탭 막대**: 기본 · 인증 · 고급 · 포트 포워딩 · **자동화**(2026-11-27, D139). **프로토콜이 SSH 가 아니면 "로그인"과 "포트 포워딩" 탭은 숨긴다. 로컬이면 "자동화" 탭도 숨긴다.** 본문 최소 높이 380px.
 - 열릴 때: 새 서버는 기본값으로 초기화(데이터 명세 §3; 폴더는 호출한 곳이 지정한 폴더 또는 최상위), 편집은 원본을 복사해 편집한다. 편집 중 변경은 [저장]을 눌러야 반영된다.
 
 ### 1.1 "일반" 탭 (위→아래)
@@ -57,6 +57,16 @@
 - 표: 머리 종류 · 바인드 주소 · 포트 · 대상 호스트 · 대상 포트 · (삭제). 행마다 종류 드롭다운(L/R/D), 바인드 주소(고정폭), 포트(숫자), 대상 호스트(고정폭), 대상 포트(숫자), [삭제](휴지통 아이콘). 종류가 D 이면 대상 호스트·대상 포트 칸 대신 흐린 글자 "SOCKS 프록시 (대상 없음)" 을 두 칸 너비로 보인다(적어 둔 대상 값은 지우지 않아 종류를 되돌리면 다시 보인다).
 - [포워딩 추가] 버튼: 새 행 기본값 종류 L, 바인드 주소 `127.0.0.1`, 포트 8080, 대상 호스트 `localhost`, 대상 포트 80.
 
+### 1.4a "자동화" 탭 (SSH·텔넷·시리얼, 2026-11-27, D139)
+
+로그인 스크립트(behavior/session.md §5.1)를 편집한다. 고급 탭이 이미 길어 따로 탭을 두었다.
+
+- 안내 박스(정보): "로그인 스크립트: 연결 직후, 서버 출력에 "기다릴 글자"가 나오면 "보낼 글자"와 Enter를 입력합니다. 위에서 아래로 한 번씩 실행하고, 다시 연결할 때마다 처음부터 합니다. 기다릴 글자를 비우면 바로 보내며, 대소문자는 구분하지 않습니다. 20초 안에 나오지 않거나 직접 입력하면 멈춥니다. 고급 탭의 "연결 후 실행할 명령"은 스크립트가 끝난 뒤에 실행됩니다."
+- 안내 박스(경고): "보낼 글자는 암호화되지 않은 채 서버 목록에 저장되고 git 동기화·백업에도 들어갑니다. 비밀번호를 직접 적지 마세요." SSH 면 이어서 "대신 {{password}} 을 적으면 이 서버에 로그인할 때 쓴 비밀번호로 바뀝니다 (비밀번호로 로그인한 경우만)." 텔넷·시리얼에는 이 둘째 문장이 없다.
+- 표(단계가 하나라도 있을 때): 머리 `#` · 기다릴 글자 · 보낼 글자 · (버튼). 행마다 번호, "기다릴 글자"(고정폭, 플레이스홀더 "예: password:"), "보낼 글자"(고정폭, 플레이스홀더 "예: sudo su -"), [위로](첫 행은 비활성) [아래로](마지막 행은 비활성) [삭제](휴지통).
+- [+ 단계 추가]: 빈 단계를 끝에 붙인다. **20단계**면 비활성.
+- 저장: 기다릴 글자와 보낼 글자가 **모두 빈 단계는 빼고** `loginScript` 에 쓴다. 남는 단계가 없으면 필드를 쓰지 않는다. 프로토콜이 로컬이면 필드를 쓰지 않는다.
+
 ### 1.5 저장 동작 (검사·정규화·볼트 저장)
 
 1. 오류 메시지를 지운다.
```

## G. 검증 (2026-11-27 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [loginscript_test.go](../../internal/session/loginscript_test.go) `TestLoginRunnerSteps` | 프롬프트 전에는 보내지 않는다 / 단계 순서 / **조각이 나뉘어 와도**(`Pass` + `word`) 찾는다 / 대소문자 무시 / 끝난 뒤에는 보내지 않는다 |
| 같은 파일 `TestLoginRunnerDoesNotReuseMatch` | 같은 `$ ` 를 두 번 기다리면 프롬프트가 두 번 나와야 한다 |
| 같은 파일 `TestLoginRunnerIgnoresAnsi` | 글자 사이에 색 시퀀스가 끼고 그 시퀀스가 조각 경계에서 잘려도 찾는다 |
| 같은 파일 `TestLoginRunnerEmptyExpect` | 기다릴 글자가 빈 단계는 출력 없이 바로 |
| 같은 파일 `TestTrimLoginSteps` | 둘 다 빈 단계는 빼고 20단계까지만 |
| 같은 파일 `TestSessionRunsLoginScript` | 세션이 출력에서 찾아 보낸다 / **`{{password}}` 가 로그인 비밀번호로 바뀐다** / 스크립트가 끝난 뒤에 접속 후 입력이 간다 |
| 같은 파일 `TestSessionLoginScriptStopsWithoutPassword` | 비밀번호가 없으면 아무것도 보내지 않고 안내만 낸다(접속 후 입력도 안 감) |
| 같은 파일 `TestSessionLoginScriptTimeout` | 시한이 지나면 멈추고 안내, 그 뒤로 보내지 않는다 |
| 같은 파일 `TestSessionLoginScriptCancelledByInput` | 터미널의 자동 응답(`ESC[?1;2c`)에는 멈추지 않고, 사용자가 입력하면 멈춘다. 멈춘 뒤 프롬프트가 나와도 보내지 않는다 |
| [macro.test.ts](../../frontend/src/lib/macro.test.ts) "단계 다듬기" 중 2개 | 저장할 때 빈 단계를 빼고 하나도 없으면 필드를 두지 않는다 / 단계 옮기기 |
| [Automation.test.tsx](../../frontend/src/screens/Automation.test.tsx) "자동화 탭" 3개 | 단계를 넣고 저장하면 `loginScript` 에 채운 단계만 남는다 / 로컬 터미널에는 탭이 없다 / 텔넷에는 탭이 있고 `{{password}}` 안내는 없다 |
| 기존 `internal/session/*_test.go`·`HostEditor.test.tsx` | 스크립트가 없는 서버의 접속 후 입력·재접속·서버 저장이 그대로 |

**자동 테스트가 확인하지 못하는 것**: 실제 서버의 프롬프트 모양(색·제목 시퀀스·줄바꿈)에서 글자가 잡히는지, `sudo` 가 비밀번호를 받는 타이밍, tmux 접속 줄과의 순서.

전체 실행 결과(2026-11-27, 다섯 문서와 추가 지시 D142·D143 을 모두 적용한 상태):

```bash
go vet ./...                              # 경고 없음
go test ./...                             # 19개 패키지 모두 ok
(cd frontend && npx tsc --noEmit)         # 오류 없음
(cd frontend && npm test)                 # PowerShell 에서. 61개 파일 439개 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

**`-race` 검사 (같은 날 추가)**: 처음에는 이 PC 에 gcc 가 없어 돌리지 못했다(`-race requires cgo`). 같은 날 gcc 를 설치하고(D144, [2026-11-27-6-gcc-race-test.md](./2026-11-27-6-gcc-race-test.md)) `go test -race ./...` 를 돌려 19개 패키지 모두 통과했다 — 경쟁 보고 없음. 읽기 루프·시한 타이머·사용자 입력이 겹치는 `TestSessionLoginScriptTimeout`·`TestSessionLoginScriptCancelledByInput` 도 캐시 없이 따로 돌려 통과(그 문서 §6.2).

```
go test ./internal/session -run Login
(frontend) npm test
```

검사하는 것: 단계 순서, 조각이 나뉘어 올 때, 색 시퀀스가 낀 프롬프트, 같은 글자를 두 번 기다릴 때, 빈 기다릴 글자, 20단계 한도, `{{password}}` 치환과 그 뒤의 접속 후 입력, 비밀번호가 없을 때 멈춤, 시한, 사용자 입력으로 취소(자동 응답은 취소하지 않음).

## H. 수동 검증 (실제 서버에서)

| # | 할 일 | 기대 |
|---|---|---|
| 1 | 비밀번호로 로그인하는 SSH 서버에 `$ ` → `sudo su -`, `password` → `{{password}}` 를 넣고 연결 | root 프롬프트까지 자동으로 간다. 비밀번호는 화면에 보이지 않는다 |
| 2 | 1번에 "연결 후 실행할 명령" `whoami` 를 더함 | 스크립트가 끝난 뒤 `whoami` 가 실행돼 `root` |
| 3 | 1번 서버의 연결을 끊고 Enter 로 다시 연결 | 처음부터 다시 실행된다 |
| 4 | 키로 로그인하는 서버에 같은 스크립트 | 비밀번호 단계에서 멈추고 노란 안내, 빈 비밀번호를 보내지 않는다 |
| 5 | 나오지 않는 글자를 기다리게 함 | 20초 뒤 노란 안내. 그동안 터미널은 평소처럼 쓸 수 있다 |
| 6 | 스크립트가 기다리는 동안 직접 입력 | "입력이 있어 로그인 스크립트를 멈췄습니다", 그 뒤로 아무것도 보내지 않는다 |
| 7 | 텔넷 장비: `login:` → 사용자 이름, `>` → `enable` | 순서대로 입력된다 |
| 8 | 세션 유지(tmux)를 켠 서버에 스크립트 | 스크립트가 끝난 뒤 tmux 에 붙는다 |
| 9 | 서버 편집 > 자동화에서 단계 순서를 ↑↓ 로 바꾸고 저장 후 다시 열기 | 순서가 그대로 저장돼 있다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 스크립트가 없는 서버(기본) | 읽기 루프가 출력마다 `feedLogin` 을 부른다 | 스크립트가 없으면 잠금 한 번 잡고 돌아온다. 접속 후 입력은 전과 같은 때(접속 직후)에 예약된다 — `startLogin` 이 곧바로 `scheduleStartup` 을 부른다 |
| 접속 후 입력의 순서 | `startLogin` 을 읽기 루프보다 먼저 부르도록 순서를 바꿨다 | 스크립트가 없을 때의 동작은 같다(예약 시각만 수 마이크로초 앞당겨짐). 기존 세션 테스트 통과 |
| 비밀번호가 화면·로그에 남는가 | 스크립트가 보내는 글자는 터미널에 따로 쓰지 않는다 | 서버가 에코하지 않는 비밀번호 프롬프트면 보이지 않는다. 세션 로그는 서버 출력만 담는다 |
| 평문 저장 | "보낼 글자"가 store.json·git 동기화·백업에 평문으로 들어간다 | 탭에 경고 박스. 비밀번호는 `{{password}}` 로 쓰게 안내 |
| 잘못된 스크립트가 계정을 잠그는 경우 | 엉뚱한 프롬프트에 비밀번호를 반복해 넣을 수 있다 | 단계는 한 번씩만 실행하고, 한 번 맞은 글자를 다시 쓰지 않는다. 비밀번호가 없으면 빈 값을 보내지 않고 멈춘다 |
| 사용자가 손으로 넘기는 경우 | 스크립트가 뒤늦게 글자를 보낼 수 있다 | 입력이 오면 멈춘다(ESC 로 시작하는 자동 응답 제외) |
| 자동 재접속 | 다시 연결될 때마다 처음부터 실행된다 | 의도한 동작(새 셸). 시한·취소 규칙은 같다 |
| 로컬 터미널 | 탭을 숨기고 저장할 때 필드를 지운다. 백엔드도 로컬이면 스크립트를 쓰지 않는다 | `loginScriptOf` |
| 서버 편집의 탭 수 | SSH 는 탭이 5개가 된다(일반·로그인·고급·포트 포워딩·자동화) | 기본 크기 창(560px)에 한 줄로 들어간다. 긴 언어(독일어 "Automatisierung")는 수동 검증에서 확인 |
| 가져오기 | PuTTY·SecureCRT·Xshell 의 로그인 스크립트는 옮기지 않는다 | 후속 과제 |

## J. 체크리스트

- [x] `internal/session/loginscript.go`: 단계 진행(`loginRunner`), 시한, 사용자 입력으로 취소, `{{password}}`
- [x] 세션 연결: 접속 성공 때 시작, 출력마다 먹이기, 끝나면 접속 후 입력 예약, 닫을 때 정리
- [x] 앱: 서버 기록의 `loginScript` 읽기, SSH 연결의 로그인 비밀번호 제공(`LoginSecret`)
- [x] 화면: 서버 편집 "자동화" 탭(단계 표, ↑↓, 경고), 저장할 때 빈 단계 빼기
- [x] 문구 16개(10개 언어)
- [x] 명세 behavior/session.md §5·§5.1 · data/store.md §3 · ui/host-editor.md §1·§1.4a · 수용 H12~H14, decisions D139
- [x] 자동 테스트 14개 추가(Go 9, 프런트 5), 전체 통과
- [ ] 수동 검증 1~9 (실제 서버)
- [x] `go test -race` — 2026-11-27 gcc 설치 뒤 통과 (D144, [-6-](./2026-11-27-6-gcc-race-test.md))
