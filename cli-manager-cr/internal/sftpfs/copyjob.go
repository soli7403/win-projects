package sftpfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 끌어 놓기 복사 작업 (spec/behavior/file-panel.md §2~§3, §5).

// G_cszCopyEvent 는 복사 작업 갱신 이벤트 이름이다.
const G_cszCopyEvent = "copyjob.update"

// ErrCopyCanceled 는 사용자가 작업을 취소했을 때의 오류다.
var ErrCopyCanceled = errors.New("취소됨")

// CopyJob 은 복사 작업 갱신 이벤트의 값이다. 끌어 놓기 한 번이 작업 하나다.
type CopyJob struct {
	M_szId         string `json:"id"`
	M_szSrcSession string `json:"srcSession"`
	M_szSrcTitle   string `json:"srcTitle"`
	M_szSrcDir     string `json:"srcDir"`
	M_szDstSession string `json:"dstSession"`
	M_szDstTitle   string `json:"dstTitle"`
	M_szDstDir     string `json:"dstDir"`
	M_szName       string `json:"name"`
	M_iCount       int    `json:"count"`
	M_iFiles       int    `json:"files"`
	M_iFilesDone   int    `json:"filesDone"`
	M_i64Bytes     int64  `json:"bytes"`
	M_i64Total     int64  `json:"total"`
	M_bRelay       bool   `json:"relay"`
	M_szPhase      string `json:"phase"` // scan | download | upload | copy
	M_szState      string `json:"state"` // queued | running | done | error | canceled
	M_szError      string `json:"error,omitempty"`
	M_i64StartedAt int64  `json:"startedAt"`
	M_i64EndedAt   int64  `json:"endedAt,omitempty"`
}

// CopyRequest 는 복사 시작 요청이다.
type CopyRequest struct {
	Src      *Service
	SrcTitle string
	Paths    []string
	Dst      *Service
	DstTitle string
	// DstDir 이 비면 대상 세션의 시작 폴더.
	DstDir    string
	Overwrite bool
	// TempDir 은 서버 → 서버 복사가 거쳐 가는 이 PC 폴더다 (비면 OS 임시 폴더 아래 termgo-transfer).
	TempDir string
	Emit    EmitFunc
}

// CopyStart 는 복사 시작 응답이다. 충돌이 있고 덮어쓰기가 아니면 JobId 는 비어 있고 아무것도 하지 않았다.
type CopyStart struct {
	M_szJobId     string   `json:"jobId"`
	M_slConflicts []string `json:"conflicts"`
}

type copyRun struct {
	mtxData   sync.Mutex
	job       CopyJob
	emit      EmitFunc
	tLast     time.Time
	bCanceled atomic.Bool
}

var (
	g_mtxCopyRuns sync.Mutex
	g_mapCopyRuns = map[string]*copyRun{}
)

// StartCopy 는 검사(대상 폴더·충돌·자기 안 복사)를 마치고 작업을 백그라운드로 시작한다.
func StartCopy(req CopyRequest) (CopyStart, error) {
	result := CopyStart{M_slConflicts: []string{}}
	if len(req.Paths) == 0 {
		return result, errors.New("복사할 항목이 없습니다")
	}
	srcBe, dstBe := req.Src.be, req.Dst.be
	szDir, errDir := req.Dst.Fs.Normalize(req.DstDir)
	if errDir != nil {
		return result, errDir
	}
	if info, errStat := dstBe.Stat(szDir); errStat != nil || !info.IsDir() {
		return result, fmt.Errorf("대상 폴더가 없습니다: %s", szDir)
	}
	slTargets := make([]string, len(req.Paths))
	for iIdx, szSrc := range req.Paths {
		szTarget, errPlan := planTarget(srcBe, req.Src.szFsKey, szSrc, dstBe, req.Dst.szFsKey, szDir)
		if errPlan != nil {
			return result, errPlan
		}
		slTargets[iIdx] = szTarget
		if _, errExist := dstBe.Lstat(szTarget); errExist == nil {
			result.M_slConflicts = append(result.M_slConflicts, dstBe.Base(szTarget))
		}
	}
	if len(result.M_slConflicts) > 0 && !req.Overwrite {
		return result, nil
	}
	result.M_slConflicts = []string{}
	pRun := &copyRun{emit: req.Emit, job: CopyJob{
		M_szId: newTransferId(), M_szSrcSession: req.Src.SessionId(), M_szSrcTitle: req.SrcTitle, M_szSrcDir: srcBe.Dir(req.Paths[0]),
		M_szDstSession: req.Dst.SessionId(), M_szDstTitle: req.DstTitle, M_szDstDir: szDir,
		M_szName: srcBe.Base(req.Paths[0]), M_iCount: len(req.Paths), M_bRelay: !srcBe.IsLocal() && !dstBe.IsLocal(),
		M_szPhase: "scan", M_szState: "queued", M_i64StartedAt: time.Now().UnixMilli(),
	}}
	g_mtxCopyRuns.Lock()
	g_mapCopyRuns[pRun.job.M_szId] = pRun
	g_mtxCopyRuns.Unlock()
	pRun.update(func(*CopyJob) {})
	result.M_szJobId = pRun.job.M_szId
	// run 을 인자로 바로 넘기면 호출한 고루틴에서 먼저 실행되므로 감싸서 백그라운드로 돌린다.
	go func() { pRun.finish(pRun.run(req, slTargets)) }()
	return result, nil
}

