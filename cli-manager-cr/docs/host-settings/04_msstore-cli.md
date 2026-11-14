# 04. Microsoft Store Developer CLI (msstore)

> 2026-11-14 설치. 결정 D82. 쓰는 법은 [../store/README.md](../store/README.md) §3.

## 무엇을

- **Microsoft Store Developer CLI 0.4.4** — 스토어 앱으로 설치(`winget install --id 9P53PC5S0PHJ --source msstore`).
  - 패키지 `Microsoft.MicrosoftStoreCLI` (PFN `Microsoft.MicrosoftStoreCLI_8wekyb3d8bbwe`)
  - 실행 파일 `%LOCALAPPDATA%\Microsoft\WindowsApps\MSStore.exe` (명령 이름 `msstore`)
  - Microsoft 문서는 .NET 9 Desktop Runtime 을 요구한다고 하지만, 스토어판은 그것 없이 실행됐다(이 PC 에는 .NET 8·10 만 있음). 그래서 .NET 9 는 설치하지 않았다.
- **자격 증명**: 설정하지 않음 — **CLI 자동 제출은 보류**(Entra 테넌트에 결제 계정이 필요, D82). 켤 때 사용자가 직접 `msstore reconfigure --tenantId … --sellerId … --clientId … --clientSecret …` 를 실행한다(값은 기록하지 않는다).
  - 설정 전 `msstore info` 는 `SellerId is not set` 으로 종료 코드 -1 → `tools/store-publish.ps1` 이 이걸로 미설정을 알아챈다.

## 확인

```powershell
msstore --version      # 0.4.4.3+…
msstore info           # 설정 후: 테넌트·판매자·클라이언트 ID 표시
msstore apps list      # 설정 후: DariTerm(9N8WBPMRNBWV) 이 보이면 정상
```

## 되돌리기

- 자격 증명만 지우기: `msstore reconfigure --reset`
- 제거: `winget uninstall --id 9P53PC5S0PHJ` (또는 설정 > 앱에서 "Microsoft Store Developer CLI" 제거)
- Partner Center 쪽: 계정 설정 > 사용자 관리 > Microsoft Entra 응용 프로그램에서 키 삭제 또는 앱 제거
