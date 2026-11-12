package app

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// 데이터 폴더 이름 (spec/data/sync-backup.md §1). 제품 이름이 TermGo → DariTerm 으로 바뀌었다 (2026-11-12).
const (
	G_cszDataEnv       = "DARITERM_DATA"
	G_cszLegacyDataEnv = "TERMGO_DATA"
	G_cszDataProduct   = "DariTerm"
	G_cszLegacyProduct = "TermGo"
)

// DataDir 은 데이터 폴더를 정한다.
//  1. 환경 변수 DARITERM_DATA, 없으면 옛 TERMGO_DATA 가 있으면 그 경로 (이전하지 않는다).
//  2. 없으면 <사용자 설정 폴더>\DariTerm\data. 이 폴더가 없고 옛 <사용자 설정 폴더>\TermGo\data 가 있으면
//     옛 폴더를 통째로 복사해 온다. 옛 폴더는 백업으로 그대로 남긴다.
//
// 폴더는 없으면 만들고 권한은 소유자 전용이다.
func DataDir() (string, error) {
	for _, szEnv := range []string{G_cszDataEnv, G_cszLegacyDataEnv} {
		if szDir := os.Getenv(szEnv); szDir != "" {
			return szDir, os.MkdirAll(szDir, 0o700)
		}
	}
	szConfig, errConfig := os.UserConfigDir()
	if errConfig != nil {
		return "", errConfig
	}
	szDir := filepath.Join(szConfig, G_cszDataProduct, "data")
	szLegacy := filepath.Join(szConfig, G_cszLegacyProduct, "data")
	if errMigrate := migrateDataDir(szLegacy, szDir); errMigrate != nil {
		return "", errMigrate
	}
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return "", errMkdir
	}
	return szDir, nil
}

// migrateDataDir 는 szNew 가 없고 szOld 가 폴더로 있으면 szOld 를 szNew 로 복사한다.
// 같은 부모의 임시 폴더에 다 복사한 뒤 이름을 바꿔, 실패해도 반쯤 복사된 szNew 가 남지 않게 한다.
// 이미 szNew 가 있거나 szOld 가 없으면 아무것도 하지 않는다.
func migrateDataDir(szOld string, szNew string) error {
	if _, errNew := os.Stat(szNew); errNew == nil {
		return nil
	} else if !errors.Is(errNew, fs.ErrNotExist) {
		return errNew
	}
	info, errOld := os.Stat(szOld)
	if errOld != nil || !info.IsDir() {
		return nil // 옛 데이터 없음: 새로 시작
	}
	szParent := filepath.Dir(szNew)
	if errMk := os.MkdirAll(szParent, 0o700); errMk != nil {
		return errMk
	}
	szTemp, errTemp := os.MkdirTemp(szParent, "data-migrate-")
	if errTemp != nil {
		return errTemp
	}
	if errCopy := copyTree(szOld, szTemp); errCopy != nil {
		_ = os.RemoveAll(szTemp)
		return fmt.Errorf("옛 데이터 폴더(%s)를 복사하지 못했습니다: %w", szOld, errCopy)
	}
	if errRename := os.Rename(szTemp, szNew); errRename != nil {
		_ = os.RemoveAll(szTemp)
		return fmt.Errorf("새 데이터 폴더(%s)를 만들지 못했습니다: %w", szNew, errRename)
	}
	return nil
}

// copyTree 는 szSrc 폴더 안의 내용을 szDst 폴더(이미 있음) 아래로 복사한다. 파일 권한을 유지한다.
func copyTree(szSrc string, szDst string) error {
	return filepath.WalkDir(szSrc, func(szPath string, entry fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		szRel, errRel := filepath.Rel(szSrc, szPath)
		if errRel != nil {
			return errRel
		}
		szTarget := filepath.Join(szDst, szRel)
		info, errInfo := entry.Info()
		if errInfo != nil {
			return errInfo
		}
		if entry.IsDir() {
			return os.MkdirAll(szTarget, info.Mode().Perm()|0o700)
		}
		if !info.Mode().IsRegular() {
			return nil // 링크·장치 파일은 데이터 폴더에 없어야 한다
		}
		return copyFile(szPath, szTarget, info.Mode().Perm())
	})
}

func copyFile(szSrc string, szDst string, mode fs.FileMode) error {
	src, errOpen := os.Open(szSrc)
	if errOpen != nil {
		return errOpen
	}
	defer src.Close()
	dst, errCreate := os.OpenFile(szDst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if errCreate != nil {
		return errCreate
	}
	_, errCopy := io.Copy(dst, src)
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy
}
