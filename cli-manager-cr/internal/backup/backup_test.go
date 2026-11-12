package backup

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDefaultName 은 기본 이름 형식을 확인한다 (acceptance Y1).
func TestDefaultName(t *testing.T) {
	if DefaultName(time.Date(2026, 11, 10, 9, 0, 0, 0, time.Local)) != "dariterm-backup-2026-11-10.tgbackup" {
		t.Fatal("기본 이름 형식 불일치")
	}
}

// TestDecodeAppRules 는 app 값 검사를 확인한다 (acceptance Y2, Y3).
func TestDecodeAppRules(t *testing.T) {
	bOurs, _ := Encode(NewFile(map[string]any{"hosts": []any{}}, json.RawMessage("null"), time.Now()))
	if _, errOurs := Decode(bOurs); errOurs != nil {
		t.Fatalf("termgo 백업을 거부함: %v", errOurs)
	}
	if _, errLegacy := Decode([]byte(`{"app":"brightterm","version":1,"store":{}}`)); errLegacy != nil {
		t.Fatalf("brightterm 백업을 거부함: %v", errLegacy)
	}
	if _, errOther := Decode([]byte(`{"app":"other"}`)); errOther != ErrNotBackup {
		t.Fatalf("다른 app 은 백업 파일이 아니어야 함: %v", errOther)
	}
	if _, errBroken := Decode([]byte(`{broken`)); errBroken != ErrUnreadable {
		t.Fatalf("깨진 JSON 은 읽을 수 없음이어야 함: %v", errBroken)
	}
}
