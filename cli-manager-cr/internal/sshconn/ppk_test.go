package sshconn

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/ssh"
)

// 시험용 PPK 만들기. 읽는 쪽(ppk.go)과 따로 짠 쓰는 쪽이다. 둘 다 같은 형식 이해에서 나왔으므로
// 이 시험은 "서로 맞는다"까지만 보증한다. PuTTYgen 이 만든 실제 파일로의 확인은 수동 검증 항목이다
// (docs/import-plans/2026-11-23-ppk-keys.md).

func ppkString(bParts ...[]byte) []byte {
	var bOut []byte
	for _, bPart := range bParts {
		var arrLength [4]byte
		binary.BigEndian.PutUint32(arrLength[:], uint32(len(bPart)))
		bOut = append(append(bOut, arrLength[:]...), bPart...)
	}
	return bOut
}

// ppkMpint 는 SSH mpint 내용(부호 비트가 서면 앞에 0 을 붙인 big-endian)을 만든다.
func ppkMpint(pValue *big.Int) []byte {
	bOut := pValue.Bytes()
	if len(bOut) > 0 && bOut[0]&0x80 != 0 {
		bOut = append([]byte{0}, bOut...)
	}
	return bOut
}

// ppkBlobsForTest 는 키의 공개·개인 부분을 PPK 형식으로 만든다.
func ppkBlobsForTest(t *testing.T, anyKey any) (string, []byte, []byte) {
	t.Helper()
	switch key := anyKey.(type) {
	case *rsa.PrivateKey:
		pIqmp := new(big.Int).ModInverse(key.Primes[1], key.Primes[0])
		return "ssh-rsa",
			ppkString([]byte("ssh-rsa"), ppkMpint(big.NewInt(int64(key.E))), ppkMpint(key.N)),
			ppkString(ppkMpint(key.D), ppkMpint(key.Primes[0]), ppkMpint(key.Primes[1]), ppkMpint(pIqmp))
	case ed25519.PrivateKey:
		return "ssh-ed25519",
			ppkString([]byte("ssh-ed25519"), key.Public().(ed25519.PublicKey)),
			ppkString(key.Seed())
	case *ecdsa.PrivateKey:
		szCurve := map[int]string{256: "nistp256", 384: "nistp384", 521: "nistp521"}[key.Curve.Params().BitSize]
		bPoint, errPoint := key.PublicKey.Bytes()
		if errPoint != nil {
			t.Fatal(errPoint)
		}
		bD, errD := key.Bytes()
		if errD != nil {
			t.Fatal(errD)
		}
		return "ecdsa-sha2-" + szCurve,
			ppkString([]byte("ecdsa-sha2-"+szCurve), []byte(szCurve), bPoint),
			ppkString(ppkMpint(new(big.Int).SetBytes(bD)))
	}
	t.Fatalf("모르는 키 종류 %T", anyKey)
	return "", nil, nil
}

func ppkWrap64(bBlob []byte) []string {
	szAll := base64.StdEncoding.EncodeToString(bBlob)
	var slLines []string
	for len(szAll) > 64 {
		slLines = append(slLines, szAll[:64])
		szAll = szAll[64:]
	}
	return append(slLines, szAll)
}

