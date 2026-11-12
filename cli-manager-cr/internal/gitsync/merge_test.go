package gitsync

import (
	"encoding/json"
	"strings"
	"testing"

	"termgo/internal/vault"
)

func raw(szText string) json.RawMessage {
	return json.RawMessage(szText)
}

// TestMergeRules 는 3방향 병합 규칙을 확인한다 (sync-backup.md §5.3, acceptance 04 Y6, Y7).
func TestMergeRules(t *testing.T) {
	base := Collection{
		"both-changed": raw(`{"v":"base"}`),
		"local-del":    raw(`{"v":"base"}`),
		"remote-edit":  raw(`{"v":"base"}`),
		"remote-del":   raw(`{"v":"base"}`),
		"untouched":    raw(`{"v":"base"}`),
	}
	local := Collection{
		"both-changed": raw(`{"v":"local"}`),
		"remote-edit":  raw(`{"v":"base"}`),
		"remote-del":   raw(`{"v":"local-kept"}`),
		"untouched":    raw(`{"v":"base"}`),
		"new-local":    raw(`{"v":"nl"}`),
	}
	remote := Collection{
		"both-changed": raw(`{"v":"remote"}`),
		"local-del":    raw(`{"v":"remote-edit"}`),
		"remote-edit":  raw(`{"v":"remote-edit"}`),
		"untouched":    raw(`{"v":"base"}`),
		"new-remote":   raw(`{"v":"nr"}`),
		"new-both":     raw(`{"v":"remote-both"}`),
	}
	local["new-both"] = raw(`{"v":"local-both"}`)
	merged := MergeCollection(base, local, remote)

	if Canonical(merged["both-changed"]) != `{"v":"local"}` {
		t.Fatalf("양쪽 수정은 이 PC 값이어야 함: %s", merged["both-changed"])
	}
	if _, bOk := merged["local-del"]; bOk {
		t.Fatal("이 PC 가 지운 항목은 저장소가 고쳐도 삭제되어야 함")
	}
	if _, bOk := merged["remote-del"]; bOk {
		t.Fatal("저장소가 지운 항목은 이 PC 가 고쳐도 삭제되어야 함")
	}
	if Canonical(merged["remote-edit"]) != `{"v":"remote-edit"}` {
		t.Fatalf("저장소만 수정한 값은 저장소 값이어야 함: %s", merged["remote-edit"])
	}
	if Canonical(merged["new-local"]) != `{"v":"nl"}` || Canonical(merged["new-remote"]) != `{"v":"nr"}` {
		t.Fatal("기준에 없던 새 항목 병합 실패")
	}
	if Canonical(merged["new-both"]) != `{"v":"local-both"}` {
		t.Fatalf("양쪽에 같은 id 로 새로 생기면 이 PC 값: %s", merged["new-both"])
	}
}

// TestCanonicalKeyOrder 는 키 순서가 달라도 같은 항목으로 보는지 확인한다.
func TestCanonicalKeyOrder(t *testing.T) {
	if Canonical(raw(`{"a":1,"b":{"x":1,"y":2}}`)) != Canonical(raw(`{ "b": {"y":2,"x":1}, "a": 1 }`)) {
		t.Fatal("키 순서만 다른 항목이 다르게 판정됨")
	}
}

// TestOrderIds 는 이 PC 배열 순서를 유지하고 새 항목을 id 오름차순으로 뒤에 붙이는지 확인한다.
func TestOrderIds(t *testing.T) {
	merged := Collection{"b": raw(`1`), "a": raw(`1`), "c": raw(`1`), "z": raw(`1`)}
	slOrder := OrderIds(merged, []string{"c", "gone", "b"})
	if strings.Join(slOrder, ",") != "c,b,a,z" {
		t.Fatalf("순서 불일치: %v", slOrder)
	}
}

// TestSyncFileFormat 은 파일 형식 왕복과 형식 오류를 확인한다 (data/sync-backup.md §5.1).
func TestSyncFileFormat(t *testing.T) {
	blob := vault.Blob{M_szIv: "iv", M_szTag: "tag", M_szCt: "ct"}
	bFile, errEncode := EncodeSyncFile(raw(`{"format":"x"}`), blob)
	if errEncode != nil {
		t.Fatal(errEncode)
	}
	if !strings.Contains(string(bFile), `"format": "termgo-sync-1"`) {
		t.Fatalf("형식 이름 누락: %s", bFile)
	}
	if !strings.Contains(string(bFile), "\n \"format\"") {
		t.Fatal("들여쓰기 1칸이 아님")
	}
	_, blobBack, errDecode := DecodeSyncFile(bFile)
	if errDecode != nil || blobBack.M_szCt != "ct" {
		t.Fatalf("왕복 실패: %v", errDecode)
	}
	if _, _, errBad := DecodeSyncFile([]byte(`{"format":"other","vault":{},"payload":{}}`)); errBad != ErrFormat {
		t.Fatalf("다른 형식은 ErrFormat 이어야 함: %v", errBad)
	}
	if _, _, errBroken := DecodeSyncFile([]byte(`{broken`)); errBroken != ErrFormat {
		t.Fatalf("깨진 JSON 은 ErrFormat 이어야 함: %v", errBroken)
	}
}

// TestDataEqual 은 컬렉션 동일성 판정을 확인한다.
func TestDataEqual(t *testing.T) {
	a := NewData()
	b := NewData()
	a.M_mapHosts["h"] = raw(`{"x":1}`)
	b.M_mapHosts["h"] = raw(`{ "x": 1 }`)
	if !DataEqual(a, b) {
		t.Fatal("같은 내용인데 다르다고 판정")
	}
	b.M_mapHosts["h"] = raw(`{"x":2}`)
	if DataEqual(a, b) {
		t.Fatal("다른 내용인데 같다고 판정")
	}
}
