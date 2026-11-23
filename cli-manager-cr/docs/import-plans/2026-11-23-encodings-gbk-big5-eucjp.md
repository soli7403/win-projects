# 문자 인코딩 GBK · Big5 · EUC-JP 추가 — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트·빌드 완료(D118). 실제 서버·화면 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm)
**관련**: [spec/protocols.md](../../spec/protocols.md) §1.1, [spec/data/store.md](../../spec/data/store.md) §3, [spec/ui/host-editor.md](../../spec/ui/host-editor.md) §1.3, [spec/import.md](../../spec/import.md), [internal/session/encoding.go](../../internal/session/encoding.go). 같은 날 문서 묶음의 앞 문서 [remove-location-picker](./2026-11-23-remove-location-picker.md).

---

## 0. 한 줄 요약

서버 편집 > 고급 > "문자 인코딩"에 **EUC-JP · GBK · Big5** 를 더했다. 전에는 UTF-8, CP949/EUC-KR, Shift_JIS, Latin-1 뿐이라 중국어 간체·번체 장비와 EUC-JP 를 쓰는 일본어 서버의 글자가 깨졌다. 화면은 10개 언어(중국어 간체·번체, 일본어 포함)인데 인코딩은 그 언어권을 덮지 못하던 것을 맞췄다.

| 설정값 | 화면 이름 | 쓰는 곳 |
|---|---|---|
| `euc-jp` | EUC-JP | 일본어 유닉스 계열의 옛 서버 |
| `gbk` | GBK | 중국어 간체. **GB2312 를 포함**하므로 GB2312 장비도 이 값으로 읽는다 |
| `big5` | Big5 | 중국어 번체(대만·홍콩) |

## 1. 배경

- [internal/session/encoding.go](../../internal/session/encoding.go) `encodingByName` 이 아는 이름은 `cp949`·`euc-kr`·`shift_jis`·`latin1` 넷이었다. 그 밖의 값은 UTF-8 로 다룬다.
- 상용 제품 비교(같은 날)에서 "인코딩: GBK, Big5, EUC-JP 가 없는데 화면은 10개 언어"로 적었던 항목이다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 변환기 | `golang.org/x/text` 의 `simplifiedchinese.GBK`, `traditionalchinese.Big5`, `japanese.EUCJP` | 이미 쓰는 모듈(`korean.EUCKR`, `japanese.ShiftJIS`)의 다른 하위 패키지다. **새 의존성 없음**(go.mod·고지 파일 변화 없음) |
| 기존 처리 흐름 | 그대로 쓴다: 출력 묶음 끝에서 잘린 글자는 다음 묶음과 이어 붙이고, 잘못된 바이트는 대체 문자(`�`)로, 그 인코딩에 없는 글자를 입력하면 UTF-8 바이트를 그대로 보낸다 | 세 인코딩 모두 글자 하나가 최대 2~3바이트라 기존의 "남은 4바이트까지는 잘린 글자로 본다" 규칙 안에 든다 |
| GB2312 · GB18030 | 따로 두지 않는다 | GBK 가 GB2312 를 포함한다. GB18030(4바이트 글자)은 요청에 없고 쓰는 장비가 드물다 — 필요하면 `simplifiedchinese.GB18030` 한 줄이다 |
| 화면 이름 | "EUC-JP", "GBK", "Big5" — 번역하지 않는다 | Shift_JIS·Latin-1 과 같은 방식. 문구 파일(10개 언어)을 건드리지 않는다 |
| 목록 순서 | UTF-8 · CP949/EUC-KR · Shift_JIS · **EUC-JP · GBK · Big5** · Latin-1 | 일본어 둘을 붙여 두고 Latin-1 을 끝에 |
| 가져오기 | PuTTY·SecureCRT 의 인코딩 이름에 `euc-jp` / `gbk`·`936`·`gb2312` / `big5`·`950` 이 들어 있으면 해당 값으로 | 전에는 UTF-8 로 떨어져 가져온 뒤 손으로 바꿔야 했다 |
| 받는 값 목록 | `session.G_cslEncodings` | 테스트가 "목록의 모든 이름이 실제로 다뤄지는지" 확인한다 |

