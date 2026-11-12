# WIP (Work-In-Progress) 체크포인트 시스템

본 디렉토리는 **요청을 단계별로 추적**하고 **세션이 중간에 끊겨도 즉시 이어서 실행**할 수 있도록 합니다.

## 1. 디렉토리 구조

```
.claude/wip/
├── README.md                          ← 이 파일 (시스템 설명)
├── _template.md                       ← 새 WIP 생성 시 복사 템플릿
├── <YYYY-MM-DD>-<slug>.md             ← 진행 중인 작업 (Status: in-progress | blocked)
└── archive/
    └── <YYYY-MM-DD>-<slug>.md         ← 완료된 작업 (Status: done)
```

## 2. 언제 사용하나

**MUST**:
- 3단계 이상의 다단계 작업
- 코드 수정, 스키마 변경, 빌드/배포 작업
- 여러 파일에 걸친 기능 추가 / 버그 픽스
- 사용자 결정 대기로 일시 중단 가능성이 있는 작업

**SKIP** (WIP 파일 만들지 않음):
- 단순 질문, 단발 파일 조회
- 1~2단계로 끝나는 trivial 변경
- 본 시스템에 대한 메타 질문

## 3. 워크플로

### 작업 시작 시
1. `.claude/wip/_template.md` 을 복사하여 `<YYYY-MM-DD>-<slug>.md` 생성
   - slug: 3~5 단어 kebab-case (예: `2026-07-02-add-user-index.md`)
2. **Context** 섹션 작성 (요청 내용 / 범위 / 관련 docs 링크)
3. **Plan** 섹션에 단계 목록 작성 (`- [ ] 1. ...`)
4. 첫 단계를 `- [~] 1. ...` (in-progress)로 표시

### 진행 중
- 각 단계 완료 시 `- [x] N. ...` 로 마크 + `Last update` 타임스탬프 갱신
- 파일 수정 시 `Files touched` 섹션에 path 추가
- 결정사항 / 발견 / 블로커는 `Notes` 섹션에 timestamp와 함께 기록

### 일시 중단 / 세션 종료 시
- 진행 중 단계를 `- [~] N. ... (paused: <이유>)` 로 표시
- **Resume hint** 섹션을 마지막에 작성: 1~2문장으로 "어디서 어떻게 이어가야 하는지" 명시
- `Status: in-progress` 또는 `Status: blocked` 유지

### 새 세션에서 재개
1. [CLAUDE.md](../../CLAUDE.md) 읽기 (자동 로드됨)
2. `.claude/wip/` 디렉토리 확인 — `Status: in-progress` 또는 `blocked` 파일 검색
3. 가장 최신 파일 열어 **Resume hint** 확인
4. `[~]` 표시된 단계 또는 다음 `[ ]` 단계부터 진행

### 작업 완료 시
- 모든 단계가 `[x]`
- **Status** 를 `done` 으로 변경
- 파일을 `.claude/wip/archive/` 로 이동 (`mv` 또는 git mv)

## 4. 파일 명명 규칙

```
<YYYY-MM-DD>-<short-slug>.md
```

같은 날 여러 작업이 있으면 slug로 구분. 충돌 시 `<YYYY-MM-DD>-<slug>-2.md` 사용.

## 5. 파일 형식

[`_template.md`](./_template.md) 참조. 핵심 필드:
- **Status** — `in-progress` / `blocked` / `done`
- **Started** / **Last update** — `YYYY-MM-DD HH:MM`
- **Active agent** — 어느 subagent 담당 (또는 `main`)
- **Branch** — git 브랜치
- **Context** / **Plan** / **Files touched** / **Notes** / **Resume hint**

## 6. git 관리

기본적으로 **git tracked** 권장 (.gitignore 에 추가하지 않음):
- 진행 중 작업이 다른 머신/세션에서도 이어짐
- 팀과 진행 상황 공유 가능
- 완료된 작업의 `archive/` 는 audit trail 역할

민감한 정보(API 키, 비밀번호)가 Notes 에 들어가지 않도록 주의.
