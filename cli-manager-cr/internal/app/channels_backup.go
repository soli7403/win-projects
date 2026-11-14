package app

import (
	"encoding/json"
	"os"
	"time"

	"dariterm/internal/backup"
)

// BackupDialogs 는 백업 저장·열기 대화상자다 (Wails 런타임이 구현한다, spec/ipc.md §2 backup.*).
// 취소하면 bOk=false 다.
type BackupDialogs interface {
	SaveFile(szDefaultName string, szFilterName string, szPattern string) (string, bool)
	OpenFile(szFilterName string, szPattern string) (string, bool)
}

// backupNow 는 시각을 돌려준다 (시험에서 바꿀 수 있게 둔다).
var backupNow = time.Now

// SetBackupDialogs 는 백업 대화상자 구현을 등록한다.
func (pService *Service) SetBackupDialogs(pDialogs BackupDialogs) {
	pService.pDialogs = pDialogs
}

// registerBackup 은 backup.* 채널을 등록한다.
func (pService *Service) registerBackup() {
	pRouter := pService.pRouter
	pRouter.Handle("backup.export", func(slArgs []json.RawMessage) (any, error) {
		return pService.exportBackup()
	})
	pRouter.Handle("backup.import", func(slArgs []json.RawMessage) (any, error) {
		return pService.importBackup()
	})
}

// exportBackup 은 store 를 먼저 저장하고 백업 파일을 쓴다. 취소하면 false, 성공하면 저장 경로.
func (pService *Service) exportBackup() (any, error) {
	if pService.pDialogs == nil {
		return false, nil
	}
	tmNow := backupNow()
	szPath, bOk := pService.pDialogs.SaveFile(backup.DefaultName(tmNow), backup.FilterName(), backup.G_cszFilterPattern)
	if !bOk {
		return false, nil
	}
	if errFlush := pService.pStore.Flush(); errFlush != nil {
		return nil, errFlush
	}
	mapStore, errSnap := pService.pStore.Snapshot()
	if errSnap != nil {
		return nil, errSnap
	}
	rawVault, errVault := pService.pVault.ExportFile()
	if errVault != nil {
		return nil, errVault
	}
	bData, errEncode := backup.Encode(backup.NewFile(mapStore, rawVault, tmNow))
	if errEncode != nil {
		return nil, errEncode
	}
	if errWrite := os.WriteFile(szPath, bData, 0o600); errWrite != nil {
		return nil, errWrite
	}
	return szPath, nil
}

// importBackup 은 백업 파일을 고르고 store 와 볼트를 통째로 바꾼다. 볼트는 잠긴 상태가 된다.
func (pService *Service) importBackup() (any, error) {
	if pService.pDialogs == nil {
		return false, nil
	}
	szPath, bOk := pService.pDialogs.OpenFile(backup.FilterName(), backup.G_cszOpenPattern)
	if !bOk {
		return false, nil
	}
	bData, errRead := os.ReadFile(szPath)
	if errRead != nil {
		return nil, errRead
	}
	file, errDecode := backup.Decode(bData)
	if errDecode != nil {
		return nil, errDecode
	}
	if errStore := pService.pStore.ReplaceAll(file.M_mapStore); errStore != nil {
		return nil, errStore
	}
	if len(file.M_rawVault) > 0 && string(file.M_rawVault) != "null" {
		if errVault := pService.pVault.ImportFile(file.M_rawVault); errVault != nil {
			return nil, errVault
		}
	}
	pService.applyLanguage()
	pService.pRouter.Emit("store.changed")
	return true, nil
}
