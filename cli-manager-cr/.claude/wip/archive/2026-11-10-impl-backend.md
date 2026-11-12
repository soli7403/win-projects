# 백엔드(Go + Wails) 구현 — 명세 기준 clean-room 구현

**Status**: done
**Started**: 2026-11-10 22:48
**Last update**: 2026-11-10 22:48
**Active agent**: impl-backend (subagent)
**Branch**: -

---

## Context
명세(`spec/`)만 보고 백엔드를 단계별 구현한다. 기존 앱·`../cli-manager`·BrightTerm 저장소는 열람하지 않는다.
- 사용자 요청: 백엔드 구현 (골격 → 세션 → SSH → SFTP → 텔넷·시리얼 → 가져오기 → 동기화·업데이트)
- 범위 / 결정사항: 깨끗한 경계(테스트 통과 단계 끝)에서 멈춘다. 커밋·push 금지. 선택지는 권장안 + `docs/decisions.md` D10~
- 관련 문서: [spec/00-overview.md](../../spec/00-overview.md), [spec/ipc.md](../../spec/ipc.md), [spec/data/store.md](../../spec/data/store.md), [spec/data/vault.md](../../spec/data/vault.md), [spec/data/sync-backup.md](../../spec/data/sync-backup.md)
- 영향 받는 모듈/바이너리: 프로젝트 루트 Go 모듈 전체 (`frontend/` 은 건드리지 않음)

---

## Plan

- [x] 1. 골격: go.mod, Makefile, wails.json, main.go(Wails), frontend/dist/.gitkeep, internal 패키지 구조, 데이터 폴더
- [x] 2. store (data/store.md, sync-backup §1·§2) + 원자 저장(.bak, .tmp, 0600) — `internal/jsonfile`, `internal/store`
- [x] 3. vault (data/vault.md §1–§7: scrypt, AES-GCM, 복구 코드, 항목, DPAPI OS 잠금) — `internal/vault`. §8(AdoptRemote, 헤더, 항목 교체)은 동기화 단계로 미룸
- [x] 4. IPC 라우터 (Invoke/TermInput/TermResize/Reply, 패닉 복구, 오류 문구, ui.request 응답 대기 10분) — `internal/ipc`
- [x] 5. 저장 데이터·볼트·cred·app.info 채널 연결, 기타 채널은 "구현되지 않은 채널" 로 남김 — `internal/app`, `main.go`
- [x] 6. 테스트 통과: `go vet ./...` exit 0, `go test -race ./...` 전부 ok, `go build ./...` exit 0
- [x] 7. 세션 코어 (2단계 일부): 상태 전이, 재접속 규칙·백오프, 12ms 묶음, 인코딩(utf-8/euc-kr/cp949 근사/shift_jis/latin1), 세션 로그, tmux 한 줄·이름 규칙, 오류 문구 번역, 안내 줄, tmux 감지 — `internal/session`
- [x] 8. 세션 채널 연결: session.open/close/reconnect, local.tmuxAvailable, TermInput/TermResize 라우팅 (main.go `App`)
- [x] 9. 전송 구현: 로컬 셸(ConPTY, `internal/transport/local_windows.go`, 실제 cmd.exe 시험 통과), 텔넷(`telnet.go`, IAC·NAWS 시험), 시리얼(`serial.go`, go.bug.st/serial, 순수 함수 시험). 목록 채널 local.shells·serial.list 연결.
- [x] 10. SSH 코어(`internal/sshconn`): 호스트 키 확인(knownHosts 기록), 인증 순서(키 암호 3회, Windows OpenSSH 파이프 에이전트, 저장 비밀번호, 키보드-인터랙티브, 비밀번호 4회는 연결 재생성), 점프 호스트 재귀(깊이 제한), keepalive(4회 실패 닫기), L/R 포워딩. in-process SSH 서버 테스트 6개 통과. 앱 연결(`internal/app/ssh_app.go`): 볼트 계정 적용, ui.request 질문 매핑, 저장 비밀번호 볼트 저장, lastUsedAt 갱신, knownHosts 저장소.
- [x] 11. SFTP(protocols.md §3, pkg/sftp): 목록·조작·업로드(큐·150ms·폴더 재귀)·업로드 선택·다운로드(폴더 대화상자)·미리보기·편집 감시(700ms·0.4초·12시간·토스트)·붙여넣기 업로드·local.saveForPrompt·tg- 정리 — `internal/sftpfs`, `internal/app/channels_sftp.go`. 클립보드(clip.*)·dialog.*·app.resetWindow/toggleFullScreen/titlebar 완료 — `internal/clipboard`, `internal/dialog`, `channels_clip.go`. 로컬 시작 폴더 안내 줄 추가. 앱 수준 IPC 통합 시험(SSH 세션→질문→답→입력→에코) 통과.
- [ ] 12. 창 상태(window.json)·종료 확인·자동 잠금 타이머·볼트 §8(헤더·AdoptRemote)·가져오기·동기화·업데이트 미착수
- [x] app.openExternal(http/https 만), app.openLogs 추가 (`channels_shell.go`)

