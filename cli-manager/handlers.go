package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"

	"termgo/internal/session"
	"termgo/internal/transport"
	"termgo/internal/vault"
)

// arg 는 JSON 인자 목록의 i 번째를 T 로 해석한다 (없으면 zero value).
func arg[T any](slArgs []json.RawMessage, i int) (T, error) {
	var v T
	if i >= len(slArgs) {
		return v, nil
	}
	err := json.Unmarshal(slArgs[i], &v)
	return v, err
}

func (pstrApp *App) registerHandlers() {
	h := pstrApp.router.Handle
	type A = []json.RawMessage

	// store
	h("store:get", func(A) (any, error) { return pstrApp.store.Get(), nil })
	h("store:patch", func(a A) (any, error) {
		m, err := arg[map[string]json.RawMessage](a, 0)
		if err != nil {
			return nil, err
		}
		pstrApp.store.Patch(m)
		return nil, nil
	})

	// vault
	h("vault:status", func(A) (any, error) { return pstrApp.vault.Status(), nil })
	h("vault:setup", func(a A) (any, error) {
		sz, _ := arg[string](a, 0)
		return pstrApp.vault.Setup(sz)
	})
	h("vault:unlock", func(a A) (any, error) { sz, _ := arg[string](a, 0); return pstrApp.vault.Unlock(sz), nil })
	h("vault:unlockOs", func(A) (any, error) { return pstrApp.vault.UnlockOS(), nil })
	h("vault:recover", func(a A) (any, error) {
		c, _ := arg[string](a, 0)
		p, _ := arg[string](a, 1)
		return pstrApp.vault.Recover(c, p), nil
	})
	h("vault:lock", func(A) (any, error) { pstrApp.vault.Lock(); return nil, nil })
	h("vault:changePassword", func(a A) (any, error) {
		o, _ := arg[string](a, 0)
		n, _ := arg[string](a, 1)
		return pstrApp.vault.ChangePassword(o, n), nil
	})
	h("vault:setOsUnlock", func(a A) (any, error) {
		b, _ := arg[bool](a, 0)
		return nil, pstrApp.vault.SetOsUnlock(b)
	})
	h("cred:list", func(A) (any, error) { return pstrApp.vault.List(), nil })
	h("cred:get", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		s, err := pstrApp.vault.Get(id)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, nil
		}
		return s, nil
	})
	h("cred:save", func(a A) (any, error) {
		in, err := arg[vault.Input](a, 0)
		if err != nil {
			return nil, err
		}
		return pstrApp.vault.Save(in)
	})
	h("cred:delete", func(a A) (any, error) { id, _ := arg[string](a, 0); return nil, pstrApp.vault.Remove(id) })

	// sessions
	h("session:open", func(a A) (any, error) {
		o, err := arg[session.OpenOptions](a, 0)
		if err != nil {
			return nil, err
		}
		return pstrApp.sessions.Open(o)
	})
	h("session:close", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		pstrApp.sessions.Close(id)
		pstrApp.sftp.Forget(id)
		return nil, nil
	})
	h("session:reconnect", func(a A) (any, error) {
		id, _ := arg[string](a, 0)
		if s := pstrApp.sessions.Get(id); s != nil {
			s.ReconnectNow()
		}
		return nil, nil
	})

	// 로컬 셸·시리얼 목록
	h("local:shells", func(A) (any, error) { return transport.ListShells(), nil })
	h("local:tmux", func(A) (any, error) { return transport.TmuxAvailable(), nil })
	h("serial:list", func(A) (any, error) { return transport.ListSerialPorts(), nil })

	// 클립보드
	h("clip:writeText", func(a A) (any, error) {
		sz, _ := arg[string](a, 0)
		return nil, wr.ClipboardSetText(pstrApp.ctx, sz)
	})
	h("clip:readText", func(A) (any, error) { return wr.ClipboardGetText(pstrApp.ctx) })

	// 대화상자
	h("dialog:readKeyFile", func(A) (any, error) {
		szHome, _ := os.UserHomeDir()
		sz, err := wr.OpenFileDialog(pstrApp.ctx, wr.OpenDialogOptions{Title: "개인 키 파일 선택", DefaultDirectory: filepath.Join(szHome, ".ssh"), ShowHiddenFiles: true})
		if err != nil || sz == "" {
			return nil, err
		}
		arr, err := os.ReadFile(sz)
		if err != nil {
			return nil, err
		}
		return map[string]string{"path": sz, "text": string(arr)}, nil
	})
	h("dialog:chooseDir", func(A) (any, error) {
		sz, err := wr.OpenDirectoryDialog(pstrApp.ctx, wr.OpenDialogOptions{CanCreateDirectories: true})
		if err != nil || sz == "" {
			return nil, err
		}
		return sz, nil
	})

	// 백업
	h("backup:export", pstrApp.backupExport)
	h("backup:import", pstrApp.backupImport)

	// 앱
	h("app:info", func(A) (any, error) {
		return map[string]string{"version": g_szVersion, "platform": runtime.GOOS, "dataDir": pstrApp.szDir}, nil
	})
	h("app:titlebar", func(A) (any, error) { return nil, nil })
	h("app:openLogs", func(A) (any, error) {
		wr.BrowserOpenURL(pstrApp.ctx, ensureDir(pstrApp.logsDir()))
		return nil, nil
	})
	h("app:openExternal", func(a A) (any, error) {
		sz, _ := arg[string](a, 0)
		if len(sz) > 8 && (sz[:7] == "http://" || sz[:8] == "https://") {
			wr.BrowserOpenURL(pstrApp.ctx, sz)
		}
		return nil, nil
	})
	h("app:resetWindow", func(A) (any, error) { pstrApp.resetWindow(); return nil, nil })
	h("app:toggleFullScreen", func(A) (any, error) { wr.WindowToggleMaximise(pstrApp.ctx); return nil, nil })
}

