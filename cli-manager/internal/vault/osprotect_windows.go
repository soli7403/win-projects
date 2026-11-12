//go:build windows

package vault

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func osProtectAvailable() bool { return true }

func osProtect(arr []byte) ([]byte, error) {
	if len(arr) == 0 {
		return nil, errors.New("빈 데이터")
	}
	in := windows.DataBlob{Size: uint32(len(arr)), Data: &arr[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func osUnprotect(arr []byte) ([]byte, error) {
	if len(arr) == 0 {
		return nil, errors.New("빈 데이터")
	}
	in := windows.DataBlob{Size: uint32(len(arr)), Data: &arr[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
