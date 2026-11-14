// Package jsonfile 은 데이터 폴더의 JSON 파일을 원자적으로 쓰고 .bak 으로 복구한다 (sync-backup.md §1).
package jsonfile

import (
	"dariterm/internal/i18n"
	"encoding/json"
	"os"
	"path/filepath"
)

// ErrNotFound 는 본 파일과 .bak 이 모두 없거나 읽을 수 없을 때 돌려준다.
var ErrNotFound = i18n.NewError("be.file.notFound")

// Write 는 값을 들여쓰기 2칸 JSON 으로 쓴다.
// 기존 파일이 있으면 .bak 으로 복사한 뒤 .tmp 에 쓰고 이름을 바꾼다. 권한은 소유자 전용(0600).
func Write(szPath string, value any) error {
	bData, errMarshal := json.MarshalIndent(value, "", "  ")
	if errMarshal != nil {
		return errMarshal
	}
	if errMkdir := os.MkdirAll(filepath.Dir(szPath), 0o700); errMkdir != nil {
		return errMkdir
	}
	if bOld, errRead := os.ReadFile(szPath); errRead == nil {
		if errBak := os.WriteFile(szPath+".bak", bOld, 0o600); errBak != nil {
			return errBak
		}
	}
	szTmp := szPath + ".tmp"
	if errWrite := os.WriteFile(szTmp, bData, 0o600); errWrite != nil {
		return errWrite
	}
	return os.Rename(szTmp, szPath)
}

// Read 는 본 파일을 읽어 out 에 해석한다. 실패하면 .bak 을 시도한다.
// 둘 다 없거나 깨졌으면 ErrNotFound 를 돌려준다.
func Read(szPath string, out any) error {
	if errMain := readOne(szPath, out); errMain == nil {
		return nil
	}
	if errBak := readOne(szPath+".bak", out); errBak == nil {
		return nil
	}
	return ErrNotFound
}

// Exists 는 본 파일 또는 .bak 이 있는지 본다.
func Exists(szPath string) bool {
	if _, errStat := os.Stat(szPath); errStat == nil {
		return true
	}
	_, errStat := os.Stat(szPath + ".bak")
	return errStat == nil
}

func readOne(szPath string, out any) error {
	bData, errRead := os.ReadFile(szPath)
	if errRead != nil {
		return errRead
	}
	return json.Unmarshal(bData, out)
}
