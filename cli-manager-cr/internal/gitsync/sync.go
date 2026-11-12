package gitsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"termgo/internal/jsonfile"
	"termgo/internal/vault"
)

// 동기화 모드 (sync.run 인자, spec/ipc.md §2).
const (
	G_cszModeAuto      = ""
	G_cszModePullVault = "pull-vault"
	G_cszModeOverwrite = "overwrite"
)

// 동기화 상태 (data/sync-backup.md §5.4, 6종).
const (
	G_cszStateOk            = "ok"
	G_cszStateUnconfigured  = "unconfigured"
	G_cszStateLocked        = "locked"
	G_cszStateNoGit         = "nogit"
	G_cszStateVaultMismatch = "vault-mismatch"
	G_cszStateError         = "error"
)

// G_ciSyncMaxAttempts 는 본 절차 최대 시도 횟수다 (푸시 거절 시 한 번 더, sync-backup.md §5.4).
const G_ciSyncMaxAttempts = 2

// 명세 문구 (data/sync-backup.md §5.4).
const (
	g_cszMsgCredentialInUrl = "저장소 주소에 비밀번호·토큰을 넣지 마세요 — 이 PC 의 git 자격 증명 도우미/SSH 키를 사용합니다"
	g_cszMsgLocked          = "볼트를 연 뒤에 동기화할 수 있습니다"
	g_cszMsgNoGit           = "git 이 설치되어 있지 않습니다"
	g_cszMsgMismatch        = "저장소의 볼트가 이 PC 의 볼트와 다릅니다"
	g_cszMsgPasswordFail    = "마스터 비밀번호가 맞지 않거나 취소되었습니다"
	g_cszMsgParseFail       = "동기화 파일 해석 실패"
)

// G_cszSyncBaseFile 은 기준 파일 이름이다 (data/sync-backup.md §5.2).
const G_cszSyncBaseFile = "sync-base.json"

// Result 는 동기화 한 번의 결과 상태다 (sync.run 결과, sync.status 이벤트 값).
type Result struct {
	M_szState     string `json:"state"`
	M_szMessage   string `json:"message,omitempty"`
	M_i64LastSync int64  `json:"lastSync"`
	M_bChanged    bool   `json:"changed"`
}

// Orders 는 이 PC 의 배열 순서(id 목록)다.
type Orders struct {
	M_slGroups   []string
	M_slHosts    []string
	M_slSnippets []string
	M_slCreds    []string
}

// LocalStore 는 이 PC 의 데이터를 읽고 쓴다 (store 배열 + 볼트 항목).
type LocalStore interface {
	// LoadData 는 이 PC 데이터와 배열 순서를 돌려준다.
	LoadData() (Data, Orders, error)
	// SaveData 는 병합 결과를 이 PC 에 반영한다 (볼트 항목 전체 교체 포함).
	SaveData(data Data, orders Orders) error
}

// VaultOps 는 동기화가 볼트에 요구하는 동작이다 (볼트 구현이 맞춘다).
type VaultOps interface {
	Locked() bool
	HeaderJSON() ([]byte, error)
	EncryptPayload(bPlain []byte) (vault.Blob, error)
	DecryptPayload(blob vault.Blob) ([]byte, error)
	AdoptRemote(header []byte, items []vault.ItemRecord, szPassword string) (bool, error)
}

// Deps 는 동기화 실행에 필요한 외부 의존성이다.
type Deps struct {
	M_szDataDir string
	M_store     LocalStore
	M_vault     VaultOps
	// M_fnAskPassword 는 "저장소 볼트 받기"에서 마스터 비밀번호를 받는다 (취소·빈 값이면 bOk=false).
	M_fnAskPassword func() (string, bool)
	M_fnNow         func() time.Time
}

// Syncer 는 동기화 실행을 직렬화하고 마지막 결과를 가진다.
type Syncer struct {
	mtxRun  sync.Mutex
	mtxData sync.Mutex
	deps    Deps
	pLast   *Result
}

// NewSyncer 는 동기화 실행기를 만든다.
func NewSyncer(deps Deps) *Syncer {
	if deps.M_fnNow == nil {
		deps.M_fnNow = time.Now
	}
	return &Syncer{deps: deps}
}

// Last 는 마지막 결과를 돌려준다 (없으면 nil).
func (pSyncer *Syncer) Last() *Result {
	pSyncer.mtxData.Lock()
	defer pSyncer.mtxData.Unlock()
	return pSyncer.pLast
}

