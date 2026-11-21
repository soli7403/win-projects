# 파일 전송이 중단되면 대상 파일이 잘린 채 남는 문제 — 임시 이름에 쓰고 끝나면 이름 바꾸기

**작성**: 2026-11-20 · **상태**: 코드 수정 완료(2026-11-20, D104), 자동 테스트 통과(Windows), 실제 서버로 하는 수동 검증은 남음(§8.2) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/protocols.md](../../spec/protocols.md) §3 "파일 쓰기(임시 이름)", [spec/behavior/file-panel.md](../../spec/behavior/file-panel.md) §3·§6, [internal/sftpfs/atomic.go](../../internal/sftpfs/atomic.go), [internal/sftpfs/backend.go](../../internal/sftpfs/backend.go), [internal/sftpfs/transfer.go](../../internal/sftpfs/transfer.go), [internal/sftpfs/copyjob.go](../../internal/sftpfs/copyjob.go), [internal/sftpfs/prompt.go](../../internal/sftpfs/prompt.go), [internal/sftpfs/atomic_test.go](../../internal/sftpfs/atomic_test.go), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고쳤는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한눈에 보기

| 항목 | 내용 |
|---|---|
| 증상 | SFTP 업로드·다운로드·끌어 놓기 복사가 중간에 끊기거나 취소되면, 대상 폴더에 **중간까지만 쓰인 파일이 정상 파일과 같은 이름으로** 남는다. 같은 이름의 파일을 덮어쓰던 중이었다면 **기존 내용도 이미 사라져 있다** |
| 원인 | 대상 파일을 처음부터 **최종 이름으로 열어 내용을 비우고**(`O_TRUNC`) 썼다. 실패·취소 때 쓰던 파일을 치우는 코드도 없었다 |
| 수정 | 대상과 같은 폴더의 임시 이름 `<대상 이름>.dtpart-<8자리 16진>` 에 쓰고, **끝까지 쓴 뒤에만** 대상 이름으로 바꾼다. 실패·취소면 임시 파일을 지우고 대상 이름은 건드리지 않는다 (D104) |
| 적용 범위 | 업로드, 다운로드, 끌어 놓기 복사(서버↔서버 포함), "이 PC 앱으로 편집" 뒤 다시 올리기, 붙여넣기용 올리기 — 파일을 쓰는 네 곳 전부 |
| 바뀐 파일 | `internal/sftpfs/atomic.go`·`owner_unix.go`·`owner_other.go`·`atomic_test.go`(새로 만듦), `backend.go`·`transfer.go`·`copyjob.go`·`prompt.go`(수정) |
| 검증 | `go build ./...`, `go test ./...` 통과(Windows). 새 테스트 8개 중 6개 통과·2개는 Windows 라 건너뜀(권한 비트, 심볼릭 링크). 수정 전 동작으로 되돌려 돌리면 새 테스트 3개가 실패하는 것을 확인. `build/bin/dariterm.exe` 빌드(2026-11-20 14:40). **실제 서버에서 끊어 보는 확인은 아직 하지 않았다**(§8.2) |
| 명세 | 사용자 결정이라 [protocols.md](../../spec/protocols.md) §3 에 "파일 쓰기(임시 이름)" 항목을 넣고, [file-panel.md](../../spec/behavior/file-panel.md) §3 의 취소 문장을 고치고 수용 13 을 추가했다 |

---

## 1. 발견 경위

2026-11-20 사용자 질문 "sftp로 파일을 전송하다 중단되면 어떻게 되나?" 에 답하려고 코드를 읽어, 중단된 파일이 잘린 채 최종 이름으로 남고 이어 받기·자동 삭제가 없음을 확인했다. 이어서 사용자 지시 "임시 이름을 사용하도록 수정해줘".

---

## 2. 수정 전 구조

파일을 쓰는 곳은 네 군데였고 모두 같은 모양이었다.

