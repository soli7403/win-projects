package vault

// vaultFile 은 vault.json 의 구조다 (data/vault.md §1).
type vaultFile struct {
	M_iVersion        int          `json:"version"`
	M_kdf             kdfParams    `json:"kdf"`
	M_wrapped         cipherBlob   `json:"wrapped"`
	M_recoveryKdf     *kdfParams   `json:"recoveryKdf,omitempty"`
	M_recoveryWrapped *cipherBlob  `json:"recoveryWrapped,omitempty"`
	M_szOsWrapped     string       `json:"osWrapped,omitempty"`
	M_szOsWrappedWin  string       `json:"osWrappedWin,omitempty"`
	M_slItems         []itemRecord `json:"items"`
}

// itemRecord 는 항목 하나다 (data/vault.md §4). 이름·종류·사용자 이름은 평문, 비밀은 암호문이다.
type itemRecord struct {
	M_szId         string `json:"id"`
	M_szName       string `json:"name"`
	M_szKind       string `json:"kind"`
	M_szUsername   string `json:"username,omitempty"`
	M_i64UpdatedAt int64  `json:"updatedAt"`
	cipherBlob
}

// SecretBody 는 암호화되는 비밀 객체다. 빈 값은 생략한다.
type SecretBody struct {
	M_szUsername   string `json:"username,omitempty"`
	M_szPassword   string `json:"password,omitempty"`
	M_szPrivateKey string `json:"privateKey,omitempty"`
	M_szPassphrase string `json:"passphrase,omitempty"`
}

// CredMeta 는 계정 목록용 메타 정보다 (비밀 없음, data/store.md §9).
type CredMeta struct {
	M_szId         string `json:"id"`
	M_szName       string `json:"name"`
	M_szKind       string `json:"kind"`
	M_szUsername   string `json:"username,omitempty"`
	M_i64UpdatedAt int64  `json:"updatedAt"`
}

// SaveInput 은 cred.save 의 입력이다 (data/vault.md §6 저장).
// 비밀 필드는 포인터다: nil(요청에서 빠짐 또는 null)은 기존 값 유지, ""는 그 값을 지운다.
// 이름·종류는 빈 값이면 기존 값을 유지한다.
type SaveInput struct {
	M_szId          string  `json:"id,omitempty"`
	M_szName        string  `json:"name,omitempty"`
	M_szKind        string  `json:"kind,omitempty"`
	M_pszUsername   *string `json:"username,omitempty"`
	M_pszPassword   *string `json:"password,omitempty"`
	M_pszPrivateKey *string `json:"privateKey,omitempty"`
	M_pszPassphrase *string `json:"passphrase,omitempty"`
}

// Status 는 vault.status 의 결과다.
type Status struct {
	M_bInitialized       bool   `json:"initialized"`
	M_bUnlocked          bool   `json:"unlocked"`
	M_bOsUnlockAvailable bool   `json:"osUnlockAvailable"`
	M_bOsUnlockEnabled   bool   `json:"osUnlockEnabled"`
	M_szOsUnlockKind     string `json:"osUnlockKind,omitempty"`
}
