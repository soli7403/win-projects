---
name: dariterm-release
description: >-
  DariTerm 새 버전(x.y.z) 배포 절차 — 검증 → 성격별 커밋 → 버전 올림·MSIX·설치 파일 빌드 → 배포 문서
  (CHANGELOG·releases·스토어 문구·제출 이력·결정·설명서·도움말) → push → 공개 릴리스 저장소(GitHub) 게시·확인 →
  Partner Center 제출 안내. "1.3.0 버전 만들어서 커밋하고 푸시해줘" / "새 버전 배포" / "dariterm-releases 에 릴리스 올려줘" /
  "스토어 등록 내용 만들어줘" 같은 요청 시 사용.
---

# dariterm-release — DariTerm 버전 배포

## 목적 / 왜 필요한가

배포 한 번에 손댈 곳이 열 군데가 넘는다(wails.json, 패키지 두 종류, SHA256, CHANGELOG, 배포 안내, 스토어 문구 두 언어,
제출 이력, 결정 기록, 설명서·매뉴얼 PDF, 도움말, 릴리스 저장소). 하나라도 빠지면 스토어 문구와 실제 기능이 어긋나거나,
설치 파일판이 새 버전을 못 찾는다. 2026-11-28 의 1.2.0 배포(D161·D166)에서 검증한 순서를 고정한다.

경로는 모두 `cli-manager-cr/` 기준. 이 PC 의 make 는 `mingw32-make`. 응답·커밋 메시지는 한국어(공통 규칙 §8·§9).

## 절차

### 0. 버전 번호 정하기

- 새 기능·설정 구역·동작 변경이 있으면 **기능 버전**(x.Y.0), 수정만이면 x.y.Z.
- **스토어에 아직 제출하지 않았고 릴리스 저장소에도 게시하지 않은 번호**면 같은 번호로 다시 만든다(D133·D166). 이미 나간 번호는 절대 다시 쓰지 않는다.
- 직전 스토어 제출의 인증 상태를 사용자에게 확인한다(진행 중인 제출이 있으면 새 제출을 못 만든다). 결과는 `docs/store/README.md` §5 "결과" 칸에 적는다.
- 다단계 작업이므로 `.claude/wip/<날짜>-release-x.y.z.md` 를 만든다(공통 규칙 §4).

### 1. 검증 (빌드 전에 전부 통과해야 한다)

```bash
go vet ./... && GOOS=linux go vet ./...
go test -race ./...
(cd frontend && npx tsc --noEmit && npm test)
go run ./tools/notices        # 의존성이 바뀌었으면. 개발용 의존성(playwright-core 등)은 고지에 안 들어간다
```

### 2. 미커밋 변경을 성격별로 커밋

- `git status` 로 범위를 보고 나눈다: **fix**(수정) / **feat**(기능 코드: internal·frontend·locales·main.go·tools) / **docs**(계획서·spec·설명서·결정).
- 넣지 않을 것: 루트의 임시 파일(`tmp_dummy` 등), `capture/` 같은 스크린샷 폴더, 비밀값.
- 메시지는 한글, 끝에 `Co-Authored-By` 줄. 접두어 예: `feat(cli-manager-cr): …`.

### 3. 버전 올리고 패키지 만들기

```bash
mingw32-make msix VERSION=x.y.z   # wails.json info.productVersion 을 올리고 앱 빌드 → build/dist/DariTerm-x.y.z.0.msix
mingw32-make installer            # VERSION 없이 → 같은 버전의 NSIS 설치 파일 (업데이트 저장소 주소를 ldflags 로 넣는다, D135)
# 같은 번호로 다시 만들 때는 msix 도 VERSION 없이
```

