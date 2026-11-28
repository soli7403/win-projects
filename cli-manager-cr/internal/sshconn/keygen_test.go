package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateKeyEd25519ParsesAndMatchesPublic(t *testing.T) {
	key, errGen := GenerateKey(G_cszKeyEd25519, 0, "kai@pc", "")
	if errGen != nil {
		t.Fatal(errGen)
	}
	if !strings.HasPrefix(key.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Fatalf("개인 키 형식: %q", key.PrivateKey[:40])
	}
	if !strings.HasPrefix(key.PublicKey, "ssh-ed25519 ") || !strings.HasSuffix(key.PublicKey, " kai@pc") {
		t.Fatalf("공개 키 줄: %q", key.PublicKey)
	}
	signer, errParse := ParseKey([]byte(key.PrivateKey))
	if errParse != nil {
		t.Fatal(errParse)
	}
	if got := ssh.FingerprintSHA256(signer.PublicKey()); got != key.Fingerprint {
		t.Fatalf("지문 %s != %s", got, key.Fingerprint)
	}
	szLine, szFp, errPub := PublicKeyOf(key.PrivateKey, "", "kai@pc")
	if errPub != nil || szLine != key.PublicKey || szFp != key.Fingerprint {
		t.Fatalf("PublicKeyOf = %q %q %v", szLine, szFp, errPub)
	}
}

func TestGenerateKeyRsaWithPassphrase(t *testing.T) {
	key, errGen := GenerateKey(G_cszKeyRsa, 3072, "", "s3cret")
	if errGen != nil {
		t.Fatal(errGen)
	}
	if !strings.HasPrefix(key.PublicKey, "ssh-rsa ") || strings.Count(key.PublicKey, " ") != 1 {
		t.Fatalf("주석 없는 RSA 공개 키 줄: %q", key.PublicKey)
	}
	var pMissing *ssh.PassphraseMissingError
	if _, errParse := ParseKey([]byte(key.PrivateKey)); !errors.As(errParse, &pMissing) {
		t.Fatalf("암호 없이 읽혀서는 안 된다: %v", errParse)
	}
	signer, errParse := ParseKeyWithPassphrase([]byte(key.PrivateKey), []byte("s3cret"))
	if errParse != nil {
		t.Fatal(errParse)
	}
	if signer.PublicKey().(ssh.CryptoPublicKey).CryptoPublicKey() == nil {
		t.Fatal("공개 키 없음")
	}
	// 암호가 걸린 OpenSSH 키도 암호 없이 공개 키를 읽는다.
	szLine, szFp, errPub := PublicKeyOf(key.PrivateKey, "", "")
	if errPub != nil || szLine != key.PublicKey || szFp != key.Fingerprint {
		t.Fatalf("PublicKeyOf(암호 없이) = %q %q %v", szLine, szFp, errPub)
	}
}

func TestGenerateKeyRejectsBadInput(t *testing.T) {
	if _, errGen := GenerateKey("dsa", 0, "", ""); !errors.Is(errGen, ErrKeyType) {
		t.Fatalf("모르는 종류: %v", errGen)
	}
	if _, errGen := GenerateKey(G_cszKeyRsa, 2048, "", ""); !errors.Is(errGen, ErrKeyBits) {
		t.Fatalf("2048비트: %v", errGen)
	}
}

func TestAuthorizedLineCleansComment(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	sshPub, _ := ssh.NewPublicKey(pub)
	if got := AuthorizedLine(sshPub, "  a\nb  "); !strings.HasSuffix(got, " a b") || strings.Contains(got, "\n") {
		t.Fatalf("주석 정리: %q", got)
	}
}

// startExecServer 는 exec 요청을 이 PC 의 sh 로 돌리는 시험용 SSH 서버다. HOME 은 szHome 이다.
func startExecServer(t *testing.T, szHome string) (*ssh.Client, func()) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	go func() {
		for {
			c, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				_, chans, reqs, errHs := ssh.NewServerConn(c, cfg)
				if errHs != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for newCh := range chans {
					ch, chReqs, errAcc := newCh.Accept()
					if errAcc != nil {
						continue
					}
					go func() {
						defer ch.Close()
						for req := range chReqs {
							if req.Type != "exec" {
								_ = req.Reply(false, nil)
								continue
							}
							var payload struct{ Command string }
							_ = ssh.Unmarshal(req.Payload, &payload)
							_ = req.Reply(true, nil)
							cmd := exec.Command("sh", "-c", payload.Command)
							cmd.Env = append(os.Environ(), "HOME="+szHome)
							cmd.Stdin, cmd.Stdout, cmd.Stderr = ch, ch, ch.Stderr()
							iCode := 0
							if errRun := cmd.Run(); errRun != nil {
								iCode = 1
							}
							_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(iCode)}))
							return
						}
					}()
				}
			}()
		}
	}()
	client, errDial := ssh.Dial("tcp", ln.Addr().String(), &ssh.ClientConfig{User: "u", HostKeyCallback: ssh.InsecureIgnoreHostKey()})
	if errDial != nil {
		_ = ln.Close()
		t.Fatal(errDial)
	}
	return client, func() { _ = client.Close(); _ = ln.Close() }
}

