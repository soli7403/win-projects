package session

import (
	"dariterm/internal/i18n"
	"strings"
	"testing"
	"time"
)

// 포트 포워딩 안내는 포워딩마다 한 줄씩 들여 쓰고, 듣는 주소의 폭을 맞춘다 (protocols.md §2.5, D136).
func TestMsgForwardsLines(t *testing.T) {
	i18n.SetLang("ko")
	szOut := MsgForwards([]ForwardInfo{
		{M_szType: "L", M_szBind: "127.0.0.1:25095", M_szDest: "100.102.205.95:22"},
		{M_szType: "L", M_szBind: "127.0.0.1:80", M_szDest: "10.0.0.1:8080"},
		{M_szType: "D", M_szBind: "127.0.0.1:1080"},
	})
	slLines := strings.Split(strings.TrimSuffix(szOut, "\r\n"), "\r\n")
	if len(slLines) != 4 {
		t.Fatalf("줄 수 = %d, 4 를 기대: %q", len(slLines), szOut)
	}
	if !strings.Contains(slLines[0], "● 포트 포워딩 3개") {
		t.Errorf("첫 줄 = %q", slLines[0])
	}
	if !strings.Contains(slLines[1], "    L  127.0.0.1:25095  →  100.102.205.95:22") {
		t.Errorf("둘째 줄 = %q", slLines[1])
	}
	// 짧은 주소는 가장 긴 주소(15자)에 맞춰 공백을 채운다.
	if !strings.Contains(slLines[2], "    L  127.0.0.1:80     →  10.0.0.1:8080") {
		t.Errorf("셋째 줄 = %q", slLines[2])
	}
	if !strings.Contains(slLines[3], "    D  127.0.0.1:1080   →  SOCKS 프록시") {
		t.Errorf("넷째 줄 = %q", slLines[3])
	}
	for iIndex, szLine := range slLines {
		if !strings.HasSuffix(szLine, G_cszColorReset) {
			t.Errorf("%d 번째 줄이 색을 되돌리지 않는다: %q", iIndex, szLine)
		}
	}
}

// 실패한 포워딩은 빨간 줄로 까닭을 적고, 첫 줄에 실패 수를 적는다.
func TestMsgForwardsFailed(t *testing.T) {
	i18n.SetLang("ko")
	szOut := MsgForwards([]ForwardInfo{
		{M_szType: "L", M_szBind: "127.0.0.1:3307", M_szDest: "db:3306"},
		{M_szType: "R", M_szBind: "0.0.0.0:9000", M_szDest: "localhost:9000", M_szError: "bind: address already in use"},
	})
	if !strings.Contains(szOut, G_cszColorWarn+"● 포트 포워딩 2개 (실패 1개)") {
		t.Errorf("첫 줄에 실패 수가 없다: %q", szOut)
	}
	if !strings.Contains(szOut, G_cszColorError+"    R  0.0.0.0:9000    ✕  실패: bind: address already in use") {
		t.Errorf("실패 줄이 다르다: %q", szOut)
	}
}

func TestMsgForwardsEmpty(t *testing.T) {
	if szOut := MsgForwards(nil); szOut != "" {
		t.Errorf("포워딩이 없으면 빈 글자여야 한다: %q", szOut)
	}
}

// forwardTransport 는 포워딩 결과를 알려 주는 테스트용 연결이다.
type forwardTransport struct {
	*fakeTransport
	slForwards []ForwardInfo
}

func (pFake *forwardTransport) Forwards() []ForwardInfo { return pFake.slForwards }

// 접속에 성공하면 연결이 알려 준 포워딩 결과가 세션 정보에 실린다 (session.md §1 forwards).
func TestSessionInfoCarriesForwards(t *testing.T) {
	pFake := &forwardTransport{
		fakeTransport: newFakeTransport(),
		slForwards:    []ForwardInfo{{M_szType: "L", M_szBind: "127.0.0.1:8080", M_szDest: "localhost:80"}},
	}
	pRec := &eventRecorder{}
	pManager := NewManager(pRec.emit)
	info, errOpen := pManager.Open(OpenConfig{
		Info:      Info{M_szTitle: "fwd", M_szProtocol: "ssh"},
		Dialer:    func(notify func(string)) (Transport, error) { return pFake, nil },
		OpenDelay: time.Millisecond,
	})
	if errOpen != nil {
		t.Fatalf("Open: %v", errOpen)
	}
	defer pManager.Close(info.M_szId)
	pSession, _ := pManager.Get(info.M_szId)
	tDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(tDeadline) {
		pSession.mtxData.Lock()
		szState, slForwards := pSession.info.M_szState, pSession.info.M_slForwards
		pSession.mtxData.Unlock()
		if szState == G_cszStateConnected {
			if len(slForwards) != 1 || slForwards[0].M_szBind != "127.0.0.1:8080" {
				t.Fatalf("forwards = %+v", slForwards)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("연결되지 않았다")
}
