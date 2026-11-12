# 결정 기록

번호는 계속 이어 붙인다 (지우지 않음 — 바꾸면 새 번호로 「D# 대체」).

| # | 날짜 | 결정 | 이유 · 대안 |
|---|---|---|---|
| D1 | 2026-11-09 | 프런트엔드는 `window.go`/`window.runtime` 전역을 쓰는 어댑터로 연결(생성된 wailsjs 모듈 미사용), 호출은 `App.Call(ch, argsJSON)` 단일 진입 | UI 코드 무수정 원칙(brief §2). 대안: 채널마다 바인딩 메서드 — 채널 수가 많아 기각 |
| D2 | 2026-11-09 | Go 모듈명 `termgo`, 패키지는 `internal/` 하위 | 임시 제품명 기준 |
| D3 | 2026-11-09 | 창은 Wails 네이티브 프레임을 쓴다 (v1.1.0 의 titleBarOverlay 커스텀 바는 미이식). 프런트의 `app:titlebar` 는 no-op | 첫 실행 가능 단계 우선. 대안: Frameless + 자체 최소/최대/닫기 버튼 — 추후 |
| D4 | 2026-11-09 | OS 키 보관은 DPAPI(`osWrappedWin` 필드)를 쓰고 v1.1.0 `osWrapped`(Electron safeStorage)는 호환하지 않는다 | safeStorage 형식(Local State 키)은 재현 불가. 마스터 비밀번호 호환은 유지 |
| D5 | 2026-11-09 | store.json 은 최상위 map 으로 보존 저장 (알 수 없는 필드 유지) | v1.1.0 데이터 호환 + 후속 기능의 필드 추가 대비 |
| D6 | 2026-11-09 | 테마 이름은 `TermGo Dark/Light` 로 바꾸고, 저장된 `BrightTerm *` 값은 설정 로드 시 자동 변환 | 이름 변경 + v1.1.0 데이터 호환 |
| D7 | 2026-11-09 | 창 상태는 store.json 이 아닌 `window.json` 에 저장 (설정 `rememberWindow` 만 settings) | 창 닫을 때마다 store 를 덮어쓰지 않기 위해 |
| D8 | 2026-11-09 | 빈 칸은 `PaneNode.sessionId === ""` 로 표현 | 타입 변경 없이 기존 코드가 안전하게 동작 (`sessions[""]` 는 undefined) |
| D9 | 2026-11-09 | 점프 호스트 "직접 입력"은 `jumpHostId="direct"` + `jumpDirect` 필드 | 별도 서버 등록 없이 서버 설정 안에 보관, 대상 `credentialId` 와 분리 |
| D10 | 2026-11-09 | tmux 명령은 `exec` 하지 않는다 (tmux 종료 시 원래 셸로 복귀) | `exit` 후 자동 재접속이 새 tmux 세션을 계속 만드는 루프 방지 |
| D11 | 2026-11-09 | git 동기화 파일 헤더에는 볼트 키 감싼 값(마스터 비밀번호로만 열림)만 평문, 항목 이름·서버·계정은 모두 암호문 안 | "어떤 정보도 평문으로 남기지 않는다" 요건 |
| D12 | 2026-11-09 | 업데이트 확인 저장소는 빌드 시 지정(`g_szUpdateRepo`), 미지정이면 확인 안 함 | 배포 저장소가 아직 없음 |
| D13 | 2026-11-10 | 프런트 경량화는 Wails 유지 + React→Preact(훅·JSX 호환) 교체, lucide-react→lucide-preact, xterm.js(WebGL) 유지. UI·동작은 동일 | 측정: react 222KB(25%) / xterm 498KB(56%) / 앱 147KB. 전면 재작성(바닐라)은 동작 불일치 위험이 커서 기각, Go 네이티브 GUI 는 터미널 에뮬레이터를 새로 만들어야 해서 기각 |
| D14 | 2026-11-10 | 정보 화면·LICENSE·main.go 에서 원저작자 `Dany Kim` 표기를 제거하고 `kai` 만 둔다. `third_party/brightterm-v1.1.0/LICENSE` 와 `docs/porting-brief.md` 는 그대로 | **사용자 요청**(3회 확인). 위험: 컴포넌트 TSX·Go 백엔드가 v1.1.0 파생 코드라 MIT 의 "저작권 고지 포함" 조건과 충돌할 수 있음 — 배포 전 전문가 확인 필요. 되돌리려면 위 세 곳에 고지를 복원 |
| D15 | 2026-11-10 | UI 런타임을 Preact 에서 다시 **React**(react 19 + react-dom + lucide-react)로 되돌린다. 코드는 그대로이고 원복 항목은 `onDoubleClick`·`spellCheck`·`lucide-react` import, vite/tsconfig 의 react 설정, `preact-setup.ts` 삭제 | 사용자 요청(UI 를 새로 만들 예정이라 React 기준으로 맞춤). 번들 685KB → 889KB(D13 의 절감 되돌림). D13 대체 |
| D16 | 2026-11-12 | 파일 패널(SFTP)은 `sftpsvc` 의 `fileSys` 추상화(원격 SFTP / 이 PC)로 로컬 터미널에서도 같은 `sftp:*` 채널로 동작한다. 로컬 세션의 시작 폴더는 서버 설정의 `local.cwd` | 프런트 API·화면을 그대로 재사용. 대안: 로컬 전용 채널 세트 — 중복이 커서 기각 |
| D17 | 2026-11-12 | 파일 패널 끌어 놓기 복사는 백그라운드 작업(`sftp:copy` → `copyjob:update`)이고 끌어 놓기 한 번 = 화면 맨 아래 한 줄(원본 → 대상, 전체 진행률, 취소). 탭/칸 위에 0.5초 머물면 전환, 폴더 행에 놓으면 그 폴더 안, 파일 행·빈 곳이면 현재 폴더, 탭에 바로 놓으면 그 세션의 마지막 폴더. 같은 이름이 있으면 확인 후 덮어쓰기, 원래 자리면 `이름 (1)` | 사용자 요청. 파일별 전송 항목은 끌어 놓기 복사에서는 보내지 않는다 |
| D18 | 2026-11-12 | 서버 → 서버 복사는 이 PC 임시 폴더(설정 `transferTempDir`, 비면 `%TEMP%\termgo-transfer`)에 받은 뒤 대상에 올리고 임시 사본을 지운다. 복사는 항상 바이너리 그대로 (줄바꿈 변환 없음) | 사용자 요청. 원격 파일시스템 동일성은 접속 대상 `user@host:port` 로 판단 (같은 서버 두 탭 사이에서 원본을 자기 자신에 덮어쓰지 않기 위해) |