// Run 은 한 번 동기화한다 (sync-backup.md §5.4). 동시에 하나만 돈다.
func (pSyncer *Syncer) Run(szRepo string, szMode string) Result {
	pSyncer.mtxRun.Lock()
	defer pSyncer.mtxRun.Unlock()
	res := pSyncer.run(szRepo, szMode)
	pSyncer.mtxData.Lock()
	pSyncer.pLast = &res
	pSyncer.mtxData.Unlock()
	return res
}

// run 은 사전 검사 후 본 절차를 최대 두 번 돈다 (푸시 거절 시 한 번 더).
func (pSyncer *Syncer) run(szRepo string, szMode string) Result {
	if szRepo == "" {
		return Result{M_szState: G_cszStateUnconfigured}
	}
	if HasCredentialInUrl(szRepo) {
		return Result{M_szState: G_cszStateError, M_szMessage: g_cszMsgCredentialInUrl}
	}
	if pSyncer.deps.M_vault.Locked() {
		return Result{M_szState: G_cszStateLocked, M_szMessage: g_cszMsgLocked}
	}
	if !GitAvailable() {
		return Result{M_szState: G_cszStateNoGit, M_szMessage: g_cszMsgNoGit}
	}
	var res Result
	for iAttempt := 0; iAttempt < G_ciSyncMaxAttempts; iAttempt++ {
		var bRetry bool
		res, bRetry = pSyncer.attempt(szRepo, szMode, iAttempt+1 < G_ciSyncMaxAttempts)
		if !bRetry {
			return res
		}
	}
	return res
}

// attempt 는 본 절차 한 번이다. 푸시가 거절되었고 bCanRetry 이면 bRetry=true 를 돌려준다.
func (pSyncer *Syncer) attempt(szRepo string, szMode string, bCanRetry bool) (Result, bool) {
	szRepoDir := filepath.Join(pSyncer.deps.M_szDataDir, "sync-repo")
	szBranch, errPrepare := prepareClone(szRepoDir, szRepo)
	if errPrepare != nil {
		return Result{M_szState: G_cszStateError, M_szMessage: errPrepare.Error()}, false
	}

	localData, orders, errLoad := pSyncer.deps.M_store.LoadData()
	if errLoad != nil {
		return Result{M_szState: G_cszStateError, M_szMessage: errLoad.Error()}, false
	}
	baseData, errBase := loadBase(pSyncer.deps.M_szDataDir)
	if errBase != nil {
		return Result{M_szState: G_cszStateError, M_szMessage: errBase.Error()}, false
	}

	bChanged := false
	remoteData := NewData()
	bHasRemote := false
	bRemoteBytes, errRead := os.ReadFile(filepath.Join(szRepoDir, G_cszSyncFileName))
	if errRead == nil && szMode != G_cszModeOverwrite {
		header, blob, errDecode := DecodeSyncFile(bRemoteBytes)
		if errDecode != nil {
			return Result{M_szState: G_cszStateError, M_szMessage: errDecode.Error()}, false
		}
		if szMode == G_cszModePullVault {
			var errPull error
			remoteData, errPull = pSyncer.pullVault(header, blob)
			if errPull != nil {
				return errorToResult(errPull), false
			}
			// 볼트를 받았으므로 이 PC 데이터는 빈 데이터로 보고 기준도 비운다 (sync-backup.md §5.4 3).
			localData = NewData()
			baseData = NewData()
			bChanged = true
		} else {
			bPlain, errOpen := pSyncer.deps.M_vault.DecryptPayload(blob)
			if errOpen != nil {
				return Result{M_szState: G_cszStateVaultMismatch, M_szMessage: g_cszMsgMismatch}, false
			}
			remoteData, errOpen = DecodeData(bPlain)
			if errOpen != nil {
				return Result{M_szState: G_cszStateError, M_szMessage: g_cszMsgParseFail}, false
			}
		}
		bHasRemote = true
	}

	merged := localData
	if bHasRemote && szMode != G_cszModeOverwrite {
		merged = MergeData(baseData, localData, remoteData)
	}
	if !DataEqual(merged, localData) {
		if errSave := pSyncer.deps.M_store.SaveData(merged, orders); errSave != nil {
			return Result{M_szState: G_cszStateError, M_szMessage: errSave.Error()}, false
		}
		bChanged = true
	}

	bNeedWrite := !bHasRemote || szMode == G_cszModeOverwrite || !DataEqual(merged, remoteData)
	if bNeedWrite {
		errPush := pSyncer.writeAndPush(szRepoDir, szBranch, merged)
		if errPush != nil {
			if bCanRetry && isPushError(errPush) {
				return Result{}, true
			}
			return errorToResult(errPush), false
		}
	}

	if errBaseSave := jsonfile.Write(filepath.Join(pSyncer.deps.M_szDataDir, G_cszSyncBaseFile), merged); errBaseSave != nil {
		return Result{M_szState: G_cszStateError, M_szMessage: errBaseSave.Error()}, false
	}
	return Result{
		M_szState:     G_cszStateOk,
		M_i64LastSync: pSyncer.deps.M_fnNow().UnixMilli(),
		M_bChanged:    bChanged,
	}, false
}

