# 2026-11-23 import-plans 14건 순서대로 적용

**Status**: done
**Started**: 2026-11-23
**Last update**: 2026-11-23
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: `docs/import-plans/2026-11-23-*.md` 를 **반드시 아래 순서대로** 적용.
- 순서·결정 번호: 0 import-1118-features(D96~D100) → 1 ppk-keys(D106) → 2 pageant-agent(D107) → 3 serial-flow-control(D108) → 4 socks-dynamic-forward(D109) → 5 proxy-connect(D110) → 6 agent-forwarding(D111) → 7 log-line-timestamp(D112) → 8 keyword-highlight(D113) → 9 zmodem(D114) → 10 store-update(D115) → 11 remove-grid-layout(D116) → 12 remove-location-picker(D117) → 13 encodings-gbk-big5-eucjp(D118)
- 번호 충돌: decisions.md 에 이미 D106(1.0.0)·D107(기간제 키 발급) 있음 → 2026-11-11 선례대로 파일 순서 `-1`/`-2` 접미 (D106-1 = 1.0.0, D106-2 = PPK, D107-1 = 키 발급, D107-2 = Pageant).

---

## Plan

- [x] 0. import-1118-features (D96~D100) — 코드 37파일·문서(decisions D96~D100, KI-011, 시나리오 6-11·12-1a, CLAUDE.md). CHANGELOG·listing 은 14단계에서 한꺼번에
- [x] 1. ppk-keys (D106-2) — CHANGELOG.md·frontend/package.json "삭제"는 적용 안 함. 기존 D106·D107 → -1, 번호 안내 추가
- [x] 2. pageant-agent (D107-2) — agent_*.go 는 끝 빈 줄 문제로 hunkapply.py 로 적용
- [x] 3. serial-flow-control (D108)
- [x] 4. socks-dynamic-forward (D109)
- [x] 5. proxy-connect (D110)
- [x] 6. agent-forwarding (D111)
- [x] 7. log-line-timestamp (D112)
- [x] 8. keyword-highlight (D113)
- [x] 9. zmodem (D114)
- [x] 10. store-update (D115)
- [x] 11. remove-grid-layout (D116)
- [x] 12. remove-location-picker (D117)
- [x] 13. encodings-gbk-big5-eucjp (D118)
- [x] 14. 전체 go vet·go test ./...·tsc·vitest(43파일 279개)·wails build 통과, CHANGELOG 1.1.0(미배포)·listing·CLAUDE.md 정리

---

## Files touched

- 코드·명세·문구: 각 계획서 §A 표 그대로 (예외: ppk-keys 의 CHANGELOG.md·frontend/package.json 삭제는 적용 안 함)
- 문서: docs/decisions.md(D96~D100, D106-2~D118, D106/D107 → -1, 번호 안내), .claude/known-issues.md(KI-011), ../CLAUDE.md(KI-012), CLAUDE.md, CHANGELOG.md(1.1.0), docs/store/listing.md, docs/scenarios(.en).md 6-11·12-1a, docs/releases/1.0.0.md·docs/store/README.md(D106-1)

---

## Notes / decisions

- [2026-11-23] 각 단계마다 go build/vet/test + tsc + vitest(PowerShell) 확인 후 다음 단계.

---

## Resume hint

- 완료. 남은 일은 각 계획서 §H 의 앱 수동 검증(특히 ZMODEM 은 실제 lrzsz 와 미대조)과 커밋(사용자 요청 시).

- 다음 단계: Plan 의 첫 `[ ]` 항목 계획서를 읽고 적용.
- 주의: 계획서의 "삭제" 항목(예: ppk-keys 의 CHANGELOG.md·frontend/package.json 삭제)은 작업 폴더 차이에서 생긴 것일 수 있으니 확인 후 적용.
