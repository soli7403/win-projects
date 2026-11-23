//go:build !windows

package sshconn

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// availableAgents 는 환경 변수 SSH_AUTH_SOCK 의 에이전트 소켓에 연결한다. 없거나 연결할 수 없으면 빈 목록이다 (D107).
func availableAgents() []agent.ExtendedAgent {
	szSocket := os.Getenv("SSH_AUTH_SOCK")
	if szSocket == "" {
		return nil
	}
	conn, errDial := net.Dial("unix", szSocket)
	if errDial != nil {
		return nil
	}
	return []agent.ExtendedAgent{agent.NewClient(conn)}
}
