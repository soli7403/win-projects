// Package gitsync 는 서버 목록·폴더·스니펫·저장된 계정을 비공개 git 저장소로 여러 PC 간에 맞춘다 (3.11).
package gitsync

import (
	"bytes"
	"encoding/json"
	"sort"
)

// Collection 은 id → 항목(JSON) 맵. 항목 내용은 해석하지 않고 JSON 동일성으로만 비교한다.
type Collection map[string]json.RawMessage

// Data 는 동기화 대상 전체 (서버 목록, 폴더, 스니펫, 저장된 계정). 설정·호스트 키는 동기화하지 않는다.
type Data struct {
	Groups   Collection `json:"groups"`
	Hosts    Collection `json:"hosts"`
	Snippets Collection `json:"snippets"`
	Creds    Collection `json:"creds"` // 볼트 항목(암호화된 비밀 + 이름/종류/사용자명)
}

// NewData 는 빈 Data.
func NewData() Data {
	return Data{Groups: Collection{}, Hosts: Collection{}, Snippets: Collection{}, Creds: Collection{}}
}

func canon(raw json.RawMessage) []byte {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return raw
	}
	out, _ := json.Marshal(v) // map 키는 정렬되어 직렬화된다
	return out
}

func same(a, b json.RawMessage) bool { return bytes.Equal(canon(a), canon(b)) }

// MergeCollection 은 base(마지막 동기화 시점) 기준 3방향 병합이다.
//   - 새 항목은 양쪽에서 모두 합친다
//   - 한쪽에서 삭제하면 다른 쪽에서도 삭제된다 (다른 쪽이 수정했어도 삭제가 이긴다)
//   - 양쪽에서 같은 항목을 수정했으면 지금 동기화하는 PC(local) 쪽 내용이 남는다
func MergeCollection(base, local, remote Collection) Collection {
	out := Collection{}
	ids := map[string]struct{}{}
	for _, c := range []Collection{base, local, remote} {
		for id := range c {
			ids[id] = struct{}{}
		}
	}
	for id := range ids {
		b, inB := base[id]
		l, inL := local[id]
		r, inR := remote[id]
		switch {
		case inB:
			if !inL || !inR {
				continue // 어느 한쪽이 삭제
			}
			lChanged, rChanged := !same(l, b), !same(r, b)
			switch {
			case lChanged: // 양쪽 수정이면 local 우선
				out[id] = l
			case rChanged:
				out[id] = r
			default:
				out[id] = l
			}
		default: // base 에 없음 = 새 항목
			if inL {
				out[id] = l // 양쪽 모두 새로 만든 같은 id 면 local
			} else if inR {
				out[id] = r
			}
		}
	}
	return out
}

// Merge 는 Data 전체를 병합한다.
func Merge(base, local, remote Data) Data {
	return Data{
		Groups:   MergeCollection(base.Groups, local.Groups, remote.Groups),
		Hosts:    MergeCollection(base.Hosts, local.Hosts, remote.Hosts),
		Snippets: MergeCollection(base.Snippets, local.Snippets, remote.Snippets),
		Creds:    MergeCollection(base.Creds, local.Creds, remote.Creds),
	}
}

// Equal 은 두 Data 가 내용상 같은지.
func Equal(a, b Data) bool {
	eq := func(x, y Collection) bool {
		if len(x) != len(y) {
			return false
		}
		for id, v := range x {
			w, ok := y[id]
			if !ok || !same(v, w) {
				return false
			}
		}
		return true
	}
	return eq(a.Groups, b.Groups) && eq(a.Hosts, b.Hosts) && eq(a.Snippets, b.Snippets) && eq(a.Creds, b.Creds)
}

// SortedIDs 는 시험·표시용.
func SortedIDs(c Collection) []string {
	out := make([]string, 0, len(c))
	for id := range c {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
