package session

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 세션 로그 (spec/behavior/session.md §6).
const (
	// G_ciLogNameMaxTries 는 같은 이름이 있을 때 순번을 붙여 다시 시도하는 최대 횟수다.
	G_ciLogNameMaxTries = 1000
	// G_ciLogBufferBytes 는 로그 쓰기 버퍼 크기다. G_cdurLogFlush 마다, 그리고 닫을 때 비운다.
	G_ciLogBufferBytes = 64 * 1024
	// G_cdurLogFlush 는 버퍼에 쌓인 로그를 디스크에 쓰는 주기다.
	G_cdurLogFlush = time.Second
	// g_ciLogCarryMax 는 미완성 ANSI 시퀀스로 보관하는 최대 길이다. 넘으면 시퀀스가 아니라고 보고 그대로 기록한다.
	g_ciLogCarryMax = 256
	// g_ciLogAliasMaxRunes 는 로그 파일 이름에 쓰는 별칭의 최대 길이다 (경로 260자 한도 대비).
	g_ciLogAliasMaxRunes = 80
)

// g_reAnsi 는 ANSI 이스케이프 시퀀스(CSI 와 OSC)를 찾는다 (session.md §6).
var g_reAnsi = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(\x07|\x1b\\\\)|\x1b[@-Z\\\\-_]")

