# Microsoft Store 제출 안내 (DariTerm)

> 2026-11-14 작성. 결정 D81. 패키지 빌드는 [../host-settings/03_msix.md](../host-settings/03_msix.md).

## 0. 이미 정해진 값 (Partner Center > 제품 ID)

| 항목 | 값 | 어디에 쓰나 |
|---|---|---|
| Package/Identity/Name | `YangHunLee.DariTerm` | `build/windows/msix/AppxManifest.xml` |
| Package/Identity/Publisher | `CN=0AEF544B-0B89-41A6-861B-39FF4DB0031F` | 같음 |
| PublisherDisplayName | `YangHun Lee` | 같음 |
| 패키지 패밀리 이름(PFN) | `YangHunLee.DariTerm_bmtfree9a9kzr` | 참고 |
| 개인정보 처리방침 | https://gist.github.com/soli7403/b6489cedf5daa8d2cef55a0bcfa9f057 | 속성 > 개인정보 처리방침 URL |
| Store ID | `9N8WBPMRNBWV` | 스토어 링크 `https://apps.microsoft.com/detail/9N8WBPMRNBWV` |

이 값을 바꾸면 업로드가 거부된다.

## 1. 패키지 만들기

```powershell
make msix     # → build\dist\DariTerm-0.1.0.0.msix
```

## 2. Partner Center 제출 순서

