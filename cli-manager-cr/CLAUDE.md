# cli-manager-cr (제품명 `DariTerm`, 옛 임시 이름 TermGo) — clean-room 재작성 프로젝트

> 공통 규칙(WIP, Known Issues, Fix Protocol, 커밋, 코딩 표준)은 [../CLAUDE.md](../CLAUDE.md).
> 이 프로젝트는 기존 `cli-manager`(BrightTerm v1.1.0 이식물)와 **코드를 공유하지 않는** 독자 구현이다.

> **📌 작업 위치 (사용자 지시, 2026-11-12, 결정 D54)**: 앞으로 **소스 수정과 빌드는 무조건 이 폴더(`cli-manager-cr`)에서** 한다. `cli-manager` 는 고치지 않는다. 실행 파일은 `build/bin/dariterm.exe` (`wails build`). 원본을 읽은 세션이 이 지시로 구현하는 경우는 [docs/clean-room-log.md](./docs/clean-room-log.md) 에 남긴다.

## ⛔ Clean-room 규칙 (이 폴더에서 일하는 모든 세션·에이전트에 적용, 최우선)

목적: 라이선스·소유권 정리. 구현자가 원본 코드의 표현을 보지 않은 채 **명세(`spec/`)만 보고** 구현한다.

**구현자(이 폴더에서 시작한 세션)는 다음을 열람·검색·복사·참조하지 않는다.**
- `../cli-manager/**` 전체 (코드, `third_party/`, `docs/porting-brief.md`, 빌드 산출물 `frontend/dist`, `build/bin` 포함). **이 폴더는 2026-11-13 삭제됐다(D65). git 이력에서 열람·복원(`git show`·`git checkout` 등)하는 것도 같은 금지 대상이다.**
- BrightTerm 저장소(`Yeosup/BrightTerm`)의 **모든 태그·브랜치** (v1.1.0 포함)
- `spec/` 에 없는 내용을 기존 코드에서 "확인"하는 행위 — 명세가 모호하면 **추측하지 말고** `spec/QUESTIONS.md` 에 질문을 적고 멈춘다. 답은 명세 작성자가 `spec/` 에 반영한다.

**허용**: `spec/**`, 이 폴더의 코드, 공개 라이브러리 문서(xterm.js, Go 표준·x/crypto, Wails 등), 공통 규칙 `../CLAUDE.md`·`../docs/`.

**명세 작성자(기존 코드를 읽은 사람/세션)는**
- `spec/` 에 **동작·데이터 형식·수치·문구**만 쓴다. 코드 조각, 함수·변수·CSS 클래스·파일 구조 이름, 알고리즘의 구현 표현을 옮겨 적지 않는다.
- 구현 세션을 겸하지 않는다. (이미 원본을 읽었으므로 이 폴더의 코드를 쓰지 않는다. 명세 수정·질문 답변만 한다.)
- 명세의 각 문서 머리에 작성 방식(관찰/기존 코드 기반)과 일자를 적는다.

**강제 수단**: [.claude/settings.json](./.claude/settings.json) 의 `permissions.deny` 로 위 경로 읽기를 막는다 (최선의 방어일 뿐, 우회될 수 있으니 규칙을 지킨다). 위반(실수로 열람 포함)을 발견하면 `docs/clean-room-log.md` 에 기록하고 해당 부분은 명세 작성자가 다시 쓰게 한다.

> 법적 효력은 보장되지 않는다. 실제 배포·고지 정리 전에 전문가 확인을 받는다. `LICENSE` 는 **YangHun Lee (kai) 단독 독점 라이선스(All rights reserved, 소스 비공개)**이고(결정 D64, 저작권자 표기는 여권 영문 이름 + 평소 이름. 설치 정보의 게시자는 DariTerm, D76), 사용 허락은 **개인 무료 · 업무용 사용자별 유료**(문의 soli97403@gmail.com)다(D73·D74). 화면·안내 문구는 10개 언어이고 문구는 `locales/<언어>.json` 에만 둔다(D75, spec/behavior/i18n.md), 참고한 BrightTerm v1.1.0(MIT)과 오픈소스 구성 요소의 고지는 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)와 정보 화면에 둔다(D63·D64). 의존성을 바꾸면 `make notices` 로 고지를 다시 만든다. 이 프로젝트에는 원본 코드가 없다는 전제다. clean-room 위반이 발견되면 그 부분을 다시 쓰기 전까지 이 전제가 깨진 것으로 보고 `docs/clean-room-log.md` 에 기록한다.

