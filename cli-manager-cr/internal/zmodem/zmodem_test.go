package zmodem

import (
	"bytes"
	"crypto/rand"
	"errors"
	"hash/crc32"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ── 시험 도구 ──────────────────────────────────────────────────────────────

// chanPipe 는 한 방향의 바이트 흐름이다. 쓰기는 막히지 않고(넉넉한 버퍼), 닫으면 읽기가 EOF 다.
type chanPipe struct {
	chData   chan []byte
	bRest    []byte
	onceDone sync.Once
	chDone   chan struct{}
}

func newChanPipe() *chanPipe {
	return &chanPipe{chData: make(chan []byte, 100000), chDone: make(chan struct{})}
}

func (pPipe *chanPipe) write(bData []byte) error {
	select {
	case <-pPipe.chDone:
		return io.ErrClosedPipe
	default:
	}
	pPipe.chData <- append([]byte(nil), bData...)
	return nil
}

func (pPipe *chanPipe) read(bBuf []byte) (int, error) {
	if len(pPipe.bRest) == 0 {
		select {
		case bData := <-pPipe.chData:
			pPipe.bRest = bData
		case <-pPipe.chDone:
			select {
			case bData := <-pPipe.chData:
				pPipe.bRest = bData
			default:
				return 0, io.EOF
			}
		}
	}
	nCopy := copy(bBuf, pPipe.bRest)
	pPipe.bRest = pPipe.bRest[nCopy:]
	return nCopy, nil
}

func (pPipe *chanPipe) close() { pPipe.onceDone.Do(func() { close(pPipe.chDone) }) }

// endpoint 는 양방향 연결의 한쪽 끝이다 (Stream). fnTamper 가 있으면 보내는 바이트를 고칠 수 있다.
type endpoint struct {
	pIn      *chanPipe
	pOut     *chanPipe
	fnTamper func(bData []byte) []byte
}

func (pEnd *endpoint) Read(bBuf []byte) (int, error) { return pEnd.pIn.read(bBuf) }
func (pEnd *endpoint) Write(bData []byte) error {
	if pEnd.fnTamper != nil {
		bData = pEnd.fnTamper(append([]byte(nil), bData...))
	}
	return pEnd.pOut.write(bData)
}

// newLink 는 서로 이어진 두 끝(이쪽 앱, 저쪽 서버)을 만든다.
func newLink() (*endpoint, *endpoint) {
	pAtoB, pBtoA := newChanPipe(), newChanPipe()
	return &endpoint{pIn: pBtoA, pOut: pAtoB}, &endpoint{pIn: pAtoB, pOut: pBtoA}
}

// memSink 는 받은 파일을 메모리에 담는다.
type memSink struct {
	mtxData   sync.Mutex
	mapFiles  map[string][]byte
	mapInfo   map[string]FileInfo
	slAborted []string
	szSkip    string // 이 이름은 건너뛴다
	errAccept error
}

type memWriter struct {
	pSink  *memSink
	szName string
	bData  bytes.Buffer
}

func (pWriter *memWriter) Write(bData []byte) (int, error) { return pWriter.bData.Write(bData) }
func (pWriter *memWriter) Commit() error {
	pWriter.pSink.mtxData.Lock()
	defer pWriter.pSink.mtxData.Unlock()
	pWriter.pSink.mapFiles[pWriter.szName] = pWriter.bData.Bytes()
	return nil
}
func (pWriter *memWriter) Abort() {
	pWriter.pSink.mtxData.Lock()
	defer pWriter.pSink.mtxData.Unlock()
	pWriter.pSink.slAborted = append(pWriter.pSink.slAborted, pWriter.szName)
}

func newMemSink() *memSink {
	return &memSink{mapFiles: map[string][]byte{}, mapInfo: map[string]FileInfo{}}
}

func (pSink *memSink) Accept(info FileInfo) (FileWriter, error) {
	if pSink.errAccept != nil {
		return nil, pSink.errAccept
	}
	pSink.mtxData.Lock()
	pSink.mapInfo[info.Name] = info
	pSink.mtxData.Unlock()
	if info.Name == pSink.szSkip {
		return nil, nil
	}
	return &memWriter{pSink: pSink, szName: info.Name}, nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func memSource(szName string, bData []byte) FileSource {
	return FileSource{Name: szName, Size: int64(len(bData)), ModTime: time.Unix(1700000000, 0),
		Open: func() (io.ReadSeekCloser, error) { return nopCloser{bytes.NewReader(bData)}, nil }}
}

func randomBytes(t *testing.T, iSize int) []byte {
	t.Helper()
	bData := make([]byte, iSize)
	if _, errRand := rand.Read(bData); errRand != nil {
		t.Fatal(errRand)
	}
	return bData
}

// allByteValues 는 0~255 를 여러 번 되풀이한 데이터다 (이스케이프가 필요한 값이 모두 들어 있다).
func allByteValues(iRepeat int) []byte {
	bData := make([]byte, 0, 256*iRepeat)
	for iRound := 0; iRound < iRepeat; iRound++ {
		for iValue := 0; iValue < 256; iValue++ {
			bData = append(bData, byte(iValue))
		}
	}
	return bData
}

// appReceive 는 "앱" 쪽을 흉내 낸다: 출력에서 시작 표시를 찾으면 Receive 를 돌린다. 표시 앞의 화면 출력도 돌려준다.
func appReceive(t *testing.T, pApp *endpoint, sink Sink, opts Options) (Result, error, string) {
	t.Helper()
	var detector Detector
	var sbScreen bytes.Buffer
	bBuf := make([]byte, 4096)
	for {
		nRead, errRead := pApp.Read(bBuf)
		if errRead != nil {
			t.Fatalf("시작 표시를 찾기 전에 연결이 끝남: %v", errRead)
		}
		bBefore, kind, bEngine := detector.Feed(bBuf[:nRead])
		sbScreen.Write(bBefore)
		if kind == KindRemoteSends {
			result, errReceive := Receive(pApp, bEngine, sink, opts)
			return result, errReceive, sbScreen.String()
		}
	}
}

// remoteSz 는 서버의 sz 를 흉내 낸다: "rz\r" + ZRQINIT 을 보내고, ZRINIT 이 오면 파일을 보낸다.
func remoteSz(pRemote *endpoint, slFiles []FileSource, opts Options) (Result, error) {
	_ = pRemote.Write(append([]byte("rz\r"), encodeHexHeader(G_cbZrqinit, [4]byte{})...))
	var detector Detector
	bBuf := make([]byte, 4096)
	for {
		nRead, errRead := pRemote.Read(bBuf)
		if errRead != nil {
			return Result{}, errRead
		}
		if _, kind, bEngine := detector.Feed(bBuf[:nRead]); kind == KindRemoteAccepts {
			return Send(pRemote, bEngine, func() ([]FileSource, error) { return slFiles, nil }, opts)
		}
	}
}

// remoteRz 는 서버의 rz 를 흉내 낸다: ZRINIT 을 보내고 파일을 받는다. iExtraRinit 만큼 ZRINIT 을 더 보낸다
// (사용자가 파일을 고르는 동안 rz 가 되풀이해 보내는 것).
func remoteRz(pRemote *endpoint, sink Sink, iExtraRinit int) (Result, error) {
	pEngine := newEngine(pRemote, nil, Options{})
	pEngine.bKeep = false
	for iIndex := 0; iIndex < iExtraRinit; iIndex++ {
		_ = pEngine.sendRinit()
	}
	var result Result
	errRun := pEngine.receiveLoop(sink, &result)
	if errRun != nil && errRun != ErrRemoteCancelled {
		_ = pRemote.Write(AbortSequence())
	}
	return result, errRun
}

// ── 형식 ───────────────────────────────────────────────────────────────────

// CRC 표준 시험값: "123456789" 의 CRC-16/XMODEM 은 0x31C3 이다.
func TestCrc16KnownValue(t *testing.T) {
	if uiGot := crc16([]byte("123456789")); uiGot != 0x31C3 {
		t.Fatalf("crc16 = %#04x", uiGot)
	}
	if crc32.ChecksumIEEE([]byte("123456789")) != 0xCBF43926 {
		t.Fatal("crc32 시험값이 다름")
	}
}

// lrzsz 가 실제로 보내는 헤더와 바이트 단위로 같다: sz 의 ZRQINIT, rz 의 ZRINIT(능력 0x23), ZFIN.
func TestHexHeadersMatchLrzsz(t *testing.T) {
	if szGot := string(encodeHexHeader(G_cbZrqinit, [4]byte{})); szGot != "**\x18B00000000000000\r\x8a\x11" {
		t.Fatalf("ZRQINIT: %q", szGot)
	}
	if szGot := string(encodeHexHeader(G_cbZrinit, [4]byte{0, 0, 0, 0x23})); szGot != "**\x18B0100000023be50\r\x8a\x11" {
		t.Fatalf("ZRINIT: %q", szGot)
	}
	if szGot := string(encodeHexHeader(g_cbZfin, [4]byte{})); szGot != "**\x18B0800000000022d\r\x8a" {
		t.Fatalf("ZFIN(XON 없음): %q", szGot)
	}
}

// 이스케이프: 제어 문자·0x7f·0xff·ZDLE 은 두 바이트로 바뀌고, 읽으면 원래 값이 된다.
func TestEscapeRoundTrip(t *testing.T) {
	bAll := allByteValues(1)
	bEscaped := appendEscaped(nil, bAll)
	for _, bOne := range bEscaped {
		if bOne != g_cbZdle && bOne < 0x20 || bOne == 0x7f || bOne == 0xff || bOne >= 0x80 && bOne < 0xa0 {
			t.Fatalf("이스케이프한 결과에 제어 문자가 그대로 있음: %#02x", bOne)
		}
	}
	pApp, pRemote := newLink()
	_ = pRemote.Write(bEscaped)
	pEngine := newEngine(pApp, nil, Options{})
	bGot, errRead := pEngine.readEscapedN(256)
	if errRead != nil || !bytes.Equal(bGot, bAll) {
		t.Fatalf("되돌린 값이 다름: %v", errRead)
	}
}

// 헤더 세 형식(16진·이진 CRC-16·이진 CRC-32)을 만들고 읽으면 같은 값이다. 앞에 섞인 잡음은 건너뛴다.
func TestHeaderRoundTripAllFormats(t *testing.T) {
	arrData := [4]byte{0x18, 0x11, 0xff, 0x0d} // 이스케이프가 필요한 값들
	for szName, bFrame := range map[string][]byte{
		"hex":   encodeHexHeader(g_cbZrpos, arrData),
		"bin16": encodeBinHeader(g_cbZrpos, arrData, false),
		"bin32": encodeBinHeader(g_cbZrpos, arrData, true),
	} {
		pApp, pRemote := newLink()
		_ = pRemote.Write(append([]byte("noise\x11 * *x "), bFrame...))
		pEngine := newEngine(pApp, nil, Options{})
		hdr, errRead := pEngine.readHeader()
		if errRead != nil || hdr.bType != g_cbZrpos || hdr.arrData != arrData {
			t.Fatalf("%s: %+v %v", szName, hdr, errRead)
		}
	}
}

// ── 시작 감지 ──────────────────────────────────────────────────────────────

func TestDetectorFindsStart(t *testing.T) {
	bZrq := encodeHexHeader(G_cbZrqinit, [4]byte{})
	var detector Detector
	bBefore, kind, bEngine := detector.Feed(append([]byte("$ sz a.txt\r\nrz\r"), bZrq...))
	if kind != KindRemoteSends || string(bBefore) != "$ sz a.txt\r\nrz\r" || !bytes.Equal(bEngine, bZrq) {
		t.Fatalf("sz 시작: %v %q %q", kind, bBefore, bEngine)
	}
	_, kind, _ = detector.Feed(encodeHexHeader(G_cbZrinit, [4]byte{0, 0, 0, 0x23}))
	if kind != KindRemoteAccepts {
		t.Fatalf("rz 시작: %v", kind)
	}
}

// 시작 표시가 출력 묶음 경계 어디에서 잘려도 찾는다. 별이 여러 개 이어져도 찾는다.
func TestDetectorAcrossChunksAndStars(t *testing.T) {
	bAll := append([]byte("ab***"), encodeHexHeader(G_cbZrqinit, [4]byte{})...)
	for iCut := 1; iCut < len(bAll); iCut++ {
		var detector Detector
		_, kindFirst, _ := detector.Feed(bAll[:iCut])
		kind := kindFirst
		var bEngine []byte
		if kind == KindNone {
			_, kind, bEngine = detector.Feed(bAll[iCut:])
		}
		if kind != KindRemoteSends {
			t.Fatalf("자른 자리 %d 에서 못 찾음", iCut)
		}
		if kindFirst == KindNone && !bytes.HasPrefix(bEngine, []byte("**\x18B00")) {
			t.Fatalf("자른 자리 %d: 엔진에 넘길 바이트가 표시로 시작해야 함: %q", iCut, bEngine)
		}
	}
}

func TestDetectorIgnoresOrdinaryOutput(t *testing.T) {
	var detector Detector
	for _, szText := range []string{"ls *.txt\r\n", "** bold **\r\n", "**\x18B02 nope", "a**\x18Bxyz", "**\x18"} {
		if bBefore, kind, _ := detector.Feed([]byte(szText)); kind != KindNone || string(bBefore) != szText {
			t.Fatalf("보통 출력을 표시로 봄: %q", szText)
		}
		detector.Reset()
	}
}

// ── 받기 (서버에서 sz) ─────────────────────────────────────────────────────

// 여러 파일을 받는다: 빈 파일, 모든 바이트 값, 큰 무작위 파일. 내용·크기·수정 시각이 맞고 화면 출력은 표시 앞부분뿐이다.
func TestReceiveFiles(t *testing.T) {
	pApp, pRemote := newLink()
	bBig := randomBytes(t, 700*1024+123)
	slFiles := []FileSource{memSource("empty.bin", nil), memSource("all.bin", allByteValues(40)), memSource("big.bin", bBig)}
	chRemote := make(chan error, 1)
	go func() {
		_, errSend := remoteSz(pRemote, slFiles, Options{})
		chRemote <- errSend
	}()
	sink := newMemSink()
	var slProgress []Progress
	result, errReceive, szScreen := appReceive(t, pApp, sink, Options{Progress: func(ev Progress) {
		if ev.State != G_ciStateData {
			slProgress = append(slProgress, ev)
		}
	}})
	if errReceive != nil {
		t.Fatal(errReceive)
	}
	if errRemote := <-chRemote; errRemote != nil {
		t.Fatalf("보내는 쪽: %v", errRemote)
	}
	if szScreen != "rz\r" {
		t.Fatalf("화면 출력: %q", szScreen)
	}
	if result.Files != 3 || result.Bytes != int64(256*40+len(bBig)) {
		t.Fatalf("결과: %+v", result)
	}
	for _, file := range slFiles {
		reader, _ := file.Open()
		bWant, _ := io.ReadAll(reader)
		if !bytes.Equal(sink.mapFiles[file.Name], bWant) {
			t.Fatalf("%s 내용이 다름 (%d / %d 바이트)", file.Name, len(sink.mapFiles[file.Name]), len(bWant))
		}
		info := sink.mapInfo[file.Name]
		if info.Size != file.Size || info.ModTime.Unix() != 1700000000 {
			t.Fatalf("%s 정보: %+v", file.Name, info)
		}
	}
	if len(slProgress) != 6 || slProgress[0].State != G_ciStateStart || slProgress[5].State != G_ciStateDone || slProgress[5].Done != int64(len(bBig)) {
		t.Fatalf("진행 알림(파일마다 시작·완료): %+v", slProgress)
	}
	if len(result.Leftover) != 0 {
		t.Fatalf("남은 바이트가 없어야 함(\"OO\" 까지 먹음): %q", result.Leftover)
	}
}

// 전송이 끝난 뒤 이어지는 셸 출력은 남은 바이트로 돌아온다.
func TestReceiveReturnsTrailingShellOutput(t *testing.T) {
	pApp, pRemote := newLink()
	go func() {
		_, _ = remoteSz(pRemote, []FileSource{memSource("a", []byte("hello"))}, Options{})
		_ = pRemote.Write([]byte("$ prompt"))
	}()
	// 보내는 쪽이 "OO" 와 프롬프트까지 다 쓴 뒤 읽기 시작하도록, 받는 쪽의 마지막 ZFIN 을 늦게 보낸다.
	pApp.fnTamper = func(bData []byte) []byte {
		if bytes.Contains(bData, []byte("**\x18B08")) {
			time.Sleep(150 * time.Millisecond)
		}
		return bData
	}
	result, errReceive, _ := appReceive(t, pApp, newMemSink(), Options{})
	if errReceive != nil {
		t.Fatal(errReceive)
	}
	// ZFIN 을 보낸 뒤에는 더 읽지 않으므로, 그때 이미 버퍼에 있던 것만 남는다(없을 수도 있다). "OO" 는 남지 않는다.
	if bytes.Contains(result.Leftover, []byte("OO")) {
		t.Fatalf("\"OO\" 는 먹어야 함: %q", result.Leftover)
	}
}

// 받는 쪽이 건너뛴 파일은 저장하지 않고 다음 파일을 받는다.
func TestReceiveSkipsFile(t *testing.T) {
	pApp, pRemote := newLink()
	chRemote := make(chan Result, 1)
	go func() {
		result, _ := remoteSz(pRemote, []FileSource{memSource("skip.me", randomBytes(t, 5000)), memSource("keep.me", []byte("kept"))}, Options{})
		chRemote <- result
	}()
	sink := newMemSink()
	sink.szSkip = "skip.me"
	result, errReceive, _ := appReceive(t, pApp, sink, Options{})
	if errReceive != nil {
		t.Fatal(errReceive)
	}
	if result.Files != 1 || result.Skipped != 1 || string(sink.mapFiles["keep.me"]) != "kept" || sink.mapFiles["skip.me"] != nil {
		t.Fatalf("결과: %+v %v", result, sink.mapFiles)
	}
	if resultRemote := <-chRemote; resultRemote.Skipped != 1 || resultRemote.Files != 1 {
		t.Fatalf("보내는 쪽 결과: %+v", resultRemote)
	}
}

// 전송 중 바이트 하나가 깨지면 CRC 로 알아채고 그 위치부터 다시 받아 내용이 온전하다.
func TestReceiveRecoversFromCorruption(t *testing.T) {
	pApp, pRemote := newLink()
	bData := randomBytes(t, 300*1024)
	var iWrites atomic.Int32
	pRemote.fnTamper = func(bFrame []byte) []byte {
		// 데이터 조각 가운데 하나(40번째 쓰기)의 한 바이트를 뒤집는다. 이스케이프 구조는 건드리지 않게 보통 글자만 고른다.
		if iWrites.Add(1) == 40 {
			for iIndex := 10; iIndex < len(bFrame)-10; iIndex++ {
				if bFrame[iIndex] >= 'a' && bFrame[iIndex] <= 'y' && bFrame[iIndex-1] != g_cbZdle {
					bFrame[iIndex]++
					break
				}
			}
		}
		return bFrame
	}
	go func() { _, _ = remoteSz(pRemote, []FileSource{memSource("x.bin", bData)}, Options{}) }()
	sink := newMemSink()
	result, errReceive, _ := appReceive(t, pApp, sink, Options{})
	if errReceive != nil {
		t.Fatal(errReceive)
	}
	if !bytes.Equal(sink.mapFiles["x.bin"], bData) {
		t.Fatalf("깨진 뒤 다시 받은 내용이 달라짐 (%d 바이트)", len(sink.mapFiles["x.bin"]))
	}
	if result.Files != 1 {
		t.Fatalf("결과: %+v", result)
	}
}

// 사용자가 취소하면 받던 파일을 버리고(Abort) 상대에게 중단을 알린다.
func TestReceiveCancelledByUser(t *testing.T) {
	pApp, pRemote := newLink()
	var bCancel atomic.Bool
	chRemote := make(chan error, 1)
	go func() {
		_, errSend := remoteSz(pRemote, []FileSource{memSource("huge.bin", randomBytes(t, 4*1024*1024))}, Options{})
		chRemote <- errSend
	}()
	sink := newMemSink()
	_, errReceive, _ := appReceive(t, pApp, sink, Options{
		Cancelled: bCancel.Load,
		Progress: func(ev Progress) {
			if ev.Done > 200*1024 {
				bCancel.Store(true)
			}
		},
	})
	if !errors.Is(errReceive, ErrCancelled) {
		t.Fatalf("취소 오류여야 함: %v", errReceive)
	}
	if len(sink.slAborted) != 1 || len(sink.mapFiles) != 0 {
		t.Fatalf("받던 파일은 버려야 함: %v %d", sink.slAborted, len(sink.mapFiles))
	}
	select {
	case errRemote := <-chRemote:
		if !errors.Is(errRemote, ErrRemoteCancelled) {
			t.Fatalf("보내는 쪽은 상대 취소로 끝나야 함: %v", errRemote)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("보내는 쪽이 중단 알림을 받지 못함")
	}
}

// 상대가 중단(CAN 여러 개)하면 받던 파일을 버리고 끝난다.
func TestReceiveRemoteAborts(t *testing.T) {
	pApp, pRemote := newLink()
	go func() {
		_ = pRemote.Write(encodeHexHeader(G_cbZrqinit, [4]byte{}))
		time.Sleep(50 * time.Millisecond)
		_ = pRemote.Write(AbortSequence())
	}()
	_, errReceive, _ := appReceive(t, pApp, newMemSink(), Options{})
	if !errors.Is(errReceive, ErrRemoteCancelled) {
		t.Fatalf("상대 취소 오류여야 함: %v", errReceive)
	}
}

// 시작 표시처럼 보였지만 헤더가 아니면(CRC 불일치) 읽은 바이트를 모두 되돌린다.
func TestReceiveNotZmodemReturnsBytes(t *testing.T) {
	pApp, pRemote := newLink()
	szFake := "**\x18B00deadbeefdead rest of line"
	_ = pRemote.Write([]byte(szFake))
	var detector Detector
	bBuf := make([]byte, 256)
	nRead, _ := pApp.Read(bBuf)
	_, kind, bEngine := detector.Feed(bBuf[:nRead])
	if kind != KindRemoteSends {
		t.Fatal("표시 모양은 맞아야 함")
	}
	result, errReceive := Receive(pApp, bEngine, newMemSink(), Options{})
	if !errors.Is(errReceive, ErrNotZmodem) || string(result.Leftover) != szFake {
		t.Fatalf("되돌린 바이트: %q %v", result.Leftover, errReceive)
	}
}

// 연결이 끊기면 연결 오류(*StreamError)로 끝난다.
func TestReceiveStreamError(t *testing.T) {
	pApp, pRemote := newLink()
	go func() {
		_ = pRemote.Write(encodeHexHeader(G_cbZrqinit, [4]byte{}))
		time.Sleep(30 * time.Millisecond)
		pApp.pIn.close()
	}()
	_, errReceive, _ := appReceive(t, pApp, newMemSink(), Options{})
	var errStream *StreamError
	if !errors.As(errReceive, &errStream) {
		t.Fatalf("연결 오류여야 함: %v", errReceive)
	}
}

// ── 보내기 (서버에서 rz) ───────────────────────────────────────────────────

func appSend(t *testing.T, pApp *endpoint, slFiles []FileSource, opts Options) (Result, error) {
	t.Helper()
	var detector Detector
	bBuf := make([]byte, 4096)
	for {
		nRead, errRead := pApp.Read(bBuf)
		if errRead != nil {
			t.Fatalf("시작 표시를 찾기 전에 연결이 끝남: %v", errRead)
		}
		if _, kind, bEngine := detector.Feed(bBuf[:nRead]); kind == KindRemoteAccepts {
			return Send(pApp, bEngine, func() ([]FileSource, error) { return slFiles, nil }, opts)
		}
	}
}

// 여러 파일을 보낸다. 파일을 고르는 동안 밀린 ZRINIT 이 여러 개 있어도 한 번씩만 보낸다.
func TestSendFiles(t *testing.T) {
	pApp, pRemote := newLink()
	bBig := randomBytes(t, 600*1024+7)
	slFiles := []FileSource{memSource("a.txt", []byte("hello zmodem")), memSource("empty", nil), memSource("big.bin", bBig)}
	sink := newMemSink()
	chRemote := make(chan Result, 1)
	go func() {
		result, _ := remoteRz(pRemote, sink, 3)
		chRemote <- result
	}()
	result, errSend := appSend(t, pApp, slFiles, Options{})
	if errSend != nil {
		t.Fatal(errSend)
	}
	resultRemote := <-chRemote
	if result.Files != 3 || resultRemote.Files != 3 {
		t.Fatalf("결과: 보낸 쪽 %+v 받은 쪽 %+v", result, resultRemote)
	}
	if string(sink.mapFiles["a.txt"]) != "hello zmodem" || len(sink.mapFiles["empty"]) != 0 || !bytes.Equal(sink.mapFiles["big.bin"], bBig) {
		t.Fatal("받은 내용이 다름")
	}
	if sink.mapInfo["big.bin"].Size != int64(len(bBig)) {
		t.Fatalf("파일 정보: %+v", sink.mapInfo["big.bin"])
	}
}

// 상대가 CRC-32 를 못 쓴다고 하면(능력 표시 없음) CRC-16 으로 보낸다.
func TestSendUsesCrc16WhenRemoteLacksCrc32(t *testing.T) {
	pApp, pRemote := newLink()
	bData := allByteValues(20)
	sink := newMemSink()
	var bSawBin32 atomic.Bool
	pApp.fnTamper = func(bFrame []byte) []byte {
		if bytes.Contains(bFrame, []byte{g_cbZpad, g_cbZdle, g_cbZbin32}) {
			bSawBin32.Store(true)
		}
		return bFrame
	}
	go func() {
		pEngine := newEngine(pRemote, nil, Options{})
		pEngine.bKeep = false
		_ = pEngine.sendHex(G_cbZrinit, [4]byte{0, 0, 0, g_cbCanFdx | g_cbCanOvIo}) // CRC-32 없음
		var result Result
		for {
			hdr, errHeader := pEngine.readHeader()
			if errHeader != nil {
				return
			}
			switch hdr.bType {
			case g_cbZfile:
				bInfo, _, _, _ := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32)
				_ = pEngine.receiveFile(parseFileInfo(bInfo), sink, &result)
			case g_cbZfin:
				_ = pEngine.sendHex(g_cbZfin, [4]byte{})
				return
			}
		}
	}()
	if _, errSend := appSend(t, pApp, []FileSource{memSource("c16.bin", bData)}, Options{}); errSend != nil {
		t.Fatal(errSend)
	}
	if bSawBin32.Load() {
		t.Fatal("CRC-32 헤더를 보내면 안 됨")
	}
	if !bytes.Equal(sink.mapFiles["c16.bin"], bData) {
		t.Fatal("받은 내용이 다름")
	}
}

// 파일을 고르지 않으면(빈 목록) 취소로 끝나고 상대에게 중단을 알린다.
func TestSendNoFilesCancels(t *testing.T) {
	pApp, pRemote := newLink()
	chRemote := make(chan error, 1)
	go func() {
		_, errRun := remoteRz(pRemote, newMemSink(), 0)
		chRemote <- errRun
	}()
	_, errSend := appSend(t, pApp, nil, Options{})
	if !errors.Is(errSend, ErrCancelled) {
		t.Fatalf("취소 오류여야 함: %v", errSend)
	}
	select {
	case errRemote := <-chRemote:
		if !errors.Is(errRemote, ErrRemoteCancelled) {
			t.Fatalf("받는 쪽은 상대 취소로 끝나야 함: %v", errRemote)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("받는 쪽이 중단 알림을 받지 못함")
	}
}

// 보내는 중에 깨져도 상대의 ZRPOS 를 받고 그 위치부터 다시 보내 내용이 온전하다.
func TestSendRecoversFromCorruption(t *testing.T) {
	pApp, pRemote := newLink()
	bData := randomBytes(t, 400*1024)
	var iWrites atomic.Int32
	pApp.fnTamper = func(bFrame []byte) []byte {
		if iWrites.Add(1) == 30 {
			for iIndex := 10; iIndex < len(bFrame)-10; iIndex++ {
				if bFrame[iIndex] >= 'a' && bFrame[iIndex] <= 'y' && bFrame[iIndex-1] != g_cbZdle {
					bFrame[iIndex]++
					break
				}
			}
		}
		return bFrame
	}
	sink := newMemSink()
	go func() { _, _ = remoteRz(pRemote, sink, 0) }()
	if _, errSend := appSend(t, pApp, []FileSource{memSource("y.bin", bData)}, Options{}); errSend != nil {
		t.Fatal(errSend)
	}
	if !bytes.Equal(sink.mapFiles["y.bin"], bData) {
		t.Fatalf("다시 보낸 뒤 내용이 달라짐 (%d 바이트)", len(sink.mapFiles["y.bin"]))
	}
}

// 받는 쪽이 저장에 실패하면(Accept 오류) 전송을 중단한다.
func TestReceiveAcceptErrorAborts(t *testing.T) {
	pApp, pRemote := newLink()
	chRemote := make(chan error, 1)
	go func() {
		_, errSend := remoteSz(pRemote, []FileSource{memSource("a", []byte("x"))}, Options{})
		chRemote <- errSend
	}()
	sink := newMemSink()
	sink.errAccept = errors.New("disk full")
	_, errReceive, _ := appReceive(t, pApp, sink, Options{})
	if errReceive == nil || errReceive.Error() != "disk full" {
		t.Fatalf("저장 오류가 그대로 나와야 함: %v", errReceive)
	}
	select {
	case errRemote := <-chRemote:
		if !errors.Is(errRemote, ErrRemoteCancelled) {
			t.Fatalf("보내는 쪽은 상대 취소로 끝나야 함: %v", errRemote)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("보내는 쪽이 중단 알림을 받지 못함")
	}
}
