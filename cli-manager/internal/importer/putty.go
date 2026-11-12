// Package importer 는 PuTTY 세션 / ~/.ssh/config / AWS EC2 에서 서버 후보를 읽어 온다 (v1.1.0 importers.ts·aws.ts).
package importer

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"

	"termgo/internal/vault"
)

// Candidate 는 ImportCandidate(shared/types.ts). Host 는 Partial<Host> 라서 map 으로 둔다.
type Candidate struct {
	Source  string         `json:"source"` // putty | ssh-config | aws
	Name    string         `json:"name"`
	Host    map[string]any `json:"host"`
	KeyFile string         `json:"keyFile,omitempty"`
	JumpVia string         `json:"jumpVia,omitempty"`
	Note    string         `json:"note,omitempty"`
}

// RegValues 는 레지스트리 키 하나(또는 Unix 세션 파일 하나)의 값들. 값은 string 또는 int.
type RegValues map[string]any

func (v RegValues) str(sz string) string {
	switch x := v[sz].(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	}
	return ""
}

func (v RegValues) num(sz string, def int) int {
	switch x := v[sz].(type) {
	case int:
		return x
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
			return n
		}
	}
	return def
}

// decodeSmart 는 UTF-8 이 유효하면 그대로, 아니면 CP949 로 해석한다.
func decodeSmart(arr []byte) string {
	if utf8.Valid(arr) {
		return string(arr)
	}
	if out, err := korean.EUCKR.NewDecoder().Bytes(arr); err == nil {
		return string(out)
	}
	return string(arr)
}

var reHex2 = regexp.MustCompile(`^[0-9A-Fa-f]{2}$`)

// DecodeSessionName 은 PuTTY 가 %XX 로 이스케이프한 세션 이름을 복원한다.
func DecodeSessionName(szRaw string) string {
	var bytes []byte
	for i := 0; i < len(szRaw); i++ {
		if szRaw[i] == '%' && i+2 < len(szRaw)+0 && i+3 <= len(szRaw) && reHex2.MatchString(szRaw[i+1:i+3]) {
			n, _ := strconv.ParseUint(szRaw[i+1:i+3], 16, 8)
			bytes = append(bytes, byte(n))
			i += 2
			continue
		}
		bytes = append(bytes, szRaw[i])
	}
	return decodeSmart(bytes)
}

var g_mCodepage = map[string]string{
	"UTF-8": "utf-8", "CP949": "cp949", "EUC-KR": "euc-kr", "Shift_JIS": "shift_jis",
	"ISO-8859-1:1998 (Latin-1, West Europe)": "latin1",
}

var reForward = regexp.MustCompile(`^[46]?([LRD])(?:([^:=]+):)?(\d+)=?([^:]*):?(\d*)$`)

// PuttyToCandidates 는 {세션 이름(이스케이프 상태): 값들} 을 후보로 바꾼다.
func PuttyToCandidates(mSessions map[string]RegValues) []Candidate {
	out := []Candidate{}
	for szRaw, v := range mSessions {
		szName := DecodeSessionName(szRaw)
		if szName == "Default Settings" && v.str("HostName") == "" {
			continue
		}
		szProto := v.str("Protocol")
		if szProto == "" {
			szProto = "ssh"
		}
		szP := "ssh"
		if szProto == "telnet" || szProto == "serial" {
			szP = szProto
		}
		if szP != "serial" && v.str("HostName") == "" {
			continue
		}
		szHost, szUser := v.str("HostName"), v.str("UserName")
		if i := strings.Index(szHost, "@"); i >= 0 {
			if szUser == "" {
				szUser = szHost[:i]
			}
			szHost = szHost[i+1:]
		}
		forwards := []map[string]any{}
		for _, part := range strings.Split(v.str("PortForwardings"), ",") {
			m := reForward.FindStringSubmatch(part)
			if m == nil || m[1] == "D" {
				continue
			}
			bind := m[2]
			if bind == "" {
				bind = "127.0.0.1"
			}
			dest := m[4]
			if dest == "" {
				dest = "localhost"
			}
			bp, _ := strconv.Atoi(m[3])
			dp, _ := strconv.Atoi(m[5])
			forwards = append(forwards, map[string]any{"id": vault.NewUUID(), "type": m[1], "bindHost": bind, "bindPort": bp, "destHost": dest, "destPort": dp})
		}
		var mSerial map[string]any
		if szP == "serial" {
			parity := []string{"none", "odd", "even", "mark", "space"}
			iPar := v.num("SerialParity", 0)
			szPar := "none"
			if iPar >= 0 && iPar < len(parity) {
				szPar = parity[iPar]
			}
			stop := 1.0
			switch v.num("SerialStopHalfbits", 2) {
			case 3:
				stop = 1.5
			case 4:
				stop = 2
			}
			flow := "none"
			switch v.num("SerialFlowControl", 1) {
			case 2:
				flow = "rtscts"
			case 1:
				flow = "xonxoff"
			}
			line := v.str("SerialLine")
			if line == "" {
				line = "COM1"
			}
			mSerial = map[string]any{
				"path": line, "baudRate": v.num("SerialSpeed", 9600), "dataBits": v.num("SerialDataBits", 8),
				"parity": szPar, "stopBits": stop, "flowControl": flow, "enterSends": "CR", "localEcho": v.num("LocalEcho", 2) == 0,
			}
			szHost = line
		}
		szCP := v.str("LineCodePage")
		if szCP == "" {
			szCP = "UTF-8"
		}
		szEnc, bOk := g_mCodepage[szCP]
		if !bOk {
			szEnc = "utf-8"
			if m, _ := regexp.MatchString(`(?i)949|euc-kr`, szCP); m {
				szEnc = "cp949"
			}
		}
		szTerm := v.str("TerminalType")
		if szTerm == "" || szTerm == "xterm" {
			szTerm = "xterm-256color"
		}
		iPort := v.num("PortNumber", 22)
		if szP == "telnet" && v.str("PortNumber") == "" {
			iPort = 23
		}
		iKeep := v.num("PingIntervalSecs", v.num("PingInterval", 0))
		if iKeep == 0 {
			iKeep = 30
		}
		szAuth := "password"
		if v.str("PublicKeyFile") != "" {
			szAuth = "key"
		}
		host := map[string]any{
			"alias": szName, "protocol": szP, "host": szHost, "port": iPort, "username": szUser,
			"termType": szTerm, "keepaliveSec": iKeep, "encoding": szEnc, "forwards": forwards, "authType": szAuth,
		}
		if s := v.str("RemoteCommand"); s != "" {
			host["startupCommand"] = s
		}
		if mSerial != nil {
			host["serial"] = mSerial
		}
		out = append(out, Candidate{Source: "putty", Name: szName, KeyFile: v.str("PublicKeyFile"), Host: host})
	}
	return out
}
