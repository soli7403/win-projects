# 로그인 스크립트의 보관함 값 — 로그인과 다른 비밀번호를 평문으로 두지 않기

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D148). 실제 서버에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-3-login-script.md](./2026-11-27-3-login-script.md)(D139, "로그인과 다른 비밀번호를 금고에 두는 길"을 후속 과제로 남긴 곳)

---

## 0. 한 줄 요약

로그인 스크립트에서 `su -`·`enable` 처럼 **로그인과 다른 비밀번호**를 보내려면 지금까지는 보낼 글자에 평문으로 적어야 했다(store.json·git 동기화·백업에 그대로 남는다). 단계마다 **보관함 항목(비밀번호)** 을 고르면 `send` 대신 그 항목의 비밀번호를 보낸다. 파일에는 항목 id 만 남는다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 저장 형식 | 단계에 `secretId`(볼트 항목 id). 고르면 `send` 는 빈 글자로 저장 | 서버의 `credentialId` 와 같은 방식. 이름을 바꿔도 깨지지 않는다 |
| 고를 수 있는 항목 | 볼트의 **비밀번호 종류**만 | 키 항목은 보낼 글자가 아니다 |
| 언제 읽나 | 그 단계를 **보낼 때** | 볼트가 연결 뒤에 잠겼다 풀렸어도 최신 값 |
| 볼트가 잠겼을 때 | 그 연결에서 **한 번 읽은 값은 세션 메모리의 사본**으로 다시 쓴다. 사본이 없으면 보내지 않고 멈춘다 | SSH 계정의 세션 사본(protocols.md §2.2, D94)과 같은 위험 수준·같은 수명. 잠금 화면 뒤에서 볼트를 몰래 열지 않는다 |
| 사본 지우기 | 세션을 닫을 때(다이얼러의 사본 지우기와 함께) | |
| 멈출 때 안내 | 잠김: "로그인 스크립트: 보관함이 잠겨 있어 보관함 비밀번호를 넣지 못해 멈췄습니다. 잠금을 풀고 다시 연결하세요." / 없음·빈 값: "로그인 스크립트: 고른 보관함 항목을 찾을 수 없거나 비밀번호가 비어 있어 멈췄습니다. 서버 편집 > 자동화에서 다시 고르세요." | 빈 비밀번호를 보내 계정이 잠기는 일을 막는다(`{{password}}` 와 같은 규칙) |
| 대안 | `{{secret:이름}}` 자리표시자 — 기각(이름을 바꾸면 깨지고 오타를 잡기 어렵다). 스크립트 전체를 볼트에 — 기각(기다릴 글자까지 숨길 까닭이 없고 동기화 병합이 복잡) | |

## 2. 구현

### 2.1 세션 — `internal/session/loginscript.go`

```go
type LoginStep struct {
	Expect string // 기다릴 글자. 비면 기다리지 않고 바로 보낸다
	Send   string // 보낼 글자. 뒤에 Enter(CR)를 붙여 보낸다
	// SecretId 가 있으면 Send 대신 보관함 항목(이 id)의 비밀번호를 보낸다 (D148).
	SecretId string
}

// ErrSecretLocked 는 보관함이 잠겨 있어 로그인 스크립트의 비밀값을 읽지 못했다는 뜻이다 (SecretResolver 가 돌려준다).
var ErrSecretLocked = errors.New("login script: vault locked")

// SecretResolver 는 보관함 항목 id 의 비밀번호를 돌려준다 (앱이 구현한다). 볼트가 잠겨 있고 세션 사본도 없으면 오류다.
type SecretResolver func(szId string) (string, error)
```

- `TrimLoginSteps`: `Expect`·`Send`·`SecretId` 가 **모두** 빈 단계만 버린다.
- 진행기(`loginRunner`)는 맞은 단계를 문자열이 아니라 **단계 그대로** 돌려주게 했다(`feedSteps`). 시험용 `feed` 는 보낼 글자만 뽑아 전과 같은 모양을 돌려준다(기존 시험을 고치지 않으려고).
- `feedLogin`:

```go
	slSteps, bFinished := pRunner.feedSteps(szText)
	for _, step := range slSteps {
		szSend := step.Send
		if step.SecretId != "" {
			// 보관함 항목의 비밀번호를 보낸다 (D148). 읽지 못하면(잠김·지워짐) 빈 글자를 보내지 않고 멈춘다.
			szSecret := ""
			var errSecret error
			if pSession.cfg.ResolveSecret != nil {
				szSecret, errSecret = pSession.cfg.ResolveSecret(step.SecretId)
			}
			if szSecret == "" {
				pRunner.stop()
				pSession.clearLogin(pRunner)
				szKey := "be.login.secretMissing"
				if errors.Is(errSecret, ErrSecretLocked) {
					szKey = "be.login.secretLocked"
				}
				pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T(szKey)))
				return
			}
			szSend = szSecret
		} else if strings.Contains(szSend, G_cszLoginPassword) {
			// … {{password}} 처리 (전과 같음)
		}
		// … 지금 연결인지 확인하고 szSend + "\r" 을 쓴다 (전과 같음)
	}
```

- `OpenConfig` 에 `ResolveSecret SecretResolver` (`internal/session/types.go`).

### 2.2 앱 — `internal/app/login_secret.go` (새 파일 전체)

```go
// loginSecretResolver 는 세션 하나의 로그인 스크립트가 보관함 비밀값을 읽는 함수를 만든다 (session.md §5.1, D148).
// 볼트가 열려 있으면 항목의 비밀번호를 읽어 세션 메모리에 사본으로 두고, 잠겨 있으면 그 사본을 쓴다
// (SSH 계정의 세션 사본과 같은 규칙, protocols.md §2.2). 사본도 없으면 session.ErrSecretLocked.
// 돌려주는 두 번째 함수는 세션을 닫을 때 불러 사본을 지운다.
func (pService *Service) loginSecretResolver() (session.SecretResolver, func()) {
	var mtxCopy sync.Mutex
	mapCopy := map[string]string{}
	resolve := func(szId string) (string, error) {
		body, errGet := pService.pVault.Get(szId)
		if errGet == nil && body != nil && body.M_szPassword != "" {
			mtxCopy.Lock()
			mapCopy[szId] = body.M_szPassword
			mtxCopy.Unlock()
			return body.M_szPassword, nil
		}
		mtxCopy.Lock()
		szCopy, bHad := mapCopy[szId]
		mtxCopy.Unlock()
		if bHad {
			return szCopy, nil
		}
		if errors.Is(errGet, vault.ErrLocked) && pService.vaultLocked() {
			return "", session.ErrSecretLocked
		}
		return "", errGet // 항목이 없거나 비밀번호가 비어 있다 (호출한 쪽이 "찾을 수 없음"으로 알린다)
	}
	clearCopy := func() {
		mtxCopy.Lock()
		defer mtxCopy.Unlock()
		for szId := range mapCopy {
			delete(mapCopy, szId)
		}
	}
	return resolve, clearCopy
}
```

`internal/app/channels_session.go`:

```go
type loginStepRecord struct {
	M_szExpect   string `json:"expect"`
	M_szSend     string `json:"send"`
	M_szSecretId string `json:"secretId"` // 있으면 send 대신 이 보관함 항목의 비밀번호를 보낸다 (D148)
}
// loginScriptOf: session.LoginStep{Expect, Send, SecretId}

	dialer, onCloseDialer := pService.dialerFor(record)
	resolveSecret, clearSecrets := pService.loginSecretResolver()
	onClose := func() {
		if onCloseDialer != nil {
			onCloseDialer()
		}
		clearSecrets()
	}
	// …
	cfg.ResolveSecret = resolveSecret
```

텔넷·시리얼 세션도 같은 해석기를 받는다(로그인 스크립트는 SSH·텔넷·시리얼 모두에 있다).

### 2.3 화면 — `frontend/src/screens/LoginScriptTable.tsx`

- 표에 열 **"보관함 값"**(드롭다운): "쓰지 않음 (보낼 글자)" + 볼트의 비밀번호 항목(`이름 (사용자)`). 서버 편집이 이미 읽어 둔 `cred.list` 결과를 `secrets` 로 넘긴다.
- 항목을 고르면 "보낼 글자" 칸 대신 흐린 상자 `🔑 보관함: {이름} 의 비밀번호`.
- 고른 항목이 목록에 없으면(볼트 잠김이면 목록이 비어 있다) 볼트가 열려 있을 때 "(지워진 항목)", 잠겨 있으면 "(보관함 잠김)".
- 경고 박스 끝에 `login.secretHint` 문장.

`frontend/src/lib/loginScript.ts` `cleanLoginScript`:

