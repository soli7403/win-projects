# 프런트엔드 C1 구현 (설정 8구역·가져오기 대화상자·동기화 패널·백업·새 버전 확인)

**Status**: done (구현·단위 검증 완료, 기준 화면 나란히 비교 미완료)
**Started**: 2026-11-10 23:30
**Last update**: 2026-11-10 23:35
**Active agent**: impl-frontend-c1 (subagent)
**Branch**: -

---

## Context

- 사용자 요청: clean-room 프런트 C1. spec 만 보고 설정 화면 8구역, 서버 가져오기 대화상자, 동기화 패널, 백업 UI, 새 버전 확인 UI 구현. 커밋·push 금지.
- 범위 / 결정사항: 권장안 진행, 선택은 docs/decisions.md 끝에 기록. 모호한 점은 spec\QUESTIONS.md 에 질문.
- 관련 문서: [spec/README.md](../../spec/README.md), [docs/clean-room-protocol.md](../../docs/clean-room-protocol.md), [spec/ui/settings.md](../../spec/ui/settings.md), [spec/ui/dialogs.md](../../spec/ui/dialogs.md)
- 영향 받는 모듈/바이너리: frontend\ (Settings*.tsx, ImportDialog.tsx, 동기화·백업·업데이트 UI, 테스트)

---

## Plan

- [x] 1. 명세 읽기 (settings.md, dialogs.md §5·§6, import.md, behavior/lifecycle.md 업데이트, data/sync-backup.md, acceptance G/I/Y) + 기존 관례 파악
- [x] 2. 설정 화면 8구역 구현 (store.patch 저장)
- [x] 3. 서버 가져오기 대화상자 + useUi 액션(openImportDialog, openSettings)
- [x] 4. 동기화 패널 + 백업 UI + 새 버전 확인 UI
- [~] 5. 기준 화면 나란히 (paused: 헤드리스 Edge 비교 미실시, 치수는 명세 값 기준으로만 맞춤) 비교 (shots 14~22) 및 차이 수정
- [x] 6. 테스트 이관 (화면 쪽 G2·G1 데이터·I6·Y4, C1.test.tsx 7건) (acceptance G1~G8, I1~I9, Y1~Y10 화면 쪽)
- [x] 7. tsc (C1 파일 0건; Sidebar.tsx 는 다른 에이전트 진행 중 오류 7건), vitest 87/87, vite build 통과 / vitest / vite build 검증 + 최종 보고

체크 마크 의미:
- `[ ]` 미시작
- `[~]` 진행 중 (paused 시 `(paused: 사유)` 추가)
- `[x]` 완료

---

## Files touched

- frontend/src/state/ui.ts (modified: settings·importOpen 액션 추가)
- frontend/src/state/settings.ts, state/update.ts, ipc/c1types.ts (created)
- frontend/src/screens/Settings.tsx, SettingsAppearance/Input/Files/Connection/Sync/Security/Backup/About.tsx, ImportDialog.tsx, C1.test.tsx (created)
- frontend/src/ui/UpdateBadge.tsx, styles/settings.css (created)
- frontend/src/App.tsx (modified: 3 import, update 구독, Settings·ImportDialog 마운트 줄)
- frontend/src/screens/Shell.tsx (modified: 설정 버튼 openSettings, UpdateBadge 한 줄, notYet import 제거)
- frontend/src/ipc/fakeBackend.ts (modified: C1 채널 가짜 응답 추가)
- spec/QUESTIONS.md (Q20·Q21 추가), docs/decisions.md (D31~D34 추가)

---

## Notes / decisions

- [2026-11-10 23:30] 시작 (wip 생성). 접근 금지 영역 열람 없음.
- [2026-11-10 23:35] 설정 8구역·가져오기·동기화·백업·업데이트 UI 구현, 검증 통과. 기준 화면 비교 미실시.

---

## Resume hint

- 다음 단계: 기준 화면(shots 14~22) 헤드리스 Edge 비교, 빠른 접속·정보 화면 연결 확인.
- 주의사항: 공유 파일(App.tsx, state/*, fakeBackend.ts, decisions.md)은 필요한 줄만 편집.
- 사용자 확인 필요: 없음 (Q20·Q21 답 대기).