// CancelCopy 는 진행 중인 작업을 취소한다. 쓰던 파일은 중간까지 남을 수 있다.
func CancelCopy(szJobId string) {
	g_mtxCopyRuns.Lock()
	pRun := g_mapCopyRuns[szJobId]
	g_mtxCopyRuns.Unlock()
	if pRun != nil {
		pRun.bCanceled.Store(true)
	}
}

// update 는 작업 값을 바꾸고 바로 알린다.
func (pRun *copyRun) update(fnChange func(pJob *CopyJob)) {
	pRun.mtxData.Lock()
	defer pRun.mtxData.Unlock()
	fnChange(&pRun.job)
	pRun.emitLocked()
}

func (pRun *copyRun) emitLocked() {
	pRun.tLast = time.Now()
	if pRun.emit != nil {
		pRun.emit(G_cszCopyEvent, pRun.job)
	}
}

// progress 는 바이트·완료 파일 수를 더하고 150ms 간격으로 알린다.
func (pRun *copyRun) progress(i64Bytes int64, iFiles int) {
	pRun.mtxData.Lock()
	defer pRun.mtxData.Unlock()
	pRun.job.M_i64Bytes += i64Bytes
	pRun.job.M_iFilesDone += iFiles
	if time.Since(pRun.tLast) >= G_cdurProgressInterval {
		pRun.emitLocked()
	}
}

func (pRun *copyRun) finish(errRun error) {
	pRun.update(func(pJob *CopyJob) {
		pJob.M_i64EndedAt = time.Now().UnixMilli()
		switch {
		case errors.Is(errRun, ErrCopyCanceled) || pRun.bCanceled.Load():
			pJob.M_szState = "canceled"
		case errRun != nil:
			pJob.M_szState, pJob.M_szError = "error", errRun.Error()
		default:
			pJob.M_szState, pJob.M_i64Bytes, pJob.M_iFilesDone = "done", pJob.M_i64Total, pJob.M_iFiles
		}
	})
	g_mtxCopyRuns.Lock()
	delete(g_mapCopyRuns, pRun.job.M_szId)
	g_mtxCopyRuns.Unlock()
}

// lockPair 는 두 세션의 복사 잠금을 세션 id 순서로 잡는다 (교착 방지). 반환 함수로 푼다.
func lockPair(pA *Service, pB *Service) func() {
	if pA == pB {
		pA.mtxCopy.Lock()
		return pA.mtxCopy.Unlock
	}
	if pB.SessionId() < pA.SessionId() {
		pA, pB = pB, pA
	}
	pA.mtxCopy.Lock()
	pB.mtxCopy.Lock()
	return func() {
		pB.mtxCopy.Unlock()
		pA.mtxCopy.Unlock()
	}
}

