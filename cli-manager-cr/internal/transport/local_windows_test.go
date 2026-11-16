//go:build windows

package transport

import (
	"os"
	"strings"
	"testing"
	"time"
)

// ConPTY 위에서 cmd.exe 를 띄워 출력이 돌아오는지 확인한다 (실제 가상 터미널 시험).
func TestLocalConPtyEchoAndExit(t *testing.T) {
	szCmd := os.Getenv("ComSpec")
	if szCmd == "" {
		szCmd = `C:\Windows\System32\cmd.exe`
	}
	pConn, errOpen := OpenLocal(LocalOptions{
		Shell: szCmd,
		Args:  []string{"/c", "echo dariterm-ok"},
		Cols:  120,
		Rows:  32,
		Env:   append(os.Environ(), "TERM=xterm-256color"),
	})
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer pConn.Close()

	chOut := make(chan string, 1)
	go func() {
		var szAll strings.Builder
		bBuf := make([]byte, 4096)
		for {
			n, errRead := pConn.Read(bBuf)
			if n > 0 {
				szAll.Write(bBuf[:n])
				if strings.Contains(szAll.String(), "dariterm-ok") {
					chOut <- szAll.String()
					return
				}
			}
			if errRead != nil {
				chOut <- szAll.String() + " [" + errRead.Error() + "]"
				return
			}
		}
	}()
	select {
	case szGot := <-chOut:
		if !strings.Contains(szGot, "dariterm-ok") {
			t.Fatalf("셸 출력이 돌아오지 않음: %q", szGot)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("셸 출력 대기 시간 초과")
	}
	if errResize := pConn.Resize(100, 30); errResize != nil {
		t.Fatal(errResize)
	}
}

func TestListShellsHasDefault(t *testing.T) {
	if len(ListShells()) == 0 {
		t.Fatal("Windows 에서는 최소한 셸 하나가 있어야 함")
	}
}

// Read 가 막혀 있는 동안 Close 해도 Read 가 오류로 돌아오고 닫힌 프로세스 핸들을 쓰지 않는다 (통합 계획 §4).
func TestLocalCloseWhileReading(t *testing.T) {
	szCmd := os.Getenv("ComSpec")
	if szCmd == "" {
		szCmd = `C:\Windows\System32\cmd.exe`
	}
	pLocal, errOpen := OpenLocal(LocalOptions{Shell: szCmd, Cols: 80, Rows: 24, Env: os.Environ()})
	if errOpen != nil {
		t.Skip(errOpen)
	}
	chErr := make(chan error, 1)
	go func() {
		bBuf := make([]byte, 4096)
		for {
			if _, errRead := pLocal.Read(bBuf); errRead != nil {
				chErr <- errRead
				return
			}
		}
	}()
	time.Sleep(500 * time.Millisecond) // 셸이 프롬프트를 내고 Read 가 막힐 때까지
	_ = pLocal.Close()
	select {
	case <-chErr:
	case <-time.After(5 * time.Second):
		t.Fatal("Close 뒤 Read 가 돌아오지 않음")
	}
	if errWrite := pLocal.Write([]byte("x")); errWrite == nil {
		t.Fatal("닫은 뒤 쓰기는 오류여야 함")
	}
	_ = pLocal.Close() // 두 번 닫아도 안전
}