// pullVault 는 저장소 볼트를 받는다: 비밀번호를 묻고, 헤더로 키를 풀어 payload 를 읽고, 볼트를 교체한다.
func (pSyncer *Syncer) pullVault(header json.RawMessage, blob vault.Blob) (Data, error) {
	szPassword, bOk := pSyncer.deps.M_fnAskPassword()
	if !bOk || szPassword == "" {
		return NewData(), &mismatchError{}
	}
	bKey, bUnwrap := vault.UnwrapHeaderKey(header, szPassword)
	if !bUnwrap {
		return NewData(), &mismatchError{}
	}
	bPlain, errOpen := vault.DecryptWithKey(bKey, blob, vault.G_cszSyncPayloadAad)
	if errOpen != nil {
		return NewData(), errors.New(g_cszMsgParseFail)
	}
	remoteData, errDecode := DecodeData(bPlain)
	if errDecode != nil {
		return NewData(), errors.New(g_cszMsgParseFail)
	}
	slItems := sortedItems(remoteData.M_mapCreds)
	bAdopted, errAdopt := pSyncer.deps.M_vault.AdoptRemote(header, slItems, szPassword)
	if errAdopt != nil {
		return NewData(), errAdopt
	}
	if !bAdopted {
		return NewData(), &mismatchError{}
	}
	return remoteData, nil
}

// mismatchError 는 볼트가 맞지 않음(또는 받기 취소)을 나타낸다.
type mismatchError struct{}

func (pErr *mismatchError) Error() string {
	return g_cszMsgPasswordFail
}

// errorToResult 는 오류를 상태로 바꾼다 (볼트 불일치는 vault-mismatch, 그 외는 error).
func errorToResult(errRun error) Result {
	var pMismatch *mismatchError
	if errors.As(errRun, &pMismatch) {
		return Result{M_szState: G_cszStateVaultMismatch, M_szMessage: g_cszMsgPasswordFail}
	}
	return Result{M_szState: G_cszStateError, M_szMessage: errRun.Error()}
}

// pushError 는 푸시 거절을 나타낸다 (재시도 대상).
type pushError struct {
	errCause error
}

func (pErr *pushError) Error() string {
	return pErr.errCause.Error()
}

func isPushError(errRun error) bool {
	var pPush *pushError
	return errors.As(errRun, &pPush)
}

// writeAndPush 는 병합 결과를 저장소 파일로 쓰고 커밋·푸시한다. 바뀐 게 없으면 커밋하지 않는다.
func (pSyncer *Syncer) writeAndPush(szRepoDir string, szBranch string, merged Data) error {
	bPlain, errEncode := EncodeData(merged)
	if errEncode != nil {
		return errEncode
	}
	blob, errEnc := pSyncer.deps.M_vault.EncryptPayload(bPlain)
	if errEnc != nil {
		return errEnc
	}
	rawHeader, errHeader := pSyncer.deps.M_vault.HeaderJSON()
	if errHeader != nil {
		return errHeader
	}
	bFile, errFile := EncodeSyncFile(rawHeader, blob)
	if errFile != nil {
		return errFile
	}
	if errWrite := os.WriteFile(filepath.Join(szRepoDir, G_cszSyncFileName), bFile, 0o600); errWrite != nil {
		return errWrite
	}
	if _, errAdd := gitRun(szRepoDir, "add", G_cszSyncFileName); errAdd != nil {
		return errAdd
	}
	if _, errDiff := gitRun(szRepoDir, "diff", "--cached", "--quiet"); errDiff == nil {
		return nil
	}
	if _, errCommit := gitRun(szRepoDir, "-c", "user.name=TermGo", "-c", "user.email=termgo@localhost", "commit", "-m", "TermGo sync", "--only", G_cszSyncFileName); errCommit != nil {
		return errCommit
	}
	if _, errPush := gitRun(szRepoDir, "push", "origin", "HEAD:refs/heads/"+szBranch); errPush != nil {
		return &pushError{errCause: errPush}
	}
	return nil
}