### 2.1 알아 둘 점

- SecureCRT 의 "Japanese (EUC)" 같은 이름은 전에 `japanese` 규칙에 걸려 Shift_JIS 로 가져왔다. 이제 `euc` 가 함께 있으면 EUC-JP 로 가져온다. **이미 가져온 서버는 바뀌지 않는다**(다시 가져오거나 서버 편집에서 고친다).
- PuTTY·SecureCRT 가 실제로 저장하는 인코딩 이름의 정확한 글자는 확인하지 못했다(두 프로그램의 세션 파일 표본에 이 인코딩이 없었다). 그래서 이름 전체가 아니라 **들어 있는 낱말**로 판정한다.
- 세션 로그는 이미 UTF-8 로 바뀐 글을 쓰므로 인코딩과 상관없이 UTF-8 파일이다(전과 같다).
- ZMODEM 은 인코딩 변환 앞에서 동작하므로 영향이 없다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/screens/HostEditor.tsx` | 코드 | 수정 |
| `internal/importer/putty.go` | 코드 | 수정 |
| `internal/importer/securecrt.go` | 코드 | 수정 |
| `internal/session/encoding.go` | 코드 | 수정 |
| `internal/importer/encoding_test.go` | 테스트 | 새 파일 |
| `internal/session/encoding_test.go` | 테스트 | 새 파일 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/import.md` | 명세 | 수정 |
| `spec/protocols.md` | 명세 | 수정 |
| `spec/ui/host-editor.md` | 명세 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/ipc/types.ts`, `frontend/src/screens/HostEditor.tsx`, `spec/data/store.md`, `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -44,7 +44,7 @@
   color?: string;
   env?: EnvKind;
   tags: string[];
-  encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'latin1';
+  encoding: 'utf-8' | 'euc-kr' | 'cp949' | 'shift_jis' | 'euc-jp' | 'gbk' | 'big5' | 'latin1';
   termType: string;
   startupCommand?: string;
   keepSession?: boolean;
```

### `frontend/src/screens/HostEditor.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditor.tsx
+++ b/frontend/src/screens/HostEditor.tsx
@@ -54,6 +54,9 @@
   { value: 'utf-8', label: 'UTF-8' },
   { value: 'cp949', label: t('enc.cp949') },
   { value: 'shift_jis', label: 'Shift_JIS' },
+  { value: 'euc-jp', label: 'EUC-JP' },
+  { value: 'gbk', label: 'GBK' },
+  { value: 'big5', label: 'Big5' },
   { value: 'latin1', label: 'Latin-1' },
 ];
 
```

### `internal/importer/putty.go` — 수정

```diff
--- a/internal/importer/putty.go
+++ b/internal/importer/putty.go
@@ -83,8 +83,15 @@
 		return "latin1"
 	}
 	szLower := strings.ToLower(szCodePage)
-	if strings.Contains(szLower, "949") || strings.Contains(szLower, "euc-kr") {
+	switch {
+	case strings.Contains(szLower, "949"), strings.Contains(szLower, "euc-kr"):
 		return "cp949"
+	case strings.Contains(szLower, "euc-jp"):
+		return "euc-jp"
+	case strings.Contains(szLower, "gbk"), strings.Contains(szLower, "936"), strings.Contains(szLower, "gb2312"):
+		return "gbk"
+	case strings.Contains(szLower, "big5"), strings.Contains(szLower, "950"):
+		return "big5"
 	}
 	return "utf-8"
 }
```

### `internal/importer/securecrt.go` — 수정

```diff
--- a/internal/importer/securecrt.go
+++ b/internal/importer/securecrt.go
@@ -50,8 +50,14 @@
 	switch {
 	case strings.Contains(szLower, "949"), strings.Contains(szLower, "euc-kr"), strings.Contains(szLower, "korean"):
 		return "cp949"
+	case strings.Contains(szLower, "euc-jp"), strings.Contains(szLower, "eucjp"), strings.Contains(szLower, "japanese") && strings.Contains(szLower, "euc"):
+		return "euc-jp" // "japanese" 만 보는 아래 규칙보다 먼저 본다
 	case strings.Contains(szLower, "shift"), strings.Contains(szLower, "japanese"):
 		return "shift_jis"
+	case strings.Contains(szLower, "gbk"), strings.Contains(szLower, "936"), strings.Contains(szLower, "gb2312"):
+		return "gbk"
+	case strings.Contains(szLower, "big5"), strings.Contains(szLower, "950"):
+		return "big5"
 	case strings.Contains(szLower, "8859-1"), strings.Contains(szLower, "latin"):
 		return "latin1"
 	}
