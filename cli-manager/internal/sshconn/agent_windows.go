//go:build windows

package sshconn

import (
	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// agentAuth 는 Windows OpenSSH 에이전트(named pipe)에 연결한다. 없으면 nil.
func agentAuth() ssh.AuthMethod {
	conn, err := winio.DialPipe(`\\.\pipe\openssh-ssh-agent`, nil)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}
