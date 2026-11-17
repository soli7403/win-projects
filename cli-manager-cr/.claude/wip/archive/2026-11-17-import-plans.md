# docs/import-plans 2026-11-17 2건 구현 (입력 순서 · 잠금/정상 종료 뒤 재접속 비밀번호)

**Status**: done
**Started**: 2026-11-17
**Last update**: 2026-11-17
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: `docs/import-plans/` 의 11월 17일 계획서 2개를 확인해서 적용.
  - [2026-11-17-term-input-order.md](../../docs/import-plans/2026-11-17-term-input-order.md): 안 A (프런트 세션별 입력 줄 세우기)
  - [2026-11-17-reconnect-password-after-screensaver.md](../../docs/import-plans/2026-11-17-reconnect-password-after-screensaver.md): 안 F + E + A (안 C 는 실측 조건부라 보류)
- 계획서의 질문(Q)은 구현자 제안대로 진행(CLAUDE.md §8), 결정 D92~ 에 기록. 두 번째 계획서는 R1 에 따라 KI 로 등록하지 않는다.

---

## Plan

- [x] 1. 입력 순서: frontend/src/ipc/inputQueue.ts + test, backend.ts 연결, 명세(ipc §1) 보장 문구
- [x] 2. 안 F: shell.go 종료 사유 보관 → transport.ErrRemoteExited → session 정상 종료 처리(재접속 안 함) + 문구 10개 언어
- [x] 3. 안 E: 다이얼러 계정 사본, applyCredential 3분류, 잠긴 동안 질문 창 차단, 닫기 시 삭제
- [x] 4. 안 A: 볼트 대기 상태, 잠금 해제 시 재개, 문구 10개 언어
- [x] 5. 명세 반영(session §3·§4, protocols §2.2, ipc), decisions D92~
- [x] 6. go vet / go test -race ./..., tsc, vitest, wails build
- [x] 7. 계획서 체크리스트·질문 답 갱신, CHANGELOG

---

## Files touched

- 프런트: frontend/src/ipc/inputQueue.ts(new) + inputQueue.test.ts(new), frontend/src/ipc/backend.ts
- spec/ipc.md §1 TermInput 순서 보장
- Go: internal/transport/telnet.go(ErrRemoteExited), internal/sshconn/{shell,client,auth}.go (+shell_exit_test.go new)
- Go: internal/session/{session,control,manager,messages,types}.go (+exit_unlock_test.go new)
- Go: internal/app/{ssh_app,dialers,channels_session,app}.go, internal/app/ssh_credcache.go(new) (+ssh_locked_test.go new)
- locales/*.json 10개 (be.remote.exited, be.session.remoteExited, be.session.waitUnlock)
- spec/behavior/session.md §3·§4, spec/protocols.md §2.2
- docs/decisions.md D92~D94, CHANGELOG.md(다음 버전), CLAUDE.md(문서 인덱스), docs/import-plans/2026-11-17-*.md(답·결정·체크리스트)

---

## Notes / decisions

- [2026-11-17] 계획서 질문은 구현자 제안대로(Q5 안 C 만 실측 전이라 보류). Q4 의 "Enter/[재연결]은 잠금 화면 표시"는 볼트가 잠기면 화면이 이미 잠금 화면이라 "다시 시도 → 아직 잠겨 있으면 같은 대기"로 구현.
- [2026-11-17] 검증: go vet, go test -race ./... 전체 통과, tsc, vitest 36 파일 216 테스트, wails build 성공. 수동 시나리오·성능 측정·실제 sshd 확인은 사용자 몫.
- [2026-11-17] Python 치환이 spec/protocols.md 의 CRLF 를 LF 로 바꿔 되돌림(그 파일만 CRLF 였음).

---

## Resume hint

- 완료. 커밋은 사용자 요청 시 성격별로 나눈다(프런트 입력 순서 / Go 재접속 / 명세·문서).
