# Windows 설치 파일 만들기 (NSIS)

**Status**: done
**Started**: 2026-11-14 13:10
**Last update**: 2026-11-14 13:10
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "파일 하나만 복사해서 쓰기 불편하다. 윈도우에서 설치할 수 있도록 설치 파일로."
- 방식(D76, 권장안): Wails 내장 NSIS(`wails build -nsis`) → `build/bin/DariTerm-amd64-installer.exe`. 시작 메뉴·바탕화면 바로가기, 프로그램 추가/제거 등록·제거 프로그램, WebView2 없으면 설치. 설치 화면은 10개 언어(OS 언어 기본) + 사용권(LICENSE) 동의 화면.
- 데이터(%APPDATA%\DariTerm\data)는 설치·제거와 무관(제거해도 남음).

## Plan

- [x] 1. NSIS 설치(winget) + docs/host-settings 기록(번호 문서·README·패키지 목록·setup.ps1·점검 스크립트)
- [x] 2. wails.json info(회사·제품·버전·저작권)
- [x] 3. `wails build -nsis` 로 build/windows/installer 템플릿 생성 → 언어·사용권 화면 손보기
- [x] 4. 빌드 확인(설치·제거는 관리자 권한이라 사용자가 확인)
- [x] 5. Makefile 대상, 문서(decisions D76, CLAUDE.md 빠른 명령), .gitignore 확인

## Files touched

## Resume hint

1단계부터.
