//go:build windows

package transport

import "golang.org/x/sys/windows"

// kernel32 의 ConPTY·프로세스 속성 함수를 직접 연다 (x/sys/windows 는 공개 래퍼가 없다).
var (
	g_modKernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	g_procCreatePseudoConsole           = g_modKernel32.NewProc("CreatePseudoConsole")
	g_procResizePseudoConsole           = g_modKernel32.NewProc("ResizePseudoConsole")
	g_procClosePseudoConsole            = g_modKernel32.NewProc("ClosePseudoConsole")
	g_procInitializeProcThreadAttrList  = g_modKernel32.NewProc("InitializeProcThreadAttributeList")
	g_procUpdateProcThreadAttribute     = g_modKernel32.NewProc("UpdateProcThreadAttribute")
	g_procDeleteProcThreadAttributeList = g_modKernel32.NewProc("DeleteProcThreadAttributeList")
	g_procCreateProcessW                = g_modKernel32.NewProc("CreateProcessW")
)

// 상수 (WinBase.h / processthreadsapi.h).
const (
	g_dwProcThreadAttrPseudoConsole = 0x00020016
	g_dwExtendedStartupInfoPresent  = 0x00080000
	g_dwCreateUnicodeEnvironment    = 0x00000400
)
