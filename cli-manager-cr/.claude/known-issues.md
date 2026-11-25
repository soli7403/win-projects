# Known Issues — cli-manager-cr

규칙(status·경고·dismiss 처리)은 [루트 known-issues](../../.claude/known-issues.md). KI 번호는 레포 전체에서 이어진다.

---

## ✅ ACTIVE

(없음)

## ⏸️ DISMISSED

(없음)

## ✔️ RESOLVED

### KI-012: ZMODEM 이 실제 lrzsz 와 맞대어 보지 않았고, 응답 없는 서버·tmux 에서 전송 상태로 남음
- **Status**: resolved
- **Severity**: medium
- **Module**: internal/zmodem, internal/session (zmodem.go, session.go)
- **Trigger areas**: ZMODEM 전송(sz·rz), 전송 중 취소, tmux·screen 안의 전송
- **증상**: (1) 자동 테스트는 이 구현끼리만 맞물려 실제 `sz`·`rz` 와 형식이 어긋날 수 있다 (2) 서버 쪽 프로그램이 응답 없이 멈추면 세션이 전송 상태로 남는다 (3) tmux 안에서는 대개 동작하지 않는다
- **원인**: ZMODEM 구현(D114) 때 lrzsz 가 있는 서버가 없었고, 엔진의 읽기에 시한이 없었다. tmux 는 시작 표시의 제어 문자(ZDLE)를 빼고 전달한다
- **History**:
  - 2026-11-23: ZMODEM 구현(D114) 때 검증의 한계로 기록 (docs/import-plans/2026-11-23-zmodem.md §2.3)
  - 2026-11-25: 시험 서버에 lrzsz 0.12.20 을 설치해 맞대어 봄 — 형식은 맞음(15B~40MB SHA256 일치). 받는 중 취소 시 데이터가 화면에 쏟아지는 결함을 찾아 고침(D123-2), 응답 시한 60초·즉시 취소·tmux 안내(D125). resolved. docs/import-plans/2026-11-25-1-zmodem-lrzsz-abort-stall-tmux.md
  - 2026-11-25: 앱에서 수동 검증(계획서 §H) 완료 — 사용자 확인

### KI-011: 찾기 막대가 열린 동안 찾은 글자가 클립보드에 계속 복사되고, 출력 중에는 강조가 늦게 갱신됨
- **Status**: resolved
- **Severity**: medium
- **Module**: frontend (screens/TerminalView.tsx — FindBar, 선택 시 자동 복사)
- **Trigger areas**: 터미널 찾기, 선택 시 자동 복사(copyOnSelect), 출력이 이어지는 세션
- **증상**(코드로 확인, 앱에서 재현해 보지는 않음):
  1. "선택 시 자동 복사"가 켜져 있으면(기본값) 찾기에서 글자를 칠 때·다음/이전을 누를 때마다 현재 항목이 클립보드에 복사된다. 출력이 이어지는 세션에서는 출력이 0.2초 멈출 때마다 다시 복사돼, 다른 곳에서 복사해 둔 내용이 덮인다.
  2. 출력이 0.2초 넘게 멈추지 않으면 새로 들어온 줄의 일치 항목은 강조되지 않는다(멈추면 한꺼번에 갱신).
  3. 찾기 막대가 열린 동안 마우스로 다른 곳을 선택해도, 출력이 멈추면 선택이 일치 항목으로 바뀐다.
  4. 강조는 버퍼 맨 위(가장 오래된 줄)부터 1000개까지만 한다. 흔한 낱말이면 최근 줄은 강조되지 않는다(다음/이전 이동은 된다).
- **원인**: 검색 애드온(@xterm/addon-search 0.16)이 현재 항목을 터미널 "선택"으로 표시하고, 출력이 올 때마다 0.2초 뒤(출력이 이어지면 계속 미룸) 전체를 다시 찾아 다시 선택한다. 앱의 자동 복사는 선택이 바뀌면 누가 선택했는지 가리지 않고 복사한다.
- **History**:
  - 2026-11-18: 사용자 질문("로그가 계속 출력중에 찾기를 하면 어떻게 되나?")으로 코드 확인 중 발견
  - 2026-11-23: 사용자 요청("KI-011을 수정해줘")으로 수정 (D120-1). 검색 애드온에 강조를 맡기지 않아 출력마다 다시 찾아 선택하던 동작을 없애고(1·3), 찾은 항목은 `lib/termFind.ts` 가 보이는 줄 0.1초 간격·스크롤 막대 최근 1000줄 1초 간격으로 직접 그린다(2·4). 자동 복사는 마우스 선택을 끝냈을 때만(`lib/copyOnSelect.ts`). 테스트 `termFind.test.ts`·`copyOnSelect.test.ts`. 앱에서의 수동 확인은 아직

