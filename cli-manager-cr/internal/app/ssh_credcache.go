package app

import (
	"net"
	"strconv"
	"sync"

	"dariterm/internal/sshconn"
)

// credCopy 는 세션이 접속에 쓴 계정 하나(대상 또는 점프 서버)의 사본이다.
type credCopy struct {
	szUsername   string
	szPassword   string
	bPrivateKey  []byte
	szPassphrase string
}

// sessionCreds 는 SSH 세션 하나가 접속에 쓴 계정 사본이다 (protocols.md §2.2 세션 사본).
// 볼트에서 읽은 값과 사용자가 입력해 인증에 성공한 값을 세션이 열려 있는 동안만 메모리에 둔다.
// 볼트가 잠긴 동안의 재접속과, 저장하지 않은 비밀번호로 다시 접속할 때 쓴다. 세션을 닫으면 clear 로 지운다.
// 열린 세션을 잠금 중에도 유지하는 정책(lifecycle §2)과 같은 위험 수준으로 본다.
type sessionCreds struct {
	mtxData  sync.Mutex
	mapItems map[string]credCopy // 키: 깊이|host:port (점프 체인의 자리마다 하나)
}

func newSessionCreds() *sessionCreds {
	return &sessionCreds{mapItems: map[string]credCopy{}}
}

// credKey 는 점프 체인에서 대상의 자리를 나타내는 키다.
func credKey(tgt *sshconn.Target) string {
	return strconv.Itoa(tgt.Depth) + "|" + net.JoinHostPort(tgt.Host, strconv.Itoa(tgt.Port))
}

// fill 은 tgt 에서 비어 있는 계정 필드를 사본으로 채운다. 채울 사본이 있었는지 돌려준다.
// 볼트에서 읽은 값이 있으면 그 값이 우선이다(비어 있는 필드만 채운다).
func (pCreds *sessionCreds) fill(tgt *sshconn.Target) bool {
	pCreds.mtxData.Lock()
	defer pCreds.mtxData.Unlock()
	item, bOk := pCreds.mapItems[credKey(tgt)]
	if !bOk {
		return false
	}
	if tgt.Username == "" {
		tgt.Username = item.szUsername
	}
	if tgt.Password == "" {
		tgt.Password = item.szPassword
	}
	if len(tgt.PrivateKey) == 0 && len(item.bPrivateKey) > 0 {
		tgt.PrivateKey = append([]byte(nil), item.bPrivateKey...)
	}
	if tgt.Passphrase == "" {
		tgt.Passphrase = item.szPassphrase
	}
	return true
}

// remember 는 접속에 성공한 대상 체인(대상과 점프 서버들)의 계정을 사본으로 남긴다.
// sshconn.Connect 는 성공하면 실제로 쓰인 사용자 이름·입력한 비밀번호·키 암호를 Target 에 채워 둔다.
func (pCreds *sessionCreds) remember(tgt *sshconn.Target) {
	pCreds.mtxData.Lock()
	defer pCreds.mtxData.Unlock()
	for pCur := tgt; pCur != nil; pCur = pCur.Jump {
		szKey := credKey(pCur)
		if old, bOk := pCreds.mapItems[szKey]; bOk {
			zeroBytes(old.bPrivateKey)
		}
		pCreds.mapItems[szKey] = credCopy{
			szUsername:   pCur.Username,
			szPassword:   pCur.Password,
			bPrivateKey:  append([]byte(nil), pCur.PrivateKey...),
			szPassphrase: pCur.Passphrase,
		}
	}
}

// clear 는 사본을 모두 지운다. 개인 키 바이트는 0 으로 덮어쓰고, 문자열은 참조만 끊는다
// (Go 문자열은 덮어쓸 수 없다. x/crypto/ssh 의 비밀번호 인증도 string 을 받는다).
func (pCreds *sessionCreds) clear() {
	pCreds.mtxData.Lock()
	defer pCreds.mtxData.Unlock()
	for _, item := range pCreds.mapItems {
		zeroBytes(item.bPrivateKey)
	}
	pCreds.mapItems = map[string]credCopy{}
}

func zeroBytes(bData []byte) {
	for iIndex := range bData {
		bData[iIndex] = 0
	}
}
