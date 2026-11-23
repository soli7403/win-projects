# Pageant 연동 (SSH 에이전트) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D107). 실제 Pageant 와의 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 2번. 앞 문서 [ppk-keys](./2026-11-23-ppk-keys.md), 뒤 문서 [serial-flow-control](./2026-11-23-serial-flow-control.md). 에이전트를 서버로 전달하는 기능은 [agent-forwarding](./2026-11-23-agent-forwarding.md).

---

## 0. 한 줄 요약

인증 방식 "SSH 에이전트 (Pageant)"가 이름과 달리 **Pageant 에는 연결하지 않았다**(Windows OpenSSH 에이전트 파이프에만 연결). 이제 OpenSSH 에이전트와 Pageant 를 **둘 다** 보고, 연결되는 에이전트의 키를 모두 시도한다. Windows 가 아닌 OS 는 `SSH_AUTH_SOCK` 소켓에 연결한다(전에는 미구현).

## 1. 배경 (원인)

- [internal/sshconn/agent_windows.go](../../internal/sshconn/agent_windows.go) 는 `\\.\pipe\openssh-ssh-agent` 를 여는 것뿐이었다. 명세(protocols.md §2.2)도 "화면 문구에는 Pageant 가 적혀 있으나 Pageant 에는 연결하지 않는다 — 알려진 불일치"라고 적어 두었다.
- PuTTY 를 쓰던 사람은 키를 Pageant 에 올려 두므로, 에이전트 인증을 골라도 키가 하나도 시도되지 않고 비밀번호를 물었다.
- `agent_other.go` 는 항상 nil 을 돌려주었다(1차 범위 밖).

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| Pageant 와 통신하는 방법 | PuTTY 공개 문서의 방식: 이름 있는 공유 메모리에 요청을 쓰고, 그 이름을 `WM_COPYDATA`(식별값 `0x804e50ba`)로 창 클래스 `Pageant` 에 보낸다. 응답은 같은 메모리에 온다 | 모든 Pageant 버전이 받는 방식. 새 Pageant 의 이름 있는 파이프는 이름을 계산하는 규칙이 복잡하고 버전에 따라 다르다 |
| 기다리는 시간 | `SendMessageTimeoutW` 30초, `SMTO_ABORTIFHUNG` | Pageant 가 키 사용 확인 창을 띄울 수 있다. 멈춘 Pageant 에 앱이 매달리지 않게 |
| `x/crypto/ssh/agent` 와 잇는 방법 | `queryConn`: "요청 하나 → 응답 하나" 함수를 바이트 흐름(`io.ReadWriter`)으로 보이게 하는 어댑터 | 에이전트 프로토콜 처리(키 목록·서명)는 라이브러리 것을 그대로 쓴다 |
| 여러 에이전트 | OpenSSH 파이프 → Pageant 순서로 모아 **모든 키**를 시도 | 어느 쪽에 키가 있든 된다. 한 에이전트가 실패해도 나머지는 쓴다 |
| 메시지 크기 | 262144 바이트까지(PuTTY 의 `AGENT_MAX_MSGLEN`) | 길이 머리가 0 이거나 이보다 크면 묻지 않고 오류 |
| OS 분리 | 공통 `agent.go`, `agent_windows.go`, `agent_other.go`(`SSH_AUTH_SOCK`) | 공통 규칙 §8 OS 중립 |
| 시험용 훅 | `g_fnAvailableAgents` 변수 | [agent-forwarding](./2026-11-23-agent-forwarding.md) 테스트가 가짜 에이전트를 넣는다(그 문서에서 추가) |

### 2.1 알려진 한계

- **앱을 관리자 권한으로 실행하면** Pageant 가 요청을 거절할 수 있다. Pageant 는 공유 메모리의 소유자가 자기와 같은 사용자인지 확인하는데, 관리자 권한 프로세스가 만든 객체의 소유자는 Administrators 그룹이 된다. PuTTY 는 소유자를 직접 바꿔 넣지만 여기서는 하지 않았다(스토어 설치본은 관리자 권한으로 돌지 않는다).
- OpenSSH 파이프 연결은 접속할 때마다 새로 열고 닫지 않는다(전부터 그랬다). 접속 횟수만큼 핸들이 남는다 — 정리 거리.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/sshconn/agent.go` | 코드 | 새 파일 |
| `internal/sshconn/agent_other.go` | 코드 | 수정 |
| `internal/sshconn/agent_windows.go` | 코드 | 수정 |
| `internal/sshconn/agent_test.go` | 테스트 | 새 파일 |
| `spec/protocols.md` | 명세 | 수정 |
| `spec/ui/host-editor.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `internal/sshconn/agent.go` — 새 파일 (전체)