```ts
  const out = (steps ?? [])
    .filter((s) => s.expect !== '' || s.send !== '' || !!s.secretId)
    .slice(0, LOGIN_STEP_MAX)
    .map((s) => {
      if (s.secretId) return { id: s.id, expect: s.expect, send: '', secretId: s.secretId };
      const { secretId: _drop, ...rest } = s;
      return rest;
    });
```

`frontend/src/ipc/types.ts` `LoginStep.secretId?: string`.

### 2.4 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `login.secret` · `login.secretN` | 보관함 값 · {n}단계 보관함 값 |
| `login.secretNone` | 쓰지 않음 (보낼 글자) |
| `login.secretSends` | 보관함: {name} 의 비밀번호 |
| `login.secretGone` · `login.secretLockedName` | (지워진 항목) · (보관함 잠김) |
| `login.secretHint` | su·enable 처럼 로그인과 다른 비밀번호는 [보관함 값]에서 보관함 항목을 고르면 파일에 평문으로 남지 않습니다. |
| `be.login.secretLocked` · `be.login.secretMissing` | §1 의 두 안내 |

## 3. 명세

| 문서 | 바꾼 것 |
|---|---|
| spec/behavior/session.md §5.1 | "보관함 값" 항목(읽는 때, 사본, 멈춤 안내) |
| spec/data/store.md §3 | `loginScript[].secretId` |
| spec/ui/host-editor.md §1.4a | 열·표시·저장 규칙 |

## 4. 테스트

| 파일 | 테스트 | 확인 |
|---|---|---|
| `internal/session/loginsecret_test.go` | TestSessionLoginScriptSendsVaultSecret | 2단계가 `send` 가 아니라 해석기의 값(`root-pw\r`)을 보내고, 해석기는 `cred-root` 로 한 번 불림. 로그인 비밀번호(`{{password}}` 값)와 다르다 |
| 〃 | TestSessionLoginScriptSecretLocked | `ErrSecretLocked` → 아무것도 보내지 않고 잠김 안내, 접속 후 입력도 보내지 않음 |
| 〃 | TestSessionLoginScriptSecretMissing | 해석기가 없거나 오류 → "찾을 수 없음" 안내 |
| 〃 | TestTrimLoginStepsKeepsSecretOnly | 보관함 값만 있는 단계는 남는다 |
| 기존 `loginscript_test.go` 9개 | 그대로 통과(feed 래퍼) | |

`go test -race ./internal/session/` 통과(해석기는 읽기 루프의 고루틴에서 불리고 사본은 잠금 아래).

## 5. 검증 결과 (2026-11-27)

`go test -race ./...`(clipboard KI-013 재실행 통과), tsc, vitest 465개, `wails build` 성공.

## 6. 위험

- 볼트 항목을 지우면 그 단계는 "찾을 수 없음"으로 멈춘다(빈 글자를 보내지 않는다). 서버 편집에서 "(지워진 항목)"으로 보인다.
- 가져오기·동기화: `secretId` 는 볼트 항목 id 라 다른 PC 에서도 같은 볼트(동기화된 볼트)면 그대로 쓴다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 설정 > 보안 > 저장된 계정에 비밀번호 항목 "root" 만들기(또는 다른 서버에서 저장) | 항목이 보인다 | |
| 2 | 서버 편집 > 자동화: `$ ` → `su -`, `Password:` → [보관함 값] root | 2행 보낼 글자 칸이 `🔑 보관함: root 의 비밀번호` | |
| 3 | 저장 뒤 store.json 에서 그 서버의 loginScript | 2단계 `send: ""`, `secretId` 만 | |
| 4 | 연결 | su 가 성공해 `#` 프롬프트 | |
| 5 | 앱을 잠그고 서버 쪽에서 연결을 끊어 재연결이 일어나게 | 잠긴 동안에도 사본으로 su 까지 된다 | |
| 6 | 앱을 다시 켜고(사본 없음) 잠근 채로 연결 시도 | 로그인이 잠금 해제를 기다린다(D94). 풀면 진행 | |
| 7 | 그 보관함 항목을 지우고 연결 | "찾을 수 없음" 안내, 비밀번호를 보내지 않음 | |

## 체크리스트

- [x] 세션 · [x] 앱 해석기·사본 · [x] 화면 · [x] 문구 · [x] 명세 · [x] 테스트 · [ ] §H