---

## 1. 목표

기존 앱과 **같은 UI flow·설정·외형·동작**의 터미널 앱을 독자 코드로 만든다. 대상 플랫폼은 Windows 우선, 코드는 OS 중립 ([../CLAUDE.md](../CLAUDE.md) §8).

| 층 | 방침 |
|---|---|
| 프런트엔드 | **React 로 새로 작성**(Wails 웹뷰), 터미널은 xterm.js. 외형은 `spec/ui/` 의 수치·색·배치 명세를 따른다 |
| 백엔드 | Go (`internal/…`). 데이터 파일(store.json, vault.json) 형식은 `spec/data/` 의 호환 명세를 따른다 |
| IPC | `spec/ipc.md` 의 채널·페이로드 계약 |

## 2. 문서 인덱스

| 파일 | 용도 |
|---|---|
| **[spec/README.md](./spec/README.md)** | **명세 인덱스 — 구현자의 유일한 기준** |
| [spec/QUESTIONS.md](./spec/QUESTIONS.md) | 구현자 → 명세 작성자 질문 큐 |
| [docs/clean-room-protocol.md](./docs/clean-room-protocol.md) | 역할·절차·체크리스트 |
| [docs/implementer-start.md](./docs/implementer-start.md) | 구현자 새 세션 시작 절차와 첫 메시지 |
| [docs/decisions.md](./docs/decisions.md) | 결정 기록 |
| [CHANGELOG.md](./CHANGELOG.md) | 버전 이력 (배포할 때마다 한 절) |
| [docs/releases/](./docs/releases/1.2.0.md) | 버전별 배포 안내(배포 파일·SHA256·설치·업그레이드·배포 전 확인). 최신 [1.2.0](./docs/releases/1.2.0.md) (D161, 2026-11-28, MSIX + 설치 파일, 같은 날 SSH 키 D162·시리얼 HEX D163·XMODEM D164 를 넣어 다시 만듦 D166, 스토어 제출·릴리스 저장소 게시는 사용자), 이전 [1.1.0](./docs/releases/1.1.0.md) (D123-1, 2026-11-25 에 11-25 변경을 합쳐 다시 만듦 D133, MSIX 만, 2026-11-25 스토어 제출·인증 대기), 이전 [1.0.0](./docs/releases/1.0.0.md) (D106-1) |
| [docs/import-plans/](./docs/import-plans/) | 수정 계획서(원인·코드·테스트). 2026-11-16 의 3건은 구현 완료(D84~D90), 2026-11-17 의 2건도 구현 완료(D92~D94, 화면 보호기 오인 판정 안 C 만 보류), 2026-11-18 의 6건(터미널 찾기 수정 4건 — 버튼·강조 색 D96·스크롤 막대 표시·남는 표시, 잠금 화면 Caps Lock 안내 D97, 서버 timeout 뒤 자동 재연결 D98, 출력 기록 설정을 열린 세션에 곧바로 적용 D99, 로그 내려 쓰기 주기 설정 D100)은 1.0.0 에 빠져 있어 2026-11-23-import-1118-features.md 한 문서로 넣었다(설계·코드·패치 포함), 2026-11-19·20 의 3건도 구현 완료(D101~D105, 앱 수동 검증은 각 계획서의 수동 표), 2026-11-23 의 13건(PPK D106-2, Pageant D107-2, 시리얼 흐름 제어 D108, SOCKS 동적 포워딩 D109, 프록시 경유 D110, 에이전트 전달 D111, 로그 줄 시각 D112, 키워드 강조 D113, ZMODEM D114, 스토어 업데이트 D115, 바둑판 삭제 D116, 열 위치 선택 삭제 D117, 인코딩 EUC-JP·GBK·Big5 D118)도 코드 반영·자동 검증 완료(버전 1.1.0 에 포함, 앱 수동 검증은 각 계획서 §H). 이어서 KI-011 수정(D120-1)과 2026-11-23-1~4(+ 버튼 그룹 D119, 로그 구역·ANSI 저장 D120-2, 최대화 아이콘 D121, 창 수 배지 D122)도 반영. 2026-11-25 의 8건도 반영(ZMODEM 실제 lrzsz 검증·취소 뒤처리 D123-2·응답 시한·tmux 안내 D125, 찾기 클립보드는 D120-1 로 이미 반영, 단축키 설정 D126, 끊김 까닭 D127, Xshell 가져오기 D128·포워딩 D129, 왼쪽 열 너비 D130, 서버 목록 구역 접기 D131·구분선 D132 — 앱 수동 검증은 ZMODEM·Xshell(1·5·6) 완료, 나머지는 각 계획서 §H). 2026-11-26 의 3건(스토어 확인 PowerShell 을 패키지 안에서 D134, Xshell `.xsh` 파일 종류, 공개 릴리스 저장소로 업데이트 확인 D135)과 2026-11-27 의 20건(포워딩 정보·연결 수·전송량 D136·D137·D142·D143·D151, 명령 변수 D138, 로그인 스크립트 D139·보관함 값 D148, 출력 트리거 D140·Windows 알림 D145·서버별 D149, 매크로 D141·일괄 입력 D146·단축키 D147, gcc `-race` D144, 가져오기 로그인 스크립트 감지 D150, 프록시 명령 D152, ProxyJump 체인 D153, 화면 보호기 잠금 판정 D154, 도움말 창·정렬·FAQ·그림·크기 조절 D155~D159, 세션 전송량 D160)도 반영(앱 수동 검증은 각 계획서 §H). **도움말 본문 `locales/help/{ko,en}.md` 는 계획서에 전체가 없어 이 저장소에서 새로 썼다(D155)** |
| [docs/host-settings/](./docs/host-settings/README.md) | 빌드 PC 에 설치한 도구(NSIS 등), 라이선스 서명 비밀키 위치, 점검·설치 스크립트 |
| [.claude/known-issues.md](./.claude/known-issues.md) | 알려진 지뢰 |

