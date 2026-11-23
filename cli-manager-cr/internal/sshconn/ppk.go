package sshconn

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/ssh"
)

// PuTTY 개인 키 파일(PPK) 읽기 (protocols.md §2.2 1번, D106).
// 형식 2(SHA-1 기반)와 형식 3(Argon2 + HMAC-SHA-256)을 읽는다. 형식은 PuTTY 공개 문서(부록 C "PPK file format")를 따른다.

// g_cszPpkHeader 는 PPK 파일 첫 줄의 머리다. 뒤에 형식 번호(2 또는 3)가 붙는다.
const g_cszPpkHeader = "PuTTY-User-Key-File-"

// g_ciPpkArgonMemoryMaxKiB 는 받아들이는 Argon2 메모리 상한(KiB)이다. 조작된 파일이 메모리를 다 쓰게 하지 못하도록 막는다.
const g_ciPpkArgonMemoryMaxKiB = 1024 * 1024

// g_ciPpkArgonPassesMax 는 받아들이는 Argon2 반복 횟수 상한이다.
const g_ciPpkArgonPassesMax = 1000

// PPK 읽기 오류.
var (
	// ErrPpkMac 은 무결성 검사(MAC)가 맞지 않을 때의 오류다. 키 암호가 틀렸거나 파일이 손상됐다.
	ErrPpkMac = errors.New("ppk: wrong passphrase or corrupted key file (MAC mismatch)")
	// ErrPpkFormat 은 파일 모양이 PPK 가 아닐 때의 오류다.
	ErrPpkFormat = errors.New("ppk: malformed key file")
)

// IsPpk 는 키 내용이 PuTTY PPK 형식인지 본다.
func IsPpk(bKey []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(bKey, " \t\r\n\uFEFF"), []byte(g_cszPpkHeader))
}

// ParseKey 는 개인 키를 읽는다. PPK 면 직접 읽고, 그 밖(OpenSSH·PEM)은 x/crypto/ssh 에 맡긴다.
// 암호가 걸린 키면 *ssh.PassphraseMissingError 를 돌려준다.
func ParseKey(bKey []byte) (ssh.Signer, error) {
	if IsPpk(bKey) {
		return parsePpk(bKey, nil)
	}
	return ssh.ParsePrivateKey(bKey)
}

// ParseKeyWithPassphrase 는 암호가 걸린 개인 키를 읽는다.
func ParseKeyWithPassphrase(bKey []byte, bPassphrase []byte) (ssh.Signer, error) {
	if IsPpk(bKey) {
		return parsePpk(bKey, bPassphrase)
	}
	return ssh.ParsePrivateKeyWithPassphrase(bKey, bPassphrase)
}

// ppkFile 은 PPK 파일을 줄 단위로 읽은 결과다.
type ppkFile struct {
	iVersion     int
	szAlgorithm  string
	szEncryption string
	szComment    string
	bPublic      []byte
	bPrivate     []byte // 암호화돼 있으면 암호문
	bMac         []byte
	szKdf        string
	iMemoryKiB   int
	iPasses      int
	iParallel    int
	bSalt        []byte
}

