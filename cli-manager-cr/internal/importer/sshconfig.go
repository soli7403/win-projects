package importer

import (
	"path/filepath"
	"strconv"
	"strings"
)

// sshBlock 은 Host 줄 하나가 여는 블록이다. 같은 줄의 여러 이름은 같은 옵션을 공유한다.
type sshBlock struct {
	slNames  []string
	mapOpts  map[string]string
	slLocal  []string
	slRemote []string
}

// ParseSSHConfig 은 ssh config 텍스트를 후보 목록으로 바꾼다 (import.md §2).
// szHome 은 IdentityFile 의 ~ 를 바꿀 홈 폴더다.
func ParseSSHConfig(szText string, szHome string) []Candidate {
	slBlocks := []*sshBlock{}
	var pCurrent *sshBlock
	mapFirstHostName := map[string]string{}

	for _, szRaw := range strings.Split(szText, "\n") {
		szLine := stripSSHComment(strings.TrimSpace(szRaw))
		if szLine == "" {
			continue
		}
		szKey, szValue := splitSSHLine(szLine)
		if szKey == "" {
			continue
		}
		szKeyLower := strings.ToLower(szKey)
		switch szKeyLower {
		case "host":
			// 같은 줄의 이름들은 하나의 블록으로 묶어 옵션을 공유한다.
			pCurrent = &sshBlock{slNames: strings.Fields(szValue), mapOpts: map[string]string{}}
			slBlocks = append(slBlocks, pCurrent)
			continue
		case "match":
			pCurrent = nil
			continue
		}
		if pCurrent == nil {
			continue
		}
		switch szKeyLower {
		case "localforward":
			pCurrent.slLocal = append(pCurrent.slLocal, szValue)
		case "remoteforward":
			pCurrent.slRemote = append(pCurrent.slRemote, szValue)
		default:
			if _, bSeen := pCurrent.mapOpts[szKeyLower]; !bSeen {
				pCurrent.mapOpts[szKeyLower] = szValue
			}
		}
	}

	// 첫 블록의 HostName 을 이름별로 기억한다 (ProxyJump 해석용).
	for _, pBlock := range slBlocks {
		for _, szName := range pBlock.slNames {
			if _, bSeen := mapFirstHostName[szName]; !bSeen && !hasSSHWildcard(szName) {
				mapFirstHostName[szName] = hostNameOf(pBlock, szName)
			}
		}
	}

	slOut := []Candidate{}
	for _, pBlock := range slBlocks {
		for _, szName := range pBlock.slNames {
			if hasSSHWildcard(szName) {
				continue
			}
			slOut = append(slOut, buildSSHCandidate(szName, pBlock, mapFirstHostName, szHome))
		}
	}
	return slOut
}

// hostNameOf 은 블록의 HostName (없으면 Host 이름) 을 돌려준다.
func hostNameOf(pBlock *sshBlock, szName string) string {
	if szHostName, bOk := pBlock.mapOpts["hostname"]; bOk && szHostName != "" {
		return szHostName
	}
	return szName
}

// buildSSHCandidate 는 블록 하나와 이름 하나로 후보를 만든다.
func buildSSHCandidate(szName string, pBlock *sshBlock, mapHostNames map[string]string, szHome string) Candidate {
	host := HostDraft{
		M_szAlias:    szName,
		M_szProtocol: "ssh",
		M_szHost:     hostNameOf(pBlock, szName),
		M_iPort:      22,
		M_szUsername: pBlock.mapOpts["user"],
		M_szAuthType: "password",
	}
	if szPort, bOk := pBlock.mapOpts["port"]; bOk {
		if iPort, errAtoi := strconv.Atoi(szPort); errAtoi == nil && iPort > 0 {
			host.M_iPort = iPort
		}
	}
	cand := Candidate{M_szSource: G_cszSourceSshConfig, M_szName: szName}
	if szIdentity := pBlock.mapOpts["identityfile"]; szIdentity != "" {
		cand.M_szKeyFile = expandHomeSSH(szIdentity, szHome)
		host.M_szAuthType = "key"
	}
	if szAlive, bOk := pBlock.mapOpts["serveraliveinterval"]; bOk {
		if iAlive, errAtoi := strconv.Atoi(szAlive); errAtoi == nil && iAlive > 0 {
			host.M_iKeepaliveSec = iAlive
		}
	}
	for _, szLocal := range pBlock.slLocal {
		if fwd, bOk := parseSSHForward("L", szLocal); bOk {
			host.M_slForwards = append(host.M_slForwards, fwd)
		}
	}
	for _, szRemote := range pBlock.slRemote {
		if fwd, bOk := parseSSHForward("R", szRemote); bOk {
			host.M_slForwards = append(host.M_slForwards, fwd)
		}
	}
	if szProxy := pBlock.mapOpts["proxyjump"]; szProxy != "" && !strings.EqualFold(szProxy, "none") {
		// 쉼표로 이은 여러 단계는 앞(이 PC 가 먼저 연결하는 서버)부터 차례로 거친다 (D153).
		// 대상 서버는 마지막 단계를 경유 서버로 삼고, 각 단계는 바로 앞 단계를 경유 서버로 삼는다.
		slChain := []string{}
		for _, szHop := range strings.Split(szProxy, ",") {
			if szHop = strings.TrimSpace(szHop); szHop == "" {
				continue
			}
			if szResolved, bSame := mapHostNames[szHop]; bSame {
				slChain = append(slChain, szResolved)
			} else {
				slChain = append(slChain, bareHost(szHop))
			}
		}
		cand.M_szNote = "ProxyJump " + szProxy
		host.M_szNotes = cand.M_szNote
		if len(slChain) > 0 {
			cand.M_szJumpVia = slChain[len(slChain)-1]
		}
		if len(slChain) > 1 {
			cand.M_slJumpChain = slChain
		}
	} else if szCommand := pBlock.mapOpts["proxycommand"]; szCommand != "" && !strings.EqualFold(szCommand, "none") {
		// ProxyCommand 는 서버의 프록시 명령으로 옮긴다 (D152). ProxyJump 가 있으면 그것이 이긴다 (OpenSSH 와 같음).
		host.M_pProxy = &ProxyDraft{M_szType: "command", M_szCommand: szCommand}
		cand.M_szNote = "ProxyCommand " + szCommand
		host.M_szNotes = cand.M_szNote
	}
	cand.M_host = host
	return cand
}

