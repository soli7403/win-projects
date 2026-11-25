# Xshell 세션 가져오기 (내보내기 파일 `.xts` · 세션 폴더) — 배경·형식·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D128). 사용자의 실제 내보내기 파일(세션 164개)이 모두 읽히는 것을 확인했다. 앱 화면에서의 수동 검증(§H)도 2026-11-25 사용자 확인 완료 · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 같은 구조의 앞선 기능 SecureCRT 가져오기(D66, [spec/import.md](../../spec/import.md) §1.5). 명세 [spec/import.md](../../spec/import.md) §1.6(새로), [spec/ipc.md](../../spec/ipc.md), [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §5
**적용 순서**: 2026-11-25 의 5번째. 앞의 [3번](./2026-11-25-3-shortcut-settings.md)·[4번](./2026-11-25-4-ki009-close-reason.md)이 `locales/*.json` 을 고쳤으므로 그 뒤에 적용한다.

> 이 문서 하나로 **무엇을 만들었는지 → Xshell 파일이 어떻게 생겼는지 → 왜 그렇게 정했는지 → 코드 전체 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

서버 가져오기에 **Xshell** 을 더했다. Xshell 의 [파일 > 내보내기]로 만든 `.xts` 파일을 고르면 세션이 폴더 구조 그대로 목록에 나온다. 이 PC 에 Xshell 이 설치돼 있으면 가져오기를 열 때 자동으로 찾는다.

| | 옮긴다 | 옮기지 않는다 |
|---|---|---|
| 세션 | 이름, 폴더 구조, 프로토콜(SSH·Telnet), 호스트, 포트, 사용자 이름, 메모, 문자 인코딩, 터미널 종류(xterm 이 아닐 때), 연결 유지 간격, 자동 실행 명령 | 시리얼·RLOGIN·SFTP·FTP·로컬·RDP 세션(건너뜀) |
| 인증 | 방식(비밀번호 / 키·에이전트) | **비밀번호·암호문**(읽지 않음), **사용자 키**(목록에 안내) |
| 그 밖 | — | 포트 포워딩·프록시(목록에 안내), 글꼴·색·로그·X11 설정 |

사용자가 준 파일로 확인한 결과: 세션 파일 164개 → 후보 164개(모두 SSH), 한글 폴더 이름 포함 이름 깨짐 0건.

## 1. 사용자 요청

2026-11-25: "xshell 의 세션 설정도 import 할 수 있나?" → (없다고 답함) → "xbackup20260508.xts 파일을 참조하여 xshell 가져오기도 추가해줘."

지금까지 가져오기는 PuTTY(레지스트리), SecureCRT(세션 폴더), `~/.ssh/config`, AWS EC2 네 가지였다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 내보내기 파일 | Xshell 의 [파일 > 내보내기]가 만드는 `.xts` 파일. 실제로는 zip 압축 파일이다 |
| 세션 파일 | 세션 하나의 설정이 든 `.xsh` 파일. 글자로 된 INI 형식 |
| 세션 폴더 | 세션 파일들이 든 폴더. 그 안의 하위 폴더가 Xshell 화면의 세션 폴더다 |
| 후보 | 가져오기 화면의 목록 한 줄. 사용자가 고른 후보만 서버로 등록된다 |
| 표본 미확인 | 받은 파일에 그 값이 든 세션이 없어 실제로 확인하지 못했다는 표시 |

## 2. Xshell 파일의 형식 (받은 파일을 보고 확인한 것)

받은 파일의 실제 값(호스트·세션 이름·폴더 이름)은 이 문서와 테스트에 적지 않는다. 아래의 예는 지어낸 값이다.

### 2.1 내보내기 파일 (`.xts`)

zip 이다(맨 앞 4바이트 `PK 03 04`). 항목 429개 중 폴더 72개. 최상위는 이렇다:

```
xts.zcf                       내보내기 정보 ([SessionInfo] Version=6.0 …)
Xshell/                       ← 세션. 이것만 읽는다
  default.xshf.xfs            (세션 아님)
  web01.xsh                   최상위 세션
  LAB/DB/db01.xsh             하위 폴더 = 세션 폴더
Xftp/  Xmanager/  Xstart/     다른 프로그램의 세션
xsl/                          Xshell 의 그 밖의 설정(색 구성, 강조, 스크립트, 로그 …)
com/SECSH/HostKeys/*.pub      알려진 호스트 키 (129개)
com/SECSH/UserKeys/           사용자 키 보관소 (받은 파일에서는 비어 있음)
```

| 확인한 것 | 값 |
|---|---|
| `.xsh` 파일 | 164개, 모두 `Xshell/` 아래 |
| 폴더 깊이 | 1단 36개, 2단 107개, 3단 21개 |
| 항목 이름의 인코딩 | **UTF-8 표시가 없다.** 한글 폴더 이름은 CP949 바이트 그대로 들어 있다(이름에 한글이 든 항목 30개, 모두 CP949 로 읽힌다) |
| 경로 구분자 | `/` |

### 2.2 세션 파일 (`.xsh`)

| 확인한 것 | 값 |
|---|---|
| 인코딩 | 164개 모두 **UTF-16LE + BOM**(`FF FE`), 줄 끝 CRLF |
| 형식 | INI: `[구역]` 줄과 `키=값` 줄 |
| 판 | `[SessionInfo] Version` 이 8.1 인 것 110개, 7.1 인 것 54개. 가져오는 키는 두 판에서 같다 |
| 구역 | 21개: `CONNECTION`, `CONNECTION:AUTHENTICATION`, `CONNECTION:SSH`, `CONNECTION:KEEPALIVE`, `CONNECTION:PROXY`, `CONNECTION:SERIAL`, `CONNECTION:TELNET`, `TERMINAL`, `TERMINAL:WINDOW`, `LOGGING`, `TRANSFER` 등 |

가져오기에 쓰는 키와, 받은 파일에 든 값의 분포:

| 구역 · 키 | 받은 파일(164개)의 값 | 쓰임 |
|---|---|---|
| `CONNECTION` · `Protocol` | 모두 `SSH` | 프로토콜 |
| `CONNECTION` · `Host` | 모두 있음 | 호스트 |
| `CONNECTION` · `Port` | `22` 158개, 그 밖 6개 | 포트 |
| `CONNECTION` · `Description` | 161개에 있음 | 메모 |
| `CONNECTION:AUTHENTICATION` · `UserName` | 모두 있음 | 사용자 이름 |
| `CONNECTION:AUTHENTICATION` · `Password` | 모두 있음(암호화된 글, 52~64자) | **읽지 않는다** |
| `CONNECTION:AUTHENTICATION` · `UserKey` | 모두 비어 있음 | 사용자 키 이름(표본 미확인) |
| `CONNECTION:SSH` · `UseAuthAgent` | 모두 `0` | 에이전트 인증(표본 미확인) |
| `CONNECTION:SSH` · `RemoteCommand` | 모두 비어 있음 | 자동 실행 명령(표본 미확인) |
| `CONNECTION:SSH` · `FwdReqCount` | 모두 `0` | 포워딩 개수(표본 미확인) |
| `CONNECTION:PROXY` · `Proxy` | 모두 비어 있음 | 프록시 이름(표본 미확인) |
| `CONNECTION:KEEPALIVE` · `KeepAlive` / `KeepAliveInterval` | 모두 `1` / `60` | 연결 유지 60초 |
| `TERMINAL` · `CodePage` | 모두 `65001` | 인코딩 UTF-8 |
| `TERMINAL` · `Type` | 모두 `xterm` | 터미널 종류 |
| `CONNECTION:SERIAL` · `BaudRate` / `DataBits` / `ComPort` | 모두 `12` / `3` / `0` | **쓰지 않는다**(§3.2) |

세션 파일의 모양(지어낸 값):

```ini
[CONNECTION]
Description=랩 DB 서버
Host=10.0.0.21
Port=22
Protocol=SSH
[CONNECTION:AUTHENTICATION]
Password=…(암호화된 글)…
UserKey=
UserName=ops
[CONNECTION:KEEPALIVE]
KeepAlive=1
KeepAliveInterval=60
[TERMINAL]
CodePage=65001
Type=xterm
```

## 3. 설계 결정

### 3.1 무엇을 읽나

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 입력 ① | **내보내기 파일**(`.xts`). 화면의 [Xshell 내보내기 파일 선택] | 요청이 이 파일이다. 다른 PC 의 세션을 옮길 때 Xshell 이 주는 표준 방법 |
| 입력 ② | **이 PC 의 세션 폴더**: 가져오기를 열 때 자동 검색 | PuTTY·SecureCRT 와 같은 방식. Xshell 을 쓰던 PC 에서는 파일을 만들 필요가 없다 |
| 입력 ③ | 세션 폴더 경로(채널 인자로만) | 화면 버튼은 두지 않았다. 폴더를 복사해 온 경우는 드물고, 버튼이 하나 더 늘어난다. 백엔드는 지원하고 테스트한다 |
| 압축 풀기 | **디스크에 풀지 않고** 메모리에서 읽는다 | 남이 준 압축 파일일 수 있다. 풀지 않으면 경로를 벗어나 파일을 쓰는 문제가 생기지 않는다 |
| 한도 | 세션 파일 하나 1MB, 세션 20000개, 읽은 내용 합계 64MB. 크기는 압축 파일 머리의 표시를 믿지 않고 읽으면서 센다 | 압축 폭탄 대비. 받은 파일의 세션 파일은 하나 10KB 남짓이다 |
| 읽는 범위 | 최상위 `Xshell/` 아래의 `.xsh` 만 | `Xftp/`·`xsl/Sessions/` 등은 터미널 세션이 아니다 |
| 폴더 이름의 인코딩 | 유효한 UTF-8 이 아니면 EUC-KR(CP949) | 받은 파일이 그렇다. PuTTY·SecureCRT 가져오기와 같은 규칙. 일본어·중국어 Windows 에서 내보낸 파일은 폴더 이름이 깨질 수 있다(§I) |
| 이상한 폴더 이름 | 빈 이름·`.`·`..` 가 든 항목은 버린다 | 폴더 이름으로 쓸 수 없다 |

### 3.2 무엇을 옮기나

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 프로토콜 | `SSH`→ssh, `TELNET`→telnet. 나머지는 건너뜀 | 이 앱에 대응하는 것만 |
| **시리얼 세션** | **건너뛴다** | 시리얼 값이 목록의 번호로 적혀 있다(`BaudRate=12`, `DataBits=3`, `ComPort=0`). 번호의 뜻을 확인할 표본이 없다. 틀린 속도로 등록하느니 가져오지 않는다 |
| 포트 | 없거나 범위 밖이면 22(텔넷 23) | 다른 가져오기와 같다 |
| 메모 | `Description` → `notes` | 받은 파일의 161개에 값이 있다 |
| 인코딩 | Windows 코드 페이지 번호를 이 앱의 이름으로(65001→utf-8, 949→cp949, 932→shift_jis …). 모르는 번호는 utf-8 | 받은 파일은 65001 뿐. 나머지는 코드 페이지 번호의 공개된 뜻 |
| 터미널 종류 | `xterm`·빈 값은 옮기지 않음(기본 xterm-256color), 다른 값은 그대로 | PuTTY 가져오기와 같은 규칙. `xterm` 을 그대로 두면 256색이 줄어든다 |
| 연결 유지 | `KeepAlive` 가 켜져 있고 간격이 양수면 그 초, 아니면 30 | 받은 파일은 모두 60초 |
| **비밀번호** | **읽지 않는다** | Xshell 이 사용자·PC 정보로 암호화해 둔다. 풀려는 시도를 하지 않는다. PuTTY·SecureCRT 도 비밀번호는 옮기지 않는다 |
| **사용자 키** | 인증만 `key` 로 두고 목록에 안내 `Xshell 사용자 키 "{이름}" 은 옮기지 않음 …` | Xshell 은 키를 세션 파일이 아니라 자기 키 보관소에 둔다(세션에는 이름만). 적용 규칙(import.md §4)에 따라 키 파일이 없는 `key` 후보는 `agent` 인증으로 등록된다 |
| **포트 포워딩** | 옮기지 않고 `포트 포워딩 {n}개는 옮기지 않음` 안내 | 포워딩 항목의 형식을 볼 표본이 없다(`FwdReqCount` 가 모두 0) |
| **프록시** | 옮기지 않고 `프록시 설정 "{이름}" 은 옮기지 않음` 안내 | 세션에는 이름만 있고 내용은 다른 곳에 있다 |
| 넣을 폴더 이름 | 기본 "Xshell" | AWS·SecureCRT 와 같은 방식 |

### 3.3 화면·채널

```
가져오기 대화상자를 연다
  └ import.scan: PuTTY + SecureCRT(기본 위치) + Xshell(기본 위치, 없으면 0개) + ssh config

[Xshell 내보내기 파일 선택]
  └ import.scanXshell (인자 없음)
       └ 파일 선택 창 "Xshell 내보내기 파일(.xts) 선택"  ── 취소 → null → 아무 일 없음
       └ ScanXshell(경로)
            ├ zip 열기 실패 → 오류 → 토스트 "Xshell 파일을 읽지 못했습니다: …"
            ├ 세션 0개     → 토스트 "선택한 파일에서 Xshell 세션을 찾지 못했습니다"
            └ 후보 n개     → 목록의 Xshell 후보를 대체, 폴더 이름 "Xshell"
[n개 가져오기] → import.apply (기존 그대로: 폴더 구조 만들기, 환경 추정, 태그 xshell)
```

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 버튼 | [Xshell 내보내기 파일 선택](압축 파일 아이콘), SecureCRT 버튼 아래 | — |
| 파일 선택 창 | 종류 `*.xts` 와 모든 파일. 새 함수 `dialog.OpenFileOf`(종류를 줄 수 있는 파일 선택) | 있던 파일 선택 함수는 종류를 정할 수 없었다 |
| 적용 단계 | **고치지 않았다** | 후보의 모양이 같다(`folderPath` 는 SecureCRT 가 이미 쓴다) |
| 안내 박스 문구(`import.intro` 등 3개) | **그대로 두었다** | 10개 언어의 긴 문장을 고쳐야 한다. Xshell 설명은 버튼 옆 문구가 한다. 그래서 안내 박스에는 Xshell 이 적혀 있지 않다 |

### 3.4 영향 범위

