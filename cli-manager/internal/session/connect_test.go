package session

import (
	"net"
	"testing"
	"time"

	"termgo/internal/model"
	"termgo/internal/store"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

// 3.7: 접속 오류가 연속으로 나도 앱이 멈추지 않고 해당 세션이 "error" 상태로만 남는다.
func TestConnectFailureStaysInPane(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	szAddr := l.Addr().(*net.TCPAddr)
	_ = l.Close() // 닫힌 포트

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var states []string
	br := ui.NewBridge(func(szCh string, args ...any) {
		if szCh == "session:state" {
			states = append(states, args[0].(Info).State)
		}
	})
	// 자동 재접속을 끈다
	st.Patch(map[string]rawMsg{"settings": rawMsg(`{"autoReconnect":false}`)})
	m := NewManager(Deps{Store: st, Vault: vault.Open(t.TempDir()), UI: br, DataDir: t.TempDir()})
	info, err := m.Open(OpenOptions{Adhoc: &AdhocTarget{Host: "127.0.0.1", Port: szAddr.Port, Username: "u"}})
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get(info.ID)
	var _ model.Host = s.host
	for i := 0; i < 100 && s.Info().State != "error"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if s.Info().State != "error" {
		t.Fatalf("state = %s (%v)", s.Info().State, states)
	}
	// 두 번째 오류도 같은 경로로 처리된다 (Enter 로 재접속 → 다시 실패)
	s.Write("\r")
	time.Sleep(300 * time.Millisecond)
	for i := 0; i < 100 && s.Info().State != "error"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if s.Info().State != "error" {
		t.Fatalf("second state = %s", s.Info().State)
	}
	m.CloseAll()
}
