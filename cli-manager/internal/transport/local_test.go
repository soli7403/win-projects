package transport

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestLocalEcho(t *testing.T) {
	szShell := "/bin/sh"
	if runtime.GOOS == "windows" {
		szShell = "cmd.exe"
	}
	var mu sync.Mutex
	var buf bytes.Buffer
	closed := make(chan string, 1)
	pstrL, err := StartLocal(LocalOptions{Shell: szShell}, "xterm-256color", 100, 30, Callbacks{
		OnData:  func(a []byte) { mu.Lock(); buf.Write(a); mu.Unlock() },
		OnClose: func(s string) { closed <- s },
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = pstrL.Write([]byte("echo termgo-ok\r"))
	bOk := false
	for i := 0; i < 50 && !bOk; i++ {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		bOk = bytes.Contains(buf.Bytes(), []byte("termgo-ok"))
		mu.Unlock()
	}
	if !bOk {
		t.Fatalf("output = %q", buf.String())
	}
	pstrL.Close()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("close callback not fired")
	}
}
