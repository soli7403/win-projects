# 호스트 설치·설정 기록 (cli-manager-cr)

이 PC(개발·빌드 PC)에 설치하거나 바꾼 것을 모두 적는다(공통 규칙 ../../../CLAUDE.md §8). 비밀값은 넣지 않는다.

| 번호 | 문서 | 내용 | 상태 |
|---|---|---|---|
| 01 | [01_installer-build.md](./01_installer-build.md) | Windows 설치 파일 빌드용 NSIS | 설치됨 |
| 02 | [02_license-signing-key.md](./02_license-signing-key.md) | 업무용 라이선스 서명 비밀키 위치(값은 기록하지 않음) | 생성됨 — 2026-11-14 이메일 백업 |
| 03 | [03_msix.md](./03_msix.md) | Microsoft Store 패키지(MSIX) 빌드 — Windows SDK·FFmpeg(기존), 시험 서명 인증서 | 인증서 생성됨 |
| 04 | [04_msstore-cli.md](./04_msstore-cli.md) | Microsoft Store Developer CLI(msstore) — 업데이트 자동 제출 | 설치됨, 보류(자격 증명 미설정 — D82) |
| 05 | [05_build-toolchain.md](./05_build-toolchain.md) | 빌드 도구 Go · Wails CLI · Node.js 버전과 의존성 올리는 법 | Go 1.27.4 · Wails v2.17.1 (2026-11-25) |

## 패키지 목록

| 관리자 | 패키지 | 버전 | 용도 | 문서 |
|---|---|---|---|---|
| winget | `NSIS.NSIS` | 3.13 | `wails build -nsis` 의 makensis (설치 파일 만들기) | 01 |
| winget | `Microsoft.WindowsSDK.10.0.26100` | 10.0.26100.7705 | makeappx·signtool (MSIX 만들기·시험 서명). 이미 있었음 | 03 |
| winget (msstore) | `9P53PC5S0PHJ` (Microsoft Store Developer CLI) | 0.4.4 | `msstore publish` 로 스토어 업데이트 제출 | 04 |
| winget | `Gyan.FFmpeg` | 7.1.1 | 타일 아이콘 크기 변환. 이미 있었음 | 03 |
| winget | `GoLang.Go` | 1.27.4 | Go 빌드 도구(관리자 권한 MSI) | 05 |
| go install | `github.com/wailsapp/wails/v2/cmd/wails` | v2.17.1 | `wails build` (go.mod 의 wails 버전과 같게) | 05 |

## 스크립트

- [setup.ps1](./setup.ps1): 위 패키지를 없을 때만 설치한다(여러 번 실행해도 같다).
- [check.ps1](./check.ps1): 설치 상태를 점검한다.

## 변경 이력

| 날짜 | 내용 |
|---|---|
| 2026-11-14 | NSIS 3.13 설치(winget), 설치 파일 빌드 도입 (D76) |
| 2026-11-14 | 라이선스 서명 키 쌍 생성(비밀키는 `%USERPROFILE%\.dariterm-signing\`, D78) |
| 2026-11-14 | Microsoft Store Developer CLI 0.4.4 설치(winget msstore 원본, D82). .NET 9 는 필요 없어 설치 안 함 |
| 2026-11-14 | MSIX 시험 서명 인증서 생성(`Cert:\CurrentUser\My`, "DariTerm MSIX test", D81) |
| 2026-11-25 | Go 1.26.2 → 1.27.4 (winget), Wails CLI v2.10.2 → v2.17.1 (go install), 빌드 도구 문서 05 추가 (D124) |
