//go:build !unix

package sftpfs

import "io/fs"

// localOwnerKey 는 소유자·그룹을 uid/gid 로 다루지 않는 OS(Windows 등)에서는 빈 값이다 (비교하지 않음).
func localOwnerKey(fs.FileInfo) string { return "" }
