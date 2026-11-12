//go:build !windows

package sshconn

import "golang.org/x/crypto/ssh"

// agentSignersFor 는 1차 범위 밖이라 항상 nil 이다 (SSH_AUTH_SOCK 미지원, protocols.md §2.2).
func agentSignersFor() func() ([]ssh.Signer, error) {
	return nil
}
