# 스토어 밖 설치본의 업데이트 확인 켜기 (공개 릴리스 저장소) — 배경·수정·코드·릴리스 절차·검증

**작성**: 2026-11-26 · **상태**: 코드·자동 테스트 완료(D135). 공개 릴리스 저장소에 첫 릴리스 v1.1.0 을 올렸고, 앱과 같은 코드로 실제 응답을 확인했다. 앱 화면에서의 확인은 §H · **대상**: cli-manager-cr(DariTerm)
**같은 날 문서**: [2026-11-26-1-store-update-child-identity.md](./2026-11-26-1-store-update-child-identity.md) (스토어판의 업데이트 확인 수정, D134). 이 문서는 **스토어가 아닌 설치본**(설치 파일로 설치한 경우) 쪽이다.

---

## 0. 한 줄 요약

설치 파일로 설치한 DariTerm 은 업데이트 확인이 꺼져 있었다(D80 — 물어볼 공개 저장소가 없었다). 공개 릴리스 전용 저장소 `soli7403/dariterm-releases` 를 만들고, **설치 파일을 빌드할 때 그 주소를 넣도록** 했다. 앱 코드는 바꾸지 않았다 — 확인 기능은 이미 있었고 주소만 비어 있었다.

## 1. 배경

- 앱에는 처음부터 "배포 저장소의 최신 릴리스를 묻는" 확인 기능이 있다([internal/update/update.go](../../internal/update/update.go)). `GET https://api.github.com/repos/{owner}/{name}/releases/latest` 의 `tag_name`(버전)과 `html_url`(릴리스 페이지)을 읽는다.
- 저장소 주소는 `internal/app.G_cszUpdateRepo`(빌드 때 `-ldflags -X` 로 넣는 값) 또는 환경 변수 `DARITERM_UPDATE_REPO` 에서 온다. **넣는 빌드 단계가 없어 늘 빈 값**이었고, 빈 값이면 설정 > 정보의 업데이트 항목 자체가 보이지 않는다(D80).
- 소스 저장소는 비공개라 로그인 없이 릴리스를 물을 수 없다. 그래서 물어볼 곳이 없었다.
- 스토어 설치본은 스토어에 묻는다(D115·D134). 이 문서의 대상이 아니다.

## 2. 정한 것

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 물어볼 곳 | 공개 저장소 **`soli7403/dariterm-releases`** — 소스 없이 README 와 릴리스(설치 파일)만 | 소스는 비공개로 두면서 로그인 없이 조회되는 곳이 필요하다. 비공개 저장소를 공개로 바꾸거나 fork 하지 않고 새 빈 저장소로 만든다(이력이 딸려 나오지 않게) |
| 주소를 두는 곳 | `tools/build-config.ps1` 한 곳. 설치 파일 빌드(`tools/installer.ps1`)가 `-ldflags` 로 넣는다 | 주소가 한 곳에만 있다 |
| 소스의 기본값 | 빈 값 그대로 | 기본값으로 넣으면 개발 빌드와 자동 테스트가 공개 API 를 부르게 된다 |
| 스토어 패키지 | 넣지 않는다(`tools/msix.ps1` 그대로) | 스토어 설치본은 스토어에 묻는다. 주소가 있어도 쓰이지 않는다 |
| 앱 하나에 저장소 하나 | DariTerm 전용 | `releases/latest` 는 저장소의 가장 최근 릴리스 하나만 본다. 다른 앱의 릴리스가 섞이면 엉뚱한 버전을 안내한다. Windows 설치 파일과 (나중의) 맥용 파일은 같은 릴리스에 함께 첨부한다 |
| 릴리스 형식 | 태그 `v<버전>`(예 `v1.1.0`), 설치 파일 첨부, "최신 릴리스" | 앱은 태그 앞의 `v` 를 떼고 숫자 세 자리를 비교한다. **초안(draft)·미리 공개(pre-release)는 `latest` 에 잡히지 않는다** |
| 새 버전을 찾았을 때 | 릴리스 페이지(`html_url`)를 연다 — 전부터 있던 동작 | 앱이 직접 내려받거나 설치하지 않는다 |

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `tools/build-config.ps1` | 빌드 설정 | 새 파일 |
| `tools/installer.ps1` | 빌드 스크립트 | 수정 (두 줄) |
| `tools/updatecheck/main.go` | 검증 도구 | 새 파일 |
| `docs/decisions.md` | 문서 | D135 추가 |
| `CHANGELOG.md` | 문서 | 미배포 절에 새 기능 한 줄 |
| `CLAUDE.md` | 문서 | 계획서 인덱스에 추가 |