```

### `internal/session/encoding.go` — 수정

```diff
--- a/internal/session/encoding.go
+++ b/internal/session/encoding.go
@@ -7,11 +7,16 @@
 	"golang.org/x/text/encoding/charmap"
 	"golang.org/x/text/encoding/japanese"
 	"golang.org/x/text/encoding/korean"
+	"golang.org/x/text/encoding/simplifiedchinese"
+	"golang.org/x/text/encoding/traditionalchinese"
 	"golang.org/x/text/transform"
 )
 
 // g_ciMaxBrokenBytes 는 잘린 글자로 볼 수 있는 최대 남은 바이트 수다. 이를 넘으면 잘못된 바이트다 (protocols.md §1.1).
 const g_ciMaxBrokenBytes = 4
+
+// G_cslEncodings 는 서버 설정 encoding 으로 받는 값 전부다 (protocols.md §1.1). utf-8 과 이 밖의 값은 UTF-8 로 다룬다.
+var G_cslEncodings = []string{"utf-8", "cp949", "euc-kr", "shift_jis", "euc-jp", "gbk", "big5", "latin1"}
 
 // encodingByName 은 설정값을 인코더로 바꾼다. utf-8 과 알 수 없는 값은 nil(UTF-8)이다.
 // cp949 와 euc-kr 은 같은 디코더를 쓴다: x/text 의 EUCKR 은 Code Page 949(UHC) 전체 표라 확장 한글("똠" 등)도 된다.
@@ -21,6 +26,13 @@
 		return korean.EUCKR
 	case "shift_jis":
 		return japanese.ShiftJIS
+	case "euc-jp":
+		return japanese.EUCJP
+	case "gbk":
+		// GBK 는 GB2312 를 포함한다. GB2312 로 보내는 장비도 이 값으로 읽힌다 (D118).
+		return simplifiedchinese.GBK
+	case "big5":
+		return traditionalchinese.Big5
 	case "latin1":
 		return charmap.ISO8859_1
 	default:
```


## C. 테스트 코드

### `internal/importer/encoding_test.go` — 새 파일 (전체)

```go
package importer

import "testing"

// 가져오기의 인코딩 이름 매핑 (import.md §1·§2). euc-jp · gbk · big5 는 D118 에서 추가.
func TestPuttyEncodingMapping(t *testing.T) {
	for szIn, szWant := range map[string]string{
		"":                  "utf-8",
		"UTF-8":             "utf-8",
		"CP949":             "cp949",
		"EUC-KR":            "euc-kr",
		"Shift_JIS":         "shift_jis",
		"EUC-JP":            "euc-jp",
		"GBK":               "gbk",
		"CP936":             "gbk",
		"GB2312":            "gbk",
		"Big5":              "big5",
		"CP950":             "big5",
		"Win1252":           "utf-8",
		"KOI8-R":            "utf-8",
		"Use font encoding": "utf-8",
	} {
		if szGot := puttyEncoding(szIn); szGot != szWant {
			t.Errorf("puttyEncoding(%q) = %q, 기대 %q", szIn, szGot, szWant)
		}
	}
}

func TestCrtEncodingMapping(t *testing.T) {
	for szIn, szWant := range map[string]string{
		"":                           "utf-8",
		"UTF-8":                      "utf-8",
		"Default":                    "utf-8",
		"Korean (EUC-KR)":            "cp949",
		"Japanese (Shift-JIS)":       "shift_jis",
		"Japanese (EUC)":             "euc-jp",
		"EUC-JP":                     "euc-jp",
		"Chinese Simplified (GBK)":   "gbk",
		"GB2312":                     "gbk",
		"Chinese Traditional (Big5)": "big5",
		"ISO-8859-1":                 "latin1",
	} {
		if szGot := crtEncoding(szIn); szGot != szWant {
			t.Errorf("crtEncoding(%q) = %q, 기대 %q", szIn, szGot, szWant)
		}
	}
}
```

### `internal/session/encoding_test.go` — 새 파일 (전체)

```go
package session

