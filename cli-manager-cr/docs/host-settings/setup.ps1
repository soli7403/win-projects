# 빌드 PC 준비(멱등): 없으면 설치한다. 목록은 README.md 의 패키지 목록과 같다.
$ErrorActionPreference = 'Stop'

$nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
if (Test-Path $nsis) {
    Write-Host "NSIS: already installed ($nsis)"
} else {
    winget install --id NSIS.NSIS -e --silent --accept-source-agreements --accept-package-agreements
}
