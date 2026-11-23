package app

import (
	"dariterm/internal/i18n"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"dariterm/internal/jsonfile"
	"dariterm/internal/storeupdate"
	"dariterm/internal/update"
)

// 새 버전을 어디에 묻는지 (lifecycle.md §6).
const (
	g_cszUpdateSourceStore   = "store"   // Microsoft Store 설치본: 스토어에 묻는다 (D115)
	g_cszUpdateSourceRelease = "release" // 배포 저장소가 정해진 빌드: 저장소의 최신 릴리스를 본다 (D80)
)

// 스토어 호출. 시험에서 바꿔 끼운다.
var (
	g_fnIsStoreInstall = storeupdate.IsStoreInstall
	g_fnStoreCheck     = storeupdate.Check
	g_fnStoreInstall   = storeupdate.Install
	g_fnOpenStorePage  = func() { _ = openWithShell(storeupdate.G_cszStorePageUri) }
)

// updateSource 는 이 실행이 새 버전을 어디에 묻는지 정한다. Windows 의 스토어 설치본이면 스토어,
// 아니고 배포 저장소가 정해져 있으면 저장소, 둘 다 아니면 빈 값(확인하지 않음)이다.
func updateSource() string {
	if g_fnIsStoreInstall() {
		return g_cszUpdateSourceStore
	}
	if updateRepo() != "" {
		return g_cszUpdateSourceRelease
	}
	return ""
}

// storeCheckResult 는 스토어에 물은 결과를 확인 결과 형식으로 바꾼다.
func storeCheckResult() update.Result {
	pUpdate, errCheck := g_fnStoreCheck()
	switch {
	case errCheck != nil:
		return update.Result{M_szStatus: update.G_cszStatusError}
	case pUpdate == nil:
		return update.Result{M_szStatus: update.G_cszStatusLatest}
	default:
		return update.Result{M_szStatus: update.G_cszStatusNewer, M_szVersion: pUpdate.Version, M_szUrl: storeupdate.G_cszStorePageUri, M_bStore: true}
	}
}

// G_cszUpdateRepo 는 빌드 때 -ldflags -X 로 정하는 배포 저장소(owner/name)다. 비어 있으면 환경 변수를 본다.
var G_cszUpdateRepo = ""

// G_cdurUpdateFirst 와 G_cdurUpdatePeriod 는 자동 확인의 첫 시점과 주기다 (lifecycle.md §6).
const (
	G_cdurUpdateFirst  = 5 * time.Second
	G_cdurUpdatePeriod = 12 * time.Hour
)

// updateState 는 마지막으로 찾은 새 버전과 종료 신호다.
type updateState struct {
	mtxData     sync.Mutex
	pNewer      *update.Result
	chStop      chan struct{}
	onceStop    sync.Once
	bInstalling atomic.Bool // 스토어 업데이트 확인 창이 떠 있는 중 (겹쳐 띄우지 않는다)
}

// updateRepo 는 배포 저장소 주소를 정한다 (환경 변수 DARITERM_UPDATE_REPO 가 빌드 값보다 우선).
func updateRepo() string {
	if szEnv := os.Getenv("DARITERM_UPDATE_REPO"); szEnv != "" {
		return szEnv
	}
	return G_cszUpdateRepo
}

// updateApi 는 API 주소다 (시험용 DARITERM_UPDATE_API, 없으면 공개 API).
func updateApi() string {
	if szEnv := os.Getenv("DARITERM_UPDATE_API"); szEnv != "" {
		return szEnv
	}
	return "https://api.github.com"
}

// updateFile 은 마지막으로 안내한 버전을 적는 파일 구조다 (lifecycle.md §6).
type updateFile struct {
	M_szNotified string `json:"notified"`
}

// checkUpdate 는 확인을 한 번 하고 필요하면 안내한다. bManual 이면 항상 안내 토스트를 띄운다.
func (pService *Service) checkUpdate(bManual bool) update.Result {
	var res update.Result
	if updateSource() == g_cszUpdateSourceStore {
		res = storeCheckResult()
	} else {
		res = update.Check(updateApi(), updateRepo(), G_cszAppVersion, &http.Client{Timeout: update.G_cdurRequestTimeout})
	}
	if res.M_szStatus != update.G_cszStatusNewer {
		return res
	}
	pState := pService.pUpdate
	pState.mtxData.Lock()
	pState.pNewer = &res
	pState.mtxData.Unlock()

	var file updateFile
	szPath := filepath.Join(pService.szDataDir, "update.json")
	if jsonfile.Exists(szPath) {
		_ = jsonfile.Read(szPath, &file)
	}
	bFirst := file.M_szNotified != res.M_szVersion
	bAnnounce := bFirst || bManual
	mapEvent := map[string]any{"version": res.M_szVersion, "url": res.M_szUrl, "announce": bAnnounce}
	if res.M_bStore {
		mapEvent["store"] = true
	}
	pService.pRouter.Emit("update.available", mapEvent)
	if bAnnounce {
		pService.pRouter.Emit("toast", map[string]any{"kind": "info", "text": i18n.T("update.announce", "version", res.M_szVersion)})
	}
	if bFirst {
		_ = jsonfile.Write(szPath, updateFile{M_szNotified: res.M_szVersion})
	}
	return res
}

