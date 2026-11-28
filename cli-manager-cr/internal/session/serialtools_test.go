package session

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dariterm/internal/xmodem"
)

// deviceEnd 는 시리얼 장치 쪽 끝이다 (xmodem.Port).
type deviceEnd struct {
	pLink *linkTransport
	bRest []byte
}

func (pEnd *deviceEnd) ReadByteWithin(durWait time.Duration) (byte, error) {
	if len(pEnd.bRest) == 0 {
		timer := time.NewTimer(durWait)
		defer timer.Stop()
		select {
		case bData := <-pEnd.pLink.chToRemote:
			pEnd.bRest = bData
		case <-pEnd.pLink.chClosed:
			return 0, io.EOF
		case <-timer.C:
			return 0, xmodem.ErrTimeout
		}
	}
	b := pEnd.bRest[0]
	pEnd.bRest = pEnd.bRest[1:]
	return b, nil
}

func (pEnd *deviceEnd) Write(bData []byte) error {
	return (&remoteEnd{pLink: pEnd.pLink}).Write(bData)
}

// fastXmodem 은 시험이 빨리 끝나도록 XMODEM 시한을 줄인다.
func fastXmodem(t *testing.T) {
	t.Helper()
	durStart, durBlock, durEach, durByte, durPoll := xmodem.G_durStart, xmodem.G_durBlock, xmodem.G_durStartEach, xmodem.G_durByte, xmodem.G_durPoll
	xmodem.G_durStart, xmodem.G_durBlock, xmodem.G_durStartEach, xmodem.G_durByte, xmodem.G_durPoll = 2*time.Second, 300*time.Millisecond, 200*time.Millisecond, 100*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() {
		xmodem.G_durStart, xmodem.G_durBlock, xmodem.G_durStartEach, xmodem.G_durByte, xmodem.G_durPoll = durStart, durBlock, durEach, durByte, durPoll
	})
}

func TestHexDumpFormat(t *testing.T) {
	if got := HexDump([]byte("AB\r\n")); got != "41 42 0D 0A "+strings.Repeat("   ", 12)+" |AB..|\r\n" {
		t.Fatalf("HexDump = %q", got)
	}
	bData := bytes.Repeat([]byte{0x7F}, 17)
	slLines := strings.Split(strings.TrimSuffix(HexDump(bData), "\r\n"), "\r\n")
	if len(slLines) != 2 || !strings.HasSuffix(slLines[0], "|"+strings.Repeat(".", 16)+"|") || !strings.HasPrefix(slLines[1], "7F    ") {
		t.Fatalf("17바이트 = %q", slLines)
	}
	if HexDump(nil) != "" {
		t.Fatal("빈 입력")
	}
}

func TestParseHex(t *testing.T) {
	for szIn, bWant := range map[string][]byte{
		"01 03 0a FF":    {0x01, 0x03, 0x0A, 0xFF},
		"01030AFF":       {0x01, 0x03, 0x0A, 0xFF},
		"0x01,0x03;0X0a": {0x01, 0x03, 0x0A},
		"a b\n  c":       {0x0A, 0x0B, 0x0C},
		"de:ad:be:ef":    {0xDE, 0xAD, 0xBE, 0xEF},
	} {
		bGot, errParse := ParseHex(szIn)
		if errParse != nil || !bytes.Equal(bGot, bWant) {
			t.Fatalf("ParseHex(%q) = %X %v", szIn, bGot, errParse)
		}
	}
	for _, szBad := range []string{"", "   ", "123", "0g", "zz", "0x", "+1"} {
		if _, errParse := ParseHex(szBad); errParse == nil {
			t.Fatalf("ParseHex(%q) 는 오류여야 한다", szBad)
		}
	}
}

func TestHexViewShowsBytesAndSendBytesIsRaw(t *testing.T) {
	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, nil)
	pSession.SetHexView(true)
	if !pSession.Snapshot().M_bHexView {
		t.Fatal("세션 정보에 hexView 가 실려야 한다")
	}
	pLink.chToApp <- []byte{0x01, 'A', 0xFF}
	waitUntil(t, func() bool {
		return strings.Contains(rec.output(), "01 41 FF ") && strings.Contains(rec.output(), "|.A.|")
	}, time.Second)
	pSession.SetHexView(false)
	pLink.chToApp <- []byte("plain")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "plain") }, time.Second)

	if errSend := pSession.SendBytes([]byte{0x00, 0xC5, 0xCD}); errSend != nil {
		t.Fatal(errSend)
	}
	select {
	case bGot := <-pLink.chToRemote:
		if !bytes.Equal(bGot, []byte{0x00, 0xC5, 0xCD}) {
			t.Fatalf("보낸 바이트 = %X (인코딩을 거치면 안 된다)", bGot)
		}
	case <-time.After(time.Second):
		t.Fatal("보내지 않았다")
	}
}

