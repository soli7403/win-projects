package sftpfs

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 편집 감시 상수 (protocols.md §3 로컬 앱으로 편집).
const (
	G_cdurEditPoll     = 700 * time.Millisecond
	G_cdurEditSettle   = 400 * time.Millisecond
	G_cdurEditMaxWatch = 12 * time.Hour
)

// EditWatch 는 로컬 편집 한 건의 감시 상태다.
type EditWatch struct {
	mtxData  sync.Mutex
	stop     chan struct{}
	szLocal  string
	szRemote string
	mtxLast  time.Time
}

// StartEdit 은 원격 파일을 임시 폴더(<임시>/termgo-edit/<무작위>/<파일명>)에 받고, 기본 프로그램으로 여는 함수를 부른 뒤
// 수정 시각을 700ms 간격으로 확인한다. 저장되면 0.4초 뒤 서버로 다시 올린다.
// fnOpen 은 로컬 파일을 연다(운영체제 기본 프로그램). fnToast 는 토스트를 보낸다 ({kind, text}).
// 세션이 사라지거나 12시간이 지나면 감시를 끝낸다. 받은 로컬 경로를 돌려준다.
func (pService *Service) StartEdit(szRemote string, fnOpen func(string) error, fnToast func(kind string, text string), fnSessionAlive func() bool) (string, error) {
	bRand := make([]byte, 8)
	_, _ = rand.Read(bRand)
	szDir := filepath.Join(os.TempDir(), "termgo-edit", fmt.Sprintf("%x", bRand))
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return "", errMkdir
	}
	szName := pService.be.Base(szRemote)
	szLocal := filepath.Join(szDir, szName)
	if errDown := pService.downloadFile(szRemote, szLocal, func(int64) {}); errDown != nil {
		return "", errDown
	}
	tStart, errMod := pService.Fs.modTimeOf(szRemote)
	if errMod != nil {
		return "", errMod
	}
	if errOpen := fnOpen(szLocal); errOpen != nil {
		return "", errOpen
	}
	watch := &EditWatch{stop: make(chan struct{}), szLocal: szLocal, szRemote: szRemote, mtxLast: tStart}
	go pService.watchEdit(watch, fnToast, fnSessionAlive)
	return szLocal, nil
}

// watchEdit 는 로컬 파일의 수정 시각을 확인해 바뀌면 올린다.
func (pService *Service) watchEdit(watch *EditWatch, fnToast func(string, string), fnSessionAlive func() bool) {
	tDeadline := time.Now().Add(G_cdurEditMaxWatch)
	tLocal, _ := os.Stat(watch.szLocal)
	var tSeen time.Time
	if tLocal != nil {
		tSeen = tLocal.ModTime()
	}
	ticker := time.NewTicker(G_cdurEditPoll)
	defer ticker.Stop()
	for {
		select {
		case <-watch.stop:
			return
		case <-ticker.C:
		}
		if time.Now().After(tDeadline) || (fnSessionAlive != nil && !fnSessionAlive()) {
			return
		}
		info, errStat := os.Stat(watch.szLocal)
		if errStat != nil || !info.ModTime().After(tSeen) {
			continue
		}
		tSeen = info.ModTime()
		time.Sleep(G_cdurEditSettle)
		errUp := pService.uploadFile(watch.szLocal, watch.szRemote, func(int64) {})
		if errUp != nil {
			fnToast("error", "업로드 실패: "+errUp.Error())
			continue
		}
		fnToast("ok", pService.be.Base(watch.szRemote)+" 저장됨 → 서버에 업로드 완료")
	}
}
