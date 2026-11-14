package main

import (
	"context"
	"embed"
	"log"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"dariterm/internal/app"
)

//go:embed all:frontend/dist
var gfsAssets embed.FS

// Emitter 는 백엔드 이벤트를 Wails 이벤트로 보낸다. 시작 전에 온 이벤트는 버린다.
type Emitter struct {
	mtxData sync.Mutex
	ctxApp  context.Context
}

// Emit 은 이벤트 이름과 값들을 프런트로 보낸다 (spec/ipc.md §4).
// 값이 여러 개인 이벤트(session.data 등)는 인자를 따로 넘긴다 (프런트의 EventsOn 콜백 인자와 같다).
func (pEmitter *Emitter) Emit(szEvent string, anyData ...any) {
	pEmitter.mtxData.Lock()
	ctxApp := pEmitter.ctxApp
	pEmitter.mtxData.Unlock()
	if ctxApp != nil {
		runtime.EventsEmit(ctxApp, szEvent, anyData...)
	}
}

func (pEmitter *Emitter) setContext(ctxApp context.Context) {
	pEmitter.mtxData.Lock()
	defer pEmitter.mtxData.Unlock()
	pEmitter.ctxApp = ctxApp
}

// App 은 프런트에 노출하는 4개 메서드다 (spec/ipc.md §1).
// 프런트는 window.go.main.App 으로 부른다. Wails Bind 대상은 main 패키지의 App 이어야 한다.
type App struct {
	pService *app.Service
}

// Invoke 는 요청-응답 채널을 처리한다.
func (pApp *App) Invoke(szChannel string, szArgsJson string) (string, error) {
	return pApp.pService.Router().Invoke(szChannel, szArgsJson)
}

// TermInput 은 터미널 입력을 세션으로 보낸다.
func (pApp *App) TermInput(szSessionId string, szText string) {
	pApp.pService.TermInput(szSessionId, szText)
}

// TermResize 는 터미널 크기를 세션으로 보낸다.
func (pApp *App) TermResize(szSessionId string, iCols int, iRows int) {
	pApp.pService.TermResize(szSessionId, iCols, iRows)
}

// Reply 는 ui.request 질문에 대한 답을 전한다.
func (pApp *App) Reply(szRequestId string, szValueJson string) {
	pApp.pService.Router().Reply(szRequestId, szValueJson)
}

func main() {
	szDataDir, errDir := app.DataDir()
	if errDir != nil {
		log.Fatalf("cannot open data folder: %v", errDir)
	}
	emitter := &Emitter{}
	pService, errService := app.NewService(szDataDir, emitter)
	if errService != nil {
		log.Fatalf("%v", errService)
	}
	pApp := &App{pService: pService}
	winHost := &lifecycleWindow{szDataDir: szDataDir, pService: pService}
	pService.SetBackupDialogs(winHost)
	iWidth, iHeight, bMaximised, wsSaved := initialWindow(pService, szDataDir)
	startState := options.Normal
	if bMaximised {
		startState = options.Maximised
	}
	app.SetWindowControl(&wailsWindow{em: emitter})

	errRun := wails.Run(&options.App{
		Title:            "DariTerm",
		Width:            iWidth,
		WindowStartState: startState,
		Height:           iHeight,
		MinWidth:         900,
		MinHeight:        560,
		AssetServer: &assetserver.Options{
			Assets: gfsAssets,
		},
		Bind: []any{pApp},
		OnStartup: func(ctxApp context.Context) {
			emitter.setContext(ctxApp)
			winHost.setContext(ctxApp)
		},
		OnDomReady: func(ctxApp context.Context) {
			winHost.placeWindow(wsSaved)
		},
		OnBeforeClose: func(ctxApp context.Context) bool {
			return winHost.beforeClose(pService)
		},
		OnShutdown: func(ctxApp context.Context) {
			if errClose := pService.Close(); errClose != nil {
				log.Printf("save on exit failed: %v", errClose)
			}
		},
		LogLevel: logger.ERROR,
	})
	if errRun != nil {
		log.Fatalf("app run failed: %v", errRun)
	}
}

// context 는 시작 뒤의 Wails 컨텍스트를 돌려준다 (없으면 nil).
func (pEmitter *Emitter) context() context.Context {
	pEmitter.mtxData.Lock()
	defer pEmitter.mtxData.Unlock()
	return pEmitter.ctxApp
}

// wailsWindow 는 창 동작(최대화 토글, 원래 크기)을 Wails 런타임으로 한다.
type wailsWindow struct {
	em *Emitter
}

// ToggleMaximise 는 최대화 ↔ 복원을 바꾼다 (진짜 전체 화면 아님, spec/ipc.md §2).
func (pWin *wailsWindow) ToggleMaximise() {
	if ctxApp := pWin.em.context(); ctxApp != nil {
		runtime.WindowToggleMaximise(ctxApp)
	}
}

// ResetWindow 는 최대화를 풀고 1440×900 으로 가운데에 둔다 (lifecycle.md §4 [원래 크기로]).
func (pWin *wailsWindow) ResetWindow() {
	if ctxApp := pWin.em.context(); ctxApp != nil {
		runtime.WindowUnmaximise(ctxApp)
		runtime.WindowSetSize(ctxApp, 1440, 900)
		runtime.WindowCenter(ctxApp)
	}
}
