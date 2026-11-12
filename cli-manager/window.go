package main

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/options"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"

	"termgo/internal/store"
)

const (
	defaultW = 1440
	defaultH = 900
)

// winState 는 창 크기·위치·최대화 상태 (window.json).
type winState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	W         int  `json:"w"`
	H         int  `json:"h"`
	Maximised bool `json:"maximised"`
	HasPos    bool `json:"hasPos"`
}

func (pstrApp *App) winFile() string { return filepath.Join(pstrApp.szDir, "window.json") }

func (pstrApp *App) loadWin() (winState, bool) {
	var w winState
	if err := store.ReadJSON(pstrApp.winFile(), &w); err != nil || w.W < 400 || w.H < 300 {
		return winState{}, false
	}
	return w, true
}

// remember 는 "창 크기·위치 기억" 설정 (기본 켬).
func (pstrApp *App) rememberWindow() bool {
	var raw = pstrApp.store.Setting("rememberWindow")
	if len(raw) == 0 {
		return true
	}
	var b bool
	_ = json.Unmarshal(raw, &b)
	return b
}

// applyWindowOptions 는 시작 시 저장된 크기·최대화 상태를 옵션에 반영한다.
func (pstrApp *App) applyWindowOptions(o *options.App) {
	o.Width, o.Height = defaultW, defaultH
	if !pstrApp.rememberWindow() {
		return
	}
	if w, ok := pstrApp.loadWin(); ok {
		o.Width, o.Height = w.W, w.H
		if w.Maximised {
			o.WindowStartState = options.Maximised
		}
	}
}

// restoreWindowPosition 은 DOM 이 준비된 뒤 위치를 복원한다. 저장 위치가 모든 화면 밖이면 주 화면 가운데로 둔다.
func (pstrApp *App) restoreWindowPosition(ctx context.Context) {
	if !pstrApp.rememberWindow() {
		wr.WindowCenter(ctx)
		return
	}
	w, ok := pstrApp.loadWin()
	if !ok || !w.HasPos || w.Maximised {
		if !ok {
			wr.WindowCenter(ctx)
		}
		return
	}
	screens, err := wr.ScreenGetAll(ctx)
	bVisible := err != nil
	for _, s := range screens {
		// 제목 표시줄 일부(왼쪽 위 80x30)가 어느 화면에든 걸쳐 있으면 보이는 것으로 본다
		if w.X+80 > 0 && w.X < s.Size.Width+0 && w.Y >= -10 && w.Y < s.Size.Height {
			bVisible = true
		}
	}
	if !bVisible {
		wr.WindowCenter(ctx)
		return
	}
	wr.WindowSetPosition(ctx, w.X, w.Y)
}

func (pstrApp *App) saveWindow(ctx context.Context) {
	if !pstrApp.rememberWindow() {
		return
	}
	var w winState
	w.Maximised = wr.WindowIsMaximised(ctx)
	if prev, ok := pstrApp.loadWin(); ok && w.Maximised {
		w = prev // 최대화 상태에서는 일반 상태의 크기·위치를 보존한다
		w.Maximised = true
	} else {
		w.W, w.H = wr.WindowGetSize(ctx)
		w.X, w.Y = wr.WindowGetPosition(ctx)
		w.HasPos = true
	}
	_ = store.WriteJSONAtomic(pstrApp.winFile(), w)
}

// beforeClose 는 연결된 세션이 있으면 종료 확인을 받고, 창 상태를 저장한다. true 를 돌려주면 종료를 막는다.
func (pstrApp *App) beforeClose(ctx context.Context) bool {
	if n := pstrApp.sessions.ConnectedCount(); n > 0 {
		r, err := wr.MessageDialog(ctx, wr.MessageDialogOptions{
			Type: wr.QuestionDialog, Title: "TermGo", Buttons: []string{"종료", "취소"}, DefaultButton: "취소", CancelButton: "취소",
			Message: "연결된 세션 " + itoa(n) + "개가 있습니다. 모두 닫고 종료할까요?" + pstrApp.keepNote(),
		})
		if err == nil && r != "종료" && r != "Yes" {
			return true
		}
	}
	pstrApp.saveWindow(ctx)
	return false
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// resetWindow 는 "원래 크기로": 기본 크기로 되돌리고 가운데로 옮긴다.
func (pstrApp *App) resetWindow() {
	if pstrApp.ctx == nil {
		return
	}
	wr.WindowUnmaximise(pstrApp.ctx)
	wr.WindowSetSize(pstrApp.ctx, defaultW, defaultH)
	wr.WindowCenter(pstrApp.ctx)
	_ = store.WriteJSONAtomic(pstrApp.winFile(), winState{W: defaultW, H: defaultH})
}

// keepNote 는 세션 유지(tmux) 창이 있을 때 종료 확인에 덧붙이는 안내.
func (pstrApp *App) keepNote() string {
	if pstrApp.sessions.KeepCount() == 0 {
		return ""
	}
	return "\n\n세션 유지(tmux) 중인 창 " + itoa(pstrApp.sessions.KeepCount()) + "개는 앱을 닫아도 안의 프로그램이 계속 실행되고, 다시 열면 그 화면에 그대로 붙습니다. 완전히 끝내려면 그 창에서 exit 를 입력하세요."
}
