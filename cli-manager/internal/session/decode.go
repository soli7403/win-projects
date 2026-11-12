package session

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
)

// encodingFor 는 호스트의 encoding 설정을 x/text 인코딩으로 바꾼다 (utf-8 이면 nil).
func encodingFor(sz string) encoding.Encoding {
	switch strings.ToLower(sz) {
	case "euc-kr", "cp949":
		return korean.EUCKR
	case "shift_jis":
		return japanese.ShiftJIS
	case "latin1":
		return charmap.ISO8859_1
	}
	return nil
}

// streamDecoder 는 청크 경계에 걸친 멀티바이트 문자를 보존하며 바이트를 문자열로 바꾼다.
type streamDecoder struct {
	enc  encoding.Encoding
	rest []byte
}

func (pstrD *streamDecoder) Write(arr []byte) string {
	buf := append(pstrD.rest, arr...)
	pstrD.rest = nil
	if pstrD.enc == nil {
		// UTF-8: 끝에 잘린 문자는 다음 청크로 넘긴다
		iEnd := len(buf)
		for i := 1; i <= utf8.UTFMax-1 && i <= len(buf); i++ {
			b := buf[len(buf)-i]
			if utf8.RuneStart(b) {
				if !utf8.FullRune(buf[len(buf)-i:]) {
					iEnd = len(buf) - i
				}
				break
			}
		}
		pstrD.rest = append([]byte(nil), buf[iEnd:]...)
		return strings.ToValidUTF8(string(buf[:iEnd]), "�")
	}
	dec := pstrD.enc.NewDecoder()
	dst := make([]byte, len(buf)*4+8)
	nDst, nSrc, err := dec.Transform(dst, buf, false)
	if err != nil && nSrc < len(buf) {
		pstrD.rest = append([]byte(nil), buf[nSrc:]...)
		if len(pstrD.rest) > 4 { // 잘린 문자가 아니라 잘못된 바이트: 버리지 않고 치환 후 계속
			out := string(dst[:nDst]) + "�"
			pstrD.rest = nil
			return out
		}
	}
	return string(dst[:nDst])
}

// encodeInput 은 터미널 입력(UTF-8 문자열)을 호스트 인코딩 바이트로 바꾼다.
func encodeInput(enc encoding.Encoding, sz string) []byte {
	if enc == nil {
		return []byte(sz)
	}
	arr, err := enc.NewEncoder().Bytes([]byte(sz))
	if err != nil {
		return []byte(sz)
	}
	return arr
}