// encodePpkForTest 는 PPK 형식 2 또는 3 파일을 만든다. szPassphrase 가 비면 암호 없는 키다.
func encodePpkForTest(t *testing.T, iVersion int, anyKey any, szComment string, szPassphrase string, szNewline string) []byte {
	t.Helper()
	szAlgorithm, bPublic, bPrivate := ppkBlobsForTest(t, anyKey)
	szEncryption := "none"
	var slKdf []string
	var bMacKey, bCipherKey, bIv []byte
	if szPassphrase != "" {
		szEncryption = "aes256-cbc"
		for len(bPrivate)%aes.BlockSize != 0 {
			bPrivate = append(bPrivate, byte(len(bPrivate))) // 채움 값은 아무 값이나 된다
		}
	}
	fnHash := sha256.New
	if iVersion == 2 {
		fnHash = sha1.New
		seed := sha1.New()
		seed.Write([]byte("putty-private-key-file-mac-key"))
		seed.Write([]byte(szPassphrase))
		bMacKey = seed.Sum(nil)
		if szPassphrase != "" {
			h0, h1 := sha1.Sum(append([]byte{0, 0, 0, 0}, szPassphrase...)), sha1.Sum(append([]byte{0, 0, 0, 1}, szPassphrase...))
			bCipherKey = append(h0[:], h1[:]...)[:32]
			bIv = make([]byte, 16)
		}
	} else if szPassphrase != "" {
		bSalt := make([]byte, 16)
		_, _ = rand.Read(bSalt)
		bDerived := argon2.IDKey([]byte(szPassphrase), bSalt, 3, 1024, 1, 80)
		bCipherKey, bIv, bMacKey = bDerived[:32], bDerived[32:48], bDerived[48:]
		slKdf = []string{"Key-Derivation: Argon2id", "Argon2-Memory: 1024", "Argon2-Passes: 3", "Argon2-Parallelism: 1", "Argon2-Salt: " + hex.EncodeToString(bSalt)}
	}
	mac := hmac.New(fnHash, bMacKey)
	mac.Write(ppkString([]byte(szAlgorithm), []byte(szEncryption), []byte(szComment), bPublic, bPrivate))
	bStored := bPrivate
	if szPassphrase != "" {
		block, _ := aes.NewCipher(bCipherKey)
		bStored = make([]byte, len(bPrivate))
		cipher.NewCBCEncrypter(block, bIv).CryptBlocks(bStored, bPrivate)
	}
	slPub, slPriv := ppkWrap64(bPublic), ppkWrap64(bStored)
	slLines := []string{
		fmt.Sprintf("PuTTY-User-Key-File-%d: %s", iVersion, szAlgorithm),
		"Encryption: " + szEncryption,
		"Comment: " + szComment,
		fmt.Sprintf("Public-Lines: %d", len(slPub)),
	}
	slLines = append(slLines, slPub...)
	slLines = append(slLines, slKdf...)
	slLines = append(slLines, fmt.Sprintf("Private-Lines: %d", len(slPriv)))
	slLines = append(slLines, slPriv...)
	slLines = append(slLines, "Private-MAC: "+hex.EncodeToString(mac.Sum(nil)), "")
	return []byte(strings.Join(slLines, szNewline))
}

func testKeysForPpk(t *testing.T) map[string]any {
	t.Helper()
	pRsa, errRsa := rsa.GenerateKey(rand.Reader, 2048)
	if errRsa != nil {
		t.Fatal(errRsa)
	}
	_, edKey, _ := ed25519.GenerateKey(rand.Reader)
	p256, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	p384, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	p521, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	return map[string]any{"rsa": pRsa, "ed25519": edKey, "p256": p256, "p384": p384, "p521": p521}
}

// PPK 형식 2·3, 암호 유무, 키 종류 5가지를 모두 읽고, 읽은 키의 공개 키가 원래 키와 같다 (D106).
func TestParsePpkAllKinds(t *testing.T) {
	for szName, anyKey := range testKeysForPpk(t) {
		pWant, errWant := ssh.NewSignerFromKey(anyKey)
		if errWant != nil {
			t.Fatal(errWant)
		}
		for _, iVersion := range []int{2, 3} {
			for _, szPass := range []string{"", "correct horse"} {
				bFile := encodePpkForTest(t, iVersion, anyKey, "키 주석 "+szName, szPass, "\r\n")
				var pGot ssh.Signer
				var errGot error
				if szPass == "" {
					pGot, errGot = ParseKey(bFile)
				} else {
					pGot, errGot = ParseKeyWithPassphrase(bFile, []byte(szPass))
				}
				if errGot != nil {
					t.Fatalf("%s v%d 암호=%q: %v", szName, iVersion, szPass, errGot)
				}
				if !bytes.Equal(pGot.PublicKey().Marshal(), pWant.PublicKey().Marshal()) {
					t.Fatalf("%s v%d: 공개 키가 다름", szName, iVersion)
				}
				// 읽은 키로 서명하고 원래 공개 키로 확인한다.
				sig, errSign := pGot.Sign(rand.Reader, []byte("hello"))
				if errSign != nil || pWant.PublicKey().Verify([]byte("hello"), sig) != nil {
					t.Fatalf("%s v%d: 서명 확인 실패 (%v)", szName, iVersion, errSign)
				}
			}
		}
	}
}

