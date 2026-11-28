# 스토어판에서 [업데이트 확인] 이 늘 "확인하지 못했습니다" 로 끝나던 문제 — 원인·수정·코드·검증

**작성**: 2026-11-26 · **상태**: 코드·자동 테스트 완료(D134). 설치된 스토어 패키지(1.1.0)의 신원으로 고치기 전 실패와 고친 뒤 성공을 모두 확인했다. 앱 화면에서의 확인은 이 수정이 든 패키지를 스토어에 올린 뒤에 할 수 있다(§H) · **대상**: cli-manager-cr(DariTerm)
**앞 문서**: [2026-11-23-store-update.md](./2026-11-23-store-update.md) (스토어 업데이트 확인을 만든 문서, D115). 이 문서는 그 문서 §2 의 가정 하나가 틀렸던 것을 바로잡는다.

---

## 0. 한 줄 요약

스토어판(1.1.0)에서 설정 > 정보의 [업데이트 확인] 을 누르면 늘 "확인하지 못했습니다 (네트워크를 확인하세요)" 가 나왔다. 네트워크 문제가 아니라, **앱이 띄운 숨긴 PowerShell 이 패키지 신원을 물려받지 못해** 스토어가 요청을 거절한 것이다. 자식 프로세스를 만들 때 "패키지 안에서 실행" 정책을 주도록 고쳤다.

## 1. 증상

- Microsoft Store 로 설치한 DariTerm 1.1.0, 설정 > 정보 > [업데이트 확인] → 오류 토스트 "확인하지 못했습니다 (네트워크를 확인하세요)". 누를 때마다 같다.
- 시작 5초 뒤·12시간마다 하는 자동 확인도 같은 까닭으로 조용히 실패하고 있었다(자동 확인은 실패를 알리지 않는다). 그래서 상태 표시줄의 "새 버전" 버튼이 나올 수 없었다.
- 버튼을 눌러 띄우는 Windows 의 업데이트 확인 창(설치)도 같은 경로라 같은 까닭으로 실패했을 것이다.

## 2. 원인

### 2.1 호출 경로

1. [internal/app/channels_update.go](../../internal/app/channels_update.go) `checkUpdate` → `storeCheckResult` → `storeupdate.Check`
2. [internal/storeupdate/storeupdate_windows.go](../../internal/storeupdate/storeupdate_windows.go) `Check` → `runScript` 가 숨긴 `powershell.exe` 로 스크립트를 실행
3. 스크립트가 `StoreContext.GetDefault().GetAppAndOptionalStorePackageUpdatesAsync()` 를 부른다
4. 스크립트가 0 이 아닌 코드로 끝나면 `Check` 가 오류를 돌려주고, `storeCheckResult` 는 오류의 내용과 상관없이 상태 `error` 로 바꾼다 → 화면은 "확인하지 못했습니다" 토스트

### 2.2 틀렸던 가정

앞 문서 §2 는 "패키지된 앱이 띄운 자식 프로세스는 같은 신원을 물려받는다" 고 적었고, 코드도 그 가정으로 `os/exec` 로 PowerShell 을 띄웠다. **실제로는 물려받지 않는다.** 패키지된 데스크톱 앱이 만든 자식 프로세스는 따로 정하지 않으면 패키지 밖에서 돈다.

패키지 신원이 없는 프로세스의 `StoreContext` 는 어느 앱의 것인지 알 수 없어 스토어 호출이 `0x803F6101` 로 끝난다.

### 2.3 확인한 것 (이 PC, 설치된 스토어판 `YangHunLee.DariTerm_1.1.0.0`)