// g_reFileNameBad 는 로그 파일 이름에서 금지 문자(`\ / : * ? " < > |`)와 제어 문자를 `_` 로 바꾸는 규칙이다.
var g_reFileNameBad = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f\x7f]`)

// 지금 열려 있는(기록 중인) 로그 파일의 절대 경로 집합. 오래된 로그 정리가 이 파일들은 건너뛴다.
var (
	g_mtxOpenLogs sync.Mutex
	g_mOpenLogs   = map[string]struct{}{}
)

// IsLogOpen 은 szPath 가 지금 세션이 기록 중인 로그 파일인지 알려 준다 (오래된 로그 정리용).
func IsLogOpen(szPath string) bool {
	szAbs, errAbs := filepath.Abs(szPath)
	if errAbs != nil {
		szAbs = szPath
	}
	g_mtxOpenLogs.Lock()
	defer g_mtxOpenLogs.Unlock()
	_, bOpen := g_mOpenLogs[szAbs]
	return bOpen
}

// StripForLog 는 로그에 남길 텍스트를 만든다: ANSI 시퀀스와 CR 을 제거한다.
func StripForLog(szText string) string {
	szOut := g_reAnsi.ReplaceAllString(szText, "")
	return strings.ReplaceAll(szOut, "\r", "")
}

// logAlias 는 별칭을 파일 이름용으로 다듬는다: 금지·제어 문자 치환, 앞뒤 공백·점 제거, 길이 제한, 비면 "session".
func logAlias(szAlias string) string {
	szOut := g_reFileNameBad.ReplaceAllString(szAlias, "_")
	szOut = strings.Trim(szOut, " .") // Windows 는 끝의 공백·점을 허용하지 않는다
	if slRunes := []rune(szOut); len(slRunes) > g_ciLogAliasMaxRunes {
		szOut = strings.Trim(string(slRunes[:g_ciLogAliasMaxRunes]), " .")
	}
	if szOut == "" {
		return "session"
	}
	return szOut
}

// LogFileName 은 "{별칭}-{YYYYMMDD-HHmmss}.log" 이름을 만든다 (session.md §6).
// iSeq 가 2 이상이면 ".log" 앞에 "-{iSeq}" 를 붙인다 (같은 초에 연 두 번째·세 번째 세션).
func LogFileName(szAlias string, tNow time.Time, iSeq int) string {
	szBase := logAlias(szAlias) + "-" + tNow.Format("20060102-150405")
	if iSeq >= 2 {
		szBase += "-" + strconv.Itoa(iSeq)
	}
	return szBase + ".log"
}

// Logger 는 세션 로그 파일에 쓴다. ANSI 시퀀스가 묶음 경계에서 잘려도 지울 수 있도록 꼬리를 보관하고,
// 디스크 쓰기는 버퍼에 모아 G_cdurLogFlush 마다 한 번 한다 (읽기 고루틴이 디스크를 기다리지 않게).
type Logger struct {
	mtxData    sync.Mutex
	szPath     string // 열린 파일의 절대 경로 (열린 로그 집합의 키)
	pFile      *os.File
	pWriter    *bufio.Writer
	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
	timerFlush *time.Timer // 버퍼에 내용이 있을 때만 걸린다
}

// OpenLog 는 데이터 폴더의 logs 아래 이 세션만의 로그 파일을 새로 만든다 (소유자 전용 권한, session.md §6).
// 같은 이름이 이미 있으면 "-2", "-3", … 을 붙여 없는 이름을 찾는다. 세션 하나에 파일 하나를 보장한다 (KI-001).
func OpenLog(szDataDir string, szAlias string, tNow time.Time) (*Logger, error) {
	szDir := filepath.Join(szDataDir, "logs")
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return nil, errMkdir
	}
	var errLast error
	for iSeq := 1; iSeq <= G_ciLogNameMaxTries; iSeq++ {
		szPath := filepath.Join(szDir, LogFileName(szAlias, tNow, iSeq))
		// O_EXCL: 이미 있으면 실패한다 (Windows 는 CREATE_NEW). 두 세션이 동시에 같은 이름을 만들어도 한쪽만 성공한다.
		pFile, errOpen := os.OpenFile(szPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if errOpen == nil {
			if szAbs, errAbs := filepath.Abs(szPath); errAbs == nil {
				szPath = szAbs
			}
			g_mtxOpenLogs.Lock()
			g_mOpenLogs[szPath] = struct{}{}
			g_mtxOpenLogs.Unlock()
			return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes)}, nil
		}
		if !errors.Is(errOpen, fs.ErrExist) {
			return nil, errOpen
		}
		errLast = errOpen
	}
	return nil, errLast
}

// Path 는 로그 파일의 절대 경로다.
func (pLogger *Logger) Path() string {
	return pLogger.szPath
}

// splitAnsiTail 은 텍스트 끝에 있는 미완성 ANSI 시퀀스를 떼어 낸다 (본문, 꼬리).
// 마지막 ESC 뒤가 완성된 시퀀스이면(또는 ESC 가 없으면) 꼬리는 비어 있다.
func splitAnsiTail(szIn string) (string, string) {
	// OSC(ESC ] … BEL 또는 ESC \)는 2바이트 ESC 규칙에도 걸리므로 먼저 본다: 종결이 아직 없으면 OSC 시작부터 꼬리다.
	if iOsc := strings.LastIndex(szIn, "\x1b]"); iOsc >= 0 {
		szOsc := szIn[iOsc:]
		if !strings.ContainsRune(szOsc[2:], 0x07) && !strings.Contains(szOsc[2:], "\x1b\\") {
			if len(szOsc) > g_ciLogCarryMax {
				return szIn, "" // 너무 길다: 시퀀스가 아니라 본문에 섞인 ESC 로 본다
			}
			return szIn[:iOsc], szOsc
		}
	}
	iEsc := strings.LastIndexByte(szIn, 0x1b)
	if iEsc < 0 {
		return szIn, ""
	}
	szRest := szIn[iEsc:]
	if len(szRest) > g_ciLogCarryMax {
		return szIn, "" // 너무 길다: 시퀀스가 아니라 본문에 섞인 ESC 로 본다
	}
	if slLoc := g_reAnsi.FindStringIndex(szRest); slLoc != nil && slLoc[0] == 0 {
		return szIn, "" // ESC 에서 시작하는 완성된 시퀀스가 있다 (뒤는 일반 글자)
	}
	return szIn[:iEsc], szRest
}

// Write 는 출력 조각을 정리해 버퍼에 쓴다. 끝이 미완성 시퀀스면 보관했다가 다음 조각과 이어 처리한다.
func (pLogger *Logger) Write(szText string) {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pWriter == nil {
		return
	}
	szBody, szTail := splitAnsiTail(pLogger.szCarry + szText)
	pLogger.szCarry = szTail
	if szBody == "" {
		return
	}
	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
	if pLogger.timerFlush == nil {
		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
	}
}

// flushTimer 는 주기 flush 다. 닫힌 뒤에는 아무 일도 하지 않는다.
func (pLogger *Logger) flushTimer() {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	pLogger.timerFlush = nil
	if pLogger.pWriter != nil {
		_ = pLogger.pWriter.Flush()
	}
}

// Close 는 남은 꼬리와 버퍼를 쓰고 파일을 닫는다. 두 번 불러도 안전하다.
func (pLogger *Logger) Close() error {
	pLogger.mtxData.Lock()
	defer pLogger.mtxData.Unlock()
	if pLogger.pFile == nil {
		return nil
	}
	if pLogger.timerFlush != nil {
		pLogger.timerFlush.Stop()
		pLogger.timerFlush = nil
	}
	if pLogger.szCarry != "" {
		_, _ = pLogger.pWriter.WriteString(StripForLog(pLogger.szCarry))
		pLogger.szCarry = ""
	}
	errFlush := pLogger.pWriter.Flush()
	errClose := pLogger.pFile.Close()
	pLogger.pFile = nil
	pLogger.pWriter = nil
	g_mtxOpenLogs.Lock()
	delete(g_mOpenLogs, pLogger.szPath)
	g_mtxOpenLogs.Unlock()
	if errFlush != nil {
		return errFlush
	}
	return errClose
}
