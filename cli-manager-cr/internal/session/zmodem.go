package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dariterm/internal/i18n"
	"dariterm/internal/zmodem"
)

// ZMODEM 파일 전송을 세션에 잇는다 (spec/behavior/zmodem.md, D114).
// 읽기 루프가 출력에서 시작 표시를 찾으면, 전송이 끝날 때까지 연결의 읽기·쓰기를 ZMODEM 엔진에 넘긴다.
// 그동안 터미널 입력은 서버로 보내지 않는다(Ctrl+C 만 전송 취소로 쓴다).

// ZmodemHooks 는 세션이 ZMODEM 전송에 쓰는 앱 쪽 기능이다. nil 이면 이 세션은 ZMODEM 을 쓰지 않는다.
type ZmodemHooks struct {
	// Enabled 는 지금 ZMODEM 이 켜져 있는지다 (설정 zmodemEnabled). 시작 표시를 볼 때마다 묻는다.
	Enabled func() bool
	// DownloadDir 는 받은 파일을 저장할 폴더다 (설정 zmodemDownloadDir, 비면 OS 의 다운로드 폴더).
	DownloadDir func() string
	// PickFiles 는 보낼 파일을 고르는 대화상자를 띄운다. 취소하면 빈 목록이다.
	PickFiles func() ([]string, error)
}

const (
	// g_cdurZmodemProgress 는 진행 표시를 고쳐 쓰는 최소 간격이다.
	g_cdurZmodemProgress = 250 * time.Millisecond
	// g_cszZmodemTempSuffix 는 받는 중인 파일 이름 뒤에 붙는 꼬리다. 다 받으면 떼어 낸다.
	g_cszZmodemTempSuffix = ".dtpart"
	// g_ciZmodemNameMaxRunes 는 저장하는 파일 이름의 최대 길이다.
	g_ciZmodemNameMaxRunes = 200
	// g_ciZmodemNameMaxTries 는 같은 이름이 있을 때 번호를 붙여 보는 최대 횟수다.
	g_ciZmodemNameMaxTries = 1000
)

// zmodemOn 은 이 세션에서 ZMODEM 시작 표시를 살필지다.
func (pSession *Session) zmodemOn() bool {
	pHooks := pSession.cfg.Zmodem
	return pHooks != nil && (pHooks.Enabled == nil || pHooks.Enabled())
}

// cancelZmodem 은 진행 중인 전송을 취소한다: 엔진에 알리고 상대에게 중단 바이트열을 보낸다.
func (pSession *Session) cancelZmodem(conn Transport) {
	if pSession.zCancel.Swap(true) {
		return // 이미 취소했다
	}
	_ = conn.Write(zmodem.AbortSequence())
}

// runZmodem 은 전송 하나를 끝까지 돌린다. 읽기 루프 고루틴에서 부른다.
// 돌려주는 바이트는 ZMODEM 이 아닌 것으로 남은 출력(화면으로 보낸다), 오류는 연결이 끊겼을 때만 nil 이 아니다.
func (pSession *Session) runZmodem(conn Transport, kind zmodem.Kind, bEngine []byte) ([]byte, error) {
	pSession.zCancel.Store(false)
	pSession.zActive.Store(true)
	defer pSession.zActive.Store(false)

	pHooks := pSession.cfg.Zmodem
	bSending := kind == zmodem.KindRemoteAccepts
	pView := &zmodemView{pSession: pSession, bSending: bSending}
	opts := zmodem.Options{Cancelled: pSession.zCancel.Load, Progress: pView.onProgress}

	var result zmodem.Result
	var errRun error
	if bSending {
		result, errRun = zmodem.Send(conn, bEngine, func() ([]zmodem.FileSource, error) {
			pSession.writeOutput(noticeLine(i18n.T("be.zm.pick")))
			if pHooks.PickFiles == nil {
				return nil, nil
			}
			slPaths, errPick := pHooks.PickFiles()
			if errPick != nil {
				return nil, errPick
			}
			return zmodemSources(slPaths), nil
		}, opts)
	} else {
		szDir := ""
		if pHooks.DownloadDir != nil {
			szDir = pHooks.DownloadDir()
		}
		pView.szDir = szDir
		result, errRun = zmodem.Receive(conn, bEngine, &zmodemSink{szDir: szDir, pView: pView}, opts)
	}

	var errStream *zmodem.StreamError
	switch {
	case errRun == nil:
		pSession.writeOutput(noticeLine(i18n.T("be.zm.done", "n", result.Files, "size", formatBytes(result.Bytes))))
	case errors.Is(errRun, zmodem.ErrNotZmodem):
		// 시작 표시처럼 보였을 뿐이다: 아무 안내 없이 읽은 바이트를 화면으로 되돌린다.
	case errors.As(errRun, &errStream):
		return result.Leftover, errStream.Err
	case errors.Is(errRun, zmodem.ErrCancelled):
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.cancelled")))
	case errors.Is(errRun, zmodem.ErrRemoteCancelled):
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.remoteCancelled")))
	default:
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.failed", "error", errRun)))
	}
	return result.Leftover, nil
}

