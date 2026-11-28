package app

import (
	"encoding/json"
	"strings"
	"testing"

	"dariterm/internal/sshconn"
)

// key.generate → 보관함에 키 계정이 생기고, key.publicKey 가 같은 공개 키를 돌려준다 (D162).
func TestKeyGenerateSavesCredAndPublicKeyMatches(t *testing.T) {
	pService, errNew := NewService(t.TempDir(), nil)
	if errNew != nil {
		t.Fatal(errNew)
	}
	pRouter := pService.Router()
	if _, errSetup := pRouter.Invoke("vault.setup", `["correct-horse-1"]`); errSetup != nil {
		t.Fatal(errSetup)
	}
	if _, errEmpty := pRouter.Invoke("key.generate", `[{"name":"  "}]`); errEmpty == nil {
		t.Fatal("이름이 비면 거절해야 한다")
	}
	szGen, errGen := pRouter.Invoke("key.generate", `[{"name":"my key","type":"ed25519","passphrase":"pp"}]`)
	if errGen != nil {
		t.Fatal(errGen)
	}
	var gen struct {
		Cred        struct{ Id, Name, Kind string }
		PublicKey   string
		Fingerprint string
	}
	if errJson := json.Unmarshal([]byte(szGen), &gen); errJson != nil {
		t.Fatal(errJson)
	}
	if gen.Cred.Kind != "key" || gen.Cred.Name != "my key" || !strings.HasPrefix(gen.PublicKey, "ssh-ed25519 ") || !strings.HasPrefix(gen.Fingerprint, "SHA256:") {
		t.Fatalf("key.generate = %s", szGen)
	}
	// 보관함의 개인 키는 암호가 걸린 OpenSSH 키이고, 키 암호도 함께 저장된다.
	szSecret, _ := pRouter.Invoke("cred.get", `["`+gen.Cred.Id+`"]`)
	var secret struct{ PrivateKey, Passphrase string }
	_ = json.Unmarshal([]byte(szSecret), &secret)
	if secret.Passphrase != "pp" {
		t.Fatalf("키 암호 저장: %s", szSecret)
	}
	if _, errParse := sshconn.ParseKeyWithPassphrase([]byte(secret.PrivateKey), []byte("pp")); errParse != nil {
		t.Fatalf("저장된 키를 읽지 못함: %v", errParse)
	}
	szPub, errPub := pRouter.Invoke("key.publicKey", `["`+gen.Cred.Id+`"]`)
	if errPub != nil || !strings.Contains(szPub, gen.Fingerprint) || !strings.Contains(szPub, strings.Fields(gen.PublicKey)[1]) {
		t.Fatalf("key.publicKey = %s %v", szPub, errPub)
	}
	// 비밀번호 계정은 공개 키가 없다.
	szPw, _ := pRouter.Invoke("cred.save", `[{"name":"pw","kind":"password","password":"x"}]`)
	var pw struct{ Id string }
	_ = json.Unmarshal([]byte(szPw), &pw)
	if _, errNotKey := pRouter.Invoke("key.publicKey", `["`+pw.Id+`"]`); errNotKey == nil {
		t.Fatal("비밀번호 계정은 공개 키를 만들 수 없어야 한다")
	}
	// 없는 세션에는 등록할 수 없다.
	if _, errInstall := pRouter.Invoke("key.install", `["nope","`+gen.Cred.Id+`"]`); errInstall == nil {
		t.Fatal("없는 세션")
	}
}
