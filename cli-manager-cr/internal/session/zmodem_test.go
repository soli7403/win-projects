package session

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dariterm/internal/zmodem"
)

// ZMODEM 을 세션에 이은 부분의 시험 (zmodem.md, D114). 서버 쪽은 같은 zmodem 패키지로 흉내 낸다.

// lrzsz 가 보내는 시작 헤더 (zmodem 패키지 시험에서 바이트 단위로 확인한 값).
const (
	g_cszTestZrqinit = "**\x18B00000000000000\r\x8a\x11" // sz 가 처음 보낸다
)

// linkTransport 는 앱 쪽 연결(Transport)이고, 반대쪽 끝(remote)은 서버 흉내가 쓴다.
type linkTransport struct {
	chToApp    chan []byte
	chToRemote chan []byte
	chClosed   chan struct{}
	onceClose  sync.Once
	bRest      []byte
}

func newLinkTransport() *linkTransport {
	return &linkTransport{chToApp: make(chan []byte, 100000), chToRemote: make(chan []byte, 100000), chClosed: make(chan struct{})}
}

func (pLink *linkTransport) Read(bBuf []byte) (int, error) {
	select {
	case bData := <-pLink.chToApp:
		return copy(bBuf, bData), nil // 시험에서는 한 번에 32KB 이하로만 보낸다
	case <-pLink.chClosed:
		return 0, io.EOF
	}
}

func (pLink *linkTransport) Write(bData []byte) error {
	pLink.chToRemote <- append([]byte(nil), bData...)
	return nil
}
func (pLink *linkTransport) Resize(iCols int, iRows int) error { return nil }
func (pLink *linkTransport) Close() error {
	pLink.onceClose.Do(func() { close(pLink.chClosed) })
	return nil
}

// remoteEnd 는 서버 쪽 끝이다 (zmodem.Stream).
type remoteEnd struct {
	pLink *linkTransport
	bRest []byte
}

func (pEnd *remoteEnd) Read(bBuf []byte) (int, error) {
	if len(pEnd.bRest) == 0 {
		select {
		case bData := <-pEnd.pLink.chToRemote:
			pEnd.bRest = bData
		case <-pEnd.pLink.chClosed:
			return 0, io.EOF
		}
	}
	nCopy := copy(bBuf, pEnd.bRest)
	pEnd.bRest = pEnd.bRest[nCopy:]
	return nCopy, nil
}

func (pEnd *remoteEnd) Write(bData []byte) error {
	for len(bData) > 0 {
		nPart := len(bData)
		if nPart > 8192 {
			nPart = 8192
		}
		pEnd.pLink.chToApp <- append([]byte(nil), bData[:nPart]...)
		bData = bData[nPart:]
	}
	return nil
}

type byteSource struct{ *bytes.Reader }

func (byteSource) Close() error { return nil }

func testSource(szName string, bData []byte) zmodem.FileSource {
	return zmodem.FileSource{Name: szName, Size: int64(len(bData)), ModTime: time.Unix(1700000000, 0),
		Open: func() (io.ReadSeekCloser, error) { return byteSource{bytes.NewReader(bData)}, nil }}
}

// remoteSz 는 서버에서 sz 를 실행한 것처럼 군다: 시작 헤더를 보내고 앱의 ZRINIT 을 기다려 파일을 보낸다.
func remoteSz(pLink *linkTransport, slFiles []zmodem.FileSource) error {
	pEnd := &remoteEnd{pLink: pLink}
	_ = pEnd.Write([]byte("rz\r" + g_cszTestZrqinit))
	var detector zmodem.Detector
	bBuf := make([]byte, 4096)
	for {
		nRead, errRead := pEnd.Read(bBuf)
		if errRead != nil {
			return errRead
		}
		if _, kind, bEngine := detector.Feed(bBuf[:nRead]); kind == zmodem.KindRemoteAccepts {
			_, errSend := zmodem.Send(pEnd, bEngine, func() ([]zmodem.FileSource, error) { return slFiles, nil }, zmodem.Options{})
			return errSend
		}
	}
}

// memFiles 는 서버 쪽(rz 흉내)이 받은 파일을 메모리에 담는다.
type memFiles struct {
	mtxData  sync.Mutex
	mapFiles map[string][]byte
}