```go
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

// agentSignersFor 는 연결할 수 있는 모든 에이전트의 서명자를 모아 돌려주는 함수를 만든다.
// 에이전트가 하나도 없으면 nil 을 돌려 이 인증 방법을 건너뛰게 한다.
func agentSignersFor() func() ([]ssh.Signer, error) {
	slAgents := availableAgents()
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
```

### `internal/sshconn/agent_other.go` — 수정

```diff
--- a/internal/sshconn/agent_other.go
+++ b/internal/sshconn/agent_other.go
@@ -2,10 +2,23 @@
 
 package sshconn
 
-import "golang.org/x/crypto/ssh"
+import (
+	"net"
+	"os"
 
-// agentSignersFor 는 1차 범위 밖이라 항상 nil 이다 (SSH_AUTH_SOCK 미지원, protocols.md §2.2).
-func agentSignersFor() func() ([]ssh.Signer, error) {
-	return nil
+	"golang.org/x/crypto/ssh/agent"
+)
+
+// availableAgents 는 환경 변수 SSH_AUTH_SOCK 의 에이전트 소켓에 연결한다. 없거나 연결할 수 없으면 빈 목록이다 (D107).
+func availableAgents() []agent.ExtendedAgent {
+	szSocket := os.Getenv("SSH_AUTH_SOCK")
+	if szSocket == "" {
+		return nil
+	}
+	conn, errDial := net.Dial("unix", szSocket)
+	if errDial != nil {
+		return nil
+	}
+	return []agent.ExtendedAgent{agent.NewClient(conn)}
 }
 
```

### `internal/sshconn/agent_windows.go` — 수정

```diff
--- a/internal/sshconn/agent_windows.go
+++ b/internal/sshconn/agent_windows.go
@@ -3,23 +3,106 @@
 package sshconn
 
 import (
+	"fmt"
 	"os"
+	"sync/atomic"
+	"unsafe"
 
-	"golang.org/x/crypto/ssh"
 	"golang.org/x/crypto/ssh/agent"
+	"golang.org/x/sys/windows"
 )
 
 // g_szOpenSSHAgentPipe 은 OpenSSH 에이전트의 이름 있는 파이프다 (protocols.md §2.2).
 const g_szOpenSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`
 