// run 은 작업 본체다: 차례 기다리기 → 크기 세기 → 복사(서버 ↔ 서버는 임시 폴더 경유).
func (pRun *copyRun) run(req CopyRequest, slTargets []string) error {
	defer lockPair(req.Src, req.Dst)()
	if pRun.bCanceled.Load() {
		return ErrCopyCanceled
	}
	pRun.update(func(pJob *CopyJob) { pJob.M_szState = "running" })
	iFiles, i64Bytes := 0, int64(0)
	for _, szSrc := range req.Paths {
		nFiles, nBytes, errScan := pRun.scan(req.Src.be, szSrc)
		if errScan != nil {
			return errScan
		}
		iFiles += nFiles
		i64Bytes += nBytes
	}
	bRelay := pRun.job.M_bRelay
	pRun.update(func(pJob *CopyJob) {
		pJob.M_iFiles, pJob.M_i64Total = iFiles, i64Bytes
		if bRelay {
			pJob.M_i64Total = i64Bytes * 2 // 받기 + 올리기
		}
	})
	for iIdx, szSrc := range req.Paths {
		var errCopy error
		if bRelay {
			errCopy = pRun.relay(req, szSrc, slTargets[iIdx])
		} else {
			pRun.update(func(pJob *CopyJob) { pJob.M_szPhase = "copy" })
			errCopy = pRun.copyTree(req.Src.be, szSrc, req.Dst.be, slTargets[iIdx], true)
		}
		if errCopy != nil {
			return errCopy
		}
	}
	return nil
}

// relay 는 서버 → 서버 복사다: 원본 서버에서 임시 폴더로 받고, 대상 서버로 올리고, 임시 사본을 지운다.
func (pRun *copyRun) relay(req CopyRequest, szSrc string, szTarget string) error {
	szBase := req.TempDir
	if szBase == "" {
		szBase = filepath.Join(os.TempDir(), "termgo-transfer")
	}
	if errMk := os.MkdirAll(szBase, 0o700); errMk != nil {
		return fmt.Errorf("임시 폴더를 만들 수 없습니다 (%s): %w", szBase, errMk)
	}
	szStage, errTemp := os.MkdirTemp(szBase, "copy-")
	if errTemp != nil {
		return fmt.Errorf("임시 폴더를 만들 수 없습니다 (%s): %w", szBase, errTemp)
	}
	defer os.RemoveAll(szStage)
	szLocal := filepath.Join(szStage, req.Src.be.Base(szSrc))
	pRun.update(func(pJob *CopyJob) { pJob.M_szPhase = "download" })
	if errDown := pRun.copyTree(req.Src.be, szSrc, LocalBackend{}, szLocal, false); errDown != nil {
		return errDown
	}
	pRun.update(func(pJob *CopyJob) { pJob.M_szPhase = "upload" })
	return pRun.copyTree(LocalBackend{}, szLocal, req.Dst.be, szTarget, true)
}

// scan 은 파일 수와 전체 크기를 센다 (링크는 따라간다).
func (pRun *copyRun) scan(be Backend, szPath string) (int, int64, error) {
	if pRun.bCanceled.Load() {
		return 0, 0, ErrCopyCanceled
	}
	info, errStat := be.Stat(szPath)
	if errStat != nil {
		return 0, 0, errStat
	}
	if !info.IsDir() {
		return 1, info.Size(), nil
	}
	infos, errRead := be.ReadDir(szPath)
	if errRead != nil {
		return 0, 0, errRead
	}
	iFiles, i64Bytes := 0, int64(0)
	for _, child := range infos {
		nFiles, nBytes, errChild := pRun.scan(be, be.Join(szPath, child.Name()))
		if errChild != nil {
			return 0, 0, errChild
		}
		iFiles += nFiles
		i64Bytes += nBytes
	}
	return iFiles, i64Bytes, nil
}

// copyTree 는 파일·폴더를 재귀로 복사한다. bCountFiles 면 끝난 파일 수를 올린다. 하나라도 실패하면 멈춘다.
func (pRun *copyRun) copyTree(srcBe Backend, szSrc string, dstBe Backend, szDst string, bCountFiles bool) error {
	if pRun.bCanceled.Load() {
		return ErrCopyCanceled
	}
	info, errStat := srcBe.Stat(szSrc)
	if errStat != nil {
		return errStat
	}
	if info.IsDir() {
		if errMk := dstBe.MkdirAll(szDst); errMk != nil {
			return errMk
		}
		infos, errRead := srcBe.ReadDir(szSrc)
		if errRead != nil {
			return errRead
		}
		for _, child := range infos {
			if errChild := pRun.copyTree(srcBe, srcBe.Join(szSrc, child.Name()), dstBe, dstBe.Join(szDst, child.Name()), bCountFiles); errChild != nil {
				return errChild
			}
		}
		return nil
	}
	if errFile := pRun.copyFile(srcBe, szSrc, dstBe, szDst); errFile != nil {
		return errFile
	}
	if bCountFiles {
		pRun.progress(0, 1)
	}
	return nil
}

