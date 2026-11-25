# 2026-11-25 계획서 8건 순서대로 적용

**Status**: done
**Started**: 2026-11-25
**Last update**: 2026-11-25
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "import-plans 아래 오늘 날짜의 md 파일을 숫자 순서대로 진행"
- 대상: docs/import-plans/2026-11-25-{1..8}-*.md (계획서에 코드·패치·문구·명세 포함)
- 결정 번호: 계획서 D123→**D123-2**(기존 1.1.0 패키지 → D123-1), D124→**D124-2**(기존 의존성 최신화 → D124-1). D125~D132 는 그대로
- 도구: scratchpad 의 plan_extract.py · applyp.sh · hunkapply.py · apply_locales.py · step.sh

---

## Plan

- [x] 0. 결정 번호 D123-1·D124-1 로 바꾸기
- [x] 1. zmodem lrzsz 취소·응답 시한·tmux (D123-2·D125, KI-012)
- [x] 2. 찾기 클립보드 KI-011 — 코드·테스트·명세 모두 D120-1 로 이미 반영(새 파일 4개 동일, 패치 역적용 확인). 바꿀 것 없음, D124-2 안 둠
- [x] 3. 단축키 설정 (D126)
- [x] 4. KI-009 끊김 까닭 (D127)
- [x] 5. Xshell 가져오기 (D128)
- [x] 6. Xshell 포트 포워딩·.xsh (D129)
- [x] 7. 왼쪽 열 너비·명령 모음 (D130)
- [x] 8. 서버 목록 구역 접기·구분선 (D131·D132)
- [x] 9. 전체 검증: go build/vet/test -race, tsc, npm test, wails build
- [x] 10. decisions·CHANGELOG·CLAUDE.md·KI 다음 번호 정리

---

## Files touched

- 1: internal/zmodem/{engine,receive,frame,detect,zmodem_test}.go, internal/session/{zmodem,session,zmodem_test}.go, tools/zmodemcheck/main.go (new), spec/behavior/zmodem.md
- 3: frontend lib/keys.ts·lib/shortcuts.ts (다시 씀), state/shortcuts.ts·screens/SettingsShortcuts.tsx (new), Settings·SettingsAbout·App·TerminalView·Modal·Welcome·Panes·TabBar·types·defaults·ui·settings.css, internal/store/store.go, 계약 json 2, 테스트 2 new + SettingsLog.test, spec 6, docs/scenarios 한·영, user-guide 한·영
- 4: internal/transport/closecause.go·closecause_test.go (new), telnet.go, serial.go, spec/protocols.md
- 5·6: internal/importer/xshell*.go (new), types.go, internal/dialog/dialog_{windows,other}.go, internal/app/channels_import.go, frontend c1types·fakeBackend·ImportDialog(.test), spec import·ipc·ui/dialogs, scenarios
- 7: frontend Shell.tsx·Sidebar.tsx·app.css, Shell.leftcol.test.tsx·leftcol.layout.test.ts (new), spec sidebar·shell·sftp-snippets·acceptance/02
- 8: Sidebar.tsx·app.css·types·defaults·store.go·계약 json 2, Sidebar.sections.test.tsx (new), spec sidebar·store·acceptance/02, scenarios
- locales/*.json (10), docs/decisions.md (D123-1·D124-1 개번, D123-2·D125~D132), .claude/known-issues.md (KI-012 추가·resolved, KI-009 resolved), CHANGELOG.md, CLAUDE.md, ../CLAUDE.md (KI-013), docs/import-plans/2026-11-23-zmodem.md (§H-1), 2026-11-25 계획서 3개(결정 번호)

---

## Notes / decisions

- [2026-11-25] 2번(KI-011)은 D120-1 로 이미 반영돼 있어 적용할 것 없음
- [2026-11-25] 검증: go vet(windows·linux) ok, go test -race 전부 ok(clipboard 는 실제 클립보드 공유로 1회 흔들림, 재실행 3회 ok), tsc ok, vitest 54파일 366건, wails build + 실행 확인. darwin vet 은 serial 라이브러리 cgo 문제로 원래 안 됨

---

## Resume hint

- 완료. 앱 수동 검증은 각 계획서 §H, 커밋은 사용자 요청 시
