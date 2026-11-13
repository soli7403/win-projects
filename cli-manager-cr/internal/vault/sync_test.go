package vault

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestEncryptDataRoundTrip 은 임의 데이터 암호화와 AAD 검사를 확인한다 (data/vault.md §8).
func TestEncryptDataRoundTrip(t *testing.T) {
	pv, errOpen := Open(filepath.Join(t.TempDir(), "vault.json"))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	if _, errLocked := pv.EncryptData("termgo-sync", []byte("x")); errLocked == nil {
		t.Fatal("미초기화 볼트 암호화는 오류여야 함")
	}
	if _, errSetup := pv.Setup("correct-horse-1"); errSetup != nil {
		t.Fatal(errSetup)
	}
	blob, errEnc := pv.EncryptData(G_cszSyncPayloadAad, []byte(`{"hosts":{}}`))
	if errEnc != nil {
		t.Fatal(errEnc)
	}
	bPlain, errDec := pv.DecryptData(G_cszSyncPayloadAad, blob)
	if errDec != nil || string(bPlain) != `{"hosts":{}}` {
		t.Fatalf("복호화 불일치: %q %v", bPlain, errDec)
	}
	if _, errWrongAad := pv.DecryptData("other-aad", blob); errWrongAad == nil {
		t.Fatal("AAD 가 다르면 복호화 실패해야 함")
	}
	bHeader, errHeader := pv.HeaderJSON()
	if errHeader != nil || strings.Contains(string(bHeader), `"items":[{`) || !strings.Contains(string(bHeader), `"wrapped"`) {
		t.Fatalf("헤더 형식 불일치: %s %v", bHeader, errHeader)
	}
}

// TestAdoptRemote 는 원격 볼트 받기: 틀린 비밀번호는 바꾸지 않고, 맞으면 헤더·항목으로 교체해 연다 (data/vault.md §8, acceptance 04 Y8, Y9).
func TestAdoptRemote(t *testing.T) {
	dir := t.TempDir()
	pRemote, _ := Open(filepath.Join(dir, "remote.json"))
	if _, errSetup := pRemote.Setup("remote-pass-1"); errSetup != nil {
		t.Fatal(errSetup)
	}
	if _, errSave := pRemote.Save(SaveInput{M_szName: "bastion", M_szKind: "password", M_pszPassword: strPtr("pw1")}); errSave != nil {
		t.Fatal(errSave)
	}
	bHeader, _ := pRemote.HeaderJSON()
	slRemoteItems, _ := pRemote.Items()

	pLocal, _ := Open(filepath.Join(dir, "local.json"))
	if _, errSetup := pLocal.Setup("local-pass-22"); errSetup != nil {
		t.Fatal(errSetup)
	}
	bOk, errWrong := pLocal.AdoptRemote(bHeader, slRemoteItems, "wrong-pass-1")
	if errWrong != nil || bOk {
		t.Fatalf("틀린 비밀번호로 받기가 성공함: %v %v", bOk, errWrong)
	}
	if len(pLocal.List()) != 0 {
		t.Fatal("실패한 받기가 볼트를 바꿈")
	}

	bOk, errAdopt := pLocal.AdoptRemote(bHeader, slRemoteItems, "remote-pass-1")
	if errAdopt != nil || !bOk {
		t.Fatalf("맞는 비밀번호로 받기 실패: %v %v", bOk, errAdopt)
	}
	if !pLocal.Status().M_bUnlocked || len(pLocal.List()) != 1 {
		t.Fatal("받은 뒤 볼트가 열려 있지 않거나 항목이 없음")
	}
	secret, errGet := pLocal.Get(pLocal.List()[0].M_szId)
	if errGet != nil || secret == nil || secret.M_szPassword != "pw1" {
		t.Fatalf("받은 항목 복호화 실패: %v", errGet)
	}
}

// TestReplaceItemsAndUnwrap 은 항목 교체와 헤더 키 풀기를 확인한다.
func TestReplaceItemsAndUnwrap(t *testing.T) {
	dir := t.TempDir()
	pv, _ := Open(filepath.Join(dir, "v.json"))
	if _, errSetup := pv.Setup("correct-horse-1"); errSetup != nil {
		t.Fatal(errSetup)
	}
	if _, errSave := pv.Save(SaveInput{M_szName: "a", M_szKind: "password"}); errSave != nil {
		t.Fatal(errSave)
	}
	slItems, errItems := pv.Items()
	if errItems != nil {
		t.Fatal(errItems)
	}
	if errReplace := pv.ReplaceItems(nil); errReplace != nil {
		t.Fatal(errReplace)
	}
	if len(pv.List()) != 0 {
		t.Fatal("항목 교체 후 목록이 비어 있지 않음")
	}
	if errReplace := pv.ReplaceItems(slItems); errReplace != nil {
		t.Fatal(errReplace)
	}
	bHeader, _ := pv.HeaderJSON()
	bKey, bOk := UnwrapHeaderKey(bHeader, "correct-horse-1")
	if !bOk || len(bKey) != 32 {
		t.Fatal("헤더에서 볼트 키를 풀지 못함")
	}
	if _, bWrong := UnwrapHeaderKey(bHeader, "nope-nope-nope"); bWrong {
		t.Fatal("틀린 비밀번호로 헤더가 풀리면 안 됨")
	}
}
