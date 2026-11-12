package sftpsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"termgo/internal/transport"
	"termgo/internal/vault"
)

// Transfer 는 Transfer(shared/types.ts) — transfer:update 이벤트로 진행 상황을 보낸다.
type Transfer struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionId"`
	Name       string `json:"name"`
	Direction  string `json:"direction"` // up | down
	Bytes      int64  `json:"bytes"`
	Total      int64  `json:"total"`
	State      string `json:"state"` // queued | running | done | error | canceled
	Error      string `json:"error,omitempty"`
	RemotePath string `json:"remotePath"`
	LocalPath  string `json:"localPath,omitempty"`
	// job 이 있으면 파일별 항목은 보내지 않고 작업 진행률에 합친다.
	job *copyJob
}

func (pstrV *Service) update(t *Transfer) {
	if t.job == nil {
		pstrV.UI.Send("transfer:update", *t)
	}
}

// progress 는 쓰기/읽기 바이트를 세어 150ms 간격으로 알린다.
type progress struct {
	svc  *Service
	t    *Transfer
	last time.Time
}

func (pstrP *progress) add(n int) {
	pstrP.t.Bytes += int64(n)
	if pstrP.t.job != nil {
		pstrP.t.job.add(n)
		return
	}
	if time.Since(pstrP.last) > 150*time.Millisecond {
		pstrP.last = time.Now()
		pstrP.svc.update(pstrP.t)
	}
}

type countingWriter struct {
	w io.Writer
	p *progress
}

func (c countingWriter) Write(b []byte) (int, error) {
	if j := c.p.t.job; j != nil && j.canceled.Load() {
		return 0, errCanceled
	}
	n, err := c.w.Write(b)
	c.p.add(n)
	return n, err
}

