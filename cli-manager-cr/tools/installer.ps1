# DariTerm Windows 설치 파일 만들기 (결정 D76).
# 결과: build\bin\DariTerm-amd64-installer.exe (+ build\bin\dariterm.exe)
# 필요: Wails CLI, NSIS 3 (makensis). 설치 방법은 docs/host-settings/01_installer-build.md.
# 사용: 저장소 루트에서  powershell -ExecutionPolicy Bypass -File tools\installer.ps1   (또는 make installer)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# makensis 가 PATH 에 없으면 기본 설치 위치를 붙인다.
if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
    $nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS'
    if (-not (Test-Path (Join-Path $nsis 'makensis.exe'))) {
        throw "NSIS(makensis)를 찾을 수 없습니다. docs/host-settings/01_installer-build.md 를 보고 설치하세요."
    }
    $env:PATH = "$env:PATH;$nsis"
}

# 사용권 동의 화면 원문: LICENSE 를 UTF-8 BOM 으로 복사(NSIS 가 한글을 깨지 않게).
$res = Join-Path $root 'build\windows\installer\resources'
New-Item -ItemType Directory -Force $res | Out-Null
$text = [System.IO.File]::ReadAllText((Join-Path $root 'LICENSE'), [System.Text.Encoding]::UTF8)
[System.IO.File]::WriteAllText((Join-Path $res 'license.txt'), $text, (New-Object System.Text.UTF8Encoding($true)))

wails build -clean -nsis
if ($LASTEXITCODE -ne 0) { throw "wails build 실패 ($LASTEXITCODE)" }
Get-Item (Join-Path $root 'build\bin\DariTerm-amd64-installer.exe') | Select-Object FullName, Length, LastWriteTime