import (
	"bytes"
	"testing"
)

// 추가한 인코딩 GBK · Big5 · EUC-JP (protocols.md §1.1, D118).
// 바이트 값은 각 문자 집합의 공개 코드표에서 가져온 것이다(이 구현이 만든 값을 되읽는 것이 아니다).
var g_slEncodingCases = []struct {
	szName string
	szText string
	bBytes []byte
}{
	{"gbk", "中文", []byte{0xD6, 0xD0, 0xCE, 0xC4}},
	{"gbk", "你好", []byte{0xC4, 0xE3, 0xBA, 0xC3}},
	{"big5", "中文", []byte{0xA4, 0xA4, 0xA4, 0xE5}},
	{"big5", "你好", []byte{0xA7, 0x41, 0xA6, 0x6E}},
	{"euc-jp", "日本語", []byte{0xC6, 0xFC, 0xCB, 0xDC, 0xB8, 0xEC}},
	{"euc-jp", "ｶﾅ", []byte{0x8E, 0xB6, 0x8E, 0xC5}}, // 반각 가나: 0x8E 뒤에 한 바이트
}

// 서버가 보낸 바이트를 글자로, 입력한 글자를 서버 바이트로 바꾼다. ASCII 는 그대로다.
func TestNewEncodingsDecodeAndEncode(t *testing.T) {
	for _, tc := range g_slEncodingCases {
		if szGot := NewDecoder(tc.szName).Decode(tc.bBytes); szGot != tc.szText {
			t.Errorf("%s 해석: %q, 기대 %q", tc.szName, szGot, tc.szText)
		}
		if bGot := Encode(tc.szName, tc.szText); !bytes.Equal(bGot, tc.bBytes) {
			t.Errorf("%s 보내기: % x, 기대 % x", tc.szName, bGot, tc.bBytes)
		}
		szMixed := "ls -l " + tc.szText + "\r\n"
		bMixed := append(append([]byte("ls -l "), tc.bBytes...), '\r', '\n')
		if szGot := NewDecoder(tc.szName).Decode(bMixed); szGot != szMixed {
			t.Errorf("%s ASCII 와 섞인 글: %q", tc.szName, szGot)
		}
	}
}

// 글자가 출력 묶음 경계 어디에서 잘려 와도 이어 붙여 해석한다(한 바이트씩 넣어도 같다).
func TestNewEncodingsKeepSplitCharacters(t *testing.T) {
	for _, tc := range g_slEncodingCases {
		for iCut := 1; iCut < len(tc.bBytes); iCut++ {
			pDecoder := NewDecoder(tc.szName)
			if szGot := pDecoder.Decode(tc.bBytes[:iCut]) + pDecoder.Decode(tc.bBytes[iCut:]); szGot != tc.szText {
				t.Errorf("%s 자른 자리 %d: %q, 기대 %q", tc.szName, iCut, szGot, tc.szText)
			}
		}
		pDecoder := NewDecoder(tc.szName)
		szGot := ""
		for _, bOne := range tc.bBytes {
			szGot += pDecoder.Decode([]byte{bOne})
		}
		if szGot != tc.szText {
			t.Errorf("%s 한 바이트씩: %q, 기대 %q", tc.szName, szGot, tc.szText)
		}
	}
}

// 그 인코딩에 없는 글자를 입력하면 바꾸지 못하므로 UTF-8 바이트를 그대로 보낸다(전부터의 규칙, protocols.md §1.1).
// 잘못된 바이트는 대체 문자로 보이고 뒤의 글자는 계속 해석된다.
func TestNewEncodingsFallbacks(t *testing.T) {
	if bGot := Encode("big5", "가"); !bytes.Equal(bGot, []byte("가")) {
		t.Errorf("Big5 에 없는 글자: % x", bGot)
	}
	szGot := NewDecoder("gbk").Decode([]byte{0xFF, 'o', 'k', 0xD6, 0xD0})
	if szGot != "�ok中" {
		t.Errorf("잘못된 바이트 뒤의 해석: %q", szGot)
	}
}

