# 프런트엔드 경량화 — React 제거 (방안 A: Wails 유지, UI/동작 동일)

**Status**: in-progress
**Started**: 2026-11-10 00:00
**Last update**: 2026-11-10 00:00
**Active agent**: main
**Branch**: -

---

## Context

- 사용자 요청: Go 에 맞춰 React 로 된 기존 UI 코드를 쓰지 않고 최적화. **A 안(Wails 유지 + React 제거, xterm.js 유지)** 으로 진행. UI flow·설정·사용자 인터페이스와 동작은 **동일**해야 한다.
- 범위 / 결정사항: 백엔드(`internal/`, IPC, `window.bt` 어댑터)는 건드리지 않는다. 프런트엔드만 교체.
- 현재: React 19 + zustand + lucide-react + xterm 6 (+ addons). `frontend/src` 약 5,000줄 (components 15개, state.ts, terms.ts, layout.ts).
- 기준 번들: `dist/assets/index-CENL6BPF.js` 888,969 B, CSS 29,017 B.
- 관련 문서: [docs/decisions.md](../../docs/decisions.md)
- 영향: `frontend/` 만.

---

## Plan

- [x] 1. 기준 측정 — react 221.8KB / xterm 388.4KB + webgl 110.2KB / 앱 146.6KB / lucide 19.9KB / zustand 0.6KB (총 ≈889KB)
- [x] 2. 교체 방식 확정: Preact(+hooks, JSX) 로 런타임 교체, lucide-react → lucide-preact. decisions.md D13 기록
- [x] 3. 런타임 교체 (react/react-dom 제거) — vite/tsconfig alias 로 react→preact/compat, 타입 오류 수정(e.target→currentTarget, onDoubleClick→onDblClick, spellcheck, dataTransfer!), autoFocus 보정(`src/preact-setup.ts`)
- [~] 4. 화면별 동일성 점검 — 스모크(Edge headless + Wails 스텁): 메인·사이드바·빠른 연결(autoFocus)·서버 더블클릭→세션/xterm 확인 완료. **미확인**: HostEditor, SettingsDialog, ImportDialog, RightPanel(SFTP/스니펫), LockScreen, 우클릭 메뉴, 드래그앤드롭(분할/탭 이동), 분할 크기 조절
- [~] 5. `tsc --noEmit` OK, `vitest run` 4/4 OK, `vite build` OK, `go build ./...` OK. **남음**: `wails build` + 실제 앱 실행 확인
- [x] 6. 번들 888,969 B → 684,765 B (−204KB, −23%), gzip 192.8KB. CSS 동일(29,017 B). 시작 시간은 미측정
- [x] 7. (사용자 요청; 결과: (b) 적용 — `LICENSE`·`main.go`·정보 화면에 `Copyright (c) 2026 kai` 추가, 원저작자 줄 유지. 컴포넌트 TSX 와 Go 백엔드가 v1.1.0 이식 파생 코드라 지울 수 없음. 파생 코드를 모두 새로 쓴 뒤 교체 가능)  원래 항목: 수정 후 MIT 라이선스·원작자 표기를 kai 로 변경 — **조건부**: `LICENSE`, `main.go`, 정보 화면(SettingsDialog), CLAUDE.md/porting-brief 의 `Dany Kim` 고지. 원본(v1.1.0)에서 이식한 코드가 남아 있는 파일은 MIT 조건상 원저작자 고지를 지울 수 없으므로, 남은 파생 코드 여부를 파일별로 확인해 (a) 파생 코드 없음 → kai 로 교체, (b) 남아 있음 → `Copyright (c) 2026 kai` 를 위에 추가하고 원저작자 줄 유지. 결과를 사용자에게 보고

---

## Files touched

(작업 시 추가)

---

## Notes / decisions

- [2026-11-10] 결정: 터미널 렌더링(xterm.js, WebGL)은 유지. 병목이 React 가 아니라 xterm 이므로 터미널 성능은 그대로 두고 프레임워크 비용만 제거한다.

---

## Resume hint

- 다음 단계: Plan 4 미확인 화면 스모크(스텁 스크립트는 스크래치패드 `smoke/smoke.mjs`, `npx vite preview --port 4173` 후 실행) → Plan 5 `wails build` + 실제 실행.
- 주의: preview 서버는 포트 4173 에 띄우므로 끝나면 종료(PowerShell Get-NetTCPConnection 로 PID 확인).
- 주의사항: UI/동작 동일이 최우선. `window.bt`·IPC·`shared/types.ts` 는 변경 금지. v1.1.1+ 소스 참조 금지.
