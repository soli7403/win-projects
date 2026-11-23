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
