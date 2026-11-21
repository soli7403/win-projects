# docs/import-plans 2026-11-19·20 계획서 3건 적용 (탭 분할 안내 · 탭 색 막대/폴더 색 물려받기 · SFTP 임시 이름 쓰기)

**Status**: done
**Started**: 2026-11-21
**Last update**: 2026-11-21
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: `docs/import-plans/` 에 올린 11월 19일 이후 계획서의 코드를 적용.
  - 2026-11-19-tab-split-hint-stuck.md (D101·D102)
  - 2026-11-20-active-tab-color-bar.md 1부 D103 + 2부 D105
  - 2026-11-20-sftp-temp-name-write.md (D104)
- 계획서는 "코드 수정 완료"로 적혀 있지만 이 폴더 코드·명세·decisions(D95 까지)에는 없었다 → 계획서의 코드·명세 변경을 그대로 옮긴다.

---

## Plan

- [x] 1. 탭 분할 안내: splitDropHint.ts(+test 16), Shell.tsx, 명세 shell.md §2.3 · 수용 G8/G8a
- [x] 2. 탭 색 막대: TabBar.tsx tabMark (+TabBar.mark.test.tsx), 명세 shell.md §2.1 · tokens.md · 수용 S4a
- [x] 3. 폴더 색·환경 물려받기: host_look.go(+test), channels_session.go, hosts.ts(+test)
- [x] 4. SFTP 임시 이름: atomic.go, owner_{unix,other}.go, backend.go, transfer/copyjob/prompt.go (+atomic_test.go), 명세 protocols §3 · file-panel §3·§6
- [x] 5. decisions D101~D105, CHANGELOG, CLAUDE.md 인덱스
- [x] 6. go vet / go test ./..., GOOS=linux vet, tsc, vitest, wails build

---

## Files touched

- split-hint: frontend/src/screens/splitDropHint.ts(new), splitDropHint.test.tsx(new), Shell.tsx, spec/ui/shell.md §2.3, spec/acceptance/02-shell-sidebar.md G8·G8a
- sftp: internal/sftpfs/{atomic,owner_unix,owner_other,atomic_test}.go(new), backend.go, transfer.go, copyjob.go, prompt.go, spec/protocols.md §3, spec/behavior/file-panel.md §3·§6
- 문서: docs/decisions.md D101~D105 + 번호 안내, CHANGELOG.md 다음 버전, CLAUDE.md 인덱스
- color-bar: frontend/src/screens/TabBar.tsx, TabBar.mark.test.tsx(new), frontend/src/lib/hosts.ts, hosts.test.ts, internal/app/host_look.go(new), host_look_test.go(new), internal/app/channels_session.go, spec/ui/shell.md §2.1, spec/ui/tokens.md, spec/acceptance/02-shell-sidebar.md S4a, docs/decisions.md D103·D105, CHANGELOG.md(다음 버전)

---

## Notes / decisions

- [2026-11-21] 1(D101·D102): splitDropHint.ts·Shell.tsx 는 계획서 코드 그대로, 테스트 16개는 §7.1 표대로 작성. 명세 shell.md §2.3, 수용 G8 개정·G8a 추가.
- [2026-11-21] 4(D104): atomic.go·owner_*.go·backend/transfer/copyjob/prompt 는 계획서 코드 그대로(prompt.go 에 io import 추가). atomic_test.go 8개는 §8.1 표대로 작성(복사 중간 취소는 대상 쓰기를 멈추는 gateBackend 로 재현). Windows 에서 6 통과·2 건너뜀. writeDirect 직행으로 되돌리면 3개 실패 확인. GOOS=linux vet 통과.
- [2026-11-21] 최종 검증: go vet, go test -race ./... 전체 통과, tsc, vitest 39 파일 249 테스트, wails build(13:09). 수동 검증은 각 계획서 수동 표(사용자 몫).
- [2026-11-21] gofmt -l 에 internal/session/exit_unlock_test.go 가 나옴 — 이번 작업 전부터, 건드리지 않음.

- [2026-11-21] 사용자 지시: 20일 것은 active-tab-color-bar 먼저 → 2·3 단계 먼저 완료. go vet, go test -race ./... 통과, tsc, vitest 38/233, wails build(12:31).
- [2026-11-21] decisions 에 D96~D100 이 없음 — 계획서 번호(D101~D105) 그대로 쓰고 번호 안내 문단 추가.
- [2026-11-21] host_look_test.go 는 계획서에 코드가 없어 §B7.1 표대로 작성. 수정 전 동작으로 되돌리면 TestSessionOpenInheritsFolderLook 3건 실패 확인.

---

## Resume hint

- 완료. 커밋은 사용자 요청 시 성격별로(프런트 탭 표시 / Go 세션 색 / Go SFTP 임시 이름 / 명세·문서).
