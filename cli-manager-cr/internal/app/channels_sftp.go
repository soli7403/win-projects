package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"dariterm/internal/clipboard"
	"dariterm/internal/dialog"
	"dariterm/internal/sftpfs"
)

// sftpEntry 는 세션의 SFTP 기능 묶음과 그 클라이언트다. 클라이언트가 바뀌면(재접속) 묶음을 새로 만든다.
type sftpEntry struct {
	client *sftp.Client
	svc    *sftpfs.Service
}

var (
	g_mtxSftp sync.Mutex
	g_mapSftp = map[string]*sftpEntry{}
)

// sftpClientProvider 는 SSH 세션 전송이 가진 sftp 클라이언트 얻기 기능이다.
type sftpClientProvider interface {
	SftpClient() (*sftp.Client, error)
}

// sftpFor 는 세션 id 의 파일 패널 기능을 돌려준다. SSH 세션은 SFTP, 로컬 터미널은 이 PC 파일시스템
// (protocols.md §3, behavior/file-panel.md §1). 그 밖의 세션이면 오류.
func (pService *Service) sftpFor(szId string) (*sftpfs.Service, error) {
	pSession, errGet := pService.pSessions.Get(szId)
	if errGet != nil {
		return nil, errGet
	}
	transport := pSession.Transport()
	if transport == nil {
		return nil, errors.New("연결되어 있지 않습니다")
	}
	info := pSession.Snapshot()
	emit := func(szEvent string, anyData ...any) {
		pService.pRouter.Emit(szEvent, anyData...)
	}
	if info.M_szProtocol == "local" {
		g_mtxSftp.Lock()
		defer g_mtxSftp.Unlock()
		if entry, bFound := g_mapSftp[szId]; bFound && entry.client == nil {
			return entry.svc, nil
		}
		svc := sftpfs.NewLocalService(szId, localStartDir(info.M_szTarget), emit)
		g_mapSftp[szId] = &sftpEntry{svc: svc}
		return svc, nil
	}
	provider, bOk := transport.(sftpClientProvider)
	if !bOk {
		return nil, errors.New("SSH로 연결된 세션에서만 사용할 수 있습니다")
	}
	client, errClient := provider.SftpClient()
	if errClient != nil {
		return nil, errClient
	}
	g_mtxSftp.Lock()
	defer g_mtxSftp.Unlock()
	if entry, bFound := g_mapSftp[szId]; bFound && entry.client == client {
		return entry.svc, nil
	}
	svc := sftpfs.NewService(szId, client, emit, info.M_szTarget)
	g_mapSftp[szId] = &sftpEntry{client: client, svc: svc}
	return svc, nil
}

// localStartDir 는 로컬 세션 대상 표기("로컬 · <폴더>")에서 시작 폴더를 꺼낸다. "~" 이거나 없으면 빈 값(= 홈).
func localStartDir(szTarget string) string {
	szCwd := strings.TrimSpace(strings.TrimPrefix(szTarget, "로컬 · "))
	if szCwd == "" || szCwd == "~" || szCwd == szTarget {
		return ""
	}
	return expandHome(szCwd)
}

// startCopy 는 sftp.copy 채널이다: 인자 (원본 세션, 경로 배열, 대상 세션, 대상 폴더, 덮어쓰기) (file-panel.md §5).
func (pService *Service) startCopy(slArgs []json.RawMessage) (any, error) {
	szSrc, errSrc := argString(slArgs, 0)
	if errSrc != nil {
		return nil, errSrc
	}
	var slPaths []string
	if errPaths := argInto(slArgs, 1, &slPaths); errPaths != nil {
		return nil, errPaths
	}
	szDst, errDst := argString(slArgs, 2)
	if errDst != nil {
		return nil, errDst
	}
	szDir := ""
	if len(slArgs) > 3 {
		szDir, _ = argString(slArgs, 3)
	}
	bOverwrite := false
	if len(slArgs) > 4 {
		_ = argInto(slArgs, 4, &bOverwrite)
	}
	srcSvc, errSrcSvc := pService.sftpFor(szSrc)
	if errSrcSvc != nil {
		return nil, errSrcSvc
	}
	dstSvc, errDstSvc := pService.sftpFor(szDst)
	if errDstSvc != nil {
		return nil, errDstSvc
	}
	return sftpfs.StartCopy(sftpfs.CopyRequest{
		Src: srcSvc, SrcTitle: pService.sessionTitle(szSrc), Paths: slPaths,
		Dst: dstSvc, DstTitle: pService.sessionTitle(szDst), DstDir: szDir, Overwrite: bOverwrite,
		TempDir: expandHome(strings.TrimSpace(pService.settingString("transferTempDir", ""))),
		Emit:    func(szEvent string, anyData ...any) { pService.pRouter.Emit(szEvent, anyData...) },
	})
}

