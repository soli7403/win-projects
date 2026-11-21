//go:build unix

package sftpfs

import (
	"fmt"
	"io/fs"
	"syscall"
)

// localOwnerKey 는 이 PC 파일의 소유자·그룹을 "uid:gid" 로 돌려준다.
func localOwnerKey(info fs.FileInfo) string {
	if pStat, bOk := info.Sys().(*syscall.Stat_t); bOk {
		return fmt.Sprintf("%d:%d", pStat.Uid, pStat.Gid)
	}
	return ""
}
