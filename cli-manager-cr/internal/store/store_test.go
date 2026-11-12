package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 설정 기본값 병합, 알 수 없는 키 보존, 지연 저장 후 재열기. data/store.md §1·§4.
func TestPatchPersistsAndPreservesUnknownKeys(t *testing.T) {
	szPath := filepath.Join(t.TempDir(), "store.json")
	if errWrite := os.WriteFile(szPath, []byte(`{"version":1,"futureKey":{"a":1},"settings":{"fontSize":16}}`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	pStore, errOpen := Open(szPath)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	mapSnap, errSnap := pStore.Snapshot()
	if errSnap != nil {
		t.Fatal(errSnap)
	}
	// 응답은 JSON 으로 나간다고 보고 한 번 직렬화·역직렬화해서 본다.
	bSnap, errMarshal := json.Marshal(mapSnap)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	var mapJson map[string]any
	if errUnmarshal := json.Unmarshal(bSnap, &mapJson); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	mapSettings := mapJson["settings"].(map[string]any)
	if mapSettings["fontSize"].(float64) != 16 {
		t.Fatalf("저장된 설정이 기본값보다 우선해야 함: %v", mapSettings["fontSize"])
	}
	if mapSettings["theme"] != "dark" {
		t.Fatalf("없는 설정 키는 기본값으로 채워야 함: %v", mapSettings["theme"])
	}
	if mapSettings["autoLockMinutes"].(float64) != 15 {
		t.Fatalf("autoLockMinutes 기본값 15 기대")
	}

	if errPatch := pStore.Patch(map[string]any{"hosts": []any{map[string]any{"alias": "web"}}}); errPatch != nil {
		t.Fatal(errPatch)
	}
	if errFlush := pStore.Flush(); errFlush != nil {
		t.Fatal(errFlush)
	}

	bRaw, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	var mapFile map[string]json.RawMessage
	if errUnmarshal := json.Unmarshal(bRaw, &mapFile); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if _, bOk := mapFile["futureKey"]; !bOk {
		t.Fatal("알 수 없는 최상위 키가 보존되어야 함")
	}
	if _, bOk := mapFile["hosts"]; !bOk {
		t.Fatal("patch 한 hosts 가 저장되어야 함")
	}
}

// groups/hosts/snippets 변경 시 동기화 예약 콜백. sync-backup.md §2.
func TestSyncKeyPatchFiresCallback(t *testing.T) {
	pStore, errOpen := Open(filepath.Join(t.TempDir(), "store.json"))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	chCalled := make(chan struct{}, 1)
	pStore.SetOnSyncKeys(func() { chCalled <- struct{}{} })
	if errPatch := pStore.Patch(map[string]any{"snippets": []any{}}); errPatch != nil {
		t.Fatal(errPatch)
	}
	select {
	case <-chCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("동기화 콜백이 호출되지 않음")
	}
	pStore.Flush()
}

// 깨진 본 파일은 .bak 으로 복구된다. sync-backup.md §1.
func TestBrokenMainFileFallsBackToBak(t *testing.T) {
	szPath := filepath.Join(t.TempDir(), "store.json")
	if errWrite := os.WriteFile(szPath+".bak", []byte(`{"version":1,"hosts":[{"alias":"bak"}]}`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errWrite := os.WriteFile(szPath, []byte(`{broken`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	pStore, errOpen := Open(szPath)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	rawHosts, bOk := pStore.Get("hosts")
	if !bOk || string(rawHosts) != `[{"alias":"bak"}]` {
		t.Fatalf(".bak 에서 복구되어야 함: %s", rawHosts)
	}
}
