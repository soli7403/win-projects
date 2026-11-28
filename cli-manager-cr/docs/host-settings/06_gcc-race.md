# 06. C 컴파일러 gcc (MinGW-w64) — `go test -race` 용

> 2026-11-27 기록. 결정 D144. 계획서 [../import-plans/2026-11-27-6-gcc-race-test.md](../import-plans/2026-11-27-6-gcc-race-test.md).
> 앱 빌드(`wails build`)에는 필요 없다. 경쟁 검출 테스트(`go test -race ./...`, Makefile `test`)에만 쓴다.

## 왜

gcc 가 없는 PC 에서는 `go test -race` 가 테스트를 하나도 돌리지 못하고 빌드 단계에서 멈춘다.

| 실행 | 오류 |
|---|---|
| `go test -race ./internal/session/` | `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` (종료 코드 2) |
| `$env:CGO_ENABLED = '1'; go test -race ./internal/session/` | `cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%` (종료 코드 1) |

- 경쟁 검출기는 LLVM ThreadSanitizer 런타임(Go 배포판의 `runtime/race` 미리 컴파일된 오브젝트)을 쓰고, 이것이 C 런타임을 참조하므로 `-race` 는 cgo 를 요구한다.
- cgo 를 켜면 `runtime/cgo` 의 C 소스를 컴파일하고 외부 링커로 링크해야 한다. Windows 에서 Go 가 찾는 C 컴파일러는 `gcc` 다.
- Go 는 `CC`(기본 `gcc`)를 PATH 에서 찾지 못하면 cgo 를 기본으로 끈다(`CGO_ENABLED=0`).
- macOS 는 Go 1.20 부터 cgo 없이 `-race` 가 되지만 Windows 는 여전히 gcc 가 있어야 한다.

## 무엇을

| 항목 | 값 |
|---|---|
| 패키지 | winget `BrechtSanders.WinLibs.POSIX.UCRT` — MinGW-w64, POSIX 스레드, UCRT 런타임 |
| 버전 | 16.2.0-14.0.0-r3 (2026-11-27 설치한 빌드 PC). **이 저장소의 주 PC(E:\work) 에는 16.1.0-14.0.0-r4 가 이미 설치돼 있었다** — `-race` 에는 차이가 없어 올리지 않았다 |
| 설치 명령 | `winget install --id BrechtSanders.WinLibs.POSIX.UCRT -e --source winget --accept-package-agreements --accept-source-agreements --disable-interactivity` |
| 관리자 권한 | 필요 없음(압축을 풀어 두는 패키지) |
| 위치 | `%LOCALAPPDATA%\Microsoft\WinGet\Packages\BrechtSanders.WinLibs.POSIX.UCRT_Microsoft.Winget.Source_8wekyb3d8bbwe\mingw64\bin` |
| PATH | winget 이 `gcc`·`g++` 등을 명령 별칭으로 등록한다. 설치 전에 열려 있던 터미널·VS Code 는 새로 열어야 보인다 |

고른 까닭: 다른 빌드 도구(Go·NSIS·Windows SDK)와 같은 winget 으로 관리되고, LLVM 이 빠진 가장 작은 판이다. UCRT 는 Windows 10 이상의 기본 C 런타임, POSIX 스레드 판이 MinGW 의 일반적인 기본이다.
기각: `…UCRT.LLVM`(clang 까지 들어 큼), `…POSIX.MSVCRT`(옛 C 런타임), `…MCF.UCRT`(덜 쓰이는 스레드 판), scoop `mingw`(scoop 없음), choco `mingw`(관리자 권한 필요), MSYS2(셸·패키지 체계를 통째로 들임).

## 설치 뒤 달라진 것

- `go env CGO_ENABLED` 가 0 → **1**. 따로 설정한 것이 아니라 gcc 가 PATH 에 생겨 Go 의 기본값이 바뀐 것이다.
- **배포 빌드에는 영향 없음** (2026-11-27 확인):

| 확인 | 명령 | 결과 |
|---|---|---|
| cgo 를 쓰는 의존성 | `go list -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' .` | 없음(빈 출력) |
| 빌드 스크립트의 CGO 설정 | `wails.json`·`Makefile`·`tools\*.ps1` 에서 `CGO` 찾기 | 없음 |
| 링크 방식 | `go build -a -tags desktop,production -ldflags=-v -o … .` 의 링커 출력에서 `host link` | 없음 — Go 자체 링커 |
| 빌드 정보 | `go version -m <exe>` | `CGO_ENABLED=1` 한 줄만 다르다 |

- **의존성을 올린 뒤**([05](./05_build-toolchain.md) "의존성 올리는 법")에는 위 `go list` 를 한 번 돌려 cgo 패키지가 새로 들어오지 않았는지 본다. 들어오면 실행 파일이 외부 링커로 만들어져 gcc 런타임 DLL(`libwinpthread-1.dll` 등)에 기댈 수 있다.
- 예전과 똑같이 빌드하고 싶으면 `CGO_ENABLED=0` 을 주고 빌드한다(지금은 결과가 같아 넣지 않았다).

## 확인

- `gcc --version` 첫 줄 `gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh, built by Brecht Sanders, …)`, `go env CGO_ENABLED` → `1`
- `powershell -NoProfile -File docs\host-settings\check.ps1` → `[OK] gcc (go test -race)  <버전>`
- `go test -race ./...` — 2026-11-27 빌드 PC 에서 19개 패키지 모두 통과(약 95초, `WARNING: DATA RACE` 없음)

## 되돌리기

`winget uninstall --id BrechtSanders.WinLibs.POSIX.UCRT -e`. 그러면 `-race` 만 다시 돌지 않는다(`CGO_ENABLED` 도 0 으로 돌아간다).
