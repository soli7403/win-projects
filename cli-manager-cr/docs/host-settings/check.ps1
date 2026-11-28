# 빌드 PC 상태 점검: 필요한 도구가 있는지 보인다. 하나라도 없으면 종료 코드 1.
$ok = $true
function Check($name, $found, $detail) {
    if ($found) { Write-Host "[OK]   $name  $detail" } else { Write-Host "[MISS] $name  $detail"; $script:ok = $false }
}
$nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
Check 'NSIS makensis' (Test-Path $nsis) $nsis
# Build toolchain (05_build-toolchain.md)
$goVer = if (Get-Command go -ErrorAction SilentlyContinue) { (go env GOVERSION) } else { '' }
Check 'go 1.27.4' ($goVer -eq 'go1.27.4') $goVer
$wailsVer = if (Get-Command wails -ErrorAction SilentlyContinue) { (wails version 2>$null | Select-Object -First 1) } else { '' }
Check 'wails v2.17.1' ($wailsVer -eq 'v2.17.1') $wailsVer
Check 'node' ([bool](Get-Command node -ErrorAction SilentlyContinue)) ''
# C compiler for go test -race (06_gcc-race.md)
$gccVer = if (Get-Command gcc -ErrorAction SilentlyContinue) { (gcc -dumpfullversion) } else { '' }
Check 'gcc (go test -race)' ([bool]$gccVer) $gccVer
# MSIX (03_msix.md)
$sdk = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin" -Directory -ErrorAction SilentlyContinue | Where-Object { Test-Path (Join-Path $_.FullName 'x64\makeappx.exe') } | Sort-Object Name -Descending | Select-Object -First 1
Check 'Windows SDK makeappx' ([bool]$sdk) $(if ($sdk) { $sdk.Name } else { '' })
Check 'ffmpeg' ([bool](Get-Command ffmpeg -ErrorAction SilentlyContinue)) ''
$cert = Get-ChildItem Cert:\CurrentUser\My | Where-Object FriendlyName -eq 'DariTerm MSIX test' | Select-Object -First 1
Write-Host ("[INFO] MSIX test cert  " + $(if ($cert) { "$($cert.Thumbprint) until $($cert.NotAfter.ToString('yyyy-MM-dd'))" } else { 'none (created by tools\msix.ps1 -TestSign)' }))
# Store update publishing (04_msstore-cli.md)
$msstore = Join-Path $env:LOCALAPPDATA 'Microsoft\WindowsApps\MSStore.exe'
Check 'msstore CLI' (Test-Path $msstore) $msstore
if (Test-Path $msstore) {
    $eapSaved = $ErrorActionPreference; $ErrorActionPreference = 'Continue'
    & $msstore info *> $null; $iInfo = $LASTEXITCODE
    $ErrorActionPreference = $eapSaved
    Write-Host ("[INFO] msstore credentials  " + $(if ($iInfo -eq 0) { 'configured' } else { 'not set (run msstore reconfigure, docs/store/README.md section 3)' }))
}
if (-not $ok) { exit 1 }
