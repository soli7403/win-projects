package sshconn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"golang.org/x/crypto/ssh/agent"
)

// oneShotRW 는 요청 하나를 읽히고 응답을 모으는 흐름이다 (가짜 Pageant 안에서 에이전트 서버를 한 번 돌리는 데 쓴다).
type oneShotRW struct {
	pIn  *bytes.Reader
	bOut bytes.Buffer
}

func (pRW *oneShotRW) Read(bBuf []byte) (int, error)  { return pRW.pIn.Read(bBuf) }
func (pRW *oneShotRW) Write(bBuf []byte) (int, error) { return pRW.bOut.Write(bBuf) }

// fakePageant 는 "요청 하나 → 응답 하나" 방식으로 답하는 가짜 에이전트다. 부른 횟수를 센다.
func fakePageant(keyring agent.Agent, piCalls *int) func([]byte) ([]byte, error) {
	return func(bRequest []byte) ([]byte, error) {
		*piCalls++
		rw := &oneShotRW{pIn: bytes.NewReader(bRequest)}
		if errServe := agent.ServeAgent(keyring, rw); errServe != nil && !errors.Is(errServe, io.EOF) {
			return nil, errServe
		}
		return rw.bOut.Bytes(), nil
	}
}

// 요청-응답 방식 에이전트(Pageant)를 x/crypto 의 에이전트 클라이언트로 쓸 수 있다: 키 목록과 서명 (D107).
func TestQueryAgentListsAndSigns(t *testing.T) {
	keyring := agent.NewKeyring()
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	if errAdd := keyring.Add(agent.AddedKey{PrivateKey: privKey, Comment: "pageant-key"}); errAdd != nil {
		t.Fatal(errAdd)
	}
	iCalls := 0
	pAgent := newQueryAgent(fakePageant(keyring, &iCalls))
	slKeys, errList := pAgent.List()
	if errList != nil || len(slKeys) != 1 || slKeys[0].Comment != "pageant-key" {
		t.Fatalf("키 목록: %v %v", slKeys, errList)
	}
	slSigners, errSigners := pAgent.Signers()
	if errSigners != nil || len(slSigners) != 1 {
		t.Fatalf("서명자: %v", errSigners)
	}
	sig, errSign := slSigners[0].Sign(rand.Reader, []byte("data"))
	if errSign != nil {
		t.Fatal(errSign)
	}
	if !ed25519.Verify(pubKey, []byte("data"), sig.Blob) {
		t.Fatal("에이전트 서명이 맞지 않음")
	}
	if iCalls != 3 {
		t.Fatalf("요청마다 한 번씩 물어야 함(목록·목록·서명): %d", iCalls)
	}
}

// 요청이 여러 번의 Write 로 나뉘어 와도 메시지 하나가 완성됐을 때 한 번만 묻는다.
func TestQueryConnAssemblesSplitWrites(t *testing.T) {
	iCalls := 0
	pConn := &queryConn{fnQuery: func(bRequest []byte) ([]byte, error) {
		iCalls++
		if !bytes.Equal(bRequest, []byte{0, 0, 0, 2, 11, 7}) {
			t.Fatalf("요청이 그대로 와야 함: %v", bRequest)
		}
		return []byte{0, 0, 0, 1, 5}, nil
	}}
	for _, bPart := range [][]byte{{0, 0}, {0, 2, 11}, {7}} {
		if _, errWrite := pConn.Write(bPart); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	if iCalls != 1 {
		t.Fatalf("한 번만 물어야 함: %d", iCalls)
	}
	bGot, _ := io.ReadAll(pConn)
	if !bytes.Equal(bGot, []byte{0, 0, 0, 1, 5}) {
		t.Fatalf("응답: %v", bGot)
	}
}

// 길이 머리가 0 이거나 너무 크면 묻지 않고 오류다. 응답의 길이 머리가 실제보다 길어도 오류다.
func TestQueryConnRejectsBadLengths(t *testing.T) {
	pConn := &queryConn{fnQuery: func([]byte) ([]byte, error) {
		t.Fatal("물으면 안 됨")
		return nil, nil
	}}
	if _, errZero := pConn.Write([]byte{0, 0, 0, 0}); !errors.Is(errZero, errAgentMessage) {
		t.Fatalf("길이 0: %v", errZero)
	}
	if _, errBig := pConn.Write([]byte{0xff, 0xff, 0xff, 0xff}); !errors.Is(errBig, errAgentMessage) {
		t.Fatalf("너무 큰 길이: %v", errBig)
	}
	if _, errReply := checkAgentReply([]byte{0, 0, 0, 9, 1, 2}); !errors.Is(errReply, errAgentMessage) {
		t.Fatalf("응답 길이 초과: %v", errReply)
	}
	bReply, errOk := checkAgentReply([]byte{0, 0, 0, 2, 1, 2, 0, 0, 0})
	if errOk != nil || !bytes.Equal(bReply, []byte{0, 0, 0, 2, 1, 2}) {
		t.Fatalf("응답은 길이만큼만 잘라야 함: %v %v", bReply, errOk)
	}
}
