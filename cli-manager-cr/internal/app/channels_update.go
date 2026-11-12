package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"termgo/internal/jsonfile"
	"termgo/internal/update"
)

// G_cszUpdateRepo 는 빌드 때 -ldflags -X 로 정하는 배포 저장소(owner/name)다. 비어 있으면 환경 변수를 본다.
var G_cszUpdateRepo = ""

// G_cdurUpdateFirst 와 G_cdurUpdatePeriod 는 자동 확인의 첫 시점과 주기다 (lifecycle.md §6).
const (
	G_cdurUpdateFirst  = 5 * time.Second
	G_cdurUpdatePeriod = 12 * time.Hour
)

// updateState 는 마지막으로 찾은 새 버전과 종료 신호다.
type updateState struct {
	mtxData  sync.Mutex
	pNewer   *update.Result
	chStop   chan struct{}
	onceStop sync.Once
}

// updateRepo 는 배포 저장소 주소를 정한다 (환경 변수 TERMGO_UPDATE_REPO 가 빌드 값보다 우선).
func updateRepo() string {
	if szEnv := os.Getenv("TERMGO_UPDATE_REPO"); szEnv != "" {
		return szEnv
	}
	return G_cszUpdateRepo
}

// updateApi 는 API 주소다 (시험용 TERMGO_UPDATE_API, 없으면 공개 API).
func updateApi() string {
	if szEnv := os.Getenv("TERMGO_UPDATE_API"); szEnv != "" {
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
	res := update.Check(updateApi(), updateRepo(), G_cszAppVersion, &http.Client{Timeout: update.G_cdurRequestTimeout})
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
	pService.pRouter.Emit("update.available", map[string]any{"version": res.M_szVersion, "url": res.M_szUrl, "announce": bAnnounce})
	if bAnnounce {
		pService.pRouter.Emit("toast", map[string]any{"kind": "info", "text": "새 버전 " + res.M_szVersion + " 이 있습니다 (상태 표시줄을 눌러 확인)"})
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

// startUpdateLoop 은 시작 5초 뒤 첫 확인, 이후 12시간마다 확인한다.
func (pService *Service) startUpdateLoop() {
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
