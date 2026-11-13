package gitsync

import (
	"dariterm/internal/vault"
)

// VaultAdapter 는 열린 볼트를 VaultOps 로 쓰게 하는 어댑터다.
type VaultAdapter struct {
	M_pVault *vault.Vault
}

// Locked 는 볼트가 잠겨 있는지 돌려준다.
func (adapter VaultAdapter) Locked() bool {
	return !adapter.M_pVault.Status().M_bUnlocked
}

// HeaderJSON 은 볼트 헤더(항목·OS 값 제외)를 돌려준다.
func (adapter VaultAdapter) HeaderJSON() ([]byte, error) {
	return adapter.M_pVault.HeaderJSON()
}

// EncryptPayload 는 payload 평문을 볼트 키로 암호화한다 (AAD = dariterm-sync).
func (adapter VaultAdapter) EncryptPayload(bPlain []byte) (vault.Blob, error) {
	return adapter.M_pVault.EncryptData(vault.G_cszSyncPayloadAad, bPlain)
}

// DecryptPayload 는 payload 암호문을 볼트 키로 복호화한다.
func (adapter VaultAdapter) DecryptPayload(blob vault.Blob) ([]byte, error) {
	return adapter.M_pVault.DecryptData(vault.G_cszSyncPayloadAad, blob)
}

// AdoptRemote 는 저장소 볼트를 받는다 (data/vault.md §8).
func (adapter VaultAdapter) AdoptRemote(header []byte, items []vault.ItemRecord, szPassword string) (bool, error) {
	return adapter.M_pVault.AdoptRemote(header, items, szPassword)
}