| 곳 | 쓰임 | 수정 전 |
|---|---|---|
| `Service.uploadFile` ([transfer.go](../../internal/sftpfs/transfer.go)) | 업로드, 편집 뒤 다시 올리기([edit.go](../../internal/sftpfs/edit.go)), 붙여넣기용 파일 올리기 | `be.OpenWrite(szRemote)` → `io.Copy` |
| `Service.downloadFile` ([transfer.go](../../internal/sftpfs/transfer.go)) | 다운로드, 편집용 내려받기 | `os.OpenFile(szLocal, O_WRONLY\|O_CREATE\|O_TRUNC, 0600)` → `io.Copy` |
| `copyRun.copyFile` ([copyjob.go](../../internal/sftpfs/copyjob.go)) | 끌어 놓기 복사(로컬↔서버, 서버→임시 폴더→서버) | `dstBe.OpenWrite(szDst)` → `io.Copy` |
| `Service.UploadPromptBytes` ([prompt.go](../../internal/sftpfs/prompt.go)) | 클립보드 이미지·버퍼 올리기 | `be.OpenWrite(szRemote)` → `Write` |

`OpenWrite` 는 서버·이 PC 모두 "없으면 만들고, 있으면 비운다"였다.

```go
// backend.go (수정 전·후 같음)
func (be sftpBackend) OpenWrite(szPath string) (io.WriteCloser, error) {
	return be.client.OpenFile(szPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}
```

```go
// copyjob.go (수정 전)
func (pRun *copyRun) copyFile(srcBe Backend, szSrc string, dstBe Backend, szDst string) error {
	src, errOpen := srcBe.OpenRead(szSrc)
	…
	dst, errCreate := dstBe.OpenWrite(szDst)          // ← 대상 이름을 바로 열어 비움
	…
	_, errCopy := io.Copy(&copyWriter{w: dst, pRun: pRun}, src)
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy                                     // ← 실패해도 쓰던 파일은 그대로 둠
}
```

---

## 3. 원인

1. **여는 순간 기존 내용이 사라진다.** `O_TRUNC` 로 대상 이름을 열면 첫 바이트를 보내기도 전에 기존 파일의 길이가 0 이 된다. 그 뒤 어디서 끊기든 기존 내용은 되돌릴 수 없다.
2. **쓰던 파일이 최종 이름을 갖고 있다.** 중간까지 쓴 파일과 끝까지 쓴 파일이 이름으로 구분되지 않는다. 폴더를 지켜보는 다른 프로그램이나 사람이 잘린 파일을 정상 파일로 쓸 수 있다.
3. **실패·취소 뒤 정리가 없다.** 오류를 돌려줄 뿐 쓰던 파일을 지우지 않았다. 명세도 "지금 쓰던 파일은 대상에 중간까지 남을 수 있다(지우지 않는다)"였다([file-panel.md](../../spec/behavior/file-panel.md) §3, 수정 전).

구현 오류라기보다 당시 명세대로였고, 사용자 결정으로 명세를 바꾼다(§9).

| 상황 (수정 전) | 대상 폴더에 남는 것 |
|---|---|
| 새 파일을 올리다 끊김 | 잘린 파일이 최종 이름으로 |
| 기존 파일을 덮어쓰다 끊김·취소 | 잘린 새 파일. **기존 내용은 없음** |
| 서버→서버 복사 중 취소 | 이 PC 임시 폴더는 정리되지만 대상 서버의 잘린 파일은 남음 |

---

## 4. 설계 결정 (D104)

### 4.1 방식

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 같은 폴더의 임시 이름에 쓰고, 끝나면 대상 이름으로 바꾼다** | 대상 이름에는 항상 "전 내용" 아니면 "끝까지 쓴 새 내용"만 있다. 같은 폴더라 이름 바꾸기가 같은 파일시스템 안에서 일어난다 | **채택** |
| B. 그 자리에 쓰되 실패하면 지운다 | 덮어쓰기의 기존 내용은 여전히 잃는다. 연결이 끊기면 지울 수도 없다 | 기각 |
| C. 이어 받기(끊긴 곳부터 다시) | 원본이 그사이 바뀌었는지 확인할 방법(해시 등)이 따로 필요하다. 요청 범위 밖 | 하지 않음 |

### 4.2 세부 결정

