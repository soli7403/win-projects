package main

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"termgo/internal/gitsync"
)

const (
	syncDebounce = 20 * time.Second // 데이터 변경 후
	syncInterval = 10 * time.Minute // 주기
)

type syncState struct {
	mu      sync.Mutex
	timer   *time.Timer
	running atomic.Bool // 동기화가 이 PC 데이터를 바꾸는 동안 변경 알림을 무시
	last    gitsync.Status
}

func (pstrApp *App) syncAutoOn() bool {
	raw := pstrApp.store.Setting("syncAuto")
	return len(raw) == 0 || string(raw) != "false"
}

// scheduleSync 는 데이터가 바뀐 뒤 약 20초 뒤에 동기화한다 (연속 변경은 하나로 합친다).
func (pstrApp *App) scheduleSync() {
	if pstrApp.sync.RepoURL() == "" || !pstrApp.syncAutoOn() || pstrApp.syn.running.Load() {
		return
	}
	pstrApp.syn.mu.Lock()
	defer pstrApp.syn.mu.Unlock()
	if pstrApp.syn.timer != nil {
		pstrApp.syn.timer.Stop()
	}
	pstrApp.syn.timer = time.AfterFunc(syncDebounce, func() { pstrApp.runSync(gitsync.ModeAuto) })
}

// runSync 는 동기화를 실행하고 결과를 렌더러에 알린다 (sync:status 이벤트). 데이터가 바뀌었으면 store:changed.
func (pstrApp *App) runSync(szMode string) gitsync.Status {
	pstrApp.syn.running.Store(true)
	defer pstrApp.syn.running.Store(false)
	st := pstrApp.sync.Sync(szMode)
	pstrApp.syn.mu.Lock()
	pstrApp.syn.last = st
	pstrApp.syn.mu.Unlock()
	if st.Changed {
		_ = pstrApp.store.Flush()
		pstrApp.emit("store:changed")
	}
	pstrApp.emit("sync:status", st)
	return st
}

// syncLoop: 볼트를 열 때(OnLockChange)와 10분마다 자동 동기화한다.
func (pstrApp *App) syncLoop() {
	t := time.NewTicker(syncInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if pstrApp.sync.RepoURL() != "" && pstrApp.syncAutoOn() && pstrApp.vault.IsUnlocked() {
				pstrApp.runSync(gitsync.ModeAuto)
			}
		case <-pstrApp.ctx.Done():
			return
		}
	}
}

func (pstrApp *App) registerSync() {
	h := pstrApp.router.Handle
	type A = []json.RawMessage
	h("sync:run", func(a A) (any, error) {
		szMode, _ := arg[string](a, 0)
		return pstrApp.runSync(szMode), nil
	})
	h("sync:status", func(A) (any, error) {
		pstrApp.syn.mu.Lock()
		last := pstrApp.syn.last
		pstrApp.syn.mu.Unlock()
		if last.LastSync == 0 {
			last.LastSync = pstrApp.sync.LastSync()
		}
		return map[string]any{
			"configured": pstrApp.sync.RepoURL() != "", "gitAvailable": gitsync.GitAvailable(),
			"last": last,
		}, nil
	})
}