앱 코드(`internal/**`, `frontend/**`)·명세·문구는 바뀌지 않는다.

## B. 코드

### B.1 `tools/build-config.ps1` (새 파일, 전체 — UTF-8 BOM)

```powershell
# 배포 빌드에 함께 넣는 값 (installer.ps1 이 불러 쓴다, 결정 D135). 스토어 패키지(msix.ps1)는 스토어에 묻으므로 쓰지 않는다.
# $DariTermUpdateRepo: 새 버전을 묻는 공개 릴리스 저장소(owner/name). 스토어 설치본은 스토어에 묻고, 그 밖의 설치본이 이 저장소의 최신 릴리스를 본다.
#   비우면 스토어 밖 설치본의 업데이트 확인이 꺼진다(D80).
$DariTermUpdateRepo = 'soli7403/dariterm-releases'
$DariTermLdflags = "-X dariterm/internal/app.G_cszUpdateRepo=$DariTermUpdateRepo"
```

PowerShell 5.1 은 BOM 없는 UTF-8 파일의 한글을 깨뜨리므로 이 프로젝트의 다른 `.ps1` 과 같이 BOM 을 붙여 저장한다.

### B.2 `tools/installer.ps1` (수정)

```diff
-wails build -clean -nsis
+. (Join-Path $PSScriptRoot 'build-config.ps1')   # $DariTermLdflags: 업데이트를 묻는 릴리스 저장소 (D135)
+wails build -clean -nsis -ldflags $DariTermLdflags
 if ($LASTEXITCODE -ne 0) { throw "wails build 실패 ($LASTEXITCODE)" }
```

### B.3 `tools/updatecheck/main.go` (새 파일, 전체)

```go
// updatecheck 는 릴리스 저장소 방식의 새 버전 확인(internal/update)을 앱 밖에서 한 번 돌려 결과를 적는 검증 도구다 (D135).
// 앱과 같은 코드로 공개 API 에 묻는다. 사용:
//
//	go run ./tools/updatecheck <owner/name> <지금 버전>     예: go run ./tools/updatecheck soli7403/dariterm-releases 1.0.0
package main

import (
	"fmt"
	"os"

	"dariterm/internal/update"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: updatecheck <owner/name> <current version>")
		os.Exit(2)
	}
	res := update.Check("https://api.github.com", os.Args[1], os.Args[2], nil)
	fmt.Printf("repo=%s current=%s -> status=%s version=%s url=%s\n", os.Args[1], os.Args[2], res.M_szStatus, res.M_szVersion, res.M_szUrl)
}
```

## C. 설치 파일 만들기

```
mingw32-make installer            # 지금 버전 그대로
mingw32-make installer VERSION=x.y.z   # 버전을 올린 뒤
→ build\dist\DariTerm-<버전>-amd64-installer.exe
```

**이 PC 의 사정**: `installer.ps1` 은 `wails` 명령을 직접 부르는데 이 PC 에는 `wails` 가 PATH 에 없다. 2026-11-26 의 1.1.0 설치 파일은 같은 절차를 손으로 돌려 만들었다.

```powershell
. .\tools\build-config.ps1
$env:PATH = "$env:PATH;${env:ProgramFiles(x86)}\NSIS"
# (LICENSE 를 build\windows\installer\resources\license.txt 로 UTF-8 BOM 복사 — installer.ps1 과 같게)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build -clean -nsis -ldflags $DariTermLdflags
Copy-Item build\bin\DariTerm-amd64-installer.exe build\dist\DariTerm-1.1.0-amd64-installer.exe
```

주소가 실행 파일에 들어갔는지는 `build\bin\dariterm.exe` 안에서 `soli7403/dariterm-releases` 글자를 찾아 확인했다.

## D. 릴리스 올리는 절차 (새 버전을 낼 때마다)

1. 설치 파일을 만든다(§C). SHA256 을 적어 둔다: `Get-FileHash build\dist\DariTerm-<버전>-amd64-installer.exe -Algorithm SHA256`
2. `https://github.com/soli7403/dariterm-releases/releases/new`
   - 태그: `v<버전>` (새로 만든다, 대상 `main`)
   - 제목: `DariTerm <버전>`
   - 설명: 내려받기 안내, 대상 Windows, SHA256, 사용 조건(개인 무료·업무용 유료)과 문의처
   - 첨부: `DariTerm-<버전>-amd64-installer.exe`
   - **"Set as the latest release" 를 켜고, pre-release·draft 로 두지 않는다**
3. 확인: `go run ./tools/updatecheck soli7403/dariterm-releases <이전 버전>` → `status=newer version=<버전>`
4. 릴리스에서 파일을 다시 내려받아 SHA256 이 같은지 본다.

