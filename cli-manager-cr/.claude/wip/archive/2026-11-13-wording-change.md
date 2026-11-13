# 화면 문구를 원본과 다른 일반 용어로 바꾸기 (D60)

**Status**: done
**Started**: 2026-11-13
**Last update**: 2026-11-13
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: 원본과 메뉴·단어가 같은 뜻의 일반 용어로 다르게 보이도록. 제안서 [docs/wording-proposal.md](../../docs/wording-proposal.md) 를 사용자가 "제안처럼" 승인.
- 방식: 스크립트로 문자열 리터럴만 바꾼다 — 통째로 일치하는 문구는 표(EXACT), 나머지는 용어 사전(조사 포함 규칙). 저장 데이터 형식·값은 그대로.
- 스크립트: scratchpad `wording.py` (세션 임시) — 표는 제안서 §1~§11.

## Plan

- [x] 1. 변환 스크립트 작성 (EXACT 표 + 용어 사전)
- [x] 2. frontend/src (테스트 포함) 적용 → 남은 옛 용어 grep
- [x] 3. spec/ 의 따옴표·백틱 안 문구 적용
- [x] 4. tsc · vitest (기존 실패 3건 외 새 실패 없음) · wails build
- [x] 5. 결정 D60, 제안서 상태 "적용됨"

## Files touched

- spec/ 24개 문서(따옴표·백틱·대괄호 안), internal/ Go 메시지 22개 파일(aws.go 검색어 "배스천" 은 유지), docs/decisions.md D60, spec/README.md 메모
- "칸"이 입력란·빈칸 뜻인 2곳(file-panel 경로칸, format.test 빈 칸)과 Go 테스트 "들여쓰기 1칸"은 되돌림
- frontend/src 51개 파일(테스트 포함), Sidebar.tsx 수동 3곳, Lock/RightPanel 테스트 기존 실패 3건 문구 맞춤

## Resume hint

스크립트가 사라졌으면 제안서 표로 다시 만든다. 적용은 git diff 로 확인.