1. **임시 이름은 `<대상 이름>.dtpart-<무작위 8자리 16진 소문자>`.** 뒤에 붙이므로 `*.csv` 같은 확장자로 폴더를 지켜보는 프로그램이 쓰는 중인 파일을 집지 않는다. 무작위 값이 있어 같은 파일을 동시에 두 번 보내거나 예전 임시 파일이 남아 있어도 겹치지 않는다. 숨김 파일(앞에 `.`)로 하지 않은 것은 남은 임시 파일을 사용자가 목록에서 볼 수 있게 하려는 것이다.
2. **대상 이름이 길면 임시 이름에는 200바이트까지만 넣는다**(글자 경계에서 자름). 파일 이름 한도(보통 255바이트)를 넘지 않게 한다. `200 + 16 = 216`.
3. **서버의 이름 바꾸기는 `posix-rename@openssh.com` 확장을 쓴다.** 표준 SFTP 이름 바꾸기는 OpenSSH 에서 대상이 있으면 실패한다. 이 확장은 대상이 있어도 한 번에 바꾼다. 확장이 없는 서버는 표준 이름 바꾸기를 해 보고, 실패했고 대상이 있으면 **대상을 지운 뒤** 다시 바꾼다. 이 경우에만 "대상이 잠깐 없는" 순간이 있고, 그 사이에 실패하면 임시 파일이 하나 남은 사본이므로 **지우지 않는다**.
4. **덮어쓸 때 기존 파일의 권한 비트를 유지한다.** 새 파일을 만들어 바꿔 넣으면 권한이 서버 기본값(보통 0644)으로 바뀐다. 실행 권한이 있던 스크립트나 0600 이던 비밀 파일이 "편집 → 저장"만으로 권한이 바뀌면 안 된다. 권한은 **내용을 쓰기 전에** 맞춘다(좁은 권한의 내용이 넓은 권한으로 잠시라도 놓이지 않게).
5. **다음 네 경우는 임시 이름을 쓰지 않고 전처럼 그 자리에 쓴다.** 기준은 "전에 되던 전송을 안 되게 하지 않고, 소유자·권한·링크를 바꾸지 않는다"이다.

   | 경우 | 이유 |
   |---|---|
   | 대상이 심볼릭 링크 등 일반 파일이 아니다 | 이름 바꾸기는 링크 자체를 일반 파일로 바꿔 버린다. 전에는 링크가 가리키는 파일에 썼다 |
   | 기존 파일에 소유자 쓰기 권한이 없다 | 전에는 "권한 없음"으로 실패했다. 이름 바꾸기는 폴더 권한만 있으면 되므로 쓰기 금지 파일을 조용히 덮어쓰게 된다 |
   | 폴더에 임시 파일을 만들 수 없다 | 파일에는 쓰기 권한이 있고 폴더에는 없는 경우. 전에는 됐다 |
   | 임시 파일의 소유자·그룹이 기존 파일과 다르거나 권한 비트를 맞출 수 없다 | 다른 계정 소유(그룹 쓰기 허용) 파일을 바꿔 넣으면 소유자가 내 계정으로 바뀐다 |

   이 네 경우에는 수정 전과 같은 약점(중단되면 잘림)이 그대로다.
6. **남은 임시 파일을 자동으로 치우지 않는다.** 연결이 끊겨 지우지 못한 임시 파일은 서버에 남는다. 이름 규칙만 보고 서버의 파일을 지우는 것은 위험해 넣지 않았다(§10).
7. **쓰기 한 곳으로 모은다.** 네 군데가 같은 함수(`writeReplace`)를 쓴다. 판단은 `Backend`(서버·이 PC 공통 인터페이스) 위에서 하므로 서버와 이 PC 가 같은 규칙으로 움직인다.

---

## 5. 수정 내용

### 5.1 새 파일 — `internal/sftpfs/atomic.go` (전체)

```go
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
```

`writeReplace` 의 순서:

