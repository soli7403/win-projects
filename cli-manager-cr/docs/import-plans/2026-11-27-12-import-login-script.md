# 가져오기에서 로그인 스크립트 — 형식 확인 전이라 "알리기"까지

**작성**: 2026-11-27 · **상태**: **일부 완료**(D150) — SecureCRT 의 로그인 스크립트를 **감지해 알리는 것**과 옮길 자리(적용 경로)는 넣었다. **실제로 옮기는 것은 표본을 받은 뒤**(§5) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-3-login-script.md](./2026-11-27-3-login-script.md)(D139 §H 끝 "PuTTY·SecureCRT·Xshell 의 로그인 스크립트는 옮기지 않는다 — 후속 과제"), [2026-11-25-5-xshell-import.md](./2026-11-25-5-xshell-import.md)

---

## 0. 한 줄 요약

Xshell 의 "로그인 스크립트(Expect/Send)"와 SecureCRT 의 "로그온 동작(Logon Actions)"이 세션 파일에 **어떤 키·형식으로** 저장되는지 확인할 수 없었다(공개 문서에 없고, 이 PC 의 SecureCRT 표본은 모두 비어 있다). 형식을 추측해 읽으면 엉뚱한 글자를 서버에 보낼 수 있어, 이번에는 **SecureCRT 의 로그인 스크립트가 켜진 세션에 "옮기지 않음" 안내를 붙이고**, 형식이 확인되면 바로 쓸 수 있게 **후보 → 서버 `loginScript` 적용 경로**를 만들었다.

## 1. 조사

### 1.1 이 PC 의 SecureCRT 표본

`%APPDATA%\VanDyke\Config\Sessions` 의 세션 파일 **184개**에서 로그인 관련 키를 모았다.

```
D:"Use Login Script"=00000000
Z:"Login Script V4"=00000000
```

- 모든 세션이 꺼짐(`0`)·빈 목록(`Z` 형 개수 0)이다. 내용이 든 `Login Script V4` 의 줄 모양은 알 수 없다.
- `Z:` 형은 `Z:"이름"=개수(16진)` 뒤에 그 수만큼 줄이 이어지는 여러 줄 값이다(가져오기가 이미 이 형식을 읽는다 — import.md §1.5).

### 1.2 공개 문서 (웹 검색, 2026-11-27)