type memFile struct {
	pOwner *memFiles
	szName string
	bData  bytes.Buffer
}

func (pFile *memFile) Write(bData []byte) (int, error) { return pFile.bData.Write(bData) }
func (pFile *memFile) Commit() error {
	pFile.pOwner.mtxData.Lock()
	defer pFile.pOwner.mtxData.Unlock()
	pFile.pOwner.mapFiles[pFile.szName] = pFile.bData.Bytes()
	return nil
}
func (pFile *memFile) Abort() {}
func (pFiles *memFiles) Accept(info zmodem.FileInfo) (zmodem.FileWriter, error) {
	return &memFile{pOwner: pFiles, szName: info.Name}, nil
}
func (pFiles *memFiles) get(szName string) []byte {
	pFiles.mtxData.Lock()
	defer pFiles.mtxData.Unlock()
	return pFiles.mapFiles[szName]
}

// remoteRz 는 서버에서 rz 를 실행한 것처럼 군다: ZRINIT 을 보내고 파일을 받는다.
// (받는 엔진에 "sz 의 시작 헤더를 받았다"고 넣어 주면 곧바로 ZRINIT 을 보내고 기다린다 — rz 와 같은 동작이다.)
func remoteRz(pLink *linkTransport, pFiles *memFiles) error {
	_, errReceive := zmodem.Receive(&remoteEnd{pLink: pLink}, []byte(g_cszTestZrqinit), pFiles, zmodem.Options{})
	return errReceive
}

func openZmodemSession(t *testing.T, pLink *linkTransport, pHooks *ZmodemHooks) (*Session, *eventRecorder) {
	t.Helper()
	rec := &eventRecorder{}
	pManager := NewManager(rec.emit)
	cfg := sshConfig(func(func(string)) (Transport, error) { return pLink, nil }, false)
	cfg.Zmodem = pHooks
	info, errOpen := pManager.Open(cfg)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	t.Cleanup(func() { pManager.Close(info.M_szId) })
	pSession, _ := pManager.Get(info.M_szId)
	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
	// 시험에서는 프런트가 없으므로 출력 확인(ack)을 대신해 준다 (출력 흐름 제어에 막히지 않게).
	chStop := make(chan struct{})
	t.Cleanup(func() { close(chStop) })
	go func() {
		for {
			select {
			case <-chStop:
				return
			case <-time.After(5 * time.Millisecond):
				pSession.AckOutput(1000)
			}
		}
	}()
	return pSession, rec
}

func (pRec *eventRecorder) screen() string {
	pRec.mtxData.Lock()
	defer pRec.mtxData.Unlock()
	return strings.Join(pRec.slData, "")
}

func testRandom(t *testing.T, iSize int) []byte {
	t.Helper()
	bData := make([]byte, iSize)
	_, _ = rand.Read(bData)
	return bData
}

// 서버에서 sz 를 실행하면 파일이 받는 폴더에 저장된다. 서버가 준 이름의 경로·금지 문자는 다듬는다.
// 전송 데이터는 화면에 나오지 않고, 전송이 끝난 뒤의 셸 출력은 다시 화면에 나온다.
func TestZmodemDownloadSavesFile(t *testing.T) {
	szDir := filepath.Join(t.TempDir(), "downloads")
	pLink := newLinkTransport()
	_, rec := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return szDir }})
	bData := testRandom(t, 200*1024)
	chRemote := make(chan error, 1)
	go func() {
		errSend := remoteSz(pLink, []zmodem.FileSource{testSource("../../etc/re:port?.bin", bData)})
		time.Sleep(50 * time.Millisecond)
		pLink.chToApp <- []byte("$ after\r\n")
		chRemote <- errSend
	}()
	szSaved := filepath.Join(szDir, "re_port_.bin")
	waitUntil(t, func() bool {
		bGot, errRead := os.ReadFile(szSaved)
		return errRead == nil && bytes.Equal(bGot, bData)
	}, 10*time.Second)
	if errRemote := <-chRemote; errRemote != nil {
		t.Fatalf("보내는 쪽: %v", errRemote)
	}
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "$ after") }, 3*time.Second)
	szScreen := rec.screen()
	if !strings.Contains(szScreen, "rz\r") {
		t.Fatalf("시작 표시 앞의 출력은 화면에 나와야 함: %q", szScreen)
	}
	if strings.Contains(szScreen, "**\x18B") || len(szScreen) > 4000 {
		t.Fatalf("전송 데이터가 화면에 나오면 안 됨 (%d 글자)", len(szScreen))
	}
	if slLeft, _ := filepath.Glob(filepath.Join(szDir, "*"+g_cszZmodemTempSuffix)); len(slLeft) != 0 {
		t.Fatalf("임시 파일이 남으면 안 됨: %v", slLeft)
	}
	pInfo, _ := os.Stat(szSaved)
	if pInfo.ModTime().Unix() != 1700000000 {
		t.Fatalf("수정 시각을 서버 것으로 맞춰야 함: %v", pInfo.ModTime())
	}
}

