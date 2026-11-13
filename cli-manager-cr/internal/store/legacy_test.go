package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMigrateNames 는 원본 앱(BrightTerm) 이름 값 변환 규칙을 확인한다 (data/store.md §4).
func TestMigrateNames(t *testing.T) {
	for szIn, szWant := range map[string]string{
		"~/.brightterm/uploads":         "~/.dariterm/uploads",
		`~\.brightterm\uploads`:         `~\.dariterm\uploads`,
		"/data/.brightterm":             "/data/.dariterm",
		"~/.dariterm/uploads":           "~/.dariterm/uploads",
		"~/my.brightterm-files/uploads": "~/my.brightterm-files/uploads",
		"/srv/brightterm/uploads":       "/srv/brightterm/uploads",
	} {
		if szGot := MigrateUploadDir(szIn); szGot != szWant {
			t.Fatalf("MigrateUploadDir(%q) = %q, 기대 %q", szIn, szGot, szWant)
		}
	}
	for szIn, szWant := range map[string]string{
		"BrightTerm Light": "DariTerm Light",
		"Nord":             "Nord",
		"DariTerm Dark":    "DariTerm Dark",
	} {
		if szGot := MigrateThemeName(szIn); szGot != szWant {
			t.Fatalf("MigrateThemeName(%q) = %q, 기대 %q", szIn, szGot, szWant)
		}
	}
}

// TestOpenMigratesOriginSettings 는 원본 앱 store.json 을 열 때 값을 바꾸고 파일에도 저장하며, 다른 설정은 그대로 두는지 확인한다.
func TestOpenMigratesOriginSettings(t *testing.T) {
	szPath := filepath.Join(t.TempDir(), "store.json")
	bOld := []byte(`{"version":1,"hosts":[],"settings":{"uploadDir":"~/.brightterm/uploads","terminalTheme":"BrightTerm Dark","fontSize":13}}`)
	if errWrite := os.WriteFile(szPath, bOld, 0o600); errWrite != nil {
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
	mapSettings := mapSnap["settings"].(map[string]any)
	if mapSettings["uploadDir"] != "~/.dariterm/uploads" || mapSettings["terminalTheme"] != "DariTerm Dark" || mapSettings["fontSize"] != float64(13) {
		t.Fatalf("변환 결과 불일치: %+v", mapSettings)
	}
	if errFlush := pStore.Flush(); errFlush != nil {
		t.Fatal(errFlush)
	}
	bSaved, _ := os.ReadFile(szPath)
	var mapFile struct {
		Settings map[string]any `json:"settings"`
	}
	if errJson := json.Unmarshal(bSaved, &mapFile); errJson != nil || mapFile.Settings["uploadDir"] != "~/.dariterm/uploads" {
		t.Fatalf("파일에 바뀐 값이 저장되지 않음: %s", bSaved)
	}
}
