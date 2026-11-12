//go:build windows

package sshconn

import (
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// g_szOpenSSHAgentPipe 은 OpenSSH 에이전트의 이름 있는 파이프다 (protocols.md §2.2).
const g_szOpenSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// agentSignersFor 는 OpenSSH 에이전트 파이프에 연결해 서명자 목록을 돌려준다.
// 연결할 수 없으면 nil 을 돌려 이 인증 방법을 건너뛰게 한다.
func agentSignersFor() func() ([]ssh.Signer, error) {
	conn, errOpen := os.OpenFile(g_szOpenSSHAgentPipe, os.O_RDWR, 0)
	if errOpen != nil {
		return nil
	}
	client := agent.NewClient(conn)
	return client.Signers
}
