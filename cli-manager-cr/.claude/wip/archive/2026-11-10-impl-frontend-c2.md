# 프런트 C2 — 우측 패널 · SFTP · 스니펫 구현

**Status**: done
**Started**: 2026-11-10 23:30
**Last update**: 2026-11-10 23:25
**Active agent**: impl-frontend-c2 (subagent)
**Branch**: -

---

## Plan

- [x] 1. 명세 읽기 + 기존 관례 파악
- [x] 2. 우측 패널 컨테이너 (RightPanel.tsx, 폭 리사이즈 260~760, 탭, 닫기, rightPanel 상태 사용)
- [x] 3. SFTP 탭 (SftpPanel.tsx, SftpList.tsx, sftpOps.ts): 상태 안내, 도구 막대, 목록, 선택, 키보드, 우클릭, 삭제, 끌어 놓기, 전송 목록
- [x] 4. 미리보기 대화상자 (PreviewDialog.tsx, useUi.preview / openPreview)
- [x] 5. 스니펫 탭·편집 (SnippetsPanel.tsx, lib/termSend.ts)
- [x] 6. 가짜 백엔드 SFTP 채널 (ipc/fakeSftp.ts 신규, fakeBackend.ts 에 한 줄 병합)
- [x] 7. 기준 화면 대조 (08-sftp.png, 09-snippets.png): 헤드리스 Edge 로 비교, 치수 보정 완료
- [x] 8. 테스트 (screens/RightPanel.test.tsx: F1·F2·F3·F4·F5·F6·F7·F9·F10·F11) — F8 은 백엔드 토스트 영역이라 미포함
- [x] 9. tsc (내 파일 오류 없음), vitest 80/80, vite build 통과
- [x] 10. decisions.md D26~D30, QUESTIONS Q16~Q19

---

## Resume hint

- 남은 것: F8(편집 저장 토스트는 백엔드 소관). 폴더 아이콘 채움 방식은 기준 화면과 미세하게 다름.
- 주의: tsc 오류 3~4건은 다른 에이전트 파일(ImportDialog, SettingsAppearance, SettingsSync, Sidebar, Shell 의 notYet 미사용)이다.
