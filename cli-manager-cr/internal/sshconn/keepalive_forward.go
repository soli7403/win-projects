package sshconn

import (
	"io"
	"net"
	"strconv"
	"time"
)

// StartKeepalive 는 keepalive 요청을 주기적으로 보낸다. 연속 4번 실패하면 onFail 을 부른다 (protocols.md §2.6).
// 한 요청의 응답이 주기 안에 오지 않으면(네트워크가 조용히 끊김) 그 요청은 실패로 센다.
// iSec 가 0 이하이면 아무 일도 하지 않는다. 연결이 닫히면 루프를 끝낸다.
func (pConn *Conn) StartKeepalive(iSec int, onFail func()) {
	if iSec <= 0 {
		return
	}
	durInterval := time.Duration(iSec) * time.Second
	fnSend := func() error {
		_, _, errSend := pConn.Client.SendRequest("keepalive@openssh.com", true, nil)
		return errSend
	}
	go func() {
		ticker := time.NewTicker(durInterval)
		defer ticker.Stop()
		iFailed := 0
		for {
			select {
			case <-pConn.chClosed:
				return
			case <-ticker.C:
			}
			if keepaliveOnce(fnSend, durInterval, pConn.chClosed) {
				iFailed = 0
				continue
			}
			select {
			case <-pConn.chClosed:
				return // 닫혀서 실패한 것: 끊김 처리하지 않는다
			default:
			}
			iFailed++
			if iFailed >= G_iKeepaliveFailMax {
				onFail()
				return
			}
		}
	}()
}

// keepaliveOnce 는 fnSend 로 요청 하나를 보내고 durTimeout 안에 끝나면 그 결과(오류 없음 = 성공)를 돌려준다.
// 시한이 지나거나 chClosed 가 닫히면 false 다. 늦게 끝나는 fnSend 고루틴은 버퍼 1 채널에 결과를 넣고 스스로 끝난다
// (연결이 닫히면 SendRequest 도 오류로 돌아오므로 고루틴이 영원히 남지 않는다).
// 보내기 함수를 받는 이유는 테스트에서 SSH 없이 "응답 없음" 을 흉내 내기 위해서다.
func keepaliveOnce(fnSend func() error, durTimeout time.Duration, chClosed <-chan struct{}) bool {
	chResult := make(chan error, 1)
	go func() { chResult <- fnSend() }()
	timer := time.NewTimer(durTimeout)
	defer timer.Stop()
	select {
	case errSend := <-chResult:
		return errSend == nil
	case <-timer.C:
		return false
	case <-chClosed:
		return false
	}
}

// ForwardResult 는 포워딩 하나를 설정한 결과다. Err 가 nil 이면 대기 중이다.
type ForwardResult struct {
	Type string // L | R | D
	Bind string // 듣는 주소 "호스트:포트" (L·D 는 이 PC, R 은 서버)
	Dest string // 대상 "호스트:포트" (D 는 빈 값)
	Err  error
	// pStats 는 이 포워딩의 연결 수·전송량 집계다 (실패한 항목은 nil).
	pStats *forwardStats
}

// Active 는 이 포워딩으로 지금 이어져 있는 연결 수다. 실패한 포워딩은 0 이다.
func (result ForwardResult) Active() int {
	if result.pStats == nil {
		return 0
	}
	return int(result.pStats.active.Load())
}

// Traffic 은 접속한 뒤 이 포워딩으로 오간 바이트 수다 (D151). Up 은 받아들인 쪽 프로그램이 보낸 것, Down 은 받은 것.
func (result ForwardResult) Traffic() (i64Up int64, i64Down int64) {
	if result.pStats == nil {
		return 0, 0
	}
	return result.pStats.up.Load(), result.pStats.down.Load()
}

// StartForwards 는 포워딩을 설정하고, 설정마다 결과를 돌려준다 (protocols.md §2.5).
// 실패한 항목이 있어도 다음 항목으로 계속한다. 터미널 안내와 화면 표시는 부른 쪽이 결과로 만든다.
// 세션이 끝나면 돌려준 Closer 들을 닫아 정리한다.
func (pConn *Conn) StartForwards(slForwards []Forward) ([]io.Closer, []ForwardResult) {
	var slClosers []io.Closer
	slResults := make([]ForwardResult, 0, len(slForwards))
	for _, fw := range slForwards {
		szBind := fw.BindHost
		if szBind == "" {
			szBind = "127.0.0.1"
		}
		szBindAddr := net.JoinHostPort(szBind, strconv.Itoa(fw.BindPort))
		szDest := net.JoinHostPort(fw.DestHost, strconv.Itoa(fw.DestPort))
		result := ForwardResult{Type: "L", Bind: szBindAddr, Dest: szDest}
		var ln net.Listener
		switch fw.Type {
		case "D":
			// 동적 포워딩: 이 PC 의 포트가 SOCKS 프록시가 되고, 요청마다 대상을 SSH 연결로 잇는다 (D109).
			result.Type, result.Dest = "D", ""
			ln, result.Err = net.Listen("tcp", szBindAddr)
			if result.Err == nil {
				ln, result.pStats = counted(ln)
				go serveSocks(ln, pConn.Client.Dial)
			}
		case "R":
			result.Type = "R"
			ln, result.Err = pConn.Client.Listen("tcp", szBindAddr)
			if result.Err == nil {
				ln, result.pStats = counted(ln)
				go acceptLoop(ln, func() (net.Conn, error) { return net.Dial("tcp", szDest) })
			}
		default:
			ln, result.Err = net.Listen("tcp", szBindAddr)
			if result.Err == nil {
				ln, result.pStats = counted(ln)
				client := pConn.Client
				go acceptLoop(ln, func() (net.Conn, error) { return client.Dial("tcp", szDest) })
			}
		}
		if result.Err == nil {
			slClosers = append(slClosers, ln)
		}
		slResults = append(slResults, result)
	}
	return slClosers, slResults
}

// counted 는 대기 소켓을 연결 수·전송량을 세는 것으로 감싼다. 닫을 때는 돌려준 소켓을 닫으면 된다.
func counted(ln net.Listener) (net.Listener, *forwardStats) {
	pStats := &forwardStats{}
	return &countingListener{Listener: ln, pStats: pStats}, pStats
}

// acceptLoop 는 들어오는 연결마다 대상에 이어 양방향으로 복사한다.
func acceptLoop(ln net.Listener, fnDial func() (net.Conn, error)) {
	for {
		inbound, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		go func(in net.Conn) {
			out, errDial := fnDial()
			if errDial != nil {
				_ = in.Close()
				return
			}
			go func() { _, _ = io.Copy(out, in); _ = out.Close() }()
			_, _ = io.Copy(in, out)
			_ = in.Close()
		}(inbound)
	}
}