```powershell
Set-Location build\dist
Compress-Archive -Path DariTerm-x.y.z-amd64-installer.exe -DestinationPath DariTerm-x.y.z-amd64-installer.zip -Force
foreach ($n in 'DariTerm-x.y.z.0.msix','DariTerm-x.y.z-amd64-installer.exe','DariTerm-x.y.z-amd64-installer.zip') { $f=Get-Item $n; "$n $($f.Length) $((Get-FileHash $f -Algorithm SHA256).Hash)" }
Select-String ..\msix-layout\AppxManifest.xml -Pattern 'Version="x.y.z.0"'                       # 매니페스트 버전
(Select-String ..\bin\dariterm.exe -Pattern 'soli7403/dariterm-releases' -SimpleMatch -List) -ne $null   # 설치 파일판에만 True
```

- `build/bin/dariterm.exe` 는 마지막 빌드(설치 파일판)다. MSIX 안의 exe(`build/msix-layout/`)에는 업데이트 저장소 주소가 **없어야** 한다(스토어에 묻는다).
- 커밋: `chore(cli-manager-cr): 버전 x.y.z 으로 올림 (…)` — `wails.json` 과 Wails 가 다시 만든 `build/windows/installer/wails_tools.nsh`.

### 4. 배포 문서

| 파일 | 할 일 |
|---|---|
| `CHANGELOG.md` | "## 미배포" → "## x.y.z — 날짜", 머리에 패키지 이름·결정 번호·배포 안내 링크 |
| `docs/releases/x.y.z.md` | 직전 버전 문서를 본떠 새로: §1 배포 파일 표(크기·SHA256·누구에게), 만든 방법, 빌드 전 검증 결과 / §2 바뀐 것 표(결정 번호), 2.1 사용자가 달라졌다고 느낄 점 / §3 배포 절차 체크리스트(스토어·릴리스 저장소·커밋) / §4 수동 확인 표(계획서 §H 링크) / **부록 A 릴리스 저장소 설명**(한국어+영어, v직전 형식) |
| `docs/store/listing.md` | 영어·한국어 각각 **이 버전의 새로운 기능**(1500자 이내 — 아래로 셈), 지난 버전 문구는 "(지난 버전 — 참고용, 붙여 넣지 않는다)" 로 남김. 기능이 늘었으면 **설명** 단락과 **기능** 목록(최대 20개, 각 200자)도. 검색어 최대 7개, 다른 제품 이름 금지 |
| `docs/store/README.md` §5 | 제출 이력 한 줄(날짜·버전·패키지·SHA256·결과 "패키지만 만듦, 제출 차례") |
| `docs/decisions.md` | 결정 한 줄(버전 이유, 패키지 해시, 같은 번호 재빌드면 그 까닭) |
| `CLAUDE.md` (프로젝트) | 문서 인덱스의 `docs/releases/` 줄을 최신 버전으로 |
| 설명서·매뉴얼 | 기능이 바뀌었으면 `docs/manual/user-guide.{ko,en}.md`, `docs/scenarios{,.en}.md` 를 고치고 `tools/manual/build.mjs` 의 기준 버전·영역 수를 맞춘 뒤 `node tools/manual/build.mjs ko` / `en` 으로 HTML·PDF 다시 만들기 |
| 도움말 | 새 기능은 `locales/help/{ko,en}.md` 주제(id 는 두 언어 같게). 주제 수를 세는 시험(`HelpDialog.test.tsx`)도 맞춘다 |

스토어 문구 글자 수 세기(wc 는 바이트를 센다 — 쓰지 않는다):

```bash
python -I -c "import sys;print(len(open(sys.argv[1],encoding='utf-8').read().strip()))" <문구만 담은 파일>
```

커밋: `docs(cli-manager-cr): x.y.z 배포 안내·버전 이력, 스토어 문구 …` → **push**(`git push origin main`, 자격 증명은 GCM).

### 5. 공개 릴리스 저장소 게시 (설치 파일판 업데이트 확인용, D135)

