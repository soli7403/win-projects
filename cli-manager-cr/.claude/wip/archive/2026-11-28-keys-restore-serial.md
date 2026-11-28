# SSH 키 만들기·등록, 세션 복원, 시리얼 HEX·XMODEM → 1.2.0 에 포함

**Status**: done
**Started**: 2026-11-28
**Last update**: 2026-11-28
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "X11 포워딩은 제외하고 추천대로 SSH 키 만들기·등록과 세션 복원을 먼저, 이어서 시리얼 HEX·XMODEM 까지 구현해주고 이것도 1.2.0에 포함해줘. 물론 테스트도 되어야해."
- 1.2.0 은 스토어 제출 전·릴리스 저장소 게시 전이라 같은 번호로 다시 만든다(D133 방식).
- 새 기능이라 명세(spec/)를 함께 쓴다. 문구는 locales 10개 언어, 도움말 ko/en.

---

## Plan

- [x] 1. 구조 파악(IPC·보관함 키·워크스페이스·시리얼 세션·ZMODEM 연결점)
- [x] 2. SSH 키 만들기(ed25519·RSA) + 공개 키 보기/복사 + 서버에 등록(열린 SSH 연결로 authorized_keys) — Go+프런트+테스트
- [x] 3. 세션 복원 — 이미 있음(lifecycle.md §3, 설정 restoreLayout 기본 켬). 손대지 않음(D165)
- [x] 4. 시리얼 HEX 보기·HEX 보내기 — Go+프런트+테스트
- [x] 5. XMODEM 보내기·받기(체크섬·CRC·1K) — Go+프런트+테스트
- [x] 6. locales 10개, 도움말 ko/en, spec, decisions, CHANGELOG·listing
- [x] 7. 전체 검증(vet·race·tsc·npm test·wails build)
- [x] 8. 1.2.0 다시 빌드(msix·installer·zip), 배포 문서 SHA256 갱신
- [x] 9. 커밋·push

---

## Files touched
- Go: internal/sshconn/keygen.go(+_test), internal/app/channels_keys.go, keys_test.go, channels_serial.go, app.go(등록), internal/xmodem/xmodem.go(+_test), internal/session/serialtools.go(+_test), session.go·control.go·types.go(hexView·xmodem 연결)
- 프런트: screens/KeyDialogs.tsx(+test), HexSendDialog.tsx, Panes.tsx(paneToolItems)+Panes.tools.test.tsx, SettingsSecurity.tsx, ipc/types.ts(hexView), ipc/fakeBackend.ts
- locales 10개: 60키 추가

---

## Notes / decisions
- 사용자 추가 지시: 매뉴얼·help 도 수정 → user-guide ko/en·scenarios ko/en 1.2.0 기준, build.mjs 로 HTML/PDF 재생성, help 주제 2개(D165)
- 결정 D162~D165 기록, D166 은 재빌드 때(해시)

---

## Resume hint

- 완료. 남은 일(사용자): 수동 확인(releases/1.2.0.md §4 11~13), 스토어 제출, 릴리스 저장소 v1.2.0 게시. 코드·테스트는 끝남(Go 전부 통과, npm test 516).
