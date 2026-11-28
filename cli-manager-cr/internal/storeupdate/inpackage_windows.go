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