| 확인 | 결과 |
|---|---|
| 실행 중인 스토어판의 자식 프로세스 신원(`GetPackageFullName`) | 앱 본체·WebView2: 패키지 이름이 나온다. 앱이 띄운 `powershell.exe`·`conhost.exe`: `15700`(패키지 아님) |
| 패키지 신원으로 직접 띄운 PowerShell 에서 확인 스크립트 | `COUNT\|0` (정상) |
| 그 PowerShell 이 다시 띄운 자식 PowerShell 에서 같은 스크립트 | 신원 `15700`, 스토어 호출 `0x803F6101` |
| 앱과 같은 코드(`storeupdate.Check`)를 패키지 신원으로 실행 (고치기 전) | `store=true`, 0.4초 만에 `exit status 1`. PowerShell 의 오류: `$task.Wait(-1)` 에서 예외 |
| 창 숨김 방식(창 없음 / 숨김 / 그대로)·`-STA`/`-MTA` 를 바꿔 가며 | 모두 같은 `0x803F6101` — 창이나 스레드 방식 문제가 아니다 |

앞 문서가 "확인함" 이라고 적은 것은 Windows 의 시험 도구(`Invoke-CommandInDesktopPackage`)로 **PowerShell 을 직접** 패키지 안에서 띄운 것이었다. 앱 → PowerShell 의 한 단계를 건너뛴 확인이어서 이 문제가 보이지 않았다(앞 문서 §2.2 의 마지막 줄이 "이 빌드를 스토어에 올려 확인해야 한다" 로 남겨 둔 바로 그 부분이다).

### 2.4 왜 "네트워크를 확인하세요" 였나

스토어 확인의 오류는 종류를 가리지 않고 모두 `error` 한 가지로 화면에 전해지고, 그 문구가 배포 저장소(인터넷) 방식 때 만든 "확인하지 못했습니다 (네트워크를 확인하세요)" 다. 문구는 이번에 바꾸지 않았다(§J).

## 3. 수정

### 3.1 고른 방법

자식 프로세스를 만들 때 프로세스 생성 속성 `PROC_THREAD_ATTRIBUTE_DESKTOP_APP_POLICY` 에 `PROCESS_CREATION_DESKTOP_APP_BREAKAWAY_OVERRIDE`(값 4 — "만드는 프로세스를 패키지 안에서 실행, 그 프로세스 하나에만 적용") 를 준다.

| 정책 값 | 자식 PowerShell 의 신원 | 스토어 응답 |
|---|---|---|
| 1 (자손을 패키지 밖에서) | 없음(`15700`) | `0x803F6101` |
| 2 (자손을 패키지 안에서) | 있음 | `COUNT\|0` |
| **4 (이 프로세스만 패키지 안에서)** | **있음** | **`COUNT\|0`** |

4 를 골랐다: 필요한 것은 PowerShell 하나의 신원이고, 2 는 그 자손까지 번진다.

### 3.2 고르지 않은 방법

| 방법 | 고르지 않은 까닭 |
|---|---|
| Go 에서 WinRT(StoreContext)를 직접 부르기 | COM 가상 함수 표를 번호로 불러야 한다. 번호 하나가 틀리면 앱이 죽는다(앞 문서가 피한 까닭 그대로). 지금 방식은 한 군데만 고치면 된다 |
| 패키지 안의 도우미 실행 파일을 따로 두기 | 도우미도 결국 WinRT 를 직접 부르거나 PowerShell 을 띄워야 해서 같은 문제로 돌아온다 |

### 3.3 바뀐 점

- Go 의 `os/exec` 는 이 속성을 줄 방법이 없다. 그래서 `golang.org/x/sys/windows` 의 `CreateProcess` 를 직접 부르는 작은 실행 함수 `runInPackage` 를 새로 만들었다: 표준 출력·표준 오류를 파이프로 받고, 시한을 넘기면 프로세스를 끝낸다. 자식이 물려받는 핸들은 표준 입출력 셋으로 한정한다.
- `runScript` 는 `os/exec` 대신 `runInPackage` 를 쓴다. 확인 스크립트와 설치 스크립트가 같은 함수를 쓰므로 **업데이트 확인 창(설치)도 함께 고쳐진다.**
- 실패했을 때 오류에 PowerShell 의 표준 오류 출력 앞부분(400자)을 싣는다. 전에는 `exit status 1` 뿐이라 까닭을 알 수 없었다.
- 검증 도구 `tools/storecheck` 를 더했다: 앱과 같은 경로(Go 프로그램 → 숨긴 PowerShell → 스토어)를 패키지 신원으로 한 번 돌려 결과와 오류를 파일에 적는다.

