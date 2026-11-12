package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"termgo/internal/app"
)

// lifecycleWindow 는 창 상태, 종료 확인, 백업 대화상자를 Wails 런타임으로 구현한다 (lifecycle.md §4·§5, app.BackupDialogs).
type lifecycleWindow struct {
	mtxData   sync.Mutex
	ctxApp    context.Context
	szDataDir string
	pService  *app.Service
}

// setContext 는 시작 때 Wails 컨텍스트를 받는다.
func (pWin *lifecycleWindow) setContext(ctxApp context.Context) {
	pWin.mtxData.Lock()
	defer pWin.mtxData.Unlock()
	pWin.ctxApp = ctxApp
}

// context 는 현재 Wails 컨텍스트를 돌려준다.
func (pWin *lifecycleWindow) context() context.Context {
	pWin.mtxData.Lock()
	defer pWin.mtxData.Unlock()
	return pWin.ctxApp
}

// rememberWindow 는 설정 "창 크기·위치 기억"을 읽는다 (기본 켬).
func rememberWindow(pService *app.Service) bool {
	szStore, errStore := pService.Router().Invoke("store.get", "[]")
	if errStore != nil {
		return true
	}
	var mapStore struct {
		Settings struct {
			RememberWindow *bool `json:"rememberWindow"`
		} `json:"settings"`
	}
	if errJson := json.Unmarshal([]byte(szStore), &mapStore); errJson != nil || mapStore.Settings.RememberWindow == nil {
		return true
	}
	return *mapStore.Settings.RememberWindow
}

// initialWindow 는 시작 때 창 크기와 최대화 여부를 정한다 (설정이 꺼져 있으면 기본값).
func initialWindow(pService *app.Service, szDataDir string) (iW int, iH int, bMaximised bool, ws app.WindowState) {
	iW, iH = app.G_ciWindowDefaultW, app.G_ciWindowDefaultH
	if !rememberWindow(pService) {
		return iW, iH, false, ws
	}
	ws, bHave := app.LoadWindowState(szDataDir)
	if !bHave || !ws.Valid() {
		return iW, iH, false, app.WindowState{}
	}
	return ws.M_iW, ws.M_iH, ws.M_bMaximised, ws
}

// placeWindow 는 화면이 준비되면 저장된 위치를 적용한다 (화면 밖이면 가운데, lifecycle.md §4).
func (pWin *lifecycleWindow) placeWindow(ws app.WindowState) {
	ctxApp := pWin.context()
	if ctxApp == nil {
		return
	}
	if !rememberWindow(pWin.pService) || !ws.M_bHasPos {
		runtime.WindowCenter(ctxApp)
		return
	}
	iScreenW, iScreenH := 0, 0
	if slScreens, errScreens := runtime.ScreenGetAll(ctxApp); errScreens == nil {
		for _, screen := range slScreens {
			if screen.IsPrimary {
				iScreenW, iScreenH = screen.Width, screen.Height
			}
		}
	}
	iX, iY, bCenter := app.StartupPlacement(ws, iScreenW, iScreenH)
	if bCenter {
		runtime.WindowCenter(ctxApp)
		return
	}
	runtime.WindowSetPosition(ctxApp, iX, iY)
}

// beforeClose 는 창 닫기 요청을 처리한다. 세션이 있으면 확인하고, 취소하면 true(닫지 않음)를 돌려준다.
func (pWin *lifecycleWindow) beforeClose(pService *app.Service) bool {
	ctxApp := pWin.context()
	slConnected := app.ConnectedSessions(pService.SessionInfos())
	if app.NeedsCloseConfirm(len(slConnected)) {
		szChoice, errDialog := runtime.MessageDialog(ctxApp, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "TermGo",
			Message:       app.CloseConfirmText(len(slConnected), app.CountKeepSessions(slConnected)),
			Buttons:       []string{"종료", "취소"},
			DefaultButton: "취소",
			CancelButton:  "취소",
		})
		if !app.IsQuitChoice(szChoice, errDialog != nil) {
			return true
		}
	}
	if rememberWindow(pService) {
		pWin.saveWindow()
	}
	return false
}

// saveWindow 는 닫기 직전의 창 상태를 window.json 에 저장한다 (최대화면 일반 상태를 보존).
func (pWin *lifecycleWindow) saveWindow() {
	ctxApp := pWin.context()
	prev, bHave := app.LoadWindowState(pWin.szDataDir)
	bMaximised := runtime.WindowIsMaximised(ctxApp)
	iX, iY := runtime.WindowGetPosition(ctxApp)
	iW, iH := runtime.WindowGetSize(ctxApp)
	_ = app.SaveWindowState(pWin.szDataDir, app.CloseState(prev, bHave, iX, iY, iW, iH, bMaximised))
}

// SaveFile 은 백업 저장 대화상자를 연다 (기본 폴더는 문서 폴더). 취소하면 bOk=false.
func (pWin *lifecycleWindow) SaveFile(szDefaultName string, szFilterName string, szPattern string) (string, bool) {
	szPath, errDialog := runtime.SaveFileDialog(pWin.context(), runtime.SaveDialogOptions{
		Title:            "백업 저장",
		DefaultDirectory: documentsDir(),
		DefaultFilename:  szDefaultName,
		Filters:          []runtime.FileFilter{{DisplayName: szFilterName, Pattern: szPattern}},
	})
	if errDialog != nil || szPath == "" {
		return "", false
	}
	return szPath, true
}

// OpenFile 은 백업 열기 대화상자를 연다. 취소하면 bOk=false.
func (pWin *lifecycleWindow) OpenFile(szFilterName string, szPattern string) (string, bool) {
	szPath, errDialog := runtime.OpenFileDialog(pWin.context(), runtime.OpenDialogOptions{
		Title:   "백업 파일 선택",
		Filters: []runtime.FileFilter{{DisplayName: szFilterName, Pattern: szPattern}},
	})
	if errDialog != nil || szPath == "" {
		return "", false
	}
	return szPath, true
}

// documentsDir 는 문서 폴더 경로를 돌려준다 (없으면 홈 폴더).
func documentsDir() string {
	szHome, errHome := os.UserHomeDir()
	if errHome != nil {
		return ""
	}
	szDocs := filepath.Join(szHome, "Documents")
	if info, errStat := os.Stat(szDocs); errStat == nil && info.IsDir() {
		return szDocs
	}
	return szHome
}
