package sshconn

import (
	"dariterm/internal/i18n"
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

// StartForwards 는 포워딩을 설정한다. 성공과 실패는 notice 로 알린다 (protocols.md §2.5). 세션이 끝나면 Close 로 정리된다.
func (pConn *Conn) StartForwards(slForwards []Forward, notice func(string)) []io.Closer {
	var slClosers []io.Closer
	for _, fw := range slForwards {
		szBind := fw.BindHost
		if szBind == "" {
			szBind = "127.0.0.1"
		}
		szBindAddr := net.JoinHostPort(szBind, strconv.Itoa(fw.BindPort))
		szDest := net.JoinHostPort(fw.DestHost, strconv.Itoa(fw.DestPort))
		if fw.Type == "D" {
			// 동적 포워딩: 이 PC 의 포트가 SOCKS 프록시가 되고, 요청마다 대상을 SSH 연결로 잇는다 (D109).
			ln, errListen := net.Listen("tcp", szBindAddr)
			if errListen != nil {
				notice(i18n.T("be.fwd.dynamicFailed", "port", fw.BindPort, "error", errListen))
				continue
			}
			notice(i18n.T("be.fwd.dynamicActive", "from", szBindAddr))
			slClosers = append(slClosers, ln)
			go serveSocks(ln, pConn.Client.Dial)
			continue
		}
		if fw.Type == "R" {
			ln, errListen := pConn.Client.Listen("tcp", szBindAddr)
			if errListen != nil {
				notice(i18n.T("be.fwd.remoteFailed", "port", fw.BindPort, "error", errListen))
				continue
			}
			notice(i18n.T("be.fwd.active", "from", szBindAddr, "to", szDest))
			slClosers = append(slClosers, ln)
			go acceptLoop(ln, func() (net.Conn, error) { return net.Dial("tcp", szDest) })
			continue
		}
		ln, errListen := net.Listen("tcp", szBindAddr)
		if errListen != nil {
			notice(i18n.T("be.fwd.localFailed", "port", fw.BindPort, "error", errListen))
			continue
		}
		notice(i18n.T("be.fwd.active", "from", szBindAddr, "to", szDest))
		slClosers = append(slClosers, ln)
		client := pConn.Client
		go acceptLoop(ln, func() (net.Conn, error) { return client.Dial("tcp", szDest) })
	}
	return slClosers
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
