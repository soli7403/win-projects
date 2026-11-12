# 파일 패널: 로컬 탐색 + 탭 간 끌어 놓기 복사 + 맨 아래 복사 목록 (cli-manager-cr)

**Status**: done
**Started**: 2026-11-12
**Last update**: 2026-11-12
**Active agent**: main
**Branch**: -

---

## Context

- 사용자 요청: 로컬 터미널에서 파일 패널이 이 PC 탐색기로; 파일/폴더를 끌어 다른 탭으로 복사(폴더 행 → 그 안, 파일 행 → 그 폴더); 서버→서버는 로컬 임시 폴더 경유(설정 가능); 백그라운드 + 창 맨 아래 한 줄씩 진행; 항상 바이너리; 줄마다 취소 버튼.
- 사용자 지시(D54): 소스 수정·빌드는 cli-manager-cr 에서만.
- 명세: spec/behavior/file-panel.md (신규), spec/data/store.md transferTempDir, spec/README.md 인덱스.

## Plan

- [x] 1. 명세 작성 spec/behavior/file-panel.md
- [x] 2. backend: sftpfs/backend.go (Backend: SFTP·로컬), fs.go Backend 기반, Listing 에 parent/sep/local
- [x] 3. backend: copyjob.go (StartCopy/CancelCopy, 진행률·취소·서버↔서버 임시 폴더 경유·plan/uniqueName), transfer.go/prompt.go/edit.go Backend 로
- [x] 4. app: sftpFor 로컬 세션 지원, sftp.copy / sftp.cancelCopy 채널, 설정 transferTempDir 기본값
- [x] 5. frontend: state/copyJobs.ts, screens/fileDrag.ts, screens/CopyJobs.tsx, SftpPanel/SftpList/TabBar/Panes/RightPanel/Shell/SettingsFiles 반영
- [x] 6. go vet / go test -race ./... (계약 고정 파일 store.get 갱신: transferTempDir 추가) / tsc / wails build (build/bin/termgo.exe 22:04)

## Files touched

- internal/sftpfs/backend.go, copyjob.go, copyjob_test.go (created); fs.go, transfer.go, prompt.go, edit.go, sftpfs_test.go
- internal/app/channels_sftp.go; internal/store/store.go
- frontend/src/state/copyJobs.ts, screens/fileDrag.ts, screens/CopyJobs.tsx (created); screens/SftpPanel.tsx, SftpList.tsx, sftpOps.ts, TabBar.tsx, Panes.tsx, RightPanel.tsx, Shell.tsx, SettingsFiles.tsx; lib/sftpPath.ts, lib/defaults.ts; ipc/types.ts, ipc/backend.ts; styles/rightpanel.css; ipc/contract/store.get.json
- spec/behavior/file-panel.md, spec/README.md, spec/data/store.md; docs/decisions.md D54; docs/clean-room-log.md

## Notes / decisions

- [2026-11-12] 프런트 vitest 가 전 파일에서 "failed to find the current suite" 로 실패 — 내가 손대지 않은 순수 함수 테스트(format.test.ts)도 같아서 도구 설정 문제(vitest 4 + setupFiles 의 afterEach import)로 판단, 미수정.

## Resume hint

- 완료. 실제 앱에서 로컬 패널·탭 간 끌어 놓기·복사 목록 수동 확인은 사용자 몫.
