# 프런트엔드 구현 (React + TS + Vite + xterm.js, frontend/)

**Status**: done (1·2 완료, 3 거의 완료, 4 부분 완료: 탭·분할·터미널·빠른 접속·요청 대화상자 동작, 5 미착수)
**Started**: 2026-11-10 22:40
**Last update**: 2026-11-10 (4단계 중간)
**Active agent**: impl-frontend (subagent)
**Branch**: -

---

## Context

- 사용자 요청: spec 만 보고 clean-room 프런트엔드 구현 (frontend\ 만 작성, 커밋·push 금지).
- 범위 / 결정사항: 단계별 진행, 검사 통과 경계에서 중단. 모호한 점은 spec\QUESTIONS.md 에 질문.
- 관련 문서: [spec/README.md](../../spec/README.md), [docs/implementer-start.md](../../docs/implementer-start.md)
- 영향 받는 모듈/바이너리: frontend\

---

## Plan

- [x] 1. 프로젝트 설정 (package.json, tsconfig, vite, vitest) + 토큰 + 공통 요소 + IPC 어댑터 + 가짜 백엔드
- [x] 2. 화면 뼈대(shell, 시작 화면·제목·상태 표시줄; 탭·사이드바·우측 패널은 자리만) + 잠금 화면
- [~] 3. 사이드바(검색·섹션·트리·우클릭·리사이즈 완료, 끌어 놓기 미구현) + 서버 편집(기본·인증·고급 탭; 포트 포워딩·시리얼·점프 호스트·키 파일 대화상자 미구현) + 폴더 편집 완료
- [~] 4. 공통 입력 요소(Controls.tsx) · 단축키 표(시작 화면 12개) 완료. 탭·분할·xterm 터미널·빠른 접속·위치 선택·인증 요청 대화상자·단축키 동작 미구현
- [ ] 5. SFTP·스니펫 → 설정 → 가져오기·동기화 패널 (미착수)
- [ ] 6. tsc / vitest / vite build 검증 + 최종 보고

---

## Files touched

frontendpackage.json, tsconfig.json, vite.config.ts, index.html, src** (ipc, state, ui, screens, lib, styles, test)
specQUESTIONS.md (Q9~Q14 추가), docsdecisions.md (D16~D19 추가)

---

## Notes / decisions

- [2026-11-10 22:40] 시작
- [2026-11-10 23:20] 검증: tsc 통과, vitest 24/24 통과, vite build 통과, Edge 스모크(잠금→설정→복구→시작 화면) 확인
- [2026-11-10 23:20] 탭·사이드바·빠른 접속·SFTP·설정 등은 미구현. 대조 스크린샷은 잠금·시작 화면만 찍음

---

## Resume hint

- 라운드 A (4단계 잔여) 처리 결과: 찾기 막대(addon-search), 칸 우클릭 규칙(붙여넣기·복사·모두 선택·지우기·찾기·SFTP 열기는 useUi.rightPanel 설정), 이미지·파일 붙여넣기 업로드와 파일 끌어 놓기(state/paste.ts), 벨, Ctrl+클릭 링크, 서버를 칸으로 끌어 놓기(HOST_MIME, 사이드바 dragstart 는 에이전트 B 가 설정), 패널 탭 분리, 폴더 그리드(openFolderGrid), 위치 선택(LocationPicker), 레이아웃 저장·복원(state/layoutSave.ts, lifecycle §3).
- 미완료 A: 칸 우클릭에서 "SFTP 열기" 의 우측 패널 렌더링(에이전트 C2), 폴더 그리드·위치 선택을 사이드바·서버 편집에서 호출하는 연결(에이전트 B), Ctrl+Alt+V·Ctrl+V 이미지 업로드의 실제 서버 동작은 미검증(가짜 백엔드는 clip.info 가 empty).
- 검증: 내 파일 tsc 오류 없음. 전체 tsc 오류는 다른 에이전트 파일(HostEditor, RightPanel) 진행 중. vitest 전부 통과.

## 통합 점검 결함 수정 라운드 (2026-11-11)

- 1) 같은 서버 분할: 로컬·즉석 세션도 분할 가능 (lib/sameTarget.ts, state/actions.ts splitSameAs, 단축키·머리글·우클릭 공통). 테스트 lib/sameTarget.test.ts.
- 2) 칸 머리글: 이미지 붙여넣기(로컬·SFTP 연결), SFTP(canSftp), 분할(세션 있을 때 항상), 크게 보기(2칸 이상), 다시 접속(연결 아닐 때). screens/Panes.tsx.
- 3) 상태 표시줄: 이미지 안내(로컬 또는 연결된 SFTP), 저장된 서버 인코딩(대문자), 열×행(workspace.sizes, 터미널 맞춤 때 갱신). screens/Shell.tsx.
- 4) 첫 실행 가져오기 제안: state/importSuggest.ts (볼트 열림 + 서버 0개 + 기록 없음 → import.scan → 후보 있으면 openImportDialog 후 appUi.importSuggested 기록). 테스트 state/importSuggest.test.ts. App.tsx 에 마운트 2줄.
- 5) 대조: 빠른 접속(행 프로토콜 아이콘, 키 표기 바닥줄, 위치 8vh 로 수정), 시작 화면(일치), 라이트 테마(정상 렌더), 잠금 해제(배치 일치). 잠금 4모드 중 재설정·복구 코드 화면은 캡처만 하고 픽셀 대조는 하지 않음.
- 검증: tsc 0, vitest 108/108, vite build 통과(dist 갱신).
- 남은 것: 칸 우클릭 "복사/붙여넣기" 의 실제 연결 검증, 가짜 백엔드 기반 상태 표시줄 테스트 없음, 즐겨찾기 표시 데이터(가짜) 없음.
