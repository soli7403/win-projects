# 빌드 PC 에 gcc 설치 — `go test -race` 가 돌지 않던 문제의 원인·설치·검증

**작성**: 2026-11-27 · **상태**: 설치·기록·검증 완료(D144). 앱 코드는 바꾸지 않았다 · **대상**: cli-manager-cr(DariTerm)의 빌드 PC(이 PC) · **Known issue**: 없음
**관련 문서**: [../host-settings/06_gcc-race.md](../host-settings/06_gcc-race.md)(호스트 설치 기록), [../host-settings/05_build-toolchain.md](../host-settings/05_build-toolchain.md)(빌드 도구), 같은 날의 계획서 [-1-](./2026-11-27-1-forward-info.md)~[-5-](./2026-11-27-5-macros.md)(여기서 "`-race` 는 돌리지 못했다"고 적었던 것을 이 문서로 채웠다)

---

## 0. 한 줄 요약

이 PC 에는 C 컴파일러가 없어 프로젝트의 검증 단계 `go test -race ./...` 가 **빌드 단계에서 멈추고 테스트가 하나도 돌지 않았다.** winget 으로 MinGW-w64 gcc 16.2.0(WinLibs)을 설치해 고쳤고, 19개 패키지가 경쟁 검출기 아래에서 모두 통과했다. 앱의 의존성에는 cgo 를 쓰는 패키지가 없어서 **배포 실행 파일은 바뀌지 않는다**(Go 자체 링커로 만들어지고 gcc 의 DLL 이 필요 없다).

## 1. 증상

- 프로젝트 [CLAUDE.md](../../CLAUDE.md) §4 빠른 명령과 [Makefile](../../Makefile) 8행의 `go test -race ./...` 가 이 PC 에서 실패했다.
- 그래서 2026-11-27 의 다섯 계획서(-1~-5)는 모두 `go test ./...`(경쟁 검출 없이)만 돌리고 "**돌리지 못한 것**: `go test -race ./...` — 이 PC 에 gcc 가 없어 cgo 를 켤 수 없다"고 적었다.
- 경쟁 검출을 가장 받아야 할 새 코드가 마침 이날 들어왔다.

| 새 코드 | 동시에 도는 것 |
|---|---|
| 로그인 스크립트 (D139, `internal/session/loginscript.go`) | 세션 읽기 루프가 출력을 넣는 동안 단계 시한 타이머(`armLoginTimeout`)가 다른 고루틴에서 상태를 본다. 사용자 입력이 오면 스크립트를 취소한다 |
| 포워딩 연결 수 (D142, `internal/sshconn/forward_count.go`) | 듣는 소켓이 연결을 받을 때 +1, 연결이 닫힐 때 -1 을 각기 다른 고루틴에서 하고, 정보 창의 `session.forwards` 요청이 또 다른 고루틴에서 읽는다 |

## 2. 원인

### 2.1 실제로 난 오류 (go1.27.4 windows/amd64)

설치 전 상태: `go env CGO_ENABLED` → `0`, `go env CC` → `gcc`, `Get-Command gcc` → 없음.

**① 그냥 실행**

```
> go test -race ./internal/session/
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
(종료 코드 2)
```

**② cgo 를 강제로 켜고 실행**

```
> $env:CGO_ENABLED = '1'; go test -race ./internal/session/
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
FAIL	dariterm/internal/session [build failed]
FAIL
(종료 코드 1)
```

### 2.2 왜 gcc 가 필요한가

