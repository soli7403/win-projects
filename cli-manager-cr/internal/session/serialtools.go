package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dariterm/internal/i18n"
	"dariterm/internal/xmodem"
)

// 시리얼 도구: HEX 보기·HEX 보내기·XMODEM (spec/behavior/serial-tools.md, D163·D164).
// 세션 종류와 상관없이 동작하지만 화면에서는 시리얼 칸에만 메뉴를 둔다.

// g_ciHexPerLine 은 HEX 보기 한 줄의 바이트 수다.
const g_ciHexPerLine = 16

// HexDump 는 받은 바이트를 "41 42 0D 0A  |AB..|" 꼴의 줄로 바꾼다. 16바이트마다 줄을 바꾸고, 조각마다 새 줄에서 시작한다.
// 글자 칸은 0x20~0x7E 만 그대로, 그 밖은 "." 이다.
func HexDump(bData []byte) string {
	var sb strings.Builder
	for iStart := 0; iStart < len(bData); iStart += g_ciHexPerLine {
		iEnd := iStart + g_ciHexPerLine
		if iEnd > len(bData) {
			iEnd = len(bData)
		}
		bLine := bData[iStart:iEnd]
		for i := 0; i < g_ciHexPerLine; i++ {
			if i < len(bLine) {
				fmt.Fprintf(&sb, "%02X ", bLine[i])
			} else {
				sb.WriteString("   ")
			}
		}
		sb.WriteString(" |")
		for _, b := range bLine {
			if b >= 0x20 && b <= 0x7E {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\r\n")
	}
	return sb.String()
}

// ErrHexInput 은 HEX 보내기 입력이 16진 바이트가 아니라는 오류다.
var ErrHexInput = errors.New("hex: invalid input")

// ParseHex 는 "01 03 0a FF", "01030AFF", "0x01,0x03" 같은 입력을 바이트로 바꾼다.
// 공백·쉼표·콜론·줄바꿈은 구분자, 낱말 앞의 0x 는 뗀다. 구분자 없이 붙은 자리는 두 글자씩 끊는다(홀수면 오류).
func ParseHex(szText string) ([]byte, error) {
	slWords := strings.FieldsFunc(szText, func(r rune) bool {
		return r == ' ' || r == ',' || r == ':' || r == '\t' || r == '\r' || r == '\n' || r == ';'
	})
	var bOut []byte
	for _, szWord := range slWords {
		szWord = strings.TrimPrefix(strings.TrimPrefix(szWord, "0x"), "0X")
		if len(szWord) == 1 {
			szWord = "0" + szWord // "A" 는 0A
		}
		if len(szWord)%2 != 0 || szWord == "" {
			return nil, ErrHexInput
		}
		for i := 0; i < len(szWord); i += 2 {
			var b byte
			if _, errScan := fmt.Sscanf(szWord[i:i+2], "%02x", &b); errScan != nil || !isHex(szWord[i]) || !isHex(szWord[i+1]) {
				return nil, ErrHexInput
			}
			bOut = append(bOut, b)
		}
	}
	if len(bOut) == 0 {
		return nil, ErrHexInput
	}
	return bOut, nil
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// SetHexView 는 받은 출력을 HEX 로 보일지 정한다. 세션 정보(hexView)에 실어 알린다.
func (pSession *Session) SetHexView(bOn bool) {
	pSession.mtxData.Lock()
	bChanged := pSession.info.M_bHexView != bOn
	pSession.info.M_bHexView = bOn
	pSession.mtxData.Unlock()
	pSession.hexView.Store(bOn)
	if bChanged {
		pSession.publishState()
	}
}

// SendBytes 는 인코딩을 거치지 않고 바이트를 그대로 보낸다 (HEX 보내기). 연결이 없거나 전송 중이면 오류.
func (pSession *Session) SendBytes(bData []byte) error {
	conn := pSession.Transport()
	if conn == nil {
		return i18n.NewError("be.serial.notConnected")
	}
	if pSession.zActive.Load() || pSession.xm.Load() != nil {
		return i18n.NewError("be.xm.busy")
	}
	return pSession.send(conn, bData)
}

// xmodemPipe 는 XMODEM 전송 동안 읽기 루프가 받은 바이트를 엔진에 넘기는 통로다 (xmodem.Port).
type xmodemPipe struct {
	ch       chan []byte
	bRest    []byte
	onceStop sync.Once
	chStop   chan struct{} // 연결이 끝나면 닫힌다
	conn     Transport
	pSession *Session
}

// feed 는 읽기 루프가 받은 조각을 넘긴다. 엔진이 늦으면 기다린다(조각은 복사해 둔다).
func (pPipe *xmodemPipe) feed(bData []byte) {
	bCopy := append([]byte(nil), bData...)
	select {
	case pPipe.ch <- bCopy:
	case <-pPipe.chStop:
	}
}

// stop 은 연결이 끝났음을 알린다.
func (pPipe *xmodemPipe) stop() {
	pPipe.onceStop.Do(func() { close(pPipe.chStop) })
}

// ReadByteWithin 은 xmodem.Port 다.
func (pPipe *xmodemPipe) ReadByteWithin(durWait time.Duration) (byte, error) {
	if len(pPipe.bRest) == 0 {
		timer := time.NewTimer(durWait)
		defer timer.Stop()
		select {
		case bData := <-pPipe.ch:
			pPipe.bRest = bData
		case <-pPipe.chStop:
			return 0, io.EOF
		case <-timer.C:
			return 0, xmodem.ErrTimeout
		}
		if len(pPipe.bRest) == 0 {
			return 0, xmodem.ErrTimeout
		}
	}
	b := pPipe.bRest[0]
	pPipe.bRest = pPipe.bRest[1:]
	return b, nil
}

// Write 는 xmodem.Port 다.
func (pPipe *xmodemPipe) Write(bData []byte) error {
	return pPipe.pSession.send(pPipe.conn, bData)
}

// XmodemSend 는 파일 하나를 XMODEM 으로 보낸다(장치가 받기를 시작하면). 곧바로 돌아가고 전송은 뒤에서 한다.
func (pSession *Session) XmodemSend(szPath string, bOneK bool) error {
	pFile, errOpen := os.Open(szPath)
	if errOpen != nil {
		return errOpen
	}
	var i64Size int64
	if info, errStat := pFile.Stat(); errStat == nil {
		i64Size = info.Size()
	}
	pPipe, errStart := pSession.beginXmodem()
	if errStart != nil {
		_ = pFile.Close()
		return errStart
	}
	szName := filepath.Base(szPath)
	pView := &xmodemView{pSession: pSession, szName: szName, bSending: true}
	pSession.writeOutput(noticeLine(i18n.T("be.xm.waitReceiver", "name", szName)))
	go func() {
		defer pFile.Close()
		i64Done, errRun := xmodem.Send(pPipe, pFile, i64Size, xmodem.Options{OneK: bOneK, Cancelled: pSession.xmCancel.Load, Progress: pView.onProgress})
		pSession.endXmodem(pPipe)
		pView.finish(i64Done, errRun)
	}()
	return nil
}

// XmodemReceive 는 XMODEM 으로 받아 szPath 에 저장한다. 받는 동안은 꼬리(.dtpart)를 붙인 이름에 쓰고 다 받으면 바꾼다.
func (pSession *Session) XmodemReceive(szPath string) error {
	szTemp := szPath + g_cszZmodemTempSuffix
	pFile, errCreate := os.Create(szTemp)
	if errCreate != nil {
		return errCreate
	}
	pPipe, errStart := pSession.beginXmodem()
	if errStart != nil {
		_ = pFile.Close()
		_ = os.Remove(szTemp)
		return errStart
	}
	szName := filepath.Base(szPath)
	pView := &xmodemView{pSession: pSession, szName: szName}
	pSession.writeOutput(noticeLine(i18n.T("be.xm.waitSender", "name", szName)))
	go func() {
		i64Done, errRun := xmodem.Receive(pPipe, pFile, xmodem.Options{Cancelled: pSession.xmCancel.Load, Progress: pView.onProgress})
		pSession.endXmodem(pPipe)
		errClose := pFile.Close()
		if errRun == nil && errClose != nil {
			errRun = errClose
		}
		if errRun == nil {
			_ = os.Remove(szPath) // 저장 대화상자에서 덮어쓰기를 골랐다
			errRun = os.Rename(szTemp, szPath)
		}
		if errRun != nil {
			_ = os.Remove(szTemp)
		}
		pView.finish(i64Done, errRun)
	}()
	return nil
}

// XmodemCancel 은 진행 중인 XMODEM 전송을 취소한다. 없으면 아무것도 하지 않는다.
func (pSession *Session) XmodemCancel() {
	if pSession.xm.Load() != nil {
		pSession.xmCancel.Store(true)
	}
}

// beginXmodem 은 전송을 시작할 수 있으면 통로를 걸어 둔다. 읽기 루프는 그때부터 받은 바이트를 통로로 보낸다.
func (pSession *Session) beginXmodem() (*xmodemPipe, error) {
	conn := pSession.Transport()
	if conn == nil {
		return nil, i18n.NewError("be.serial.notConnected")
	}
	if pSession.zActive.Load() {
		return nil, i18n.NewError("be.xm.busy")
	}
	pPipe := &xmodemPipe{ch: make(chan []byte, 64), chStop: make(chan struct{}), conn: conn, pSession: pSession}
	if !pSession.xm.CompareAndSwap(nil, pPipe) {
		return nil, i18n.NewError("be.xm.busy")
	}
	pSession.xmCancel.Store(false)
	return pPipe, nil
}

// endXmodem 은 통로를 거둔다. 그 뒤에 받은 바이트는 다시 화면으로 간다.
func (pSession *Session) endXmodem(pPipe *xmodemPipe) {
	pSession.xm.CompareAndSwap(pPipe, nil)
	pPipe.stop()
}

// feedXmodem 은 읽기 루프에서 부른다. 전송 중이면 조각을 엔진에 넘기고 true 다.
func (pSession *Session) feedXmodem(bData []byte) bool {
	pPipe := pSession.xm.Load()
	if pPipe == nil {
		return false
	}
	pPipe.feed(bData)
	return true
}

// stopXmodemOnClose 는 연결이 끝났을 때 진행 중인 전송을 끝낸다.
func (pSession *Session) stopXmodemOnClose() {
	if pPipe := pSession.xm.Load(); pPipe != nil {
		pPipe.stop()
	}
}

// xmodemView 는 전송 진행을 터미널 한 줄에 고쳐 쓴다.
type xmodemView struct {
	pSession *Session
	szName   string
	bSending bool
	tmLast   time.Time
	bShown   bool
	mtxData  sync.Mutex
}

func (pView *xmodemView) onProgress(i64Done int64, i64Total int64) {
	pView.mtxData.Lock()
	defer pView.mtxData.Unlock()
	if time.Since(pView.tmLast) < g_cdurZmodemProgress {
		return
	}
	pView.tmLast = time.Now()
	pView.bShown = true
	szKey := "be.xm.receiving"
	if pView.bSending {
		szKey = "be.xm.sending"
	}
	szText := i18n.T(szKey, "name", pView.szName, "size", formatBytes(i64Done))
	if i64Total > 0 {
		szText += fmt.Sprintf(" / %s (%d%%)", formatBytes(i64Total), i64Done*100/i64Total)
	}
	pView.pSession.writeOutput("\r\x1b[2K\x1b[36m" + szText + "\x1b[0m")
}

func (pView *xmodemView) finish(i64Done int64, errRun error) {
	pView.mtxData.Lock()
	bShown := pView.bShown
	pView.mtxData.Unlock()
	if bShown {
		pView.pSession.writeOutput("\r\n")
	}
	var szText string
	switch {
	case errRun == nil:
		szText = i18n.T("be.xm.done", "name", pView.szName, "size", formatBytes(i64Done))
	case errors.Is(errRun, xmodem.ErrCancelled):
		szText = i18n.T("be.xm.cancelled")
	case errors.Is(errRun, xmodem.ErrRemoteCancelled):
		szText = i18n.T("be.xm.remoteCancelled")
	case errors.Is(errRun, xmodem.ErrNoStart):
		szText = i18n.T("be.xm.noStart")
	default:
		szText = i18n.T("be.xm.failed", "error", errRun)
	}
	pView.pSession.writeOutput(noticeLine(szText))
}

// xmodemState 는 Session 에 넣는 XMODEM 상태다 (session.go 의 필드).
type xmodemState struct {
	xm       atomic.Pointer[xmodemPipe] // 진행 중인 전송의 통로 (없으면 nil)
	xmCancel atomic.Bool                // 사용자가 전송을 취소했다
	hexView  atomic.Bool                // 받은 출력을 HEX 로 보인다
}
