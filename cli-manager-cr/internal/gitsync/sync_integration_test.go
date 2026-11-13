package gitsync

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"termgo/internal/vault"
)

// memLocal 은 테스트용 이 PC 데이터 저장소다.
type memLocal struct {
	pVault *vault.Vault
	data   Data
}

func (pMem *memLocal) LoadData() (Data, Orders, error) {
	data := NewData()
	for szId, raw := range pMem.data.M_mapGroups {
		data.M_mapGroups[szId] = raw
	}
	for szId, raw := range pMem.data.M_mapHosts {
		data.M_mapHosts[szId] = raw
	}
	for szId, raw := range pMem.data.M_mapSnippets {
		data.M_mapSnippets[szId] = raw
	}
	slItems, _ := pMem.pVault.Items()
	for _, item := range slItems {
		bItem, _ := json.Marshal(item)
		data.M_mapCreds[item.M_szId] = bItem
	}
	orders := Orders{M_slCreds: []string{}}
	for szId := range data.M_mapCreds {
		orders.M_slCreds = append(orders.M_slCreds, szId)
	}
	sort.Strings(orders.M_slCreds)
	return data, Orders{M_slHosts: sortedKeys(data.M_mapHosts), M_slGroups: sortedKeys(data.M_mapGroups), M_slSnippets: sortedKeys(data.M_mapSnippets), M_slCreds: orders.M_slCreds}, nil
}

func (pMem *memLocal) SaveData(data Data, orders Orders) error {
	pMem.data = NewData()
	for szId, raw := range data.M_mapGroups {
		pMem.data.M_mapGroups[szId] = raw
	}
	for szId, raw := range data.M_mapHosts {
		pMem.data.M_mapHosts[szId] = raw
	}
	for szId, raw := range data.M_mapSnippets {
		pMem.data.M_mapSnippets[szId] = raw
	}
	return pMem.pVault.ReplaceItems(sortedItems(data.M_mapCreds))
}

func sortedKeys(mapValues Collection) []string {
	slOut := []string{}
	for szId := range mapValues {
		slOut = append(slOut, szId)
	}
	sort.Strings(slOut)
	return slOut
}

// pcFixture 는 PC 한 대(데이터 폴더, 볼트, 저장소, 동기화기)다.
type pcFixture struct {
	szDir   string
	local   *memLocal
	syncer  *Syncer
	szAsked string
	bAnswer bool
}

func newPC(t *testing.T, szPassword string) *pcFixture {
	t.Helper()
	szDir := t.TempDir()
	pVault, errOpen := vault.Open(filepath.Join(szDir, "vault.json"))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	if szPassword != "" {
		if _, errSetup := pVault.Setup(szPassword); errSetup != nil {
			t.Fatal(errSetup)
		}
	}
	pFix := &pcFixture{szDir: szDir, local: &memLocal{pVault: pVault, data: NewData()}}
	pFix.syncer = NewSyncer(Deps{
		M_szDataDir: szDir,
		M_store:     pFix.local,
		M_vault:     VaultAdapter{M_pVault: pVault},
		M_fnAskPassword: func() (string, bool) {
			return pFix.szAsked, pFix.bAnswer
		},
		M_fnNow: time.Now,
	})
	return pFix
}

func setHost(pFix *pcFixture, szId string, szAlias string, szHost string) {
	raw, _ := json.Marshal(map[string]any{"id": szId, "alias": szAlias, "host": szHost})
	pFix.local.data.M_mapHosts[szId] = raw
}

func gitAt(t *testing.T, szDir string, slArgs ...string) string {
	t.Helper()
	pCmd := exec.Command("git", slArgs...)
	pCmd.Dir = szDir
	bOut, errRun := pCmd.CombinedOutput()
	if errRun != nil {
		t.Fatalf("git %v: %v %s", slArgs, errRun, bOut)
	}
	return string(bOut)
}

