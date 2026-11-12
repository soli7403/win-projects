package app

import (
	"os"
	"path/filepath"
	"testing"
)

// 옛 TermGo 데이터 폴더가 있고 새 폴더가 없으면 통째로 복사하고, 옛 폴더는 남긴다.
func TestMigrateDataDirCopies(t *testing.T) {
	szRoot := t.TempDir()
	szOld := filepath.Join(szRoot, "TermGo", "data")
	szNew := filepath.Join(szRoot, "DariTerm", "data")
	mustWriteData(t, filepath.Join(szOld, "store.json"), `{"hosts":[1]}`)
	mustWriteData(t, filepath.Join(szOld, "logs", "a.log"), "log")

	if errMigrate := migrateDataDir(szOld, szNew); errMigrate != nil {
		t.Fatal(errMigrate)
	}
	for szRel, szWant := range map[string]string{"store.json": `{"hosts":[1]}`, filepath.Join("logs", "a.log"): "log"} {
		bGot, errRead := os.ReadFile(filepath.Join(szNew, szRel))
		if errRead != nil || string(bGot) != szWant {
			t.Fatalf("%s = %q, %v", szRel, bGot, errRead)
		}
	}
	if _, errOld := os.Stat(filepath.Join(szOld, "store.json")); errOld != nil {
		t.Fatal("옛 폴더는 남아 있어야 함")
	}
	// 임시 폴더가 남지 않는다.
	slEntries, _ := os.ReadDir(filepath.Join(szRoot, "DariTerm"))
	if len(slEntries) != 1 {
		t.Fatalf("DariTerm 아래 항목 = %v", slEntries)
	}
}

// 새 폴더가 이미 있으면 덮어쓰지 않는다. 옛 폴더가 없으면 아무 일도 없다.
func TestMigrateDataDirSkips(t *testing.T) {
	szRoot := t.TempDir()
	szOld := filepath.Join(szRoot, "TermGo", "data")
	szNew := filepath.Join(szRoot, "DariTerm", "data")
	if errNone := migrateDataDir(szOld, szNew); errNone != nil {
		t.Fatal(errNone)
	}
	if _, errStat := os.Stat(szNew); errStat == nil {
		t.Fatal("옛 폴더가 없으면 새 폴더를 만들지 않음(DataDir 이 만든다)")
	}
	mustWriteData(t, filepath.Join(szOld, "store.json"), "old")
	mustWriteData(t, filepath.Join(szNew, "store.json"), "new")
	if errMigrate := migrateDataDir(szOld, szNew); errMigrate != nil {
		t.Fatal(errMigrate)
	}
	if bGot, _ := os.ReadFile(filepath.Join(szNew, "store.json")); string(bGot) != "new" {
		t.Fatalf("새 폴더가 덮어써짐: %q", bGot)
	}
}

func mustWriteData(t *testing.T, szPath string, szBody string) {
	t.Helper()
	if errMk := os.MkdirAll(filepath.Dir(szPath), 0o700); errMk != nil {
		t.Fatal(errMk)
	}
	if errWrite := os.WriteFile(szPath, []byte(szBody), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
}
