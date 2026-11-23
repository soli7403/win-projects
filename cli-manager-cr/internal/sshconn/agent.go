package sshconn

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSH 에이전트 공통 부분 (protocols.md §2.2 2번). 운영체제별 연결은 agent_windows.go · agent_other.go 에 있다.

// g_ciAgentMaxMessage 는 에이전트 메시지 하나의 최대 길이다 (PuTTY 의 AGENT_MAX_MSGLEN 과 같은 값).
const g_ciAgentMaxMessage = 262144

// errAgentMessage 는 에이전트와 주고받은 메시지 모양이 틀렸을 때의 오류다.
var errAgentMessage = errors.New("ssh agent: malformed message")

// g_fnAvailableAgents 는 연결할 수 있는 에이전트 목록을 돌려주는 함수다. 시험에서 가짜 에이전트로 바꾼다.
var g_fnAvailableAgents = availableAgents

// agentSignersFor 는 연결할 수 있는 모든 에이전트의 서명자를 모아 돌려주는 함수를 만든다.
// 에이전트가 하나도 없으면 nil 을 돌려 이 인증 방법을 건너뛰게 한다.
func agentSignersFor() func() ([]ssh.Signer, error) {
	slAgents := g_fnAvailableAgents()
	if len(slAgents) == 0 {
		return nil
	}
	return func() ([]ssh.Signer, error) {
		var slAll []ssh.Signer
		for _, pAgent := range slAgents {
			slSigners, errSigners := pAgent.Signers()
			if errSigners != nil {
				continue // 한 에이전트가 실패해도 나머지는 쓴다
			}
			slAll = append(slAll, slSigners...)
		}
		return slAll, nil
	}
}

// queryConn 은 "요청 하나를 보내면 응답 하나가 온다"는 방식의 에이전트(Pageant)를
// x/crypto/ssh/agent 가 쓰는 바이트 흐름(io.ReadWriter)으로 보이게 한다.
// Write 로 들어온 바이트를 모아 메시지 하나(길이 4바이트 + 내용)가 완성되면 fnQuery 를 부르고,
// 그 응답을 Read 가 돌려준다.
type queryConn struct {
	mtxData sync.Mutex
	fnQuery func(bRequest []byte) ([]byte, error)
	bIn     []byte // 아직 메시지 하나가 되지 않은 요청 바이트
	bOut    []byte // 아직 읽어 가지 않은 응답 바이트
}

func (pConn *queryConn) Write(bData []byte) (int, error) {
	pConn.mtxData.Lock()
	defer pConn.mtxData.Unlock()
	pConn.bIn = append(pConn.bIn, bData...)
	for len(pConn.bIn) >= 4 {
		uiLength := binary.BigEndian.Uint32(pConn.bIn[:4])
		if uiLength == 0 || uiLength > g_ciAgentMaxMessage-4 {
			pConn.bIn = nil
			return 0, errAgentMessage
		}
		iTotal := 4 + int(uiLength)
		if len(pConn.bIn) < iTotal {
			break
		}
		bReply, errQuery := pConn.fnQuery(pConn.bIn[:iTotal])
		pConn.bIn = pConn.bIn[iTotal:]
		if errQuery != nil {
			return 0, errQuery
		}
		pConn.bOut = append(pConn.bOut, bReply...)
	}
	return len(bData), nil
}

func (pConn *queryConn) Read(bBuf []byte) (int, error) {
	pConn.mtxData.Lock()
	defer pConn.mtxData.Unlock()
	if len(pConn.bOut) == 0 {
		return 0, io.EOF
	}
	nCopy := copy(bBuf, pConn.bOut)
	pConn.bOut = pConn.bOut[nCopy:]
	return nCopy, nil
}

// newQueryAgent 는 요청-응답 함수로 에이전트 클라이언트를 만든다.
func newQueryAgent(fnQuery func(bRequest []byte) ([]byte, error)) agent.ExtendedAgent {
	return agent.NewClient(&queryConn{fnQuery: fnQuery})
}

// checkAgentReply 는 응답의 길이 머리가 실제 길이와 맞는지 보고, 머리를 포함한 응답 전체를 돌려준다.
func checkAgentReply(bView []byte) ([]byte, error) {
	if len(bView) < 4 {
		return nil, errAgentMessage
	}
	uiLength := binary.BigEndian.Uint32(bView[:4])
	if uiLength == 0 || uint64(uiLength)+4 > uint64(len(bView)) {
		return nil, errAgentMessage
	}
	return append([]byte(nil), bView[:4+uiLength]...), nil
}
