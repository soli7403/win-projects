# 빌드 PC 상태 점검: 필요한 도구가 있는지 보인다. 하나라도 없으면 종료 코드 1.
$ok = $true
function Check($name, $found, $detail) {
    if ($found) { Write-Host "[OK]   $name  $detail" } else { Write-Host "[MISS] $name  $detail"; $script:ok = $false }
}
$nsis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
Check 'NSIS makensis' (Test-Path $nsis) $nsis
Check 'wails' ([bool](Get-Command wails -ErrorAction SilentlyContinue)) ''
Check 'go' ([bool](Get-Command go -ErrorAction SilentlyContinue)) ''
Check 'node' ([bool](Get-Command node -ErrorAction SilentlyContinue)) ''
if (-not $ok) { exit 1 }
