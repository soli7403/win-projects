// Package importer 는 PuTTY·SecureCRT·Xshell·ssh config·AWS EC2 에서 서버 후보를 읽고 저장소 항목으로 적용한다 (spec/import.md).
package importer

// Forward 는 포트 포워딩 한 건이다 (data/store.md §3 forwards). id 는 적용 때 만든다.
type Forward struct {
	M_szType     string `json:"type"`
	M_szBindHost string `json:"bindHost"`
	M_iBindPort  int    `json:"bindPort"`
	M_szDestHost string `json:"destHost"`
	M_iDestPort  int    `json:"destPort"`
}

// Serial 은 시리얼 설정이다 (data/store.md §3 serial).
type Serial struct {
	M_szPath        string  `json:"path"`
	M_iBaudRate     int     `json:"baudRate"`
	M_iDataBits     int     `json:"dataBits"`
	M_szParity      string  `json:"parity"`
	M_fStopBits     float64 `json:"stopBits"`
	M_szFlowControl string  `json:"flowControl"`
	M_szEnterSends  string  `json:"enterSends"`
	M_bLocalEcho    bool    `json:"localEcho"`
}

// HostDraft 는 후보의 서버 부분 정보다 (store 서버 항목의 일부 필드).
type HostDraft struct {
	M_szAlias          string    `json:"alias"`
	M_szProtocol       string    `json:"protocol"`
	M_szHost           string    `json:"host"`
	M_iPort            int       `json:"port"`
	M_szUsername       string    `json:"username,omitempty"`
	M_szAuthType       string    `json:"authType"`
	M_iKeepaliveSec    int       `json:"keepaliveSec,omitempty"`
	M_szEncoding       string    `json:"encoding,omitempty"`
	M_szTermType       string    `json:"termType,omitempty"`
	M_szStartupCommand string    `json:"startupCommand,omitempty"`
	M_slForwards       []Forward `json:"forwards,omitempty"`
	M_pSerial          *Serial   `json:"serial,omitempty"`
	M_szNotes          string    `json:"notes,omitempty"`
	// M_slLoginScript 는 로그인 스크립트다 (data/store.md §3 loginScript, D150). 읽을 수 있는 가져오기만 채운다.
	M_slLoginScript []LoginStep `json:"loginScript,omitempty"`
	// M_pProxy 는 프록시 경유 설정이다 (지금은 ssh config 의 ProxyCommand 만, D152).
	M_pProxy *ProxyDraft `json:"proxy,omitempty"`
}

// ProxyDraft 는 가져온 서버의 프록시 경유 설정이다 (data/store.md §3 proxy).
type ProxyDraft struct {
	M_szType    string `json:"type"`              // 지금은 command 만
	M_szCommand string `json:"command,omitempty"` // 프록시 명령 줄 (%h %p %r)
}

// LoginStep 은 로그인 스크립트의 한 단계다 (data/store.md §3 loginScript).
type LoginStep struct {
	M_szExpect string `json:"expect"`
	M_szSend   string `json:"send"`
}

// Candidate 는 가져오기 후보 한 건이다 (spec/import.md 머리).
type Candidate struct {
	M_szSource  string    `json:"source"`
	M_szName    string    `json:"name"`
	M_host      HostDraft `json:"host"`
	M_szKeyFile string    `json:"keyFile,omitempty"`
	M_szJumpVia string    `json:"jumpVia,omitempty"`
	// M_slJumpChain 은 경유 서버가 여러 단계일 때 바깥(이 PC 가 먼저 연결하는 서버)부터의 주소 목록이다.
	// 마지막 항목이 M_szJumpVia 와 같다. 한 단계면 비어 있다 (ssh config 의 ProxyJump a,b, D153).
	M_slJumpChain []string `json:"jumpChain,omitempty"`
	M_szNote      string   `json:"note,omitempty"`
	// M_slFolderPath 는 넣을 폴더 아래에 만들 하위 폴더 이름들이다 (SecureCRT·Xshell, import.md §1.5·§1.6).
	M_slFolderPath []string `json:"folderPath,omitempty"`
}

// G_cszSourcePutty, G_cszSourceSecureCrt, G_cszSourceXshell, G_cszSourceSshConfig, G_cszSourceAws 는 후보 출처 이름이다.
const (
	G_cszSourcePutty     = "putty"
	G_cszSourceSecureCrt = "securecrt"
	G_cszSourceXshell    = "xshell"
	G_cszSourceSshConfig = "ssh-config"
	G_cszSourceAws       = "aws"
)
