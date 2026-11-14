//go:build windows

package license

import (
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// DomainJoined 는 이 PC 가 회사 도메인(Active Directory) 또는 Azure AD(Entra ID)에 가입돼 있는지 본다 (D78).
// 결과는 이 PC 안에서만 쓰고 어디에도 보내지 않는다. 도메인 이름도 꺼내지 않는다.
func DomainJoined() bool {
	var pName *uint16
	var uiStatus uint32
	if errJoin := windows.NetGetJoinInformation(nil, &pName, &uiStatus); errJoin == nil {
		if pName != nil {
			_ = windows.NetApiBufferFree((*byte)(unsafe.Pointer(pName)))
		}
		if uiStatus == windows.NetSetupDomainName {
			return true
		}
	}
	// Azure AD 가입: JoinInfo 아래에 가입 정보 키가 하나 이상 있다.
	key, errOpen := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CloudDomainJoin\JoinInfo`, registry.ENUMERATE_SUB_KEYS)
	if errOpen != nil {
		return false
	}
	defer key.Close()
	slNames, errRead := key.ReadSubKeyNames(1)
	return errRead == nil && len(slNames) > 0
}
