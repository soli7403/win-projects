# clean-room 재작성 — 명세 단계 (Phase A)

**Status**: done
**Started**: 2026-11-10 00:00
**Last update**: 2026-11-10 00:00
**Active agent**: main (명세 작성자 역할 전용)
**Branch**: -

---

## Context

- 사용자 요청: 라이선스·소유권 정리가 목적. 명세 기반 clean-room 재작성을 **별도 프로젝트**로 진행. 화면의 동작만 보고 구현. 프런트는 React 로 새로 작성.
- 결정: [docs/decisions.md](../../docs/decisions.md) D1~D4. 절차: [docs/clean-room-protocol.md](../../docs/clean-room-protocol.md).
- **이 세션은 기존 코드를 읽었으므로 명세 작성자 전용.** 구현(Phase B/C)은 이 폴더에서 시작한 **새 세션**이 한다.
- 영향: `cli-manager-cr/` 만. `cli-manager/` 는 수정하지 않는다 (읽기만).

---

## Plan

- [x] 1. 프로젝트 골격: CLAUDE.md(clean-room 규칙), settings.json(deny), docs/, spec/ 인덱스, QUESTIONS, known-issues, 루트 CLAUDE.md §11 행 추가
- [x] 2. 기존 앱 화면 스크린샷 23장 → `spec/ui/shots/`, 주요 치수 15종 측정 대조 일치(README 참조). 동적 상태·실제 출력 화면은 구현 후 대조
- [x] 3. `spec/data/*` — store·vault·sync-backup draft 완료
- [x] 4. `spec/ipc.md` — 새 채널 이름 체계로 정의(draft)
- [x] 5. `spec/ui/*` — tokens·shell·panes·sidebar·dialogs·host-editor·settings·lock·sftp-snippets draft 완료
- [x] 6. `spec/behavior/*`(shortcuts·terminal·session·lifecycle), `protocols.md`, `import.md`, `00-overview.md` draft 완료
- [x] 7. `spec/acceptance/01~04` — 화면·영역별 5개 이상
- [~] 8. 코드 표현 혼입 검토 — 1차 grep 점검 완료(결과 없음). **남음**: 샘플 정독 검토, 문서 `ready` 승격(스크린샷 대조·구현자 시범 읽기 후)
- [x] 9. 구현 세션 시작 안내 — [docs/implementer-start.md](../../docs/implementer-start.md)

체크 마크: `[ ]` 미시작 · `[~]` 진행 중 · `[x]` 완료

---

## Files touched

- `cli-manager-cr/**` (created: CLAUDE.md, .claude/settings.json, .claude/known-issues.md, .gitignore, docs/{README,clean-room-protocol,clean-room-log,decisions}.md, spec/{README,QUESTIONS}.md)
- `CLAUDE.md` (modified: §11 프로젝트 목록)

---

## Notes / decisions

- [2026-11-10] 명세에는 동작·수치·형식만. 코드 표현·내부 구조 이름은 쓰지 않는다. UI 문구는 사용자에게 보이는 사실이라 기록한다.
- [2026-11-10] 기존 `cli-manager` 의 Preact 전환(D13)은 이 프로젝트와 무관하게 유지한다.

---

## Resume hint

- 다음 단계: ① 샘플 정독으로 코드 표현 혼입 재검토 ② 문서를 `ready` 로 승격 ③ 구현자 세션(새 세션, `cli-manager-cr` 에서 시작) 안내 — 백엔드(C) 또는 프런트(B)부터
- 주의사항: 명세 작성 세션은 `cli-manager-cr` 에 구현 코드를 쓰지 않는다. 구현자 세션은 `cli-manager-cr` 에서 새로 시작.
- 사용자 확인 필요: 없음 (되돌리기 어려운 항목 없음). 단 배포·고지 정리 전 전문가 확인.