// 받는 값 목록의 모든 이름이 실제로 다뤄진다: utf-8 만 변환기가 없고 나머지는 있다.
func TestEncodingNamesAllHandled(t *testing.T) {
	for _, szName := range G_cslEncodings {
		if bHas := encodingByName(szName) != nil; bHas == (szName == "utf-8") {
			t.Errorf("%s: 변환기 유무가 기대와 다름", szName)
		}
	}
	if encodingByName("no-such") != nil {
		t.Error("모르는 이름은 UTF-8 로 다뤄야 함")
	}
}
```


## F. 명세

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -49,7 +49,7 @@
 | color | 문자열(선택) | |
 | env | 환경 값(선택) | |
 | tags | 문자열 배열 | |
-| encoding | `utf-8` `euc-kr` `cp949` `shift_jis` `latin1` | 새 서버 기본 `utf-8` |
+| encoding | `utf-8` `euc-kr` `cp949` `shift_jis` `euc-jp` `gbk` `big5` `latin1` (`euc-jp`·`gbk`·`big5` 는 2026-11-23 D118) | 새 서버 기본 `utf-8` |
 | termType | 문자열 | 새 서버 기본 `xterm-256color` |
 | startupCommand | 문자열(선택) | 접속 직후 보낼 명령 |
 | keepSession | 불리언(선택) | 세션 유지(tmux) |
```

### `spec/import.md` — 수정

```diff
--- a/spec/import.md
+++ b/spec/import.md
@@ -12,7 +12,7 @@
 - **호스트·사용자**: `HostName` 에 `사용자@호스트` 가 들어 있으면 분리(`UserName` 이 이미 있으면 그 값 우선).
 - **포트**: `PortNumber`(없으면 22, 텔넷이고 값이 없으면 23).
 - **포트 포워딩**: `PortForwardings` 를 쉼표로 나눠 각 항목을 `[4|6]{L|R|D}[바인드:]포트=대상호스트:대상포트` 로 해석. **`D`(동적)는 건너뜀**. 바인드가 없으면 `127.0.0.1`, 대상 호스트가 없으면 `localhost`. 각 항목은 새 id 를 가진 포워딩.
-- **인코딩**: `LineCodePage`(없으면 UTF-8) 매핑 — `UTF-8`→utf-8, `CP949`→cp949, `EUC-KR`→euc-kr, `Shift_JIS`→shift_jis, `ISO-8859-1:1998 (Latin-1, West Europe)`→latin1, 그 외는 이름에 `949` 또는 `euc-kr`(대소문자 무시)가 들어 있으면 cp949, 아니면 utf-8.
+- **인코딩**: `LineCodePage`(없으면 UTF-8) 매핑 — `UTF-8`→utf-8, `CP949`→cp949, `EUC-KR`→euc-kr, `Shift_JIS`→shift_jis, `ISO-8859-1:1998 (Latin-1, West Europe)`→latin1, 그 외는 이름에(대소문자 무시) `949` 또는 `euc-kr` 이 들어 있으면 cp949, `euc-jp` 면 euc-jp, `gbk`·`936`·`gb2312` 면 gbk, `big5`·`950` 이면 big5(뒤의 셋은 2026-11-23 D118), 아니면 utf-8.
 - **터미널 종류**: `TerminalType`(비어 있거나 `xterm` 이면 `xterm-256color`).
 - **연결 유지**: `PingIntervalSecs`, 없으면 `PingInterval`, 값이 0 이거나 없으면 30.
 - **인증**: `PublicKeyFile` 이 있으면 `key`(후보의 keyFile = 그 경로), 아니면 `password`.
@@ -33,7 +33,7 @@
 - 건너뛰는 세션: ssh·telnet 인데 `Hostname` 이 빈 것.
 - **호스트·사용자**: `Hostname`, `Username`.
 - **포트**: SSH2 는 `[SSH2] Port`, SSH1 은 `[SSH1] Port`, 텔넷은 `Port`. 없거나 0 이면 22(텔넷 23).
-- **인코딩**: `Output Transformer Name` — `UTF-8`(또는 없음·`Default`)→utf-8, 이름에 `949`·`euc-kr`·`korean`(대소문자 무시)이 있으면 cp949, `shift`·`japanese` 가 있으면 shift_jis, `8859-1`·`latin` 이 있으면 latin1, 그 외 utf-8.
+- **인코딩**: `Output Transformer Name` — `UTF-8`(또는 없음·`Default`)→utf-8, 이름에 `949`·`euc-kr`·`korean`(대소문자 무시)이 있으면 cp949, `euc-jp`·`eucjp` 또는 (`japanese` 와 `euc` 가 함께) 있으면 euc-jp, 그 밖에 `shift`·`japanese` 가 있으면 shift_jis, `gbk`·`936`·`gb2312` 가 있으면 gbk, `big5`·`950` 이 있으면 big5(euc-jp·gbk·big5 는 2026-11-23 D118), `8859-1`·`latin` 이 있으면 latin1, 그 외 utf-8.
 - **터미널 종류**: 옮기지 않는다(적용 기본값 xterm-256color). SecureCRT 기본 에뮬레이션(VT100)을 그대로 쓰면 원격 프로그램의 색이 줄기 때문.
 - **연결 유지**: `Idle NO-OP Check` 가 0 이 아니고 `Idle NO-OP Timeout` 이 양수면 그 초, 아니면 30.
 - **인증·키 파일**: `Use Global Public Key` 가 0 이면 세션의 `Identity Filename V2`, 아니면(없음 포함) 설정 폴더 `SSH2.ini` 의 `Identity Filename V2`. 값이 비어 있지 않으면 인증 `key`·keyFile 은 그 경로(값에 `::` 가 있으면 그 앞까지, `%이름%`·`${이름}` 은 환경 변수로 바꾸고 `${VDS_CONFIG_PATH}` 는 설정 폴더). 비어 있으면 `password`. **저장된 비밀번호(`Password V2` 등)는 읽지 않는다.**
```

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -13,6 +13,9 @@
 | utf-8 | UTF-8. 청크 끝에 잘린 멀티바이트 글자는 다음 데이터와 이어 붙여 해석. 잘못된 바이트는 `�` 로 치환 |
 | cp949, euc-kr | EUC-KR(CP949) |
 | shift_jis | Shift_JIS |
