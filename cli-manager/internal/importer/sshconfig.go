package importer

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"termgo/internal/vault"
)

var reSSHForward = regexp.MustCompile(`^(?:(\[[^\]]+\]|[^\s:]+):)?(\d+)\s+(\[[^\]]+\]|[^\s:]+):(\d+)$`)

// sshForward 는 "[bind:]port host:hostport" (LocalForward / RemoteForward 인자).
func sshForward(szType, szV string) map[string]any {
	m := reSSHForward.FindStringSubmatch(strings.TrimSpace(szV))
	if m == nil {
		return nil
	}
	bind := strings.Trim(m[1], "[]")
	if bind == "" {
		bind = "127.0.0.1"
	}
	bp, _ := strconv.Atoi(m[2])
	dp, _ := strconv.Atoi(m[4])
	return map[string]any{"id": vault.NewUUID(), "type": szType, "bindHost": bind, "bindPort": bp, "destHost": strings.Trim(m[3], "[]"), "destPort": dp}
}

type sshBlock struct {
	names []string
	o     map[string]string
	fw    []map[string]any
}

var reKV = regexp.MustCompile(`^(\S+)\s*=?\s*(.*)$`)

// ParseSSHConfig 는 ~/.ssh/config 텍스트를 후보로 바꾼다.
func ParseSSHConfig(szText, szHome string) []Candidate {
	out := []Candidate{}
	jumps := []string{}
	var cur *sshBlock
	flush := func() {
		if cur == nil {
			return
		}
		for _, n := range cur.names {
			if strings.ContainsAny(n, "*?!") {
				continue
			}
			szJump := ""
			if pj := cur.o["proxyjump"]; pj != "" && pj != "none" {
				szJump = strings.TrimSpace(strings.Split(pj, ",")[0])
			}
			jumps = append(jumps, szJump)
			szHostName := cur.o["hostname"]
			if szHostName == "" {
				szHostName = n
			}
			iPort := 22
			if p, err := strconv.Atoi(cur.o["port"]); err == nil {
				iPort = p
			}
			szKey := cur.o["identityfile"]
			if strings.HasPrefix(szKey, "~") {
				szKey = szHome + szKey[1:]
			}
			szAuth := "password"
			if szKey != "" {
				szAuth = "key"
			}
			fws := []map[string]any{}
			for _, f := range cur.fw {
				c := map[string]any{}
				for k, v := range f {
					c[k] = v
				}
				c["id"] = vault.NewUUID()
				fws = append(fws, c)
			}
			host := map[string]any{
				"alias": n, "protocol": "ssh", "host": szHostName, "port": iPort, "username": cur.o["user"],
				"authType": szAuth, "forwards": fws,
			}
			if k, err := strconv.Atoi(cur.o["serveraliveinterval"]); err == nil && k > 0 {
				host["keepaliveSec"] = k
			}
			if cur.o["proxyjump"] != "" {
				host["notes"] = "ProxyJump " + cur.o["proxyjump"]
			}
			out = append(out, Candidate{Source: "ssh-config", Name: n, KeyFile: szKey, Host: host})
		}
	}
	for _, raw := range strings.Split(szText, "\n") {
		line := raw
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := reKV.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		k := strings.ToLower(m[1])
		v := strings.Trim(m[2], `"`)
		switch {
		case k == "host":
			flush()
			cur = &sshBlock{names: strings.Fields(v), o: map[string]string{}}
		case k == "match":
			flush()
			cur = nil
		case cur != nil && (k == "localforward" || k == "remoteforward"):
			t := "L"
			if k == "remoteforward" {
				t = "R"
			}
			if f := sshForward(t, v); f != nil {
				cur.fw = append(cur.fw, f)
			}
		case cur != nil:
			if _, has := cur.o[k]; !has {
				cur.o[k] = v
			}
		}
	}
	flush()
	// ProxyJump: 같은 파일의 Host 별칭이면 그 HostName, 아니면 user@host:port 의 host
	rePort := regexp.MustCompile(`:\d+$`)
	for i := range out {
		j := jumps[i]
		if j == "" {
			continue
		}
		via := ""
		for _, x := range out {
			if x.Name == j {
				via, _ = x.Host["host"].(string)
			}
		}
		if via == "" {
			if at := strings.Index(j, "@"); at >= 0 {
				j = j[at+1:]
			}
			via = rePort.ReplaceAllString(j, "")
		}
		out[i].JumpVia = via
	}
	return out
}

// ScanSSHConfig 는 ~/.ssh/config 를 읽는다 (없으면 빈 목록).
func ScanSSHConfig() []Candidate {
	szHome, err := os.UserHomeDir()
	if err != nil {
		return []Candidate{}
	}
	arr, err := os.ReadFile(filepath.Join(szHome, ".ssh", "config"))
	if err != nil {
		return []Candidate{}
	}
	return ParseSSHConfig(string(arr), szHome)
}