```
대상 Lstat ─ 일반 파일 아님 / 쓰기 금지 ──────────────┐
   │                                                    │
임시 파일 만들기 ─ 실패 ───────────────────────────────┤
   │                                                    ├─▶ writeDirect (전과 같은 동작)
(기존 파일 있으면) 소유자 비교·권한 맞추기 ─ 안 됨 ────┘     (임시 파일은 지움)
   │
내용 쓰기 ─ 실패·취소 ─▶ 임시 파일 삭제, 대상은 그대로, 오류 반환
   │
닫기 ─ 실패 ─▶ 위와 같음
   │
대상 이름으로 바꾸기 ─ 실패 ─▶ 임시 파일 삭제(대상이 사라진 경우만 남김), 오류 반환
   │
  완료
```

- 읽는 쪽(`fnWrite` 안의 `io.Copy`)은 호출하는 곳이 그대로 갖고 있어 진행률·취소 처리는 바뀌지 않는다.
- 닫기(`Close`) 오류도 쓰기 실패로 본다. 서버가 마지막 쓰기를 닫을 때 거부하는 경우(디스크 가득 등)에 이름을 바꾸지 않는다.

### 5.2 새 파일 — `owner_unix.go`, `owner_other.go`

이 PC 파일의 소유자·그룹은 OS 마다 얻는 방법이 달라 build tag 로 나눈다(공통 규칙 §8 OS 중립).

```go
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
```

```go
//go:build !unix

package sftpfs

import "io/fs"

// localOwnerKey 는 소유자·그룹을 uid/gid 로 다루지 않는 OS(Windows 등)에서는 빈 값이다 (비교하지 않음).
func localOwnerKey(fs.FileInfo) string { return "" }
```

### 5.3 `backend.go` — `Backend` 에 두 가지 추가

```go
// Backend 인터페이스에 추가
	// Replace 는 이름을 바꾸되 szNew 가 이미 있으면 그것을 대신한다 (임시 이름으로 쓴 파일을 제자리에 놓을 때).
	Replace(szOld string, szNew string) error
	// OwnerKey 는 파일의 소유자·그룹을 비교용 문자열로 돌려준다. 알 수 없으면 빈 값.
	OwnerKey(info fs.FileInfo) string
```

```go
// G_cszPosixRename 은 대상이 있어도 바꾸는 이름 바꾸기 확장 이름이다 (OpenSSH).
const G_cszPosixRename = "posix-rename@openssh.com"

// Replace 는 OpenSSH 확장(posix-rename)이 있으면 그것으로 한 번에 바꾼다.
// 없는 서버는 표준 이름 바꾸기가 대상이 있을 때 실패할 수 있어, 실패하면 대상을 지우고 다시 한다.
func (be sftpBackend) Replace(szOld string, szNew string) error {
	if _, bPosix := be.client.HasExtension(G_cszPosixRename); bPosix {
		return be.client.PosixRename(szOld, szNew)
	}
	errRename := be.client.Rename(szOld, szNew)
	if errRename == nil {
		return nil
	}
	if _, errStat := be.client.Lstat(szNew); errStat != nil {
		return errRename
	}
	if errRemove := be.client.Remove(szNew); errRemove != nil {
		return errRename
	}
	return be.client.Rename(szOld, szNew)
}

// OwnerKey 는 서버가 알려 준 uid:gid 다.
func (be sftpBackend) OwnerKey(info fs.FileInfo) string {
	if pStat, bOk := info.Sys().(*sftp.FileStat); bOk {
		return fmt.Sprintf("%d:%d", pStat.UID, pStat.GID)
	}
	return ""
}
```

```go
// Replace 는 os.Rename 이다 (Windows·Unix 모두 대상 파일이 있으면 대신한다).
func (LocalBackend) Replace(szOld string, szNew string) error { return os.Rename(szOld, szNew) }
func (LocalBackend) OwnerKey(info fs.FileInfo) string         { return localOwnerKey(info) }
```

- 기존 `Rename`(패널의 "이름 바꾸기")은 그대로다. 대상이 있으면 실패하는 동작을 바꾸지 않으려고 `Replace` 를 따로 뒀다.
- `HasExtension`·`PosixRename` 은 `github.com/pkg/sftp` v1.13.12 의 공개 API 다(의존성 추가 없음).

### 5.4 쓰는 네 곳