+| euc-jp | EUC-JP (2026-11-23, D118) |
+| gbk | GBK — GB2312 를 포함한다 (2026-11-23, D118) |
+| big5 | Big5 (2026-11-23, D118) |
 | latin1 | ISO-8859-1 |
 - 비-UTF-8 인코딩에서 잘린 글자는 다음 청크로 넘긴다. 남은 바이트가 4바이트를 넘으면(잘린 글자가 아니라 잘못된 바이트) 버리지 않고 `�` 하나로 치환하고 계속한다.
 - 입력 변환 실패 시 원본 UTF-8 바이트를 그대로 보낸다.
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -46,7 +46,7 @@
 3. **세션 유지 (tmux)** 카드 — 표시 조건: SSH, 또는 로컬이면서 (Windows 가 아니거나 선택한 셸이 WSL 인 경우). 제목 "연결 유지 (tmux)", 설명 "창이나 앱을 닫아도 tmux 안의 프로그램은 계속 실행되고, 다시 열면 그 화면에 그대로 붙습니다. " + (로컬이고 Windows 면 "Windows 로컬은 WSL 셸일 때만 지원합니다." 아니면 "대상에 tmux 가 있어야 합니다."), 오른쪽 스위치. **새 로컬 터미널을 만들 때 tmux 가 있으면 이 스위치를 기본으로 켠다**(값이 아직 정해지지 않았을 때만).
    - 스위치가 켜져 있으면 필드 "tmux 세션 이름"(고정폭, 도움말 "비워 두면 서버 표시 이름(또는 로컬 폴더 이름)으로 만듭니다", 플레이스홀더 별칭 또는 "dariterm").
 4. **접속 후 자동 실행 명령**(로컬이면 "시작할 때 실행할 명령") — 고정폭 입력. 도움말 로컬 "예: claude --continue", 그 외 "예: cd /var/www && sudo -i"(플레이스홀더도 같음). 세션 유지가 켜져 있으면 줄바꿈 후 "세션을 처음 만들 때만 실행됩니다 (이미 있는 세션에 다시 붙을 때는 실행되지 않음)".
