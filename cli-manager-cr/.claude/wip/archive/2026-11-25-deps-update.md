# 오픈소스 의존성 최신화 (Wails 포함)

**Status**: done
**Started**: 2026-11-25
**Last update**: 2026-11-25
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "Wails를 포함한 오픈 소스를 최신 버전으로 update"
- 범위: go.mod 직접·간접 의존성, frontend/package.json, Wails CLI. Wails 는 v2 최신(v2.17.1) — v3 는 API 가 다른 재작성이라 제외
- 영향: 전체 빌드, THIRD_PARTY_NOTICES.md (make notices)

---

## Plan

- [x] 1. Go 모듈 최신화 (go get -u, go mod tidy)
- [x] 2. Wails CLI v2.17.1 설치 + Go 1.27.4 (사용자 추가 요청) + host-settings 기록
- [x] 3. npm 패키지 최신화 (메이저 포함 시도, 깨지면 되돌림)
- [x] 4. 검증: go build / go vet / go test -race / tsc / vitest / wails build
- [x] 5. make notices, CHANGELOG·decisions 기록

---

## Files touched

- go.mod, go.sum (go 1.27.4, wails v2.17.1 외)
- frontend/package.json, package-lock.json, frontend/run-vitest.mjs (created)
- THIRD_PARTY_NOTICES.md, frontend/src/generated/third-party-notices.md (make notices)
- docs/host-settings/{05_build-toolchain.md (created), README.md, setup.ps1, check.ps1}
- docs/decisions.md (D124), CHANGELOG.md, CLAUDE.md (빠른 명령)

---

## Notes / decisions

- [2026-11-25] Wails v3 는 재작성이라 제외. Vitest 5 소문자 드라이브 문제 → run-vitest.mjs. 검증: go test -race 전체 ok, tsc ok, vitest 311/311, wails build + 실행 확인

---

## Resume hint

- 완료. 커밋은 사용자 요청 시
