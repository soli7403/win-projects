package gitsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"termgo/internal/store"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

const (
	szFileName = "termgo-sync.json"
	szAAD      = "termgo-sync"
	szFormat   = "termgo-sync-1"
)

// 상태 값 (Status.State)
const (
	StOK            = "ok"
	StUnconfigured  = "unconfigured"
	StLocked        = "locked"
	StNoGit         = "nogit"
	StVaultMismatch = "vault-mismatch" // 저장소의 볼트가 이 PC 볼트와 다름 → 사용자가 고른다
	StError         = "error"
)

// 동기화 방식 (Sync 인자)
const (
	ModeAuto      = ""
	ModePullVault = "pull-vault" // 저장소 볼트 받기
	ModeOverwrite = "overwrite"  // 이 PC 볼트로 덮어쓰기
)

// Status 는 sync:run / sync:status 결과.
type Status struct {
	State    string `json:"state"`
	Message  string `json:"message,omitempty"`
	LastSync int64  `json:"lastSync,omitempty"`
	// Changed 는 이번 동기화로 이 PC 데이터가 바뀌었는지 (렌더러가 다시 읽는다)
	Changed bool `json:"changed,omitempty"`
}

// File 은 저장소에 올리는 파일 하나. 헤더는 같은 마스터 비밀번호로 볼트를 열기 위한 최소 정보만 담고,
// 서버 주소·계정명 등 모든 내용은 payload(볼트 키로 암호화) 안에 있다.
type File struct {
	Format  string     `json:"format"`
	Vault   vault.File `json:"vault"`
	Payload vault.Enc  `json:"payload"`
}

// Service 는 git 동기화 서비스.
type Service struct {
	Dir   string // 데이터 폴더
	Store *store.Store
	Vault *vault.Vault
	UI    *ui.Bridge

	mu       sync.Mutex
	lastSync int64
}

func (pstrS *Service) repoDir() string  { return filepath.Join(pstrS.Dir, "sync-repo") }
func (pstrS *Service) basePath() string { return filepath.Join(pstrS.Dir, "sync-base.json") }

// RepoURL 은 설정된 저장소 주소.
func (pstrS *Service) RepoURL() string {
	var sz string
	_ = json.Unmarshal(pstrS.Store.Setting("syncRepo"), &sz)
	return sz
}

func idOf(raw json.RawMessage) string {
	var o struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &o)
	return o.ID
}

func toCollection(raw json.RawMessage) Collection {
	var arr []json.RawMessage
	_ = json.Unmarshal(raw, &arr)
	c := Collection{}
	for _, it := range arr {
		if id := idOf(it); id != "" {
			c[id] = it
		}
	}
	return c
}

// collect 는 이 PC 의 동기화 대상 데이터를 모은다.
func (pstrS *Service) collect() Data {
	d := Data{
		Groups:   toCollection(pstrS.Store.Raw("groups")),
		Hosts:    toCollection(pstrS.Store.Raw("hosts")),
		Snippets: toCollection(pstrS.Store.Raw("snippets")),
		Creds:    Collection{},
	}
	for _, it := range pstrS.Vault.Items() {
		raw, _ := json.Marshal(it)
		d.Creds[it.ID] = raw
	}
	return d
}

