package sshconn

import (
	"errors"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

// ErrHostKeyRejected 는 사용자가 호스트 키를 승인하지 않았을 때의 오류다 (protocols.md §2.3).
var ErrHostKeyRejected = errors.New("호스트 키를 승인하지 않았습니다 (host key rejected)")

// hostKeyCallbackFor 는 host:port 기준으로 저장된 지문과 비교하는 콜백을 만든다.
// 저장된 지문과 같으면 통과, 처음이거나 바뀌었으면 사용자에게 묻고 승인하면 기록한다.
func hostKeyCallbackFor(tgt *Target, store HostKeyStore, prompt Prompter) ssh.HostKeyCallback {
	return func(szHostname string, addrRemote net.Addr, keyPublic ssh.PublicKey) error {
		szHostPort := net.JoinHostPort(tgt.Host, strconv.Itoa(tgt.Port))
		szFingerprint := ssh.FingerprintSHA256(keyPublic)
		szOld, bKnown := store.Lookup(szHostPort)
		if bKnown && szOld == szFingerprint {
			return nil
		}
		bApproved := prompt.HostKey(tgt.Title, szHostPort, keyPublic.Type(), szFingerprint, bKnown, szOld)
		if !bApproved {
			return ErrHostKeyRejected
		}
		store.Save(szHostPort, keyPublic.Type(), szFingerprint)
		return nil
	}
}