-// agentSignersFor 는 OpenSSH 에이전트 파이프에 연결해 서명자 목록을 돌려준다.
-// 연결할 수 없으면 nil 을 돌려 이 인증 방법을 건너뛰게 한다.
-func agentSignersFor() func() ([]ssh.Signer, error) {
-	conn, errOpen := os.OpenFile(g_szOpenSSHAgentPipe, os.O_RDWR, 0)
-	if errOpen != nil {
-		return nil
-	}
-	client := agent.NewClient(conn)
-	return client.Signers
+// Pageant 와 통신할 때 쓰는 값이다 (PuTTY 공개 문서 "Pageant 프로토콜": 창 메시지 WM_COPYDATA + 공유 메모리).
+const (
+	g_cszPageantWindow     = "Pageant"  // 창 클래스 이름이자 창 제목
+	g_cuiPageantCopyDataId = 0x804e50ba // COPYDATASTRUCT.dwData 에 넣는 식별값
+	g_cuiWmCopyData        = 0x004A
+	g_cuiSmtoAbortIfHung   = 0x0002
+	g_cuiPageantTimeoutMs  = 30000 // Pageant 가 키 사용 확인 창을 띄울 수 있어 넉넉히 기다린다
+)
+
+var (
+	g_pUser32               = windows.NewLazySystemDLL("user32.dll")
+	g_pProcFindWindowW      = g_pUser32.NewProc("FindWindowW")
+	g_pProcSendMessageTimeW = g_pUser32.NewProc("SendMessageTimeoutW")
+	g_ui32PageantSeq        atomic.Uint32
+)
+
+// copyDataStruct 는 WM_COPYDATA 로 넘기는 구조체(COPYDATASTRUCT)다.
+type copyDataStruct struct {
+	dwData uintptr
+	cbData uint32
+	lpData uintptr
 }
 
+// availableAgents 는 연결할 수 있는 에이전트를 순서대로 돌려준다: OpenSSH 에이전트 파이프 → Pageant (D107).
+func availableAgents() []agent.ExtendedAgent {
+	var slAgents []agent.ExtendedAgent
+	if conn, errOpen := os.OpenFile(g_szOpenSSHAgentPipe, os.O_RDWR, 0); errOpen == nil {
+		slAgents = append(slAgents, agent.NewClient(conn))
+	}
+	if pageantWindow() != 0 {
+		slAgents = append(slAgents, newQueryAgent(pageantQuery))
+	}
+	return slAgents
+}
+
+// pageantWindow 는 실행 중인 Pageant 의 창 핸들이다. 없으면 0.
+func pageantWindow() uintptr {
+	pName, errName := windows.UTF16PtrFromString(g_cszPageantWindow)
+	if errName != nil {
+		return 0
+	}
+	hWnd, _, _ := g_pProcFindWindowW.Call(uintptr(unsafe.Pointer(pName)), uintptr(unsafe.Pointer(pName)))
+	return hWnd
+}
+
+// pageantQuery 는 요청 메시지 하나(길이 머리 포함)를 Pageant 에 보내고 응답(길이 머리 포함)을 받는다.
+// 방법: 이름 있는 공유 메모리를 만들어 요청을 쓰고, 그 이름을 WM_COPYDATA 로 Pageant 창에 보낸다.
+// Pageant 는 같은 메모리에 응답을 써 놓고 돌아온다.
+func pageantQuery(bRequest []byte) ([]byte, error) {
+	if len(bRequest) > g_ciAgentMaxMessage {
+		return nil, errAgentMessage
+	}
+	hWnd := pageantWindow()
+	if hWnd == 0 {
+		return nil, fmt.Errorf("pageant: not running")
+	}
+	// 이름은 요청마다 달라야 한다(여러 접속이 동시에 물을 수 있다).
+	szMapName := fmt.Sprintf("PageantRequest%08x%04x", windows.GetCurrentThreadId(), g_ui32PageantSeq.Add(1)&0xffff)
+	pMapName, errName := windows.UTF16PtrFromString(szMapName)
+	if errName != nil {
+		return nil, errName
+	}
+	hMap, errMap := windows.CreateFileMapping(windows.InvalidHandle, nil, windows.PAGE_READWRITE, 0, g_ciAgentMaxMessage, pMapName)
+	if errMap != nil && hMap == 0 {
+		return nil, fmt.Errorf("pageant: %w", errMap)
+	}
+	defer windows.CloseHandle(hMap)
+	pView, errView := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, 0)
+	if errView != nil {
+		return nil, fmt.Errorf("pageant: %w", errView)
+	}
+	defer windows.UnmapViewOfFile(pView)
+	// pView 는 Go 힙이 아닌 공유 메모리의 주소다. uintptr → Pointer 직접 변환은 vet 가 막으므로 한 번 거쳐 바꾼다.
+	bView := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&pView))), g_ciAgentMaxMessage)
+	copy(bView, bRequest)
+
+	// Pageant 는 이름을 ANSI 문자열(끝에 0)로 받는다. 이름은 ASCII 뿐이다.
+	bNameAnsi := append([]byte(szMapName), 0)
+	cds := copyDataStruct{dwData: g_cuiPageantCopyDataId, cbData: uint32(len(bNameAnsi)), lpData: uintptr(unsafe.Pointer(&bNameAnsi[0]))}
+	var uiResult uintptr
+	uiSent, _, errSend := g_pProcSendMessageTimeW.Call(hWnd, g_cuiWmCopyData, 0, uintptr(unsafe.Pointer(&cds)),
+		g_cuiSmtoAbortIfHung, g_cuiPageantTimeoutMs, uintptr(unsafe.Pointer(&uiResult)))
+	if uiSent == 0 {
+		return nil, fmt.Errorf("pageant: no answer (%v)", errSend)
+	}
+	if uiResult == 0 {
+		return nil, fmt.Errorf("pageant: request refused")
+	}
+	return checkAgentReply(bView)
+}
+
```


## C. 테스트 코드

### `internal/sshconn/agent_test.go` — 새 파일 (전체)

```go
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
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`host.agentNote`** (변경)