## 3. 작업 순서

1. **명세 단계** (명세 작성자): `spec/` 문서 작성 → 완료 기준은 `spec/README.md` 의 체크리스트
2. **구현 단계** (구현자, 새 세션): 명세만 보고 구현. 모호하면 `QUESTIONS.md`
3. **대조 단계**: 구현을 기존 앱과 화면 단위로 나란히 실행해 차이를 **관찰**하고, 차이는 명세 수정 → 구현 수정 순으로 반영 (구현자가 기존 코드를 열어 맞추지 않는다)
4. **새 기능·화면을 바꿀 때 도움말도 함께 고친다**: 본문 `locales/help/{ko,en}.md`(id 는 언어마다 같게), 화면이 바뀌면 frontend 에서 `npm run help:capture` 로 그림을 다시 찍는다(spec/ui/help.md, D155·D158).

## 4. 빠른 명령

```bash
go build ./...
go test -race ./...
(cd frontend && npx tsc --noEmit)
(cd frontend && npm test)   # vitest 5. npx vitest 직접 실행은 소문자 드라이브(e:\)에서 실패 — run-vitest.mjs 가 맞춘다 (D124-1)
wails build    # 프런트(npm run build)까지 함께 빌드 → build/bin/dariterm.exe (wails.json frontend:build)
make installer VERSION=x.y.z  # Windows 설치 파일 → build/dist/DariTerm-<버전>-amd64-installer.exe (NSIS 필요, D76, docs/host-settings/01). VERSION 생략 시 지금 버전
make msix VERSION=x.y.z  # 버전 올리고 Microsoft Store 패키지 → build/dist/DariTerm-<버전>.0.msix → Partner Center 에 수동 업로드 (D81·D82, docs/store/README.md §3)
# make store-publish VERSION=x.y.z  # (보류) msstore CLI 자동 제출 — Entra 결제 계정이 필요해 쓰지 않음 (D82)
# 버전은 wails.json info.productVersion 한 곳(D82). 이 PC 의 make 는 mingw32-make
```
