package sshconn

import (
	"dariterm/internal/i18n"
	"io"
	"net"
	"strconv"
	"time"
)

// StartKeepalive 는 keepalive 요청을 주기적으로 보낸다. 연속 4번 실패하면 onFail 을 부른다 (protocols.md §2.6).
// iSec 가 0 이하이면 아무 일도 하지 않는다.
func (pConn *Conn) StartKeepalive(iSec int, onFail func()) {
	if iSec <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Duration(iSec) * time.Second)
		defer ticker.Stop()
		iFailed := 0
		for range ticker.C {
			_, _, errSend := pConn.Client.SendRequest("keepalive@openssh.com", true, nil)
			if errSend != nil {
				iFailed++
			} else {
				iFailed = 0
			}
			if iFailed >= G_iKeepaliveFailMax {
				onFail()
				return
			}
		}
	}()
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
