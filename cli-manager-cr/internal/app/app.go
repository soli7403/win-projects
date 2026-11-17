// Package app 은 저장소·볼트·IPC 를 묶어 채널 핸들러를 등록한다 (spec/ipc.md §2 의 저장 데이터·볼트·계정 채널).
package app

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"time"

	"dariterm/internal/i18n"
	"dariterm/internal/ipc"
	"dariterm/internal/license"
	"dariterm/internal/logclean"
	"dariterm/internal/session"
	"dariterm/internal/store"
	"dariterm/internal/vault"
)

// G_cszAppVersion 은 앱 버전이다. 실행 파일은 시작할 때 wails.json 의 info.productVersion 으로 덮어쓴다(main.go, D82).
// 여기 값은 그 값을 못 읽었을 때와 테스트용 기본값이다.
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
	pLogClean *logclean.Cleaner
}

// G_ciSessionLogKeepDaysDefault 는 설정 sessionLogKeepDays 의 기본값이다 (data/store.md, session.md §6).
const G_ciSessionLogKeepDaysDefault = 30

// DataDir 은 datadir.go (DariTerm 폴더).

// NewService 는 데이터 폴더의 store.json·vault.json 을 열고 채널을 등록한다.
// 실패하면 오류를 돌려준다 (앱은 이때 오류로 종료한다, lifecycle.md §1).
func NewService(szDataDir string, emitter ipc.Emitter) (*Service, error) {
	pStore, errStore := store.Open(filepath.Join(szDataDir, "store.json"))
	if errStore != nil {
		return nil, i18n.Errorf("be.app.storeOpen", errStore)
	}
	pVault, errVault := vault.Open(filepath.Join(szDataDir, "vault.json"))
	if errVault != nil {
		return nil, i18n.Errorf("be.app.vaultOpen", errVault)
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
		if bOpen && pService.pSessions != nil {
			// 잠금 해제를 기다리던 세션을 곧바로 다시 접속한다 (session.md §4).
			go pService.pSessions.ResumeUnlockWaiters()
		}
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
	pService.applyLanguage()
	pService.registerStore()
	pService.registerVault()
	pService.registerCred()
	pService.registerApp()
	pService.registerLicense()
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
	pService.startLogClean()
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
	pService.pLogClean.Stop()
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
		iKeepBefore := pService.logKeepDays()
		errPatch := pService.pStore.Patch(mapPatch)
		pService.applyLanguage()
		if pService.logKeepDays() != iKeepBefore {
			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
		}
		return nil, errPatch
	})
}

// logKeepDays 는 세션 로그 보관 일수 설정이다 (0 = 지우지 않음).
func (pService *Service) logKeepDays() int {
	return pService.settingInt("sessionLogKeepDays", G_ciSessionLogKeepDaysDefault)
}

// startLogClean 은 오래된 세션 로그 정리 루프를 시작한다 (시작 10초 뒤, 그 뒤 1시간마다).
// 지금 기록 중인 로그 파일은 건너뛴다.
func (pService *Service) startLogClean() {
	pService.pLogClean = logclean.New(filepath.Join(pService.szDataDir, "logs"), pService.logKeepDays, session.IsLogOpen)
	pService.pLogClean.Start()
}

// applyLanguage 는 백엔드 문구 언어를 설정(settings.language, 비면 OS 언어)에 맞춘다 (D75).
func (pService *Service) applyLanguage() {
	szSetting := ""
	if mapSnap, errSnap := pService.pStore.Snapshot(); errSnap == nil {
		if mapSettings, bOk := mapSnap["settings"].(map[string]any); bOk {
			szSetting, _ = mapSettings["language"].(string)
		}
	}
	i18n.SetLang(i18n.Resolve(szSetting))
}

func (pService *Service) registerApp() {
	pService.pRouter.Handle("app.info", func(slArgs []json.RawMessage) (any, error) {
		return map[string]any{
			"version":  G_cszAppVersion,
			"platform": runtime.GOOS,
			"dataDir":  pService.szDataDir,
			// OS 표시 언어를 지원 언어로 맞춘 값(D75): 첫 실행 언어 선택의 기본값.
			"osLanguage": i18n.Match(i18n.DetectOS()),
			// 회사 도메인(AD/Azure AD) 가입 PC 인지(D78): 개인용 안내에만 쓰고 어디에도 보내지 않는다.
			"domainJoined": license.DomainJoined(),
			// 업데이트 확인 기능을 쓰는 빌드인지(D80). false 면 화면에서 업데이트 항목을 감춘다.
			"updateEnabled": UpdateEnabled(),
		}, nil
	})
}
