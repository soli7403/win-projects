package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDialogs 는 백업 대화상자 대역이다.
type fakeDialogs struct {
	szPath string
}

func (pFake *fakeDialogs) SaveFile(szDefaultName string, szFilterName string, szPattern string) (string, bool) {
	return pFake.szPath, true
}

func (pFake *fakeDialogs) OpenFile(szFilterName string, szPattern string) (string, bool) {
	return pFake.szPath, true
}

// TestBackupRoundTrip 은 내보내기 파일 형식과 가져오기 뒤 볼트 잠금 상태를 확인한다 (acceptance Y1, Y2).
func TestBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pSrc, errSrc := NewService(filepath.Join(dir, "src"), nil)
	if errSrc != nil {
		t.Fatal(errSrc)
	}
	szRouterCall := func(pService *Service, szChannel string, szArgs string) string {
		szOut, errCall := pService.Router().Invoke(szChannel, szArgs)
		if errCall != nil {
			t.Fatalf("%s: %v", szChannel, errCall)
		}
		return szOut
	}
	szRouterCall(pSrc, "vault.setup", `["correct-horse-1"]`)
	szRouterCall(pSrc, "store.patch", `[{"hosts":[{"id":"h1","alias":"web","host":"10.0.0.1"}]}]`)
	szPath := filepath.Join(dir, "termgo-backup-test.tgbackup")
	pSrc.SetBackupDialogs(&fakeDialogs{szPath: szPath})
	if szExport := szRouterCall(pSrc, "backup.export", "[]"); !strings.Contains(szExport, "termgo-backup") {
		t.Fatalf("내보내기 경로 불일치: %s", szExport)
	}
	bFile, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	var mapFile map[string]any
	if errJson := json.Unmarshal(bFile, &mapFile); errJson != nil || mapFile["app"] != "termgo" {
		t.Fatalf("백업 app 값 불일치: %v", errJson)
	}

	pDst, errDst := NewService(filepath.Join(dir, "dst"), nil)
	if errDst != nil {
		t.Fatal(errDst)
	}
	pDst.SetBackupDialogs(&fakeDialogs{szPath: szPath})
	szRouterCall(pDst, "backup.import", "[]")
	if !strings.Contains(szRouterCall(pDst, "store.get", "[]"), `"alias":"web"`) {
		t.Fatal("가져온 저장 데이터가 없음")
	}
	if szStatus := szRouterCall(pDst, "vault.status", "[]"); !strings.Contains(szStatus, `"unlocked":false`) || !strings.Contains(szStatus, `"initialized":true`) {
		t.Fatalf("가져온 볼트는 잠긴 상태여야 함: %s", szStatus)
	}
	if !strings.Contains(szRouterCall(pDst, "vault.unlock", `["correct-horse-1"]`), "true") {
		t.Fatal("가져온 볼트를 마스터 비밀번호로 열지 못함")
	}

	szBad := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(szBad, []byte(`{"app":"other"}`), 0o600)
	pDst.SetBackupDialogs(&fakeDialogs{szPath: szBad})
	if _, errBad := pDst.Router().Invoke("backup.import", "[]"); errBad == nil || errBad.Error() != "백업 파일이 아닙니다" {
		t.Fatalf("app 불일치 오류 문구 불일치: %v", errBad)
	}
}