// 같은 이름이 이미 있으면 덮어쓰지 않고 번호를 붙여 저장한다.
func TestZmodemDownloadKeepsExistingFile(t *testing.T) {
	szDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(szDir, "a.txt"), []byte("old"), 0o644)
	pLink := newLinkTransport()
	openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return szDir }})
	go func() { _ = remoteSz(pLink, []zmodem.FileSource{testSource("a.txt", []byte("new"))}) }()
	waitUntil(t, func() bool {
		bGot, _ := os.ReadFile(filepath.Join(szDir, "a (2).txt"))
		return string(bGot) == "new"
	}, 5*time.Second)
	if bOld, _ := os.ReadFile(filepath.Join(szDir, "a.txt")); string(bOld) != "old" {
		t.Fatalf("기존 파일을 건드리면 안 됨: %q", string(bOld))
	}
}

// 서버에서 rz 를 실행하면 고른 파일을 보낸다. 폴더는 빼고 보낸다.
func TestZmodemUploadSendsPickedFiles(t *testing.T) {
	szDir := t.TempDir()
	bData := testRandom(t, 150*1024)
	szFile := filepath.Join(szDir, "upload me.bin")
	_ = os.WriteFile(szFile, bData, 0o644)
	pLink := newLinkTransport()
	iPicked := 0
	_, rec := openZmodemSession(t, pLink, &ZmodemHooks{PickFiles: func() ([]string, error) {
		iPicked++
		return []string{szDir, szFile}, nil
	}})
	pFiles := &memFiles{mapFiles: map[string][]byte{}}
	chRemote := make(chan error, 1)
	go func() { chRemote <- remoteRz(pLink, pFiles) }()
	select {
	case errRemote := <-chRemote:
		if errRemote != nil {
			t.Fatalf("받는 쪽: %v", errRemote)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("보내기가 끝나지 않음")
	}
	if !bytes.Equal(pFiles.get("upload me.bin"), bData) {
		t.Fatalf("서버가 받은 내용이 다름 (%d 바이트)", len(pFiles.get("upload me.bin")))
	}
	if iPicked != 1 {
		t.Fatalf("파일 고르기는 한 번만: %d", iPicked)
	}
	_ = rec
}

// 전송 중에는 터미널 입력을 서버로 보내지 않는다. Ctrl+C 를 누르면 전송을 취소하고 받던 파일을 지운다.
// 취소한 뒤에는 입력이 다시 서버로 간다.
func TestZmodemCancelWithCtrlC(t *testing.T) {
	szDir := t.TempDir()
	pLink := newLinkTransport()
	pSession, _ := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return szDir }})
	// 서버 흉내: 시작 헤더를 보낸 뒤, 앱의 ZRINIT 에 ZFILE 로 답하고 데이터를 조금 보낸 채 멈춘다(끝내지 않는다).
	pEnd := &remoteEnd{pLink: pLink}
	var bSawAbort atomic.Bool
	var bSawTyped atomic.Bool
	var slRemoteGot bytes.Buffer
	var mtxGot sync.Mutex
	chHold := make(chan struct{})
	go func() {
		_ = pEnd.Write([]byte(g_cszTestZrqinit))
		// 큰 파일을 보내되, Open 한 reader 가 중간에서 멈추게 해 전송을 붙잡아 둔다.
		src := zmodem.FileSource{Name: "slow.bin", Size: 10 * 1024 * 1024, ModTime: time.Now(),
			Open: func() (io.ReadSeekCloser, error) { return &holdReader{chHold: chHold, iBefore: 64 * 1024}, nil }}
		var detector zmodem.Detector
		bBuf := make([]byte, 4096)
		for {
			nRead, errRead := pEnd.Read(bBuf)
			if errRead != nil {
				return
			}
			if _, kind, bEngine := detector.Feed(bBuf[:nRead]); kind == zmodem.KindRemoteAccepts {
				go func() {
					_, _ = zmodem.Send(&tapEnd{pEnd: pEnd, fnSeen: func(bData []byte) {
						mtxGot.Lock()
						slRemoteGot.Write(bData)
						bAll := slRemoteGot.Bytes()
						if bytes.Contains(bAll, []byte{0x18, 0x18, 0x18, 0x18, 0x18}) {
							bSawAbort.Store(true)
						}
						if bytes.Contains(bAll, []byte("typed-during")) {
							bSawTyped.Store(true)
						}
						mtxGot.Unlock()
					}}, bEngine, func() ([]zmodem.FileSource, error) { return []zmodem.FileSource{src}, nil }, zmodem.Options{})
				}()
				return
			}
		}
	}()
	szTemp := filepath.Join(szDir, "slow.bin"+g_cszZmodemTempSuffix)
	waitUntil(t, func() bool {
		pInfo, errStat := os.Stat(szTemp)
		return errStat == nil && pInfo.Size() > 0
	}, 5*time.Second)
	pSession.Input("typed-during")
	pSession.Input("\x03")
	close(chHold) // 붙잡아 둔 보내는 쪽을 풀어 준다: 다음 데이터가 오면 앱이 취소를 알아챈다
	waitUntil(t, func() bool {
		_, errStat := os.Stat(szTemp)
		return os.IsNotExist(errStat) && !pSession.zActive.Load()
	}, 5*time.Second)
	waitUntil(t, bSawAbort.Load, 3*time.Second)
	if bSawTyped.Load() {
		t.Fatal("전송 중의 입력이 서버로 가면 안 됨")
	}
	if slFiles, _ := os.ReadDir(szDir); len(slFiles) != 0 {
		t.Fatalf("취소하면 받던 파일이 남지 않아야 함: %d개", len(slFiles))
	}
	// 취소 뒤에는 입력이 다시 서버로 간다.
	for len(pLink.chToRemote) > 0 {
		<-pLink.chToRemote
	}
	pSession.Input("after")
	select {
	case bGot := <-pLink.chToRemote:
		if string(bGot) != "after" {
			t.Fatalf("취소 뒤 입력: %q", string(bGot))
		}
	case <-time.After(time.Second):
		t.Fatal("취소 뒤에는 입력이 서버로 가야 함")
	}
}

