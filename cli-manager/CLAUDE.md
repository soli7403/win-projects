# cli-manager (임시 제품명 `TermGo`) — 작업 진입점 (Claude / 개발자용)

BrightTerm v1.1.0(MIT) 을 기반으로 Electron 터미널 앱을 **Go 백엔드 + Wails 프런트엔드**로 포팅하고, 이후 기능을 독자 구현하는 프로젝트. 대상 플랫폼은 Windows 우선.

> 공통 규칙(WIP, Known Issues, Fix Protocol, 커밋, 코딩 표준)은 [../CLAUDE.md](../CLAUDE.md). 알려진 지뢰 [.claude/known-issues.md](./.claude/known-issues.md).

## ⚠️ 라이선스 필수 규칙 (작업 전 반드시 확인)

- `Yeosup/BrightTerm` 의 **v1.1.0 태그만** 참조한다. **v1.1.1 이후 태그·main 소스는 열람·참조·복사·번역 금지** (별도 Source-Available 라이선스).
- v1.1.0 이후 기능은 [docs/porting-brief.md](./docs/porting-brief.md) §3 의 동작 설명만 근거로 독자 설계한다.
- 저작권 표기는 `Copyright (c) 2026 kai` 만 둔다 (사용자 결정 D14 — 원저작자 `Dany Kim` 표기는 뺐다. v1.1.0 파생 코드가 남아 있어 MIT 조건과 충돌할 수 있다는 점을 사용자에게 고지했고, 배포 전 전문가 확인 항목). 업스트림 원문 라이선스는 `third_party/brightterm-v1.1.0/LICENSE` 에 그대로 둔다.
- BrightTerm 이름·로고 사용 금지. **광고 배너 기능·이미지·캐시는 이식하지 않는다.**

---

## 1. 산출물

| 바이너리 | 역할 |
|---|---|
| **termgo** (Wails 앱, `wails build` → `build/bin/termgo.exe`) | 터미널 앱 (SSH/SFTP/텔넷/시리얼/로컬 PTY). 진입점 [main.go](./main.go) / [app.go](./app.go), IPC 라우터 [internal/ipc](./internal/ipc) |

---

## 2. 문서 인덱스

| 파일 | 용도 |
|---|---|
| [docs/README.md](./docs/README.md) | 운영 안내 |
| **[docs/porting-brief.md](./docs/porting-brief.md)** | **포팅 작업 지침 (원문) — 라이선스 규칙·아키텍처·기능 명세 3.1~3.13·작업 순서** |
| [docs/plan.md](./docs/plan.md) | 설계 원본 |
| [docs/decisions.md](./docs/decisions.md) | 결정 기록 |
| **[../docs/11_coding-standard.md](../docs/11_coding-standard.md)** | **Go 코딩 표준** |
| [../docs/12_engineering-rules.md](../docs/12_engineering-rules.md) | 구현 공통 규칙 |
| **[.claude/known-issues.md](./.claude/known-issues.md)** | **알려진 지뢰** |
| [../.claude/wip/README.md](../.claude/wip/README.md) | WIP / Resume 체크포인트 (이 프로젝트 파일은 `.claude/wip/`) |

---

## 3. 모듈별 전문 Subagent

| 작업 영역 | Agent |
|---|---|
| 변경 파일 결함 리뷰 (범용) | `code-defect-reviewer` (hook 자동 호출) |

> 새 agent 는 [../.claude/templates/agent-expert.md](../.claude/templates/agent-expert.md) 복사해서 작성 (예정: ssh / vault / ui-bridge).

---

## 4. 이 프로젝트의 규칙

- 데이터 파일(서버 목록·볼트·설정)은 v1.1.0 형식 호환이 기본. 포기해야 하면 사용자에게 먼저 확인.
- 렌더러는 `window.bt` 호환 어댑터로 Wails 에 연결하고 UI 코드는 가능한 한 손대지 않는다.
- 터미널 출력은 Go 쪽에서 10~16ms 단위로 묶어 전송.
- 작업 순서는 porting-brief §4 를 따른다.

## 5. 빠른 명령

```bash
go build ./...
go test -race ./...
wails dev      # 개발 (PATH 에 ~/go/bin)
wails build    # build/bin/termgo.exe
(cd frontend && npx tsc --noEmit)
```
