package app

import (
	"os"
	"path/filepath"
)

// 데이터 폴더 이름 (spec/data/sync-backup.md §1).
const (
	G_cszDataEnv     = "DARITERM_DATA"
	G_cszDataProduct = "DariTerm"
)

// DataDir 은 데이터 폴더를 정한다: 환경 변수 DARITERM_DATA 가 있으면 그 경로, 없으면 <사용자 설정 폴더>\DariTerm\data.
// 폴더는 없으면 만들고 권한은 소유자 전용이다.
func DataDir() (string, error) {
	if szDir := os.Getenv(G_cszDataEnv); szDir != "" {
		return szDir, os.MkdirAll(szDir, 0o700)
	}
	szConfig, errConfig := os.UserConfigDir()
	if errConfig != nil {
		return "", errConfig
	}
	szDir := filepath.Join(szConfig, G_cszDataProduct, "data")
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return "", errMkdir
	}
	return szDir, nil
}
