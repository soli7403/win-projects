package sftpfs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// jobWatcher 는 copyjob.update 이벤트를 모아 작업이 끝날 때까지 기다린다.
type jobWatcher struct {
	chDone chan CopyJob
}

func newJobWatcher() *jobWatcher { return &jobWatcher{chDone: make(chan CopyJob, 8)} }

func (pWatch *jobWatcher) emit(szEvent string, anyData ...any) {
	if szEvent != G_cszCopyEvent || len(anyData) == 0 {
		return
	}
	job := anyData[0].(CopyJob)
	switch job.M_szState {
	case "done", "error", "canceled":
		pWatch.chDone <- job
	}
}

func (pWatch *jobWatcher) wait(t *testing.T) CopyJob {
	t.Helper()
	select {
	case job := <-pWatch.chDone:
		return job
	case <-time.After(10 * time.Second):
		t.Fatal("복사 작업이 끝나지 않음")
	}
	return CopyJob{}
}

var g_bBinary = []byte("a\r\nb\nc\rd\x00\xff\xfe\x1a\r\n")

func mustWrite(t *testing.T, szPath string, bData []byte) {
	t.Helper()
	if errMk := os.MkdirAll(filepath.Dir(szPath), 0o755); errMk != nil {
		t.Fatal(errMk)
	}
	if errWrite := os.WriteFile(szPath, bData, 0o644); errWrite != nil {
		t.Fatal(errWrite)
	}
}

func mustEqualFile(t *testing.T, szPath string, bWant []byte) {
	t.Helper()
	bGot, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if !bytes.Equal(bGot, bWant) {
		t.Fatalf("%s 바이트가 다름: %q, 기대 %q", szPath, bGot, bWant)
	}
}

// 서버 → 서버는 이 PC 임시 폴더를 거치고, 끝나면 임시 사본이 없다. 바이트는 그대로. file-panel.md §2.5, §6-4·10·11.
func TestCopyRelayServerToServer(t *testing.T) {
	szRootA, szRootB, szTemp := t.TempDir(), t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(szRootA, "proj", "bin.dat"), g_bBinary)
	mustWrite(t, filepath.Join(szRootA, "proj", "sub", "t.txt"), []byte("hello"))
	_ = os.Mkdir(filepath.Join(szRootB, "in"), 0o755)
	pWatch := newJobWatcher()
	svcA := NewService("a", newPipeClient(t, szRootA), nil, "u@a:22")
	svcB := NewService("b", newPipeClient(t, szRootB), nil, "u@b:22")
	req := CopyRequest{Src: svcA, SrcTitle: "A", Paths: []string{"proj"}, Dst: svcB, DstTitle: "B", DstDir: "in", TempDir: szTemp, Emit: pWatch.emit}

	start, errStart := StartCopy(req)
	if errStart != nil || start.M_szJobId == "" {
		t.Fatalf("시작 실패: %+v %v", start, errStart)
	}
	job := pWatch.wait(t)
	if job.M_szState != "done" || !job.M_bRelay || job.M_iFiles != 2 || job.M_iFilesDone != 2 || job.M_i64Total != int64(2*(len(g_bBinary)+5)) {
		t.Fatalf("작업 = %+v", job)
	}
	mustEqualFile(t, filepath.Join(szRootB, "in", "proj", "bin.dat"), g_bBinary)
	mustEqualFile(t, filepath.Join(szRootB, "in", "proj", "sub", "t.txt"), []byte("hello"))
	if slLeft, _ := os.ReadDir(szTemp); len(slLeft) != 0 {
		t.Fatalf("임시 폴더에 남은 것: %v", slLeft)
	}

	// 같은 이름이 있으면 덮어쓰기가 아니면 시작하지 않는다 (§2.4).
	again, errAgain := StartCopy(req)
	if errAgain != nil || again.M_szJobId != "" || len(again.M_slConflicts) != 1 || again.M_slConflicts[0] != "proj" {
		t.Fatalf("충돌 응답 = %+v %v", again, errAgain)
	}
}

