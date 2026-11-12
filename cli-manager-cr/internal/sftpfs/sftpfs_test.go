package sftpfs

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/sftp"
)

// newPipeClient 는 메모리 파이프로 sftp 서버와 클라이언트를 잇는다. szRoot 가 서버의 작업 폴더다 (원격 경로는 여기 기준 상대 경로).
func newPipeClient(t *testing.T, szRoot string) *sftp.Client {
	t.Helper()
	cs, ss := net.Pipe()
	srv, errSrv := sftp.NewServer(ss, sftp.WithServerWorkingDirectory(szRoot))
	if errSrv != nil {
		t.Fatal(errSrv)
	}
	go func() { _ = srv.Serve() }()
	client, errClient := sftp.NewClientPipe(cs, cs)
	if errClient != nil {
		t.Fatal(errClient)
	}
	t.Cleanup(func() { _ = client.Close(); _ = srv.Close() })
	return client
}

// 목록: 폴더 먼저, 이름 오름차순(대소문자 무시), 권한 문자열. protocols.md §3.
func TestListOrderAndPerm(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "A.txt"), []byte("a"), 0o644)
	_ = os.Mkdir(filepath.Join(root, "zdir"), 0o755)
	_ = os.Mkdir(filepath.Join(root, "Adir"), 0o755)

	fsys := NewFs(newPipeClient(t, root))
	listing, errList := fsys.List("")
	if errList != nil {
		t.Fatal(errList)
	}
	var names []string
	for _, entry := range listing.M_slEntries {
		names = append(names, entry.M_szName)
	}
	if strings.Join(names, ",") != "Adir,zdir,A.txt,b.txt" {
		t.Fatalf("정렬 불일치: %v", names)
	}
	if !listing.M_slEntries[0].M_bIsDir || listing.M_slEntries[0].M_szPerm[0] != 'd' {
		t.Fatal("폴더 항목은 d 권한 문자열이어야 함")
	}
}

// 업로드 → 다운로드 왕복(폴더 재귀 포함), 재귀 삭제. protocols.md §3 업로드·다운로드·삭제.
func TestUploadDownloadRemoveRoundTrip(t *testing.T) {
	root := t.TempDir()
	svc := NewService("s1", newPipeClient(t, root), func(string, ...any) {}, "u@h:22")
	srcDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(srcDir, "tree", "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "tree", "sub", "f.txt"), []byte("hello sftp"), 0o644)
	_ = os.Mkdir(filepath.Join(root, "rem"), 0o755)

	slUploaded, errUp := svc.UploadPaths([]string{filepath.Join(srcDir, "tree")}, "rem")
	if errUp != nil {
		t.Fatal(errUp)
	}
	if len(slUploaded) != 1 {
		t.Fatalf("올린 경로 수: %v", slUploaded)
	}
	dstDir := t.TempDir()
	if _, errDown := svc.DownloadPaths([]string{"rem/tree"}, dstDir); errDown != nil {
		t.Fatal(errDown)
	}
	bData, errRead := os.ReadFile(filepath.Join(dstDir, "tree", "sub", "f.txt"))
	if errRead != nil || !bytes.Equal(bData, []byte("hello sftp")) {
		t.Fatalf("왕복 내용 불일치: %q %v", bData, errRead)
	}
	if errRemove := svc.Fs.Remove([]string{"rem/tree"}); errRemove != nil {
		t.Fatal(errRemove)
	}
	if _, errStat := os.Stat(filepath.Join(root, "rem", "tree")); !os.IsNotExist(errStat) {
		t.Fatal("재귀 삭제 후 폴더가 남으면 안 됨")
	}
}

// 미리보기: 텍스트, NUL 포함 파일은 없음, 이미지는 data URL. protocols.md §3.
func TestPreviewRules(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "a.txt"), []byte("plain"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "bin.dat"), []byte{'a', 0, 'b'}, 0o644)
	_ = os.WriteFile(filepath.Join(root, "p.png"), []byte{0x89, 'P', 'N', 'G'}, 0o644)
	fsys := NewFs(newPipeClient(t, root))

	pText, errText := fsys.Preview("a.txt")
	if errText != nil || pText == nil || pText.M_szKind != "text" || pText.M_szData != "plain" {
		t.Fatalf("텍스트 미리보기: %+v %v", pText, errText)
	}
	pBin, errBin := fsys.Preview("bin.dat")
	if errBin != nil || pBin != nil {
		t.Fatalf("NUL 포함 파일은 미리보기 없음: %+v %v", pBin, errBin)
	}
	pImg, errImg := fsys.Preview("p.png")
	if errImg != nil || pImg == nil || pImg.M_szKind != "image" || !strings.HasPrefix(pImg.M_szData, "data:image/png;base64,") {
		t.Fatalf("이미지 미리보기: %+v %v", pImg, errImg)
	}
}

// 전송 큐: 같은 세션의 작업은 등록 순서대로 끝나고 done 이벤트가 나간다.
func TestQueueEmitsDoneInOrder(t *testing.T) {
	var mtxEmit sync.Mutex
	var slStates []string
	emit := func(szEvent string, anyData ...any) {
		if szEvent != "transfer.update" {
			return
		}
		mtxEmit.Lock()
		defer mtxEmit.Unlock()
		tr := anyData[0].(Transfer)
		if tr.M_szState == "done" {
			slStates = append(slStates, tr.M_szName)
		}
	}
	q := NewQueue("s", emit)
	chA := q.enqueue(&Transfer{M_szName: "a"}, func(report func(int64)) error { return nil })
	chB := q.enqueue(&Transfer{M_szName: "b"}, func(report func(int64)) error { return nil })
	<-chA
	<-chB
	mtxEmit.Lock()
	defer mtxEmit.Unlock()
	if strings.Join(slStates, ",") != "a,b" {
		t.Fatalf("완료 순서: %v", slStates)
	}
}
