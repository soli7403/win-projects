package session

import "testing"

func TestStreamDecoderUTF8Split(t *testing.T) {
	arr := []byte("가나")
	d := &streamDecoder{}
	sz := d.Write(arr[:4]) + d.Write(arr[4:])
	if sz != "가나" {
		t.Fatalf("got %q", sz)
	}
}

func TestStreamDecoderEUCKRSplit(t *testing.T) {
	enc := encodingFor("euc-kr")
	arr := encodeInput(enc, "한글")
	d := &streamDecoder{enc: enc}
	sz := d.Write(arr[:1]) + d.Write(arr[1:3]) + d.Write(arr[3:])
	if sz != "한글" {
		t.Fatalf("got %q", sz)
	}
}
