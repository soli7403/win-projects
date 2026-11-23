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
