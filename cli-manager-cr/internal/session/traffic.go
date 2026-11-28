package session

import (
	"sync"
	"sync/atomic"
	"time"
)

// G_cdurTrafficInterval 은 세션 전송량 이벤트(session.traffic)를 보내는 간격이다 (session.md §8, D160).
const G_cdurTrafficInterval = time.Second

// TrafficInfo 는 세션 하나의 전송량이다 (spec/ipc.md session.traffic, D160).
// 세션을 연 뒤 연결에서 읽은(In)·연결에 쓴(Out) 바이트 누적이고, 재접속해도 이어서 센다.
// SSH 는 채널 데이터 기준이다(암호화·프로토콜 머리 제외). SFTP·포트 포워딩은 따로 센다(여기에 들지 않는다).
type TrafficInfo struct {
	M_i64In  int64 `json:"in"`
	M_i64Out int64 `json:"out"`
}

// trafficCounter 는 세션 하나의 전송량 집계다. 읽기 루프·입력·ZMODEM 이 동시에 더하므로 atomic 이다.
type trafficCounter struct {
	in  atomic.Int64
	out atomic.Int64
}

func (pCounter *trafficCounter) addIn(nRead int) {
	if nRead > 0 {
		pCounter.in.Add(int64(nRead))
	}
}

func (pCounter *trafficCounter) addOut(nWritten int) {
	if nWritten > 0 {
		pCounter.out.Add(int64(nWritten))
	}
}

func (pCounter *trafficCounter) snapshot() TrafficInfo {
	return TrafficInfo{M_i64In: pCounter.in.Load(), M_i64Out: pCounter.out.Load()}
}

// Traffic 은 세션을 연 뒤 지금까지의 전송량이다.
func (pSession *Session) Traffic() TrafficInfo {
	return pSession.traffic.snapshot()
}

// send 는 연결에 바이트를 쓰고, 성공하면 보낸 양으로 센다. 세션이 연결에 쓰는 곳은 모두 이것을 거친다.
func (pSession *Session) send(conn Transport, bData []byte) error {
	errWrite := conn.Write(bData)
	if errWrite == nil {
		pSession.traffic.addOut(len(bData))
	}
	return errWrite
}

// trafficLoop 는 Manager 의 주기 보고 상태다.
type trafficLoop struct {
	once   sync.Once
	chStop chan struct{}
	// mapSent 는 마지막으로 보낸 값이다. 보고 고루틴(또는 시험의 collectTraffic 직접 호출) 하나만 쓴다.
	mapSent map[string]TrafficInfo
}

// StartTraffic 은 durInterval 마다 값이 바뀐 세션의 전송량을 session.traffic 이벤트 하나로 보낸다 (D160).
// 바뀐 세션이 없으면 보내지 않는다. StopTraffic 으로 멈춘다. 두 번 불러도 한 번만 돈다.
func (pManager *Manager) StartTraffic(durInterval time.Duration) {
	pLoop := &pManager.traffic
	pLoop.once.Do(func() {
		chStop := make(chan struct{})
		pManager.mtxData.Lock()
		pLoop.chStop = chStop
		pManager.mtxData.Unlock()
		go func() {
			ticker := time.NewTicker(durInterval)
			defer ticker.Stop()
			for {
				select {
				case <-chStop:
					return
				case <-ticker.C:
				}
				if mapChanged := pManager.collectTraffic(); len(mapChanged) > 0 {
					pManager.emit("session.traffic", mapChanged)
				}
			}
		}()
	})
}

// StopTraffic 은 주기 보고를 멈춘다. 시작하지 않았거나 이미 멈췄으면 아무 일도 하지 않는다.
func (pManager *Manager) StopTraffic() {
	pManager.mtxData.Lock()
	chStop := pManager.traffic.chStop
	pManager.traffic.chStop = nil
	pManager.mtxData.Unlock()
	if chStop != nil {
		close(chStop)
	}
}

// collectTraffic 은 마지막 보고 뒤 값이 바뀐 세션의 전송량을 돌려준다. 닫힌 세션의 기록은 지운다.
func (pManager *Manager) collectTraffic() map[string]TrafficInfo {
	pManager.mtxData.Lock()
	mapNow := make(map[string]TrafficInfo, len(pManager.mapSessions))
	for szId, pSession := range pManager.mapSessions {
		mapNow[szId] = pSession.Traffic()
	}
	pManager.mtxData.Unlock()

	pLoop := &pManager.traffic
	if pLoop.mapSent == nil {
		pLoop.mapSent = map[string]TrafficInfo{}
	}
	for szId := range pLoop.mapSent {
		if _, bOpen := mapNow[szId]; !bOpen {
			delete(pLoop.mapSent, szId)
		}
	}
	mapChanged := map[string]TrafficInfo{}
	for szId, info := range mapNow {
		if pLoop.mapSent[szId] != info {
			mapChanged[szId] = info
			pLoop.mapSent[szId] = info
		}
	}
	return mapChanged
}
