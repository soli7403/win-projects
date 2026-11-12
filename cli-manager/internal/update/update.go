// Package update 는 배포 저장소의 최신 릴리스를 확인한다 (3.6).
// 요청에는 사용자 정보(서버 목록·IP·계정 등)를 전혀 담지 않는다 — 앱 이름/버전이 든 User-Agent 만 보낸다.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Result 는 확인 결과.
type Result struct {
	Status  string `json:"status"` // newer | latest | error | unconfigured
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
}

// Checker 는 GitHub Releases API 로 최신 버전을 조회한다.
type Checker struct {
	Repo    string // "owner/name" — 비어 있으면 unconfigured
	Current string
	BaseURL string // 시험용으로 바꿀 수 있다 (기본 https://api.github.com)
	Client  *http.Client
}

var reNum = regexp.MustCompile(`\d+`)

// versionParts 는 "v1.2.3-beta" → [1 2 3].
func versionParts(sz string) []int {
	sz = strings.TrimPrefix(strings.TrimSpace(sz), "v")
	if i := strings.IndexAny(sz, "-+"); i >= 0 {
		sz = sz[:i]
	}
	var out []int
	for _, m := range reNum.FindAllString(sz, 3) {
		n, _ := strconv.Atoi(m)
		out = append(out, n)
	}
	for len(out) < 3 {
		out = append(out, 0)
	}
	return out
}

// Newer 는 a 가 b 보다 높은 버전인지.
func Newer(a, b string) bool {
	x, y := versionParts(a), versionParts(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// Check 는 최신 릴리스를 조회한다. 네트워크 실패는 error 상태로만 돌려준다 (호출자가 조용히 무시).
func (pstrC *Checker) Check(ctx context.Context) Result {
	if strings.TrimSpace(pstrC.Repo) == "" {
		return Result{Status: "unconfigured"}
	}
	base := pstrC.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	cl := pstrC.Client
	if cl == nil {
		cl = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/releases/latest", base, pstrC.Repo), nil)
	if err != nil {
		return Result{Status: "error"}
	}
	req.Header.Set("User-Agent", "TermGo/"+pstrC.Current)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := cl.Do(req)
	if err != nil {
		return Result{Status: "error"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{Status: "error"}
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if json.NewDecoder(resp.Body).Decode(&rel) != nil || rel.TagName == "" {
		return Result{Status: "error"}
	}
	if Newer(rel.TagName, pstrC.Current) {
		return Result{Status: "newer", Version: strings.TrimPrefix(rel.TagName, "v"), URL: rel.HTMLURL}
	}
	return Result{Status: "latest", Version: strings.TrimPrefix(rel.TagName, "v")}
}