화면·문구·채널(IPC)·명세는 바뀌지 않는다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/storeupdate/inpackage_windows.go` | 코드 | 새 파일 |
| `internal/storeupdate/storeupdate_windows.go` | 코드 | 수정 |
| `internal/storeupdate/inpackage_windows_test.go` | 테스트 | 새 파일 |
| `tools/storecheck/main.go` | 검증 도구 | 새 파일 |
| `docs/decisions.md` | 문서 | D134 추가 |
| `CHANGELOG.md` | 문서 | 수정 항목 추가 |
| `docs/import-plans/2026-11-23-store-update.md` | 문서 | 틀린 가정에 바로잡는 주석 |

## B. 코드

### B.1 `internal/storeupdate/storeupdate_windows.go` (수정)

```diff
 import (
-	"context"
 	"fmt"
-	"os/exec"
 	"strings"
 	"sync"
 	"syscall"
 	"time"
 	"unsafe"
-
-	"dariterm/internal/procutil"
 )
 
 // Windows 의 스토어 기능(Windows.Services.Store.StoreContext)은 WinRT API 라서, 이 앱의 다른 OS 대화상자와 같이
-// 숨긴 PowerShell 을 거쳐 부른다. 패키지된 앱이 띄운 자식 프로세스는 같은 패키지 신원을 물려받으므로
-// StoreContext 가 이 앱의 것으로 동작한다. 스크립트가 실패하면 오류를 돌려줄 뿐 앱에는 영향이 없다.
+// 숨긴 PowerShell 을 거쳐 부른다. 자식 프로세스는 그냥 띄우면 패키지 신원이 없으므로 runInPackage 로
+// 패키지 안에서 실행해야 StoreContext 가 이 앱의 것으로 동작한다 (D134). 스크립트가 실패하면 오류를 돌려줄 뿐 앱에는 영향이 없다.
```

```diff
 // runScript 는 숨긴 PowerShell 로 스크립트를 실행하고 표준 출력 줄을 돌려준다.
 func runScript(szScript string, durTimeout time.Duration) ([]string, error) {
-	ctxRun, fnCancel := context.WithTimeout(context.Background(), durTimeout)
-	defer fnCancel()
-	pCmd := exec.CommandContext(ctxRun, "powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
-	procutil.HideConsole(pCmd)
-	bOut, errRun := pCmd.Output()
-	if errRun != nil {
-		return nil, fmt.Errorf("storeupdate: %w", errRun)
-	}
+	bOut, errRun := runInPackage("powershell.exe", []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", szScript}, durTimeout)
+	if errRun != nil {
+		return nil, errRun
+	}
 	var slLines []string
```

### B.2 `internal/storeupdate/inpackage_windows.go` (새 파일, 전체)

```go
//go:build windows

package storeupdate

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 패키지된 앱이 띄운 자식 프로세스는 기본으로 패키지 밖에서 돈다(패키지 신원이 없다). 그러면 스토어가
// 0x803F6101 로 거절한다. 자식을 만들 때 "패키지 안에서 실행" 정책을 주면 같은 신원을 받는다 (D134).
const (
	// g_ciAttrDesktopAppPolicy 는 프로세스 생성 속성 PROC_THREAD_ATTRIBUTE_DESKTOP_APP_POLICY 다.
	g_ciAttrDesktopAppPolicy = 0x00020012
	// g_ciPolicyInsidePackage 는 PROCESS_CREATION_DESKTOP_APP_BREAKAWAY_OVERRIDE 다: 만드는 프로세스만 패키지 안에서 돈다.
	g_ciPolicyInsidePackage = 0x04
	// g_ciStderrLimit 은 오류 문구에 싣는 표준 오류 출력의 최대 길이다.
	g_ciStderrLimit = 400
)

// runInPackage 는 프로그램을 이 앱의 패키지 신원으로 창 없이 실행하고 표준 출력을 돌려준다.
// 시한을 넘기면 프로세스를 끝내고 오류를 돌려준다. 종료 코드가 0 이 아니면 표준 오류 출력의 앞부분을 오류에 싣는다.
func runInPackage(szExe string, slArgs []string, durTimeout time.Duration) ([]byte, error) {
	szPath, errLook := exec.LookPath(szExe)
	if errLook != nil {
		return nil, fmt.Errorf("storeupdate: %w", errLook)
	}
	pNull, errNull := os.Open(os.DevNull)
	if errNull != nil {
		return nil, fmt.Errorf("storeupdate: %w", errNull)
	}
	defer func() { _ = pNull.Close() }()
	pOutRead, pOutWrite, errOut := os.Pipe()
	if errOut != nil {
		return nil, fmt.Errorf("storeupdate: %w", errOut)
	}
	pErrRead, pErrWrite, errErr := os.Pipe()
	if errErr != nil {
		_ = pOutRead.Close()
		_ = pOutWrite.Close()
		return nil, fmt.Errorf("storeupdate: %w", errErr)
	}

	hProcess, errStart := startInPackage(szPath, slArgs, pNull, pOutWrite, pErrWrite)
	// 쓰는 쪽은 자식이 가졌다. 이쪽 것을 닫아야 자식이 끝났을 때 읽기가 끝난다.
	_ = pOutWrite.Close()
	_ = pErrWrite.Close()
	if errStart != nil {
		_ = pOutRead.Close()
		_ = pErrRead.Close()
		return nil, fmt.Errorf("storeupdate: %w", errStart)
	}
	defer func() { _ = windows.CloseHandle(hProcess) }()

	chOut := readAllAsync(pOutRead)
	chErr := readAllAsync(pErrRead)
	uiWait, errWait := windows.WaitForSingleObject(hProcess, uint32(durTimeout/time.Millisecond))
	if errWait != nil || uiWait != windows.WAIT_OBJECT_0 {
		// 읽는 고루틴은 자식이 끝나 파이프가 닫히면 스스로 끝난다.
		_ = windows.TerminateProcess(hProcess, 1)
		return nil, fmt.Errorf("storeupdate: timed out after %s", durTimeout)
	}
	var uiCode uint32
	if errCode := windows.GetExitCodeProcess(hProcess, &uiCode); errCode != nil {
		return nil, fmt.Errorf("storeupdate: %w", errCode)
	}
	bOut := <-chOut
	bErr := <-chErr
	if uiCode != 0 {
		szErr := strings.TrimSpace(string(bErr))
		if len(szErr) > g_ciStderrLimit {
			szErr = szErr[:g_ciStderrLimit]
		}
		return nil, fmt.Errorf("storeupdate: exit status %d: %s", uiCode, szErr)
	}
	return bOut, nil
}

// readAllAsync 는 파일을 끝까지 읽어 채널로 돌려주고 닫는다.
func readAllAsync(pFile *os.File) <-chan []byte {
	chData := make(chan []byte, 1)
	go func() {
		bData, _ := io.ReadAll(pFile)
		_ = pFile.Close()
		chData <- bData
	}()
	return chData
}

// startInPackage 는 표준 입출력을 주어진 파일로 잇고 "패키지 안에서 실행" 정책으로 프로세스를 만든다.
// 돌려준 프로세스 핸들은 부른 쪽이 닫는다.
func startInPackage(szPath string, slArgs []string, pIn *os.File, pOut *os.File, pErr *os.File) (windows.Handle, error) {
	slHandles := []windows.Handle{windows.Handle(pIn.Fd()), windows.Handle(pOut.Fd()), windows.Handle(pErr.Fd())}
	uiPolicy := uint32(g_ciPolicyInsidePackage)

	pList, errList := windows.NewProcThreadAttributeList(2)
	if errList != nil {
		return 0, errList
	}
	defer pList.Delete()
	if errUpdate := pList.Update(g_ciAttrDesktopAppPolicy, unsafe.Pointer(&uiPolicy), unsafe.Sizeof(uiPolicy)); errUpdate != nil {
		return 0, errUpdate
	}
	// 자식이 물려받는 핸들을 이 셋으로 한정한다.
	if errUpdate := pList.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&slHandles[0]), uintptr(len(slHandles))*unsafe.Sizeof(slHandles[0])); errUpdate != nil {
		return 0, errUpdate
	}
	pPath, errPath := windows.UTF16PtrFromString(szPath)
	if errPath != nil {
		return 0, errPath
	}
	pCmdLine, errCmdLine := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{szPath}, slArgs...)))
	if errCmdLine != nil {
		return 0, errCmdLine
	}

	var info windows.StartupInfoEx
	info.Cb = uint32(unsafe.Sizeof(info))
	info.Flags = windows.STARTF_USESTDHANDLES
	info.StdInput = slHandles[0]
	info.StdOutput = slHandles[1]
	info.StdErr = slHandles[2]
	info.ProcThreadAttributeList = pList.List()

	// 핸들을 물려줄 수 있게 바꾸는 동안 다른 프로세스 생성이 끼어들지 않게 한다.
	syscall.ForkLock.Lock()
	defer syscall.ForkLock.Unlock()
	for _, hEach := range slHandles {
		if errInherit := windows.SetHandleInformation(hEach, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); errInherit != nil {
			return 0, errInherit
		}
	}
	var procInfo windows.ProcessInformation
	uiFlags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_NO_WINDOW | windows.CREATE_UNICODE_ENVIRONMENT)
	if errCreate := windows.CreateProcess(pPath, pCmdLine, nil, nil, true, uiFlags, nil, nil, &info.StartupInfo, &procInfo); errCreate != nil {
		return 0, errCreate
	}
	_ = windows.CloseHandle(procInfo.Thread)
	return procInfo.Process, nil
}
```

### B.3 `tools/storecheck/main.go` (새 파일, 전체)

```go
// storecheck 는 스토어 업데이트 확인(internal/storeupdate)을 앱 밖에서 한 번 돌려 결과와 오류 내용을 파일에 적는 검증 도구다.
// 패키지 신원이 있어야 스토어가 답하므로 설치된 스토어판의 신원으로 실행한다 (D134):
//
//	go build -o <폴더>\storecheck.exe ./tools/storecheck
//	Invoke-CommandInDesktopPackage -PackageFamilyName 'YangHunLee.DariTerm_bmtfree9a9kzr' -AppId 'DariTerm' `
//	  -Command '<폴더>\storecheck.exe' -Args '"<폴더>\storecheck.txt"'
//
// 앱과 같은 경로(이 프로그램 → 숨긴 PowerShell → 스토어)를 타므로, 앱의 [업데이트 확인] 이 실패할 때 까닭을 볼 수 있다.
package main

