package sshconn

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH 키 만들기와 서버 등록 (spec/behavior/ssh-keys.md, D162).

// 만들 수 있는 키 종류.
const (
	G_cszKeyEd25519 = "ed25519"
	G_cszKeyRsa     = "rsa"
)

// G_ciRsaDefaultBits 는 RSA 키의 기본 길이다. 3072·4096 만 받는다.
const G_ciRsaDefaultBits = 4096

// 서버 등록 명령이 내는 결과 표시. 명령의 다른 출력과 섞이지 않을 이름이다.
const (
	g_cszInstallAdded  = "DARITERM_KEY_ADDED"
	g_cszInstallExists = "DARITERM_KEY_EXISTS"
)

// G_durInstallKey 안에 등록 명령이 끝나지 않으면 실패로 본다. 시험에서 줄일 수 있게 변수다.
var G_durInstallKey = 20 * time.Second

// 키 만들기·등록 오류.
var (
	// ErrKeyType 은 모르는 키 종류다.
	ErrKeyType = errors.New("keygen: unsupported key type")
	// ErrKeyBits 는 받지 않는 RSA 길이다.
	ErrKeyBits = errors.New("keygen: RSA key length must be 3072 or 4096")
	// ErrInstallTimeout 은 등록 명령이 시한 안에 끝나지 않았다는 오류다.
	ErrInstallTimeout = errors.New("install key: server did not answer in time")
)

// GeneratedKey 는 새로 만든 키다.
type GeneratedKey struct {
	PrivateKey  string // OpenSSH 형식("-----BEGIN OPENSSH PRIVATE KEY-----"), 암호를 주면 암호가 걸린다
	PublicKey   string // authorized_keys 한 줄 (주석 포함, 줄바꿈 없음)
	Fingerprint string // "SHA256:…"
}

// GenerateKey 는 새 키 쌍을 만든다. szType 은 ed25519(기본) 또는 rsa, iBits 는 RSA 길이(0 이면 4096).
func GenerateKey(szType string, iBits int, szComment string, szPassphrase string) (GeneratedKey, error) {
	var privKey crypto.PrivateKey
	var pubKey crypto.PublicKey
	switch szType {
	case "", G_cszKeyEd25519:
		pub, priv, errGen := ed25519.GenerateKey(rand.Reader)
		if errGen != nil {
			return GeneratedKey{}, errGen
		}
		privKey, pubKey = priv, pub
	case G_cszKeyRsa:
		if iBits == 0 {
			iBits = G_ciRsaDefaultBits
		}
		if iBits != 3072 && iBits != 4096 {
			return GeneratedKey{}, ErrKeyBits
		}
		priv, errGen := rsa.GenerateKey(rand.Reader, iBits)
		if errGen != nil {
			return GeneratedKey{}, errGen
		}
		privKey, pubKey = priv, &priv.PublicKey
	default:
		return GeneratedKey{}, ErrKeyType
	}
	var pBlock *pem.Block
	var errMarshal error
	if szPassphrase == "" {
		pBlock, errMarshal = ssh.MarshalPrivateKey(privKey, szComment)
	} else {
		pBlock, errMarshal = ssh.MarshalPrivateKeyWithPassphrase(privKey, szComment, []byte(szPassphrase))
	}
	if errMarshal != nil {
		return GeneratedKey{}, errMarshal
	}
	sshPub, errPub := ssh.NewPublicKey(pubKey)
	if errPub != nil {
		return GeneratedKey{}, errPub
	}
	return GeneratedKey{
		PrivateKey:  string(pem.EncodeToMemory(pBlock)),
		PublicKey:   AuthorizedLine(sshPub, szComment),
		Fingerprint: ssh.FingerprintSHA256(sshPub),
	}, nil
}

// AuthorizedLine 은 공개 키의 authorized_keys 한 줄이다. 주석은 줄바꿈·앞뒤 공백을 지우고 붙인다.
func AuthorizedLine(pub ssh.PublicKey, szComment string) string {
	szLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	szComment = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(szComment))
	if szComment != "" {
		szLine += " " + szComment
	}
	return szLine
}

