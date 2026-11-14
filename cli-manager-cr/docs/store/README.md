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
make msix     # → build\bin\DariTerm-0.1.0.0.msix
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
4. **패키지**: `build\bin\DariTerm-0.1.0.0.msix` 를 끌어다 놓는다.
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

## 3. 다음 버전 올리기

1. `wails.json` 의 `info.productVersion` 을 올린다 (예: 0.1.1). 앱 안 버전(`G_cszAppVersion`)도 맞춘다(D80 메모).
2. `make msix` → 새 `DariTerm-0.1.1.0.msix`
3. Partner Center > 새 제출 > 패키지만 바꿔서 제출. 스토어판은 스토어가 자동 업데이트한다(앱의 업데이트 확인은 꺼져 있다, D80).

## 4. 알아둘 것

- **개인정보 처리방침**: 앱이 비밀번호·키를 PC 안에(암호화해) 저장하고 사용자가 정한 서버에 접속하므로, 처리방침 URL 을 넣는 것이 안전하다. 개발자에게 아무것도 보내지 않는다는 내용이다. 공개 주소가 필요하다(예: 공개 GitHub 저장소·GitHub Pages·Notion 공개 페이지). 공개는 사용자가 직접 한다.
- **WebView2**: 화면은 Microsoft Edge WebView2 런타임으로 그린다. Windows 11 과 최신 Windows 10 에는 들어 있다. NSIS 설치판과 달리 MSIX 는 WebView2 를 따로 설치하지 않는다.
- **데이터 위치**: 스토어판이 `%APPDATA%\DariTerm` 에 새로 만드는 파일은 앱 전용 위치로 돌려지고, 제거하면 함께 지워진다. 다만 NSIS 설치판 데이터가 이미 있는 PC 에서는 그 기존 파일을 제자리에서 읽고 고친다(Windows 10 1903+ 동작). 한 PC 에 두 판을 함께 두지 않는 것이 깔끔하다. 옮길 때는 앱의 백업/복원을 쓴다.
- **두 배포판 함께**: NSIS 설치 파일(`make installer`)은 그대로 둔다. 스토어 밖에서 받는 사람용이다.

## 5. 제출 이력

| 날짜 | 버전 | 내용 | 결과 |
|---|---|---|---|
| 2026-11-14 | 0.1.0.0 | 첫 제출(Partner Center 수동). 무료, 전 세계, 개발자 도구. 스토어 목록 영어·한국어(나머지 8개 언어는 목록에서 뺌 — 앱 화면은 10개 언어 그대로), 인증 통과 즉시 게시. runFullTrust 사유 447자(칸 제한 500자) | 인증 대기 |
