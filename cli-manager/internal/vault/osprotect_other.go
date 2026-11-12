//go:build !windows

package vault

import "errors"

// Windows 외 OS 는 OS 키 보관 연동 전(brief §3.12 선택 항목) — 마스터 비밀번호만 쓴다.
func osProtectAvailable() bool { return false }

func osProtect([]byte) ([]byte, error) { return nil, errors.New("지원하지 않는 OS") }

func osUnprotect([]byte) ([]byte, error) { return nil, errors.New("지원하지 않는 OS") }
