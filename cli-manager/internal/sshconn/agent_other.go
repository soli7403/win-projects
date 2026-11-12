//go:build !windows

package sshconn

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// agentAuth 는 SSH_AUTH_SOCK 의 에이전트에 연결한다. 없으면 nil.
func agentAuth() ssh.AuthMethod {
	szSock := os.Getenv("SSH_AUTH_SOCK")
	if szSock == "" {
		return nil
	}
	conn, err := net.Dial("unix", szSock)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}
