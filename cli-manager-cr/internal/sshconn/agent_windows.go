//go:build windows

package sshconn

import (
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"

	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows"
)

// g_szOpenSSHAgentPipe 은 OpenSSH 에이전트의 이름 있는 파이프다 (protocols.md §2.2).
const g_szOpenSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// Pageant 와 통신할 때 쓰는 값이다 (PuTTY 공개 문서 "Pageant 프로토콜": 창 메시지 WM_COPYDATA + 공유 메모리).
const (
	g_cszPageantWindow     = "Pageant"  // 창 클래스 이름이자 창 제목
	g_cuiPageantCopyDataId = 0x804e50ba // COPYDATASTRUCT.dwData 에 넣는 식별값
	g_cuiWmCopyData        = 0x004A
	g_cuiSmtoAbortIfHung   = 0x0002
	g_cuiPageantTimeoutMs  = 30000 // Pageant 가 키 사용 확인 창을 띄울 수 있어 넉넉히 기다린다
)

var (
	g_pUser32               = windows.NewLazySystemDLL("user32.dll")
	g_pProcFindWindowW      = g_pUser32.NewProc("FindWindowW")
	g_pProcSendMessageTimeW = g_pUser32.NewProc("SendMessageTimeoutW")
	g_ui32PageantSeq        atomic.Uint32
)

// copyDataStruct 는 WM_COPYDATA 로 넘기는 구조체(COPYDATASTRUCT)다.
type copyDataStruct struct {
	dwData uintptr
	cbData uint32
	lpData uintptr
}

// availableAgents 는 연결할 수 있는 에이전트를 순서대로 돌려준다: OpenSSH 에이전트 파이프 → Pageant (D107).
func availableAgents() []agent.ExtendedAgent {
	var slAgents []agent.ExtendedAgent
	if conn, errOpen := os.OpenFile(g_szOpenSSHAgentPipe, os.O_RDWR, 0); errOpen == nil {
		slAgents = append(slAgents, agent.NewClient(conn))
	}
	if pageantWindow() != 0 {
		slAgents = append(slAgents, newQueryAgent(pageantQuery))
	}
	return slAgents
}

// pageantWindow 는 실행 중인 Pageant 의 창 핸들이다. 없으면 0.
func pageantWindow() uintptr {
	pName, errName := windows.UTF16PtrFromString(g_cszPageantWindow)
	if errName != nil {
		return 0
	}
	hWnd, _, _ := g_pProcFindWindowW.Call(uintptr(unsafe.Pointer(pName)), uintptr(unsafe.Pointer(pName)))
	return hWnd
}

// pageantQuery 는 요청 메시지 하나(길이 머리 포함)를 Pageant 에 보내고 응답(길이 머리 포함)을 받는다.
// 방법: 이름 있는 공유 메모리를 만들어 요청을 쓰고, 그 이름을 WM_COPYDATA 로 Pageant 창에 보낸다.
// Pageant 는 같은 메모리에 응답을 써 놓고 돌아온다.
func pageantQuery(bRequest []byte) ([]byte, error) {
	if len(bRequest) > g_ciAgentMaxMessage {
		return nil, errAgentMessage
	}
	hWnd := pageantWindow()
	if hWnd == 0 {
		return nil, fmt.Errorf("pageant: not running")
	}
	// 이름은 요청마다 달라야 한다(여러 접속이 동시에 물을 수 있다).
	szMapName := fmt.Sprintf("PageantRequest%08x%04x", windows.GetCurrentThreadId(), g_ui32PageantSeq.Add(1)&0xffff)
	pMapName, errName := windows.UTF16PtrFromString(szMapName)
	if errName != nil {
		return nil, errName
	}
	hMap, errMap := windows.CreateFileMapping(windows.InvalidHandle, nil, windows.PAGE_READWRITE, 0, g_ciAgentMaxMessage, pMapName)
	if errMap != nil && hMap == 0 {
		return nil, fmt.Errorf("pageant: %w", errMap)
	}
	defer windows.CloseHandle(hMap)
	pView, errView := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, 0)
	if errView != nil {
		return nil, fmt.Errorf("pageant: %w", errView)
	}
	defer windows.UnmapViewOfFile(pView)
	// pView 는 Go 힙이 아닌 공유 메모리의 주소다. uintptr → Pointer 직접 변환은 vet 가 막으므로 한 번 거쳐 바꾼다.
	bView := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&pView))), g_ciAgentMaxMessage)
	copy(bView, bRequest)

	// Pageant 는 이름을 ANSI 문자열(끝에 0)로 받는다. 이름은 ASCII 뿐이다.
	bNameAnsi := append([]byte(szMapName), 0)
	cds := copyDataStruct{dwData: g_cuiPageantCopyDataId, cbData: uint32(len(bNameAnsi)), lpData: uintptr(unsafe.Pointer(&bNameAnsi[0]))}
	var uiResult uintptr
	uiSent, _, errSend := g_pProcSendMessageTimeW.Call(hWnd, g_cuiWmCopyData, 0, uintptr(unsafe.Pointer(&cds)),
		g_cuiSmtoAbortIfHung, g_cuiPageantTimeoutMs, uintptr(unsafe.Pointer(&uiResult)))
	if uiSent == 0 {
		return nil, fmt.Errorf("pageant: no answer (%v)", errSend)
	}
	if uiResult == 0 {
		return nil, fmt.Errorf("pageant: request refused")
	}
	return checkAgentReply(bView)
}
