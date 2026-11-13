package update

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestParseAndCompare 은 버전 비교 규칙을 확인한다 (lifecycle.md §6).
func TestParseAndCompare(t *testing.T) {
	if ParseVersion("v1.2-beta.1") != [3]int{1, 2, 0} || ParseVersion("2") != [3]int{2, 0, 0} || ParseVersion("1.2.3+build") != [3]int{1, 2, 3} {
		t.Fatal("버전 해석 불일치")
	}
	cases := []struct {
		szLatest, szCurrent string
		bWant               bool
	}{
		{"v1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"0.9.9", "1.0.0", false},
		{"1.10.0", "1.9.9", true},
		{"v2", "1.9.9", true},
	}
	for _, c := range cases {
		if IsNewer(c.szLatest, c.szCurrent) != c.bWant {
			t.Fatalf("비교 불일치: %s vs %s", c.szLatest, c.szCurrent)
		}
	}
}

// TestCheckWithFakeServer 는 가짜 서버로 newer, latest, error, unconfigured 를 확인한다.
func TestCheckWithFakeServer(t *testing.T) {
	var szSeenUA, szSeenAccept, szSeenPath string
	pServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		szSeenUA = r.Header.Get("User-Agent")
		szSeenAccept = r.Header.Get("Accept")
		szSeenPath = r.URL.Path
		if r.URL.Path == "/repos/owner/name/releases/latest" {
			_, _ = w.Write([]byte(`{"tag_name":"v2.3.4","html_url":"https://example.com/rel"}`))
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer pServer.Close()

	res := Check(pServer.URL, "owner/name", "2.0.0", nil)
	if res.M_szStatus != G_cszStatusNewer || res.M_szVersion != "2.3.4" || res.M_szUrl != "https://example.com/rel" {
		t.Fatalf("newer 결과 불일치: %+v", res)
	}
	if szSeenUA != "DariTerm/2.0.0" || szSeenAccept != "application/vnd.github+json" || szSeenPath != "/repos/owner/name/releases/latest" {
		t.Fatalf("요청 헤더/경로 불일치: %s %s %s", szSeenUA, szSeenAccept, szSeenPath)
	}
	if res := Check(pServer.URL, "owner/name", "2.3.4", nil); res.M_szStatus != G_cszStatusLatest {
		t.Fatalf("latest 결과 불일치: %+v", res)
	}
	if res := Check(pServer.URL, "owner/missing", "2.0.0", nil); res.M_szStatus != G_cszStatusError {
		t.Fatalf("404 는 error 여야 함: %+v", res)
	}
	if res := Check(pServer.URL, "", "2.0.0", nil); res.M_szStatus != G_cszStatusUnconfigured {
		t.Fatalf("저장소 미지정은 unconfigured 여야 함: %+v", res)
	}
	if res := Check("http://127.0.0.1:1", "owner/name", "2.0.0", nil); res.M_szStatus != G_cszStatusError {
		t.Fatalf("연결 실패는 error 여야 함: %+v", res)
	}
}
