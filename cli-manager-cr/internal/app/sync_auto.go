package app

import (
	"sync"
	"sync/atomic"
	"time"

	"dariterm/internal/gitsync"
)

// G_cdurCloseWait 은 종료 때 세션 정리를 기다리는 최대 시간이다 (3초). 넘기면 정리를 두고 종료한다.
const G_cdurCloseWait = 3 * time.Second

// G_cdurSyncDebounce 은 데이터 변경 뒤 자동 동기화까지의 대기다 (sync-backup.md §5.5: 바뀐 뒤 20초).
const G_cdurSyncDebounce = 20 * time.Second

// G_cdurSyncPeriod 는 볼트가 열려 있을 때의 주기 동기화 간격이다 (10분).
const G_cdurSyncPeriod = 10 * time.Minute

// syncState 는 자동 동기화 상태다.
type syncState struct {
	mtxData       sync.Mutex
	pService      *Service
	syncer        *gitsync.Syncer
	timerDebounce *time.Timer
	bRunning      atomic.Bool
	chStop        chan struct{}
	onceStop      sync.Once
}

// newSyncState 는 동기화 실행기와 자동 실행 상태를 만든다.
func newSyncState(pService *Service) *syncState {
	pState := &syncState{pService: pService, chStop: make(chan struct{})}
	pState.syncer = gitsync.NewSyncer(gitsync.Deps{
		M_szDataDir:     pService.szDataDir,
		M_store:         storeLocal{pService: pService},
		M_vault:         gitsync.VaultAdapter{M_pVault: pService.pVault},
		M_fnAskPassword: pService.askPassword,
		M_fnNow:         time.Now,
	})
	return pState
}

// finishSync 는 동기화 결과를 알린다 (데이터가 바뀌었으면 저장 후 store.changed, 항상 sync.status).
func (pService *Service) finishSync(res gitsync.Result) {
	if res.M_bChanged {
		_ = pService.pStore.Flush()
		pService.pRouter.Emit("store.changed")
	}
	pService.pRouter.Emit("sync.status", res)
}

// runAutoSync 는 저장소 주소가 있고 자동 동기화가 켜졌을 때 한 번 돈다. 이미 도는 중이면 건너뛴다.
func (pService *Service) runAutoSync() {
	szRepo, bAuto := pService.syncRepoSetting()
	if szRepo == "" || !bAuto {
		return
	}
	if !pService.pSync.bRunning.CompareAndSwap(false, true) {
		return
	}
	defer pService.pSync.bRunning.Store(false)
	res := pService.pSync.syncer.Run(szRepo, gitsync.G_cszModeAuto)
	pService.finishSync(res)
}

// scheduleSync 는 데이터 변경 뒤 20초 후 자동 동기화를 건다. 연속 변경은 마지막 변경 기준으로 합친다.
// 동기화가 도는 중의 변경 알림은 무시한다 (sync-backup.md §5.5).
func (pService *Service) scheduleSync() {
	if pService.pSync == nil || pService.pSync.bRunning.Load() {
		return
	}
	pState := pService.pSync
	pState.mtxData.Lock()
	defer pState.mtxData.Unlock()
	if pState.timerDebounce != nil {
		pState.timerDebounce.Stop()
	}
	pState.timerDebounce = time.AfterFunc(G_cdurSyncDebounce, pService.runAutoSync)
}

// onVaultLockChanged 는 볼트가 열릴 때 한 번 자동 동기화를 돌린다 (sync-backup.md §5.5).
func (pService *Service) onVaultLockChanged(bOpen bool) {
	if bOpen {
		go pService.runAutoSync()
	}
}

// startSyncLoop 는 볼트가 열려 있을 때 10분마다 자동 동기화를 돌린다.
func (pService *Service) startSyncLoop() {
	go func() {
		ticker := time.NewTicker(G_cdurSyncPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-pService.pSync.chStop:
				return
			case <-ticker.C:
				if pService.pVault.Status().M_bUnlocked {
					pService.runAutoSync()
				}
			}
		}
	}()
}

// stopSync 는 자동 동기화 타이머와 주기 루프를 멈춘다 (종료 시).
func (pService *Service) stopSync() {
	pState := pService.pSync
	pState.onceStop.Do(func() {
		close(pState.chStop)
	})
	pState.mtxData.Lock()
	defer pState.mtxData.Unlock()
	if pState.timerDebounce != nil {
		pState.timerDebounce.Stop()
	}
}
