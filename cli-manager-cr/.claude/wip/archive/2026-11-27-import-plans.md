# 2026-11-26·27 계획서 23건 순서대로 적용

**Status**: done
**Started**: 2026-11-27 16:15
**Last update**: 2026-11-27 17:25
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "import-plans 아래 어제 오늘 다른 곳에서 수정한 것들을 적용해줘. 반드시 순서대로 적용해야한다."
- 대상: docs/import-plans/2026-11-26-{1..3}-*.md, 2026-11-27-{1..20}-*.md (숫자 순서). 결정 번호 D134~D160 (decisions.md 끝은 D133)
- 7번 이후 계획서는 코드 조각 + 설명이라 일부는 설명을 보고 구현
- 도구: scratchpad 의 plan_extract2.py · applyp.sh · hunkapply.py · apply_locales.py

---

## Plan

- [x] 26-1 스토어 업데이트 자식 신원 (D134)
- [x] 26-2 Xshell .xsh 파일 종류
- [x] 26-3 릴리스 저장소 업데이트 확인 (D135)
- [x] 27-1 포워딩 정보 (D136·D137·D142·D143)
- [x] 27-2 명령 변수 (D138)
- [x] 27-3 로그인 스크립트 (D139)
- [x] 27-4 출력 트리거 (D140)
- [x] 27-5 매크로 (D141·D143)
- [x] 27-6 gcc race 테스트 (D144, 호스트 기록)
- [x] 27-7 트리거 Windows 알림 (D145)
- [x] 27-8 매크로 일괄 입력 (D146)
- [x] 27-9 매크로 단축키 (D147)
- [x] 27-10 로그인 스크립트 금고 비밀 (D148)
- [x] 27-11 서버별 트리거 (D149)
- [x] 27-12 가져오기 로그인 스크립트 (D150)
- [x] 27-13 포워딩 전송량 (D151)
- [x] 27-14 프록시 명령 (D152)
- [x] 27-15 ssh config ProxyJump 체인 (D153)
- [x] 27-16 화면 보호기 잠금 감지 (D154)
- [x] 27-17 도움말 창 (D155)
- [x] 27-18 도움말 정렬·FAQ·그림 (D156~D158)
- [x] 27-19 도움말 크기 조절·그림 보기 (D159)
- [x] 27-20 세션 전송량 제목 (D160)
- [x] 전체 검증: go build/vet(windows·linux)/test -race 전부 ok, tsc ok, vitest 67파일 504개, wails build ok
- [x] decisions(D134~D160)·CHANGELOG(미배포 절)·CLAUDE.md 정리

---

## Files touched

---

## Notes / decisions

- [2026-11-27] 도움말 본문 ko/en 은 계획서에 전체가 없어 직접 씀 — 다른 곳 원본이 있으면 교체
- [2026-11-27] 27-14 프록시 명령: 연결 종료 때 명령 아래 프로세스까지 끝내게 보강(Job 객체/프로세스 그룹)
- [2026-11-27] 27-6: 이 PC 는 gcc 16.1.0 이 이미 있어 설치하지 않음
- [2026-11-27] KI-013(clipboard 흔들림)은 이번 -race 전체 실행에서 재현 안 됨 → 등록 안 함

---

## Resume hint

