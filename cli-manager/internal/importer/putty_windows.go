//go:build windows

package importer

import (
	"golang.org/x/sys/windows/registry"
)

// ScanPutty 는 HKCU\Software\SimonTatham\PuTTY\Sessions 를 읽는다.
func ScanPutty() []Candidate {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\SimonTatham\PuTTY\Sessions`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return []Candidate{}
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return []Candidate{}
	}
	m := map[string]RegValues{}
	for _, n := range names {
		sk, err := registry.OpenKey(k, n, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		vals := RegValues{}
		if vn, err := sk.ReadValueNames(-1); err == nil {
			for _, name := range vn {
				if _, t, err := sk.GetValue(name, nil); err == nil {
					switch t {
					case registry.DWORD, registry.QWORD:
						if v, _, err := sk.GetIntegerValue(name); err == nil {
							vals[name] = int(v)
						}
					case registry.SZ, registry.EXPAND_SZ:
						if v, _, err := sk.GetStringValue(name); err == nil {
							vals[name] = v
						}
					}
				}
			}
		}
		sk.Close()
		m[n] = vals
	}
	return PuttyToCandidates(m)
}
