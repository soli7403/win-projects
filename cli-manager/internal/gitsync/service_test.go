package gitsync

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"termgo/internal/store"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

func mkService(t *testing.T, szRepo string, copyVaultFrom string, setupPW string) *Service {
	t.Helper()
	szDir := t.TempDir()
	if copyVaultFrom != "" {
		arr, err := os.ReadFile(filepath.Join(copyVaultFrom, "vault.json"))
		if err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(szDir, "vault.json"), arr, 0o600)
	}
	st, err := store.Open(szDir)
	if err != nil {
		t.Fatal(err)
	}
	st.Patch(map[string]json.RawMessage{"settings": json.RawMessage(`{"syncRepo":` + mustJSON(szRepo) + `}`)})
	v := vault.Open(szDir)
	if setupPW != "" {
		if _, err := v.Setup(setupPW); err != nil {
			t.Fatal(err)
		}
	}
	return &Service{Dir: szDir, Store: st, Vault: v, UI: ui.NewBridge(func(string, ...any) {})}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func hostIDs(s *Service) []string { return SortedIDs(toCollection(s.Store.Raw("hosts"))) }

func setHosts(s *Service, hosts ...string) {
	arr := make([]json.RawMessage, 0, len(hosts))
	for _, h := range hosts {
		arr = append(arr, json.RawMessage(h))
	}
	raw, _ := json.Marshal(arr)
	s.Store.Patch(map[string]json.RawMessage{"hosts": raw})
}

func TestGitSyncTwoPCs(t *testing.T) {
	if !GitAvailable() {
		t.Skip("git 없음")
	}
	szBare := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", szBare).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}

	a := mkService(t, szBare, "", "master password")
	setHosts(a, `{"id":"h1","alias":"서버1","host":"10.0.0.1"}`, `{"id":"h2","alias":"서버2","host":"10.0.0.2"}`)
	pw := "비밀번호"
	user := "root"
	if _, err := a.Vault.Save(vault.Input{ID: "c1", Name: "내 계정", Kind: "password", Username: &user, Password: &pw}); err != nil {
		t.Fatal(err)
	}
	if st := a.Sync(ModeAuto); st.State != StOK {
		t.Fatalf("A first sync: %+v", st)
	}

	// 저장소에는 파일 하나만, 서버 주소·계정명은 평문으로 없다
	clone := a.repoDir()
	ents, _ := os.ReadDir(clone)
	var files []string
	for _, e := range ents {
		if e.Name() != ".git" {
			files = append(files, e.Name())
		}
	}
	if len(files) != 1 || files[0] != szFileName {
		t.Fatalf("repo files = %v", files)
	}
	arr, _ := os.ReadFile(filepath.Join(clone, szFileName))
	for _, leak := range []string{"10.0.0.1", "서버1", "내 계정", "root", "비밀번호"} {
		if strings.Contains(string(arr), leak) {
			t.Fatalf("sync file leaks %q in plaintext", leak)
		}
	}

	// B: 같은 마스터 비밀번호(같은 볼트)로 열어 받는다
	b := mkService(t, szBare, a.Dir, "")
	if !b.Vault.Unlock("master password") {
		t.Fatal("B unlock")
	}
	if st := b.Sync(ModeAuto); st.State != StOK || !st.Changed {
		t.Fatalf("B sync: %+v", st)
	}
	if got := hostIDs(b); len(got) != 2 {
		t.Fatalf("B hosts = %v", got)
	}
	if sec, err := b.Vault.Get("c1"); err != nil || sec == nil || sec.Password != pw {
		t.Fatalf("B credential = %+v %v", sec, err)
	}

	// B 에서 h2 삭제 + h3 추가 → A 에 반영되어 h2 는 삭제된다 (한쪽 삭제는 다른 PC 에도 삭제)
	setHosts(b, `{"id":"h1","alias":"서버1","host":"10.0.0.1"}`, `{"id":"h3","alias":"서버3","host":"10.0.0.3"}`)
	if st := b.Sync(ModeAuto); st.State != StOK {
		t.Fatalf("B sync2: %+v", st)
	}
	// 그 사이 A 에서 h1 을 수정 (양쪽 수정이 겹치지 않음)
	setHosts(a, `{"id":"h1","alias":"서버1-수정","host":"10.0.0.1"}`, `{"id":"h2","alias":"서버2","host":"10.0.0.2"}`)
	if st := a.Sync(ModeAuto); st.State != StOK || !st.Changed {
		t.Fatalf("A sync2: %+v", st)
	}
	ids := hostIDs(a)
	if strings.Join(ids, ",") != "h1,h3" {
		t.Fatalf("A hosts after merge = %v", ids)
	}
	var h1 struct{ Alias string }
	_ = json.Unmarshal(toCollection(a.Store.Raw("hosts"))["h1"], &h1)
	if h1.Alias != "서버1-수정" {
		t.Fatalf("local edit lost: %q", h1.Alias)
	}

	// C: 다른 볼트 — 저장소 볼트와 달라서 사용자가 고르도록 vault-mismatch
	c := mkService(t, szBare, "", "another master")
	if st := c.Sync(ModeAuto); st.State != StVaultMismatch {
		t.Fatalf("C must report vault mismatch: %+v", st)
	}
	// "이 PC 볼트로 덮어쓰기"
	if st := c.Sync(ModeOverwrite); st.State != StOK {
		t.Fatalf("C overwrite: %+v", st)
	}
	// 이제 A(원래 볼트)는 저장소 볼트가 달라 mismatch 가 되고, "저장소 볼트 받기" 는 C 의 비밀번호로 연다
	if st := a.Sync(ModeAuto); st.State != StVaultMismatch {
		t.Fatalf("A after C overwrite: %+v", st)
	}
	a.UI = ui.NewBridge(func(szCh string, args ...any) {})
	var br *ui.Bridge
	br = ui.NewBridge(func(szCh string, args ...any) {
		if szCh == "ui:request" {
			go br.Respond(args[0].(map[string]any)["reqId"].(string), `{"password":"another master","save":false}`)
		}
	})
	a.UI = br
	if st := a.Sync(ModePullVault); st.State != StOK {
		t.Fatalf("A pull vault: %+v", st)
	}
	if !vault.Open(a.Dir).Unlock("another master") {
		t.Fatal("A must now open with the repo vault's master password")
	}
}
