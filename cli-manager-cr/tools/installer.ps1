# DariTerm Windows 설치 파일 만들기 (결정 D76).
# 결과: build\dist\DariTerm-<버전>-amd64-installer.exe
#       (wails 가 만든 build\bin\DariTerm-amd64-installer.exe 를 버전 이름으로 복사한다. build\bin 은 다음 빌드 때 비워지지만
#        build\dist 는 그대로라 MSIX(tools\msix.ps1)와 이전 버전 결과물이 함께 남는다)
# 필요: Wails CLI, NSIS 3 (makensis). 설치 방법은 docs/host-settings/01_installer-build.md.
# 사용: 저장소 루트에서  powershell -ExecutionPolicy Bypass -File tools\installer.ps1 [-Version x.y.z]
#       (또는 make installer / make installer VERSION=0.1.1 — 이 PC 의 make 는 mingw32-make)
# -Version: 버전을 올린 뒤 만든다(wails.json 한 곳, D82, 지금보다 커야 함).
param([string]$Version)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if ($Version) {
    & (Join-Path $PSScriptRoot 'set-version.ps1') -Version $Version
}

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

. (Join-Path $PSScriptRoot 'build-config.ps1')   # $DariTermLdflags: 업데이트를 묻는 릴리스 저장소 (D135)
wails build -clean -nsis -ldflags $DariTermLdflags
if ($LASTEXITCODE -ne 0) { throw "wails build 실패 ($LASTEXITCODE)" }

$ver = ([IO.File]::ReadAllText((Join-Path $root 'wails.json'), [Text.Encoding]::UTF8) | ConvertFrom-Json).info.productVersion
$dist = Join-Path $root 'build\dist'
New-Item -ItemType Directory -Force $dist | Out-Null
$out = Join-Path $dist "DariTerm-$ver-amd64-installer.exe"
Copy-Item (Join-Path $root 'build\bin\DariTerm-amd64-installer.exe') $out -Force
Write-Host "installer: $out"
