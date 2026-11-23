# win-projects — 공통 작업 규칙 (Claude / 개발자용)

> 이 폴더 아래 여러 Go 프로젝트가 있다. **공통** 규칙·설정·문서는 여기(`.claude/`, `docs/`), **프로젝트 개별** 항목은 각 프로젝트 폴더 아래에 둔다.

## 0. 구조

| 위치 | 내용 |
|---|---|
| `win-projects/CLAUDE.md` | 공통 규칙 (본 파일) |
| `win-projects/.claude/` | 공통 settings, hooks, agents(code-defect-reviewer, templates/agent-expert.md), templates/skill, tools |
| `win-projects/docs/` | 공통 문서 (코딩 표준 [11_coding-standard.md](./docs/11_coding-standard.md), 문서 템플릿 README.md) |
| `<project>/CLAUDE.md` | 프로젝트 진입점 (산출물, docs 인덱스, 전용 agent) |
| `<project>/.claude/` | wip/, plan/, known-issues.md, 프로젝트 전용 agents/skills |
| `<project>/docs/` | 프로젝트 개별 문서 (README.md 인덱스, commands-history.md 등) |

신규 프로젝트: `<project>/` 생성 후 CLAUDE.md, docs/README.md, .claude/{wip,plan,known-issues.md}를 만든다 (`cli-manager-cr` 참고).

---

## 4. WIP / Resume Protocol (체크포인트)

다단계 요청은 **세션이 끊겨도 즉시 이어서 실행** 가능하도록 `<project>/.claude/wip/`에 영구 기록합니다.

**필수 행동** (경로는 현재 프로젝트 폴더 기준):
- **새 다단계 작업** → 즉시 `<project>/.claude/wip/<YYYY-MM-DD>-<slug>.md` 생성 (`_template.md` 복사)
- **각 단계 완료마다** → `[ ]` → `[x]`, Files touched 갱신
- **중단 시** → 진행 단계 `[~]` 표시 + **Resume hint** 작성
- **새 세션 시작 시** → `.claude/wip/` 스캔하여 `Status: in-progress` 파일 자동 발견 → Resume hint 부터 이어 진행
- **완료 시** → `Status: done` + `.claude/wip/archive/`로 이동

상세: `<project>/.claude/wip/README.md`

**SKIP**: 단순 질문, 단발 조회, 1~2단계 trivial 수정, 메타 작업

---

## 5. Known Issues 처리 규칙 (Notify & Dismiss)

알려진 지뢰는 `<project>/.claude/known-issues.md`에 status(`active`/`dismissed`/`resolved`)와 함께 관리됩니다.

**필수 행동** (경로는 현재 프로젝트 폴더 기준):
- **세션 시작 시** → 본 파일을 한 번 스캔하여 `active` 항목 인지
- **작업 진행 중** → 영향 영역(Trigger areas)에 해당하는 `active` 항목은 사용자에게 **한 번 경고**
- **`dismissed` / `resolved` 항목은 출력에 포함하지 않음** (명시 요청 시만)
- **사용자가 "문제 아니다" / "intentional" / "dismiss" / "괜찮아" 표명 시** → 즉시 `status: active → dismissed` 편집 + History 에 timestamp + 사유 한 줄 → **이후 경고하지 않음**
- **새 지뢰 발견 시** → 사용자 보고 후 `active`로 추가
- **코드/스키마 수정으로 해결 시** → `status: resolved` + 수정 정보 기록

---

## 6. Fix Protocol — 문제 수정 요청 시 필수 워크플로

사용자가 **"X 수정해줘 / 고쳐줘 / 버그 픽스 / 안 되는데 봐줘"** 등 수정을 요청할 때 반드시 아래 순서를 따른다 (직접 편집 전).

### 6.1 원인 분석 (Cause) — 추측 금지, 코드 확인 우선
- grep / Read 로 실제 코드를 확인하여 **정확한 원인 설명** (`[파일:line](상대경로#Lline)` 형식)
- **왜** 그 동작이 의도와 다른지 (호출 경로 / 데이터 흐름 / 조건 분기)
- 관련 `<project>/.claude/known-issues.md` 항목이 있으면 KI 번호 매칭
- 신규 코드는 [docs/11_coding-standard.md](./docs/11_coding-standard.md) 준수

### 6.2 수정 방안 (Fix Plan)
- 어느 파일을 어떻게 변경할지 **명확히** (before / after)
- **영향 범위** + **회귀 위험** + 빌드/테스트 검증 단계