// noticeLine 은 터미널 안내 한 줄(청록 ●)을 만든다.
func noticeLine(szText string) string {
	return "\r\x1b[2K\x1b[36m● " + szText + "\x1b[0m\r\n"
}

// formatBytes 는 바이트 수를 읽기 쉬운 단위로 적는다 (1024 단위).
func formatBytes(i64Bytes int64) string {
	const fUnit = 1024.0
	fValue := float64(i64Bytes)
	switch {
	case i64Bytes < 0:
		return "?"
	case fValue < fUnit:
		return strconv.FormatInt(i64Bytes, 10) + " B"
	case fValue < fUnit*fUnit:
		return fmt.Sprintf("%.1f KB", fValue/fUnit)
	case fValue < fUnit*fUnit*fUnit:
		return fmt.Sprintf("%.1f MB", fValue/(fUnit*fUnit))
	default:
		return fmt.Sprintf("%.2f GB", fValue/(fUnit*fUnit*fUnit))
	}
}

// zmodemView 는 전송 진행을 터미널에 보인다. 진행률은 한 줄을 고쳐 쓰고(로그에는 남기지 않는다),
// 파일 하나가 끝나면 결과 한 줄을 남긴다.
type zmodemView struct {
	pSession  *Session
	bSending  bool
	szDir     string
	tLastDraw time.Time
	bLineOpen bool   // 진행 줄을 그려 둔 상태 (끝낼 때 줄을 바꿔야 한다)
	szSavedAs string // 받은 파일이 저장된 경로 (zmodemSink 가 알려 준다)
}

func (pView *zmodemView) onProgress(ev zmodem.Progress) {
	szName := displayName(ev.Name)
	switch ev.State {
	case zmodem.G_ciStateStart:
		pView.tLastDraw = time.Time{}
	case zmodem.G_ciStateData:
		if time.Since(pView.tLastDraw) < g_cdurZmodemProgress {
			return
		}
		pView.tLastDraw = time.Now()
		szKey := "be.zm.receiving"
		if pView.bSending {
			szKey = "be.zm.sending"
		}
		szAmount := formatBytes(ev.Done)
		if ev.Total > 0 {
			szAmount = strconv.FormatInt(ev.Done*100/ev.Total, 10) + "% (" + formatBytes(ev.Done) + " / " + formatBytes(ev.Total) + ")"
		}
		// 진행 줄은 화면에만 보낸다: 로그에는 CR 이 지워져 한 줄로 길게 붙기 때문이다.
		pView.pSession.batcher.Add("\r\x1b[2K\x1b[36m● " + i18n.T(szKey, "name", szName, "amount", szAmount) + "\x1b[0m")
		pView.bLineOpen = true
	case zmodem.G_ciStateDone:
		pView.bLineOpen = false
		if pView.bSending {
			pView.pSession.writeOutput(noticeLine(i18n.T("be.zm.sent", "name", szName, "size", formatBytes(ev.Done))))
		} else {
			pView.pSession.writeOutput(noticeLine(i18n.T("be.zm.received", "name", szName, "size", formatBytes(ev.Done), "path", pView.szSavedAs)))
		}
	case zmodem.G_ciStateSkipped:
		pView.pSession.writeOutput(noticeLine(i18n.T("be.zm.skipped", "name", szName)))
	}
}

// endLine 은 그려 둔 진행 줄이 있으면 지운다 (취소·실패 안내를 그 자리에 쓰기 위해).
func (pView *zmodemView) endLine() {
	if pView.bLineOpen {
		pView.pSession.batcher.Add("\r\x1b[2K")
		pView.bLineOpen = false
	}
}

// displayName 은 화면에 보일 파일 이름이다: 제어 문자를 빼서 터미널을 조작하지 못하게 한다.
func displayName(szName string) string {
	return strings.Map(func(ch rune) rune {
		if ch < 0x20 || ch == 0x7f {
			return '_'
		}
		return ch
	}, szName)
}

