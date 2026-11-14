# 수익화 최소안: 개인 무료 · 업무용 유료 사용 조건 + 정보 화면 링크

**Status**: done
**Started**: 2026-11-14
**Last update**: 2026-11-14
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: 수익화 권장안 중 최소안부터. 기술적 잠금 없이(신뢰 방식) 사용 조건과 구매 경로만.
- 범위: LICENSE(개인 무료 / 업무용 사용자별 라이선스, 평가 30일), 정보 화면(사용 조건 요약·전문, [구매하기]·[후원하기]·[홈페이지]), 명세, D73.
- 주소(구매·후원·홈페이지)는 미정 → 빌드 설정(frontend/.env 의 VITE_DARITERM_*_URL)으로 받고 비면 버튼 숨김.

## Plan

- [x] 1. LICENSE 개정(한·영)
- [x] 2. frontend/src/lib/links.ts + 정보 화면(사용 조건 요약·전문, 링크 버튼)
- [x] 3. 테스트(SettingsAbout)
- [x] 4. 명세 ui/settings.md §8, D73, CLAUDE.md 라이선스 문장
- [x] 5. tsc·vitest·wails build

## Files touched
- LICENSE, CLAUDE.md, docs/decisions.md(D73), spec/ui/settings.md §8
- frontend/src/lib/links.ts(신규), screens/SettingsAbout.tsx·.test.tsx, vite.config.ts, frontend/.env.example(신규)

## Notes / decisions

- 평가 기간 30일은 흔한 관례로 넣은 기본안 — 사용자 확인 필요.

## Resume hint

- 완료. tsc·vitest 179·wails build 통과. 남은 결정: 주소(구매·후원·홈페이지), 평가 30일·영리 개인 사용 범위 확인.