체크 마크 의미:
- `[ ]` 미시작
- `[~]` 진행 중 (paused 시 `(paused: 사유)` 추가)
- `[x]` 완료

---

## Files touched

- `.claude/wip/2026-11-10-impl-backend.md` (created, updated)
- `go.mod`, `go.sum`, `wails.json`, `Makefile`, `main.go` (created)
- `frontend/dist/.gitkeep` (created, 빌드용 자리표시자만)
- `internal/jsonfile/jsonfile.go` (created)
- `internal/store/store.go`, `store_test.go` (created)
- `internal/vault/{crypto,types,vault,ops,items,dpapi_windows,dpapi_other}.go`, `vault_test.go` (created)
- `internal/ipc/{router,ask}.go`, `router_test.go` (created)
- `internal/app/{app,args,channels_vault}.go`, `app_test.go` (created)
- `docs/decisions.md` (D10–D15 appended, 기존 행 미수정)
- `spec/QUESTIONS.md` (Q1–Q8 추가, 기존 행 없음)

---

## Notes / decisions

- [2026-11-10 22:48] 발견: 코딩 표준(docs/11_coding-standard.md)은 헝가리안 명명을 요구 — 신규 코드에 적용 (D15).
- [2026-11-10 22:48] 발견: Go 1.26.2, Wails v2.10.2 CLI 설치됨. x/sys v0.50.0 에 CryptProtectData 존재 확인 → DPAPI 구현 가능.
- [2026-11-10 23:30] 결정: 저장·볼트 로직은 Wails 밖 순수 Go (D10). 기존 spec 공백은 QUESTIONS Q1–Q8.
- [2026-11-10 23:30] 검증: vet·race 테스트·build 모두 통과. vault 테스트는 scrypt 때문에 race 모드에서 약 40초.
- [2026-11-10 23:30] 미구현(의도적): 세션·SSH·SFTP·텔넷·시리얼·가져오기·동기화·업데이트·창 상태·vault §8 (AdoptRemote 등). Windows DPAPI 실제 호출은 시험 안 함(테스트는 플랫폼 무관 경로만).
- [2026-11-10 23:30] 주의: `main.go` 의 `Emitter`/`Bridge` 는 Wails 실행 시험을 하지 않았다 (빌드만 확인).

---

## Resume hint

- 다음 단계: 11번 SFTP 부터. `internal/sftp`(pkg/sftp, 목록·조작·전송 큐·진행 150ms·미리보기·편집 감시) → `internal/app/channels_sftp.go` 에 채널 등록. 그다음 클립보드(Windows: 파일 목록 CF_HDROP, 이미지 PNG 는 숨김 powershell 방식), dialog.*, 업로드 이름 규칙 `tg-…`.
- 주의: 실제 SSH 셸 접속을 앱 단위로 시험한 테스트는 없음(sshconn 테스트와 app 빌드만). session 쪽에서 `Dialer(notify)` 시그니처 사용(D24).
- 주의사항: 채널 추가 시 `internal/app/channels_*.go` 로 나누고 `registerXxx` 를 `NewService` 에서 호출. `frontend/` 건드리지 않는다. 커밋 금지. Hungarian 접두어 준수(D15).
- 미구현 명세 항목: session.open 시 서버 `lastUsedAt` 갱신(store.md §3), 자동 실행 명령 CR 변환은 구현됨. `session.md §5` 의 "접속 직후 처리" 중 lastUsedAt 만 남음.
- 사용자 확인 필요: QUESTIONS Q9–Q14 (프런트 질문 포함, Q9 는 답 반영됨, Q10–Q14 답 대기). 답 전까지 임시 구현 유지.
