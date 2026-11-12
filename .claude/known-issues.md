# Known Issues (win-projects 공통)

본 파일은 작업 중 주의해야 할 **알려진 문제·함정·코드 스멜의 마스터 리스트**입니다. 각 항목은 `status` 필드를 가지며, Claude/agent 는 이를 기준으로 사용자에게 경고할지 결정합니다.

---

## 📋 Status 분류

- **`active`** — 현재 유효한 문제 → 영향 영역 작업 시 사용자에게 경고 (notify)
- **`dismissed`** — 사용자가 "문제 아니다" 확인 → **다음부터 경고하지 않음**
- **`resolved`** — 코드/스키마/문서가 수정되어 해결 → 작업 시 history 로만 인지

---

## 🤖 Claude/Agent 동작 규칙 (필수)

### A. 세션 시작 시
1. 본 파일을 한 번 스캔하여 `active` 항목 목록을 인지

### B. 작업 진행 시
1. 작업이 어느 항목의 `Trigger areas`(파일/모듈/주제)에 해당하는지 매칭
2. 매칭되는 `active` 항목을 **사용자에게 명시적으로 한 번 경고**
3. `dismissed` / `resolved` 항목은 **출력에 포함하지 않음** (사용자가 명시 요청한 경우만)

### C. 사용자가 "이건 문제 아니다" / "intentional" / "dismiss" / "괜찮아" 등으로 표명할 때 (필수)
1. **즉시** 본 파일을 편집하여 해당 항목의 `status: active` → `status: dismissed` 변경
2. `History` 에 timestamp + 사용자 결정 사유 한 줄 추가:
   ```
   - YYYY-MM-DD: dismissed by user — <한 줄 사유>
   ```
3. 이후 같은 항목은 더 이상 경고하지 않음

### D. 새 지뢰 발견 시
1. 사용자에게 보고 후 본 파일에 `active` 상태로 새 KI-XXX 항목 추가
2. 형식: 아래 ACTIVE 섹션의 entry 형식 따르기

### E. 문제가 코드 수정으로 해결되었을 때
1. `status: active` → `status: resolved` + History 에 수정 commit/PR 정보 기록

---

## ✅ ACTIVE (현재 유효 — 경고 대상)

<!--
  새 항목 형식 (복사해서 채움):

### KI-001: <한 줄 제목>
- **Status**: active
- **Severity**: high | medium | low
- **Module**: <영향 패키지/바이너리>
- **Trigger areas**: <이 항목을 경고해야 하는 파일/모듈/주제>
- **Location**: [file.go:NN](../<path>#LNN)
- **증상**: <무엇이 잘못되는가>
- **원인**: <호출 경로 / 데이터 흐름>
- **History**:
  - YYYY-MM-DD: <발견 경위>
-->

(아직 등록된 항목 없음)

---

## ⏸️ DISMISSED

(없음)

---

## ✔️ RESOLVED

(없음)
