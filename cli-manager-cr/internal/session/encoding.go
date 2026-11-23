package session

import (
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// g_ciMaxBrokenBytes 는 잘린 글자로 볼 수 있는 최대 남은 바이트 수다. 이를 넘으면 잘못된 바이트다 (protocols.md §1.1).
const g_ciMaxBrokenBytes = 4

// G_cslEncodings 는 서버 설정 encoding 으로 받는 값 전부다 (protocols.md §1.1). utf-8 과 이 밖의 값은 UTF-8 로 다룬다.
var G_cslEncodings = []string{"utf-8", "cp949", "euc-kr", "shift_jis", "euc-jp", "gbk", "big5", "latin1"}

// encodingByName 은 설정값을 인코더로 바꾼다. utf-8 과 알 수 없는 값은 nil(UTF-8)이다.
// cp949 와 euc-kr 은 같은 디코더를 쓴다: x/text 의 EUCKR 은 Code Page 949(UHC) 전체 표라 확장 한글("똠" 등)도 된다.
func encodingByName(szName string) encoding.Encoding {
	switch szName {
	case "cp949", "euc-kr":
		return korean.EUCKR
	case "shift_jis":
		return japanese.ShiftJIS
	case "euc-jp":
		return japanese.EUCJP
	case "gbk":
		// GBK 는 GB2312 를 포함한다. GB2312 로 보내는 장비도 이 값으로 읽힌다 (D118).
		return simplifiedchinese.GBK
	case "big5":
		return traditionalchinese.Big5
	case "latin1":
		return charmap.ISO8859_1
	default:
		return nil
	}
}

// Decoder 는 서버 출력을 UTF-8 문자열로 바꾼다. 청크 끝의 잘린 글자는 다음 청크와 이어 붙인다.
type Decoder struct {
	encText encoding.Encoding
	bCarry  []byte
}

// NewDecoder 는 설정 인코딩 이름으로 디코더를 만든다.
func NewDecoder(szName string) *Decoder {
	return &Decoder{encText: encodingByName(szName)}
}

// Decode 는 청크 하나를 해석한다. 잘린 끝 부분은 보관했다가 다음 호출에서 이어 쓴다.
func (pDecoder *Decoder) Decode(bChunk []byte) string {
	bIn := append(pDecoder.bCarry, bChunk...)
	pDecoder.bCarry = nil
	if pDecoder.encText == nil {
		return pDecoder.decodeUtf8(bIn)
	}
	return pDecoder.decodeOther(bIn)
}

func (pDecoder *Decoder) decodeUtf8(bIn []byte) string {
	// 끝의 잘린 UTF-8 글자(최대 3바이트)만 보관한다.
	nKeep := 0
	for nBack := 1; nBack <= 3 && nBack <= len(bIn); nBack++ {
		bByte := bIn[len(bIn)-nBack]
		if bByte&0xC0 == 0x80 {
			continue
		}
		if bByte >= 0xC0 {
			nNeed := 2
			if bByte >= 0xF0 {
				nNeed = 4
			} else if bByte >= 0xE0 {
				nNeed = 3
			}
			if nBack < nNeed {
				nKeep = nBack
			}
		}
		break
	}
	if nKeep > 0 {
		pDecoder.bCarry = append([]byte{}, bIn[len(bIn)-nKeep:]...)
		bIn = bIn[:len(bIn)-nKeep]
	}
	return sanitizeUtf8(bIn)
}

// sanitizeUtf8 은 잘못된 바이트 연속을 U+FFFD 로 바꾼다.
func sanitizeUtf8(bIn []byte) string {
	if utf8.Valid(bIn) {
		return string(bIn)
	}
	szOut := make([]rune, 0, len(bIn))
	for len(bIn) > 0 {
		r, nSize := utf8.DecodeRune(bIn)
		szOut = append(szOut, r)
		bIn = bIn[nSize:]
	}
	return string(szOut)
}

func (pDecoder *Decoder) decodeOther(bIn []byte) string {
	decoder := pDecoder.encText.NewDecoder()
	bDst := make([]byte, len(bIn)*4+8)
	var szOut []byte
	for len(bIn) > 0 {
		nDst, nSrc, errTransform := decoder.Transform(bDst, bIn, false)
		szOut = append(szOut, bDst[:nDst]...)
		bIn = bIn[nSrc:]
		if errTransform == nil {
			break
		}
		if errTransform == transform.ErrShortSrc {
			// 아직 완성되지 않은 글자. 남은 바이트가 짧으면 다음 청크를 기다린다.
			if len(bIn) <= g_ciMaxBrokenBytes {
				pDecoder.bCarry = append([]byte{}, bIn...)
				break
			}
			// 잘린 글자가 아니라 잘못된 바이트다: 한 바이트를 U+FFFD 로 바꾸고 계속한다.
			szOut = append(szOut, []byte("�")...)
			bIn = bIn[1:]
			continue
		}
		// 다른 오류(ErrShortDst 등)는 버퍼를 키워 다시 시도한다.
		if errTransform == transform.ErrShortDst {
			bDst = make([]byte, len(bDst)*2)
			continue
		}
		szOut = append(szOut, []byte("�")...)
		bIn = bIn[1:]
	}
	return string(szOut)
}

// Encode 는 입력 문자열(UTF-8)을 서버 인코딩 바이트로 바꾼다. 변환에 실패하면 UTF-8 바이트를 그대로 쓴다.
func Encode(szName string, szText string) []byte {
	encText := encodingByName(szName)
	if encText == nil {
		return []byte(szText)
	}
	bOut, errEncode := encText.NewEncoder().Bytes([]byte(szText))
	if errEncode != nil {
		return []byte(szText)
	}
	return bOut
}
