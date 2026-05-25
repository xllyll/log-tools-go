package xencoding

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DecodeLogLine converts raw log bytes to valid UTF-8 for PostgreSQL.
func DecodeLogLine(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	if gbk, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw); err == nil && utf8.Valid(gbk) {
		return string(gbk)
	}
	return strings.ToValidUTF8(string(raw), "\uFFFD")
}

// SanitizeUTF8 ensures text is valid UTF-8 (defense in depth before DB insert).
func SanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	if gbk, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), s); err == nil && utf8.ValidString(gbk) {
		return gbk
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}
