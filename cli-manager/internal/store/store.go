// Package store 는 v1.1.0 의 store.json(서버·그룹·스니펫·설정) 호환 저장소다.
// 알 수 없는 필드도 잃지 않도록 최상위는 map 으로 보관하고, 자주 쓰는 부분만 해석한다.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store 는 store.json 전체를 메모리에 들고 있다가 지연 저장한다.
type Store struct {
	mu     sync.Mutex
	szFile string
	mData  map[string]json.RawMessage
	timer  *time.Timer
	// OnChange 는 Patch 로 최상위 키가 바뀔 때 (키 목록과 함께) 불린다 — git 동기화 예약용.
	OnChange func(keys []string)
}

// DataDir 는 데이터 폴더(%APPDATA%/TermGo 등)를 만들고 돌려준다. TERMGO_DATA 로 덮어쓴다.
func DataDir() (string, error) {
	szDir := os.Getenv("TERMGO_DATA")
	if szDir == "" {
		szBase, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		szDir = filepath.Join(szBase, "TermGo", "data")
	}
	return szDir, os.MkdirAll(szDir, 0o700)
}

// WriteJSONAtomic 은 임시 파일에 쓴 뒤 rename 하고, 이전 내용은 .bak 으로 남긴다.
func WriteJSONAtomic(szFile string, v any) error {
	arr, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(szFile); err == nil {
		_ = os.WriteFile(szFile+".bak", old, 0o600)
	}
	szTmp := szFile + ".tmp"
	if err := os.WriteFile(szTmp, arr, 0o600); err != nil {
		return err
	}
	return os.Rename(szTmp, szFile)
}

// ReadJSON 은 파일을 읽고, 깨져 있으면 .bak 을 시도한다. 둘 다 없으면 os.ErrNotExist.
func ReadJSON(szFile string, out any) error {
	var lastErr error = os.ErrNotExist
	for _, szF := range []string{szFile, szFile + ".bak"} {
		arr, err := os.ReadFile(szF)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(arr, out); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

// Open 은 data 폴더의 store.json 을 읽는다 (없으면 기본값).
func Open(szDir string) (*Store, error) {
	pstrS := &Store{szFile: filepath.Join(szDir, "store.json"), mData: map[string]json.RawMessage{}}
	if err := ReadJSON(pstrS.szFile, &pstrS.mData); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, szK := range []string{"groups", "hosts", "snippets", "knownHosts"} {
		if _, bOk := pstrS.mData[szK]; !bOk {
			pstrS.mData[szK] = json.RawMessage("[]")
		}
	}
	pstrS.mData["version"] = json.RawMessage("1")
	return pstrS, nil
}

// Get 은 렌더러에 줄 StoreData(JSON)를 만든다. settings 는 비어 있으면 {} (렌더러가 기본값 병합).
func (pstrS *Store) Get() map[string]json.RawMessage {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	m := make(map[string]json.RawMessage, len(pstrS.mData)+1)
	for k, v := range pstrS.mData {
		m[k] = v
	}
	m["settings"] = mergeSettings(m["settings"])
	return m
}

// Patch 는 최상위 키 단위로 덮어쓴다 (v1.1.0 store:patch 와 동일).
func (pstrS *Store) Patch(m map[string]json.RawMessage) {
	pstrS.mu.Lock()
	for k, v := range m {
		pstrS.mData[k] = v
	}
	pstrS.scheduleLocked()
	fn := pstrS.OnChange
	pstrS.mu.Unlock()
	if fn != nil {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		fn(keys)
	}
}

// ReplaceAll 은 백업 가져오기용.
func (pstrS *Store) ReplaceAll(m map[string]json.RawMessage) error {
	pstrS.mu.Lock()
	pstrS.mData = m
	pstrS.mu.Unlock()
	return pstrS.Flush()
}

// Raw 는 최상위 키 하나를 돌려준다.
func (pstrS *Store) Raw(szKey string) json.RawMessage {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	return pstrS.mData[szKey]
}

func (pstrS *Store) scheduleLocked() {
	if pstrS.timer != nil {
		pstrS.timer.Stop()
	}
	pstrS.timer = time.AfterFunc(300*time.Millisecond, func() { _ = pstrS.Flush() })
}

// Flush 는 즉시 저장한다.
func (pstrS *Store) Flush() error {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	if pstrS.timer != nil {
		pstrS.timer.Stop()
		pstrS.timer = nil
	}
	return WriteJSONAtomic(pstrS.szFile, pstrS.mData)
}
