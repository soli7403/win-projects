package sftpfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

// 임시 이름으로 쓰고 끝나면 이름 바꾸기 (D104, spec/protocols.md §3 "파일 쓰기(임시 이름)").

// atomicSide 는 같은 테스트를 서버(메모리 파이프 SFTP)와 이 PC 파일시스템에서 돌리기 위한 묶음이다.
type atomicSide struct {
	szName string
	be     Backend
	szRoot string                     // 실제 디스크 폴더
	fnPath func(szName string) string // Backend 에 넘길 경로
}

func atomicSides(t *testing.T) []atomicSide {
	szRemote, szLocal := t.TempDir(), t.TempDir()
	return []atomicSide{
		{"sftp", sftpBackend{client: newPipeClient(t, szRemote)}, szRemote, func(szName string) string { return szName }},
		{"local", LocalBackend{}, szLocal, func(szName string) string { return filepath.Join(szLocal, szName) }},
	}
}

// noPartLeft 는 폴더에 임시 파일이 남지 않았는지 본다.
func noPartLeft(t *testing.T, szDir string) {
	t.Helper()
	slEntries, errRead := os.ReadDir(szDir)
	if errRead != nil {
		t.Fatal(errRead)
	}
	for _, entry := range slEntries {
		if IsPartName(entry.Name()) {
			t.Fatalf("임시 파일이 남음: %s", entry.Name())
		}
	}
}

func writeString(szData string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, errWrite := io.WriteString(w, szData)
		return errWrite
	}
}

var errBroken = errors.New("연결 끊김(흉내)")

// 두 글자 쓰다 실패 → 기존 파일 그대로, 새 파일은 생기지 않음, 임시 파일 없음.
func TestWriteReplaceFailureKeepsTarget(t *testing.T) {
	for _, side := range atomicSides(t) {
		t.Run(side.szName, func(t *testing.T) {
			mustWrite(t, filepath.Join(side.szRoot, "old.txt"), []byte("old content"))
			fnFail := func(w io.Writer) error {
				_, _ = io.WriteString(w, "ne")
				return errBroken
			}
			if errWrite := writeReplace(side.be, side.fnPath("old.txt"), fnFail); !errors.Is(errWrite, errBroken) {
				t.Fatalf("오류 = %v", errWrite)
			}
			mustEqualFile(t, filepath.Join(side.szRoot, "old.txt"), []byte("old content"))
			if errWrite := writeReplace(side.be, side.fnPath("new.txt"), fnFail); !errors.Is(errWrite, errBroken) {
				t.Fatalf("오류 = %v", errWrite)
			}
			if _, errStat := os.Stat(filepath.Join(side.szRoot, "new.txt")); !os.IsNotExist(errStat) {
				t.Fatal("실패한 새 파일이 대상 이름으로 생김")
			}
			noPartLeft(t, side.szRoot)
		})
	}
}

// 덮어쓰기·새 파일 모두 새 내용, 임시 파일 없음. 테스트 서버는 posix-rename 을 알린다.
func TestWriteReplaceSuccess(t *testing.T) {
	for _, side := range atomicSides(t) {
		t.Run(side.szName, func(t *testing.T) {
			if pSftp, bOk := side.be.(sftpBackend); bOk {
				if _, bPosix := pSftp.client.HasExtension(G_cszPosixRename); !bPosix {
					t.Fatal("테스트 서버가 posix-rename 을 알리지 않음")
				}
			}
			mustWrite(t, filepath.Join(side.szRoot, "a.txt"), []byte("old"))
			if errWrite := writeReplace(side.be, side.fnPath("a.txt"), writeString("new content")); errWrite != nil {
				t.Fatal(errWrite)
			}
			mustEqualFile(t, filepath.Join(side.szRoot, "a.txt"), []byte("new content"))
			if errWrite := writeReplace(side.be, side.fnPath("b.txt"), writeString("fresh")); errWrite != nil {
				t.Fatal(errWrite)
			}
			mustEqualFile(t, filepath.Join(side.szRoot, "b.txt"), []byte("fresh"))
			noPartLeft(t, side.szRoot)
		})
	}
}

// 0750 파일을 덮어써도 0750.
func TestWriteReplaceKeepsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 는 권한 비트가 없다")
	}
	for _, side := range atomicSides(t) {
		t.Run(side.szName, func(t *testing.T) {
			szDisk := filepath.Join(side.szRoot, "run.sh")
			mustWrite(t, szDisk, []byte("#!/bin/sh\n"))
			if errChmod := os.Chmod(szDisk, 0o750); errChmod != nil {
				t.Fatal(errChmod)
			}
			if errWrite := writeReplace(side.be, side.fnPath("run.sh"), writeString("#!/bin/sh\necho hi\n")); errWrite != nil {
				t.Fatal(errWrite)
			}
			info, errStat := os.Stat(szDisk)
			if errStat != nil {
				t.Fatal(errStat)
			}
			if info.Mode().Perm() != 0o750 {
				t.Fatalf("권한 = %o, 기대 750", info.Mode().Perm())
			}
			mustEqualFile(t, szDisk, []byte("#!/bin/sh\necho hi\n"))
		})
	}
}

