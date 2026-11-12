package sftpfs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// G_szPromptPrefix 는 붙여넣기용 업로드 파일 이름 접두어다 (protocols.md §3, 이 제품의 접두어 tg-).
const G_szPromptPrefix = "tg-"

// g_reUnsafeName 은 원래 이름에서 공백, ', ", `, $, \ 를 _ 로 바꾸는 규칙이다.
var g_reUnsafeName = regexp.MustCompile("[\\s'\"`$\\\\]")

// PromptStem 은 "tg-YYYYMMDD-HHMMSS-<4자리 16진>" 접두부를 만든다.
func PromptStem(tNow time.Time, szRand4 string) string {
	return G_szPromptPrefix + tNow.Format("20060102-150405") + "-" + szRand4
}

// PromptImageName 은 클립보드 이미지·버퍼 데이터의 파일 이름이다. 확장자는 주어진 이름의 것(없으면 .png).
func PromptImageName(tNow time.Time, szRand4 string, szGivenName string) string {
	szExt := filepath.Ext(szGivenName)
	if szExt == "" {
		szExt = ".png"
	}
	return PromptStem(tNow, szRand4) + szExt
}

// PromptFileName 은 파일의 붙여넣기용 이름이다: tg-…-<4자리>-<원래 이름 정리>.
func PromptFileName(tNow time.Time, szRand4 string, szOriginal string) string {
	return PromptStem(tNow, szRand4) + "-" + g_reUnsafeName.ReplaceAllString(szOriginal, "_")
}

// RandHex4 는 4자리 16진 무작위 문자열이다 (테스트에서는 고정 값을 넣는다).
func RandHex4() string {
	return fmt.Sprintf("%04x", time.Now().UnixNano()&0xffff)
}

// CleanupOldPrompt 는 폴더의 tg- 접두 파일 중 수정 시각이 iDays 일보다 오래된 것을 지운다 (protocols.md §3).
// iDays 가 0 이하이면 아무 일도 하지 않는다. 지운 개수를 돌려준다.
func CleanupOldPrompt(szDir string, iDays int, tNow time.Time) (int, error) {
	if iDays <= 0 {
		return 0, nil
	}
	entries, errRead := os.ReadDir(szDir)
	if errRead != nil {
		return 0, errRead
	}
	nRemoved := 0
	limit := tNow.Add(-time.Duration(iDays) * 24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), G_szPromptPrefix) {
			continue
		}
		info, errInfo := entry.Info()
		if errInfo != nil {
			continue
		}
		if info.ModTime().Before(limit) {
			if errRemove := os.Remove(filepath.Join(szDir, entry.Name())); errRemove == nil {
				nRemoved++
			}
		}
	}
	return nRemoved, nil
}

// CleanupOldRemotePrompt 는 원격 폴더에 대해 같은 정리를 한다 (SFTP 쪽).
func (pService *Service) CleanupOldRemotePrompt(szRemoteDir string, iDays int, tNow time.Time) (int, error) {
	if iDays <= 0 {
		return 0, nil
	}
	infos, errRead := pService.be.ReadDir(szRemoteDir)
	if errRead != nil {
		return 0, errRead
	}
	nRemoved := 0
	limit := tNow.Add(-time.Duration(iDays) * 24 * time.Hour)
	for _, info := range infos {
		if info.IsDir() || !strings.HasPrefix(info.Name(), G_szPromptPrefix) {
			continue
		}
		if info.ModTime().Before(limit) {
			if errRemove := pService.be.RemoveFile(path.Join(szRemoteDir, info.Name())); errRemove == nil {
				nRemoved++
			}
		}
	}
	return nRemoved, nil
}

// UploadPromptBytes 는 이미지·버퍼 데이터를 원격 폴더에 tg- 이름으로 올린다.
func (pService *Service) UploadPromptBytes(bData []byte, szGivenName string, szRemoteDir string, tNow time.Time) (string, error) {
	szName := PromptImageName(tNow, RandHex4(), szGivenName)
	szRemote := path.Join(szRemoteDir, szName)
	dst, errCreate := pService.be.OpenWrite(szRemote)
	if errCreate != nil {
		return "", errCreate
	}
	defer dst.Close()
	if _, errWrite := dst.Write(bData); errWrite != nil {
		return "", errWrite
	}
	return szRemote, nil
}

// UploadPromptPaths 는 붙여넣기용 파일·폴더를 올린다. 최상위 파일은 tg- 이름으로, 폴더는 구조 그대로 올린다.
// 반환은 올린 원격 경로 배열이다.
func (pService *Service) UploadPromptPaths(slLocal []string, szRemoteDir string, tNow time.Time) ([]string, error) {
	var slOut []string
	for _, szLocal := range slLocal {
		info, errStat := os.Stat(szLocal)
		if errStat != nil {
			return slOut, errStat
		}
		if info.IsDir() {
			if errUp := pService.uploadOne(szLocal, path.Join(szRemoteDir, info.Name()), &slOut); errUp != nil {
				return slOut, errUp
			}
			continue
		}
		szRemote := path.Join(szRemoteDir, PromptFileName(tNow, RandHex4(), info.Name()))
		if errFile := pService.uploadFile(szLocal, szRemote, func(int64) {}); errFile != nil {
			return slOut, errFile
		}
		slOut = append(slOut, szRemote)
	}
	return slOut, nil
}
