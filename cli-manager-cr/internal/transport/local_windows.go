//go:build windows

package transport

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// LocalConn 은 ConPTY 위에서 도는 셸 하나다.
type LocalConn struct {
	hPty     windows.Handle
	hProcess windows.Handle
	hInWrite windows.Handle
	hOutRead windows.Handle
	mtxWrite sync.Mutex
	mtxClose sync.Once
}

func coordValue(iCols int, iRows int) uintptr {
	return uintptr(uint32(uint16(iCols)) | uint32(uint16(iRows))<<16)
}

// OpenLocal 은 셸을 가상 터미널(ConPTY) 위에 띄운다.
func OpenLocal(opts LocalOptions) (*LocalConn, error) {
	if opts.Cols < 2 {
		opts.Cols = 120
	}
	if opts.Rows < 2 {
		opts.Rows = 32
	}
	var hInRead, hInWrite, hOutRead, hOutWrite windows.Handle
	if errPipe := windows.CreatePipe(&hInRead, &hInWrite, nil, 0); errPipe != nil {
		return nil, errPipe
	}
	if errPipe := windows.CreatePipe(&hOutRead, &hOutWrite, nil, 0); errPipe != nil {
		return nil, errPipe
	}
	var hPty windows.Handle
	rCreate, _, _ := g_procCreatePseudoConsole.Call(coordValue(opts.Cols, opts.Rows), uintptr(hInRead), uintptr(hOutWrite), 0, uintptr(unsafe.Pointer(&hPty)))
	if rCreate != 0 {
		return nil, fmt.Errorf("가상 터미널 생성 실패: 0x%x", rCreate)
	}
	// 가상 터미널이 입출력 쪽 핸들을 가져갔으므로 이쪽 복사본은 닫는다.
	_ = windows.CloseHandle(hInRead)
	_ = windows.CloseHandle(hOutWrite)

	var cbAttr uintptr
	_, _, _ = g_procInitializeProcThreadAttrList.Call(0, 1, 0, uintptr(unsafe.Pointer(&cbAttr)))
	// 속성 목록은 포인터 정렬이 필요하므로 uintptr 배열로 잡는다.
	slAttr := make([]uintptr, (cbAttr+unsafe.Sizeof(uintptr(0))-1)/unsafe.Sizeof(uintptr(0)))
	pAttr := unsafe.Pointer(&slAttr[0])
	rInit, _, errInit := g_procInitializeProcThreadAttrList.Call(uintptr(pAttr), 1, 0, uintptr(unsafe.Pointer(&cbAttr)))
	if rInit == 0 {
		return nil, fmt.Errorf("프로세스 속성 초기화 실패: %v", errInit)
	}
	// 가상 터미널 속성의 값은 HPCON 핸들 값 자체다 (포인터가 아님, ConPTY 예제와 같다).
	rUpdate, _, errUpdate := g_procUpdateProcThreadAttribute.Call(uintptr(pAttr), 0, g_dwProcThreadAttrPseudoConsole, uintptr(hPty), unsafe.Sizeof(hPty), 0, 0)
	if rUpdate == 0 {
		return nil, fmt.Errorf("가상 터미널 연결 실패: %v", errUpdate)
	}
	defer g_procDeleteProcThreadAttributeList.Call(uintptr(pAttr))

	var siEx windows.StartupInfoEx
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))
	// 표준 핸들을 부모에게서 물려받지 않도록 빈 핸들을 명시한다 (가상 터미널이 입출력을 맡는다).
	siEx.StartupInfo.Flags = 0x00000100 // STARTF_USESTDHANDLES
	siEx.ProcThreadAttributeList = (*windows.ProcThreadAttributeList)(pAttr)

	szCmdLine := quoteArg(opts.Shell)
	for _, szArg := range opts.Args {
		szCmdLine += " " + quoteArg(szArg)
	}
	pCmd, errCmd := windows.UTF16PtrFromString(szCmdLine)
	if errCmd != nil {
		return nil, errCmd
	}
	var pCwd *uint16
	if opts.Cwd != "" {
		pCwd, _ = windows.UTF16PtrFromString(opts.Cwd)
	}
	pEnv := envBlock(opts.Env)

	var pi windows.ProcessInformation
	rProc, _, errProc := g_procCreateProcessW.Call(
		0,
		uintptr(unsafe.Pointer(pCmd)),
		0, 0, 0,
		g_dwExtendedStartupInfoPresent|g_dwCreateUnicodeEnvironment,
		uintptr(unsafe.Pointer(&pEnv[0])),
		uintptr(unsafe.Pointer(pCwd)),
		uintptr(unsafe.Pointer(&siEx)),
		uintptr(unsafe.Pointer(&pi)),
	)
	if rProc == 0 {
		return nil, fmt.Errorf("셸 실행 실패: %v", errProc)
	}
	_ = windows.CloseHandle(pi.Thread)
	return &LocalConn{hPty: hPty, hProcess: pi.Process, hInWrite: hInWrite, hOutRead: hOutRead}, nil
}