// fallbackBackend 는 임시 파일 만들기 실패·소유자 다름을 흉내 내고 Replace 호출을 센다.
type fallbackBackend struct {
	Backend
	bFailPart   bool
	bOwnerDiff  bool
	mtx         sync.Mutex
	iReplaceCnt int
}

func (pBe *fallbackBackend) OpenWrite(szPath string) (io.WriteCloser, error) {
	if pBe.bFailPart && IsPartName(pBe.Base(szPath)) {
		return nil, fs.ErrPermission
	}
	return pBe.Backend.OpenWrite(szPath)
}

func (pBe *fallbackBackend) OwnerKey(info fs.FileInfo) string {
	if pBe.bOwnerDiff && IsPartName(info.Name()) {
		return "2:2"
	}
	return "1:1"
}

func (pBe *fallbackBackend) Replace(szOld string, szNew string) error {
	pBe.mtx.Lock()
	pBe.iReplaceCnt++
	pBe.mtx.Unlock()
	return pBe.Backend.Replace(szOld, szNew)
}

// 임시 파일을 만들 수 없거나 소유자가 다르면 실패하지 않고 그 자리에 쓰며 임시 파일이 남지 않는다.
func TestWriteReplaceFallsBackToDirect(t *testing.T) {
	slCases := []struct {
		szName string
		pBe    *fallbackBackend
	}{
		{"임시 파일 못 만듦", &fallbackBackend{Backend: LocalBackend{}, bFailPart: true}},
		{"소유자 다름", &fallbackBackend{Backend: LocalBackend{}, bOwnerDiff: true}},
	}
	for _, tc := range slCases {
		t.Run(tc.szName, func(t *testing.T) {
			szRoot := t.TempDir()
			szPath := filepath.Join(szRoot, "f.txt")
			mustWrite(t, szPath, []byte("old"))
			if errWrite := writeReplace(tc.pBe, szPath, writeString("direct")); errWrite != nil {
				t.Fatal(errWrite)
			}
			mustEqualFile(t, szPath, []byte("direct"))
			if tc.pBe.iReplaceCnt != 0 {
				t.Fatal("그 자리에 써야 하는데 이름 바꾸기를 했음")
			}
			noPartLeft(t, szRoot)
		})
	}
}

// 링크에 쓰면 링크는 링크로 남고 가리키는 파일이 바뀐다.
func TestWriteReplaceKeepsSymlink(t *testing.T) {
	szRoot := t.TempDir()
	szReal, szLink := filepath.Join(szRoot, "real.conf"), filepath.Join(szRoot, "link.conf")
	mustWrite(t, szReal, []byte("old"))
	if errLink := os.Symlink(szReal, szLink); errLink != nil {
		t.Skipf("심볼릭 링크를 만들 수 없음: %v", errLink)
	}
	if errWrite := writeReplace(LocalBackend{}, szLink, writeString("new")); errWrite != nil {
		t.Fatal(errWrite)
	}
	info, errStat := os.Lstat(szLink)
	if errStat != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("링크가 링크로 남지 않음: %v %v", info, errStat)
	}
	mustEqualFile(t, szReal, []byte("new"))
	noPartLeft(t, szRoot)
}

// 임시 이름 형식, 같은 폴더, 매번 다른 이름, 긴 이름도 255바이트 이하·글자 안 깨짐, IsPartName 판정.
func TestPartName(t *testing.T) {
	be := LocalBackend{}
	szDir := t.TempDir()
	reForm := regexp.MustCompile(`^data\.csv\.dtpart-[0-9a-f]{8}$`)
	szA, szB := partPath(be, filepath.Join(szDir, "data.csv")), partPath(be, filepath.Join(szDir, "data.csv"))
	if !reForm.MatchString(filepath.Base(szA)) || filepath.Dir(szA) != szDir {
		t.Fatalf("임시 이름 형식·폴더: %s", szA)
	}
	if szA == szB {
		t.Fatal("임시 이름이 매번 달라야 함")
	}
	szLong := strings.Repeat("가", 86) // 258바이트
	szPart := filepath.Base(partPath(be, filepath.Join(szDir, szLong)))
	if len(szPart) > 255 || !utf8.ValidString(szPart) || !IsPartName(szPart) {
		t.Fatalf("긴 이름: %d바이트, 올바른 UTF-8=%v", len(szPart), utf8.ValidString(szPart))
	}
	for szName, bWant := range map[string]bool{
		filepath.Base(szA):    true,
		"a.txt":               false,
		"a.dtpart-1234567":    false,
		"a.dtpart-1234567g":   false,
		"a.dtpart-1234ABCD":   false,
		".dtpart-12345678":    false,
		"a.dtpart-12345678.x": false,
	} {
		if IsPartName(szName) != bWant {
			t.Errorf("IsPartName(%q) = %v, 기대 %v", szName, !bWant, bWant)
		}
	}
}

