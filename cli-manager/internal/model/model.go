// Package model 은 store.json 의 Host / Group 등 렌더러와 공유하는 데이터 모양(shared/types.ts)이다.
package model

import "termgo/internal/transport"

// PortForward 는 포트 포워딩 한 건.
type PortForward struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // L | R
	BindHost string `json:"bindHost"`
	BindPort int    `json:"bindPort"`
	DestHost string `json:"destHost"`
	DestPort int    `json:"destPort"`
}

// JumpHostDirect 는 jumpHostId 가 이 값일 때 Host.JumpDirect 의 베스천을 쓴다 (3.2 직접 입력).
const JumpHostDirect = "direct"

// JumpDirect 는 별도 서버로 등록하지 않고 서버 설정 안에 둔 베스천.
type JumpDirect struct {
	Host         string  `json:"host"`
	Port         int     `json:"port"`
	Username     string  `json:"username"`
	AuthType     string  `json:"authType"` // password | key
	CredentialID *string `json:"credentialId,omitempty"`
}

// Host 는 서버 항목.
type Host struct {
	ID             string                   `json:"id"`
	GroupID        *string                  `json:"groupId"`
	Alias          string                   `json:"alias"`
	Protocol       string                   `json:"protocol"` // ssh | telnet | serial | local
	Host           string                   `json:"host"`
	Port           int                      `json:"port"`
	Username       string                   `json:"username"`
	AuthType       string                   `json:"authType"` // password | key | agent | ask
	CredentialID   *string                  `json:"credentialId,omitempty"`
	JumpHostID     *string                  `json:"jumpHostId,omitempty"`
	JumpDirect     *JumpDirect              `json:"jumpDirect,omitempty"`
	Color          string                   `json:"color,omitempty"`
	Env            string                   `json:"env,omitempty"`
	Tags           []string                 `json:"tags"`
	Encoding       string                   `json:"encoding"`
	TermType       string                   `json:"termType"`
	StartupCommand string                   `json:"startupCommand,omitempty"`
	KeepSession    bool                     `json:"keepSession,omitempty"`
	TmuxName       string                   `json:"tmuxName,omitempty"`
	KeepaliveSec   int                      `json:"keepaliveSec"`
	Forwards       []PortForward            `json:"forwards"`
	Serial         *transport.SerialOptions `json:"serial,omitempty"`
	Local          *transport.LocalOptions  `json:"local,omitempty"`
	Notes          string                   `json:"notes,omitempty"`
	Favorite       bool                     `json:"favorite,omitempty"`
	LastUsedAt     int64                    `json:"lastUsedAt,omitempty"`
	Sort           int                      `json:"sort"`
}

// Group 은 서버 그룹.
type Group struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parentId"`
	Name     string  `json:"name"`
	Color    string  `json:"color,omitempty"`
	Env      string  `json:"env,omitempty"`
	Sort     int     `json:"sort"`
}

// KnownHost 는 승인된 호스트 키.
type KnownHost struct {
	HostPort    string `json:"hostPort"`
	KeyType     string `json:"keyType"`
	Fingerprint string `json:"fingerprint"`
	AddedAt     int64  `json:"addedAt"`
}
