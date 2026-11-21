package app

import (
	"encoding/json"
	"testing"
	"time"
)

// 세션 색·환경을 폴더에서 물려받는다 (spec/behavior/session.md §1, D105).

func lookGroups() []groupRecord {
	return []groupRecord{
		{M_szId: "purple", M_szColor: "#8b5cf6"},
		{M_szId: "prodUnderPurple", M_szParentId: "purple", M_szEnv: "prod"},
		{M_szId: "emptyUnderPurple", M_szParentId: "purple"},
		{M_szId: "devFolder", M_szEnv: "dev"},
		{M_szId: "loopA", M_szParentId: "loopB"},
		{M_szId: "loopB", M_szParentId: "loopA"},
	}
}

func TestResolveLook(t *testing.T) {
	slGroups := lookGroups()
	slCases := []struct {
		szName  string
		record  hostRecord
		szColor string
		szEnv   string
	}{
		{"폴더 색 물려받기(보고된 경우)", hostRecord{M_szGroupId: "purple"}, "#8b5cf6", "none"},
		{"서버 환경 none 이어도 물려받기", hostRecord{M_szGroupId: "purple", M_szEnv: "none"}, "#8b5cf6", "none"},
		{"서버 색 우선", hostRecord{M_szGroupId: "purple", M_szColor: "#123456"}, "#123456", "none"},
		{"서버 환경색이 폴더 색보다 먼저", hostRecord{M_szGroupId: "purple", M_szEnv: "dev"}, "#22c55e", "dev"},
		{"가까운 폴더의 환경색이 먼 폴더 색보다 먼저", hostRecord{M_szGroupId: "prodUnderPurple"}, "#ef4444", "prod"},
		{"빈 폴더 건너뛰기", hostRecord{M_szGroupId: "emptyUnderPurple"}, "#8b5cf6", "none"},
		{"서버 환경이 폴더 환경보다 먼저", hostRecord{M_szGroupId: "prodUnderPurple", M_szEnv: "stage"}, "#f59e0b", "stage"},
		{"서버 색 + 폴더 환경", hostRecord{M_szGroupId: "devFolder", M_szColor: "#123456"}, "#123456", "dev"},
		{"폴더 없는 서버", hostRecord{}, "", "none"},
		{"없어진 폴더", hostRecord{M_szGroupId: "gone"}, "", "none"},
		{"모르는 환경 값", hostRecord{M_szGroupId: "purple", M_szEnv: "weird"}, "#8b5cf6", "none"},
		{"순환하는 폴더", hostRecord{M_szGroupId: "loopA"}, "", "none"},
	}
	for _, tc := range slCases {
		szColor, szEnv := resolveLook(tc.record, slGroups)
		if szColor != tc.szColor || szEnv != tc.szEnv {
			t.Errorf("%s: (%q, %q) 기대, (%q, %q) 받음", tc.szName, tc.szColor, tc.szEnv, szColor, szEnv)
		}
	}
}

// 저장 데이터 형식(parentId·groupId 가 null) 그대로 읽어도 같은 결과.
func TestLookRecordsFromStoreJson(t *testing.T) {
	var slGroups []groupRecord
	if errGroups := json.Unmarshal([]byte(`[
		{"id":"g1","parentId":null,"name":"보라","color":"#8b5cf6","sort":0},
		{"id":"g2","parentId":"g1","name":"운영","env":"prod","sort":1}
	]`), &slGroups); errGroups != nil {
		t.Fatal(errGroups)
	}
	var slHosts []hostRecord
	if errHosts := json.Unmarshal([]byte(`[
		{"id":"h1","groupId":"g1","alias":"a","host":"x"},
		{"id":"h2","groupId":"g2","alias":"b","host":"y"},
		{"id":"h3","groupId":null,"alias":"c","host":"z"}
	]`), &slHosts); errHosts != nil {
		t.Fatal(errHosts)
	}
	slWant := [][2]string{{"#8b5cf6", "none"}, {"#ef4444", "prod"}, {"", "none"}}
	for iIdx, record := range slHosts {
		szColor, szEnv := resolveLook(record, slGroups)
		if szColor != slWant[iIdx][0] || szEnv != slWant[iIdx][1] {
			t.Errorf("%s: %v 기대, (%q, %q) 받음", record.M_szId, slWant[iIdx], szColor, szEnv)
		}
	}
}

// store.patch 로 폴더·서버를 넣고 session.open 을 부르면 세션 정보의 color·env 가 폴더 것을 물려받는다.
func TestSessionOpenInheritsFolderLook(t *testing.T) {
	em := &testEmitter{chAsk: make(chan map[string]any, 1), chToast: make(chan map[string]any, 1)}
	pService, errNew := NewService(t.TempDir(), em)
	if errNew != nil {
		t.Fatal(errNew)
	}
	defer pService.Close()
	host := func(szId string, groupId any) map[string]any {
		// 알 수 없는 프로토콜이라 접속은 바로 실패한다. 색·환경은 열 때 정해진다.
		return map[string]any{"id": szId, "alias": szId, "protocol": "ftpx", "host": "x", "groupId": groupId, "tags": []string{}}
	}
	raw, _ := json.Marshal(map[string]any{
		"groups": []map[string]any{
			{"id": "g1", "parentId": nil, "name": "보라", "color": "#8b5cf6", "sort": 0},
			{"id": "g2", "parentId": nil, "name": "운영", "env": "prod", "sort": 1},
		},
		"hosts": []map[string]any{host("h1", "g1"), host("h2", "g2"), host("h3", nil)},
	})
	if _, errPatch := pService.Router().Invoke("store.patch", `[`+string(raw)+`]`); errPatch != nil {
		t.Fatal(errPatch)
	}
	slWant := map[string][2]string{"h1": {"#8b5cf6", "none"}, "h2": {"#ef4444", "prod"}, "h3": {"", "none"}}
	for szHost, want := range slWant {
		szInfo, errOpen := pService.Router().Invoke("session.open", `[{"hostId":"`+szHost+`"}]`)
		if errOpen != nil {
			t.Fatal(errOpen)
		}
		var info struct {
			M_szId    string `json:"id"`
			M_szColor string `json:"color"`
			M_szEnv   string `json:"env"`
		}
		if errInfo := json.Unmarshal([]byte(szInfo), &info); errInfo != nil {
			t.Fatal(errInfo)
		}
		if info.M_szColor != want[0] || info.M_szEnv != want[1] {
			t.Errorf("%s: %v 기대, (%q, %q) 받음", szHost, want, info.M_szColor, info.M_szEnv)
		}
		_, _ = pService.Router().Invoke("session.close", `["`+info.M_szId+`"]`)
	}
	time.Sleep(50 * time.Millisecond)
}
