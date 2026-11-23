# 2026-11-23-1 ~ 4 계획서 순서대로 적용

**Status**: done
**Started**: 2026-11-23
**Last update**: 2026-11-23
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: `docs/import-plans/2026-11-23-1-*.md` ~ `2026-11-23-4-*.md` 를 순서대로 적용.
- 번호: 1 plus-button-own-group(D119) → 2 log-settings-section-ansi(D120 → D120-2, KI-011 수정이 먼저 D120 을 써서 D120-1) → 3 pane-zoom-icons(D121) → 4 tab-pane-count-badge(D122)
- 도구: scratchpad 의 plan_extract.py / applyp.sh / hunkapply.py / apply_locales.py (앞 작업과 같음)

---

## Plan

- [x] 1. plus-button-own-group (D119) — shell.md 는 문맥이 달라 "그룹별 선택" 뒤에 직접, acceptance 02 는 머리 없는 diff 라 G5 뒤에 직접
- [x] 2. log-settings-section-ansi (D120-2) — 시나리오 "연결" 행은 공식 문서에 보관 기간이 더 있어 삭제 줄을 맞춰 적용, 보관 기간 설명은 새 "로그" 행에 되살림
- [x] 3. pane-zoom-icons (D121)
- [x] 4. tab-pane-count-badge (D122) — tokens.md 는 문맥 행이 달라 배지 행만 교체
- [x] 5. go vet·go test ./...·tsc·vitest(49파일 311개)·wails build 통과, decisions D119·D120-2·D121·D122(+ D120 → D120-1), CHANGELOG 1.1.0, CLAUDE.md

---

## Files touched

---

## Notes / decisions

---

## Resume hint

- 완료. 남은 일: 각 계획서 §H 수동 검증, 커밋(요청 시).

- 다음 단계: Plan 의 첫 `[ ]` 항목. KI-011 수정(D120-1)으로 TerminalView.tsx·Panes.tsx·spec/ui/panes.md·settings.md 가 계획서 기준과 다를 수 있다.
