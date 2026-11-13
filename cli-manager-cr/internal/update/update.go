// Package update 는 배포 저장소의 최신 릴리스를 조회해 새 버전을 찾는다 (spec/behavior/lifecycle.md §6).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 상태 값 (spec/ipc.md update.check).
const (
	G_cszStatusNewer        = "newer"
	G_cszStatusLatest       = "latest"
	G_cszStatusError        = "error"
	G_cszStatusUnconfigured = "unconfigured"
)

// G_cdurRequestTimeout 은 요청 타임아웃, G_cdurTotalTimeout 은 전체 제한이다 (lifecycle.md §6).
const (
	G_cdurRequestTimeout = 10 * time.Second
	G_cdurTotalTimeout   = 15 * time.Second
)

// Result 는 확인 결과다.
type Result struct {
	M_szStatus  string `json:"status"`
	M_szVersion string `json:"version,omitempty"`
	M_szUrl     string `json:"url,omitempty"`
}

// ParseVersion 은 태그 문자열을 숫자 세 개(major, minor, patch)로 바꾼다.
// 앞의 v 와 - 또는 + 뒤의 접미는 뗀다. 부족한 자리와 숫자가 아닌 자리는 0 이다.
func ParseVersion(szTag string) [3]int {
	szText := strings.TrimPrefix(strings.TrimSpace(szTag), "v")
	if iCut := strings.IndexAny(szText, "-+"); iCut >= 0 {
		szText = szText[:iCut]
	}
	var arrOut [3]int
	slParts := strings.Split(szText, ".")
	for iIndex := 0; iIndex < 3 && iIndex < len(slParts); iIndex++ {
		if iValue, errAtoi := strconv.Atoi(slParts[iIndex]); errAtoi == nil && iValue >= 0 {
			arrOut[iIndex] = iValue
		}
	}
	return arrOut
}

// IsNewer 는 szLatest 가 szCurrent 보다 큰지 본다.
func IsNewer(szLatest string, szCurrent string) bool {
	arrLatest := ParseVersion(szLatest)
	arrCurrent := ParseVersion(szCurrent)
	for iIndex := 0; iIndex < 3; iIndex++ {
		if arrLatest[iIndex] != arrCurrent[iIndex] {
			return arrLatest[iIndex] > arrCurrent[iIndex]
		}
	}
	return false
}

// Check 는 GET {API}/repos/{저장소}/releases/latest 로 최신 릴리스를 조회한다.
// 저장소가 비면 unconfigured. 네트워크 실패·200 이외·해석 실패는 error (호출자가 조용히 무시한다).
func Check(szApi string, szRepo string, szCurrent string, pClient *http.Client) Result {
	if szRepo == "" {
		return Result{M_szStatus: G_cszStatusUnconfigured}
	}
	if pClient == nil {
		pClient = &http.Client{Timeout: G_cdurRequestTimeout}
	}
	ctxCall, fnCancel := context.WithTimeout(context.Background(), G_cdurTotalTimeout)
	defer fnCancel()
	szUrl := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(szApi, "/"), szRepo)
	pReq, errReq := http.NewRequestWithContext(ctxCall, http.MethodGet, szUrl, nil)
	if errReq != nil {
		return Result{M_szStatus: G_cszStatusError}
	}
	pReq.Header.Set("User-Agent", "DariTerm/"+szCurrent)
	pReq.Header.Set("Accept", "application/vnd.github+json")
	pResp, errDo := pClient.Do(pReq)
	if errDo != nil {
		return Result{M_szStatus: G_cszStatusError}
	}
	defer pResp.Body.Close()
	if pResp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, pResp.Body)
		return Result{M_szStatus: G_cszStatusError}
	}
	var release struct {
		TagName string `json:"tag_name"`
		HtmlUrl string `json:"html_url"`
	}
	if errJson := json.NewDecoder(pResp.Body).Decode(&release); errJson != nil || release.TagName == "" {
		return Result{M_szStatus: G_cszStatusError}
	}
	szVersion := strings.TrimPrefix(release.TagName, "v")
	if IsNewer(release.TagName, szCurrent) {
		return Result{M_szStatus: G_cszStatusNewer, M_szVersion: szVersion, M_szUrl: release.HtmlUrl}
	}
	return Result{M_szStatus: G_cszStatusLatest}
}