-5. 로컬: "터미널 종류" 한 칸. 그 외: 3열 — "문자 인코딩"(옵션 4개: "UTF-8"(utf-8), "CP949 / EUC-KR (구형 장비)"(cp949), "Shift_JIS"(shift_jis), "Latin-1"(latin1). 저장 데이터에 `euc-kr` 값이 있어도 읽을 수 있어야 한다) / "터미널 종류" / "연결 확인 간격 (초)"(숫자, 비면 0).
+5. 로컬: "터미널 종류" 한 칸. 그 외: 3열 — "문자 인코딩"(옵션 7개, 이 순서: "UTF-8"(utf-8), "CP949 / EUC-KR (구형 장비)"(cp949), "Shift_JIS"(shift_jis), "EUC-JP"(euc-jp), "GBK"(gbk), "Big5"(big5), "Latin-1"(latin1) — 가운데 셋은 2026-11-23 D118 추가. 저장 데이터에 `euc-kr` 값이 있어도 읽을 수 있어야 한다) / "터미널 종류" / "연결 확인 간격 (초)"(숫자, 비면 0).
 
 **프록시 경유**(SSH 만, 2026-11-23, D110): 경유 서버 항목 아래에 "프록시 경유" 드롭다운(없음 / HTTP 프록시 (CONNECT) / SOCKS5 프록시), 도움말 "회사 방화벽처럼 서버에 바로 연결할 수 없을 때, 이 프록시를 거쳐 SSH 연결을 맺습니다." 종류를 고르면 카드가 나온다: 2열(2:1) "프록시 주소"(고정폭, 플레이스홀더 `proxy.example.com`) / "포트"(숫자, 플레이스홀더는 종류별 기본 포트 8080·1080), 2열 "프록시 사용자 이름"(도움말 "프록시가 로그인을 요구할 때만 넣습니다. 비우면 인증 없이 연결합니다.") / "비밀번호"(사용자 이름이 비면 비활성. 저장된 것이 있으면 플레이스홀더가 가림 문자, 도움말은 경유 서버 카드와 같은 규칙). 경유 서버를 골라 둔 상태면 카드 맨 위에 안내 "경유 서버를 쓰는 동안에는 이 프록시를 쓰지 않습니다. …". 저장: 종류와 주소가 있으면 `proxy` 를 쓰고(포트 0 은 기본 포트), 사용자 이름과 비밀번호를 넣었으면 `{별칭} 프록시` 이름의 비밀번호 계정으로 저장해 연결한다. 사용자 이름을 비우면 계정 연결을 끊는다. 검사: 종류를 골랐는데 주소가 비면 "프록시 주소를 입력하세요", 비밀번호를 넣었는데 앱이 잠겨 있으면 "앱이 잠겨 있어 프록시 비밀번호를 저장할 수 없습니다"(둘 다 고급 탭으로 이동).
 
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 | 확인하는 것 |
|---|---|
| [encoding_test.go](../../internal/session/encoding_test.go) `TestNewEncodingsDecodeAndEncode` | **공개 코드표의 바이트 값**으로 확인한다(이 구현이 만든 값을 되읽는 것이 아니다): GBK `中文` = `D6 D0 CE C4`, `你好` = `C4 E3 BA C3` / Big5 `中文` = `A4 A4 A4 E5`, `你好` = `A7 41 A6 6E` / EUC-JP `日本語` = `C6 FC CB DC B8 EC`, 반각 가나 `ｶﾅ` = `8E B6 8E C5`. 해석과 보내기 양쪽, ASCII 와 섞인 글 |
| `TestNewEncodingsKeepSplitCharacters` | 글자가 묶음 경계의 **모든 자리**에서 잘려 와도 같다 / 한 바이트씩 넣어도 같다 |
| `TestNewEncodingsFallbacks` | Big5 에 없는 글자(한글)를 입력하면 UTF-8 바이트 그대로 / 잘못된 바이트는 대체 문자가 되고 뒤의 글자는 계속 해석된다 |
| `TestEncodingNamesAllHandled` | 받는 값 목록의 이름마다 변환기가 있다(utf-8 만 없다) / 모르는 이름은 UTF-8 |
| [importer/encoding_test.go](../../internal/importer/encoding_test.go) `TestPuttyEncodingMapping`·`TestCrtEncodingMapping` | 가져오기 매핑 25가지(기존 값 포함). `Japanese (EUC)` → euc-jp, `Japanese (Shift-JIS)` → shift_jis |
| 기존 `TestDecoderKeepsSplitCharacters` 등 | 그대로 통과 |