// holdReader 는 iBefore 바이트까지는 바로 주고, 그 뒤로는 chHold 가 닫힐 때까지 기다리는 파일이다.
type holdReader struct {
	chHold  chan struct{}
	iBefore int
	iPos    int
}

func (pReader *holdReader) Read(bBuf []byte) (int, error) {
	if pReader.iPos >= pReader.iBefore {
		<-pReader.chHold
	}
	for iIndex := range bBuf {
		bBuf[iIndex] = byte(pReader.iPos + iIndex)
	}
	pReader.iPos += len(bBuf)
	return len(bBuf), nil
}
func (pReader *holdReader) Seek(i64Offset int64, iWhence int) (int64, error) {
	pReader.iPos = int(i64Offset)
	return i64Offset, nil
}
func (pReader *holdReader) Close() error { return nil }

// tapEnd 는 서버 쪽 끝에서 읽은 바이트를 엿본다.
type tapEnd struct {
	pEnd   *remoteEnd
	fnSeen func(bData []byte)
}

func (pTap *tapEnd) Read(bBuf []byte) (int, error) {
	nRead, errRead := pTap.pEnd.Read(bBuf)
	if nRead > 0 {
		pTap.fnSeen(bBuf[:nRead])
	}
	return nRead, errRead
}
func (pTap *tapEnd) Write(bData []byte) error { return pTap.pEnd.Write(bData) }

