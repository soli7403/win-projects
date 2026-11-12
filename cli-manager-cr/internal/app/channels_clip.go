package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"termgo/internal/clipboard"
	"termgo/internal/dialog"
)

// WindowControl 은 창 동작(최대화 토글, 원래 크기) 을 맡는 쪽이 구현한다 (main.go 의 Wails 런타임).
type WindowControl interface {
	ToggleMaximise()
	ResetWindow()
}

// g_pWindowControl 은 main.go 가 시작 때 등록하는 창 동작 구현이다. 없으면 창 동작은 무시한다.
var g_pWindowControl WindowControl

// SetWindowControl 은 창 동작 구현을 등록한다.
func SetWindowControl(wc WindowControl) {
	g_pWindowControl = wc
}

// registerClipboard 는 clip.*, dialog.*, app.resetWindow·toggleFullScreen·titlebar 를 등록한다.
func (pService *Service) registerClipboard() {
	pRouter := pService.pRouter
	pRouter.Handle("clip.writeText", func(slArgs []json.RawMessage) (any, error) {
		szText, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return nil, clipboard.WriteText(szText)
	})
	pRouter.Handle("clip.readText", func(slArgs []json.RawMessage) (any, error) {
		return clipboard.ReadText()
	})
	pRouter.Handle("clip.info", func(slArgs []json.RawMessage) (any, error) {
		return clipboard.GetInfo(), nil
	})
	pRouter.Handle("clip.hasImage", func(slArgs []json.RawMessage) (any, error) {
		return clipboard.HasImage(), nil
	})
	pRouter.Handle("dialog.readKeyFile", func(slArgs []json.RawMessage) (any, error) {
		szHome, _ := os.UserHomeDir()
		szPath, errPick := dialog.OpenKeyFile("개인 키 파일 선택", filepath.Join(szHome, ".ssh"))
		if errPick != nil || szPath == "" {
			return json.RawMessage("null"), errPick
		}
		bData, errRead := os.ReadFile(szPath)
		if errRead != nil {
			return nil, errRead
		}
		return map[string]any{"path": szPath, "text": string(bData)}, nil
	})
	pRouter.Handle("dialog.chooseDir", func(slArgs []json.RawMessage) (any, error) {
		szPath, errPick := dialog.ChooseDir("폴더 선택", "", true)
		if errPick != nil || szPath == "" {
			return json.RawMessage("null"), errPick
		}
		return szPath, nil
	})
	pRouter.Handle("app.resetWindow", func(slArgs []json.RawMessage) (any, error) {
		if g_pWindowControl != nil {
			g_pWindowControl.ResetWindow()
		}
		return nil, nil
	})
	pRouter.Handle("app.toggleFullScreen", func(slArgs []json.RawMessage) (any, error) {
		// 진짜 전체 화면이 아니라 최대화 ↔ 복원 토글이다 (spec/ipc.md §2).
		if g_pWindowControl != nil {
			g_pWindowControl.ToggleMaximise()
		}
		return nil, nil
	})
	pRouter.Handle("app.titlebar", func(slArgs []json.RawMessage) (any, error) {
		// 제목 표시줄 색 지정은 현재 동작이 없다 (spec/ipc.md §2).
		return nil, nil
	})
}
