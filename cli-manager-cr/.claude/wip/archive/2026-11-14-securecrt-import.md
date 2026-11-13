# SecureCRT 세션(폴더 구조 포함) 가져오기

**Status**: done
**Started**: 2026-11-14
**Last update**: 2026-11-14
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: 지금은 PuTTY 만 설정을 가져온다. SecureCRT 의 폴더 구조도 가져오게 설계·구현.
- 원본 앱에 없는 신규 기능(사용자 요청). 형식은 이 PC 의 SecureCRT 설정(`%APPDATA%\VanDyke\Config\Sessions`, 세션 190개)과 공개 자료로 확인. 비밀번호는 옮기지 않는다.
- 관련 문서: spec/import.md, spec/ipc.md, spec/ui/dialogs.md §5, docs/decisions.md D66

## Plan

- [x] 1. 기존 importer·채널·대화상자 확인, 실제 .ini 형식 확인(BOM UTF-8, S:/D:(16진)/Z:(줄 수+들여쓴 줄)/B:, 포워딩 표 `이름|로컬|1|대상|포트||`)
- [x] 2. 명세 갱신 (import.md §1.5, ipc.md, dialogs.md §5) + D66
- [x] 3. 백엔드: importer/securecrt*.go, Candidate.folderPath, apply 의 폴더 경로 생성, import.scan 포함 + import.scanSecureCrt
- [x] 4. 프런트: 타입·대화상자(폴더 선택 버튼, 폴더 경로 표시, 출처 라벨)·가짜 백엔드
- [x] 5. 테스트(Go·vitest) + 빌드 + 실제 폴더로 검증

## Files touched
- spec/import.md(§1.5·§4), spec/ipc.md, spec/ui/dialogs.md §5, spec/acceptance/04 I10~I12, docs/decisions.md D66
- internal/importer/securecrt.go, securecrt_ini.go, securecrt_scan.go, securecrt_windows.go, securecrt_other.go, securecrt_test.go (created)
- internal/importer/types.go, apply.go, apply_test.go, internal/app/channels_import.go
- frontend/src/ipc/c1types.ts, fakeBackend.ts, screens/ImportDialog.tsx, ImportDialog.test.tsx

## Notes / decisions

- 터미널 종류는 옮기지 않는다(기본 xterm-256color). SecureCRT 기본 VT100 을 그대로 쓰면 색이 줄어든다.
- 시리얼·방화벽(경유 세션) 키는 이 PC 표본이 없어 최선 추정 — 명세에 "표본 미확인" 표시.

## Resume hint

- 완료. go test -race ./... 통과, vitest 172 통과(PowerShell 에서 실행), wails build 성공. 실제 폴더 검증: 세션 189·폴더 10·포워딩 41.