// sessionTitle 은 세션 제목이다 (없으면 빈 값).
func (pService *Service) sessionTitle(szId string) string {
	if pSession, errGet := pService.pSessions.Get(szId); errGet == nil {
		return pSession.Snapshot().M_szTitle
	}
	return ""
}

// sftpOrigin 은 sftp.uploadForPrompt 의 원본 객체다 (spec/ipc.md §2).
type sftpOrigin struct {
	Kind  string   `json:"kind"`
	Paths []string `json:"paths"`
	Name  string   `json:"name"`
	Data  *struct {
		B64 string `json:"__b64"`
	} `json:"data"`
}

// registerSftp 는 sftp.* 채널과 local.saveForPrompt 를 등록한다.
func (pService *Service) registerSftp() {
	pRouter := pService.pRouter
	withSvc := func(fnBody func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error)) func([]json.RawMessage) (any, error) {
		return func(slArgs []json.RawMessage) (any, error) {
			szId, errId := argString(slArgs, 0)
			if errId != nil {
				return nil, errId
			}
			svc, errSvc := pService.sftpFor(szId)
			if errSvc != nil {
				return nil, errSvc
			}
			return fnBody(svc, slArgs[1:])
		}
	}
	pRouter.Handle("sftp.home", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		return svc.Fs.Home()
	}))
	pRouter.Handle("sftp.list", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szPath, _ := argString(slArgs, 0)
		return svc.Fs.List(szPath)
	}))
	pRouter.Handle("sftp.mkdir", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szPath, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return nil, svc.Fs.Mkdir(szPath)
	}))
	pRouter.Handle("sftp.rename", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szOld, errOld := argString(slArgs, 0)
		if errOld != nil {
			return nil, errOld
		}
		szNew, errNew := argString(slArgs, 1)
		if errNew != nil {
			return nil, errNew
		}
		return nil, svc.Fs.Rename(szOld, szNew)
	}))
	pRouter.Handle("sftp.remove", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		var slPaths []string
		if errArg := argInto(slArgs, 0, &slPaths); errArg != nil {
			return nil, errArg
		}
		return nil, svc.Fs.Remove(slPaths)
	}))
	pRouter.Handle("sftp.chmod", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szPath, errPath := argString(slArgs, 0)
		if errPath != nil {
			return nil, errPath
		}
		var iMode int
		if errArg := argInto(slArgs, 1, &iMode); errArg != nil {
			return nil, errArg
		}
		return nil, svc.Fs.Chmod(szPath, iMode)
	}))
	pRouter.Handle("sftp.upload", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		var slLocal []string
		if errArg := argInto(slArgs, 0, &slLocal); errArg != nil {
			return nil, errArg
		}
		szRemoteDir, errDir := argString(slArgs, 1)
		if errDir != nil {
			return nil, errDir
		}
		return svc.UploadPaths(slLocal, szRemoteDir)
	}))
	pRouter.Handle("sftp.copy", pService.startCopy)
	pRouter.Handle("sftp.cancelCopy", func(slArgs []json.RawMessage) (any, error) {
		szJob, errJob := argString(slArgs, 0)
		if errJob != nil {
			return nil, errJob
		}
		sftpfs.CancelCopy(szJob)
		return nil, nil
	})
	pRouter.Handle("sftp.uploadPick", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szRemoteDir, errDir := argString(slArgs, 0)
		if errDir != nil {
			return nil, errDir
		}
		slLocal, errPick := dialog.OpenFiles("업로드할 파일 선택")
		if errPick != nil || len(slLocal) == 0 {
			return []string{}, errPick
		}
		return svc.UploadPaths(slLocal, szRemoteDir)
	}))
	pRouter.Handle("sftp.download", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		var slRemote []string
		if errArg := argInto(slArgs, 0, &slRemote); errArg != nil {
			return nil, errArg
		}
		szLocalDir := ""
		if len(slArgs) > 1 {
			szLocalDir, _ = argString(slArgs, 1)
		}
		if szLocalDir == "" {
			szPicked, errPick := dialog.ChooseDir("저장할 폴더 선택", downloadsDir(), true)
			if errPick != nil || szPicked == "" {
				return false, errPick
			}
			szLocalDir = szPicked
		}
		return svc.DownloadPaths(slRemote, szLocalDir)
	}))
	pRouter.Handle("sftp.preview", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		szPath, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		pPrev, errPrev := svc.Fs.Preview(szPath)
		if errPrev != nil || pPrev == nil {
			return json.RawMessage("null"), errPrev
		}
		return pPrev, nil
	}))
	pRouter.Handle("sftp.edit", func(slArgs []json.RawMessage) (any, error) {
		szId, errId := argString(slArgs, 0)
		if errId != nil {
			return nil, errId
		}
		svc, errSvc := pService.sftpFor(szId)
		if errSvc != nil {
			return nil, errSvc
		}
		szPath, errArg := argString(slArgs, 1)
		if errArg != nil {
			return nil, errArg
		}
		// 로컬 터미널은 그 파일을 그대로 기본 프로그램으로 연다 (file-panel.md §1).
		if svc.Fs.Backend().IsLocal() {
			return szPath, openWithShell(szPath)
		}
		// 세션이 사라지면 편집 감시를 끝낸다 (protocols.md §3, 12시간 상한은 sftpfs 에 있다).
		alive := func() bool {
			_, errGet := pService.pSessions.Get(szId)
			return errGet == nil
		}
		return svc.StartEdit(szPath, openWithShell, func(kind string, text string) {
			pRouter.Emit("toast", map[string]any{"kind": kind, "text": text})
		}, alive)
	})
	pRouter.Handle("sftp.uploadForPrompt", withSvc(func(svc *sftpfs.Service, slArgs []json.RawMessage) (any, error) {
		var origin sftpOrigin
		if errArg := argInto(slArgs, 0, &origin); errArg != nil {
			return nil, errArg
		}
		szDir := pService.remoteUploadDir(svc)
		if errDir := ensureUploadDir(svc.Fs, szDir); errDir != nil {
			return nil, errDir
		}
		slOut, errUp := pService.uploadPrompt(svc, origin, szDir)
		if errUp == nil {
			go func() {
				_, _ = svc.CleanupOldRemotePrompt(szDir, pService.settingInt("uploadCleanupDays", 7), time.Now())
			}()
		}
		return slOut, errUp
	}))
	pRouter.Handle("local.saveForPrompt", func(slArgs []json.RawMessage) (any, error) {
		var origin sftpOrigin
		if errArg := argInto(slArgs, 0, &origin); errArg != nil {
			return nil, errArg
		}
		return pService.saveLocalForPrompt(origin)
	})
}

