package session

import (
	"crypto/rand"
	"dariterm/internal/i18n"
	"fmt"
	"sync"
)

// Manager 는 열린 세션을 id 로 찾는다.
type Manager struct {
	mtxData     sync.Mutex
	mapSessions map[string]*Session
	emit        EmitFunc
}

// NewManager 는 빈 세션 관리자를 만든다.
func NewManager(emit EmitFunc) *Manager {
	return &Manager{mapSessions: map[string]*Session{}, emit: emit}
}

// Open 은 세션을 하나 열고 정보를 돌려준다. 접속은 OpenDelay 뒤에 시작한다 (spec/ipc.md session.open).
func (pManager *Manager) Open(cfg OpenConfig) (Info, error) {
	szId, errId := newSessionId()
	if errId != nil {
		return Info{}, errId
	}
	cfg.Info.M_szId = szId
	pSession := newSession(cfg, pManager.emit)
	pManager.mtxData.Lock()
	pManager.mapSessions[szId] = pSession
	pManager.mtxData.Unlock()
	pSession.start()
	return pSession.Snapshot(), nil
}

// Get 은 id 의 세션을 찾는다. 없으면 오류.
func (pManager *Manager) Get(szId string) (*Session, error) {
	pManager.mtxData.Lock()
	defer pManager.mtxData.Unlock()
	pSession, bOk := pManager.mapSessions[szId]
	if !bOk {
		return nil, i18n.Errorf("be.session.notFound", nil)
	}
	return pSession, nil
}

// Close 는 세션을 닫고 목록에서 뺀다. 없는 id 는 무시한다.
func (pManager *Manager) Close(szId string) {
	pManager.mtxData.Lock()
	pSession, bOk := pManager.mapSessions[szId]
	if bOk {
		delete(pManager.mapSessions, szId)
	}
	pManager.mtxData.Unlock()
	if bOk {
		pSession.Close()
	}
}

// CloseAll 은 모든 세션을 닫는다 (앱 종료).
func (pManager *Manager) CloseAll() {
	pManager.mtxData.Lock()
	slSessions := make([]*Session, 0, len(pManager.mapSessions))
	for _, pSession := range pManager.mapSessions {
		slSessions = append(slSessions, pSession)
	}
	pManager.mapSessions = map[string]*Session{}
	pManager.mtxData.Unlock()
	for _, pSession := range slSessions {
		pSession.Close()
	}
}

// Count 는 열린 세션 수를 돌려준다 (종료 확인 문구에 쓴다).
func (pManager *Manager) Count() int {
	pManager.mtxData.Lock()
	defer pManager.mtxData.Unlock()
	return len(pManager.mapSessions)
}

// newSessionId 는 UUID v4 문자열을 만든다.
func newSessionId() (string, error) {
	bUuid := make([]byte, 16)
	if _, errRand := rand.Read(bUuid); errRand != nil {
		return "", errRand
	}
	bUuid[6] = (bUuid[6] & 0x0f) | 0x40
	bUuid[8] = (bUuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bUuid[0:4], bUuid[4:6], bUuid[6:8], bUuid[8:10], bUuid[10:16]), nil
}
