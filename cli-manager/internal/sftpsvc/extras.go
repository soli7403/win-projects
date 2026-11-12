package sftpsvc

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"termgo/internal/transport"
	"termgo/internal/vault"
)

var g_mImgMime = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".bmp": "image/bmp", ".svg": "image/svg+xml",
}

// PreviewResult 는 미리보기 (이미지는 data URL, 텍스트는 본문).
type PreviewResult struct {
	Kind string `json:"kind"` // image | text
	Data string `json:"data"`
}

// Preview 는 작은 이미지/텍스트 파일을 읽어 돌려준다 (너무 크거나 바이너리면 nil).
func (pstrV *Service) Preview(szSession, szPath string) (*PreviewResult, error) {
	fsys, err := pstrV.fsFor(szSession)
	if err != nil {
		return nil, err
	}
	st, err := fsys.Stat(szPath)
	if err != nil {
		return nil, err
	}
	szExt := strings.ToLower(filepath.Ext(szPath))
	if mime, ok := g_mImgMime[szExt]; ok {
		if st.Size() > 20*1024*1024 {
			return nil, nil
		}
		arr, err := readAll(fsys, szPath)
		if err != nil {
			return nil, err
		}
		return &PreviewResult{Kind: "image", Data: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(arr)}, nil
	}
	if st.Size() > 512*1024 {
		return nil, nil
	}
	arr, err := readAll(fsys, szPath)
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(arr, 0) >= 0 {
		return nil, nil
	}
	return &PreviewResult{Kind: "text", Data: string(arr)}, nil
}

func readAll(fsys fileSys, szPath string) ([]byte, error) {
	f, err := fsys.Open(szPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	return buf.Bytes(), err
}

// OpenWithDefault 는 파일을 기본 프로그램으로 연다.
func OpenWithDefault(szFile string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", szFile).Start()
	case "darwin":
		return exec.Command("open", szFile).Start()
	}
	return exec.Command("xdg-open", szFile).Start()
}

// Edit 는 원격 파일을 임시 폴더로 받아 기본 프로그램으로 열고, 저장할 때마다 다시 올린다.
// 로컬 세션이면 파일을 그대로 기본 프로그램으로 연다.
func (pstrV *Service) Edit(szSession, szRemote string) (string, error) {
	if fsys, err := pstrV.fsFor(szSession); err == nil && fsys.Local() {
		return szRemote, OpenWithDefault(szRemote)
	}
	cl, err := pstrV.client(szSession)
	if err != nil {
		return "", err
	}
	arrID := make([]byte, 4)
	_, _ = rand.Read(arrID)
	szDir := filepath.Join(os.TempDir(), "termgo-edit", hex.EncodeToString(arrID))
	if err := os.MkdirAll(szDir, 0o700); err != nil {
		return "", err
	}
	szLocal := filepath.Join(szDir, path.Base(szRemote))
	t := &Transfer{ID: vault.NewUUID(), SessionID: szSession, Name: path.Base(szRemote), Direction: "down", State: "queued", RemotePath: szRemote, LocalPath: szLocal}
	if st, err := cl.Stat(szRemote); err == nil {
		t.Total = st.Size()
	}
	if err := finish(pstrV, t, pstrV.getFile(cl, t, szRemote, szLocal)); err != nil {
		return "", err
	}
	st0, err := os.Stat(szLocal)
	if err != nil {
		return "", err
	}
	go pstrV.watchEdit(szSession, szRemote, szLocal, st0.ModTime())
	if err := OpenWithDefault(szLocal); err != nil {
		return "", err
	}
	return szLocal, nil
}

func (pstrV *Service) watchEdit(szSession, szRemote, szLocal string, last time.Time) {
	tk := time.NewTicker(700 * time.Millisecond)
	defer tk.Stop()
	deadline := time.Now().Add(12 * time.Hour)
	for range tk.C {
		if time.Now().After(deadline) || pstrV.Sessions.Get(szSession) == nil {
			return
		}
		st, err := os.Stat(szLocal)
		if err != nil || !st.ModTime().After(last) {
			continue
		}
		time.Sleep(400 * time.Millisecond) // 저장이 끝나길 잠깐 기다린다
		last = st.ModTime()
		cl, err := pstrV.client(szSession)
		if err == nil {
			t := &Transfer{ID: vault.NewUUID(), SessionID: szSession, Name: path.Base(szRemote), Direction: "up", Total: st.Size(), RemotePath: szRemote, LocalPath: szLocal}
			err = finish(pstrV, t, pstrV.putFile(cl, t, szLocal, szRemote))
		}
		if err != nil {
			pstrV.UI.Send("toast", map[string]string{"kind": "error", "text": "업로드 실패: " + err.Error()})
		} else {
			pstrV.UI.Send("toast", map[string]string{"kind": "ok", "text": path.Base(szRemote) + " 저장됨 → 서버에 업로드 완료"})
		}
	}
}

// PromptSource 는 uploadForPrompt / saveForPrompt 의 입력.
// buffer 의 data 는 어댑터가 {"__b64":"..."} 로 바꿔 보낸다.
type PromptSource struct {
	Kind  string          `json:"kind"` // clipboardImage | files | buffer
	Paths []string        `json:"paths"`
	Name  string          `json:"name"`
	Data  json.RawMessage `json:"data"`
}

func (pstrP PromptSource) bytes() ([]byte, error) {
	var m struct {
		B64 string `json:"__b64"`
	}
	if err := json.Unmarshal(pstrP.Data, &m); err != nil || m.B64 == "" {
		return nil, errors.New("데이터를 해석할 수 없습니다")
	}
	return base64.StdEncoding.DecodeString(m.B64)
}