// PublicKeyOf 는 개인 키(OpenSSH·PEM·PPK)의 공개 키 한 줄과 지문을 돌려준다.
// 암호가 걸린 OpenSSH 키는 암호 없이도 공개 키를 읽는다(파일에 평문으로 들어 있다). 그 밖은 szPassphrase 로 푼다.
func PublicKeyOf(szPrivate string, szPassphrase string, szComment string) (string, string, error) {
	bKey := []byte(szPrivate)
	signer, errParse := ParseKey(bKey)
	if errParse != nil {
		var pMissing *ssh.PassphraseMissingError
		if errors.As(errParse, &pMissing) && pMissing.PublicKey != nil {
			return AuthorizedLine(pMissing.PublicKey, szComment), ssh.FingerprintSHA256(pMissing.PublicKey), nil
		}
		if szPassphrase == "" {
			return "", "", errParse
		}
		signer, errParse = ParseKeyWithPassphrase(bKey, []byte(szPassphrase))
		if errParse != nil {
			return "", "", errParse
		}
	}
	pub := signer.PublicKey()
	return AuthorizedLine(pub, szComment), ssh.FingerprintSHA256(pub), nil
}

// InstallCommand 는 서버에서 공개 키 한 줄(표준 입력)을 ~/.ssh/authorized_keys 에 넣는 명령이다.
// 이미 같은 키(종류와 본문이 같으면 주석은 달라도)가 있으면 넣지 않는다. 새 파일·폴더는 umask 077 로 만든다.
// 파일 끝에 줄바꿈이 없으면 먼저 줄을 바꾼다. 로그인 셸이 무엇이든 sh 로 돌린다.
const InstallCommand = `sh -c 'umask 077; d="$HOME/.ssh"; f="$d/authorized_keys"; ` +
	`mkdir -p "$d" && touch "$f" && IFS= read -r k && kb=$(printf "%s" "$k" | cut -d " " -f 1,2) && ` +
	`if grep -qF -- "$kb" "$f"; then echo ` + g_cszInstallExists + `; ` +
	`else if [ -s "$f" ] && [ -n "$(tail -c 1 "$f")" ]; then echo >> "$f"; fi; ` +
	`printf "%s\n" "$k" >> "$f" && echo ` + g_cszInstallAdded + `; fi'`

// InstallPublicKey 는 열린 연결에서 공개 키를 서버의 authorized_keys 에 넣는다.
// 돌려주는 값은 이미 있었는지(true 면 넣지 않음)다.
func InstallPublicKey(client *ssh.Client, szLine string) (bool, error) {
	szLine = strings.TrimSpace(szLine)
	if _, _, _, _, errParse := ssh.ParseAuthorizedKey([]byte(szLine)); errParse != nil {
		return false, errParse
	}
	session, errSession := client.NewSession()
	if errSession != nil {
		return false, errSession
	}
	defer session.Close()
	var bufOut, bufErr bytes.Buffer
	session.Stdin = strings.NewReader(szLine + "\n")
	session.Stdout = &bufOut
	session.Stderr = &bufErr
	ctx, cancel := context.WithTimeout(context.Background(), G_durInstallKey)
	defer cancel()
	chDone := make(chan error, 1)
	go func() { chDone <- session.Run(InstallCommand) }()
	var errRun error
	select {
	case errRun = <-chDone:
	case <-ctx.Done():
		_ = session.Close()
		return false, ErrInstallTimeout
	}
	szOut := bufOut.String()
	switch {
	case strings.Contains(szOut, g_cszInstallExists):
		return true, nil
	case strings.Contains(szOut, g_cszInstallAdded):
		return false, nil
	}
	szDetail := strings.TrimSpace(bufErr.String())
	if szDetail == "" {
		szDetail = strings.TrimSpace(szOut)
	}
	if len([]rune(szDetail)) > 200 {
		szDetail = string([]rune(szDetail)[:200]) + "…"
	}
	if errRun == nil {
		errRun = errors.New("unexpected output")
	}
	if szDetail != "" {
		return false, fmt.Errorf("%w: %s", errRun, szDetail)
	}
	return false, errRun
}

// InstallPublicKey 는 이 연결로 공개 키를 서버에 등록한다 (key.install).
func (pConn *Conn) InstallPublicKey(szLine string) (bool, error) {
	return InstallPublicKey(pConn.Client, szLine)
}