import (
	"fmt"
	"os"
	"time"

	"dariterm/internal/storeupdate"
)

func main() {
	szOut := "storecheck.txt"
	if len(os.Args) > 1 {
		szOut = os.Args[1]
	}
	timeStart := time.Now()
	szText := fmt.Sprintf("store=%v\n", storeupdate.IsStoreInstall())
	pUpdate, errCheck := storeupdate.Check()
	szText += fmt.Sprintf("elapsed=%s\nupdate=%+v\nerr=%v\n", time.Since(timeStart).Round(time.Millisecond), pUpdate, errCheck)
	if errWrite := os.WriteFile(szOut, []byte(szText), 0o600); errWrite != nil {
		fmt.Fprintln(os.Stderr, errWrite)
		os.Exit(1)
	}
}
```

## C. 테스트

### C.1 `internal/storeupdate/inpackage_windows_test.go` (새 파일)

| 테스트 | 확인하는 것 |
|---|---|
| `TestRunInPackageOutput` | 자식의 표준 출력을 받는다. 따옴표·줄바꿈이 든 인자가 그대로 전해진다 |
| `TestRunInPackageFailure` | 종료 코드가 0 이 아니면 오류이고, 표준 오류 출력이 오류 문구에 실린다 |
| `TestRunInPackageTimeout` | 시한(2초)을 넘기면 프로세스를 끝내고 곧바로 오류를 돌려준다 |
| `TestRunInPackageMissingProgram` | 없는 프로그램은 오류다 |

이 테스트들은 패키지 밖(개발 PC)에서 돈다. 패키지 밖에서는 정책 속성이 아무 영향 없이 프로세스가 만들어진다는 것도 함께 확인된다(개발 빌드에서 이 함수가 불려도 깨지지 않는다 — 실제로는 `IsStoreInstall` 이 false 라 불리지 않는다).

### C.2 실행 결과 (2026-11-26)

```
go build ./...                                       → 성공
go vet ./internal/storeupdate/ ./tools/storecheck/   → 지적 없음
go test -count=1 ./internal/storeupdate/ ./internal/app/
  ok  dariterm/internal/storeupdate  2.918s
  ok  dariterm/internal/app          6.259s