var reUnsafeName = regexp.MustCompile("[\\s'\"`$\\\\]")

func stamp() string { return time.Now().Format("20060102-150405") }

func rand4() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// UploadForPrompt 는 클립보드 이미지/파일을 설정한 업로드 폴더에 올리고 원격 경로를 돌려준다.
func (pstrV *Service) UploadForPrompt(szSession string, src PromptSource) ([]string, error) {
	q := pstrV.queue(szSession)
	q.Lock()
	defer q.Unlock()
	cl, err := pstrV.client(szSession)
	if err != nil {
		return nil, err
	}
	var szCfg string
	_ = json.Unmarshal(pstrV.Store.Setting("uploadDir"), &szCfg)
	if szCfg == "" {
		szCfg = "~/.termgo/uploads"
	}
	szCfg = strings.Replace(szCfg, ".brightterm", ".termgo", 1)
	szDir, err := pstrV.expand(szSession, szCfg)
	if err != nil {
		return nil, err
	}
	mkdirAll(cl, szDir)
	out := []string{}
	writeBuf := func(szName string, arr []byte) (string, error) {
		szRemote := path.Join(szDir, szName)
		t := &Transfer{ID: vault.NewUUID(), SessionID: szSession, Name: szName, Direction: "up", Total: int64(len(arr)), State: "running", RemotePath: szRemote}
		pstrV.update(t)
		f, err := cl.Create(szRemote)
		if err == nil {
			_, err = f.Write(arr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		return szRemote, finish(pstrV, t, err)
	}
	switch src.Kind {
	case "clipboardImage":
		arr, err := ClipboardImagePNG()
		if err != nil {
			return nil, err
		}
		r, err := writeBuf(fmt.Sprintf("bt-%s-%s.png", stamp(), rand4()), arr)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	case "buffer":
		arr, err := src.bytes()
		if err != nil {
			return nil, err
		}
		szExt := path.Ext(src.Name)
		if szExt == "" {
			szExt = ".png"
		}
		r, err := writeBuf(fmt.Sprintf("bt-%s-%s%s", stamp(), rand4(), szExt), arr)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	default:
		for _, f := range src.Paths {
			st, err := os.Stat(f)
			if err != nil {
				return out, err
			}
			if st.IsDir() {
				r, err := pstrV.uploadPath(cl, szSession, f, szDir)
				if err != nil {
					return out, err
				}
				out = append(out, r)
				continue
			}
			szRemote := path.Join(szDir, fmt.Sprintf("bt-%s-%s-%s", stamp(), rand4(), reUnsafeName.ReplaceAllString(filepath.Base(f), "_")))
			t := &Transfer{ID: vault.NewUUID(), SessionID: szSession, Name: filepath.Base(f), Direction: "up", Total: st.Size(), State: "queued", RemotePath: szRemote, LocalPath: f}
			pstrV.update(t)
			if err := finish(pstrV, t, pstrV.putFile(cl, t, f, szRemote)); err != nil {
				return out, err
			}
			out = append(out, szRemote)
		}
	}
	go pstrV.cleanupRemote(cl, szDir)
	return out, nil
}

func (pstrV *Service) cleanupRemote(cl *sftp.Client, szDir string) {
	iDays := pstrV.Store.SettingInt("uploadCleanupDays")
	if iDays <= 0 {
		return
	}
	items, err := cl.ReadDir(szDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-time.Duration(iDays) * 24 * time.Hour)
	for _, it := range items {
		if strings.HasPrefix(it.Name(), "bt-") && it.ModTime().Before(cutoff) {
			_ = cl.Remove(path.Join(szDir, it.Name()))
		}
	}
}

// SaveForPromptLocal 은 로컬 터미널용: 이미지는 이 PC 의 업로드 폴더에 저장하고 경로를, 파일은 원래 경로를 돌려준다.
func (pstrV *Service) SaveForPromptLocal(src PromptSource) ([]string, error) {
	if src.Kind == "files" {
		return src.Paths, nil
	}
	var szCfg string
	_ = json.Unmarshal(pstrV.Store.Setting("uploadDir"), &szCfg)
	if szCfg == "" {
		szCfg = "~/.termgo/uploads"
	}
	szCfg = strings.Replace(szCfg, ".brightterm", ".termgo", 1)
	szDir := transport.ExpandHome(szCfg)
	if err := os.MkdirAll(szDir, 0o700); err != nil {
		return nil, err
	}
	var arr []byte
	szExt := ".png"
	var err error
	if src.Kind == "clipboardImage" {
		arr, err = ClipboardImagePNG()
	} else {
		arr, err = src.bytes()
		if e := path.Ext(src.Name); e != "" {
			szExt = e
		}
	}
	if err != nil {
		return nil, err
	}
	szFile := filepath.Join(szDir, fmt.Sprintf("bt-%s-%s%s", stamp(), rand4(), szExt))
	if err := os.WriteFile(szFile, arr, 0o600); err != nil {
		return nil, err
	}
	go cleanupLocal(szDir, pstrV.Store.SettingInt("uploadCleanupDays"))
	return []string{szFile}, nil
}

func cleanupLocal(szDir string, iDays int) {
	if iDays <= 0 {
		return
	}
	cutoff := time.Now().Add(-time.Duration(iDays) * 24 * time.Hour)
	ents, _ := os.ReadDir(szDir)
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), "bt-") {
			continue
		}
		if fi, err := e.Info(); err == nil && fi.Mode().IsRegular() && fi.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(szDir, e.Name()))
		}
	}
}
