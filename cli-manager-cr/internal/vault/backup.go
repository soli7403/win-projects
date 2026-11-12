package vault

import (
	"encoding/json"
)

// ExportFile 은 볼트 파일 구조 전체를 돌려준다 (백업용, data/vault.md §6). 볼트가 없으면 nil.
func (pv *Vault) ExportFile() (json.RawMessage, error) {
	pv.mtxData.Lock()
	defer pv.mtxData.Unlock()
	if pv.pfile == nil {
		return nil, nil
	}
	return json.Marshal(pv.pfile)
}

// ImportFile 은 백업의 볼트 구조로 통째로 교체한다. OS 보호 값은 비우고 볼트는 잠긴 상태가 된다 (data/vault.md §6).
func (pv *Vault) ImportFile(raw json.RawMessage) error {
	var file vaultFile
	if errJson := json.Unmarshal(raw, &file); errJson != nil {
		return errJson
	}
	if file.M_slItems == nil {
		file.M_slItems = []itemRecord{}
	}
	file.M_szOsWrapped = ""
	file.M_szOsWrappedWin = ""
	pv.mtxData.Lock()
	wipe(pv.bKey)
	pv.bKey = nil
	pv.pfile = &file
	errSave := pv.saveLocked()
	pv.mtxData.Unlock()
	if errSave != nil {
		return errSave
	}
	pv.notifyLock(false)
	return nil
}