| 단계 | 내용 |
|---|---|
| 경쟁 검출기의 정체 | Go 가 직접 만든 것이 아니라 LLVM 의 **ThreadSanitizer(TSan)** 런타임을 가져다 쓴다. TSan 은 C++ 로 작성돼 있다 |
| Go 컴파일러가 하는 일 | `-race` 면 메모리 읽기·쓰기마다, 그리고 고루틴 생성·채널·뮤텍스 같은 동기화 지점마다 TSan 에 알리는 호출을 끼워 넣는다 |
| TSan 런타임이 하는 일 | 주소마다 최근 접근(누가·읽기/쓰기)을 섀도 메모리에 적고, 고루틴 사이의 선후 관계를 벡터 시계로 따진다. 선후 관계가 없는 두 접근이 같은 주소에서 만나고 하나라도 쓰기이면 `WARNING: DATA RACE` |
| 런타임의 형태 | 미리 컴파일된 오브젝트로 Go 배포판에 들어 있다(`runtime/race` 의 `race_windows_amd64.syso`). gcc 로 다시 컴파일하지 않는다 |
| cgo 가 필요한 까닭 | 이 오브젝트는 C 런타임을 참조한다. 그래서 `-race` 는 cgo 를 요구한다(① 의 오류) |
| gcc 가 필요한 까닭 | cgo 를 켜면 `runtime/cgo` 패키지의 C 소스를 먼저 컴파일해야 하고, 링크도 외부 링커로 한다. Windows 에서 Go 가 찾는 C 컴파일러는 `gcc` 다(② 의 오류 — 링크까지 가지 않고 `runtime/cgo` 컴파일에서 멈춘다) |
| 왜 `CGO_ENABLED` 가 0 이었나 | Go 는 `CC`(기본 `gcc`)를 PATH 에서 찾지 못하면 cgo 를 기본으로 끈다 |

- macOS 는 Go 1.20 부터 cgo 없이 `-race` 가 되지만, Windows 는 여전히 gcc 가 있어야 한다(이 PC 에서 ①·② 로 확인).
- 앱 빌드(`go build`, `wails build`)와 `-race` 없는 `go test` 는 순수 Go 라 gcc 가 필요 없다. 그래서 이제까지 빌드·배포에는 문제가 드러나지 않았다.

## 3. 고친 방법

### 3.1 무엇을 설치했나

| 항목 | 값 |
|---|---|
| 패키지 | winget `BrechtSanders.WinLibs.POSIX.UCRT` — MinGW-w64, POSIX 스레드, UCRT 런타임 |
| 버전 | 16.2.0-14.0.0-r3 |
| `gcc --version` | `gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh, built by Brecht Sanders, r3) 16.2.0` |
| 설치 명령 | `winget install --id BrechtSanders.WinLibs.POSIX.UCRT -e --source winget --accept-package-agreements --accept-source-agreements --disable-interactivity` |
| 관리자 권한 | 필요 없음(압축을 풀어 두는 패키지) |
| 위치 | `%LOCALAPPDATA%\Microsoft\WinGet\Packages\BrechtSanders.WinLibs.POSIX.UCRT_Microsoft.Winget.Source_8wekyb3d8bbwe\mingw64\bin` |
| PATH | winget 이 `gcc`·`g++`·`x86_64-w64-mingw32-gcc` 등을 명령 별칭으로 등록한다(설치 출력의 `Command line alias added: …`). 설치 전에 열려 있던 터미널·VS Code 는 새로 열어야 보인다 |

### 3.2 고른 까닭과 다른 안

설치 전에 본 이 PC 의 패키지 관리자: winget 있음, choco 있음, scoop 없음.

| 안 | 판단 |
|---|---|
| **winget `BrechtSanders.WinLibs.POSIX.UCRT`** | **채택.** 다른 빌드 도구(Go·NSIS·Windows SDK)와 같은 winget 으로 관리된다. LLVM 이 빠진 가장 작은 판이다. UCRT 는 Windows 10 이상의 기본 C 런타임이다. POSIX 스레드 판이 MinGW 의 일반적인 기본이다 |
| winget `…WinLibs.POSIX.UCRT.LLVM` | 기각 — clang 등 LLVM 도구까지 들어 커진다. 쓸 일이 없다 |
| winget `…WinLibs.POSIX.MSVCRT` | 기각 — 옛 C 런타임(MSVCRT). 새로 깔 이유가 없다 |
| winget `…WinLibs.MCF.UCRT` | 기각 — MCF 스레드는 덜 쓰이는 판이다 |
| scoop `mingw` | 기각 — scoop 이 없다. 관리자를 하나 더 늘리지 않는다 |
| choco `mingw` | 기각 — 관리자 권한이 필요하고, 이 프로젝트의 기록은 winget 기준이다 |
| MSYS2 | 기각 — 셸·패키지 체계를 통째로 들인다. gcc 하나 쓰려고는 크다 |