```go
// transfer.go — before
	dst, errCreate := pService.be.OpenWrite(szRemote)
	if errCreate != nil {
		return errCreate
	}
	_, errCopy := io.Copy(dst, &progressReader{r: src, report: report})
	if errClose := dst.Close(); errCopy == nil {
		errCopy = errClose
	}
	return errCopy

// transfer.go — after (uploadFile)
	return writeReplace(pService.be, szRemote, func(dst io.Writer) error {
		_, errCopy := io.Copy(dst, &progressReader{r: src, report: report})
		return errCopy
	})
```

```go
// transfer.go — before (downloadFile)
	dst, errCreate := os.OpenFile(szLocal, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	…
// after
	return writeReplace(privateLocal{}, szLocal, func(dst io.Writer) error {
		_, errCopy := io.Copy(dst, &progressReader{r: src, report: report})
		return errCopy
	})
```

```go
// copyjob.go — after (copyFile)
	return writeReplace(dstBe, szDst, func(dst io.Writer) error {
		_, errCopy := io.Copy(&copyWriter{w: dst, pRun: pRun}, src)
		return errCopy
	})
```

```go
// prompt.go — before (UploadPromptBytes)
	dst, errCreate := pService.be.OpenWrite(szRemote)
	if errCreate != nil {
		return "", errCreate
	}
	defer dst.Close()
	if _, errWrite := dst.Write(bData); errWrite != nil {
		return "", errWrite
	}
	return szRemote, nil

// after
	errWrite := writeReplace(pService.be, szRemote, func(dst io.Writer) error {
		_, errData := dst.Write(bData)
		return errData
	})
	if errWrite != nil {
		return "", errWrite
	}
	return szRemote, nil
```

- `downloadFile` 은 새 파일을 0600 으로 만들던 것을 `privateLocal` 로 유지한다(끌어 놓기 복사의 이 PC 쪽은 전처럼 0644).
- `UploadPromptBytes` 는 닫기 오류를 버리던 것(`defer dst.Close()`)도 함께 고쳐졌다. 닫기에 실패하면 오류를 돌려준다.
- `copyjob.go` 의 `CancelCopy` 주석("쓰던 파일은 중간까지 남을 수 있다")을 새 동작에 맞게 고쳤다.
- [edit.go](../../internal/sftpfs/edit.go) 는 고치지 않았다. `uploadFile`·`downloadFile` 을 부르므로 자동으로 새 방식이 된다.

### 5.5 수정 뒤 동작 (§3 의 표와 대응)

| 상황 | 수정 뒤 대상 폴더 |
|---|---|
| 새 파일을 올리다 끊김 | 대상 이름의 파일은 **생기지 않는다**. 연결이 살아 있으면 임시 파일도 지워진다. 연결이 끊겼으면 `이름.dtpart-xxxxxxxx` 가 남는다 |
| 기존 파일을 덮어쓰다 끊김·취소 | 대상 이름의 파일은 **전 내용 그대로**. 임시 파일은 위와 같다 |
| 끝까지 전송 | 대상 이름에 새 내용, 임시 파일 없음. 덮어쓰기였다면 권한 비트는 기존 것 |
| 서버→서버 복사 중 취소 | 이 PC 임시 폴더 정리(전과 같음) + 대상 서버의 대상 이름은 그대로 |
| 전송 중 대상 폴더 목록을 새로고침 | `이름.dtpart-xxxxxxxx` 가 잠깐 보인다(크기가 늘어남). 끝나면 대상 이름으로 바뀐다 |

---

## 6. 영향 범위와 회귀 위험

| 영역 | 영향 |
|---|---|
| 전송 큐·진행률·전송 항목 상태([transfer.go](../../internal/sftpfs/transfer.go)), 복사 작업 이벤트([copyjob.go](../../internal/sftpfs/copyjob.go)) | 바뀌지 않았다. 읽기·진행 보고·취소 확인은 같은 코드다 |
| 프런트엔드, IPC 계약, 데이터 파일, 문구(`locales/`) | 바뀌지 않았다 |
| 패널의 "이름 바꾸기"(`Rename`) | 바뀌지 않았다 |
| 서버와의 왕복 횟수 | 파일마다 늘었다: 대상 `Lstat` 1번 + 이름 바꾸기 1번, 덮어쓰기면 임시 파일 `Lstat` 1번(권한이 다르면 `Chmod` 1번) 더. 작은 파일이 아주 많은 폴더는 전보다 느려질 수 있다(재 보지 않았다) |
| 대상 폴더의 빈 공간 | 덮어쓰는 동안 기존 파일과 임시 파일이 **함께** 있다. 큰 파일을 덮어쓸 때 그 크기만큼 여유가 더 필요하다 |

