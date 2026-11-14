// Package backup 은 백업 파일(.dtbackup) 의 형식을 다룬다 (spec/data/sync-backup.md §3).
package backup

import (
	"dariterm/internal/i18n"
	"encoding/json"
	"fmt"
	"time"
)

// G_cszAppName 은 이 앱이 쓰는 백업 app 값이다. 읽을 때는 원본 앱(BrightTerm) 백업의 G_cszAppNameOrigin 도 인정한다 (sync-backup.md §3).
const (
	G_cszAppName     = "dariterm"
	G_iFormatVersion = 1
)

// G_cszAppNameOrigin 은 원본 앱(BrightTerm) 백업의 app 값이다 (가져오기만 인정).
const G_cszAppNameOrigin = "brightterm"

// knownApp 은 백업 app 값이 이 앱 또는 원본 앱인지 본다.
func knownApp(szApp string) bool {
	return szApp == G_cszAppName || szApp == G_cszAppNameOrigin
}

// FilterName 은 대화상자 필터 이름이다(지금 언어, D75).
func FilterName() string {
	return i18n.T("be.backup.filter")
}

// G_cszFilterPattern 과 G_cszOpenPattern 은 대화상자 필터다.
const (
	G_cszFilterPattern = "*.dtbackup"
	G_cszOpenPattern   = "*.dtbackup;*.btbackup;*.json"
)

// ErrNotBackup 은 백업 파일이 아닐 때 돌려준다.
var ErrNotBackup = i18n.NewError("be.backup.notBackup")

// ErrUnreadable 은 JSON 으로 읽을 수 없을 때 돌려준다.
var ErrUnreadable = i18n.NewError("be.backup.unreadable")

// File 은 백업 파일 구조다 (data/sync-backup.md §3 표).
type File struct {
	M_szApp         string          `json:"app"`
	M_iVersion      int             `json:"version"`
	M_i64ExportedAt int64           `json:"exportedAt"`
	M_mapStore      map[string]any  `json:"store"`
	M_rawVault      json.RawMessage `json:"vault"`
}

// DefaultName 은 기본 파일 이름 dariterm-backup-YYYY-MM-DD.dtbackup 을 만든다.
func DefaultName(tmNow time.Time) string {
	return fmt.Sprintf("dariterm-backup-%s.dtbackup", tmNow.Format("2006-01-02"))
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

// Decode 는 백업 파일을 해석한다. app 이 dariterm·brightterm 이 아니면 ErrNotBackup 이다.
func Decode(bData []byte) (File, error) {
	var file File
	if errJson := json.Unmarshal(bData, &file); errJson != nil {
		return File{}, ErrUnreadable
	}
	if !knownApp(file.M_szApp) {
		return File{}, ErrNotBackup
	}
	if file.M_mapStore == nil {
		file.M_mapStore = map[string]any{}
	}
	return file, nil
}