### 3.3 설치 뒤 확인

```
> gcc --version   (첫 줄)
gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh, built by Brecht Sanders, r3) 16.2.0
> go env CGO_ENABLED
1
```

`CGO_ENABLED` 가 0 → **1** 로 바뀌었다. 따로 설정하지 않았다 — gcc 가 PATH 에 생겨서 Go 의 기본값이 바뀐 것이다.

## 4. 배포 빌드에 미치는 영향 (없음 — 확인함)

cgo 가 기본으로 켜지면, 의존성 중 C 코드가 있는 패키지가 있을 때 실행 파일이 외부 링커로 만들어지고 gcc 런타임 DLL(`libwinpthread-1.dll` 등)에 기댈 수 있다. 그렇게 되면 사용자 PC 에서 실행이 안 될 수 있으므로 세 가지로 확인했다.

| 확인 | 명령 | 결과 |
|---|---|---|
| cgo 를 쓰는 의존성 | `go list -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' .` | **없음**(빈 출력) |
| 빌드 스크립트의 CGO 설정 | `wails.json`·`Makefile`·`tools\*.ps1` 에서 `CGO` 찾기 | 없음(아무도 강제하지 않는다) |
| 링크 방식 | `go build -a -tags desktop,production -ldflags=-v -o … .` 의 링커 출력에서 `host link` 찾기 | **없음** — 외부 링커를 부르지 않고 Go 자체 링커로 만든다 |
| 빌드 정보 | `go version -m <exe>` | `CGO_ENABLED=1` 로 찍힌다. **달라지는 것은 이 한 줄뿐**이다 |

- 의존성을 올린 뒤(05 문서의 "의존성 올리는 법")에는 위 `go list` 를 한 번 돌려 cgo 패키지가 새로 들어오지 않았는지 본다. 06 문서에도 적었다.
- 꼭 예전과 똑같이 빌드하고 싶으면 `CGO_ENABLED=0` 을 주고 빌드하면 된다. 지금은 결과가 같아 넣지 않았다.

## 5. 바꾼 파일

앱 코드(`internal/`, `frontend/`)·명세(`spec/`)·문구(`locales/`)는 바꾸지 않았다. 바꾼 것은 호스트 설치 기록(공통 규칙 §8)과 계획서·결정 기록뿐이다.

| 파일 | 바꾼 것 |
|---|---|
| `docs/host-settings/06_gcc-race.md` | **새 문서** — 왜·무엇을·설치 뒤 달라진 것·확인·되돌리기 |
| `docs/host-settings/README.md` | 문서 표에 06, 패키지 목록에 WinLibs, 변경 이력에 2026-11-27 |
| `docs/host-settings/05_build-toolchain.md` | 도구 표에 gcc 한 줄 |
| `docs/host-settings/setup.ps1` | gcc 가 없을 때만 설치(멱등) |
| `docs/host-settings/check.ps1` | gcc 버전 점검 |
| `docs/import-plans/2026-11-27-6-gcc-race-test.md` | 이 문서 |
| `docs/import-plans/2026-11-27-1~5-*.md` | "돌리지 못한 것"을 결과로, 체크리스트의 `-race` 항목에 표시 |
| `docs/decisions.md` | D144 |
| `CLAUDE.md` | 문서 인덱스의 import-plans 칸에 이 문서 |

