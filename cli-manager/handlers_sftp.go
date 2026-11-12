package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"

	"termgo/internal/sftpsvc"
)

func (pstrApp *App) registerSftp() {
	h := pstrApp.router.Handle
	type A = []json.RawMessage
	sv := pstrApp.sftp
	h("sftp:home", func(a A) (any, error) { id, _ := arg[string](a, 0); return sv.Home(id) })
	h("sftp:list", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		return sv.List(id, p)
	})
	h("sftp:mkdir", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		return nil, sv.Mkdir(id, p)
	})
	h("sftp:rename", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		f, _ := arg[string](a, 1)
		t, _ := arg[string](a, 2)
		return nil, sv.Rename(id, f, t)
	})
	h("sftp:remove", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		ps, _ := arg[[]string](a, 1)
		return nil, sv.Remove(id, ps)
	})
	h("sftp:chmod", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		m, _ := arg[uint32](a, 2)
		return nil, sv.Chmod(id, p, m)
	})
	h("sftp:upload", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		ls, _ := arg[[]string](a, 1)
		d, _ := arg[string](a, 2)
		return sv.Upload(id, ls, d)
	})
	// 파일 패널 끌어 놓기: 세션(로컬/원격) → 세션 폴더 복사
	h("sftp:copy", func(a A) (any, error) {
		src, _ := arg[string](a, 0)
		ps, _ := arg[[]string](a, 1)
		dst, _ := arg[string](a, 2)
		d, _ := arg[string](a, 3)
		bOver, _ := arg[bool](a, 4)
		return sv.Copy(src, ps, dst, d, bOver)
	})
	h("sftp:cancelJob", func(a A) (any, error) { id, _ := arg[string](a, 0); sv.CancelJob(id); return nil, nil })
	h("sftp:uploadPick", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		d, _ := arg[string](a, 1)
		ls, err := wr.OpenMultipleFilesDialog(pstrApp.ctx, wr.OpenDialogOptions{Title: "업로드할 파일 선택"})
		if err != nil || len(ls) == 0 {
			return []string{}, err
		}
		return sv.Upload(id, ls, d)
	})
	h("sftp:download", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		rs, _ := arg[[]string](a, 1)
		dir, _ := arg[string](a, 2)
		if dir == "" {
			szHome, _ := os.UserHomeDir()
			var err error
			dir, err = wr.OpenDirectoryDialog(pstrApp.ctx, wr.OpenDialogOptions{Title: "저장할 폴더 선택", DefaultDirectory: filepath.Join(szHome, "Downloads"), CanCreateDirectories: true})
			if err != nil || dir == "" {
				return false, err
			}
		}
		if err := sv.Download(id, rs, dir); err != nil {
			return nil, err
		}
		return dir, nil
	})
	h("sftp:downloadTemp", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		rs, _ := arg[[]string](a, 1)
		szHome, _ := os.UserHomeDir()
		dir := filepath.Join(szHome, "Downloads")
		return dir, sv.Download(id, rs, dir)
	})
	h("sftp:preview", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		r, err := sv.Preview(id, p)
		if r == nil {
			return nil, err
		}
		return r, err
	})
	h("sftp:edit", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		return sv.Edit(id, p)
	})
	h("sftp:uploadForPrompt", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		src, err := arg[sftpsvc.PromptSource](a, 1)
		if err != nil {
			return nil, err
		}
		return sv.UploadForPrompt(id, src)
	})
	h("local:saveForPrompt", func(a A) (any, error) {
		src, err := arg[sftpsvc.PromptSource](a, 0)
		if err != nil {
			return nil, err
		}
		return sv.SaveForPromptLocal(src)
	})
	// 클립보드: 파일 목록·이미지 (텍스트는 handlers.go)
	h("clip:hasImage", func(A) (any, error) { return sftpsvc.ClipboardHasImage(), nil })
	h("clip:info", func(A) (any, error) {
		var files []string
		for _, f := range sftpsvc.ClipboardFiles() {
			if _, err := os.Stat(f); err == nil {
				files = append(files, f)
			}
		}
		if len(files) > 0 {
			return map[string]any{"type": "files", "files": files}, nil
		}
		if sz, _ := wr.ClipboardGetText(pstrApp.ctx); sz != "" {
			return map[string]any{"type": "text", "text": sz}, nil
		}
		if sftpsvc.ClipboardHasImage() {
			return map[string]any{"type": "image"}, nil
		}
		return map[string]any{"type": "empty"}, nil
	})
}
