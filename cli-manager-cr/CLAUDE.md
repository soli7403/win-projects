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
| [docs/host-settings/](./docs/host-settings/README.md) | 빌드 PC 에 설치한 도구(NSIS 등), 라이선스 서명 비밀키 위치, 점검·설치 스크립트 |
| [.claude/known-issues.md](./.claude/known-issues.md) | 알려진 지뢰 |

## 3. 작업 순서

1. **명세 단계** (명세 작성자): `spec/` 문서 작성 → 완료 기준은 `spec/README.md` 의 체크리스트
2. **구현 단계** (구현자, 새 세션): 명세만 보고 구현. 모호하면 `QUESTIONS.md`
3. **대조 단계**: 구현을 기존 앱과 화면 단위로 나란히 실행해 차이를 **관찰**하고, 차이는 명세 수정 → 구현 수정 순으로 반영 (구현자가 기존 코드를 열어 맞추지 않는다)

## 4. 빠른 명령

```bash
go build ./...
go test -race ./...
(cd frontend && npx tsc --noEmit)
wails build    # 프런트(npm run build)까지 함께 빌드 → build/bin/dariterm.exe (wails.json frontend:build)
make installer # Windows 설치 파일 → build/bin/DariTerm-amd64-installer.exe (NSIS 필요, D76, docs/host-settings/01)
make msix VERSION=x.y.z  # 버전 올리고 Microsoft Store 패키지 → build/bin/DariTerm-<버전>.0.msix → Partner Center 에 수동 업로드 (D81·D82, docs/store/README.md §3)
# make store-publish VERSION=x.y.z  # (보류) msstore CLI 자동 제출 — Entra 결제 계정이 필요해 쓰지 않음 (D82)
# 버전은 wails.json info.productVersion 한 곳(D82). 이 PC 의 make 는 mingw32-make
```
