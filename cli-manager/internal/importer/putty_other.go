//go:build !windows

package importer

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanPutty 는 Unix PuTTY 의 ~/.putty/sessions (세션당 Key=Value 파일)를 읽는다.
func ScanPutty() []Candidate {
	szHome, err := os.UserHomeDir()
	if err != nil {
		return []Candidate{}
	}
	szDir := filepath.Join(szHome, ".putty", "sessions")
	ents, err := os.ReadDir(szDir)
	if err != nil {
		return []Candidate{}
	}
	m := map[string]RegValues{}
	for _, e := range ents {
		arr, err := os.ReadFile(filepath.Join(szDir, e.Name()))
		if err != nil {
			continue
		}
		vals := RegValues{}
		for _, l := range strings.Split(string(arr), "\n") {
			l = strings.TrimRight(l, "\r")
			if i := strings.Index(l, "="); i > 0 {
				vals[l[:i]] = l[i+1:]
			}
		}
		m[e.Name()] = vals
	}
	return PuttyToCandidates(m)
}
