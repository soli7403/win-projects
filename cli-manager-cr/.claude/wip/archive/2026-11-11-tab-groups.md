# 탭 그룹 2분할 (SecureCRT 방식)

**Status**: done
**Started**: 2026-11-11 22:30
**Last update**: 2026-11-11 22:30
**Active agent**: main
**Branch**: -

## Context
- 사용자 요청: 작업 영역을 좌우 2개 탭 그룹으로 나눈다. 각 그룹이 자기 탭 막대를 갖고 선택된 탭을 바로 보여 준다. 최대 2그룹, 서버(탭)는 그룹 사이를 옮길 수 있다.
- 결정: 탭에 group(0|1) 부여, 그룹별 활성 탭(activeByGroup), activeTabId 는 포커스 그룹의 활성 탭으로 유지(기존 코드 호환).
- 관련 spec: ui/shell.md §2.3(신규), data/store.md §5, acceptance/02-shell-sidebar.md

## Plan
- [x] 1. spec 갱신 (shell.md, store.md, acceptance, decisions D51)
- [x] 2. workspace.ts 상태·액션 (group, activeByGroup, moveTabToGroup, mergeGroups)
- [x] 3. layoutSave 저장/복원
- [x] 4. TabBar(group prop·메뉴·드래그), Shell(2열·드롭 영역), CSS
- [x] 5. actions.ts 탭 이동/전환 단축키를 그룹 기준으로
- [x] 6. 테스트(workspace 그룹 단위 테스트), tsc/vitest/빌드

## Files touched
- spec/ui/shell.md §2.3, spec/data/store.md §5, spec/acceptance/02-shell-sidebar.md(G1~G9), docs/decisions.md D51
- frontend/src/state/{workspace,layoutSave,actions}.ts, workspace.test.ts(신규)
- frontend/src/screens/{TabBar,Shell}.tsx, styles/app.css

## Resume hint
- 남은 일: 사용자가 실제 화면(드래그 나누기·열 안 탭 막대 모양)을 확인. Playwright 확장이 없어 화면 검증은 못 함
