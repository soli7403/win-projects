//go:build windows

package storeupdate

import (
	"strings"
	"testing"
	"time"
)

// 패키지 밖(개발 PC)에서도 runInPackage 는 정책 속성을 무시하고 프로세스를 만든다 (D134).

// 자식의 표준 출력을 받고, 따옴표·줄바꿈이 든 인자가 그대로 전해진다.
func TestRunInPackageOutput(t *testing.T) {
	bOut, errRun := runInPackage("powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command",
		"Write-Output 'a \"b\" c'; Write-Output \"line1`nline2\""}, 30*time.Second)
	if errRun != nil {
		t.Fatalf("실행: %v", errRun)
	}
	szOut := strings.ReplaceAll(string(bOut), "\r\n", "\n")
	if !strings.Contains(szOut, `a "b" c`) || !strings.Contains(szOut, "line1\nline2") {
		t.Fatalf("출력: %q", szOut)
	}
}

// 종료 코드가 0 이 아니면 오류이고, 표준 오류 출력이 오류 문구에 실린다.
func TestRunInPackageFailure(t *testing.T) {
	_, errRun := runInPackage("cmd.exe", []string{"/c", "echo boom-message 1>&2 & exit 3"}, 30*time.Second)
	if errRun == nil {
		t.Fatal("오류가 나야 한다")
	}
	if !strings.Contains(errRun.Error(), "exit status 3") || !strings.Contains(errRun.Error(), "boom-message") {
		t.Fatalf("오류 문구: %v", errRun)
	}
}

// 시한(2초)을 넘기면 프로세스를 끝내고 곧바로 오류를 돌려준다.
func TestRunInPackageTimeout(t *testing.T) {
	tmStart := time.Now()
	_, errRun := runInPackage("powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 30"}, 2*time.Second)
	if errRun == nil || !strings.Contains(errRun.Error(), "timed out") {
		t.Fatalf("시한 오류가 나야 한다: %v", errRun)
	}
	if durElapsed := time.Since(tmStart); durElapsed > 10*time.Second {
		t.Fatalf("너무 오래 걸림: %s", durElapsed)
	}
}

// 없는 프로그램은 오류다.
func TestRunInPackageMissingProgram(t *testing.T) {
	if _, errRun := runInPackage("no-such-program-dariterm-test.exe", nil, 5*time.Second); errRun == nil {
		t.Fatal("오류가 나야 한다")
	}
}
