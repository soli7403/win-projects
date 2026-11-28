//go:build windows

package sshconn

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// g_ciCreateNoWindow 는 CREATE_NO_WINDOW 플래그다 (명령 창을 띄우지 않는다).
const g_ciCreateNoWindow = 0x08000000

// shellCommand 는 명령 줄을 cmd.exe 로 실행하는 명령을 만든다 (명령 프롬프트에 친 것과 같은 해석, D152).
// 인자 따옴표를 Go 가 다시 감싸지 않도록 명령 줄을 그대로 넘긴다.
// /d 는 AutoRun 레지스트리를 건너뛰고, /s 는 바깥 따옴표 한 쌍만 벗겨 안의 따옴표("C:\Program Files\…")를 그대로 둔다.
func shellCommand(szLine string) *exec.Cmd {
	szComSpec := os.Getenv("ComSpec")
	if szComSpec == "" {
		szComSpec = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	pCmd := exec.Command(szComSpec)
	pCmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       `"` + szComSpec + `" /d /s /c "` + szLine + `"`,
		CreationFlags: g_ciCreateNoWindow,
	}
	return pCmd
}

// treeKiller 는 띄운 명령을 Job 객체에 넣고, 그 아래 프로세스(cmd.exe 가 띄운 ssh.exe 등)까지 한 번에 끝내는 함수를 돌려준다.
// cmd.exe 만 끝내면 그 아래 프로그램이 남아 파이프가 닫히지 않고 프로세스가 샌다.
// Job 을 만들지 못하면 명령 하나만 끝낸다.
func treeKiller(pCmd *exec.Cmd) func() {
	fnKillOne := func() {
		if pCmd.Process != nil {
			_ = pCmd.Process.Kill()
		}
	}
	hJob, errJob := windows.CreateJobObject(nil, nil)
	if errJob != nil {
		return fnKillOne
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, errSet := windows.SetInformationJobObject(hJob, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); errSet != nil {
		_ = windows.CloseHandle(hJob)
		return fnKillOne
	}
	hProcess, errOpen := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pCmd.Process.Pid))
	if errOpen != nil {
		_ = windows.CloseHandle(hJob)
		return fnKillOne
	}
	errAssign := windows.AssignProcessToJobObject(hJob, hProcess)
	_ = windows.CloseHandle(hProcess)
	if errAssign != nil {
		_ = windows.CloseHandle(hJob)
		return fnKillOne
	}
	return func() {
		_ = windows.TerminateJobObject(hJob, 1)
		_ = windows.CloseHandle(hJob)
		fnKillOne()
	}
}
