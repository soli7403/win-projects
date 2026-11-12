package update

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v1.2.0", "1.1.9", true}, {"1.0.0", "1.0.0", false}, {"v1.0.1", "v1.0.10", false},
		{"2.0.0-beta", "1.9.9", true}, {"0.3.0-dev", "0.3.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%s,%s) = %v", c.a, c.b, got)
		}
	}
}

func TestCheck(t *testing.T) {
	var gotUA, gotPath, gotQuery string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotPath, gotQuery = r.Header.Get("User-Agent"), r.URL.Path, r.URL.RawQuery
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"tag_name":"v9.0.0","html_url":"https://example.test/r/9"}`))
	}))
	defer srv.Close()

	c := &Checker{Repo: "me/termgo", Current: "0.3.0", BaseURL: srv.URL}
	r := c.Check(context.Background())
	if r.Status != "newer" || r.Version != "9.0.0" || r.URL == "" {
		t.Fatalf("result = %+v", r)
	}
	// 사용자 정보가 요청에 섞이지 않는다: 본문·쿼리 없음, 경로는 저장소만
	if len(gotBody) != 0 || gotQuery != "" || gotPath != "/repos/me/termgo/releases/latest" || gotUA != "TermGo/0.3.0" {
		t.Fatalf("request leaked data: ua=%q path=%q query=%q body=%q", gotUA, gotPath, gotQuery, gotBody)
	}
	if (&Checker{Current: "1"}).Check(context.Background()).Status != "unconfigured" {
		t.Fatal("empty repo must be unconfigured")
	}
	bad := &Checker{Repo: "x/y", Current: "1", BaseURL: "http://127.0.0.1:1"}
	if bad.Check(context.Background()).Status != "error" {
		t.Fatal("network failure must be error")
	}
}
