# IPC 계약 점검·수정 (import 제외 전 채널)

**Status**: done
**Started**: 2026-11-11 00:00
**Last update**: 2026-11-11 20:00
**Active agent**: IPC 계약 점검 구현자 (subagent)
**Branch**: -

---

## Context
- 사용자 요청: spec/ipc.md 와 spec/data/*.md 기준으로 백엔드·프런트 IPC 불일치 점검 및 수정, 계약 고정 테스트 작성.
- 범위 / 결정사항: import.* 관련 파일(ImportDialog.tsx, c1types.ts, importSuggest.ts, fakeBackend 의 import 응답)은 C1 에이전트 담당이므로 건드리지 않는다. 기존 cli-manager 열람 금지(clean-room).
- 관련 문서: [docs/clean-room-protocol.md](../../docs/clean-room-protocol.md), [spec/ipc.md](../../spec/ipc.md)
- 영향 받는 모듈/바이너리: internal/app, internal/store, internal/vault, frontend/src/ipc, frontend/src/state

---

## Plan

- [x] 1. 명세 읽기 (ipc.md, data/*.md) 및 백엔드 핸들러 응답 구조 조사
- [x] 2. 프런트 채널 사용처·타입 조사
- [x] 3. docs/ipc-contract-audit.md 계약 대조표 작성
- [x] 4. 불일치 수정 (백엔드·프런트·가짜 백엔드)
- [x] 5. Go 계약 고정 테스트 + frontend/src/ipc/contract/*.json 생성
- [x] 6. vitest 계약 검증 테스트
- [x] 7. null 방어 점검 및 수정 (store 배열 키, normalizeStore)
- [x] 8. 검증 (go vet, go build, go test -race, tsc, vitest, vite build) 모두 통과
- [x] 9. docs/decisions.md 끝에 D45~D47, QUESTIONS.md 에 Q24~Q26 추가, 최종 보고

체크 마크 의미:
- `[ ]` 미시작
- `[~]` 진행 중 (paused 시 `(paused: 사유)` 추가)
- `[x]` 완료

---

## Files touched
<수정/생성 시마다 추가>

---

## Notes / decisions
- [2026-11-11 00:00] 시작. 금지 영역(cli-manager, BrightTerm, scratchpad 기존 파일, 상위 폴더 훑기) 접근 없음.

---

## Resume hint
- 다음 단계: 명세 읽기와 백엔드 핸들러 조사.
- 주의사항: import.* 파일 수정 금지. spec/ 본문 수정 금지. 커밋·push 금지.
- 사용자 확인 필요: 없음