// copyFile 은 파일 하나를 바이트 그대로 옮긴다 (줄바꿈·인코딩 변환 없음).
func (pRun *copyRun) copyFile(srcBe Backend, szSrc string, dstBe Backend, szDst string) error {
	src, errOpen := srcBe.OpenRead(szSrc)
	if errOpen != nil {
		return errOpen
	}
	defer src.Close()
	dst, errCreate := dstBe.OpenWrite(szDst)
	if errCreate != nil {
		return errCreate
	}
	_, errCopy := io.Copy(&copyWriter{w: dst, pRun: pRun}, src)
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy
}

// copyWriter 는 쓴 바이트를 작업 진행에 더하고, 취소되면 멈춘다.
type copyWriter struct {
	w    io.Writer
	pRun *copyRun
}

func (pWriter *copyWriter) Write(bData []byte) (int, error) {
	if pWriter.pRun.bCanceled.Load() {
		return 0, ErrCopyCanceled
	}
	nWritten, errWrite := pWriter.w.Write(bData)
	pWriter.pRun.progress(int64(nWritten), 0)
	return nWritten, errWrite
}

// planTarget 은 szSrc 를 dst 의 szDir 아래로 복사할 경로를 정한다 (file-panel.md §2.4).
// 같은 파일시스템에서 원래 자리면 "이름 (n)" 새 이름, 폴더를 자기 안으로 넣으면 오류.
func planTarget(srcBe Backend, szSrcKey string, szSrc string, dstBe Backend, szDstKey string, szDir string) (string, error) {
	szName := srcBe.Base(szSrc)
	szTarget := dstBe.Join(szDir, szName)
	if szSrcKey != szDstKey {
		return szTarget, nil
	}
	if samePath(dstBe, szTarget, szSrc) {
		return uniqueName(dstBe, szDir, szName), nil
	}
	if info, errStat := srcBe.Stat(szSrc); errStat == nil && info.IsDir() && isInside(dstBe, szSrc, szDir) {
		return "", errors.New("폴더를 자기 자신 안으로 복사할 수 없습니다: " + szName)
	}
	return szTarget, nil
}

// ignoresCase 는 경로 대소문자를 구분하지 않는 파일시스템인지다 (Windows·macOS 의 이 PC).
func ignoresCase(be Backend) bool { return be.IsLocal() && runtime.GOOS != "linux" }

func samePath(be Backend, szA string, szB string) bool {
	if ignoresCase(be) {
		return strings.EqualFold(szA, szB)
	}
	return szA == szB
}

// isInside 는 szChild 가 szParent 이거나 그 아래인지다.
func isInside(be Backend, szParent string, szChild string) bool {
	if samePath(be, szParent, szChild) {
		return true
	}
	szPrefix := strings.TrimSuffix(szParent, be.Separator()) + be.Separator()
	if ignoresCase(be) {
		return strings.HasPrefix(strings.ToLower(szChild), strings.ToLower(szPrefix))
	}
	return strings.HasPrefix(szChild, szPrefix)
}

// uniqueName 은 szDir 에 없는 "이름 (n).확장자" 경로다 (폴더는 확장자를 나누지 않는다).
func uniqueName(be Backend, szDir string, szName string) string {
	szExt := path.Ext(szName)
	if info, errStat := be.Stat(be.Join(szDir, szName)); (errStat == nil && info.IsDir()) || szExt == szName {
		szExt = ""
	}
	szStem := strings.TrimSuffix(szName, szExt)
	for iNum := 1; ; iNum++ {
		szTry := be.Join(szDir, fmt.Sprintf("%s (%d)%s", szStem, iNum, szExt))
		if _, errStat := be.Lstat(szTry); errors.Is(errStat, fs.ErrNotExist) || iNum >= 9999 {
			return szTry
		}
	}
}
