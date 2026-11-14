# DariTerm 버전 올리기 (결정 D82). 버전은 wails.json 의 info.productVersion 한 곳이다.
# 새 버전은 x.y.z 형식이고 지금 버전보다 커야 한다(스토어는 더 큰 버전만 받는다). 아니면 아무것도 바꾸지 않고 멈춘다.
# 사용: powershell -ExecutionPolicy Bypass -File tools\set-version.ps1 -Version 0.1.1
#       (보통은 tools\msix.ps1 -Version / tools\installer.ps1 -Version 이 대신 부른다)
param([Parameter(Mandatory = $true)][string]$Version)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$wailsPath = Join-Path $root 'wails.json'
$wailsText = [IO.File]::ReadAllText($wailsPath, [Text.Encoding]::UTF8)
$current = ($wailsText | ConvertFrom-Json).info.productVersion
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "버전은 x.y.z 형식이어야 합니다: $Version" }
if ([version]$Version -le [version]$current) { throw "새 버전($Version)이 지금 버전($current)보다 커야 합니다." }
$wailsText = $wailsText -replace '("productVersion"\s*:\s*")[^"]*(")', "`${1}$Version`${2}"
[IO.File]::WriteAllText($wailsPath, $wailsText, (New-Object Text.UTF8Encoding($false)))
Write-Host "version: $current -> $Version (wails.json)"