// readPpk 는 PPK 본문을 필드로 나눈다. 줄바꿈은 LF·CRLF·CR 을 모두 받는다.
func readPpk(bKey []byte) (*ppkFile, error) {
	szText := strings.TrimLeft(string(bKey), " \t\r\n\uFEFF")
	szText = strings.ReplaceAll(strings.ReplaceAll(szText, "\r\n", "\n"), "\r", "\n")
	slLines := strings.Split(szText, "\n")
	pFile := &ppkFile{}
	for iLine := 0; iLine < len(slLines); iLine++ {
		szName, szValue, bFound := strings.Cut(slLines[iLine], ": ")
		if !bFound {
			if strings.TrimSpace(slLines[iLine]) == "" {
				continue
			}
			return nil, ErrPpkFormat
		}
		switch {
		case strings.HasPrefix(szName, g_cszPpkHeader):
			iVersion, errVersion := strconv.Atoi(strings.TrimPrefix(szName, g_cszPpkHeader))
			if errVersion != nil {
				return nil, ErrPpkFormat
			}
			pFile.iVersion = iVersion
			pFile.szAlgorithm = strings.TrimSpace(szValue)
		case szName == "Encryption":
			pFile.szEncryption = strings.TrimSpace(szValue)
		case szName == "Comment":
			pFile.szComment = szValue
		case szName == "Public-Lines", szName == "Private-Lines":
			iCount, errCount := strconv.Atoi(strings.TrimSpace(szValue))
			if errCount != nil || iCount < 0 || iLine+iCount >= len(slLines) {
				return nil, ErrPpkFormat
			}
			bBlob, errBlob := base64.StdEncoding.DecodeString(strings.Join(slLines[iLine+1:iLine+1+iCount], ""))
			if errBlob != nil {
				return nil, ErrPpkFormat
			}
			if szName == "Public-Lines" {
				pFile.bPublic = bBlob
			} else {
				pFile.bPrivate = bBlob
			}
			iLine += iCount
		case szName == "Private-MAC":
			bMac, errMac := hex.DecodeString(strings.TrimSpace(szValue))
			if errMac != nil {
				return nil, ErrPpkFormat
			}
			pFile.bMac = bMac
		case szName == "Key-Derivation":
			pFile.szKdf = strings.TrimSpace(szValue)
		case szName == "Argon2-Memory":
			pFile.iMemoryKiB, _ = strconv.Atoi(strings.TrimSpace(szValue))
		case szName == "Argon2-Passes":
			pFile.iPasses, _ = strconv.Atoi(strings.TrimSpace(szValue))
		case szName == "Argon2-Parallelism":
			pFile.iParallel, _ = strconv.Atoi(strings.TrimSpace(szValue))
		case szName == "Argon2-Salt":
			bSalt, errSalt := hex.DecodeString(strings.TrimSpace(szValue))
			if errSalt != nil {
				return nil, ErrPpkFormat
			}
			pFile.bSalt = bSalt
		}
	}
	if pFile.iVersion == 0 || pFile.szAlgorithm == "" || len(pFile.bPublic) == 0 || len(pFile.bPrivate) == 0 || len(pFile.bMac) == 0 {
		return nil, ErrPpkFormat
	}
	return pFile, nil
}

// parsePpk 는 PPK 를 읽어 서명자를 만든다. 암호가 걸려 있는데 bPassphrase 가 nil 이면 *ssh.PassphraseMissingError 다.
func parsePpk(bKey []byte, bPassphrase []byte) (ssh.Signer, error) {
	pFile, errRead := readPpk(bKey)
	if errRead != nil {
		return nil, errRead
	}
	if pFile.iVersion != 2 && pFile.iVersion != 3 {
		return nil, fmt.Errorf("ppk: unsupported file format version %d", pFile.iVersion)
	}
	bEncrypted := false
	switch pFile.szEncryption {
	case "none":
	case "aes256-cbc":
		bEncrypted = true
	default:
		return nil, fmt.Errorf("ppk: unsupported encryption %q", pFile.szEncryption)
	}
	if bEncrypted && bPassphrase == nil {
		return nil, &ssh.PassphraseMissingError{}
	}
	if !bEncrypted {
		bPassphrase = nil // 암호가 없는 키에 준 암호는 쓰지 않는다
	}
	bCipherKey, bIv, bMacKey, fnHash, errKeys := ppkKeys(pFile, bEncrypted, bPassphrase)
	if errKeys != nil {
		return nil, errKeys
	}
	bPrivate := pFile.bPrivate
	if bEncrypted {
		if len(bPrivate) == 0 || len(bPrivate)%aes.BlockSize != 0 {
			return nil, ErrPpkFormat
		}
		block, errCipher := aes.NewCipher(bCipherKey)
		if errCipher != nil {
			return nil, errCipher
		}
		bPlain := make([]byte, len(bPrivate))
		cipher.NewCBCDecrypter(block, bIv).CryptBlocks(bPlain, bPrivate)
		bPrivate = bPlain
	}
	// MAC: 알고리즘·암호화·주석·공개 부분·(복호화한) 개인 부분을 각각 길이를 앞세운 문자열로 이어 붙인 것.
	mac := hmac.New(fnHash, bMacKey)
	for _, bPart := range [][]byte{[]byte(pFile.szAlgorithm), []byte(pFile.szEncryption), []byte(pFile.szComment), pFile.bPublic, bPrivate} {
		var arrLength [4]byte
		binary.BigEndian.PutUint32(arrLength[:], uint32(len(bPart)))
		mac.Write(arrLength[:])
		mac.Write(bPart)
	}
	if !hmac.Equal(mac.Sum(nil), pFile.bMac) {
		return nil, ErrPpkMac
	}
	anyKey, errBuild := ppkBuildKey(pFile.szAlgorithm, pFile.bPublic, bPrivate)
	if errBuild != nil {
		return nil, errBuild
	}
	return ssh.NewSignerFromKey(anyKey)
}