```bash
go build ./...
go vet ./internal/session/ ./internal/importer/
go test ./internal/session/ ./internal/importer/ ./internal/app/    # 통과
(cd frontend && npx tsc --noEmit)                                   # 통과
(cd frontend && npx vitest run)                                     # PowerShell 에서. 43개 파일 279개 통과
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build         # build/bin/dariterm.exe
```

**자동 테스트가 확인하지 못하는 것**: 실제 서버의 출력, 화면의 글꼴. 터미널 글꼴(기본 `Cascadia Mono, D2Coding, Consolas, Malgun Gothic`)에 한자 글리프가 없으면 Windows 가 다른 글꼴로 대신 그린다 — 글자는 나오지만 폭이 어긋나 보일 수 있다.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 서버 편집 > 고급 > 문자 인코딩 | 목록이 UTF-8 · CP949 / EUC-KR · Shift_JIS · EUC-JP · GBK · Big5 · Latin-1 순서로 7개 |
| 2 | UTF-8 리눅스 서버에 인코딩을 GBK 로 한 서버 항목으로 접속 → `echo 中文 \| iconv -f utf8 -t gbk` | `中文` 이 바르게 보인다 |
| 3 | 같은 세션에서 `cat \| xxd` 를 띄우고 `中` 을 입력 | `d6 d0` 이 찍힌다(입력이 GBK 로 나간다) |
| 4 | Big5 로 같은 확인(`iconv -t big5`) | `中文` 이 보이고 `中` 입력은 `a4 a4` |
| 5 | EUC-JP 로 같은 확인(`iconv -t euc-jp`, `日本語`) | 바르게 보이고 `日` 입력은 `c6 fc` |
| 6 | 한자가 긴 줄(화면 폭을 넘는 `cat`)과 빠른 출력 | 글자가 깨지지 않는다(묶음 경계) |
| 7 | 인코딩이 UTF-8·CP949 인 기존 서버 | 전과 같다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 기존 인코딩 | `switch` 에 `case` 셋을 더했을 뿐이다 | 기존 테스트, `TestEncodingNamesAllHandled` |
| 저장 데이터 | `encoding` 에 새 값 셋. **옛 버전 앱**은 모르는 값을 UTF-8 로 다뤄 글자가 깨져 보인다(데이터는 상하지 않는다). 서버 편집을 열면 목록에 없는 값이라 선택 칸이 비어 보일 수 있다 | git 동기화로 옛 버전과 섞어 쓸 때만 해당 |
| 가져오기 | `Japanese (EUC)` 류가 Shift_JIS 대신 EUC-JP 로 온다(바른 쪽으로 바뀜) | `TestCrtEncodingMapping` |
| 실행 파일 크기 | 중국어 간체·번체 변환 표가 들어가 조금 커진다 | 재 보지 않았다 |

## J. 체크리스트

- [x] `encoding.go`: `case` 3개, `G_cslEncodings`
- [x] 가져오기 매핑(`putty.go`, `securecrt.go`)
- [x] 프런트: `Host.encoding` 타입, 서버 편집의 목록
- [x] 명세 protocols.md §1.1·data/store.md §3·ui/host-editor.md §1.3·import.md 2곳, decisions D118
- [x] 자동 테스트 6개 추가, 전체 통과, 빌드
- [ ] 수동 검증 1~7
