# 구현자 세션 시작 안내

> 구현은 **기존 코드를 한 번도 읽지 않은 새 세션**이 한다. 이 문서는 사람이 새 세션을 열 때 따라 하는 절차다.

## 1. 세션 열기 전 확인
- [ ] 작업 폴더를 **`E:\work\claude\win-projects\cli-manager-cr`** 로 연다 (상위 폴더나 `cli-manager` 에서 열지 않는다. 그래야 [.claude/settings.json](../.claude/settings.json) 의 `deny` 가 적용된다).
- [ ] 이전에 `cli-manager` 를 읽은 대화를 이어서(resume) 쓰지 않는다. **새 대화**로 시작한다.
- [ ] 한 번에 한 단계만 맡긴다: 백엔드(C) 또는 프런트(B). 같은 세션이 명세 수정까지 겸하지 않는다.

## 2. 새 세션에 붙여 넣을 첫 메시지 (백엔드 예)

```
이 프로젝트는 clean-room 재작성이다. CLAUDE.md 의 "Clean-room 규칙"을 먼저 읽고 지켜라.
- ../cli-manager 와 BrightTerm 저장소는 열람·검색·참조 금지.
- 기준은 spec/ 문서뿐이다. 모호하면 추측하지 말고 spec/QUESTIONS.md 에 질문을 적고 그 부분은 건너뛰어라.
- spec/README.md 의 "알려진 공백"은 질문 대상이다.

작업: 백엔드(Go)를 spec/00-overview.md §6 의 순서로 구현한다.
1) 데이터 폴더·store·vault·IPC 틀 → 2) 세션(로컬 셸) → 3) SSH → 4) SFTP → 5) 텔넷·시리얼 → 6) 가져오기 → 7) 동기화·업데이트.
기준 문서: spec/data/*, spec/ipc.md, spec/protocols.md, spec/behavior/session.md·lifecycle.md, spec/import.md.
각 단계가 끝나면 spec/acceptance/ 의 해당 시나리오를 테스트로 옮겨 통과시켜라.
다단계 작업이므로 .claude/wip/ 에 WIP 파일을 만들어 진행을 기록하고, 선택지는 권장안으로 진행하며 docs/decisions.md 에 남겨라.
```

프런트(B)는 위 메시지에서 "백엔드(Go)" 부분을 아래로 바꾼다.
```
작업: 프런트엔드(React + TypeScript + xterm.js)를 spec/00-overview.md §6 의 순서로 구현한다.
외형 토큰·셸 → 잠금 화면 → 사이드바·서버 편집 → 탭·분할·터미널 → 빠른 접속 → SFTP·스니펫 → 설정 → 나머지.
기준 문서: spec/ui/*, spec/behavior/shortcuts.md·terminal.md, spec/ipc.md. 백엔드가 없으면 ipc.md 계약대로 만든 스텁으로 화면을 먼저 만든다.
```

## 3. 명세 작성자(기존 세션)가 할 일
- 구현자가 `spec/QUESTIONS.md` 에 질문을 쓰면, 답을 질문 칸이 아니라 **해당 명세 본문**에 반영하고 질문 상태를 `answered` 로 바꾼다.
- 구현자가 보낸 요청 외에 `cli-manager-cr` 의 구현 코드를 쓰지 않는다.

## 4. 대조 단계 (구현이 어느 정도 되면)
1. 기존 앱(`cli-manager\build\bin\termgo.exe`)과 새 앱을 **나란히 실행**해 화면 단위로 차이를 관찰한다.
2. 스크린샷을 `spec/ui/shots/` 에 둔다.
3. 차이는 **명세 수정 → 구현 수정** 순서로 반영한다. 구현자가 기존 코드를 열어 맞추지 않는다.
4. 위반이 있었으면 [clean-room-log.md](./clean-room-log.md) 에 적는다.

## 5. 배포 전
[clean-room-protocol.md](./clean-room-protocol.md) §4 체크리스트와 전문가 확인(UI 문구 원문, 알고리즘 설명의 상세 정도, LICENSE 표기).
