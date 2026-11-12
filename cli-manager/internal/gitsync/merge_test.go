package gitsync

import (
	"encoding/json"
	"reflect"
	"testing"
)

func col(kv ...string) Collection {
	c := Collection{}
	for i := 0; i+1 < len(kv); i += 2 {
		c[kv[i]] = json.RawMessage(kv[i+1])
	}
	return c
}

func TestMergeCollection(t *testing.T) {
	base := col("a", `{"v":1}`, "b", `{"v":1}`, "c", `{"v":1}`, "d", `{"v":1}`)
	local := col("a", `{"v":2}`, "b", `{"v":1}`, "d", `{"v":5}`, "n1", `{"v":"L"}`)                  // a 수정, c 삭제, d 수정, n1 신규
	remote := col("a", `{"v":1}`, "b", `{"v":9}`, "c", `{"v":1}`, "d", `{"v":6}`, "n2", `{"v":"R"}`) // b 수정, d 수정, n2 신규
	got := MergeCollection(base, local, remote)

	want := col("a", `{"v":2}`, "b", `{"v":9}`, "d", `{"v":5}`, "n1", `{"v":"L"}`, "n2", `{"v":"R"}`)
	if !reflect.DeepEqual(SortedIDs(got), SortedIDs(want)) {
		t.Fatalf("ids = %v, want %v", SortedIDs(got), SortedIDs(want))
	}
	for id, w := range want {
		if !same(got[id], w) {
			t.Errorf("%s = %s, want %s", id, got[id], w)
		}
	}
}

func TestMergeDeletionWins(t *testing.T) {
	base := col("x", `{"v":1}`)
	// 원격이 삭제, 로컬은 수정 → 삭제가 이긴다 (한쪽 삭제는 다른 PC 에도 삭제)
	got := MergeCollection(base, col("x", `{"v":2}`), col())
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	// 로컬이 삭제, 원격은 그대로 → 삭제
	if got = MergeCollection(base, col(), col("x", `{"v":1}`)); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestMergeFirstSyncAndKeyOrder(t *testing.T) {
	// base 가 없는(처음 연결) 경우: 양쪽 항목을 합치고, 같은 id 는 local 우선
	got := MergeCollection(col(), col("a", `{"v":"L"}`, "s", `{"v":1}`), col("a", `{"v":"R"}`, "r", `{"v":1}`))
	if string(got["a"]) != `{"v":"L"}` || len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	// JSON 키 순서·공백이 달라도 같은 내용으로 본다
	if !same(json.RawMessage(`{"a":1,"b":2}`), json.RawMessage(`{ "b":2, "a":1 }`)) {
		t.Fatal("canonical compare")
	}
}
