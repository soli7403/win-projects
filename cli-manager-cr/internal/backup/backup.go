// Package backup 은 백업 파일(.tgbackup) 의 형식을 다룬다 (spec/data/sync-backup.md §3).
package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// G_cszAppName 은 이 앱이 쓰는 백업 app 값이다. G_cszAppNameLegacy 도 읽을 때 인정한다.
const (
	G_cszAppName       = "termgo"
	G_cszAppNameLegacy = "brightterm"
	G_iFormatVersion   = 1
)

// G_cszFilterName 과 G_cszFilterPattern 은 대화상자 필터다.
const (
	G_cszFilterName    = "백업 (*.tgbackup)"
	G_cszFilterPattern = "*.tgbackup"
	G_cszOpenPattern   = "*.tgbackup;*.btbackup;*.json"
)

// ErrNotBackup 은 백업 파일이 아닐 때 돌려준다.
var ErrNotBackup = errors.New("백업 파일이 아닙니다")

// ErrUnreadable 은 JSON 으로 읽을 수 없을 때 돌려준다.
var ErrUnreadable = errors.New("백업 파일을 읽을 수 없습니다")

// File 은 백업 파일 구조다 (data/sync-backup.md §3 표).
type File struct {
	M_szApp         string          `json:"app"`
	M_iVersion      int             `json:"version"`
	M_i64ExportedAt int64           `json:"exportedAt"`
	M_mapStore      map[string]any  `json:"store"`
	M_rawVault      json.RawMessage `json:"vault"`
}

// DefaultName 은 기본 파일 이름 dariterm-backup-YYYY-MM-DD.tgbackup 을 만든다 (파일 안 앱 식별자는 호환을 위해 termgo 유지).
func DefaultName(tmNow time.Time) string {
	return fmt.Sprintf("dariterm-backup-%s.tgbackup", tmNow.Format("2006-01-02"))
}

// NewFile 은 내보낼 파일 구조를 만든다. rawVault 가 nil 이면 볼트 없음으로 쓴다.
func NewFile(mapStore map[string]any, rawVault json.RawMessage, tmNow time.Time) File {
	return File{
		M_szApp:         G_cszAppName,
		M_iVersion:      G_iFormatVersion,
		M_i64ExportedAt: tmNow.UnixMilli(),
		M_mapStore:      mapStore,
		M_rawVault:      rawVault,
	}
}

// Encode 는 들여쓰기 2칸 JSON 으로 바꾼다.
func Encode(file File) ([]byte, error) {
	return json.MarshalIndent(file, "", "  ")
}

// Decode 는 백업 파일을 해석한다. app 이 termgo 나 brightterm 이 아니면 ErrNotBackup 이다.
func Decode(bData []byte) (File, error) {
	var file File
	if errJson := json.Unmarshal(bData, &file); errJson != nil {
		return File{}, ErrUnreadable
	}
	if file.M_szApp != G_cszAppName && file.M_szApp != G_cszAppNameLegacy {
		return File{}, ErrNotBackup
	}
	if file.M_mapStore == nil {
		file.M_mapStore = map[string]any{}
	}
	return file, nil
}
