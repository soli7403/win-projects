package session

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// g_reAnsi 는 ANSI 이스케이프 시퀀스(CSI 와 OSC)를 찾는다 (session.md §6).
var g_reAnsi = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b[@-Z\\\\-_]")

// g_reFileNameBad 는 로그 파일 이름에서 `\ / : * ? " < > |` 를 `_` 로 바꾸는 규칙이다.
var g_reFileNameBad = regexp.MustCompile(`[\\/:*?"<>|]`)

// StripForLog 는 로그에 남길 텍스트를 만든다: ANSI 시퀀스와 CR 을 제거한다.
func StripForLog(szText string) string {
	szOut := g_reAnsi.ReplaceAllString(szText, "")
	return strings.ReplaceAll(szOut, "\r", "")
}

// LogFileName 은 "{별칭}-{YYYYMMDD-HHmm}.log" 이름을 만든다.
func LogFileName(szAlias string, tNow time.Time) string {
	return g_reFileNameBad.ReplaceAllString(szAlias, "_") + "-" + tNow.Format("20060102-1504") + ".log"
}

// Logger 는 세션 로그 파일을 이어 쓴다.
type Logger struct {
	mtxData sync.Mutex
	pFile   *os.File
}

// OpenLog 는 데이터 폴더의 logs 아래 로그 파일을 연다 (소유자 전용 권한, 이어 쓰기).
func OpenLog(szDataDir string, szAlias string, tNow time.Time) (*Logger, error) {
	szDir := filepath.Join(szDataDir, "logs")
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return nil, errMkdir
	}
	pFile, errOpen := os.OpenFile(filepath.Join(szDir, LogFileName(szAlias, tNow)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if errOpen != nil {
		return nil, errOpen
	}
	return &Logger{pFile: pFile}, nil
}

// Write 는 출력 조각을 정리해 기록한다.
func (pLogger *Logger) Write(szText string) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pFile == nil {
		return
	}
	_, _ = pLogger.pFile.WriteString(StripForLog(szText))
}

// Close 는 파일을 닫는다. 두 번 불러도 안전하다.
func (pLogger *Logger) Close() error {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pFile == nil {
		return nil
	}
	errClose := pLogger.pFile.Close()
	pLogger.pFile = nil
	return errClose
}