// 설정으로 끄거나(Enabled=false) 연결 기능이 없으면(Zmodem=nil) 시작 표시도 그대로 화면에 나온다.
func TestZmodemDisabledPassesThrough(t *testing.T) {
	for _, pHooks := range []*ZmodemHooks{nil, {Enabled: func() bool { return false }}} {
		pLink := newLinkTransport()
		_, rec := openZmodemSession(t, pLink, pHooks)
		pLink.chToApp <- []byte("x" + g_cszTestZrqinit)
		waitUntil(t, func() bool { return strings.Contains(rec.screen(), "B00000000000000") }, 2*time.Second)
		select {
		case bGot := <-pLink.chToRemote:
			t.Fatalf("꺼져 있으면 응답하지 않아야 함: %q", bGot)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// 보낼 파일을 고르지 않으면 취소하고 서버에 중단을 알린다. 세션은 그대로 쓴다.
func TestZmodemUploadPickCancelled(t *testing.T) {
	pLink := newLinkTransport()
	pSession, _ := openZmodemSession(t, pLink, &ZmodemHooks{PickFiles: func() ([]string, error) { return nil, nil }})
	chRemote := make(chan error, 1)
	go func() { chRemote <- remoteRz(pLink, &memFiles{mapFiles: map[string][]byte{}}) }()
	select {
	case errRemote := <-chRemote:
		if errRemote != zmodem.ErrRemoteCancelled {
			t.Fatalf("서버 쪽은 상대 취소로 끝나야 함: %v", errRemote)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("서버 쪽이 중단 알림을 받지 못함")
	}
	waitUntil(t, func() bool { return !pSession.zActive.Load() }, 2*time.Second)
	if pSession.Snapshot().M_szState != G_cszStateConnected {
		t.Fatal("세션은 연결된 채여야 함")
	}
}

func TestSafeFileName(t *testing.T) {
	for szIn, szWant := range map[string]string{
		"report.pdf":             "report.pdf",
		"/var/log/app.log":       "app.log",
		`C:\Users\x\secret.txt`:  "secret.txt",
		"../../.ssh/id_rsa":      "id_rsa",
		"a:b*c?.txt":             "a_b_c_.txt",
		"trailing. . ":           "trailing",
		"":                       "zmodem-file",
		"..":                     "zmodem-file",
		"dir/":                   "zmodem-file",
		"CON":                    "_CON",
		"nul.txt":                "_nul.txt",
		"com1.log":               "_com1.log",
		"tab\there":              "tab_here",
		"한글 파일.txt":              "한글 파일.txt",
		strings.Repeat("가", 300): strings.Repeat("가", 200),
	} {
		if szGot := SafeFileName(szIn); szGot != szWant {
			t.Errorf("SafeFileName(%q) = %q, 기대 %q", szIn, szGot, szWant)
		}
	}
}

func TestUniquePathSkipsExistingAndInProgress(t *testing.T) {
	szDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(szDir, "a.tar.gz"), nil, 0o644)
	_ = os.WriteFile(filepath.Join(szDir, "a.tar (2).gz"+g_cszZmodemTempSuffix), nil, 0o644) // 다른 전송이 받는 중
	szGot, errPath := uniquePath(szDir, "a.tar.gz")
	if errPath != nil || filepath.Base(szGot) != "a.tar (3).gz" {
		t.Fatalf("uniquePath = %q %v", szGot, errPath)
	}
	if szFresh, _ := uniquePath(szDir, "b"); filepath.Base(szFresh) != "b" {
		t.Fatalf("없는 이름은 그대로: %q", szFresh)
	}
}

func TestFormatBytes(t *testing.T) {
	for i64In, szWant := range map[int64]string{-1: "?", 0: "0 B", 1023: "1023 B", 1024: "1.0 KB", 1536: "1.5 KB", 5 * 1024 * 1024: "5.0 MB", 3 * 1024 * 1024 * 1024: "3.00 GB"} {
		if szGot := formatBytes(i64In); szGot != szWant {
			t.Errorf("formatBytes(%d) = %q, 기대 %q", i64In, szGot, szWant)
		}
	}
}