- **저장 형식**: 바뀌지 않았다. 가져온 서버의 태그가 `xshell` 이다.
- **IPC**: 채널 `import.scanXshell` 추가. `import.scan` 결과에 Xshell 후보가 들어갈 수 있다. 후보의 `source` 에 `xshell` 추가.
- **Xshell 이 없는 PC**: 자동 검색 결과가 전과 같다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/importer/xshell.go` | 코드(Go) | **새 파일** |
| `internal/importer/xshell_windows.go`, `xshell_other.go` | 코드(Go) | **새 파일**(기본 세션 폴더 찾기) |
| `internal/importer/types.go` | 코드(Go) | 수정(출처 이름) |
| `internal/dialog/dialog_windows.go`, `dialog_other.go` | 코드(Go) | 수정(`OpenFileOf`) |
| `internal/app/channels_import.go` | 코드(Go) | 수정(채널, 자동 검색) |
| `frontend/src/screens/ImportDialog.tsx` | 코드 | 수정(버튼) |
| `frontend/src/ipc/c1types.ts`, `fakeBackend.ts` | 코드 | 수정 |
| `internal/importer/xshell_test.go` | 테스트 | **새 파일**(8개, 하나는 선택 실행) |
| `frontend/src/screens/ImportDialog.test.tsx` | 테스트 | 수정(5개 추가) |
| `locales/*.json` (10개) | 문구 | 키 11개 추가 |
| `spec/import.md`(§1.6), `spec/ipc.md`, `spec/ui/dialogs.md` | 명세 | 수정 |
| `docs/scenarios.md`, `docs/scenarios.en.md` | 문서 | 수정(10-3a) |
| `CHANGELOG.md`, `docs/decisions.md`(D128), `CLAUDE.md` | 문서 | 수정 |

## B. 코드

### `internal/importer/xshell.go` — 새 파일

```go
package importer

import (
	"archive/zip"
	"dariterm/internal/i18n"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
)

// Xshell 파일 이름·한도 (import.md §1.6).
const (
	g_cszXshExt      = ".xsh"     // 세션 파일
	g_cszXtsExt      = ".xts"     // 내보내기 파일(zip)
	g_cszXshSessions = "Sessions" // 설치된 Xshell 의 세션 폴더 이름
	g_cszXtsRoot     = "xshell"   // 내보내기 파일 안에서 세션이 든 최상위 폴더 (소문자 비교)
	g_ciXshMaxBytes  = 1 << 20    // 세션 파일 하나의 한도(넘으면 건너뜀)
	g_ciXtsMaxFiles  = 20000      // 내보내기 파일에서 읽는 세션 수 한도
	g_ciXtsMaxBytes  = 64 << 20   // 내보내기 파일에서 읽는 세션 내용의 합계 한도
	g_cszXshNoteSep  = " · "      // 후보 note 를 잇는 구분자
	g_cszXshSecConn  = "CONNECTION"
	g_cszXshSecAuth  = "CONNECTION:AUTHENTICATION"
	g_cszXshSecSsh   = "CONNECTION:SSH"
	g_cszXshSecKeep  = "CONNECTION:KEEPALIVE"
	g_cszXshSecProxy = "CONNECTION:PROXY"
	g_cszXshSecTerm  = "TERMINAL"
)

// XshSession 은 세션 파일 하나를 읽은 결과다. 이름은 파일 이름(확장자 제외), 폴더는 세션 폴더 기준 하위 폴더 이름들이다.
type XshSession struct {
	M_szName   string
	M_slFolder []string
	M_values   xshValues
}

// xshValues 는 세션 파일의 값이다. 키는 "구역 이름(대문자)\x00키 이름(소문자)".
type xshValues map[string]string

// str 은 구역·키의 값을 앞뒤 공백을 떼어 돌려준다 (없으면 빈 문자열). 구역·키 이름은 대소문자를 가리지 않는다.
func (values xshValues) str(szSection string, szKey string) string {
	return strings.TrimSpace(values[strings.ToUpper(szSection)+"\x00"+strings.ToLower(szKey)])
}

// num 은 구역·키의 값을 10진 정수로 돌려준다 (없거나 숫자가 아니면 0).
func (values xshValues) num(szSection string, szKey string) int {
	iValue, errConv := strconv.Atoi(values.str(szSection, szKey))
	if errConv != nil {
		return 0
	}
	return iValue
}

// decodeXshText 는 세션 파일 바이트를 글로 바꾼다: UTF-16(LE·BE) BOM 이면 그것으로, UTF-8 BOM 은 버리고,
// 그 밖에는 유효한 UTF-8 이면 그대로 아니면 EUC-KR(CP949)로 해석한다.
func decodeXshText(bRaw []byte) string {
	if len(bRaw) >= 2 && ((bRaw[0] == 0xFF && bRaw[1] == 0xFE) || (bRaw[0] == 0xFE && bRaw[1] == 0xFF)) {
		bLittle := bRaw[0] == 0xFF
		bBody := bRaw[2:]
		slUnits := make([]uint16, 0, len(bBody)/2)
		for iPos := 0; iPos+1 < len(bBody); iPos += 2 {
			if bLittle {
				slUnits = append(slUnits, uint16(bBody[iPos])|uint16(bBody[iPos+1])<<8)
			} else {
				slUnits = append(slUnits, uint16(bBody[iPos])<<8|uint16(bBody[iPos+1]))
			}
		}
		return string(utf16.Decode(slUnits))
	}
	if len(bRaw) >= 3 && bRaw[0] == 0xEF && bRaw[1] == 0xBB && bRaw[2] == 0xBF {
		bRaw = bRaw[3:]
	}
	return decodeLocalText(bRaw)
}

// decodeLocalText 는 유효한 UTF-8 이면 그대로, 아니면 EUC-KR(CP949)로 해석한다 (해석하지 못하면 원래 바이트 그대로).
func decodeLocalText(bRaw []byte) string {
	if utf8.Valid(bRaw) {
		return string(bRaw)
	}
	bDecoded, errDecode := korean.EUCKR.NewDecoder().Bytes(bRaw)
	if errDecode != nil {
		return string(bRaw)
	}
	return string(bDecoded)
}

// ParseXsh 는 세션 파일 내용을 값으로 바꾼다. 형식은 INI: `[구역]` 줄과 `키=값` 줄. 그 밖의 줄은 무시한다.
func ParseXsh(bRaw []byte) xshValues {
	values := xshValues{}
	szSection := ""
	for _, szLine := range strings.Split(decodeXshText(bRaw), "\n") {
		szLine = strings.TrimRight(szLine, "\r")
		szTrim := strings.TrimSpace(szLine)
		if len(szTrim) >= 2 && szTrim[0] == '[' && szTrim[len(szTrim)-1] == ']' {
			szSection = strings.ToUpper(strings.TrimSpace(szTrim[1 : len(szTrim)-1]))
			continue
		}
		iEq := strings.Index(szLine, "=")
		if iEq <= 0 {
			continue
		}
		szKey := szSection + "\x00" + strings.ToLower(strings.TrimSpace(szLine[:iEq]))
		if _, bSeen := values[szKey]; !bSeen { // 같은 키가 두 번 나오면 처음 것
			values[szKey] = szLine[iEq+1:]
		}
	}
	return values
}

// xshProtocol 은 Protocol 값을 프로토콜로 바꾼다. 가져오지 않는 것(SERIAL·RLOGIN·SFTP·FTP·LOCAL·RDP 등)은 bOk=false.
func xshProtocol(szValue string) (string, bool) {
	switch strings.ToUpper(szValue) {
	case "SSH", "":
		return "ssh", true
	case "TELNET":
		return "telnet", true
	}
	return "", false
}

// xshEncoding 은 CodePage(Windows 코드 페이지 번호)를 서버 인코딩 이름으로 바꾼다. 모르는 번호는 utf-8.
func xshEncoding(iCodePage int) string {
	switch iCodePage {
	case 949:
		return "cp949"
	case 51949:
		return "euc-kr"
	case 932:
		return "shift_jis"
	case 51932, 20932:
		return "euc-jp"
	case 936, 54936:
		return "gbk"
	case 950:
		return "big5"
	case 1252, 28591:
		return "latin1"
	}
	return "utf-8"
}

// ParseXshSession 은 세션 하나를 후보로 바꾼다. 건너뛰면 bOk=false (import.md §1.6).
// 저장된 비밀번호(Password)·암호문(Passphrase)은 읽지 않는다.
func ParseXshSession(session XshSession) (Candidate, bool) {
	values := session.M_values
	szProtocol, bOk := xshProtocol(values.str(g_cszXshSecConn, "Protocol"))
	if !bOk {
		return Candidate{}, false
	}
	szHost := values.str(g_cszXshSecConn, "Host")
	if szHost == "" {
		return Candidate{}, false
	}
	iPort := values.num(g_cszXshSecConn, "Port")
	if iPort <= 0 || iPort > 65535 {
		iPort = 22
		if szProtocol == "telnet" {
			iPort = 23
		}
	}
	host := HostDraft{
		M_szAlias:       session.M_szName,
		M_szProtocol:    szProtocol,
		M_szHost:        szHost,
		M_iPort:         iPort,
		M_szUsername:    values.str(g_cszXshSecAuth, "UserName"),
		M_szAuthType:    "password",
		M_szEncoding:    xshEncoding(values.num(g_cszXshSecTerm, "CodePage")),
		M_iKeepaliveSec: 30,
		M_szNotes:       values.str(g_cszXshSecConn, "Description"),
	}
	if szTerm := values.str(g_cszXshSecTerm, "Type"); szTerm != "" && !strings.EqualFold(szTerm, "xterm") {
		host.M_szTermType = szTerm // xterm 은 적용 기본값(xterm-256color)에 맡긴다
	}
	if values.num(g_cszXshSecKeep, "KeepAlive") != 0 {
		if iInterval := values.num(g_cszXshSecKeep, "KeepAliveInterval"); iInterval > 0 {
			host.M_iKeepaliveSec = iInterval
		}
	}

	slNotes := []string{}
	if szProtocol == "ssh" {
		// 사용자 키는 Xshell 의 키 보관소에 있어 파일로 옮길 수 없다: 인증만 키로 두고(적용 때 에이전트가 된다) 알린다.
		if szKey := values.str(g_cszXshSecAuth, "UserKey"); szKey != "" {
			host.M_szAuthType = "key"
			slNotes = append(slNotes, i18n.T("be.xsh.userKey", "name", szKey))
		} else if values.num(g_cszXshSecSsh, "UseAuthAgent") != 0 {
			host.M_szAuthType = "key"
		}
		host.M_szStartupCommand = values.str(g_cszXshSecSsh, "RemoteCommand")
		if iForwards := values.num(g_cszXshSecSsh, "FwdReqCount"); iForwards > 0 {
			slNotes = append(slNotes, i18n.T("be.xsh.forwards", "count", iForwards))
		}
	}
	if szProxy := values.str(g_cszXshSecProxy, "Proxy"); szProxy != "" {
		slNotes = append(slNotes, i18n.T("be.xsh.proxy", "name", szProxy))
	}
	return Candidate{
		M_szSource:     G_cszSourceXshell,
		M_szName:       session.M_szName,
		M_host:         host,
		M_szNote:       strings.Join(slNotes, g_cszXshNoteSep),
		M_slFolderPath: append([]string{}, session.M_slFolder...),
	}, true
}

// xshSessionOf 는 세션 폴더 기준 상대 경로(/ 구분)와 내용으로 세션을 만든다. 세션 파일이 아니거나 경로가 이상하면 bOk=false.
func xshSessionOf(szRel string, bRaw []byte) (XshSession, bool) {
	slParts := strings.Split(szRel, "/")
	szFile := slParts[len(slParts)-1]
	if len(szFile) <= len(g_cszXshExt) || !strings.EqualFold(szFile[len(szFile)-len(g_cszXshExt):], g_cszXshExt) {
		return XshSession{}, false
	}
	slFolder := []string{}
	for _, szPart := range slParts[:len(slParts)-1] {
		if szPart == "" || szPart == "." || szPart == ".." {
			return XshSession{}, false
		}
		slFolder = append(slFolder, szPart)
	}
	return XshSession{M_szName: szFile[:len(szFile)-len(g_cszXshExt)], M_slFolder: slFolder, M_values: ParseXsh(bRaw)}, true
}

// sortXshSessions 는 폴더 경로·이름의 사전순으로 정렬한다.
func sortXshSessions(slSessions []XshSession) {
	fnKey := func(session XshSession) string {
		return strings.Join(append(append([]string{}, session.M_slFolder...), session.M_szName), "\x00")
	}
	sort.SliceStable(slSessions, func(iLeft, iRight int) bool { return fnKey(slSessions[iLeft]) < fnKey(slSessions[iRight]) })
}

// ReadXshDir 는 세션 폴더 아래 .xsh 파일을 모두 읽는다. 읽지 못한 파일·폴더와 너무 큰 파일은 건너뛴다.
func ReadXshDir(szSessionsDir string) []XshSession {
	slOut := []XshSession{}
	_ = filepath.WalkDir(szSessionsDir, func(szPath string, pEntry fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			if pEntry != nil && pEntry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if pEntry.IsDir() || !strings.EqualFold(filepath.Ext(szPath), g_cszXshExt) {
			return nil
		}
		if pInfo, errInfo := pEntry.Info(); errInfo != nil || pInfo.Size() > g_ciXshMaxBytes {
			return nil
		}
		szRel, errRel := filepath.Rel(szSessionsDir, szPath)
		if errRel != nil {
			return nil
		}
		bRaw, errRead := os.ReadFile(szPath)
		if errRead != nil {
			return nil
		}
		if session, bOk := xshSessionOf(filepath.ToSlash(szRel), bRaw); bOk {
			slOut = append(slOut, session)
		}
		return nil
	})
	sortXshSessions(slOut)
	return slOut
}

// ReadXtsFile 은 Xshell 내보내기 파일(.xts, zip)에서 최상위 "Xshell" 폴더 아래의 .xsh 파일을 읽는다.
// 디스크에 풀지 않고 메모리에서 읽는다. zip 이 아니면 오류다. 한도를 넘는 파일은 건너뛴다.
func ReadXtsFile(szPath string) ([]XshSession, error) {
	pZip, errOpen := zip.OpenReader(szPath)
	if errOpen != nil {
		return nil, i18n.Errorf("be.xsh.badFile", errOpen)
	}
	defer pZip.Close()
	slOut := []XshSession{}
	iTotal := 0
	for _, pFile := range pZip.File {
		if len(slOut) >= g_ciXtsMaxFiles || iTotal >= g_ciXtsMaxBytes {
			break
		}
		if pFile.FileInfo().IsDir() || pFile.UncompressedSize64 > g_ciXshMaxBytes {
			continue
		}
		// 이름은 UTF-8 표시가 없으면 내보낸 PC 의 코드 페이지 바이트다: 유효한 UTF-8 이 아니면 EUC-KR 로 해석한다.
		szName := strings.ReplaceAll(decodeLocalText([]byte(pFile.Name)), "\\", "/")
		iSlash := strings.Index(szName, "/")
		if iSlash < 0 || !strings.EqualFold(szName[:iSlash], g_cszXtsRoot) {
			continue
		}
		szRel := szName[iSlash+1:]
		if !strings.EqualFold(filepath.Ext(szRel), g_cszXshExt) {
			continue
		}
		bRaw, errRead := readZipEntry(pFile, g_ciXshMaxBytes)
		if errRead != nil {
			continue
		}
		iTotal += len(bRaw)
		if session, bOk := xshSessionOf(szRel, bRaw); bOk {
			slOut = append(slOut, session)
		}
	}
	sortXshSessions(slOut)
	return slOut, nil
}

// readZipEntry 는 zip 항목을 iMax 바이트까지만 읽는다 (머리의 크기 표시를 믿지 않는다). 넘으면 오류다.
func readZipEntry(pFile *zip.File, iMax int) ([]byte, error) {
	pReader, errOpen := pFile.Open()
	if errOpen != nil {
		return nil, errOpen
	}
	defer pReader.Close()
	bRaw, errRead := io.ReadAll(io.LimitReader(pReader, int64(iMax)+1))
	if errRead != nil {
		return nil, errRead
	}
	if len(bRaw) > iMax {
		return nil, io.ErrUnexpectedEOF
	}
	return bRaw, nil
}

// ResolveXshDir 는 고른 폴더에서 세션 폴더를 정한다. 안에 Sessions 가 있으면 그것, 없으면 고른 폴더 자체다.
func ResolveXshDir(szPicked string) string {
	szSessions := filepath.Join(szPicked, g_cszXshSessions)
	if pInfo, errStat := os.Stat(szSessions); errStat == nil && pInfo.IsDir() {
		return szSessions
	}
	return szPicked
}

// ScanXshell 은 Xshell 세션을 후보로 바꾼다 (import.md §1.6).
// szPath 가 비면 기본 세션 폴더(없으면 빈 목록), .xts 파일이면 내보내기 파일, 폴더면 그 폴더를 읽는다.
// 경로가 없거나 내보내기 파일을 열지 못하면 오류다.
func ScanXshell(szPath string) ([]Candidate, error) {
	var slSessions []XshSession
	if szPath == "" {
		szDir := XshellSessionsDir()
		if szDir == "" {
			return []Candidate{}, nil
		}
		slSessions = ReadXshDir(szDir)
	} else {
		pInfo, errStat := os.Stat(szPath)
		if errStat != nil {
			return nil, i18n.Errorf("be.xsh.badFile", errStat)
		}
		if pInfo.IsDir() {
			slSessions = ReadXshDir(ResolveXshDir(szPath))
		} else {
			slRead, errRead := ReadXtsFile(szPath)
			if errRead != nil {
				return nil, errRead
			}
			slSessions = slRead
		}
	}
	slCands := []Candidate{}
	for _, session := range slSessions {
		if cand, bOk := ParseXshSession(session); bOk {
			slCands = append(slCands, cand)
		}
	}
	return slCands, nil
}

// xshellSessionsUnder 는 문서 폴더에서 Xshell 세션 폴더를 찾는다: 새 판부터
// `NetSarang Computer\{8,7,6}\Xshell\Sessions`, 그다음 옛 판의 `NetSarang\Xshell\Sessions`. 없으면 빈 문자열.
func xshellSessionsUnder(szDocuments string) string {
	if szDocuments == "" {
		return ""
	}
	slTries := []string{}
	for _, szVersion := range []string{"8", "7", "6"} {
		slTries = append(slTries, filepath.Join(szDocuments, "NetSarang Computer", szVersion, "Xshell", g_cszXshSessions))
	}
	slTries = append(slTries, filepath.Join(szDocuments, "NetSarang", "Xshell", g_cszXshSessions))
	for _, szTry := range slTries {
		if pInfo, errStat := os.Stat(szTry); errStat == nil && pInfo.IsDir() {
			return szTry
		}
	}
	return ""
}
```

### `internal/importer/xshell_windows.go` — 새 파일

```go
//go:build windows

package importer

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// g_cszShellFoldersKey 는 "문서" 폴더의 실제 위치가 있는 레지스트리 키다 (OneDrive 등으로 옮긴 경우를 따른다).
const g_cszShellFoldersKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`

// XshellSessionsDir 는 이 PC 의 Xshell 세션 폴더를 돌려준다. 없으면 빈 문자열 (import.md §1.6).
// "문서" 폴더는 레지스트리의 Personal 값(환경 변수를 펼친 것), 그다음 %USERPROFILE%\Documents 순으로 본다.
func XshellSessionsDir() string {
	slDocs := []string{}
	if pKey, errOpen := registry.OpenKey(registry.CURRENT_USER, g_cszShellFoldersKey, registry.QUERY_VALUE); errOpen == nil {
		szPersonal, _, errValue := pKey.GetStringValue("Personal")
		pKey.Close()
		if errValue == nil && szPersonal != "" {
			if szExpanded, errExpand := registry.ExpandString(szPersonal); errExpand == nil && szExpanded != "" {
				szPersonal = szExpanded
			}
			slDocs = append(slDocs, szPersonal)
		}
	}
	if szHome, errHome := os.UserHomeDir(); errHome == nil && szHome != "" {
		slDocs = append(slDocs, filepath.Join(szHome, "Documents"))
	}
	for _, szDocs := range slDocs {
		if szDir := xshellSessionsUnder(szDocs); szDir != "" {
			return szDir
		}
	}
	return ""
}
```

### `internal/importer/xshell_other.go` — 새 파일

```go
//go:build !windows

package importer

// XshellSessionsDir 는 이 PC 의 Xshell 세션 폴더를 돌려준다. Xshell 은 Windows 프로그램이라 그 밖의 OS 에서는 빈 문자열이다
// (내보내기 파일이나 복사한 폴더는 ScanXshell 에 경로를 주어 읽을 수 있다, import.md §1.6).
func XshellSessionsDir() string {
	return ""
}
```

### `internal/importer/types.go` — 수정

```diff
--- a/internal/importer/types.go
+++ b/internal/importer/types.go
@@ -1,4 +1,4 @@
-// Package importer 는 PuTTY·ssh config·AWS EC2 에서 서버 후보를 읽고 저장소 항목으로 적용한다 (spec/import.md).
+// Package importer 는 PuTTY·SecureCRT·Xshell·ssh config·AWS EC2 에서 서버 후보를 읽고 저장소 항목으로 적용한다 (spec/import.md).
 package importer
 
 // Forward 는 포트 포워딩 한 건이다 (data/store.md §3 forwards). id 는 적용 때 만든다.
@@ -47,14 +47,15 @@
 	M_szKeyFile string    `json:"keyFile,omitempty"`
 	M_szJumpVia string    `json:"jumpVia,omitempty"`
 	M_szNote    string    `json:"note,omitempty"`
-	// M_slFolderPath 는 넣을 폴더 아래에 만들 하위 폴더 이름들이다 (SecureCRT, import.md §1.5).
+	// M_slFolderPath 는 넣을 폴더 아래에 만들 하위 폴더 이름들이다 (SecureCRT·Xshell, import.md §1.5·§1.6).
 	M_slFolderPath []string `json:"folderPath,omitempty"`
 }
 
-// G_cszSourcePutty, G_cszSourceSecureCrt, G_cszSourceSshConfig, G_cszSourceAws 는 후보 출처 이름이다.
+// G_cszSourcePutty, G_cszSourceSecureCrt, G_cszSourceXshell, G_cszSourceSshConfig, G_cszSourceAws 는 후보 출처 이름이다.
 const (
 	G_cszSourcePutty     = "putty"
 	G_cszSourceSecureCrt = "securecrt"
+	G_cszSourceXshell    = "xshell"
 	G_cszSourceSshConfig = "ssh-config"
 	G_cszSourceAws       = "aws"
 )
```

### `internal/dialog/dialog_windows.go` — 수정

```diff
--- a/internal/dialog/dialog_windows.go
+++ b/internal/dialog/dialog_windows.go
@@ -54,6 +54,15 @@
 	return slOut[0], nil
 }
 
+// OpenFileOf 는 정한 종류의 파일 하나를 고르는 대화상자다. szFilter 는 "설명|*.확장자" 꼴(여러 개는 | 로 잇는다). 취소하면 빈 문자열.
+func OpenFileOf(szTitle string, szInitial string, szFilter string) (string, error) {
+	slOut, errRun := runSTA(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = ` + psQuote(szTitle) + `; $d.InitialDirectory = ` + psQuote(szInitial) + `; $d.Filter = ` + psQuote(szFilter) + `; if ($d.ShowDialog() -eq 'OK') { Write-Output $d.FileName }`)
+	if errRun != nil || len(slOut) == 0 {
+		return "", errRun
+	}
+	return slOut[0], nil
+}
+
 func boolPs(bValue bool) string {
 	if bValue {
 		return "$true"
```

### `internal/dialog/dialog_other.go` — 수정

```diff
--- a/internal/dialog/dialog_other.go
+++ b/internal/dialog/dialog_other.go
@@ -19,3 +19,8 @@
 
 // OpenKeyFile 는 비 Windows 에서 지원하지 않는다.
 func OpenKeyFile(szTitle string, szInitial string) (string, error) { return "", errUnsupported }
+
+// OpenFileOf 는 비 Windows 에서 지원하지 않는다.
+func OpenFileOf(szTitle string, szInitial string, szFilter string) (string, error) {
+	return "", errUnsupported
+}
```

### `internal/app/channels_import.go` — 수정

```diff
--- a/internal/app/channels_import.go
+++ b/internal/app/channels_import.go
@@ -40,6 +40,9 @@
 		szHome, _ := os.UserHomeDir()
 		slCands := importer.ScanPutty(importer.ReadPuttySessions)
 		slCands = append(slCands, importer.ScanSecureCrt("")...)
+		// 이 PC 에 설치된 Xshell 의 세션 폴더(없으면 빈 목록, import.md §1.6). 기본 폴더 읽기는 오류를 내지 않는다.
+		slXshell, _ := importer.ScanXshell("")
+		slCands = append(slCands, slXshell...)
 		slCands = append(slCands, importer.ScanSshConfig(szHome)...)
 		return slCands, nil
 	})
@@ -60,6 +63,30 @@
 		}
 		return importer.ScanSecureCrt(szDir), nil
 	})
+	pRouter.Handle("import.scanXshell", func(slArgs []json.RawMessage) (any, error) {
+		// 경로(내보내기 파일 .xts 또는 세션 폴더)를 안 주면 내보내기 파일을 고르게 한다. 취소하면 null (spec/ipc.md).
+		szPath := ""
+		if len(slArgs) > 0 {
+			szPath, _ = argString(slArgs, 0)
+		}
+		if szPath == "" {
+			szHome, _ := os.UserHomeDir()
+			szFilter := i18n.T("be.dlg.xshellFilter") + " (*.xts)|*.xts|*.*|*.*"
+			szPicked, errPick := dialog.OpenFileOf(i18n.T("be.dlg.pickXshell"), szHome, szFilter)
+			if errPick != nil {
+				return nil, errPick
+			}
+			if szPicked == "" {
+				return nil, nil
+			}
+			szPath = szPicked
+		}
+		slCands, errScan := importer.ScanXshell(szPath)
+		if errScan != nil {
+			return nil, errScan
+		}
+		return slCands, nil
+	})
 	pRouter.Handle("import.scanAws", func(slArgs []json.RawMessage) (any, error) {
 		szProfile := ""
 		if len(slArgs) > 0 {
```

### `frontend/src/ipc/c1types.ts` — 수정

```diff
--- a/frontend/src/ipc/c1types.ts
+++ b/frontend/src/ipc/c1types.ts
@@ -1,5 +1,5 @@
 // 설정·가져오기·동기화·업데이트 화면의 IPC 페이로드 타입. 근거: spec/ipc.md §2, spec/import.md §4.
-export type ImportSource = 'putty' | 'securecrt' | 'ssh-config' | 'aws';
+export type ImportSource = 'putty' | 'securecrt' | 'xshell' | 'ssh-config' | 'aws';
 
 /** 후보 안의 서버 정보(data/store.md 서버 필드의 부분 집합). spec/ipc.md "가져오기 후보 객체". 모든 필드는 없을 수 있다. */
 export interface ImportHost {
@@ -26,7 +26,7 @@
   keyFile?: string;
   jumpVia?: string;
   note?: string;
-  /** 넣을 폴더 아래에 만들 하위 폴더 이름들(SecureCRT, spec/import.md §1.5). */
+  /** 넣을 폴더 아래에 만들 하위 폴더 이름들(SecureCRT·Xshell, spec/import.md §1.5·§1.6). */
   folderPath?: string[];
 }
 
