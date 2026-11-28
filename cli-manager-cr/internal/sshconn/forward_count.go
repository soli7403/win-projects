package sshconn

import (
	"net"
	"sync"
	"sync/atomic"
)

// forwardStats 는 포워딩 하나의 집계다: 지금 열려 있는 연결 수(D142)와, 접속한 뒤 지금까지 오간 바이트 수(D151).
// 바이트는 이 포워딩이 받아들인 쪽(L·D 는 이 PC 의 프로그램, R 은 서버 쪽 프로그램) 연결 기준이다:
//   - up   = 그 연결에서 읽은 바이트 (프로그램 → 대상으로 보낸 것)
//   - down = 그 연결에 쓴 바이트   (대상 → 프로그램으로 받은 것)
//
// 동적(D) 포워딩은 SOCKS 협상 몇 바이트도 함께 센다.
type forwardStats struct {
	active atomic.Int64
	up     atomic.Int64
	down   atomic.Int64
}

// countingListener 는 받아들인 연결의 수와 전송량을 센다 (포트 포워딩 정보 창, D142·D151).
// 포워딩의 대기 소켓을 이것으로 감싸면 중계 코드(acceptLoop·serveSocks)를 고치지 않고 셀 수 있다:
// 연결을 받을 때 하나 늘리고, 그 연결을 닫을 때 하나 줄인다. 읽고 쓸 때 바이트를 더한다.
type countingListener struct {
	net.Listener
	pStats *forwardStats
}

// Accept 는 연결을 받아 수를 하나 늘리고, 세는 연결로 감싸 돌려준다.
func (pListener *countingListener) Accept() (net.Conn, error) {
	conn, errAccept := pListener.Listener.Accept()
	if errAccept != nil {
		return nil, errAccept
	}
	pListener.pStats.active.Add(1)
	return &countedConn{Conn: conn, pStats: pListener.pStats}, nil
}

// countedConn 은 읽고 쓴 바이트를 세고, 닫힐 때 수를 한 번만 줄이는 연결이다 (Close 가 여러 번 불려도 한 번).
// net.Conn 을 품기만 하고 ReadFrom·WriteTo 를 드러내지 않으므로 io.Copy 도 늘 Read·Write 를 거친다.
type countedConn struct {
	net.Conn
	pStats    *forwardStats
	onceClose sync.Once
}

func (pConn *countedConn) Read(bBuf []byte) (int, error) {
	nRead, errRead := pConn.Conn.Read(bBuf)
	if nRead > 0 {
		pConn.pStats.up.Add(int64(nRead))
	}
	return nRead, errRead
}

func (pConn *countedConn) Write(bBuf []byte) (int, error) {
	nWritten, errWrite := pConn.Conn.Write(bBuf)
	if nWritten > 0 {
		pConn.pStats.down.Add(int64(nWritten))
	}
	return nWritten, errWrite
}

func (pConn *countedConn) Close() error {
	pConn.onceClose.Do(func() { pConn.pStats.active.Add(-1) })
	return pConn.Conn.Close()
}
