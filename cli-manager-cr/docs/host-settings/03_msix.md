# 03. Microsoft Store 패키지(MSIX) 빌드

> 2026-11-14. 결정 D81. 스토어 제출 절차는 [../store/README.md](../store/README.md).

## 무엇을

- **새로 설치한 것 없음.** 이미 있던 것을 쓴다:
  - Windows SDK 10.0.26100 (`winget` `Microsoft.WindowsSDK.10.0.26100`): `makeappx.exe`, `signtool.exe` (`C:\Program Files (x86)\Windows Kits\10\bin\<버전>\x64\`). `tools/msix.ps1` 이 가장 새 버전을 찾아 쓴다.
  - FFmpeg 7.1.1 (`winget` `Gyan.FFmpeg`): `build/appicon.png` 를 타일 크기(44·50·71·150, 310×150)로 줄인다.
- **새로 만든 것**: 시험 서명용 자체 서명 인증서(`-TestSign` 을 처음 실행할 때만)
  - 위치 `Cert:\CurrentUser\My`, 이름(FriendlyName) `DariTerm MSIX test`
  - 주체 `CN=0AEF544B-0B89-41A6-861B-39FF4DB0031F` (매니페스트 Publisher 와 같아야 함)
  - 2026-11-14 생성, 만료 2027-11-14, 지문 `374716C5C40D1C635CDC8D0B84CBF4D53461C248`
  - 스토어 업로드에는 쓰지 않는다. 스토어가 자기 인증서로 다시 서명한다.

## 쓰는 법

```powershell
# 저장소 cli-manager-cr 에서
make msix                                                    # 스토어 업로드용 (서명 없음)
powershell -ExecutionPolicy Bypass -File tools\msix.ps1 -TestSign   # + 이 PC 시험 설치용 서명본
```

결과(`build\bin\`):

| 파일 | 용도 |
|---|---|
| `DariTerm-<버전>.0.msix` | Partner Center 에 올리는 파일 (서명 없음) |
| `DariTerm-<버전>.0-test.msix` | 시험 설치용, 자체 서명 (`-TestSign`) |
| `DariTerm-test.cer` | 위 서명의 공개 인증서 (`-TestSign`) |

버전은 `wails.json` 의 `info.productVersion`(x.y.z) 에 `.0` 을 붙인다. 스토어는 올릴 때마다 버전이 커져야 하므로 다음 제출 전에 `productVersion` 을 올린다.

## 이 PC 에서 시험 설치 (관리자 권한 필요 — 직접 실행)

```powershell
# 관리자 PowerShell. 시험 인증서를 "신뢰할 수 있는 사람"에 등록 (한 번만)
Import-Certificate -FilePath build\bin\DariTerm-test.cer -CertStoreLocation Cert:\LocalMachine\TrustedPeople
# 일반 PowerShell 에서 설치
Add-AppxPackage build\bin\DariTerm-0.1.0.0-test.msix
```

- 설치 후 시작 메뉴의 DariTerm 으로 실행한다.
- 스토어판 데이터: MSIX 는 `%APPDATA%` 에 새로 만드는 파일을 앱 전용 위치(`%LOCALAPPDATA%\Packages\YangHunLee.DariTerm_bmtfree9a9kzr\LocalCache\Roaming\DariTerm\`)로 돌린다. 이 PC 처럼 NSIS 설치판 데이터(`%APPDATA%\DariTerm\data`)가 이미 있으면 그 파일을 제자리에서 읽고 고친다. 시험 전에 NSIS 판 데이터를 백업해 두거나 옮겨 둔다.
- 지우기: `Get-AppxPackage YangHunLee.DariTerm | Remove-AppxPackage`. 인증서는 `certlm.msc` > 신뢰할 수 있는 사람에서 `0AEF544B-…` 를 지운다.

## 되돌리기

- 시험 인증서 삭제: `Get-ChildItem Cert:\CurrentUser\My | Where-Object FriendlyName -eq 'DariTerm MSIX test' | Remove-Item`
