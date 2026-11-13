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
