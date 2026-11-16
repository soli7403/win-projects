// Package logclean 은 세션 로그 폴더에서 보관 기간이 지난 로그 파일을 주기적으로 지운다
// (spec/behavior/session.md §6 "보관 기간", 설정 sessionLogKeepDays).
package logclean

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// G_cdurCleanFirstDelay 는 앱 시작 뒤 첫 정리까지의 대기다 (시작 직후 디스크 부담을 피한다).
	G_cdurCleanFirstDelay = 10 * time.Second
	// G_cdurCleanInterval 은 정리 주기다.
	G_cdurCleanInterval = time.Hour
	// G_ciKeepDaysMax 는 보관 일수 상한이다 (설정 화면 범위와 같다).
	G_ciKeepDaysMax = 36500
	// g_cszLogExt 는 정리 대상 확장자다. 로그 폴더의 다른 파일은 건드리지 않는다.
	g_cszLogExt = ".log"
)

// Result 는 정리 한 번의 결과다.
type Result struct {
	M_iRemoved int      // 지운 파일 수
	M_iFailed  int      // 지우려 했으나 실패한 파일 수 (다음 주기에 다시 본다)
	M_slErrors []string // 실패 사유 (진단용, 최대 몇 개만)
}

// g_ciMaxErrors 는 Result 에 남기는 실패 사유의 최대 개수다.
const g_ciMaxErrors = 8

// CleanOnce 는 szDir 바로 아래의 .log 파일 중 수정 시각이 iKeepDays 일보다 오래된 것을 지운다.
// iKeepDays 가 0 이하이면 아무것도 지우지 않는다. 폴더가 없으면 빈 결과다.
// fnInUse 가 true 를 돌려주는 경로(지금 기록 중인 로그)는 건너뛴다. 하위 폴더는 보지 않는다.
func CleanOnce(szDir string, iKeepDays int, tNow time.Time, fnInUse func(string) bool) (Result, error) {
	var result Result
	if iKeepDays <= 0 {
		return result, nil
	}
	if iKeepDays > G_ciKeepDaysMax {
		iKeepDays = G_ciKeepDaysMax
	}
	slEntries, errRead := os.ReadDir(szDir)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return result, nil
		}
		return result, errRead
	}
	tCutoff := tNow.Add(-time.Duration(iKeepDays) * 24 * time.Hour)
	for _, entry := range slEntries {
		if !entry.Type().IsRegular() {
			continue // 폴더·링크 등은 대상이 아니다
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), g_cszLogExt) {
			continue
		}
		info, errInfo := entry.Info()
		if errInfo != nil {
			continue // 그사이 지워졌다
		}
		if !info.ModTime().Before(tCutoff) {
			continue
		}
		szPath := filepath.Join(szDir, entry.Name())
		if fnInUse != nil && fnInUse(szPath) {
			continue
		}
		if errRemove := os.Remove(szPath); errRemove != nil {
			if os.IsNotExist(errRemove) {
				continue
			}
			result.M_iFailed++
			if len(result.M_slErrors) < g_ciMaxErrors {
				result.M_slErrors = append(result.M_slErrors, errRemove.Error())
			}
			continue
		}
		result.M_iRemoved++
	}
	return result, nil
}

// Cleaner 는 정리를 시작 직후 한 번과 그 뒤 주기마다 백그라운드에서 돌린다.
type Cleaner struct {
	szDir       string
	fnKeepDays  func() int
	fnInUse     func(string) bool
	fnReport    func(Result, error) // 정리 결과 알림 (nil 이면 무시, 테스트·진단용)
	durFirst    time.Duration
	durInterval time.Duration
	chKick      chan struct{}
	chStop      chan struct{}
	onceStart   sync.Once
	onceStop    sync.Once
	wgLoop      sync.WaitGroup
}

// New 는 정리기를 만든다. fnKeepDays 는 정리할 때마다 불러 지금 설정의 보관 일수를 읽는다.
// fnInUse 는 지금 기록 중인 파일이면 true 를 돌려준다 (session.IsLogOpen).
func New(szDir string, fnKeepDays func() int, fnInUse func(string) bool) *Cleaner {
	return &Cleaner{
		szDir:       szDir,
		fnKeepDays:  fnKeepDays,
		fnInUse:     fnInUse,
		durFirst:    G_cdurCleanFirstDelay,
		durInterval: G_cdurCleanInterval,
		chKick:      make(chan struct{}, 1),
		chStop:      make(chan struct{}),
	}
}

// SetTiming 은 첫 정리 대기와 주기를 바꾼다. Start 전에만 부른다 (테스트용).
func (pCleaner *Cleaner) SetTiming(durFirst time.Duration, durInterval time.Duration) {
	pCleaner.durFirst = durFirst
	pCleaner.durInterval = durInterval
}

// SetReport 는 정리 결과 알림 함수를 둔다. Start 전에만 부른다.
func (pCleaner *Cleaner) SetReport(fnReport func(Result, error)) {
	pCleaner.fnReport = fnReport
}

// Start 는 백그라운드 정리 루프를 시작한다. 두 번 불러도 한 번만 시작한다.
func (pCleaner *Cleaner) Start() {
	pCleaner.onceStart.Do(func() {
		pCleaner.wgLoop.Add(1)
		go pCleaner.go_runLoop()
	})
}

// Kick 은 다음 주기를 기다리지 않고 곧 한 번 정리하게 한다 (설정 값이 바뀌었을 때). 막히지 않는다.
func (pCleaner *Cleaner) Kick() {
	select {
	case pCleaner.chKick <- struct{}{}:
	default: // 이미 예약돼 있다
	}
}

// Stop 은 루프를 멈추고 끝날 때까지 기다린다. 두 번 불러도 안전하고, Start 전에 불러도 된다.
func (pCleaner *Cleaner) Stop() {
	pCleaner.onceStop.Do(func() {
		close(pCleaner.chStop)
	})
	pCleaner.wgLoop.Wait()
}

// go_runLoop 는 첫 대기 → 정리 → 주기마다 정리를 반복한다. Kick 이 오면 바로 정리하고 주기를 다시 센다.
func (pCleaner *Cleaner) go_runLoop() {
	defer pCleaner.wgLoop.Done()
	timer := time.NewTimer(pCleaner.durFirst)
	defer timer.Stop()
	for {
		select {
		case <-pCleaner.chStop:
			return
		case <-timer.C:
		case <-pCleaner.chKick:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
		pCleaner.runOnce()
		timer.Reset(pCleaner.durInterval)
	}
}

// runOnce 는 지금 설정으로 정리를 한 번 한다.
func (pCleaner *Cleaner) runOnce() {
	iKeepDays := 0
	if pCleaner.fnKeepDays != nil {
		iKeepDays = pCleaner.fnKeepDays()
	}
	result, errClean := CleanOnce(pCleaner.szDir, iKeepDays, time.Now(), pCleaner.fnInUse)
	if pCleaner.fnReport != nil {
		pCleaner.fnReport(result, errClean)
	}
}
