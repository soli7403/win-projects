package ipc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRouterCall(t *testing.T) {
	pstrR := NewRouter()
	pstrR.Handle("echo", func(slArgs []json.RawMessage) (any, error) { return slArgs, nil })
	pstrR.Handle("void", func([]json.RawMessage) (any, error) { return nil, nil })

	szOut, err := pstrR.Call("echo", `[1,"a"]`)
	if err != nil || szOut != `[1,"a"]` {
		t.Fatalf("echo = %q, %v", szOut, err)
	}
	if szOut, err = pstrR.Call("void", `[]`); err != nil || szOut != "" {
		t.Fatalf("void = %q, %v", szOut, err)
	}
	if _, err = pstrR.Call("nope", `[]`); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("unknown channel err = %v", err)
	}
	pstrR.Handle("boom", func([]json.RawMessage) (any, error) { panic("x") })
	if _, err = pstrR.Call("boom", `[]`); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("panic must become error: %v", err)
	}
	if _, err = pstrR.Call("echo", `{bad`); err == nil {
		t.Fatal("bad json must fail")
	}
}
