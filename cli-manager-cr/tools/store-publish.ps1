# DariTerm Microsoft Store 업데이트 자동 제출 (결정 D82). msstore CLI(Microsoft Store Developer CLI)로
#   [버전 올리기] → MSIX 만들기(tools\msix.ps1) → 업로드 → 제출(인증 요청)까지 한 번에 한다.
# 전제: 첫 게시가 끝난 앱(Partner Center 에서 수동, 2026-11-14), msstore 자격 증명 설정(msstore reconfigure — docs/store/README.md §3).
# 사용(저장소 cli-manager-cr 에서):
#   powershell -ExecutionPolicy Bypass -File tools\store-publish.ps1 -Version 0.1.1            # 버전 올리고 제출
#   powershell -ExecutionPolicy Bypass -File tools\store-publish.ps1 -Version 0.1.1 -NoCommit  # 초안까지만(Partner Center 에서 확인 후 제출)
#   powershell -ExecutionPolicy Bypass -File tools\store-publish.ps1 -Status                   # 제출 상태만 보기
#   또는  make store-publish VERSION=0.1.1
param(
  [string]$Version,          # 새 버전 x.y.z (wails.json info.productVersion 을 이 값으로 바꾼다). 비우면 지금 버전 그대로
  [switch]$NoCommit,         # 업로드만 하고 초안으로 둔다(인증 요청 안 함)
  [ValidateRange(0, 100)][int]$Rollout = -1, # 단계적 배포 비율(%). 비우면 전체 배포
  [switch]$Status            # 제출 상태만 보고 끝낸다
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$productId = '9N8WBPMRNBWV' # Partner Center Store ID (docs/store/README.md §0)

# msstore 찾기 (스토어판은 %LOCALAPPDATA%\Microsoft\WindowsApps\MSStore.exe)
$msstore = (Get-Command msstore -ErrorAction SilentlyContinue).Source
if (-not $msstore) { $msstore = Join-Path $env:LOCALAPPDATA 'Microsoft\WindowsApps\MSStore.exe' }
if (-not (Test-Path $msstore)) { throw 'msstore CLI 가 없습니다. docs/host-settings/04_msstore-cli.md 대로 설치하세요.' }

# 자격 증명 확인: 설정 전이면 'SellerId is not set' 으로 실패한다.
# (PowerShell 5.1 은 Stop 상태에서 네이티브 프로그램의 stderr 를 오류로 바꾸므로 이 확인만 Continue 로 돌린다)
$eapSaved = $ErrorActionPreference
$ErrorActionPreference = 'Continue'
& $msstore info *> $null
$iInfoExit = $LASTEXITCODE
$ErrorActionPreference = $eapSaved
if ($iInfoExit -ne 0) {
  throw 'msstore 자격 증명이 설정되지 않았습니다. docs/store/README.md §3 의 "msstore reconfigure" 를 먼저 실행하세요.'
}

if ($Status) {
  & $msstore submission status $productId
  exit $LASTEXITCODE
}

# 1)·2) 버전 올리기(-Version, wails.json 한 곳 D82) + MSIX 만들기(앱 빌드 포함)는 msix.ps1 이 한다.
$msixArgs = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $PSScriptRoot 'msix.ps1'))
if ($Version) { $msixArgs += @('-Version', $Version) }
& powershell @msixArgs
if ($LASTEXITCODE -ne 0) { throw "msix.ps1 실패 ($LASTEXITCODE)" }
$Version = ([IO.File]::ReadAllText((Join-Path $root 'wails.json'), [Text.Encoding]::UTF8) | ConvertFrom-Json).info.productVersion
$msix = Join-Path $root "build\dist\DariTerm-$Version.0.msix"
if (-not (Test-Path $msix)) { throw "패키지가 없습니다: $msix" }

# 3) 업로드 + 제출. 이 계정은 시장 그룹 가격 방식이라 CLI 가 가격을 못 읽고 멈출 수 있어 무료(Free)를 명시한다.
#    msstore publish 는 마지막 게시본으로 새 초안을 만든다(대기 중인 초안은 지워진다) — Partner Center 에서 손댄 초안이 있으면 먼저 정리할 것.
$publishArgs = @('publish', (Split-Path -Parent $msix), '--inputFile', $msix, '--appId', $productId, '--priceId', 'Free')
if ($NoCommit) { $publishArgs += '--noCommit' }
if ($Rollout -ge 0) { $publishArgs += @('--packageRolloutPercentage', "$Rollout") }
& $msstore @publishArgs
if ($LASTEXITCODE -ne 0) { throw "msstore publish 실패 ($LASTEXITCODE)" }

& $msstore submission status $productId
Write-Host ''
Write-Host "done: DariTerm $Version submitted$(if ($NoCommit) { ' as draft (commit in Partner Center)' })."
Write-Host 'next: commit wails.json (version) and record the submission in docs/store/README.md section 5.'
