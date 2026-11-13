# Clean-room 기록

역할 분리 상태와 위반·접촉 이력을 남긴다.

## 오염(접촉) 현황

| 대상 | 접촉 | 역할 제한 |
|---|---|---|
| 2026-11-10 세션의 Claude (명세 작성 세션) | `cli-manager/frontend/src` 일부(state.ts, App.tsx, api.ts, bt-adapter.ts, SettingsDialog.tsx 일부 등)와 `internal/vault/vault.go`, `internal/store/defaults.go`, 그리고 frontend 의 shared/types.ts, platform.ts, themes.ts, components(TitleBar·StatusBar·Welcome·PlaceDialog·QuickConnect·LockScreen·UiRequestDialog·ImportDialog·SyncPanel·ui)를 읽음. 이어서 같은 세션에서 frontend 의 모든 components·state·layout·layoutPersist·terms·styles 와 Go 백엔드(app, handlers*, window, autolock, updatecheck, syncwire, store, vault, ipc, ui, session, sshconn, transport, sftpsvc, importer, gitsync, update)를 읽고 `spec/` 전체를 작성. 프런트엔드를 Preact 로 교체하는 작업 수행 | **명세 작성자 전용.** 이 폴더의 구현 코드는 쓰지 않는다 |

| 구현자 에이전트 B(프런트), C(백엔드) — 2026-11-10 시작 | 시작 시 컨텍스트 비어 있음. `spec/` 만 읽도록 지시. 접근 제한은 지시문 기반(settings deny 미적용) | 구현 전용. 명세 수정 금지(질문만) |

| 구현자 에이전트 6명(백엔드 2·프런트 4)과 계약 점검 에이전트 1명, 2026-11-10~11 | 모두 시작 시 컨텍스트가 비어 있었고 최종 보고에서 접근 금지 영역(`cli-manager`, BrightTerm, 기존 scratchpad)을 열람하지 않았다고 보고함(자기 보고, 외부 검증 없음). 계약 점검 에이전트는 작업 중 임시 파일 2개를 scratchpad 밖 `%TEMP%` 루트에 만들었다가 삭제함(내용은 채널 이름 목록) | 구현 전용 |

| 2026-11-12 세션의 Claude | 같은 세션에서 `cli-manager` 의 `internal/sftpsvc`·`internal/session`·`internal/store/defaults.go`·`handlers_sftp.go`·frontend(RightPanel·TitleBar·Workspace·SettingsDialog·App·state·api·shared/types) 를 읽고 수정(로컬 파일 탐색·끌어 놓기 복사 기능). 이어서 `spec/behavior/file-panel.md` 작성 | 원래 명세 작성자 역할. **사용자 지시(D54)로 이후 이 폴더의 구현도 맡음** — 아래 위반 기록 참조 |

## 위반 기록

| 일자 | 내용 | 조치 |
|---|---|---|
| 2026-11-12 | 원본(`cli-manager`)을 읽은 세션이 사용자 지시(D54)로 이 폴더의 구현 코드를 쓰기 시작함 — 역할 분리 규칙과 충돌. 이 세션은 `cli-manager` 파일을 이 폴더로 복사하지 않고, 이 폴더의 기존 코드 구조와 `spec/` 에 맞춰 새로 쓴다 | 사용자 결정으로 진행. 이 세션이 쓴 부분은 배포 전 법적 확인 대상. 필요하면 해당 부분을 clean 구현자가 `spec/behavior/file-panel.md` 만 보고 다시 쓴다 |
| 2026-11-13 | 고지(D63)를 넣으려고 사용자 허락을 받아 `cli-manager/third_party/brightterm-v1.1.0/LICENSE` **한 파일만** 읽음(코드 아님, MIT 원문). 다른 파일은 열지 않음(파일 이름 확인용 목록 조회만) | 원문을 `THIRD_PARTY_NOTICES.md`·`frontend/src/lib/notices.ts` 에 그대로 옮김 |
