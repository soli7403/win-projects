# 05. 빌드 도구 (Go · Wails CLI · Node.js)

> 2026-11-25 기록. 결정 D124-1. 이 PC 에 원래 있던 도구를 이 문서에서 버전까지 관리하기 시작했다.

## 무엇을

| 도구 | 버전 | 설치 방법 | 위치 |
|---|---|---|---|
| Go | **1.27.4** (2026-11-25, 1.26.2 에서 올림) | `winget upgrade --id GoLang.Go -e` (관리자 권한, MSI) | `C:\Program Files\Go` |
| Wails CLI | **v2.17.1** (2026-11-25, v2.10.2 에서 올림) | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1` | `%USERPROFILE%\go\bin\wails.exe` |
| Node.js | 24.14.1 (바꾸지 않음) | 원래 설치돼 있음 | — |
| gcc (MinGW-w64) | 16.2.0 (2026-11-27) | `go test -race` 에만 필요 — [06_gcc-race.md](./06_gcc-race.md) | winget 패키지 폴더 |

- Wails CLI 버전은 `go.mod` 의 `github.com/wailsapp/wails/v2` 와 같게 맞춘다(다르면 `wails build` 가 경고한다).
- `go.mod` 의 `go` 지시문도 설치한 Go 와 같게 둔다(지금 `go 1.27.4`). 낮은 Go 로 빌드하면 `GOTOOLCHAIN=auto` 가 1.27.4 를 내려받는다.
- Go 를 올리면 열려 있던 터미널·VS Code 는 새 PATH 를 못 볼 수 있다 — 새로 연다.

## 의존성 올리는 법 (저장소 cli-manager-cr 에서)

```bash
go get -u ./... && go mod tidy
go install github.com/wailsapp/wails/v2/cmd/wails@<go.mod 의 wails 버전>
(cd frontend && npm outdated)   # 바꿀 것을 보고 npm install --save-exact <패키지>@latest
make notices                    # 오픈소스 고지 다시 만들기
go test -race ./... && (cd frontend && npx tsc --noEmit && npm test) && wails build
```

- 프런트 테스트는 `npm test`(= `node run-vitest.mjs run`)로 돌린다. Vitest 5 는 작업 경로가 소문자 드라이브(`e:\`, Git Bash 의 `/e/...`)이면 실패하므로 `run-vitest.mjs` 가 드라이브 문자를 대문자로 맞춘다. `npx vitest` 를 직접 쓰려면 `E:\` 경로에서 실행한다.

## 되돌리기

- Go: `winget install --id GoLang.Go -e --version 1.26.2 --force` 후 `go.mod` 의 `go` 지시문을 되돌린다.
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`.