// ppkKeys 는 형식에 맞는 암호 키·IV·MAC 키와 MAC 해시를 만든다.
func ppkKeys(pFile *ppkFile, bEncrypted bool, bPassphrase []byte) (bCipherKey []byte, bIv []byte, bMacKey []byte, fnHash func() hash.Hash, err error) {
	if pFile.iVersion == 2 {
		// 형식 2: 암호 키 = SHA-1(0x00000000 ‖ 암호) ‖ SHA-1(0x00000001 ‖ 암호) 의 앞 32바이트, IV 는 0.
		// MAC 키 = SHA-1("putty-private-key-file-mac-key" ‖ 암호), MAC 은 HMAC-SHA-1.
		macSeed := sha1.New()
		macSeed.Write([]byte("putty-private-key-file-mac-key"))
		macSeed.Write(bPassphrase)
		if bEncrypted {
			var bStretched []byte
			for iSeq := byte(0); iSeq < 2; iSeq++ {
				h := sha1.New()
				h.Write([]byte{0, 0, 0, iSeq})
				h.Write(bPassphrase)
				bStretched = h.Sum(bStretched)
			}
			bCipherKey = bStretched[:32]
			bIv = make([]byte, aes.BlockSize)
		}
		return bCipherKey, bIv, macSeed.Sum(nil), sha1.New, nil
	}
	// 형식 3: 암호가 없으면 MAC 키는 빈 값. 있으면 Argon2 로 80바이트를 만들어 암호 키 32 · IV 16 · MAC 키 32 로 나눈다.
	if !bEncrypted {
		return nil, nil, []byte{}, sha256.New, nil
	}
	if pFile.iMemoryKiB <= 0 || pFile.iMemoryKiB > g_ciPpkArgonMemoryMaxKiB ||
		pFile.iPasses <= 0 || pFile.iPasses > g_ciPpkArgonPassesMax ||
		pFile.iParallel <= 0 || pFile.iParallel > 255 || len(pFile.bSalt) == 0 {
		return nil, nil, nil, nil, ErrPpkFormat
	}
	var bDerived []byte
	switch pFile.szKdf {
	case "Argon2id":
		bDerived = argon2.IDKey(bPassphrase, pFile.bSalt, uint32(pFile.iPasses), uint32(pFile.iMemoryKiB), uint8(pFile.iParallel), 80)
	case "Argon2i":
		bDerived = argon2.Key(bPassphrase, pFile.bSalt, uint32(pFile.iPasses), uint32(pFile.iMemoryKiB), uint8(pFile.iParallel), 80)
	default:
		// Argon2d 는 쓰는 라이브러리(x/crypto/argon2)에 없다. PuTTYgen 기본값은 Argon2id 다.
		return nil, nil, nil, nil, fmt.Errorf("ppk: unsupported key derivation %q", pFile.szKdf)
	}
	return bDerived[:32], bDerived[32:48], bDerived[48:80], sha256.New, nil
}

