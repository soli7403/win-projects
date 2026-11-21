package sftpfs

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"unicode/utf8"
)

// 임시 이름으로 쓰고, 다 쓰면 대상 이름으로 바꾸는 파일 쓰기 (spec/protocols.md §3 "파일 쓰기", D104).
// 전송이 중간에 끊기거나 취소돼도 대상 이름의 파일은 잘리지 않는다.

// G_cszPartMark 는 쓰는 중인 임시 파일 이름의 표시다: "<대상 이름>.dtpart-<8자리 16진>".
const G_cszPartMark = ".dtpart-"

// G_ciPartStemMax 는 임시 이름에 넣는 대상 이름의 최대 바이트 수다 (파일 이름 한도 255 를 넘지 않게).
const G_ciPartStemMax = 200

// G_cmodeKeep 은 덮어쓸 때 기존 파일에서 그대로 옮기는 권한 비트다.
const G_cmodeKeep = fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky

// IsPartName 은 쓰는 중인 임시 파일 이름인지 본다.
func IsPartName(szName string) bool {
	iMark := len(szName) - len(G_cszPartMark) - 8
	if iMark <= 0 || szName[iMark:iMark+len(G_cszPartMark)] != G_cszPartMark {
		return false
	}
	for _, ch := range szName[iMark+len(G_cszPartMark):] {
		if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

// partPath 는 szPath 와 같은 폴더의 임시 파일 경로를 만든다.
func partPath(be Backend, szPath string) string {
	szName := be.Base(szPath)
	for len(szName) > G_ciPartStemMax {
		_, nSize := utf8.DecodeLastRuneInString(szName)
		szName = szName[:len(szName)-nSize]
	}
	bRand := make([]byte, 4)
	_, _ = rand.Read(bRand)
	return be.Join(be.Dir(szPath), fmt.Sprintf("%s%s%x", szName, G_cszPartMark, bRand))
}

// writeDirect 는 대상 이름을 바로 열어(없으면 만들고, 있으면 비우고) 쓴다. 중간에 끊기면 잘린 파일이 남는다.
func writeDirect(be Backend, szPath string, fnWrite func(w io.Writer) error) error {
	dst, errCreate := be.OpenWrite(szPath)
	if errCreate != nil {
		return errCreate
	}
	errWrite := fnWrite(dst)
	if errClose := dst.Close(); errWrite == nil {
		errWrite = errClose
	}
	return errWrite
}

// writeReplace 는 szPath 에 fnWrite 가 쓰는 내용을 넣는다. 같은 폴더의 임시 이름에 쓰고, 끝까지 쓴 뒤에만 szPath 로 바꾼다.
// fnWrite 가 실패하면(연결 끊김·취소 포함) 임시 파일을 지우고 szPath 는 건드리지 않는다.
//
// 다음 경우는 전처럼 szPath 에 바로 쓴다 (writeDirect):
//   - szPath 가 일반 파일이 아니다 (링크를 일반 파일로 바꾸지 않는다)
//   - 기존 파일에 소유자 쓰기 권한이 없다 (쓰기 금지 파일을 이름 바꾸기로 우회해 덮어쓰지 않는다)
//   - 임시 파일을 만들 수 없다 (폴더에 쓰기 권한이 없고 파일에만 있는 경우 등)
//   - 임시 파일의 소유자·그룹이 기존 파일과 다르거나 권한 비트를 맞출 수 없다 (소유자·권한을 바꾸지 않는다)
func writeReplace(be Backend, szPath string, fnWrite func(w io.Writer) error) error {
	infoOld, errOld := be.Lstat(szPath)
	bExists := errOld == nil
	if bExists && (!infoOld.Mode().IsRegular() || infoOld.Mode().Perm()&0o200 == 0) {
		return writeDirect(be, szPath, fnWrite)
	}
	szPart := partPath(be, szPath)
	dst, errCreate := be.OpenWrite(szPart)
	if errCreate != nil {
		return writeDirect(be, szPath, fnWrite)
	}
	if bExists && !matchOld(be, szPart, infoOld) {
		_ = dst.Close()
		_ = be.RemoveFile(szPart)
		return writeDirect(be, szPath, fnWrite)
	}
	errWrite := fnWrite(dst)
	if errClose := dst.Close(); errWrite == nil {
		errWrite = errClose
	}
	if errWrite != nil {
		_ = be.RemoveFile(szPart)
		return errWrite
	}
	if errSwap := be.Replace(szPart, szPath); errSwap != nil {
		// 기존 파일이 사라진 채 실패했으면(덮어쓰는 이름 바꾸기가 없는 서버) 임시 파일이 하나 남은 사본이므로 지우지 않는다.
		_, errNow := be.Lstat(szPath)
		if !(bExists && errors.Is(errNow, os.ErrNotExist)) {
			_ = be.RemoveFile(szPart)
		}
		return errSwap
	}
	return nil
}

// matchOld 는 방금 만든 임시 파일이 기존 파일의 소유자·그룹과 같은지 보고, 권한 비트를 기존 파일에 맞춘다.
// 내용을 쓰기 전에 부른다 (권한이 좁은 파일의 내용이 넓은 권한으로 잠시라도 놓이지 않게).
func matchOld(be Backend, szPart string, infoOld fs.FileInfo) bool {
	infoPart, errPart := be.Lstat(szPart)
	if errPart != nil {
		return false
	}
	szOwnerOld, szOwnerPart := be.OwnerKey(infoOld), be.OwnerKey(infoPart)
	if szOwnerOld != "" && szOwnerPart != "" && szOwnerOld != szOwnerPart {
		return false
	}
	modeOld := infoOld.Mode() & G_cmodeKeep
	if infoPart.Mode()&G_cmodeKeep == modeOld {
		return true
	}
	return be.Chmod(szPart, modeOld) == nil
}

// privateLocal 은 새 파일을 소유자 전용(0600)으로 만드는 이 PC 파일시스템이다 (서버에서 받은 파일).
type privateLocal struct{ LocalBackend }

func (privateLocal) OpenWrite(szPath string) (io.WriteCloser, error) {
	return os.OpenFile(szPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
}