```

### `frontend/src/ipc/fakeBackend.ts` — 수정

```diff
--- a/frontend/src/ipc/fakeBackend.ts
+++ b/frontend/src/ipc/fakeBackend.ts
@@ -258,6 +258,9 @@
       { source: 'putty', name: 'serial-lab', host: { alias: 'serial-lab', protocol: 'serial', host: '', port: 0, authType: 'password', serial: { path: '/dev/ttyUSB0', baudRate: 115200 } } },
       { source: 'securecrt', name: 'gw_tunneling', folderPath: ['lab', 'tunnel'], host: { alias: 'gw_tunneling', protocol: 'ssh', host: '203.0.113.5', port: 22, username: 'ops', authType: 'password', keepaliveSec: 30 } },
     ],
+    'import.scanXshell': () => [
+      { source: 'xshell', name: 'db01', folderPath: ['LAB', 'DB'], host: { alias: 'db01', protocol: 'ssh', host: '10.0.0.21', port: 22, username: 'ops', authType: 'password', keepaliveSec: 60, notes: 'lab db' } },
+    ],
     'import.scanSecureCrt': () => [
       { source: 'securecrt', name: 'web01', folderPath: ['MDS'], host: { alias: 'web01', protocol: 'ssh', host: '10.0.0.13', port: 22, username: 'ops', authType: 'password', keepaliveSec: 30 } },
     ],
```

### `frontend/src/screens/ImportDialog.tsx` — 수정

```diff
--- a/frontend/src/screens/ImportDialog.tsx
+++ b/frontend/src/screens/ImportDialog.tsx
@@ -1,6 +1,6 @@
 // 서버 가져오기 대화상자. 근거: spec/ui/dialogs.md §5, spec/import.md §1.5·§4, spec/acceptance I1·I6·I9.
 import { useEffect, useMemo, useState } from 'react';
-import { Download, KeyRound, Cloud, FolderOpen } from 'lucide-react';
+import { Download, KeyRound, Cloud, FolderOpen, FileArchive } from 'lucide-react';
 import { Modal } from '../ui/Modal';
 import { Field } from '../ui/Controls';
 import { getBackend, errorMessage } from '../ipc/backend';
@@ -12,7 +12,7 @@
 
 type Cand = ImportCandidate & { key: string };
 
-const SOURCE_LABEL: Record<string, string> = { putty: 'PuTTY', securecrt: 'SecureCRT', 'ssh-config': 'ssh config', aws: 'AWS' };
+const SOURCE_LABEL: Record<string, string> = { putty: 'PuTTY', securecrt: 'SecureCRT', xshell: 'Xshell', 'ssh-config': 'ssh config', aws: 'AWS' };
 
 /** 후보의 서버 정보. host 가 빠져 있어도 예외 없이 빈 객체로 본다(spec/ipc.md "가져오기 후보 객체"). */
 export function hostOf(c: Partial<ImportCandidate> | null | undefined): ImportHost {
@@ -57,6 +57,7 @@
   const [scanning, setScanning] = useState(true);
   const [awsBusy, setAwsBusy] = useState(false);
   const [crtBusy, setCrtBusy] = useState(false);
+  const [xshBusy, setXshBusy] = useState(false);
   const [applying, setApplying] = useState(false);
 
   // 후보 자동 검색. 초기 선택: 이미 있는 서버를 뺀 나머지 전부.
@@ -70,6 +71,8 @@
         setCands(withKey);
         // PuTTY 후보 없이 SecureCRT 후보만 있으면 기본 폴더 이름을 SecureCRT 로 (dialogs.md §5).
         if (!withKey.some((c) => c.source === 'putty') && withKey.some((c) => c.source === 'securecrt')) setFolder('SecureCRT');
+        // PuTTY·SecureCRT 후보가 없고 Xshell 후보만 있으면 "Xshell" (dialogs.md §5).
+        else if (!withKey.some((c) => c.source === 'putty' || c.source === 'securecrt') && withKey.some((c) => c.source === 'xshell')) setFolder('Xshell');
         setSel(new Set(withKey.filter((c) => !sameHostExists(c, useApp.getState().store?.hosts ?? [])).map((c) => c.key)));
       })
       .catch((e) => pushToast({ kind: 'error', text: errorMessage(e) }))
