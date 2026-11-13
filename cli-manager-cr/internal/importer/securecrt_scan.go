package importer

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReadCrtSessions 는 세션 폴더 아래 .ini 파일을 모두 읽는다. 폴더가 없으면 빈 목록이다 (import.md §1.5 폴더 구조).
// 읽지 못한 파일·폴더는 건너뛴다. 순서는 폴더 경로·이름의 사전순이다.
func ReadCrtSessions(szSessionsDir string) []CrtSession {
	slOut := []CrtSession{}
	if pInfo, errStat := os.Stat(szSessionsDir); errStat != nil || !pInfo.IsDir() {
		return slOut
	}
	_ = filepath.WalkDir(szSessionsDir, func(szPath string, pEntry fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			if pEntry != nil && pEntry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if pEntry.IsDir() || !strings.EqualFold(filepath.Ext(szPath), ".ini") {
			return nil
		}
		szRel, errRel := filepath.Rel(szSessionsDir, szPath)
		if errRel != nil {
			return nil
		}
		slParts := strings.Split(filepath.ToSlash(szRel), "/")
		szFile := slParts[len(slParts)-1]
		szLower := strings.ToLower(szFile)
		if szLower == g_cszCrtFolderData || (len(slParts) == 1 && szLower == g_cszCrtDefault) {
			return nil
		}
		bRaw, errRead := os.ReadFile(szPath)
		if errRead != nil {
			return nil
		}
		slOut = append(slOut, CrtSession{
			M_szName:   szFile[:len(szFile)-len(filepath.Ext(szFile))],
			M_slFolder: slParts[:len(slParts)-1],
			M_values:   ParseCrtIni(bRaw),
		})
		return nil
	})
	sort.SliceStable(slOut, func(iLeft, iRight int) bool {
		szLeft := strings.Join(append(append([]string{}, slOut[iLeft].M_slFolder...), slOut[iLeft].M_szName), "\x00")
		szRight := strings.Join(append(append([]string{}, slOut[iRight].M_slFolder...), slOut[iRight].M_szName), "\x00")
		return szLeft < szRight
	})
	return slOut
}

// crtGlobalKey 는 설정 폴더 SSH2.ini 의 전역 키 파일 경로를 읽는다 (없으면 빈 문자열).
func crtGlobalKey(szConfigDir string) string {
	bRaw, errRead := os.ReadFile(filepath.Join(szConfigDir, "SSH2.ini"))
	if errRead != nil {
		return ""
	}
	return ParseCrtIni(bRaw).str("Identity Filename V2")
}

// ScanSecureCrt 는 폴더(설정 폴더 또는 세션 폴더)의 SecureCRT 세션을 후보로 바꾼다. 비우면 기본 설정 폴더를 쓴다.
func ScanSecureCrt(szDir string) []Candidate {
	if szDir == "" {
		szDir = SecureCrtConfigDir()
		if szDir == "" {
			return []Candidate{}
		}
	}
	szSessionsDir, szConfigDir := ResolveCrtDirs(szDir)
	szGlobalKey := crtGlobalKey(szConfigDir)
	slCands := []Candidate{}
	slKept := []CrtSession{}
	for _, session := range ReadCrtSessions(szSessionsDir) {
		if cand, bOk := ParseCrtSession(session, szGlobalKey, szConfigDir); bOk {
			slCands = append(slCands, cand)
			slKept = append(slKept, session)
		}
	}
	ResolveCrtJumps(slCands, slKept)
	return slCands
}