### 5.1 `docs/host-settings/README.md`

```diff
 | 05 | [05_build-toolchain.md](./05_build-toolchain.md) | 빌드 도구 Go · Wails CLI · Node.js 버전과 의존성 올리는 법 | Go 1.27.4 · Wails v2.17.1 (2026-11-25) |
+| 06 | [06_gcc-race.md](./06_gcc-race.md) | C 컴파일러 gcc — `go test -race` 용(앱 빌드에는 필요 없음) | 설치됨 16.2.0 (2026-11-27) |
```

```diff
 | go install | `github.com/wailsapp/wails/v2/cmd/wails` | v2.17.1 | `wails build` (go.mod 의 wails 버전과 같게) | 05 |
+| winget | `BrechtSanders.WinLibs.POSIX.UCRT` | 16.2.0-14.0.0-r3 | MinGW-w64 gcc — `go test -race` 의 cgo | 06 |
```

```diff
 | 2026-11-25 | Go 1.26.2 → 1.27.4 (winget), Wails CLI v2.10.2 → v2.17.1 (go install), 빌드 도구 문서 05 추가 (D124-1) |
+| 2026-11-27 | MinGW-w64 gcc 16.2.0 설치(winget WinLibs), `go test -race ./...` 가 돌게 됨 — 문서 06. `CGO_ENABLED` 기본값이 1 로 바뀜(앱 의존성에 cgo 없음, 배포 빌드 영향 없음) |
```

### 5.2 `docs/host-settings/05_build-toolchain.md`

```diff
 | Node.js | 24.14.1 (바꾸지 않음) | 원래 설치돼 있음 | — |
+| gcc (MinGW-w64) | 16.2.0 (2026-11-27) | `go test -race` 에만 필요 — [06_gcc-race.md](./06_gcc-race.md) | winget 패키지 폴더 |
```

### 5.3 `docs/host-settings/setup.ps1`

Wails CLI 확인 다음, NSIS 확인 앞에 넣었다. 다른 항목과 같이 **이미 있으면 건너뛴다**(여러 번 실행해도 같다). 주석은 다른 줄과 같이 영문이다(Windows PowerShell 5.1 이 BOM 없는 UTF-8 한글을 깨뜨리지 않게).

```diff
     go install "github.com/wailsapp/wails/v2/cmd/wails@$wailsWant"
 }
 
+# C compiler for go test -race (06_gcc-race.md): MinGW-w64 gcc from WinLibs
+if (Get-Command gcc -ErrorAction SilentlyContinue) {
+    Write-Host "gcc: already installed"
+} else {
+    winget install --id BrechtSanders.WinLibs.POSIX.UCRT -e --source winget --silent --accept-source-agreements --accept-package-agreements
+}
+
 $nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
 if (Test-Path $nsis) {
```

- 판정 기준을 "PATH 에 `gcc` 가 있는가"로 둔 까닭: 다른 경로(MSYS2 등)로 깔린 gcc 가 이미 있으면 그것으로도 `-race` 가 돈다. 굳이 하나 더 깔지 않는다.

### 5.4 `docs/host-settings/check.ps1`

```diff
 Check 'node' ([bool](Get-Command node -ErrorAction SilentlyContinue)) ''
+# C compiler for go test -race (06_gcc-race.md)
+$gccVer = if (Get-Command gcc -ErrorAction SilentlyContinue) { (gcc -dumpfullversion) } else { '' }
+Check 'gcc (go test -race)' ([bool]$gccVer) $gccVer
```

- gcc 가 없으면 `[MISS]` 로 보이고 종료 코드 1 이 된다(다른 필수 도구와 같은 대우).

### 5.5 `docs/host-settings/06_gcc-race.md` (새 문서)

