package session

import "time"

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
type Dialer func(notify func(string)) (Transport, error)

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
	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
	StartupInput string
	// StartupDelay 는 접속 후 StartupInput 을 보내기까지의 대기다 (tmux 500ms, 자동 실행 명령 400ms).
	StartupDelay time.Duration
	// OpenDelay 는 열기 후 접속을 시작하기까지의 대기다 (기본 30ms, 프런트가 터미널을 등록할 시간).
	OpenDelay time.Duration
}
