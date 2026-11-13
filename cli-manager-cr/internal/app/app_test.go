package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// IPC 경로로 볼트 설정 → 계정 저장 → 조회 → 잠금 흐름을 확인한다 (acceptance 01 L5, V2, V6).
func TestVaultAndCredThroughInvoke(t *testing.T) {
	pService, errNew := NewService(t.TempDir(), nil)
	if errNew != nil {
		t.Fatal(errNew)
	}
	pRouter := pService.Router()

	szStatus, errStatus := pRouter.Invoke("vault.status", "[]")
	if errStatus != nil || !strings.Contains(szStatus, `"initialized":false`) {
		t.Fatalf("초기 상태 불일치: %s %v", szStatus, errStatus)
	}
	szSetup, errSetup := pRouter.Invoke("vault.setup", `["correct-horse-1"]`)
	if errSetup != nil {
		t.Fatal(errSetup)
	}
	var mapSetup map[string]string
	if errJson := json.Unmarshal([]byte(szSetup), &mapSetup); errJson != nil || mapSetup["recoveryCode"] == "" {
		t.Fatalf("복구 코드 결과 불일치: %s", szSetup)
	}
	szSave, errSave := pRouter.Invoke("cred.save", `[{"name":"bastion","kind":"password","username":"ops","password":"pw"}]`)
	if errSave != nil || !strings.Contains(szSave, `"name":"bastion"`) {
		t.Fatalf("cred.save: %s %v", szSave, errSave)
	}
	var mapMeta map[string]any
	if errJson := json.Unmarshal([]byte(szSave), &mapMeta); errJson != nil {
		t.Fatal(errJson)
	}
	szId := mapMeta["id"].(string)

	szSecret, errGet := pRouter.Invoke("cred.get", `["`+szId+`"]`)
	if errGet != nil || !strings.Contains(szSecret, `"password":"pw"`) {
		t.Fatalf("cred.get: %s %v", szSecret, errGet)
	}
	if _, errLock := pRouter.Invoke("vault.lock", "[]"); errLock != nil {
		t.Fatal(errLock)
	}
	if _, errLocked := pRouter.Invoke("cred.get", `["`+szId+`"]`); errLocked == nil || errLocked.Error() != "앱이 잠겨 있습니다" {
		t.Fatalf("잠긴 조회 오류 문구 불일치: %v", errLocked)
	}
	szList, errList := pRouter.Invoke("cred.list", "[]")
	if errList != nil || !strings.Contains(szList, "bastion") {
		t.Fatalf("잠긴 상태 목록: %s %v", szList, errList)
	}
	if _, errUnlock := pRouter.Invoke("vault.unlock", `["correct-horse-1"]`); errUnlock != nil {
		t.Fatal(errUnlock)
	}
	szNull, errNull := pRouter.Invoke("cred.get", `["missing"]`)
	if errNull != nil || szNull != "null" {
		t.Fatalf("없는 항목은 null 이어야 함: %q %v", szNull, errNull)
	}
}

func TestStoreGetHasDefaults(t *testing.T) {
	pService, errNew := NewService(t.TempDir(), nil)
	if errNew != nil {
		t.Fatal(errNew)
	}
	szStore, errStore := pService.Router().Invoke("store.get", "[]")
	if errStore != nil || !strings.Contains(szStore, `"terminalTheme":"TermGo Dark"`) {
		t.Fatalf("store.get 기본값 누락: %v", errStore)
	}
}
