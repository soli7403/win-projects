package store

import (
	"encoding/json"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	szDir := t.TempDir()
	pstrS, err := Open(szDir)
	if err != nil {
		t.Fatal(err)
	}
	pstrS.Patch(map[string]json.RawMessage{"hosts": json.RawMessage(`[{"id":"a","extra":1}]`)})
	if err := pstrS.Flush(); err != nil {
		t.Fatal(err)
	}
	pstrS2, err := Open(szDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(pstrS2.Raw("hosts")); got == "" || got == "[]" {
		t.Fatalf("hosts lost: %q", got)
	}
	if len(pstrS2.Get()["settings"]) < 50 {
		t.Fatal("settings default")
	}
}