// ppkReader 는 SSH 형식(길이 4바이트 + 내용)의 값을 차례로 읽는다.
type ppkReader struct {
	bRest []byte
	bBad  bool
}

func (pReader *ppkReader) bytes() []byte {
	if pReader.bBad || len(pReader.bRest) < 4 {
		pReader.bBad = true
		return nil
	}
	uiLength := binary.BigEndian.Uint32(pReader.bRest[:4])
	if uint64(uiLength) > uint64(len(pReader.bRest)-4) {
		pReader.bBad = true
		return nil
	}
	bOut := pReader.bRest[4 : 4+uiLength]
	pReader.bRest = pReader.bRest[4+uiLength:]
	return bOut
}

func (pReader *ppkReader) mpint() *big.Int {
	return new(big.Int).SetBytes(pReader.bytes())
}

// ppkBuildKey 는 공개·개인 부분으로 Go 개인 키를 만든다 (RSA·Ed25519·ECDSA P-256/384/521).
func ppkBuildKey(szAlgorithm string, bPublic []byte, bPrivate []byte) (any, error) {
	pub := &ppkReader{bRest: bPublic}
	priv := &ppkReader{bRest: bPrivate}
	if string(pub.bytes()) != szAlgorithm {
		return nil, ErrPpkFormat
	}
	switch szAlgorithm {
	case "ssh-rsa":
		pE, pN := pub.mpint(), pub.mpint()
		pD, pP, pQ := priv.mpint(), priv.mpint(), priv.mpint()
		if pub.bBad || priv.bBad || !pE.IsInt64() || pE.Int64() <= 0 || pE.Int64() > 1<<31-1 {
			return nil, ErrPpkFormat
		}
		pKey := &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: pN, E: int(pE.Int64())}, D: pD, Primes: []*big.Int{pP, pQ}}
		if errValidate := pKey.Validate(); errValidate != nil {
			return nil, errValidate
		}
		pKey.Precompute()
		return pKey, nil
	case "ssh-ed25519":
		bPub, bSeed := pub.bytes(), priv.bytes()
		if pub.bBad || priv.bBad || len(bPub) != ed25519.PublicKeySize || len(bSeed) != ed25519.SeedSize {
			return nil, ErrPpkFormat
		}
		key := ed25519.NewKeyFromSeed(bSeed)
		if !bytes.Equal(key.Public().(ed25519.PublicKey), bPub) {
			return nil, ErrPpkFormat
		}
		return &key, nil
	case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		var curve elliptic.Curve
		switch szAlgorithm {
		case "ecdsa-sha2-nistp256":
			curve = elliptic.P256()
		case "ecdsa-sha2-nistp384":
			curve = elliptic.P384()
		default:
			curve = elliptic.P521()
		}
		_ = pub.bytes() // 곡선 이름
		bPoint := pub.bytes()
		pD := priv.mpint()
		iSize := (curve.Params().BitSize + 7) / 8
		if pub.bBad || priv.bBad || len(pD.Bytes()) > iSize {
			return nil, ErrPpkFormat
		}
		pKey, errKey := ecdsa.ParseRawPrivateKey(curve, pD.FillBytes(make([]byte, iSize)))
		if errKey != nil {
			return nil, errKey
		}
		bGot, errPoint := pKey.PublicKey.Bytes()
		if errPoint != nil || !bytes.Equal(bGot, bPoint) {
			return nil, ErrPpkFormat
		}
		return pKey, nil
	default:
		return nil, fmt.Errorf("ppk: unsupported key type %q", szAlgorithm)
	}
}
