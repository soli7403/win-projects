package main

import "time"

// autoLockLoop 은 설정한 분 동안 입력이 없으면 볼트를 잠근다 (v1.1.0 startAutoLock 대응).
func (pstrApp *App) autoLockLoop() {
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for range t.C {
		iMin := pstrApp.store.SettingInt("autoLockMinutes")
		if iMin > 0 && pstrApp.vault.IsUnlocked() && systemIdle() >= time.Duration(iMin)*time.Minute {
			pstrApp.vault.Lock()
		}
	}
}
