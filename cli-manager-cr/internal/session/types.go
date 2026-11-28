package session

import (
	"errors"
	"time"
)

// Transport 는 프로토콜 연결 하나다 (SSH·텔넷·시리얼·로컬 셸이 구현한다).
// 접속은 Dialer 가 끝낸 뒤 돌려주므로 Transport 는 이미 연결된 상태다.
type Transport interface {
	// Read 는 출력을 읽는다. 연결이 끝나면 오류를 돌려준다 (io.EOF 는 이유 없음).
	Read(bBuf []byte) (int, error)
	// Write 는 입력 바이트를 보낸다 (이미 서버 인코딩으로 바뀐 바이트).
	Write(bData []byte) error
	// Resize 는 터미널 크기를 알린다. 알릴 수 없는 프로토콜은 nil 을 돌려준다.
	Resize(iCols int, iRows int) error
	// Close 는 연결을 닫는다.
	Close() error
}

// Dialer 는 접속을 시도하고 성공하면 Transport 를 돌려준다. notify 는 터미널 안내 줄을 보낸다.
// 앱 잠금(볼트)이 풀려야 접속할 수 있으면 ErrWaitUnlock(을 감싼 오류)을 돌려준다.
type Dialer func(notify func(string)) (Transport, error)

// ErrWaitUnlock 은 앱 잠금이 풀리기를 기다려야 접속할 수 있다는 뜻이다 (session.md §4 잠금 해제 대기).
// 세션은 실패로 세지 않고 "재접속 중"으로 기다리다가 Manager.ResumeUnlockWaiters 로 다시 접속한다.
var ErrWaitUnlock = errors.New("waiting for app unlock")

// Info 는 session.open 결과와 session.state 이벤트의 세션 정보다 (session.md §1, spec/ipc.md §2).
type Info struct {
	M_szId           string  `json:"id"`
	M_pszHostId      *string `json:"hostId"`
	M_szTitle        string  `json:"title"`
	M_szProtocol     string  `json:"protocol"`
	M_szTarget       string  `json:"target"`
	M_szColor        string  `json:"color,omitempty"`
	M_szEnv          string  `json:"env,omitempty"`
	M_szState        string  `json:"state"`
	M_i64ConnectedAt int64   `json:"connectedAt,omitempty"`
	M_szMessage      string  `json:"message,omitempty"`
	M_bCanSftp       bool    `json:"canSftp"`
	M_bKeepSession   bool    `json:"keepSession,omitempty"`
	// M_slForwards 는 지금 연결에 설정한 포트 포워딩과 그 결과다 (session.md §1, D136). 없으면 싣지 않는다.
	M_slForwards []ForwardInfo `json:"forwards,omitempty"`
	// M_bHexView 는 받은 출력을 HEX 로 보이는 중인지다 (serial-tools.md §1, D163). 꺼져 있으면 싣지 않는다.
	M_bHexView bool `json:"hexView,omitempty"`
}

// ForwardInfo 는 포트 포워딩 하나의 설정 결과다 (session.md §1 forwards).
type ForwardInfo struct {
	M_szType  string `json:"type"`            // L | R | D
	M_szBind  string `json:"bind"`            // 듣는 주소 "호스트:포트"
	M_szDest  string `json:"dest,omitempty"`  // 대상 "호스트:포트" (D 는 없음)
	M_szError string `json:"error,omitempty"` // 실패했으면 그 까닭, 성공이면 없음
	M_iConns  int    `json:"conns"`           // 이 포워딩으로 지금 이어져 있는 연결 수 (D142)
	M_i64Up   int64  `json:"up"`              // 접속한 뒤 받아들인 쪽 프로그램이 보낸 바이트 (D151)
	M_i64Down int64  `json:"down"`            // 접속한 뒤 받아들인 쪽 프로그램이 받은 바이트 (D151)
}

// ForwardReporter 는 포트 포워딩 결과를 알려 줄 수 있는 Transport 다 (SSH).
// 세션은 접속에 성공하면 이것으로 Info 의 forwards 를 채운다.
type ForwardReporter interface {
	Forwards() []ForwardInfo
}

// OpenConfig 는 세션 하나를 여는 설정이다.
type OpenConfig struct {
	Info          Info
	Encoding      string
	AutoReconnect bool
	IsSerial      bool
	Dialer        Dialer
	DataDir       string
	LogEnabled    bool
	LogAlias      string
	// LogSyncSec 은 세션 로그의 내려 쓰기 주기(초, 0~5)다. 0 이면 받는 즉시 쓰고 내려 쓴다 (session.md §6).
	LogSyncSec int
	// LogTimestamp 가 true 면 세션 로그의 줄마다 앞에 시각을 붙인다 (session.md §6, D112).
	LogTimestamp bool
	// LogAnsi 가 true 면 세션 로그에 ANSI 시퀀스와 CR 을 지우지 않고 받은 그대로 쓴다 (session.md §6, D120).
	LogAnsi bool
	// Zmodem 은 ZMODEM 파일 전송에 쓰는 앱 쪽 기능이다. nil 이면 이 세션은 ZMODEM 을 쓰지 않는다 (zmodem.md, D114).
	Zmodem *ZmodemHooks
	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
	StartupInput string
	// LoginScript 는 접속 직후 실행하는 로그인 스크립트다 (session.md §5.1, D139). 비면 쓰지 않는다.
	// 있으면 StartupInput 은 스크립트가 끝까지 실행된 뒤에 보낸다.
	LoginScript []LoginStep
	// ResolveSecret 은 로그인 스크립트의 보관함 값(LoginStep.SecretId)을 읽는다 (D148). nil 이면 그 단계에서 멈춘다.
	ResolveSecret SecretResolver
	// StartupDelay 는 접속 후 StartupInput 을 보내기까지의 대기다 (tmux 500ms, 자동 실행 명령 400ms).
	StartupDelay time.Duration
	// OpenDelay 는 열기 후 접속을 시작하기까지의 대기다 (기본 30ms, 프런트가 터미널을 등록할 시간).
	OpenDelay time.Duration
	// OnClose 는 세션을 닫을 때 한 번 부른다 (다이얼러가 들고 있던 계정 사본 지우기 등). nil 이면 부르지 않는다.
	OnClose func()
}
