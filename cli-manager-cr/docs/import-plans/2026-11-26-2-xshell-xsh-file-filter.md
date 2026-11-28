# Xshell 파일 선택 창에서 세션 파일(`.xsh`)이 보이지 않는 문제 — 원인·넣는 방법·검증

**작성**: 2026-11-26 · **상태**: 넣을 곳은 `C:\work\win-projects-main-1125`(릴리스 1.1.0 기준 폴더). 그 폴더에는 **아직 넣지 않았다.** `C:\work\win-projects-main` 에는 들어 있다 — 2026-11-26 에 `-1125` 를 이 폴더로 동기화하면서 이 두 줄은 이 폴더의 코드를 남겼다(사용자 지시) · **대상**: cli-manager-cr(DariTerm)
**관련 문서**: [2026-11-25-6-xshell-port-forwarding.md](./2026-11-25-6-xshell-port-forwarding.md) (D129 — 세션 파일 `.xsh` 하나를 고를 수 있게 한 문서)

---

## 0. 한 줄 요약

`-1125` 폴더의 [Xshell 내보내기 파일 선택] 창은 파일 종류가 `*.xts` 뿐이라 세션 파일(`.xsh`)이 목록에 보이지 않는다. 파일 종류 한 줄(과 그 위 주석 한 줄)을 바꾸면 된다. 읽는 쪽 코드·명세·문구는 이미 `.xsh` 를 다루므로 **바꿀 곳은 한 파일의 두 줄**이다.

## 1. 증상

- 서버 가져오기 > [Xshell 내보내기 파일 선택] 을 누르면 뜨는 파일 선택 창의 종류가 "Xshell 내보내기 파일 (*.xts)" 과 "모든 파일" 둘이다.
- 세션 파일(`이름.xsh`)이 있는 폴더로 가도 기본 종류에서는 **파일이 하나도 보이지 않는다.** 종류를 "모든 파일"로 바꿔야 고를 수 있다.
- 고르기만 하면 가져오기 자체는 된다(읽는 쪽은 `.xsh` 를 안다).

## 2. 원인

두 폴더를 비교해 찾았다(2026-11-26). `-1125` 폴더는 D129 의 나머지(읽는 코드, 명세, 테스트)는 다 들어 있고 **파일 선택 창의 종류 한 줄만 빠졌다.**

| 부분 | `-1125` 폴더 | 판정 |
|---|---|---|
| 세션 파일 읽기 `internal/importer/xshell.go` | `.xsh` 를 읽는다 (두 폴더의 `internal/importer` 는 내용이 같다) | 맞음 |
| 명세 `spec/ipc.md` 의 `import.scanXshell` | "종류 `*.xts;*.xsh` 와 모든 파일" 이라고 적혀 있다 | 맞음 |
| **파일 선택 창의 종류** `internal/app/channels_import.go` 74행 | `(*.xts)\|*.xts\|*.*\|*.*` | **명세와 다름** |

즉 코드가 명세를 따라가지 못한 것이다. 명세·문구·읽는 쪽은 손댈 것이 없다.

## 3. 넣는 방법

### 3.1 바꿀 파일

`internal/app/channels_import.go` 하나. `import.scanXshell` 채널 안의 두 줄.

```diff
 	pRouter.Handle("import.scanXshell", func(slArgs []json.RawMessage) (any, error) {
-		// 경로(내보내기 파일 .xts 또는 세션 폴더)를 안 주면 내보내기 파일을 고르게 한다. 취소하면 null (spec/ipc.md).
+		// 경로(내보내기 파일 .xts, 세션 파일 .xsh, 세션 폴더)를 안 주면 파일을 고르게 한다. 취소하면 null (spec/ipc.md).
 		szPath := ""
 		if len(slArgs) > 0 {
 			szPath, _ = argString(slArgs, 0)
 		}
 		if szPath == "" {
 			szHome, _ := os.UserHomeDir()
-			szFilter := i18n.T("be.dlg.xshellFilter") + " (*.xts)|*.xts|*.*|*.*"
+			szFilter := i18n.T("be.dlg.xshellFilter") + " (*.xts, *.xsh)|*.xts;*.xsh|*.*|*.*"
 			szPicked, errPick := dialog.OpenFileOf(i18n.T("be.dlg.pickXshell"), szHome, szFilter)
```