// uploadFile 이 원본 읽기에서 실패하면 서버의 기존 파일 그대로. 성공하면 바뀐다. downloadFile 도 바뀌고 양쪽에 임시 파일 없음.
func TestTransferKeepsOldUntilDone(t *testing.T) {
	szRemote, szLocal := t.TempDir(), t.TempDir()
	svc := NewService("s", newPipeClient(t, szRemote), nil, "u@h:22")
	fnReport := func(int64) {}
	mustWrite(t, filepath.Join(szRemote, "f.txt"), []byte("server old"))
	// 폴더는 열 수는 있지만 읽으면 실패한다 (읽는 도중 끊김의 흉내).
	if errUp := svc.uploadFile(t.TempDir(), "f.txt", fnReport); errUp == nil {
		t.Fatal("폴더를 원본으로 올리면 실패해야 함")
	}
	mustEqualFile(t, filepath.Join(szRemote, "f.txt"), []byte("server old"))
	noPartLeft(t, szRemote)

	mustWrite(t, filepath.Join(szLocal, "up.txt"), []byte("local new"))
	if errUp := svc.uploadFile(filepath.Join(szLocal, "up.txt"), "f.txt", fnReport); errUp != nil {
		t.Fatal(errUp)
	}
	mustEqualFile(t, filepath.Join(szRemote, "f.txt"), []byte("local new"))

	mustWrite(t, filepath.Join(szLocal, "down.txt"), []byte("local old"))
	if errDown := svc.downloadFile("f.txt", filepath.Join(szLocal, "down.txt"), fnReport); errDown != nil {
		t.Fatal(errDown)
	}
	mustEqualFile(t, filepath.Join(szLocal, "down.txt"), []byte("local new"))
	noPartLeft(t, szRemote)
	noPartLeft(t, szLocal)
}

// gateBackend 는 대상 파일의 첫 쓰기 뒤에 멈춰, 그 사이에 취소할 수 있게 한다.
type gateBackend struct {
	Backend
	chFirst   chan struct{}
	chRelease chan struct{}
	once      sync.Once
}

type gateWriter struct {
	io.WriteCloser
	pBe *gateBackend
}

func (pW *gateWriter) Write(bData []byte) (int, error) {
	nWritten, errWrite := pW.WriteCloser.Write(bData)
	pW.pBe.once.Do(func() {
		close(pW.pBe.chFirst)
		<-pW.pBe.chRelease
	})
	return nWritten, errWrite
}

func (pBe *gateBackend) OpenWrite(szPath string) (io.WriteCloser, error) {
	w, errOpen := pBe.Backend.OpenWrite(szPath)
	if errOpen != nil {
		return nil, errOpen
	}
	return &gateWriter{WriteCloser: w, pBe: pBe}, nil
}

// 끌어 놓기 복사를 파일 쓰는 도중 취소 → canceled, 서버의 기존 파일 그대로, 임시 파일 없음.
func TestCopyCancelMidFileKeepsTarget(t *testing.T) {
	szLocal, szRemote := t.TempDir(), t.TempDir()
	bBig := bytes.Repeat([]byte("0123456789abcdef"), 16*1024) // 256KB: 쓰기가 여러 번 나뉜다
	mustWrite(t, filepath.Join(szLocal, "big.bin"), bBig)
	mustWrite(t, filepath.Join(szRemote, "big.bin"), []byte("old"))
	svcL := NewLocalService("l", szLocal, nil)
	svcR := NewService("r", newPipeClient(t, szRemote), nil, "u@r:22")
	pGate := &gateBackend{Backend: svcR.be, chFirst: make(chan struct{}), chRelease: make(chan struct{})}
	svcR.be = pGate
	pWatch := newJobWatcher()
	start, errStart := StartCopy(CopyRequest{Src: svcL, Paths: []string{filepath.Join(szLocal, "big.bin")}, Dst: svcR, DstDir: "", Overwrite: true, Emit: pWatch.emit})
	if errStart != nil || start.M_szJobId == "" {
		t.Fatalf("시작 실패: %+v %v", start, errStart)
	}
	<-pGate.chFirst
	CancelCopy(start.M_szJobId)
	close(pGate.chRelease)
	if job := pWatch.wait(t); job.M_szState != "canceled" {
		t.Fatalf("상태 = %q", job.M_szState)
	}
	mustEqualFile(t, filepath.Join(szRemote, "big.bin"), []byte("old"))
	noPartLeft(t, szRemote)
}