- SecureCRT: "Session Options > Connection > Logon Actions" 에서 Expect/Send 짝을 위에서 아래로 쓴다는 설명은 있으나 **세션 파일의 저장 형식은 없다**.
  ([Session Options/Connection/Logon Actions](https://documentation.help/SecureCRT/SO_Connection_Login_Actions.htm), [SecureCRT Log On Options](https://www.vandyke.com/support/howto/logonoptions.html))
- Xshell: "Connection > Login Scripts" 의 Expect and Send 규칙 설명은 있으나 `.xsh` 의 키 이름은 없다.
  ([Xshell 8 manual](https://cdn.netsarang.net/docs/Xshell8_manual.pdf))
- PuTTY: 로그인 스크립트 기능이 없다.

### 1.3 판단

| 안 | 판단 |
|---|---|
| 키 이름·형식을 추측해 읽기 | 기각 — 잘못 읽으면 "보낼 글자"에 엉뚱한 값(옵션 숫자 등)이 들어가 서버에 입력된다. 비밀번호 칸이면 계정 잠금까지 갈 수 있다 |
| **감지해 알리기 + 적용 경로만** | **채택** — 말없이 빠지던 것을 알 수 있게 하고, 형식이 확인되면 읽기 함수 하나만 더하면 된다 |

## 2. 구현

### 2.1 SecureCRT 감지 — `internal/importer/securecrt.go`

```go
	// 로그온 동작(Logon Actions)의 Expect/Send 는 저장 형식을 확인하지 못해 옮기지 않는다 — 켜져 있으면 알린다 (D150).
	// 형식(Z:"Login Script V4" 의 줄)을 표본으로 확인하면 host.M_slLoginScript 로 옮긴다.
	if crtHasLoginScript(values) {
		cand.M_szNote = joinNote(cand.M_szNote, i18n.T("be.imp.loginScriptSkipped"))
	}

// crtHasLoginScript 는 세션의 로그온 동작(Expect/Send)이 켜져 있고 내용이 있는지 본다.
// 이 PC 의 표본에서는 꺼진 세션이 D:"Use Login Script"=00000000 과 빈 Z:"Login Script V4"=00000000 이다.
func crtHasLoginScript(values crtValues) bool {
	iUse, _ := values.num("Use Login Script")
	if iUse == 0 {
		return false
	}
	for _, szKey := range []string{"Login Script V4", "Login Script V3", "Login Script V2"} {
		for _, szLine := range values.lines(szKey) {
			if strings.TrimSpace(szLine) != "" {
				return true
			}
		}
	}
	return false
}

// joinNote 는 후보 메모에 한 줄을 덧붙인다.
func joinNote(szNote string, szAdd string) string {
	if szNote == "" {
		return szAdd
	}
	return szNote + " · " + szAdd
}
```

방화벽(경유 서버) 메모를 붙이는 `ResolveCrtJumps` 가 메모를 **덮어쓰던 것**도 `joinNote` 로 덧붙이게 고쳤다(로그인 스크립트 안내가 지워지지 않게).

### 2.2 적용 경로 — `internal/importer/types.go`, `apply.go`

```go
// HostDraft
	// M_slLoginScript 는 로그인 스크립트다 (data/store.md §3 loginScript, D150). 읽을 수 있는 가져오기만 채운다.
	M_slLoginScript []LoginStep `json:"loginScript,omitempty"`

// LoginStep 은 로그인 스크립트의 한 단계다 (data/store.md §3 loginScript).
type LoginStep struct {
	M_szExpect string `json:"expect"`
	M_szSend   string `json:"send"`
}

// buildHost
	if len(host.M_slLoginScript) > 0 && szProtocol != "local" {
		slSteps := []map[string]any{}
		for _, step := range host.M_slLoginScript {
			szStepId, errStep := uid.New()
			if errStep != nil {
				return nil, "", errStep
			}
			slSteps = append(slSteps, map[string]any{"id": szStepId, "expect": step.M_szExpect, "send": step.M_szSend})
		}
		mapHost["loginScript"] = slSteps
	}
```

후보는 화면을 거쳐 `import.apply` 로 그대로 돌아오므로(`c1types.ts` 의 주석대로) 프런트는 고칠 것이 없다.

### 2.3 문구 (10개 언어)

`be.imp.loginScriptSkipped` — "로그인 스크립트(로그온 동작)는 옮기지 않음 — 서버 편집 > 자동화에서 다시 넣으세요"

## 3. 명세

spec/import.md §1.5 "로그인 스크립트" 항목, §2 끝 "적용" 항목(후보의 `host.loginScript`·`host.proxy`).

## 4. 테스트 — `internal/importer/loginscript_import_test.go`

| 테스트 | 확인 |
|---|---|
| TestCrtLoginScriptDetected | 꺼짐(표본과 같은 값)·켜졌지만 빈 값 → 안내 없음, 켜지고 내용 있음 → 안내. 어떤 경우에도 단계를 만들지 않는다 |
| TestApplyCarriesLoginScript | 후보에 단계가 있으면 서버 `loginScript` 로, 단계마다 서로 다른 새 id |

## 5. 남은 일 — 표본이 있으면 넣는 순서

1. SecureCRT 에서 세션 하나에 Logon Actions 를 두세 줄(예: `login:` → `kai`, `Password:` → (비밀번호 칸은 비워 두거나 가짜 값), `$ ` → `sudo -i`) 넣고 저장 → 그 세션의 `.ini` 를 받는다. Xshell 도 같은 방식으로 `.xsh` 를 받는다.
2. 그 줄 모양으로 `ParseCrtLoginScript(lines) []LoginStep` · Xshell 쪽 읽기를 만들고, 후보의 `host.M_slLoginScript` 를 채운다. 비밀번호로 보이는 칸(Send as asterisks 등)은 옮기지 않고 메모로 알린다.
3. 표본 파일로 시험을 만든다(실제 값은 지운 표본).
4. 감지 안내는 "옮김"으로 바꾼다.

## 6. 검증 결과 (2026-11-27)

`go test -race ./internal/importer/` 통과, 전체 검증은 다른 계획서와 같다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | SecureCRT 에서 Logon Actions 를 켠 세션을 만들고 DariTerm 가져오기 | 그 세션 줄의 출처 툴팁(메모)에 "로그인 스크립트(로그온 동작)는 옮기지 않음…" | |
| 2 | 1 의 세션 `.ini` 를 개발자에게 전달 | §5 진행 | |

## 체크리스트

- [x] 조사 · [x] SecureCRT 감지 · [x] 메모 덮어쓰기 수정 · [x] 적용 경로 · [x] 문구 · [x] 명세 · [x] 테스트 · [ ] **표본으로 실제 읽기(§5)** · [ ] §H