사용자가 게시를 요청했을 때만 한다(외부 공개). **사용자의 Chrome(GitHub 로그인)을 Claude in Chrome 으로** 쓴다.
git 자격 증명으로 릴리스 API 를 부르는 방식은 권한 검사에서 막히니 쓰지 않는다.

1. `tabs_context_mcp` → `navigate` 로 `https://github.com/soli7403/dariterm-releases/releases/new`. 처음엔 탭이 "새 탭"으로 보일 수 있다 — `wait` 2초 뒤 다시 보면 열려 있다(권한 문제로 착각하지 말 것).
2. `find` 로 입력 요소를 찾고 **먼저 설치 파일을 첨부**: `file_upload` 를 "Attach binaries" 의 file-input 에(`build/dist/DariTerm-x.y.z-amd64-installer.exe`, 10MB 미만). 이때 초안이 자동 저장돼 주소가 `releases/edit/untagged-…` 로 바뀌는데 정상이다.
3. 태그: "Tag: Select tag" → 검색 칸에 `vx.y.z` → **Create new tag** → 창의 **Create**(대상 `main`, "will be created on publish").
4. `form_input` 으로 제목 `DariTerm x.y.z`, 설명 = 배포 안내의 부록 A.
5. 아래로 내려 첨부 파일 이름·크기와 Release label **Latest** 를 확인하고 **Publish release**. 몇 초 뒤 주소가 `releases/tag/vx.y.z` 가 되면 끝. 쓴 탭은 닫는다.
6. 확인:

```bash
go run ./tools/updatecheck soli7403/dariterm-releases <직전 버전>   # status=newer version=x.y.z
go run ./tools/updatecheck soli7403/dariterm-releases x.y.z          # status=latest
# 릴리스에서 내려받은 파일의 크기·SHA256 이 로컬과 같은지
curl -sL -o <임시>/i.exe https://github.com/soli7403/dariterm-releases/releases/download/vx.y.z/DariTerm-x.y.z-amd64-installer.exe
```

7. 배포 안내 §3 의 릴리스 항목을 `[x]` 로 하고 주소·확인 결과를 적어 커밋·push.

### 6. Microsoft Store 제출 (사용자가 직접)

Partner Center 는 사용자가 한다. 알려 줄 것:

1. 직전 제출의 인증이 끝났는지(안 끝났으면 기다린다).
2. DariTerm > 제출 업데이트 → **패키지**: `build\dist\DariTerm-x.y.z.0.msix`(SHA256) 올리고 이전 패키지 제거.
3. **스토어 등록 정보** English·한국어 각각 listing.md 에서: 이 버전의 새로운 기능(필수), 바뀌었으면 설명·기능 목록. 짧은 설명·검색어·저작권·추가 사용 조건은 보통 그대로.
4. 제출 → 결과를 알려 주면 §5 제출 이력 "결과" 칸에 적는다.

## 주의사항

- 스토어 제출·릴리스 게시는 밖으로 나가는 일이다. 사용자가 그 단계를 요청했을 때만 하고, 스토어 제출은 맡기지 않는다.
- 같은 번호로 다시 만들면 앞의 패키지는 덮어쓰인다 — 배포 안내·결정에 "옛 해시는 올리지 않는다"를 남긴다.
- 스토어 MSIX 는 서명하지 않는다(스토어가 서명). 직접 설치하지 않는다. MSIX 는 릴리스 저장소에 올리지 않는다(소스·비밀값도).
- XMODEM 등 기능 설명을 스토어 문구에 넣을 때 다른 제품 이름(PuTTY·SecureCRT·Xshell)은 **검색어**에 넣지 않는다(설명 본문은 가능).
- 릴리스 페이지의 Delete 버튼 등 확인 창을 띄우는 요소는 누르지 않는다(확장이 멈춘다).
- 설명서(user-guide·scenarios)는 쉽게 뒤처진다 — 버전마다 "이 설명서는 버전 … 기준" 줄과 build.mjs 의 기준 버전을 확인한다.