바꾸기 전(ko): `Windows에서는 Pageant, 그 외에는 SSH_AUTH_SOCK 에이전트를 사용합니다. 에이전트 인증이 실패하면 비밀번호를 묻습니다.`

```json
  "host.agentNote": "Windows에서는 OpenSSH 에이전트와 Pageant를, 그 외에는 SSH_AUTH_SOCK 에이전트를 사용합니다. 에이전트 인증이 실패하면 비밀번호를 묻습니다.",
  "host.agentNote": "Uses the OpenSSH agent and Pageant on Windows, and the SSH_AUTH_SOCK agent elsewhere. If agent authentication fails, you are asked for a password.",
  "host.agentNote": "Windows では OpenSSH エージェントと Pageant を、それ以外では SSH_AUTH_SOCK のエージェントを使用します。エージェント認証に失敗するとパスワードを尋ねます。",
  "host.agentNote": "在 Windows 上使用 OpenSSH 代理和 Pageant，在其他系统上使用 SSH_AUTH_SOCK 代理。代理认证失败时会询问密码。",
  "host.agentNote": "在 Windows 上使用 OpenSSH 代理程式和 Pageant，在其他系統上使用 SSH_AUTH_SOCK 代理程式。代理程式驗證失敗時會詢問密碼。",
  "host.agentNote": "Verwendet unter Windows den OpenSSH-Agenten und Pageant, sonst den Agenten aus SSH_AUTH_SOCK. Schlägt die Agent-Authentifizierung fehl, wird nach einem Passwort gefragt.",
  "host.agentNote": "Utilise l'agent OpenSSH et Pageant sous Windows, et l'agent SSH_AUTH_SOCK ailleurs. Si l'authentification par agent échoue, un mot de passe est demandé.",
  "host.agentNote": "Usa el agente de OpenSSH y Pageant en Windows, y el agente de SSH_AUTH_SOCK en otros sistemas. Si la autenticación con el agente falla, se pide una contraseña.",
  "host.agentNote": "Usa o agente OpenSSH e o Pageant no Windows, e o agente SSH_AUTH_SOCK nos demais sistemas. Se a autenticação pelo agente falhar, uma senha é solicitada.",
  "host.agentNote": "В Windows используются агент OpenSSH и Pageant, в остальных системах — агент SSH_AUTH_SOCK. Если аутентификация через агент не удалась, запрашивается пароль.",
```


## F. 명세

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -34,7 +34,7 @@
 - **볼트가 잠긴 동안에는 어떤 질문 창도 띄우지 않는다**(2026-11-17): 사용자 이름·비밀번호·키 암호·키보드-인터랙티브·호스트 키 요청이 필요해지면 접속을 멈추고 잠금 해제를 기다린다(session.md §4). 잠금 화면 뒤의 창에 자동으로 답하지 않는다. 볼트를 만들지 않았으면 지금처럼 묻는다.
 - **인증 방법을 다음 순서로 시도**한다.
   1. 계정에 개인 키가 있으면 공개 키 인증. 키가 암호로 보호되어 있고 키 암호(passphrase)가 저장돼 있지 않거나 틀리면 "키 암호 입력" 요청을 **최대 3번**. 끝내 못 읽으면 터미널에 안내 "개인 키를 읽을 수 없습니다: {오류}" 를 보이고 이 방법은 건너뜀. **읽는 형식**(2026-11-23, D106): OpenSSH·PEM, 그리고 **PuTTY PPK 형식 2·3**(키 종류 RSA·Ed25519·ECDSA P-256/384/521, 암호화 없음 또는 AES-256-CBC, 형식 3 의 키 유도는 Argon2id·Argon2i). PPK 는 파일의 무결성 값(MAC)이 맞아야 읽으며, 암호가 틀리면 MAC 이 맞지 않으므로 키 암호를 다시 묻는다. DSA 키와 Argon2d 로 만든 PPK 는 읽지 않는다(안내 줄에 이유가 나온다).
