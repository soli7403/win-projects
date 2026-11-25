# 빌드 PC 준비(멱등): 없으면 설치한다. 목록은 README.md 의 패키지 목록과 같다.
$ErrorActionPreference = 'Stop'

# Build toolchain (05_build-toolchain.md): Go 1.27.4, Wails CLI v2.17.1 (same as go.mod)
$goWant = 'go1.27.4'
$goNow = if (Get-Command go -ErrorAction SilentlyContinue) { (go env GOVERSION) } else { '' }
if ($goNow -eq $goWant) {
    Write-Host "Go: already $goWant"
} elseif ($goNow) {
    winget upgrade --id GoLang.Go -e --version 1.27.4 --silent --accept-source-agreements --accept-package-agreements
} else {
    winget install --id GoLang.Go -e --version 1.27.4 --silent --accept-source-agreements --accept-package-agreements
}
$wailsWant = 'v2.17.1'
$wailsNow = if (Get-Command wails -ErrorAction SilentlyContinue) { (wails version 2>$null | Select-Object -First 1) } else { '' }
if ($wailsNow -eq $wailsWant) {
    Write-Host "Wails CLI: already $wailsWant"
} else {
    go install "github.com/wailsapp/wails/v2/cmd/wails@$wailsWant"
}

$nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
if (Test-Path $nsis) {
    Write-Host "NSIS: already installed ($nsis)"
} else {
    winget install --id NSIS.NSIS -e --silent --accept-source-agreements --accept-package-agreements
}

# MSIX (03_msix.md): Windows SDK(makeappx, signtool), FFmpeg
$sdk = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin" -Directory -ErrorAction SilentlyContinue | Where-Object { Test-Path (Join-Path $_.FullName 'x64\makeappx.exe') }
if ($sdk) {
    Write-Host "Windows SDK: already installed"
} else {
    winget install --id Microsoft.WindowsSDK.10.0.26100 -e --silent --accept-source-agreements --accept-package-agreements
}
if (Get-Command ffmpeg -ErrorAction SilentlyContinue) {
    Write-Host "FFmpeg: already installed"
} else {
    winget install --id Gyan.FFmpeg -e --silent --accept-source-agreements --accept-package-agreements
}

# Store update publishing (04_msstore-cli.md): Microsoft Store Developer CLI. Credentials are set by the user (msstore reconfigure).
$msstore = Join-Path $env:LOCALAPPDATA 'Microsoft\WindowsApps\MSStore.exe'
if (Test-Path $msstore) {
    Write-Host "Microsoft Store Developer CLI: already installed"
} else {
    winget install --id 9P53PC5S0PHJ --source msstore --silent --accept-source-agreements --accept-package-agreements
}