절: 왜(설치 전 두 오류의 표) · 무엇을(패키지·버전·설치 명령·위치·PATH, 고른 까닭) · 설치 뒤 달라진 것(`CGO_ENABLED` 1, 배포 빌드 영향 없음, 의존성을 올린 뒤의 확인) · 확인(2026-11-27 결과) · 되돌리기. 내용은 이 문서 §2~§4·§7 과 같다.

## 6. 검증

### 6.1 전체 `go test -race ./...` (2026-11-27)

`exit=0`, 걸린 시간 약 95초. 테스트 파일이 없는 패키지(`?`)는 뺐다.

| 패키지 | 결과 | 시간 |
|---|---|---|
| `dariterm` | ok | 2.7s |
| `internal/app` | ok | 53.9s |
| `internal/backup` | ok | 3.3s |
| `internal/clipboard` | ok | 3.3s |
| `internal/gitsync` | ok | 32.2s |
| `internal/i18n` | ok | 3.6s |
| `internal/importer` | ok | 3.3s |
| `internal/ipc` | ok | 3.6s |
| `internal/license` | ok | 3.2s |
| `internal/logclean` | ok | 3.2s |
| `internal/session` | ok | 22.7s |
| `internal/sftpfs` | ok | 3.4s |
| `internal/sshconn` | ok | 3.8s |
| `internal/store` | ok | 2.2s |
| `internal/storeupdate` | ok | 4.8s |
| `internal/transport` | ok | 3.3s |
| `internal/update` | ok | 2.1s |
| `internal/vault` | ok | 71.7s |
| `internal/zmodem` | ok | 3.2s |

**19개 패키지 모두 통과, `WARNING: DATA RACE` 없음.** `-race` 는 실행이 몇 배 느려진다. 가장 오래 걸린 `internal/vault` 는 비밀번호 키 유도(일부러 느린 계산)를 여러 번 하는 테스트라서 그렇다.

### 6.2 이날 들어온 동시성 코드만 따로 (`-count=1`, 캐시 없이)

`go test -race -count=1 -v -run 'Login|Forward|Count' ./internal/session/ ./internal/sshconn/`

| 패키지 | 테스트 | 무엇을 보는가 | 결과 |
|---|---|---|---|
| session | `TestMsgForwardsLines`·`TestMsgForwardsFailed`·`TestMsgForwardsEmpty` | 포워딩 안내 블록(D136) | PASS |
| session | `TestSessionInfoCarriesForwards` | 세션 정보의 `forwards`(D137) | PASS |
| session | `TestLoginRunnerSteps`·`…DoesNotReuseMatch`·`…IgnoresAnsi`·`…EmptyExpect`·`TestTrimLoginSteps` | 로그인 스크립트 단계 진행(D139) | PASS |
| session | `TestSessionRunsLoginScript` | 실제 세션 읽기 루프에서 스크립트 실행 | PASS |
| session | `TestSessionLoginScriptStopsWithoutPassword` | 비밀번호가 없을 때 멈춤 | PASS |
| session | `TestSessionLoginScriptTimeout` | **타이머 고루틴과 읽기 루프가 겹치는 경우** | PASS |
| session | `TestSessionLoginScriptCancelledByInput` | **사용자 입력으로 취소 — 입력·읽기·타이머 세 쪽이 겹친다** | PASS |
| sshconn | `TestForwardCountsOpenConnections` | **받기·닫기 콜백의 +1/-1 과 읽기가 겹치는 경우**(D142) | PASS |
| sshconn | `TestForwardCountDropsFailedDial` | 대상 연결 실패 시 수를 되돌림 | PASS |
| sshconn | `TestForwardResultActiveWithoutCounter` | 세지 않는 포워딩(동적 등) | PASS |
| sshconn | `TestAgentForwardReachesLocalAgent`·`…OffByDefault`·`…FailuresOnlyNotify` | 에이전트 전달(D111) — 이름이 걸려 함께 돌았다 | PASS |

`-v` 로 센 `internal/session`·`internal/sshconn` 두 패키지의 실행 테스트(`=== RUN`) 수는 119개, 모두 통과.

