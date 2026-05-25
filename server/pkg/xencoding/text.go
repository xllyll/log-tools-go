package xencoding

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DecodeLogLine converts raw log bytes to PostgreSQL-safe UTF-8 text.
func DecodeLogLine(raw []byte) string {
	raw = bytes.ReplaceAll(raw, []byte{0}, nil)
	if len(raw) == 0 {
		return ""
	}
	var s string
	if utf8.Valid(raw) {
		s = string(raw)
	} else if gbk, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw); err == nil && utf8.Valid(gbk) {
		s = string(gbk)
	} else {
		s = strings.ToValidUTF8(string(raw), "\uFFFD")
	}
	return SanitizeForDB(s)
}

// SanitizeForDB removes bytes PostgreSQL TEXT rejects and ensures valid UTF-8.
func SanitizeForDB(s string) string {
	if s == "" {
		return ""
	}
	// PostgreSQL TEXT 不允许 NUL (0x00)
	s = strings.ReplaceAll(s, "\x00", "")
	if s == "" {
		return ""
	}
	if utf8.ValidString(s) {
		return s
	}
	if gbk, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), s); err == nil && utf8.ValidString(gbk) {
		return strings.ReplaceAll(gbk, "\x00", "")
	}
	return strings.ReplaceAll(strings.ToValidUTF8(s, "\uFFFD"), "\x00", "")
}

// SanitizeUTF8 is an alias kept for callers.
func SanitizeUTF8(s string) string {
	return SanitizeForDB(s)
}
