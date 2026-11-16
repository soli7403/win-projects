# docs/import-plans 3건 구현 + 로그 보관 기간 설정·자동 정리 서비스

**Status**: done
**Started**: 2026-11-16
**Last update**: 2026-11-16
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: `docs/import-plans/` 의 계획서 3개(로그 파일 이름 충돌 KI-001, 출력 흐름 제어(역압), 세션·전송 잠재 문제 통합)를 구현하고, 추가로 로그 파일 최대 보관 일수 설정 + 주기적으로 오래된 로그를 지우는 Go 서비스 모듈을 구현.
- 결정: D84~D90. Known issues KI-001~KI-008 resolved, KI-009 dismissed.

---

## Plan

- [x] 1. 명세 반영 (session.md §6·§8, ipc.md §2·§4, protocols.md §2.6·§5, host-editor.md, settings.md §4, store.md)
- [x] 2. 묶음① log.go: KI-001 + ANSI 꼬리 보관 + 별칭 정리 + bufio 버퍼 쓰기 + 열린 로그 목록(IsLogOpen)
- [x] 3. 묶음② keepalive 응답 시한 + Conn 닫힘 신호
- [x] 4. 묶음③ serial 읽기 시한 50ms + 에코, 흐름 제어 안내 UI(10개 언어)
- [x] 5. 묶음④ local_windows 핸들 배타
- [x] 6. 묶음⑤ 닫은 뒤 session.data 차단(isCurrent, Batcher.Close), cp949 주석·테스트
- [x] 7. 묶음⑥ 역압: credit.go, session/control/batch, session.ack 채널, 프런트 ack·WebGL·숨은 탭
- [x] 8. 로그 보관 기간 설정(sessionLogKeepDays, 기본 30) + internal/logclean + 설정 화면
- [x] 9. go vet / go test -race ./... (17 패키지), tsc, vitest(35 파일 209 테스트), make notices, wails build
- [x] 10. known-issues, decisions D84~D90

---

## Files touched

- spec: behavior/session.md, ipc.md, protocols.md, ui/host-editor.md, ui/settings.md, data/store.md
- Go: internal/session/{log,credit,batch,session,control,encoding}.go (+ log_test, credit_test, session_test), internal/sshconn/{client,keepalive_forward}.go (+test), internal/transport/{serial,local_windows}.go (+serial_test, local_windows_test), internal/logclean/ (new), internal/app/{app,channels_session}.go, internal/store/store.go
- 프런트: lib/outputAck.ts(new), state/workspace.ts (+workspace.flow.test.ts), screens/TerminalView.tsx, screens/SettingsConnection.tsx, screens/HostEditorParts.tsx, ipc/{types,fakeBackend}.ts, ipc/contract/store.get*.json, lib/defaults.ts, package.json(@xterm/addon-webgl 0.19.0)
- locales/*.json 10개 (serial.flowNotYet, conn.logKeepDays, conn.logKeepDaysHelp)
- THIRD_PARTY_NOTICES.md, frontend/src/generated/third-party-notices.md
- docs/decisions.md, .claude/known-issues.md, ../CLAUDE.md(다음 KI 번호), CLAUDE.md(문서 인덱스)

---

## Notes / decisions

- [2026-11-16] vitest 는 Git Bash 의 소문자 드라이브(e:) 경로에서 "failed to find the current suite" 로 실패 — PowerShell 에서 `E:\` 경로로 실행하면 정상. 코드 문제 아님.
- [2026-11-16] 수동 검증(실제 SSH 서버·시리얼 장치·RDP)은 사용자 몫으로 남김 — 계획서 각 "수동 검증" 표 참고.

---

## Resume hint

- 완료. 커밋은 사용자 요청 시 성격별로 나눈다(명세 / Go 수정 / 프런트 / 문서).
