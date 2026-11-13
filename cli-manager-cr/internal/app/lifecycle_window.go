package app

import (
	"os"
	"path/filepath"

	"dariterm/internal/jsonfile"
)

// G_ciWindowMinW 와 G_ciWindowMinH 는 저장된 창 크기가 유효한 최소 값이다 (data/sync-backup.md §4).
const (
	G_ciWindowMinW = 400
	G_ciWindowMinH = 300
	// G_ciWindowDefaultW 와 G_ciWindowDefaultH 는 기본 창 크기다 (lifecycle.md §4).
	G_ciWindowDefaultW = 1440
	G_ciWindowDefaultH = 900
	// G_ciWindowProbeW 와 G_ciWindowProbeH 는 위치가 화면에 걸치는지 볼 영역 크기다 (왼쪽 위 약 80×30).
	G_ciWindowProbeW = 80
	G_ciWindowProbeH = 30
)

// WindowState 는 window.json 의 구조다 (data/sync-backup.md §4).
type WindowState struct {
	M_iX         int  `json:"x"`
	M_iY         int  `json:"y"`
	M_iW         int  `json:"w"`
	M_iH         int  `json:"h"`
	M_bMaximised bool `json:"maximised"`
	M_bHasPos    bool `json:"hasPos"`
}

// Valid 는 크기가 저장할 만한지 본다 (w ≥ 400, h ≥ 300).
func (ws WindowState) Valid() bool {
	return ws.M_iW >= G_ciWindowMinW && ws.M_iH >= G_ciWindowMinH
}

// WindowFilePath 는 window.json 경로다.
func WindowFilePath(szDataDir string) string {
	return filepath.Join(szDataDir, "window.json")
}

// LoadWindowState 는 window.json 을 읽는다. 없거나 깨졌으면 bOk=false.
func LoadWindowState(szDataDir string) (WindowState, bool) {
	var ws WindowState
	szPath := WindowFilePath(szDataDir)
	if !jsonfile.Exists(szPath) {
		return ws, false
	}
	if errRead := jsonfile.Read(szPath, &ws); errRead != nil {
		return ws, false
	}
	return ws, true
}

// SaveWindowState 는 window.json 을 원자적으로 쓴다.
func SaveWindowState(szDataDir string, ws WindowState) error {
	if errMk := os.MkdirAll(szDataDir, 0o700); errMk != nil {
		return errMk
	}
	return jsonfile.Write(WindowFilePath(szDataDir), ws)
}

// StartupPlacement 은 시작 때 창 위치를 정한다 (lifecycle.md §4).
// 저장된 위치가 주 화면(0,0 ~ 화면 크기) 안의 기준 영역과 겹치지 않으면 가운데(bCenter=true) 로 연다.
func StartupPlacement(ws WindowState, iScreenW int, iScreenH int) (iX int, iY int, bCenter bool) {
	if !ws.M_bHasPos {
		return 0, 0, true
	}
	bOverlap := ws.M_iX < iScreenW && ws.M_iX+G_ciWindowProbeW > 0 && ws.M_iY < iScreenH && ws.M_iY+G_ciWindowProbeH > 0
	if !bOverlap {
		return 0, 0, true
	}
	return ws.M_iX, ws.M_iY, false
}

// CloseState 는 창을 닫을 때 저장할 상태다. 최대화로 닫으면 이전의 일반 크기·위치를 보존하고 최대화 표시만 켠다.
func CloseState(prev WindowState, bHavePrev bool, iX int, iY int, iW int, iH int, bMaximised bool) WindowState {
	if bMaximised && bHavePrev && prev.Valid() {
		prev.M_bMaximised = true
		return prev
	}
	if bMaximised {
		return WindowState{M_iW: G_ciWindowDefaultW, M_iH: G_ciWindowDefaultH, M_bMaximised: true}
	}
	return WindowState{M_iX: iX, M_iY: iY, M_iW: iW, M_iH: iH, M_bHasPos: true}
}
