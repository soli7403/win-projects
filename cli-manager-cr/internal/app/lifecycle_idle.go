package app

import (
	"sync"
	"time"
)

// G_cdurAutoLockInterval 은 자동 잠금 검사 주기다 (lifecycle.md §2: 20초).
const G_cdurAutoLockInterval = 20 * time.Second

// G_cdurSuspendGap 은 검사 사이 벽시계 간격이 이보다 크면 절전 등으로 멈췄던 것으로 본다.
const G_cdurSuspendGap = 90 * time.Second

// lockState 는 자동 잠금 루프의 종료 신호다.
type lockState struct {
	chStop   chan struct{}
	onceStop sync.Once
}

// ShouldAutoLock 은 자동 잠금을 해야 하는지 판정한다 (lifecycle.md §2).
// 자동 잠금 분이 0 이하이거나 볼트가 잠겨 있으면 하지 않는다. 잠금·절전 신호가 있으면 잠그고,
// 아니면 마지막 입력 이후 경과가 설정 분 이상일 때 잠근다. 입력 시각을 알 수 없으면 유휴로는 잠그지 않는다.
func ShouldAutoLock(iMinutes int, bUnlocked bool, bLockOrSuspend bool, bIdleKnown bool, i64IdleMs int64) bool {
	if iMinutes <= 0 || !bUnlocked {
		return false
	}
	if bLockOrSuspend {
		return true
	}
	return bIdleKnown && i64IdleMs >= int64(iMinutes)*60*1000
}

// SuspendDetected 는 검사 두 번 사이 벽시계 간격으로 절전을 추정한다.
func SuspendDetected(tmPrev time.Time, tmNow time.Time) bool {
	return tmNow.Sub(tmPrev) > G_cdurSuspendGap
}

// autoLockMinutes 는 설정의 자동 잠금 분을 읽는다 (없으면 0).
func (pService *Service) autoLockMinutes() int {
	mapSettings, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return 0
	}
	mapSet, _ := mapSettings["settings"].(map[string]any)
	fMinutes, bOk := mapSet["autoLockMinutes"].(float64)
	if !bOk {
		return 0
	}
	return int(fMinutes)
}

// autoLockTick 은 한 번의 검사다. 잠금 조건이 맞으면 볼트를 잠근다 (세션은 유지).
func (pService *Service) autoLockTick(bSuspend bool) {
	iMinutes := pService.autoLockMinutes()
	if iMinutes <= 0 || !pService.pVault.Status().M_bUnlocked {
		return
	}
	bLockedDesktop := desktopLockedOS()
	i64Idle, bIdleKnown := idleMillisOS()
	if ShouldAutoLock(iMinutes, true, bSuspend || bLockedDesktop, bIdleKnown, i64Idle) {
		pService.pVault.Lock()
	}
}

// startAutoLock 은 20초마다 자동 잠금을 검사한다.
func (pService *Service) startAutoLock() {
	pState := pService.pLock
	go func() {
		ticker := time.NewTicker(G_cdurAutoLockInterval)
		defer ticker.Stop()
		tmPrev := time.Now()
		for {
			select {
			case <-pState.chStop:
				return
			case tmNow := <-ticker.C:
				bSuspend := SuspendDetected(tmPrev, tmNow)
				tmPrev = tmNow
				pService.autoLockTick(bSuspend)
			}
		}
	}()
}

// stopAutoLock 은 자동 잠금 루프를 멈춘다.
func (pService *Service) stopAutoLock() {
	pState := pService.pLock
	pState.onceStop.Do(func() {
		close(pState.chStop)
	})
}