type backupFile struct {
	App        string                     `json:"app"`
	Version    int                        `json:"version"`
	ExportedAt int64                      `json:"exportedAt"`
	Store      map[string]json.RawMessage `json:"store"`
	Vault      *vault.File                `json:"vault"`
}

func (pstrApp *App) backupExport(a []json.RawMessage) (any, error) {
	szHome, _ := os.UserHomeDir()
	szPath, err := wr.SaveFileDialog(pstrApp.ctx, wr.SaveDialogOptions{
		Title:            "백업 내보내기",
		DefaultDirectory: filepath.Join(szHome, "Documents"),
		DefaultFilename:  "termgo-backup-" + time.Now().Format("2006-01-02") + ".btbackup",
		Filters:          []wr.FileFilter{{DisplayName: "백업 (*.btbackup)", Pattern: "*.btbackup"}},
	})
	if err != nil || szPath == "" {
		return false, err
	}
	if err := pstrApp.store.Flush(); err != nil {
		return nil, err
	}
	// v1.1.0 백업 형식과 호환되도록 app 값은 "brightterm" 을 유지한다.
	arr, err := json.MarshalIndent(backupFile{App: "brightterm", Version: 1, ExportedAt: time.Now().UnixMilli(), Store: pstrApp.store.Get(), Vault: pstrApp.vault.ExportRaw()}, "", "  ")
	if err != nil {
		return nil, err
	}
	return szPath, os.WriteFile(szPath, arr, 0o600)
}

func (pstrApp *App) backupImport(a []json.RawMessage) (any, error) {
	szPath, err := wr.OpenFileDialog(pstrApp.ctx, wr.OpenDialogOptions{
		Filters: []wr.FileFilter{{DisplayName: "백업 (*.btbackup;*.json)", Pattern: "*.btbackup;*.json"}},
	})
	if err != nil || szPath == "" {
		return false, err
	}
	arr, err := os.ReadFile(szPath)
	if err != nil {
		return nil, err
	}
	var b backupFile
	if err := json.Unmarshal(arr, &b); err != nil {
		return nil, err
	}
	if b.App != "brightterm" && b.App != "termgo" {
		return nil, errors.New("백업 파일이 아닙니다")
	}
	if err := pstrApp.store.ReplaceAll(b.Store); err != nil {
		return nil, err
	}
	if b.Vault != nil {
		if err := pstrApp.vault.ImportRaw(*b.Vault); err != nil {
			return nil, err
		}
	}
	return true, nil
}
