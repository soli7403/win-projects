package sftpfs

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
)

// G_cdurProgressInterval 은 진행 이벤트 최소 간격이다 (protocols.md §3 업로드).
const G_cdurProgressInterval = 150 * time.Millisecond

// EmitFunc 은 이벤트 발행 함수다.
type EmitFunc func(szEvent string, anyData ...any)

// Transfer 는 전송 항목 하나다 (spec/ipc.md §4 transfer.update).
type Transfer struct {
	M_szId         string `json:"id"`
	M_szSessionId  string `json:"sessionId"`
	M_szName       string `json:"name"`
	M_szDirection  string `json:"direction"` // upload | download
	M_i64Bytes     int64  `json:"bytes"`
	M_i64Total     int64  `json:"total"`
	M_szState      string `json:"state"` // queued | running | done | error | canceled
	M_szError      string `json:"error,omitempty"`
	M_szRemotePath string `json:"remotePath"`
	M_szLocalPath  string `json:"localPath,omitempty"`
}

// job 은 큐에 들어가는 전송 한 건이다.
type job struct {
	transfer *Transfer
	run      func(report func(bytes int64)) error
	done     chan error
}

// Queue 는 세션 하나의 전송 큐다. 한 번에 하나씩 순서대로 처리한다 (protocols.md §3).
type Queue struct {
	mtxData   sync.Mutex
	emit      EmitFunc
	slPending []*job
	bRunning  bool
	szSession string
}

// NewQueue 는 세션용 전송 큐를 만든다.
func NewQueue(szSessionId string, emit EmitFunc) *Queue {
	return &Queue{emit: emit, szSession: szSessionId}
}

// enqueue 는 작업을 넣고 완료 채널을 돌려준다.
func (pQueue *Queue) enqueue(pTransfer *Transfer, fnRun func(report func(int64)) error) chan error {
	pJob := &job{transfer: pTransfer, run: fnRun, done: make(chan error, 1)}
	// 큐에 들어간 전송은 대기 중(queued)이다 (spec/ui/sftp-snippets.md).
	if pTransfer.M_szState == "" {
		pTransfer.M_szState = "queued"
	}
	pQueue.emitTransfer(pTransfer)
	pQueue.mtxData.Lock()
	pQueue.slPending = append(pQueue.slPending, pJob)
	if !pQueue.bRunning {
		pQueue.bRunning = true
		go pQueue.work()
	}
	pQueue.mtxData.Unlock()
	return pJob.done
}

func (pQueue *Queue) work() {
	for {
		pQueue.mtxData.Lock()
		if len(pQueue.slPending) == 0 {
			pQueue.bRunning = false
			pQueue.mtxData.Unlock()
			return
		}
		pJob := pQueue.slPending[0]
		pQueue.slPending = pQueue.slPending[1:]
		pQueue.mtxData.Unlock()
		pQueue.runJob(pJob)
	}
}

// runJob 은 한 작업을 실행하며 진행 이벤트를 150ms 간격으로 보낸다.
func (pQueue *Queue) runJob(pJob *job) {
	pTransfer := pJob.transfer
	pTransfer.M_szState = "running"
	pQueue.emitTransfer(pTransfer)
	tLast := time.Now()
	errRun := pJob.run(func(iBytes int64) {
		pTransfer.M_i64Bytes = iBytes
		if time.Since(tLast) >= G_cdurProgressInterval {
			tLast = time.Now()
			pQueue.emitTransfer(pTransfer)
		}
	})
	if errRun != nil {
		pTransfer.M_szState = "error"
		pTransfer.M_szError = errRun.Error()
	} else {
		pTransfer.M_szState = "done"
		if pTransfer.M_i64Total > 0 {
			pTransfer.M_i64Bytes = pTransfer.M_i64Total
		}
	}
	pQueue.emitTransfer(pTransfer)
	pJob.done <- errRun
}

