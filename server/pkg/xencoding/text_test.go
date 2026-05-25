package xencoding

import "testing"

func TestSanitizeForDB_stripsNull(t *testing.T) {
	in := "hello\x00world"
	got := SanitizeForDB(in)
	if got != "helloworld" {
		t.Fatalf("got %q want helloworld", got)
	}
}

func TestDecodeLogLine_stripsNull(t *testing.T) {
	got := DecodeLogLine([]byte("a\x00b"))
	if got != "ab" {
		t.Fatalf("got %q want ab", got)
	}
}

func TestSanitizeForDB_validUTF8WithBinary(t *testing.T) {
	raw := []byte{0xE4, 0xBD, 0xA0, 0x00, 0xE5, 0xA5, 0xBD} // 你\x00好
	got := SanitizeForDB(string(raw))
	if got != "你好" {
		t.Fatalf("got %q want 你好", got)
	}
}
