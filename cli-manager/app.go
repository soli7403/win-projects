package main

import (
	"context"
	"os"
	"path/filepath"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"

	"termgo/internal/gitsync"
	"termgo/internal/ipc"
	"termgo/internal/session"
	"termgo/internal/sftpsvc"
	"termgo/internal/store"
	"termgo/internal/ui"
	"termgo/internal/vault"
)

// g_szVersion 은 빌드 시 -ldflags "-X main.g_szVersion=..." 로 덮어쓴다.
var g_szVersion = "0.1.0-dev"

// App 은 Wails 에 바인딩되는 객체. window.bt 어댑터가 Call/Write/Resize/Respond 를 부른다.
type App struct {
	ctx      context.Context
	router   *ipc.Router
	bridge   *ui.Bridge
	store    *store.Store
	vault    *vault.Vault
	sessions *session.Manager
	sftp     *sftpsvc.Service
	upd      updateState
	sync     *gitsync.Service
	syn      syncState
	szDir    string
}

// NewApp 은 데이터 폴더를 열고 서비스와 채널 핸들러를 구성한다.
func NewApp() (*App, error) {
	szDir, err := store.DataDir()
	if err != nil {
		return nil, err
	}
	st, err := store.Open(szDir)
	if err != nil {
		return nil, err
	}
	pstrApp := &App{router: ipc.NewRouter(), store: st, vault: vault.Open(szDir), szDir: szDir}
	pstrApp.bridge = ui.NewBridge(pstrApp.emit)
	pstrApp.vault.OnLockChange = func(bUnlocked bool) {
		pstrApp.emit("vault:changed", bUnlocked)
		if bUnlocked && pstrApp.sync != nil && pstrApp.sync.RepoURL() != "" && pstrApp.syncAutoOn() {
			go pstrApp.runSync(gitsync.ModeAuto) // 볼트를 열 때 동기화
		}
	}
	pstrApp.vault.OnItemsChange = func() { pstrApp.scheduleSync() }
	st.OnChange = func(keys []string) {
		for _, k := range keys {
			if k == "groups" || k == "hosts" || k == "snippets" {
				pstrApp.scheduleSync()
				return
			}
		}
	}
	pstrApp.sessions = session.NewManager(session.Deps{Store: st, Vault: pstrApp.vault, UI: pstrApp.bridge, DataDir: szDir})
	pstrApp.sftp = sftpsvc.New(pstrApp.sessions, pstrApp.bridge, st)
	pstrApp.registerHandlers()
	pstrApp.registerSftp()
	pstrApp.registerImport()
	pstrApp.sync = &gitsync.Service{Dir: szDir, Store: st, Vault: pstrApp.vault, UI: pstrApp.bridge}
	pstrApp.registerUpdate()
	pstrApp.registerSync()
	pstrApp.registerPending()
	return pstrApp, nil
}

func (pstrApp *App) startup(ctx context.Context) {
	pstrApp.ctx = ctx
	go pstrApp.autoLockLoop()
	go pstrApp.updateLoop()
	go pstrApp.syncLoop()
}

func (pstrApp *App) shutdown(context.Context) {
	pstrApp.sessions.CloseAll()
	_ = pstrApp.store.Flush()
}

// emit 은 렌더러의 bt.on(ch, cb) 로 이벤트를 보낸다.
func (pstrApp *App) emit(szCh string, args ...any) {
	if pstrApp.ctx != nil {
		wr.EventsEmit(pstrApp.ctx, szCh, args...)
	}
}

// Call 은 v1.1.0 의 ipcRenderer.invoke 대응. 인자·결과는 JSON 문자열.
func (pstrApp *App) Call(szCh, szArgsJSON string) (string, error) {
	return pstrApp.router.Call(szCh, szArgsJSON)
}

// Write 는 세션 입력 (session:write).
func (pstrApp *App) Write(szID, szData string) {
	if s := pstrApp.sessions.Get(szID); s != nil {
		s.Write(szData)
	}
}

// Resize 는 세션 크기 변경 (session:resize).
func (pstrApp *App) Resize(szID string, iCols, iRows int) {
	if s := pstrApp.sessions.Get(szID); s != nil {
		s.SetSize(iCols, iRows)
	}
}

// Respond 는 UI 요청(ui:request)에 대한 응답 (ui:respond).
func (pstrApp *App) Respond(szReqID, szValueJSON string) {
	pstrApp.bridge.Respond(szReqID, szValueJSON)
}

func (pstrApp *App) logsDir() string { return filepath.Join(pstrApp.szDir, "logs") }

func ensureDir(sz string) string { _ = os.MkdirAll(sz, 0o700); return sz }
