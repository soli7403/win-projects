package store

import (
	"encoding/json"
	"time"

	"termgo/internal/model"
)

// Hosts 는 저장된 서버 목록을 해석한다.
func (pstrS *Store) Hosts() []model.Host {
	var out []model.Host
	_ = json.Unmarshal(pstrS.Raw("hosts"), &out)
	return out
}

// HostByID 는 id 로 서버를 찾는다.
func (pstrS *Store) HostByID(szID string) (model.Host, bool) {
	for _, h := range pstrS.Hosts() {
		if h.ID == szID {
			return h, true
		}
	}
	return model.Host{}, false
}

// Groups 는 그룹 목록.
func (pstrS *Store) Groups() []model.Group {
	var out []model.Group
	_ = json.Unmarshal(pstrS.Raw("groups"), &out)
	return out
}

// FindKnownHost 는 승인된 호스트 키를 찾는다.
func (pstrS *Store) FindKnownHost(szHostPort string) (model.KnownHost, bool) {
	var arr []model.KnownHost
	_ = json.Unmarshal(pstrS.Raw("knownHosts"), &arr)
	for _, k := range arr {
		if k.HostPort == szHostPort {
			return k, true
		}
	}
	return model.KnownHost{}, false
}

// SetKnownHost 는 호스트 키를 저장(교체)한다.
func (pstrS *Store) SetKnownHost(k model.KnownHost) {
	var arr []model.KnownHost
	_ = json.Unmarshal(pstrS.Raw("knownHosts"), &arr)
	out := arr[:0:0]
	for _, x := range arr {
		if x.HostPort != k.HostPort {
			out = append(out, x)
		}
	}
	out = append(out, k)
	raw, _ := json.Marshal(out)
	pstrS.Patch(map[string]json.RawMessage{"knownHosts": raw})
}

// PatchHost 는 서버 하나의 필드를 (알 수 없는 필드를 보존하며) 바꾼다.
func (pstrS *Store) PatchHost(szID string, mFields map[string]any) {
	var arr []map[string]any
	_ = json.Unmarshal(pstrS.Raw("hosts"), &arr)
	for _, m := range arr {
		if m["id"] == szID {
			for k, v := range mFields {
				m[k] = v
			}
		}
	}
	raw, _ := json.Marshal(arr)
	pstrS.Patch(map[string]json.RawMessage{"hosts": raw})
}

// TouchHost 는 lastUsedAt 을 갱신한다.
func (pstrS *Store) TouchHost(szID string) {
	pstrS.PatchHost(szID, map[string]any{"lastUsedAt": time.Now().UnixMilli()})
}

// Setting 은 settings 의 값을 (기본값 병합 후) 돌려준다.
func (pstrS *Store) Setting(szKey string) json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(mergeSettings(pstrS.Raw("settings")), &m)
	return m[szKey]
}

// SettingBool 은 bool 설정 하나.
func (pstrS *Store) SettingBool(szKey string) bool {
	var b bool
	_ = json.Unmarshal(pstrS.Setting(szKey), &b)
	return b
}

// SettingInt 는 정수 설정 하나.
func (pstrS *Store) SettingInt(szKey string) int {
	var n int
	_ = json.Unmarshal(pstrS.Setting(szKey), &n)
	return n
}
