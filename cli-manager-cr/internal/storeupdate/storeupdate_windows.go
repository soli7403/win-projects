//go:build windows

package storeupdate

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"dariterm/internal/procutil"
)

// Windows 의 스토어 기능(Windows.Services.Store.StoreContext)은 WinRT API 라서, 이 앱의 다른 OS 대화상자와 같이
// 숨긴 PowerShell 을 거쳐 부른다. 패키지된 앱이 띄운 자식 프로세스는 같은 패키지 신원을 물려받으므로
// StoreContext 가 이 앱의 것으로 동작한다. 스크립트가 실패하면 오류를 돌려줄 뿐 앱에는 영향이 없다.

const (
	// g_ciNoPackage 는 GetCurrentPackageFullName 이 "패키지된 앱이 아님"을 알리는 값(APPMODEL_ERROR_NO_PACKAGE)이다.
	g_ciNoPackage = 15700
	// G_cdurCheckTimeout 은 새 버전 확인의 시한이다.
	G_cdurCheckTimeout = 30 * time.Second
	// G_cdurInstallTimeout 은 내려받기·설치(사용자 확인 포함)의 시한이다.
	G_cdurInstallTimeout = 60 * time.Minute
)

var (
	g_procPackageFullName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
	g_procEnumWindows     = syscall.NewLazyDLL("user32.dll").NewProc("EnumWindows")
	g_procWindowThreadPid = syscall.NewLazyDLL("user32.dll").NewProc("GetWindowThreadProcessId")
	g_procIsWindowVisible = syscall.NewLazyDLL("user32.dll").NewProc("IsWindowVisible")
	g_procGetWindow       = syscall.NewLazyDLL("user32.dll").NewProc("GetWindow")
	g_procCurrentPid      = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcessId")

	g_onceStore sync.Once
	g_bStore    bool
)

// IsStoreInstall 은 이 프로세스가 패키지된 앱(Microsoft Store 설치본)으로 실행 중인지 알려 준다.
// 설치 파일(NSIS)로 설치했거나 빌드 폴더에서 바로 실행한 경우는 false 다.
func IsStoreInstall() bool {
	g_onceStore.Do(func() {
		if g_procPackageFullName.Find() != nil {
			return // Windows 8 이전: 패키지라는 것이 없다
		}
		var uiLength uint32
		rCode, _, _ := g_procPackageFullName.Call(uintptr(unsafe.Pointer(&uiLength)), 0)
		g_bStore = rCode != g_ciNoPackage && uiLength > 0
	})
	return g_bStore
}