-  2. 인증 방식이 `agent` 또는 `key` 이면 SSH 에이전트가 있을 때 에이전트 인증. **Windows 에서는 OpenSSH 에이전트의 이름 있는 파이프(`\.\pipe\openssh-ssh-agent`)에만 연결한다 — 연결할 수 없으면 이 방법은 건너뛴다. (화면 문구에는 "Pageant" 가 적혀 있으나 Pageant 에는 연결하지 않는다 — 알려진 불일치, 문구는 명세 그대로 둔다.)** 그 외 OS 는 `SSH_AUTH_SOCK`(1차 범위 밖).
+  2. 인증 방식이 `agent` 또는 `key` 이면 SSH 에이전트가 있을 때 에이전트 인증. **Windows 에서는 OpenSSH 에이전트의 이름 있는 파이프(`openssh-ssh-agent`)와 실행 중인 Pageant 를 차례로 보고, 연결되는 에이전트들의 키를 모두 시도한다**(2026-11-23, D107 — 전에는 OpenSSH 파이프에만 연결했다). Pageant 와는 PuTTY 공개 문서의 방식(공유 메모리에 요청을 쓰고 창 메시지로 알림, 응답을 30초까지 기다림)으로 주고받는다. 둘 다 없으면 이 방법은 건너뛴다. 한 에이전트가 답하지 않아도 나머지 에이전트의 키는 쓴다. 그 외 OS 는 환경 변수 `SSH_AUTH_SOCK` 의 소켓에 연결한다. 알려진 한계: 앱을 관리자 권한으로 실행하면 Pageant 가 요청을 거절할 수 있다(Pageant 는 요청한 쪽이 같은 사용자인지 확인한다).
   3. 계정에 비밀번호가 있으면 비밀번호 인증.
   4. 키보드-인터랙티브: 질문이 **하나이고 에코가 없고 질문에 "pass" 가 들어 있으면** 비밀번호 요청 화면으로 처리(아래), 그 외는 "추가 인증" 요청(서버가 준 안내문·질문·에코 여부 그대로).
   5. 비밀번호 직접 입력: 서버 이름·사용자 이름이 들어간 안내 `{사용자}@{호스트} 비밀번호` 로 비밀번호 요청. **실패하면 최대 4번까지 다시 묻는다.** 두 번째 시도부터 요청에 "실패" 표시(failed).
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -33,7 +33,7 @@
   - 편집으로 열 때 이 서버가 쓰는 계정이 있고 **계정 이름이 이 서버 별칭과 다르며 같은 계정을 쓰는 서버가 2개 이상**이면 처음부터 "저장된 계정 사용"으로 시작한다.
   - **이 서버 전용 + 비밀번호**: 입력창 "비밀번호"(가려짐). 저장된 계정이 있으면 플레이스홀더 "••••••••  (저장됨)" 과 도움말 "저장된 비밀번호가 있습니다. 바꾸려면 새로 입력하세요.", 없으면 플레이스홀더 "비밀번호" 와 도움말 "비워 두면 처음 연결할 때 묻고, 저장 여부를 선택할 수 있습니다."
   - **이 서버 전용 + 개인 키**: [키 파일 선택] 버튼 + 선택한 경로(없으면 저장된 키가 있을 때 "저장된 키 사용 중", 아니면 "선택된 키 없음") / "또는 키 내용 붙여넣기"(4줄 입력, 플레이스홀더 `-----BEGIN OPENSSH PRIVATE KEY-----`) / "키 암호 (있으면)". 도움말 "OpenSSH, PEM, PuTTY PPK(v2·v3) 지원. 키 내용은 암호화되어 저장됩니다." PuTTY PPK 는 형식 2·3 모두 변환 없이 그대로 저장한다(2026-11-23, D106 — 전에는 v3 를 막고 변환을 안내했다).
