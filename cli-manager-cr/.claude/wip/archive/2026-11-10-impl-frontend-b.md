# 프런트엔드 B: 서버 편집·폴더 편집·사이드바 잔여 (frontend/)

**Status**: done
**Started**: 2026-11-10 23:30
**Last update**: 2026-11-10 23:30
**Active agent**: impl-frontend-b (subagent)
**Branch**: -

---

## Context

- 사용자 요청: clean-room 프런트 구현자 B. spec 만 기준으로 서버 편집 잔여(포워딩·시리얼·로컬·점프·tmux·공유 계정·키 파일·인코딩·저장 동작), 폴더 편집 대조, 사이드바 잔여(끌어 놓기·즐겨찾기/최근·선택 규칙·키보드·선택 동기화) 구현.
- 범위 / 결정사항: 소유 파일만 수정. 공유 파일은 필요한 줄만. 커밋·push 금지. 기존 cli-manager 열람 금지.
- 관련 문서: [spec/ui/host-editor.md](../../spec/ui/host-editor.md), [spec/ui/sidebar.md](../../spec/ui/sidebar.md), [spec/acceptance/02-shell-sidebar.md](../../spec/acceptance/02-shell-sidebar.md), [spec/acceptance/04-hosts-sftp-settings-sync.md](../../spec/acceptance/04-hosts-sftp-settings-sync.md)
- 영향 받는 모듈/바이너리: frontend\src\screens\{Sidebar,HostEditor,FolderEditor}.tsx, frontend\src\lib\tree.ts, 신규 lib\hostForm.ts

---

## Plan

- [x] 0. wip 생성, 명세·기존 파일 파악
- [x] 1. lib/hostForm.ts + hostForm.test.ts (H1·H2·H3·H5·H6·H7·H9·H10) 9/9 통과
- [x] 2. 서버 편집 UI(HostEditor.tsx + HostEditorParts.tsx + lib/hostSave.ts): 기본·인증·고급·포트 포워딩 탭, 저장 파이프라인. 미검증: 실제 백엔드 연동, 기준 화면 비교
- [x] 3. 폴더 편집 대조(§2): 저장 검사를 canSaveFolder 로 통일, 나머지는 명세와 일치 확인
- [x] 4. 사이드바: 부제 포트 제거, 끌어 놓기(서버·폴더·빈 영역), 클릭 순환·더블클릭 규칙, 선택 동기화, 분할/하나 더 연결 버튼, 연결 점, 위치 선택·가져오기·그리드 액션 호출. 미완: 우클릭 설정 일부는 명세 확인만
- [~] 5. 검증: vitest 87/87, 내 파일 tsc 에러 0, vite build 통과. 미완: 기준 화면(10~13, 사이드바) Edge 비교 안 함, 전체 tsc 는 타 에이전트 RightPanel.test.tsx 때문에 실패
- [x] 6. docs/decisions.md D35~D38 추가 (최종 보고 대기)

---

## Files touched

(작성 중)

---

## Notes / decisions

- [2026-11-10 23:30] 시작

---

## Resume hint
- 구현 완료. 화면 대조: 헤드리스 Edge 로 서버 편집 4탭 캡처(scratchpad impl-frontend-b, host-*.png)해 기준 10 과 비교, 환경 색 점·호스트 도움말 차이 수정. 남음: 인증·고급·포트 포워딩 탭 세부 치수, 사이드바 대조(샘플 데이터 상이), 시리얼·텔넷 기본 탭 캡처. H4·H8·H11 화면 테스트 추가됨(3건 통과).

- 다음 단계: 1단계 lib/hostForm.ts 작성.
