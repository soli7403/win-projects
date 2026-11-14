# 01. Windows 설치 파일 빌드 (NSIS)

> 2026-11-14 설치. 결정 D76.

## 무엇을

- **NSIS 3.13** (`winget install --id NSIS.NSIS -e`). 설치 위치 `C:\Program Files (x86)\NSIS` (`makensis.exe`).
  - winget 설치는 시스템 PATH 에 넣지 않으므로 `tools/installer.ps1` 이 기본 위치를 PATH 에 붙여 쓴다.
- 이미 있던 것(이 문서에서 설치하지 않음): Go, Node.js, Wails CLI v2.10.

## 쓰는 법

```powershell
# 저장소 cli-manager-cr 에서
make installer
# 또는
powershell -ExecutionPolicy Bypass -File tools\installer.ps1
```

결과: `build\dist\DariTerm-<버전>-amd64-installer.exe` (`make installer VERSION=x.y.z` 면 버전을 올린 뒤). `build\bin\` 은 빌드 중간물이라 다음 빌드 때 비워지고, `build\dist\` 는 MSIX·이전 버전과 함께 남는다(D83).

## 설치 파일 구성 (build/windows/installer/project.nsi)

- 설치 위치 `C:\Program Files\DariTerm` (관리자 권한 필요), 시작 메뉴·바탕화면 바로가기, 프로그램 추가/제거 등록(키 `DariTerm`), 제거 프로그램.
- 게시자·저작권 표기는 `YangHun Lee` (여권 영문 이름. 평소 이름 kai 는 설치 정보에 넣지 않는다, 사용자 지시 2026-11-14).
- 첫 설치 파일이 쓰던 옛 위치(`Program Files\kai\DariTerm`)와 옛 등록 키(`kaiDariTerm`)는 새 설치 때 지운다.
- WebView2 런타임이 없으면 설치한다.
- 화면 언어: Windows 표시 언어에 맞춰 10개 중 자동(없으면 영어). 사용권 동의 화면은 저장소 `LICENSE`.
- 앱이 실행 중이면 닫을 때까지 [다시 시도]/[취소].
- 제거해도 사용자 데이터(`%APPDATA%\DariTerm\data`)는 남긴다. WebView2 캐시(`%APPDATA%\dariterm.exe`)는 지운다.
- 제품 정보(이름·회사·버전·저작권)는 `wails.json` 의 `info`. 버전을 올릴 때는 `wails.json` 의 `productVersion` 과 Go `G_cszAppVersion`(internal/app/app.go)을 함께 바꾼다.

## 되돌리기

`winget uninstall --id NSIS.NSIS -e`