### KI-010: 오른쪽 클릭·Ctrl+V 붙여넣기가 0.3~0.8초 늦음
- **Status**: resolved
- **Severity**: medium
- **Module**: internal/clipboard (clipboard_windows.go)
- **Trigger areas**: clip.info, 붙여넣기, 클립보드 파일 목록
- **증상**: 오른쪽 클릭(기본 설정에서는 붙여넣기)과 Ctrl+V 가 눈에 띄게 늦었다.
- **원인**: clip.info 가 파일 목록을 확인하려고 붙여넣기마다 `powershell.exe` 를 새로 띄웠다(이 PC 측정 0.3~0.85초). 클립보드에 텍스트만 있어도 거쳤다.
- **History**:
  - 2026-11-16: 사용자 보고("마우스 오른쪽 버튼으로 복사하는 것이 느린데")
  - 2026-11-16: CF_HDROP 을 Win32(IsClipboardFormatAvailable → GetClipboardData → DragQueryFileW)로 바로 읽도록 수정, 없으면 즉시 반환 (D91)

### KI-009: 텔넷·시리얼 끊김 이유가 원래 오류 대신 고정 문구
- **Status**: resolved
- **Severity**: low
- **Module**: internal/transport (telnet.go, serial.go)
- **Trigger areas**: 텔넷·시리얼 끊김 안내 문구
- **증상**: 끊긴 원인(소켓 오류 등)이 아니라 "연결이 종료되었습니다" / "포트가 닫혔습니다" 만 보인다.
- **원인**: spec/protocols.md §4·§5 가 고정 문구를 요구한다.
- **History**:
  - 2026-11-16: 세션·전송 계층 점검(docs/import-plans/2026-11-16-session-transport-issues.md §8)에서 발견. 명세대로의 동작이라 dismissed (사유: 명세의 고정 문구).
  - 2026-11-25: 사용자 요청("KI-009 도 직접적인 원인을 끊김 안내 문구에 적어줘. 사용자가 확인할 수 있게")으로 수정 (D127). 텔넷·시리얼이 읽기·쓰기 오류를 버리지 않고 이유에 붙인다(`internal/transport/closecause.go` `telnetClosed`·`serialClosed`): 텔넷은 "서버가 연결을 닫았습니다" 또는 "네트워크 오류 ({까닭})", 시리얼은 "포트가 닫혔습니다 ({까닭})". `errors.Is(…, ErrRemoteClosed/ErrPortClosed)` 는 그대로 참. 명세 spec/protocols.md §4·§5 도 고쳤다. 테스트 `closecause_test.go` 6개. 실제 장비(텔넷 서버·시리얼 장치)에서의 문구 확인은 계획서 docs/import-plans/2026-11-25-4-ki009-close-reason.md §H

### KI-001: 같은 별칭 세션을 같은 분에 열면 세션 로그가 한 파일에 섞임
- **Status**: resolved
- **Severity**: medium
- **Module**: internal/session (log.go)
- **Trigger areas**: 세션 로그 파일 이름·열기
- **증상**: `{별칭}-{YYYYMMDD-HHmm}.log` 를 이어 쓰기로 열어 같은 분에 연 세션들의 출력이 한 파일에 번갈아 섞였다.
- **원인**: 이름이 분 단위 + `O_APPEND|O_CREATE`.
- **History**:
  - 2026-11-16: 발견, 계획서 docs/import-plans/2026-11-16-log-filename-collision.md
  - 2026-11-16: 초 단위 이름 + 겹치면 `-2`… 접미, `O_EXCL` 로 새 파일만 만들도록 수정(log.go), 명세 session.md §6 개정 (D85)

### KI-002: SSH keepalive 가 조용히 끊긴 연결을 몇 분 뒤에야 감지
- **Status**: resolved
- **Severity**: high
- **Module**: internal/sshconn (keepalive_forward.go, client.go)
- **Trigger areas**: SSH keepalive, 끊김 감지·자동 재접속
- **증상**: 상대가 FIN/RST 없이 사라지면 "4번 실패" 가 세어지지 않고 OS TCP 시한까지 멈춰 있었다.
- **원인**: `SendRequest(wantReply=true)` 에 시한이 없어 응답이 안 오면 루프가 막혔다. 닫은 뒤에도 루프가 살아 있었다.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §1)
  - 2026-11-16: 요청마다 keepalive 간격만큼 시한(keepaliveOnce), Conn 닫힘 신호로 루프 종료, Close 는 Once (D86)

