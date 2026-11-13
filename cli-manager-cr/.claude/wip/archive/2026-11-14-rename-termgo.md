# 남은 termgo 이름을 DariTerm 으로

**Status**: done
**Started**: 2026-11-14
**Last update**: 2026-11-14
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: 저장이 termgo 로 된다(실제 store.json `uploadDir: ~/.termgo/uploads`, `terminalTheme: TermGo Dark`). 남은 termgo 를 모두 DariTerm 으로.
- 분류: ① 저장값 이전(store 열 때) ② 보이는/내부 이름 변경(옛 값도 인식) ③ 다른 PC·기존 파일과 맞물린 식별자(동기화 파일·AAD 등) — 별도 판단.
- 관련: decisions D69, spec/data/store.md, protocols.md, sync-backup.md

## Plan

- [x] 1. store 열 때 이전: uploadDir 의 .termgo/.brightterm → .dariterm, terminalTheme `TermGo `/`BrightTerm ` → `DariTerm `
- [x] 2. 테마 이름 DariTerm Dark/Light (Go 기본값·프런트·호환 읽기)
- [x] 3. TERM_PROGRAM, User-Agent, 환경 변수 DARITERM_UPDATE_*(옛 TERMGO_* 도 읽음), git 커밋 메일, tmux 기본 이름
- [x] 4. 업로드 접두어 dt-(정리는 tg- 도), 백업 확장자 .dtbackup(가져오기는 .tgbackup 도)
- [x] 5. 프런트 내부 이름(MIME, 이벤트, package name), Go 모듈 termgo → dariterm
- [x] 6. 명세·문서 갱신 + D69
- [x] 7. 테스트·빌드

## Files touched
- internal/store/legacy.go·legacy_test.go(신규), store.go, backup, sftpfs/prompt.go, app/channels_update.go·dialers.go, gitsync/sync.go, session/messages.go, update/update.go, go.mod + 전체 import 경로
- frontend: termThemes, defaults, 테마 사용처, MIME·이벤트 이름, hostForm, fakeSftp, SettingsFiles, package.json/lock, 테스트
- spec 15개 문서, docs/decisions.md D69

## Notes / decisions

- 유지(호환): 옛 데이터 폴더 이전 코드(TermGo, TERMGO_DATA), 동기화 형식 `termgo-sync-1`·파일 `termgo-sync.json`·AAD `termgo-sync`, 백업 안 앱 식별자 — 사용자 판단 대상으로 보고.

## Resume hint

- 완료. go test -race ./..., tsc, vitest 176, wails build 통과. 동기화 식별자 3개는 유지(사용자에게 보고).