func TestInstallPublicKeyAddsOnceAndKeepsExisting(t *testing.T) {
	if _, errLook := exec.LookPath("sh"); errLook != nil {
		t.Skip("sh 가 없다")
	}
	szHome := filepath.ToSlash(t.TempDir())
	client, stop := startExecServer(t, szHome)
	defer stop()
	szAuth := filepath.Join(szHome, ".ssh", "authorized_keys")
	if errMk := os.MkdirAll(filepath.Dir(szAuth), 0o700); errMk != nil {
		t.Fatal(errMk)
	}
	// 끝에 줄바꿈이 없는 기존 파일: 새 키는 다음 줄에 들어가야 한다.
	if errWrite := os.WriteFile(szAuth, []byte("ssh-ed25519 AAAAexisting old@pc"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	key, _ := GenerateKey(G_cszKeyEd25519, 0, "kai@pc", "")
	bExists, errInstall := InstallPublicKey(client, key.PublicKey)
	if errInstall != nil || bExists {
		t.Fatalf("첫 등록 = %v %v", bExists, errInstall)
	}
	// 주석만 다른 같은 키는 이미 있는 것으로 본다.
	szOther := strings.TrimSuffix(key.PublicKey, " kai@pc") + " other"
	bExists, errInstall = InstallPublicKey(client, szOther)
	if errInstall != nil || !bExists {
		t.Fatalf("두 번째 등록 = %v %v", bExists, errInstall)
	}
	bData, _ := os.ReadFile(szAuth)
	slLines := strings.Split(strings.TrimRight(string(bData), "\n"), "\n")
	if len(slLines) != 2 || slLines[0] != "ssh-ed25519 AAAAexisting old@pc" || slLines[1] != key.PublicKey {
		t.Fatalf("authorized_keys = %q", string(bData))
	}
}

func TestInstallPublicKeyCreatesFileAndRejectsGarbage(t *testing.T) {
	if _, errLook := exec.LookPath("sh"); errLook != nil {
		t.Skip("sh 가 없다")
	}
	szHome := filepath.ToSlash(t.TempDir())
	client, stop := startExecServer(t, szHome)
	defer stop()
	if _, errInstall := InstallPublicKey(client, "not a key"); errInstall == nil {
		t.Fatal("공개 키가 아닌 줄은 보내지 않아야 한다")
	}
	key, _ := GenerateKey(G_cszKeyEd25519, 0, "", "")
	if bExists, errInstall := InstallPublicKey(client, key.PublicKey); errInstall != nil || bExists {
		t.Fatalf("새 파일 등록 = %v %v", bExists, errInstall)
	}
	bData, errRead := os.ReadFile(filepath.Join(szHome, ".ssh", "authorized_keys"))
	if errRead != nil || string(bData) != key.PublicKey+"\n" {
		t.Fatalf("새 파일 = %q %v", string(bData), errRead)
	}
}

func TestInstallPublicKeyReportsServerError(t *testing.T) {
	if _, errLook := exec.LookPath("sh"); errLook != nil {
		t.Skip("sh 가 없다")
	}
	// HOME 의 .ssh 자리에 파일이 있으면 mkdir 이 실패한다: 까닭과 함께 오류여야 한다.
	szHome := filepath.ToSlash(t.TempDir())
	if errWrite := os.WriteFile(filepath.Join(szHome, ".ssh"), []byte("x"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	client, stop := startExecServer(t, szHome)
	defer stop()
	key, _ := GenerateKey(G_cszKeyEd25519, 0, "", "")
	if _, errInstall := InstallPublicKey(client, key.PublicKey); errInstall == nil {
		t.Fatal("실패를 알려야 한다")
	}
}