// uploadPrompt 는 붙여넣기 원본 종류별로 원격 폴더에 올린다.
func (pService *Service) uploadPrompt(svc *sftpfs.Service, origin sftpOrigin, szDir string) ([]string, error) {
	tNow := time.Now()
	switch origin.Kind {
	case "clipboardImage":
		bPng, errPng := clipboard.ImagePNG()
		if errPng != nil {
			return nil, errPng
		}
		szRemote, errUp := svc.UploadPromptBytes(bPng, "clipboard.png", szDir, tNow)
		if errUp != nil {
			return nil, errUp
		}
		return []string{szRemote}, nil
	case "buffer":
		bData, errDecode := decodeBuffer(origin)
		if errDecode != nil {
			return nil, errDecode
		}
		szRemote, errUp := svc.UploadPromptBytes(bData, origin.Name, szDir, tNow)
		if errUp != nil {
			return nil, errUp
		}
		return []string{szRemote}, nil
	default:
		return svc.UploadPromptPaths(origin.Paths, szDir, tNow)
	}
}

// saveLocalForPrompt 는 로컬 세션용 저장이다: 파일은 원래 경로, 이미지·버퍼는 업로드 폴더에 dt- 이름으로 저장한다.
func (pService *Service) saveLocalForPrompt(origin sftpOrigin) ([]string, error) {
	szDir := expandHome(pService.settingString("uploadDir", "~/.dariterm/uploads"))
	switch origin.Kind {
	case "clipboardImage":
		bPng, errPng := clipboard.ImagePNG()
		if errPng != nil {
			return nil, errPng
		}
		return saveBytesLocal(szDir, bPng, "clipboard.png", pService)
	case "buffer":
		bData, errDecode := decodeBuffer(origin)
		if errDecode != nil {
			return nil, errDecode
		}
		return saveBytesLocal(szDir, bData, origin.Name, pService)
	default:
		return origin.Paths, nil
	}
}