-  - **SSH 에이전트**: 안내 박스 "Windows에서는 Pageant, 그 외에는 SSH_AUTH_SOCK 에이전트를 사용합니다. 에이전트 인증이 실패하면 비밀번호를 묻습니다."
+  - **SSH 에이전트**: 안내 박스 "Windows에서는 OpenSSH 에이전트와 Pageant를, 그 외에는 SSH_AUTH_SOCK 에이전트를 사용합니다. 에이전트 인증이 실패하면 비밀번호를 묻습니다."
   - **매번 묻기**: 안내 박스 "연결할 때마다 비밀번호를 묻습니다. 입력 창에서 "저장"을 선택하면 다음부터 자동으로 로그인됩니다."
 
 ### 1.3 "고급" 탭
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 ([internal/sshconn/agent_test.go](../../internal/sshconn/agent_test.go)) | 확인하는 것 |
|---|---|
| `TestQueryAgentListsAndSigns` | 가짜 Pageant(요청마다 `agent.ServeAgent` 를 한 번 돌림)를 통해 키 목록을 받고, 그 키로 한 서명이 실제 공개 키로 확인된다. 요청마다 한 번씩만 묻는다 |
| `TestQueryConnAssemblesSplitWrites` | 요청이 여러 번의 `Write` 로 나뉘어 와도 메시지 하나가 완성됐을 때 한 번만 묻는다 |
| `TestQueryConnRejectsBadLengths` | 길이 0·너무 큰 길이는 묻지 않고 오류 / 응답 길이 머리가 실제보다 길면 오류 / 응답은 길이만큼만 자른다 |

```bash
go vet ./internal/sshconn/      # unsafe 사용 경고 없음
go test ./internal/sshconn/     # 통과
GOOS=linux go build ./internal/sshconn/   # 통과 (agent_other.go)
```

Windows 쪽 `pageantQuery`(공유 메모리·창 메시지)는 Pageant 가 있어야 돌므로 **자동 테스트가 없다**. 이 PC 에는 PuTTY 가 설치되어 있지 않다.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | Pageant 를 띄우고 키 하나를 올린다. 서버의 인증 방식을 "SSH 에이전트"로 하고 연결 | 비밀번호를 묻지 않고 접속된다 |
| 2 | Pageant 에서 "키 사용 전 확인"을 켠 키 | Pageant 의 확인 창이 뜨고, 허용하면 접속된다 |
| 3 | Pageant 를 끄고 Windows OpenSSH 에이전트(`ssh-add`)만 쓴다 | 전과 같이 접속된다 |
| 4 | 둘 다 끈다 | 에이전트 방법을 건너뛰고 비밀번호를 묻는다(전과 같음) |
| 5 | 서로 다른 키를 Pageant 와 OpenSSH 에이전트에 하나씩 올리고, 각 키만 받는 서버 두 대에 연결 | 둘 다 접속된다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| Pageant 가 없는 PC | `FindWindowW` 가 0 을 돌려주면 아무 일도 하지 않는다 | — |
| Pageant 가 멈춰 있을 때 | 접속이 최대 30초 늦어질 수 있다(그 뒤 다음 방법으로) | `SMTO_ABORTIFHUNG` |
| `unsafe` | 공유 메모리 주소를 슬라이스로 보는 한 곳. `go vet` 통과 형태로 썼다 | 크기는 상수(262144), 응답 길이를 검사한 뒤 복사 |
| 서버의 인증 시도 횟수 제한 | 두 에이전트의 키를 모두 시도하므로 키가 아주 많으면 서버의 `MaxAuthTries` 에 걸릴 수 있다 | 전부터 있던 성질(OpenSSH 에이전트 하나로도 같다) |

## J. 체크리스트

- [x] `agent.go`(공통 어댑터), `agent_windows.go`(OpenSSH 파이프 + Pageant), `agent_other.go`(`SSH_AUTH_SOCK`)
- [x] 문구 `host.agentNote` 변경(10개 언어), 명세 protocols.md §2.2 2번·ui/host-editor.md §1.2, decisions D107
- [x] 자동 테스트 3개 통과, 리눅스 빌드 통과
- [ ] 수동 검증 1~5 (Pageant 실물)
