# 로컬 터미널 파일 탐색 + 파일 패널 드래그 복사 (탭 간)

**Status**: done
**Started**: 2026-11-12
**Last update**: 2026-11-12
**Active agent**: main
**Branch**: -

---

## Context

- 사용자 요청: (1) 로컬 터미널에서 SFTP 패널이 로컬 폴더 탐색기로 (2) 파일/폴더를 드래그해 다른 탭으로 복사 — 폴더 행이면 그 안, 파일 행이면 그 파일의 폴더 (3) 서버→서버는 로컬 임시 폴더 경유, 임시 폴더 설정 (4) 백그라운드 + 맨 아래 한 줄씩 진행 표시 (5) 항상 바이너리 복사
- 결정: docs/decisions.md D16~D18

## Plan

- [x] 1. backend: fs.go (fileSys, remoteFS, localFS), Service.fsFor
- [x] 2. backend: List/Mkdir/Rename/Remove/Chmod/Preview/Edit/Upload/Download 를 fileSys 기반으로
- [x] 3. backend: Copy 작업(jobs.go: 진행률·취소, 서버↔서버 임시 폴더 경유) + 핸들러 + 테스트
- [x] 4. frontend: 로컬 세션에서 패널 활성화, 경로 구분자 처리
- [x] 5. frontend: 행 드래그 / 행·빈곳 드롭 / 탭·칸 hover 전환 / 탭 드롭, CopyJobs 맨 아래 목록, 설정 transferTempDir
- [x] 6. go vet·go test -race·tsc·vitest·vite build, decisions.md D16~D18

## Files touched

- internal/sftpsvc/fs.go, jobs.go, fs_test.go (created)
- internal/sftpsvc/sftpsvc.go, transfer.go, extras.go; internal/session/session.go (LocalCwd); internal/store/defaults.go; handlers_sftp.go (sftp:copy, sftp:cancelJob)
- frontend: src/fsDrag.ts, src/components/CopyJobs.tsx (created); RightPanel.tsx, TitleBar.tsx, Workspace.tsx, SettingsDialog.tsx, App.tsx, state.ts, api.ts, styles.css, shared/types.ts

## Resume hint

- 완료. 실제 앱(wails dev)에서 탭 간 드래그 수동 확인은 사용자 몫.
