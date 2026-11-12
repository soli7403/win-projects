package session

// Snapshots 는 열린 세션의 정보 목록을 돌려준다 (종료 확인 문구용). 순서는 보장하지 않는다.
func (pManager *Manager) Snapshots() []Info {
	pManager.mtxData.Lock()
	slSessions := make([]*Session, 0, len(pManager.mapSessions))
	for _, pSession := range pManager.mapSessions {
		slSessions = append(slSessions, pSession)
	}
	pManager.mtxData.Unlock()
	slOut := make([]Info, 0, len(slSessions))
	for _, pSession := range slSessions {
		slOut = append(slOut, pSession.Snapshot())
	}
	return slOut
}