회귀 위험:

1. **이 PC 에서 다른 프로그램이 열어 둔 파일을 덮어쓰기**: Windows 는 열려 있는 파일 위로 이름을 바꾸지 못하는 경우가 있다(상대가 삭제 공유 없이 연 경우). 전에는 그 자리에 쓰는 것이 되던 경우도 이제 오류가 난다. 기존 파일과 그 프로그램은 영향받지 않고 임시 파일은 지워진다.
2. **덮어쓴 파일이 새 파일(새 inode)이 된다**: 하드 링크가 여럿이던 파일은 그 이름만 새 파일을 가리키게 된다. ACL·확장 속성·SELinux 문맥은 옮기지 않는다(권한 비트와 "소유자가 같은지"만 본다). 덮어쓰기 대상을 열어 두고 따라 읽는 프로그램(`tail -f` 등)은 옛 파일을 계속 본다.
3. **소유자를 알려 주지 않는 서버**: uid/gid 를 보내지 않는 서버는 비교 값이 같게 나와 소유자 확인이 걸러내지 못한다.
4. **`posix-rename` 이 없는 서버**: §4.2 의 3 대로 "지우고 바꾸기"를 한다. 이 경로는 **자동 테스트가 없다**(테스트용 서버가 확장을 항상 알린다).
5. **임시 파일 생성이 실패하면 조용히 예전 방식으로 쓴다**: 일시적인 오류로 임시 파일을 못 만든 경우에도 그 자리에 쓰므로, 그 전송은 보호받지 못한다. 사용자에게 알리지는 않는다.

---

## 7. 알아 둘 한계

- **연결이 끊기면 서버의 임시 파일은 남는다.** 지우라는 요청을 보낼 수 없기 때문이다. 다시 전송하면 새 임시 이름을 쓰므로 예전 것은 그대로 남는다. 사용자가 지워야 한다(이름에 `.dtpart-` 가 있다). 붙여넣기용 올리기 폴더의 것은 이름이 `dt-` 로 시작하므로 "오래된 업로드 자동 삭제"가 함께 지운다.
- **앱이 강제 종료되면** 이 PC·서버 모두 임시 파일이 남는다.
- **이어 받기는 없다.** 다시 전송하면 처음부터다.
- **§4.2 의 5 에 해당하는 대상**(링크, 쓰기 금지, 폴더 쓰기 권한 없음, 소유자 다름)은 수정 전과 같다.
- **내용 검증은 없다.** 끝까지 썼는지만 보고, 크기·해시를 원본과 견주지 않는다.

---

## 8. 검증

### 8.1 자동 (2026-11-20 실행, Windows)

