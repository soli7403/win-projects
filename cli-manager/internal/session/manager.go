package session

import (
	"errors"
	"sync"

	"termgo/internal/model"
	"termgo/internal/store"
	"termgo/internal/vault"
)

var g_mEnvColor = map[string]string{
	"prod": "#ef4444", "stage": "#f59e0b", "dev": "#22c55e", "device": "#3b82f6",
}

func resolveColor(s *store.Store, h model.Host) string {
	if h.Color != "" {
		return h.Color
	}
	if c, ok := g_mEnvColor[h.Env]; ok {
		return c
	}
	groups := s.Groups()
	for szGID, i := deref(h.GroupID), 0; szGID != "" && i < 32; i++ {
		var g *model.Group
		for k := range groups {
			if groups[k].ID == szGID {
				g = &groups[k]
				break
			}
		}
		if g == nil {
			break
		}
		if g.Color != "" {
			return g.Color
		}
		if c, ok := g_mEnvColor[g.Env]; ok {
			return c
		}
		szGID = deref(g.ParentID)
	}
	return ""
}

func resolveEnv(s *store.Store, h model.Host) string {
	if h.Env != "" && h.Env != "none" {
		return h.Env
	}
	groups := s.Groups()
	for szGID, i := deref(h.GroupID), 0; szGID != "" && i < 32; i++ {
		var g *model.Group
		for k := range groups {
			if groups[k].ID == szGID {
				g = &groups[k]
				break
			}
		}
		if g == nil {
			break
		}
		if g.Env != "" && g.Env != "none" {
			return g.Env
		}
		szGID = deref(g.ParentID)
	}
	return "none"
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// OpenOptions 는 session:open 인자.
type OpenOptions struct {
	HostID string       `json:"hostId"`
	Adhoc  *AdhocTarget `json:"adhoc"`
	Cols   int          `json:"cols"`
	Rows   int          `json:"rows"`
}

// AdhocTarget 은 저장하지 않고 바로 여는 접속 대상.
type AdhocTarget struct {
	Host     string `json:"host"`
	Cwd      string `json:"cwd"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Protocol string `json:"protocol"`
}

func adhocHost(t AdhocTarget) model.Host {
	szProto := t.Protocol
	if szProto == "" {
		szProto = "ssh"
	}
	h := model.Host{
		ID: "adhoc-" + vault.NewUUID(), Protocol: szProto, AuthType: "ask", Tags: []string{},
		Encoding: "utf-8", TermType: "xterm-256color", Forwards: []model.PortForward{},
	}
	if szProto == "local" {
		h.Alias = "로컬 터미널"
		h.Local = &localOpts{Cwd: t.Cwd}
		return h
	}
	h.Host, h.Username, h.KeepaliveSec = t.Host, t.Username, 30
	h.Port = t.Port
	if h.Port == 0 {
		h.Port = 22
		if szProto == "telnet" {
			h.Port = 23
		}
	}
	h.Alias = t.Host
	if t.Username != "" {
		h.Alias = t.Username + "@" + t.Host
	}
	return h
}

// Manager 는 열려 있는 세션 표.
type Manager struct {
	d  Deps
	mu sync.Mutex
	m  map[string]*Session
}

// NewManager 는 빈 매니저를 만든다.
func NewManager(d Deps) *Manager { return &Manager{d: d, m: map[string]*Session{}} }

// Open 은 세션을 만들고 곧 접속을 시작한다.
func (pstrM *Manager) Open(o OpenOptions) (Info, error) {
	var h model.Host
	bPersistent := false
	if o.HostID != "" {
		if found, ok := pstrM.d.Store.HostByID(o.HostID); ok {
			h, bPersistent = found, true
		}
	}
	if !bPersistent {
		if o.Adhoc == nil {
			return Info{}, errors.New("서버를 찾을 수 없습니다")
		}
		h = adhocHost(*o.Adhoc)
	}
	pstrS := newSession(pstrM.d, h, bPersistent)
	if o.Cols > 0 && o.Rows > 0 {
		pstrS.cols, pstrS.rows = o.Cols, o.Rows
	}
	pstrM.mu.Lock()
	pstrM.m[pstrS.info.ID] = pstrS
	pstrM.mu.Unlock()
	// 렌더러가 터미널을 등록할 시간을 잠깐 준 뒤 접속
	go func() {
		timeSleep30()
		pstrS.Connect()
	}()
	return pstrS.Info(), nil
}

// Get 은 id 로 세션을 찾는다 (없으면 nil).
func (pstrM *Manager) Get(szID string) *Session {
	pstrM.mu.Lock()
	defer pstrM.mu.Unlock()
	return pstrM.m[szID]
}

// Close 는 세션을 닫고 표에서 뺀다.
func (pstrM *Manager) Close(szID string) {
	pstrM.mu.Lock()
	s := pstrM.m[szID]
	delete(pstrM.m, szID)
	pstrM.mu.Unlock()
	if s != nil {
		s.Close()
	}
}

// CloseAll 은 모든 세션을 닫는다.
func (pstrM *Manager) CloseAll() {
	pstrM.mu.Lock()
	all := pstrM.m
	pstrM.m = map[string]*Session{}
	pstrM.mu.Unlock()
	for _, s := range all {
		s.Close()
	}
}

// ConnectedCount 는 연결된 세션 수.
func (pstrM *Manager) ConnectedCount() int {
	pstrM.mu.Lock()
	defer pstrM.mu.Unlock()
	n := 0
	for _, s := range pstrM.m {
		if s.Info().State == "connected" {
			n++
		}
	}
	return n
}

// KeepCount 는 연결된 세션 중 tmux 세션 유지 중인 수.
func (pstrM *Manager) KeepCount() int {
	pstrM.mu.Lock()
	defer pstrM.mu.Unlock()
	n := 0
	for _, s := range pstrM.m {
		if i := s.Info(); i.State == "connected" && i.KeepSession {
			n++
		}
	}
	return n
}