// TestSyncScenarios 는 빈 저장소 올리기(Y5), 볼트 불일치(Y8, Y9), 받기, 수정 충돌(Y6), 삭제 우선(Y7)을 확인한다.
func TestSyncScenarios(t *testing.T) {
	if !GitAvailable() {
		t.Skip("git 이 없음")
	}
	szBare := t.TempDir()
	gitAt(t, szBare, "init", "--bare", "-q", ".")

	pA := newPC(t, "master-pass-1")
	setHost(pA, "h-1", "web", "10.1.1.1")
	res := pA.syncer.Run(szBare, G_cszModeAuto)
	if res.M_szState != G_cszStateOk {
		t.Fatalf("첫 동기화 실패: %+v", res)
	}
	bShown := gitAt(t, szBare, "show", "main:"+G_cszSyncFileName)
	if strings.Contains(bShown, "10.1.1.1") || !strings.Contains(bShown, `"format": "termgo-sync-1"`) {
		t.Fatalf("저장소 파일 형식/평문 노출 점검 실패 (Y5)")
	}

	pB := newPC(t, "other-pass-2")
	if res := pB.syncer.Run(szBare, G_cszModeAuto); res.M_szState != G_cszStateVaultMismatch || res.M_szMessage != "저장소의 보관함이 이 PC 의 보관함과 다릅니다" {
		t.Fatalf("보관함 불일치 상태 아님 (Y8): %+v", res)
	}
	pB.szAsked, pB.bAnswer = "wrong-pass-9", true
	if res := pB.syncer.Run(szBare, G_cszModePullVault); res.M_szState != G_cszStateVaultMismatch || res.M_szMessage != "마스터 비밀번호가 맞지 않거나 취소되었습니다" {
		t.Fatalf("받기 비밀번호 틀림 상태 아님 (Y9): %+v", res)
	}
	pB.szAsked, pB.bAnswer = "master-pass-1", true
	if res := pB.syncer.Run(szBare, G_cszModePullVault); res.M_szState != G_cszStateOk {
		t.Fatalf("보관함 받기 실패: %+v", res)
	}
	if mapHosts := pB.local.data.M_mapHosts; len(mapHosts) != 1 {
		t.Fatalf("받은 뒤 서버가 없음: %d", len(mapHosts))
	}

	// Y6: 두 PC 가 같은 서버를 고친다. 나중에 동기화한 PC 의 값이 남는다.
	setHost(pA, "h-1", "web-A", "10.1.1.1")
	if res := pA.syncer.Run(szBare, G_cszModeAuto); res.M_szState != G_cszStateOk {
		t.Fatalf("A 동기화 실패: %+v", res)
	}
	setHost(pB, "h-1", "web-B", "10.1.1.1")
	if res := pB.syncer.Run(szBare, G_cszModeAuto); res.M_szState != G_cszStateOk {
		t.Fatalf("B 동기화 실패: %+v", res)
	}
	var mapHost map[string]any
	_ = json.Unmarshal(pB.local.data.M_mapHosts["h-1"], &mapHost)
	if mapHost["alias"] != "web-B" {
		t.Fatalf("동기화한 PC 의 값이 남아야 함 (Y6): %v", mapHost["alias"])
	}

	// Y7: A 가 서버를 지우고, B 가 같은 서버를 고친 뒤 동기화 → 삭제가 이긴다.
	delete(pA.local.data.M_mapHosts, "h-1")
	if res := pA.syncer.Run(szBare, G_cszModeAuto); res.M_szState != G_cszStateOk {
		t.Fatalf("A 삭제 동기화 실패: %+v", res)
	}
	setHost(pB, "h-1", "web-B2", "10.1.1.1")
	if res := pB.syncer.Run(szBare, G_cszModeAuto); res.M_szState != G_cszStateOk {
		t.Fatalf("B 수정 동기화 실패: %+v", res)
	}
	if _, bStill := pB.local.data.M_mapHosts["h-1"]; bStill {
		t.Fatal("삭제가 이겨야 함 (Y7)")
	}
}

// TestSyncPreChecks 는 저장소 주소 비밀번호(Y4)와 미설정, 잠긴 볼트 상태를 확인한다.
func TestSyncPreChecks(t *testing.T) {
	pFix := newPC(t, "master-pass-1")
	if res := pFix.syncer.Run("", G_cszModeAuto); res.M_szState != G_cszStateUnconfigured {
		t.Fatalf("미설정 상태 아님: %+v", res)
	}
	if res := pFix.syncer.Run("https://user:pw@example.com/repo.git", G_cszModeAuto); res.M_szState != G_cszStateError || !strings.Contains(res.M_szMessage, "비밀번호·토큰을 넣지 마세요") {
		t.Fatalf("주소 비밀번호 검사 실패 (Y4): %+v", res)
	}
	pFix.local.pVault.Lock()
	if res := pFix.syncer.Run("/some/path", G_cszModeAuto); res.M_szState != G_cszStateLocked {
		t.Fatalf("잠김 상태 아님: %+v", res)
	}
	if HasCredentialInUrl("git@github.com:user/repo.git") {
		t.Fatal("scp 형식 주소를 자격 증명으로 오인")
	}
}