### 6.3 점검 스크립트

`powershell -NoProfile -File docs\host-settings\check.ps1`

```
[OK]   NSIS makensis  C:\Program Files (x86)\NSIS\makensis.exe
[OK]   go 1.27.4  go1.27.4
[MISS] wails v2.17.1
[OK]   node
[OK]   gcc (go test -race)  16.2.0
[OK]   Windows SDK makeappx  10.0.26100.0
[MISS] ffmpeg
[INFO] MSIX test cert  none (created by tools\msix.ps1 -TestSign)
[MISS] msstore CLI  C:\Users\nable\AppData\Local\Microsoft\WindowsApps\MSStore.exe
```

- gcc 는 `[OK] 16.2.0`.
- `[MISS]` 셋은 **이번 설치와 관계없이 원래 이 PC 에 없던 것**이다. Wails CLI 는 이 PC 에서 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` 로 대신한다(v2.10.2 CLI 는 Go 1.27 에서 실패). ffmpeg·msstore 는 MSIX 를 만드는 PC 에만 필요하다. 그래서 종료 코드는 1 이다.

## 7. 영향 범위와 위험

| 항목 | 내용 |
|---|---|
| 앱 동작 | 바뀌지 않음(코드 변경 없음) |
| 배포 실행 파일 | 바뀌지 않음 — §4. 빌드 정보의 `CGO_ENABLED=1` 한 줄만 다르다 |
| 빌드·테스트 시간 | `-race` 없는 명령은 같다. `-race` 는 이제 실제로 돌기 때문에 약 95초가 든다(전에는 곧바로 실패했다) |
| 다른 PC | 이 PC 만의 설치다. 다른 빌드 PC 는 `setup.ps1` 을 돌리면 같은 상태가 된다 |
| 다른 OS | 해당 없음. Linux·macOS 는 보통 C 컴파일러가 이미 있다 |
| 위험 | 나중에 cgo 를 쓰는 의존성이 들어오면 그때부터 실행 파일이 외부 링커로 만들어져 gcc 런타임 DLL 에 기댈 수 있다. 의존성을 올린 뒤 §4 의 `go list` 로 확인한다 |
| 되돌리기 | `winget uninstall --id BrechtSanders.WinLibs.POSIX.UCRT -e`. 그러면 `-race` 만 다시 돌지 않는다(`CGO_ENABLED` 도 0 으로 돌아간다) |

## H. 수동 확인

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | VS Code 를 다시 연 뒤 통합 터미널에서 `gcc --version` | `… 16.2.0` | |
| 2 | 같은 터미널에서 `go env CGO_ENABLED` | `1` | |
| 3 | `make test`(또는 `go test -race ./...`) | 19개 패키지 ok | 2026-11-27 Claude 세션에서 확인(§6.1) |
| 4 | `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` 뒤 앱 실행 | 전과 같이 뜬다(gcc DLL 을 찾는 오류가 없다) | |

## 8. 체크리스트

- [x] 오류 재현(§2.1 ①·②)
- [x] gcc 설치(winget WinLibs POSIX UCRT 16.2.0)
- [x] `gcc --version`, `go env CGO_ENABLED` 확인
- [x] `go test -race ./...` 전체 통과(§6.1)
- [x] 이날의 동시성 코드 따로 통과(§6.2)
- [x] 배포 빌드 영향 없음 확인(§4)
- [x] 호스트 기록: 06 새 문서, README(표·패키지·이력), 05, `setup.ps1`, `check.ps1`
- [x] `check.ps1` 실행(§6.3)
- [x] 2026-11-27-1~5 계획서의 "돌리지 못한 것" 갱신
- [x] `docs/decisions.md` D144, `CLAUDE.md` 인덱스
- [ ] 수동 확인 1·2·4 (§H)
- 해당 없음: `CHANGELOG.md`(앱이 바뀌지 않았다), 명세·문구
