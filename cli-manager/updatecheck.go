package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"termgo/internal/store"
	"termgo/internal/update"
)

// g_szUpdateRepo 는 배포 저장소 "owner/name". 빌드 시 -ldflags "-X main.g_szUpdateRepo=owner/name" 로 정한다.
// 비어 있으면 새 버전 확인을 하지 않는다 (TERMGO_UPDATE_REPO 환경 변수로도 지정 가능).
var g_szUpdateRepo = ""

const updateInterval = 12 * time.Hour

type updateState struct {
	mu       sync.Mutex
	latest   *update.Result
	notified string
}

func (pstrApp *App) updateFile() string { return filepath.Join(pstrApp.szDir, "update.json") }

func (pstrApp *App) checker() *update.Checker {
	szRepo := g_szUpdateRepo
	if e := os.Getenv("TERMGO_UPDATE_REPO"); e != "" {
		szRepo = e
	}
	return &update.Checker{Repo: szRepo, Current: g_szVersion, BaseURL: os.Getenv("TERMGO_UPDATE_API")}
}

// runUpdateCheck 는 한 번 확인한다. 새 버전이면 상태에 기록하고 update:available 이벤트를 보낸다 (처음 발견했을 때 한 번만 안내).
func (pstrApp *App) runUpdateCheck(bManual bool) update.Result {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r := pstrApp.checker().Check(ctx)
	if r.Status == "newer" {
		var saved struct {
			Notified string `json:"notified"`
		}
		_ = store.ReadJSON(pstrApp.updateFile(), &saved)
		bFirst := saved.Notified != r.Version
		pstrApp.upd.mu.Lock()
		pstrApp.upd.latest = &r
		pstrApp.upd.mu.Unlock()
		pstrApp.emit("update:available", map[string]any{"version": r.Version, "url": r.URL, "announce": bFirst || bManual})
		if bFirst {
			_ = store.WriteJSONAtomic(pstrApp.updateFile(), map[string]string{"notified": r.Version})
		}
	}
	return r
}

// updateLoop: 시작 시와 이후 12시간마다 (설정에서 끌 수 있음). 실패는 조용히 무시한다 (폐쇄망).
func (pstrApp *App) updateLoop() {
	select {
	case <-time.After(5 * time.Second):
	case <-pstrApp.ctx.Done():
		return
	}
	for {
		if raw := pstrApp.store.Setting("updateCheck"); len(raw) == 0 || string(raw) != "false" {
			pstrApp.runUpdateCheck(false)
		}
		select {
		case <-time.After(updateInterval):
		case <-pstrApp.ctx.Done():
			return
		}
	}
}

func (pstrApp *App) registerUpdate() {
	h := pstrApp.router.Handle
	h("update:check", func([]json.RawMessage) (any, error) { return pstrApp.runUpdateCheck(true), nil })
	h("update:status", func([]json.RawMessage) (any, error) {
		pstrApp.upd.mu.Lock()
		defer pstrApp.upd.mu.Unlock()
		return pstrApp.upd.latest, nil
	})
}
