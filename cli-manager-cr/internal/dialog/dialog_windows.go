//go:build windows

// Package dialog 는 운영체제 파일·폴더 선택 대화상자를 연다 (spec/ipc.md §2 dialog.*).
package dialog

import (
	"os/exec"
	"strings"
	"syscall"
)

// runSTA 는 STA 로 PowerShell 스크립트를 실행하고 표준 출력 줄을 돌려준다. 콘솔 창은 띄우지 않는다.
func runSTA(szScript string) ([]string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, errRun := cmd.Output()
	if errRun != nil {
		return nil, errRun
	}
	var slOut []string
	for _, szLine := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if szLine != "" {
			slOut = append(slOut, szLine)
		}
	}
	return slOut, nil
}

// psQuote 는 PowerShell 작은따옴표 문자열을 만든다.
func psQuote(szValue string) string {
	return "'" + strings.ReplaceAll(szValue, "'", "''") + "'"
}

// OpenFiles 는 여러 파일을 고르는 대화상자다. 취소하면 빈 목록.
func OpenFiles(szTitle string) ([]string, error) {
	return runSTA(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = ` + psQuote(szTitle) + `; $d.Multiselect = $true; if ($d.ShowDialog() -eq 'OK') { $d.FileNames | ForEach-Object { Write-Output $_ } }`)
}

// ChooseDir 는 폴더 선택 대화상자다. 취소하면 빈 문자열과 nil.
func ChooseDir(szTitle string, szInitial string, bAllowNew bool) (string, error) {
	slOut, errRun := runSTA(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.FolderBrowserDialog; $d.Description = ` + psQuote(szTitle) + `; $d.ShowNewFolderButton = ` + boolPs(bAllowNew) + `; $d.SelectedPath = ` + psQuote(szInitial) + `; if ($d.ShowDialog() -eq 'OK') { Write-Output $d.SelectedPath }`)
	if errRun != nil || len(slOut) == 0 {
		return "", errRun
	}
	return slOut[0], nil
}

// OpenKeyFile 은 개인 키 파일을 고르는 대화상자다 (숨김 파일 표시, 시작 폴더 ~/.ssh). 취소하면 빈 문자열.
func OpenKeyFile(szTitle string, szInitial string) (string, error) {
	slOut, errRun := runSTA(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = ` + psQuote(szTitle) + `; $d.InitialDirectory = ` + psQuote(szInitial) + `; $d.Filter = 'All files (*.*)|*.*'; if ($d.ShowDialog() -eq 'OK') { Write-Output $d.FileName }`)
	if errRun != nil || len(slOut) == 0 {
		return "", errRun
	}
	return slOut[0], nil
}

// OpenFileOf 는 정한 종류의 파일 하나를 고르는 대화상자다. szFilter 는 "설명|*.확장자" 꼴(여러 개는 | 로 잇는다). 취소하면 빈 문자열.
func OpenFileOf(szTitle string, szInitial string, szFilter string) (string, error) {
	slOut, errRun := runSTA(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = ` + psQuote(szTitle) + `; $d.InitialDirectory = ` + psQuote(szInitial) + `; $d.Filter = ` + psQuote(szFilter) + `; if ($d.ShowDialog() -eq 'OK') { Write-Output $d.FileName }`)
	if errRun != nil || len(slOut) == 0 {
		return "", errRun
	}
	return slOut[0], nil
}

func boolPs(bValue bool) string {
	if bValue {
		return "$true"
	}
	return "$false"
}
