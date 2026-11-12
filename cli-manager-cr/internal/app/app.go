// Package app 은 저장소·볼트·IPC 를 묶어 채널 핸들러를 등록한다 (spec/ipc.md §2 의 저장 데이터·볼트·계정 채널).
package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"termgo/internal/ipc"
	"termgo/internal/session"
	"termgo/internal/store"
	"termgo/internal/vault"
)

// G_cszAppVersion 은 앱 버전이다. 빌드 때 덮어쓸 수 있다.
var G_cszAppVersion = "0.1.0"

// Service 는 백엔드 한 벌(저장소, 볼트, 라우터)이다.
type Service struct {
	pRouter   *ipc.Router
	pStore    *store.Store
	pVault    *vault.Vault
	pSessions *session.Manager
	szDataDir string
	pSync     *syncState
	pDialogs  BackupDialogs
	pUpdate   *updateState
	pLock     *lockState
}

// DataDir 은 datadir.go (DariTerm 폴더, 옛 TermGo 폴더에서 복사 이전).

// NewService 는 데이터 폴더의 store.json·vault.json 을 열고 채널을 등록한다.
// 실패하면 오류를 돌려준다 (앱은 이때 오류로 종료한다, lifecycle.md §1).
func NewService(szDataDir string, emitter ipc.Emitter) (*Service, error) {
	pStore, errStore := store.Open(filepath.Join(szDataDir, "store.json"))
	if errStore != nil {
		return nil, fmt.Errorf("저장 데이터를 열 수 없습니다: %w", errStore)
	}
	pVault, errVault := vault.Open(filepath.Join(szDataDir, "vault.json"))
	if errVault != nil {
		return nil, fmt.Errorf("볼트를 열 수 없습니다: %w", errVault)
	}
	pService := &Service{
		pRouter:   ipc.NewRouter(emitter),
		pStore:    pStore,
		pVault:    pVault,
		szDataDir: szDataDir,
	}
	pVault.SetOnLockChanged(func(bOpen bool) {
		pService.pRouter.Emit("vault.changed", bOpen)
		pService.onVaultLockChanged(bOpen)
	})
	pStore.SetOnSyncKeys(func() {
		pService.scheduleSync()
	})
	pVault.SetOnItemsChanged(func() {
		pService.scheduleSync()
	})
	pService.pSessions = session.NewManager(func(szEvent string, anyData ...any) {
		pService.pRouter.Emit(szEvent, anyData...)
	})
	pService.registerStore()
	pService.registerVault()
	pService.registerCred()
	pService.registerApp()
	pService.registerSession()
	pService.registerShell()
	pService.registerImport()
	pService.pSync = newSyncState(pService)
	pService.registerSync()
	pService.registerBackup()
	pService.pUpdate = &updateState{chStop: make(chan struct{})}
	pService.registerUpdate()
	pService.startUpdateLoop()
	pService.pLock = &lockState{chStop: make(chan struct{})}
	pService.startAutoLock()
	pService.startSyncLoop()
	pService.registerSftp()
	pService.registerClipboard()
	return pService, nil
}

// TermInput 은 터미널 입력을 세션으로 보낸다 (spec/ipc.md §1 TermInput).
func (pService *Service) TermInput(szSessionId string, szText string) {
	pSession, errGet := pService.pSessions.Get(szSessionId)
	if errGet != nil {
		return
	}
	pSession.Input(szText)
}

// TermResize 는 터미널 크기를 세션으로 보낸다 (spec/ipc.md §1 TermResize).
func (pService *Service) TermResize(szSessionId string, iCols int, iRows int) {
	pSession, errGet := pService.pSessions.Get(szSessionId)
	if errGet != nil {
		return
	}
	pSession.Resize(iCols, iRows)
}

// Router 는 IPC 라우터를 돌려준다.
func (pService *Service) Router() *ipc.Router {
	return pService.pRouter
}

// Close 는 종료 시 store 를 즉시 저장하고 볼트를 잠근다.
func (pService *Service) Close() error {
	pService.stopSync()
	pService.stopAutoLock()
	pService.stopUpdate()
	chDone := make(chan struct{})
	go func() {
		pService.pSessions.CloseAll()
		close(chDone)
	}()
	select {
	case <-chDone:
	case <-time.After(G_cdurCloseWait):
		// 세션 정리가 막혀도 프로세스 종료를 붙잡지 않는다 (제한 시간 뒤 진행).
	}
	pService.pVault.Lock()
	return pService.pStore.Flush()
}

func (pService *Service) registerStore() {
	pService.pRouter.Handle("store.get", func(slArgs []json.RawMessage) (any, error) {
		return pService.pStore.Snapshot()
	})
	pService.pRouter.Handle("store.patch", func(slArgs []json.RawMessage) (any, error) {
		var mapPatch map[string]any
		if errArg := argInto(slArgs, 0, &mapPatch); errArg != nil {
			return nil, errArg
		}
		return nil, pService.pStore.Patch(mapPatch)
	})
}

func (pService *Service) registerApp() {
	pService.pRouter.Handle("app.info", func(slArgs []json.RawMessage) (any, error) {
		return map[string]any{
			"version":  G_cszAppVersion,
			"platform": runtime.GOOS,
			"dataDir":  pService.szDataDir,
		}, nil
	})
}