func (pQueue *Queue) emitTransfer(pTransfer *Transfer) {
	if pQueue.emit == nil {
		return
	}
	snapshot := *pTransfer
	pQueue.emit("transfer.update", snapshot)
}

// newTransferId 는 전송 id(16진 32자)를 만든다.
func newTransferId() string {
	bRand := make([]byte, 16)
	_, _ = rand.Read(bRand)
	return fmt.Sprintf("%x", bRand)
}

// Service 는 한 세션의 파일 패널 기능 묶음이다 (SSH 는 SFTP, 로컬 터미널은 이 PC). 클라이언트는 연결이 바뀌면 다시 받는다.
type Service struct {
	Fs    *Fs
	Queue *Queue
	be    Backend
	// szFsKey 는 같은 파일시스템인지 비교하는 값이다 (로컬은 "local", SSH 는 "ssh:" + 접속 대상).
	szFsKey string
	// mtxCopy 는 이 세션이 걸린 끌어 놓기 복사 작업을 하나씩 돌린다 (file-panel.md §2.5).
	mtxCopy sync.Mutex
}

// NewService 는 sftp 클라이언트와 세션 큐로 기능 묶음을 만든다. szTarget 은 접속 대상(사용자@호스트:포트)이다.
func NewService(szSessionId string, client *sftp.Client, emit EmitFunc, szTarget string) *Service {
	pFs := NewFs(client)
	return &Service{Fs: pFs, Queue: NewQueue(szSessionId, emit), be: pFs.be, szFsKey: "ssh:" + szTarget}
}

// NewLocalService 는 로컬 터미널 세션용 기능 묶음이다 (이 PC 파일시스템). szStart 는 시작 폴더.
func NewLocalService(szSessionId string, szStart string, emit EmitFunc) *Service {
	pFs := NewLocalFs(szStart)
	return &Service{Fs: pFs, Queue: NewQueue(szSessionId, emit), be: pFs.be, szFsKey: "local"}
}

// SessionId 는 이 묶음의 세션 id 다.
func (pService *Service) SessionId() string { return pService.Queue.szSession }

// UploadPaths 는 로컬 파일·폴더를 원격 폴더 아래로 올린다. 한 파일이라도 실패하면 거기서 오류로 끝낸다.
// 반환은 올린 원격 경로 배열이다.
func (pService *Service) UploadPaths(slLocal []string, szRemoteDir string) ([]string, error) {
	var slOut []string
	for _, szLocal := range slLocal {
		szRemote, errPlan := planTarget(LocalBackend{}, "local", szLocal, pService.be, pService.szFsKey, szRemoteDir)
		if errPlan != nil {
			return slOut, errPlan
		}
		if errBuild := pService.uploadOne(szLocal, szRemote, &slOut); errBuild != nil {
			return slOut, errBuild
		}
	}
	return slOut, nil
}

// uploadOne 은 파일이면 작업 하나를, 폴더면 원격 폴더를 만들고 안을 재귀로 올린다.
func (pService *Service) uploadOne(szLocal string, szRemote string, pslOut *[]string) error {
	info, errStat := os.Stat(szLocal)
	if errStat != nil {
		return errStat
	}
	if info.IsDir() {
		if errMkdir := pService.be.MkdirAll(szRemote); errMkdir != nil {
			return errMkdir
		}
		entries, errRead := os.ReadDir(szLocal)
		if errRead != nil {
			return errRead
		}
		for _, entry := range entries {
			if errChild := pService.uploadOne(filepath.Join(szLocal, entry.Name()), pService.be.Join(szRemote, entry.Name()), pslOut); errChild != nil {
				return errChild
			}
		}
		return nil
	}
	pTransfer := &Transfer{M_szId: newTransferId(), M_szSessionId: pService.Queue.szSession, M_szName: info.Name(),
		M_szDirection: "upload", M_i64Total: info.Size(), M_szRemotePath: szRemote, M_szLocalPath: szLocal}
	errRun := <-pService.Queue.enqueue(pTransfer, func(report func(int64)) error {
		return pService.uploadFile(szLocal, szRemote, report)
	})
	if errRun != nil {
		return errRun
	}
	*pslOut = append(*pslOut, szRemote)
	return nil
}

