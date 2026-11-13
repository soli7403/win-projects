package vault

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const g_szTestPassword = "correct-horse-1"

func strPtr(szValue string) *string {
	return &szValue
}

// "" 명시는 그 값을 지우고, 생략(nil)은 유지한다 (data/vault.md §6). 평문 사용자 이름은 비밀 값과 같다.
func TestSaveEmptyStringClearsAndOmittedKeeps(t *testing.T) {
	pv, _ := newTestVault(t)
	if _, errSetup := pv.Setup(g_szTestPassword); errSetup != nil {
		t.Fatal(errSetup)
	}
	meta, errSave := pv.Save(SaveInput{M_szName: "h", M_szKind: "password", M_pszUsername: strPtr("u"), M_pszPassword: strPtr("p")})
	if errSave != nil {
		t.Fatal(errSave)
	}
	if _, errClear := pv.Save(SaveInput{M_szId: meta.M_szId, M_pszPassword: strPtr("")}); errClear != nil {
		t.Fatal(errClear)
	}
	body, _ := pv.Get(meta.M_szId)
	if body.M_szPassword != "" || body.M_szUsername != "u" {
		t.Fatalf("\"\" 는 지우고 생략은 유지해야 함: %+v", body)
	}
	if pv.List()[0].M_szUsername != "u" {
		t.Fatal("평문 사용자 이름은 비밀 객체의 값과 같아야 함")
	}
}