- 완료. 앱 수동 검증은 각 계획서 §H, 커밋은 사용자 요청 시
- 진행 메모: 결정 행·CHANGELOG 줄은 scratchpad/notes/{decisions,changelog,changelog-new}.md 에 모으는 중 (세션이 끊기면 계획서에서 다시 만든다)
- 26-1: storeupdate inpackage_windows.go(+test 직접 작성)·storeupdate_windows.go·tools/storecheck, 2026-11-23-store-update.md 주석
- 26-2: internal/app/channels_import.go
- 26-3: tools/build-config.ps1(BOM)·installer.ps1·tools/updatecheck
- 27-1: 코드·문구·명세 반영. Automation.test.tsx 는 27-5 의 전체본으로 만든다
- 27-2: snippetVars·VarsDialog·문구·명세. SnippetsPanel 연결은 27-5 의 전체본(그 안에 포함), app.css 는 27-1 조각에 이미 포함
- 27-3: 반영(ssh_app 의 password 는 27-1 조각에 이미). macro.test·Automation.test 몫은 27-5 전체본
- 27-4: 반영. 깨진 diff 첫 덩어리 대신 TerminalView 의 beep 함수·SettingsAppearance 의 SettingsHighlight 를 손으로 뺌. App.tsx 의 startMacros 는 27-5 에서 state/macros 가 생겨야 tsc 통과
- 27-5: 반영(SnippetsPanel·Automation.test 전체본, shell.snippets 값 변경은 따로 반영)
- 27-6: host-settings README·05·setup·check 반영, 06 문서 새로 씀(이 PC 는 gcc 16.1.0 이미 있음, 설치 안 함)
- 27-7: 설명 보고 구현 — channels_clip app.notify, main.go Notify, store/defaults/types triggerOsNotify, 계약 json 재생성, triggers.ts notifyOs, SettingsRules 토글, 문구 3개(10개 언어 직접 번역), 명세 4곳, state/followups.test.ts 새로. KI-013(clipboard 흔들림)은 계획서에 정의 없음 → 전체 검증 때 판단
- 27-8: macros.ts runMacroOnTargets·quiet, SnippetsPanel execute, 문구 2, 명세 terminal §11, followups.test D146 4개
- 27-9: keys.ts·actions.ts·macros.ts(lastRun·replayLastMacro·stopMacrosOnTargets), 문구 6, 명세 shortcuts §5.1·terminal §11, keys.test·SettingsShortcuts.test 수정, followups D147 3개
- 27-10: session loginscript(SecretId·feedSteps·ErrSecretLocked)·types ResolveSecret, app login_secret.go(새)·channels_session, loginsecret_test.go(새 4개), 프런트 types·loginScript·LoginScriptTable·HostEditor, 문구 9, 명세 3
- 27-11: triggers currentRules(sessionId)·hostCompiled, types Host.triggerRules, hostSave cleanHostTriggers, HostEditor 자동화 탭(모든 프로토콜)·TriggerRuleList(SettingsRules 에서 분리), 문구 3, 명세 4, Automation.test 1개 수정, followups D149 2개
- 27-12: importer securecrt(crtHasLoginScript·joinNote)·types(LoginStep)·apply(loginScript), loginscript_import_test.go(새 2개), 문구 1, 명세 import.md
- 27-13: sshconn forward_count(forwardStats)·keepalive_forward(Traffic)·test, session ForwardInfo up/down, ssh_app, 프런트 types·forwards.ts(trafficTotal)·ForwardsDialog, 문구 2, 명세 3
- 27-14: sshconn proxy_command(.go/_windows/_unix, Job 객체로 트리 종료 — 계획서보다 보강)·proxy.go·client.go·test(통합 포함), app proxyRecord·buildProxy, 프런트 hostForm·hostSave·HostEditor·HostEditorParts, 문구 8, 명세 3
- 27-15: importer types(jumpChain·ProxyDraft)·sshconfig·apply(linkJumpChain·proxy), jumpchain_test.go(새 4개), c1types, 문구 1, 명세 import·ipc
- 27-16: app lifecycle_windows(세션 잠금·CloseDesktop)·lifecycle_windows_test(새, 이 PC PASS), 명세 lifecycle §2, 11-17 계획서 안 C 체크
- 27-17~19: 한 흐름으로 반영 — help.ts(-18 부록 A + -19 크기), HelpDialog(-19 부록 A), Modal(-19), helpCapture·main.tsx·help-capture.mjs·playwright-core, ui.ts help 상태, Shell 버튼, Settings set-side·이 구역 도움말, App, settings.css(기본 CSS 직접), 그림 24장 복사, 문구 14, **본문 ko/en 은 계획서에 없어 직접 씀**, help.test.ts 직접 씀, 명세 help.md(새)·shell·settings·dialogs·README. 전체 vitest 500 통과
- 27-20: session traffic.go(+test)·session/control/loginscript/zmodem/manager, app.go, 프런트 traffic.ts·backend·actions·workspace·format·Panes·app.css·fakeBackend, Panes.traffic.test, 문구 1, 도움말·설명서·시나리오 diff, 그림 2장, 매뉴얼 html/pdf 다시 빌드(html 은 첨부와 동일), 명세 ipc·session·panes §2.7