// autoUpdateCheck 는 설정이 켜져 있을 때만 주기 확인을 한다.
func (pService *Service) autoUpdateCheck() {
	mapSettings, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return
	}
	mapSet, _ := mapSettings["settings"].(map[string]any)
	if bOn, bOk := mapSet["updateCheck"].(bool); bOk && !bOn {
		return
	}
	pService.checkUpdate(false)
}

// UpdateEnabled 는 업데이트 확인 기능을 쓰는 실행인지다: Microsoft Store 설치본이거나(D115)
// 배포 저장소 주소가 있어야 켜진다(D80). 설치 파일로 설치했고 저장소도 없으면 꺼져 있다.
func UpdateEnabled() bool {
	return updateSource() != ""
}

// runStoreInstall 은 Windows 의 스토어 업데이트 확인 창을 띄우고 결과를 토스트로 알린다 (lifecycle.md §6.1).
// 사용자가 확인하면 Windows 가 내려받아 설치하며, 그 과정에서 앱이 닫힐 수 있다.
// 창을 띄우지 못했거나 Windows 가 오류 상태를 알려 주면 Microsoft Store 의 제품 페이지를 대신 연다.
func (pService *Service) runStoreInstall() {
	fnToast := func(szKind string, szText string) {
		pService.pRouter.Emit("toast", map[string]any{"kind": szKind, "text": szText})
	}
	szState, errInstall := g_fnStoreInstall()
	switch {
	case errInstall != nil:
		fnToast("error", i18n.T("be.update.storeFailed"))
		g_fnOpenStorePage()
	case szState == storeupdate.G_cszStateCompleted:
		fnToast("ok", i18n.T("be.update.storeDone"))
	case szState == storeupdate.G_cszStateCanceled:
		fnToast("info", i18n.T("be.update.storeCanceled"))
	case szState == "NoUpdate":
		pState := pService.pUpdate
		pState.mtxData.Lock()
		pState.pNewer = nil
		pState.mtxData.Unlock()
		fnToast("ok", i18n.T("update.latest"))
	default:
		fnToast("error", i18n.T("be.update.storeState", "state", szState))
		g_fnOpenStorePage()
	}
}

// startUpdateLoop 은 시작 5초 뒤 첫 확인, 이후 12시간마다 확인한다. 기능이 꺼진 빌드면 아무것도 하지 않는다.
func (pService *Service) startUpdateLoop() {
	if !UpdateEnabled() {
		return
	}
	pState := pService.pUpdate
	go func() {
		timerFirst := time.NewTimer(G_cdurUpdateFirst)
		defer timerFirst.Stop()
		select {
		case <-pState.chStop:
			return
		case <-timerFirst.C:
			pService.autoUpdateCheck()
		}
		ticker := time.NewTicker(G_cdurUpdatePeriod)
		defer ticker.Stop()
		for {
			select {
			case <-pState.chStop:
				return
			case <-ticker.C:
				pService.autoUpdateCheck()
			}
		}
	}()
}

// stopUpdate 는 업데이트 루프를 멈춘다.
func (pService *Service) stopUpdate() {
	pState := pService.pUpdate
	pState.onceStop.Do(func() {
		close(pState.chStop)
	})
}

// registerUpdate 는 update.* 채널을 등록한다.
func (pService *Service) registerUpdate() {
	pRouter := pService.pRouter
	pRouter.Handle("update.check", func(slArgs []json.RawMessage) (any, error) {
		return pService.checkUpdate(true), nil
	})
	// update.install: 스토어 설치본이면 Windows 의 업데이트 확인 창을 띄운다(끝을 기다리지 않고 바로 돌아온다).
	// 그 밖의 실행에서는 아무 일도 하지 않고 started=false 를 돌려준다 — 화면이 내려받기 페이지를 연다.
	pRouter.Handle("update.install", func(slArgs []json.RawMessage) (any, error) {
		if updateSource() != g_cszUpdateSourceStore {
			return map[string]any{"started": false}, nil
		}
		pState := pService.pUpdate
		if pState.bInstalling.CompareAndSwap(false, true) {
			go func() {
				defer pState.bInstalling.Store(false)
				pService.runStoreInstall()
			}()
		}
		return map[string]any{"started": true}, nil
	})
	pRouter.Handle("update.status", func(slArgs []json.RawMessage) (any, error) {
		pState := pService.pUpdate
		pState.mtxData.Lock()
		defer pState.mtxData.Unlock()
		if pState.pNewer == nil {
			return nil, nil
		}
		return *pState.pNewer, nil
	})
}