| 명령 (`cli-manager-cr/` 에서) | 결과 |
|---|---|
| `go build ./...` | 통과 |
| `go vet ./internal/sftpfs/` | 통과 |
| `GOOS=linux go vet ./internal/sftpfs/` | 통과 (`owner_unix.go` 가 리눅스에서 컴파일되는지) |
| `go test -count=1 ./...` | 전부 통과. `internal/sftpfs` 는 19개 중 17개 통과·2개 건너뜀 |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` | `build/bin/dariterm.exe` 생성 (2026-11-20 14:40) |

`go test -race` 는 이 PC 에 C 컴파일러(gcc)가 없어 돌리지 못했다.

새 테스트 [atomic_test.go](../../internal/sftpfs/atomic_test.go) 8개. "sftp"는 메모리 파이프로 이은 테스트용 SFTP 서버, "local"은 이 PC 파일시스템이다.

| 테스트 | 확인하는 것 | 결과 |
|---|---|---|
| `TestWriteReplaceFailureKeepsTarget` (sftp·local) | 두 글자 쓰고 실패 → 기존 파일 내용 그대로, 새 파일은 안 생김, 폴더에 임시 파일 없음 | 통과 |
| `TestWriteReplaceSuccess` (sftp·local) | 덮어쓰기·새 파일 모두 새 내용, 임시 파일 없음 (서버 쪽은 `posix-rename` 경로) | 통과 |
| `TestWriteReplaceKeepsMode` (sftp·local) | 0750 파일을 덮어써도 0750 | **건너뜀**(Windows 는 권한 비트가 없음) |
| `TestWriteReplaceFallsBackToDirect` | 임시 파일을 만들 수 없을 때, 임시 파일의 소유자가 다를 때 → 실패하지 않고 그 자리에 쓰며 임시 파일이 남지 않음 | 통과 |
| `TestWriteReplaceKeepsSymlink` | 링크에 쓰면 링크는 링크로 남고 가리키는 파일이 바뀜 | **건너뜀**(이 PC 계정에 심볼릭 링크를 만들 권한이 없음) |
| `TestPartName` | 임시 이름 형식, 같은 폴더, 매번 다른 이름, 256바이트 이름도 255바이트 이하·글자 안 깨짐, `IsPartName` 판정 | 통과 |
| `TestTransferKeepsOldUntilDone` | `uploadFile` 이 원본 읽기에서 실패하면 서버의 기존 파일 그대로. 성공하면 바뀜. `downloadFile` 도 바뀌고 양쪽에 임시 파일 없음 | 통과 |
| `TestCopyCancelMidFileKeepsTarget` | 끌어 놓기 복사를 파일 쓰는 도중 취소 → 상태 `canceled`, 서버의 기존 파일 `old` 그대로, 임시 파일 없음 | 통과 |

**수정 전 동작과의 대조**: `writeReplace` 가 곧바로 `writeDirect` 를 부르게 잠깐 바꿔(= 수정 전 동작) 같은 테스트를 돌렸더니 3개가 실패했다. 확인 뒤 되돌렸다.

| 실패한 테스트 | 수정 전 동작에서 나온 값 |
|---|---|
| `TestWriteReplaceFailureKeepsTarget` (sftp·local) | 기존 파일이 `"ne"` 로 바뀜 (기대 `"old content"`) |
| `TestTransferKeepsOldUntilDone` | 서버 파일이 빈 파일이 됨 (기대 `"server old"`) |
| `TestCopyCancelMidFileKeepsTarget` | 서버 파일이 빈 파일이 됨 (기대 `"old"`) |

기존 테스트(왕복 업로드·다운로드, 서버→서버 복사, 대기 중 취소 등)는 고치지 않고 그대로 통과한다.

**자동으로 확인하지 못한 것**: 권한 비트 유지, 심볼릭 링크, 쓰기 금지 파일, `posix-rename` 없는 서버의 "지우고 바꾸기", 실제 연결 끊김(테스트는 쓰기 함수가 오류를 내는 것으로 흉내 낸다), 리눅스·macOS 에서의 실행(컴파일만 확인).

### 8.2 수동 (남음) — `build/bin/dariterm.exe` (2026-11-20 14:40 빌드)

준비: OpenSSH 서버(리눅스) 하나에 SSH 세션을 열고 SFTP 창을 연다. 수백 MB 이상의 파일 `big.bin` 과, 서버에 내용이 다른 같은 이름의 파일을 둔다(`sha256sum big.bin` 을 적어 둔다).

| # | 조작 | 기대 |
|---|---|---|
| 1 | `big.bin` 을 업로드(덮어쓰기)하는 중에 서버에서 `ls -la` | `big.bin`(전 크기 그대로)과 `big.bin.dtpart-xxxxxxxx`(커지는 중)가 함께 보인다 |
| 2 | 1 을 끝까지 둔다 | `big.bin` 이 새 내용(해시가 원본과 같음), `.dtpart-` 파일 없음 |
| 3 | 다시 덮어쓰기 업로드 중 **네트워크를 끊는다**(Wi-Fi 끄기 등) | 전송 항목이 오류. 다시 접속해 보면 `big.bin` 의 해시는 **끊기 전 것 그대로**. `.dtpart-` 파일이 하나 남아 있다(§7) — 직접 지운다 |
| 4 | 이 PC 폴더 → 서버 폴더로 끌어 놓기(덮어쓰기 확인) 중 "취소" | "취소됨". 서버의 `big.bin` 은 전 내용 그대로, `.dtpart-` 파일 없음 |
| 5 | 서버 A → 서버 B 끌어 놓기 중 "취소" | 4 와 같고 이 PC 임시 폴더도 비어 있다 |
| 6 | 서버의 `big.bin` 을 다운로드해 이 PC 의 같은 이름 파일을 덮어쓰는 중 네트워크를 끊는다 | 이 PC 의 파일은 전 내용 그대로, 같은 폴더에 `.dtpart-` 파일 없음 |
| 7 | 서버에서 `chmod 750 run.sh` 한 파일을 "이 PC 앱으로 편집" → 저장 | "저장됨 → 서버에 업로드 완료". `ls -l` 로 권한이 여전히 `rwxr-x---` |
| 8 | `chmod 600` 인 파일을 편집 → 저장 | 권한이 여전히 `rw-------` |
| 9 | 심볼릭 링크(`ln -s real.conf link.conf`)를 편집 → 저장 | `link.conf` 는 여전히 링크, `real.conf` 내용이 바뀜 |
| 10 | 다른 계정 소유이고 그룹 쓰기가 되는 파일을 편집 → 저장 | 내용이 바뀌고 소유자가 그대로(`ls -l`) |
| 11 | 폴더에는 쓰기 권한이 없고 파일에만 있는 파일을 편집 → 저장 | 전처럼 저장된다 |
| 12 | `chmod 444` 인 파일에 덮어쓰기 업로드 | 전처럼 권한 오류, 파일 내용 그대로 |
| 13 | 터미널에 이미지 붙여넣기(붙여넣기용 올리기) | 전처럼 `dt-…png` 가 올라가고 `.dtpart-` 파일 없음 |
| 14 | 이 PC 에서 다른 프로그램으로 열어 둔 파일을 다운로드로 덮어쓰기 | 오류가 날 수 있다(§6 위험 1). 열어 둔 파일은 그대로, `.dtpart-` 파일 없음 |
| 15 | 파일이 수천 개인 폴더를 업로드 | 전과 견주어 체감 속도 확인(§6 왕복 횟수) |

---

## 9. 명세

사용자 결정이라 명세에 반영했다.

- [spec/protocols.md](../../spec/protocols.md) §3: "파일 쓰기(임시 이름)" 항목 추가 — 임시 이름 형식, 끝까지 쓴 뒤에만 이름 바꾸기, 실패·취소 시 정리, 권한 유지, 그 자리에 쓰는 네 경우, 서버 이름 바꾸기 방식.
- [spec/behavior/file-panel.md](../../spec/behavior/file-panel.md) §3: "취소: 지금 쓰던 파일은 대상에 중간까지 남을 수 있다(지우지 않는다)" → "취소·실패: 임시 파일을 지우고 대상 이름의 파일은 건드리지 않는다". §6 수용 7 에 임시 파일이 남지 않음을 덧붙이고 수용 13(덮어쓰다 취소·끊김) 추가.
- [docs/decisions.md](../decisions.md): D104.

---

## 10. 남은 일

- §8.2 수동 검증. 특히 3(실제 끊김), 7~10(권한·링크·소유자)은 자동 테스트가 Windows 에서 건너뛰거나 흉내로만 확인한 부분이다.
- 리눅스·macOS 에서 `go test ./internal/sftpfs/` 실행(`TestWriteReplaceKeepsMode`, `TestWriteReplaceKeepsSymlink`).
- 남은 `.dtpart-` 파일을 치우는 방법은 정하지 않았다. 후보: 패널 목록에서 임시 파일을 흐리게 표시하고 "남은 임시 파일 지우기" 메뉴를 두는 것. 자동 삭제는 다른 세션이 지금 쓰고 있는 임시 파일을 지울 수 있어 시각 기준(예: 하루 지난 것)이 필요하다.
- `posix-rename` 이 없는 서버로의 덮어쓰기 확인(대상 서버가 있으면).