올리면 안 되는 것: 소스, 비밀값(라이선스 서명 비밀키 등), 스토어용 MSIX(스토어가 배포한다).

첫 릴리스 v1.1.0 은 GitHub 의 릴리스 API 로 올렸다(이 PC 에 `gh` 가 없어 git 에 저장된 github.com 자격 증명을 썼다). 웹 화면에서 올려도 결과는 같다.

## G. 확인한 것 (2026-11-26)

### G.1 자동

```
go build ./...                                      → 성공
go vet ./tools/updatecheck/                         → 지적 없음
go test -count=1 ./internal/app/ ./internal/update/ → ok
```

소스 기본값이 빈 값이라 기존 테스트(`TestUpdateOffWhenNotStoreAndNoRepo` 등)는 그대로 통과한다.

### G.2 실제 저장소에 물어 본 결과 (`tools/updatecheck`)

| 상황 | 결과 |
|---|---|
| 릴리스를 올리기 전 (저장소에 README 만) | `status=error` — 릴리스가 없으면 API 가 404 를 준다. 앱에서는 "확인하지 못했습니다" |
| 지금 버전 1.1.0 | `status=latest` |
| 지금 버전 1.0.0 | `status=newer version=1.1.0 url=https://github.com/soli7403/dariterm-releases/releases/tag/v1.1.0` |
| 지금 버전 1.2.0 (릴리스보다 큼) | `status=latest` |

### G.3 올린 파일

| | |
|---|---|
| 릴리스 | https://github.com/soli7403/dariterm-releases/releases/tag/v1.1.0 |
| 파일 | `DariTerm-1.1.0-amd64-installer.exe` (8,560,366 바이트) |
| SHA256 | `B40BB82ACC72B502AE6F5DC8716353165DB5A1ED04E7603311C88369549F8CF2` |
| 대조 | 릴리스에서 다시 내려받은 파일의 SHA256 이 같다 |
| 내용 | 1.1.0 소스 + 2026-11-26 의 수정 2건(D134, `.xsh` 파일 종류) + 릴리스 저장소 주소. 스토어판 1.1.0 과는 이 셋이 다르다 |

## H. 수동 검증

설치 파일로 설치했거나, 저장소 주소를 넣어 빌드한 `build\bin\dariterm.exe` 를 그대로 실행해서 본다.

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 설정 > 정보 | "업데이트 자동 확인" 토글과 [업데이트 확인] 이 보인다 | |
| 2 | [업데이트 확인] (앱 1.1.0, 릴리스 v1.1.0) | "최신 버전입니다" | |
| 3 | 더 새 릴리스를 올린 뒤 [업데이트 확인] | "새 버전 {버전} 이 있습니다", 상태 표시줄에 "새 버전 {버전}" 버튼 | |
| 4 | 그 버튼을 누른다 | 브라우저로 릴리스 페이지가 열린다 | |
| 5 | 인터넷을 끊고 [업데이트 확인] | "확인하지 못했습니다 (네트워크를 확인하세요)" | |
| 6 | 스토어 설치본에서 [업데이트 확인] | 스토어에 묻는다(이 저장소를 보지 않는다) | |
| 7 | 개발 빌드(주소 없이 `wails build`) | 업데이트 항목이 보이지 않는다 | |

3·4번은 앱 버전보다 큰 릴리스가 있어야 볼 수 있다. 지금은 앱과 릴리스가 모두 1.1.0 이다.

## J. 남은 것·알아 둘 것

- **코드 서명이 없다**: 내려받은 설치 파일을 실행하면 SmartScreen 경고가 나올 수 있다. 스토어판은 스토어가 서명한다.
- **`installer.ps1` 이 `wails` 를 직접 부른다**: PATH 에 없는 PC 에서는 실패한다(§C). `go run` 으로 바꾸거나 CLI 를 설치하면 된다(설치하면 `docs/host-settings/` 에 기록).
- **API 호출 한도**: 로그인 없는 GitHub API 는 IP 마다 시간당 60번이다. 앱은 시작 5초 뒤와 12시간마다 한 번 묻으므로 보통은 닿지 않는다. 같은 IP 뒤에 사용자가 많으면(회사 NAT) 한도에 걸려 "확인하지 못했습니다" 가 나올 수 있다.
- **맥 버전**: 만들면 같은 릴리스에 함께 첨부한다. 맥에서도 이 확인 경로를 탄다(스토어 판정은 Windows 전용). 맥 빌드 자체는 아직 확인한 적이 없다.
- **버전**: 다음 릴리스는 1.1.0 보다 커야 설치된 앱이 새 버전으로 알아본다.