// prepareClone 은 작업용 클론을 준비하고 브랜치 이름을 돌려준다 (sync-backup.md §5.4 1).
func prepareClone(szRepoDir string, szRepo string) (string, error) {
	if _, errStat := os.Stat(filepath.Join(szRepoDir, ".git")); errStat != nil {
		if errMk := os.MkdirAll(filepath.Dir(szRepoDir), 0o700); errMk != nil {
			return "", errMk
		}
		if _, errClone := gitRun("", "clone", szRepo, szRepoDir); errClone != nil {
			return "", errClone
		}
	} else {
		szCurrent, errURL := gitRun(szRepoDir, "remote", "get-url", "origin")
		if errURL != nil || trimNewline(szCurrent) != szRepo {
			if _, errSet := gitRun(szRepoDir, "remote", "set-url", "origin", szRepo); errSet != nil {
				return "", errSet
			}
		}
	}
	if _, errFetch := gitRun(szRepoDir, "fetch", "origin"); errFetch != nil {
		return "", errFetch
	}
	for _, szBranch := range []string{"main", "master"} {
		if _, errRef := gitRun(szRepoDir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+szBranch); errRef == nil {
			if _, errCo := gitRun(szRepoDir, "checkout", "-B", szBranch, "origin/"+szBranch); errCo != nil {
				return "", errCo
			}
			if _, errReset := gitRun(szRepoDir, "reset", "--hard", "origin/"+szBranch); errReset != nil {
				return "", errReset
			}
			return szBranch, nil
		}
	}
	// 원격이 비어 있으면 main 으로 시작한다.
	if _, errHead := gitRun(szRepoDir, "symbolic-ref", "HEAD", "refs/heads/main"); errHead != nil {
		return "", errHead
	}
	return "main", nil
}

// trimNewline 은 명령 출력 끝의 줄바꿈과 공백을 뗀다.
func trimNewline(szText string) string {
	for len(szText) > 0 && (szText[len(szText)-1] == '\n' || szText[len(szText)-1] == '\r' || szText[len(szText)-1] == ' ') {
		szText = szText[:len(szText)-1]
	}
	return szText
}

// loadBase 는 기준 파일을 읽는다. 없으면 빈 데이터다.
func loadBase(szDataDir string) (Data, error) {
	szPath := filepath.Join(szDataDir, G_cszSyncBaseFile)
	if !jsonfile.Exists(szPath) {
		return NewData(), nil
	}
	data := NewData()
	if errRead := jsonfile.Read(szPath, &data); errRead != nil {
		return NewData(), nil
	}
	return DecodeDataFrom(data), nil
}

// DecodeDataFrom 은 빈 컬렉션을 채운 복사본을 돌려준다.
func DecodeDataFrom(data Data) Data {
	if data.M_mapGroups == nil {
		data.M_mapGroups = Collection{}
	}
	if data.M_mapHosts == nil {
		data.M_mapHosts = Collection{}
	}
	if data.M_mapSnippets == nil {
		data.M_mapSnippets = Collection{}
	}
	if data.M_mapCreds == nil {
		data.M_mapCreds = Collection{}
	}
	return data
}

// sortedItems 는 creds 컬렉션을 id 순서 목록으로 바꾼다 (볼트 항목 교체용).
func sortedItems(mapCreds Collection) []vault.ItemRecord {
	slIds := make([]string, 0, len(mapCreds))
	for szId := range mapCreds {
		slIds = append(slIds, szId)
	}
	sort.Strings(slIds)
	slOut := []vault.ItemRecord{}
	for _, szId := range slIds {
		var record vault.ItemRecord
		if errJson := json.Unmarshal(mapCreds[szId], &record); errJson == nil {
			slOut = append(slOut, record)
		}
	}
	return slOut
}