// quoteArg 는 공백이 있는 인자를 큰따옴표로 감싼다.
func quoteArg(szArg string) string {
	if strings.ContainsAny(szArg, " \t") {
		return `"` + szArg + `"`
	}
	return szArg
}

// envBlock 은 "이름=값" 목록을 UTF-16 환경 블록(끝 NUL 두 개)으로 만든다.
func envBlock(slEnv []string) []uint16 {
	var slOut []uint16
	for _, szItem := range slEnv {
		slOut = append(slOut, windows.StringToUTF16(szItem)...)
	}
	return append(slOut, 0)
}

// Read 는 셸 출력을 읽는다. 셸이 끝나면 종료 이유(코드 포함)를 오류로 돌려준다.
func (pLocal *LocalConn) Read(bBuf []byte) (int, error) {
	var nRead uint32
	errRead := windows.ReadFile(pLocal.hOutRead, bBuf, &nRead, nil)
	if errRead != nil || nRead == 0 {
		return 0, pLocal.exitReason()
	}
	return int(nRead), nil
}

// exitReason 은 셸 종료 이유 문구를 만든다 (protocols.md §6).
func (pLocal *LocalConn) exitReason() error {
	_, _ = windows.WaitForSingleObject(pLocal.hProcess, windows.INFINITE)
	var dwCode uint32
	if errCode := windows.GetExitCodeProcess(pLocal.hProcess, &dwCode); errCode == nil && dwCode != 0 {
		return fmt.Errorf("셸이 종료되었습니다 (코드 %d)", dwCode)
	}
	return fmt.Errorf("셸이 종료되었습니다")
}

// Write 는 입력(UTF-8 바이트)을 셸로 보낸다.
func (pLocal *LocalConn) Write(bData []byte) error {
	pLocal.mtxWrite.Lock()
	defer pLocal.mtxWrite.Unlock()
	var nWritten uint32
	return windows.WriteFile(pLocal.hInWrite, bData, &nWritten, nil)
}

// Resize 는 가상 터미널 크기를 바꾼다.
func (pLocal *LocalConn) Resize(iCols int, iRows int) error {
	if iCols < 2 || iRows < 2 {
		return nil
	}
	rResize, _, errResize := g_procResizePseudoConsole.Call(uintptr(pLocal.hPty), coordValue(iCols, iRows))
	if rResize != 0 {
		return fmt.Errorf("크기 변경 실패: %v", errResize)
	}
	return nil
}

// Close 는 셸을 강제 종료하고 핸들을 닫는다 (사용자가 닫을 때).
func (pLocal *LocalConn) Close() error {
	pLocal.mtxClose.Do(func() {
		_ = windows.TerminateProcess(pLocal.hProcess, 1)
		go g_procClosePseudoConsole.Call(uintptr(pLocal.hPty))
		_ = windows.CloseHandle(pLocal.hInWrite)
		_ = windows.CloseHandle(pLocal.hOutRead)
		_ = windows.CloseHandle(pLocal.hProcess)
	})
	return nil
}