// copyFile 은 파일 하나를 src → dst 로 복사한다. 항상 바이너리 그대로 옮긴다:
// 줄바꿈(CRLF/LF)·인코딩 변환을 하지 않는다 (SFTP 에는 텍스트 모드가 없고 os.Open/Create 도 바이트 그대로).
func (pstrV *Service) copyFile(src fileSys, szSrc string, dst fileSys, szDst string, t *Transfer) error {
	t.State = "running"
	pstrV.update(t)
	in, err := src.Open(szSrc)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := dst.Create(szDst)
	if err != nil {
		return err
	}
	_, err = io.Copy(countingWriter{out, &progress{svc: pstrV, t: t}}, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

func (pstrV *Service) putFile(cl *sftp.Client, t *Transfer, szLocal, szRemote string) error {
	return pstrV.copyFile(localFS{}, szLocal, remoteFS{cl: cl}, szRemote, t)
}

func (pstrV *Service) getFile(cl *sftp.Client, t *Transfer, szRemote, szLocal string) error {
	return pstrV.copyFile(remoteFS{cl: cl}, szRemote, localFS{}, szLocal, t)
}

func finish(pstrV *Service, t *Transfer, err error) error {
	if err != nil {
		t.State, t.Error = "error", err.Error()
	} else {
		t.State, t.Bytes = "done", t.Total
	}
	pstrV.update(t)
	return err
}

// xfer 는 전송 항목 표시 정보.
type xfer struct {
	szSession string
	// bDown: 세션 → 이 PC (아니면 세션 쪽으로 들어감)
	bDown bool
	// bKeepGoing: 파일 하나가 실패해도 전송 목록에 표시하고 계속
	bKeepGoing bool
	// job: 끌어 놓기 복사 작업 (있으면 파일별 항목 대신 작업 진행률)
	job *copyJob
	// bNoCount: job 의 완료 파일 수를 올리지 않음
	bNoCount bool
}

// copyTree 는 szSrc(파일/폴더)를 dst 의 szDst 경로로 복사한다.
func (pstrV *Service) copyTree(src fileSys, szSrc string, dst fileSys, szDst string, x xfer) error {
	st, err := src.Stat(szSrc)
	if err != nil {
		return err
	}
	if st.IsDir() {
		if err := dst.MkdirAll(szDst); err != nil {
			return err
		}
		items, err := src.ReadDir(szSrc)
		if err != nil {
			return err
		}
		for _, it := range items {
			if err := pstrV.copyTree(src, src.Join(szSrc, it.Name()), dst, dst.Join(szDst, it.Name()), x); err != nil {
				return err
			}
		}
		return nil
	}
	t := &Transfer{ID: vault.NewUUID(), SessionID: x.szSession, Name: src.Base(szSrc), Total: st.Size(), State: "queued", job: x.job}
	if x.bDown {
		t.Direction, t.RemotePath, t.LocalPath = "down", szSrc, szDst
	} else {
		t.Direction, t.RemotePath, t.LocalPath = "up", szDst, szSrc
	}
	pstrV.update(t)
	err = finish(pstrV, t, pstrV.copyFile(src, szSrc, dst, szDst, t))
	if x.job != nil && err == nil && !x.bNoCount {
		x.job.fileDone()
	}
	if x.bKeepGoing {
		return nil
	}
	return err
}

// lockQueues 는 여러 세션의 전송 잠금을 정해진 순서로 잡는다 (교착 방지). 반환 함수로 푼다.
func (pstrV *Service) lockQueues(slSessions ...string) func() {
	mSeen := map[string]bool{}
	var slKeys []string
	for _, s := range slSessions {
		if s != "" && !mSeen[s] {
			mSeen[s] = true
			slKeys = append(slKeys, s)
		}
	}
	sort.Strings(slKeys)
	qs := make([]interface{ Unlock() }, 0, len(slKeys))
	for _, k := range slKeys {
		q := pstrV.queue(k)
		q.Lock()
		qs = append(qs, q)
	}
	return func() {
		for i := len(qs) - 1; i >= 0; i-- {
			qs[i].Unlock()
		}
	}
}

func (pstrV *Service) uploadPath(cl *sftp.Client, szSession, szLocal, szRemoteDir string) (string, error) {
	dst := remoteFS{szKey: szSession, cl: cl}
	szRemote := dst.Join(szRemoteDir, filepath.Base(szLocal))
	return szRemote, pstrV.copyTree(localFS{}, szLocal, dst, szRemote, xfer{szSession: szSession})
}

// Upload 는 이 PC 의 파일/폴더를 세션 폴더로 복사한다 (세션별 직렬). 로컬 세션이면 PC 안 복사.
func (pstrV *Service) Upload(szSession string, slLocals []string, szDir string) ([]string, error) {
	defer pstrV.lockQueues(szSession)()
	dst, err := pstrV.fsFor(szSession)
	if err != nil {
		return nil, err
	}
	szDir, err = pstrV.expand(szSession, szDir)
	if err != nil {
		return nil, err
	}
	src := localFS{}
	out := []string{}
	for _, l := range slLocals {
		szTo, err := plan(src, l, dst, szDir)
		if err != nil {
			return out, err
		}
		if err := pstrV.copyTree(src, l, dst, szTo, xfer{szSession: szSession}); err != nil {
			return out, err
		}
		out = append(out, szTo)
	}
	return out, nil
}

// Download 는 세션의 파일/폴더를 이 PC 폴더로 복사한다 (세션별 직렬).
func (pstrV *Service) Download(szSession string, slPaths []string, szLocalDir string) error {
	defer pstrV.lockQueues(szSession)()
	src, err := pstrV.fsFor(szSession)
	if err != nil {
		return err
	}
	dst := localFS{}
	for _, r := range slPaths {
		szTo, err := plan(src, r, dst, szLocalDir)
		if err != nil {
			return err
		}
		if err := pstrV.copyTree(src, r, dst, szTo, xfer{szSession: szSession, bDown: true, bKeepGoing: true}); err != nil {
			return err
		}
	}
	return nil
}

// CopyResult 는 sftp:copy 결과. Conflicts 가 있으면 작업을 시작하지 않았다.
type CopyResult struct {
	JobID     string   `json:"jobId"`
	Conflicts []string `json:"conflicts"`
}

// Copy 는 한 세션의 파일/폴더를 다른(또는 같은) 세션 폴더로 복사하는 작업을 백그라운드로 시작한다.
// 진행 상황은 copyjob:update 이벤트로 보낸다.
// bOverwrite 가 false 이고 같은 이름이 이미 있으면 시작하지 않고 Conflicts 로 알린다.
// 원래 자리에 다시 복사하면 "이름 (1)" 로 만든다. 빈 szDstDir 은 대상 세션의 시작 폴더.
func (pstrV *Service) Copy(szSrcSession string, slPaths []string, szDstSession, szDstDir string, bOverwrite bool) (*CopyResult, error) {
	if len(slPaths) == 0 {
		return nil, errors.New("복사할 항목이 없습니다")
	}
	src, err := pstrV.fsFor(szSrcSession)
	if err != nil {
		return nil, err
	}
	dst, err := pstrV.fsFor(szDstSession)
	if err != nil {
		return nil, err
	}
	if szDstDir == "" {
		szDstDir, err = pstrV.start(szDstSession, dst)
	} else {
		szDstDir, err = pstrV.expand(szDstSession, szDstDir)
	}
	if err != nil {
		return nil, err
	}
	if st, err := dst.Stat(szDstDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("대상 폴더가 없습니다: %s", szDstDir)
	}
	res := &CopyResult{Conflicts: []string{}}
	slTo := make([]string, len(slPaths))
	for i, p := range slPaths {
		if slTo[i], err = plan(src, p, dst, szDstDir); err != nil {
			return nil, err
		}
		if _, err := dst.Lstat(slTo[i]); err == nil {
			res.Conflicts = append(res.Conflicts, dst.Base(slTo[i]))
		}
	}
	if len(res.Conflicts) > 0 && !bOverwrite {
		return res, nil
	}
	res.Conflicts = []string{}
	j := CopyJob{
		SrcSession: szSrcSession, SrcTitle: pstrV.title(szSrcSession), SrcDir: src.Dir(slPaths[0]),
		DstSession: szDstSession, DstTitle: pstrV.title(szDstSession), DstDir: szDstDir,
		Name: src.Base(slPaths[0]), Count: len(slPaths), Relay: !src.Local() && !dst.Local(), Phase: "scan",
	}
	pstrJ := pstrV.newJob(j)
	res.JobID = pstrJ.j.ID
	go func() {
		pstrV.endJob(pstrJ, pstrV.runCopy(pstrJ, src, slPaths, dst, slTo))
	}()
	return res, nil
}

func (pstrV *Service) title(szSession string) string {
	if s := pstrV.Sessions.Get(szSession); s != nil {
		return s.Info().Title
	}
	return ""
}

// runCopy 는 작업 본체: 세션 전송 잠금 → 크기 세기 → 복사 (서버 ↔ 서버는 임시 폴더 경유).
func (pstrV *Service) runCopy(pstrJ *copyJob, src fileSys, slPaths []string, dst fileSys, slTo []string) error {
	defer pstrV.lockQueues(pstrJ.j.SrcSession, pstrJ.j.DstSession)()
	pstrJ.set(func(j *CopyJob) { j.State = "running" })
	var iFiles int
	var iBytes int64
	for _, p := range slPaths {
		n, b, err := scan(src, p, pstrJ)
		if err != nil {
			return err
		}
		iFiles += n
		iBytes += b
	}
	bRelay := pstrJ.j.Relay
	pstrJ.set(func(j *CopyJob) {
		j.Files, j.Total = iFiles, iBytes
		if bRelay {
			j.Total = iBytes * 2
		}
	})
	for i, p := range slPaths {
		var err error
		if bRelay {
			err = pstrV.relay(pstrJ, src, p, dst, slTo[i])
		} else {
			pstrJ.set(func(j *CopyJob) { j.Phase = "copy" })
			err = pstrV.copyTree(src, p, dst, slTo[i], xfer{szSession: pstrJ.j.DstSession, job: pstrJ})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// plan 은 szSrc 를 dst 의 szDir 아래로 복사할 대상 경로를 정한다.
// 같은 파일시스템에서 원래 자리면 새 이름, 폴더를 자기 안으로 복사하면 오류.
func plan(src fileSys, szSrc string, dst fileSys, szDir string) (string, error) {
	szName := src.Base(szSrc)
	szTo := dst.Join(szDir, szName)
	if src.Key() != dst.Key() {
		return szTo, nil
	}
	if samePath(dst, szTo, szSrc) {
		return uniqueName(dst, szDir, szName), nil
	}
	if st, err := src.Stat(szSrc); err == nil && st.IsDir() && isWithin(dst, szSrc, szDir) {
		return "", errors.New("폴더를 자기 자신 안으로 복사할 수 없습니다: " + szName)
	}
	return szTo, nil
}

// foldCase 는 대소문자를 구분하지 않는 파일시스템인지 (Windows·macOS 로컬).
func foldCase(fsys fileSys) bool { return fsys.Local() && runtime.GOOS != "linux" }

func samePath(fsys fileSys, a, b string) bool {
	if foldCase(fsys) {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isWithin 은 szChild 가 szParent 이거나 그 아래인지.
func isWithin(fsys fileSys, szParent, szChild string) bool {
	if samePath(fsys, szParent, szChild) {
		return true
	}
	szPre := strings.TrimSuffix(szParent, fsys.Sep()) + fsys.Sep()
	if foldCase(fsys) {
		return strings.HasPrefix(strings.ToLower(szChild), strings.ToLower(szPre))
	}
	return strings.HasPrefix(szChild, szPre)
}

// uniqueName 은 szDir 에 없는 "이름 (n).확장자" 경로.
func uniqueName(fsys fileSys, szDir, szName string) string {
	szExt := filepath.Ext(szName)
	if st, err := fsys.Stat(fsys.Join(szDir, szName)); (err == nil && st.IsDir()) || szExt == szName {
		szExt = ""
	}
	szStem := strings.TrimSuffix(szName, szExt)
	for i := 1; ; i++ {
		szTry := fsys.Join(szDir, fmt.Sprintf("%s (%d)%s", szStem, i, szExt))
		if _, err := fsys.Lstat(szTry); errors.Is(err, fs.ErrNotExist) || i > 9999 {
			return szTry
		}
	}
}

// TempDir 는 서버 ↔ 서버 복사 때 거쳐 가는 이 PC 의 임시 폴더 (설정 transferTempDir, 비면 OS 임시 폴더).
func (pstrV *Service) TempDir() string {
	var szCfg string
	if pstrV.Store != nil {
		_ = json.Unmarshal(pstrV.Store.Setting("transferTempDir"), &szCfg)
	}
	if szCfg = strings.TrimSpace(szCfg); szCfg != "" {
		return transport.ExpandHome(szCfg)
	}
	return filepath.Join(os.TempDir(), "termgo-transfer")
}

// relay 는 원격 → 원격 복사를 이 PC 의 임시 폴더를 거쳐 한다
// (원본 서버에서 내려받기 → 대상 서버로 올리기 → 임시 사본 삭제).
// 파일 수는 올리기에서만 센다.
func (pstrV *Service) relay(pstrJ *copyJob, src fileSys, szSrc string, dst fileSys, szDst string) error {
	szBase := pstrV.TempDir()
	if err := os.MkdirAll(szBase, 0o700); err != nil {
		return fmt.Errorf("임시 폴더를 만들 수 없습니다 (%s): %w", szBase, err)
	}
	szStage, err := os.MkdirTemp(szBase, "copy-")
	if err != nil {
		return fmt.Errorf("임시 폴더를 만들 수 없습니다 (%s): %w", szBase, err)
	}
	defer os.RemoveAll(szStage)
	szLocal := filepath.Join(szStage, src.Base(szSrc))
	pstrJ.set(func(j *CopyJob) { j.Phase = "download" })
	down := xfer{szSession: pstrJ.j.SrcSession, bDown: true, job: pstrJ}
	if err := pstrV.copyTreeNoCount(src, szSrc, localFS{}, szLocal, down); err != nil {
		return err
	}
	pstrJ.set(func(j *CopyJob) { j.Phase = "upload" })
	return pstrV.copyTree(localFS{}, szLocal, dst, szDst, xfer{szSession: pstrJ.j.DstSession, job: pstrJ})
}

// copyTreeNoCount 는 파일 수를 세지 않는 copyTree (relay 의 받기 단계).
func (pstrV *Service) copyTreeNoCount(src fileSys, szSrc string, dst fileSys, szDst string, x xfer) error {
	x.bNoCount = true
	return pstrV.copyTree(src, szSrc, dst, szDst, x)
}