GOOS=linux go build ./internal/storeupdate/ ./internal/app/ → 성공 (새 파일은 Windows 전용 build tag)
```

`go test -race` 는 이 PC 에서 cgo 가 꺼져 있어 돌리지 못했다.

## G. 패키지 신원으로 한 확인 (고치기 전·후)

`tools/storecheck` 를 설치된 스토어판(1.1.0)의 신원으로 실행했다(실행 방법은 B.3 머리 주석).

| | 결과 |
|---|---|
| 고치기 전 | `store=true` · `elapsed=429ms` · `err=storeupdate: exit status 1` (PowerShell: `$task.Wait(-1)` 에서 예외, 안쪽 오류 `0x803F6101`) |
| **고친 뒤** | `store=true` · `elapsed=956ms` · `update=<nil>` · `err=<nil>` — 스토어가 답했고 1.1.0 이 최신이라 업데이트가 없다 |

## H. 수동 검증 (남은 것)

지금 설치된 스토어판 1.1.0 에는 이 수정이 없다. **이 수정이 든 패키지(1.1.1 이상)를 스토어에 올려 설치한 뒤** 확인한다.

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 스토어판에서 설정 > 정보 > [업데이트 확인] | "최신 버전입니다" (오류 토스트가 아니다) | **사용자 확인 (2026-11-26)**: 이 수정이 든 빌드(`build/bin/dariterm.exe`)를 설치된 스토어판 1.1.0 의 패키지 신원으로 띄워 정상 동작 확인. 스토어에 올린 패키지로의 확인은 남음 |
| 2 | 그 뒤 더 새 버전을 스토어에 올리고 [업데이트 확인] | "새 버전 {버전} 이 있습니다", 상태 표시줄에 "새 버전 {버전}" 버튼 | |
| 3 | 그 버튼을 누른다 | Windows 의 업데이트 확인 창이 앱 창 위에 뜬다 | |
| 4 | 인터넷을 끊고 [업데이트 확인] | "확인하지 못했습니다 (네트워크를 확인하세요)" | |

3번(설치 창)은 여전히 실제로 띄워 보지 못했다. 신원 문제는 같은 함수로 풀렸지만, 창이 앱 창 위에 뜨는지는 새 버전이 스토어에 있을 때만 볼 수 있다. 앞 문서 §H 의 표도 그대로 유효하다.

## J. 정리 거리 (이번에 하지 않은 것)

- **오류 문구**: 스토어 확인이 실패하면 까닭과 상관없이 "네트워크를 확인하세요" 로 안내한다. 이번처럼 네트워크와 무관한 실패를 네트워크 탓으로 보이게 한다. 스토어 경로의 문구를 따로 두는 것을 생각해 볼 만하다(10개 언어 문구가 바뀌므로 따로 결정).
- **오류를 남길 곳이 없다**: `storeCheckResult` 는 오류 내용을 버린다. 앱에 진단 로그가 없어 이번에도 앱 밖에서 다시 돌려 봐야 까닭을 알 수 있었다. 이제 오류에 PowerShell 의 말이 실리므로 남길 곳만 생기면 된다.
- **`-race` 테스트**: 이 PC 에서 cgo 가 꺼져 있어 돌리지 못한다.
