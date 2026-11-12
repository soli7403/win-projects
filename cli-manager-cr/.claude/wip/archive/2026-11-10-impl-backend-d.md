# 백엔드 D 구현 — 가져오기·볼트 §8·동기화·백업·업데이트·수명주기

**Status**: done (미완 항목은 Resume hint 참고)
**Started**: 2026-11-10 23:30
**Last update**: 2026-11-10 (최종)
**Active agent**: impl-backend-d (subagent)
**Branch**: -

---

## Context
명세(`spec/`)만 보고 백엔드 D 영역을 구현했다. 기존 앱·`../cli-manager`·BrightTerm 저장소는 열람하지 않았다.
- 사용자 요청: 가져오기, 볼트 §8, 동기화, 백업, 업데이트 확인, 수명주기
- 범위 / 결정사항: 권장안 진행, 커밋·push 금지, 결정은 `docs/decisions.md` 끝(D39~D43)
- 관련 문서: [spec/import.md](../../spec/import.md), [spec/data/vault.md](../../spec/data/vault.md), [spec/data/sync-backup.md](../../spec/data/sync-backup.md), [spec/behavior/lifecycle.md](../../spec/behavior/lifecycle.md)

---

## Plan

- [x] 1. 가져오기 순수 파서 (PuTTY 값 맵, ssh config, EC2 JSON) + 단위 테스트
- [x] 2. 가져오기 레지스트리 읽기(Windows build tag) + aws CLI 실행 + 적용(import.apply)
- [x] 3. 볼트 §8 (EncryptData/DecryptData, 헤더, 항목 교체, AdoptRemote) + 테스트
- [x] 4. 동기화 병합 순수 함수 + 파일 형식 + 단위 테스트
- [x] 5. 동기화 실행(git 호출, 상태 6종) + 로컬 bare 저장소 통합 테스트 + sync.run/sync.status 채널
- [x] 6. 동기화 자동 실행(볼트 열림·20초·10분) + 이벤트
- [x] 7. 백업 export/import + 테스트
- [x] 8. 업데이트 확인(버전 비교, 가짜 HTTP 테스트) + update.json + 채널
- [x] 9. 수명주기: window.json, 종료 확인, 자동 잠금 타이머 (Windows 잠금·절전 휴리스틱, D40)
- [x] 10. go vet / go build / go test -race 통과, 인수 시나리오 테스트화

체크 마크 의미:
- `[ ]` 미시작
- `[~]` 진행 중 (paused 시 `(paused: 사유)` 추가)
- `[x]` 완료

---

## Files touched

- `.claude/wip/2026-11-10-impl-backend-d.md` (created, updated)
- `internal/importer/{types,putty,putty_windows,putty_other,sshconfig,aws,scan,apply}.go`, `importer_test.go`, `apply_test.go` (created)
- `internal/uid/uid.go`, `internal/procutil/{procutil,hide_windows,hide_other}.go` (created)
- `internal/app/channels_import.go`, `channels_sync.go`, `sync_auto.go`, `channels_backup.go`, `channels_update.go`, `lifecycle_idle.go`, `lifecycle_window.go`, `lifecycle_close.go`, `lifecycle_windows.go`, `lifecycle_other.go`, `lifecycle_test.go`, `backup_test.go` (created)
- `internal/app/app.go` (줄 단위: 필드 4개, 훅 4곳, register 호출, Close 의 stop 호출)
- `internal/vault/sync.go`, `sync_test.go`, `backup.go` (created, §8 및 백업 추가만)
- `internal/store/replace.go` (created, ReplaceAll 추가만)
- `internal/gitsync/{merge,format,git,sync,vaultops}.go`, `merge_test.go`, `sync_integration_test.go` (created)
- `internal/backup/{backup,backup_test}.go` (created)
- `internal/update/{update,update_test}.go` (created)
- `internal/session/list.go` (created, Snapshots 추가만)
- `main_window.go` (created, 타입 lifecycleWindow), `main.go` (줄 단위: winHost 생성·옵션·OnDomReady·OnBeforeClose)
- `docs/decisions.md` (D39~D43 추가), `spec/QUESTIONS.md` (Q22~Q23 추가)

---

## Notes / decisions

- 검증: `go vet ./...` 통과, `go test -race ./...` 전부 ok (vault 약 85초).
- 통합 테스트: 로컬 bare 저장소로 Y5, Y6, Y7, Y8, Y9, Y4 시나리오 통과. 가져오기 I1~I9 는 파서·적용 단위 테스트로 옮김.
- main.go 의 기존 `wailsWindow`(창 제어, 다른 에이전트)와 이름 충돌을 피해 내 타입은 `lifecycleWindow`.

---

## Resume hint

- 미완: Y10(20초 자동 동기화 실제 시간 시험), X1~X3(레이아웃 복원은 프런트 담당), X7(첫 실행 가져오기 제안 신호 미구현), 창 위치 다중 모니터(Q23), 실기 Windows 잠금·절전 감지(D40) 검증.
- 질문: QUESTIONS Q22(새 PC 받기 전 잠금 절차), Q23(창 위치 기준).
- 주의: 자동 동기화 연결은 `internal/app/sync_auto.go`, 종료 확인과 창 상태는 `main_window.go`.