// 같은 파일시스템의 원래 자리면 "이름 (1)", 폴더를 자기 안으로는 거부. 로컬 → 로컬. §2.4.
func TestCopyLocalSamePlaceAndInside(t *testing.T) {
	szRoot := t.TempDir()
	mustWrite(t, filepath.Join(szRoot, "a.txt"), g_bBinary)
	mustWrite(t, filepath.Join(szRoot, "dir", "x.txt"), []byte("x"))
	svc := NewLocalService("l", szRoot, nil)
	pWatch := newJobWatcher()

	start, errStart := StartCopy(CopyRequest{Src: svc, Paths: []string{filepath.Join(szRoot, "a.txt")}, Dst: svc, DstDir: "", Emit: pWatch.emit})
	if errStart != nil || start.M_szJobId == "" {
		t.Fatalf("시작 실패: %+v %v", start, errStart)
	}
	if job := pWatch.wait(t); job.M_szState != "done" || job.M_szPhase != "copy" {
		t.Fatalf("작업 = %+v", job)
	}
	mustEqualFile(t, filepath.Join(szRoot, "a (1).txt"), g_bBinary)
	mustEqualFile(t, filepath.Join(szRoot, "a.txt"), g_bBinary)

	_, errInside := StartCopy(CopyRequest{Src: svc, Paths: []string{filepath.Join(szRoot, "dir")}, Dst: svc, DstDir: filepath.Join(szRoot, "dir"), Emit: pWatch.emit})
	if errInside == nil {
		t.Fatal("폴더를 자기 안으로 복사하면 오류여야 함")
	}
	if _, errMissing := StartCopy(CopyRequest{Src: svc, Paths: []string{filepath.Join(szRoot, "a.txt")}, Dst: svc, DstDir: filepath.Join(szRoot, "nope"), Emit: pWatch.emit}); errMissing == nil {
		t.Fatal("대상 폴더가 없으면 오류여야 함")
	}
}

// 차례를 기다리는 동안 취소하면 아무것도 쓰지 않고 canceled. 로컬 → 서버. §3.
func TestCopyCancelWhileQueued(t *testing.T) {
	szLocal, szRemote := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(szLocal, "f.bin"), g_bBinary)
	svcL := NewLocalService("l", szLocal, nil)
	svcR := NewService("r", newPipeClient(t, szRemote), nil, "u@r:22")
	pWatch := newJobWatcher()
	svcR.mtxCopy.Lock() // 다른 작업이 진행 중인 것처럼
	start, errStart := StartCopy(CopyRequest{Src: svcL, Paths: []string{filepath.Join(szLocal, "f.bin")}, Dst: svcR, DstDir: "", Emit: pWatch.emit})
	if errStart != nil {
		t.Fatal(errStart)
	}
	CancelCopy(start.M_szJobId)
	svcR.mtxCopy.Unlock()
	if job := pWatch.wait(t); job.M_szState != "canceled" {
		t.Fatalf("상태 = %q", job.M_szState)
	}
	if _, errStat := os.Stat(filepath.Join(szRemote, "f.bin")); errStat == nil {
		t.Fatal("취소한 작업이 파일을 썼음")
	}
}

// 로컬 세션: 빈 경로는 시작 폴더, 목록에 상위 폴더·구분자·로컬 여부. §1.
func TestLocalListing(t *testing.T) {
	szRoot := t.TempDir()
	mustWrite(t, filepath.Join(szRoot, "k.txt"), []byte("k"))
	listing, errList := NewLocalFs(szRoot).List("")
	if errList != nil {
		t.Fatal(errList)
	}
	if !listing.M_bLocal || listing.M_szSep != string(filepath.Separator) || listing.M_szParent != filepath.Dir(listing.M_szPath) || len(listing.M_slEntries) != 1 {
		t.Fatalf("목록 = %+v", listing)
	}
	if listing.M_slEntries[0].M_szPath != filepath.Join(listing.M_szPath, "k.txt") {
		t.Fatalf("항목 경로 = %s", listing.M_slEntries[0].M_szPath)
	}
}
