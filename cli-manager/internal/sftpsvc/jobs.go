package sftpsvc

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"termgo/internal/vault"
)

// errCanceled 는 사용자가 복사 작업을 취소했을 때.
var errCanceled = errors.New("취소됨")

// CopyJob 은 CopyJob(shared/types.ts) — 끌어 놓기 한 번 = 작업 하나. copyjob:update 이벤트로 보낸다.
type CopyJob struct {
	ID         string `json:"id"`
	SrcSession string `json:"srcSession"`
	SrcTitle   string `json:"srcTitle"`
	SrcDir     string `json:"srcDir"`
	DstSession string `json:"dstSession"`
	DstTitle   string `json:"dstTitle"`
	DstDir     string `json:"dstDir"`
	// Name 은 첫 항목 이름, Count 는 끌어 온 항목 수.
	Name  string `json:"name"`
	Count int    `json:"count"`
	// Files / FilesDone 은 폴더 안까지 센 파일 수.
	Files     int   `json:"files"`
	FilesDone int   `json:"filesDone"`
	Bytes     int64 `json:"bytes"`
	Total     int64 `json:"total"`
	// Relay: 서버 → 서버 (이 PC 임시 폴더 경유). Total 은 받기+올리기 합.
	Relay bool `json:"relay"`
	// Phase: scan | download | upload | copy
	Phase string `json:"phase"`
	// State: queued | running | done | error | canceled
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	StartedAt int64  `json:"startedAt"`
	EndedAt   int64  `json:"endedAt,omitempty"`
}

type copyJob struct {
	svc      *Service
	mu       sync.Mutex
	j        CopyJob
	last     time.Time
	canceled atomic.Bool
}

func (pstrJ *copyJob) emitLocked() {
	pstrJ.last = time.Now()
	pstrJ.svc.UI.Send("copyjob:update", pstrJ.j)
}

// set 은 상태를 바꾸고 바로 알린다.
func (pstrJ *copyJob) set(fn func(j *CopyJob)) {
	pstrJ.mu.Lock()
	defer pstrJ.mu.Unlock()
	fn(&pstrJ.j)
	pstrJ.emitLocked()
}

// add 는 전송 바이트를 더하고 150ms 간격으로 알린다.
func (pstrJ *copyJob) add(n int) {
	pstrJ.mu.Lock()
	defer pstrJ.mu.Unlock()
	pstrJ.j.Bytes += int64(n)
	if time.Since(pstrJ.last) > 150*time.Millisecond {
		pstrJ.emitLocked()
	}
}

func (pstrJ *copyJob) fileDone() {
	pstrJ.mu.Lock()
	defer pstrJ.mu.Unlock()
	pstrJ.j.FilesDone++
	if time.Since(pstrJ.last) > 150*time.Millisecond {
		pstrJ.emitLocked()
	}
}

// newJob 은 작업을 등록하고 queued 로 알린다.
func (pstrV *Service) newJob(j CopyJob) *copyJob {
	j.ID, j.State, j.StartedAt = vault.NewUUID(), "queued", time.Now().UnixMilli()
	pstrJ := &copyJob{svc: pstrV, j: j}
	pstrV.mu.Lock()
	if pstrV.jobs == nil {
		pstrV.jobs = map[string]*copyJob{}
	}
	pstrV.jobs[j.ID] = pstrJ
	pstrV.mu.Unlock()
	pstrJ.set(func(*CopyJob) {})
	return pstrJ
}

func (pstrV *Service) endJob(pstrJ *copyJob, err error) {
	pstrJ.set(func(j *CopyJob) {
		j.EndedAt = time.Now().UnixMilli()
		switch {
		case errors.Is(err, errCanceled) || pstrJ.canceled.Load():
			j.State = "canceled"
		case err != nil:
			j.State, j.Error = "error", err.Error()
		default:
			j.State, j.Bytes, j.FilesDone = "done", j.Total, j.Files
		}
	})
	pstrV.mu.Lock()
	delete(pstrV.jobs, pstrJ.j.ID)
	pstrV.mu.Unlock()
}

// CancelJob 은 진행 중인 복사 작업을 취소한다 (지금 쓰던 파일은 중간까지 남을 수 있다).
func (pstrV *Service) CancelJob(szID string) {
	pstrV.mu.Lock()
	pstrJ := pstrV.jobs[szID]
	pstrV.mu.Unlock()
	if pstrJ != nil {
		pstrJ.canceled.Store(true)
	}
}

// scan 은 복사할 파일 수와 전체 크기를 센다.
func scan(fsys fileSys, szPath string, pstrJ *copyJob) (int, int64, error) {
	if pstrJ.canceled.Load() {
		return 0, 0, errCanceled
	}
	st, err := fsys.Stat(szPath)
	if err != nil {
		return 0, 0, err
	}
	if !st.IsDir() {
		return 1, st.Size(), nil
	}
	items, err := fsys.ReadDir(szPath)
	if err != nil {
		return 0, 0, err
	}
	var iFiles int
	var iBytes int64
	for _, it := range items {
		n, b, err := scan(fsys, fsys.Join(szPath, it.Name()), pstrJ)
		if err != nil {
			return 0, 0, err
		}
		iFiles += n
		iBytes += b
	}
	return iFiles, iBytes, nil
}
