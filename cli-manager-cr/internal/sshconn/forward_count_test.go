package sshconn

import (
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

// waitActive 는 연결 수가 iWant 가 될 때까지 기다린다.
func waitActive(t *testing.T, pStats *forwardStats, iWant int64) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if pStats.active.Load() == iWant {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("연결 수 = %d, %d 를 기대", pStats.active.Load(), iWant)
}

// waitTraffic 은 전송량이 기대값이 될 때까지 기다린다.
func waitTraffic(t *testing.T, result ForwardResult, i64WantUp int64, i64WantDown int64) {
	t.Helper()
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		if i64Up, i64Down := result.Traffic(); i64Up == i64WantUp && i64Down == i64WantDown {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	i64Up, i64Down := result.Traffic()
	t.Fatalf("전송량 = ↑%d ↓%d, ↑%d ↓%d 을 기대", i64Up, i64Down, i64WantUp, i64WantDown)
}

// echo 는 연결에 글자를 보내고 같은 글자가 되돌아오는지 본다.
func echo(t *testing.T, conn net.Conn, szText string) {
	t.Helper()
	if _, errWrite := conn.Write([]byte(szText)); errWrite != nil {
		t.Fatal(errWrite)
	}
	bBuf := make([]byte, len(szText))
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, errRead := io.ReadFull(conn, bBuf); errRead != nil || string(bBuf) != szText {
		t.Fatalf("되돌아온 것 = %q, %v", bBuf, errRead)
	}
}

// 포워딩으로 이어진 연결이 열릴 때 늘고 닫힐 때 준다 (포트 포워딩 정보 창의 "연결 N개", D142).
func TestForwardCountsOpenConnections(t *testing.T) {
	iEchoPort := startEcho(t)
	lnRaw, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	ln, pStats := counted(lnRaw)
	t.Cleanup(func() { _ = ln.Close() })
	szDest := net.JoinHostPort("127.0.0.1", strconv.Itoa(iEchoPort))
	go acceptLoop(ln, func() (net.Conn, error) { return net.DialTimeout("tcp", szDest, 2*time.Second) })

	if result := (ForwardResult{pStats: pStats}); result.Active() != 0 {
		t.Fatalf("처음 연결 수 = %d", result.Active())
	}
	connA, errA := net.Dial("tcp", lnRaw.Addr().String())
	if errA != nil {
		t.Fatal(errA)
	}
	connB, errB := net.Dial("tcp", lnRaw.Addr().String())
	if errB != nil {
		t.Fatal(errB)
	}
	waitActive(t, pStats, 2)

	// 이어져 있는 동안 데이터가 오가고, 전송량이 쌓인다 (D151).
	echo(t, connA, "ping")
	echo(t, connB, "hello!")
	result := ForwardResult{pStats: pStats}
	// 중계 쪽은 Write 가 끝난 뒤에 더하므로, 클라이언트가 받은 직후에는 아직 덜 찼을 수 있다.
	waitTraffic(t, result, 10, 10)

	_ = connA.Close()
	waitActive(t, pStats, 1)
	_ = connB.Close()
	waitActive(t, pStats, 0)
	// 연결이 닫혀도 전송량은 줄지 않는다.
	if i64Up, i64Down := result.Traffic(); i64Up != 10 || i64Down != 10 {
		t.Fatalf("닫은 뒤 전송량 = ↑%d ↓%d", i64Up, i64Down)
	}
}

// 대상에 닿지 못한 연결도 닫히면 수에서 빠진다.
func TestForwardCountDropsFailedDial(t *testing.T) {
	lnRaw, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	ln, pStats := counted(lnRaw)
	t.Cleanup(func() { _ = ln.Close() })
	go acceptLoop(ln, func() (net.Conn, error) { return nil, io.ErrClosedPipe })
	conn, errDial := net.Dial("tcp", lnRaw.Addr().String())
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	// 서버 쪽이 곧바로 닫는다: 읽기가 끝나면 수는 0 이어야 한다.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Read(make([]byte, 1))
	waitActive(t, pStats, 0)
	if i64Up, i64Down := (ForwardResult{pStats: pStats}).Traffic(); i64Up != 0 || i64Down != 0 {
		t.Fatalf("대상에 닿지 못한 연결의 전송량 = ↑%d ↓%d", i64Up, i64Down)
	}
}

// 실패한 포워딩(세는 것이 없음)의 연결 수는 0 이다.
func TestForwardResultActiveWithoutCounter(t *testing.T) {
	if iActive := (ForwardResult{Err: io.EOF}).Active(); iActive != 0 {
		t.Fatalf("= %d", iActive)
	}
	if i64Up, i64Down := (ForwardResult{Err: io.EOF}).Traffic(); i64Up != 0 || i64Down != 0 {
		t.Fatalf("전송량 = %d %d", i64Up, i64Down)
	}
}