// g_mReservedNames 는 Windows 가 파일 이름으로 받지 않는 장치 이름이다 (확장자가 붙어도 안 된다).
var g_mReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SafeFileName 은 서버가 준 파일 이름을 이 PC 에 저장할 이름으로 다듬는다.
// 경로는 떼어 내고(폴더 밖에 쓰지 못하게), 금지·제어 문자는 `_` 로 바꾸고, 끝의 공백·점을 지우고,
// 장치 이름은 앞에 `_` 를 붙이고, 길이를 줄이고, 비면 "zmodem-file" 로 한다.
func SafeFileName(szName string) string {
	if iCut := strings.LastIndexAny(szName, `/\`); iCut >= 0 {
		szName = szName[iCut+1:]
	}
	szOut := strings.Trim(g_reFileNameBad.ReplaceAllString(szName, "_"), " .")
	if slRunes := []rune(szOut); len(slRunes) > g_ciZmodemNameMaxRunes {
		szOut = strings.Trim(string(slRunes[:g_ciZmodemNameMaxRunes]), " .")
	}
	if szOut == "" {
		return "zmodem-file"
	}
	szStem, _, _ := strings.Cut(szOut, ".")
	if g_mReservedNames[strings.ToUpper(szStem)] {
		szOut = "_" + szOut
	}
	return szOut
}

// uniquePath 는 szDir 안에서 아직 없는 경로를 찾는다: "이름.확장자" → "이름 (2).확장자" → …
// 받는 중인 임시 파일(.dtpart)이 있는 이름도 쓰는 중으로 본다.
func uniquePath(szDir string, szName string) (string, error) {
	szExt := filepath.Ext(szName)
	szStem := strings.TrimSuffix(szName, szExt)
	for iSeq := 1; iSeq <= g_ciZmodemNameMaxTries; iSeq++ {
		szTry := szName
		if iSeq >= 2 {
			szTry = szStem + " (" + strconv.Itoa(iSeq) + ")" + szExt
		}
		szPath := filepath.Join(szDir, szTry)
		_, errFinal := os.Lstat(szPath)
		_, errTemp := os.Lstat(szPath + g_cszZmodemTempSuffix)
		if errors.Is(errFinal, os.ErrNotExist) && errors.Is(errTemp, os.ErrNotExist) {
			return szPath, nil
		}
	}
	return "", i18n.NewError("be.zm.noName")
}

// zmodemSink 는 받은 파일을 폴더에 저장한다. 받는 동안은 "<이름>.dtpart" 에 쓰고 다 받으면 이름을 바꾼다.
// 같은 이름이 있으면 덮어쓰지 않고 번호를 붙인다.
type zmodemSink struct {
	szDir string
	pView *zmodemView
}

func (pSink *zmodemSink) Accept(info zmodem.FileInfo) (zmodem.FileWriter, error) {
	if pSink.szDir == "" {
		return nil, i18n.NewError("be.zm.noDir")
	}
	if errMkdir := os.MkdirAll(pSink.szDir, 0o755); errMkdir != nil {
		return nil, errMkdir
	}
	szPath, errPath := uniquePath(pSink.szDir, SafeFileName(info.Name))
	if errPath != nil {
		return nil, errPath
	}
	pFile, errOpen := os.OpenFile(szPath+g_cszZmodemTempSuffix, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errOpen != nil {
		return nil, errOpen
	}
	return &zmodemFile{pFile: pFile, szPath: szPath, tModTime: info.ModTime, pView: pSink.pView}, nil
}

// zmodemFile 은 받는 파일 하나다.
type zmodemFile struct {
	pFile    *os.File
	szPath   string // 다 받은 뒤의 최종 경로
	tModTime time.Time
	pView    *zmodemView
}

func (pOut *zmodemFile) Write(bData []byte) (int, error) {
	return pOut.pFile.Write(bData)
}

// Commit 은 파일을 닫고 최종 이름으로 바꾼다. 그사이 같은 이름이 생겼으면 번호를 붙인 새 이름으로 한다.
func (pOut *zmodemFile) Commit() error {
	szTemp := pOut.szPath + g_cszZmodemTempSuffix
	if errClose := pOut.pFile.Close(); errClose != nil {
		_ = os.Remove(szTemp)
		return errClose
	}
	szFinal := pOut.szPath
	if _, errStat := os.Lstat(szFinal); errStat == nil {
		// 받는 동안 다른 프로그램이 같은 이름을 만들었다: 덮어쓰지 않는다.
		_ = os.Rename(szTemp, szTemp+".tmp") // 임시 이름을 비켜 두어야 uniquePath 가 원래 이름을 다시 고르지 않는다
		szTemp += ".tmp"
		szNew, errPath := uniquePath(filepath.Dir(szFinal), filepath.Base(szFinal))
		if errPath != nil {
			_ = os.Remove(szTemp)
			return errPath
		}
		szFinal = szNew
	}
	if errRename := os.Rename(szTemp, szFinal); errRename != nil {
		_ = os.Remove(szTemp)
		return errRename
	}
	if !pOut.tModTime.IsZero() {
		_ = os.Chtimes(szFinal, time.Now(), pOut.tModTime)
	}
	if pOut.pView != nil {
		pOut.pView.szSavedAs = szFinal
	}
	return nil
}

// Abort 는 받다 만 파일을 닫고 지운다.
func (pOut *zmodemFile) Abort() {
	_ = pOut.pFile.Close()
	_ = os.Remove(pOut.szPath + g_cszZmodemTempSuffix)
}

// zmodemSources 는 고른 경로들을 보낼 파일 목록으로 바꾼다. 폴더와 읽을 수 없는 항목은 뺀다.
func zmodemSources(slPaths []string) []zmodem.FileSource {
	var slOut []zmodem.FileSource
	for _, szPath := range slPaths {
		info, errStat := os.Stat(szPath)
		if errStat != nil || !info.Mode().IsRegular() {
			continue
		}
		szOpen := szPath
		slOut = append(slOut, zmodem.FileSource{
			Name:    filepath.Base(szPath),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Open:    func() (io.ReadSeekCloser, error) { return os.Open(szOpen) },
		})
	}
	return slOut
}
