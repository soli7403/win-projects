# DariTerm Microsoft Store 패키지(MSIX) 만들기 (결정 D81).
# 결과: build\bin\DariTerm-<버전>.msix (스토어 업로드용, 서명 없음 — 스토어가 서명한다)
#       -TestSign 이면 build\bin\DariTerm-<버전>-test.msix + DariTerm-test.cer (이 PC 시험 설치용 자체 서명)
# 필요: Wails CLI, Windows SDK(makeappx, signtool), ffmpeg(아이콘 크기 변환). docs/host-settings/03_msix.md
# 사용: 저장소 루트에서  powershell -ExecutionPolicy Bypass -File tools\msix.ps1 [-TestSign]   (또는 make msix)
param([switch]$TestSign)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$sdk = Get-ChildItem 'C:\Program Files (x86)\Windows Kits\10\bin' -Directory | Where-Object { Test-Path (Join-Path $_.FullName 'x64\makeappx.exe') } | Sort-Object Name -Descending | Select-Object -First 1
if (-not $sdk) { throw 'Windows SDK(makeappx)를 찾을 수 없습니다.' }
$makeappx = Join-Path $sdk.FullName 'x64\makeappx.exe'
$signtool = Join-Path $sdk.FullName 'x64\signtool.exe'

# 버전: wails.json productVersion(x.y.z) → MSIX x.y.z.0 (네 번째 자리는 스토어 규칙상 0)
$wails = Get-Content (Join-Path $root 'wails.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$ver = "$($wails.info.productVersion).0"

# 1) 앱 빌드 (스토어판도 같은 실행 파일. 업데이트 확인은 배포 저장소가 없어 꺼져 있다 — D80)
wails build -clean
if ($LASTEXITCODE -ne 0) { throw "wails build 실패 ($LASTEXITCODE)" }

# 2) 패키지 폴더 구성
$layout = Join-Path $root 'build\msix-layout'
if (Test-Path $layout) { Remove-Item -Recurse -Force $layout }
New-Item -ItemType Directory -Force (Join-Path $layout 'Assets') | Out-Null
Copy-Item (Join-Path $root 'build\bin\dariterm.exe') $layout
$icon = Join-Path $root 'build\appicon.png'
$assets = @{ 'Square44x44Logo.png' = 44; 'Square71x71Logo.png' = 71; 'Square150x150Logo.png' = 150; 'StoreLogo.png' = 50 }
foreach ($name in $assets.Keys) {
  $n = $assets[$name]
  & ffmpeg -v error -y -i $icon -vf "scale=${n}:${n}:flags=lanczos" (Join-Path $layout "Assets\$name")
}
# 넓은 타일: 가운데 아이콘(150), 양옆 투명
& ffmpeg -v error -y -i $icon -vf "scale=150:150:flags=lanczos,format=rgba,pad=310:150:80:0:color=0x00000000" (Join-Path $layout 'Assets\Wide310x150Logo.png')
$manifest = (Get-Content (Join-Path $root 'build\windows\msix\AppxManifest.xml') -Raw -Encoding UTF8).Replace('{{VERSION}}', $ver)
[IO.File]::WriteAllText((Join-Path $layout 'AppxManifest.xml'), $manifest, (New-Object Text.UTF8Encoding($false)))

# 3) 묶기
$out = Join-Path $root "build\bin\DariTerm-$ver.msix"
& $makeappx pack /d $layout /p $out /o /h SHA256
if ($LASTEXITCODE -ne 0) { throw "makeappx 실패 ($LASTEXITCODE)" }
Write-Host "store package: $out"

# 4) 시험 설치용 자체 서명(선택). 인증서 주체는 매니페스트 Publisher 와 같아야 한다.
if ($TestSign) {
  $subject = 'CN=0AEF544B-0B89-41A6-861B-39FF4DB0031F'
  $cert = Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -eq $subject -and $_.FriendlyName -eq 'DariTerm MSIX test' } | Select-Object -First 1
  if (-not $cert) {
    $cert = New-SelfSignedCertificate -Type Custom -Subject $subject -FriendlyName 'DariTerm MSIX test' -KeyUsage DigitalSignature `
      -CertStoreLocation Cert:\CurrentUser\My -TextExtension @('2.5.29.37={text}1.3.6.1.5.5.7.3.3', '2.5.29.19={text}')
  }
  $test = Join-Path $root "build\bin\DariTerm-$ver-test.msix"
  Copy-Item $out $test -Force
  & $signtool sign /fd SHA256 /sha1 $cert.Thumbprint $test
  if ($LASTEXITCODE -ne 0) { throw "signtool 실패 ($LASTEXITCODE)" }
  Export-Certificate -Cert $cert -FilePath (Join-Path $root 'build\bin\DariTerm-test.cer') | Out-Null
  Write-Host "test package: $test (trust build\bin\DariTerm-test.cer in LocalMachine\TrustedPeople first)"
}