// parseSSHForward 는 [바인드:]포트 호스트:포트 형식을 해석한다. 바인드가 없으면 127.0.0.1.
func parseSSHForward(szType string, szValue string) (Forward, bool) {
	slFields := strings.Fields(szValue)
	if len(slFields) != 2 {
		return Forward{}, false
	}
	szBind, iBindPort, bBindOk := splitHostPort(slFields[0])
	if !bBindOk {
		return Forward{}, false
	}
	if !strings.Contains(slFields[0], ":") {
		szBind = "127.0.0.1"
	}
	szDest, iDestPort, bDestOk := splitHostPort(slFields[1])
	if !bDestOk {
		return Forward{}, false
	}
	if szBind == "" {
		szBind = "127.0.0.1"
	}
	return Forward{
		M_szType:     szType,
		M_szBindHost: szBind,
		M_iBindPort:  iBindPort,
		M_szDestHost: szDest,
		M_iDestPort:  iDestPort,
	}, true
}

// splitHostPort 는 host:port 를 나눈다. 마지막 콜론 기준, IPv6 대괄호는 벗긴다. 포트 숫자가 아니면 bOk=false.
func splitHostPort(szValue string) (string, int, bool) {
	iColon := strings.LastIndex(szValue, ":")
	if iColon < 0 {
		iPort, errAtoi := strconv.Atoi(szValue)
		return "", iPort, errAtoi == nil
	}
	iPort, errAtoi := strconv.Atoi(szValue[iColon+1:])
	if errAtoi != nil {
		return "", 0, false
	}
	szHost := strings.TrimSuffix(strings.TrimPrefix(szValue[:iColon], "["), "]")
	return szHost, iPort, true
}

// bareHost 는 [사용자@]호스트[:포트] 에서 호스트만 취한다.
func bareHost(szValue string) string {
	if iAt := strings.LastIndex(szValue, "@"); iAt >= 0 {
		szValue = szValue[iAt+1:]
	}
	if szHost, _, bOk := splitHostPort(szValue); bOk && strings.Contains(szValue, ":") {
		return szHost
	}
	return szValue
}

// expandHomeSSH 는 값 앞의 ~ 를 홈 폴더로 바꾼다.
func expandHomeSSH(szPath string, szHome string) string {
	if szHome != "" && (szPath == "~" || strings.HasPrefix(szPath, "~/") || strings.HasPrefix(szPath, `~\`)) {
		return filepath.Join(szHome, szPath[1:])
	}
	return szPath
}

// hasSSHWildcard 는 이름에 와일드카드(*, ?)나 부정(!)이 있는지 본다.
func hasSSHWildcard(szName string) bool {
	return strings.ContainsAny(szName, "*?!")
}

// stripSSHComment 는 # 뒤를 지운다 (줄 처음이나 공백 뒤의 #만 주석으로 본다).
func stripSSHComment(szLine string) string {
	for iIndex := 0; iIndex < len(szLine); iIndex++ {
		if szLine[iIndex] == '#' && (iIndex == 0 || szLine[iIndex-1] == ' ' || szLine[iIndex-1] == '\t') {
			return strings.TrimSpace(szLine[:iIndex])
		}
	}
	return szLine
}

// splitSSHLine 은 "키 값" 또는 "키=값" 을 나눈다. 값의 양끝 큰따옴표는 뗀다.
func splitSSHLine(szLine string) (string, string) {
	iSplit := strings.IndexAny(szLine, " \t=")
	if iSplit < 0 {
		return szLine, ""
	}
	szKey := szLine[:iSplit]
	szValue := strings.TrimLeft(szLine[iSplit:], " \t=")
	szValue = strings.TrimSpace(szValue)
	if len(szValue) >= 2 && szValue[0] == '"' && szValue[len(szValue)-1] == '"' {
		szValue = szValue[1 : len(szValue)-1]
	}
	return szKey, szValue
}
