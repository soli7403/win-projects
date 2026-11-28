package session

import (
	"sync"
	"testing"
	"time"
)

// trafficRecorder 는 session.traffic 이벤트만 모은다.
type trafficRecorder struct {
	eventRecorder
	mtxTraffic sync.Mutex
	slTraffic  []map[string]TrafficInfo
}

func (pRec *trafficRecorder) emit(szEvent string, anyData ...any) {
	if szEvent == "session.traffic" {
		pRec.mtxTraffic.Lock()
		pRec.slTraffic = append(pRec.slTraffic, anyData[0].(map[string]TrafficInfo))
		pRec.mtxTraffic.Unlock()
		return
	}
	pRec.eventRecorder.emit(szEvent, anyData...)
}

func (pRec *trafficRecorder) last() (map[string]TrafficInfo, int) {
	pRec.mtxTraffic.Lock()
	defer pRec.mtxTraffic.Unlock()
	if len(pRec.slTraffic) == 0 {
		return nil, 0
	}
	return pRec.slTraffic[len(pRec.slTraffic)-1], len(pRec.slTraffic)
}

// openConnected 는 가짜 연결로 세션을 열고 연결될 때까지 기다린다.
func openConnected(t *testing.T, pManager *Manager, rec *eventRecorder) (string, *fakeTransport) {
	t.Helper()
	transport := newFakeTransport()
	info, errOpen := pManager.Open(sshConfig(func(func(string)) (Transport, error) { return transport, nil }, false))
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	waitUntil(t, func() bool { return rec.hasState(G_cszStateConnected) }, time.Second)
	return info.M_szId, transport
}

// 연결에서 읽은 바이트는 in, 입력으로 쓴 바이트는 out 으로 센다 (D160). 한글은 UTF-8 바이트 수다.
func TestTrafficCountsReadAndInput(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	pSession, _ := pManager.Get(szId)

	transport.chIn <- []byte("hello")
	transport.chIn <- []byte("한글") // 6 바이트
	pSession.Input("ls\r")
	waitUntil(t, func() bool { return pSession.Traffic() == TrafficInfo{M_i64In: 11, M_i64Out: 3} }, time.Second)
	pManager.Close(szId)
	if got := pSession.Traffic(); got.M_i64In != 11 || got.M_i64Out != 3 {
		t.Fatalf("닫은 뒤에도 값은 그대로여야 함: %+v", got)
	}
}

// 보고는 값이 바뀐 세션만 싣고, 닫힌 세션의 기록은 지운다.
func TestCollectTrafficOnlyChanged(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	pSession, _ := pManager.Get(szId)

	if mapChanged := pManager.collectTraffic(); len(mapChanged) != 0 {
		t.Fatalf("아직 오간 것이 없으면 보고하지 않는다: %v", mapChanged)
	}
	transport.chIn <- []byte("abc")
	waitUntil(t, func() bool { return pSession.Traffic().M_i64In == 3 }, time.Second)
	if mapChanged := pManager.collectTraffic(); mapChanged[szId] != (TrafficInfo{M_i64In: 3}) {
		t.Fatalf("바뀐 세션을 실어야 함: %v", mapChanged)
	}
	if mapChanged := pManager.collectTraffic(); len(mapChanged) != 0 {
		t.Fatalf("그대로면 다시 싣지 않는다: %v", mapChanged)
	}
	pSession.Input("x")
	if mapChanged := pManager.collectTraffic(); mapChanged[szId] != (TrafficInfo{M_i64In: 3, M_i64Out: 1}) {
		t.Fatalf("입력도 바뀐 것으로 본다: %v", mapChanged)
	}
	pManager.Close(szId)
	pManager.collectTraffic()
	if _, bLeft := pManager.traffic.mapSent[szId]; bLeft {
		t.Fatal("닫힌 세션의 기록은 지워야 함")
	}
}

// 주기 보고가 session.traffic 하나로 보내고, StopTraffic 뒤에는 보내지 않는다. 두 번 멈춰도 괜찮다.
func TestStartTrafficEmitsAndStops(t *testing.T) {
	rec := &trafficRecorder{}
	pManager := NewManager(rec.emit)
	pManager.StartTraffic(10 * time.Millisecond)
	pManager.StartTraffic(10 * time.Millisecond) // 두 번째는 무시
	szId, transport := openConnected(t, pManager, &rec.eventRecorder)
	transport.chIn <- []byte("12345")
	waitUntil(t, func() bool {
		mapLast, _ := rec.last()
		return mapLast[szId].M_i64In == 5
	}, time.Second)
	pManager.StopTraffic()
	pManager.StopTraffic()
	_, iBefore := rec.last()
	transport.chIn <- []byte("more")
	time.Sleep(60 * time.Millisecond)
	if _, iAfter := rec.last(); iAfter != iBefore {
		t.Fatalf("멈춘 뒤에는 보내지 않아야 함: %d → %d", iBefore, iAfter)
	}
	pManager.CloseAll()
}

// ZMODEM 전송 중 엔진이 쓰는 연결도 같은 카운터로 센다.
func TestZmodemLinkCountsTraffic(t *testing.T) {
	transport := newFakeTransport()
	var counter trafficCounter
	pLink := &zmodemLink{conn: transport, pTraffic: &counter}
	if errWrite := pLink.Write([]byte("abcd")); errWrite != nil {
		t.Fatal(errWrite)
	}
	transport.chIn <- []byte("xyz")
	bBuf := make([]byte, 16)
	if nRead, errRead := pLink.Read(bBuf); errRead != nil || nRead != 3 {
		t.Fatalf("읽기: %d %v", nRead, errRead)
	}
	if got := counter.snapshot(); got != (TrafficInfo{M_i64In: 3, M_i64Out: 4}) {
		t.Fatalf("ZMODEM 연결 전송량: %+v", got)
	}
}