### KI-003: 시리얼 로컬 에코가 장치가 뭔가 보낼 때까지 안 보임
- **Status**: resolved
- **Severity**: high
- **Module**: internal/transport (serial.go)
- **Trigger areas**: 시리얼 Read·로컬 에코
- **증상**: 조용한 장치에서 입력한 글자가 화면에 안 나오다가 장치가 데이터를 보내면 한꺼번에 나왔다.
- **원인**: 포트 Read 에 시한이 없어 대기 중인 에코를 내보낼 기회가 없었다.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §2)
  - 2026-11-16: 읽기 시한 50ms, 시한마다 대기 에코를 먼저 내보냄. 흐름 제어 미적용 안내 문구도 서버 편집에 추가 (D87)

### KI-004: 세션 로그에 ANSI 조각이 남고, 로그 쓰기가 읽기 고루틴에서 동기 디스크 I/O
- **Status**: resolved
- **Severity**: medium
- **Module**: internal/session (log.go)
- **Trigger areas**: 세션 로그 내용·성능
- **증상**: `[31m` 같은 조각이 로그에 남았다. 느린 디스크에서 로그가 켜진 세션 출력이 밀렸다.
- **원인**: 묶음 단위로만 정규식 제거(경계에서 잘린 시퀀스), 묶음마다 `WriteString`.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §3·§10)
  - 2026-11-16: 미완성 시퀀스 꼬리 보관(최대 256B) + bufio 64KB, 1초마다·닫을 때 flush (D88)

### KI-005: 로컬 셸 Close 뒤 닫힌 프로세스 핸들 사용
- **Status**: resolved
- **Severity**: medium
- **Module**: internal/transport (local_windows.go)
- **Trigger areas**: 로컬(ConPTY) 세션 닫기
- **증상**: 닫은 뒤 읽기 고루틴의 exitReason 이 이미 닫힌 hProcess 로 Wait/GetExitCode 를 불렀다(핸들 값 재사용 위험).
- **원인**: Close 가 핸들을 닫는 것과 exitReason 의 사용 사이에 배타가 없었다.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §4)
  - 2026-11-16: mtxProcess 로 배타 + 닫힘 표시, Terminate 를 잠금 밖에서 먼저, 닫은 입력 핸들에 쓰지 않음 (D89)

### KI-006: 닫은 세션의 session.data 가 한두 번 더 옴
- **Status**: resolved
- **Severity**: low
- **Module**: internal/session (session.go, batch.go)
- **Trigger areas**: 세션 닫기, 접속 실패 안내
- **증상**: 닫은 직후 그 id 로 출력 이벤트가 오고, 닫은 뒤 늦게 끝난 접속 시도의 실패 안내가 나갔다.
- **원인**: readLoop 의 마지막 조각, onConnectFailed 가 세대 검사 전에 안내를 썼다.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §7)
  - 2026-11-16: isCurrent(세대·닫힘) 검사, Batcher.Close 로 닫은 뒤 Add 무시 (D84)

### KI-007: 로그 파일 별칭이 길거나 제어 문자를 품으면 열기 실패·이상한 이름
- **Status**: resolved
- **Severity**: low
- **Module**: internal/session (log.go)
- **Trigger areas**: 세션 로그 파일 이름
- **증상**: 아주 긴 별칭은 경로 한도로 로그 없이 열렸고, 탭·개행이 파일 이름에 들어갔다.
- **원인**: 금지 문자만 바꾸고 길이·제어 문자·끝 공백/점을 다루지 않았다.
- **History**:
  - 2026-11-16: 발견(통합 계획서 §9)
  - 2026-11-16: 제어 문자 치환, 앞뒤 공백·점 제거, 80자 자름, 비면 `session` (D88)

### KI-008: 한 세션의 고속 출력이 앱 전체를 느리게 함
- **Status**: resolved
- **Severity**: high
- **Module**: internal/session, internal/app, frontend (state/workspace.ts, screens/TerminalView.tsx)
- **Trigger areas**: 출력 전달 경로(session.data), xterm 렌더링
- **증상**: 초당 수 MB 출력 세션이 있으면 다른 탭의 입력·출력·UI 가 같이 느려지고 메모리가 불어났다.
- **원인**: 역압 없이 읽는 만큼 프런트로 보냄 + JS 스레드 하나 + DOM 렌더러 + 숨은 탭도 즉시 처리.
- **History**:
  - 2026-11-16: 발견, 계획서 docs/import-plans/2026-11-16-output-backpressure.md
  - 2026-11-16: ack 크레딧(512KB/128KB, 안전 시한 3초) + 묶음 256KB 상한 + WebGL 렌더러(실패 시 DOM) + 숨은 탭 100ms 몰아쓰기 (D84)