// uploadFile 은 로컬 파일 하나를 원격에 쓴다.
func (pService *Service) uploadFile(szLocal string, szRemote string, report func(int64)) error {
	src, errOpen := os.Open(szLocal)
	if errOpen != nil {
		return errOpen
	}
	defer src.Close()
	dst, errCreate := pService.be.OpenWrite(szRemote)
	if errCreate != nil {
		return errCreate
	}
	_, errCopy := io.Copy(dst, &progressReader{r: src, report: report})
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy
}

// DownloadPaths 는 원격 파일·폴더를 로컬 폴더 아래로 받는다. 개별 파일 실패는 그 항목에만 표시하고 계속한다.
// 반환은 저장한 로컬 폴더 경로다.
func (pService *Service) DownloadPaths(slRemote []string, szLocalDir string) (string, error) {
	for _, szRemote := range slRemote {
		szLocal, errPlan := planTarget(pService.be, pService.szFsKey, szRemote, LocalBackend{}, "local", szLocalDir)
		if errPlan != nil {
			return szLocalDir, errPlan
		}
		if errBuild := pService.downloadOne(szRemote, szLocal); errBuild != nil {
			return szLocalDir, errBuild
		}
	}
	return szLocalDir, nil
}

// downloadOne 은 원격 파일이면 작업 하나를, 폴더면 로컬 폴더를 만들고 재귀로 받는다.
func (pService *Service) downloadOne(szRemote string, szLocal string) error {
	info, errStat := pService.be.Stat(szRemote)
	if errStat != nil {
		return errStat
	}
	if info.IsDir() {
		if errMkdir := os.MkdirAll(szLocal, 0o700); errMkdir != nil {
			return errMkdir
		}
		infos, errRead := pService.be.ReadDir(szRemote)
		if errRead != nil {
			return errRead
		}
		for _, child := range infos {
			_ = pService.downloadOne(pService.be.Join(szRemote, child.Name()), filepath.Join(szLocal, child.Name()))
		}
		return nil
	}
	pTransfer := &Transfer{M_szId: newTransferId(), M_szSessionId: pService.Queue.szSession, M_szName: info.Name(),
		M_szDirection: "download", M_i64Total: info.Size(), M_szRemotePath: szRemote, M_szLocalPath: szLocal}
	// 다운로드는 개별 실패를 항목에만 남기고 계속한다.
	<-pService.Queue.enqueue(pTransfer, func(report func(int64)) error {
		return pService.downloadFile(szRemote, szLocal, report)
	})
	return nil
}

// downloadFile 은 원격 파일 하나를 로컬에 쓴다.
func (pService *Service) downloadFile(szRemote string, szLocal string, report func(int64)) error {
	src, errOpen := pService.be.OpenRead(szRemote)
	if errOpen != nil {
		return errOpen
	}
	defer src.Close()
	if errMkdir := os.MkdirAll(filepath.Dir(szLocal), 0o700); errMkdir != nil {
		return errMkdir
	}
	dst, errCreate := os.OpenFile(szLocal, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if errCreate != nil {
		return errCreate
	}
	_, errCopy := io.Copy(dst, &progressReader{r: src, report: report})
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy
}

// progressReader 는 읽은 바이트를 누적해 알린다.
type progressReader struct {
	r      io.Reader
	n      int64
	report func(int64)
}

func (pReader *progressReader) Read(bBuf []byte) (int, error) {
	nRead, errRead := pReader.r.Read(bBuf)
	pReader.n += int64(nRead)
	if nRead > 0 && pReader.report != nil {
		pReader.report(pReader.n)
	}
	return nRead, errRead
}