[Partner Center](https://partner.microsoft.com/dashboard/apps-and-games/overview) > DariTerm > **제출 시작**. 각 항목을 채운다.

1. **가격 및 가용성**
   - 시장: 모든 시장 (또는 원하는 나라만)
   - 가격: **무료** — 개인 사용은 무료이고, 업무용은 앱 안에서 라이선스 키를 넣는 방식이다(D73·D78). 데스크톱 앱(Win32)은 스토어 결제 대신 자체 판매를 써도 된다.
   - 공개 시점: 인증 통과 후 바로 / 수동 공개 중에 고른다 (처음에는 "수동 공개"를 권장 — 통과 후 직접 공개 버튼을 누른다)
2. **속성**
   - 범주: **개발자 도구**(Developer tools)
   - 개인정보 처리방침 URL: **https://gist.github.com/soli7403/b6489cedf5daa8d2cef55a0bcfa9f057** (공개 GitHub Gist, 2026-11-14 게시. 원문 [privacy-policy.md](./privacy-policy.md) — 고치면 Gist 도 같이 고친다. 주소는 그대로 유지된다)
   - 웹사이트·지원 연락처: `soli97403@gmail.com`
   - 시스템 요구사항: 키보드·마우스 권장
3. **연령 등급**: 설문(IARC)에 답한다. 폭력·도박·사용자 간 대화 기능이 없으므로 "아니요"로 답하면 보통 3+ 가 나온다. 터미널이 인터넷에 접속하므로 "인터넷 접속/웹 콘텐츠" 질문에는 사실대로 답한다.
4. **패키지**: `build\dist\DariTerm-0.1.0.0.msix` 를 끌어다 놓는다.
   - 장치 패밀리: **Windows 10/11 데스크톱**만 체크
   - **제한된 기능(runFullTrust) 사유** 칸이 나오면 아래 문장을 그대로 넣는다:

     > DariTerm is a Win32 desktop terminal (SSH, Telnet, serial, local shell) packaged as MSIX. It needs runFullTrust to start local shells (cmd, PowerShell, WSL) via ConPTY, open serial COM ports, connect over TCP to user-configured SSH/Telnet servers, and read/write user-chosen files for SFTP and backup. These are not possible in AppContainer. It installs no services or drivers, needs no admin rights, and keeps its data only in the user's AppData.

     (입력 칸은 **500자 제한** — 위 글은 447자, 한 문단으로 넣는다)

5. **스토어 등록 정보**(언어별): [listing.md](./listing.md) 의 영어·한국어 초안을 붙여 넣는다. 영어는 꼭 넣는다.
   - 스크린샷: [screenshots/en/](./screenshots/en/) (영어 목록), [screenshots/ko/](./screenshots/ko/) (한국어 목록) — 1920×1080 PNG 7장. 아래 순서로 올린다(첫 장이 대표 그림).
     1. `01-split-broadcast` 화면 분할 + 동시 입력 · 2. `02-sftp` 파일 창 · 3. `03-quick-connect` 빠른 접속 · 4. `04-servers` 서버 목록·시작 화면 · 5. `05-add-server` 서버 추가 · 6. `06-import` 가져오기 · 7. `07-settings` 설정
     - 다시 찍기: `(cd frontend && npx vite --port 5179)` 를 띄우고 `node tools/store/screenshots.mjs http://localhost:5179/ docs/store/screenshots/en en` (ko 도 같게). 가짜 데이터라 실제 서버 정보가 들어가지 않는다.
   - Microsoft Store 로고(선택이지만 권장, 영어·한국어 페이지에 같은 그림): [logos/](./logos/) — 다시 만들기 `node tools/store/logos.mjs`
     - 9:16 포스터 아트 ← `poster-1440x2160.png` · 1:1 박스 아트 ← `box-1080x1080.png` · 1:1 앱 타일 아이콘 300×300 ← `tile-300x300.png`
     - 150×150, 71×71 은 비워 둔다(패키지 안 로고를 쓴다)
   - 트레일러 및 추가 자산: 트레일러는 없음. **16:9 슈퍼히어로 아트** ← `logos/hero-1920x1080.png` (규칙상 제품 이름 글자 없이 앱 화면만). Xbox 이미지 3종은 비워 둔다(Xbox 미지원)
6. **제출 옵션**: 인증 담당자 메모(Notes for certification)에 아래를 넣는다:

   > No account or sign-in is required. On first launch: choose a language, choose "Personal use", then set an app lock password (any value) and confirm the recovery code screen. To test without a server, open "Local terminal" (Ctrl+Alt+T), which starts a local shell on this PC. SSH, Telnet and serial connections need the tester's own server or device. A business license key is optional; without one the app runs fully in Personal mode.

   (Partner Center 화면에서는 "제출 옵션 > 인증에 대한 참고 사항"의 **추가 테스트 정보** 링크로 들어가 넣는다. 게시 보류는 "지금 게시를 선택할 때까지 게시하지 않음"을 권장)

7. **스토어에 제출** → 인증은 보통 1~3 영업일. 결과는 메일과 Partner Center 에 온다.

## 3. 다음 버전 올리기 (D82)

버전은 **`wails.json` 의 `info.productVersion` 한 곳**에서만 정한다. 앱 안 버전·설치 파일·MSIX 가 모두 이 값을 쓴다(main.go 가 시작할 때 읽음).

### 3.1 지금 방식: 수동 업데이트 (권장, 2026-11-14 결정)

1. 패키지 만들기(버전도 함께 올림):
   ```powershell
   # 저장소 cli-manager-cr 에서 (이 PC 의 make 는 mingw32-make)
   mingw32-make msix VERSION=0.1.1
   # 또는  powershell -ExecutionPolicy Bypass -File tools\msix.ps1 -Version 0.1.1
   ```
   → `build\dist\DariTerm-0.1.1.0.msix`. 버전이 지금보다 작거나 같으면 만들기 전에 멈춘다.
2. Partner Center > DariTerm > **제출 업데이트** → 이전 제출이 복사된 새 초안.
3. **패키지**: 새 msix 를 끌어다 놓고, 이전 msix 는 **제거**. (선택) Store 목록 > "이 버전의 새로운 기능".
4. **인증을 위해 제출**. 이전 제출이 인증 중이면 새 제출을 만들 수 없으니 통과를 기다린다.
5. [CHANGELOG.md](../../CHANGELOG.md) 에 버전 절을 쓰고, [listing.md](./listing.md) 의 "이 버전의 새로운 기능"(영어·한국어)을 바꿔 3번에 붙여 넣는다.
6. `wails.json` 버전 변경을 커밋하고 §5 제출 이력에 한 줄 적는다.

### 3.2 보류: CLI 자동 제출 (msstore)

`tools/store-publish.ps1`(`make store-publish VERSION=x.y.z`)은 만들어 두었지만 **쓰지 않는다**. msstore CLI 는 Microsoft Entra 테넌트의 앱 자격 증명이 필요한데, Partner Center 에서 테넌트를 만들려 하면 **Microsoft 결제 계정(billing account) 생성 + Microsoft Customer Agreement 동의 + 결제 설정** 화면으로 이어진다(2026-11-14 확인). 혼자 개발하는 앱의 드문 업데이트에는 수동(3.1)으로 충분해서 보류했다. 업데이트가 잦아지거나 회사 테넌트가 생기면 아래로 켠다.

#### 켤 때: 자격 증명 준비 (사용자가 직접)

1. Partner Center > ⚙ **계정 설정** > **테넌트**: Microsoft Entra ID 를 연결하거나 **만들기**(결제 계정 화면으로 이어짐 — 위 참고).
2. 계정 설정 > **사용자 관리** > **Microsoft Entra 응용 프로그램** > **새 Microsoft Entra 응용 프로그램 만들기**(이름 예: `DariTerm Store CLI`) → 역할 **관리자(Manager)**.
3. 만든 응용 프로그램을 눌러 **테넌트 ID**, **클라이언트 ID** 를 적고, **새 키 추가** → **키(클라이언트 비밀)** 를 바로 적는다(이 화면을 떠나면 다시 볼 수 없다).
4. 계정 설정 > 조직 프로필 > **법률 정보**(개발자 탭)에서 **판매자 ID(Seller ID)** 를 적는다.
5. **본인 PowerShell 창**에서 한 번 실행한다(값은 저장소·대화·문서에 남기지 않는다. CLI 가 이 PC 의 사용자 프로필에 보관한다. 지우기: `msstore reconfigure --reset`):
   ```powershell
   msstore reconfigure --tenantId <테넌트 ID> --sellerId <판매자 ID> --clientId <클라이언트 ID> --clientSecret <키>
   msstore apps list      # DariTerm(9N8WBPMRNBWV) 이 보이면 성공
   ```
6. 키에는 만료일이 있다(보통 1~2년). 만료되면 3번에서 새 키를 만들고 5번을 다시 한다.

#### 켠 뒤: 업데이트 제출

```powershell
# 저장소 cli-manager-cr 에서 (이 PC 의 make 는 mingw32-make)
mingw32-make store-publish VERSION=0.1.1
# 또는
powershell -ExecutionPolicy Bypass -File tools\store-publish.ps1 -Version 0.1.1
```

스크립트가 하는 일: 버전 확인(지금보다 커야 함) → `wails.json` 버전 변경 → `tools\msix.ps1`(앱 빌드 + MSIX) → `msstore publish`(업로드 + 인증 요청) → 제출 상태 출력.

| 옵션 | 뜻 |
|---|---|
| `-Version x.y.z` | 새 버전. 비우면 지금 버전 그대로 |
| `-NoCommit` | 업로드만 하고 초안으로 둔다 → Partner Center 에서 확인 후 직접 제출 (처음 몇 번은 이걸 권장) |
| `-Rollout 20` | 단계적 배포(20% 사용자에게 먼저). 이후 `msstore submission rollout update 9N8WBPMRNBWV 100` 또는 `... rollout finalize 9N8WBPMRNBWV` |
| `-Status` | 제출 상태만 보기 (`msstore submission status 9N8WBPMRNBWV` 와 같다) |

제출 뒤:
- `wails.json` 버전 변경을 커밋하고, 아래 §5 제출 이력에 한 줄 적는다.
- 스토어판은 스토어가 자동 업데이트한다(앱의 업데이트 확인은 꺼져 있다, D80).

주의:
- **`msstore publish` 는 마지막 게시본으로 새 초안을 만든다** — Partner Center 에서 만들어 둔 초안이 있으면 지워진다. 한 제출은 CLI 나 웹 한쪽으로만 다룬다(웹에서 고친 제출은 API 로 이어갈 수 없다).
- 이 계정은 **시장 그룹 가격 방식**이라 CLI 가 가격을 못 읽고 멈출 수 있어 스크립트가 `--priceId Free` 를 넣는다. 유료로 바꾸면 이 줄을 고친다.
- 소개 글·스크린샷 변경은 지금처럼 Partner Center 에서 한다(CLI 로도 가능하지만 `submission get/update` JSON 을 다뤄야 한다).

## 4. 알아둘 것

- **개인정보 처리방침**: 앱이 비밀번호·키를 PC 안에(암호화해) 저장하고 사용자가 정한 서버에 접속하므로, 처리방침 URL 을 넣는 것이 안전하다. 개발자에게 아무것도 보내지 않는다는 내용이다. 공개 주소가 필요하다(예: 공개 GitHub 저장소·GitHub Pages·Notion 공개 페이지). 공개는 사용자가 직접 한다.
- **WebView2**: 화면은 Microsoft Edge WebView2 런타임으로 그린다. Windows 11 과 최신 Windows 10 에는 들어 있다. NSIS 설치판과 달리 MSIX 는 WebView2 를 따로 설치하지 않는다.
- **데이터 위치**: 스토어판이 `%APPDATA%\DariTerm` 에 새로 만드는 파일은 앱 전용 위치로 돌려지고, 제거하면 함께 지워진다. 다만 NSIS 설치판 데이터가 이미 있는 PC 에서는 그 기존 파일을 제자리에서 읽고 고친다(Windows 10 1903+ 동작). 한 PC 에 두 판을 함께 두지 않는 것이 깔끔하다. 옮길 때는 앱의 백업/복원을 쓴다.
- **두 배포판 함께**: NSIS 설치 파일(`make installer`)은 그대로 둔다. 스토어 밖에서 받는 사람용이다.

## 5. 제출 이력

| 날짜 | 버전 | 내용 | 결과 |
|---|---|---|---|
| 2026-11-21 | 1.0.0.0 | 첫 정식 버전(D106-1). 업데이트 패키지 `build\dist\DariTerm-1.0.0.0.msix`(`make msix VERSION=1.0.0`, SHA256 `9D57F0D4B6D5B2BE1B5685B16A05A4B117FF1941EE346D1B4398F9D7A3EB481E`). 스토어 밖 설치 파일 `DariTerm-1.0.0-amd64-installer.exe`(SHA256 `449BFE886D3D2CC75CFC9183356E596DFD53B1EE4B4A699698110CE9874EDD8F`) 도 함께. 변경은 [CHANGELOG.md](../../CHANGELOG.md) 1.0.0, "이 버전의 새로운 기능" 문구는 [listing.md](./listing.md), 배포 안내는 [../releases/1.0.0.md](../releases/1.0.0.md). 0.2.0.0·0.3.0.0 을 아직 제출하지 않았으면 초안의 이전 패키지를 빼고 이것만 올린다 | 2026-11-21 제출(Submission 3, 패키지 1.0.0.0 만) — 인증 끝남(1.1.0 제출 전에) |
| 2026-11-28 | 1.2.0.0 | 기능 버전(D161, 같은 날 SSH 키·시리얼 도구를 넣어 다시 만듦 D166): 로그인 스크립트·출력 트리거·매크로·프록시 명령·도움말 창·SSH 키·시리얼 HEX·XMODEM 등(D134~D165). 업데이트 패키지 `build\dist\DariTerm-1.2.0.0.msix`(`make msix VERSION=1.2.0`, 8,146,439 바이트, SHA256 `CFF86FBECA2C0F9B7883146D087389560AF7691C2241A415B6B99723871B4A8E`). 스토어 밖 설치 파일 `DariTerm-1.2.0-amd64-installer.exe`(SHA256 `135565D3D79EA95744ABD89D7CB3CA18AF034D2266B94EE69D3A42FDD211BC8A`)도 함께. 변경은 [CHANGELOG.md](../../CHANGELOG.md) 1.2.0, 배포 안내 [releases/1.2.0.md](../releases/1.2.0.md), "이 버전의 새로운 기능"·설명·기능 목록은 [listing.md](./listing.md). 1.1.0.0 인증이 끝난 뒤 이전 패키지를 빼고 이것만 올린다 | 패키지만 만듦, 제출 차례(1.1.0 인증 끝남) |
| 2026-11-25 | 1.1.0.0 | 기능 버전 다시 만듦(D133): 2026-11-23 패키지를 제출하기 전에 2026-11-25 변경(단축키 설정·Xshell 가져오기·ZMODEM 수정 등)을 합쳤다. 업데이트 패키지 `build\dist\DariTerm-1.1.0.0.msix`(`make msix`, 6,887,896 바이트, SHA256 `A73534B533A3D2AC5571B19DCC03BFF4ACFA427CCBC7EB55A2F71C98E3AE664C`). 변경은 [CHANGELOG.md](../../CHANGELOG.md) 1.1.0, 배포 안내 [releases/1.1.0.md](../releases/1.1.0.md), "이 버전의 새로운 기능" 문구는 [listing.md](./listing.md) | 2026-11-25 제출(Partner Center 수동) — 인증 끝남(2026-11-28 사용자 확인) |
| 2026-11-23 | 1.1.0.0 | (제출 안 함 — 2026-11-25 에 다시 만든 패키지로 대체, D133) 기능 버전(D123-1). 업데이트 패키지 `build\dist\DariTerm-1.1.0.0.msix`(`make msix VERSION=1.1.0`, SHA256 `6C234F9EEAAD3135B35B8DF064056DF3012A314BE06B1A099F459158CB60F8AD`). 설치 파일은 만들지 않았다. 변경은 [CHANGELOG.md](../../CHANGELOG.md) 1.1.0, "이 버전의 새로운 기능" 문구는 [listing.md](./listing.md) 1.1.0, 배포 안내는 [../releases/1.1.0.md](../releases/1.1.0.md). 1.0.0.0 제출의 인증이 끝난 뒤 이전 패키지를 빼고 이것만 올린다 | 패키지만 만듦, 아직 제출 안 함 |
| 2026-11-17 | 0.3.0.0 | 업데이트 패키지 `build\dist\DariTerm-0.3.0.0.msix`(`make msix VERSION=0.3.0`, SHA256 `4786A1814B679D168F0AF07BBA8F6004A1A785FFF6A91FF06212FE8B5C8D7825`). 스토어 밖 설치 파일 `DariTerm-0.3.0-amd64-installer.exe` 도 함께. 변경은 [CHANGELOG.md](../../CHANGELOG.md) 0.3.0, "이 버전의 새로운 기능" 문구는 [listing.md](./listing.md). 0.2.0.0 을 아직 제출하지 않았으면 그 초안의 0.2.0.0 패키지를 빼고 이것만 올린다 | 따로 제출하지 않음 — 변경은 1.0.0.0 에 포함되어 나감 |
| 2026-11-16 | 0.2.0.0 | 업데이트 패키지 `build\dist\DariTerm-0.2.0.0.msix`(`make msix VERSION=0.2.0`). 변경은 [CHANGELOG.md](../../CHANGELOG.md) 0.2.0, "이 버전의 새로운 기능" 문구는 [listing.md](./listing.md) | 인증 통과·게시됨 (2026-11-21 사용자 확인) |
| 2026-11-14 | 0.1.0.0 | 첫 제출(Partner Center 수동). 무료, 전 세계, 개발자 도구. 스토어 목록 영어·한국어(나머지 8개 언어는 목록에서 뺌 — 앱 화면은 10개 언어 그대로), 인증 통과 즉시 게시. runFullTrust 사유 447자(칸 제한 500자) | 인증 통과·게시됨 (2026-11-16 Partner Center 에서 확인) |
