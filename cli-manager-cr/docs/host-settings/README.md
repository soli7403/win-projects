# 호스트 설치·설정 기록 (cli-manager-cr)

이 PC(개발·빌드 PC)에 설치하거나 바꾼 것을 모두 적는다(공통 규칙 ../../../CLAUDE.md §8). 비밀값은 넣지 않는다.

| 번호 | 문서 | 내용 | 상태 |
|---|---|---|---|
| 01 | [01_installer-build.md](./01_installer-build.md) | Windows 설치 파일 빌드용 NSIS | 설치됨 |
| 02 | [02_license-signing-key.md](./02_license-signing-key.md) | 업무용 라이선스 서명 비밀키 위치(값은 기록하지 않음) | 생성됨 — 2026-11-14 이메일 백업 |

## 패키지 목록

| 관리자 | 패키지 | 버전 | 용도 | 문서 |
|---|---|---|---|---|
| winget | `NSIS.NSIS` | 3.13 | `wails build -nsis` 의 makensis (설치 파일 만들기) | 01 |

## 스크립트

- [setup.ps1](./setup.ps1): 위 패키지를 없을 때만 설치한다(여러 번 실행해도 같다).
- [check.ps1](./check.ps1): 설치 상태를 점검한다.

## 변경 이력

| 날짜 | 내용 |
|---|---|
| 2026-11-14 | NSIS 3.13 설치(winget), 설치 파일 빌드 도입 (D76) |
| 2026-11-14 | 라이선스 서명 키 쌍 생성(비밀키는 `%USERPROFILE%\.dariterm-signing\`, D78) |