// 두 파일 모두 깨졌으면 미초기화로 시작하고 깨진 파일을 .corrupt- 이름으로 보존한다 (결정 S1).
func TestBrokenVaultIsPreservedAndTreatedAsUninitialized(t *testing.T) {
	szPath := filepath.Join(t.TempDir(), "vault.json")
	if errWrite := os.WriteFile(szPath, []byte(`{broken`), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	pv, errOpen := Open(szPath)
	if errOpen != nil {
		t.Fatalf("깨진 보관함도 열기는 성공해야 함: %v", errOpen)
	}
	if pv.Status().M_bInitialized {
		t.Fatal("깨진 보관함은 미초기화로 보아야 함")
	}
	matches, _ := filepath.Glob(szPath + ".corrupt-*")
	if len(matches) != 1 {
		t.Fatalf("깨진 파일이 보존되어야 함: %v", matches)
	}
}

func newTestVault(t *testing.T) (*Vault, string) {
	t.Helper()
	szPath := filepath.Join(t.TempDir(), "vault.json")
	pv, errOpen := Open(szPath)
	if errOpen != nil {
		t.Fatalf("Open: %v", errOpen)
	}
	return pv, szPath
}

// 볼트 설정 직후 열림, 복구 코드는 XXXX-XXXX-XXXX-XXXX-XXXX-XXXX 형식 (data/vault.md §5). 시나리오 V1.
func TestSetupCreatesOpenVaultWithRecoveryCode(t *testing.T) {
	pv, szPath := newTestVault(t)
	szCode, errSetup := pv.Setup(g_szTestPassword)
	if errSetup != nil {
		t.Fatalf("Setup: %v", errSetup)
	}
	reCode := regexp.MustCompile(`^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{4}(-[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{4}){5}$`)
	if !reCode.MatchString(szCode) {
		t.Fatalf("복구 코드 형식 오류: %q", szCode)
	}
	if !pv.Status().M_bUnlocked {
		t.Fatalf("설정 직후 보관함은 열려 있어야 함")
	}
	if pv.pfile.M_kdf.M_iN != 131072 || pv.pfile.M_kdf.M_iR != 8 || pv.pfile.M_kdf.M_iP != 1 {
		t.Fatalf("scrypt 매개변수 불일치: %+v", pv.pfile.M_kdf)
	}
	if _, errSetup2 := pv.Setup(g_szTestPassword); errSetup2 != ErrAlreadySetup {
		t.Fatalf("두 번째 Setup 오류 기대, 실제: %v", errSetup2)
	}
	_ = szPath
}

func TestSetupRejectsShortPassword(t *testing.T) {
	pv, _ := newTestVault(t)
	if _, errSetup := pv.Setup("abc"); errSetup != ErrPasswordShort {
		t.Fatalf("짧은 비밀번호 오류 기대, 실제: %v", errSetup)
	}
	if pv.Status().M_bInitialized {
		t.Fatalf("실패한 설정으로 보관함이 만들어지면 안 됨")
	}
}

// 잠금 → 틀린 비밀번호 false → 맞는 비밀번호 열림, 항목 보존. 시나리오 L7, V4.
func TestLockUnlockAndPasswordChange(t *testing.T) {
	pv, szPath := newTestVault(t)
	if _, errSetup := pv.Setup(g_szTestPassword); errSetup != nil {
		t.Fatal(errSetup)
	}
	pv.Lock()
	if pv.Unlock("wrong-password") {
		t.Fatal("틀린 비밀번호로 열리면 안 됨")
	}
	if !pv.Unlock(g_szTestPassword) {
		t.Fatal("맞는 비밀번호로 열려야 함")
	}
	bOk, errChange := pv.ChangePassword(g_szTestPassword, "new-password-2")
	if errChange != nil || !bOk {
		t.Fatalf("ChangePassword: ok=%v err=%v", bOk, errChange)
	}
	pv.Lock()
	reopened, errOpen := Open(szPath)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	if reopened.Unlock(g_szTestPassword) {
		t.Fatal("이전 비밀번호는 실패해야 함")
	}
	if !reopened.Unlock("new-password-2") {
		t.Fatal("새 비밀번호로 열려야 함")
	}
}

// 복구 코드 입력은 대문자화·하이픈 제거 후 사용, 새 비밀번호로 재설정. 시나리오 L9, L10, V5.
func TestRecoverWithLowercaseCode(t *testing.T) {
	pv, _ := newTestVault(t)
	szCode, errSetup := pv.Setup(g_szTestPassword)
	if errSetup != nil {
		t.Fatal(errSetup)
	}
	pv.Lock()
	bOk, errRecover := pv.Recover(strings.ToLower(szCode), "recovered-pw-3")
	if errRecover != nil || !bOk {
		t.Fatalf("Recover: ok=%v err=%v", bOk, errRecover)
	}
	if !pv.Status().M_bUnlocked {
		t.Fatal("복구 후 보관함은 열려야 함")
	}
	pv.Lock()
	if !pv.Unlock("recovered-pw-3") {
		t.Fatal("복구한 새 비밀번호로 열려야 함")
	}
	if bBad, _ := pv.Recover("AAAA-AAAA-AAAA-AAAA-AAAA-AAAA", "recovered-pw-3"); bBad {
		t.Fatal("틀린 복구 코드는 false 여야 함")
	}
}

// 저장 규칙: 이름·종류 평문, 비밀은 암호문, 빈 필드는 기존 값 유지. 시나리오 V2, V6, V7.
func TestCredentialSaveKeepsExistingFields(t *testing.T) {
	pv, szPath := newTestVault(t)
	if _, errSetup := pv.Setup(g_szTestPassword); errSetup != nil {
		t.Fatal(errSetup)
	}
	meta, errSave := pv.Save(SaveInput{M_szName: "prod-db", M_szKind: "password", M_pszUsername: strPtr("root"), M_pszPassword: strPtr("s3cret")})
	if errSave != nil {
		t.Fatal(errSave)
	}
	// 요청에서 빠진 비밀 필드는 유지된다 (이름만 바꿈).
	if _, errSave2 := pv.Save(SaveInput{M_szId: meta.M_szId, M_szName: "prod-db-2"}); errSave2 != nil {
		t.Fatal(errSave2)
	}
	body, errGet := pv.Get(meta.M_szId)
	if errGet != nil || body == nil {
		t.Fatalf("Get: %v", errGet)
	}
	if body.M_szPassword != "s3cret" || body.M_szUsername != "root" {
		t.Fatalf("기존 비밀이 유지되어야 함: %+v", body)
	}
	list := pv.List()
	if len(list) != 1 || list[0].M_szName != "prod-db-2" || list[0].M_szUsername != "root" {
		t.Fatalf("목록 메타 불일치: %+v", list)
	}
	bRaw, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if strings.Contains(string(bRaw), "s3cret") {
		t.Fatal("vault.json 에 비밀 평문이 있으면 안 됨")
	}
	pv.Lock()
	if _, errLocked := pv.Get(meta.M_szId); errLocked != ErrLocked {
		t.Fatalf("잠긴 보관함 조회 오류 기대, 실제: %v", errLocked)
	}
	if len(pv.List()) != 1 {
		t.Fatal("잠겨 있어도 목록은 조회되어야 함")
	}
}