### 3.2 종류 문자열의 형식

`dialog.OpenFileOf` 는 이 문자열을 Windows 파일 선택 창의 종류로 그대로 넘긴다. 형식은 `보이는 이름|패턴|보이는 이름|패턴`.

| 조각 | 뜻 |
|---|---|
| `Xshell 내보내기 파일 (*.xts, *.xsh)` | 창에 보이는 이름. 괄호 안은 사람이 읽는 표시라 쉼표로 적는다 |
| `*.xts;*.xsh` | 실제로 거르는 패턴. **여러 개는 세미콜론**으로 잇는다(쉼표를 쓰면 걸러지지 않는다) |
| `*.*\|*.*` | 두 번째 종류 "모든 파일" |

### 3.3 바꾸지 않는 것

- 문구 `be.dlg.xshellFilter`("Xshell 내보내기 파일")·`be.dlg.pickXshell` — 10개 언어 그대로.
- 명세 `spec/ipc.md` — 이미 `*.xts;*.xsh` 로 적혀 있다.
- 프런트엔드 — 채널을 부르기만 하고 종류를 모른다.
- 결정 기록 — 새 결정이 아니라 D129 의 빠진 줄이다. 번호를 새로 받지 않는다.

## 4. 영향 범위와 위험

- 영향: [Xshell 내보내기 파일 선택] 창의 기본 종류에 `.xsh` 파일이 함께 보인다. 그 밖의 동작은 같다.
- 위험: 없음에 가깝다. 문자열 하나이고, `.xsh` 를 고른 뒤의 경로(읽기·후보 표시)는 이미 테스트가 있다(`internal/importer/xshell_test.go`).
- 다른 OS: 파일 선택 창은 Windows 에서만 동작한다(`dialog_other.go` 는 "지원하지 않음"을 돌려준다). 바뀌는 것이 없다.

## 5. 검증

### 5.1 자동

```
go build ./...
go test ./internal/app/ ./internal/importer/
```

종류 문자열 자체를 보는 자동 테스트는 없다(파일 선택 창은 Windows 의 창이라 테스트에서 띄우지 않는다).

### 5.2 수동

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 서버 가져오기 > [Xshell 내보내기 파일 선택] | 파일 종류가 "Xshell 내보내기 파일 (*.xts, *.xsh)" 으로 보인다 | |
| 2 | `.xsh` 파일이 있는 폴더로 간다 | 종류를 바꾸지 않아도 `.xsh` 파일이 보인다 | |
| 3 | `.xsh` 하나를 고른다 | 그 세션 하나가 후보로 나온다(포워딩이 있으면 함께) | |
| 4 | `.xts` 파일을 고른다 | 전과 같이 세션 전체가 후보로 나온다 | |
| 5 | 종류를 "모든 파일"로 바꾼다 | 모든 파일이 보인다 | |

## 6. 넣는 순서 (체크리스트)

- [ ] `-1125` 폴더의 `internal/app/channels_import.go` 에 §3.1 의 두 줄을 바꾼다
- [ ] `go build ./...`, `go test ./internal/app/ ./internal/importer/`
- [ ] 수동 검증 1~5 (§5.2)
- [ ] `CHANGELOG.md` 의 다음 버전 "수정"에 한 줄: "Xshell 파일 선택 창에서 세션 파일(`.xsh`)이 보이지 않던 문제 (D129)"
- [ ] 이 문서를 `-1125` 폴더의 `docs/import-plans/` 에도 둔다(상태 줄을 "넣음"으로 고쳐서)