@@ -142,6 +145,24 @@
     }
   };
 
+  // Xshell 내보내기 파일(.xts) 선택: 취소(null)면 아무것도 안 함. 결과는 Xshell 후보를 대체, 폴더 이름은 "Xshell".
+  const scanXshell = async () => {
+    setXshBusy(true);
+    try {
+      const found = await getBackend().invoke<ImportCandidate[] | null>('import.scanXshell', []);
+      if (found === null || found === undefined) return;
+      if (!Array.isArray(found) || found.length === 0) {
+        pushToast({ kind: 'info', text: t('import.xshNone') });
+        return;
+      }
+      replaceSource('xshell', found, 'Xshell');
+    } catch (e) {
+      pushToast({ kind: 'error', text: t('import.xshFailed', { error: errorMessage(e) }) });
+    } finally {
+      setXshBusy(false);
+    }
+  };
+
   const chosen = useMemo(() => cands.filter((c) => sel.has(c.key)), [cands, sel]);
 
   const apply = async () => {
@@ -201,6 +222,13 @@
         <span className="set-note">{t('import.crtNote')}</span>
       </div>
 
+      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
+        <button className="btn" type="button" disabled={xshBusy} onClick={() => void scanXshell()} style={{ flex: 'none' }}>
+          <FileArchive size={14} /> {xshBusy ? t('import.xshBusy') : t('import.xsh')}
+        </button>
+        <span className="set-note">{t('import.xshNote')}</span>
+      </div>
+
       {scanning && <div className="faint">{t('import.scanning')}</div>}
       {!scanning && cands.length === 0 && (
         <div>
```


## C. 테스트 코드

### `internal/importer/xshell_test.go` — 새 파일

```go
package importer

import (
	"archive/zip"
	"bytes"
	"dariterm/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/korean"
)

// xshBytes 는 Xshell 이 쓰는 모양(UTF-16LE + BOM, CRLF)으로 세션 파일 내용을 만든다.
func xshBytes(slLines ...string) []byte {
	slUnits := utf16.Encode([]rune(strings.Join(slLines, "\r\n") + "\r\n"))
	bOut := []byte{0xFF, 0xFE}
	for _, uUnit := range slUnits {
		bOut = append(bOut, byte(uUnit), byte(uUnit>>8))
	}
	return bOut
}

// xshSample 은 실제 세션 파일에서 본 구역·키 모양을 흉내 낸 SSH 세션이다(값은 지어낸 것).
func xshSample(szHost string, slExtra ...string) []byte {
	slLines := []string{
		"[CONNECTION:PROXY]", "Proxy=", "StartUp=0",
		"[CONNECTION:SERIAL]", "BaudRate=12", "ComPort=0", "DataBits=3",
		"[SessionInfo]", "Version=8.1", "Description=Xshell session file",
		"[CONNECTION:SSH]", "AgentForwarding=0", "FwdReqCount=0", "RemoteCommand=", "UseAuthAgent=0",
		"[CONNECTION]", "Description=랩 DB 서버", "Host=" + szHost, "Port=22", "Protocol=SSH",
		"[TERMINAL]", "CodePage=65001", "Type=xterm",
		"[CONNECTION:AUTHENTICATION]", "AuthMethodList=01,10,20,30", "Password=AAAAsecretAAAA", "Passphrase=", "UserKey=", "UserName=ops",
		"[CONNECTION:KEEPALIVE]", "KeepAlive=1", "KeepAliveInterval=60",
	}
	return xshBytes(append(slLines, slExtra...)...)
}

func useKoreanImport(t *testing.T) {
	t.Helper()
	szBefore := i18n.Lang()
	i18n.SetLang("ko")
	t.Cleanup(func() { i18n.SetLang(szBefore) })
}

// 세션 파일 해석: UTF-16LE·UTF-16BE·UTF-8(BOM)·CP949, 구역·키는 대소문자를 가리지 않고, 같은 키는 처음 것 (import.md §1.6).
func TestParseXshEncodingsAndSections(t *testing.T) {
	const szText = "[Connection]\r\nHost=10.0.0.1\r\nDescription=운영 서버\r\nhost=ignored\r\n; 주석\r\n[CONNECTION:AUTHENTICATION]\r\nUserName= root \r\n"
	slUnits := utf16.Encode([]rune(szText))
	bLe, bBe := []byte{0xFF, 0xFE}, []byte{0xFE, 0xFF}
	for _, uUnit := range slUnits {
		bLe = append(bLe, byte(uUnit), byte(uUnit>>8))
		bBe = append(bBe, byte(uUnit>>8), byte(uUnit))
	}
	bCp949, errEncode := korean.EUCKR.NewEncoder().Bytes([]byte(szText))
	if errEncode != nil {
		t.Fatal(errEncode)
	}
	for szName, bRaw := range map[string][]byte{
		"utf16le": bLe, "utf16be": bBe, "utf8bom": append([]byte{0xEF, 0xBB, 0xBF}, szText...), "utf8": []byte(szText), "cp949": bCp949,
	} {
		values := ParseXsh(bRaw)
		if values.str("CONNECTION", "Host") != "10.0.0.1" || values.str("connection", "HOST") != "10.0.0.1" {
			t.Fatalf("%s: Host %q", szName, values.str("CONNECTION", "Host"))
		}
		if values.str("CONNECTION", "Description") != "운영 서버" {
			t.Fatalf("%s: Description %q", szName, values.str("CONNECTION", "Description"))
		}
		if values.str("CONNECTION:AUTHENTICATION", "UserName") != "root" {
			t.Fatalf("%s: UserName %q", szName, values.str("CONNECTION:AUTHENTICATION", "UserName"))
		}
	}
}

// 세션 → 후보: 호스트·포트·사용자·메모·인코딩·연결 유지·폴더, 비밀번호는 어디에도 들어가지 않는다 (import.md §1.6).
func TestParseXshSessionFields(t *testing.T) {
	useKoreanImport(t)
	session, bOk := xshSessionOf("LAB/DB/db01.xsh", xshSample("10.0.0.21"))
	if !bOk {
		t.Fatal("세션 파일로 봐야 함")
	}
	cand, bOk := ParseXshSession(session)
	if !bOk {
		t.Fatal("후보가 돼야 함")
	}
	host := cand.M_host
	if cand.M_szSource != "xshell" || cand.M_szName != "db01" || host.M_szAlias != "db01" {
		t.Fatalf("이름·출처: %+v", cand)
	}
	if strings.Join(cand.M_slFolderPath, "/") != "LAB/DB" {
		t.Fatalf("폴더: %q", cand.M_slFolderPath)
	}
	if host.M_szProtocol != "ssh" || host.M_szHost != "10.0.0.21" || host.M_iPort != 22 || host.M_szUsername != "ops" {
		t.Fatalf("접속 정보: %+v", host)
	}
	if host.M_szAuthType != "password" || cand.M_szKeyFile != "" || cand.M_szNote != "" {
		t.Fatalf("인증: %+v / %+v", host, cand)
	}
	if host.M_szEncoding != "utf-8" || host.M_szTermType != "" || host.M_iKeepaliveSec != 60 || host.M_szNotes != "랩 DB 서버" {
		t.Fatalf("그 밖: %+v", host)
	}
	if host.M_szStartupCommand != "" || len(host.M_slForwards) != 0 || host.M_pSerial != nil {
		t.Fatalf("없어야 하는 값: %+v", host)
	}
	if bJson := strings.ToLower(strings.Join([]string{cand.M_szNote, host.M_szNotes, host.M_szUsername, host.M_szHost}, "|")); strings.Contains(bJson, "secret") {
		t.Fatal("비밀번호가 후보에 들어가면 안 됨")
	}
}

// 값에 따른 갈래: 텔넷 기본 포트, 코드 페이지, 터미널 종류, 연결 유지 꺼짐, 사용자 키·에이전트·포워딩·프록시 안내.
func TestParseXshSessionVariants(t *testing.T) {
	useKoreanImport(t)
	fnCand := func(slLines ...string) (Candidate, bool) {
		session, _ := xshSessionOf("s.xsh", xshBytes(slLines...))
		return ParseXshSession(session)
	}
	cand, bOk := fnCand("[CONNECTION]", "Host=sw1", "Port=0", "Protocol=TELNET", "[TERMINAL]", "CodePage=949", "Type=vt100", "[CONNECTION:KEEPALIVE]", "KeepAlive=0", "KeepAliveInterval=60")
	if !bOk || cand.M_host.M_szProtocol != "telnet" || cand.M_host.M_iPort != 23 || cand.M_host.M_szEncoding != "cp949" || cand.M_host.M_szTermType != "vt100" || cand.M_host.M_iKeepaliveSec != 30 {
		t.Fatalf("텔넷: %+v", cand.M_host)
	}
	for iCodePage, szWant := range map[string]string{"65001": "utf-8", "51949": "euc-kr", "932": "shift_jis", "51932": "euc-jp", "936": "gbk", "950": "big5", "1252": "latin1", "12345": "utf-8", "": "utf-8"} {
		c, _ := fnCand("[CONNECTION]", "Host=h", "Protocol=SSH", "[TERMINAL]", "CodePage="+iCodePage)
		if c.M_host.M_szEncoding != szWant {
			t.Fatalf("코드 페이지 %q → %q (기대 %q)", iCodePage, c.M_host.M_szEncoding, szWant)
		}
	}
	cand, _ = fnCand("[CONNECTION]", "Host=h", "Port=2222", "Protocol=SSH",
		"[CONNECTION:AUTHENTICATION]", "UserKey=id_lab", "Password=zzz",
		"[CONNECTION:SSH]", "FwdReqCount=2", "RemoteCommand=tmux attach",
		"[CONNECTION:PROXY]", "Proxy=corp")
	if cand.M_host.M_iPort != 2222 || cand.M_host.M_szAuthType != "key" || cand.M_szKeyFile != "" || cand.M_host.M_szStartupCommand != "tmux attach" {
		t.Fatalf("키 세션: %+v / %+v", cand.M_host, cand)
	}
	const szWantNote = `Xshell 사용자 키 "id_lab" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정) · 포트 포워딩 2개는 옮기지 않음 · 프록시 설정 "corp" 은 옮기지 않음`
	if cand.M_szNote != szWantNote {
		t.Fatalf("안내:\n%q\n기대:\n%q", cand.M_szNote, szWantNote)
	}
	cand, _ = fnCand("[CONNECTION]", "Host=h", "Protocol=SSH", "[CONNECTION:SSH]", "UseAuthAgent=1")
	if cand.M_host.M_szAuthType != "key" || cand.M_szNote != "" {
		t.Fatalf("에이전트: %+v", cand)
	}
	// 건너뛰는 것: 호스트가 빈 세션, 가져오지 않는 프로토콜.
	if _, bSkip := fnCand("[CONNECTION]", "Host=", "Protocol=SSH"); bSkip {
		t.Fatal("호스트가 비면 건너뛰어야 함")
	}
	for _, szProtocol := range []string{"SERIAL", "RLOGIN", "SFTP", "FTP", "LOCAL", "RDP"} {
		if _, bSkip := fnCand("[CONNECTION]", "Host=h", "Protocol="+szProtocol); bSkip {
			t.Fatalf("%s 는 건너뛰어야 함", szProtocol)
		}
	}
}

// 세션 폴더 읽기: 하위 폴더 구조, .xsh 만, 사전순. 고른 폴더 안에 Sessions 가 있으면 그것을 읽는다.
func TestScanXshellFolder(t *testing.T) {
	szRoot := t.TempDir()
	fnWrite := func(szRel string, bData []byte) {
		szPath := filepath.Join(szRoot, filepath.FromSlash(szRel))
		if errMkdir := os.MkdirAll(filepath.Dir(szPath), 0o755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		if errWrite := os.WriteFile(szPath, bData, 0o600); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	fnWrite("Sessions/web01.xsh", xshSample("10.0.0.1"))
	fnWrite("Sessions/LAB/DB/db01.XSH", xshSample("10.0.0.21"))
	fnWrite("Sessions/LAB/app01.xsh", xshSample("10.0.0.11"))
	fnWrite("Sessions/LAB/readme.txt", []byte("not a session"))
	fnWrite("Sessions/default.xshf.xfs", xshSample("ignored"))
	fnWrite("outside.xsh", xshSample("10.9.9.9")) // Sessions 밖: 읽지 않는다

	for _, szPick := range []string{szRoot, filepath.Join(szRoot, "Sessions")} {
		slCands, errScan := ScanXshell(szPick)
		if errScan != nil {
			t.Fatal(errScan)
		}
		slGot := []string{}
		for _, cand := range slCands {
			slGot = append(slGot, strings.Join(append(append([]string{}, cand.M_slFolderPath...), cand.M_szName), "/")+"="+cand.M_host.M_szHost)
		}
		if szGot := strings.Join(slGot, " "); szGot != "LAB/DB/db01=10.0.0.21 LAB/app01=10.0.0.11 web01=10.0.0.1" {
			t.Fatalf("%s: %s", szPick, szGot)
		}
	}
	if _, errScan := ScanXshell(filepath.Join(szRoot, "nope")); errScan == nil {
		t.Fatal("없는 경로는 오류여야 함")
	}
}

// writeXts 는 내보내기 파일을 만든다. 항목 이름은 준 바이트 그대로(UTF-8 표시 없이) 넣는다.
func writeXts(t *testing.T, mapFiles map[string][]byte, slOrder []string) string {
	t.Helper()
	var buf bytes.Buffer
	pWriter := zip.NewWriter(&buf)
	for _, szName := range slOrder {
		pEntry, errCreate := pWriter.CreateHeader(&zip.FileHeader{Name: szName, Method: zip.Deflate, NonUTF8: true})
		if errCreate != nil {
			t.Fatal(errCreate)
		}
		if _, errWrite := pEntry.Write(mapFiles[szName]); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	if errClose := pWriter.Close(); errClose != nil {
		t.Fatal(errClose)
	}
	szPath := filepath.Join(t.TempDir(), "export.xts")
	if errWrite := os.WriteFile(szPath, buf.Bytes(), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	return szPath
}

// 내보내기 파일(.xts): 최상위 Xshell 폴더 아래의 .xsh 만 읽고, CP949 로 적힌 폴더 이름을 한글로 읽는다 (import.md §1.6).
func TestScanXshellExportFile(t *testing.T) {
	bKoreanDir, errEncode := korean.EUCKR.NewEncoder().Bytes([]byte("운영 센터"))
	if errEncode != nil {
		t.Fatal(errEncode)
	}
	szKoreanDir := string(bKoreanDir)
	mapFiles := map[string][]byte{
		"xts.zcf":                             []byte("[SessionInfo]\r\nVersion=6.0\r\n"),
		"Xshell/default.xshf.xfs":             xshSample("ignored"),
		"Xshell/web01.xsh":                    xshSample("10.0.0.1"),
		"Xshell/LAB/DB/db01.xsh":              xshSample("10.0.0.21"),
		"Xshell/" + szKoreanDir + "/gw.xsh":   xshSample("10.0.0.254"),
		"Xshell/../evil.xsh":                  xshSample("10.6.6.6"), // 폴더 이름이 이상한 항목: 버린다
		"Xftp/files.xsh":                      xshSample("10.7.7.7"), // 다른 프로그램 폴더: 읽지 않는다
		"xsl/Sessions/note.xsh":               xshSample("10.8.8.8"),
		"com/SECSH/HostKeys/key_10.0.0.1.pub": []byte("ssh-rsa AAAA"),
		"Xshell/big.xsh":                      append(xshSample("10.5.5.5"), make([]byte, g_ciXshMaxBytes)...), // 한도 초과: 건너뛴다
	}
	slOrder := []string{"xts.zcf", "Xshell/default.xshf.xfs", "Xshell/web01.xsh", "Xshell/LAB/DB/db01.xsh", "Xshell/" + szKoreanDir + "/gw.xsh", "Xshell/../evil.xsh", "Xftp/files.xsh", "xsl/Sessions/note.xsh", "com/SECSH/HostKeys/key_10.0.0.1.pub", "Xshell/big.xsh"}
	slCands, errScan := ScanXshell(writeXts(t, mapFiles, slOrder))
	if errScan != nil {
		t.Fatal(errScan)
	}
	slGot := []string{}
	for _, cand := range slCands {
		slGot = append(slGot, strings.Join(append(append([]string{}, cand.M_slFolderPath...), cand.M_szName), "/")+"="+cand.M_host.M_szHost)
	}
	if szGot := strings.Join(slGot, " | "); szGot != "LAB/DB/db01=10.0.0.21 | web01=10.0.0.1 | 운영 센터/gw=10.0.0.254" {
		t.Fatalf("후보: %s", szGot)
	}
}

// zip 이 아닌 파일은 오류다.
func TestScanXshellBadFile(t *testing.T) {
	useKoreanImport(t)
	szPath := filepath.Join(t.TempDir(), "x.xts")
	if errWrite := os.WriteFile(szPath, []byte("not a zip"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	_, errScan := ScanXshell(szPath)
	if errScan == nil || !strings.HasPrefix(errScan.Error(), "Xshell 내보내기 파일을 열지 못했습니다: ") {
		t.Fatalf("오류: %v", errScan)
	}
}

// 문서 폴더에서 세션 폴더 찾기: 새 판(8)이 먼저, 옛 위치는 마지막. 없으면 빈 문자열.
func TestXshellSessionsUnder(t *testing.T) {
	szDocs := t.TempDir()
	if szDir := xshellSessionsUnder(szDocs); szDir != "" {
		t.Fatalf("없으면 빈 문자열: %q", szDir)
	}
	szOld := filepath.Join(szDocs, "NetSarang", "Xshell", "Sessions")
	sz7 := filepath.Join(szDocs, "NetSarang Computer", "7", "Xshell", "Sessions")
	sz8 := filepath.Join(szDocs, "NetSarang Computer", "8", "Xshell", "Sessions")
	for _, tc := range []struct{ szMake, szWant string }{{szOld, szOld}, {sz7, sz7}, {sz8, sz8}} {
		if errMkdir := os.MkdirAll(tc.szMake, 0o755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		if szDir := xshellSessionsUnder(szDocs); szDir != tc.szWant {
			t.Fatalf("%q 를 만든 뒤: %q", tc.szMake, szDir)
		}
	}
	if szDir := xshellSessionsUnder(""); szDir != "" {
		t.Fatalf("문서 폴더를 모르면 빈 문자열: %q", szDir)
	}
}

// 실제 내보내기 파일로 맞대어 보기(선택): 환경 변수 DARITERM_XTS 에 .xts 경로를 주면 읽어 수만 적는다.
// 파일의 내용(호스트·이름)은 출력하지 않는다. 변수가 없으면 건너뛴다.
func TestXshellRealExport(t *testing.T) {
	szPath := os.Getenv("DARITERM_XTS")
	if szPath == "" {
		t.Skip("DARITERM_XTS 가 없음")
	}
	slSessions, errRead := ReadXtsFile(szPath)
	if errRead != nil {
		t.Fatal(errRead)
	}
	slCands, errScan := ScanXshell(szPath)
	if errScan != nil {
		t.Fatal(errScan)
	}
	mapProtocol, mapEncoding, mapDepth, mapAuth := map[string]int{}, map[string]int{}, map[int]int{}, map[string]int{}
	iUser, iNotes, iNote, iBadName, iKeep60 := 0, 0, 0, 0, 0
	for _, cand := range slCands {
		mapProtocol[cand.M_host.M_szProtocol]++
		mapEncoding[cand.M_host.M_szEncoding]++
		mapDepth[len(cand.M_slFolderPath)]++
		mapAuth[cand.M_host.M_szAuthType]++
		if cand.M_host.M_szUsername != "" {
			iUser++
		}
		if cand.M_host.M_szNotes != "" {
			iNotes++
		}
		if cand.M_szNote != "" {
			iNote++
		}
		if cand.M_host.M_iKeepaliveSec == 60 {
			iKeep60++
		}
		for _, szPart := range append(append([]string{}, cand.M_slFolderPath...), cand.M_szName) {
			if strings.ContainsRune(szPart, '�') || strings.ContainsAny(szPart, "¼¿") {
				iBadName++
			}
		}
		if cand.M_host.M_szHost == "" || cand.M_host.M_iPort <= 0 {
			t.Fatalf("호스트·포트가 빈 후보가 있음 (%d번째)", len(mapProtocol))
		}
	}
	t.Logf("세션 파일 %d개 → 후보 %d개, 프로토콜 %v, 인코딩 %v, 폴더 깊이 %v, 인증 %v", len(slSessions), len(slCands), mapProtocol, mapEncoding, mapDepth, mapAuth)
	t.Logf("사용자 이름 있음 %d, 메모 있음 %d, 옮기지 않음 안내 %d, 연결 유지 60초 %d, 깨진 이름 %d", iUser, iNotes, iNote, iKeep60, iBadName)
	if iBadName != 0 {
		t.Fatal("폴더·세션 이름이 깨짐")
	}
}
```

### `frontend/src/screens/ImportDialog.test.tsx` — 수정

```diff
--- a/frontend/src/screens/ImportDialog.test.tsx
+++ b/frontend/src/screens/ImportDialog.test.tsx
@@ -30,13 +30,17 @@
   { source: 'putty', name: 'broken-entry' },
 ];
 
-function mockBackend(scan: unknown, crt: unknown = null) {
+function mockBackend(scan: unknown, crt: unknown = null, xsh: unknown = null) {
   const calls: [string, unknown[]][] = [];
   const b: Backend = {
     async invoke(channel: string, args: unknown[] = []): Promise<any> {
       calls.push([channel, args]);
       if (channel === 'import.scan') return scan;
       if (channel === 'import.scanSecureCrt') return crt;
+      if (channel === 'import.scanXshell') {
+        if (xsh instanceof Error) throw xsh;
+        return xsh;
+      }
       if (channel === 'import.apply') return { hosts: (args[0] as unknown[]).length, keys: 0, skipped: [] };
       if (channel === 'store.get') return structuredClone(useApp.getState().store);
       if (channel === 'vault.status') return { initialized: true, unlocked: true, osUnlockAvailable: false, osUnlockEnabled: false };
@@ -187,3 +191,75 @@
     expect(sameHostExists(c, [{ host: '10.1.1.2', port: 2222, username: 'ubuntu' }])).toBe(false);
   });
 });
+
+// Xshell 내보내기 파일(.xts) 가져오기 (D128, spec/ui/dialogs.md §5, spec/import.md §1.6).
+describe('가져오기 — Xshell 내보내기 파일', () => {
+  const XSH = [
+    { source: 'xshell', name: 'db01', folderPath: ['LAB', 'DB'], host: { alias: 'db01', protocol: 'ssh', host: '10.0.0.21', port: 22, username: 'ops', authType: 'password', keepaliveSec: 60 } },
+    { source: 'xshell', name: 'gw', host: { alias: 'gw', protocol: 'ssh', host: '10.0.0.1', port: 2222, username: 'root', authType: 'key' }, note: 'Xshell 사용자 키 "id_lab" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정)' },
+  ];
+
+  it('[Xshell 내보내기 파일 선택]의 결과가 목록에 더해지고 폴더 이름이 "Xshell" 이 된다', async () => {
+    seed();
+    const calls = mockBackend(SAMPLE, null, XSH);
+    render(<ImportDialog />);
+    await screen.findByText('prod-web02');
+    fireEvent.click(screen.getByRole('button', { name: /Xshell 내보내기 파일 선택/ }));
+    expect(await screen.findByText('db01')).toBeTruthy();
+    expect(screen.getByText('LAB/DB/')).toBeTruthy(); // 폴더 구조
+    expect(screen.getAllByText('Xshell').length).toBeGreaterThan(0); // 출처
+    expect(screen.getByText('prod-web02')).toBeTruthy(); // 다른 출처 후보는 그대로
+    expect((screen.getByLabelText('넣을 폴더 이름') as HTMLInputElement).value).toBe('Xshell');
+    expect(calls.filter(([c]) => c === 'import.scanXshell')).toEqual([['import.scanXshell', []]]);
+  });
+
+  it('고른 Xshell 후보를 그대로 import.apply 로 보낸다', async () => {
+    seed();
+    const calls = mockBackend([], null, XSH);
+    render(<ImportDialog />);
+    await screen.findByText('가져올 세션을 찾지 못했습니다.');
+    fireEvent.click(screen.getByRole('button', { name: /Xshell 내보내기 파일 선택/ }));
+    await screen.findByText('db01');
+    fireEvent.click(screen.getByRole('button', { name: '2개 가져오기' }));
+    await waitFor(() => expect(calls.some(([c]) => c === 'import.apply')).toBe(true));
+    const [sent, folder] = calls.find(([c]) => c === 'import.apply')![1] as [typeof XSH, string];
+    expect(sent.map((c) => c.name)).toEqual(['db01', 'gw']);
+    expect(sent[0].folderPath).toEqual(['LAB', 'DB']);
+    expect(folder).toBe('Xshell');
+  });
+
+  it('파일 선택을 취소(null)하면 아무 일도 없다', async () => {
+    seed();
+    const calls = mockBackend(SAMPLE, null, null);
+    render(<ImportDialog />);
+    await screen.findByText('prod-web02');
+    fireEvent.click(screen.getByRole('button', { name: /Xshell 내보내기 파일 선택/ }));
+    await waitFor(() => expect(calls.some(([c]) => c === 'import.scanXshell')).toBe(true));
+    expect(useApp.getState().toasts.length).toBe(0);
+  });
+
+  it('세션이 없으면 정보 토스트, 읽지 못하면 오류 토스트', async () => {
+    seed();
+    mockBackend(SAMPLE, null, []);
+    const first = render(<ImportDialog />);
+    await screen.findByText('prod-web02');
+    fireEvent.click(screen.getByRole('button', { name: /Xshell 내보내기 파일 선택/ }));
+    await waitFor(() => expect(useApp.getState().toasts.map((x) => x.text)).toContain('선택한 파일에서 Xshell 세션을 찾지 못했습니다'));
+    first.unmount();
+
+    seed();
+    mockBackend(SAMPLE, null, new Error('zip: not a valid zip file'));
+    render(<ImportDialog />);
+    await screen.findByText('prod-web02');
+    fireEvent.click(screen.getByRole('button', { name: /Xshell 내보내기 파일 선택/ }));
+    await waitFor(() => expect(useApp.getState().toasts.map((x) => x.text)).toContain('Xshell 파일을 읽지 못했습니다: zip: not a valid zip file'));
+  });
+
+  it('자동 검색에 Xshell 후보만 있으면 기본 폴더 이름이 "Xshell" 이다', async () => {
+    seed();
+    mockBackend(XSH);
+    render(<ImportDialog />);
+    await screen.findByText('db01');
+    expect((screen.getByLabelText('넣을 폴더 이름') as HTMLInputElement).value).toBe('Xshell');
+  });
+});
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키 11개를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

넣는 자리: `import.xshNone`·`import.xshFailed` 는 `import.crtFailed` 다음, `import.xshBusy`·`import.xsh`·`import.xshNote` 는 `import.crtNote` 다음, `be.dlg.pickXshell`·`be.dlg.xshellFilter` 는 `be.dlg.pickCrt` 다음, `be.xsh.*` 4개는 `be.crt.firewall` 다음. 있던 문구는 고치지 않았다.

**`import.xshNone`** (추가)

```json
  "import.xshNone": "선택한 파일에서 Xshell 세션을 찾지 못했습니다",
  "import.xshNone": "No Xshell sessions found in the selected file",
  "import.xshNone": "選択したファイルに Xshell のセッションが見つかりませんでした",
  "import.xshNone": "所选文件中未找到 Xshell 会话",
  "import.xshNone": "所選檔案中未找到 Xshell 工作階段",
  "import.xshNone": "In der ausgewählten Datei wurden keine Xshell-Sitzungen gefunden",
  "import.xshNone": "Aucune session Xshell dans le fichier sélectionné",
  "import.xshNone": "No se encontraron sesiones de Xshell en el archivo",
  "import.xshNone": "Nenhuma sessão do Xshell encontrada no arquivo",
  "import.xshNone": "В выбранном файле нет сеансов Xshell",
```

**`import.xshFailed`** (추가)

```json
  "import.xshFailed": "Xshell 파일을 읽지 못했습니다: {error}",
  "import.xshFailed": "Could not read the Xshell file: {error}",
  "import.xshFailed": "Xshell ファイルを読み込めませんでした: {error}",
  "import.xshFailed": "无法读取 Xshell 文件：{error}",
  "import.xshFailed": "無法讀取 Xshell 檔案：{error}",
  "import.xshFailed": "Xshell-Datei konnte nicht gelesen werden: {error}",
  "import.xshFailed": "Impossible de lire le fichier Xshell : {error}",
  "import.xshFailed": "No se pudo leer el archivo de Xshell: {error}",
  "import.xshFailed": "Não foi possível ler o arquivo do Xshell: {error}",
  "import.xshFailed": "Не удалось прочитать файл Xshell: {error}",
```

**`import.xshBusy`** (추가)

```json
  "import.xshBusy": "Xshell 파일 읽는 중...",
  "import.xshBusy": "Reading Xshell file...",
  "import.xshBusy": "Xshell ファイルを読み込み中...",
  "import.xshBusy": "正在读取 Xshell 文件...",
  "import.xshBusy": "正在讀取 Xshell 檔案...",
  "import.xshBusy": "Xshell-Datei wird gelesen...",
  "import.xshBusy": "Lecture du fichier Xshell...",
  "import.xshBusy": "Leyendo archivo de Xshell...",
  "import.xshBusy": "Lendo arquivo do Xshell...",
  "import.xshBusy": "Чтение файла Xshell...",
```

**`import.xsh`** (추가)

```json
  "import.xsh": "Xshell 내보내기 파일 선택",
  "import.xsh": "Choose Xshell export file",
  "import.xsh": "Xshell のエクスポートファイルを選択",
  "import.xsh": "选择 Xshell 导出文件",
  "import.xsh": "選擇 Xshell 匯出檔",
  "import.xsh": "Xshell-Exportdatei wählen",
  "import.xsh": "Choisir le fichier d'export Xshell",
  "import.xsh": "Elegir archivo de exportación de Xshell",
  "import.xsh": "Escolher arquivo de exportação do Xshell",
  "import.xsh": "Выбрать файл экспорта Xshell",
```

**`import.xshNote`** (추가)

```json
  "import.xshNote": "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일에서 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다. 비밀번호·사용자 키·포트 포워딩·프록시 설정은 옮기지 않습니다.",
  "import.xshNote": "Reads sessions from an .xts file made with [File > Export] in Xshell. The folder structure is recreated under the target folder. Passwords, user keys, port forwarding and proxy settings are not imported.",
  "import.xshNote": "Xshell の [ファイル > エクスポート] で作成した .xts ファイルからセッションを読み込みます。フォルダー構成は追加先フォルダーの下にそのまま作られます。パスワード・ユーザーキー・ポート転送・プロキシ設定は移しません。",
  "import.xshNote": "从 Xshell 的 [文件 > 导出] 生成的 .xts 文件读取会话。文件夹结构会在目标文件夹下原样创建。密码、用户密钥、端口转发和代理设置不迁移。",
  "import.xshNote": "從 Xshell 的 [檔案 > 匯出] 產生的 .xts 檔讀取工作階段。資料夾結構會在目標資料夾下原樣建立。密碼、使用者金鑰、連接埠轉送與 Proxy 設定不移轉。",
  "import.xshNote": "Liest Sitzungen aus einer .xts-Datei, die in Xshell mit [Datei > Exportieren] erstellt wurde. Die Ordnerstruktur entsteht unter dem Zielordner. Passwörter, Benutzerschlüssel, Portweiterleitungen und Proxy-Einstellungen werden nicht übernommen.",
  "import.xshNote": "Lit les sessions d'un fichier .xts créé avec [Fichier > Exporter] dans Xshell. L'arborescence est recréée sous le dossier cible. Les mots de passe, clés utilisateur, redirections de ports et paramètres de proxy ne sont pas repris.",
  "import.xshNote": "Lee las sesiones de un archivo .xts creado con [Archivo > Exportar] en Xshell. La estructura de carpetas se crea bajo la carpeta de destino. No se trasladan contraseñas, claves de usuario, reenvíos de puertos ni ajustes de proxy.",
  "import.xshNote": "Lê as sessões de um arquivo .xts criado com [Arquivo > Exportar] no Xshell. A estrutura de pastas é criada na pasta de destino. Senhas, chaves de usuário, encaminhamento de portas e configurações de proxy não são transferidos.",
  "import.xshNote": "Читает сеансы из файла .xts, созданного в Xshell через [Файл > Экспорт]. Структура папок создаётся внутри целевой папки. Пароли, пользовательские ключи, проброс портов и настройки прокси не переносятся.",
```

**`be.dlg.pickXshell`** (추가)

```json
  "be.dlg.pickXshell": "Xshell 내보내기 파일(.xts) 선택",
  "be.dlg.pickXshell": "Choose Xshell export file (.xts)",
  "be.dlg.pickXshell": "Xshell のエクスポートファイル (.xts) を選択",
  "be.dlg.pickXshell": "选择 Xshell 导出文件 (.xts)",
  "be.dlg.pickXshell": "選擇 Xshell 匯出檔 (.xts)",
  "be.dlg.pickXshell": "Xshell-Exportdatei (.xts) wählen",
  "be.dlg.pickXshell": "Choisir le fichier d'export Xshell (.xts)",
  "be.dlg.pickXshell": "Elegir archivo de exportación de Xshell (.xts)",
  "be.dlg.pickXshell": "Escolher arquivo de exportação do Xshell (.xts)",
  "be.dlg.pickXshell": "Выбор файла экспорта Xshell (.xts)",
```

**`be.dlg.xshellFilter`** (추가)

```json
  "be.dlg.xshellFilter": "Xshell 내보내기 파일",
  "be.dlg.xshellFilter": "Xshell export file",
  "be.dlg.xshellFilter": "Xshell エクスポートファイル",
  "be.dlg.xshellFilter": "Xshell 导出文件",
  "be.dlg.xshellFilter": "Xshell 匯出檔",
  "be.dlg.xshellFilter": "Xshell-Exportdatei",
  "be.dlg.xshellFilter": "Fichier d'export Xshell",
  "be.dlg.xshellFilter": "Archivo de exportación de Xshell",
  "be.dlg.xshellFilter": "Arquivo de exportação do Xshell",
  "be.dlg.xshellFilter": "Файл экспорта Xshell",
```

**`be.xsh.userKey`** (추가)

```json
  "be.xsh.userKey": "Xshell 사용자 키 \"{name}\" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정)",
  "be.xsh.userKey": "Xshell user key \"{name}\" is not imported (export the key and set it in the server editor)",
  "be.xsh.userKey": "Xshell のユーザーキー「{name}」は移しません（キーをエクスポートしてサーバー編集で指定）",
  "be.xsh.userKey": "未迁移 Xshell 用户密钥“{name}”（请导出密钥后在服务器编辑中指定）",
  "be.xsh.userKey": "未移轉 Xshell 使用者金鑰「{name}」（請匯出金鑰後在伺服器編輯中指定）",
  "be.xsh.userKey": "Xshell-Benutzerschlüssel „{name}“ wird nicht übernommen (Schlüssel exportieren und in der Serverbearbeitung angeben)",
  "be.xsh.userKey": "Clé utilisateur Xshell « {name} » non importée (exportez la clé et indiquez-la dans l'éditeur de serveur)",
  "be.xsh.userKey": "No se importa la clave de usuario de Xshell «{name}» (exporte la clave e indíquela en el editor del servidor)",
  "be.xsh.userKey": "A chave de usuário do Xshell “{name}” não é importada (exporte a chave e informe-a no editor do servidor)",
  "be.xsh.userKey": "Пользовательский ключ Xshell «{name}» не переносится (экспортируйте ключ и укажите его в редакторе сервера)",
```

**`be.xsh.forwards`** (추가)

```json
  "be.xsh.forwards": "포트 포워딩 {count}개는 옮기지 않음",
  "be.xsh.forwards": "{count} port forwarding rule(s) not imported",
  "be.xsh.forwards": "ポート転送 {count} 件は移しません",
  "be.xsh.forwards": "未迁移 {count} 条端口转发",
  "be.xsh.forwards": "未移轉 {count} 筆連接埠轉送",
  "be.xsh.forwards": "{count} Portweiterleitung(en) nicht übernommen",
  "be.xsh.forwards": "{count} redirection(s) de port non importée(s)",
  "be.xsh.forwards": "No se importan {count} reenvío(s) de puertos",
  "be.xsh.forwards": "{count} encaminhamento(s) de porta não importado(s)",
  "be.xsh.forwards": "Проброс портов ({count}) не переносится",
```

**`be.xsh.proxy`** (추가)

```json
  "be.xsh.proxy": "프록시 설정 \"{name}\" 은 옮기지 않음",
  "be.xsh.proxy": "Proxy setting \"{name}\" is not imported",
  "be.xsh.proxy": "プロキシ設定「{name}」は移しません",
  "be.xsh.proxy": "未迁移代理设置“{name}”",
  "be.xsh.proxy": "未移轉 Proxy 設定「{name}」",
  "be.xsh.proxy": "Proxy-Einstellung „{name}“ wird nicht übernommen",
  "be.xsh.proxy": "Paramètre de proxy « {name} » non importé",
  "be.xsh.proxy": "No se importa la configuración de proxy «{name}»",
  "be.xsh.proxy": "A configuração de proxy “{name}” não é importada",
  "be.xsh.proxy": "Настройка прокси «{name}» не переносится",
```

**`be.xsh.badFile`** (추가)

```json
  "be.xsh.badFile": "Xshell 내보내기 파일을 열지 못했습니다",
  "be.xsh.badFile": "Could not open the Xshell export file",
  "be.xsh.badFile": "Xshell のエクスポートファイルを開けませんでした",
  "be.xsh.badFile": "无法打开 Xshell 导出文件",
  "be.xsh.badFile": "無法開啟 Xshell 匯出檔",
  "be.xsh.badFile": "Xshell-Exportdatei konnte nicht geöffnet werden",
  "be.xsh.badFile": "Impossible d'ouvrir le fichier d'export Xshell",
  "be.xsh.badFile": "No se pudo abrir el archivo de exportación de Xshell",
  "be.xsh.badFile": "Não foi possível abrir o arquivo de exportação do Xshell",
  "be.xsh.badFile": "Не удалось открыть файл экспорта Xshell",
```


## F. 명세·문서

### `spec/import.md` — 수정

```diff
--- a/spec/import.md
+++ b/spec/import.md
@@ -1,8 +1,8 @@
-# 서버 가져오기 — PuTTY · ssh config · AWS · 적용 규칙
+# 서버 가져오기 — PuTTY · SecureCRT · Xshell · ssh config · AWS · 적용 규칙
 
 > 작성 방식: 기존 앱 동작 열람. 작성일 2026-11-10. 상태: **draft**. 화면은 ui/dialogs.md §5.
 
-후보(candidate) 한 건: `source`(`putty`/`securecrt`/`ssh-config`/`aws`), `name`, 서버 부분 정보(`host`), `keyFile`(선택), `jumpVia`(경유 서버 주소, 선택), `note`(선택), `folderPath`(넣을 폴더 아래의 하위 폴더 이름 배열, 선택 — SecureCRT 만 씀).
+후보(candidate) 한 건: `source`(`putty`/`securecrt`/`xshell`/`ssh-config`/`aws`), `name`, 서버 부분 정보(`host`), `keyFile`(선택), `jumpVia`(경유 서버 주소, 선택), `note`(선택), `folderPath`(넣을 폴더 아래의 하위 폴더 이름 배열, 선택 — SecureCRT·Xshell 이 씀).
 
 ## 1. PuTTY (Windows 레지스트리)
 - 읽는 곳: `HKEY_CURRENT_USER\Software\SimonTatham\PuTTY\Sessions` 아래 하위 키 각각이 세션. 값은 문자열(REG_SZ/REG_EXPAND_SZ) 또는 정수(DWORD/QWORD)만 읽는다. 키가 없으면 빈 목록. 비 Windows 는 지원하지 않는다(빈 목록).
@@ -43,6 +43,32 @@
 - **시리얼 세션**(표본 미확인): `Com Port`(없으면 COM1)를 장치와 호스트로, `Baud Rate`(기본 9600), `Data Bits`(기본 8), `Parity`(0=none,1=odd,2=even,3=mark,4=space; 범위 밖 none), `Stop Bits`(0→1, 1→1.5, 2→2), 흐름 제어 none, Enter 전송값 CR, 로컬 에코 끔.
 - 후보 이름·서버 별칭 = 세션 이름.
 
+## 1.6 Xshell (내보내기 파일 · 세션 폴더)
+
+> 작성 방식: **원본 앱에 없는 신규 기능**(사용자 요청, 2026-11-25, D128). 형식은 사용자가 준 Xshell 내보내기 파일(`.xts`, 세션 파일 164개 — 판 8.1 이 110개·7.1 이 54개, 모두 SSH·비밀번호 인증·UTF-8) 관찰로 정했다. "(표본 미확인)" 표시는 그 파일에 해당하는 값이 없어 실제로 확인하지 못한 최선 추정이다.
+
+- **읽는 곳**(셋 중 하나):
+  - **내보내기 파일**(`.xts`, Xshell 의 [파일 > 내보내기]로 만든 것): zip 이다. 그 안에서 **최상위 폴더 이름이 `Xshell`**(대소문자 무시)인 항목 아래만 읽는다(`Xftp`·`xsl`·`com` 등 다른 폴더는 보지 않는다). 디스크에 풀지 않고 읽는다. zip 으로 열 수 없으면 오류 "Xshell 내보내기 파일을 열지 못했습니다: {원인}".
+    - 항목 이름은 UTF-8 표시가 없는 경우가 있다(내보낸 PC 의 코드 페이지 바이트): **유효한 UTF-8 이 아니면 EUC-KR(CP949)로 해석**한다. 구분자 `\` 는 `/` 로 본다.
+    - 한도: 세션 파일 하나 1MB(넘으면 그 파일은 건너뜀), 세션 20000개, 읽은 내용 합계 64MB(넘으면 거기서 멈춤).
+  - **세션 폴더**: 고른 폴더 안에 `Sessions` 폴더가 있으면 그것을, 없으면 고른 폴더 자체를 세션 폴더로 본다.
+  - **이 PC 의 기본 세션 폴더**(자동 검색용, 표본 미확인): "문서" 폴더 아래 `NetSarang Computer\8\Xshell\Sessions`, 없으면 `…\7\…`, `…\6\…`, 그다음 옛 판의 `NetSarang\Xshell\Sessions` 순으로 처음 있는 것. "문서" 폴더는 Windows 에서 `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders` 의 `Personal`(환경 변수를 펼침), 그다음 `%USERPROFILE%\Documents`. 비 Windows 는 기본 폴더가 없다(빈 목록). 없으면 빈 목록.
+- **폴더 구조**: 세션 폴더(내보내기 파일에서는 `Xshell/`) 아래 하위 폴더를 모두 따라 내려간다. 확장자 `.xsh`(대소문자 무시) 파일 하나가 세션 하나. 세션 이름 = 파일 이름에서 `.xsh` 를 뗀 것. `folderPath` = 그 파일까지의 폴더 이름들(최상위 세션은 빈 배열). 폴더 이름이 빈 것·`.`·`..` 인 항목은 버린다. 다른 확장자(`default.xshf.xfs` 등)는 읽지 않는다. 순서는 폴더 경로·이름의 사전순.
+- **파일 형식**: 맨 앞이 UTF-16 BOM(`FF FE` 또는 `FE FF`)이면 UTF-16 으로 읽는다(표본은 모두 `FF FE`). UTF-8 BOM 은 버리고, 그 밖에는 유효한 UTF-8 이면 그대로 아니면 EUC-KR. 줄 끝 CR 무시. INI 형식: `[구역]` 줄과 `키=값` 줄(첫 `=` 에서 나눔, 키·값의 앞뒤 공백을 뗌). 구역·키 이름은 대소문자를 가리지 않는다. 같은 구역의 같은 키가 두 번 나오면 처음 것. 그 밖의 줄은 무시.
+- **프로토콜**: `[CONNECTION] Protocol` 이 `SSH`(또는 없음)이면 ssh, `TELNET` 이면 telnet(표본 미확인). 그 밖(`SERIAL`·`RLOGIN`·`SFTP`·`FTP`·`LOCAL`·`RDP` 등)은 **건너뜀**. 시리얼을 건너뛰는 까닭: 시리얼 값(`[CONNECTION:SERIAL]` 의 `BaudRate=12`·`DataBits=3`·`ComPort=0` 등)이 목록의 번호로 적혀 있는데 번호의 뜻을 확인할 표본이 없다.
+- 건너뛰는 세션: `[CONNECTION] Host` 가 빈 것.
+- **호스트·포트·사용자**: `[CONNECTION] Host`, `[CONNECTION] Port`(없거나 1~65535 밖이면 22, 텔넷 23), `[CONNECTION:AUTHENTICATION] UserName`.
+- **메모**: `[CONNECTION] Description` 이 비어 있지 않으면 `notes`.
+- **인코딩**: `[TERMINAL] CodePage`(Windows 코드 페이지 번호) — 65001→utf-8, 949→cp949, 51949→euc-kr, 932→shift_jis, 51932·20932→euc-jp, 936·54936→gbk, 950→big5, 1252·28591→latin1, 그 밖·없음→utf-8. (표본은 65001 뿐)
+- **터미널 종류**: `[TERMINAL] Type` 이 비어 있거나 `xterm` 이면 옮기지 않는다(적용 기본값 xterm-256color). 다른 값은 그대로(표본 미확인).
+- **연결 유지**: `[CONNECTION:KEEPALIVE] KeepAlive` 가 0 이 아니고 `KeepAliveInterval` 이 양수면 그 초, 아니면 30.
+- **인증**: 기본 `password`. ssh 세션에서 `[CONNECTION:AUTHENTICATION] UserKey` 가 비어 있지 않으면 `key`(keyFile 은 없음 — 적용 규칙 §4 에 따라 서버는 `agent` 인증이 된다)이고 후보 `note` 에 `Xshell 사용자 키 "{이름}" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정)`. 사용자 키가 없고 `[CONNECTION:SSH] UseAuthAgent` 가 0 이 아니면 `key`(안내 없음). (둘 다 표본 미확인) **저장된 비밀번호(`Password`)·암호문(`Passphrase`)은 읽지 않는다.**
+  - 사용자 키를 옮기지 않는 까닭: Xshell 은 키를 세션 파일이 아니라 자기 키 보관소에 두고 세션에는 이름만 적는다.
+- **자동 실행 명령**: ssh 세션의 `[CONNECTION:SSH] RemoteCommand` 가 비어 있지 않으면 `startupCommand`(표본 미확인).
+- **옮기지 않고 알리는 것**(후보 `note`, 여러 개면 ` · ` 로 잇는다): 위의 사용자 키, ssh 세션의 `[CONNECTION:SSH] FwdReqCount` 가 양수면 `포트 포워딩 {개수}개는 옮기지 않음`(포워딩 항목의 형식을 확인할 표본이 없다), `[CONNECTION:PROXY] Proxy` 가 비어 있지 않으면 `프록시 설정 "{이름}" 은 옮기지 않음`.
+- 그 밖의 값(글꼴·색 구성·탭 색·로그·X11 전달·암호화 알고리즘 목록 등)은 읽지 않는다.
+- 후보 이름·서버 별칭 = 세션 이름.
+
 ## 2. ssh config (`~/.ssh/config`, 없으면 빈 목록)
 - 줄 단위로 읽는다. `#` 뒤는 주석. 빈 줄 무시. `키 값` 또는 `키=값` 형식, 키는 대소문자 무시, 값의 양끝 큰따옴표 제거.
 - `Host` 줄이 새 블록을 시작한다(여러 이름은 공백으로 구분 → 이름마다 후보 하나). **`*` `?` `!` 가 들어간 이름(와일드카드·부정)은 건너뜀.** `Match` 줄은 블록을 끝내고 그 뒤 값은 어느 블록에도 속하지 않는다.
@@ -69,8 +95,8 @@
    - **키 파일**: `keyFile` 이 있으면 파일을 읽어 — 읽지 못하면 건너뛰기 목록에 `{이름}: 키 파일을 읽지 못함 ({경로})` 추가. 읽었고 **볼트가 열려 있으면** 볼트에 항목(이름 `{후보 이름} 키`, 종류 key, 사용자 이름=서버 사용자)으로 저장하고 서버에 연결(키 개수 +1). 볼트가 잠겨 있으면 복사하지 않는다.
    - **환경**: 후보 이름(소문자)이 `prod`/`prd`/`live`/`운영` 으로 **시작**하면 prod, `stg`/`stage`/`staging`/`func`/`기능` 이면 stage(기능테스트), `perf`/`성능` 이면 perf(성능테스트), `dev`/`test`/`개발` 이면 dev, 그 외 프로토콜이 시리얼이면 device, 아니면 none.
    - **기본값**: 프로토콜 없으면 ssh, 포트 없으면 22, **인증**: 키 파일을 볼트에 저장했으면 `key`, 아니고 후보의 인증이 `key` 였으면 `agent`, 그 외 `password`. 별칭 없으면 후보 이름. 인코딩 없으면 utf-8, 터미널 종류 없으면 xterm-256color, 연결 유지 값이 없거나 0 이면 30, 포워딩 없으면 빈 목록.
-   - **태그**: 출처 한 개(`putty`/`securecrt`/`ssh-config`/`aws`). **자동 실행 명령·시리얼·메모는 비어 있지 않을 때만** 복사.
+   - **태그**: 출처 한 개(`putty`/`securecrt`/`xshell`/`ssh-config`/`aws`). **자동 실행 명령·시리얼·메모는 비어 있지 않을 때만** 복사.
 3. **점프 호스트 연결**: 점프 주소(`jumpVia`)가 있는 후보마다, 이번에 새로 만든 서버 중 **호스트가 같은 것**을 먼저 찾고 없으면 기존 서버 전체에서 찾는다. 찾았고 자기 자신이 아니면 `jumpHostId` 로 연결. 못 찾으면 건너뛰기 목록에 `{이름}: 점프 호스트 {주소} 를 찾지 못해 직접 접속으로 등록` 추가.
 4. 폴더·서버 목록을 저장한다. 결과: `{hosts: 후보 수, keys: 볼트에 저장한 키 수, skipped: 건너뛰기 메시지 배열}`.
 - **중복 판정은 화면에서만** 한다(초기 선택 해제용, ui/dialogs.md §5). 적용 단계는 중복을 검사하지 않고 선택된 것을 모두 추가한다.
-- `후보 검색`(import.scan)은 PuTTY 후보, 기본 위치의 SecureCRT 후보, ssh config 후보 순으로 이어 붙인 목록을 돌려준다.
+- `후보 검색`(import.scan)은 PuTTY 후보, 기본 위치의 SecureCRT 후보, 기본 위치의 Xshell 후보(§1.6), ssh config 후보 순으로 이어 붙인 목록을 돌려준다.
```

### `spec/ipc.md` — 수정

````diff
--- a/spec/ipc.md
+++ b/spec/ipc.md
@@ -93,14 +93,14 @@
 ### 가져오기 후보 객체 (import.scan / import.scanAws 결과, import.apply 입력)
 후보 하나는 다음 JSON 객체다. **서버 정보는 평평하게 펼치지 않고 `host` 객체 안에 둔다.**
 ```
-{ "source": "putty" | "securecrt" | "ssh-config" | "aws",
+{ "source": "putty" | "securecrt" | "xshell" | "ssh-config" | "aws",
   "name": 문자열,
   "host": { "alias", "protocol", "host", "port", "username"?, "authType", "keepaliveSec"?, "encoding"?, "termType"?,
             "forwards"?, "serial"?, "startupCommand"?, "notes"? },   // data/store.md 서버 필드의 부분 집합
   "keyFile"?: 문자열(키 파일 경로),
   "jumpVia"?: 문자열(점프 호스트 주소),
   "note"?: 문자열,
-  "folderPath"?: 문자열 배열(넣을 폴더 아래 하위 폴더, SecureCRT) }
+  "folderPath"?: 문자열 배열(넣을 폴더 아래 하위 폴더, SecureCRT·Xshell) }
 ```
 - 화면은 이 구조를 그대로 읽는다: 종류 열 = `host.protocol`(없으면 `ssh`), 주소 열 = `host.username`·`host.host`·`host.port`(시리얼은 `host.serial.path`·`host.serial.baudRate`), 이름 열 = `name`, 출처 = `source`, 중복 판정 = `host.host`·`host.port`·`host.username` 이 기존 서버와 같은지.
 - `import.apply` 에는 사용자가 고른 **후보 객체를 변경 없이 그대로** 보내고, 두 번째 인자로 폴더 이름을 보낸다.
@@ -109,8 +109,9 @@
 ### 가져오기·동기화·업데이트
 | 채널 | 인자 | 결과 |
 |---|---|---|
-| `import.scan` | | 후보 배열(PuTTY 레지스트리 + 기본 위치 SecureCRT + ssh config) — **후보 JSON 구조는 위 "가져오기 후보 객체"** |
+| `import.scan` | | 후보 배열(PuTTY 레지스트리 + 기본 위치 SecureCRT + 기본 위치 Xshell + ssh config) — **후보 JSON 구조는 위 "가져오기 후보 객체"** |
 | `import.scanSecureCrt` | 폴더 경로(선택) | 후보 배열(SecureCRT, import.md §1.5). 경로를 안 주면 폴더 선택 대화상자("SecureCRT 설정 폴더 선택", 시작 폴더는 기본 설정 폴더, 새 폴더 만들기 없음) — 취소하면 null |
+| `import.scanXshell` | 경로(선택): 내보내기 파일(`.xts`) 또는 세션 폴더 | 후보 배열(Xshell, import.md §1.6) / 오류(메시지). 경로를 안 주면 파일 선택 대화상자("Xshell 내보내기 파일(.xts) 선택", 종류 `*.xts` 와 모든 파일, 시작 폴더는 홈) — 취소하면 null (2026-11-25, D128) |
 | `import.scanAws` | 프로필(선택) | 후보 배열 / 오류(메시지) |
 | `import.apply` | 후보 배열, 폴더 이름 | `{hosts, keys, skipped}` |
 | `sync.run` | 모드(`""` / `pull-vault` / `overwrite`) | 상태 `{state, message?, lastSync?, changed?}` |
````

### `spec/ui/dialogs.md` — 수정

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -104,16 +104,18 @@
 1. **안내 박스**: "PuTTY 저장 세션(레지스트리), SecureCRT 세션 폴더, ~/.ssh/config를 읽어 옵니다. 호스트·포트·사용자·키 파일·포트 포워딩·시리얼 설정·인코딩이 옮겨집니다. 비밀번호는 옮기지 않으므로 처음 연결할 때 한 번 입력하면 저장됩니다." 볼트가 잠겨 있으면 문장 끝에 " (앱이 잠겨 있으면 키 파일은 저장되지 않습니다)" 추가.
 2. **AWS EC2 불러오기** 버튼(구름 아이콘, 진행 중 "AWS 전 리전 조회 중...") + 설명 "aws CLI 자격 증명으로 인스턴스를 읽습니다. 사설 IP 서버는 같은 VPC 의 경유 서버를 경유 서버로 잇고, 키는 ~/.ssh/<키 이름>.pem 에서 찾습니다."
    - **SecureCRT 폴더 선택** 버튼(폴더 아이콘, 진행 중 "SecureCRT 폴더 읽는 중...") + 설명 "다른 위치(다른 PC에서 복사한 Config 폴더 등)의 SecureCRT 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다." (신규 기능, D66)
+   - **Xshell 내보내기 파일 선택** 버튼(압축 파일 아이콘, 진행 중 "Xshell 파일 읽는 중...") + 설명 "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일에서 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다. 비밀번호·사용자 키·포트 포워딩·프록시 설정은 옮기지 않습니다." (신규 기능, 2026-11-25, D128)
 3. 후보 검색 중이면 "검색 중...". 후보가 없으면 "가져올 세션을 찾지 못했습니다." 줄바꿈 "PuTTY·SecureCRT에 저장된 세션이 없거나 ~/.ssh/config가 없습니다."
 4. 후보가 있으면:
-   - 필드 "넣을 폴더 이름"(기본값 "PuTTY 가져오기", 단 자동 검색 결과에 PuTTY 후보가 없고 SecureCRT 후보가 있으면 "SecureCRT"), 도움말 "이름이 prod-/stg-/dev- 로 시작하면 운영/검증/개발 환경이 자동 지정됩니다. 비우면 최상위에 넣습니다. SecureCRT 폴더 구조는 이 폴더 아래에 만들어집니다."
-   - 표(머리: 전체 선택 체크박스, 이름, 주소, 종류, 출처). 행마다 체크박스, 굵은 이름(`folderPath` 가 있으면 이름 앞에 흐린 글씨로 `폴더/폴더/`), 주소(`사용자@호스트:포트`, 시리얼은 `경로 @ 속도`, 키 파일이 있으면 열쇠 표시), 종류(프로토콜 대문자, 기본 SSH), 출처(`PuTTY` / `SecureCRT` / `ssh config` / AWS 는 `AWS`에 경유 서버 경유 시 ` · 배스천 경유`, 키 파일 없음 메모가 있으면 경고 표시; 툴팁에 메모).
+   - 필드 "넣을 폴더 이름"(기본값 "PuTTY 가져오기", 단 자동 검색 결과에 PuTTY 후보가 없고 SecureCRT 후보가 있으면 "SecureCRT", PuTTY·SecureCRT 후보가 없고 Xshell 후보가 있으면 "Xshell"), 도움말 "이름이 prod-/stg-/dev- 로 시작하면 운영/검증/개발 환경이 자동 지정됩니다. 비우면 최상위에 넣습니다. SecureCRT 폴더 구조는 이 폴더 아래에 만들어집니다."
+   - 표(머리: 전체 선택 체크박스, 이름, 주소, 종류, 출처). 행마다 체크박스, 굵은 이름(`folderPath` 가 있으면 이름 앞에 흐린 글씨로 `폴더/폴더/`), 주소(`사용자@호스트:포트`, 시리얼은 `경로 @ 속도`, 키 파일이 있으면 열쇠 표시), 종류(프로토콜 대문자, 기본 SSH), 출처(`PuTTY` / `SecureCRT` / `Xshell` / `ssh config` / AWS 는 `AWS`에 경유 서버 경유 시 ` · 배스천 경유`, 키 파일 없음 메모가 있으면 경고 표시; 툴팁에 메모).
    - **초기 선택**: 이미 같은 호스트·포트·사용자의 서버가 있는 후보는 선택하지 않고, 나머지는 모두 선택.
 
 동작:
 - 대화상자가 열리면 후보를 자동 검색한다.
 - AWS 조회: 결과가 없으면 정보 토스트 "AWS 에서 가져올 인스턴스를 찾지 못했습니다". 결과가 있으면 기존 AWS 후보를 대체하고(다른 출처 후보는 유지) 새 후보를 선택(중복 제외)하며 폴더 이름을 "AWS"로 바꾼다. 실패하면 오류 토스트 "AWS 조회 실패: {오류} — 터미널에서 aws configure 로 자격 증명을 확인하세요".
 - SecureCRT 폴더 선택: 폴더 선택을 취소하면 아무것도 안 함. 결과가 없으면 정보 토스트 "선택한 폴더에서 SecureCRT 세션을 찾지 못했습니다". 결과가 있으면 기존 SecureCRT 후보를 대체하고(다른 출처 후보는 유지) 새 후보를 선택(중복 제외)하며 폴더 이름을 "SecureCRT"로 바꾼다. 실패하면 오류 토스트 "SecureCRT 폴더를 읽지 못했습니다: {오류}".
+- Xshell 내보내기 파일 선택(2026-11-25, D128): 파일 선택을 취소하면 아무것도 안 함. 결과가 없으면 정보 토스트 "선택한 파일에서 Xshell 세션을 찾지 못했습니다". 결과가 있으면 기존 Xshell 후보를 대체하고(다른 출처 후보는 유지) 새 후보를 선택(중복 제외)하며 폴더 이름을 "Xshell"로 바꾼다. 실패하면 오류 토스트 "Xshell 파일을 읽지 못했습니다: {오류}". 이 PC 에 Xshell 이 설치돼 있으면 그 세션은 대화상자를 열 때의 자동 검색에 이미 들어 있다.
 - 가져오기 실행: 선택된 후보를 폴더 이름과 함께 백엔드에 적용 → 저장 데이터를 다시 읽고 → 성공 토스트 "{n}개 서버를 가져왔습니다"(키를 볼트에 저장했으면 " (키 {k}개 암호화해 저장)" 덧붙임) → 건너뛴 항목이 있으면 오류 토스트에 줄바꿈으로 나열 → 닫기. 실패하면 오류 토스트.
 
 ## 6. 동기화(git) 패널 — 설정 화면의 한 구역
```

### `docs/scenarios.md` — 수정

```diff
--- a/docs/scenarios.md
+++ b/docs/scenarios.md
@@ -152,6 +152,7 @@
 | 10-1 | 가져오기 열기 | 사이드바 [가져오기], 빠른 연결 `> 가져오기`, 시작 화면 | PuTTY 저장 세션(레지스트리), SecureCRT 세션 폴더, `~/.ssh/config` 를 자동으로 찾아 목록으로 보인다. |
 | 10-2 | 고르기 | 체크, 전체 선택 | 이름·주소·종류·출처, 키 파일 표시. "넣을 폴더 이름"(기본 "PuTTY 가져오기", prod-/stg-/dev- 로 시작하면 환경 자동). |
 | 10-3 | SecureCRT 다른 위치 | [SecureCRT 폴더 선택] | 다른 PC 에서 복사한 Config 폴더도 읽는다. 폴더 구조와 포트 포워딩을 그대로. |
+| 10-3a | Xshell | [Xshell 내보내기 파일 선택] | Xshell 의 [파일 > 내보내기]로 만든 `.xts` 파일에서 세션(이름·폴더 구조·호스트·포트·사용자·메모·인코딩)을 읽는다. 이 PC 에 Xshell 이 있으면 그 세션은 자동으로 목록에 나온다. 비밀번호·사용자 키·포트 포워딩·프록시·시리얼 세션은 옮기지 않는다. |
 | 10-4 | AWS | [AWS EC2 불러오기] | aws CLI 자격 증명으로 모든 리전 인스턴스. 사설 IP 는 같은 VPC 배스천을 경유 서버로, 키는 `~/.ssh/<키 이름>.pem`. |
 | 10-5 | 가져오기 | [n개 가져오기] | "n개 서버를 가져왔습니다 (키 m개 암호화해 저장)". 비밀번호는 옮기지 않는다(처음 연결 때 한 번 입력). |
 
```

### `docs/scenarios.en.md` — 수정

```diff
--- a/docs/scenarios.en.md
+++ b/docs/scenarios.en.md
@@ -151,6 +151,7 @@
 | 10-1 | Open Import | Sidebar [Import], Quick connect `> Import`, start screen | Finds PuTTY saved sessions (registry), SecureCRT session folders and `~/.ssh/config` and lists them. |
 | 10-2 | Choose | Tick items, select all | Name, address, type, source, key file. "Target folder name" (default "PuTTY import"; names starting with prod-/stg-/dev- get that environment). |
 | 10-3 | SecureCRT elsewhere | [Choose SecureCRT folder] | Reads a Config folder copied from another PC too, keeping folder structure and port forwarding. |
+| 10-3a | Xshell | [Choose Xshell export file] | Reads sessions (name, folder structure, host, port, user, notes, encoding) from an `.xts` file made with [File > Export] in Xshell. If Xshell is installed on this PC its sessions are listed automatically. Passwords, user keys, port forwarding, proxies and serial sessions are not imported. |
 | 10-4 | AWS | [Load AWS EC2] | All regions via your aws CLI credentials. Private IPs go through a bastion in the same VPC as the jump server; keys from `~/.ssh/<key name>.pem`. |
 | 10-5 | Import | [Import N] | "Imported N servers (M keys saved encrypted)". Passwords are not imported (enter once on first connection). |
 
```

`CHANGELOG.md` 의 미배포 절 "새 기능"에 한 줄, `docs/decisions.md` 에 D128 을 넣었다.

---

## G. 검증 (2026-11-25 실행)

### G.1 백엔드 — 새 파일 [internal/importer/xshell_test.go](../../internal/importer/xshell_test.go)

테스트의 값(호스트·이름)은 모두 지어낸 것이다. 세션 파일은 실제와 같은 모양(UTF-16LE + BOM, CRLF, 같은 구역·키)으로 만든다.

| 테스트 | 확인하는 것 |
|---|---|
| `TestParseXshEncodingsAndSections` | 같은 내용을 UTF-16LE·UTF-16BE·UTF-8(BOM)·UTF-8·CP949 로 줘도 같게 읽힌다. 구역·키는 대소문자를 가리지 않고, 같은 키는 처음 것, 값의 앞뒤 공백을 뗀다 |
| `TestParseXshSessionFields` | 세션 → 후보: 출처 `xshell`, 이름, 폴더, 호스트·포트·사용자, 인증 `password`, 인코딩, 연결 유지 60초, 메모. **비밀번호 글이 후보 어디에도 없다** |
| `TestParseXshSessionVariants` | 텔넷(포트 0 → 23), 코드 페이지 9가지, 터미널 종류, 연결 유지 꺼짐 → 30, 사용자 키·포워딩·프록시 안내 문구(` · ` 로 이음), 에이전트, 건너뛰기(호스트 없음, SERIAL·RLOGIN·SFTP·FTP·LOCAL·RDP) |
| `TestScanXshellFolder` | 세션 폴더: 하위 폴더 구조, `.xsh` 만(대소문자 무시), 사전순. 고른 폴더 안에 `Sessions` 가 있으면 그것을 읽고 그 밖의 `.xsh` 는 읽지 않는다. 없는 경로는 오류 |
| `TestScanXshellExportFile` | 내보내기 파일: `Xshell/` 아래만, **CP949 로 적힌 한글 폴더 이름이 한글로** 읽힌다, `..` 가 든 항목·다른 프로그램 폴더·1MB 넘는 파일은 버린다 |
| `TestScanXshellBadFile` | zip 이 아닌 파일 → "Xshell 내보내기 파일을 열지 못했습니다: …" |
| `TestXshellSessionsUnder` | 문서 폴더에서 세션 폴더 찾기: 8 → 7 → 6 → 옛 위치 순, 없으면 빈 문자열 |
| `TestXshellRealExport` | (선택 실행) 환경 변수 `DARITERM_XTS` 에 실제 `.xts` 경로를 주면 읽어 **수만** 적는다. 변수가 없으면 건너뛴다 |

### G.2 실제 내보내기 파일로 확인

사용자가 준 파일로 `TestXshellRealExport` 를 돌린 결과(값은 출력하지 않고 수만 센다):

```
세션 파일 164개 → 후보 164개, 프로토콜 map[ssh:164], 인코딩 map[utf-8:164], 폴더 깊이 map[1:36 2:107 3:21], 인증 map[password:164]
사용자 이름 있음 164, 메모 있음 161, 옮기지 않음 안내 0, 연결 유지 60초 164, 깨진 이름 0
```

§2 에서 파일을 따로 살펴 센 수(세션 164개, 폴더 깊이, 메모 161개, 한글 이름)와 같다. 건너뛴 세션이 없고, 한글 폴더 이름이 깨지지 않았다.

### G.3 프런트 — [frontend/src/screens/ImportDialog.test.tsx](../../frontend/src/screens/ImportDialog.test.tsx) 에 5개 추가

| 테스트 | 확인하는 것 |
|---|---|
| [Xshell 내보내기 파일 선택]의 결과가 목록에 더해지고 폴더 이름이 "Xshell" | 폴더 경로 표시(`LAB/DB/`), 출처 "Xshell", 다른 출처 후보는 그대로, 채널을 인자 없이 한 번 부른다 |
| 고른 Xshell 후보를 그대로 import.apply 로 보낸다 | 후보 2개, `folderPath`, 폴더 이름 "Xshell" |
| 파일 선택을 취소(null)하면 아무 일도 없다 | 알림 없음 |
| 세션이 없으면 정보 토스트, 읽지 못하면 오류 토스트 | 문구 2가지 |
| 자동 검색에 Xshell 후보만 있으면 기본 폴더 이름이 "Xshell" | — |

### G.4 실행한 명령

```bash
go build ./... && go vet ./internal/importer/ ./internal/app/ ./internal/dialog/     # 통과
go test -count=1 ./...                 # 전부 통과 (이 PC 는 cgo 가 꺼져 있어 -race 는 돌리지 못했다)
DARITERM_XTS="<받은 파일 경로>" go test -count=1 -run Xshell -v ./internal/importer/    # §G.2
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 51개 파일 348개 통과 (앞 문서까지 343개 + 이 문서 5개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

자동 테스트가 확인하지 못하는 것:
- 실제 앱에서 파일 선택 창이 뜨고 `.xts` 만 걸러 보이는지(`dialog.OpenFileOf` 는 PowerShell 로 Windows 창을 띄운다).
- 가져온 서버로 실제 접속이 되는지.
- 이 PC 에 Xshell 이 없어 **기본 세션 폴더 자동 검색**은 실제 설치본으로 확인하지 못했다(폴더 찾는 순서만 임시 폴더로 시험).
- 받은 파일에 없던 값: 텔넷 세션, 사용자 키·에이전트 세션, 포워딩·프록시가 든 세션, UTF-8 이 아닌 코드 페이지.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 사이드바 [가져오기] | SecureCRT 버튼 아래에 [Xshell 내보내기 파일 선택]과 설명 |
| 2 | 버튼 → 받은 `.xts` 파일 선택 | 파일 선택 창에 `.xts` 만 보인다. 목록에 164개가 `폴더/폴더/이름` 으로 나오고 출처 "Xshell", 넣을 폴더 이름 "Xshell". 한글 폴더 이름이 깨지지 않는다 |
| 3 | 몇 개만 골라 [n개 가져오기] | 사이드바에 "Xshell" 폴더와 그 아래 폴더 구조. 서버의 호스트·포트·사용자·메모가 Xshell 과 같다 |
| 4 | 가져온 서버에 접속 | 비밀번호를 한 번 묻고 접속된다(비밀번호는 옮기지 않았다) |
| 5 | 같은 파일을 다시 가져오기 | 이미 있는 서버(호스트·포트·사용자가 같음)는 선택이 풀려 있다 |
| 6 | 파일 선택 창에서 [취소] | 아무 일도 없다 |
| 7 | `.xts` 가 아닌 파일(아무 텍스트 파일)을 고름 | 오류 알림 "Xshell 파일을 읽지 못했습니다: Xshell 내보내기 파일을 열지 못했습니다: …" |
| 8 | Xshell 이 설치된 PC 에서 가져오기를 연다 | 버튼을 누르지 않아도 Xshell 세션이 목록에 있다 |
| 9 | Xshell 에서 사용자 키 인증·포트 포워딩·프록시를 쓰는 세션을 만들어 내보낸 뒤 가져오기 | 그 줄의 툴팁에 "옮기지 않음" 안내. 사용자 키 세션은 인증이 "SSH 에이전트"로 등록 |
| 10 | 텔넷 세션이 든 파일 | 종류 TELNET, 포트 23 |
| 11 | 시리얼 세션이 든 파일 | 그 세션은 목록에 없다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 다른 가져오기 | `import.scan` 에 Xshell 검색이 끼었다 | 기본 폴더가 없으면 빈 목록이고 오류를 내지 않는다. 기존 가져오기 테스트 통과 |
| 가져오기를 여는 속도 | Xshell 세션이 많은 PC 에서는 파일을 그만큼 읽는다 | 세션 파일은 작다(10KB 남짓). 164개를 읽는 데 0.06초 |
| 남이 준 `.xts` | 압축 폭탄, 경로에 `..` | 풀지 않고 읽는다. 파일당 1MB·세션 20000개·합계 64MB 한도, `..` 가 든 항목은 버린다. 테스트가 본다 |
| 폴더 이름의 인코딩 | 일본어·중국어 Windows 에서 내보낸 파일은 폴더 이름이 깨질 수 있다(CP949 로 읽는다) | 세션 이름·호스트 등 파일 **내용**은 UTF-16 이라 깨지지 않는다. 깨진 폴더 이름은 가져온 뒤 고칠 수 있다 |
| 시리얼 세션 | 조용히 빠진다(알림이 없다) | 버튼 옆 설명과 문서에 적지는 않았다 — 시나리오 문서에는 적었다. 필요하면 시리얼 세션이 든 파일을 받아 번호의 뜻을 확인하고 더한다 |
| 표본 미확인 값 | 텔넷·사용자 키·에이전트·자동 실행 명령·포워딩 개수·프록시 이름은 키 이름만 보고 만들었다(값이 든 세션이 없었다) | 키 이름은 받은 파일에 있는 그대로다. 수동 검증 9·10 |
| 사용자 키 세션 | `agent` 인증으로 등록돼, 에이전트에 그 키가 없으면 접속이 안 된다 | 목록에 안내를 붙였다. 서버 편집에서 키를 지정하면 된다 |
| `Description` | Xshell 의 설명이 메모로 들어간다 | 비어 있으면 넣지 않는다 |
| 번역 | 한국어·영어 외 8개 언어의 문구 11개는 이 작업을 한 세션이 번역했다 | 검수가 필요할 수 있다 |
| 안내 박스 | "PuTTY … SecureCRT … 를 읽어 옵니다"에 Xshell 이 없다(§3.3) | 버튼 옆 설명이 있다. 고치려면 3개 문구 × 10개 언어 |
| 받은 파일 | `c:\work\win-projects-main\xbackup20260508.xts` 에는 실제 서버 주소와 암호화된 비밀번호가 들어 있다. 저장소 폴더 바로 위에 있다 | 이 저장소는 git 저장소가 아니지만, 저장소로 만들 때 이 파일이 들어가지 않게 한다. 가져오기를 마쳤으면 지우는 것이 낫다 |

되돌리기: 새 파일 4개(`xshell*.go`, `xshell_test.go`)를 지우고 §B·§C 의 `-` 줄로 되돌린 뒤, 문구 11개(§E)를 10개 파일에서 지운다. 이미 가져온 서버는 보통 서버라 그대로 남는다(태그 `xshell`).

## J. 체크리스트

- [x] `xshell.go`: `.xsh` 해석(UTF-16), 세션 → 후보, 세션 폴더·내보내기 파일 읽기(한도, 이름 인코딩), `ScanXshell`
- [x] `xshell_windows.go`·`xshell_other.go`: 기본 세션 폴더
- [x] `types.go` 출처 `xshell`, `dialog.OpenFileOf`, 채널 `import.scanXshell`, `import.scan` 에 추가
- [x] 프런트: 출처 이름·버튼·기본 폴더 이름, 가짜 백엔드
- [x] 문구 11개(10개 언어)
- [x] Go 테스트 8개, 프런트 테스트 5개. `go test ./...`·`tsc`·`vitest`(348개)·빌드
- [x] 실제 내보내기 파일로 확인(164개 → 164개)
- [x] 명세 import.md §1.6·ipc.md·ui/dialogs.md §5, 시나리오 ko·en, CHANGELOG, decisions D128, CLAUDE.md
- [x] 수동 검증 1~11 — 2026-11-25 사용자가 앱에서 확인함
