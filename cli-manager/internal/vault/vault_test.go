package vault

import "testing"

func ptr(s string) *string { return &s }

func TestSetupUnlockSaveGet(t *testing.T) {
	szDir := t.TempDir()
	v := Open(szDir)
	m, err := v.Setup("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Save(Input{Name: "n", Kind: "password", Username: ptr("root"), Password: ptr("pw")}); err != nil {
		t.Fatal(err)
	}
	meta := v.List()
	if len(meta) != 1 {
		t.Fatalf("list = %v", meta)
	}

	v2 := Open(szDir)
	if v2.Unlock("wrong password") {
		t.Fatal("wrong password unlocked")
	}
	if !v2.Unlock("correct horse") {
		t.Fatal("unlock failed")
	}
	sec, err := v2.Get(meta[0].ID)
	if err != nil || sec == nil || sec.Password != "pw" || sec.Username != "root" {
		t.Fatalf("get = %+v, %v", sec, err)
	}
	// 부분 갱신: password 만 바꾸면 username 유지
	if _, err := v2.Save(Input{ID: meta[0].ID, Name: "n", Kind: "password", Password: ptr("pw2")}); err != nil {
		t.Fatal(err)
	}
	if sec, _ = v2.Get(meta[0].ID); sec.Password != "pw2" || sec.Username != "root" {
		t.Fatalf("partial update = %+v", sec)
	}
	if !v2.ChangePassword("correct horse", "new password") {
		t.Fatal("change password")
	}
	v3 := Open(szDir)
	if !v3.Unlock("new password") {
		t.Fatal("unlock after change")
	}
	if !v3.Recover(m["recoveryCode"], "third password") || !Open(szDir).Unlock("third password") {
		t.Fatal("recover")
	}
	v3.Lock()
	if _, err := v3.Get(meta[0].ID); err == nil {
		t.Fatal("get on locked vault must fail")
	}
}
