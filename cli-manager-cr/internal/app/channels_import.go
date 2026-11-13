package app

import (
	"encoding/json"
	"os"

	"dariterm/internal/dialog"
	"dariterm/internal/importer"
	"dariterm/internal/vault"
)

// vaultKeyStore 는 볼트를 가져오기의 KeyStore 로 쓰는 어댑터다.
type vaultKeyStore struct {
	pService *Service
}

// Unlocked 는 볼트가 열려 있는지 돌려준다.
func (pKeys vaultKeyStore) Unlocked() bool {
	return pKeys.pService.pVault.Status().M_bUnlocked
}

// SaveKey 는 개인 키를 종류 key 의 볼트 항목으로 저장한다.
func (pKeys vaultKeyStore) SaveKey(szName string, szUsername string, szPrivateKey string) (string, error) {
	input := vault.SaveInput{M_szName: szName, M_szKind: "key", M_pszPrivateKey: &szPrivateKey}
	if szUsername != "" {
		input.M_pszUsername = &szUsername
	}
	meta, errSave := pKeys.pService.pVault.Save(input)
	if errSave != nil {
		return "", errSave
	}
	return meta.M_szId, nil
}

// registerImport 는 import.* 채널을 등록한다 (spec/ipc.md §2 가져오기).
func (pService *Service) registerImport() {
	pRouter := pService.pRouter
	pRouter.Handle("import.scan", func(slArgs []json.RawMessage) (any, error) {
		szHome, _ := os.UserHomeDir()
		slCands := importer.ScanPutty(importer.ReadPuttySessions)
		slCands = append(slCands, importer.ScanSecureCrt("")...)
		slCands = append(slCands, importer.ScanSshConfig(szHome)...)
		return slCands, nil
	})
	pRouter.Handle("import.scanSecureCrt", func(slArgs []json.RawMessage) (any, error) {
		szDir := ""
		if len(slArgs) > 0 {
			szDir, _ = argString(slArgs, 0)
		}
		if szDir == "" {
			szPicked, errPick := dialog.ChooseDir("SecureCRT 설정 폴더 선택", importer.SecureCrtConfigDir(), false)
			if errPick != nil {
				return nil, errPick
			}
			if szPicked == "" {
				return nil, nil
			}
			szDir = szPicked
		}
		return importer.ScanSecureCrt(szDir), nil
	})
	pRouter.Handle("import.scanAws", func(slArgs []json.RawMessage) (any, error) {
		szProfile := ""
		if len(slArgs) > 0 {
			szProfile, _ = argString(slArgs, 0)
		}
		szHome, _ := os.UserHomeDir()
		slCands, errScan := importer.ScanAws(importer.RunAwsCli(szProfile), szHome)
		if errScan != nil {
			return nil, errScan
		}
		return slCands, nil
	})
	pRouter.Handle("import.apply", func(slArgs []json.RawMessage) (any, error) {
		var slCands []importer.Candidate
		if errArg := argInto(slArgs, 0, &slCands); errArg != nil {
			return nil, errArg
		}
		szFolder, errFolder := argString(slArgs, 1)
		if errFolder != nil {
			return nil, errFolder
		}
		slGroups, slHosts, errLoad := pService.loadGroupsHosts()
		if errLoad != nil {
			return nil, errLoad
		}
		out, errApply := importer.ApplyCandidates(slCands, szFolder, slGroups, slHosts, vaultKeyStore{pService: pService})
		if errApply != nil {
			return nil, errApply
		}
		if errPatch := pService.pStore.Patch(map[string]any{"groups": out.Groups, "hosts": out.Hosts}); errPatch != nil {
			return nil, errPatch
		}
		pService.pRouter.Emit("store.changed")
		return out.Result, nil
	})
}

// loadGroupsHosts 는 store 의 groups·hosts 를 일반 맵 배열로 읽는다 (알 수 없는 필드 보존).
func (pService *Service) loadGroupsHosts() ([]map[string]any, []map[string]any, error) {
	slGroups, errGroups := loadMapArray(pService, "groups")
	if errGroups != nil {
		return nil, nil, errGroups
	}
	slHosts, errHosts := loadMapArray(pService, "hosts")
	if errHosts != nil {
		return nil, nil, errHosts
	}
	return slGroups, slHosts, nil
}

// loadMapArray 는 store 의 최상위 배열 키를 맵 배열로 읽는다. 없으면 빈 배열.
func loadMapArray(pService *Service, szKey string) ([]map[string]any, error) {
	rawValue, bOk := pService.pStore.Get(szKey)
	slOut := []map[string]any{}
	if !bOk {
		return slOut, nil
	}
	if errUnmarshal := json.Unmarshal(rawValue, &slOut); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	return slOut, nil
}