func saveBytesLocal(szDir string, bData []byte, szGiven string, pService *Service) ([]string, error) {
	if errMk := os.MkdirAll(szDir, 0o700); errMk != nil {
		return nil, errMk
	}
	szName := sftpfs.PromptImageName(time.Now(), sftpfs.RandHex4(), szGiven)
	szPath := filepath.Join(szDir, szName)
	if errWrite := os.WriteFile(szPath, bData, 0o600); errWrite != nil {
		return nil, errWrite
	}
	_, _ = sftpfs.CleanupOldPrompt(szDir, pService.settingInt("uploadCleanupDays", 7), time.Now())
	return []string{szPath}, nil
}

// remoteUploadDir 는 설정 uploadDir 를 원격 경로로 바꾼다 (~ 는 원격 홈). 없으면 만든다.
func (pService *Service) remoteUploadDir(svc *sftpfs.Service) string {
	szDir := pService.settingString("uploadDir", "~/.dariterm/uploads")
	if strings.HasPrefix(szDir, "~/") {
		if szHome, errHome := svc.Fs.Home(); errHome == nil {
			return szHome + szDir[1:]
		}
	}
	return szDir
}

// ensureUploadDir 는 서버 업로드 폴더를 중간 폴더까지 만든다 (behavior/terminal.md §5 "없으면 자동으로 만든다").
// 기본값 ~/.dariterm/uploads 처럼 상위 폴더도 없을 수 있어 한 단계만 만드는 Mkdir 로는 부족하다.
func ensureUploadDir(pFs *sftpfs.Fs, szDir string) error {
	if errMk := pFs.MkdirAll(szDir); errMk != nil {
		return fmt.Errorf("업로드 폴더를 만들지 못했습니다 (%s): %w", szDir, errMk)
	}
	return nil
}

// decodeBuffer 는 {"__b64": "..."} 로 감싼 바이트를 푼다.
func decodeBuffer(origin sftpOrigin) ([]byte, error) {
	if origin.Data == nil {
		return nil, errors.New("데이터가 없습니다")
	}
	return base64.StdEncoding.DecodeString(origin.Data.B64)
}

// downloadsDir 는 다운로드 대화상자의 시작 폴더다 (사용자 다운로드 폴더, 없으면 홈).
func downloadsDir() string {
	if szHome, errHome := os.UserHomeDir(); errHome == nil {
		return filepath.Join(szHome, "Downloads")
	}
	return ""
}

// settingString 과 settingInt 는 설정 값을 읽는다 (없으면 기본값).
func (pService *Service) settingString(szKey string, szDefault string) string {
	mapSnap, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return szDefault
	}
	mapSettings, bOk := mapSnap["settings"].(map[string]any)
	if !bOk {
		return szDefault
	}
	szValue, bIsStr := mapSettings[szKey].(string)
	if !bIsStr {
		return szDefault
	}
	return szValue
}

func (pService *Service) settingInt(szKey string, iDefault int) int {
	mapSnap, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return iDefault
	}
	mapSettings, bOk := mapSnap["settings"].(map[string]any)
	if !bOk {
		return iDefault
	}
	fValue, bIsNum := mapSettings[szKey].(float64)
	if !bIsNum {
		return iDefault
	}
	return int(fValue)
}
