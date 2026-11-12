//go:build windows

package vault

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// G_bOsUnlockSupported 는 이 플랫폼에서 OS 잠금 해제(DPAPI)를 쓸 수 있는지 나타낸다.
const G_bOsUnlockSupported = true

// dpapiProtect 는 bPlain 을 현재 Windows 사용자 계정의 DPAPI 로 보호한다.
func dpapiProtect(bPlain []byte) ([]byte, error) {
	blobIn := windows.DataBlob{Size: uint32(len(bPlain))}
	if len(bPlain) > 0 {
		blobIn.Data = &bPlain[0]
	}
	var blobOut windows.DataBlob
	if errCall := windows.CryptProtectData(&blobIn, nil, nil, 0, nil, 0, &blobOut); errCall != nil {
		return nil, errCall
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(blobOut.Data)))
	return copyBlob(&blobOut), nil
}

// dpapiUnprotect 는 DPAPI 로 보호된 bCipher 를 푼다.
func dpapiUnprotect(bCipher []byte) ([]byte, error) {
	blobIn := windows.DataBlob{Size: uint32(len(bCipher))}
	if len(bCipher) > 0 {
		blobIn.Data = &bCipher[0]
	}
	var blobOut windows.DataBlob
	if errCall := windows.CryptUnprotectData(&blobIn, nil, nil, 0, nil, 0, &blobOut); errCall != nil {
		return nil, errCall
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(blobOut.Data)))
	return copyBlob(&blobOut), nil
}

// copyBlob 은 DPAPI 가 할당한 버퍼를 Go 메모리로 복사한다.
func copyBlob(pblob *windows.DataBlob) []byte {
	bOut := make([]byte, pblob.Size)
	if pblob.Size > 0 {
		copy(bOut, unsafe.Slice(pblob.Data, pblob.Size))
	}
	return bOut
}