// g_cszPreamble 은 두 스크립트가 함께 쓰는 앞부분이다: WinRT 형식을 불러오고, 비동기 작업을 기다리는 함수를 만든다.
const g_cszPreamble = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Services.Store.StoreContext, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdate, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdateResult, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdateStatus, Windows.Services.Store, ContentType = WindowsRuntime]
function Wait-Async($op, $kind, $types) {
  $m = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq $kind
  } | Select-Object -First 1
  $task = $m.MakeGenericMethod($types).Invoke($null, @($op))
  $null = $task.Wait(-1)
  $task.Result
}
$ctx = [Windows.Services.Store.StoreContext]::GetDefault()
$listType = [System.Collections.Generic.IReadOnlyList[Windows.Services.Store.StorePackageUpdate]]
`

// g_cszCheckScript 는 스토어에 이 앱의 업데이트 목록을 묻고 한 줄씩 적는다.
const g_cszCheckScript = g_cszPreamble + `$list = Wait-Async ($ctx.GetAppAndOptionalStorePackageUpdatesAsync()) 'IAsyncOperation` + "`" + `1' @($listType)
foreach ($u in $list) {
  $v = $u.Package.Id.Version
  Write-Output ("UPDATE|{0}|{1}.{2}.{3}.{4}|{5}" -f $u.Package.Id.Name, $v.Major, $v.Minor, $v.Build, $v.Revision, $u.Mandatory)
}
Write-Output ("COUNT|{0}" -f @($list).Count)
`

// g_cszInstallScript 는 업데이트를 내려받아 설치하도록 요청한다. Windows 가 사용자에게 확인 창을 보인다.
// 데스크톱 앱은 확인 창의 주인이 될 창 핸들을 먼저 알려 줘야 한다(IInitializeWithWindow). __HWND__ 자리에 핸들 값을 넣는다.
const g_cszInstallScript = g_cszPreamble + `Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
[ComImport, Guid("3E68D4BD-7135-4D10-8018-9FB6D9F33FA1"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
public interface IDtInitializeWithWindow { void Initialize(IntPtr hwnd); }
public static class DtStoreWindow {
  public static void Attach(object target, long hwnd) { ((IDtInitializeWithWindow)target).Initialize(new IntPtr(hwnd)); }
}
"@
[DtStoreWindow]::Attach($ctx, __HWND__)
$list = Wait-Async ($ctx.GetAppAndOptionalStorePackageUpdatesAsync()) 'IAsyncOperation` + "`" + `1' @($listType)
if (@($list).Count -eq 0) { Write-Output 'STATE|NoUpdate'; exit 0 }
$result = Wait-Async ($ctx.RequestDownloadAndInstallStorePackageUpdatesAsync($list)) 'IAsyncOperationWithProgress` + "`" + `2' @([Windows.Services.Store.StorePackageUpdateResult], [Windows.Services.Store.StorePackageUpdateStatus])
Write-Output ("STATE|{0}" -f $result.OverallState)
`

// runScript 는 숨긴 PowerShell 로 스크립트를 실행하고 표준 출력 줄을 돌려준다.
func runScript(szScript string, durTimeout time.Duration) ([]string, error) {
	ctxRun, fnCancel := context.WithTimeout(context.Background(), durTimeout)
	defer fnCancel()
	pCmd := exec.CommandContext(ctxRun, "powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
	procutil.HideConsole(pCmd)
	bOut, errRun := pCmd.Output()
	if errRun != nil {
		return nil, fmt.Errorf("storeupdate: %w", errRun)
	}
	var slLines []string
	for _, szLine := range strings.Split(strings.ReplaceAll(string(bOut), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(szLine) != "" {
			slLines = append(slLines, szLine)
		}
	}
	return slLines, nil
}

// Check 는 스토어에 새 버전이 있는지 묻는다. 없으면 (nil, nil) 이다.
func Check() (*Update, error) {
	if !IsStoreInstall() {
		return nil, ErrNotStore
	}
	slLines, errRun := runScript(g_cszCheckScript, G_cdurCheckTimeout)
	if errRun != nil {
		return nil, errRun
	}
	return parseCheckOutput(slLines)
}

// Install 은 Windows 의 업데이트 확인 창을 띄워 내려받기·설치를 요청하고, 끝난 상태를 돌려준다
// ("Completed", "Canceled", "NoUpdate", 그 밖에 Windows 가 알려 준 오류 상태 이름).
// 설치가 진행되면 Windows 가 앱을 닫을 수 있으므로 이 함수는 돌아오지 않을 수도 있다.
func Install() (string, error) {
	if !IsStoreInstall() {
		return "", ErrNotStore
	}
	szScript := strings.Replace(g_cszInstallScript, "__HWND__", fmt.Sprintf("%d", mainWindow()), 1)
	slLines, errRun := runScript(szScript, G_cdurInstallTimeout)
	if errRun != nil {
		return "", errRun
	}
	return parseInstallOutput(slLines)
}

// mainWindow 는 이 프로세스의 보이는 최상위 창 핸들을 찾는다 (주인 없는 창 = 앱의 주 창). 못 찾으면 0.
func mainWindow() uintptr {
	rPid, _, _ := g_procCurrentPid.Call()
	var hFound uintptr
	fnEach := syscall.NewCallback(func(hWnd uintptr, lParam uintptr) uintptr {
		var uiPid uint32
		_, _, _ = g_procWindowThreadPid.Call(hWnd, uintptr(unsafe.Pointer(&uiPid)))
		if uintptr(uiPid) != rPid {
			return 1
		}
		if rVisible, _, _ := g_procIsWindowVisible.Call(hWnd); rVisible == 0 {
			return 1
		}
		if rOwner, _, _ := g_procGetWindow.Call(hWnd, 4); rOwner != 0 { // GW_OWNER
			return 1
		}
		hFound = hWnd
		return 0 // 찾았다: 그만 돈다
	})
	_, _, _ = g_procEnumWindows.Call(fnEach, 0)
	return hFound
}