### 6.3 다중 옵션이 있을 때
- 평소에는 **권장안을 골라 진행**하고, 고른 것과 이유를 응답에 한 줄로 밝힌다 (§8).
- 데이터 손실·운영 중단·호환성 파괴처럼 **되돌리기 어려운 차이**가 있으면 AskUserQuestion 으로 선택받는다 — 각 옵션의 trade-off 와 추천 옵션 `(Recommended)` + 이유.

### 6.4 사용자 승인 후 수정 진행
- 파일 Edit/Write 는 `.claude/settings.json` `permissions.allow` 로 **자동 허용** (확인 창 없음). 승인은 §6.3 의 되돌리기 어려운 선택에만 받는다

### 6.5 수정 완료 후
- 변경 요약 (Before / After / 영향 범위) + 회귀 검증 결과
- 관련 known-issues 항목 해결 시 `status: active → resolved`

---


---

## 8. 작업 방식

- **응답은 한국어로** 쓴다 (진행 문구·결과 요약·명령 설명 포함). 코드·명령·파일 경로는 원문 그대로.
- **기능 작업의 선택지는 묻지 말고 권장안으로 진행**하고, 고른 것은 응답에 한 줄 + 해당 프로젝트 `docs/decisions.md` 에 남긴다.
- **되돌리기 어렵거나 밖으로 나가는 일은 사용자에게 맡긴다** — 관리자 권한 배포, 운영 데이터·실제 돈이 걸린 전환, 외부 공개, 안전장치 끄기.
- 커밋·push 는 사용자가 요청할 때만 한다.
- **OS 중립**: 주 개발·실행 환경은 Windows 이지만 코드는 OS 에 관계없이 동작해야 한다. 경로는 `path/filepath`, 줄바꿈·권한·셸 의존을 피하고, OS 별 분기는 build tag(`_windows.go`, `_unix.go`)로 격리한다.
- **호스트에 설치·설정·서비스 등록을 하면 전부 기록**한다: 해당 프로젝트 `docs/host-settings/` (번호 문서 + README 표·변경 이력 + 패키지 목록(winget/scoop 등) + 멱등 `setup.ps1` + 상태 점검 스크립트). 비밀값은 넣지 않는다.

---

## 9. Git 커밋 규칙

- **커밋 메시지(제목·본문)는 무조건 한글로 쓴다.** 종류 접두어(`feat(api):`, `fix(api):`, `docs:` 등)와 파일·함수·식별자 이름만 영문 그대로 둔다.
- 영문으로 커밋했는데 아직 push 전이면 한글 메시지로 다시 만든다.
- 커밋은 성격별로 나눈다 (서버 수정 / 기능 / 문서 등).

---

## 10. 새 프로젝트 추가 체크리스트

1. `<project>/` 생성: `CLAUDE.md`, `docs/`, `.claude/{known-issues.md,wip/archive/,plan/}`, `go.mod`, `Makefile`, `cmd/`, `internal/`, `deploy/`
2. `<project>/CLAUDE.md` 머리에 "공통 규칙은 ../CLAUDE.md" 링크, 아래 §11 표에 행 추가
3. `<project>/.claude/settings.json` 은 `cli-manager-cr` 것을 복사 (공통 hooks 를 `../.claude/hooks` 로 호출)
4. 프로젝트 문서는 `<project>/docs/` 에만, 공통 문서는 `win-projects/docs/` 에만 둔다
5. 비밀값 파일은 처음부터 `.gitignore` 로 제외
6. 서비스 작업을 시작하면 먼저 그 프로젝트의 `CLAUDE.md` 를 읽는다

## 11. 프로젝트 목록

| 프로젝트 | 상태 | 역할 | 진입점 |
|---|---|---|---|
| cli-manager-cr | **사용 중** (제품명 DariTerm) | BrightTerm v1.1.0(MIT) 동작을 참고한 clean-room 독자 재작성. 소스 수정·빌드는 여기서만 | [cli-manager-cr/CLAUDE.md](./cli-manager-cr/CLAUDE.md) |

> `cli-manager`(BrightTerm v1.1.0 이식물, 옛 TermGo)는 2026-11-13 사용자 지시로 폴더를 삭제했다. git 이력에는 남아 있지만 **clean-room 규칙상 이력에서도 열람·복원하지 않는다**.

**KI 번호는 레포 전체에서 이어진다.** 다음 번호: KI-012.
공통 항목은 `.claude/known-issues.md`, 프로젝트 항목은 `<project>/.claude/known-issues.md`.
WIP 재개 시 모두 스캔: `grep -l 'Status\*\*: in-progress' .claude/wip/*.md */.claude/wip/*.md`
