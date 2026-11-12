package sftpsvc

import (
	"os"
	"path/filepath"
	"testing"

	"termgo/internal/ui"
)

func writeFile(t *testing.T, szPath, szBody string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(szPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(szPath, []byte(szBody), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlan(t *testing.T) {
	szRoot := t.TempDir()
	writeFile(t, filepath.Join(szRoot, "a.txt"), "A")
	writeFile(t, filepath.Join(szRoot, "a (1).txt"), "A1")
	writeFile(t, filepath.Join(szRoot, "dir", "x.txt"), "X")
	fsys := localFS{}
	other := remoteFS{szKey: "ssh:u@h:22"}

	tests := []struct {
		name    string
		src     string
		dst     fileSys
		dir     string
		want    string
		wantErr bool
	}{
		{"같은 자리 파일 → 새 이름", filepath.Join(szRoot, "a.txt"), fsys, szRoot, filepath.Join(szRoot, "a (2).txt"), false},
		{"같은 자리 폴더 → 새 이름", filepath.Join(szRoot, "dir"), fsys, szRoot, filepath.Join(szRoot, "dir (1)"), false},
		{"다른 폴더", filepath.Join(szRoot, "a.txt"), fsys, filepath.Join(szRoot, "dir"), filepath.Join(szRoot, "dir", "a.txt"), false},
		{"폴더를 자기 안으로", filepath.Join(szRoot, "dir"), fsys, filepath.Join(szRoot, "dir"), "", true},
		{"다른 파일시스템은 그대로", filepath.Join(szRoot, "a.txt"), other, "/tmp", "/tmp/a.txt", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := plan(fsys, tc.src, tc.dst, tc.dir)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsWithin(t *testing.T) {
	r := remoteFS{}
	if !isWithin(r, "/a/b", "/a/b/c") || !isWithin(r, "/a/b", "/a/b") {
		t.Fatal("하위 경로를 못 찾음")
	}
	if isWithin(r, "/a/b", "/a/bc") {
		t.Fatal("/a/bc 는 /a/b 아래가 아니다")
	}
}

func TestCopyTreeLocal(t *testing.T) {
	szRoot := t.TempDir()
	writeFile(t, filepath.Join(szRoot, "src", "sub", "f.txt"), "hello")
	writeFile(t, filepath.Join(szRoot, "src", "g.txt"), "world")
	svc := &Service{UI: ui.NewBridge(func(string, ...any) {})}
	szDst := filepath.Join(szRoot, "out", "src")
	if err := os.MkdirAll(filepath.Dir(szDst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := svc.copyTree(localFS{}, filepath.Join(szRoot, "src"), localFS{}, szDst, xfer{szSession: "s"}); err != nil {
		t.Fatal(err)
	}
	for szRel, szWant := range map[string]string{filepath.Join("sub", "f.txt"): "hello", "g.txt": "world"} {
		arr, err := os.ReadFile(filepath.Join(szDst, szRel))
		if err != nil || string(arr) != szWant {
			t.Fatalf("%s = %q, %v", szRel, arr, err)
		}
	}
}

func TestRunCopyJob(t *testing.T) {
	szRoot := t.TempDir()
	writeFile(t, filepath.Join(szRoot, "src", "d", "1.txt"), "12345")
	writeFile(t, filepath.Join(szRoot, "src", "d", "2.txt"), "678")
	var last CopyJob
	svc := New(nil, ui.NewBridge(func(szCh string, args ...any) {
		if szCh == "copyjob:update" {
			last = args[0].(CopyJob)
		}
	}), nil)
	pstrJ := svc.newJob(CopyJob{SrcSession: "a", DstSession: "b", Count: 1})
	szTo := filepath.Join(szRoot, "dst", "d")
	if err := os.MkdirAll(filepath.Dir(szTo), 0o755); err != nil {
		t.Fatal(err)
	}
	svc.endJob(pstrJ, svc.runCopy(pstrJ, localFS{}, []string{filepath.Join(szRoot, "src", "d")}, localFS{}, []string{szTo}))
	if last.State != "done" || last.Files != 2 || last.FilesDone != 2 || last.Total != 8 || last.Bytes != 8 {
		t.Fatalf("job = %+v", last)
	}
	if arr, _ := os.ReadFile(filepath.Join(szTo, "2.txt")); string(arr) != "678" {
		t.Fatalf("2.txt = %q", arr)
	}

	pstrJ = svc.newJob(CopyJob{SrcSession: "a", DstSession: "b", Count: 1})
	svc.CancelJob(pstrJ.j.ID)
	svc.endJob(pstrJ, svc.runCopy(pstrJ, localFS{}, []string{filepath.Join(szRoot, "src", "d")}, localFS{}, []string{szTo + "x"}))
	if last.State != "canceled" {
		t.Fatalf("취소 상태 = %q", last.State)
	}
}

// 복사는 항상 바이너리 — 줄바꿈·NUL·상위 바이트가 한 바이트도 바뀌지 않아야 한다.
func TestCopyIsBinary(t *testing.T) {
	szRoot := t.TempDir()
	arrWant := []byte("a\r\nb\nc\rd\x00\xff\xfe\x1a\r\n")
	szSrc := filepath.Join(szRoot, "bin.dat")
	if err := os.WriteFile(szSrc, arrWant, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(nil, ui.NewBridge(func(string, ...any) {}), nil)
	szDst := filepath.Join(szRoot, "out.dat")
	if err := svc.copyTree(localFS{}, szSrc, localFS{}, szDst, xfer{szSession: "s"}); err != nil {
		t.Fatal(err)
	}
	arrGot, err := os.ReadFile(szDst)
	if err != nil {
		t.Fatal(err)
	}
	if string(arrGot) != string(arrWant) {
		t.Fatalf("바이트가 바뀜: got %q, want %q", arrGot, arrWant)
	}
}