// order 는 기존 배열 순서를 최대한 유지하며 컬렉션을 배열로 만든다 (새 항목은 id 순으로 뒤에).
func order(c Collection, szKey string, st *store.Store) json.RawMessage {
	var prev []json.RawMessage
	_ = json.Unmarshal(st.Raw(szKey), &prev)
	out := []json.RawMessage{}
	seen := map[string]bool{}
	for _, it := range prev {
		if id := idOf(it); id != "" {
			if v, ok := c[id]; ok {
				out = append(out, v)
				seen[id] = true
			}
		}
	}
	rest := []string{}
	for id := range c {
		if !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	for _, id := range rest {
		out = append(out, c[id])
	}
	arr, _ := json.Marshal(out)
	return arr
}

// apply 는 병합 결과를 이 PC 에 반영한다.
func (pstrS *Service) apply(d Data) error {
	items := []vault.Item{}
	for _, id := range SortedIDs(d.Creds) {
		var it vault.Item
		if err := json.Unmarshal(d.Creds[id], &it); err == nil {
			items = append(items, it)
		}
	}
	if err := pstrS.Vault.ReplaceItems(items); err != nil {
		return err
	}
	pstrS.Store.Patch(map[string]json.RawMessage{
		"groups":   order(d.Groups, "groups", pstrS.Store),
		"hosts":    order(d.Hosts, "hosts", pstrS.Store),
		"snippets": order(d.Snippets, "snippets", pstrS.Store),
	})
	return nil
}

func (pstrS *Service) loadBase() Data {
	d := NewData()
	_ = store.ReadJSON(pstrS.basePath(), &d)
	if d.Groups == nil {
		d = NewData()
	}
	return d
}

func (pstrS *Service) saveBase(d Data) { _ = store.WriteJSONAtomic(pstrS.basePath(), d) }

// readRemote 는 저장소 작업 폴더의 동기화 파일을 읽는다 (없으면 errNoFile).
func (pstrS *Service) readRemote() (*File, error) {
	arr, err := os.ReadFile(filepath.Join(pstrS.repoDir(), szFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNoFile
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(arr, &f); err != nil || f.Format != szFormat {
		return nil, errors.New("동기화 파일 형식을 알 수 없습니다")
	}
	return &f, nil
}

func (pstrS *Service) writeRemote(d Data) error {
	h := pstrS.Vault.Header()
	if h == nil {
		return errors.New("볼트 없음")
	}
	plain, err := json.Marshal(d)
	if err != nil {
		return err
	}
	enc, err := pstrS.Vault.Encrypt(plain, szAAD)
	if err != nil {
		return err
	}
	arr, err := json.MarshalIndent(File{Format: szFormat, Vault: *h, Payload: enc}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pstrS.repoDir(), szFileName), arr, 0o600)
}

func (pstrS *Service) fail(err error) Status {
	return Status{State: StError, Message: err.Error(), LastSync: pstrS.lastSync}
}

var reURLCred = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://[^/@:]+:[^/@]+@`)

// Sync 는 한 번 동기화한다. 동시에 하나만 실행된다.
func (pstrS *Service) Sync(szMode string) Status {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	szURL := pstrS.RepoURL()
	switch {
	case szURL == "":
		return Status{State: StUnconfigured}
	case reURLCred.MatchString(szURL):
		return Status{State: StError, Message: "저장소 주소에 비밀번호·토큰을 넣지 마세요 — 이 PC 의 git 자격 증명 도우미/SSH 키를 사용합니다"}
	case !pstrS.Vault.IsUnlocked():
		return Status{State: StLocked, Message: "볼트를 연 뒤에 동기화할 수 있습니다"}
	case !GitAvailable():
		return Status{State: StNoGit, Message: "git 이 설치되어 있지 않습니다"}
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ { // 푸시가 경합으로 거절되면 한 번 더
		st, retry := pstrS.syncOnce(szURL, szMode)
		if !retry {
			return st
		}
		lastErr = errors.New(st.Message)
	}
	return pstrS.fail(lastErr)
}

func (pstrS *Service) syncOnce(szURL, szMode string) (Status, bool) {
	szBranch, err := ensureRepo(pstrS.repoDir(), szURL)
	if err != nil {
		return pstrS.fail(err), false
	}
	local := pstrS.collect()
	remoteFile, err := pstrS.readRemote()
	if err != nil && !errors.Is(err, errNoFile) {
		return pstrS.fail(err), false
	}

	var remote *Data
	changed := false
	if remoteFile != nil && szMode != ModeOverwrite {
		if szMode == ModePullVault {
			// 저장소 볼트 받기: 그 볼트의 마스터 비밀번호로 열고 이 PC 볼트를 교체한다
			raw, ok := pstrS.UI.Ask(map[string]any{
				"kind": "password", "sessionTitle": "git 동기화", "prompt": "저장소 볼트의 마스터 비밀번호",
				"username": "", "allowSave": false,
			})
			var reply struct {
				Password string `json:"password"`
			}
			if ok {
				_ = json.Unmarshal(raw, &reply)
			}
			if !ok || reply.Password == "" || !pstrS.Vault.AdoptRemote(remoteFile.Vault, nil, reply.Password) {
				return Status{State: StVaultMismatch, Message: "마스터 비밀번호가 맞지 않거나 취소되었습니다"}, false
			}
			local = NewData() // 이 PC 의 기존 항목은 새 볼트로 대체
			changed = true
		}
		plain, err := pstrS.Vault.Decrypt(remoteFile.Payload, szAAD)
		if err != nil {
			// 저장소의 볼트가 이 PC 볼트와 다른 볼트 (처음 연결 등) — 사용자가 받기/덮어쓰기를 고른다
			return Status{State: StVaultMismatch, Message: "저장소의 볼트가 이 PC 의 볼트와 다릅니다"}, false
		}
		var d Data
		if json.Unmarshal(plain, &d) != nil {
			return pstrS.fail(errors.New("동기화 파일 해석 실패")), false
		}
		if d.Groups == nil {
			d = NewData()
		}
		remote = &d
	}

	base := pstrS.loadBase()
	if szMode == ModePullVault {
		base = NewData()
	}
	merged := local
	if remote != nil {
		merged = Merge(base, local, *remote)
	}
	if !Equal(merged, pstrS.collect()) || szMode == ModePullVault {
		if err := pstrS.apply(merged); err != nil {
			return pstrS.fail(err), false
		}
		changed = changed || !Equal(merged, local)
	}
	if remote == nil || !Equal(merged, *remote) {
		if err := pstrS.writeRemote(merged); err != nil {
			return pstrS.fail(err), false
		}
		if err := commitPush(pstrS.repoDir(), szBranch, szFileName); err != nil {
			return Status{State: StError, Message: err.Error()}, true
		}
	}
	pstrS.saveBase(merged)
	pstrS.lastSync = time.Now().UnixMilli()
	return Status{State: StOK, LastSync: pstrS.lastSync, Changed: changed}, false
}

// LastSync 는 마지막 성공 시각(ms).
func (pstrS *Service) LastSync() int64 {
	pstrS.mu.Lock()
	defer pstrS.mu.Unlock()
	return pstrS.lastSync
}
