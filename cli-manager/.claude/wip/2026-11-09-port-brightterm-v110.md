# BrightTerm v1.1.0 → Go + Wails 포팅

**Status**: in-progress
**Started**: 2026-11-09
**Last update**: 2026-11-09 (§3 기능 3.12 제외 구현 완료)
**Active agent**: main
**Branch**: -

---

## Context

- 사용자 요청: [docs/porting-brief.md](../../docs/porting-brief.md) 대로 포팅 (brief §4 순서).
- 소스: `Yeosup/BrightTerm` **v1.1.0 태그만** (shallow clone, 스크래치패드 `bt-src`). v1.1.1+ 열람 금지.
- 제품명 임시 `TermGo`. 광고 배너 이식 안 함.

## Plan (brief §4)

- [x] 1. v1.1.0 체크아웃, Wails 프로젝트 생성, React UI 이식, `window.bt` 어댑터 (`frontend/src/bt-adapter.ts`)
  - 광고(AdBanner·배너 이미지·ads/openBanner API·Banner 타입·CSS) 제거, 로고 SVG 교체, BrightTerm 문자열 → TermGo, 정보 화면에 MIT 고지
  - 검증: `tsc --noEmit` OK, `vite build` OK, `go vet`/`go test` OK, `wails build` OK (`build/bin/termgo.exe`)
- [~] 2. Go 백엔드 — 실행 가능 단계 도달 (2026-11-09)
  - [x] store(+설정 기본값 병합) / vault(v1.1.0 형식 호환, DPAPI OS 잠금해제) / ui 브리지 / session(재접속·인코딩·12ms 출력 묶음)
  - [x] 트랜스포트: 로컬 PTY(ConPTY) · SSH(점프·포워딩·호스트키·에이전트) · 텔넷 · 시리얼(흐름제어 미지원)
  - [x] 핸들러: store/vault/cred/session/clip/dialog/backup/app/local:shells/serial:list
  - [x] 검증: go test 전체 통과(SSH는 인프로세스 서버로 호스트키·비밀번호 시험), 실제 앱 실행 → 로컬 터미널 입출력 확인(스크린샷)
  - [x] SFTP(`internal/sftpsvc`: 목록·전송·미리보기·편집·프롬프트 업로드) · 가져오기(`internal/importer`: PuTTY 레지스트리/ssh config/AWS CLI, 단위 시험 통과) · 클립보드 파일/이미지(Windows)
  - [ ] SFTP 실서버 통합 시험(미실시), 파일 드롭 경로(Go OnFileDrop), 타이틀바(네이티브 프레임, D3)
- [ ] 3. 볼트(v1.1.0 형식 호환), 설정·서버 목록 저장(store), 가져오기(PuTTY/ssh config/AWS)
- [ ] 4. 광고 제거 확인, v1.1.0 기능 동등성 테스트
- [~] 5. §3 기능: 3.7 → 3.1 → 3.2 → 3.9 → 3.10 → 3.5 → 3.4 → 3.8 → 3.3 → 3.6 → 3.11 → (3.12)
  - [x] 3.7 오류 안정화 (ipc 패닉 복구, Connect 복구, 연속 접속 실패 시험)
  - [x] 3.9 Shift+Enter, [x] 3.10 OSC52(쓰기만, 읽기 거절) — 설정 → 입력에서 끔
  - [x] 3.5 창 크기·위치 기억(window.json, 화면 밖이면 가운데, "원래 크기로") + 종료 확인 대화상자
  - [x] 3.1 사이드바↔열린 창 연동 (`state.activateHost/openOrActivate`, Sidebar 선택·스크롤 동기화, ▶ 로 하나 더 열기)
  - [x] 3.2 점프 호스트 직접 입력 (HostEditor + `jumpHostId='direct'`/`jumpDirect`, 베스천 전용 볼트 항목 "<별칭> 베스천", 대상 계정과 분리 — Go 통합 시험 통과)
  - [x] 3.4 탭·분할 레이아웃 복원 (`layoutPersist.ts`, store.json `layout`, 볼트 해제 후 1회 복원, 설정 → 연결에서 끔)
  - [x] 3.8 빈 칸·칸 옮기기·끌어 놓기 (`layout.ts` swap/move + vitest 4건, EmptyPane, 칸 머리글 드래그, 사이드바 서버 끌어 놓기, "위치 골라서 연결…" 대화상자)
  - [x] 3.3 세션 유지(tmux) — `session/tmux.go`; **실제 tmux(WSL)로 검증**: 생성 시에만 mouse on/status off/자동 실행, 창 닫아도 유지, 재접속 시 자동 실행 안 함. "유지" 배지·닫기 안내·Shift/⌥ 드래그 선택
  - [x] 3.6 새 버전 알림 (`internal/update`, 시작 5초 후 + 12시간마다, 요청에 사용자 정보 없음). **배포 저장소 미설정** → `-ldflags "-X main.g_szUpdateRepo=owner/name"` 또는 TERMGO_UPDATE_REPO
  - [x] 3.11 git 동기화 (`internal/gitsync`: 3방향 병합, 암호화 파일 하나, 볼트 받기/덮어쓰기, 로컬 bare 저장소로 2~3 PC 시나리오 시험 통과, 설정 → 동기화(git))
  - [ ] 3.12 Linux 키링 (선택, Windows 우선이라 보류)
- [ ] 6. 기능별 시나리오 테스트

## Files touched

- `go.mod`, `main.go`, `app.go`, `wails.json`, `.gitignore`, `LICENSE`(MIT, 원본), `third_party/brightterm-v1.1.0/LICENSE`
- `internal/ipc/router.go`, `router_test.go`
- `frontend/**` (v1.1.0 renderer 이식 + `bt-adapter.ts`, `vite.config.ts`, `tsconfig.json`, `package.json`)

## Resume hint (최신)

남은 일: 3.12(선택), 파일 드롭 경로(Go OnFileDrop), 타이틀바(D3), SFTP 실서버 시험, §4-4 v1.1.0 동등성 점검, 3.4/3.8/3.1 UI 수동 시험(자동화 불가 — WebView 디버그 포트 없음), `g_szUpdateRepo` 지정.

## (이전) Resume hint

다음은 3.1(사이드바 연동, 프런트 state.ts/Sidebar.tsx) → 3.2(점프 호스트 직접 입력: HostEditor + sshconn.Dial) → 3.4/3.8(레이아웃 복원·빈 칸, layout.ts) → 3.3(tmux). (이전 안내: (아래는 시작 시 안내, 일부 완료) v1.1.0 `src/main/{sessions,store,vault}.ts` 와 `shared/types.ts` 를 읽고 (스크래치패드의 bt-src 가 없으면 `git clone --depth 1 --branch v1.1.0 https://github.com/Yeosup/BrightTerm`), 아래를 순서대로:
1. `internal/store` (v1.1.0 JSON 형식 호환) + `store:get/patch` 핸들러
2. `internal/vault` (scrypt+AES-GCM, 형식 호환) + `vault:*`, `cred:*`
3. `internal/session` 인터페이스 + 출력 10~16ms 배치 + `session:*` 이벤트
4. 로컬 PTY(go-pty) → SSH → ...

알려진 갭: 파일 드롭 경로(`pathForFile`)는 Wails 에서 불가 → Go `OnFileDrop` 으로 대체 필요. 타이틀바(`app:titlebar`)는 Electron 커스텀 프레임 가정 — Wails Frameless 여부 결정 필요(decisions.md).
