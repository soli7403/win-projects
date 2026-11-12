package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"termgo/internal/importer"
	"termgo/internal/vault"
)

var (
	reEnvProd  = regexp.MustCompile(`^(prod|prd|live|운영)`)
	reEnvStage = regexp.MustCompile(`^(stg|stage|staging)`)
	reEnvDev   = regexp.MustCompile(`^(dev|test|개발)`)
)

func (pstrApp *App) registerImport() {
	h := pstrApp.router.Handle
	type A = []json.RawMessage
	h("import:scan", func(A) (any, error) {
		out := append(importer.ScanPutty(), importer.ScanSSHConfig()...)
		return out, nil
	})
	h("import:scanAws", func(a A) (any, error) {
		szProfile, _ := arg[string](a, 0)
		return importer.ScanAws(szProfile)
	})
	h("import:apply", func(a A) (any, error) {
		cands, err := arg[[]importer.Candidate](a, 0)
		if err != nil {
			return nil, err
		}
		szGroup, _ := arg[string](a, 1)
		return pstrApp.importApply(cands, szGroup)
	})
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

// importApply 는 선택한 후보를 서버 목록(과 볼트의 키)에 추가한다 (v1.1.0 importApply).
func (pstrApp *App) importApply(cands []importer.Candidate, szGroupName string) (map[string]any, error) {
	var groups, hosts []map[string]any
	_ = json.Unmarshal(pstrApp.store.Raw("groups"), &groups)
	_ = json.Unmarshal(pstrApp.store.Raw("hosts"), &hosts)

	var groupID any
	if szGroupName != "" {
		for _, g := range groups {
			if str(g, "name") == szGroupName && g["parentId"] == nil {
				groupID = g["id"]
			}
		}
		if groupID == nil {
			id := vault.NewUUID()
			groupID = id
			groups = append(groups, map[string]any{"id": id, "parentId": nil, "name": szGroupName, "sort": len(groups), "env": "none"})
		}
	}
	skipped := []string{}
	iKeys := 0
	nBefore := len(hosts)
	for _, c := range cands {
		hm := c.Host
		var credID any
		if c.KeyFile != "" {
			if arr, err := os.ReadFile(c.KeyFile); err != nil {
				skipped = append(skipped, fmt.Sprintf("%s: 키 파일을 읽지 못함 (%s)", c.Name, c.KeyFile))
			} else if pstrApp.vault.IsUnlocked() {
				sz, szUser := string(arr), str(hm, "username")
				meta, err := pstrApp.vault.Save(vault.Input{Name: c.Name + " 키", Kind: "key", Username: &szUser, PrivateKey: &sz})
				if err == nil {
					credID = meta.ID
					iKeys++
				}
			}
		}
		lower := strings.ToLower(c.Name)
		szEnv := "none"
		switch {
		case reEnvProd.MatchString(lower):
			szEnv = "prod"
		case reEnvStage.MatchString(lower):
			szEnv = "stage"
		case reEnvDev.MatchString(lower):
			szEnv = "dev"
		case str(hm, "protocol") == "serial":
			szEnv = "device"
		}
		szProto := str(hm, "protocol")
		if szProto == "" {
			szProto = "ssh"
		}
		iPort := 22
		if f, ok := hm["port"].(float64); ok {
			iPort = int(f)
		} else if n, ok := hm["port"].(int); ok {
			iPort = n
		}
		szAuth := "password"
		if credID != nil {
			szAuth = "key"
		} else if str(hm, "authType") == "key" {
			szAuth = "agent"
		}
		alias := str(hm, "alias")
		if alias == "" {
			alias = c.Name
		}
		enc := str(hm, "encoding")
		if enc == "" {
			enc = "utf-8"
		}
		term := str(hm, "termType")
		if term == "" {
			term = "xterm-256color"
		}
		keep := 30
		if f, ok := hm["keepaliveSec"].(float64); ok && f > 0 {
			keep = int(f)
		} else if n, ok := hm["keepaliveSec"].(int); ok && n > 0 {
			keep = n
		}
		fw := hm["forwards"]
		if fw == nil {
			fw = []any{}
		}
		host := map[string]any{
			"id": vault.NewUUID(), "groupId": groupID, "alias": alias, "protocol": szProto, "host": str(hm, "host"),
			"port": iPort, "username": str(hm, "username"), "authType": szAuth, "credentialId": credID,
			"tags": []string{c.Source}, "encoding": enc, "termType": term, "keepaliveSec": keep, "forwards": fw,
			"env": szEnv, "sort": len(hosts),
		}
		for _, k := range []string{"startupCommand", "serial", "notes"} {
			if v, ok := hm[k]; ok && v != nil && v != "" {
				host[k] = v
			}
		}
		hosts = append(hosts, host)
	}
	// 점프 호스트 연결: 새로 가져온 서버 먼저, 없으면 기존 서버에서 주소로 찾는다
	added := hosts[nBefore:]
	for i, c := range cands {
		if c.JumpVia == "" {
			continue
		}
		var jh map[string]any
		for _, h := range added {
			if str(h, "host") == c.JumpVia {
				jh = h
				break
			}
		}
		if jh == nil {
			for _, h := range hosts {
				if str(h, "host") == c.JumpVia {
					jh = h
					break
				}
			}
		}
		if jh != nil && jh["id"] != added[i]["id"] {
			added[i]["jumpHostId"] = jh["id"]
		} else {
			skipped = append(skipped, fmt.Sprintf("%s: 점프 호스트 %s 를 찾지 못해 직접 접속으로 등록", c.Name, c.JumpVia))
		}
	}
	rg, _ := json.Marshal(groups)
	rh, _ := json.Marshal(hosts)
	pstrApp.store.Patch(map[string]json.RawMessage{"groups": rg, "hosts": rh})
	return map[string]any{"hosts": len(cands), "keys": iKeys, "skipped": skipped}, nil
}