func TestXmodemSendThroughSession(t *testing.T) {
	fastXmodem(t)
	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, nil)
	bData := make([]byte, 5000)
	rand.New(rand.NewSource(1)).Read(bData)
	bData[len(bData)-1] = 'x'
	szPath := filepath.Join(t.TempDir(), "fw.bin")
	if errWrite := os.WriteFile(szPath, bData, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errStart := pSession.XmodemSend(szPath, true); errStart != nil {
		t.Fatal(errStart)
	}
	if errBusy := pSession.XmodemSend(szPath, false); errBusy == nil {
		t.Fatal("전송 중에는 두 번째 전송을 받지 않아야 한다")
	}
	pSession.Input("typed") // 전송 중 입력은 장치로 가면 안 된다
	var bufGot bytes.Buffer
	if _, errRecv := xmodem.Receive(&deviceEnd{pLink: pLink}, &bufGot, xmodem.Options{}); errRecv != nil {
		t.Fatal(errRecv)
	}
	if !bytes.Equal(bufGot.Bytes(), bData) {
		t.Fatalf("장치가 받은 %d바이트가 다르다", bufGot.Len())
	}
	waitUntil(t, func() bool { return pSession.xm.Load() == nil }, time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "fw.bin") }, time.Second)
	// 전송이 끝난 뒤의 출력은 다시 화면으로 간다.
	pLink.chToApp <- []byte("prompt>")
	waitUntil(t, func() bool { return strings.Contains(rec.output(), "prompt>") }, time.Second)
}

func TestXmodemReceiveThroughSessionSavesFile(t *testing.T) {
	fastXmodem(t)
	pLink := newLinkTransport()
	pSession, _ := openZmodemSession(t, pLink, nil)
	bData := []byte(strings.Repeat("log line\r\n", 300))
	szPath := filepath.Join(t.TempDir(), "dump.txt")
	if errStart := pSession.XmodemReceive(szPath); errStart != nil {
		t.Fatal(errStart)
	}
	if _, errSend := xmodem.Send(&deviceEnd{pLink: pLink}, bytes.NewReader(bData), int64(len(bData)), xmodem.Options{}); errSend != nil {
		t.Fatal(errSend)
	}
	waitUntil(t, func() bool { return pSession.xm.Load() == nil }, 2*time.Second)
	waitUntil(t, func() bool {
		bGot, errRead := os.ReadFile(szPath)
		return errRead == nil && bytes.Equal(bGot, bData)
	}, 2*time.Second)
	if _, errStat := os.Stat(szPath + g_cszZmodemTempSuffix); !os.IsNotExist(errStat) {
		t.Fatal("받는 중 파일(.dtpart)이 남으면 안 된다")
	}
}

func TestXmodemCtrlCCancelsAndRemovesPartial(t *testing.T) {
	fastXmodem(t)
	pLink := newLinkTransport()
	pSession, _ := openZmodemSession(t, pLink, nil)
	szPath := filepath.Join(t.TempDir(), "never.bin")
	if errStart := pSession.XmodemReceive(szPath); errStart != nil {
		t.Fatal(errStart)
	}
	time.Sleep(50 * time.Millisecond)
	pSession.Input("\x03")
	waitUntil(t, func() bool { return pSession.xm.Load() == nil }, 2*time.Second)
	waitUntil(t, func() bool {
		_, errTemp := os.Stat(szPath + g_cszZmodemTempSuffix)
		_, errFinal := os.Stat(szPath)
		return os.IsNotExist(errTemp) && os.IsNotExist(errFinal)
	}, time.Second)
	// 장치 쪽에는 취소(CAN)가 갔다.
	var bSent []byte
	for bDone := false; !bDone; {
		select {
		case bPart := <-pLink.chToRemote:
			bSent = append(bSent, bPart...)
		default:
			bDone = true
		}
	}
	if !bytes.Contains(bSent, []byte{xmodem.CAN, xmodem.CAN}) {
		t.Fatalf("취소를 알리지 않았다: %X", bSent)
	}
}
