// Package sshconn 은 SSH 접속(호스트 키 확인, 인증 순서, 점프 호스트, 포워딩, keepalive, 셸)을 구현한다 (spec/protocols.md §2).
package sshconn

import "time"

// 시간 상수 (protocols.md §2.1, §2.6).
const (
	G_cdurConnectTimeout = 20 * time.Second
	G_iKeepaliveFailMax  = 4
	G_iPassphraseMax     = 3
	G_iPasswordMax       = 4
	G_iJumpDepthMax      = 4 // 경유 서버 4단계를 넘으면 오류 (대상 기준 깊이 5)
)

// Prompter 는 사용자에게 묻는 질문 창구다 (spec/ipc.md §3 의 ui.request). 취소이면 bOk=false 다.
type Prompter interface {
	// HostKey 는 호스트 키를 신뢰할지 묻는다.
	HostKey(szSessionTitle string, szHostPort string, szKeyType string, szFingerprint string, bChanged bool, szOldFingerprint string) bool
	// Password 는 비밀번호를 묻는다. 응답은 {password, save}.
	Password(szSessionTitle string, szPrompt string, szUsername string, bFailed bool) (szPassword string, bSave bool, bOk bool)
	// Passphrase 는 개인 키 암호를 묻는다.
	Passphrase(szSessionTitle string) (string, bool)
	// Username 은 사용자 이름을 묻는다.
	Username(szSessionTitle string) (string, bool)
	// Keyboard 는 키보드-인터랙티브 질문에 답한다.
	Keyboard(szSessionTitle string, szName string, szInstructions string, slPrompts []KeyboardPrompt) ([]string, bool)
}

// KeyboardPrompt 는 키보드-인터랙티브 질문 하나다.
type KeyboardPrompt struct {
	Prompt string `json:"prompt"`
	Echo   bool   `json:"echo"`
}

// HostKeyStore 는 저장된 호스트 키(knownHosts)를 읽고 쓴다.
type HostKeyStore interface {
	// Lookup 은 host:port 의 저장된 지문을 돌려준다. 없으면 ok=false.
	Lookup(szHostPort string) (szFingerprint string, bOk bool)
	// Save 는 신뢰한 키를 기록한다 (추가 또는 갱신).
	Save(szHostPort string, szKeyType string, szFingerprint string)
}

// Target 은 접속 대상 하나다. 점프 호스트 체인은 Jump 로 이어진다.
type Target struct {
	Title        string // 세션 제목 (ask 화면에 쓴다)
	Host         string
	Port         int
	Username     string // 비면 Prompter.Username 으로 묻는다
	AuthType     string // password | key | agent | ask
	Password     string // 볼트에서 읽은 비밀번호 (없으면 "")
	PrivateKey   []byte // 볼트에서 읽은 개인 키 (없으면 nil)
	Passphrase   string // 볼트에서 읽은 키 암호 (없으면 "")
	AllowSave    bool   // 비밀번호 요청에 "암호화해 저장" 을 허용할지
	KeepaliveSec int
	TermType     string
	Jump         *Target // 경유 서버 (없으면 nil)
	Depth        int     // 점프 깊이 (내부용)
}

// Forward 는 포트 포워딩 설정 하나다 (data/store.md §3 forwards).
type Forward struct {
	Type     string // "L" 또는 "R"
	BindHost string
	BindPort int
	DestHost string
	DestPort int
}