// 암호가 걸린 PPK 를 암호 없이 읽으면 "암호 필요", 틀린 암호면 MAC 오류다 (keyLoader 가 다시 묻는 근거).
func TestParsePpkPassphraseErrors(t *testing.T) {
	_, edKey, _ := ed25519.GenerateKey(rand.Reader)
	for _, iVersion := range []int{2, 3} {
		bFile := encodePpkForTest(t, iVersion, edKey, "c", "secret", "\n")
		_, errNone := ParseKey(bFile)
		var errMissing *ssh.PassphraseMissingError
		if !errors.As(errNone, &errMissing) {
			t.Fatalf("v%d: 암호 필요 오류여야 함: %v", iVersion, errNone)
		}
		if _, errWrong := ParseKeyWithPassphrase(bFile, []byte("nope")); !errors.Is(errWrong, ErrPpkMac) {
			t.Fatalf("v%d: 틀린 암호는 MAC 오류여야 함: %v", iVersion, errWrong)
		}
	}
}

// 내용을 한 글자 바꾸면 MAC 이 맞지 않는다. PPK 가 아닌 글자·모르는 형식 번호는 형식 오류다.
func TestParsePpkTamperAndFormat(t *testing.T) {
	_, edKey, _ := ed25519.GenerateKey(rand.Reader)
	bFile := encodePpkForTest(t, 3, edKey, "comment", "", "\n")
	bTampered := bytes.Replace(bFile, []byte("Comment: comment"), []byte("Comment: Comment"), 1)
	if _, errTamper := ParseKey(bTampered); !errors.Is(errTamper, ErrPpkMac) {
		t.Fatalf("주석을 바꾸면 MAC 오류여야 함: %v", errTamper)
	}
	if _, errShort := ParseKey([]byte("PuTTY-User-Key-File-3: ssh-ed25519\nEncryption: none\n")); !errors.Is(errShort, ErrPpkFormat) {
		t.Fatalf("잘린 파일은 형식 오류여야 함: %v", errShort)
	}
	bV9 := bytes.Replace(bFile, []byte("Key-File-3"), []byte("Key-File-9"), 1)
	if _, errV9 := ParseKey(bV9); errV9 == nil || !strings.Contains(errV9.Error(), "version 9") {
		t.Fatalf("모르는 형식 번호: %v", errV9)
	}
	if IsPpk([]byte("-----BEGIN OPENSSH PRIVATE KEY-----")) || !IsPpk(append([]byte("\xef\xbb\xbf\r\n"), bFile...)) {
		t.Fatal("IsPpk 판정이 틀림")
	}
}

// OpenSSH 형식 키는 전과 같이 x/crypto/ssh 가 읽는다.
func TestParseKeyOpenSSHStillWorks(t *testing.T) {
	_, edKey, _ := ed25519.GenerateKey(rand.Reader)
	pBlock, errMarshal := ssh.MarshalPrivateKey(edKey, "")
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	bPem := pemEncodeForTest(pBlock.Type, pBlock.Bytes)
	if _, errParse := ParseKey(bPem); errParse != nil {
		t.Fatalf("OpenSSH 키: %v", errParse)
	}
}

func pemEncodeForTest(szType string, bBody []byte) []byte {
	return []byte("-----BEGIN " + szType + "-----\n" + strings.Join(ppkWrap64(bBody), "\n") + "\n-----END " + szType + "-----\n")
}

// DARITERM_PPK_DUMP_DIR 을 주면 시험용 PPK 파일들을 그 폴더에 쓴다 (다른 구현으로 읽어 보는 교차 확인용).
func TestDumpPpkFixtures(t *testing.T) {
	szDir := os.Getenv("DARITERM_PPK_DUMP_DIR")
	if szDir == "" {
		t.Skip("DARITERM_PPK_DUMP_DIR 없음")
	}
	for szName, anyKey := range testKeysForPpk(t) {
		pSigner, _ := ssh.NewSignerFromKey(anyKey)
		_ = os.WriteFile(filepath.Join(szDir, szName+".pub"), ssh.MarshalAuthorizedKey(pSigner.PublicKey()), 0o600)
		for _, iVersion := range []int{2, 3} {
			for szTag, szPass := range map[string]string{"plain": "", "enc": "correct horse"} {
				szFile := fmt.Sprintf("%s-v%d-%s.ppk", szName, iVersion, szTag)
				if errWrite := os.WriteFile(filepath.Join(szDir, szFile), encodePpkForTest(t, iVersion, anyKey, "dump "+szName, szPass, "\r\n"), 0o600); errWrite != nil {
					t.Fatal(errWrite)
				}
			}
		}
	}
}
